// Package agentspage hosts the Agents home — the autonomy surface at
// GET /agents. It composes the shell's page chrome (top nav, sub-nav,
// page-hero fragment, tabbed-metric-strip fragment, footer) with
// Agents-specific section partials for live sessions, awaiting-
// approval rows, completed-today timeline, and a scheduled/automations
// preview split. The handler reads live state via a small set of
// data-fetchers under ./data; SSE updates are wired by the sibling
// internal/serve/api/agents.go.
//
// This package owns NO chat dispatch and NO runner internals. The
// live session ledger is consumed read-only via the dependency-
// injected fetchers; the runner is owned by hero-code. See:
// hero-agents-home (.hero/planning/features/hero-agents-home/spec.md).
package agentspage

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/pages/agentspage/data"
	"github.com/hero-engine/hero/internal/serve/shell"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Deps is the small, injectable bundle of state the Agents handler
// reads. Keeping this struct interface-free lets the server wire
// whatever concrete dependencies it already owns (project root, hero
// dir, session ledger snapshotter, propose store snapshotter, etc.)
// without dragging the serve package into pages/agentspage.
type Deps struct {
	// ProjectRoot is the absolute path to the project being served.
	// Empty disables data fetchers that need filesystem access.
	ProjectRoot string

	// HeroDir is the absolute path to .hero/ inside ProjectRoot.
	HeroDir string

	// Workspace is the human label shown in the eyebrow / footer
	// chrome. The shell owns the actual chrome — this is passed
	// through for the hero-text composition only.
	Workspace string

	// Branch is the current git branch ("" when not in a git
	// checkout).
	Branch string

	// UserName is the display name used in the eyebrow strings.
	UserName string

	// LiveSessions returns the canonical snapshot of in-flight agent
	// sessions. Nil-safe: missing fetcher renders an empty list.
	// This is the *exported* hook the Now home will consume in a
	// follow-up to replace its mocked agents card.
	LiveSessions func() []data.SessionRow

	// Proposals returns the pending proposal envelopes across all
	// sessions. Nil-safe.
	Proposals func() []data.ProposalRow

	// ScheduledTasks returns the registered cron tasks. Nil-safe.
	ScheduledTasks func() []data.ScheduledRow

	// AutomationRules returns the registered automation rules.
	// Nil-safe.
	AutomationRules func() []data.AutomationRow

	// RegisterFragment lets the api package mount fragment endpoints
	// against the same template set the handler uses, so SSE clients
	// fetch HTML identical to the initial render. Nil-safe.
	RegisterFragment func(section string, render func(w http.ResponseWriter, r *http.Request))
}

// Register installs the Agents home on the shell router using the
// supplied dependency bundle. The "agents" stub registered by
// shell.RegisterStubHomes must already have been dropped by the
// caller — we do not double-register.
func Register(r *shell.Router, deps Deps) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("agentspage: load templates: %w", err)
	}
	h := &handler{router: r, tmpl: tmpl, deps: deps}
	return r.RegisterHome(shell.Home{
		Slug:   "agents",
		Label:  "Agents",
		Href:   "/agents",
		Render: h.handle,
		Items: []shell.ItemRoute{
			{Pattern: "GET /agents/proposals", Render: h.renderProposals},
			{Pattern: "GET /agents/scheduled", Render: h.renderScheduled},
			{Pattern: "GET /agents/automations", Render: h.renderAutomations},
			{Pattern: "GET /agents/health", Render: h.renderHealth},
			{Pattern: "GET /agents/credentials", Render: h.renderCredentials},
			// Per-session detail. No live session ledger yet — renders
			// a clearly-marked coming-soon stub with the requested id.
			{Pattern: "GET /agents/session/{id}", Render: h.renderSessionDetail},
		},
	})
}

// SectionFragment returns the rendered HTML for a single Agents
// section. Exposed for use by the SSE fragment endpoints in
// internal/serve/api. Output matches the initial render exactly so
// client-side replacement preserves layout.
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

// chatInputFor returns the inline chat-input config for the Agents
// home. Variant is "inline" — ambient, never the primary affordance
// (that role stays on Now). Per polish-v2 Fix 5, the empty-state
// notice is NOT paired here.
func (h *handler) chatInputFor(activeSlug string) shell.ChatInput {
	chips := []shell.ChatContextChip{{Kind: "page", Label: "page: /agents"}}
	if activeSlug != "" && activeSlug != "sessions" {
		chips = append(chips, shell.ChatContextChip{Kind: "view", Label: "view: " + activeSlug})
	}
	return shell.ChatInput{
		Variant:     "inline",
		Placeholder: "Ask Hero about agents…",
		Context:     chips,
	}
}

// renderHeroAndChat writes the page-hero followed by the inline chat-
// input fragment, in that order, into w. Centralizes the Fix-5
// placement contract: chat-input renders immediately below the hero
// on every Agents view.
func (h *handler) renderHeroAndChat(out io.Writer, hero shell.PageHero, activeSlug string) error {
	if err := h.router.RenderFragment(out, "page-hero", hero); err != nil {
		return err
	}
	return h.router.RenderFragment(out, "chat-input", h.chatInputFor(activeSlug))
}

func (h *handler) handle(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	sessions := h.loadSessions(ed)

	page := h.buildPage(req, ed, sessions, "sessions")
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "agentspage: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// buildPage assembles the Page envelope. inner is the body content
// renderer (the page-hero + metric-strip are always layered above).
func (h *handler) buildPage(req *http.Request, ed edition.Edition, sessions data.Sessions, activeSlug string) shell.Page {
	return h.buildPageWith(req, ed, sessions, activeSlug, func(out io.Writer) error {
		return h.tmpl.ExecuteTemplate(out, "page.html", pageData{Sessions: sessions})
	})
}

// buildPageWith is the shared compositor used by every Agents route.
// Each sub-route swaps `inner` for the body that view should render.
func (h *handler) buildPageWith(req *http.Request, ed edition.Edition, sessions data.Sessions, activeSlug string, inner func(io.Writer) error) shell.Page {
	hero := h.buildPageHero(ed, sessions)
	strip := h.buildMetricStrip(sessions)
	subNav := h.buildSubNav(ed, sessions, activeSlug)

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, activeSlug); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return inner(out)
	}

	return shell.Page{
		ActiveHome: "agents",
		PageTitle:  "Agents · Hero",
		Content:    content,
		HeadExtra:  template.HTML(agentsStyles + agentsScript),
		SubNav:     subNav,
	}
}

// renderProposals handles GET /agents/proposals — full approvals view
// using the existing approvals.html partial.
func (h *handler) renderProposals(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	sessions := h.loadSessions(ed)
	page := h.buildPageWith(req, ed, sessions, "proposals", func(out io.Writer) error {
		return h.tmpl.ExecuteTemplate(out, "approvals.html", sessions)
	})
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "agentspage: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// renderScheduled handles GET /agents/scheduled — uses the existing
// scheduled-preview partial as the section body.
func (h *handler) renderScheduled(w http.ResponseWriter, req *http.Request) {
	ed := edition.Resolve()
	sessions := h.loadSessions(ed)
	page := h.buildPageWith(req, ed, sessions, "scheduled", func(out io.Writer) error {
		return h.tmpl.ExecuteTemplate(out, "scheduled-preview.html", sessions)
	})
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "agentspage: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) renderAutomations(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "automations", "Automations",
		"Automation rule listing + edit lands once the rules pipeline exposes its config over HTTP.")
}

func (h *handler) renderHealth(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "health", "Health",
		"Adapter health + rate-limit / quota status surfaces once adapters report their state.")
}

func (h *handler) renderCredentials(w http.ResponseWriter, req *http.Request) {
	h.renderStub(w, req, "credentials", "Credentials",
		"Credentials management is a team / cloud / enterprise feature; the local edition surfaces a locked state.")
}

// renderSessionDetail handles GET /agents/session/{id}. The workspace
// has no live session ledger yet, so the detail page renders the
// shared coming-soon stub with the requested id visible in the title
// + subhead. When the ledger lands the stub flips to a real session
// transcript view.
func (h *handler) renderSessionDetail(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if id == "" {
		http.NotFound(w, req)
		return
	}
	ed := edition.Resolve()
	sessions := h.loadSessions(ed)

	hero := shell.PageHero{
		Eyebrow: template.HTML(htmlEscape(fmt.Sprintf("hero · %s · agents", firstNonEmpty(h.deps.Branch, "main")))),
		Title:   "Session " + id,
		Subhead: template.HTML("No live session store connected &mdash; sessions render here once hero-code emits live events."),
		Actions: []shell.PageHeroAction{
			{Kind: "ghost", Label: "Back to Sessions", Href: "/agents"},
		},
	}
	strip := h.buildMetricStrip(sessions)
	subNav := h.buildSubNav(ed, sessions, "sessions")

	content := func(out io.Writer) error {
		if err := h.renderHeroAndChat(out, hero, "session:"+id); err != nil {
			return err
		}
		if err := h.router.RenderFragment(out, "tabbed-metric-strip", strip); err != nil {
			return err
		}
		return h.router.RenderFragment(out, "coming-soon", stubData{
			Home: "agents",
			Slug: "session-" + id,
			View: "Session " + id,
			Note: "This view will render live transcript, tool calls, and cost ticker once the runner emits live events.",
		})
	}
	page := shell.Page{
		ActiveHome: "agents",
		PageTitle:  "Agents · Session " + id + " · Hero",
		SubNav:     subNav,
		Content:    content,
		HeadExtra:  template.HTML(agentsStyles + agentsScript),
	}
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "agentspage: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

// renderStub renders the home chrome + sub-nav with the shared coming-
// soon shell card in the body.
func (h *handler) renderStub(w http.ResponseWriter, req *http.Request, slug, view, note string) {
	ed := edition.Resolve()
	sessions := h.loadSessions(ed)
	page := h.buildPageWith(req, ed, sessions, slug, func(out io.Writer) error {
		return h.router.RenderFragment(out, "coming-soon", stubData{
			Home: "agents", Slug: slug, View: view, Note: note,
		})
	})
	if err := h.router.RenderPage(w, req, page); err != nil {
		http.Error(w, "agentspage: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

type stubData struct {
	Home string
	Slug string
	View string
	Note string
}

// loadSessions composes the sessions-view payload, wiring approvals
// and the scheduled/automations preview totals through the same
// dependency hooks.
func (h *handler) loadSessions(ed edition.Edition) data.Sessions {
	s := data.LoadSessions(data.SessionsInputs{
		ProjectRoot:  h.deps.ProjectRoot,
		HeroDir:      h.deps.HeroDir,
		Edition:      string(ed),
		UserName:     h.deps.UserName,
		LiveSessions: h.deps.LiveSessions,
	})

	// Layer proposals over the empty defaults from LoadSessions.
	p := data.LoadProposals(data.ProposalsInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		Edition:     string(ed),
		Proposals:   h.deps.Proposals,
	})
	s.Approvals = p.Rows
	s.ApprovalsCount = p.Awaiting

	// Layer scheduled/automations totals + preview rows. The preview
	// pulls the first three of each list.
	sched := data.LoadScheduled(data.ScheduledInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		Tasks:       h.deps.ScheduledTasks,
	})
	auto := data.LoadAutomations(data.AutomationsInputs{
		ProjectRoot: h.deps.ProjectRoot,
		HeroDir:     h.deps.HeroDir,
		Rules:       h.deps.AutomationRules,
	})
	s.ScheduledTotal = sched.Total
	s.AutomationTotal = auto.Total
	s.NextScheduled = data.CompactScheduledPreview(sched, 3)
	s.TopAutomations = data.CompactAutomationsPreview(auto, 3)

	// Recompute hero subhead with the now-known approval count.
	s.PendingLabel = formatPending(s.ApprovalsCount)
	return s
}

// renderSection produces the standalone HTML fragment for one section,
// for the SSE fragment-replacement endpoints. Returns an error for
// unknown section names so callers can 404 cleanly.
func (h *handler) renderSection(section string) ([]byte, error) {
	ed := edition.Resolve()
	sessions := h.loadSessions(ed)

	var tplName string
	switch section {
	case "sessions":
		tplName = "sessions.html"
	case "approvals":
		tplName = "approvals.html"
	case "completed":
		tplName = "completed.html"
	case "scheduled-preview":
		tplName = "scheduled-preview.html"
	default:
		return nil, fmt.Errorf("unknown section %q", section)
	}

	var buf strings.Builder
	if err := h.tmpl.ExecuteTemplate(&buf, tplName, sessions); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// pageData is the outer-template input.
type pageData struct {
	Sessions data.Sessions
}

// buildSubNav composes the six-tab sub-nav row. activeSlug picks the
// active tab; valid slugs are sessions|proposals|scheduled|automations|
// health|credentials. `Credentials` renders locked under the local
// edition with the "team server only" meta.
func (h *handler) buildSubNav(ed edition.Edition, s data.Sessions, activeSlug string) *shell.SubNav {
	credVariant := ""
	credMeta := ""
	if ed == edition.Local {
		credVariant = "locked"
		credMeta = "Visible on team, cloud, enterprise editions"
	}
	tabs := []shell.SubNavTab{
		{Label: "Sessions", Href: "/agents", Active: activeSlug == "sessions", Badge: badgeStr(s.LiveCount)},
		{Label: "Proposals", Href: "/agents/proposals", Active: activeSlug == "proposals", Badge: badgeStr(s.ApprovalsCount), Variant: amberIf(s.ApprovalsCount > 0)},
		{Label: "Scheduled", Href: "/agents/scheduled", Active: activeSlug == "scheduled", Badge: badgeStr(s.ScheduledTotal)},
		{Label: "Automations", Href: "/agents/automations", Active: activeSlug == "automations", Badge: badgeStr(s.AutomationTotal)},
		{Label: "Health", Href: "/agents/health", Active: activeSlug == "health"},
		{Label: "Credentials", Href: "/agents/credentials", Active: activeSlug == "credentials", Variant: credVariant, LockMeta: credMeta},
	}
	return &shell.SubNav{Tabs: tabs}
}

// buildPageHero composes the page-hero data block from current counts.
func (h *handler) buildPageHero(ed edition.Edition, s data.Sessions) shell.PageHero {
	branch := h.deps.Branch
	if branch == "" {
		branch = "main"
	}
	eyebrow := fmt.Sprintf("hero · %s · agents", branch)

	subParts := []string{
		fmt.Sprintf("<strong>%d live</strong>", s.LiveCount),
		fmt.Sprintf("%d today", s.CompletedCount),
		fmt.Sprintf("<strong>%s</strong> spent today", htmlEscape(s.SpendTodayValue)),
		fmt.Sprintf("<strong>%d awaiting your approval</strong>", s.ApprovalsCount),
	}
	subhead := strings.Join(subParts, `<span class="dot-sep">·</span>`)

	actions := []shell.PageHeroAction{
		{Kind: "primary", Label: "Start session", Href: "#"},
	}
	if s.ApprovalsCount > 0 {
		actions = append(actions, shell.PageHeroAction{
			Kind: "ghost", Label: fmt.Sprintf("Approve all pending (%d)", s.ApprovalsCount), Href: "/agents/proposals",
		})
	}
	actions = append(actions, shell.PageHeroAction{Kind: "ghost", Label: "Pause my agents", Href: "#"})

	_ = ed
	return shell.PageHero{
		Eyebrow: template.HTML(htmlEscape(eyebrow)),
		Title:   "Sessions",
		Subhead: template.HTML(subhead),
		Actions: actions,
	}
}

// buildMetricStrip projects the data.MetricStrip into the shell's
// shared metric-strip shape.
func (h *handler) buildMetricStrip(s data.Sessions) shell.MetricStrip {
	out := shell.MetricStrip{AllLink: "#"}
	for _, t := range s.Metric.Tabs {
		out.Tabs = append(out.Tabs, shell.MetricTab{
			Slug:   t.Slug,
			Label:  t.Label,
			Active: t.Active,
			Tiles:  t.Tiles,
		})
	}
	return out
}

// formatPending returns the "N awaiting your approval" string with
// singular/plural agreement.
func formatPending(n int) string {
	if n == 1 {
		return "1 awaiting your approval"
	}
	return fmt.Sprintf("%d awaiting your approval", n)
}

// badgeStr returns the string form of a count, or "" when zero (which
// causes the sub-nav template to render no badge at all).
func badgeStr(n int) string {
	if n <= 0 {
		return ""
	}
	return itoa(n)
}

// amberIf returns "amber" when the predicate is true, otherwise "".
func amberIf(cond bool) string {
	if cond {
		return "amber"
	}
	return ""
}

// firstNonEmpty returns the first non-empty string from the args, or "".
func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// itoa is a small local int-to-string helper that keeps this file's
// imports minimal (avoiding a strconv pull for one call site).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// htmlEscape is a thin wrapper around template.HTMLEscapeString to
// keep call sites short.
func htmlEscape(s string) string { return template.HTMLEscapeString(s) }
