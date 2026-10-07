package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// followup-audit-staleness-git AC-1..AC-3: committed files are judged by
// commit order, so a branch switch or clone that rewrites mtimes no longer
// flags a valid audit; real later edits still do; finished specs are exempt.
func TestAuditStalenessUsesCommitOrderForCommittedFiles(t *testing.T) {
	root := t.TempDir()
	gitRun(t, root, "init", "-q")
	dir := filepath.Join(root, ".hero", "planning", "features", "x")
	os.MkdirAll(dir, 0o755)
	specPath := filepath.Join(dir, "spec.md")
	auditPath := filepath.Join(dir, "delivery-audit.md")
	body := func(status string) string {
		return "---\ntitle: X\nslug: x\ntype: feature\nstatus: " + status + "\n---\n# X\n"
	}
	os.WriteFile(specPath, []byte(body("delivering")), 0o644)
	gitRun(t, root, "add", "-A")
	t.Setenv("GIT_COMMITTER_DATE", "2026-10-01T00:00:00Z")
	gitRun(t, root, "commit", "-q", "-m", "spec")
	os.WriteFile(auditPath, []byte("# Delivery audit — x\n\n**Verdict:** SHIP\n"), 0o644)
	gitRun(t, root, "add", "-A")
	t.Setenv("GIT_COMMITTER_DATE", "2026-10-02T00:00:00Z")
	gitRun(t, root, "commit", "-q", "-m", "audit")
	t.Setenv("GIT_COMMITTER_DATE", "")

	parse := func() *Spec {
		data, _ := os.ReadFile(specPath)
		info, _ := os.Stat(specPath)
		s, err := Parse(string(data), specPath, info.ModTime())
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	// A checkout rewrites the spec's mtime after the audit's: still current.
	future := time.Now().Add(time.Hour)
	os.Chtimes(specPath, future, future)
	if r := FindAuditReport(parse()); !r.Found || r.Stale {
		t.Fatalf("committed audit after committed spec flagged stale by mtime alone: %+v", r)
	}

	// An uncommitted edit after the audit is a real change: stale.
	os.WriteFile(specPath, []byte(body("delivering")+"\nEdited.\n"), 0o644)
	os.Chtimes(specPath, future, future)
	if r := FindAuditReport(parse()); r.Found || !r.Stale {
		t.Fatalf("uncommitted edit after the audit must be stale: %+v", r)
	}

	// Committing that edit after the audit keeps it stale.
	gitRun(t, root, "add", "-A")
	t.Setenv("GIT_COMMITTER_DATE", "2026-10-03T00:00:00Z")
	gitRun(t, root, "commit", "-q", "-m", "edit")
	t.Setenv("GIT_COMMITTER_DATE", "")
	if r := FindAuditReport(parse()); r.Found || !r.Stale {
		t.Fatalf("spec committed after its audit must be stale: %+v", r)
	}

	// A finished spec is never stale: verify rewrites it after the audit.
	os.WriteFile(specPath, []byte(body("completed")), 0o644)
	os.Chtimes(specPath, future, future)
	if r := FindAuditReport(parse()); !r.Found || r.Stale {
		t.Fatalf("finished spec must keep its audit: %+v", r)
	}
}
