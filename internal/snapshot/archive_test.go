package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEvaluateTriggers_ManualAlwaysFires(t *testing.T) {
	now := time.Now()
	hits := EvaluateTriggers(EvaluateTriggersInput{
		Now:             now,
		ManualRequested: true,
		ManualLabel:     "hand",
		Config:          ArchiveConfig{StalenessCutoff: "monthly", MilestonesEnabled: true},
	})
	if len(hits) != 1 || hits[0].Trigger != TriggerManual {
		t.Errorf("expected one manual hit; got %+v", hits)
	}
	if hits[0].Label != "hand" {
		t.Errorf("label = %q, want hand", hits[0].Label)
	}
}

func TestEvaluateTriggers_MilestoneSuppressesStaleness(t *testing.T) {
	old := time.Now().Add(-60 * 24 * time.Hour)
	archives := []ArchiveRecord{{Date: old.Format("2006-01-02")}}
	hits := EvaluateTriggers(EvaluateTriggersInput{
		Now:                       time.Now(),
		ExistingArchives:          archives,
		NewlyCompletedInitiatives: []string{"web-surfaces-restructure"},
		Config: ArchiveConfig{
			StalenessCutoff:   "monthly",
			MilestonesEnabled: true,
		},
	})
	// Should have ONE hit: milestone. Staleness suppressed.
	if len(hits) != 1 {
		t.Errorf("expected 1 hit, got %d: %+v", len(hits), hits)
	}
	if hits[0].Trigger != TriggerMilestone {
		t.Errorf("expected milestone, got %s", hits[0].Trigger)
	}
}

func TestEvaluateTriggers_StalenessFires(t *testing.T) {
	old := time.Now().Add(-60 * 24 * time.Hour)
	archives := []ArchiveRecord{{Date: old.Format("2006-01-02")}}
	hits := EvaluateTriggers(EvaluateTriggersInput{
		Now:              time.Now(),
		ExistingArchives: archives,
		Config: ArchiveConfig{
			StalenessCutoff:   "monthly",
			MilestonesEnabled: true,
		},
	})
	if len(hits) != 1 || hits[0].Trigger != TriggerStaleness {
		t.Errorf("expected staleness hit; got %+v", hits)
	}
}

func TestEvaluateTriggers_FirstRunNoStaleness(t *testing.T) {
	// No existing archives → first-run safety means no staleness.
	hits := EvaluateTriggers(EvaluateTriggersInput{
		Now:    time.Now(),
		Config: ArchiveConfig{StalenessCutoff: "weekly", MilestonesEnabled: true},
	})
	if len(hits) != 0 {
		t.Errorf("expected no first-run staleness, got %+v", hits)
	}
}

func TestEvaluateTriggers_StalenessOff(t *testing.T) {
	old := time.Now().Add(-365 * 24 * time.Hour)
	archives := []ArchiveRecord{{Date: old.Format("2006-01-02")}}
	hits := EvaluateTriggers(EvaluateTriggersInput{
		Now:              time.Now(),
		ExistingArchives: archives,
		Config: ArchiveConfig{
			StalenessCutoff:   "off",
			MilestonesEnabled: true,
		},
	})
	if len(hits) != 0 {
		t.Errorf("staleness=off should suppress; got %+v", hits)
	}
}

func TestMaybeWrite_ArchiveContents(t *testing.T) {
	dir := t.TempDir()
	out, err := MaybeWrite(MaybeWriteInput{
		Rendered:  []byte("# Project Snapshot — test\n\nbody\n"),
		HeroDir:   dir,
		Now:       time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		GitCommit: "abc123",
		Triggers:  []TriggerHit{{Trigger: TriggerManual, Label: "hand"}},
	})
	if err != nil {
		t.Fatalf("MaybeWrite: %v", err)
	}
	if len(out.Written) != 1 {
		t.Fatalf("expected 1 archive written, got %d", len(out.Written))
	}
	data, err := os.ReadFile(out.Written[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	// Required invariants.
	if !strings.Contains(body, "snapshot_date: 2026-05-18") {
		t.Errorf("missing snapshot_date in: %s", body)
	}
	if !strings.Contains(body, "historical: true") {
		t.Errorf("missing historical: true")
	}
	if !strings.Contains(body, "not_current: true") {
		t.Errorf("missing not_current: true")
	}
	if !strings.Contains(body, "Historical archive captured 2026-05-18") {
		t.Errorf("missing banner line")
	}
	if !strings.Contains(body, "# Project Snapshot — test") {
		t.Errorf("missing rendered body")
	}
}

func TestMaybeWrite_SameDayIdempotent(t *testing.T) {
	dir := t.TempDir()
	render := []byte("# Snapshot\n")
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	// First fire.
	r1, err := MaybeWrite(MaybeWriteInput{
		Rendered: render,
		HeroDir:  dir,
		Now:      now,
		Triggers: []TriggerHit{{Trigger: TriggerStaleness, Label: "auto-staleness"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(r1.Written) != 1 {
		t.Fatalf("first write = %d", len(r1.Written))
	}
	// Second fire, same day, no label change → no-op.
	r2, err := MaybeWrite(MaybeWriteInput{
		Rendered: render,
		HeroDir:  dir,
		Now:      now,
		Triggers: []TriggerHit{{Trigger: TriggerStaleness, Label: "auto-staleness"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(r2.Written) != 0 {
		t.Errorf("expected idempotent no-op, got %d writes", len(r2.Written))
	}
	if len(r2.Skipped) != 1 {
		t.Errorf("expected skipped=1, got %d", len(r2.Skipped))
	}
}

func TestMaybeWrite_LabeledManualSuffix(t *testing.T) {
	dir := t.TempDir()
	render := []byte("# Snapshot\n")
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		out, err := MaybeWrite(MaybeWriteInput{
			Rendered: render,
			HeroDir:  dir,
			Now:      now,
			Triggers: []TriggerHit{{Trigger: TriggerManual, Label: "v1-ship"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Written) != 1 {
			t.Errorf("iter %d: expected 1 write, got %d", i, len(out.Written))
		}
	}
	// Both files should exist (second got -2 suffix).
	entries, _ := os.ReadDir(filepath.Join(dir, ArchiveDirName))
	if len(entries) != 2 {
		t.Errorf("expected 2 files, got %d: %+v", len(entries), entries)
	}
}

func TestArchiveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	out, _ := MaybeWrite(MaybeWriteInput{
		Rendered:  []byte("# Snap\n"),
		HeroDir:   dir,
		Now:       now,
		GitCommit: "deadbeef",
		Triggers:  []TriggerHit{{Trigger: TriggerMilestone, Label: "hero-pm"}},
	})
	if len(out.Written) != 1 {
		t.Fatal("no archive written")
	}
	rec, err := Read(out.Written[0].Path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if rec.Date != "2026-05-18" {
		t.Errorf("date = %q", rec.Date)
	}
	if rec.Trigger != TriggerMilestone {
		t.Errorf("trigger = %q", rec.Trigger)
	}
	if rec.Label != "hero-pm" {
		t.Errorf("label = %q", rec.Label)
	}
	if rec.GitCommit != "deadbeef" {
		t.Errorf("git_commit = %q", rec.GitCommit)
	}
	if !rec.Historical || !rec.NotCurrent {
		t.Errorf("missing isolation flags")
	}
}

func TestApplyRetention_LastN(t *testing.T) {
	dir := t.TempDir()
	render := []byte("# Snap\n")
	dates := []string{"2026-01-01", "2026-02-01", "2026-03-01", "2026-04-01"}
	for _, d := range dates {
		ts, _ := time.Parse("2006-01-02", d)
		_, err := MaybeWrite(MaybeWriteInput{
			Rendered: render,
			HeroDir:  dir,
			Now:      ts,
			Triggers: []TriggerHit{{Trigger: TriggerManual, Label: "snap-" + d}},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	removed, err := ApplyRetention(dir, ArchiveConfig{Retention: "last-n", RetentionCount: 2})
	if err != nil {
		t.Fatalf("ApplyRetention: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("expected 2 removed, got %d", len(removed))
	}
	left, _ := List(dir)
	if len(left) != 2 {
		t.Errorf("expected 2 remaining, got %d", len(left))
	}
	// Newest two should remain.
	if left[0].Date != "2026-04-01" || left[1].Date != "2026-03-01" {
		t.Errorf("unexpected remaining: %+v", left)
	}
}

func TestApplyRetention_All(t *testing.T) {
	dir := t.TempDir()
	render := []byte("# Snap\n")
	for _, d := range []string{"2026-01-01", "2026-02-01"} {
		ts, _ := time.Parse("2006-01-02", d)
		_, err := MaybeWrite(MaybeWriteInput{
			Rendered: render,
			HeroDir:  dir,
			Now:      ts,
			Triggers: []TriggerHit{{Trigger: TriggerManual, Label: d}},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	removed, _ := ApplyRetention(dir, ArchiveConfig{Retention: "all"})
	if len(removed) != 0 {
		t.Errorf("expected no removals for 'all', got %d", len(removed))
	}
}

func TestDiff_Basic(t *testing.T) {
	a := "line1\nline2\nline3\n"
	b := "line1\nline2x\nline3\n"
	out := Diff(a, b)
	if !strings.Contains(out, "- line2") || !strings.Contains(out, "+ line2x") {
		t.Errorf("expected line2 diff; got:\n%s", out)
	}
}
