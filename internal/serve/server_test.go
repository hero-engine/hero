package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
		Port:        0,
		AutoWatch:   false,
	})

	if srv.port != 7437 {
		t.Errorf("default port = %d, want 7437", srv.port)
	}

	if srv.Bus() == nil {
		t.Error("bus should not be nil")
	}

	// Should have one project registered
	if srv.ProjectCount() != 1 {
		t.Errorf("project count = %d, want 1", srv.ProjectCount())
	}
}

func TestNewServer_MultiProject(t *testing.T) {
	// Create two project directories
	dir1 := filepath.Join(t.TempDir(), "project-alpha")
	dir2 := filepath.Join(t.TempDir(), "project-beta")
	os.MkdirAll(filepath.Join(dir1, ".hero", "specs", "feat-a"), 0o755)
	os.MkdirAll(filepath.Join(dir2, ".hero", "specs", "feat-b"), 0o755)

	// Write minimal specs
	os.WriteFile(filepath.Join(dir1, ".hero", "specs", "feat-a", "spec.md"),
		[]byte("---\ntitle: Feature A\ntype: feature\nstatus: planning\n---\n# Feature A\n"), 0o644)
	os.WriteFile(filepath.Join(dir2, ".hero", "specs", "feat-b", "spec.md"),
		[]byte("---\ntitle: Feature B\ntype: feature\nstatus: delivering\n---\n# Feature B\n"), 0o644)
	os.WriteFile(filepath.Join(dir1, "hero.json"), []byte(`{"directory": ".hero"}`), 0o644)
	os.WriteFile(filepath.Join(dir2, "hero.json"), []byte(`{"directory": ".hero"}`), 0o644)

	// Create a registry
	regPath := filepath.Join(t.TempDir(), "projects.json")
	reg, _ := LoadRegistryFrom(regPath)
	reg.Add(dir1)
	reg.Add(dir2)
	reg.Save()

	// Create server with registry
	srv := NewServer(ServerConfig{
		HeroDir:      filepath.Join(dir1, ".hero"),
		ProjectRoot:  dir1,
		Version:      "test",
		Port:         0,
		RegistryPath: regPath,
	})

	// After loading registry, should have both projects
	srv.loadRegistryProjects()

	if srv.ProjectCount() < 2 {
		t.Errorf("project count = %d, want >= 2", srv.ProjectCount())
	}

	// Both slugs should exist
	if srv.GetProject("project-alpha") == nil {
		t.Error("expected project-alpha")
	}
	if srv.GetProject("project-beta") == nil {
		t.Error("expected project-beta")
	}
}

func TestServer_AddRemoveProject(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
		Port:        0,
	})

	// Create another project
	dir2 := filepath.Join(t.TempDir(), "second-project")
	heroDir2 := filepath.Join(dir2, ".hero")
	os.MkdirAll(filepath.Join(heroDir2, "specs", "test-spec"), 0o755)
	os.WriteFile(filepath.Join(heroDir2, "specs", "test-spec", "spec.md"),
		[]byte("---\ntitle: Test\ntype: feature\nstatus: planning\n---\n# Test\n"), 0o644)
	os.WriteFile(filepath.Join(dir2, "hero.json"), []byte(`{"directory": ".hero"}`), 0o644)

	initialCount := srv.ProjectCount()

	err := srv.AddProject("second-project", dir2, heroDir2, false)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	if srv.ProjectCount() != initialCount+1 {
		t.Errorf("count = %d, want %d", srv.ProjectCount(), initialCount+1)
	}

	// Idempotent
	err = srv.AddProject("second-project", dir2, heroDir2, false)
	if err != nil {
		t.Fatalf("AddProject (idempotent): %v", err)
	}
	if srv.ProjectCount() != initialCount+1 {
		t.Errorf("count after idempotent add = %d, want %d", srv.ProjectCount(), initialCount+1)
	}

	err = srv.RemoveProject("second-project")
	if err != nil {
		t.Fatalf("RemoveProject: %v", err)
	}
	if srv.ProjectCount() != initialCount {
		t.Errorf("count after remove = %d, want %d", srv.ProjectCount(), initialCount)
	}
}

func TestServer_RunAndShutdown(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
		Port:        18437,
		AutoWatch:   false,
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", srv.port))
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("health status = %d, want 200", resp.StatusCode)
	}

	// Test projects endpoint
	resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/projects", srv.port))
	if err != nil {
		t.Fatalf("GET /api/projects: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	count := body["count"].(float64)
	if count < 1 {
		t.Errorf("projects count = %v, want >= 1", count)
	}

	// Shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestSlugFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/project/.hero/specs/my-feature/spec.md", "my-feature"},
		{"/project/.hero/knowledge/conventions/naming/spec.md", "naming"},
		{"/project/.hero/specs/auth-login/spec.md", "auth-login"},
	}

	for _, tc := range tests {
		got := slugFromPath(tc.path)
		if got != tc.want {
			t.Errorf("slugFromPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestServer_EventsIncludeProject(t *testing.T) {
	heroDir, projectRoot := setupTestWorkspace(t)
	srv := NewServer(ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     "test",
	})

	id, ch := srv.Bus().Subscribe(10)
	defer srv.Bus().Unsubscribe(id)

	slug := srv.Projects()[0]

	srv.Bus().Publish(Event{
		Type:    EventSpecCreated,
		Project: slug,
		Slug:    "test-spec",
	})

	select {
	case ev := <-ch:
		if ev.Project != slug {
			t.Errorf("project = %q, want %q", ev.Project, slug)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
