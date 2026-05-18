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

	// The page must render the page chrome, metric strip, view toolbar
	// (back on root per v4 Fix 1, with Horizons active), and each
	// section's stable id so SSE fragment swaps can locate them — even
	// on an empty workspace.
	mustContain := []string{
		`<nav class="topnav">`,
		`class="rm-filters"`,
		`class="view-toolbar"`,
		`id="work-toolbar"`,
		`class="view-tab active">Horizons</a>`,
		`id="work-roadmap"`,
		`id="work-blocked"`,
		`id="work-shipped"`,
		`class="metric-tab`,
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
}

// TestRegister_ViewToolbarActiveStateMatchesRoute asserts the v4 Fix 1
// contract: each Work sub-route renders the view-toolbar with exactly
// one matching tab marked `active`. (Pre-v4 the active class was hard-
// coded to Horizons on every route.)
func TestRegister_ViewToolbarActiveStateMatchesRoute(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	cases := []struct {
		path string
		want string
	}{
		{"/work", `class="view-tab active">Horizons</a>`},
		{"/work/kanban", `class="view-tab active">Kanban</a>`},
		{"/work/graph", `class="view-tab active">Graph</a>`},
		{"/work/blocked", `class="view-tab active">Blocked`},
	}
	for _, tc := range cases {
		resp, err := http.Get(srv.URL + tc.path)
		if err != nil {
			t.Errorf("GET %s: %v", tc.path, err)
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		body := string(raw)
		if !strings.Contains(body, tc.want) {
			t.Errorf("GET %s: missing active marker %q", tc.path, tc.want)
		}
		// And no other tab is active on the same page: count the
		// occurrences of `view-tab active` — must be exactly one.
		if got := strings.Count(body, `view-tab active`); got != 1 {
			t.Errorf("GET %s: expected exactly 1 active view-tab, got %d", tc.path, got)
		}
	}
}

func TestSectionFragment_RendersStandalone(t *testing.T) {
	deps := Deps{UserName: "test-user"}
	for _, section := range []string{"roadmap", "blocked", "shipped", "toolbar"} {
		body, err := SectionFragment(deps, section, "")
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
	_, err := SectionFragment(Deps{}, "nope", "")
	if err == nil {
		t.Errorf("expected error for unknown section, got nil")
	}
}

// TestRegister_ViewToolbarEmptyActiveRendersNoActiveTab pins the v5
// Fix 6 contract for toolbarData.Active's zero value: rendering the
// view-toolbar fragment with Active="" produces zero `view-tab active`
// strings. New callers can rely on this "no tab active" fallback
// instead of accidentally highlighting Horizons.
func TestRegister_ViewToolbarEmptyActiveRendersNoActiveTab(t *testing.T) {
	tmpl, err := loadTemplates()
	if err != nil {
		t.Fatalf("loadTemplates: %v", err)
	}
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "view-toolbar.html", toolbarData{Active: ""}); err != nil {
		t.Fatalf("execute view-toolbar.html: %v", err)
	}
	out := buf.String()
	if got := strings.Count(out, "view-tab active"); got != 0 {
		t.Errorf("toolbarData{Active:\"\"} rendered %d `view-tab active`, want 0\nrendered:\n%s", got, out)
	}
	// Sanity: the toolbar itself should still render (tabs present, just
	// none active).
	if !strings.Contains(out, "view-toolbar") {
		t.Errorf("toolbar fragment empty / missing view-toolbar marker:\n%s", out)
	}
}

// TestSectionFragment_ToolbarHonorsViewParam pins v5 Fix 4: the toolbar
// fragment renders with the supplied view as the active tab. Empty
// view defaults to horizons.
func TestSectionFragment_ToolbarHonorsViewParam(t *testing.T) {
	deps := Deps{UserName: "test-user"}
	cases := []struct {
		view string
		want string
	}{
		{"", `class="view-tab active">Horizons</a>`},
		{"horizons", `class="view-tab active">Horizons</a>`},
		{"kanban", `class="view-tab active">Kanban</a>`},
		{"graph", `class="view-tab active">Graph</a>`},
		{"blocked", `class="view-tab active">Blocked`},
	}
	for _, tc := range cases {
		body, err := SectionFragment(deps, "toolbar", tc.view)
		if err != nil {
			t.Errorf("SectionFragment(toolbar, %q): %v", tc.view, err)
			continue
		}
		if !strings.Contains(string(body), tc.want) {
			t.Errorf("view=%q: missing active marker %q\nbody:\n%s", tc.view, tc.want, string(body))
		}
		// Exactly one active tab.
		if got := strings.Count(string(body), "view-tab active"); got != 1 {
			t.Errorf("view=%q: expected 1 active view-tab, got %d", tc.view, got)
		}
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
