package work

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

	resp, err := http.Get(srv.URL + "/work")
	if err != nil {
		t.Fatalf("GET /work: %v", err)
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

	// The page must render the page chrome, metric strip, view toolbar,
	// and each section's stable id so SSE fragment swaps can locate
	// them — even on an empty workspace.
	mustContain := []string{
		`<nav class="topnav">`,
		// view-toolbar is gone from the default /work view (Fix 2 in
		// hero-surface-polish-v2); rm-filters is the sole filter UI.
		`class="rm-filters"`,
		`id="work-roadmap"`,
		`id="work-blocked"`,
		`id="work-shipped"`,
		`class="metric-tab`,
	}
	mustNotContain := []string{
		`class="view-toolbar"`,
		`id="work-toolbar"`,
	}
	for _, bad := range mustNotContain {
		if strings.Contains(body, bad) {
			t.Errorf("/work unexpectedly contains %q (view-toolbar removed in polish-v2)", bad)
		}
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
}

func TestSectionFragment_RendersStandalone(t *testing.T) {
	deps := Deps{UserName: "test-user"}
	for _, section := range []string{"roadmap", "blocked", "shipped", "toolbar"} {
		body, err := SectionFragment(deps, section)
		if err != nil {
			t.Errorf("SectionFragment(%q): %v", section, err)
			continue
		}
		if !strings.Contains(string(body), `id="work-`+section+`"`) {
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
		{"/work/blocked", `id="work-toolbar"`},
		{"/work/kanban", `coming soon`},
		{"/work/graph", `coming soon`},
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
