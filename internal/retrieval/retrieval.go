// Package retrieval provides a unified facade over the graph and FTS5 search
// indexes. Every caller that needs ranked content goes through here; engine-
// specific SQL is contained in this package rather than scattered across
// callers.
//
// Routing (set by Query fields):
//   - Filters or Types present      → FTS5 spec-corpus path (type/status/tag
//     filters are spec-corpus-specific).
//   - Plain text, no filters        → graph node-key match first, FTS5
//     fallback (preserves the "I know exactly what I want" case).
//   - SemanticOK=true               → vector stub (falls through to
//     graph/FTS5 until Phase C adds an embedding index).
//
// Type boosts are scoring MULTIPLIERS, not hard cutoffs. Every node type can
// appear in results; high-signal types (Feature, Decision, ContextDoc) score
// higher than low-signal types (Commit, Person) for the same text-match
// quality. This is the architectural fix for the hand-rolled weight workaround
// introduced in commit d9997ea:
//
//	"Graph search now weights node types: specs/knowledge/code rank above
//	 commits/issues/people. The 'Task' search on go-task/task used to return
//	 99 commit messages and 1 useful result; now returns ContextDocs /
//	 Symbols at the top."
//
// Phase B (swap FTS5 for Bleve) and Phase C (vector search) are explicit
// non-goals for this package until those phases begin.
package retrieval

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/embeddings"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/index"
)

// Query describes what to retrieve. The routing path is determined by which
// fields are set.
type Query struct {
	Text       string
	Types      []string          // node/spec types to include; non-empty → FTS5 path
	Filters    map[string]string // key-value filters (status, tag, since); non-empty → FTS5 path
	Limit      int               // 0 → default (20)
	SemanticOK bool              // future: enable vector search when available (Phase C)
	// IncludeSuperseded skips the de-weight applied to specs whose
	// `superseded_by:` field is set. The `[SUPERSEDED → <slug>]`
	// annotation is still added to the snippet — de-weight is the rank
	// effect; the marker is the visibility effect. Default false.
	IncludeSuperseded bool

	// skipSupersedeDeweight is an internal flag set by retrieveHybrid
	// when it invokes the lexical sub-path. It tells retrieveViaFTS /
	// retrieveViaNodeIndex to keep emitting the [SUPERSEDED → <slug>]
	// annotation but skip their own score multiplier, so the de-weight
	// is applied exactly once — by fuseRRF — against the post-fusion
	// RRF score. See spec embeddings-superseded-respect, Decision 5.
	// Do not set this from external callers.
	skipSupersedeDeweight bool

	// IncludeKnowledge merges hand-authored .hero/knowledge/** results into
	// the primary results. Set by `hero ask` (a knowledge-base question);
	// left false by default `hero search` so knowledge never appears in
	// work-search. Spec: knowledge-surfacing.
	IncludeKnowledge bool

	// KnowledgeOnly restricts retrieval to the knowledge corpus. Set by
	// `hero search --knowledge`. Types, when present, filter by knowledge
	// kind (e.g. "battlecards", "decisions").
	KnowledgeOnly bool

	// DomainFilter is applied by lexical retrieval before ranking limits so
	// disabled-domain hits cannot starve enabled or explicitly scoped results.
	DomainFilter index.DomainFilter
}

// supersededDeweight is the score multiplier applied to results whose
// `superseded_by:` frontmatter field is set. Calibrated so a strong
// text match on a superseded spec can still beat a weak match on a
// non-superseded peer — superseded specs aren't hidden, they're moved
// down the list. Mirrors the typeBoost shape (multiplicative) so the
// scoring axis stays consistent.
const supersededDeweight = 0.3

// Result is a single ranked retrieval hit.
type Result struct {
	NodeID    int64
	Type      string
	Key       string
	Title     string
	Snippet   string
	Score     float64
	Source    string // "graph" | "fts5" | "vector"
	Status    string // spec status when Source=="fts5"; empty for graph nodes
	ClaimedBy string // non-empty when a spec is claimed
	Path      string // on-disk path when Source=="fts5"; empty for graph nodes
	Repo      string // remote-origin key; non-empty when result is from a sibling repo
	Domain    string // durable domain provenance (core / engineering / pm / qa / ...)
}

// Retriever wraps the graph store, FTS5 index, and optional embedding model.
// Instantiate with New; call Close when done.
type Retriever struct {
	store    *graph.Store // nil when graph DB is absent or failed to open
	fts      *index.DB
	embModel *embeddings.Model   // nil when embeddings unavailable
	embStore *embeddings.Storage // nil when embeddings unavailable
	// supersedeRespect mirrors config.RetrievalSupersedeRespect — when
	// false, the hybrid path skips the supersede overlay entirely
	// (fuseRRF runs unchanged). Defaults to true; loaded from project
	// config in New and harmlessly true in environments without config.
	supersedeRespect bool
}

// New opens the FTS5 index (required) and the graph store (optional — silently
// absent in environments without a populated graph, e.g. fresh workspaces or
// test environments that only initialise the FTS5 index).
//
// Embeddings are loaded best-effort: if no model is installed or the vector
// storage can't be opened, the Retriever works without semantic search.
func New(heroDir string) (*Retriever, error) {
	fts, err := index.Open(heroDir)
	if err != nil {
		return nil, fmt.Errorf("opening FTS index: %w", err)
	}
	r := &Retriever{fts: fts, supersedeRespect: true}
	if store, err := graph.Open(heroDir); err == nil {
		r.store = store
	}

	// Read the supersede-respect knob from project config. Failures
	// fall back to the default (true) — the flag is for rollback in
	// production, not a load-bearing prerequisite for retrieval.
	projectRoot := strings.TrimSuffix(heroDir, "/.hero")
	if projectRoot != heroDir {
		if cfg, err := config.Load(projectRoot); err == nil {
			r.supersedeRespect = cfg.RetrievalSupersedeRespect()
		}
	}

	// Best-effort: load embeddings for hybrid search.
	embModel, embStore, _ := LoadEmbeddings(heroDir, fts.RawDB())
	r.embModel = embModel
	r.embStore = embStore

	return r, nil
}

// LoadEmbeddings attempts to load the embedding model and open vector storage.
// Returns nil, nil, nil if embeddings are not available (no model, no index,
// or disabled via config). A non-nil error indicates a real failure (e.g.
// corrupt model files) rather than absence.
func LoadEmbeddings(heroDir string, indexDB *sql.DB) (*embeddings.Model, *embeddings.Storage, error) {
	if indexDB == nil {
		return nil, nil, nil
	}

	// Determine model name from config (best-effort — default if config unavailable).
	modelName := "hero-embed-v1"
	projectRoot := strings.TrimSuffix(heroDir, "/.hero")
	if projectRoot != heroDir {
		if cfg, err := config.Load(projectRoot); err == nil {
			modelName = cfg.EmbeddingsModel()
		}
	}

	model, err := embeddings.LoadModelFromConfig(modelName)
	if err != nil {
		return nil, nil, err
	}
	if model == nil {
		return nil, nil, nil // no model installed
	}

	store, err := embeddings.OpenStorage(indexDB)
	if err != nil {
		return nil, nil, fmt.Errorf("opening embedding storage: %w", err)
	}

	return model, store, nil
}

// Close releases both underlying handles.
func (r *Retriever) Close() error {
	if r.store != nil {
		r.store.Close()
	}
	return r.fts.Close()
}

// Retrieve returns ranked results for q. See package doc for routing rules.
func (r *Retriever) Retrieve(q Query) ([]Result, error) {
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}

	// `hero search --knowledge` → knowledge corpus only.
	if q.KnowledgeOnly {
		return r.retrieveKnowledge(q, limit)
	}

	base, err := r.retrieveBase(q, limit)
	if err != nil {
		return nil, err
	}

	// `hero ask` merges knowledge into the primary results. Work and knowledge
	// scores live on different scales (graph rank vs FTS rank vs synthetic), so
	// a score sort would let one corpus starve the other. Round-robin interleave
	// instead — knowledge first, since ask is a knowledge-base question — which
	// guarantees each corpus fair representation and preserves each list's own
	// internal ranking.
	if q.IncludeKnowledge {
		kres, _ := r.retrieveKnowledge(q, limit)
		if len(kres) > 0 {
			base = interleave(kres, base, limit)
		}
	}
	return base, nil
}

// interleave round-robins two ranked lists (a first) up to limit, dropping
// duplicates by Key.
func interleave(a, b []Result, limit int) []Result {
	out := make([]Result, 0, limit)
	seen := make(map[string]bool)
	i, j := 0, 0
	for len(out) < limit && (i < len(a) || j < len(b)) {
		if i < len(a) {
			if r := a[i]; !seen[r.Key] {
				seen[r.Key] = true
				out = append(out, r)
			}
			i++
		}
		if len(out) >= limit {
			break
		}
		if j < len(b) {
			if r := b[j]; !seen[r.Key] {
				seen[r.Key] = true
				out = append(out, r)
			}
			j++
		}
	}
	return out
}

// retrieveBase runs the work-corpus retrieval (graph → node index → FTS5),
// unchanged from the original Retrieve body. Knowledge is layered on by the
// caller so default `hero search` stays work-only.
func (r *Retriever) retrieveBase(q Query, limit int) ([]Result, error) {
	// Hybrid search: when SemanticOK is set and an embedding model is loaded,
	// run lexical retrieval then fuse with vector results via RRF.
	if q.SemanticOK && r.embModel != nil && r.embStore != nil {
		return r.retrieveHybrid(q, limit)
	}

	// Filters or explicit type constraints → FTS5 spec-corpus path.
	if len(q.Filters) > 0 || len(q.Types) > 0 {
		return r.retrieveViaFTS(q, limit)
	}

	// Plain text with no filters → unified node index (BM25) first.
	// Falls through to graph LIKE match (AC-5: "I know exactly what I want")
	// and then to spec-corpus FTS5 if the node index has no hits.
	if r.store != nil {
		results, err := r.retrieveViaNodeIndex(q, limit)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		results, err = r.retrieveViaGraph(q, limit)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	}
	return r.retrieveViaFTS(q, limit)
}

// retrieveKnowledge searches the isolated knowledge corpus (fts_knowledge).
// q.Types, when set, filters by knowledge kind. Results carry Source
// "knowledge" and a real on-disk Path so passage extraction (hero ask) works.
func (r *Retriever) retrieveKnowledge(q Query, limit int) ([]Result, error) {
	hits, err := r.fts.SearchKnowledge(q.Text, q.Types, limit)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(hits))
	for i, h := range hits {
		// Rank-descending synthetic score so knowledge interleaves sanely
		// with FTS spec scores when merged for `hero ask`.
		score := 1.0 - float64(i)*0.01
		results = append(results, Result{
			Type:    h.Kind,
			Key:     h.Slug,
			Title:   h.Title,
			Snippet: h.Content,
			Score:   score,
			Source:  "knowledge",
			Path:    h.Path,
			Domain:  h.Domain,
		})
	}
	return results, nil
}

// NudgeFiles returns file-based context for hero relevant. It delegates to the
// FTS5 index's BuildNudge, which checks conventions, past specs, and in-flight
// work that touched the given file paths.
func (r *Retriever) NudgeFiles(filePaths []string) (*index.NudgeResult, error) {
	return r.fts.BuildNudge(filePaths)
}

// ---------------------------------------------------------------------------
// Unified node index path (Phase B — BM25 via fts_nodes)
// ---------------------------------------------------------------------------

func (r *Retriever) retrieveViaNodeIndex(q Query, limit int) ([]Result, error) {
	ftsQuery := index.SanitizeFTSQuery(q.Text)
	if ftsQuery == "" {
		return nil, nil
	}

	indexDB := r.fts.RawDB()
	if indexDB == nil {
		return nil, nil
	}

	// LEFT JOIN specs by slug so we can pull `superseded_by` for spec-
	// backed nodes (Feature/Bug/Initiative/...) without a second query.
	// Non-spec nodes (Commit, Symbol, ...) have no specs row and the
	// joined column comes back as NULL → empty string after scan.
	baseQuery := `
		SELECT ni.node_type, ni.key, ni.path, ni.repo, ni.domain,
		       snippet(fts_nodes, 0, '>>>', '<<<', '...', 24) AS title_snip,
		       snippet(fts_nodes, 1, '>>>', '<<<', '...', 32) AS body_snip,
		       fts_nodes.rank AS bm25_rank,
		       COALESCE(s.superseded_by, '') AS superseded_by
		  FROM fts_nodes
		  JOIN node_index ni ON ni.rowid = fts_nodes.rowid
		  LEFT JOIN specs s ON s.slug = ni.key
		 WHERE fts_nodes MATCH ?`
	args := []any{ftsQuery}
	where, whereArgs, domainOrder, orderArgs := retrievalDomainSQL("ni", q.DomainFilter)
	if where != "" {
		baseQuery += " AND " + where
		args = append(args, whereArgs...)
	}
	baseQuery += " ORDER BY "
	if domainOrder != "" {
		baseQuery += domainOrder + ", "
		args = append(args, orderArgs...)
	}
	baseQuery += "fts_nodes.rank LIMIT ?"
	args = append(args, limit*3) // over-fetch; we re-sort by boosted score
	rows, err := indexDB.Query(baseQuery, args...)
	if err != nil {
		return nil, nil // graceful: fts_nodes may be empty/absent
	}
	defer rows.Close()

	type cand struct {
		nodeType     string
		key          string
		path         string
		repo         string
		domain       string
		titleSnip    string
		bodySnip     string
		bm25Rank     float64
		score        float64
		supersededBy string
	}
	var cands []cand

	for rows.Next() {
		var c cand
		var rank sql.NullFloat64
		if err := rows.Scan(&c.nodeType, &c.key, &c.path, &c.repo, &c.domain, &c.titleSnip, &c.bodySnip, &rank, &c.supersededBy); err != nil {
			continue
		}
		if rank.Valid {
			c.bm25Rank = rank.Float64
		}
		// FTS5 rank is negative (more negative = better). Score = -rank * typeBoost.
		c.score = -c.bm25Rank * typeBoost(c.nodeType)
		// Skip the per-path de-weight when called as the lexical sub-step
		// of retrieveHybrid — fuseRRF applies it once against the fused
		// RRF score to avoid double-penalty. Annotation still fires below.
		if c.supersededBy != "" && !q.IncludeSuperseded && !q.skipSupersedeDeweight {
			c.score *= supersededDeweight
		}
		cands = append(cands, c)
	}

	if len(cands) == 0 {
		return nil, nil
	}

	sort.SliceStable(cands, func(i, j int) bool {
		leftRank := retrievalDomainRank(cands[i].domain, q.DomainFilter)
		rightRank := retrievalDomainRank(cands[j].domain, q.DomainFilter)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return cands[i].score > cands[j].score
	})
	if len(cands) > limit {
		cands = cands[:limit]
	}

	results := make([]Result, len(cands))
	for i, c := range cands {
		title := c.titleSnip
		if title == "" {
			title = c.key
		}
		snippet := c.bodySnip
		if snippet == "" {
			snippet = c.titleSnip
		}
		if c.supersededBy != "" {
			snippet = "[SUPERSEDED → " + c.supersededBy + "] " + snippet
		}
		results[i] = Result{
			Type:    c.nodeType,
			Key:     c.key,
			Title:   title,
			Snippet: snippet,
			Score:   c.score,
			Source:  "graph",
			Path:    c.path,
			Repo:    c.repo,
			Domain:  c.domain,
		}
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// Graph path (AC-5 fallback — exact key/title LIKE match)
// ---------------------------------------------------------------------------

// typeBoost returns the scoring multiplier for a graph node type.
//
// These are MULTIPLIERS, not cutoffs. See package doc for the d9997ea
// motivation: a Feature with a title match scores (1+1)*10 = 20, while a
// Commit with only a body match scores 1*1 = 1, so the Feature always wins
// without any hard limit on what types can appear.
func typeBoost(nodeType string) float64 {
	switch nodeType {
	case "Tripwire":
		return 12
	case "Feature", "Bug", "Initiative", "Decision":
		return 10
	case "Convention", "Rule":
		return 9
	case "ContextDoc", "Note":
		return 8
	case "Symbol", "Package":
		return 6
	case "File":
		return 5
	case "Issue":
		return 3
	case "Commit", "Person":
		return 1
	default:
		return 4
	}
}

// fts5TypeBoost maps FTS5 spec types to the same multiplier scale so results
// from both sources are scored on a consistent axis.
func fts5TypeBoost(specType string) float64 {
	switch strings.ToLower(specType) {
	case "tripwire":
		return 12
	case "feature", "bug", "initiative", "decision":
		return 10
	case "convention", "rule":
		return 9
	case "context", "note":
		return 8
	default:
		return 4
	}
}

func (r *Retriever) retrieveViaGraph(q Query, limit int) ([]Result, error) {
	text := strings.ToLower(q.Text)

	// Fetch up to 200 candidates; scoring + sort below brings the best to the
	// front before we cap at limit.
	baseQuery := `SELECT id, type, key, repo, domain,
		        COALESCE(json_extract(props, '$.title'),   '') AS title,
		        COALESCE(json_extract(props, '$.body'),    '') AS body,
		        COALESCE(json_extract(props, '$.subject'), '') AS subject
		   FROM nodes
		  WHERE valid_to IS NULL
		    AND (lower(key) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.title'),   '')) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.body'),    '')) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.subject'), '')) LIKE '%' || ? || '%')`
	args := []any{text, text, text, text}
	where, whereArgs, domainOrder, orderArgs := retrievalDomainSQL("nodes", q.DomainFilter)
	if where != "" {
		baseQuery += " AND " + where
		args = append(args, whereArgs...)
	}
	baseQuery += " ORDER BY "
	if domainOrder != "" {
		baseQuery += domainOrder + ", "
		args = append(args, orderArgs...)
	}
	baseQuery += "ingested_at DESC LIMIT 200"
	rows, err := r.store.DB().Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type cand struct {
		id       int64
		nodeType string
		key      string
		repo     string
		domain   string
		title    string
		body     string
		score    float64
	}
	var cands []cand

	for rows.Next() {
		var c cand
		var subject string
		if err := rows.Scan(&c.id, &c.nodeType, &c.key, &c.repo, &c.domain, &c.title, &c.body, &subject); err != nil {
			return nil, err
		}
		if c.body == "" && subject != "" {
			c.body = subject
		}

		// Base score: any match = 1.0; key or title match adds another 1.0.
		// Multiply by the type boost to produce the final score.
		base := 1.0
		if strings.Contains(strings.ToLower(c.key), text) ||
			strings.Contains(strings.ToLower(c.title), text) {
			base += 1.0
		}
		c.score = base * typeBoost(c.nodeType)
		cands = append(cands, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(cands, func(i, j int) bool {
		leftRank := retrievalDomainRank(cands[i].domain, q.DomainFilter)
		rightRank := retrievalDomainRank(cands[j].domain, q.DomainFilter)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return cands[i].score > cands[j].score
	})
	if len(cands) > limit {
		cands = cands[:limit]
	}

	results := make([]Result, len(cands))
	for i, c := range cands {
		title := c.title
		if title == "" {
			title = c.key
		}
		snippet := oneLine(c.body)
		if len(snippet) > 160 {
			snippet = snippet[:160] + "…"
		}
		results[i] = Result{
			NodeID:  c.id,
			Type:    c.nodeType,
			Key:     c.key,
			Title:   title,
			Snippet: snippet,
			Score:   c.score,
			Source:  "graph",
			Repo:    c.repo,
			Domain:  c.domain,
		}
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// FTS5 path
// ---------------------------------------------------------------------------

func (r *Retriever) retrieveViaFTS(q Query, limit int) ([]Result, error) {
	var specType, status, tag, since, subproject string
	if len(q.Types) > 0 {
		specType = q.Types[0]
	}
	if q.Filters != nil {
		if v, ok := q.Filters["type"]; ok && specType == "" {
			specType = v
		}
		status = q.Filters["status"]
		tag = q.Filters["tag"]
		since = q.Filters["since"]
		subproject = q.Filters["subproject"]
	}

	var hits []index.SearchResult
	var err error

	hasFilter := specType != "" || status != "" || tag != "" || since != "" || (subproject != "" && subproject != "all")

	switch {
	case q.Text == "":
		hits, err = r.fts.ListFilteredScopedDomains(specType, status, tag, since, subproject, q.DomainFilter)
	case hasFilter:
		hits, err = r.fts.SearchFilteredScopedDomains(q.Text, specType, status, tag, since, subproject, q.DomainFilter)
	default:
		hits, err = r.fts.SearchDomains(q.Text, q.DomainFilter)
	}
	if err != nil {
		return nil, err
	}

	results := make([]Result, len(hits))
	for i, h := range hits {
		score := fts5TypeBoost(string(h.Type))
		snippet := h.Snippet
		if h.SupersededBy != "" {
			// Annotation goes on regardless of IncludeSuperseded — the
			// marker is the visibility cue any caller printing the
			// snippet needs. De-weight is the rank effect, opt-out via
			// IncludeSuperseded.
			snippet = "[SUPERSEDED → " + h.SupersededBy + "] " + snippet
			// Skip de-weight when invoked as the lexical sub-step of
			// retrieveHybrid; fuseRRF applies it once against the fused
			// RRF score (spec embeddings-superseded-respect, Decision 5).
			if !q.IncludeSuperseded && !q.skipSupersedeDeweight {
				score *= supersededDeweight
			}
		}
		results[i] = Result{
			Type:      string(h.Type),
			Key:       h.Slug,
			Title:     h.Title,
			Snippet:   snippet,
			Score:     score,
			Source:    "fts5",
			Status:    string(h.Status),
			ClaimedBy: h.ClaimedBy,
			Path:      h.Path,
			Domain:    h.Domain,
		}
	}

	// Re-sort by score so de-weighted superseded specs fall below
	// non-superseded peers regardless of original rank order. Stable
	// sort preserves FTS rank tie-breaks within each tier.
	sort.SliceStable(results, func(i, j int) bool {
		leftRank := retrievalDomainRank(results[i].Domain, q.DomainFilter)
		rightRank := retrievalDomainRank(results[j].Domain, q.DomainFilter)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func retrievalDomainSQL(alias string, filter index.DomainFilter) (where string, whereArgs []any, order string, orderArgs []any) {
	if filter.All || len(filter.Allowed) == 0 {
		return "", nil, "", nil
	}
	fallback := filter.Fallback
	if fallback == "" {
		fallback = "engineering"
	}
	expr := "COALESCE(NULLIF(" + alias + ".domain, ''), ?)"
	placeholders := make([]string, len(filter.Allowed))
	whereArgs = append(whereArgs, fallback)
	for i, domain := range filter.Allowed {
		placeholders[i] = "?"
		whereArgs = append(whereArgs, domain)
	}
	where = expr + " IN (" + strings.Join(placeholders, ",") + ")"

	orderDomains := filter.Order
	if len(orderDomains) == 0 {
		orderDomains = filter.Allowed
	}
	var cases strings.Builder
	cases.WriteString("CASE " + expr)
	orderArgs = append(orderArgs, fallback)
	seen := map[string]bool{}
	rank := 0
	for _, domain := range orderDomains {
		if domain == "" || seen[domain] {
			continue
		}
		seen[domain] = true
		fmt.Fprintf(&cases, " WHEN ? THEN %d", rank)
		orderArgs = append(orderArgs, domain)
		rank++
	}
	fmt.Fprintf(&cases, " ELSE %d END", rank)
	return where, whereArgs, cases.String(), orderArgs
}

func retrievalDomainRank(domain string, filter index.DomainFilter) int {
	if filter.All || len(filter.Allowed) == 0 {
		return 0
	}
	if domain == "" {
		domain = filter.Fallback
	}
	order := filter.Order
	if len(order) == 0 {
		order = filter.Allowed
	}
	for i, candidate := range order {
		if domain == candidate {
			return i
		}
	}
	return len(order)
}

// ---------------------------------------------------------------------------
// Hybrid search (Phase C — BM25 + vector via RRF)
// ---------------------------------------------------------------------------

// retrieveHybrid runs the normal lexical path and a vector similarity query,
// then fuses results via Reciprocal Rank Fusion.
//
// Supersede de-weighting in this path is applied exactly once, in fuseRRF,
// against the post-fusion RRF score (spec embeddings-superseded-respect,
// Decision 5). The lexical sub-call is told to skip its own multiplier via
// the internal skipSupersedeDeweight flag, but it still emits the
// [SUPERSEDED → <slug>] annotation. The vector path produces no annotation
// of its own; fuseRRF stamps the marker on any superseded fused result so
// vector-only hits also carry the redirect.
func (r *Retriever) retrieveHybrid(q Query, limit int) ([]Result, error) {
	// Build the supersede overlay once per query from the specs table
	// when respect is enabled. Cheap (small map, indexed query) and
	// avoids re-embedding on supersede. Empty when no specs carry
	// superseded_by or when the operator has rolled the feature back
	// via the RetrievalSupersedeRespect config knob.
	var overlay map[string]string
	if r.supersedeRespect {
		overlay, _ = loadSupersededOverlay(r.fts.RawDB())
	} else {
		overlay = map[string]string{}
	}

	// Run the lexical path (graph/FTS5) with SemanticOK=false to avoid
	// recursion. When the overlay is active, set skipSupersedeDeweight
	// so the sub-call keeps the annotation but defers the multiplier to
	// fuseRRF. When respect is off, the sub-call behaves exactly as a
	// non-hybrid lexical caller — parent-spec behavior, untouched.
	lexicalQ := q
	lexicalQ.SemanticOK = false
	lexicalQ.skipSupersedeDeweight = r.supersedeRespect
	lexical, err := r.Retrieve(lexicalQ)
	if err != nil {
		return nil, err
	}

	// Embed the query and retrieve similar chunks.
	queryVec := r.embModel.Embed(q.Text)
	vectorHits, err := r.embStore.QuerySimilar(queryVec, limit*2, nil)
	if err != nil {
		// Vector search failed — fall back to lexical-only. Re-apply
		// the supersede de-weight that the sub-call skipped, since
		// fuseRRF won't run.
		if r.supersedeRespect && !q.IncludeSuperseded {
			applySupersedeDeweight(lexical, overlay)
		}
		return lexical, nil
	}

	return fuseRRF(lexical, vectorHits, overlay, q.IncludeSuperseded, 60, limit), nil
}

// loadSupersededOverlay returns a map of {superseded-slug → replacement-slug}
// built from the specs table. Used by retrieveHybrid to apply the supersede
// de-weight + annotation in fuseRRF without re-embedding any chunk.
//
// Returns an empty map (and nil error) when no specs carry superseded_by or
// when the table is absent (older index schemas). The overlay is read once
// per Retrieve call; cardinality is small and the query hits an index.
func loadSupersededOverlay(db *sql.DB) (map[string]string, error) {
	out := map[string]string{}
	if db == nil {
		return out, nil
	}
	rows, err := db.Query(`SELECT slug, superseded_by FROM specs WHERE superseded_by != ''`)
	if err != nil {
		return out, nil // graceful: table or column may be absent
	}
	defer rows.Close()
	for rows.Next() {
		var slug, by string
		if err := rows.Scan(&slug, &by); err != nil {
			continue
		}
		out[slug] = by
	}
	return out, nil
}

// applySupersedeDeweight multiplies the score and prefixes the snippet of
// every result whose Key is in the overlay. Used by the lexical-only
// fallback path in retrieveHybrid when QuerySimilar fails — fuseRRF
// normally owns this in the hybrid path.
func applySupersedeDeweight(results []Result, overlay map[string]string) {
	if len(overlay) == 0 {
		return
	}
	for i := range results {
		by, ok := overlay[results[i].Key]
		if !ok {
			continue
		}
		results[i].Score *= supersededDeweight
		results[i].Snippet = annotateSuperseded(results[i].Snippet, by)
	}
}

// annotateSuperseded prefixes snippet with [SUPERSEDED → <by>] unless the
// marker is already present (idempotency: the lexical sub-call may have
// stamped it before handing the result up to fuseRRF).
func annotateSuperseded(snippet, by string) string {
	if strings.HasPrefix(snippet, "[SUPERSEDED → ") {
		return snippet
	}
	return "[SUPERSEDED → " + by + "] " + snippet
}

// fuseRRF merges lexical and vector result sets using Reciprocal Rank Fusion.
// k=60 per Cormack et al. For each result, the RRF score is the sum of
// 1/(k+rank) across the rankings it appears in.
//
// Matching logic:
//   - Vector chunks with corpus "spec" match on ScoredChunk.SourceID == Result.Key
//   - Other corpora match on ScoredChunk.ID == Result.Key
//   - Results appearing in both rankings get Source "hybrid"
//   - Results in only one ranking keep their original Source
//
// Supersede overlay (spec embeddings-superseded-respect):
//   - After fusion, every result whose Key appears in supersededBy has its
//     fused RRF score multiplied by supersededDeweight (when includeSuperseded
//     is false) and its Snippet prefixed with [SUPERSEDED → <slug>].
//   - The annotation is applied regardless of includeSuperseded — the marker
//     is the visibility cue; the multiplier is the rank effect.
//   - Annotation is idempotent: if the lexical sub-path already stamped the
//     marker, fuseRRF does not stamp it again.
//   - For non-spec corpora the overlay never matches (it only contains spec
//     slugs), so vector hits on knowledge/convention/code/event pass through
//     unchanged.
//   - The de-weight is applied exactly once per fused result regardless of
//     whether it came from lexical, vector, or both — the lexical sub-call
//     in retrieveHybrid runs with skipSupersedeDeweight=true, so its score
//     is not yet de-weighted when it arrives here.
func fuseRRF(lexical []Result, vector []embeddings.ScoredChunk, supersededBy map[string]string, includeSuperseded bool, k int, limit int) []Result {
	type fused struct {
		result   Result
		rrfScore float64
		inBoth   bool
	}

	// Index by a canonical key.
	byKey := make(map[string]*fused)

	// Score lexical results.
	for rank, r := range lexical {
		score := 1.0 / float64(k+rank+1) // rank is 0-indexed; +1 to make 1-indexed
		byKey[r.Key] = &fused{
			result:   r,
			rrfScore: score,
		}
	}

	// Score vector results and merge.
	for rank, vc := range vector {
		score := 1.0 / float64(k+rank+1)

		// Determine the matching key for this vector chunk.
		matchKey := vc.ID
		if vc.Corpus == "spec" {
			matchKey = vc.SourceID
		}

		if existing, ok := byKey[matchKey]; ok {
			// Already seen from lexical — add vector score.
			existing.rrfScore += score
			existing.inBoth = true
		} else {
			// Vector-only result. Build a Result from the chunk metadata.
			byKey[matchKey] = &fused{
				result: Result{
					Type:    corpusToNodeType(vc.Corpus),
					Key:     matchKey,
					Title:   matchKey,
					Snippet: truncateSnippet(vc.Section, 160),
					Score:   score,
					Source:  "vector",
				},
				rrfScore: score,
			}
		}
	}

	// Collect, apply the supersede overlay (de-weight + annotation), and
	// sort by RRF score descending. Overlay is applied to the fused score
	// so each result is penalized exactly once regardless of which sub-
	// path(s) surfaced it.
	results := make([]Result, 0, len(byKey))
	for _, f := range byKey {
		f.result.Score = f.rrfScore
		if f.inBoth {
			f.result.Source = "hybrid"
		}
		if by, ok := supersededBy[f.result.Key]; ok {
			if !includeSuperseded {
				f.result.Score *= supersededDeweight
			}
			f.result.Snippet = annotateSuperseded(f.result.Snippet, by)
		}
		results = append(results, f.result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// corpusToNodeType maps an embedding corpus name to the closest graph node
// type for display purposes.
func corpusToNodeType(corpus string) string {
	switch corpus {
	case "spec":
		return "Feature"
	case "knowledge":
		return "ContextDoc"
	case "convention":
		return "Convention"
	case "event":
		return "Note"
	case "code":
		return "Symbol"
	default:
		return "Unknown"
	}
}

// truncateSnippet trims s to maxLen characters, appending an ellipsis if truncated.
func truncateSnippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return s
}
