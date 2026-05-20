package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_HappyPath(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"),
		[]byte(`{"name":"x","methodology":"solo"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := LoadConfig(ConfigInputs{ProjectRoot: dir, HeroDir: heroDir})
	if !out.HeroJSONExists {
		t.Fatal("expected HeroJSONExists=true")
	}
	if !strings.Contains(out.PrettyJSON, "methodology") {
		t.Errorf("PrettyJSON should include methodology, got %q", out.PrettyJSON)
	}
	if !strings.HasPrefix(out.OpenInEditorURL, "file://") {
		t.Errorf("OpenInEditorURL = %q, expected file:// prefix", out.OpenInEditorURL)
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	out := LoadConfig(ConfigInputs{ProjectRoot: dir, HeroDir: filepath.Join(dir, ".hero")})
	if out.HeroJSONExists {
		t.Error("expected HeroJSONExists=false when hero.json is missing")
	}
	if out.PrettyJSON != "" {
		t.Errorf("PrettyJSON should be empty, got %q", out.PrettyJSON)
	}
}
