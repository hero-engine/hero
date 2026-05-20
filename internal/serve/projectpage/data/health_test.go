package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHealth_HappyPath(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	cacheDir := filepath.Join(heroDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "health.json"),
		[]byte(`{"captured_at":"2026-05-19T10:00:00Z","rows":[{"name":"specs","status":"pass","message":"ok"}]}`),
		0o644); err != nil {
		t.Fatal(err)
	}

	out := LoadHealth(HealthInputs{HeroDir: heroDir})
	if !out.HasArtifact {
		t.Fatal("expected HasArtifact=true")
	}
	if len(out.Rows) != 1 {
		t.Fatalf("Rows len = %d, want 1", len(out.Rows))
	}
	if !out.AllClear {
		t.Error("expected AllClear=true when every row passes")
	}
	if out.CapturedAtPretty == "" {
		t.Error("expected CapturedAtPretty non-empty")
	}
}

func TestLoadHealth_NoArtifact(t *testing.T) {
	dir := t.TempDir()
	out := LoadHealth(HealthInputs{HeroDir: filepath.Join(dir, ".hero")})
	if out.HasArtifact {
		t.Error("expected HasArtifact=false when no cached artifact exists")
	}
	if out.CapturedAtPretty != "" {
		t.Errorf("CapturedAtPretty should be empty, got %q", out.CapturedAtPretty)
	}
	if out.AllClear {
		t.Error("expected AllClear=false when no rows")
	}
}
