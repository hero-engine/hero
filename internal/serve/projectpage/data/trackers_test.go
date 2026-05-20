package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTrackers_HappyPath(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "imports"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"),
		[]byte(`{"tracker":{"type":"github","project":"acme/widgets"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "imports", "marker.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := LoadTrackers(TrackersInputs{ProjectRoot: dir, HeroDir: heroDir})
	if !out.Configured {
		t.Fatal("expected Configured=true")
	}
	if out.Type != "github" {
		t.Errorf("Type = %q, want github", out.Type)
	}
	if out.Project != "acme/widgets" {
		t.Errorf("Project = %q, want acme/widgets", out.Project)
	}
	if out.LastSyncAt.IsZero() {
		t.Error("expected LastSyncAt non-zero since imports/marker.json exists")
	}
}

func TestLoadTrackers_NoTrackerConfigured(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := LoadTrackers(TrackersInputs{ProjectRoot: dir, HeroDir: heroDir})
	if out.Configured {
		t.Error("expected Configured=false when no tracker in hero.json")
	}
}
