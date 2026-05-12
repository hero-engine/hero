package active

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempHeroDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return heroDir
}

func TestRegisterAndActiveSpecs(t *testing.T) {
	heroDir := tempHeroDir(t)

	if err := Register(heroDir, "s1", "csv-export", "/deliver"); err != nil {
		t.Fatal(err)
	}
	if err := Register(heroDir, "s2", "auth-bug", "/diagnose"); err != nil {
		t.Fatal(err)
	}

	slugs := ActiveSpecs(heroDir)
	if len(slugs) != 2 {
		t.Fatalf("expected 2 active specs, got %d", len(slugs))
	}

	// Verify file exists on disk
	if _, err := os.Stat(filepath.Join(heroDir, registryFileName)); err != nil {
		t.Fatal("registry file should exist on disk")
	}
}

func TestUnregister(t *testing.T) {
	heroDir := tempHeroDir(t)

	Register(heroDir, "s1", "csv-export", "/deliver")
	Register(heroDir, "s2", "auth-bug", "/diagnose")

	if err := Unregister(heroDir, "s1"); err != nil {
		t.Fatal(err)
	}

	slugs := ActiveSpecs(heroDir)
	if len(slugs) != 1 {
		t.Fatalf("expected 1 active spec, got %d", len(slugs))
	}
	if slugs[0] != "auth-bug" {
		t.Fatalf("expected auth-bug, got %s", slugs[0])
	}
}

func TestPrune(t *testing.T) {
	heroDir := tempHeroDir(t)

	// Register with a manually backdated session
	r := Load(heroDir)
	r.Sessions["old"] = Session{
		Spec:    "stale-spec",
		Command: "/deliver",
		Started: time.Now().UTC().Add(-48 * time.Hour),
	}
	r.Sessions["recent"] = Session{
		Spec:    "fresh-spec",
		Command: "/deliver",
		Started: time.Now().UTC(),
	}
	if err := r.Save(heroDir); err != nil {
		t.Fatal(err)
	}

	pruned, err := Prune(heroDir, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Fatalf("expected 1 pruned, got %d", pruned)
	}

	slugs := ActiveSpecs(heroDir)
	if len(slugs) != 1 || slugs[0] != "fresh-spec" {
		t.Fatalf("expected only fresh-spec, got %v", slugs)
	}
}

func TestDuplicateSpecMultipleSessions(t *testing.T) {
	heroDir := tempHeroDir(t)

	Register(heroDir, "s1", "csv-export", "/deliver")
	Register(heroDir, "s2", "csv-export", "/deliver")

	slugs := ActiveSpecs(heroDir)
	if len(slugs) != 1 {
		t.Fatalf("expected 1 unique spec, got %d", len(slugs))
	}
}

func TestEmptyRegistry(t *testing.T) {
	heroDir := tempHeroDir(t)

	slugs := ActiveSpecs(heroDir)
	if len(slugs) != 0 {
		t.Fatalf("expected 0 specs, got %d", len(slugs))
	}
}
