// Package knowledge hosts the Knowledge home — the corpus surface at
// GET /knowledge. It composes the shell's page chrome (top nav, sub-nav
// row, page-hero fragment, tabbed-metric-strip fragment, footer) with
// Knowledge-specific section partials for: browse (default), provenance
// chain (the Why view), plain-English summary, suggested neighbors, and
// worth-re-checking (staleness). The handler reads live state via a
// small set of data-fetchers under ./data and SSE updates are wired by
// the sibling internal/serve/api/knowledge.go.
//
// Per hero-knowledge-home, this package owns NO chat dispatch and NO
// agent-runner internals. Live traversal and contradiction detection
// belong to consumed specs (traversal-queries, knowledge-contradiction-
// detection); this home renders what those substrates produce.
package knowledge

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/pages/knowledge/data"
	"github.com/hero-engine/hero/internal/serve/shell"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Deps is the small, injectable bundle of state the Knowledge handler
// reads. Mirrors the Now home's Deps shape so the server wiring stays
// symmetric.
type Deps struct {
	// ProjectRoot is the absolute path to the project being served.
	ProjectRoot string

	// HeroDir is the absolute path to .hero/ inside ProjectRoot. Empty
	// disables data fetchers that read knowledge entries / events.
	HeroDir string

	// Workspace is the human label shown in the eyebrow / footer chrome.
	Workspace string

	// Branch is the current git branch ("" when not in a git checkout).
	Branch string

	// UserName is the display name used in the eyebrow / personalization
	// strings.
	UserName string

	// RegisterFragment lets the api package mount fragment endpoints
	// against the same template set the handler uses, so SSE clients
	// fetch HTML identical to the initial render. Nil-safe.
	RegisterFragment func(section string, render func(w http.ResponseWriter, r *http.Request))
}

// Register installs the Knowledge home on the shell router using the
// provided dependency bundle. The Knowledge home occupies the
// "knowledge" slug — the placeholder registered by
// shell.RegisterStubHomes must be dropped from stubs.go first by the
// caller (we do not double-register).
func Register(r *shell.Router, deps Deps) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("knowledge: load templates: %w", err)
	}
	h := &handler{
		router: r,
		tmpl:   tmpl,
		deps:   deps,
	}
	return r.RegisterHome(shell.Home{
		Slug:   "knowledge",
		Label:  "Knowledge",
		Href:   "/knowledge",
		Render: h.handle,
	})
}

// SectionFragment returns the rendered HTML for a single Knowledge
// section. Exposed for use by the SSE fragment endpoints in
// internal/serve/api. The output is exactly what the initial /knowledge
// render produces for that section, so client-side replacement
// preserves layout.
func SectionFragment(deps Deps, section string) ([]byte, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	h := &handler{tmpl: tmpl, deps: deps}
	return h.renderSection(section)
}

// loadTemplates parses every .html under the embedded templates/
// directory into a single template set.
func loadTemplates() (*template.Template, error) {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("templates subdir: %w", err)
	}
	return template.ParseFS(sub, "*.html")
}

type handler struct {
	router *shell.Router
	tmpl   *template.Template
	deps   Deps
}

func (h *handler) handle(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	page := h.buildPage(req, ed)
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "knowledge: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// buildPage assembles the Page envelope passed to shell.Router.RenderPage.
// All data-fetcher invocations live here so the handler stays a thin
// composition point.
func (h *handler) buildPage(req *http.Request, ed edition.Edition) shell.Page {
	corpus := data.LoadCorpus(data.CorpusInputs{HeroDir: h.deps.HeroDir})
	staleness := data.LoadStaleness(data.StalenessInputs{HeroDir: h.deps.HeroDir})
	why := data.LoadWhy(data.WhyInputs{HeroDir: h.deps.HeroDir})
	summary := data.LoadSummary(data.SummaryInputs{HeroDir: h.deps.HeroDir})
	neighbors := data.LoadNeighbors(data.NeighborsInputs{HeroDir: h.deps.HeroDir})

	// Capture-event count for the "new this week" tile.
	newThisWeek := corpus.NewThisWeek
	if newThisWeek == 0 {
		newThisWeek = data.CountCorpusEventsLastWeek(h.deps.HeroDir)
	}

	hero := buildPageHero(h.deps, ed, corpus.TotalEntries)
	strip := buildMetricStrip(corpus, staleness, newThisWeek)
	subNav := buildSubNav(staleness)

	pd := pageData{
		Corpus:    corpus,
		Why:       why,
		Summary:   summary,
		Neighbors: neighbors,
		Staleness: staleness,
	}

	content := func(out io.Writer) error {
		if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "page.html", pd)
	}

	return shell.Page{
		ActiveHome: "knowledge",
		PageTitle:  "Knowledge · Hero",
		SubNav:     subNav,
		Content:    content,
		HeadExtra:  template.HTML(knowledgeStyles + knowledgeScript),
	}
}

// renderSection produces the standalone HTML fragment for one section,
// for the SSE fragment-replacement endpoints. Returns an error for
// unknown section names so callers can 404 cleanly.
func (h *handler) renderSection(section string) ([]byte, error) {
	var tplName string
	var payload any
	switch section {
	case "browse":
		tplName = "browse.html"
		payload = data.LoadCorpus(data.CorpusInputs{HeroDir: h.deps.HeroDir})
	case "provenance":
		tplName = "provenance.html"
		payload = data.LoadWhy(data.WhyInputs{HeroDir: h.deps.HeroDir})
	case "summary":
		tplName = "summary.html"
		payload = data.LoadSummary(data.SummaryInputs{HeroDir: h.deps.HeroDir})
	case "neighbors":
		tplName = "neighbors.html"
		payload = data.LoadNeighbors(data.NeighborsInputs{HeroDir: h.deps.HeroDir})
	case "staleness":
		tplName = "staleness.html"
		payload = data.LoadStaleness(data.StalenessInputs{HeroDir: h.deps.HeroDir})
	default:
		return nil, fmt.Errorf("unknown section %q", section)
	}
	var buf strings.Builder
	if err := h.tmpl.ExecuteTemplate(&buf, tplName, payload); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// pageData is the outer-template input.
type pageData struct {
	Corpus    data.Corpus
	Why       data.Why
	Summary   data.Summary
	Neighbors data.Neighbors
	Staleness data.Staleness
}

// buildPageHero composes the page-hero data block from corpus state.
func buildPageHero(deps Deps, ed edition.Edition, totalEntries int) shell.PageHero {
	eyebrow := fmt.Sprintf("hero · %s · knowledge", firstNonEmpty(deps.Branch, "main"))
	subhead := ""
	switch totalEntries {
	case 0:
		subhead = "No corpus entries yet — capture one with <code>/note</code> or open the writer."
	case 1:
		subhead = "<strong>1 entry</strong> in the corpus"
	default:
		subhead = fmt.Sprintf("<strong>%d entries</strong> in the corpus", totalEntries)
	}

	editionLabel := "Solo"
	if ed != edition.Local {
		editionLabel = strings.Title(string(ed)) //nolint:staticcheck // single-byte upper is fine here
	}

	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   "Knowledge",
		Subhead: template.HTML(subhead),
		Actions: []shell.PageHeroAction{
			{Kind: "primary", Label: "Write entry", Href: "/knowledge/write"},
			{Kind: "ghost", Label: "Why does this exist?", Href: "/knowledge/why"},
			{Kind: "chip", Label: editionLabel, Chip: editionLabel},
		},
	}
}

// buildMetricStrip composes the orientation strip matching the four
// quiet tiles from the mockup: corpus entries · stale flags · reuse
// rate · new this week. Today the strip has a single tab — the
// shell's tabbed-metric-strip fragment still requires the slice
// envelope.
func buildMetricStrip(corpus data.Corpus, staleness data.Staleness, newThisWeek int) shell.MetricStrip {
	stale := template.HTML("—")
	if staleness.Available {
		stale = template.HTML(strconv.Itoa(staleness.Total))
	}
	staleAccent := ""
	if staleness.Available && staleness.Total > 0 {
		staleAccent = "warn"
	}
	return shell.MetricStrip{
		Tabs: []shell.MetricTab{
			{
				Slug: "corpus", Label: "Corpus", Active: true,
				Tiles: []shell.MetricTile{
					{Value: template.HTML(strconv.Itoa(corpus.TotalEntries)), Label: "corpus entries"},
					{Value: stale, Label: "stale flags", Accent: staleAccent},
					{Value: template.HTML("—"), Label: "reuse rate · 7d"},
					{Value: template.HTML(strconv.Itoa(newThisWeek)), Label: "new this week"},
				},
			},
		},
	}
}

// buildSubNav returns the six-tab Knowledge sub-nav row matching the
// mockup. Browse is the active default. When the staleness pipeline is
// live the `Staleness` tab carries a count badge.
func buildSubNav(s data.Staleness) *shell.SubNav {
	badge := ""
	if s.Available && s.Total > 0 {
		badge = strconv.Itoa(s.Total)
	}
	return &shell.SubNav{
		Tabs: []shell.SubNavTab{
			{Label: "Browse", Href: "/knowledge", Active: true},
			{Label: "Search", Href: "/knowledge/search"},
			{Label: "Why", Href: "/knowledge/why"},
			{Label: "Staleness", Href: "/knowledge/staleness", Badge: badge},
			{Label: "Recent", Href: "/knowledge/recent"},
			{Label: "Write", Href: "/knowledge/write"},
		},
	}
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
