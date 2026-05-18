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
	"strings"

	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/pages/work/data"
	"github.com/hero-engine/hero/internal/serve/shell"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Deps is the small, injectable bundle of state the Work handler reads.
// Keeping this an interface-free struct lets the server wire whatever
// concrete dependencies it already owns (project root, hero dir, user
// name, …) without dragging the serve package into pages/work.
type Deps struct {
	// ProjectRoot is the absolute path to the project being served.
	// Empty disables data fetchers that need filesystem access.
	ProjectRoot string

	// HeroDir is the absolute path to .hero/ inside ProjectRoot. Empty
	// disables data fetchers that read specs / events.log.
	HeroDir string

	// Workspace is the human label shown in the eyebrow / footer chrome.
	Workspace string

	// Branch is the current git branch ("" when not in a git checkout).
	Branch string

	// UserName is the display name used in the eyebrow / personalization
	// strings.
	UserName string
}

// Register installs the Work home on the shell router using the
// provided dependency bundle. The Work home occupies the "work" slug —
// the placeholder registered by shell.RegisterStubHomes must be
// dropped first by the caller (we do not double-register).
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
	})
}

// SectionFragment returns the rendered HTML for a single Work section.
// Exposed for use by the SSE fragment endpoints in internal/serve/api.
// The output is exactly what the initial /work render produces for
// that section, so client-side replacement preserves layout.
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
		http.Error(w, "work: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// buildPage assembles the Page envelope passed to shell.Router.RenderPage.
// All data-fetcher invocations live here so the handler stays a thin
// composition point.
func (h *handler) buildPage(req *http.Request, ed edition.Edition) shell.Page {
	counts := data.LoadCounts(data.CountsInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
	})
	roadmap := data.LoadRoadmap(data.RoadmapInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
	})
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

	hero := buildPageHero(h.deps, ed, counts)
	strip := buildMetricStrip(metrics)

	pd := pageData{
		Toolbar: toolbarData{BlockedCount: roadmap.BlockedCount},
		Roadmap: roadmap,
		Blocked: blocked,
		Shipped: shipped,
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
		ActiveHome: "work",
		PageTitle:  "Work · Hero",
		Content:    content,
		HeadExtra:  template.HTML(workStyles + workScript),
	}
}

// renderSection produces the standalone HTML fragment for one Work
// section, for the SSE fragment-replacement endpoints. Returns an
// error for unknown section names so callers can 404 cleanly.
func (h *handler) renderSection(section string) ([]byte, error) {
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
		payload = toolbarData{BlockedCount: rm.BlockedCount}
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
	Toolbar toolbarData
	Roadmap data.Roadmap
	Blocked data.Blocked
	Shipped data.RecentlyShipped
}

type toolbarData struct {
	BlockedCount int
}

// buildPageHero composes the page-hero data block from current counts.
// Subhead format: `<n> specs · <n> delivering · <n> blocked · <sprint
// status>`. Blocked count is colored warn when ≥ 1.
func buildPageHero(deps Deps, ed edition.Edition, c data.PageCounts) shell.PageHero {
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

	_ = ed // edition gating for actions lands later

	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   "Work",
		Subhead: template.HTML(subhead),
		Actions: []shell.PageHeroAction{
			{Kind: "primary", Label: "New spec", Href: "#"},
			{Kind: "ghost", Label: "Import from tracker", Href: "#"},
			{Kind: "ghost", Label: "Plan sprint", Href: "#"},
		},
	}
}

// buildMetricStrip composes the three-tab Work metric strip. The first
// tab is active by default.
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
