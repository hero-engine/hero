package peering

import (
	"os"
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

// TestManifestNamePrefersConfigOverDirectory guards against the
// worktree-name-stamping bug: repo.name must come from the committed
// hero.json:name when set, not from the live working directory's
// basename (which differs per git worktree even though every worktree
// shares the same hero.json).
func TestManifestNamePrefersConfigOverDirectory(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(filepath.Join(heroDir, "knowledge", "conventions"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.PeerID = "11111111-1111-4111-8111-111111111111"
	cfg.Name = "canonical-repo-name"
	if err := cfg.Save(root); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	m, err := GenerateManifest(root)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}
	if m.Repo.Name != "canonical-repo-name" {
		t.Errorf("repo.name = %q, want the persisted cfg.Name, not the tempdir's own basename (%q)", m.Repo.Name, filepath.Base(root))
	}
}

// TestManifestNameFallsBackToDirectory checks the pre-migration
// fallback: a workspace with no persisted name (older workspaces,
// from before `hero init` started minting one) keeps the old
// directory-basename behavior rather than failing.
func TestManifestNameFallsBackToDirectory(t *testing.T) {
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

	m, err := GenerateManifest(root)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}
	if m.Repo.Name != filepath.Base(root) {
		t.Errorf("repo.name = %q, want fallback to directory basename %q", m.Repo.Name, filepath.Base(root))
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
