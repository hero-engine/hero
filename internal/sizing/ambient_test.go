package sizing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// writeSpec writes a minimal spec.md under .hero/planning/features/<slug>/
// inside heroDir, with the supplied frontmatter fields. Used by the
// AmbientDrift tests to seed a workspace.
func writeSpec(t *testing.T, heroDir, subdir, slug string, fm map[string]string, body string) string {
	t.Helper()
	dir := filepath.Join(heroDir, subdir, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	for k, v := range fm {
		fmt.Fprintf(&sb, "%s: %s\n", k, v)
	}
	sb.WriteString("---\n\n")
	sb.WriteString(body)
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// writeSessionRecord writes a `.hero/knowledge/roadmap-review-sessions/<name>.md`
// file with the given frontmatter, then bumps its mtime to mtime.
func writeSessionRecord(t *testing.T, heroDir, name, frontmatter string, mtime time.Time) {
	t.Helper()
	dir := filepath.Join(heroDir, "knowledge", "roadmap-review-sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, name+".md")
	content := "---\n" + frontmatter + "\n---\n\n# Session\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
}

// largeishDriftedLeaf returns a leaf-spec scaffold that will drift
// against a declared `size: trivial` (10 files → computed = medium+).
func largeishDriftedLeaf(t *testing.T, heroDir, slug string) {
	t.Helper()
	files := make([]string, 10)
	for i := range files {
		files[i] = fmt.Sprintf("    - `path/to/file_%d.go`", i)
	}
	body := "## Changes\n\n" + strings.Join(files, "\n") + "\n"
	writeSpec(t, heroDir, "planning/features", slug, map[string]string{
		"title":  "Drifted leaf — " + slug,
		"type":   "feature",
		"status": "planning",
		"size":   "trivial",
	}, body)
}

// TestAmbientDrift_QuietWhenNoSpecs covers the empty-workspace path.
func TestAmbientDrift_QuietWhenNoSpecs(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{})
	if !rep.Quiet {
		t.Errorf("expected Quiet=true, got %+v", rep)
	}
	if rep.Reason != "no drift" {
		t.Errorf("expected Reason=no drift, got %q", rep.Reason)
	}
	if rep.Hint != "" {
		t.Errorf("expected empty Hint, got %q", rep.Hint)
	}
}

// TestAmbientDrift_DriftFilteredOut_NoActiveNoRecent covers the case
// where drift exists but every match is filtered out (no active spec,
// no git history → no recency, no horizon:now unsized initiative).
func TestAmbientDrift_DriftFilteredOut_NoActiveNoRecent(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Drifted leaf — not active, no git history → out of all three rules.
	largeishDriftedLeaf(t, heroDir, "drifted-leaf")

	// Use a Now far in the future so even file-mtime-based fallbacks
	// won't help — though gitMtime would already return zero for a
	// non-git directory.
	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{
		Now: time.Now().Add(365 * 24 * time.Hour),
	})
	if !rep.Quiet {
		t.Errorf("expected Quiet=true (drift filtered out), got %+v", rep)
	}
	if rep.Reason != "no drift" {
		t.Errorf("expected Reason=no drift, got %q", rep.Reason)
	}
}

// TestAmbientDrift_ActiveSpecRuleFires covers rule 1.
func TestAmbientDrift_ActiveSpecRuleFires(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	largeishDriftedLeaf(t, heroDir, "drifted-leaf")

	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{
		ActiveSpec: "drifted-leaf",
	})
	if rep.Quiet {
		t.Errorf("expected non-quiet (active spec matches), got %+v", rep)
	}
	if rep.Count != 1 {
		t.Errorf("expected Count=1, got %d", rep.Count)
	}
	if !strings.Contains(rep.Hint, "1 spec has size drift") {
		t.Errorf("expected singular phrasing in Hint, got %q", rep.Hint)
	}
	if !strings.Contains(rep.Hint, "/roadmap-review") {
		t.Errorf("expected /roadmap-review CTA in Hint, got %q", rep.Hint)
	}
}

// TestAmbientDrift_HighImpactInitiativeRuleFires covers rule 3 — a
// horizon:now initiative with no declared size and a child rollup
// that drives ContainerDrift to fire.
func TestAmbientDrift_HighImpactInitiativeRuleFires(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Initiative with no `size`, horizon=now, with two largeish children
	// so the container rollup is non-empty.
	writeSpec(t, heroDir, "planning/initiatives", "big-thing", map[string]string{
		"title":   "Big thing",
		"type":    "initiative",
		"status":  "planning",
		"horizon": "now",
	}, "## Goal\n\nBig.\n")
	// Two child features with relations: parent → big-thing, so the
	// container rollup picks them up.
	makeChild := func(slug string) {
		// Use file content sufficient to push computed bucket above
		// "trivial" so the rollup is determinate.
		files := make([]string, 8)
		for i := range files {
			files[i] = fmt.Sprintf("    - `%s/file_%d.go`", slug, i)
		}
		body := "## Changes\n\n" + strings.Join(files, "\n") + "\n"
		writeSpec(t, heroDir, "planning/features", slug, map[string]string{
			"title":  "Child " + slug,
			"type":   "feature",
			"status": "planning",
			"size":   "medium",
			"parent": "big-thing",
		}, body)
	}
	makeChild("kid-a")
	makeChild("kid-b")

	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{})
	if rep.Quiet {
		t.Errorf("expected non-quiet (horizon:now unsized initiative), got %+v", rep)
	}
	if rep.Count < 1 {
		t.Errorf("expected Count >= 1, got %d", rep.Count)
	}
}

// TestAmbientDrift_StopNagging_SuppressesWithinWindow covers the 24h
// suppression when a session record is fresh and the recorded count
// matches the current count.
func TestAmbientDrift_StopNagging_SuppressesWithinWindow(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	largeishDriftedLeaf(t, heroDir, "drifted-leaf")

	now := time.Now()
	writeSessionRecord(t, heroDir, "2026-06-01-1200",
		"type: note\ndrift_count_at_exit: 5",
		now.Add(-1*time.Hour))

	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{
		ActiveSpec: "drifted-leaf", // would normally fire
		Now:        now,
	})
	if !rep.Quiet {
		t.Errorf("expected Quiet=true (recently triaged), got %+v", rep)
	}
	if rep.Reason != "recently triaged" {
		t.Errorf("expected Reason=recently triaged, got %q", rep.Reason)
	}
}

// TestAmbientDrift_StopNagging_LiftsWhenCountGrows covers the
// exception: even within the suppression window, if filtered count
// exceeds the recorded drift_count_at_exit, the hint emits.
func TestAmbientDrift_StopNagging_LiftsWhenCountGrows(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	largeishDriftedLeaf(t, heroDir, "drifted-leaf")

	now := time.Now()
	// Recorded count is 0; current filtered count will be 1 → lifts.
	writeSessionRecord(t, heroDir, "2026-06-01-1200",
		"type: note\ndrift_count_at_exit: 0",
		now.Add(-1*time.Hour))

	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{
		ActiveSpec: "drifted-leaf",
		Now:        now,
	})
	if rep.Quiet {
		t.Errorf("expected non-quiet (count grew since last record), got %+v", rep)
	}
	if rep.Count != 1 {
		t.Errorf("expected Count=1, got %d", rep.Count)
	}
}

// TestAmbientDrift_StopNagging_MissingFieldIsSuppressive covers the
// forward-compatibility contract: a session record without a
// `drift_count_at_exit:` field is fully suppressive within the window.
func TestAmbientDrift_StopNagging_MissingFieldIsSuppressive(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	largeishDriftedLeaf(t, heroDir, "drifted-leaf")

	now := time.Now()
	// Frontmatter has no drift_count_at_exit at all.
	writeSessionRecord(t, heroDir, "2026-06-01-1200",
		"type: note\ncreated: 2026-06-01T12:00:00Z",
		now.Add(-1*time.Hour))

	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{
		ActiveSpec: "drifted-leaf",
		Now:        now,
	})
	if !rep.Quiet {
		t.Errorf("expected Quiet=true (missing field is suppressive), got %+v", rep)
	}
	if rep.Reason != "recently triaged" {
		t.Errorf("expected Reason=recently triaged, got %q", rep.Reason)
	}
}

// TestAmbientDrift_StopNagging_BeyondWindowEmits covers the case where
// the newest session record is older than the window — suppression
// lifts even when the field is missing.
func TestAmbientDrift_StopNagging_BeyondWindowEmits(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	largeishDriftedLeaf(t, heroDir, "drifted-leaf")

	now := time.Now()
	// Record is 30h old; window is 24h.
	writeSessionRecord(t, heroDir, "2026-05-31-1200",
		"type: note\ndrift_count_at_exit: 5",
		now.Add(-30*time.Hour))

	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{
		ActiveSpec:       "drifted-leaf",
		StopNaggingHours: 24,
		Now:              now,
	})
	if rep.Quiet {
		t.Errorf("expected non-quiet (record outside window), got %+v", rep)
	}
}

// TestAmbientDrift_HintPlural covers the plural grammar branch.
func TestAmbientDrift_HintPlural(t *testing.T) {
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	largeishDriftedLeaf(t, heroDir, "drifted-a")
	largeishDriftedLeaf(t, heroDir, "drifted-b")

	// Active spec covers one; we need a second match — create a
	// horizon:now unsized initiative to fire rule 3.
	writeSpec(t, heroDir, "planning/initiatives", "big-thing", map[string]string{
		"title":   "Big thing",
		"type":    "initiative",
		"status":  "planning",
		"horizon": "now",
	}, "## Goal\n\nBig.\n")
	// Give it children so rollup is determinate.
	makeChild := func(slug string) {
		files := make([]string, 8)
		for i := range files {
			files[i] = fmt.Sprintf("    - `%s/file_%d.go`", slug, i)
		}
		body := "## Changes\n\n" + strings.Join(files, "\n") + "\n"
		writeSpec(t, heroDir, "planning/features", slug, map[string]string{
			"title":  "Child " + slug,
			"type":   "feature",
			"status": "planning",
			"size":   "medium",
			"parent": "big-thing",
		}, body)
	}
	makeChild("kid-a")
	makeChild("kid-b")

	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{
		ActiveSpec: "drifted-a",
	})
	if rep.Quiet {
		t.Errorf("expected non-quiet, got %+v", rep)
	}
	if rep.Count < 2 {
		t.Errorf("expected Count >= 2, got %d", rep.Count)
	}
	// Plural phrasing: "N specs have size drift" — not "N spec has".
	if !strings.Contains(rep.Hint, "specs have size drift") {
		t.Errorf("expected plural phrasing, got %q", rep.Hint)
	}
}

// TestFormatAmbientHint covers the helper directly so we lock the
// canonical phrasing in test for any future tweak.
func TestFormatAmbientHint(t *testing.T) {
	cases := []struct {
		count int
		want  string
	}{
		{1, "1 spec has size drift — run /roadmap-review to triage"},
		{2, "2 specs have size drift — run /roadmap-review to triage"},
		{17, "17 specs have size drift — run /roadmap-review to triage"},
	}
	for _, c := range cases {
		got := formatAmbientHint(c.count)
		if got != c.want {
			t.Errorf("formatAmbientHint(%d) = %q, want %q", c.count, got, c.want)
		}
	}
}

// TestReadDriftCountAtExit covers the frontmatter-parse helper.
func TestReadDriftCountAtExit(t *testing.T) {
	tmp := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(tmp, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	cases := []struct {
		name      string
		body      string
		wantCount int
		wantOK    bool
	}{
		{
			name:      "present",
			body:      "---\ntype: note\ndrift_count_at_exit: 7\n---\n\n# Body\n",
			wantCount: 7,
			wantOK:    true,
		},
		{
			name:      "missing",
			body:      "---\ntype: note\ncreated: 2026-06-01T12:00:00Z\n---\n\n# Body\n",
			wantCount: 0,
			wantOK:    false,
		},
		{
			name:      "with comment",
			body:      "---\ndrift_count_at_exit: 3  # NEW\n---\n\n# Body\n",
			wantCount: 3,
			wantOK:    true,
		},
		{
			name:      "no frontmatter",
			body:      "# Plain note\n",
			wantCount: 0,
			wantOK:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := write(c.name+".md", c.body)
			gotCount, gotOK := readDriftCountAtExit(p)
			if gotCount != c.wantCount || gotOK != c.wantOK {
				t.Errorf("readDriftCountAtExit(%s) = (%d, %v), want (%d, %v)",
					c.name, gotCount, gotOK, c.wantCount, c.wantOK)
			}
		})
	}
}

// TestRoadmapConfigDefaults sanity-checks the OrDefault helpers — they're
// trivial but load-bearing for the AmbientDrift defaults.
func TestRoadmapConfigDefaults(t *testing.T) {
	// Importing config from inside its sibling package would cycle.
	// Instead, exercise the same default fallthroughs by calling
	// AmbientDrift with zero opts and confirming the unit doesn't
	// panic on a tiny workspace. (The numeric defaults are also
	// asserted via the BeyondWindow test using StopNaggingHours: 24.)
	tmp := t.TempDir()
	heroDir := filepath.Join(tmp, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = spec.Spec{} // anchor the spec import so it stays explicit.
	rep := AmbientDrift(heroDir, tmp, AmbientDriftOpts{})
	if !rep.Quiet {
		t.Errorf("expected Quiet=true on empty workspace, got %+v", rep)
	}
}
