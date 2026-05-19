package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// GitHub Copilot loader paths (source-verified against
// github.com/microsoft/vscode-copilot-chat):
//
//	Project skills:  .github/skills/<name>/SKILL.md   (also reads .claude/skills/)
//	Personal skills: $HOME/.copilot/skills/<name>/SKILL.md
//	  Source: vscode-copilot-chat/src/platform/customInstructions/common/promptTypes.ts:19-44
//	  Gated by setting `chat.useAgentSkills`.
//
//	Single instructions: .github/copilot-instructions.md
//	  Source: customInstructionsService.ts:307-322 + promptTypes.ts:24
//	  Gated by setting `chat.codeGeneration.useInstructionFiles` (default on).
//
//	Path-scoped instructions: .github/instructions/<name>.instructions.md
//	  Source: configurable via `chat.instructionsFilesLocations` (default `.github/instructions`)
//	  Optional `applyTo:` glob frontmatter governs auto-attach.
//
//	Prompt files:    .github/prompts/<name>.prompt.md  (default; configurable via `chat.promptFilesLocations`)
//	  Source: VS Code core; surfaced as user-invokable prompts.
//
//	NO LOADER for .github/copilot/agents|commands|skills/ at any
//	scope. The .github/copilot/ subdir tree is not ingested by
//	Copilot Chat — proven by zero source references.
//
// Hero implementation:
//   - Symlinks (or renders fallback) canonical skills into
//     .github/skills/<name>/SKILL.md (same SKILL.md format).
//   - Renders canonical agents and commands as Copilot .prompt.md
//     files at .github/prompts/agents/<name>.prompt.md and
//     .github/prompts/commands/<name>.prompt.md (subdir-namespaced
//     to avoid collisions). VS Code's prompt-file discovery scans
//     the configured root recursively.
//   - Writes .github/copilot-instructions.md (already correct).
//   - Cleans up dead bytes from prior installs at .github/copilot/{agents,commands,skills}/.
//   - Project-only target — Copilot has no per-user filesystem global
//     for these surfaces (settings live in VS Code config UI).

func runCopilot(opts Options) (*Result, error) {
	if opts.Mode != ModeProject || opts.TargetDir == "" {
		return nil, fmt.Errorf("copilot target only supports project mode")
	}

	info, statErr := os.Stat(opts.TargetDir)
	if statErr != nil || !info.IsDir() {
		return nil, fmt.Errorf("target directory does not exist: %s", opts.TargetDir)
	}

	result := &Result{}

	// Cleanup dead bytes from prior installs. .github/copilot/{agents,commands,skills}/
	// were never read by Copilot Chat — every file there is dead.
	legacyBase := filepath.Join(opts.TargetDir, ".github", "copilot")
	for _, kindDir := range []string{"agents", "commands", "skills"} {
		legacyPath := filepath.Join(legacyBase, kindDir)
		if err := removeIfHeroAuthored(opts, result, legacyPath, kindDir, false); err != nil {
			return nil, fmt.Errorf("cleanup %s: %w", legacyPath, err)
		}
	}
	// Try to remove .github/copilot/ if empty.
	_ = os.Remove(legacyBase)

	// Skills → .github/skills/<name>/SKILL.md (rendered from embedded).
	skillsDest := filepath.Join(opts.TargetDir, ".github", "skills")
	if err := installSkillsNested(opts, result, skillsDest); err != nil {
		return nil, fmt.Errorf("installing skills to %s: %w", skillsDest, err)
	}

	// Agents → .github/prompts/agents/<name>.prompt.md (rendered).
	promptsBase := filepath.Join(opts.TargetDir, ".github", "prompts")
	if err := renderToFile(opts, result, "agents", filepath.Join(promptsBase, "agents"), renderCopilotPromptFile); err != nil {
		return nil, fmt.Errorf("render agents to copilot prompts: %w", err)
	}

	// Commands → .github/prompts/commands/<name>.prompt.md (rendered).
	if err := renderToFile(opts, result, "commands", filepath.Join(promptsBase, "commands"), renderCopilotPromptFile); err != nil {
		return nil, fmt.Errorf("render commands to copilot prompts: %w", err)
	}

	// Single instructions file at .github/copilot-instructions.md.
	instructionsPath := filepath.Join(opts.TargetDir, ".github", "copilot-instructions.md")
	if err := installInstructionsMd(opts, result, instructionsPath, "copilot"); err != nil {
		return nil, fmt.Errorf("installing copilot-instructions.md: %w", err)
	}

	return result, nil
}
