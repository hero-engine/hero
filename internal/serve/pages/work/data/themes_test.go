package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadThemes_BelowThresholdEmpty locks the "render nothing"
// invariant: when no cluster reaches the confidence threshold, the
// section payload must be empty (not "no themes yet" or any other
// degraded copy). Bad clusters are worse than no clusters.
func TestLoadThemes_BelowThresholdEmpty(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "planning", "features"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	themes := LoadThemes(ThemesInputs{
		HeroDir: heroDir,
		Window:  7 * 24 * time.Hour,
	})
	if len(themes.Clusters) != 0 {
		t.Errorf("expected zero clusters in empty workspace, got %d", len(themes.Clusters))
	}
	if themes.Aggregate {
		t.Errorf("Aggregate should be false in single-project mode")
	}
}

func TestLoadThemes_AggregateFlag(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	themes := LoadThemes(ThemesInputs{
		HeroDir:   heroDir,
		Aggregate: []ThemesProject{{Slug: "a", HeroDir: heroDir}},
	})
	if !themes.Aggregate {
		t.Errorf("Aggregate flag should be true when projects supplied")
	}
}
