package rollup

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/shell"
)

// newTestRouter builds a shell router and registers the project home
// rooted at the supplied repo dir.
func newTestRouter(t *testing.T, root string) *shell.Router {
	t.Helper()
	r := shell.New(edition.Local, nil, "test", "main", "tester", "v0")
	shell.RegisterStubHomes(r)
	if err := Register(r, Deps{
		ProjectRoot: root,
		HeroDir:     filepath.Join(root, ".hero"),
		Workspace:   "test",
		Branch:      "main",
		UserName:    "tester",
	}); err != nil {
		t.Fatalf("register rollup: %v", err)
	}
	return r
}

func TestRollupHome_Returns200(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/foo\n")
	mustWrite(t, filepath.Join(dir, "cmd", "x", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(dir, "internal", "x", "x.go"), "package x\n")
	mustMkdir(t, filepath.Join(dir, ".hero"))

	r := newTestRouter(t, dir)
	handler := r.Handler()

	req := httptest.NewRequest("GET", "/rollup", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Surfaces") {
		t.Errorf("response missing Surfaces; body: %s", rec.Body.String())
	}
}

func TestRollupSurfaceDetail_404OnUnknown(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/foo\n")
	mustMkdir(t, filepath.Join(dir, "internal"))
	mustMkdir(t, filepath.Join(dir, ".hero"))

	r := newTestRouter(t, dir)
	handler := r.Handler()

	req := httptest.NewRequest("GET", "/rollup/surface/nope-not-a-surface", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestRollupArchive_404OnMissing(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/foo\n")
	mustMkdir(t, filepath.Join(dir, ".hero"))

	r := newTestRouter(t, dir)
	handler := r.Handler()

	req := httptest.NewRequest("GET", "/rollup/snapshots/2099-01-01", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
