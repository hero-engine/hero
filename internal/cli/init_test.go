package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesDirectories(t *testing.T) {
	env := newTestEnvEmpty(t)

	output, err := runCmd("init")
	if err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	if !strings.Contains(output, "Initialized hero workspace") {
		t.Errorf("init output missing initialization message: %q", output)
	}

	// Verify all directories were created
	expectedDirs := []string{
		".hero",
		".hero/planning",
		".hero/planning/features",
		".hero/planning/bugs",
		".hero/planning/initiatives",
		".hero/specs",
		".hero/knowledge",
		".hero/knowledge/conventions",
		".hero/knowledge/decisions",
		".hero/knowledge/rules",
		".hero/knowledge/external",
		".hero/knowledge/context",
		".hero/knowledge/templates",
		".hero/knowledge/notes",
	}

	for _, d := range expectedDirs {
		full := filepath.Join(env.dir, d)
		info, err := os.Stat(full)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", d)
		}
	}
}

func TestInitWritesHeroJSON(t *testing.T) {
	env := newTestEnvEmpty(t)

	_, err := runCmd("init")
	if err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	configPath := filepath.Join(env.dir, ".hero", "hero.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("hero.json not created: %v", err)
	}

	content := string(data)
	// Verify it has the expected default values
	if !strings.Contains(content, `"folder": ".hero"`) {
		t.Error("hero.json missing folder field")
	}
	if !strings.Contains(content, `"nudge_level": "gentle"`) {
		t.Error("hero.json missing nudge_level default")
	}
	if !strings.Contains(content, `"stale_days": 14`) {
		t.Error("hero.json missing stale_days default")
	}
}

func TestInitRefusesReinit(t *testing.T) {
	_ = newTestEnv(t) // already has .hero/

	_, err := runCmd("init")
	if err == nil {
		t.Fatal("init should return error when workspace already exists")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestInitCustomFolder(t *testing.T) {
	env := newTestEnvEmpty(t)

	output, err := runCmd("init", "--folder", ".custom")
	if err != nil {
		t.Fatalf("init --folder .custom returned error: %v", err)
	}

	if !strings.Contains(output, ".custom") {
		t.Errorf("init output should mention custom folder: %q", output)
	}

	customDir := filepath.Join(env.dir, ".custom")
	if _, err := os.Stat(customDir); err != nil {
		t.Errorf("custom folder not created: %v", err)
	}

	// hero.json should be in the custom folder
	configPath := filepath.Join(customDir, "hero.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("hero.json not in custom folder: %v", err)
	}
}

func TestInitOutputListsDirectories(t *testing.T) {
	_ = newTestEnvEmpty(t)

	output, err := runCmd("init")
	if err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	expectedParts := []string{
		"planning/features/",
		"planning/bugs/",
		"planning/initiatives/",
		"specs/",
		"knowledge/",
		"conventions/",
		"decisions/",
		"rules/",
		"notes/",
		"hero.json",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("init output missing %q", part)
		}
	}
}
