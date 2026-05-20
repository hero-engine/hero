package data

import (
	"os"
	"path/filepath"
	"testing"
)

// writeHealth installs a cached health.json under heroDir with the
// supplied rows.
func writeHealth(t *testing.T, heroDir string, rows []cachedHealthRow) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(heroDir, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Minimal JSON; CapturedAt left zero is fine for the rule.
	body := `{"captured_at":"2026-05-01T12:00:00Z","rows":[`
	for i, r := range rows {
		if i > 0 {
			body += ","
		}
		body += `{"name":"` + r.Name + `","status":"` + r.Status + `","message":"` + r.Message + `"}`
	}
	body += `]}`
	if err := os.WriteFile(filepath.Join(heroDir, "cache", "health.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHealthRollup_AllGreen(t *testing.T) {
	dir := mkProject(t, "g")
	hero := filepath.Join(dir, ".hero")
	writeHealth(t, hero, []cachedHealthRow{{Name: "stale specs", Status: "pass"}})

	out := LoadHealthRollup(HealthRollupInputs{
		Projects: []DirectoryProject{{Slug: "g", ProjectRoot: dir, HeroDir: hero}},
	})
	if !out.AllClear {
		t.Errorf("AllClear = false, want true (no non-pass rows). PerProject=%+v", out.PerProject)
	}
	if out.OverallColor != "green" {
		t.Errorf("OverallColor = %q, want green", out.OverallColor)
	}
}

func TestHealthRollup_YellowOnlyStale(t *testing.T) {
	dir := mkProject(t, "y")
	hero := filepath.Join(dir, ".hero")
	writeHealth(t, hero, []cachedHealthRow{
		{Name: "stale specs", Status: "warn"},
		{Name: "missing kickoff", Status: "warn"},
	})

	out := LoadHealthRollup(HealthRollupInputs{
		Projects: []DirectoryProject{{Slug: "y", ProjectRoot: dir, HeroDir: hero}},
	})
	if out.OverallColor != "yellow" {
		t.Errorf("OverallColor = %q, want yellow", out.OverallColor)
	}
	if out.StaleSpecs != 1 || out.MissingKickoffs != 1 {
		t.Errorf("counts: stale=%d kickoffs=%d", out.StaleSpecs, out.MissingKickoffs)
	}
}

func TestHealthRollup_RedOnDrift(t *testing.T) {
	dir := mkProject(t, "r")
	hero := filepath.Join(dir, ".hero")
	writeHealth(t, hero, []cachedHealthRow{
		{Name: "drift", Status: "fail"},
	})

	out := LoadHealthRollup(HealthRollupInputs{
		Projects: []DirectoryProject{{Slug: "r", ProjectRoot: dir, HeroDir: hero}},
	})
	if out.OverallColor != "red" {
		t.Errorf("OverallColor = %q, want red", out.OverallColor)
	}
	if out.Drift != 1 {
		t.Errorf("Drift = %d, want 1", out.Drift)
	}
}

func TestHealthRollup_RedOnBrokenManifest(t *testing.T) {
	dir := mkProject(t, "rm")
	hero := filepath.Join(dir, ".hero")
	// Health artifact reports zero items, but a configured peer's
	// manifest is missing on disk — broken-manifest should drive red.
	if err := os.WriteFile(filepath.Join(dir, ".hero", "hero.json"),
		[]byte(`{"repos":{"missing-peer":"`+filepath.Join(t.TempDir(), "nowhere")+`"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	writeHealth(t, hero, []cachedHealthRow{{Name: "stale specs", Status: "pass"}})

	out := LoadHealthRollup(HealthRollupInputs{
		Projects: []DirectoryProject{{Slug: "rm", ProjectRoot: dir, HeroDir: hero}},
	})
	if out.OverallColor != "red" {
		t.Errorf("OverallColor = %q, want red (broken peer manifest); PerProject=%+v", out.OverallColor, out.PerProject)
	}
}

func TestHealthRollup_NoArtifactNoCounts(t *testing.T) {
	dir := mkProject(t, "u")
	hero := filepath.Join(dir, ".hero")
	// No health.json present.
	out := LoadHealthRollup(HealthRollupInputs{
		Projects: []DirectoryProject{{Slug: "u", ProjectRoot: dir, HeroDir: hero}},
	})
	if !out.AllClear {
		// Empty registry of items folds into green.
		t.Errorf("AllClear = false, want true on zero-input rollup")
	}
	if len(out.PerProject) != 1 || out.PerProject[0].Color != "unknown" {
		t.Errorf("expected one project with unknown colour, got %+v", out.PerProject)
	}
}
