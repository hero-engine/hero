package peering

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

// TestManifestDefaultEmpty checks the principle of least authority:
// a freshly-init'd workspace publishes ZERO conventions until they
// are explicitly marked peer-surface.
func TestManifestDefaultEmpty(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "knowledge", "conventions"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.PeerID = "11111111-1111-4111-8111-111111111111"
	if err := cfg.Save(root); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	// One unmarked convention — should NOT be published.
	convDir := filepath.Join(heroDir, "knowledge", "conventions", "naming")
	if err := os.MkdirAll(convDir, 0o755); err != nil {
		t.Fatalf("mkdir conv: %v", err)
	}
	content := "---\ntype: convention\nstatus: active\ntags: [internal]\n---\n\n# Naming\n"
	if err := os.WriteFile(filepath.Join(convDir, "spec.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write conv: %v", err)
	}

	m, err := GenerateManifest(root)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}
	if m.Repo.PeerID != cfg.PeerID {
		t.Errorf("peer_id mismatch: %q vs %q", m.Repo.PeerID, cfg.PeerID)
	}
	if len(m.Conventions) != 0 {
		t.Errorf("default publish set should be empty, got %d entries", len(m.Conventions))
	}
}

// TestManifestPublishesOptIns checks both opt-in mechanisms:
// frontmatter tag and config glob.
func TestManifestPublishesOptIns(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "knowledge", "conventions"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.PeerID = "22222222-2222-4222-8222-222222222222"
	cfg.Peering = &config.PeeringConfig{
		Display:            "Hero Backend",
		ScopeHint:          "backend",
		PublishConventions: []string{"auth-*"},
	}
	if err := cfg.Save(root); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	makeConv := func(slug, frontmatter string) {
		dir := filepath.Join(heroDir, "knowledge", "conventions", slug)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", slug, err)
		}
		body := "---\ntype: convention\nstatus: active\n" + frontmatter + "---\n\n# " + slug + "\n"
		if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", slug, err)
		}
	}
	// (a) Tagged convention — published via peer-surface tag.
	makeConv("error-envelope", "tags: [peer-surface, http-response]\n")
	// (b) Glob-matched convention — published via publish_conventions.
	makeConv("auth-bearer-token", "tags: []\n")
	// (c) Unmarked — should NOT appear.
	makeConv("naming", "tags: [internal]\n")

	m, err := GenerateManifest(root)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}
	if m.Repo.Display != "Hero Backend" {
		t.Errorf("display: %q", m.Repo.Display)
	}
	if m.Repo.ScopeHint != "backend" {
		t.Errorf("scope_hint: %q", m.Repo.ScopeHint)
	}
	if len(m.Conventions) != 2 {
		t.Fatalf("expected 2 conventions, got %d (%+v)", len(m.Conventions), m.Conventions)
	}
	have := map[string]bool{}
	for _, c := range m.Conventions {
		have[c.Slug] = true
	}
	if !have["error-envelope"] {
		t.Error("missing error-envelope (tagged opt-in)")
	}
	if !have["auth-bearer-token"] {
		t.Error("missing auth-bearer-token (glob opt-in)")
	}
	if have["naming"] {
		t.Error("unmarked naming convention should NOT be published")
	}
}

// TestWriteAndGenerateAndWriteManifest verifies the wrapper writes a
// file and that the content is parseable.
func TestGenerateAndWriteManifest(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.PeerID = "33333333-3333-4333-8333-333333333333"
	_ = cfg.Save(root)

	if err := GenerateAndWriteManifest(root); err != nil {
		t.Fatalf("GenerateAndWriteManifest: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(heroDir, PeerManifestFileName))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(data), "peer_id: 33333333-3333-4333-8333-333333333333") {
		t.Errorf("manifest missing peer_id: %s", data)
	}
}

// TestManifestNameSurvivesLinkedWorktree is the regression case: running
// `hero index` from inside a linked worktree whose directory name is
// unrelated to the project (e.g. a session ID) must still publish the
// main checkout's directory name as repo.name, not the worktree's own
// basename. See gitutil.RepoRoot.
func TestManifestNameSurvivesLinkedWorktree(t *testing.T) {
	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Main checkout, named like a real project — this is the name that
	// must end up in the manifest regardless of which worktree generated it.
	parent := t.TempDir()
	mainDir := filepath.Join(parent, "inkwyrm")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatalf("mkdir mainDir: %v", err)
	}
	runGit(mainDir, "init", "-b", "main")
	runGit(mainDir, "config", "user.email", "test@test.com")
	runGit(mainDir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(mainDir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(mainDir, "add", ".")
	runGit(mainDir, "commit", "-m", "initial commit")

	// Linked worktree with a directory name unrelated to the project —
	// this is what previously leaked into repo.name.
	worktreeDir := filepath.Join(parent, "dreamy-wilbur-ff9b6f")
	runGit(mainDir, "worktree", "add", "-b", "wt-branch", worktreeDir)

	heroDir := filepath.Join(worktreeDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("mkdir heroDir: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.PeerID = "44444444-4444-4444-8444-444444444444"
	if err := cfg.Save(worktreeDir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	m, err := GenerateManifest(worktreeDir)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}
	if m.Repo.Name != "inkwyrm" {
		t.Errorf("Repo.Name = %q, want %q (main checkout dir, not the worktree dir %q)",
			m.Repo.Name, "inkwyrm", filepath.Base(worktreeDir))
	}
}
