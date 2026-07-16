package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plantSkillDir creates a skill dir at dest/<name>/SKILL.md, standing in
// for one an earlier install left behind.
func plantSkillDir(t *testing.T, dest, name string) string {
	t.Helper()
	dir := filepath.Join(dest, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return dir
}

func mustNotExist(t *testing.T, path, why string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("%s: %s still exists", why, path)
	}
}

func mustExist(t *testing.T, path, why string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("%s: %s missing (%v)", why, path, err)
	}
}

// A command-<name> dir with no canonical source is an orphan Hero rendered
// under a namespace it owns. Codex's loader walks .agents/skills/*/SKILL.md,
// so leaving it there keeps loading a workflow that no longer exists.
func TestCodexInstallPrunesOrphanedCommandSkill(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	skillsDest := filepath.Join(targetDir, ".agents", "skills")
	orphan := plantSkillDir(t, skillsDest, "command-prime")
	// Superseded layout: an older renderer used this prefix. Nothing
	// writes it now, so every such dir is dead.
	legacyOrphan := plantSkillDir(t, skillsDest, "source-command-handoff")

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetCodex,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	mustNotExist(t, orphan, "orphaned command skill not pruned")
	mustNotExist(t, legacyOrphan, "superseded-layout skill not pruned")

	// The install's own output survives the prune.
	mustExist(t, filepath.Join(skillsDest, "command-design", "SKILL.md"), "rendered command skill")
	mustExist(t, filepath.Join(skillsDest, "spec-format", "SKILL.md"), "rendered canonical skill")
}

// .agents/skills is a cross-tool standard location and .claude/skills holds
// hand-written team skills. Hero must not remove a dir it cannot prove it
// wrote — the guard that makes the prune safe to run at all.
func TestInstallLeavesForeignSkillDirsAlone(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target Target
		dest   []string
	}{
		{"codex", TargetCodex, []string{".agents", "skills"}},
		{"claude", TargetClaude, []string{".claude", "skills"}},
		{"opencode", TargetOpenCode, []string{".opencode", "skills"}},
		{"copilot", TargetCopilot, []string{".github", "skills"}},
		{"generic", TargetGeneric, []string{".ai", "skills"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sourceDir := t.TempDir()
			targetDir := t.TempDir()
			createContent(t, sourceDir)

			dest := filepath.Join(targetDir, filepath.Join(tc.dest...))
			foreign := plantSkillDir(t, dest, "our-team-deploy-runbook")

			opts := Options{
				SourceDir: sourceDir,
				Target:    tc.target,
				Mode:      ModeProject,
				TargetDir: targetDir,
			}
			if _, err := Run(opts); err != nil {
				t.Fatalf("Run: %v", err)
			}

			mustExist(t, foreign, "user-authored skill was deleted")
		})
	}
}

// A `command-foo` dir under .claude/skills is the user's — Claude reads
// commands from .claude/commands, so Hero never renders that prefix there
// and has no claim to it.
func TestClaudeDoesNotClaimCommandPrefixInSkills(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	dest := filepath.Join(targetDir, ".claude", "skills")
	userSkill := plantSkillDir(t, dest, "command-center")

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	mustExist(t, userSkill, "user skill under a prefix Hero doesn't own at this dest")
}

// The durable half: a canonical skill that disappears between two installs
// is pruned on the second, because the first recorded it as Hero's.
func TestInstallPrunesSkillDroppedFromSourceViaManifest(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)
	// install-state.json (where the manifest lives) is only written into an
	// existing .hero/ workspace.
	if err := os.MkdirAll(filepath.Join(targetDir, ".hero"), 0o755); err != nil {
		t.Fatalf("MkdirAll .hero: %v", err)
	}

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: targetDir,
		Force:     true,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	installed := filepath.Join(targetDir, ".claude", "skills", "test-strategy")
	mustExist(t, installed, "first install")

	st, err := ReadInstallState(targetDir)
	if err != nil {
		t.Fatalf("ReadInstallState: %v", err)
	}
	if got := st.Targets["claude"].SkillDirs; len(got) != 2 {
		t.Fatalf("expected 2 recorded skill dirs, got %v", got)
	}

	// The skill's canonical source is renamed away.
	if err := os.Remove(filepath.Join(sourceDir, "skills", "test-strategy.md")); err != nil {
		t.Fatalf("Remove source skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "skills", "test-approach.md"), []byte("# Test Approach"), 0o644); err != nil {
		t.Fatalf("WriteFile renamed skill: %v", err)
	}

	if _, err := Run(opts); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	mustNotExist(t, installed, "renamed-away skill recorded by the prior install")
	mustExist(t, filepath.Join(targetDir, ".claude", "skills", "test-approach", "SKILL.md"), "renamed skill")
}

// --dry-run reports the prune without performing it.
func TestPruneStaleSkillDirsDryRun(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	createContent(t, sourceDir)

	skillsDest := filepath.Join(targetDir, ".agents", "skills")
	orphan := plantSkillDir(t, skillsDest, "command-prime")

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetCodex,
		Mode:      ModeProject,
		TargetDir: targetDir,
		DryRun:    true,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	mustExist(t, orphan, "dry run deleted a skill")
}

// Every nested-skills dest must converge. Rendering into a dest without
// pruning it is the defect this file exists for, so a target wired up with
// installSkillsNested and no prune fails here rather than years later on a
// user's disk.
func TestEveryNestedSkillsTargetPrunes(t *testing.T) {
	matches, err := filepath.Glob("target_*.go")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	for _, path := range matches {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile %s: %v", path, err)
		}
		body := string(src)
		if !strings.Contains(body, "installSkillsNested(") {
			continue
		}
		// Codex calls pruneStaleSkillDirs directly — it merges commands into
		// the same dest and needs the combined written set.
		if strings.Contains(body, "pruneNestedSkills(") || strings.Contains(body, "pruneStaleSkillDirs(") {
			continue
		}
		t.Errorf("%s renders nested skills but never prunes the dest — orphaned skill dirs will load forever (see prune.go)", path)
	}
}

// Global mode writes to ~/.agents/skills, shared with whatever else the
// user keeps there, and has no .hero/ workspace to read a manifest from.
// Only the owned-prefix proof applies.
func TestCodexGlobalPrunesOwnedPrefixOnly(t *testing.T) {
	sourceDir := t.TempDir()
	home := t.TempDir()
	createContent(t, sourceDir)
	t.Setenv("HOME", home)

	skillsDest := filepath.Join(home, ".agents", "skills")
	orphan := plantSkillDir(t, skillsDest, "command-prime")
	foreign := plantSkillDir(t, skillsDest, "my-personal-skill")

	opts := Options{
		SourceDir: sourceDir,
		Target:    TargetCodex,
		Mode:      ModeGlobal,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	mustNotExist(t, orphan, "orphaned command skill not pruned in global mode")
	mustExist(t, foreign, "user-authored global skill was deleted")
}
