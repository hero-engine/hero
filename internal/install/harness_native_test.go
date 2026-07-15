package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	hero "github.com/hero-engine/hero"
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
// a migration story) lands in every domain × every one of the six targets'
// native root instruction file. The guidance is authored once as the shared
// domain-agnostic operationalGuidanceSection wired into defaultSections, so
// it must reach every pack (engineering, pm, sales, chat) and every target
// (claude→CLAUDE.md, all others→AGENTS.md). This is the enforcement for
// tripwire `harness-changes-cover-all-targets`: dropping the shared section
// fails the test naming the domain/target that lost the guidance.
//
// It also proves the guidance is sourced from the shared section, not any
// pack body: pm/sales/chat bodies never carried it, and the engineering
// body no longer does (see TestEngineeringBodyOmitsOperationalGuidance).
func TestHarnessNative_DoctorRoutingGuidanceAllTargets(t *testing.T) {
	// Substrings encoding the two required behaviors: prefer MCP over a bare
	// shelled-out `hero`, and route schema/version confusion to `hero doctor`
	// (explicitly NOT `hero upgrade`).
	guidance := []string{
		"Prefer Hero's MCP tools over shelling out to a bare `hero`",
		"run `hero doctor`",
		"do NOT run `hero upgrade`",
		// The shared section's own H2 heading — proves the source is the
		// section contributor, not a paragraph buried in a pack body.
		"## Hero Binary & MCP Surface",
		"Tracker connections use stable IDs under `integrations.connections`",
		"hero sync import",
		"--token-stdin",
	}
	targets := []Target{
		TargetClaude, TargetCodex, TargetOpenCode,
		TargetCursor, TargetCopilot, TargetGeneric,
	}
	// Per-domain Options mutator supplying that domain's real pack body.
	// engineering/pm/sales are installable, so they go through the real
	// embedded DomainFS pipeline. chat is a client-embedded, non-installable
	// pack (no DomainFS case — see content.go), so its real on-disk body is
	// fed through the AgentsMdBodyOverride seam. Both paths prove the shared
	// section renders on top of a pack body that never carried the guidance.
	repoRoot, err := repoRootFromHere()
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	domainOpts := func(t *testing.T, domain string) func(*Options) {
		if domain == "chat" {
			body, err := os.ReadFile(filepath.Join(repoRoot, "domains", "chat", "AGENTS.md"))
			if err != nil {
				t.Fatalf("read chat pack AGENTS.md: %v", err)
			}
			return func(o *Options) {
				o.Domain = domain
				o.AgentsMdBodyOverride = body
			}
		}
		domainFS, err := hero.DomainFS(domain)
		if err != nil {
			t.Fatalf("DomainFS(%s): %v", domain, err)
		}
		contentFS := hero.OverlayFS(domainFS, hero.CoreFS())
		return func(o *Options) {
			o.Domain = domain
			o.ContentFS = contentFS
		}
	}
	for _, domain := range []string{"engineering", "pm", "sales", "chat"} {
		mutate := domainOpts(t, domain)
		for _, target := range targets {
			t.Run(domain+"/"+string(target), func(t *testing.T) {
				h := newInstallHarness(t)
				mkHeroDir(t, h.TargetDir)
				h.Run(target, mutate)
				file := nativeInstructionFile(target)
				for _, want := range guidance {
					h.mustContain(file, want)
				}
			})
		}
	}
}

// TestEngineeringBodyOmitsOperationalGuidance guards against a lingering
// duplicate source: the shared operationalGuidanceSection is now the single
// author of the doctor/MCP routing guidance, so the engineering pack body
// (both the Go fallback and the mirrored domains/engineering/AGENTS.md) must
// NOT still carry a private copy. A stray copy would double-render the
// guidance in an engineering install.
func TestEngineeringBodyOmitsOperationalGuidance(t *testing.T) {
	const marker = "Prefer Hero's MCP tools over shelling out to a bare `hero`"

	goBody := generateEngineeringAgentsMdBody(resolveContentPathsForBody(Options{}))
	if strings.Contains(goBody, marker) {
		t.Errorf("generateEngineeringAgentsMdBody still contains the operational guidance paragraph; it must be sourced only from the shared section")
	}

	repoRoot, err := repoRootFromHere()
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	packBytes, err := os.ReadFile(filepath.Join(repoRoot, "domains", "engineering", "AGENTS.md"))
	if err != nil {
		t.Fatalf("read engineering pack AGENTS.md: %v", err)
	}
	if strings.Contains(string(packBytes), marker) {
		t.Errorf("domains/engineering/AGENTS.md still contains the operational guidance paragraph; regenerate with HERO_REGEN_PACK_AGENTS=1")
	}
}

// TestHarnessNative_OperationalGuidanceFallbackDomain proves that a domain
// with no own AGENTS.md — a future/unknown pack that falls through to the
// engineering Go fallback body — still receives the shared operational
// guidance. Because the engineering fallback body no longer carries the
// paragraph, its presence here can only come from the shared section.
func TestHarnessNative_OperationalGuidanceFallbackDomain(t *testing.T) {
	h := newInstallHarness(t)
	mkHeroDir(t, h.TargetDir)
	// No ContentFS with an AGENTS.md and a non-engineering domain forces
	// loadPackAgentsMdBody down the engineering Go fallback path.
	h.Run(TargetClaude, func(o *Options) {
		o.Domain = "widgets"
	})
	h.mustContain("CLAUDE.md", "## Hero Binary & MCP Surface")
	h.mustContain("CLAUDE.md", "run `hero doctor`")
	h.mustContain("CLAUDE.md", "do NOT run `hero upgrade`")
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
