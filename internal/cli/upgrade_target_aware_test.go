package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/managed"
	"github.com/hero-engine/hero/internal/version"
)

// upgrade_target_aware_test.go — CLI coverage for harness-native, target-
// aware upgrade (harness-native-install-target-aware-upgrade): upgrade
// regenerates only the native instruction files of previously-installed
// targets, backfills a pre-state repo, and prunes orphans opt-in only.

// upgradeTestFS is a minimal content FS for upgrade tests.
func upgradeTestFS() fstest.MapFS {
	return fstest.MapFS{
		"agents/engineer.md":       {Data: []byte("---\nname: engineer\ndescription: e\n---\n# e\n")},
		"commands/design.md":       {Data: []byte("---\ndescription: d\n---\n# d\n")},
		"skills/go-stack/SKILL.md": {Data: []byte("---\ndescription: s\n---\n# s\n")},
	}
}

func stampOldVersion(t *testing.T, heroDir string) {
	t.Helper()
	if err := version.StampInit(heroDir, "0.9.0"); err != nil {
		t.Fatalf("StampInit: %v", err)
	}
}

// Claude was never a target → upgrade must NOT create CLAUDE.md.
func TestUpgrade_ClaudeNeverInstalled_NoCLAUDEMd(t *testing.T) {
	env := newTestEnv(t)
	upgradeContentFS = upgradeTestFS()
	defer func() { upgradeContentFS = nil }()
	stampOldVersion(t, env.heroDir)
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	// Persist an opencode-only install (no claude).
	if err := install.PersistInferredTargets(env.dir, []install.Target{install.TargetOpenCode}, "0.9.0"); err != nil {
		t.Fatalf("persist: %v", err)
	}
	// Give opencode a content dir so its installer has somewhere to land.
	if err := os.MkdirAll(filepath.Join(env.dir, ".opencode", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := runCmd("upgrade"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	if _, err := os.Stat(filepath.Join(env.dir, "CLAUDE.md")); err == nil {
		t.Errorf("upgrade must NOT create CLAUDE.md when claude was never installed")
	}
	if _, err := os.Stat(filepath.Join(env.dir, "AGENTS.md")); err != nil {
		t.Errorf("opencode's AGENTS.md should have been regenerated: %v", err)
	}
}

// Claude WAS a target → upgrade regenerates CLAUDE.md's managed region.
func TestUpgrade_ClaudeInstalled_RegeneratesCLAUDEMd(t *testing.T) {
	env := newTestEnv(t)
	upgradeContentFS = upgradeTestFS()
	defer func() { upgradeContentFS = nil }()
	stampOldVersion(t, env.heroDir)
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	// Persist a claude install with a content dir and a stale CLAUDE.md
	// managed region.
	if err := install.PersistInferredTargets(env.dir, []install.Target{install.TargetClaude}, "0.9.0"); err != nil {
		t.Fatalf("persist: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(env.dir, ".claude", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	stale := "# CLAUDE.md\n\n" + managed.RenderManagedRegion("v0", "STALE BODY") + "\n"
	if err := os.WriteFile(filepath.Join(env.dir, "CLAUDE.md"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runCmd("upgrade"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(env.dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	if strings.Contains(string(data), "STALE BODY") {
		t.Errorf("CLAUDE.md managed region should have been regenerated, still stale:\n%s", string(data))
	}
	if !strings.Contains(string(data), "hero:managed-start") {
		t.Errorf("CLAUDE.md missing managed markers after upgrade")
	}
	// No phantom AGENTS.md for a claude-only repo.
	if _, err := os.Stat(filepath.Join(env.dir, "AGENTS.md")); err == nil {
		t.Errorf("claude-only upgrade must not create AGENTS.md")
	}
}

// Backfill: a pre-state repo with only a Hero-managed CLAUDE.md stub (no
// install-state, no content dirs) infers {claude}, persists it, and
// regenerates CLAUDE.md.
func TestUpgrade_Backfill_StubOnly(t *testing.T) {
	env := newTestEnv(t)
	upgradeContentFS = upgradeTestFS()
	defer func() { upgradeContentFS = nil }()
	stampOldVersion(t, env.heroDir)
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	stub := "# CLAUDE.md\n\n" + managed.RenderManagedRegion("v0", "OLD") + "\n"
	if err := os.WriteFile(filepath.Join(env.dir, "CLAUDE.md"), []byte(stub), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runCmd("upgrade"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// Inferred set persisted.
	got := install.PreviouslyInstalledTargets(env.dir)
	if len(got) != 1 || got[0] != install.TargetClaude {
		t.Errorf("backfill should have persisted {claude}, got %v", got)
	}
	// CLAUDE.md regenerated, no AGENTS.md conjured.
	data, _ := os.ReadFile(filepath.Join(env.dir, "CLAUDE.md"))
	if strings.Contains(string(data), "OLD") {
		t.Errorf("CLAUDE.md should have been regenerated during backfill")
	}
	if _, err := os.Stat(filepath.Join(env.dir, "AGENTS.md")); err == nil {
		t.Errorf("backfill must not create AGENTS.md for a claude-only repo")
	}
}

// Orphan maintain-not-delete: a claude-only upgrade with a legacy phantom
// AGENTS.md (Hero-managed) leaves it in place by default, refreshed.
func TestUpgrade_OrphanAgentsMd_MaintainedByDefault(t *testing.T) {
	env := newTestEnv(t)
	upgradeContentFS = upgradeTestFS()
	defer func() { upgradeContentFS = nil }()
	stampOldVersion(t, env.heroDir)
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	if err := install.PersistInferredTargets(env.dir, []install.Target{install.TargetClaude}, "0.9.0"); err != nil {
		t.Fatalf("persist: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(env.dir, ".claude", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Phantom AGENTS.md with user content outside the markers.
	phantom := "# AGENTS.md\n\n" + managed.RenderManagedRegion("v0", "X") + "\nUSER KEEP\n"
	if err := os.WriteFile(filepath.Join(env.dir, "AGENTS.md"), []byte(phantom), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runCmd("upgrade"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(env.dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("phantom AGENTS.md must NOT be deleted by default: %v", err)
	}
	if !strings.Contains(string(data), "USER KEEP") {
		t.Errorf("user content outside markers must be preserved")
	}
}

// Orphan prune opt-in: --prune-orphaned-instruction-files removes a
// Hero-managed-only phantom AGENTS.md; a user-content phantom is preserved.
func TestUpgrade_OrphanPrune_OptIn(t *testing.T) {
	env := newTestEnv(t)
	upgradeContentFS = upgradeTestFS()
	defer func() { upgradeContentFS = nil }()
	stampOldVersion(t, env.heroDir)
	rootCmd.Version = "1.0.0"
	defer func() { rootCmd.Version = "" }()

	if err := install.PersistInferredTargets(env.dir, []install.Target{install.TargetClaude}, "0.9.0"); err != nil {
		t.Fatalf("persist: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(env.dir, ".claude", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Managed-only phantom (H1 + region, no user content) → prunable.
	managedOnly := "# AGENTS.md\n\n" + managed.RenderManagedRegion("v0", "X") + "\n"
	if err := os.WriteFile(filepath.Join(env.dir, "AGENTS.md"), []byte(managedOnly), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runCmd("upgrade", "--prune-orphaned-instruction-files"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	if _, err := os.Stat(filepath.Join(env.dir, "AGENTS.md")); err == nil {
		t.Errorf("managed-only phantom AGENTS.md should have been pruned with the flag")
	}
}
