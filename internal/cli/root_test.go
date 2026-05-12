package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0h ago"},
		{3 * time.Hour, "3h ago"},
		{23 * time.Hour, "23h ago"},
		{24 * time.Hour, "1 day ago"},
		{48 * time.Hour, "2 days ago"},
		{72 * time.Hour, "3 days ago"},
		{7 * 24 * time.Hour, "7 days ago"},
	}

	for _, tt := range tests {
		got := formatAge(tt.d)
		if got != tt.want {
			t.Errorf("formatAge(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFindProjectRoot_WithHeroDir(t *testing.T) {
	env := newTestEnv(t)

	// We're in env.dir which has .hero/ — should find it
	got := resolvePath(findProjectRoot())
	want := resolvePath(env.dir)
	if got != want {
		t.Errorf("findProjectRoot() = %q, want %q", got, want)
	}
}

// resolvePath resolves symlinks to handle macOS /var -> /private/var
func resolvePath(p string) string {
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return resolved
}

func TestFindProjectRoot_WithGitDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	got := resolvePath(findProjectRoot())
	want := resolvePath(dir)
	if got != want {
		t.Errorf("findProjectRoot() = %q, want %q", got, want)
	}
}

func TestFindProjectRoot_Subdirectory(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	subDir := filepath.Join(dir, "src", "pkg")

	os.MkdirAll(heroDir, 0o755)
	os.MkdirAll(subDir, 0o755)

	origDir, _ := os.Getwd()
	os.Chdir(subDir)
	defer os.Chdir(origDir)

	got := resolvePath(findProjectRoot())
	want := resolvePath(dir)
	if got != want {
		t.Errorf("findProjectRoot() from subdirectory = %q, want %q", got, want)
	}
}

func TestFindProjectRoot_NoMarkers(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Should return cwd when no markers found (will walk to root and fall back)
	got := findProjectRoot()
	// We just check it doesn't crash and returns something
	if got == "" {
		t.Error("findProjectRoot() returned empty string")
	}
}

func TestHasFilters(t *testing.T) {
	// Reset all
	searchType = ""
	searchStatus = ""
	searchTag = ""
	searchSince = ""

	if hasFilters() {
		t.Error("hasFilters should be false when no filters set")
	}

	searchType = "feature"
	if !hasFilters() {
		t.Error("hasFilters should be true when type filter set")
	}

	searchType = ""
	searchTag = "api"
	if !hasFilters() {
		t.Error("hasFilters should be true when tag filter set")
	}

	// Reset
	searchTag = ""
}

func TestVersionFlag(t *testing.T) {
	SetVersion("1.2.3")
	output, err := runCmd("--version")
	if err != nil {
		t.Fatalf("--version returned error: %v", err)
	}
	if !strings.Contains(output, "1.2.3") {
		t.Errorf("--version output %q does not contain version", output)
	}
}
