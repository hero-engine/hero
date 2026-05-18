// Package now hosts the Now home — the personal cold-start surface at
// GET /now. It composes the shell's page chrome (top nav, page-hero
// fragment, tabbed-metric-strip fragment, chat-input fragment, footer)
// with Now-specific section partials for: needs-your-input, on-your-
// plate, your-agents, since-you-were-here. The handler reads live
// state via a small set of data-fetchers under ./data and SSE updates
// are wired by the sibling internal/serve/api/now.go.
//
// Per hero-now-home, this package owns NO chat dispatch and NO model
// inference; the Quick launch section mounts the shell-owned chat-input
// fragment and forwards submission to /api/chat/*.
package now

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/pages/now/data"
	"github.com/hero-engine/hero/internal/serve/shell"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Deps is the small, injectable bundle of state the Now handler reads.
// Keeping this an interface-free struct lets the server wire whatever
// concrete dependencies it already owns (project root, hero dir, propose
// store, etc.) without dragging the serve package into pages/now.
type Deps struct {
	// ProjectRoot is the absolute path to the project being served.
	// Empty disables data fetchers that need filesystem access.
	ProjectRoot string

	// HeroDir is the absolute path to .hero/ inside ProjectRoot. Empty
	// disables data fetchers that read events.log / specs.
	HeroDir string

	// Workspace is the human label shown in the eyebrow / footer chrome.
	// The shell owns the actual chrome — this is passed through for the
	// hero-text composition only.
	Workspace string

	// Branch is the current git branch ("" when not in a git checkout).
	Branch string

	// UserName is the display name used in the eyebrow / personalization
	// strings.
	UserName string

	// Proposals returns the proposal envelopes pending across all
	// sessions for the active project. Nil-safe — when nil or empty the
	// inbox just omits proposal rows.
	Proposals func() []*data.ProposalRow

	// RegisterFragment lets the api package mount fragment endpoints
	// against the same template set the handler uses, so SSE clients
	// fetch HTML identical to the initial render. Nil-safe.
	RegisterFragment func(section string, render func(w http.ResponseWriter, r *http.Request))
}

// Register installs the Now home on the shell router using the
// provided dependency bundle. The Now home occupies the "now" slug —
// the placeholder registered by shell.RegisterStubHomes must be
// dropped first by the caller (we do not double-register).
func Register(r *shell.Router, deps Deps) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("now: load templates: %w", err)
	}

	h := &handler{
		router: r,
		tmpl:   tmpl,
		deps:   deps,
	}
	return r.RegisterHome(shell.Home{
		Slug:   "now",
		Label:  "Now",
		Href:   "/now",
		Render: h.handle,
	})
}

// SectionFragment returns the rendered HTML for a single Now section.
// Exposed for use by the SSE fragment endpoints in internal/serve/api.
// The output is exactly what the initial /now render produces for that
// section, so client-side replacement preserves layout.
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
	cfg, _ := config.Load(h.deps.ProjectRoot)
	ed := edition.Resolve()
	methodology := resolveMethodology(cfg)

	page := h.buildPage(req, cfg, ed, methodology)

	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "now: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// buildPage assembles the Page envelope passed to shell.Router.RenderPage.
// All data-fetcher invocations live here so the handler stays a thin
// composition point.
func (h *handler) buildPage(req *http.Request, cfg config.Config, ed edition.Edition, methodology string) shell.Page {
	inbox := data.LoadInbox(data.InboxInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		Edition:     string(ed),
		Proposals:   h.callProposals(),
	})
	plate := data.LoadPlate(data.PlateInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		UserName:    h.deps.UserName,
	})
	agents := data.LoadAgents(data.AgentsInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		Edition:     string(ed),
	})
	changes := data.LoadChanges(data.ChangesInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
	})
	metrics := data.LoadMetrics(data.MetricsInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		UserName:    h.deps.UserName,
		Methodology: methodology,
	})

	hero := buildPageHero(h.deps, ed, len(inbox.Rows), agents.RunningCount, agents.LastActivePretty)
	strip := buildMetricStrip(methodology, metrics)

	pd := pageData{
		ChatPlaceholder: "Describe what you want to work on, ask a question, or paste an error…",
		IntentChips: []string{
			"/design", "/diagnose", "/deliver", "/review", "/ask",
		},
		TryPrompts: []data.TryPrompt{
			{Label: "finish per-feature-smoke-coverage", Href: "#"},
			{Label: "why is scan enrichment looping?", Href: "#"},
			{Label: "design rate-limited peer calls", Href: "#"},
		},
		Inbox:   inbox,
		Plate:   plate,
		Agents:  agents,
		Changes: changes,
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
		ActiveHome: "now",
		PageTitle:  "Now · Hero",
		Content:    content,
		HeadExtra:  template.HTML(nowStyles + nowScript),
	}
}

// renderSection produces the standalone HTML fragment for one section,
// for the SSE fragment-replacement endpoints. Returns the empty slice
// for unknown section names so callers can 404 cleanly.
func (h *handler) renderSection(section string) ([]byte, error) {
	cfg, _ := config.Load(h.deps.ProjectRoot)
	ed := edition.Resolve()

	var tplName string
	var payload any
	switch section {
	case "inbox":
		tplName = "inbox.html"
		payload = data.LoadInbox(data.InboxInputs{
			ProjectRoot: h.deps.ProjectRoot,
			HeroDir:     h.deps.HeroDir,
			Edition:     string(ed),
			Proposals:   h.callProposals(),
		})
	case "plate":
		tplName = "plate.html"
		payload = data.LoadPlate(data.PlateInputs{
			ProjectRoot: h.deps.ProjectRoot,
			HeroDir:     h.deps.HeroDir,
			UserName:    h.deps.UserName,
		})
	case "agents":
		tplName = "agents.html"
		payload = data.LoadAgents(data.AgentsInputs{
			ProjectRoot: h.deps.ProjectRoot,
			HeroDir:     h.deps.HeroDir,
			Edition:     string(ed),
		})
	case "changes":
		tplName = "changes.html"
		payload = data.LoadChanges(data.ChangesInputs{
			ProjectRoot: h.deps.ProjectRoot,
			HeroDir:     h.deps.HeroDir,
		})
	default:
		_ = cfg
		return nil, fmt.Errorf("unknown section %q", section)
	}

	var buf strings.Builder
	if err := h.tmpl.ExecuteTemplate(&buf, tplName, payload); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func (h *handler) callProposals() []*data.ProposalRow {
	if h.deps.Proposals == nil {
		return nil
	}
	return h.deps.Proposals()
}

// pageData is the outer-template input.
type pageData struct {
	ChatPlaceholder string
	IntentChips     []string
	TryPrompts      []data.TryPrompt
	Inbox           data.Inbox
	Plate           data.Plate
	Agents          data.Agents
	Changes         data.Changes
}

// resolveMethodology returns the active methodology, defaulting to
// "solo" when the config field is empty or unrecognized. Per the spec's
// risks section, an unknown value never crashes the page — it falls
// through to the solo defaults.
func resolveMethodology(cfg config.Config) string {
	m := strings.ToLower(strings.TrimSpace(cfg.Methodology))
	switch m {
	case "scrum", "shape-up", "kanban", "solo":
		return m
	default:
		return "solo"
	}
}

// firstTabFor returns the (slug, label) pair for the methodology-aware
// first metric tab.
func firstTabFor(methodology string) (slug, label string) {
	switch methodology {
	case "scrum":
		return "sprint", "This sprint"
	case "shape-up":
		return "cycle", "This cycle"
	case "kanban", "solo":
		return "week", "This week"
	default:
		return "week", "This week"
	}
}

// buildPageHero composes the page-hero data block from current counts.
func buildPageHero(deps Deps, ed edition.Edition, inboxCount, runningCount int, lastActive string) shell.PageHero {
	eyebrow := fmt.Sprintf("hero · %s · %s edition", firstNonEmpty(deps.Branch, "main"), string(ed))

	parts := []string{}
	switch inboxCount {
	case 0:
		// Skip — empty inbox tells its own story in the section below.
	case 1:
		parts = append(parts, "<strong>1 needs your input</strong>")
	default:
		parts = append(parts, fmt.Sprintf("<strong>%d need your input</strong>", inboxCount))
	}
	switch runningCount {
	case 0:
		parts = append(parts, "<strong>no agent running</strong>")
	case 1:
		parts = append(parts, "<strong>1 agent running</strong>")
	default:
		parts = append(parts, fmt.Sprintf("<strong>%d agents running</strong>", runningCount))
	}
	if lastActive != "" {
		parts = append(parts, "since "+template.HTMLEscapeString(lastActive))
	}
	subhead := strings.Join(parts, `<span class="dot-sep">·</span>`)

	editionLabel := "Solo"
	if ed != edition.Local {
		editionLabel = strings.Title(string(ed)) //nolint:staticcheck // single-byte upper is fine here
	}

	return shell.PageHero{
		Eyebrow: template.HTML(template.HTMLEscapeString(eyebrow)),
		Title:   "Now",
		Subhead: template.HTML(subhead),
		Actions: []shell.PageHeroAction{
			{Kind: "primary", Label: "Open Inbox", Href: "#now-inbox"},
			{Kind: "ghost", Label: "What changed since last", Href: "#now-changes"},
			{Kind: "chip", Label: editionLabel, Chip: editionLabel},
		},
	}
}

// buildMetricStrip composes the MetricStrip from the methodology-aware
// data and the cross-methodology My-week / ROI tiles. The first tab is
// active.
func buildMetricStrip(methodology string, m data.Metrics) shell.MetricStrip {
	firstSlug, firstLabel := firstTabFor(methodology)
	return shell.MetricStrip{
		AllLink: "#",
		Tabs: []shell.MetricTab{
			{
				Slug:   firstSlug,
				Label:  firstLabel,
				Active: true,
				Tiles:  m.FirstTabTiles,
			},
			{
				Slug:  "week",
				Label: "My week",
				Tiles: m.MyWeekTiles,
			},
			{
				Slug:  "roi",
				Label: "Hero ROI",
				Tiles: m.ROITiles,
			},
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
