// Package work hosts the Work home — the spec-and-delivery surface at
// GET /work. It composes the shell's page chrome (top nav, page-hero
// fragment, tabbed-metric-strip fragment, footer) with Work-specific
// section partials: view toolbar, Now/Next/Later roadmap centerpiece,
// blocked list, and recently-shipped timeline. The handler reads live
// state via a small set of data-fetchers under ./data; SSE updates
// are wired by the sibling internal/serve/api/work.go.
//
// Per hero-work-home, this package owns NO chat dispatch and NO agent
// runner internals; the import_test guards both.
package work

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
	"github.com/hero-engine/hero/internal/serve/pages/work/data"
	"github.com/hero-engine/hero/internal/serve/shell"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Deps is the small, injectable bundle of state the Work handler reads.
type Deps struct {
	ProjectRoot string
	HeroDir     string
	Workspace   string
	Branch      string
	UserName    string
	// ChatInteractiveConnected reports whether at least one interactive
	// chat adapter is currently connected. The handler uses this to
	// flip the inline chat-input into its disabled state. Nil-safe:
	// nil is treated as "no adapter" so the input renders disabled.
	//
	// Kept as a function rather than importing chat.* so this package
	// stays free of chat / runner dependencies (see import_test).
	ChatInteractiveConnected func() bool
}

// Register installs the Work home on the shell router using the
// provided dependency bundle. The Work home occupies the "work" slug.
// The Horizons view is the default; Kanban / Graph / Blocked each
// register their own item route so the view-toolbar links never 404.
func Register(r *shell.Router, deps Deps) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("work: load templates: %w", err)
	}

	h := &handler{
		router: r,
		tmpl:   tmpl,
		deps:   deps,
	}
	return r.RegisterHome(shell.Home{
		Slug:   "work",
		Label:  "Work",
		Href:   "/work",
		Render: h.handle,
		Items: []shell.ItemRoute{
			{Pattern: "GET /work/kanban", Render: h.renderKanban},
			{Pattern: "GET /work/graph", Render: h.renderGraph},
			{Pattern: "GET /work/blocked", Render: h.renderBlocked},
			// Per-spec detail route. `/work/spec/{slug}` is a separate
			// branch from the view-toolbar siblings above so the slug
			// wildcard can't collide with kanban/graph/blocked.
			{Pattern: "GET /work/spec/{slug}", Render: h.renderSpecDetail},
		},
	})
}

// SectionFragment returns the rendered HTML for a single Work section.
// view, when non-empty and the section is "toolbar", selects which view-
// tab to mark active in the rendered fragment. Valid values mirror
// toolbarData.Active: "horizons" | "kanban" | "graph" | "blocked".
// Empty falls back to "horizons" (the home root view).
func SectionFragment(deps Deps, section, view string) ([]byte, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	h := &handler{tmpl: tmpl, deps: deps}
	return h.renderSection(section, view)
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

// chatInputFor returns the inline chat-input config for the Work
// home. Variant is "inline" (40px tall, ambient — never the primary
// affordance on a non-Now home). Per polish-v2 Fix 5, the empty-state
// notice is NOT paired here — that stays Now-only.
func (h *handler) chatInputFor(activeSlug string) shell.ChatInput {
	chips := []shell.ChatContextChip{{Kind: "page", Label: "page: /work"}}
	if activeSlug != "" {
		chips = append(chips, shell.ChatContextChip{Kind: "view", Label: "view: " + activeSlug})
	}
	in := shell.ChatInput{
		Variant:     "inline",
		Placeholder: "Ask Hero about work…",
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
// available — either the probe is nil (treated as "no adapter") or
// the probe returns false. The four non-Now homes use this to flip
// the inline chat-input to its disabled state without importing
// chat / runner internals (see import_test).
func isChatDisabled(probe func() bool) bool {
	if probe == nil {
		return true
	}
	return !probe()
}

// renderHeroAndChat writes the page-hero followed by the inline chat-
// input fragment, in that order, into w. Centralizes the Fix-5
// placement contract: chat-input renders immediately below the hero
// on every Work view.
func (h *handler) renderHeroAndChat(out io.Writer, hero shell.PageHero, activeSlug string) error {
	if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
		return err
	}
	return h.router.RenderFragment(out, "chat-input", h.chatInputFor(activeSlug))
}

func (h *handler) handle(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	page := h.buildPage(req, ed)

	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "work: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// buildPage assembles the Page envelope for the default Horizons view.
// Reads the query string for filter + pagination state. Per v4 Fix 1,
// the root view also renders the view-toolbar with Horizons active so
// the toolbar chrome is consistent across /work, /work/kanban,
// /work/graph, and /work/blocked.
func (h *handler) buildPage(req *http.Request, ed edition.Edition) shell.Page {
	q := req.URL.Query()
	rmIn := data.RoadmapInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		TypeFilter:  q.Get("type"),
		AgeFilter:   q.Get("age"),
		ShowAll:     q.Get("all") == "1",
		Page:        parsePage(q.Get("page")),
	}

	counts := data.LoadCounts(data.CountsInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
	})
	roadmap := data.LoadRoadmap(rmIn)
	blocked := data.LoadBlocked(data.BlockedInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
	})
	shipped := data.LoadRecentlyShipped(data.ShippedInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
	})
	metrics := data.LoadMetrics(data.MetricsInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		Counts:      counts,
	})

	hero := buildPageHero(h.deps, ed, counts, "")
	strip := buildMetricStrip(metrics)

	pd := pageData{
		Roadmap: roadmap,
		Blocked: blocked,
		Shipped: shipped,
	}

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, "horizons"); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		if err := h.tmpl.ExecuteTemplate(out, "view-toolbar.html", toolbarData{BlockedCount: roadmap.BlockedCount, Active: "horizons"}); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "page.html", pd)
	}

	return shell.Page{
		ActiveHome: "work",
		PageTitle:  "Work · Hero",
		Content:    content,
		HeadExtra:  template.HTML(workStyles + workScript),
	}
}

// renderBlocked handles GET /work/blocked — full blocked list (the
// existing blocked.html partial renders the section). The view-toolbar
// is rendered above with the Blocked tab active.
func (h *handler) renderBlocked(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	counts := data.LoadCounts(data.CountsInputs{
		ProjectRoot: h.deps.ProjectRoot, HeroDir: h.deps.HeroDir,
	})
	blocked := data.LoadBlocked(data.BlockedInputs{
		ProjectRoot: h.deps.ProjectRoot, HeroDir: h.deps.HeroDir,
	})
	metrics := data.LoadMetrics(data.MetricsInputs{
		ProjectRoot: h.deps.ProjectRoot, HeroDir: h.deps.HeroDir, Counts: counts,
	})

	hero := buildPageHero(h.deps, ed, counts, "Blocked")
	strip := buildMetricStrip(metrics)

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, "blocked"); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		if err := h.tmpl.ExecuteTemplate(out, "view-toolbar.html", toolbarData{BlockedCount: blocked.Total, Active: "blocked"}); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "blocked.html", blocked)
	}

	page := shell.Page{
		ActiveHome: "work",
		PageTitle:  "Work · Blocked · Hero",
		Content:    content,
		HeadExtra:  template.HTML(workStyles + workScript),
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "work: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// renderSpecDetail handles GET /work/spec/{slug}. Resolves the spec
// from .hero/specs/ or .hero/planning/{features,bugs,initiatives}/ and
// renders its title, metadata, rendered markdown body, and relations.
// Returns 404 (the shell's default not-found page) when the slug
// doesn't resolve.
func (h *handler) renderSpecDetail(w http.ResponseWriter, req *http.Request) {
	slug := req.PathValue("slug")
	detail := data.LoadSpec(h.deps.HeroDir, slug)
	if detail == nil {
		http.NotFound(w, req)
		return
	}

	ed := edition.Resolve()
	counts := data.LoadCounts(data.CountsInputs{
		ProjectRoot: h.deps.ProjectRoot, HeroDir: h.deps.HeroDir,
	})
	metrics := data.LoadMetrics(data.MetricsInputs{
		ProjectRoot: h.deps.ProjectRoot, HeroDir: h.deps.HeroDir, Counts: counts,
	})

	hero := buildSpecDetailPageHero(h.deps, detail)
	strip := buildMetricStrip(metrics)
	crumb := &shell.PageBreadcrumb{Crumbs: []shell.BreadcrumbCrumb{
		{Label: "Work", Href: "/work"},
		{Label: "Spec"},
		{Label: detail.Slug, Current: true},
	}}

	// Per polish-v3 Fix 2: spec detail skips the in-page view-toggle
	// row (the user is no longer choosing a view); the breadcrumb
	// above takes over for orientation.
	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, "spec:"+detail.Slug); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "spec-detail.html", detail)
	}

	// Per v4 Fix 4: browser <title> includes the display title after
	// the slug so the tab is searchable by either. Falls back to slug-
	// only when the loader couldn't parse a title.
	title := "Work · Spec · " + detail.Slug
	if strings.TrimSpace(detail.Title) != "" && detail.Title != detail.Slug {
		title += " — " + detail.Title
	}
	title += " · Hero"

	_ = ed
	page := shell.Page{
		ActiveHome: "work",
		PageTitle:  title,
		Breadcrumb: crumb,
		Content:    content,
		HeadExtra:  template.HTML(workStyles + workScript),
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "work: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// buildSpecDetailPageHero composes the page-hero for the spec detail
// view. Title is the spec title (the breadcrumb above shows the
// `Work › Spec › <slug>` crumb trail per polish-v3 Fix 2). Subhead
// lists type / status / horizon plus the Goal one-liner when short.
func buildSpecDetailPageHero(deps Deps, d *data.SpecDetail) shell.PageHero {
	branch := deps.Branch
	if branch == "" {
		branch = "main"
	}
	eyebrow := fmt.Sprintf("hero · %s · work", branch)

	parts := []string{}
	if d.Type != "" {
		parts = append(parts, template.HTMLEscapeString(d.Type))
	}
	if d.Status != "" {
		parts = append(parts, template.HTMLEscapeString(d.Status))
	}
	if d.Horizon != "" {
		parts = append(parts, "horizon "+template.HTMLEscapeString(d.Horizon))
	}
	subhead := strings.Join(parts, `<span class="dot-sep">·</span>`)

	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   d.Title,
		Subhead: template.HTML(subhead),
		Actions: []shell.PageHeroAction{
			{Kind: "ghost", Label: "Back to Work", Href: "/work"},
		},
	}
}

// renderKanban / renderGraph — substrate-pending stubs.
func (h *handler) renderKanban(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "kanban", "Kanban",
		"Kanban projection of the horizons (status columns by claimed agent) lands in a follow-up.")
}

func (h *handler) renderGraph(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "graph", "Graph",
		"Spec graph view (relations as nodes/edges) lands once traversal-queries exposes its JSON over HTTP.")
}

func (h *handler) renderStub(w http.ResponseWriter, req *http.Request, slug, view, note string) {
	ed := edition.Resolve()
	counts := data.LoadCounts(data.CountsInputs{
		ProjectRoot: h.deps.ProjectRoot, HeroDir: h.deps.HeroDir,
	})
	roadmap := data.LoadRoadmap(data.RoadmapInputs{
		ProjectRoot: h.deps.ProjectRoot, HeroDir: h.deps.HeroDir,
	})
	metrics := data.LoadMetrics(data.MetricsInputs{
		ProjectRoot: h.deps.ProjectRoot, HeroDir: h.deps.HeroDir, Counts: counts,
	})

	hero := buildPageHero(h.deps, ed, counts, view)
	strip := buildMetricStrip(metrics)

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, slug); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		if err := h.tmpl.ExecuteTemplate(out, "view-toolbar.html", toolbarData{BlockedCount: roadmap.BlockedCount, Active: slug}); err != nil {
			return err
		}
		return h.router.RenderFragment(out, "coming-soon", stubData{
			Home: "work", Slug: slug, View: view, Note: note,
		})
	}
	page := shell.Page{
		ActiveHome: "work",
		PageTitle:  "Work · " + view + " · Hero",
		Content:    content,
		HeadExtra:  template.HTML(workStyles + workScript),
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "work: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

type stubData struct {
	Home string
	Slug string
	View string
	Note string
}

// renderSection produces the standalone HTML fragment for one Work
// section, for the SSE fragment-replacement endpoints. view selects
// the active toolbar tab when section is "toolbar"; ignored for other
// sections. Empty view defaults to "horizons".
func (h *handler) renderSection(section, view string) ([]byte, error) {
	var tplName string
	var payload any

	switch section {
	case "roadmap":
		tplName = "roadmap.html"
		payload = data.LoadRoadmap(data.RoadmapInputs{
			ProjectRoot: h.deps.ProjectRoot,
			HeroDir:     h.deps.HeroDir,
		})
	case "blocked":
		tplName = "blocked.html"
		payload = data.LoadBlocked(data.BlockedInputs{
			ProjectRoot: h.deps.ProjectRoot,
			HeroDir:     h.deps.HeroDir,
		})
	case "shipped":
		tplName = "shipped.html"
		payload = data.LoadRecentlyShipped(data.ShippedInputs{
			ProjectRoot: h.deps.ProjectRoot,
			HeroDir:     h.deps.HeroDir,
		})
	case "toolbar":
		tplName = "view-toolbar.html"
		rm := data.LoadRoadmap(data.RoadmapInputs{
			ProjectRoot: h.deps.ProjectRoot,
			HeroDir:     h.deps.HeroDir,
		})
		// SSE fragment refresh — the active tab is supplied by the
		// caller via `?view=` (per v5 Fix 4) so a toolbar swap on
		// `/work/kanban` doesn't snap the active tab back to Horizons.
		// Empty defaults to horizons (root view).
		if view == "" {
			view = "horizons"
		}
		payload = toolbarData{BlockedCount: rm.BlockedCount, Active: view}
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
	Roadmap data.Roadmap
	Blocked data.Blocked
	Shipped data.RecentlyShipped
}

type toolbarData struct {
	BlockedCount int
	// Active is the slug of the currently-active view-tab.
	// Valid values: "horizons" | "kanban" | "graph" | "blocked".
	// Empty renders no tab as active (callers should always set it).
	// This zero-value behavior is documented and tested in
	// TestRegister_ViewToolbarEmptyActiveRendersNoActiveTab.
	Active string
}

// buildPageHero composes the page-hero data block from current counts.
// subView, when non-empty, is appended to the title as `Work · <Sub>`
// per polish-v3 Fix 5.
func buildPageHero(deps Deps, ed edition.Edition, c data.PageCounts, subView string) shell.PageHero {
	branch := deps.Branch
	if branch == "" {
		branch = "main"
	}
	eyebrow := fmt.Sprintf("hero · %s", branch)

	parts := []string{
		fmt.Sprintf("<strong>%d spec%s</strong>", c.Total, plural(c.Total)),
		fmt.Sprintf("<strong>%d delivering</strong>", c.Delivering),
	}
	if c.Blocked > 0 {
		parts = append(parts, fmt.Sprintf(`<strong style="color:var(--warn);">%d blocked</strong>`, c.Blocked))
	} else {
		parts = append(parts, "<strong>0 blocked</strong>")
	}
	parts = append(parts, template.HTMLEscapeString(c.SprintState))
	subhead := strings.Join(parts, `<span class="dot-sep">·</span>`)

	_ = ed

	title := "Work"
	if subView != "" {
		title = "Work · " + subView
	}
	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   title,
		Subhead: template.HTML(subhead),
		Actions: []shell.PageHeroAction{
			{Kind: "primary", Label: "New spec", Href: "#"},
			{Kind: "ghost", Label: "Import from tracker", Href: "#"},
			{Kind: "ghost", Label: "Plan sprint", Href: "#"},
		},
	}
}

// buildMetricStrip composes the three-tab Work metric strip.
func buildMetricStrip(m data.Metrics) shell.MetricStrip {
	return shell.MetricStrip{
		AllLink: "#",
		Tabs: []shell.MetricTab{
			{Slug: "sprint", Label: "This sprint", Active: true, Tiles: m.SprintTiles},
			{Slug: "throughput", Label: "Throughput", Tiles: m.ThroughputTiles},
			{Slug: "quality", Label: "Quality", Tiles: m.QualityTiles},
		},
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func parsePage(s string) int {
	if s == "" {
		return 1
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 1
	}
	return n
}
