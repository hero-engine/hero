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

	// Hand-flipping status to completed while still in planning/ keeps the
	// check: an uncommitted edit after the audit is still stale.
	os.WriteFile(specPath, []byte(body("completed")), 0o644)
	os.Chtimes(specPath, future, future)
	if r := FindAuditReport(parse()); r.Found || !r.Stale {
		t.Fatalf("completed-but-unarchived spec edited after its audit must be stale: %+v", r)
	}

	// A spec and audit committed together are one reviewed unit.
	os.WriteFile(auditPath, []byte("# Delivery audit — x\n\n**Verdict:** SHIP\n\nRe-audited.\n"), 0o644)
	gitRun(t, root, "add", "-A")
	t.Setenv("GIT_COMMITTER_DATE", "2026-10-04T00:00:00Z")
	gitRun(t, root, "commit", "-q", "-m", "together")
	t.Setenv("GIT_COMMITTER_DATE", "")
	os.Chtimes(specPath, future, future)
	if r := FindAuditReport(parse()); !r.Found || r.Stale {
		t.Fatalf("spec and audit committed together must be accepted: %+v", r)
	}
}

// followup-audit-staleness-git AC-3: only archived specs (.hero/specs/) are
// exempt, since verify rewrites and moves them after their audit.
func TestArchivedSpecIsExemptFromStaleness(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".hero", "specs", "y")
	os.MkdirAll(dir, 0o755)
	specPath := filepath.Join(dir, "spec.md")
	os.WriteFile(filepath.Join(dir, "delivery-audit.md"), []byte("# Delivery audit — y\n\n**Verdict:** SHIP\n"), 0o644)
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(specPath, []byte("---\ntitle: Y\nslug: y\ntype: feature\nstatus: completed\n---\n# Y\n"), 0o644)
	data, _ := os.ReadFile(specPath)
	info, _ := os.Stat(specPath)
	s, err := Parse(string(data), specPath, info.ModTime())
	if err != nil {
		t.Fatal(err)
	}
	if r := FindAuditReport(s); !r.Found || r.Stale {
		t.Fatalf("archived completed spec must keep its audit: %+v", r)
	}
}

// Pre-release sweep: the archive exemption does not depend on the hero
// folder's name, and only the directory under the hero folder decides.
func TestArchivedPathIgnoresFolderName(t *testing.T) {
	root := t.TempDir()
	writeConfig := func(dir string) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "hero.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	workspace := func(rel string) string {
		dir := filepath.Join(root, filepath.FromSlash(rel))
		writeConfig(dir)
		return dir
	}
	custom := workspace("a/.workspace")
	underPlanning := workspace("planning/repo/.hero")
	underSpecs := workspace("specs/repo/.hero")
	noConfig := filepath.Join(root, "bare", ".hero")
	// A .hero without hero.json beneath a directory that holds one.
	writeConfig(filepath.Join(root, "outer"))
	nested := filepath.Join(root, "outer", "repo", ".hero")
	// A stray hero.json inside an archived spec folder.
	writeConfig(filepath.Join(custom, "specs", "stray"))
	for path, want := range map[string]bool{
		filepath.Join(custom, "specs", "x", "spec.md"):                true,
		filepath.Join(custom, "specs", "init", "child", "spec.md"):    true,
		filepath.Join(custom, "specs", "planning", "spec.md"):         true,
		filepath.Join(custom, "planning", "features", "x", "spec.md"): false,
		filepath.Join(custom, "planning", "x", "specs", "spec.md"):    false,
		filepath.Join(underPlanning, "specs", "x", "spec.md"):         true,
		filepath.Join(underSpecs, "planning", "x", "spec.md"):         false,
		filepath.Join(noConfig, "specs", "x", "spec.md"):              true,
		filepath.Join(noConfig, "planning", "x", "spec.md"):           false,
		filepath.Join(nested, "specs", "x", "spec.md"):                true,
		filepath.Join(nested, "planning", "x", "spec.md"):             false,
		filepath.Join(custom, "specs", "stray", "spec.md"):            true,
		filepath.Join(root, "elsewhere", "specs", "x", "spec.md"):     false,
	} {
		if got := isArchivedPath(path); got != want {
			t.Errorf("isArchivedPath(%q) = %v, want %v", path, got, want)
		}
	}
}
