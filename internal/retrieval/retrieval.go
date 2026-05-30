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
}

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
}

// Retriever wraps the graph store, FTS5 index, and optional embedding model.
// Instantiate with New; call Close when done.
type Retriever struct {
	store    *graph.Store        // nil when graph DB is absent or failed to open
	fts      *index.DB
	embModel *embeddings.Model   // nil when embeddings unavailable
	embStore *embeddings.Storage // nil when embeddings unavailable
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
	r := &Retriever{fts: fts}
	if store, err := graph.Open(heroDir); err == nil {
		r.store = store
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

	rows, err := indexDB.Query(`
		SELECT ni.node_type, ni.key, ni.path,
		       snippet(fts_nodes, 0, '>>>', '<<<', '...', 24) AS title_snip,
		       snippet(fts_nodes, 1, '>>>', '<<<', '...', 32) AS body_snip,
		       fts_nodes.rank AS bm25_rank
		  FROM fts_nodes
		  JOIN node_index ni ON ni.rowid = fts_nodes.rowid
		 WHERE fts_nodes MATCH ?
		 ORDER BY fts_nodes.rank
		 LIMIT ?`,
		ftsQuery, limit*3, // over-fetch; we re-sort by boosted score
	)
	if err != nil {
		return nil, nil // graceful: fts_nodes may be empty/absent
	}
	defer rows.Close()

	type cand struct {
		nodeType  string
		key       string
		path      string
		titleSnip string
		bodySnip  string
		bm25Rank  float64
		score     float64
	}
	var cands []cand

	for rows.Next() {
		var c cand
		var rank sql.NullFloat64
		if err := rows.Scan(&c.nodeType, &c.key, &c.path, &c.titleSnip, &c.bodySnip, &rank); err != nil {
			continue
		}
		if rank.Valid {
			c.bm25Rank = rank.Float64
		}
		// FTS5 rank is negative (more negative = better). Score = -rank * typeBoost.
		c.score = -c.bm25Rank * typeBoost(c.nodeType)
		cands = append(cands, c)
	}

	if len(cands) == 0 {
		return nil, nil
	}

	sort.SliceStable(cands, func(i, j int) bool {
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
		results[i] = Result{
			Type:    c.nodeType,
			Key:     c.key,
			Title:   title,
			Snippet: snippet,
			Score:   c.score,
			Source:  "graph",
			Path:    c.path,
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
	rows, err := r.store.DB().Query(
		`SELECT id, type, key,
		        COALESCE(json_extract(props, '$.title'),   '') AS title,
		        COALESCE(json_extract(props, '$.body'),    '') AS body,
		        COALESCE(json_extract(props, '$.subject'), '') AS subject
		   FROM nodes
		  WHERE valid_to IS NULL
		    AND (lower(key) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.title'),   '')) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.body'),    '')) LIKE '%' || ? || '%'
		         OR lower(COALESCE(json_extract(props, '$.subject'), '')) LIKE '%' || ? || '%')
		  ORDER BY ingested_at DESC
		  LIMIT 200`,
		text, text, text, text,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type cand struct {
		id       int64
		nodeType string
		key      string
		title    string
		body     string
		score    float64
	}
	var cands []cand

	for rows.Next() {
		var c cand
		var subject string
		if err := rows.Scan(&c.id, &c.nodeType, &c.key, &c.title, &c.body, &subject); err != nil {
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
		hits, err = r.fts.ListFilteredScoped(specType, status, tag, since, subproject)
	case hasFilter:
		hits, err = r.fts.SearchFilteredScoped(q.Text, specType, status, tag, since, subproject)
	default:
		hits, err = r.fts.Search(q.Text)
	}
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}

	results := make([]Result, len(hits))
	for i, h := range hits {
		results[i] = Result{
			Type:      string(h.Type),
			Key:       h.Slug,
			Title:     h.Title,
			Snippet:   h.Snippet,
			Score:     fts5TypeBoost(string(h.Type)),
			Source:    "fts5",
			Status:    string(h.Status),
			ClaimedBy: h.ClaimedBy,
			Path:      h.Path,
		}
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// Hybrid search (Phase C — BM25 + vector via RRF)
// ---------------------------------------------------------------------------

// retrieveHybrid runs the normal lexical path and a vector similarity query,
// then fuses results via Reciprocal Rank Fusion.
func (r *Retriever) retrieveHybrid(q Query, limit int) ([]Result, error) {
	// Run the lexical path (graph/FTS5) with SemanticOK=false to avoid recursion.
	lexicalQ := q
	lexicalQ.SemanticOK = false
	lexical, err := r.Retrieve(lexicalQ)
	if err != nil {
		return nil, err
	}

	// Embed the query and retrieve similar chunks.
	queryVec := r.embModel.Embed(q.Text)
	vectorHits, err := r.embStore.QuerySimilar(queryVec, limit*2, nil)
	if err != nil {
		// Vector search failed — fall back to lexical-only.
		return lexical, nil
	}

	return fuseRRF(lexical, vectorHits, 60, limit), nil
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
func fuseRRF(lexical []Result, vector []embeddings.ScoredChunk, k int, limit int) []Result {
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

	// Collect and sort by RRF score descending.
	results := make([]Result, 0, len(byKey))
	for _, f := range byKey {
		f.result.Score = f.rrfScore
		if f.inBoth {
			f.result.Source = "hybrid"
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
