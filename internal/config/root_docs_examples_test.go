package config

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var rootHeroConfigExample = regexp.MustCompile(
	`(?s)<!-- hero-config -->\s*` + "```json" + `\s*(.*?)\s*` + "```",
)

func TestRootDocumentationHeroConfigExamplesLoadThroughProductionDecoder(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	found := 0
	for _, name := range []string{"README.md", "GETTING-STARTED.md", "MCP-SETUP.md"} {
		content, err := os.ReadFile(filepath.Join(repoRoot, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for index, match := range rootHeroConfigExample.FindAllSubmatch(content, -1) {
			found++
			root := t.TempDir()
			heroDir := filepath.Join(root, DefaultFolder)
			if err := os.MkdirAll(heroDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), match[1], 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(root); err != nil {
				t.Errorf("%s hero-config example %d failed config.Load: %v", name, index+1, err)
			}
		}
	}
	if found != 2 {
		t.Fatalf("found %d root hero-config examples, want 2", found)
	}
}
