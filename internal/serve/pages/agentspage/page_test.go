package agentspage

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

	resp, err := http.Get(srv.URL + "/agents")
	if err != nil {
		t.Fatalf("GET /agents: %v", err)
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
		// Sub-nav fragment, default tab Sessions active.
		`class="subnav`,
		`subnav-tab active`,
		`Sessions`,
		`Proposals`,
		`Scheduled`,
		`Automations`,
		// Stable section ids for SSE fragment swap.
		`id="agents-sessions"`,
		`id="agents-approvals"`,
		`id="agents-completed"`,
		`id="agents-scheduled-preview"`,
		// Metric strip rendered with the live-now pane active.
		`class="metric-tab`,
		`Right now`,
		// Per v5 Fix 3: /agents root page-title is "Agents", consistent
		// with sibling homes. Sub-route titles still render as
		// "Agents · <Sub>" (covered in TestAgentsSubRouteTitle below).
		`class="page-title">Agents</h1>`,
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
}

func TestSectionFragment_RendersStandalone(t *testing.T) {
	deps := Deps{UserName: "test-user"}
	for _, section := range []string{"sessions", "approvals", "completed", "scheduled-preview"} {
		body, err := SectionFragment(deps, section)
		if err != nil {
			t.Errorf("SectionFragment(%q): %v", section, err)
			continue
		}
		if !strings.Contains(string(body), `id="agents-`+section+`"`) {
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

// TestAgentsRootTitleIsAgents pins the v5 Fix 3 contract: the home root
// page-title renders "Agents" (was "Sessions" pre-v5) and NOT
// "Agents · <Sub>". Sub-routes still get the dotted form (separate test).
func TestAgentsRootTitleIsAgents(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/agents")
	if err != nil {
		t.Fatalf("GET /agents: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	body := string(raw)

	if !strings.Contains(body, `class="page-title">Agents</h1>`) {
		t.Errorf("/agents root missing `<h1>Agents</h1>` page-title")
	}
	if strings.Contains(body, `class="page-title">Sessions</h1>`) {
		t.Errorf("/agents root page-title still says 'Sessions' (v5 Fix 3 regression)")
	}
	if strings.Contains(body, `class="page-title">Agents · `) {
		t.Errorf("/agents root should not render the 'Agents · <Sub>' form")
	}
}

// TestAgentsSubRouteTitleStillDotted asserts that the v5 Fix 3 change
// does NOT regress the sub-route title format. `/agents/proposals`
// still renders `<h1>Agents · Proposals</h1>`.
func TestAgentsSubRouteTitleStillDotted(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/agents/proposals")
	if err != nil {
		t.Fatalf("GET /agents/proposals: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	body := string(raw)

	if !strings.Contains(body, `Agents · Proposals`) {
		t.Errorf("/agents/proposals missing `Agents · Proposals` title (v5 Fix 3 regression)")
	}
}

func TestSubRoutes_ReturnTwoHundred(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	cases := []struct {
		path     string
		mustHave string
	}{
		{"/agents/proposals", `id="agents-approvals"`},
		{"/agents/scheduled", `id="agents-scheduled-preview"`},
		{"/agents/automations", `coming soon`},
		{"/agents/health", `coming soon`},
		{"/agents/credentials", `coming soon`},
	}
	for _, tc := range cases {
		resp, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Errorf("GET %s: %v", tc.path, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", tc.path, resp.StatusCode)
			resp.Body.Close()
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(raw), tc.mustHave) {
			t.Errorf("GET %s: missing %q", tc.path, tc.mustHave)
		}
	}
}
