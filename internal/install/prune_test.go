package install

import (
	"bytes"
	"io"
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

// ---------------------------------------------------------------------------
// File-manifest prune tests (pruneStaleFiles). Table-driven over all six
// targets where the criterion is per-target. See manifest-driven-prune spec.
// ---------------------------------------------------------------------------

// fileTgt describes one target's flat-agent dest shape for the file-prune
// tests. pruneActor is true for every target now that codex's .codex/agents
// cleanup is .md-scoped (removeLegacyDirMatching) rather than a wholesale
// wipe: a dropped codex .toml agent survives the dead-bytes cleanup and is
// removed provenance-safely by pruneStaleFiles, exactly like the flat files
// of every other target. (Before codex-agents-wholesale-wipe was fixed,
// codex was pruneActor:false because removeLegacyDir wiped the dir wholesale
// before the prune ever ran.)
//
// userFile is the basename of a user-authored file planted at the agent
// dest to prove the prune leaves it alone. It must NOT collide with the
// dead-bytes cleanup's blast radius at that dest — for codex the cleanup
// removes *.md, so its user file is a .toml (an inert user .md in
// .codex/agents is not protected and is not the load-bearing vector).
type fileTgt struct {
	name       string
	target     Target
	agentDest  func(stem string) string // TargetDir-relative dest for an agent stem
	userFile   string                   // basename of a planted user-authored file that must survive a prune
	pruneActor bool
}

var fileTgts = []fileTgt{
	{"claude", TargetClaude, func(s string) string { return filepath.Join(".claude", "agents", s+".md") }, "my-custom-agent.md", true},
	{"codex", TargetCodex, func(s string) string { return filepath.Join(".codex", "agents", s+".toml") }, "my-custom-agent.toml", true},
	{"copilot", TargetCopilot, func(s string) string { return filepath.Join(".github", "prompts", "agents", s+".prompt.md") }, "my-custom-agent.md", true},
	{"cursor", TargetCursor, func(s string) string { return filepath.Join(".cursor", "rules", "agents", s+".md") }, "my-custom-agent.md", true},
	{"opencode", TargetOpenCode, func(s string) string { return filepath.Join(".opencode", "agents", s+".md") }, "my-custom-agent.md", true},
	{"generic", TargetGeneric, func(s string) string { return filepath.Join(".ai", "agents", s+".md") }, "my-custom-agent.md", true},
}

// setupPruneWorkspace builds a source content tree and a target workspace
// with a .hero/ dir (where install-state.json — the manifest — lives).
func setupPruneWorkspace(t *testing.T) (sourceDir, targetDir string) {
	t.Helper()
	sourceDir = t.TempDir()
	targetDir = t.TempDir()
	createContent(t, sourceDir)
	if err := os.MkdirAll(filepath.Join(targetDir, ".hero"), 0o755); err != nil {
		t.Fatalf("MkdirAll .hero: %v", err)
	}
	return sourceDir, targetDir
}

func addSourceAgent(t *testing.T, sourceDir, stem string) {
	t.Helper()
	p := filepath.Join(sourceDir, "agents", stem+".md")
	if err := os.WriteFile(p, []byte("# "+stem+" Agent"), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", p, err)
	}
}

func removeSourceAgent(t *testing.T, sourceDir, stem string) {
	t.Helper()
	if err := os.Remove(filepath.Join(sourceDir, "agents", stem+".md")); err != nil {
		t.Fatalf("Remove source agent %s: %v", stem, err)
	}
}

func runInstall(t *testing.T, opts Options) *Result {
	t.Helper()
	r, err := Run(opts)
	if err != nil {
		t.Fatalf("Run(%s): %v", opts.Target, err)
	}
	return r
}

// manifestFiles returns the recorded Files manifest for a target.
func manifestFiles(t *testing.T, targetDir string, target Target) []string {
	t.Helper()
	st, err := ReadInstallState(targetDir)
	if err != nil {
		t.Fatalf("ReadInstallState: %v", err)
	}
	return st.Targets[string(target)].Files
}

func sliceHas(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

// captureStderr redirects os.Stderr around fn and returns what was written.
// The file prune reports via os.Stderr (matching pruneStaleSkillDirs).
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	_ = w.Close()
	os.Stderr = old
	out := <-done
	_ = r.Close()
	return out
}

// AC-1, AC-4, AC-5: a product-dropped agent is pruned for every target, the
// manifest records the actual rendered dest (.toml for codex, .prompt.md for
// copilot), and the removal is reported.
func TestPruneStaleFiles_RemovesDroppedAgent(t *testing.T) {
	for _, tc := range fileTgts {
		t.Run(tc.name, func(t *testing.T) {
			sourceDir, targetDir := setupPruneWorkspace(t)
			addSourceAgent(t, sourceDir, "foo")

			opts := Options{SourceDir: sourceDir, Target: tc.target, Mode: ModeProject, TargetDir: targetDir, Force: true}
			runInstall(t, opts)

			fooRel := filepath.ToSlash(tc.agentDest("foo"))
			fooAbs := filepath.Join(targetDir, tc.agentDest("foo"))
			mustExist(t, fooAbs, "first install rendered foo")

			// AC-5 / AC-8: manifest records the actual rendered dest path.
			if got := manifestFiles(t, targetDir, tc.target); !sliceHas(got, fooRel) {
				t.Fatalf("manifest missing %s; got %v", fooRel, got)
			}

			// Product drops foo.
			removeSourceAgent(t, sourceDir, "foo")
			out := captureStderr(t, func() { runInstall(t, opts) })

			mustNotExist(t, fooAbs, "dropped agent pruned")
			mustExist(t, filepath.Join(targetDir, tc.agentDest("engineer")), "kept agent survives")

			// AC-8: manifest no longer lists foo.
			if got := manifestFiles(t, targetDir, tc.target); sliceHas(got, fooRel) {
				t.Errorf("manifest still lists dropped %s; got %v", fooRel, got)
			}

			// AC-1: pruneStaleFiles reports the removal for every target,
			// codex included — its .toml agent now survives the .md-scoped
			// dead-bytes cleanup and is pruned here by manifest provenance.
			if tc.pruneActor {
				if !strings.Contains(out, "removed — dropped from product") {
					t.Errorf("missing prune report; stderr=%q", out)
				}
				if !strings.Contains(out, "foo") {
					t.Errorf("prune report did not name foo; stderr=%q", out)
				}
			}
		})
	}
}

// AC-2: a file the user authored (never recorded in the manifest) survives a
// prune byte-for-byte — codex included. Codex's user file is a .toml (its
// .codex/agents dead-bytes cleanup is .md-scoped, so a live .toml agent is
// exactly the load-bearing file that must survive); every other target uses a
// .md. See codex-agents-wholesale-wipe-destroys-user-files.
func TestPruneStaleFiles_NeverRemovesUserFile(t *testing.T) {
	for _, tc := range fileTgts {
		if !tc.pruneActor {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			sourceDir, targetDir := setupPruneWorkspace(t)
			addSourceAgent(t, sourceDir, "foo")

			opts := Options{SourceDir: sourceDir, Target: tc.target, Mode: ModeProject, TargetDir: targetDir, Force: true}
			runInstall(t, opts)

			userAbs := filepath.Join(targetDir, filepath.Dir(tc.agentDest("foo")), tc.userFile)
			want := []byte("# My Own Agent — do not touch\n")
			if err := os.WriteFile(userAbs, want, 0o644); err != nil {
				t.Fatalf("plant user file: %v", err)
			}

			removeSourceAgent(t, sourceDir, "foo")
			runInstall(t, opts)

			mustNotExist(t, filepath.Join(targetDir, tc.agentDest("foo")), "dropped agent pruned")
			got, err := os.ReadFile(userAbs)
			if err != nil {
				t.Fatalf("user file removed: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("user file mutated: got %q want %q", got, want)
			}
		})
	}
}

// AC-3, AC-6: a nil/empty prior manifest (fresh scheme, pre-Files TargetState)
// prunes nothing and records a fresh manifest for the next run.
func TestPruneStaleFiles_NoPriorManifestIsNoOp(t *testing.T) {
	sourceDir, targetDir := setupPruneWorkspace(t)
	addSourceAgent(t, sourceDir, "foo")

	opts := Options{SourceDir: sourceDir, Target: TargetClaude, Mode: ModeProject, TargetDir: targetDir, Force: true}
	runInstall(t, opts)

	// Simulate a pre-Files TargetState: SkillDirs recorded, Files absent.
	st, err := ReadInstallState(targetDir)
	if err != nil {
		t.Fatalf("ReadInstallState: %v", err)
	}
	ts := st.Targets["claude"]
	ts.Files = nil
	st.Targets["claude"] = ts
	if err := WriteInstallState(targetDir, st); err != nil {
		t.Fatalf("WriteInstallState: %v", err)
	}

	removeSourceAgent(t, sourceDir, "foo")
	runInstall(t, opts)

	// AC-6: nothing pruned off a nil prior manifest — foo lingers on disk.
	mustExist(t, filepath.Join(targetDir, ".claude", "agents", "foo.md"), "no prune off nil prior manifest")
	// AC-6: a fresh manifest is recorded for subsequent runs.
	got := manifestFiles(t, targetDir, TargetClaude)
	if len(got) == 0 {
		t.Fatal("fresh manifest not recorded after first run under new scheme")
	}
	if sliceHas(got, ".claude/agents/foo.md") {
		t.Errorf("fresh manifest should reflect the current (foo-less) render; got %v", got)
	}
}

// AC-13: install-state.json absent (fresh clone of a gitignored workspace)
// prunes nothing.
func TestPruneStaleFiles_FreshCloneNoState(t *testing.T) {
	sourceDir, targetDir := setupPruneWorkspace(t)
	addSourceAgent(t, sourceDir, "foo")

	opts := Options{SourceDir: sourceDir, Target: TargetClaude, Mode: ModeProject, TargetDir: targetDir, Force: true}
	runInstall(t, opts)

	if err := os.Remove(filepath.Join(targetDir, ".hero", "install-state.json")); err != nil {
		t.Fatalf("remove install-state.json: %v", err)
	}

	removeSourceAgent(t, sourceDir, "foo")
	runInstall(t, opts)

	mustExist(t, filepath.Join(targetDir, ".claude", "agents", "foo.md"), "no prune when state file absent")
	mustExist(t, filepath.Join(targetDir, ".hero", "install-state.json"), "fresh state recorded")
}

// AC-7: --dry-run reports what would be pruned, deletes nothing, and writes
// no manifest.
func TestPruneStaleFiles_DryRunDeletesNothing(t *testing.T) {
	sourceDir, targetDir := setupPruneWorkspace(t)
	addSourceAgent(t, sourceDir, "foo")

	opts := Options{SourceDir: sourceDir, Target: TargetClaude, Mode: ModeProject, TargetDir: targetDir, Force: true}
	runInstall(t, opts)

	removeSourceAgent(t, sourceDir, "foo")
	dryOpts := opts
	dryOpts.DryRun = true
	out := captureStderr(t, func() { runInstall(t, dryOpts) })

	mustExist(t, filepath.Join(targetDir, ".claude", "agents", "foo.md"), "dry-run must not delete")
	if !strings.Contains(out, "would remove — dropped from product") {
		t.Errorf("dry-run did not report the would-be prune; stderr=%q", out)
	}
	// Manifest unwritten under dry-run: it still lists foo.
	if got := manifestFiles(t, targetDir, TargetClaude); !sliceHas(got, ".claude/agents/foo.md") {
		t.Errorf("dry-run rewrote the manifest; got %v", got)
	}
}

// AC-11: Cursor's flat skill files are pruned when the product drops a skill.
func TestPruneStaleFiles_CursorFlatSkill(t *testing.T) {
	sourceDir, targetDir := setupPruneWorkspace(t)

	opts := Options{SourceDir: sourceDir, Target: TargetCursor, Mode: ModeProject, TargetDir: targetDir, Force: true}
	runInstall(t, opts)

	skillRel := filepath.ToSlash(filepath.Join(".cursor", "rules", "skills", "test-strategy.md"))
	skillAbs := filepath.Join(targetDir, ".cursor", "rules", "skills", "test-strategy.md")
	mustExist(t, skillAbs, "flat skill rendered")
	if got := manifestFiles(t, targetDir, TargetCursor); !sliceHas(got, skillRel) {
		t.Fatalf("manifest missing flat skill %s; got %v", skillRel, got)
	}

	// Product drops the skill.
	if err := os.Remove(filepath.Join(sourceDir, "skills", "test-strategy.md")); err != nil {
		t.Fatalf("remove source skill: %v", err)
	}
	out := captureStderr(t, func() { runInstall(t, opts) })

	mustNotExist(t, skillAbs, "dropped flat skill pruned")
	mustExist(t, filepath.Join(targetDir, ".cursor", "rules", "skills", "spec-format.md"), "kept flat skill survives")
	if !strings.Contains(out, "dropped from product") {
		t.Errorf("flat-skill prune not reported; stderr=%q", out)
	}
}

// AC-10: the file prune leaves nested skill dirs (governed by prune.go /
// SkillDirs) alone, and does not error when both prunes fire in one run.
func TestPruneStaleFiles_LeavesNestedSkillDirs(t *testing.T) {
	sourceDir, targetDir := setupPruneWorkspace(t)
	addSourceAgent(t, sourceDir, "foo")

	opts := Options{SourceDir: sourceDir, Target: TargetClaude, Mode: ModeProject, TargetDir: targetDir, Force: true}
	runInstall(t, opts)

	skillDir := filepath.Join(targetDir, ".claude", "skills", "test-strategy")
	mustExist(t, filepath.Join(skillDir, "SKILL.md"), "nested skill rendered")

	// Drop a flat agent (file prune) AND a nested skill (skill-dir prune) in
	// the same run.
	removeSourceAgent(t, sourceDir, "foo")
	if err := os.Remove(filepath.Join(sourceDir, "skills", "test-strategy.md")); err != nil {
		t.Fatalf("remove source skill: %v", err)
	}
	runInstall(t, opts) // must not error

	mustNotExist(t, filepath.Join(targetDir, ".claude", "agents", "foo.md"), "flat agent pruned by file prune")
	mustNotExist(t, skillDir, "nested skill pruned by skill-dir prune")
	mustExist(t, filepath.Join(targetDir, ".claude", "skills", "spec-format", "SKILL.md"), "kept nested skill survives")

	// The file manifest holds only flat files — no nested-skill dir paths.
	for _, f := range manifestFiles(t, targetDir, TargetClaude) {
		if strings.Contains(f, "/skills/") && strings.Contains(f, "/SKILL.md") {
			t.Errorf("file manifest wrongly recorded a nested skill path: %s", f)
		}
	}
}

// AC-12: an empty rendered set (broken/empty content source) prunes nothing —
// the guard against a catastrophic wipe.
func TestPruneStaleFiles_EmptyRenderIsNoOp(t *testing.T) {
	sourceDir, targetDir := setupPruneWorkspace(t)
	addSourceAgent(t, sourceDir, "foo")

	opts := Options{SourceDir: sourceDir, Target: TargetClaude, Mode: ModeProject, TargetDir: targetDir, Force: true}
	runInstall(t, opts)
	mustExist(t, filepath.Join(targetDir, ".claude", "agents", "foo.md"), "first install rendered foo")

	// Break the source: empty the content dirs so this run renders nothing.
	for _, kind := range []string{"agents", "commands", "skills"} {
		if err := os.RemoveAll(filepath.Join(sourceDir, kind)); err != nil {
			t.Fatalf("empty source %s: %v", kind, err)
		}
		if err := os.MkdirAll(filepath.Join(sourceDir, kind), 0o755); err != nil {
			t.Fatalf("recreate empty %s: %v", kind, err)
		}
	}

	runInstall(t, opts)

	// The prior manifest's flat files must NOT be deleted off an empty render.
	mustExist(t, filepath.Join(targetDir, ".claude", "agents", "foo.md"), "empty render must not prune")
	mustExist(t, filepath.Join(targetDir, ".claude", "agents", "engineer.md"), "empty render must not prune")
}

// AC-9: root instruction files are never recorded in a manifest and never
// pruned.
func TestPruneStaleFiles_NeverManifestsInstructionFiles(t *testing.T) {
	for _, tc := range []struct {
		name        string
		target      Target
		instruction string // TargetDir-relative
	}{
		{"claude", TargetClaude, "CLAUDE.md"},
		{"codex", TargetCodex, "AGENTS.md"},
		{"copilot", TargetCopilot, filepath.Join(".github", "copilot-instructions.md")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sourceDir, targetDir := setupPruneWorkspace(t)
			addSourceAgent(t, sourceDir, "foo")

			opts := Options{SourceDir: sourceDir, Target: tc.target, Mode: ModeProject, TargetDir: targetDir, Force: true}
			runInstall(t, opts)

			instrAbs := filepath.Join(targetDir, tc.instruction)
			mustExist(t, instrAbs, "instruction file written")

			instrSlash := filepath.ToSlash(tc.instruction)
			assertNotManifested := func(when string) {
				for _, f := range manifestFiles(t, targetDir, tc.target) {
					if f == instrSlash || f == "AGENTS.md" || f == "CLAUDE.md" {
						t.Errorf("%s: instruction file %q leaked into manifest", when, f)
					}
				}
			}
			assertNotManifested("after first install")

			// A second run exercises the prune; the instruction file survives.
			removeSourceAgent(t, sourceDir, "foo")
			runInstall(t, opts)
			mustExist(t, instrAbs, "instruction file survives the prune")
			assertNotManifested("after prune run")
		})
	}
}

// AC-8: the manifest is replaced each run, not merged — a second run's set
// wholly supersedes the first.
func TestRecordTargetInstall_FilesReplaced(t *testing.T) {
	sourceDir, targetDir := setupPruneWorkspace(t)
	addSourceAgent(t, sourceDir, "foo")

	opts := Options{SourceDir: sourceDir, Target: TargetClaude, Mode: ModeProject, TargetDir: targetDir, Force: true}
	runInstall(t, opts)
	if got := manifestFiles(t, targetDir, TargetClaude); !sliceHas(got, ".claude/agents/foo.md") {
		t.Fatalf("first manifest missing foo.md; got %v", got)
	}

	// Swap foo for bar in the product.
	removeSourceAgent(t, sourceDir, "foo")
	addSourceAgent(t, sourceDir, "bar")
	runInstall(t, opts)

	got := manifestFiles(t, targetDir, TargetClaude)
	if !sliceHas(got, ".claude/agents/bar.md") {
		t.Errorf("replaced manifest missing bar.md; got %v", got)
	}
	if sliceHas(got, ".claude/agents/foo.md") {
		t.Errorf("manifest merged instead of replaced — still lists foo.md; got %v", got)
	}
}
