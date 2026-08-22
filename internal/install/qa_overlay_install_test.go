package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQAOverlay_AllTargetsRenderNativeCoreAndQASurfaces(t *testing.T) {
	cases := []struct {
		name      string
		target    Target
		coreAgent string
		qaAgent   string
		qaSkill   string
		qaCommand string
		engAgent  string
	}{
		{
			name: "claude", target: TargetClaude,
			coreAgent: ".claude/agents/session-primer.md", qaAgent: ".claude/agents/qa-delivery-lead.md",
			qaSkill: ".claude/skills/risk-based-testing/SKILL.md", qaCommand: ".claude/commands/author-cases.md",
			engAgent: ".claude/agents/feature-delivery-lead.md",
		},
		{
			name: "opencode", target: TargetOpenCode,
			coreAgent: ".opencode/agents/session-primer.md", qaAgent: ".opencode/agents/qa-delivery-lead.md",
			qaSkill: ".opencode/skills/risk-based-testing/SKILL.md", qaCommand: ".opencode/commands/author-cases.md",
			engAgent: ".opencode/agents/feature-delivery-lead.md",
		},
		{
			name: "cursor", target: TargetCursor,
			coreAgent: ".cursor/rules/agents/session-primer.md", qaAgent: ".cursor/rules/agents/qa-delivery-lead.md",
			qaSkill: ".cursor/rules/skills/risk-based-testing.md", qaCommand: ".cursor/rules/commands/author-cases.md",
			engAgent: ".cursor/rules/agents/feature-delivery-lead.md",
		},
		{
			name: "generic", target: TargetGeneric,
			coreAgent: ".ai/agents/session-primer.md", qaAgent: ".ai/agents/qa-delivery-lead.md",
			qaSkill: ".ai/skills/risk-based-testing/SKILL.md", qaCommand: ".ai/commands/author-cases.md",
			engAgent: ".ai/agents/feature-delivery-lead.md",
		},
		{
			name: "codex", target: TargetCodex,
			coreAgent: ".codex/agents/session-primer.toml", qaAgent: ".codex/agents/qa-delivery-lead.toml",
			qaSkill: ".agents/skills/risk-based-testing/SKILL.md", qaCommand: ".agents/skills/command-author-cases/SKILL.md",
			engAgent: ".codex/agents/feature-delivery-lead.toml",
		},
		{
			name: "copilot", target: TargetCopilot,
			coreAgent: ".github/prompts/agents/session-primer.prompt.md", qaAgent: ".github/prompts/agents/qa-delivery-lead.prompt.md",
			qaSkill: ".github/skills/risk-based-testing/SKILL.md", qaCommand: ".github/prompts/commands/author-cases.prompt.md",
			engAgent: ".github/prompts/agents/feature-delivery-lead.prompt.md",
		},
		{
			name: "grok", target: TargetGrok,
			coreAgent: ".grok/agents/session-primer.md", qaAgent: ".grok/agents/qa-delivery-lead.md",
			qaSkill: ".grok/skills/risk-based-testing/SKILL.md", qaCommand: ".grok/skills/command-author-cases/SKILL.md",
			engAgent: ".grok/agents/feature-delivery-lead.md",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := runOverlayInstall(t, tc.target, "qa")
			for label, relative := range map[string]string{
				"core agent": tc.coreAgent,
				"QA agent":   tc.qaAgent,
				"QA skill":   tc.qaSkill,
				"QA command": tc.qaCommand,
			} {
				if info, err := os.Stat(filepath.Join(dir, relative)); err != nil || info.Size() == 0 {
					t.Errorf("%s missing or empty at %s: %v", label, relative, err)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, tc.engAgent)); !os.IsNotExist(err) {
				t.Errorf("engineering-only agent leaked into primary QA install at %s", tc.engAgent)
			}
		})
	}
}
