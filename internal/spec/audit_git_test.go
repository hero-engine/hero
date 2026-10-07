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

// followup-audit-staleness-git AC-3: only archived specs (<hero dir>/specs/)
// are exempt, since verify rewrites and moves them after their audit.
// Pre-release sweep: "archived" is decided against the hero dir Discover
// was given, so a custom hero folder, a repo beneath a planning/ or specs/
// directory, and spec folders with either name are classified correctly.
func TestArchivedSpecIsExemptFromStaleness(t *testing.T) {
	root := filepath.Join(t.TempDir(), "planning", "specs", "repo")
	write := func(rel, content string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	completed := func(slug string) string {
		return "---\ntitle: T\nslug: " + slug + "\ntype: feature\nstatus: completed\n---\n# T\n"
	}
	audit := func(slug string) string { return "# Delivery audit — " + slug + "\n\n**Verdict:** SHIP\n" }
	// A custom hero folder (config "folder": "work"), as config.Load resolves
	// it. Every spec is edited after its audit; explicit mtimes keep the
	// ordering independent of filesystem timestamp resolution.
	auditTime := time.Now().Add(-time.Hour)
	for _, dir := range []string{"specs/y", "specs/planning", "planning/features/z", "planning/features/specs"} {
		slug := filepath.Base(dir)
		write("work/"+dir+"/delivery-audit.md", audit(slug))
		write("work/"+dir+"/spec.md", completed(slug))
		if err := os.Chtimes(filepath.Join(root, "work", dir, "delivery-audit.md"), auditTime, auditTime); err != nil {
			t.Fatal(err)
		}
	}

	specs, err := Discover(filepath.Join(root, "work"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"y": true, "planning": true, "z": false, "specs": false}
	for _, s := range specs {
		w, ok := want[s.Slug]
		if !ok {
			continue
		}
		delete(want, s.Slug)
		if s.Archived != w {
			t.Errorf("%s: Archived = %v, want %v", s.Slug, s.Archived, w)
		}
		r := FindAuditReport(s)
		if w && (!r.Found || r.Stale) {
			t.Errorf("%s: archived completed spec must keep its audit: %+v", s.Slug, r)
		}
		if !w && !r.Stale {
			t.Errorf("%s: unarchived spec edited after its audit must be stale: %+v", s.Slug, r)
		}
	}
	if len(want) != 0 {
		t.Fatalf("not discovered: %v", want)
	}
}
