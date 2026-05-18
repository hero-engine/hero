package data

import (
	"strings"
	"testing"
)

func TestLoadMetrics_ReturnsFourTilesPerTab(t *testing.T) {
	cases := []string{"scrum", "shape-up", "kanban", "solo", ""}
	for _, m := range cases {
		got := LoadMetrics(MetricsInputs{Methodology: m})
		if len(got.FirstTabTiles) != 4 {
			t.Errorf("methodology=%q: first-tab tiles = %d, want 4", m, len(got.FirstTabTiles))
		}
		if len(got.MyWeekTiles) != 4 {
			t.Errorf("methodology=%q: my-week tiles = %d, want 4", m, len(got.MyWeekTiles))
		}
		if len(got.ROITiles) != 4 {
			t.Errorf("methodology=%q: roi tiles = %d, want 4", m, len(got.ROITiles))
		}
	}
}

func TestLoadMetrics_PlaceholdersWhenNoProject(t *testing.T) {
	got := LoadMetrics(MetricsInputs{Methodology: "kanban"})
	first := string(got.FirstTabTiles[0].Value)
	if !strings.Contains(first, "—") {
		t.Errorf("first-tab tile value = %q, want placeholder dash", first)
	}
}

func TestSparklineSVG(t *testing.T) {
	out := string(sparklineSVG([]int{1, 2, 3, 4}))
	if !strings.Contains(out, "<polyline") {
		t.Errorf("sparkline output missing polyline: %q", out)
	}
	if !strings.Contains(out, "metric-sparkline") {
		t.Errorf("sparkline missing class: %q", out)
	}

	empty := string(sparklineSVG(nil))
	if !strings.Contains(empty, "<svg") {
		t.Errorf("empty sparkline missing fallback svg: %q", empty)
	}
}
