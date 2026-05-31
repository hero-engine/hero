package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func mkHeroRoot(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, HeroDir), 0o755); err != nil {
		t.Fatalf("mkdir hero: %v", err)
	}
}

func TestLocateAtRoot(t *testing.T) {
	root := t.TempDir()
	mkHeroRoot(t, root)

	ws, err := Locate(root)
	if err != nil {
		t.Fatalf("locate: %v", err)
	}
	if ws.Root != root {
		t.Errorf("root = %s, want %s", ws.Root, root)
	}
	if ws.IsSatellite {
		t.Errorf("expected non-satellite at root")
	}
}

func TestLocateWalkUp(t *testing.T) {
	root := t.TempDir()
	mkHeroRoot(t, root)
	sub := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := Locate(sub)
	if err != nil {
		t.Fatalf("locate: %v", err)
	}
	if ws.Root != root {
		t.Errorf("root = %s, want %s", ws.Root, root)
	}
	if ws.IsSatellite {
		t.Errorf("expected non-satellite for plain subfolder")
	}
}

func TestLocateSatelliteMarker(t *testing.T) {
	root := t.TempDir()
	mkHeroRoot(t, root)
	sat := filepath.Join(root, "engines", "mlx")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteMarker(sat, root, "engines/mlx", "test"); err != nil {
		t.Fatal(err)
	}

	deeper := filepath.Join(sat, "src", "x")
	if err := os.MkdirAll(deeper, 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := Locate(deeper)
	if err != nil {
		t.Fatalf("locate: %v", err)
	}
	if ws.Root != root {
		t.Errorf("root = %s, want %s", ws.Root, root)
	}
	if !ws.IsSatellite {
		t.Errorf("expected satellite=true")
	}
	if ws.MarkerScope != "engines/mlx" {
		t.Errorf("scope = %q, want engines/mlx", ws.MarkerScope)
	}
	if ws.SatellitePath != sat {
		t.Errorf("satellite path = %s, want %s", ws.SatellitePath, sat)
	}
}

func TestLocateNoWorkspace(t *testing.T) {
	dir := t.TempDir()
	// Bound the walk-up at dir so a stray .hero/ in a shared ancestor
	// (e.g. /tmp/.hero left by another tool) can't satisfy the lookup.
	if _, err := Locate(dir, WithStopAt(dir)); err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestLocateSatelliteWithBrokenRoot(t *testing.T) {
	dir := t.TempDir()
	sat := filepath.Join(dir, "thing")
	if err := os.MkdirAll(sat, 0o755); err != nil {
		t.Fatal(err)
	}
	// Marker points to ../.. but no .hero/ exists.
	if err := WriteMarker(sat, dir, "thing", "test"); err != nil {
		t.Fatal(err)
	}

	if _, err := Locate(sat); err == nil {
		t.Errorf("expected error when marker points to non-workspace")
	}
}

func TestRemoveMarker(t *testing.T) {
	dir := t.TempDir()
	root := t.TempDir()
	mkHeroRoot(t, root)
	if err := WriteMarker(dir, root, "x", "v"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveMarker(dir); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, SatelliteMarker)); !os.IsNotExist(err) {
		t.Errorf("marker should be gone, got %v", err)
	}
	// Idempotent on already-removed.
	if err := RemoveMarker(dir); err != nil {
		t.Errorf("remove on missing should be nil, got %v", err)
	}
}
