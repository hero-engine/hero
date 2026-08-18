package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttentionLifecycleGuidanceReachesAllHarnessNativeSurfaces(t *testing.T) {
	cases := []struct {
		name   string
		target Target
		root   string
		skill  string
		resume string
	}{
		{"opencode", TargetOpenCode, "AGENTS.md", ".opencode/skills/attention-lifecycle-awareness/SKILL.md", ".opencode/commands/resume.md"},
		{"cursor", TargetCursor, "AGENTS.md", ".cursor/rules/skills/attention-lifecycle-awareness.md", ".cursor/rules/commands/resume.md"},
		{"claude", TargetClaude, "CLAUDE.md", ".claude/skills/attention-lifecycle-awareness/SKILL.md", ".claude/commands/resume.md"},
		{"copilot", TargetCopilot, "AGENTS.md", ".github/skills/attention-lifecycle-awareness/SKILL.md", ".github/prompts/commands/resume.prompt.md"},
		{"codex", TargetCodex, "AGENTS.md", ".agents/skills/attention-lifecycle-awareness/SKILL.md", ".agents/skills/command-resume/SKILL.md"},
		{"generic", TargetGeneric, "AGENTS.md", ".ai/skills/attention-lifecycle-awareness/SKILL.md", ".ai/commands/resume.md"},
		{"grok", TargetGrok, "AGENTS.md", ".grok/skills/attention-lifecycle-awareness/SKILL.md", ".grok/skills/command-resume/SKILL.md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := runOverlayInstall(t, tc.target, "engineering")
			for path, markers := range map[string][]string{
				tc.root: {
					"## Attention Lifecycle Awareness",
					"`hero_attention_snapshot` exactly once with `limit: 8`",
					"Never replay a write merely to confirm it",
					"never as empty",
					"Never append a generic inbox dump",
				},
				tc.skill: {
					"name: attention-lifecycle-awareness",
					"invoke it exactly once with `limit: 8`",
					"Do not poll solely to construct a recap",
					"Never translate unavailable or stale into empty",
				},
				tc.resume: {
					"`hero_attention_snapshot` is advertised",
					"call it exactly",
					"once with `limit: 8`",
					"do not call `hero_mail_show`",
				},
			} {
				content, err := os.ReadFile(filepath.Join(dir, path))
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				for _, marker := range markers {
					if !strings.Contains(string(content), marker) {
						t.Errorf("%s missing %q", path, marker)
					}
				}
			}
		})
	}
}

func TestAttentionLifecycleRootGuidanceIsEngineeringOnly(t *testing.T) {
	dir := runOverlayInstall(t, TargetClaude, "pm")
	content, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "## Attention Lifecycle Awareness") {
		t.Fatal("engineering Attention lifecycle guidance leaked into pm root instructions")
	}
}
