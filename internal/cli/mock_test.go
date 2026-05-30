package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMockCmd_ListEmpty(t *testing.T) {
	_ = newTestEnv(t)

	out, err := runCmd("spec", "mock", "--list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No mockups found") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestMockCmd_ListWithMocks(t *testing.T) {
	env := newTestEnv(t)

	// Create some mock directories with index.html
	for _, slug := range []string{"login-page", "dashboard", "settings"} {
		dir := filepath.Join(env.heroDir, "mocks", slug)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	out, err := runCmd("spec", "mock", "--list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "login-page") {
		t.Errorf("expected login-page in output, got: %s", out)
	}
	if !strings.Contains(out, "dashboard") {
		t.Errorf("expected dashboard in output, got: %s", out)
	}
	if !strings.Contains(out, "settings") {
		t.Errorf("expected settings in output, got: %s", out)
	}
	if !strings.Contains(out, "3 mockup(s)") {
		t.Errorf("expected count line, got: %s", out)
	}
	// All HTML-only mocks should show [html] tag
	if !strings.Contains(out, "[html]") {
		t.Errorf("expected [html] tag in output, got: %s", out)
	}
}

func TestMockCmd_ListNativeTag(t *testing.T) {
	env := newTestEnv(t)

	// Create an HTML-only mock
	htmlDir := filepath.Join(env.heroDir, "mocks", "web-page")
	if err := os.MkdirAll(htmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(htmlDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a native mock (has screenshot.png alongside index.html)
	nativeDir := filepath.Join(env.heroDir, "mocks", "settings-screen")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "screenshot.png"), []byte("fake-png"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd("spec", "mock", "--list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The native mock should show [native]
	if !strings.Contains(out, "[native]") {
		t.Errorf("expected [native] tag for settings-screen, got: %s", out)
	}
	// The HTML mock should show [html]
	if !strings.Contains(out, "[html]") {
		t.Errorf("expected [html] tag for web-page, got: %s", out)
	}
	if !strings.Contains(out, "2 mockup(s)") {
		t.Errorf("expected 2 mockup count, got: %s", out)
	}
}

func TestMockCmd_ListSkipsNonDirs(t *testing.T) {
	env := newTestEnv(t)

	mocksDir := filepath.Join(env.heroDir, "mocks")
	// Create a file (not dir) in mocks — should be skipped
	if err := os.WriteFile(filepath.Join(mocksDir, "stray-file.txt"), []byte("not a mock"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create a dir without index.html — should be skipped
	if err := os.MkdirAll(filepath.Join(mocksDir, "empty-dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a valid mock
	validDir := filepath.Join(mocksDir, "valid-mock")
	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd("spec", "mock", "--list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "valid-mock") {
		t.Errorf("expected valid-mock, got: %s", out)
	}
	if strings.Contains(out, "stray-file") {
		t.Errorf("should not list stray files, got: %s", out)
	}
	if strings.Contains(out, "empty-dir") {
		t.Errorf("should not list dirs without index.html, got: %s", out)
	}
	if !strings.Contains(out, "1 mockup(s)") {
		t.Errorf("expected 1 mockup count, got: %s", out)
	}
}

func TestMockCmd_OpenNotFound(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("spec", "mock", "--open", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent mock")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestMockCmd_DefaultIsList(t *testing.T) {
	_ = newTestEnv(t)

	out, err := runCmd("spec", "mock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default behavior should be the same as --list
	if !strings.Contains(out, "No mockups found") {
		t.Errorf("expected list output by default, got: %s", out)
	}
}

func TestMockCmd_NoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	out, err := runCmd("spec", "mock", "--list")
	// Even without a workspace, it should handle gracefully
	// (no mocks dir = no mocks found)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No mockups found") {
		t.Errorf("expected empty message, got: %s", out)
	}
}
