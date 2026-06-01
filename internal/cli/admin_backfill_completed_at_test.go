package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRun executes a git command in dir with deterministic identity so
// commits succeed in CI environments without a configured user.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE=2025-01-15T08:00:00Z",
		"GIT_COMMITTER_DATE=2025-01-15T08:00:00Z",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestBackfill_HappyPath(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	env.addSpec("specs/csv-export/spec.md", `---
title: CSV Export
type: feature
status: completed
---
# CSV Export
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add csv-export")

	out, err := runCmd("admin", "backfill-completed-at")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if !strings.Contains(out, "Stamped: 1") {
		t.Errorf("expected 'Stamped: 1' in output:\n%s", out)
	}

	data, err := os.ReadFile(filepath.Join(env.heroDir, "specs/csv-export/spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "completed_at: 2025-01-15T08:00:00Z") {
		t.Errorf("git-derived stamp missing or wrong:\n%s", body)
	}
}

func TestBackfill_SkipsAlreadyStamped(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	env.addSpec("specs/csv-export/spec.md", `---
title: CSV Export
type: feature
status: completed
completed_at: 2024-06-01T12:00:00Z
---
# CSV Export
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add csv-export pre-stamped")

	out, err := runCmd("admin", "backfill-completed-at")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if !strings.Contains(out, "Skipped (already stamped): 1") {
		t.Errorf("expected skipped count of 1:\n%s", out)
	}

	data, err := os.ReadFile(filepath.Join(env.heroDir, "specs/csv-export/spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "completed_at: 2024-06-01T12:00:00Z") {
		t.Errorf("original stamp was overwritten:\n%s", string(data))
	}
}

func TestBackfill_SkipsNoGitHistory(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	// Add the spec to the working tree but NEVER commit it. git log
	// returns empty, so backfill must report "no git history" rather
	// than synthesizing a time.
	env.addSpec("specs/csv-export/spec.md", `---
title: CSV Export
type: feature
status: completed
---
# CSV Export
`)

	out, err := runCmd("admin", "backfill-completed-at")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if !strings.Contains(out, "No git history: 1") {
		t.Errorf("expected 'No git history: 1':\n%s", out)
	}

	data, err := os.ReadFile(filepath.Join(env.heroDir, "specs/csv-export/spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "completed_at:") {
		t.Errorf("uncommitted spec should not be stamped:\n%s", string(data))
	}
}

func TestBackfill_DryRun(t *testing.T) {
	env := newTestEnv(t)
	gitRun(t, env.dir, "init", "-q")

	env.addSpec("specs/csv-export/spec.md", `---
title: CSV Export
type: feature
status: completed
---
# CSV Export
`)
	gitRun(t, env.dir, "add", ".")
	gitRun(t, env.dir, "commit", "-q", "-m", "add csv-export")

	out, err := runCmd("admin", "backfill-completed-at", "--dry-run")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if !strings.Contains(out, "Stamped: 1") {
		t.Errorf("expected 'Stamped: 1' (counts shown even in dry-run):\n%s", out)
	}
	if !strings.Contains(out, "(dry-run") {
		t.Errorf("expected dry-run marker:\n%s", out)
	}

	data, err := os.ReadFile(filepath.Join(env.heroDir, "specs/csv-export/spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "completed_at:") {
		t.Errorf("dry-run must not write:\n%s", string(data))
	}
}
