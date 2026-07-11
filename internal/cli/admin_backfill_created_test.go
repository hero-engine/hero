package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackfillCreated_HappyPath(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	// No created: field — CreatedAt will fall back to mtime until stamped.
	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add csv-export")

	out, err := runCmd("admin", "backfill-created")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if !strings.Contains(out, "Stamped: 1") {
		t.Errorf("expected 'Stamped: 1':\n%s", out)
	}

	body := readSpecRaw(t, env.heroDir, "planning/features/csv-export/spec.md")
	// gitRun commits with GIT_AUTHOR_DATE 2025-01-15 — the first-commit date.
	if !strings.Contains(body, "created: 2025-01-15") {
		t.Errorf("git-first-commit stamp missing or wrong:\n%s", body)
	}
}

func TestBackfillCreated_SkipsAlreadyStamped(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
created: 2024-06-01
---
# CSV Export
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add pre-stamped")

	out, err := runCmd("admin", "backfill-created")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if !strings.Contains(out, "Skipped (already stamped): 1") {
		t.Errorf("expected skipped count of 1:\n%s", out)
	}
	body := readSpecRaw(t, env.heroDir, "planning/features/csv-export/spec.md")
	if !strings.Contains(body, "created: 2024-06-01") {
		t.Errorf("original created: was overwritten:\n%s", body)
	}
}

func TestBackfillCreated_UncommittedStampsToday(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	// Never committed → no git history. Unlike completed_at, created backfill
	// synthesizes today's date: for a just-authored file, today IS the
	// creation date.
	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)

	out, err := runCmd("admin", "backfill-created")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if !strings.Contains(out, "Stamped: 1") {
		t.Errorf("expected uncommitted spec to be stamped with today:\n%s", out)
	}
	body := readSpecRaw(t, env.heroDir, "planning/features/csv-export/spec.md")
	today := time.Now().UTC().Format("2006-01-02")
	if !strings.Contains(body, "created: "+today) {
		t.Errorf("expected created: %s, got:\n%s", today, body)
	}
}

func TestBackfillCreated_DryRun(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add csv-export")

	out, err := runCmd("admin", "backfill-created", "--dry-run")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if !strings.Contains(out, "Stamped: 1") || !strings.Contains(out, "(dry-run") {
		t.Errorf("expected dry-run counts + marker:\n%s", out)
	}
	body := readSpecRaw(t, env.heroDir, "planning/features/csv-export/spec.md")
	if strings.Contains(body, "created:") {
		t.Errorf("dry-run must not write:\n%s", body)
	}
}

func TestBackfillCreated_Idempotent(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add csv-export")

	if _, err := runCmd("admin", "backfill-created"); err != nil {
		t.Fatalf("first backfill: %v", err)
	}
	out, err := runCmd("admin", "backfill-created")
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if !strings.Contains(out, "Stamped: 0") || !strings.Contains(out, "Skipped (already stamped): 1") {
		t.Errorf("second run should stamp nothing:\n%s", out)
	}
}

// TestCheckReconcileStampsCreated covers AC-6: `hero check --reconcile`
// self-heals a work spec missing created: by stamping it from the first git
// commit, without a manual backfill.
func TestCheckReconcileStampsCreated(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add csv-export")

	if _, err := runCmd("check", "--reconcile"); err != nil {
		t.Fatalf("check --reconcile: %v", err)
	}

	body := readSpecRaw(t, env.heroDir, "planning/features/csv-export/spec.md")
	if !strings.Contains(body, "created: 2025-01-15") {
		t.Errorf("check --reconcile should stamp created: from first commit:\n%s", body)
	}
}

// TestCheckReportsMissingCreatedWithoutReconcile confirms plain `hero check`
// reports the gap but writes nothing.
func TestCheckReportsMissingCreatedWithoutReconcile(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add csv-export")

	out, err := runCmd("check")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(out, "missing created:") && !strings.Contains(out, "Missing created:") {
		t.Errorf("plain check should report missing created:\n%s", out)
	}
	body := readSpecRaw(t, env.heroDir, "planning/features/csv-export/spec.md")
	if strings.Contains(body, "created:") {
		t.Errorf("plain check must not write created:\n%s", body)
	}
}

func readSpecRaw(t *testing.T, heroDir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(heroDir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
