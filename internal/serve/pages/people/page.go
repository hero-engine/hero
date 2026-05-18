// Package people hosts the People & ROI home — the team-pulse and
// canonical Hero-ROI surface at GET /people. It composes the shell's
// page chrome (top nav, sub-nav row, page-hero, tabbed-metric-strip,
// footer) with the People + ROI section partials. The handler reads
// live state via a small set of data-fetchers under ./data and the
// canonical ROI metric computations come from internal/serve/metrics.
//
// Per hero-people-and-roi-home, this package owns NO chat dispatch and
// NO model inference. The Money chain values rendered here are the
// canonical computations; the Now home's ROI tab is a separate UI tab
// that today renders placeholders — rewiring Now to read from this
// package's metrics output is a follow-up.
package people

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/metrics"
	"github.com/hero-engine/hero/internal/serve/pages/people/data"
	"github.com/hero-engine/hero/internal/serve/shell"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Deps is the small, injectable bundle of state the People handler reads.
// Mirrors the Now home's Deps shape — value-typed and interface-free —
// so the server wires whatever concrete dependencies it already owns.
type Deps struct {
	// ProjectRoot is the absolute path to the project being served.
	// Empty disables data fetchers that need filesystem access.
	ProjectRoot string

	// HeroDir is the absolute path to .hero/ inside ProjectRoot. Empty
	// disables data fetchers that read events.log / specs.
	HeroDir string

	// Workspace is the human label shown in the eyebrow / footer chrome.
	Workspace string

	// Branch is the current git branch ("" when not in a git checkout).
	Branch string

	// UserName is the display name used in the eyebrow / personalization
	// strings.
	UserName string

	// APISpendUSD returns the Hero API spend over the active window in
	// dollars. Nil-safe — when nil the Net value / ROI multiple tiles
	// degrade to the "adapter cost reporting not configured" state.
	APISpendUSD func() float64
}

// Register installs the People home on the shell router using the
// provided dependency bundle. The People slug ("people") must be
// dropped from the shell's stub registrations by the caller — this
// function does not double-register.
func Register(r *shell.Router, deps Deps) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("people: load templates: %w", err)
	}

	h := &handler{
		router: r,
		tmpl:   tmpl,
		deps:   deps,
	}
	return r.RegisterHome(shell.Home{
		Slug:   "people",
		Label:  "People",
		Href:   "/people",
		Render: h.handle,
	})
}

// SectionFragment returns the rendered HTML for a single People section.
// Exposed for use by the SSE fragment endpoints in internal/serve/api.
func SectionFragment(deps Deps, section string) ([]byte, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	h := &handler{tmpl: tmpl, deps: deps}
	return h.renderSection(section)
}

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

	pulse := data.LoadPulse(data.PulseInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		Edition:     string(ed),
		UserName:    h.deps.UserName,
	})
	overview := data.LoadOverview(data.OverviewInputs{
		HeroDir:     h.deps.HeroDir,
		Edition:     string(ed),
		WindowDays:  28,
		APISpendUSD: h.callSpend(),
		Coefficients: metrics.DefaultCoefficients(),
	})

	hero := buildPageHero(h.deps, ed, overview)
	strip := buildMetricStrip(overview)
	subNav := buildSubNav("pulse")

	pd := pageData{
		Pulse:   pulse,
		ROI:     overview,
		ShowROI: true,
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

	page := shell.Page{
		ActiveHome: "people",
		PageTitle:  "People · Hero",
		SubNav:     subNav,
		Content:    content,
		HeadExtra:  template.HTML(peopleStyles + peopleScript),
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "people: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// renderSection produces the standalone HTML fragment for one section,
// used by SSE fragment-replacement endpoints.
func (h *handler) renderSection(section string) ([]byte, error) {
	ed := edition.Resolve()

	var tplName string
	var payload any
	switch section {
	case "pulse":
		tplName = "pulse.html"
		payload = data.LoadPulse(data.PulseInputs{
			ProjectRoot: h.deps.ProjectRoot,
			HeroDir:     h.deps.HeroDir,
			Edition:     string(ed),
			UserName:    h.deps.UserName,
		})
	case "overview":
		tplName = "overview.html"
		payload = data.LoadOverview(data.OverviewInputs{
			HeroDir:      h.deps.HeroDir,
			Edition:      string(ed),
			WindowDays:   28,
			APISpendUSD:  h.callSpend(),
			Coefficients: metrics.DefaultCoefficients(),
		})
	default:
		return nil, fmt.Errorf("unknown section %q", section)
	}

	var buf strings.Builder
	if err := h.tmpl.ExecuteTemplate(&buf, tplName, payload); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func (h *handler) callSpend() float64 {
	if h.deps.APISpendUSD == nil {
		return 0
	}
	return h.deps.APISpendUSD()
}

// pageData is the outer-template input.
type pageData struct {
	Pulse   data.Pulse
	ROI     data.ROIOverview
	ShowROI bool
}

// buildPageHero composes the page-hero data block. The eyebrow shows
// the workspace + branch + edition. The subhead carries the canonical
// People & ROI summary line.
func buildPageHero(deps Deps, ed edition.Edition, overview data.ROIOverview) shell.PageHero {
	eyebrow := fmt.Sprintf("hero · %s · roi", firstNonEmpty(deps.Branch, "main"))
	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   "People & ROI",
		Subhead: overview.SubheadHTML,
		Actions: []shell.PageHeroAction{
			{Kind: "ghost", Label: overview.WindowLabel, Href: "#"},
			{Kind: "ghost", Label: "Compare prior 4w", Href: "#"},
			{Kind: "chip", Label: strings.Title(string(ed)), Chip: string(ed)}, //nolint:staticcheck // single-byte upper is fine here
		},
	}
}

// buildMetricStrip routes the Money/Throughput/Quality tile sets from
// the overview payload into the shared tabbed-metric-strip fragment.
// Money is the default-active tab per the spec.
func buildMetricStrip(o data.ROIOverview) shell.MetricStrip {
	return shell.MetricStrip{
		Tabs: []shell.MetricTab{
			{Slug: "money", Label: "Money", Active: true, Tiles: o.MoneyTiles},
			{Slug: "throughput", Label: "Throughput", Tiles: o.ThroughputTiles},
			{Slug: "quality", Label: "Quality", Tiles: o.QualityTiles},
		},
	}
}

// buildSubNav builds the People & ROI sub-nav row. activeSlug picks the
// active tab; valid slugs are pulse|activity|handoffs|profiles|
// roi-overview|velocity|autonomy|knowledge|individual|export. The faded
// Export tab uses the "locked" variant with an Enterprise lock-meta
// label so the shell template renders it consistently with other
// enterprise-locked surfaces.
func buildSubNav(activeSlug string) *shell.SubNav {
	tabs := []shell.SubNavTab{
		{Label: "Pulse", Href: "/people", Active: activeSlug == "pulse"},
		{Label: "Activity", Href: "/people/activity", Active: activeSlug == "activity"},
		{Label: "Handoffs", Href: "/people/handoffs", Active: activeSlug == "handoffs"},
		{Label: "Profiles", Href: "/people/profiles", Active: activeSlug == "profiles"},
		{Label: "ROI Overview", Href: "/people/roi", Active: activeSlug == "roi-overview"},
		{Label: "Velocity", Href: "/people/roi/velocity", Active: activeSlug == "velocity"},
		{Label: "Autonomy", Href: "/people/roi/autonomy", Active: activeSlug == "autonomy"},
		{Label: "Knowledge reuse", Href: "/people/roi/knowledge", Active: activeSlug == "knowledge"},
		{Label: "My productivity", Href: "/people/roi/individual", Active: activeSlug == "individual"},
		{Label: "Export", Href: "/people/roi/export", Variant: "locked", LockMeta: "Enterprise", Badge: "Enterprise"},
	}
	return &shell.SubNav{Tabs: tabs}
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
