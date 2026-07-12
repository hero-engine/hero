package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureManagedGitignoreBlock_CreatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	if err := ensureManagedGitignoreBlock(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(got)
	for _, want := range []string{
		gitignoreMarkerStart,
		gitignoreMarkerEnd,
		".hero/hero.local.json",
		".hero/graph.db",
		".hero/graph.db-wal",
		".hero/graph.db-shm",
		".hero/index.db",
		".hero/index.db-wal",
		".hero/index.db-shm",
		".hero/next/*.local.md",
		".hero/knowledge/code/",
		".hero/satellites.local.json",
		".hero/cache/",
		".hero/sessions/",
		".hero/install-state.json",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}

	// Guardrail: events.log is the committed append-only ledger (velocity/
	// pulse read from it; graph.db is merely a regenerable cache of it). It
	// must never be ignored — fence against a future edit adding it.
	if strings.Contains(body, "events.log") {
		t.Errorf("events.log must NOT be gitignored (source-of-truth ledger):\n%s", body)
	}
}

func TestEnsureManagedGitignoreBlock_PreservesUserContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	original := `# user-managed
node_modules/
*.log

# more user content
secrets.env
`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureManagedGitignoreBlock(path); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(path)
	body := string(got)
	for _, want := range []string{"node_modules/", "*.log", "secrets.env"} {
		if !strings.Contains(body, want) {
			t.Errorf("user content lost: %q missing from:\n%s", want, body)
		}
	}
}

func TestEnsureManagedGitignoreBlock_IdempotentOnReRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	if err := os.WriteFile(path, []byte("user-line-before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureManagedGitignoreBlock(path); err != nil {
		t.Fatal(err)
	}
	once, _ := os.ReadFile(path)

	if err := ensureManagedGitignoreBlock(path); err != nil {
		t.Fatal(err)
	}
	twice, _ := os.ReadFile(path)

	if string(once) != string(twice) {
		t.Errorf("not idempotent:\nonce  = %q\ntwice = %q", once, twice)
	}
	// Block should appear exactly once.
	if strings.Count(string(twice), gitignoreMarkerStart) != 1 {
		t.Errorf("marker duplicated:\n%s", twice)
	}
}

func TestEnsureManagedGitignoreBlock_RefreshesUpdatedEntries(t *testing.T) {
	// Simulate a stale managed block from a prior version that had a
	// different set of entries. The refresh should replace it with
	// the current canonical list, not concatenate.
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")

	stale := gitignoreMarkerStart + "\nold-entry\n" + gitignoreMarkerEnd + "\n"
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ensureManagedGitignoreBlock(path); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "old-entry") {
		t.Errorf("stale entry survived refresh:\n%s", got)
	}
	if !strings.Contains(string(got), ".hero/next/*.local.md") {
		t.Errorf("new entries missing after refresh:\n%s", got)
	}
	// An existing install re-running `hero init` must gain the machine-local
	// entries added after its managed block was first written — this is the
	// zero-migration rollout guarantee.
	for _, want := range []string{".hero/cache/", ".hero/sessions/", ".hero/install-state.json"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("entry %q missing after refresh (rollout gap):\n%s", want, got)
		}
	}
}
