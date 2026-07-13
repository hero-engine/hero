package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/managed"
)

// harness_native_test.go — coverage for the harness-native, target-aware
// instruction-file model (harness-native-install-target-aware-upgrade):
// each target writes only the root instruction file it natively reads,
// upgrade/backfill infers the prior target set, and orphan pruning is
// opt-in and non-destructive.

func mkHeroDir(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// managedBody returns the body between the Hero managed markers of the file
// at path, failing the test if no region is present.
func managedBody(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	region := managed.FindManagedRegion(string(data))
	if !region.Present {
		t.Fatalf("no managed region in %s", path)
	}
	return region.Body
}

// TestHarnessNative_PerTargetFileSet asserts the exact root instruction file
// each of the six targets writes: claude → CLAUDE.md only; every other
// target → AGENTS.md only.
func TestHarnessNative_PerTargetFileSet(t *testing.T) {
	cases := []struct {
		target Target
		want   string
		absent string
	}{
		{TargetClaude, "CLAUDE.md", "AGENTS.md"},
		{TargetCodex, "AGENTS.md", "CLAUDE.md"},
		{TargetOpenCode, "AGENTS.md", "CLAUDE.md"},
		{TargetCursor, "AGENTS.md", "CLAUDE.md"},
		{TargetCopilot, "AGENTS.md", "CLAUDE.md"},
		{TargetGeneric, "AGENTS.md", "CLAUDE.md"},
	}
	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			h := newInstallHarness(t)
			mkHeroDir(t, h.TargetDir)
			h.Run(tc.target, nil)
			h.mustBeRegularFile(tc.want)
			h.mustContain(tc.want, "hero:managed-start")
			h.mustNotExist(tc.absent)
		})
	}
}

// TestHarnessNative_DoctorRoutingGuidanceAllTargets asserts that the
// version/schema routing guidance (prefer the MCP surface; on a schema/
// version mismatch run `hero doctor`, not `hero upgrade`; don't confabulate
// a migration story) lands in every one of the six targets' native root
// instruction file. This is the enforcement for tripwire
// `harness-changes-cover-all-targets`: any target missing the guidance
// fails the test naming that target. Authored once in
// generateEngineeringAgentsMdBody (mirrored to domains/engineering/AGENTS.md),
// so a single edit must reach claude→CLAUDE.md and every other target→
// AGENTS.md.
func TestHarnessNative_DoctorRoutingGuidanceAllTargets(t *testing.T) {
	// Substrings encoding the two required behaviors: prefer MCP over a bare
	// shelled-out `hero`, and route schema/version confusion to `hero doctor`
	// (explicitly NOT `hero upgrade`).
	guidance := []string{
		"Prefer Hero's MCP tools over shelling out to a bare `hero`",
		"run `hero doctor`",
		"do NOT run `hero upgrade`",
	}
	for _, target := range []Target{
		TargetClaude, TargetCodex, TargetOpenCode,
		TargetCursor, TargetCopilot, TargetGeneric,
	} {
		t.Run(string(target), func(t *testing.T) {
			h := newInstallHarness(t)
			mkHeroDir(t, h.TargetDir)
			h.Run(target, nil)
			file := nativeInstructionFile(target)
			for _, want := range guidance {
				h.mustContain(file, want)
			}
		})
	}
}

// TestHarnessNative_MultiTargetIncludingClaude asserts that a multi-target
// install including claude produces BOTH files, each with a Hero-managed
// region.
func TestHarnessNative_MultiTargetIncludingClaude(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)

	h.Run(TargetClaude, nil)
	h.Run(TargetCodex, nil)

	h.mustBeRegularFile("CLAUDE.md")
	h.mustContain("CLAUDE.md", "hero:managed-start")
	h.mustBeRegularFile("AGENTS.md")
	h.mustContain("AGENTS.md", "hero:managed-start")
}

// TestHarnessNative_SameManagedBody asserts that for a claude + non-Codex
// target install, CLAUDE.md and AGENTS.md carry the same Hero-managed block
// body (shared defaultSections generator). Codex is excluded because it
// appends a Codex-specific workflow addendum to its body by design.
func TestHarnessNative_SameManagedBody(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)

	h.Run(TargetClaude, nil)
	h.Run(TargetCursor, nil)

	claudeBody := managedBody(t, filepath.Join(h.TargetDir, "CLAUDE.md"))
	agentsBody := managedBody(t, filepath.Join(h.TargetDir, "AGENTS.md"))
	if claudeBody != agentsBody {
		t.Errorf("managed body differs between CLAUDE.md and AGENTS.md for claude+cursor\n--- CLAUDE.md ---\n%s\n--- AGENTS.md ---\n%s", claudeBody, agentsBody)
	}
}

// TestHarnessNative_TwoNonClaudeTargetsShareAgentsMd asserts that two
// non-Claude targets both write the same root AGENTS.md, and the second
// write is a byte-for-byte no-op (idempotent multi-target write).
func TestHarnessNative_TwoNonClaudeTargetsShareAgentsMd(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)

	h.Run(TargetCursor, nil)
	h.mustBeRegularFile("AGENTS.md")
	first, err := os.ReadFile(filepath.Join(h.TargetDir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}

	h.Run(TargetGeneric, nil)
	second, err := os.ReadFile(filepath.Join(h.TargetDir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("second non-Claude target changed AGENTS.md (should be idempotent no-op)")
	}
	h.mustNotExist("CLAUDE.md")
}

// TestHarnessNative_Idempotent asserts a second identical install run leaves
// the instruction file byte-for-byte unchanged.
func TestHarnessNative_Idempotent(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)

	h.Run(TargetClaude, nil)
	first, err := os.ReadFile(filepath.Join(h.TargetDir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	h.Run(TargetClaude, nil)
	second, err := os.ReadFile(filepath.Join(h.TargetDir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("second install changed CLAUDE.md (should be idempotent)")
	}
}

// TestHarnessNative_InstallPersistsTargets asserts a multi-target install
// records every installed target in install-state.json `targets`.
func TestHarnessNative_InstallPersistsTargets(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)

	h.Run(TargetClaude, func(o *Options) { o.Version = "v0.0.0-test" })
	h.Run(TargetCodex, func(o *Options) { o.Version = "v0.0.0-test" })

	got := PreviouslyInstalledTargets(h.TargetDir)
	want := map[Target]bool{TargetClaude: true, TargetCodex: true}
	if len(got) != len(want) {
		t.Fatalf("PreviouslyInstalledTargets = %v, want claude+codex", got)
	}
	for _, tgt := range got {
		if !want[tgt] {
			t.Errorf("unexpected persisted target %q", tgt)
		}
	}
}

// --- backfill inference ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func managedOnlyFile(name string) string {
	return "# " + name + "\n\n" + managed.RenderManagedRegion("v1", "Hero body content")
}

func targetSet(ts []Target) map[Target]bool {
	m := map[Target]bool{}
	for _, t := range ts {
		m[t] = true
	}
	return m
}

// Both CLAUDE.md and AGENTS.md present, but only .claude/ content dir exists:
// infer {claude}. The stray AGENTS.md is NOT adopted as a non-Claude target.
func TestInferInstalledTargets_BothFilesOnlyClaudeContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".claude", "agents", "x.md"), "x")
	writeFile(t, filepath.Join(root, "CLAUDE.md"), managedOnlyFile("CLAUDE.md"))
	writeFile(t, filepath.Join(root, "AGENTS.md"), managedOnlyFile("AGENTS.md"))

	got := targetSet(InferInstalledTargets(root))
	if !got[TargetClaude] {
		t.Errorf("expected claude inferred, got %v", got)
	}
	if len(got) != 1 {
		t.Errorf("expected only {claude}; a phantom AGENTS.md must not add a non-Claude target; got %v", got)
	}
}

// Only a legacy Hero-managed CLAUDE.md stub, no content dirs: infer {claude}.
func TestInferInstalledTargets_StubOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CLAUDE.md"), managedOnlyFile("CLAUDE.md"))

	got := targetSet(InferInstalledTargets(root))
	if len(got) != 1 || !got[TargetClaude] {
		t.Errorf("expected {claude} from managed CLAUDE.md stub, got %v", got)
	}
}

// Neither file, no content dirs: empty set.
func TestInferInstalledTargets_Neither(t *testing.T) {
	root := t.TempDir()
	if got := InferInstalledTargets(root); len(got) != 0 {
		t.Errorf("expected empty inference, got %v", got)
	}
}

// AGENTS.md present, no CLAUDE.md, only a non-Claude content dir (.ai/):
// infer the non-Claude target; claude is NOT inferred.
func TestInferInstalledTargets_AgentsOnlyNonClaudeContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".ai", "agents", "x.md"), "x")
	writeFile(t, filepath.Join(root, "AGENTS.md"), managedOnlyFile("AGENTS.md"))

	got := targetSet(InferInstalledTargets(root))
	if got[TargetClaude] {
		t.Errorf("claude must not be inferred from a lone AGENTS.md, got %v", got)
	}
	if !got[TargetGeneric] {
		t.Errorf("expected generic (.ai content) inferred, got %v", got)
	}
}

// A non-managed (pure user) CLAUDE.md with no content dirs must NOT infer
// claude — it's a user file, not evidence of a Hero install.
func TestInferInstalledTargets_UserCLAUDEMdNotInferred(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "# My notes\n\nno hero markers here\n")

	if got := InferInstalledTargets(root); len(got) != 0 {
		t.Errorf("a user CLAUDE.md with no managed region must not infer claude, got %v", got)
	}
}

// --- orphan prune policy ---

// prune deletes an orphan whose entire content is Hero-managed.
func TestOrphanPolicy_PrunesManagedOnly(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	// A generic install produces a managed-only AGENTS.md (H1 + region).
	h.Run(TargetGeneric, nil)
	h.mustBeRegularFile("AGENTS.md")

	opts := Options{Target: TargetGeneric, Mode: ModeProject, TargetDir: h.TargetDir, SourceDir: h.SourceDir}
	action, err := ApplyOrphanInstructionFilePolicy(opts, "AGENTS.md", true)
	if err != nil {
		t.Fatalf("ApplyOrphanInstructionFilePolicy: %v", err)
	}
	if action != OrphanPruned {
		t.Errorf("expected OrphanPruned, got %q", action)
	}
	h.mustNotExist("AGENTS.md")
}

// prune preserves an orphan that has any user content outside the markers.
func TestOrphanPolicy_PreservesUserContentEvenWithPrune(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	content := "# AGENTS.md\n\n" + managed.RenderManagedRegion("v1", "body") + "\nUSER TAIL LINE\n"
	writeFile(t, filepath.Join(h.TargetDir, "AGENTS.md"), content)

	opts := Options{Target: TargetGeneric, Mode: ModeProject, TargetDir: h.TargetDir, SourceDir: h.SourceDir}
	action, err := ApplyOrphanInstructionFilePolicy(opts, "AGENTS.md", true)
	if err != nil {
		t.Fatalf("ApplyOrphanInstructionFilePolicy: %v", err)
	}
	if action == OrphanPruned {
		t.Fatalf("a file with user content must NOT be pruned")
	}
	h.mustBeRegularFile("AGENTS.md")
	h.mustContain("AGENTS.md", "USER TAIL LINE")
}

// without the prune flag, a managed-only orphan is never deleted (maintained).
func TestOrphanPolicy_NoPruneNeverDeletes(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	h.Run(TargetGeneric, nil)
	h.mustBeRegularFile("AGENTS.md")

	opts := Options{Target: TargetGeneric, Mode: ModeProject, TargetDir: h.TargetDir, SourceDir: h.SourceDir}
	action, err := ApplyOrphanInstructionFilePolicy(opts, "AGENTS.md", false)
	if err != nil {
		t.Fatalf("ApplyOrphanInstructionFilePolicy: %v", err)
	}
	if action != OrphanMaintained {
		t.Errorf("expected OrphanMaintained without prune, got %q", action)
	}
	h.mustBeRegularFile("AGENTS.md")
}

// an absent file is never created by the orphan policy.
func TestOrphanPolicy_AbsentFileNotCreated(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)

	opts := Options{Target: TargetClaude, Mode: ModeProject, TargetDir: h.TargetDir, SourceDir: h.SourceDir}
	action, err := ApplyOrphanInstructionFilePolicy(opts, "CLAUDE.md", true)
	if err != nil {
		t.Fatalf("ApplyOrphanInstructionFilePolicy: %v", err)
	}
	if action != OrphanAbsent {
		t.Errorf("expected OrphanAbsent, got %q", action)
	}
	h.mustNotExist("CLAUDE.md")
}
