package install

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// content_test.go — coverage for the shared content-install primitives
// across every target. Regression guards for two bugs in the same family:
//
//   - Cursor shipped zero skills: target_cursor.go used installFlat for
//     skills, which skips directories — and skills are now directories
//     containing SKILL.md. Fixed by installSkillsFlat.
//   - installFlat had no README exclusion, so domain content READMEs
//     (pm/sales ship agents/README.md etc.) installed as pseudo-agents.

// TestAllTargets_ShipSkills asserts every install target materializes a
// nonzero number of skill files from the canonical nested source layout,
// at the layout its harness actually loads.
func TestAllTargets_ShipSkills(t *testing.T) {
	cases := []struct {
		target    Target
		skillsDir string // dest dir holding installed skills
		specSkill string // where the seeded spec-format skill must land
	}{
		{TargetClaude, ".claude/skills", ".claude/skills/spec-format/SKILL.md"},
		{TargetOpenCode, ".opencode/skills", ".opencode/skills/spec-format/SKILL.md"},
		{TargetCursor, ".cursor/rules/skills", ".cursor/rules/skills/spec-format.md"},
		{TargetCodex, ".agents/skills", ".agents/skills/spec-format/SKILL.md"},
		{TargetCopilot, ".github/skills", ".github/skills/spec-format/SKILL.md"},
		{TargetGeneric, ".ai/skills", ".ai/skills/spec-format/SKILL.md"},
		{TargetGrok, ".grok/skills", ".grok/skills/spec-format/SKILL.md"},
	}

	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			h := newInstallHarness(t)
			if err := os.MkdirAll(filepath.Join(h.TargetDir, ".hero"), 0o755); err != nil {
				t.Fatal(err)
			}
			h.Run(tc.target, nil)

			h.mustBeRegularFile(tc.specSkill)

			if n := countFilesUnder(t, filepath.Join(h.TargetDir, tc.skillsDir)); n == 0 {
				t.Errorf("target %s shipped zero skill files under %s", tc.target, tc.skillsDir)
			}
		})
	}
}

func TestAllTargetsInstallIdenticalDeferredWorkConsentGuidance(t *testing.T) {
	const body = `---
name: deferred-work-suggestions
description: Consent-bound deferred work.
---
Suggestions are advisory output, not Focus and not a personal commitment.
Never turn unfinished required steps, acceptance criteria, Completion Ledger items, or harness todos into suggestions.
Invoke hero_focus_suggest once, then continue and finish the current task.
Never create Focus directly. Only the user may accept Today, Later, or Do Next.
`
	cases := []struct {
		target Target
		path   string
	}{
		{TargetOpenCode, ".opencode/skills/deferred-work-suggestions/SKILL.md"},
		{TargetCursor, ".cursor/rules/skills/deferred-work-suggestions.md"},
		{TargetClaude, ".claude/skills/deferred-work-suggestions/SKILL.md"},
		{TargetCopilot, ".github/skills/deferred-work-suggestions/SKILL.md"},
		{TargetCodex, ".agents/skills/deferred-work-suggestions/SKILL.md"},
		{TargetGeneric, ".ai/skills/deferred-work-suggestions/SKILL.md"},
		{TargetGrok, ".grok/skills/deferred-work-suggestions/SKILL.md"},
	}
	for _, tc := range cases {
		t.Run(string(tc.target), func(t *testing.T) {
			h := newInstallHarness(t)
			source := filepath.Join(h.SourceDir, "skills", "deferred-work-suggestions", "SKILL.md")
			if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(source, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			h.Run(tc.target, nil)
			h.mustBeRegularFile(tc.path)
			installed, err := os.ReadFile(filepath.Join(h.TargetDir, tc.path))
			if err != nil {
				t.Fatal(err)
			}
			if string(installed) != body {
				t.Fatalf("%s guidance drifted:\n%s", tc.target, installed)
			}
			for _, phrase := range []string{"advisory output, not Focus", "unfinished required steps", "hero_focus_suggest once", "Only the user may accept"} {
				if !strings.Contains(string(installed), phrase) {
					t.Errorf("%s missing semantic boundary %q", tc.target, phrase)
				}
			}
		})
	}
}

func TestAllTargetsInstallMailSourceDedupGuidance(t *testing.T) {
	sourceBody, err := os.ReadFile(filepath.Join("..", "..", "core", "skills", "auto-knowledge-capture", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	const marker = "For intent-bearing content whose typed source is `mail:<message-id>`"
	if !strings.Contains(string(sourceBody), marker) {
		t.Fatalf("canonical skill missing marker %q", marker)
	}
	cases := []struct {
		target Target
		path   string
	}{
		{TargetOpenCode, ".opencode/skills/auto-knowledge-capture/SKILL.md"},
		{TargetCursor, ".cursor/rules/skills/auto-knowledge-capture.md"},
		{TargetClaude, ".claude/skills/auto-knowledge-capture/SKILL.md"},
		{TargetCopilot, ".github/skills/auto-knowledge-capture/SKILL.md"},
		{TargetCodex, ".agents/skills/auto-knowledge-capture/SKILL.md"},
		{TargetGeneric, ".ai/skills/auto-knowledge-capture/SKILL.md"},
		{TargetGrok, ".grok/skills/auto-knowledge-capture/SKILL.md"},
	}
	for _, testCase := range cases {
		t.Run(string(testCase.target), func(t *testing.T) {
			h := newInstallHarness(t)
			source := filepath.Join(h.SourceDir, "skills", "auto-knowledge-capture", "SKILL.md")
			if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(source, sourceBody, 0o644); err != nil {
				t.Fatal(err)
			}
			h.Run(testCase.target, nil)
			installed, err := os.ReadFile(filepath.Join(h.TargetDir, testCase.path))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(installed), marker) {
				t.Fatalf("%s did not receive canonical Mail dedup guidance", testCase.target)
			}
		})
	}
}

// AC-10
func TestAllTargetsInstallAsyncPeeringGuidance(t *testing.T) {
	sourceBody, err := os.ReadFile(filepath.Join("..", "..", "domains", "engineering", "skills", "cross-repo-peering", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	markers := []string{
		"Peering is durable Project Mail, not model execution",
		"hero handoff receive <message-id>",
		"receiver-owned artifact",
		"peering.subagent",
	}
	cases := []struct {
		target Target
		path   string
	}{
		{TargetOpenCode, ".opencode/skills/cross-repo-peering/SKILL.md"},
		{TargetCursor, ".cursor/rules/skills/cross-repo-peering.md"},
		{TargetClaude, ".claude/skills/cross-repo-peering/SKILL.md"},
		{TargetCopilot, ".github/skills/cross-repo-peering/SKILL.md"},
		{TargetCodex, ".agents/skills/cross-repo-peering/SKILL.md"},
		{TargetGeneric, ".ai/skills/cross-repo-peering/SKILL.md"},
		{TargetGrok, ".grok/skills/cross-repo-peering/SKILL.md"},
	}
	for _, testCase := range cases {
		t.Run(string(testCase.target), func(t *testing.T) {
			h := newInstallHarness(t)
			source := filepath.Join(h.SourceDir, "skills", "cross-repo-peering", "SKILL.md")
			if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(source, sourceBody, 0o644); err != nil {
				t.Fatal(err)
			}
			h.Run(testCase.target, nil)
			installed, err := os.ReadFile(filepath.Join(h.TargetDir, testCase.path))
			if err != nil {
				t.Fatal(err)
			}
			for _, marker := range markers {
				if !strings.Contains(string(installed), marker) {
					t.Errorf("%s missing async peering guidance %q", testCase.target, marker)
				}
			}
		})
	}
}

// TestInstall_ExcludesContentReadmes asserts that README.md files in the
// source content tree (documentation for humans browsing agents/,
// commands/, skills/ — the pm and sales domains ship them) are never
// installed as content.
func TestInstall_ExcludesContentReadmes(t *testing.T) {
	seedReadmes := func(h *installHarness) {
		for _, rel := range []string{"agents/README.md", "commands/README.md", "skills/README.md"} {
			full := filepath.Join(h.SourceDir, rel)
			if err := os.WriteFile(full, []byte("# Directory readme\n"), 0o644); err != nil {
				h.t.Fatal(err)
			}
		}
	}

	t.Run("claude", func(t *testing.T) {
		h := newInstallHarness(t)
		seedReadmes(h)
		h.Run(TargetClaude, nil)

		h.mustNotExist(".claude/agents/README.md")
		h.mustNotExist(".claude/commands/README.md")
		h.mustNotExist(".claude/skills/README.md")
		h.mustNotExist(".claude/skills/README/SKILL.md")
	})

	t.Run("cursor", func(t *testing.T) {
		h := newInstallHarness(t)
		seedReadmes(h)
		h.Run(TargetCursor, nil)

		h.mustNotExist(".cursor/rules/agents/README.md")
		h.mustNotExist(".cursor/rules/commands/README.md")
		h.mustNotExist(".cursor/rules/skills/README.md")
	})
}

// countFilesUnder returns the number of regular files anywhere under dir.
// A missing dir counts as zero.
func countFilesUnder(t *testing.T, dir string) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			count++
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walking %s: %v", dir, err)
	}
	return count
}
