package people

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

	resp, err := http.Get(srv.URL + "/people")
	if err != nil {
		t.Fatalf("GET /people: %v", err)
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

	// /people defaults to the Pulse view ONLY — ROI sections live at
	// /people/roi.
	mustContain := []string{
		`<nav class="topnav">`,
		`class="subnav"`,
		`id="people-pulse"`,
		`class="metric-tab`,
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
	mustNotContain := []string{
		`id="people-time-spent"`,
		`id="people-savings"`,
		`id="people-trend"`,
		`id="people-contributors"`,
		`id="people-changes"`,
	}
	for _, bad := range mustNotContain {
		if strings.Contains(body, bad) {
			t.Errorf("/people unexpectedly contains %q (should only render on /people/roi)", bad)
		}
	}
}

func TestROIOverviewRoute_RendersFullROI(t *testing.T) {
	r := newTestRouter(t)
	if err := Register(r, Deps{UserName: "test-user"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/people/roi")
	if err != nil {
		t.Fatalf("GET /people/roi: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	body := string(raw)
	for _, want := range []string{
		`id="people-time-spent"`,
		`id="people-savings"`,
		`id="people-trend"`,
		`id="people-contributors"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/people/roi missing %q", want)
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

	paths := []string{
		"/people/activity",
		"/people/handoffs",
		"/people/profiles",
		"/people/roi/velocity",
		"/people/roi/autonomy",
		"/people/roi/knowledge",
		"/people/roi/individual",
		"/people/roi/export",
	}
	for _, p := range paths {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Errorf("GET %s: %v", p, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", p, resp.StatusCode)
			resp.Body.Close()
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(raw), `coming soon`) {
			t.Errorf("GET %s: missing stub marker", p)
		}
	}
}

func TestSectionFragment_RendersStandalone(t *testing.T) {
	deps := Deps{UserName: "test-user"}
	for _, section := range []string{"pulse", "overview"} {
		body, err := SectionFragment(deps, section)
		if err != nil {
			t.Errorf("SectionFragment(%q): %v", section, err)
			continue
		}
		if section == "pulse" && !strings.Contains(string(body), `id="people-pulse"`) {
			t.Errorf("section %q output missing id; got first 200 chars: %.200s", section, string(body))
		}
	}
}

func TestSectionFragment_UnknownSection(t *testing.T) {
	_, err := SectionFragment(Deps{}, "nope")
	if err == nil {
		t.Errorf("expected error for unknown section, got nil")
	}
}

func TestBuildSubNav_PulseDefault(t *testing.T) {
	sn := buildSubNav("pulse")
	if sn == nil || len(sn.Tabs) == 0 {
		t.Fatalf("sub-nav empty")
	}
	var pulseActive bool
	var exportLocked bool
	for _, tab := range sn.Tabs {
		if tab.Label == "Pulse" && tab.Active {
			pulseActive = true
		}
		if tab.Label == "Export" && tab.Variant == "locked" {
			exportLocked = true
		}
	}
	if !pulseActive {
		t.Errorf("Pulse tab not active by default")
	}
	if !exportLocked {
		t.Errorf("Export tab not in locked variant")
	}
}
