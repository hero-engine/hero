package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadWeek_TileShape(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tiles := LoadWeek(WeekInputs{
		ProjectRoot: root,
		HeroDir:     heroDir,
		Now:         time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
	})
	if len(tiles) != 4 {
		t.Fatalf("expected 4 tiles, got %d", len(tiles))
	}
	wantLabels := []string{"touched this week", "shipped this week", "started this week", "stale (>14d)"}
	for i, w := range wantLabels {
		if tiles[i].Label != w {
			t.Errorf("tile %d label: got %q, want %q", i, tiles[i].Label, w)
		}
		if tiles[i].Href == "" {
			t.Errorf("tile %d (%s): expected non-empty Href", i, w)
		}
	}
}

func TestLoadWeek_FeedCounts(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// One delivery_start, one delivery_complete, two distinct slugs
	// touched within the window; one event outside the window that
	// should not be counted.
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	in := now.Add(-3 * 24 * time.Hour).Format(time.RFC3339)
	old := now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	body := strings.Join([]string{
		`{"ts":"` + in + `","type":"delivery_start","slug":"alpha","agent":"hero"}`,
		`{"ts":"` + in + `","type":"delivery_complete","slug":"beta","agent":"hero"}`,
		`{"ts":"` + in + `","type":"spec_updated","slug":"alpha","agent":"hero"}`,
		`{"ts":"` + old + `","type":"delivery_complete","slug":"gamma","agent":"hero"}`,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(heroDir, "events.log"), []byte(body), 0o644); err != nil {
		t.Fatalf("write events.log: %v", err)
	}

	tiles := LoadWeek(WeekInputs{ProjectRoot: root, HeroDir: heroDir, Now: now})
	want := map[string]string{
		"touched this week":  "2",
		"shipped this week":  "1",
		"started this week":  "1",
	}
	for _, tile := range tiles {
		if w, ok := want[tile.Label]; ok {
			got := string(tile.Value)
			if got != w {
				t.Errorf("%s: got %q, want %q", tile.Label, got, w)
			}
		}
	}
}

func TestLoadWeek_StaleAccent(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tiles := LoadWeek(WeekInputs{
		ProjectRoot: root,
		HeroDir:     heroDir,
		Now:         time.Now(),
	})
	stale := tiles[3]
	if stale.Label != "stale (>14d)" {
		t.Fatalf("unexpected fourth tile: %+v", stale)
	}
	if string(stale.Value) == "0" && stale.Accent != "" {
		t.Errorf("zero-stale tile should have no accent, got %q", stale.Accent)
	}
}
