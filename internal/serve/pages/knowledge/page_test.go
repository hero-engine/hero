package knowledge

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

	resp, err := http.Get(srv.URL + "/knowledge")
	if err != nil {
		t.Fatalf("GET /knowledge: %v", err)
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

	// /knowledge defaults to Browse only — Why / Staleness / etc. live
	// at their own item routes.
	mustContain := []string{
		`<nav class="topnav">`,
		`class="subnav"`, // sub-nav row renders
		`id="knowledge-browse"`,
		`Knowledge`,
		`class="metric-tab`,
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
	// The Why / Staleness sections must NOT render on the default Browse
	// route — they have their own item routes now.
	mustNotContain := []string{
		`id="knowledge-provenance"`,
		`id="knowledge-staleness"`,
	}
	for _, bad := range mustNotContain {
		if strings.Contains(body, bad) {
			t.Errorf("/knowledge unexpectedly contains %q (should only render on sub-route)", bad)
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
		path       string
		mustHave   string
		activeLbl  string
	}{
		{"/knowledge/why", `id="knowledge-provenance"`, "Why"},
		{"/knowledge/staleness", `id="knowledge-staleness"`, "Staleness"},
		{"/knowledge/search", `coming soon`, "Search"},
		{"/knowledge/recent", `coming soon`, "Recent"},
		{"/knowledge/write", `coming soon`, "Write"},
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
		body := string(raw)
		if !strings.Contains(body, tc.mustHave) {
			t.Errorf("GET %s: missing %q", tc.path, tc.mustHave)
		}
	}
}

func TestSectionFragment_RendersStandalone(t *testing.T) {
	deps := Deps{UserName: "test-user"}
	for _, section := range []string{"browse", "provenance", "summary", "neighbors", "staleness"} {
		body, err := SectionFragment(deps, section)
		if err != nil {
			t.Errorf("SectionFragment(%q): %v", section, err)
			continue
		}
		if !strings.Contains(string(body), `id="knowledge-`+section+`"`) {
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
