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
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve/chat"
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

	// ChatRegistry is the connected-adapter registry used to resolve
	// the chat capability for the page. Nil disables capability
	// detection — the page renders as if no adapter is connected,
	// which is the safe default for offline / unit-test environments.
	ChatRegistry *chat.Registry

	// LiveSessions returns the canonical live-session snapshot the
	// Currently-running block populates from. Nil is safe — the
	// agents loader renders the existing empty state in that case.
	// Shape matches the Agents home's session ledger so both pages
	// surface the same source of truth.
	LiveSessions func() []data.SessionRow
}

// Register installs the Now home on the shell router using the
// provided dependency bundle. The Now home occupies the "now" slug —
// the placeholder registered by shell.RegisterStubHomes must be
// dropped first by the caller (we do not double-register).
func Register(r *shell.Router, deps Deps) error {
	tmpl, err := loadTemplatesFor(r)
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
	tmpl, err := loadTemplatesFor(nil)
	if err != nil {
		return nil, err
	}
	h := &handler{tmpl: tmpl, deps: deps}
	return h.renderSection(section)
}

// QuickLaunchFragment returns the rendered Quick launch section HTML.
// Used by the /api/now/quicklaunch SSE-driven fragment endpoint so
// adapter connect / disconnect events can re-swap the section without a
// full page reload.
func QuickLaunchFragment(deps Deps) ([]byte, error) {
	tmpl, err := loadTemplatesFor(nil)
	if err != nil {
		return nil, err
	}
	h := &handler{tmpl: tmpl, deps: deps}
	return h.renderQuickLaunch()
}

// SubheadText returns the plain-text page-hero subhead for the current
// inbox / agent / activity state. Exposed for the `event: hero` SSE
// payload so the client can swap [data-page-hero-subhead] in place
// without re-fetching the whole hero block.
func SubheadText(deps Deps) string {
	ed := edition.Resolve()
	inbox := data.LoadInbox(data.InboxInputs{
		ProjectRoot: deps.ProjectRoot,
		HeroDir:     deps.HeroDir,
		Edition:     string(ed),
		Proposals:   callProposals(deps),
	})
	agents := data.LoadAgents(data.AgentsInputs{
		ProjectRoot:  deps.ProjectRoot,
		HeroDir:      deps.HeroDir,
		Edition:      string(ed),
		LiveSessions: deps.LiveSessions,
	})
	return subheadPlainText(len(inbox.Rows), agents.RunningCount, agents.LastActivePretty)
}

// loadTemplatesFor parses every .html under the embedded templates/
// directory into a single template set. When router is non-nil the
// resulting set carries `chatInput` and `emptyStateNotice` template
// funcs that render the corresponding shell-owned fragments inline.
// router may be nil when the template set is being used outside the
// shell (e.g. unit tests, the SectionFragment helper) — in that case
// the helpers return empty HTML so templates still execute.
func loadTemplatesFor(router *shell.Router) (*template.Template, error) {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("templates subdir: %w", err)
	}
	funcs := template.FuncMap{
		"chatInput": func(in shell.ChatInput) template.HTML {
			return renderShellFragment(router, "chat-input", in)
		},
		"emptyStateNotice": func(in shell.EmptyState) template.HTML {
			return renderShellFragment(router, "empty-state-notice", in)
		},
	}
	return template.New("").Funcs(funcs).ParseFS(sub, "*.html")
}

// renderShellFragment evaluates a shell-owned fragment and returns its
// HTML. Errors are swallowed and rendered as an HTML comment so a
// template execution never fails because of a missing shell template.
func renderShellFragment(router *shell.Router, name string, data any) template.HTML {
	if router == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := router.RenderFragment(&buf, name, data); err != nil {
		return template.HTML(fmt.Sprintf("<!-- now: render %s: %s -->", name, template.HTMLEscapeString(err.Error())))
	}
	return template.HTML(buf.String())
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
		ProjectRoot:  h.deps.ProjectRoot,
		HeroDir:      h.deps.HeroDir,
		Edition:      string(ed),
		LiveSessions: h.deps.LiveSessions,
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
	noAdapter, emptyState := resolveAdapterState(h.deps)

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
		QuickLaunch: quickLaunchData{
			IntentChips: []string{"/design", "/diagnose", "/deliver", "/review", "/ask"},
			TryPrompts: []data.TryPrompt{
				{Label: "finish per-feature-smoke-coverage", Href: "#"},
				{Label: "why is scan enrichment looping?", Href: "#"},
				{Label: "design rate-limited peer calls", Href: "#"},
			},
			ChatInput:  buildChatInput(noAdapter),
			NoAdapter:  noAdapter,
			EmptyState: emptyState,
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
			ProjectRoot:  h.deps.ProjectRoot,
			HeroDir:      h.deps.HeroDir,
			Edition:      string(ed),
			LiveSessions: h.deps.LiveSessions,
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
	return callProposals(h.deps)
}

// callProposals is the package-level helper backing handler.callProposals.
// Exposed as a free function so SectionFragment / SubheadText /
// QuickLaunchFragment can share the same nil-safe pull without
// constructing a handler value.
func callProposals(deps Deps) []*data.ProposalRow {
	if deps.Proposals == nil {
		return nil
	}
	return deps.Proposals()
}

// pageData is the outer-template input.
type pageData struct {
	ChatPlaceholder string
	IntentChips     []string
	TryPrompts      []data.TryPrompt
	// QuickLaunch carries the per-section data the Quick launch template
	// renders (chat-input params, no-adapter state, empty-state copy,
	// intent chips, try-prompts). Owned here so /api/now/quicklaunch
	// can re-render the same struct out-of-band.
	QuickLaunch quickLaunchData
	Inbox       data.Inbox
	Plate       data.Plate
	Agents      data.Agents
	Changes     data.Changes
}

// quickLaunchData is the input passed to the Quick launch section
// template when rendered standalone via /api/now/quicklaunch. Mirrors
// the subset of pageData the section actually reads.
type quickLaunchData struct {
	IntentChips []string
	TryPrompts  []data.TryPrompt
	ChatInput   shell.ChatInput
	NoAdapter   bool
	EmptyState  shell.EmptyState
}

// buildChatInput composes the Quick launch chat-input fragment params.
// Variant "hero" picks the 64px-tall styling; Context attaches the
// page identity so the chat island dispatches with /now in scope.
//
// When noAdapter is true the input flips into its disabled state with
// the same "Connect a chat adapter to enable" copy the four non-Now
// homes use — restores Now-vs-rest symmetry that previously diverged
// (spec dashboard-adapter-state-hardcoded).
func buildChatInput(noAdapter bool) shell.ChatInput {
	in := shell.ChatInput{
		Variant:     "hero",
		Placeholder: "Tell Hero what to do next…",
		Context: []shell.ChatContextChip{
			{Kind: "page", Label: "page: /now"},
		},
	}
	if noAdapter {
		in.Disabled = true
		in.Placeholder = "Connect a chat adapter to enable"
		in.ConnectHref = "/settings/chat"
	}
	return in
}

// resolveAdapterState returns (noAdapter, emptyState) for the current
// chat capability snapshot. noAdapter is true when chat.Resolve picks
// nothing for the interactive kind; emptyState carries the standard
// install/settings CTA copy from the hero-now-home-followups spec.
//
// A nil ChatRegistry on Deps is treated as "no adapter connected" —
// matches the offline / unit-test default.
func resolveAdapterState(deps Deps) (bool, shell.EmptyState) {
	var interactive string
	if deps.ChatRegistry != nil {
		cap := chat.Resolve(deps.ChatRegistry, "")
		interactive = cap.Interactive
	}
	if interactive != "" {
		return false, shell.EmptyState{}
	}
	return true, shell.EmptyState{
		Headline:      "Hero needs hero-code (or a Hero IDE adapter) to run agent work.",
		Body:          template.HTML("<code>/ask</code> and <code>/note</code> still work right here."),
		PrimaryAction: shell.EmptyStateAction{Label: "Install hero-code", Href: "https://heroengine.ai/install/hero-code"},
		GhostAction:   shell.EmptyStateAction{Label: "Already running it elsewhere →", Href: "/settings/chat"},
		FootNote:      "Using Claude Code, Cursor, or Codex with the Hero IDE adapter? Make sure it's running and connected.",
	}
}

// subheadPlainText renders the page-hero subhead as the plain-text
// payload published on the `event: hero` SSE channel. Mirrors the
// HTML composition in buildPageHero but strips markup so the client
// can drop the string straight into textContent.
//
// Spec: dashboard-now-headline-misleading-when-empty. The previous
// composition appended "since X ago" whenever lastActive was non-empty,
// which read as a quiet-workspace claim when paired with "no agent
// running" — even though X reflected only the most recent emitted
// event. "since X ago" now renders only alongside an actually-running
// session.
func subheadPlainText(inboxCount, runningCount int, lastActive string) string {
	if inboxCount == 0 && runningCount == 0 {
		return "no live activity right now"
	}
	parts := []string{}
	if inboxCount == 1 {
		parts = append(parts, "1 needs your input")
	} else if inboxCount > 1 {
		parts = append(parts, fmt.Sprintf("%d need your input", inboxCount))
	}
	switch runningCount {
	case 0:
		parts = append(parts, "no agent running")
	case 1:
		parts = append(parts, "1 agent running")
	default:
		parts = append(parts, fmt.Sprintf("%d agents running", runningCount))
	}
	if runningCount > 0 && lastActive != "" {
		parts = append(parts, "since "+lastActive)
	}
	return strings.Join(parts, " · ")
}

// renderQuickLaunch produces the standalone Quick launch section HTML
// used by /api/now/quicklaunch. Pulls capability + chat-input data via
// the same helpers buildPage uses so the swap-in fragment is byte-for-
// byte identical to the initial render.
func (h *handler) renderQuickLaunch() ([]byte, error) {
	noAdapter, emptyState := resolveAdapterState(h.deps)
	qd := quickLaunchData{
		IntentChips: []string{"/design", "/diagnose", "/deliver", "/review", "/ask"},
		TryPrompts: []data.TryPrompt{
			{Label: "finish per-feature-smoke-coverage", Href: "#"},
			{Label: "why is scan enrichment looping?", Href: "#"},
			{Label: "design rate-limited peer calls", Href: "#"},
		},
		ChatInput:  buildChatInput(noAdapter),
		NoAdapter:  noAdapter,
		EmptyState: emptyState,
	}
	var buf strings.Builder
	if err := h.tmpl.ExecuteTemplate(&buf, "quicklaunch.html", qd); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
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
//
// Spec: dashboard-now-headline-misleading-when-empty. "since X ago" no
// longer renders unless an agent is actually running; when nothing is
// happening at all the subhead collapses to "no live activity right
// now" instead of composing two empty signals into a false story.
func buildPageHero(deps Deps, ed edition.Edition, inboxCount, runningCount int, lastActive string) shell.PageHero {
	eyebrow := fmt.Sprintf("hero · %s · %s edition", firstNonEmpty(deps.Branch, "main"), string(ed))

	var subhead string
	if inboxCount == 0 && runningCount == 0 {
		subhead = "<strong>no live activity right now</strong>"
	} else {
		parts := []string{}
		if inboxCount == 1 {
			parts = append(parts, "<strong>1 needs your input</strong>")
		} else if inboxCount > 1 {
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
		if runningCount > 0 && lastActive != "" {
			parts = append(parts, "since "+template.HTMLEscapeString(lastActive))
		}
		subhead = strings.Join(parts, `<span class="dot-sep">·</span>`)
	}

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
