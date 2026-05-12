package install

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSatellitesLocalMissing(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSatellitesLocal(heroDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if s.Version != satellitesLocalVersion {
		t.Errorf("expected version %d, got %d", satellitesLocalVersion, s.Version)
	}
}

func TestUpsertAndFind(t *testing.T) {
	s := &SatellitesLocal{Version: 1}
	s.Upsert(SatelliteEntry{Path: "engines/mlx", Targets: []string{"claude"}})
	s.Upsert(SatelliteEntry{Path: "engines/mlx", Targets: []string{"claude", "codex"}}) // update

	if got := s.Find("engines/mlx"); got == nil {
		t.Fatalf("not found")
	} else if len(got.Targets) != 2 {
		t.Errorf("expected 2 targets after update, got %v", got.Targets)
	}
	if len(s.Satellites) != 1 {
		t.Errorf("expected 1 entry, got %d", len(s.Satellites))
	}
}

func TestRemove(t *testing.T) {
	s := &SatellitesLocal{Version: 1}
	s.Upsert(SatelliteEntry{Path: "a"})
	s.Upsert(SatelliteEntry{Path: "b"})
	if !s.Remove("a") {
		t.Errorf("expected remove true")
	}
	if s.Find("a") != nil {
		t.Errorf("a should be gone")
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	s := &SatellitesLocal{Version: 1}
	s.Upsert(SatelliteEntry{Path: "x", Targets: []string{"claude"}, InstalledAt: time.Now().UTC().Truncate(time.Second)})
	if err := SaveSatellitesLocal(heroDir, s); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := LoadSatellitesLocal(heroDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Find("x") == nil {
		t.Errorf("entry lost on reload")
	}
}
