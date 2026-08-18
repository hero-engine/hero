package install

// ContentKind names a category of canonical content the installer
// renders into a target. Each target installs zero or more kinds and
// must declare a HarnessContract for every kind it actually loads
// (vs. just installs as dead bytes).
type ContentKind string

const (
	KindAgents   ContentKind = "agents"
	KindCommands ContentKind = "commands"
	KindSkills   ContentKind = "skills"
)

// ContractFormat names the file format the destination is parsed in.
// Different formats need different validators; today we support YAML
// frontmatter (markdown files) and TOML (Codex subagents).
type ContractFormat string

const (
	FormatYAMLFrontmatter ContractFormat = "yaml-frontmatter"
	FormatTOML            ContractFormat = "toml"
	FormatFreeform        ContractFormat = "freeform" // no validation; file just needs to exist
)

// HarnessContract describes the shape an installed file must take for
// the consuming harness to register and use it. Each (Target,
// ContentKind) cell declares its contract.
//
// Fields are minimal — add a field only when at least one declared
// contract uses it.
type HarnessContract struct {
	// Format is the destination file's format. Determines which
	// frontmatter parser the contract validator uses. Defaults to
	// FormatYAMLFrontmatter (markdown + YAML frontmatter).
	Format ContractFormat

	// RequiredFields lists keys (frontmatter keys for YAML, top-level
	// keys for TOML) that must be present and non-empty.
	// Empty slice means no required-key checks.
	RequiredFields []string

	// FilenameRequired, when set, must match the destination file's
	// basename. For nested kinds (skills land at <name>/SKILL.md),
	// match against the trailing path segment ("SKILL.md"). Empty
	// string disables the check.
	FilenameRequired string

	// FilenameSuffix, when set, requires the filename to end with the
	// given suffix (e.g. ".toml" for Codex agents, ".prompt.md" for
	// Copilot prompts). Mutually exclusive with FilenameRequired.
	FilenameSuffix string

	// ContentValidator, when set, runs against the file's full bytes
	// after format/field checks pass. Reserved for whole-file
	// validators (e.g. a future opencode.json schema check). Nil
	// disables.
	ContentValidator func([]byte) error
}

// targetContracts is the per-target registry. Lookups go through
// ContractsFor — never index directly so the meta-test can detect
// missing cells.
//
// Cells where the consuming harness has NO loader for a content kind
// are intentionally absent (not present in the inner map). The
// meta-test treats absence as "documented gap" when the legacy install
// no longer writes there. See cleanup paths in each target_*.go.
var targetContracts = map[Target]map[ContentKind]HarnessContract{
	// Claude Code reads markdown + YAML frontmatter for all three kinds.
	// Source: https://code.claude.com/docs/en/sub-agents (agents),
	//         https://code.claude.com/docs/en/skills (commands + skills)
	TargetClaude: {
		KindAgents: {
			Format:         FormatYAMLFrontmatter,
			RequiredFields: []string{"name", "description"},
		},
		KindCommands: {
			Format:         FormatYAMLFrontmatter,
			RequiredFields: []string{"description"},
		},
		KindSkills: {
			Format:           FormatYAMLFrontmatter,
			RequiredFields:   []string{"description"},
			FilenameRequired: "SKILL.md",
		},
	},

	// OpenCode reads markdown + YAML for all three. Loader is lenient
	// on agent/command frontmatter; skills require `name`.
	// Source: github.com/sst/opencode packages/opencode/src/config/{agent,command}.ts and skill/index.ts
	TargetOpenCode: {
		KindAgents: {
			Format:         FormatYAMLFrontmatter,
			RequiredFields: []string{"description"},
		},
		KindCommands: {
			Format:         FormatYAMLFrontmatter,
			RequiredFields: []string{"description"},
		},
		KindSkills: {
			Format:           FormatYAMLFrontmatter,
			RequiredFields:   []string{"name", "description"},
			FilenameRequired: "SKILL.md",
		},
	},

	// Cursor has no agent/command/skill primitives — everything
	// installed under .cursor/rules/ is read as plain text. Hero
	// keeps the kind subdirs for human organization. Contract
	// asserts the file has a non-empty body.
	// Source: https://cursor.com/docs/context/rules
	TargetCursor: {
		KindAgents:   {Format: FormatFreeform},
		KindCommands: {Format: FormatFreeform},
		// Skills are flattened to <name>.md (installSkillsFlat) —
		// Cursor rules are flat files, not nested SKILL.md dirs.
		KindSkills: {Format: FormatFreeform, FilenameSuffix: ".md"},
	},

	// Codex CLI reads TOML for agents (`developer_instructions`
	// required); markdown SKILL.md for skills (`name`+`description`
	// required); has NO loader for commands (we install commands as
	// skills via .agents/skills/ — see target_codex.go).
	// Source: codex-rs/core/src/config/agent_roles.rs and core-skills/src/loader.rs
	TargetCodex: {
		KindAgents: {
			Format:         FormatTOML,
			RequiredFields: []string{"developer_instructions"},
			FilenameSuffix: ".toml",
		},
		// KindCommands intentionally absent — Codex has no command loader.
		KindSkills: {
			Format:           FormatYAMLFrontmatter,
			RequiredFields:   []string{"name", "description"},
			FilenameRequired: "SKILL.md",
		},
	},

	// Copilot reads SKILL.md for skills, .prompt.md for user-invoked
	// prompts (Hero installs both agents and commands as prompts,
	// subdir-namespaced). Single instructions file is a separate
	// concern handled by installInstructionsMd.
	// Source: github.com/microsoft/vscode-copilot-chat
	//         src/platform/customInstructions/common/promptTypes.ts
	TargetCopilot: {
		KindAgents: {
			Format:         FormatYAMLFrontmatter,
			FilenameSuffix: ".prompt.md",
		},
		KindCommands: {
			Format:         FormatYAMLFrontmatter,
			FilenameSuffix: ".prompt.md",
		},
		KindSkills: {
			Format:           FormatYAMLFrontmatter,
			FilenameRequired: "SKILL.md",
		},
	},

	// Generic is Hero's catch-all — no consuming loader. The contract
	// matches Hero's authoring standard so any future tool picking up
	// .ai/ benefits from consistent shape.
	TargetGeneric: {
		KindAgents: {
			Format:         FormatYAMLFrontmatter,
			RequiredFields: []string{"name", "description"},
		},
		KindCommands: {
			Format:         FormatYAMLFrontmatter,
			RequiredFields: []string{"description"},
		},
		KindSkills: {
			Format:           FormatYAMLFrontmatter,
			RequiredFields:   []string{"description"},
			FilenameRequired: "SKILL.md",
		},
	},

	// Grok Build loads Markdown agents and nested skills. Hero workflows are
	// command-* skills, so there is intentionally no commands contract cell.
	TargetGrok: {
		KindAgents: {
			Format:         FormatYAMLFrontmatter,
			RequiredFields: []string{"name", "description"},
		},
		KindSkills: {
			Format:           FormatYAMLFrontmatter,
			RequiredFields:   []string{"name", "description"},
			FilenameRequired: "SKILL.md",
		},
	},
}

// ContractsFor returns the contract for the (target, kind) cell or
// (zero, false) if none is declared. Callers use the boolean to
// distinguish "no contract declared yet" from "contract declared with
// zero requirements".
func ContractsFor(target Target, kind ContentKind) (HarnessContract, bool) {
	kinds, ok := targetContracts[target]
	if !ok {
		return HarnessContract{}, false
	}
	c, ok := kinds[kind]
	return c, ok
}
