package now

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve/chat"
	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/session"
	"github.com/hero-engine/hero/internal/serve/shell"
)

func newTestRouter(t *testing.T) *shell.Router {
	t.Helper()
	store, err := session.Open(t.TempDir() + "/shell-sessions.db")
	if err != nil {
		t.Fatalf("session.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return shell.New(edition.Local, store, "hero", "main", "test-user", "test")
}

func TestRegister_RendersAllSections(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{
		Workspace: "hero",
		Branch:    "main",
		UserName:  "test-user",
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/now")
	if err != nil {
		t.Fatalf("GET /now: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	body := string(raw)

	mustContain := []string{
		`<nav class="topnav">`,
		`id="now-inbox"`,
		`id="now-plate"`,
		`id="now-agents"`,
		`id="now-changes"`,
		`id="now-quicklaunch"`,
		`class="metric-tab`,
		`Tell Hero what to do next`,
		// hero-now-home-followups Fix 1: Quick launch mounts the shell-
		// owned chat-input fragment instead of the hand-rolled <input>.
		`data-chat-input-variant="hero"`,
		// Fix 2: with no chat adapter wired (default in this test) the
		// empty-state notice renders above the chat input.
		`empty-state-notice`,
		`Hero needs hero-code`,
		// Fix 3: the page-hero subhead is wrapped in a stable DOM hook
		// so the `event: hero` SSE channel can swap it in place.
		`data-page-hero-subhead`,
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
}

func TestRegister_NoEmptyStateWhenAdapterConnected(t *testing.T) {
	r := newTestRouter(t)
	reg := chat.NewRegistry()
	if err := reg.Register("test-adapter", &fakeAdapter{}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	if err := Register(r, Deps{
		Workspace:    "hero",
		Branch:       "main",
		UserName:     "test-user",
		ChatRegistry: reg,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/now")
	if err != nil {
		t.Fatalf("GET /now: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	body := string(raw)

	// The chat-input fragment must still be present, but the no-adapter
	// empty-state notice must not.
	if !strings.Contains(body, `data-chat-input-variant="hero"`) {
		t.Errorf("chat-input fragment missing when adapter connected")
	}
	if strings.Contains(body, `Hero needs hero-code`) {
		t.Errorf("empty-state notice rendered even though adapter is connected")
	}
}

// Spec dashboard-now-headline-misleading-when-empty: the subhead
// composition no longer joins "no agent running" with "since X ago"
// — that pairing implied a quiet workspace even when sparse events
// just reflected the underemission bug.
func TestSubheadPlainText_Cases(t *testing.T) {
	cases := []struct {
		name         string
		inboxCount   int
		runningCount int
		lastActive   string
		want         string
	}{
		// AC3: everything empty → single truthful chip, no false "since"
		{"all-empty", 0, 0, "", "no live activity right now"},
		// AC1: empty + stale event → must NOT compose "no agent · since"
		{"no-agent-stale-event", 0, 0, "19h ago", "no live activity right now"},
		// AC2 (negative side): no agent + inbox → since clause omitted
		{"inbox-only", 3, 0, "19h ago", "3 need your input · no agent running"},
		// AC2 (positive side): running agent → since clause keeps
		{"running-with-since", 0, 1, "5m ago", "1 agent running · since 5m ago"},
		{"running-without-since", 0, 1, "", "1 agent running"},
		// Mixed: inbox + running + lastActive → fully composed
		{"all-three", 2, 1, "12m ago", "2 need your input · 1 agent running · since 12m ago"},
		// One inbox item — singular phrasing
		{"single-inbox-running", 1, 1, "1m ago", "1 needs your input · 1 agent running · since 1m ago"},
	}
	for _, c := range cases {
		got := subheadPlainText(c.inboxCount, c.runningCount, c.lastActive)
		if got != c.want {
			t.Errorf("%s: subheadPlainText(%d,%d,%q) = %q, want %q",
				c.name, c.inboxCount, c.runningCount, c.lastActive, got, c.want)
		}
	}
}

// buildPageHero must apply the same rule as subheadPlainText so the
// SSE refresh and the initial render never disagree.
func TestBuildPageHero_NoMisleadingSinceClause(t *testing.T) {
	hero := buildPageHero(Deps{}, edition.Local, 0, 0, "19h ago")
	subhead := string(hero.Subhead)
	if strings.Contains(subhead, "since") {
		t.Errorf("empty workspace subhead should not include 'since': %q", subhead)
	}
	if !strings.Contains(subhead, "no live activity right now") {
		t.Errorf("expected fallback subhead, got: %q", subhead)
	}

	hero = buildPageHero(Deps{}, edition.Local, 0, 1, "5m ago")
	subhead = string(hero.Subhead)
	if !strings.Contains(subhead, "since 5m ago") {
		t.Errorf("running-agent subhead should include 'since': %q", subhead)
	}

	hero = buildPageHero(Deps{}, edition.Local, 2, 0, "19h ago")
	subhead = string(hero.Subhead)
	if strings.Contains(subhead, "since") {
		t.Errorf("no-running subhead should not include 'since' even with inbox: %q", subhead)
	}
}

// Spec dashboard-adapter-state-hardcoded: Now's chat input must honor
// the same adapter probe as the four other homes. Previously
// buildChatInput always returned Disabled:false, so the install banner
// could render alongside a chat input that looked usable.
func TestBuildChatInput_DisablesWhenNoAdapter(t *testing.T) {
	got := buildChatInput(true)
	if !got.Disabled {
		t.Error("chat input should be Disabled when noAdapter=true")
	}
	if got.ConnectHref == "" {
		t.Error("disabled chat input should carry a ConnectHref")
	}
	if !strings.Contains(got.Placeholder, "Connect a chat adapter") {
		t.Errorf("placeholder = %q, want disabled-state copy", got.Placeholder)
	}
}

func TestBuildChatInput_EnabledWhenAdapterConnected(t *testing.T) {
	got := buildChatInput(false)
	if got.Disabled {
		t.Error("chat input should not be Disabled when noAdapter=false")
	}
	if got.ConnectHref != "" {
		t.Errorf("enabled input should not set ConnectHref, got %q", got.ConnectHref)
	}
	if got.Placeholder == "" {
		t.Error("enabled input should keep its default placeholder")
	}
}

// fakeAdapter is a minimal chat.HeroAdapter implementation for tests.
// Reports interactive capability so chat.Resolve picks it.
type fakeAdapter struct{}

func (fakeAdapter) Name() string       { return "fake" }
func (fakeAdapter) Version() string    { return "0.0.0" }
func (fakeAdapter) Kinds() []chat.Kind { return []chat.Kind{chat.KindInteractive} }
func (fakeAdapter) Close() error       { return nil }
func (fakeAdapter) Stream(ctx context.Context, req chat.DispatchRequest) (<-chan chat.Event, error) {
	ch := make(chan chat.Event)
	close(ch)
	return ch, nil
}

func TestResolveMethodology(t *testing.T) {
	cases := map[string]string{
		"":          "solo",
		"scrum":     "scrum",
		"SHAPE-UP":  "shape-up",
		"kanban":    "kanban",
		"solo":      "solo",
		"waterfall": "solo", // unknown → solo
	}
	for in, want := range cases {
		got := resolveMethodology(config.Config{Methodology: in})
		if got != want {
			t.Errorf("resolveMethodology(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstTabFor(t *testing.T) {
	cases := []struct {
		methodology string
		wantSlug    string
		wantLabel   string
	}{
		{"scrum", "sprint", "This sprint"},
		{"shape-up", "cycle", "This cycle"},
		{"kanban", "week", "This week"},
		{"solo", "week", "This week"},
		{"", "week", "This week"},
	}
	for _, c := range cases {
		slug, label := firstTabFor(c.methodology)
		if slug != c.wantSlug || label != c.wantLabel {
			t.Errorf("firstTabFor(%q) = (%q, %q), want (%q, %q)",
				c.methodology, slug, label, c.wantSlug, c.wantLabel)
		}
	}
}

func TestSectionFragment_RendersStandalone(t *testing.T) {
	deps := Deps{UserName: "test-user"}
	for _, section := range []string{"inbox", "plate", "agents", "changes"} {
		body, err := SectionFragment(deps, section)
		if err != nil {
			t.Errorf("SectionFragment(%q): %v", section, err)
			continue
		}
		if !strings.Contains(string(body), `id="now-`+section+`"`) {
			t.Errorf("section %q output missing id; got %q", section, string(body))
		}
	}
}

func TestSectionFragment_UnknownSection(t *testing.T) {
	_, err := SectionFragment(Deps{}, "nope")
	if err == nil {
		t.Errorf("expected error for unknown section, got nil")
	}
}
