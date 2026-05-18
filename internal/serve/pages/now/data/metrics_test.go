package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// writeEventLog appends a single newline-delimited JSON event to
// .hero/events.log under dir. Each event is the minimal subset the
// feed reader needs.
func writeEventLog(t *testing.T, dir string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "events.log")
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write events.log: %v", err)
	}
}

func TestCountCompletedSince_CountsDeliveryComplete(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	writeEventLog(t, dir,
		`{"ts":"`+now+`","type":"delivery_complete","slug":"a","message":""}`,
		`{"ts":"`+now+`","type":"delivery_complete","slug":"b","message":""}`,
		`{"ts":"`+now+`","type":"decision_made","slug":"c","message":""}`,
		// spec.complete is intentionally NOT counted — it was a draft
		// verb that never had an emitter. Only delivery_complete
		// signals a finished spec (see polish-v2 Fix 4).
		`{"ts":"`+now+`","type":"spec.complete","slug":"d","message":""}`,
	)
	got := countCompletedSince(dir, 7*24*time.Hour)
	if got != 2 {
		t.Errorf("countCompletedSince = %d, want 2 (delivery_complete only)", got)
	}
}

func TestIsAgentAuthored(t *testing.T) {
	cases := map[string]bool{
		"claude-sonnet-4":   true,
		"gpt-4":             true,
		"ai/codex":          true,
		"engineer-1":        true,
		"agent/foo":         true,
		"mcp/hero":          true,
		"human/chet":        false,
		"":                  false,
		"some-random-tool":  false,
	}
	for in, want := range cases {
		if got := isAgentAuthored(in); got != want {
			t.Errorf("isAgentAuthored(%q) = %v, want %v", in, got, want)
		}
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
