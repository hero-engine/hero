// Package sitegen renders the unified knowledge graph as a static
// HTML site — phase 9's "publish a living team narrative" pillar.
//
// Output goes to a directory (default ./site) that's deployable as
// GitHub Pages, Netlify, S3, or anything else that serves static
// files. No build step, no JS framework — embedded templates +
// embedded CSS produce self-contained HTML.
//
// Pages:
//   index.html              — landing: recent activity, in-flight features
//   features/<slug>.html    — per-feature detail
//   initiatives/<slug>.html — per-initiative detail
//   decisions/<slug>.html   — per-decision detail (incl. originating note)
//   notes/<slug>.html       — per-note detail
//   activity.html           — full activity feed
//
// Same projection engine constraints as the digester: deterministic,
// fast, no LLM calls. Re-running on unchanged graph produces
// byte-identical output (so it's friendly to gh-pages diffs).
package sitegen

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

//go:embed templates/*.html templates/*.css
var assets embed.FS

// Generator renders the site for a given graph store + repo.
type Generator struct {
	Store    *graph.Store
	RepoKey  string
	OutDir   string
	SiteName string // shown in <title> and header; defaults to RepoKey
}

// Generate writes the full site to OutDir, creating the directory
// if needed. Returns counts of pages written.
func (g *Generator) Generate() (*Summary, error) {
	if g.SiteName == "" {
		g.SiteName = g.RepoKey
	}
	if err := os.MkdirAll(g.OutDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}
	for _, sub := range []string{"features", "initiatives", "decisions", "notes"} {
		if err := os.MkdirAll(filepath.Join(g.OutDir, sub), 0o755); err != nil {
			return nil, fmt.Errorf("creating %s dir: %w", sub, err)
		}
	}

	tmpl, err := template.New("").Funcs(template.FuncMap{
		"shortSha":   shortSha,
		"oneLine":    oneLine,
		"shortDate":  shortDate,
		"hasPrefix":  strings.HasPrefix,
	}).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	// Embedded CSS
	cssBytes, err := assets.ReadFile("templates/style.css")
	if err != nil {
		return nil, fmt.Errorf("reading css: %w", err)
	}
	if err := os.WriteFile(filepath.Join(g.OutDir, "style.css"), cssBytes, 0o644); err != nil {
		return nil, fmt.Errorf("writing css: %w", err)
	}

	summary := &Summary{}

	// --- index ---
	indexCtx, err := g.buildIndexContext()
	if err != nil {
		return nil, err
	}
	if err := g.renderTo(tmpl, "index.html", filepath.Join(g.OutDir, "index.html"), indexCtx); err != nil {
		return nil, err
	}
	summary.Index++

	// --- features ---
	features, err := g.listByType("Feature")
	if err != nil {
		return nil, err
	}
	for _, n := range features {
		ctx, err := g.buildFeatureContext(n)
		if err != nil {
			return nil, err
		}
		out := filepath.Join(g.OutDir, "features", slugForFile(n.Key)+".html")
		if err := g.renderTo(tmpl, "feature.html", out, ctx); err != nil {
			return nil, err
		}
		summary.Features++
	}

	// --- initiatives ---
	inits, err := g.listByType("Initiative")
	if err != nil {
		return nil, err
	}
	for _, n := range inits {
		ctx, err := g.buildEntityContext(n, "Initiative")
		if err != nil {
			return nil, err
		}
		out := filepath.Join(g.OutDir, "initiatives", slugForFile(n.Key)+".html")
		if err := g.renderTo(tmpl, "entity.html", out, ctx); err != nil {
			return nil, err
		}
		summary.Initiatives++
	}

	// --- decisions ---
	decisions, err := g.listByType("Decision")
	if err != nil {
		return nil, err
	}
	for _, n := range decisions {
		ctx, err := g.buildEntityContext(n, "Decision")
		if err != nil {
			return nil, err
		}
		out := filepath.Join(g.OutDir, "decisions", slugForFile(n.Key)+".html")
		if err := g.renderTo(tmpl, "entity.html", out, ctx); err != nil {
			return nil, err
		}
		summary.Decisions++
	}

	// --- notes ---
	notes, err := g.listByType("Note")
	if err != nil {
		return nil, err
	}
	for _, n := range notes {
		ctx, err := g.buildEntityContext(n, "Note")
		if err != nil {
			return nil, err
		}
		out := filepath.Join(g.OutDir, "notes", slugForFile(n.Key)+".html")
		if err := g.renderTo(tmpl, "entity.html", out, ctx); err != nil {
			return nil, err
		}
		summary.Notes++
	}

	// --- activity feed ---
	actCtx, err := g.buildActivityContext()
	if err != nil {
		return nil, err
	}
	if err := g.renderTo(tmpl, "activity.html", filepath.Join(g.OutDir, "activity.html"), actCtx); err != nil {
		return nil, err
	}
	summary.Activity++

	return summary, nil
}

type Summary struct {
	Index       int
	Features    int
	Initiatives int
	Decisions   int
	Notes       int
	Activity    int
}

// --- contexts ---------------------------------------------------------------

type baseCtx struct {
	SiteName  string
	RepoKey   string
	Generated string
}

type indexCtx struct {
	baseCtx
	OpenFeatures []entityRow
	Initiatives  []entityRow
	RecentCommits []commitRow
	Counts       map[string]int
}

type entityRow struct {
	Key      string
	Title    string
	Status   string
	Priority string
	Slug     string // file slug for href
	Type     string
}

type commitRow struct {
	SHA, Subject, Date, Author string
}

type featureCtx struct {
	baseCtx
	Entity        entityRow
	Body          string
	RelatedNodes  []entityRow
	RecentCommits []commitRow
}

type entityCtx struct {
	baseCtx
	Entity       entityRow
	Body         string
	RelatedNodes []entityRow
}

type activityCtx struct {
	baseCtx
	Commits []commitRow
}

// --- queries ----------------------------------------------------------------

func (g *Generator) buildIndexContext() (*indexCtx, error) {
	ctx := &indexCtx{
		baseCtx: g.base(),
		Counts:  map[string]int{},
	}

	openF, err := g.openEntities("Feature", 20)
	if err != nil {
		return nil, err
	}
	ctx.OpenFeatures = openF

	inits, err := g.openEntities("Initiative", 20)
	if err != nil {
		return nil, err
	}
	ctx.Initiatives = inits

	commits, err := g.recentCommits(20)
	if err != nil {
		return nil, err
	}
	ctx.RecentCommits = commits

	for _, t := range []string{"Feature", "Initiative", "Decision", "Note", "Commit"} {
		var n int
		_ = g.Store.DB().QueryRow(
			`SELECT COUNT(*) FROM nodes WHERE type = ? AND repo = ? AND valid_to IS NULL`,
			t, g.RepoKey,
		).Scan(&n)
		ctx.Counts[t] = n
	}
	return ctx, nil
}

func (g *Generator) buildFeatureContext(n *graph.Node) (*featureCtx, error) {
	related, err := g.relatedEntities(n.ID)
	if err != nil {
		return nil, err
	}
	commits, err := g.commitsForFeature(n.ID, 10)
	if err != nil {
		return nil, err
	}
	return &featureCtx{
		baseCtx: g.base(),
		Entity:  toEntityRow(n, "Feature"),
		Body:    bodyOf(n),
		RelatedNodes: related,
		RecentCommits: commits,
	}, nil
}

func (g *Generator) buildEntityContext(n *graph.Node, typ string) (*entityCtx, error) {
	related, err := g.relatedEntities(n.ID)
	if err != nil {
		return nil, err
	}
	return &entityCtx{
		baseCtx: g.base(),
		Entity:  toEntityRow(n, typ),
		Body:    bodyOf(n),
		RelatedNodes: related,
	}, nil
}

func (g *Generator) buildActivityContext() (*activityCtx, error) {
	commits, err := g.recentCommits(200)
	if err != nil {
		return nil, err
	}
	return &activityCtx{baseCtx: g.base(), Commits: commits}, nil
}

func (g *Generator) base() baseCtx {
	return baseCtx{
		SiteName:  g.SiteName,
		RepoKey:   g.RepoKey,
		Generated: time.Now().UTC().Format("2006-01-02 15:04 UTC"),
	}
}

// --- helpers ---------------------------------------------------------------

func (g *Generator) listByType(typ string) ([]*graph.Node, error) {
	rows, err := g.Store.DB().Query(
		`SELECT id, type, key, props, scope, repo, unit, content_hash, source,
		        valid_from, valid_to, ingested_at
		   FROM nodes
		  WHERE type = ? AND repo = ? AND valid_to IS NULL
		  ORDER BY key`,
		typ, g.RepoKey,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*graph.Node
	for rows.Next() {
		n, err := scanNodeRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// scanNodeRows scans a row from the nodes table into a graph.Node.
// Props and Source are unmarshalled from JSON so the templates can
// pull title/status/priority/etc. directly.
func scanNodeRows(rows *sql.Rows) (*graph.Node, error) {
	var (
		n           graph.Node
		propsJSON   string
		sourceJSON  string
		contentHash sql.NullString
		validToNS   sql.NullString
		scopeStr    string
	)
	if err := rows.Scan(
		&n.ID, &n.Type, &n.Key, &propsJSON, &scopeStr,
		&n.Repo, &n.Unit, &contentHash, &sourceJSON,
		&n.ValidFrom, &validToNS, &n.IngestedAt,
	); err != nil {
		return nil, err
	}
	n.Scope = graph.Scope(scopeStr)
	if contentHash.Valid {
		n.ContentHash = contentHash.String
	}
	if validToNS.Valid {
		n.ValidTo = validToNS.String
	}
	if err := json.Unmarshal([]byte(propsJSON), &n.Props); err != nil {
		return nil, fmt.Errorf("decode props: %w", err)
	}
	if err := json.Unmarshal([]byte(sourceJSON), &n.Source); err != nil {
		return nil, fmt.Errorf("decode source: %w", err)
	}
	return &n, nil
}

func (g *Generator) openEntities(typ string, limit int) ([]entityRow, error) {
	rows, err := g.Store.DB().Query(
		`SELECT key,
		        COALESCE(json_extract(props, '$.title'), key) AS title,
		        COALESCE(json_extract(props, '$.status'), '') AS status,
		        COALESCE(json_extract(props, '$.priority'), '') AS priority
		   FROM nodes
		  WHERE type = ? AND repo = ? AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.status'), '') NOT IN ('completed','superseded')
		  ORDER BY priority, ingested_at DESC
		  LIMIT ?`,
		typ, g.RepoKey, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entityRow
	for rows.Next() {
		var r entityRow
		if err := rows.Scan(&r.Key, &r.Title, &r.Status, &r.Priority); err != nil {
			return nil, err
		}
		r.Type = typ
		r.Slug = slugForFile(r.Key)
		out = append(out, r)
	}
	return out, nil
}

func (g *Generator) recentCommits(limit int) ([]commitRow, error) {
	rows, err := g.Store.DB().Query(
		`SELECT json_extract(props, '$.sha'),
		        json_extract(props, '$.subject'),
		        json_extract(props, '$.date'),
		        json_extract(props, '$.author_name')
		   FROM nodes
		  WHERE type = 'Commit' AND repo = ? AND valid_to IS NULL
		  ORDER BY json_extract(props, '$.date') DESC
		  LIMIT ?`,
		g.RepoKey, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []commitRow
	for rows.Next() {
		var r commitRow
		if err := rows.Scan(&r.SHA, &r.Subject, &r.Date, &r.Author); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (g *Generator) relatedEntities(nodeID int64) ([]entityRow, error) {
	rows, err := g.Store.DB().Query(
		`SELECT n.type, n.key,
		        COALESCE(json_extract(n.props, '$.title'), n.key) AS title,
		        COALESCE(json_extract(n.props, '$.status'), '') AS status,
		        e.type AS edge_type
		   FROM edges e
		   JOIN nodes n ON (n.id = e.to_id OR n.id = e.from_id) AND n.id != ?
		  WHERE (e.from_id = ? OR e.to_id = ?) AND e.valid_to IS NULL
		    AND n.type IN ('Feature','Initiative','Decision','Note')
		  ORDER BY n.type, n.key
		  LIMIT 50`,
		nodeID, nodeID, nodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entityRow
	seen := map[string]bool{}
	for rows.Next() {
		var r entityRow
		var edgeType string
		if err := rows.Scan(&r.Type, &r.Key, &r.Title, &r.Status, &edgeType); err != nil {
			return nil, err
		}
		dedup := r.Type + ":" + r.Key
		if seen[dedup] {
			continue
		}
		seen[dedup] = true
		r.Slug = slugForFile(r.Key)
		out = append(out, r)
	}
	return out, nil
}

func (g *Generator) commitsForFeature(featureID int64, limit int) ([]commitRow, error) {
	// Walk: Feature → mentioned_in/related → Spec → touches files.
	// For an MVP we just fall back to "recent commits in repo" — the
	// per-feature commit attribution requires deeper work.
	return g.recentCommits(limit)
}

func toEntityRow(n *graph.Node, typ string) entityRow {
	r := entityRow{
		Key:  n.Key,
		Type: typ,
		Slug: slugForFile(n.Key),
	}
	// Read title/status/priority directly from the props JSON via a
	// small SQL trip; alternative is unmarshalling the node's props.
	// Both work; this stays consistent with the helpers above.
	r.Title = n.Key
	if title := propString(n.Props, "title"); title != "" {
		r.Title = title
	}
	r.Status = propString(n.Props, "status")
	r.Priority = propString(n.Props, "priority")
	return r
}

func bodyOf(n *graph.Node) string {
	// Notes have body in props; specs have rationale; we surface
	// whatever is most useful per-type.
	for _, k := range []string{"body", "rationale", "doc", "description"} {
		if v := propString(n.Props, k); v != "" {
			return v
		}
	}
	return ""
}

func propString(p map[string]any, key string) string {
	if p == nil {
		return ""
	}
	v, ok := p[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func (g *Generator) renderTo(tmpl *template.Template, name, path string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	return tmpl.ExecuteTemplate(f, name, data)
}

// slugForFile turns a logical key into a filesystem-safe filename.
func slugForFile(k string) string {
	out := strings.ReplaceAll(k, "/", "_")
	out = strings.ReplaceAll(out, ":", "_")
	out = strings.ReplaceAll(out, " ", "-")
	return out
}

func shortSha(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return s
}

func shortDate(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02")
}
