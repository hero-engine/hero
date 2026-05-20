package projectpage

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProjectPage_MissingPathBanner_FiresWhenRootGone(t *testing.T) {
	// Simulate /p/<slug>/project against a registered project whose
	// path no longer exists on disk.
	missing := filepath.Join(t.TempDir(), "gone")
	heroDir := filepath.Join(missing, ".hero")
	// Intentionally do NOT create either directory — we want os.Stat
	// to return ErrNotExist for ProjectRoot.

	r := newTestRouter(t, Deps{
		ProjectRoot: missing,
		HeroDir:     heroDir,
		Slug:        "gone-project",
		RegistryEntry: &RegistryEntry{
			Path:         missing,
			RegisteredAt: time.Now(),
		},
		IsFallbackProject: false,
	})

	req := httptest.NewRequest("GET", "/project", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `class="project-missing-path-banner"`) {
		t.Errorf("expected missing-path banner div in body")
	}
	if !strings.Contains(rec.Body.String(), "data-banner=\"missing-path\"") {
		t.Errorf("expected banner deregister button with data-banner attribute")
	}
}

func TestProjectPage_MissingPathBanner_SuppressedOnFallback(t *testing.T) {
	// Single-project fallback: IsFallbackProject=true. Even though the
	// path is missing, the banner must NOT render.
	missing := filepath.Join(t.TempDir(), "also-gone")
	heroDir := filepath.Join(missing, ".hero")

	r := newTestRouter(t, Deps{
		ProjectRoot: missing,
		HeroDir:     heroDir,
		Slug:        "also-gone",
		RegistryEntry: &RegistryEntry{
			Path:         missing,
			RegisteredAt: time.Now(),
		},
		IsFallbackProject: true,
	})

	req := httptest.NewRequest("GET", "/project", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `class="project-missing-path-banner"`) {
		t.Errorf("missing-path banner must NOT render on fallback project")
	}
}

func TestProjectPage_DangerZone_RegisteredCollapsedByDefault(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	r := newTestRouter(t, Deps{
		ProjectRoot: dir,
		HeroDir:     heroDir,
		Slug:        "fixture",
		RegistryEntry: &RegistryEntry{
			Path:         dir,
			RegisteredAt: time.Now(),
		},
	})
	req := httptest.NewRequest("GET", "/project", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `data-section="danger"`) {
		t.Fatalf("danger section missing from body")
	}
	if !strings.Contains(body, `data-default-collapsed="true"`) {
		t.Errorf("danger section should default to collapsed")
	}
	if !strings.Contains(body, `project-danger-form`) {
		t.Errorf("danger form should be present in markup")
	}
	if !strings.Contains(body, `class="project-danger-submit"`) {
		t.Errorf("danger submit button should be present")
	}
	// Submit button must be rendered disabled (typed-confirm gates it).
	if !strings.Contains(body, `class="project-danger-submit"`) || !strings.Contains(body, `disabled`) {
		t.Errorf("danger submit button should be rendered with the disabled attribute")
	}
}

func TestProjectPage_DangerZone_HiddenWhenNotRegistered(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// No RegistryEntry → not registered.
	r := newTestRouter(t, Deps{
		ProjectRoot: dir,
		HeroDir:     heroDir,
		Slug:        "fixture",
	})
	req := httptest.NewRequest("GET", "/project", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `data-section="danger"`) {
		t.Errorf("danger section must be hidden when project is not registered")
	}
}

func TestProjectPage_RegistryRemoveButton_PresentWhenRegistered(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	r := newTestRouter(t, Deps{
		ProjectRoot: dir,
		HeroDir:     heroDir,
		Slug:        "fixture",
		RegistryEntry: &RegistryEntry{
			Path:         dir,
			RegisteredAt: time.Now(),
		},
	})
	req := httptest.NewRequest("GET", "/project", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "Remove from registry") {
		t.Errorf("expected Remove button text in body")
	}
	if !strings.Contains(body, `class="project-registry-remove"`) {
		t.Errorf("expected project-registry-remove class")
	}
}

func TestProjectPage_RegistryRemoveButton_AbsentWhenNotRegistered(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	r := newTestRouter(t, Deps{ProjectRoot: dir, HeroDir: heroDir, Slug: "fixture"})
	req := httptest.NewRequest("GET", "/project", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), "Remove from registry") {
		t.Errorf("Remove button must not render when project unregistered")
	}
}
