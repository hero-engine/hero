package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStack_HappyPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	heroDir := filepath.Join(dir, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "knowledge", "conventions", "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "knowledge", "conventions", "alpha", "spec.md"),
		[]byte("---\ntitle: Alpha\ntype: convention\nstatus: active\n---\n# Alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := LoadStack(StackInputs{ProjectRoot: dir, HeroDir: heroDir})
	if !out.Detected {
		t.Errorf("Detected should be true; Languages=%v", out.Languages)
	}
	if len(out.Languages) < 2 {
		t.Errorf("expected at least 2 languages (Go + JS/TS), got %v", out.Languages)
	}
	if out.ActiveConventions != 1 {
		t.Errorf("ActiveConventions = %d, want 1", out.ActiveConventions)
	}
	if len(out.RecentConventions) != 1 {
		t.Errorf("RecentConventions len = %d, want 1", len(out.RecentConventions))
	}
}

func TestLoadStack_NoInputs(t *testing.T) {
	out := LoadStack(StackInputs{})
	if out.Detected {
		t.Error("Detected should be false when ProjectRoot is empty")
	}
	if len(out.Languages) != 0 {
		t.Errorf("Languages should be empty, got %v", out.Languages)
	}
	if out.ActiveConventions != 0 {
		t.Errorf("ActiveConventions should be 0, got %d", out.ActiveConventions)
	}
}
