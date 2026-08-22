package install

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	hero "github.com/hero-engine/hero"
)

// overlay_install_test.go — verifies install-core-domain-merge.
//
// Every install path passes an overlay FS (active domain on top of
// CoreFS) into Options.ContentFS. The renderer reads via opts.sourceFS()
// and is layer-agnostic; the assertion here is that whatever the
// renderer writes to disk includes both core- and domain-sourced files.

// runOverlayInstall runs install for the given target+domain with a
// fresh project dir, using the real embedded CoreFS overlaid by the
// resolved domain. Pre-inits a .hero/ workspace so AGENTS.md /
// CLAUDE.md installers don't early-return.
func runOverlayInstall(t *testing.T, target Target, domain string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	domainFS, err := hero.DomainFS(domain)
	if err != nil {
		t.Fatalf("DomainFS(%s): %v", domain, err)
	}
	merged := hero.OverlayFS(domainFS, hero.CoreFS())

	opts := Options{
		ContentFS: merged,
		Target:    target,
		Mode:      ModeProject,
		TargetDir: dir,
		Force:     true,
		Domain:    domain,
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("install (%s, %s): %v", target, domain, err)
	}
	return dir
}

// TestOverlay_AllTargetsIncludeCoreAndDomain asserts every harness target's
// install output contains at least one file sourced from core/ and at
// least one from the active domain. Uses the PM domain because it
// shares no agent filenames with core — session-primer.md landing in
// the install output proves the core layer reached the renderer.
//
// Maps to acceptance criterion "WHEN hero install --target <T> runs
// for each of T in {claude, cursor, opencode, codex, copilot, generic, grok}
// THE SYSTEM SHALL include core content in the install output".
func TestOverlay_AllTargetsIncludeCoreAndDomain(t *testing.T) {
	cases := []struct {
		target Target
		// coreAgentAt is a destination file path whose source is
		// core/agents/session-primer.md — present in core/ but not in
		// domains/pm/.
		coreAgentAt string
		// domainAgentAt is a destination file path whose source is a
		// PM-only agent (e.g. prd-author.md).
		domainAgentAt   string
		domainSkillAt   string
		domainCommandAt string
	}{
		{
			target:          TargetClaude,
			coreAgentAt:     ".claude/agents/session-primer.md",
			domainAgentAt:   ".claude/agents/prd-author.md",
			domainSkillAt:   ".claude/skills/pm-agent-doctrine/SKILL.md",
			domainCommandAt: ".claude/commands/prd.md",
		},
		{
			target:          TargetOpenCode,
			coreAgentAt:     ".opencode/agents/session-primer.md",
			domainAgentAt:   ".opencode/agents/prd-author.md",
			domainSkillAt:   ".opencode/skills/pm-agent-doctrine/SKILL.md",
			domainCommandAt: ".opencode/commands/prd.md",
		},
		{
			target:          TargetCursor,
			coreAgentAt:     ".cursor/rules/agents/session-primer.md",
			domainAgentAt:   ".cursor/rules/agents/prd-author.md",
			domainSkillAt:   ".cursor/rules/skills/pm-agent-doctrine.md",
			domainCommandAt: ".cursor/rules/commands/prd.md",
		},
		{
			target:          TargetGeneric,
			coreAgentAt:     ".ai/agents/session-primer.md",
			domainAgentAt:   ".ai/agents/prd-author.md",
			domainSkillAt:   ".ai/skills/pm-agent-doctrine/SKILL.md",
			domainCommandAt: ".ai/commands/prd.md",
		},
		{
			target:          TargetCodex,
			coreAgentAt:     ".codex/agents/session-primer.toml",
			domainAgentAt:   ".codex/agents/prd-author.toml",
			domainSkillAt:   ".agents/skills/pm-agent-doctrine/SKILL.md",
			domainCommandAt: ".agents/skills/command-prd/SKILL.md",
		},
		{
			target:          TargetCopilot,
			coreAgentAt:     ".github/prompts/agents/session-primer.prompt.md",
			domainAgentAt:   ".github/prompts/agents/prd-author.prompt.md",
			domainSkillAt:   ".github/skills/pm-agent-doctrine/SKILL.md",
			domainCommandAt: ".github/prompts/commands/prd.prompt.md",
		},
		{
			target:          TargetGrok,
			coreAgentAt:     ".grok/agents/session-primer.md",
			domainAgentAt:   ".grok/agents/prd-author.md",
			domainSkillAt:   ".grok/skills/pm-agent-doctrine/SKILL.md",
			domainCommandAt: ".grok/skills/command-prd/SKILL.md",
		},
	}

	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			dir := runOverlayInstall(t, tc.target, "pm")
			if _, err := os.Stat(filepath.Join(dir, tc.coreAgentAt)); err != nil {
				t.Errorf("expected core-sourced agent %s: %v", tc.coreAgentAt, err)
			}
			if _, err := os.Stat(filepath.Join(dir, tc.domainAgentAt)); err != nil {
				t.Errorf("expected domain-sourced agent %s: %v", tc.domainAgentAt, err)
			}
			if _, err := os.Stat(filepath.Join(dir, tc.domainSkillAt)); err != nil {
				t.Errorf("expected domain-sourced skill %s: %v", tc.domainSkillAt, err)
			}
			if _, err := os.Stat(filepath.Join(dir, tc.domainCommandAt)); err != nil {
				t.Errorf("expected domain-sourced command %s: %v", tc.domainCommandAt, err)
			}
		})
	}
}

// TestOverlay_DomainSwitchCoreSurvives verifies that, when a workspace
// is reinstalled under a different domain (the domain-switch flow's
// effective behavior), files whose only source is core stay present
// and bytes-identical. This is the "left untouched in content" part
// of the domain-switch acceptance criterion. Uses PM → sales because
// PM has agents that sales does not, and core-only agents (e.g.
// session-primer.md) are sourced from core under both domains —
// so their bytes must not change.
func TestOverlay_DomainSwitchCoreSurvives(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Install PM first.
	pmFS, err := hero.DomainFS("pm")
	if err != nil {
		t.Fatalf("DomainFS(pm): %v", err)
	}
	if _, err := Run(Options{
		ContentFS: hero.OverlayFS(pmFS, hero.CoreFS()),
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: dir,
		Force:     true,
		Domain:    "pm",
	}); err != nil {
		t.Fatalf("install pm: %v", err)
	}

	sessionPrimer := filepath.Join(dir, ".claude/agents/session-primer.md")
	beforeSwitch, err := os.ReadFile(sessionPrimer)
	if err != nil {
		t.Fatalf("read session-primer: %v", err)
	}

	// Switch to sales (which has no agents) — session-primer is now
	// pure-core under the active overlay. Bytes must be unchanged.
	salesFS, err := hero.DomainFS("sales")
	if err != nil {
		t.Fatalf("DomainFS(sales): %v", err)
	}
	if _, err := Run(Options{
		ContentFS: hero.OverlayFS(salesFS, hero.CoreFS()),
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: dir,
		Force:     true,
		Domain:    "sales",
	}); err != nil {
		t.Fatalf("install sales: %v", err)
	}

	afterSwitch, err := os.ReadFile(sessionPrimer)
	if err != nil {
		t.Fatalf("read session-primer after switch: %v", err)
	}
	if string(beforeSwitch) != string(afterSwitch) {
		t.Errorf("core-only file session-primer.md content drifted across pm→sales switch")
	}
}

// TestOverlay_DomainShadowsCoreOnConflict asserts the domain wins when
// both layers carry a file at the same relative path. Uses a synthetic
// "domain" FS (fstest.MapFS) over the real core, so the test owns both
// sides of the collision.
func TestOverlay_DomainShadowsCoreOnConflict(t *testing.T) {
	// Build a fake "domain" FS that re-declares one core file.
	// agents/session-primer.md exists in core; the domain ships a
	// different body at the same path. After install, the destination
	// must carry the domain bytes, not core's.
	const domainBody = "---\nname: session-primer\ndescription: domain-override\n---\n# OVERRIDE\n"
	fakeDomain := fstest.MapFS{
		"agents/session-primer.md": &fstest.MapFile{Data: []byte(domainBody)},
	}

	var overlay fs.FS = hero.OverlayFS(fakeDomain, hero.CoreFS())

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		ContentFS: overlay,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: dir,
		Force:     true,
		Domain:    "engineering",
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("install: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".claude", "agents", "session-primer.md"))
	if err != nil {
		t.Fatalf("read session-primer.md: %v", err)
	}
	if string(got) != domainBody {
		t.Errorf("expected domain body to win, got:\n%s", got)
	}
}

// TestOverlay_UpgradeTwice_NoFalsePositiveDrift covers the trust-map
// regression risk: install once, then re-run install — the second
// invocation must not report any skip ("preserved drift") on core
// files. The bytes-equal idempotency check in copyFileFromFS handles
// this so long as the source FS keeps returning the same bytes; this
// test guards against a future refactor that breaks that.
func TestOverlay_UpgradeTwice_NoFalsePositiveDrift(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	domainFS, err := hero.DomainFS("engineering")
	if err != nil {
		t.Fatalf("DomainFS: %v", err)
	}
	merged := hero.OverlayFS(domainFS, hero.CoreFS())

	// First install: writes core + domain.
	opts := Options{
		ContentFS: merged,
		Target:    TargetClaude,
		Mode:      ModeProject,
		TargetDir: dir,
		Force:     true,
		Domain:    "engineering",
		Version:   "test-v1",
	}
	if _, err := Run(opts); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Second run (no --force): identical bytes; idempotent no-op path
	// in copyFileFromFS must NOT mark any core file as skipped/drifted.
	opts.Force = false
	opts.Version = "test-v2"
	res, err := Run(opts)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	for _, skipped := range res.Skipped {
		// Core files should pass the bytes-equal idempotent check and
		// produce zero skips. Any skip referencing a core path is the
		// trust-map regression this test guards against.
		t.Errorf("upgrade re-install produced skip: %s", skipped)
	}
}
