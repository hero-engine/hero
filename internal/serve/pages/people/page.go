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
//
// Routing: /people renders the Pulse view; /people/roi renders the ROI
// Overview; all other sub-nav anchors register stubs so the sub-nav
// never 404s.
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
type Deps struct {
	ProjectRoot string
	HeroDir     string
	Workspace   string
	Branch      string
	UserName    string
	APISpendUSD func() float64
}

// Register installs the People home on the shell router. /people defaults
// to the Pulse view; /people/roi renders the ROI Overview; the other
// sub-nav anchors register stubs so every link returns 200.
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
		Render: h.renderPulse,
		Items: []shell.ItemRoute{
			{Pattern: "GET /people/roi", Render: h.renderROIOverview},
			{Pattern: "GET /people/activity", Render: h.renderActivity},
			{Pattern: "GET /people/handoffs", Render: h.renderHandoffs},
			{Pattern: "GET /people/profiles", Render: h.renderProfiles},
			{Pattern: "GET /people/roi/velocity", Render: h.renderVelocity},
			{Pattern: "GET /people/roi/autonomy", Render: h.renderAutonomy},
			{Pattern: "GET /people/roi/knowledge", Render: h.renderKnowledge},
			{Pattern: "GET /people/roi/individual", Render: h.renderIndividual},
			{Pattern: "GET /people/roi/export", Render: h.renderExport},
		},
	})
}

// SectionFragment returns the rendered HTML for a single People section.
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

// renderPulse handles GET /people — the default Pulse view (presence +
// recent activity feed only; no ROI sections).
func (h *handler) renderPulse(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	pulse := data.LoadPulse(data.PulseInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		Edition:     string(ed),
		UserName:    h.deps.UserName,
	})

	hero := buildPulsePageHero(h.deps, ed, pulse)
	strip := buildPulseStrip(pulse)
	subNav := buildSubNav("pulse")

	content := func(out io.Writer) error {
		if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "pulse.html", pulse)
	}
	h.serve(w, req, content, subNav, "People · Hero")
}

// renderROIOverview handles GET /people/roi — the full ROI Overview
// view (Money strip + How time was spent + savings + 12-week trend +
// contributors + what changed).
func (h *handler) renderROIOverview(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	overview := data.LoadOverview(data.OverviewInputs{
		HeroDir:      h.deps.HeroDir,
		Edition:      string(ed),
		WindowDays:   28,
		APISpendUSD:  h.callSpend(),
		Coefficients: metrics.DefaultCoefficients(),
	})

	hero := buildROIPageHero(h.deps, ed, overview)
	strip := buildROIStrip(overview)
	subNav := buildSubNav("roi-overview")

	content := func(out io.Writer) error {
		if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.tmpl.ExecuteTemplate(out, "overview.html", overview)
	}
	h.serve(w, req, content, subNav, "People · ROI · Hero")
}

func (h *handler) renderActivity(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "activity", "Activity",
		"Per-user activity timeline lands once the events log has per-user attribution.")
}
func (h *handler) renderHandoffs(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "handoffs", "Handoffs",
		"Cross-repo handoff ledger lands once peer.* events expose their full payloads.")
}
func (h *handler) renderProfiles(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "profiles", "Profiles",
		"Per-contributor profile pages land once the per-user rollup exists.")
}
func (h *handler) renderVelocity(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "velocity", "Velocity",
		"Velocity deep-dive view lands once the trend pipeline is wired.")
}
func (h *handler) renderAutonomy(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "autonomy", "Autonomy",
		"Autonomy ratio deep-dive lands once propose-lifecycle events accumulate.")
}
func (h *handler) renderKnowledge(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "knowledge", "Knowledge reuse",
		"Knowledge-reuse ROI lands once the retrieval-attribution log is wired.")
}
func (h *handler) renderIndividual(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "individual", "My productivity",
		"Per-user productivity view lands once individual attribution exists.")
}
func (h *handler) renderExport(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "export", "Export",
		"Export is an enterprise feature.")
}

// renderStub renders the home chrome + sub-nav with the shared coming-
// soon shell card in the body.
func (h *handler) renderStub(w http.ResponseWriter, req *http.Request, slug, view, note string) {
	ed := edition.Resolve()
	pulse := data.LoadPulse(data.PulseInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		Edition:     string(ed),
		UserName:    h.deps.UserName,
	})

	hero := buildPulsePageHero(h.deps, ed, pulse)
	strip := buildPulseStrip(pulse)
	subNav := buildSubNav(slug)

	content := func(out io.Writer) error {
		if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.router.RenderFragment(out, "coming-soon", stubData{
			Home: "people", Slug: slug, View: view, Note: note,
		})
	}
	h.serve(w, req, content, subNav, "People · "+view+" · Hero")
}

func (h *handler) serve(w http.ResponseWriter, req *http.Request, content func(io.Writer) error, subNav *shell.SubNav, title string) {
	page := shell.Page{
		ActiveHome: "people",
		PageTitle:  title,
		SubNav:     subNav,
		Content:    content,
		HeadExtra:  template.HTML(peopleStyles + peopleScript),
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "people: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

type stubData struct {
	Home string
	Slug string
	View string
	Note string
}

// renderSection produces the standalone HTML fragment for one section.
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

// buildPulsePageHero composes the page-hero data for the Pulse view.
// Subhead reflects presence + activity counts only — no ROI dollars.
func buildPulsePageHero(deps Deps, ed edition.Edition, p data.Pulse) shell.PageHero {
	eyebrow := fmt.Sprintf("hero · %s · people", firstNonEmpty(deps.Branch, "main"))
	subhead := template.HTML("Team pulse — who is working, on what, and what just shipped.")
	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   "People",
		Subhead: subhead,
		Actions: []shell.PageHeroAction{
			{Kind: "ghost", Label: "Open ROI Overview", Href: "/people/roi"},
			{Kind: "chip", Label: strings.Title(string(ed)), Chip: string(ed)}, //nolint:staticcheck
		},
	}
}

// buildROIPageHero composes the page-hero data for the ROI Overview view.
func buildROIPageHero(deps Deps, ed edition.Edition, overview data.ROIOverview) shell.PageHero {
	eyebrow := fmt.Sprintf("hero · %s · roi", firstNonEmpty(deps.Branch, "main"))
	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   "People & ROI",
		Subhead: overview.SubheadHTML,
		Actions: []shell.PageHeroAction{
			{Kind: "ghost", Label: overview.WindowLabel, Href: "#"},
			{Kind: "ghost", Label: "Compare prior 4w", Href: "#"},
			{Kind: "chip", Label: strings.Title(string(ed)), Chip: string(ed)}, //nolint:staticcheck
		},
	}
}

// buildPulseStrip is a single-tab metric strip for the Pulse view.
// We keep the chrome consistent with the ROI view but render quieter
// tiles since the live numbers we have are minimal.
func buildPulseStrip(p data.Pulse) shell.MetricStrip {
	return shell.MetricStrip{
		Tabs: []shell.MetricTab{
			{
				Slug: "pulse", Label: "Right now", Active: true,
				Tiles: []shell.MetricTile{
					{Value: template.HTML("—"), Label: "teammates active"},
					{Value: template.HTML("—"), Label: "agents attached"},
					{Value: template.HTML("—"), Label: "awaiting you"},
					{Value: template.HTML("—"), Label: "active in last 24h"},
				},
			},
		},
	}
}

// buildROIStrip routes the Money/Throughput/Quality tile sets from
// the overview payload into the shared tabbed-metric-strip fragment.
// Money is the default-active tab per the spec.
func buildROIStrip(o data.ROIOverview) shell.MetricStrip {
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
// roi-overview|velocity|autonomy|knowledge|individual|export.
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
		{Label: "Export", Href: "/people/roi/export", Active: activeSlug == "export", Variant: "locked", LockMeta: "Enterprise", Badge: "Enterprise"},
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
