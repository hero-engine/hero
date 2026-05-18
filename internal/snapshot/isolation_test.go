package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// TestIsolation_FrontmatterFlags asserts every archive file carries
// historical: true and not_current: true. Invariant #2 from the
// project-snapshot spec.
func TestIsolation_FrontmatterFlags(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	mustMkdirT(t, heroDir)
	out, err := MaybeWrite(MaybeWriteInput{
		Rendered: []byte("# Snap\n\nbody\n"),
		HeroDir:  heroDir,
		Now:      time.Now(),
		Triggers: []TriggerHit{{Trigger: TriggerManual, Label: "test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Written) != 1 {
		t.Fatalf("no archive written")
	}
	data, _ := os.ReadFile(out.Written[0].Path)
	body := string(data)
	if !strings.Contains(body, "historical: true") {
		t.Errorf("missing historical: true")
	}
	if !strings.Contains(body, "not_current: true") {
		t.Errorf("missing not_current: true")
	}
}

// TestIsolation_BannerMandatory asserts the writer always emits the
// historical-archive banner at the top of the body. Invariant #3.
func TestIsolation_BannerMandatory(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	mustMkdirT(t, heroDir)
	out, _ := MaybeWrite(MaybeWriteInput{
		Rendered: []byte("# Snap\n"),
		HeroDir:  heroDir,
		Now:      time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
		Triggers: []TriggerHit{{Trigger: TriggerMilestone, Label: "v1.0"}},
	})
	if len(out.Written) != 1 {
		t.Fatal("no archive written")
	}
	data, _ := os.ReadFile(out.Written[0].Path)
	body := string(data)
	// Banner must follow frontmatter immediately.
	if !strings.Contains(body, "Historical archive captured 2026-05-18") {
		t.Errorf("missing banner line; got:\n%s", body)
	}
	// Read() validates the banner — verify it refuses files without it.
	tampered := strings.ReplaceAll(body, "Historical archive captured", "Live data dump captured")
	tmpPath := filepath.Join(heroDir, "snapshots", "tampered.md")
	_ = os.WriteFile(tmpPath, []byte(tampered), 0o644)
	if _, err := Read(tmpPath); err == nil {
		t.Errorf("Read should reject a file missing the banner")
	}
}

// TestIsolation_DiscoverSkipsSnapshots asserts the spec corpus
// discoverer does not return any archive file as a spec. Invariant
// #1 — snapshots are excluded from default search/index paths.
func TestIsolation_DiscoverSkipsSnapshots(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	mustMkdirT(t, filepath.Join(heroDir, "snapshots"))

	// Write an archive that looks like a spec (so a naive walker would
	// pick it up).
	body := `---
title: This Looks Like A Spec
type: feature
status: planning
snapshot_date: 2026-05-18
trigger: manual
historical: true
not_current: true
---

> Historical archive captured 2026-05-18.

# Body
`
	archivePath := filepath.Join(heroDir, "snapshots", "2026-05-18.md")
	if err := os.WriteFile(archivePath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// Also write a real spec to confirm discovery still runs.
	specDir := filepath.Join(heroDir, "planning", "features", "real-feature")
	mustMkdirT(t, specDir)
	realSpec := `---
title: Real Feature
type: feature
status: planning
---

# Real
`
	_ = os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(realSpec), 0o644)

	specs, err := spec.Discover(heroDir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, s := range specs {
		if strings.Contains(s.Path, "/snapshots/") {
			t.Errorf("Discover returned an archive as a spec: %s", s.Path)
		}
	}
	// Real spec should still be found.
	found := false
	for _, s := range specs {
		if s.Slug == "real-feature" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Discover skipped a non-snapshot spec")
	}
}

// TestIsolation_ListReturnsArchives confirms the explicit
// history-query path DOES return archives (the opt-in surface).
// Symmetric to the Discover test.
func TestIsolation_ListReturnsArchives(t *testing.T) {
	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")
	mustMkdirT(t, heroDir)
	_, err := MaybeWrite(MaybeWriteInput{
		Rendered: []byte("# Snap\n"),
		HeroDir:  heroDir,
		Now:      time.Now(),
		Triggers: []TriggerHit{{Trigger: TriggerManual, Label: "list-test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	archives, err := List(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(archives) != 1 {
		t.Errorf("List returned %d archives, want 1", len(archives))
	}
}

func mustMkdirT(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
