package now

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
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
		`class="metric-tab`,
		`Tell Hero what to do next`,
		`now-launch-input`,
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q", want)
		}
	}
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
