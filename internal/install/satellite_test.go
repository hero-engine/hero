package install

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hero-engine/hero/internal/workspace"
)

// setupRoot creates a fake workspace root with a minimal .claude/ install
// containing agents/, commands/, skills/.
func setupRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, sub := range SymlinkedDirs {
		path := filepath.Join(root, ".claude", sub)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		// Add a sentinel file so we can verify the symlink resolves.
		if err := os.WriteFile(filepath.Join(path, "sentinel.md"), []byte("hello"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestDetectInstalledTargets(t *testing.T) {
	root := setupRoot(t)
	got := DetectInstalledTargets(root)
	if len(got) != 1 || got[0] != TargetClaude {
		t.Errorf("got %v, want [claude]", got)
	}
}

func TestMaterializeFullSatellite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks may be unavailable on Windows in CI")
	}
	root := setupRoot(t)
	sat := filepath.Join(root, "engines", "mlx")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Materialize(SatelliteOptions{
		RootDir:      root,
		SatelliteDir: sat,
		Scope:        "engines/mlx",
		Version:      "test",
	})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if res.Degraded {
		t.Errorf("expected non-degraded on POSIX")
	}

	// Verify symlink resolves.
	for _, sub := range SymlinkedDirs {
		linkPath := filepath.Join(sat, ".claude", sub)
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("readlink %s: %v", linkPath, err)
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(linkPath), target))
		want := filepath.Join(root, ".claude", sub)
		if resolved != want {
			t.Errorf("symlink %s -> %s, want %s", linkPath, resolved, want)
		}
		// Confirm sentinel reachable.
		data, err := os.ReadFile(filepath.Join(linkPath, "sentinel.md"))
		if err != nil || string(data) != "hello" {
			t.Errorf("sentinel through symlink failed: %v %q", err, data)
		}
	}

	// Verify .hero-satellite marker.
	marker := filepath.Join(sat, workspace.SatelliteMarker)
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("marker missing: %v", err)
	}

	// Verify CLAUDE.md per-target marker.
	claudeMd := filepath.Join(sat, "CLAUDE.md")
	if data, err := os.ReadFile(claudeMd); err != nil {
		t.Errorf("CLAUDE.md missing: %v", err)
	} else if !contains(string(data), "<!-- hero:satellite -->") {
		t.Errorf("CLAUDE.md missing hero:satellite marker")
	}
}

func TestMaterializeRefusesAtRoot(t *testing.T) {
	root := setupRoot(t)
	_, err := Materialize(SatelliteOptions{
		RootDir:      root,
		SatelliteDir: root,
	})
	if err == nil {
		t.Errorf("expected error materializing at root")
	}
}

func TestMaterializeRefusesOutsideRoot(t *testing.T) {
	root := setupRoot(t)
	outside := t.TempDir()
	_, err := Materialize(SatelliteOptions{
		RootDir:      root,
		SatelliteDir: outside,
	})
	if err == nil {
		t.Errorf("expected error materializing outside root")
	}
}

func TestRemoveSatelliteCleansUp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks may be unavailable on Windows in CI")
	}
	root := setupRoot(t)
	sat := filepath.Join(root, "x")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(SatelliteOptions{
		RootDir:      root,
		SatelliteDir: sat,
		Scope:        "x",
		Version:      "test",
	}); err != nil {
		t.Fatal(err)
	}

	if err := RemoveSatellite(sat, []Target{TargetClaude}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sat, ".claude", "agents")); !os.IsNotExist(err) {
		t.Errorf("agents symlink should be gone, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(sat, workspace.SatelliteMarker)); !os.IsNotExist(err) {
		t.Errorf("hero-satellite marker should be gone, got %v", err)
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
