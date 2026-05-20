package projectpage

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/shell"
)

// newTestRouter builds a shell router with the Project section page
// registered. Stub homes provide the rest of the top-nav so active-
// state assertions have something to compare against.
func newTestRouter(t *testing.T, deps Deps) *shell.Router {
	t.Helper()
	r := shell.New(edition.Local, nil, "test", "main", "tester", "v0")
	shell.RegisterStubHomes(r)
	if err := Register(r, deps); err != nil {
		t.Fatalf("register projectpage: %v", err)
	}
	return r
}

func mustWriteSpec(t *testing.T, root, relpath, body string) {
	t.Helper()
	abs := filepath.Join(root, relpath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectPage_RendersAllSections_WithMissingInputs(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}

	r := newTestRouter(t, Deps{
		ProjectRoot: dir,
		HeroDir:     heroDir,
		Slug:        "fixture",
	})

	req := httptest.NewRequest("GET", "/project", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()

	// Every section's <section data-section="..."> must appear in the
	// rendered page, even when the project has no peers, no tracker,
	// no health, etc.
	wantSections := []string{
		`data-section="identity"`,
		`data-section="health"`,
		`data-section="stack"`,
		`data-section="registry"`,
		`data-section="peers"`,
		`data-section="trackers"`,
		`data-section="knowledge"`,
		`data-section="config"`,
	}
	for _, w := range wantSections {
		if !strings.Contains(body, w) {
			t.Errorf("response missing %q", w)
		}
	}

	// Empty-state copy must render on missing inputs (not crash).
	wantEmptyStates := []string{
		"No peers registered",                    // peers section
		"No tracker configured",                  // trackers section
		"as of: never",                           // health, peers
	}
	for _, w := range wantEmptyStates {
		if !strings.Contains(body, w) {
			t.Errorf("response missing empty-state copy %q", w)
		}
	}
}

func TestProjectPage_SingleProjectFallback(t *testing.T) {
	// In single-project mode, deps.Slug is empty and the handler
	// synthesizes it from the project root's basename.
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}

	r := newTestRouter(t, Deps{
		ProjectRoot: dir,
		HeroDir:     heroDir,
		// Slug intentionally empty.
	})

	req := httptest.NewRequest("GET", "/project", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	// Title should include the project root's basename.
	if !strings.Contains(rec.Body.String(), filepath.Base(dir)) {
		t.Errorf("expected response to include project basename %q", filepath.Base(dir))
	}
}

func TestProjectPage_TopNavActiveState(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	r := newTestRouter(t, Deps{ProjectRoot: dir, HeroDir: heroDir, Slug: "fixture"})

	req := httptest.NewRequest("GET", "/project", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()

	// The Project tab carries href="/project" — the active class
	// should appear on that anchor. Match the rendered chunk that
	// includes the Project label.
	if !strings.Contains(body, `class="nav-tab active"`) {
		t.Errorf("expected at least one active nav-tab in body; got: %s", body)
	}
	// Sanity: the rendered page should contain the Project label.
	if !strings.Contains(body, ">Project<") && !strings.Contains(body, "Project\n") {
		t.Errorf("expected Project label in nav; body: %s", body)
	}
}

func TestProjectPage_404OnUnknownURL(t *testing.T) {
	// The shell router 404s any path that doesn't match a registered
	// home Href / item route. This is the standard "unknown slug at
	// /p/<unknown>/project" behavior at the shell-router boundary —
	// the slug-resolution layer (internal/serve.projectHandler) sits
	// above the shell router and is exercised by serve-level tests.
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	r := newTestRouter(t, Deps{ProjectRoot: dir, HeroDir: heroDir, Slug: "fixture"})

	req := httptest.NewRequest("GET", "/project/does-not-exist", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}
