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

	// ChatInteractiveConnected reports whether at least one interactive
	// chat adapter is currently connected. Nil-safe: nil renders the
	// inline chat-input in its disabled state. Kept as a function so
	// this package stays free of chat / runner dependencies.
	ChatInteractiveConnected func() bool
}

// Register installs the Knowledge home on the shell router using the
// provided dependency bundle. The Knowledge home occupies the
// "knowledge" slug — the placeholder registered by
// shell.RegisterStubHomes must be dropped from stubs.go first by the
// caller (we do not double-register).
//
// The home root renders the Browse view; the Why / Staleness / Search /
// Recent / Write sub-views each register their own item route so the
// sub-nav anchors never 404. Stubbed views render the shared
// `coming-soon` shell card.
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
		Render: h.renderBrowse,
		Items: []shell.ItemRoute{
			{Pattern: "GET /knowledge/why", Render: h.renderWhy},
			{Pattern: "GET /knowledge/staleness", Render: h.renderStaleness},
			{Pattern: "GET /knowledge/search", Render: h.renderSearch},
			{Pattern: "GET /knowledge/recent", Render: h.renderRecent},
			{Pattern: "GET /knowledge/write", Render: h.renderWrite},
			// Wildcard slug — Go 1.22 ServeMux gives precedence to the
			// specifically-registered patterns above, so /knowledge/why
			// etc. still route to their own handlers; this only fires
			// on unrecognized slugs.
			{Pattern: "GET /knowledge/{slug}", Render: h.renderEntryDetail},
		},
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

// chatInputFor returns the inline chat-input config for the
// Knowledge home. Variant is "inline" (40px tall, ambient — never the
// primary affordance on a non-Now home). Per polish-v2 Fix 5, the
// empty-state notice is NOT paired here — that stays Now-only; this
// input renders as-is regardless of adapter state.
func (h *handler) chatInputFor(activeSlug string) shell.ChatInput {
	chips := []shell.ChatContextChip{{Kind: "page", Label: "page: /knowledge"}}
	if activeSlug != "" && activeSlug != "browse" {
		chips = append(chips, shell.ChatContextChip{Kind: "view", Label: "view: " + activeSlug})
	}
	in := shell.ChatInput{
		Variant:     "inline",
		Placeholder: "Ask Hero about knowledge…",
		Context:     chips,
	}
	if isChatDisabled(h.deps.ChatInteractiveConnected) {
		in.Disabled = true
		in.Placeholder = "Connect a chat adapter to enable"
		in.ConnectHref = "/settings/chat"
	}
	return in
}

// isChatDisabled returns true when no interactive chat adapter is
// available. Mirrors the helper in the other non-Now home packages.
// Kept as a func-arg so this package stays free of chat/runner deps.
func isChatDisabled(probe func() bool) bool {
	if probe == nil {
		return true
	}
	return !probe()
}

// renderHeroAndChat writes the page-hero followed by the inline chat-
// input fragment, in that order, into w. Centralizes the Fix-5
// placement contract: chat-input renders immediately below the hero
// on every Knowledge view.
func (h *handler) renderHeroAndChat(out io.Writer, hero shell.PageHero, activeSlug string) error {
	if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
		return err
	}
	return h.router.RenderFragment(out, "chat-input", h.chatInputFor(activeSlug))
}

// renderBrowse handles GET /knowledge — the default Browse view (corpus
// listing + facet filters).
func (h *handler) renderBrowse(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	corpus := data.LoadCorpus(data.CorpusInputs{HeroDir: h.deps.HeroDir})
	staleness := data.LoadStaleness(data.StalenessInputs{HeroDir: h.deps.HeroDir})
	newThisWeek := corpus.NewThisWeek
	if newThisWeek == 0 {
		newThisWeek = data.CountCorpusEventsLastWeek(h.deps.HeroDir)
	}

	hero := buildPageHero(h.deps, ed, corpus.TotalEntries, "")
	strip := buildMetricStrip(corpus, staleness, newThisWeek)
	subNav := buildSubNav(staleness, "browse")

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, "browse"); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "browse.html", corpus)
	}
	h.serve(w, req, content, subNav, "Knowledge · Hero")
}

// renderWhy handles GET /knowledge/why — the provenance chain view.
func (h *handler) renderWhy(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	corpus := data.LoadCorpus(data.CorpusInputs{HeroDir: h.deps.HeroDir})
	staleness := data.LoadStaleness(data.StalenessInputs{HeroDir: h.deps.HeroDir})
	why := data.LoadWhy(data.WhyInputs{HeroDir: h.deps.HeroDir})
	summary := data.LoadSummary(data.SummaryInputs{HeroDir: h.deps.HeroDir})
	neighbors := data.LoadNeighbors(data.NeighborsInputs{HeroDir: h.deps.HeroDir})
	newThisWeek := corpus.NewThisWeek
	if newThisWeek == 0 {
		newThisWeek = data.CountCorpusEventsLastWeek(h.deps.HeroDir)
	}

	hero := buildPageHero(h.deps, ed, corpus.TotalEntries, "Why")
	strip := buildMetricStrip(corpus, staleness, newThisWeek)
	subNav := buildSubNav(staleness, "why")

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, "why"); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		if err := h.tmpl.ExecuteTemplate(out, "provenance.html", why); err != nil {
			return err
		}
		if err := h.tmpl.ExecuteTemplate(out, "summary.html", summary); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "neighbors.html", neighbors)
	}
	h.serve(w, req, content, subNav, "Knowledge · Why · Hero")
}

// renderStaleness handles GET /knowledge/staleness.
func (h *handler) renderStaleness(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	corpus := data.LoadCorpus(data.CorpusInputs{HeroDir: h.deps.HeroDir})
	staleness := data.LoadStaleness(data.StalenessInputs{HeroDir: h.deps.HeroDir})
	newThisWeek := corpus.NewThisWeek
	if newThisWeek == 0 {
		newThisWeek = data.CountCorpusEventsLastWeek(h.deps.HeroDir)
	}

	hero := buildPageHero(h.deps, ed, corpus.TotalEntries, "Staleness")
	strip := buildMetricStrip(corpus, staleness, newThisWeek)
	subNav := buildSubNav(staleness, "staleness")

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, "staleness"); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "staleness.html", staleness)
	}
	h.serve(w, req, content, subNav, "Knowledge · Staleness · Hero")
}

// renderSearch / renderRecent / renderWrite — substrate-pending stubs.

func (h *handler) renderSearch(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "search", "Search",
		"Search will route into the unified retrieval layer once that pipeline is exposed via HTTP.")
}

func (h *handler) renderRecent(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "recent", "Recent",
		"Recent knowledge captures are available in .hero/knowledge/ — the UI listing lands in a follow-up.")
}

func (h *handler) renderWrite(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "write", "Write",
		"Writer surface — capture via /note from chat for now.")
}

// renderEntryDetail handles GET /knowledge/{slug}. Loads the entry
// from .hero/knowledge/ and renders its title, metadata, rendered
// markdown body, and relations footer. Returns 404 (the shell's
// default not-found page) when the slug doesn't resolve.
func (h *handler) renderEntryDetail(w http.ResponseWriter, req *http.Request) {
	slug := req.PathValue("slug")
	entry := data.LoadEntry(h.deps.HeroDir, slug)
	if entry == nil {
		http.NotFound(w, req)
		return
	}

	ed := edition.Resolve()
	corpus := data.LoadCorpus(data.CorpusInputs{HeroDir: h.deps.HeroDir})
	staleness := data.LoadStaleness(data.StalenessInputs{HeroDir: h.deps.HeroDir})
	newThisWeek := corpus.NewThisWeek
	if newThisWeek == 0 {
		newThisWeek = data.CountCorpusEventsLastWeek(h.deps.HeroDir)
	}

	hero := buildEntryPageHero(h.deps, ed, entry)
	strip := buildMetricStrip(corpus, staleness, newThisWeek)
	// Per polish-v3 Fix 2: highlight the Browse tab (the entry was
	// reached from Browse) so the detail view shows where it lives.
	subNav := buildSubNav(staleness, "browse")
	crumb := &shell.PageBreadcrumb{Crumbs: []shell.BreadcrumbCrumb{
		{Label: "Knowledge", Href: "/knowledge"},
		{Label: entry.Title, Current: true},
	}}

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, ""); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "detail.html", entry)
	}
	h.serveDetail(w, req, content, subNav, crumb, "Knowledge · "+entry.Title+" · Hero")
}

// buildEntryPageHero composes the per-entry page-hero. Eyebrow
// repeats the canonical breadcrumb; title is the entry title; subhead
// lists kind / status / created / last-touched.
func buildEntryPageHero(deps Deps, ed edition.Edition, e *data.Entry) shell.PageHero {
	eyebrow := fmt.Sprintf("hero · %s · knowledge", firstNonEmpty(deps.Branch, "main"))

	var subParts []string
	if e.Kind != "" {
		subParts = append(subParts, template.HTMLEscapeString(e.Kind))
	}
	if e.Type != "" && e.Type != e.Kind {
		subParts = append(subParts, template.HTMLEscapeString(e.Type))
	}
	if e.Status != "" {
		subParts = append(subParts, template.HTMLEscapeString(e.Status))
	}
	if e.CreatedPretty != "" {
		subParts = append(subParts, "created "+template.HTMLEscapeString(e.CreatedPretty))
	}
	if e.UpdatedPretty != "" {
		subParts = append(subParts, "last touched "+template.HTMLEscapeString(e.UpdatedPretty))
	}
	subhead := strings.Join(subParts, `<span class="dot-sep">·</span>`)

	_ = ed
	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   e.Title,
		Subhead: template.HTML(subhead),
		Actions: []shell.PageHeroAction{
			{Kind: "ghost", Label: "Back to Browse", Href: "/knowledge"},
			{Kind: "chip", Label: "knowledge", Chip: "knowledge"},
		},
	}
}

// renderStub renders the home chrome + sub-nav with the standard coming-
// soon shell card in the body.
func (h *handler) renderStub(w http.ResponseWriter, req *http.Request, slug, view, note string) {
	ed := edition.Resolve()
	corpus := data.LoadCorpus(data.CorpusInputs{HeroDir: h.deps.HeroDir})
	staleness := data.LoadStaleness(data.StalenessInputs{HeroDir: h.deps.HeroDir})
	newThisWeek := corpus.NewThisWeek
	if newThisWeek == 0 {
		newThisWeek = data.CountCorpusEventsLastWeek(h.deps.HeroDir)
	}

	hero := buildPageHero(h.deps, ed, corpus.TotalEntries, view)
	strip := buildMetricStrip(corpus, staleness, newThisWeek)
	subNav := buildSubNav(staleness, slug)

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, slug); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.router.RenderFragment(out, "coming-soon", stubData{
			Home: "knowledge", Slug: slug, View: view, Note: note,
		})
	}
	h.serve(w, req, content, subNav, "Knowledge · "+view+" · Hero")
}

// serve is the thin compositor that delegates to the shell. Title is
// the browser <title>; per polish-v3 Fix 5, sub-routes include the
// sub-view name (e.g. "Knowledge · Staleness · Hero").
func (h *handler) serve(w http.ResponseWriter, req *http.Request, content func(io.Writer) error, subNav *shell.SubNav, title string) {
	h.serveDetail(w, req, content, subNav, nil, title)
}

// serveDetail is the full compositor — same as serve plus an optional
// breadcrumb row above the page hero. Used by detail routes.
func (h *handler) serveDetail(w http.ResponseWriter, req *http.Request, content func(io.Writer) error, subNav *shell.SubNav, crumb *shell.PageBreadcrumb, title string) {
	page := shell.Page{
		ActiveHome: "knowledge",
		PageTitle:  title,
		SubNav:     subNav,
		Breadcrumb: crumb,
		Content:    content,
		HeadExtra:  template.HTML(knowledgeStyles + knowledgeScript),
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "knowledge: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// stubData is the input bundle for the shared `coming-soon` template.
type stubData struct {
	Home string
	Slug string
	View string
	Note string
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

// buildPageHero composes the page-hero data block from corpus state.
// subView is the active sub-view label (e.g. "Staleness", "Why"); when
// non-empty it's appended to the page-hero title as `Knowledge · <Sub>`
// per polish-v3 Fix 5.
func buildPageHero(deps Deps, ed edition.Edition, totalEntries int, subView string) shell.PageHero {
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

	title := "Knowledge"
	if subView != "" {
		title = "Knowledge · " + subView
	}

	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   title,
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
// rate · new this week.
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

// buildSubNav returns the six-tab Knowledge sub-nav row. activeSlug
// picks the active tab; valid slugs are browse|search|why|staleness|
// recent|write.
func buildSubNav(s data.Staleness, activeSlug string) *shell.SubNav {
	badge := ""
	if s.Available && s.Total > 0 {
		badge = strconv.Itoa(s.Total)
	}
	return &shell.SubNav{
		Tabs: []shell.SubNavTab{
			{Label: "Browse", Href: "/knowledge", Active: activeSlug == "browse"},
			{Label: "Search", Href: "/knowledge/search", Active: activeSlug == "search"},
			{Label: "Why", Href: "/knowledge/why", Active: activeSlug == "why"},
			{Label: "Staleness", Href: "/knowledge/staleness", Active: activeSlug == "staleness", Badge: badge},
			{Label: "Recent", Href: "/knowledge/recent", Active: activeSlug == "recent"},
			{Label: "Write", Href: "/knowledge/write", Active: activeSlug == "write"},
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
