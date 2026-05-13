---
title: Harness Install Paths — Land Files Where Each Tool Actually Loads Them
type: feature
status: completed
priority: P0
completed: 2026-05-12
severity: high
created: 2026-05-12
tags: [install, paths, codex, copilot, claude, opencode, cursor, generic, foundation]
relations:
  - target: install-upgrade-contract-coverage
    kind: child
  - target: install-contract-registry-foundation
    kind: builds-on
  - target: claude-subagent-frontmatter-registration
    kind: builds-on
  - target: multi-harness-install-collision
    kind: builds-on
---

# Harness Install Paths — Land Files Where Each Tool Actually Loads Them

## Goal

Every supported install target writes agents, commands, and
skills to paths the consuming harness actually loads from — at
both project and global scope — in the format the harness
expects. Symlinks to canonical `.hero/` content where the format
matches; render-at-install where it differs.

After this lands: `hero install --target X` produces files X
actually consumes. Today three of six targets ship installs the
consuming tool silently ignores.

## Kickoff

Source-verified install paths per harness — landed. Codex agents
now render as TOML, Copilot installs to the paths VS Code Copilot
Chat actually reads, legacy dead bytes get cleaned up, and
`hero upgrade` delegates to the per-target installer so migration
just works.

**Status:** completed — full suite passes, vet clean, manual
install smokes confirm 34 agents / 27 commands / 45 skills land
correctly across all six targets, upgrade-from-legacy preserves
user content while migrating Hero-authored bytes.

**What landed:**
- [internal/install/render.go](internal/install/render.go) — `renderToFile`, `renderCodexAgentToml`, `renderCopilotPromptFile`, `parseSimpleFrontmatter`.
- [internal/install/cleanup.go](internal/install/cleanup.go) — `removeIfHeroAuthored`, `dirIsOnlyHeroAuthored`, `matchesNestedSkillCanonical`.
- [internal/install/target_codex.go](internal/install/target_codex.go) — TOML rendering for agents (`developer_instructions`), `.agents/skills/` install, cleanup of legacy `.codex/{agents/*.md,commands/}`.
- [internal/install/target_copilot.go](internal/install/target_copilot.go) — `.prompt.md` rendering for agents+commands at `.github/prompts/{agents,commands}/`, `.github/skills/<n>/SKILL.md` install, cleanup of legacy `.github/copilot/{agents,commands,skills}/`.
- [internal/install/contracts.go](internal/install/contracts.go) — added `Format` (yaml-frontmatter / toml / freeform) and `FilenameSuffix`; per-target contracts now reflect each harness's actual loader requirements with source citations.
- [internal/install/install.go](internal/install/install.go) — added `Options.TrustedChecksums` for upgrade migration.
- [internal/install/files.go](internal/install/files.go) — `isTrustedHeroInstalledFile` lets upgrade refresh prior-version Hero-installed bytes without `--force`.
- [internal/install/linking.go](internal/install/linking.go) — auto-migrates legacy regular directories into the canonical-symlink layout when contents are detectably Hero-authored.
- [internal/cli/upgrade.go](internal/cli/upgrade.go) — `upgradeTarget` now invokes `install.Run` per target (cleanup + rendering + symlinks + hooks all happen automatically); per-file checksums recorded by walking through symlinks via `EvalSymlinks`.
- [internal/install/harness_smoke_test.go](internal/install/harness_smoke_test.go) — new smoke tests for codex, copilot, generic at the corrected paths.
- [internal/install/migration_test.go](internal/install/migration_test.go) — `TestMigration_CodexLegacyLayoutCleanup`, `TestMigration_CopilotLegacyLayoutCleanup`, `TestMigration_PreservesUserEditedLegacyContent`.

**Validation done:**
- `go test ./...` — full suite passes.
- `go vet ./...` — clean.
- Manual: fresh install per target — Codex shows 34 TOML agents + 45 skills at `.agents/skills/`; Copilot shows 34 prompt files at `.github/prompts/agents/` + 27 commands + 45 skills at `.github/skills/`; zero `.github/copilot/` content.
- Manual: legacy-layout fixture + reinstall — Hero-authored bytes cleaned up, user-authored bytes preserved with warning, new layout installed.

→ `hero spec complete .hero/planning/features/harness-install-paths-match-loaders/spec.md`

**Why this matters:** users on the prior release have install
content the consuming tool never reads. After this lands, a
single `hero upgrade` migrates them to the corrected layout with
zero manual cleanup — the foundation for the rest of the
install-upgrade-contract-coverage initiative.

**Skip:** Copilot `.agent.md` proposed-API integration, OpenCode
command `template:` audit, `installInstructionsMd` managed-region
parity, real-harness validation hook (parent initiative #6).

## Source-verified install matrix

Each cell sourced from reading the consuming tool's loader code
(or, for closed-source Cursor, its docs). Citations inline.

### Claude Code (project + global)

| Kind | Project path | Global path | Format | Required FM | Source |
|---|---|---|---|---|---|
| Agents | `.claude/agents/<n>.md` | `~/.claude/agents/<n>.md` | md+YAML | `name`, `description` | [docs](https://code.claude.com/docs/en/sub-agents) |
| Commands | `.claude/commands/<n>.md` | `~/.claude/commands/<n>.md` | md+YAML | none | [docs](https://code.claude.com/docs/en/skills) |
| Skills | `.claude/skills/<n>/SKILL.md` | `~/.claude/skills/<n>/SKILL.md` | md+YAML | none (description recommended) | [docs](https://code.claude.com/docs/en/skills) |
| Instructions | `CLAUDE.md` | `~/.claude/CLAUDE.md` | freeform | n/a | [docs](https://code.claude.com/docs/en/settings) |
| Hooks/perms | `.claude/settings.json` | `~/.claude/settings.json` | JSON | structured schema | [docs](https://code.claude.com/docs/en/settings) |

**Hero today:** correct. No path changes.

### OpenCode (project + global)

Source: [`packages/opencode/src/config/agent.ts:107-137`](https://github.com/sst/opencode), [`config/command.ts:15-62`](https://github.com/sst/opencode), [`skill/index.ts:21-25,163-221`](https://github.com/sst/opencode), [`session/instruction.ts:13-17,106-147`](https://github.com/sst/opencode).

| Kind | Project path | Global path | Format | Required FM | Notes |
|---|---|---|---|---|---|
| Agents | `.opencode/agent[s]/**/*.md` | `~/.config/opencode/agent[s]/**/*.md` (also `~/.opencode/`) | md+YAML | none (loader is lenient — agent file with NO frontmatter loads) | both `agent/` and `agents/` walked |
| Commands | `.opencode/command[s]/**/*.md` | `~/.config/opencode/command[s]/**/*.md` | md+YAML | none enforced; **strict schema** (extra frontmatter keys throw); body becomes prompt template | `template:` field is overwritten by markdown body |
| Skills | `.opencode/skill[s]/**/SKILL.md` PLUS cross-tool fallback `.claude/skills/**/SKILL.md` AND `.agents/skills/**/SKILL.md` (both project-walked and global at `$HOME`) | `~/.config/opencode/skill[s]/...` and the same cross-tool dirs | md+YAML | `name:` (silent drop if missing); description optional but skills without description are filtered from model presentation | Cross-tool reading means installing once to `.claude/skills/` covers OpenCode too |
| Instructions | `AGENTS.md` (or `CLAUDE.md` fallback, walked from cwd up) | `~/.config/opencode/AGENTS.md` (or `~/.claude/CLAUDE.md` fallback) | freeform | n/a | AGENTS.md preferred |
| Config / MCP | `opencode.json` | `~/.config/opencode/opencode.json` | JSON, **strict top-level schema** (extra keys throw) | MCP via `mcp` key | hooks are TS plugins, not JSON |

**Hero today:** correct paths. ⚠️ Need to verify Hero's canonical
command frontmatter only uses `description`, `agent`, `model`,
`subtask` keys — anything else throws under OpenCode's strict
schema. Audit during delivery.

### Cursor (project; no filesystem global)

Source: docs only — Cursor is closed source. [Cursor Rules](https://cursor.com/docs/context/rules).

| Kind | Project path | Global | Format | Notes |
|---|---|---|---|---|
| Rules | `.cursor/rules/*.md` or `*.mdc` (recursive subdirs OK) | User Rules in Settings UI; no FS path | md / .mdc | `.mdc` supports optional `description`/`globs`/`alwaysApply` |
| Instructions | `AGENTS.md` at root | n/a | freeform | alternative to `.cursor/rules` |

Cursor has **no agent loader, no command loader, no skill
loader**. Everything in `.cursor/rules/` reads as plain rules.

**Hero today:** correct. Hero's nested `.cursor/rules/{agents,commands,skills}/` works — Cursor reads them as rule files, the subdir layout is human organization.

### Codex CLI (project + global)

Source: [`codex-rs/core/src/config/agent_roles.rs:73-96, 217-225, 518-550`](https://github.com/openai/codex), [`codex-rs/core-skills/src/loader.rs:106-110, 270-374`](https://github.com/openai/codex), [`codex-rs/tui/src/slash_command.rs:8-76`](https://github.com/openai/codex), [`hooks/src/engine/discovery.rs`](https://github.com/openai/codex), [`docs/agents_md.md`](https://github.com/openai/codex/blob/main/docs/agents_md.md).

| Kind | Project path | Global path | Format | Required FM | Notes |
|---|---|---|---|---|---|
| **Agents** | `.codex/agents/*.toml` | `~/.codex/agents/*.toml` | **TOML** | `developer_instructions`. Optional: `name`, `description`, `nickname_candidates` | Loader filters `.toml` extension only. Hero's current `.md` install is dead. |
| Commands | **NO LOADER ANYWHERE** | **NO LOADER ANYWHERE** | n/a | n/a | SlashCommand is a built-in Rust enum. Migration code converts `.claude/commands/*` to skills under `.agents/skills/`. |
| Skills | BOTH `.codex/skills/<n>/SKILL.md` (via project config layer) AND `.agents/skills/<n>/SKILL.md` (via dedicated repo walk) | BOTH `~/.codex/skills/` (deprecated) AND `~/.agents/skills/` (current) | md+YAML | `name`, `description` | `.agents/skills/` is the cross-tool standard — also read by OpenCode |
| Instructions | `AGENTS.md` (project root, walked from cwd up; `AGENTS.override.md` priority) | `~/.codex/AGENTS.md` (concatenated with project) | freeform | n/a | |
| Hooks | `.codex/hooks.json` OR `.codex/config.toml` `[[hooks.<Event>]]` | `~/.codex/hooks.json` OR `~/.codex/config.toml` | TOML/JSON | requires `[features] codex_hooks = true` | |

**Hero today (BROKEN):**
- Writes `.codex/agents/<n>.md` — Codex skips (only reads `.toml`).
- Writes `.codex/commands/<n>.md` — no loader exists.
- Writes `.codex/skills/<n>/SKILL.md` — actually works.
- AGENTS.md, hooks — correct.

### GitHub Copilot (project; no FS global)

Source: [`microsoft/vscode-copilot-chat`](https://github.com/microsoft/vscode-copilot-chat) — `customInstructions/common/promptTypes.ts:19-44`, `customInstructionsService.ts:226-322,473`, `chatSessions/vscode-node/copilotCloudSessionsProvider.ts:782`.

| Kind | Project path | Personal/Home | Format | Required FM | Notes |
|---|---|---|---|---|---|
| Agents | NO `.github/copilot/agents/` loader. `.agent.md` is a VS Code proposed-API surface; default location set by `chat.agentLocations` (no fixed default) | n/a | md+YAML (when configured) | n/a | Cloud-agent integration scans `.github/agents/*.md` for correlation only — not injected as system prompts |
| Commands | NO `.github/copilot/commands/` loader. `.prompt.md` files at `.github/prompts/*.prompt.md` (default; configurable via `chat.promptFilesLocations`) | n/a | md+YAML | none enforced | User-invoked prompts |
| Skills | `.github/skills/<n>/SKILL.md` AND `.claude/skills/<n>/SKILL.md` (workspace; gated by `chat.useAgentSkills`) | `.copilot/skills/<n>/SKILL.md` (under `$HOME`) | md+YAML SKILL.md | none enforced | Filename literally `SKILL.md` (case-insensitive) |
| Instructions (single) | `.github/copilot-instructions.md` | n/a | freeform | n/a | gated by `chat.codeGeneration.useInstructionFiles` (default on) |
| Path-scoped instructions | `.github/instructions/<n>.instructions.md` (configurable via `chat.instructionsFilesLocations`) | n/a | md+YAML | optional `applyTo:` glob (controls auto-attach) | filename suffix `.instructions.md` required |
| MCP | `.vscode/mcp.json` | `~/.config/Code/User/mcp.json` (platform-equivalent) | JSON | structured | via VS Code core, not Copilot Chat directly |

**Hero today (BROKEN):**
- Writes `.github/copilot/agents/`, `commands/`, `skills/` — Copilot ignores all three subdirs entirely.
- Writes `.github/copilot-instructions.md` — correct.

### Generic (project only)

| Kind | Path | Loader | Notes |
|---|---|---|---|
| Agents | `.ai/agents/<n>.md` | none — Hero convention | catch-all for unknown tools |
| Commands | `.ai/commands/<n>.md` | none | same |
| Skills | `.ai/skills/<n>/SKILL.md` | none | same |
| Instructions | `AGENTS.md` at root | freeform; widely-read convention | yes |

**Hero today:** correct. `.ai/` is intentional Hero convention.

## Per-harness implementation changes

### Claude (`internal/install/target_claude.go`)

- **No path changes.** Already correct.
- Update top-of-file docstring to cite docs URLs and note both
  project and global modes are correct.
- Add global-mode harness smoke test (`TestHarness_SmokeClaudeGlobal`).

### OpenCode (`internal/install/target_opencode.go`)

- **No path changes.** Already correct.
- Update docstring to cite source URLs and note the cross-tool
  skill fallback (OpenCode reads `.claude/skills/` and
  `.agents/skills/` too).
- Audit canonical commands: confirm frontmatter only uses
  `description`/`agent`/`model`/`subtask` keys (OpenCode rejects
  unknown). Surface as a finding if Hero's commands have other
  fields.
- Add global-mode harness smoke test
  (`TestHarness_SmokeOpenCodeGlobal`) asserting
  `~/.config/opencode/{agent,command,skill}s/`.

### Cursor (`internal/install/target_cursor.go`)

- **No path changes.** Cursor reads `.cursor/rules/` as plain
  rules; nested kind dirs are organizational.
- Update docstring to call out: no agent/command/skill loader,
  Hero ships rule files. No global filesystem path.

### Codex (`internal/install/target_codex.go`) — BREAKING

**Stop installing:**
- `.codex/agents/<n>.md` — wrong format (Codex requires `.toml`).
- `.codex/commands/<n>.md` — no loader exists.
- (Keep `.codex/skills/` for now; it works. But add `.agents/skills/` as the new preferred location — see below.)

**Add installs:**
- **Agents (project + global, render TOML):** Hero introduces a
  new render step `renderAgentsAsCodexToml(canonicalAgentsDir,
  destDir)`. For each canonical agent markdown file, parse the
  YAML frontmatter (`name`, `description`) and the body, write
  out:
  ```toml
  name = "<filename-stem>"
  description = "<from frontmatter>"
  developer_instructions = """
  <full markdown body>
  """
  ```
  Destinations: `<projectRoot>/.codex/agents/<name>.toml`
  (project), `~/.codex/agents/<name>.toml` (global). Cannot
  symlink (format differs from canonical markdown source).
- **Skills (project + global):** add `linkOrRenderDir` for
  skills targeting `<projectRoot>/.agents/skills/` (project) and
  `~/.agents/skills/` (global). Symlink to canonical
  `.hero/skills/`. Keep `.codex/skills/` install too for
  back-compat with anyone reading the project config layer
  directly.
- **Commands (NOT installed at any scope):** Codex has no
  command loader. Hero's commands surface to Codex via skills
  instead — already covered by the skills install. Document this
  in the docstring; do not write `.codex/commands/` or
  `~/.codex/prompts/` (the latter has been removed from current
  Codex source).

**Keep:**
- AGENTS.md at project root and `~/.codex/AGENTS.md` (already
  correct via `installAgentsMd`).
- `wireCodexHooks` writing `.codex/hooks.json` /
  `~/.codex/hooks.json` (already correct).

**Cleanup of dead bytes from prior installs:** when running
`runCodex`, detect Hero-installed `.codex/agents/<n>.md` (markdown
files matching canonical content) and remove them. Same for
`.codex/commands/`. Mirror the `cleanupFlatSkills` pattern in
`internal/install/content.go`. Skip removal if the file is not
detectably Hero-authored (don't delete user content).

**Update docstring** at the top of `target_codex.go` to cite the
authoritative source code paths.

### Copilot (`internal/install/target_copilot.go`) — BREAKING

**Stop installing:**
- `.github/copilot/agents/` — Copilot ignores `.github/copilot/` subdirs entirely.
- `.github/copilot/commands/` — same.
- `.github/copilot/skills/` — same. Move skills to `.github/skills/` (see below).

**Add installs:**
- **Skills:** install canonical skills at
  `<projectRoot>/.github/skills/<n>/SKILL.md` via
  `linkOrRenderDir` symlinking to canonical `.hero/skills/`.
  Same SKILL.md format — no rendering needed.
- **Agents:** render each canonical agent as a Copilot
  `.prompt.md` file at
  `<projectRoot>/.github/prompts/<n>.prompt.md`. Body is the
  agent markdown; frontmatter carries `description` (from
  canonical) plus a `mode: 'agent'` marker (or whatever Copilot's
  prompt-file frontmatter conventions are — default to just
  `description:` if unsure, since prompt files don't enforce
  frontmatter). One file per Hero agent.
- **Commands:** render each canonical command as a `.prompt.md`
  file at `<projectRoot>/.github/prompts/<n>.prompt.md`.
  Collision with agents: prefix to avoid (`agent-<n>.prompt.md`
  vs `cmd-<n>.prompt.md`), OR install agents under
  `.github/prompts/agents/` and commands under
  `.github/prompts/commands/` if VS Code's prompt discovery
  recurses (default scan IS recursive per VS Code docs — confirm
  during delivery).
- **Path-scoped instructions:** out of scope. Hero has no source
  content shaped for `.github/instructions/<n>.instructions.md`
  with `applyTo:` glob semantics. Surface as Followup.

**Keep:**
- `.github/copilot-instructions.md` via `installInstructionsMd`
  (already correct).

**Project-only constraint** unchanged. Copilot has no FS global
scope; per-user MCP/skills via VS Code Settings is out of scope.

**Cleanup of dead bytes:** detect and remove Hero-installed
`.github/copilot/{agents,commands,skills}/` content. Same
careful-symlink-or-canonical-bytes detection.

**Update docstring** to cite source code and current paths.

### Generic (`internal/install/target_generic.go`)

- **No path changes.**
- Update docstring to clarify `.ai/` is a Hero convention with
  no consuming loader (catch-all for tools without dedicated
  installers).

## Upgrade & migration from the prior release

Users running the previous Hero release have dead bytes left over
from broken install paths:

- `.codex/agents/<n>.md` (markdown — Codex never read these)
- `.codex/commands/<n>.md` (no loader at any scope)
- `.github/copilot/agents/<n>.md`
- `.github/copilot/commands/<n>.md`
- `.github/copilot/skills/<n>/SKILL.md`

After this spec lands, those users need a clean migration to the
corrected layout (`.codex/agents/*.toml`, `.agents/skills/`,
`.github/prompts/*.prompt.md`, `.github/skills/`). Today's
`hero upgrade` cannot deliver this because it does its own
file-level walk instead of invoking the per-target installer.
Specifically:
- It refreshes existing destination files in place — so old
  `.codex/agents/*.md` content gets re-written, not removed.
- It never invokes `runCodex` / `runCopilot` so the new TOML and
  `.prompt.md` rendering never happens.
- It never runs cleanup of legacy paths.

### Required change: `hero upgrade` delegates to the per-target installer

Refactor [`internal/cli/upgrade.go`](internal/cli/upgrade.go)
`upgradeTarget` to invoke the target's `Run` (via
`install.Run(opts)` with `Force: true` semantics for unmodified
files only) instead of walking destination dirs directly. This
makes upgrade automatically inherit:

- Cleanup of dead bytes from prior install layouts (per-target
  cleanup logic added in this spec).
- Render-at-install for format-divergent targets (Codex TOML,
  Copilot `.prompt.md`).
- New destination paths (`.agents/skills/`, `.github/skills/`,
  `.github/prompts/`) without duplicating path logic.
- Correct symlink behavior under P2 canonical layout.
- MCP wiring (already done by `Run`; remove the duplicate
  `RegisterMCP` call in `upgradeTarget`).

Preserve the `--force` semantic: today, customized destination
files are skipped unless `--force` is set. Carry that through to
`Run` by:
- Detecting modified destination files (via the existing
  `version.IsFileModified` helper) BEFORE invoking `Run`.
- For modified files, either skip the install for that file or
  print a per-file warning. The cleanest path: set a new
  `Options.SkipModified bool` field that the per-target
  installer respects, defaulting to `false` (install behavior
  unchanged); upgrade sets it to `true` unless `--force` is
  set.

### Required: cleanup of legacy install paths

Both `runCodex` and `runCopilot` get a cleanup step BEFORE the
new install. Pattern mirrors `cleanupFlatSkills` in
[`internal/install/content.go`](internal/install/content.go):

```go
// cleanupLegacyCodexPaths removes Hero-installed dead bytes from
// the prior release's incorrect Codex install layout. Only
// removes entries detectable as Hero-authored (symlink to
// .hero/, or rendered copy whose bytes match canonical embedded
// source). User content is left in place with a warning.
func cleanupLegacyCodexPaths(opts Options, result *Result) error {
    legacy := []string{
        filepath.Join(opts.TargetDir, ".codex", "agents"),
        filepath.Join(opts.TargetDir, ".codex", "commands"),
    }
    for _, dir := range legacy {
        if err := removeIfHeroAuthored(opts, result, dir); err != nil {
            return err
        }
    }
    return nil
}
```

`removeIfHeroAuthored` shared helper in `content.go`:

1. Read each entry under `dir`.
2. If entry is a symlink and its target is under `.hero/`, remove the symlink.
3. If entry is a regular file and its bytes match the canonical embedded source for that name, remove the file.
4. Otherwise, leave it and emit a warning to `result` ("not removed: looks user-edited").
5. After processing entries, remove the `dir` itself if it's now empty.

Same cleanup for Copilot:
```go
legacy := []string{
    filepath.Join(opts.TargetDir, ".github", "copilot", "agents"),
    filepath.Join(opts.TargetDir, ".github", "copilot", "commands"),
    filepath.Join(opts.TargetDir, ".github", "copilot", "skills"),
}
```

Also remove `.github/copilot/` itself if empty after cleanup.

### Migration smoke tests

Add `internal/install/migration_test.go` (new file):

- `TestUpgrade_FromLegacyCodexLayout`:
  1. Stand up a fixture project with `.codex/agents/engineer.md`,
     `.codex/commands/design.md`, `.codex/skills/spec-format/SKILL.md`,
     `AGENTS.md`, `.codex/hooks.json` — simulating prior release output.
  2. Run `hero upgrade --target codex` (or call the upgrade entry point).
  3. Assert `.codex/agents/engineer.toml` exists and parses correctly.
  4. Assert `.codex/agents/engineer.md` does NOT exist (cleaned up).
  5. Assert `.codex/commands/` does NOT exist (cleaned up).
  6. Assert `.agents/skills/spec-format/SKILL.md` exists.
  7. Assert AGENTS.md and `.codex/hooks.json` are intact (not regressed).
- `TestUpgrade_FromLegacyCopilotLayout`:
  1. Stand up `.github/copilot/agents/`, `commands/`, `skills/`,
     `.github/copilot-instructions.md`.
  2. Run `hero upgrade --target copilot`.
  3. Assert `.github/copilot/{agents,commands,skills}` removed.
  4. Assert `.github/skills/spec-format/SKILL.md` exists.
  5. Assert `.github/prompts/agents/engineer.prompt.md` (or chosen layout) exists.
  6. Assert `.github/copilot-instructions.md` is intact.
- `TestUpgrade_PreservesUserEditedLegacyContent`:
  1. Stand up legacy layout.
  2. Modify one file (so it's no longer Hero-authored bytes).
  3. Run `hero upgrade`.
  4. Assert the modified file is preserved with a warning logged.

### Migration story for users

Document in upgrade output and release notes:

- After upgrading the Hero binary, run `hero upgrade` (or
  `hero install --migrate` for satellite repair simultaneously).
- Cleanup is automatic for unmodified Hero-installed files.
- User-edited files in legacy locations are preserved with a
  warning telling the user they can manually delete them once
  reviewed.

Add an upgrade output line:

```
  cleanup .codex/agents/ (5 Hero-installed files removed; format changed to .toml)
  cleanup .codex/commands/ (5 Hero-installed files removed; no loader exists in current Codex)
  cleanup .github/copilot/agents/ (5 Hero-installed files removed; Copilot reads .github/prompts/ instead)
  ...
```

Specific user-facing benefit: a single `hero upgrade` after the
new release lands and the user's project is fully on the
corrected install paths, with no manual intervention.

## New install primitive: format-rendering pipeline

Two render targets need new code: Codex agent TOML and Copilot
prompt-file rendering.

Add a generic helper in `internal/install/render.go` (new file):

```go
// renderTarget describes a per-target rendering of canonical
// content into a destination format the harness consumes
// natively. Used when the format differs from canonical (e.g.
// Codex agents are TOML, Hero canonical agents are markdown).
type renderTarget struct {
    Kind     ContentKind
    DestDir  string
    Render   func(canonical canonicalEntry) ([]byte, string, error)
    // Render returns (rendered bytes, destination filename, err).
}

// renderCanonical iterates the canonical content for kind, runs
// each entry through Render, and writes to DestDir/<filename>.
func renderCanonical(opts Options, result *Result, rt renderTarget) error { ... }
```

Two concrete renderers:

```go
// codexAgentTomlRenderer reads a canonical agent markdown file and
// emits a Codex .toml subagent definition.
func codexAgentTomlRenderer(entry canonicalEntry) ([]byte, string, error) {
    // parse frontmatter to get name + description
    // body becomes developer_instructions
    // write TOML with proper escaping for triple-quoted strings
}

// copilotPromptRenderer reads a canonical agent or command markdown
// file and emits a Copilot .prompt.md file. The rendering is mostly
// pass-through; just renames the destination filename and may
// adjust frontmatter keys to Copilot prompt-file conventions.
func copilotPromptRenderer(entry canonicalEntry) ([]byte, string, error) { ... }
```

Both renderers READ canonical bytes once and emit rendered bytes.
No per-harness source files in `agents/`, `commands/`, `skills/`
canonical dirs. Single source preserved.

## Updates to install-contract-registry

In `internal/install/contracts.go`:

- **Codex:**
  - `KindAgents` → `RequiredFrontmatter: ["developer_instructions"]` (TOML field), `FilenameRequired` matches `<n>.toml` pattern. New `ContentValidator` may be useful to confirm TOML parses.
  - `KindCommands` → empty contract (no loader; document why).
  - `KindSkills` → `RequiredFrontmatter: ["name", "description"]`, `FilenameRequired: "SKILL.md"`.
- **Copilot:**
  - `KindAgents` → declared with `FilenameRequired: "<n>.prompt.md"` pattern; `RequiredFrontmatter: ["description"]` (or empty if Hero just renames).
  - `KindCommands` → same as agents (both render as `.prompt.md`).
  - `KindSkills` → `RequiredFrontmatter` matching canonical skill contract; `FilenameRequired: "SKILL.md"`.
- **Generic:** declare contracts matching canonical content
  (agents need name+description; commands and skills need
  description; skills need SKILL.md filename).
- **The HarnessContract type may need extension:**
  - Today's `RequiredFrontmatter` assumes YAML frontmatter parsing. Codex agents use TOML — the contract validator needs to detect TOML vs YAML (or have separate `RequiredTomlKeys` field, OR have a `Format string` field with values "yaml-frontmatter", "toml", "freeform"). Keep extension minimal — add `Format ContractFormat` enum if needed.

## Test changes

Per-target smoke tests at project AND global scope:

- `TestHarness_SmokeClaude` (project) — exists; extend with
  global counterpart `TestHarness_SmokeClaudeGlobal`.
- `TestHarness_SmokeOpenCode` (project) — exists; extend with
  global.
- `TestHarness_SmokeCursor` (project only — Cursor has no FS
  global).
- `TestHarness_SmokeCodex` + `TestHarness_SmokeCodexGlobal` —
  new. Asserts:
  - `.codex/agents/engineer.toml` exists, parses as TOML, has `developer_instructions`.
  - `.codex/agents/engineer.md` does NOT exist (regression guard).
  - `.codex/commands` directory does NOT exist (no loader).
  - `.agents/skills/spec-format/SKILL.md` exists (preferred location).
  - `.codex/skills/spec-format/SKILL.md` MAY exist (back-compat keep) — verify and document.
  - `AGENTS.md`, `.codex/hooks.json` — exist.
- `TestHarness_SmokeCopilot` — new. Asserts:
  - `.github/prompts/<agent or cmd>.prompt.md` files exist with appropriate naming/namespacing.
  - `.github/skills/spec-format/SKILL.md` exists.
  - `.github/copilot/{agents,commands,skills}` do NOT exist.
  - `.github/copilot-instructions.md` exists.
- `TestHarness_SmokeGeneric` — new. Asserts current `.ai/{agents,commands,skills}/` layout.

The harness needs a global-mode test capability. Extend
`installHarness` with:
- A `globalDir` field (also `t.TempDir()`).
- A `RunGlobal(target)` method that wires `Mode: ModeGlobal` and
  passes a controlled HOME via opts (each target's
  `resolveXPaths` already accepts such a path or honors HOME env
  override — verify per-target during delivery; add overrides
  where missing).

## Boundaries

- Do NOT change canonical content in `agents/`, `commands/`,
  `skills/`, or `core/`/`domains/` mirrors. Single canonical
  source preserved.
- Do NOT add a Copilot `.agent.md` install via the proposed
  chat API. Surface unstable; defer to followup once the API
  stabilizes.
- Do NOT add `.github/instructions/<n>.instructions.md` install.
  Hero has no source content shaped for `applyTo:` globs.
- Do NOT install `~/.codex/prompts/`. Loader was removed from
  current Codex source.
- Do NOT delete user-edited content from legacy install
  locations during cleanup. Only remove detectably Hero-authored
  symlinks or rendered copies whose bytes match canonical
  source.
- Do NOT introduce frontmatter translation between
  same-format-family targets (e.g. Claude markdown ↔ OpenCode
  markdown). Both consume the same canonical file via symlink.
- Do NOT re-author canonical agents in TOML to make Codex happy.
  Render-at-install is the right boundary — canonical stays
  markdown.

## Risks

- **TOML rendering correctness for Codex agents.** Hero must
  emit valid TOML. Triple-quoted multi-line strings must escape
  properly; `developer_instructions = """ ... """` body needs
  to handle `"""` and newlines correctly. **Mitigation:** use a
  Go TOML library (e.g. `github.com/pelletier/go-toml/v2`); add
  unit test that round-trips canonical agent → TOML → parsed
  TOML and verifies fields.
- **Copilot prompt-file collision.** Agents and commands both
  render to `.prompt.md`. **Mitigation:** subdir-namespace
  (`.github/prompts/agents/<n>.prompt.md`,
  `.github/prompts/commands/<n>.prompt.md`); confirm VS Code's
  prompt-file discovery recurses (per docs default
  `.github/prompts/` is the root; recursion is VS-Code-config
  driven).
- **Cleanup over-deletion.** Removing legacy `.codex/agents/`
  and `.github/copilot/` content could remove user-authored
  files. **Mitigation:** only delete entries detectable as
  Hero-installed: symlinks pointing at `.hero/`, OR rendered
  copies whose bytes match canonical embedded source. Anything
  else logs a warning and is left in place.
- **HarnessContract field expansion (Format).** Today's contract
  assumes YAML frontmatter. Adding TOML support means a new
  `Format` field on HarnessContract. Mitigation: keep
  enumeration small (yaml-frontmatter / toml / freeform);
  preserve backward compatibility with the foundation spec by
  defaulting Format to yaml-frontmatter.
- **Global-mode HOME isolation.** Existing harness only tests
  project mode. Adding global tests means writing to
  controlled-HOME temp dirs across all six targets. Mitigation:
  add `globalDir` field to `installHarness`; per-target
  `resolveXPaths` may need updates to honor an explicit override
  rather than `os.UserHomeDir()`. Audit each target during
  delivery.
- **Canonical command frontmatter vs OpenCode strict schema.**
  OpenCode rejects unknown top-level keys in command frontmatter.
  Hero's canonical commands today have only `description:` —
  audit during delivery and surface any other keys as a content
  fix.
- **`.agents/skills/` collision with non-Hero tools.** The
  `.agents/skills/` path is the cross-tool agent-skills
  standard. If a user has another tool installing skills there,
  name collisions surface (Codex/OpenCode will warn-and-overwrite
  on duplicates per source). Acceptable — Hero's slugs are
  unique.
- **Adjacent registry: satellite-harness-coverage.** The
  satellite materializer's `targetLayouts` registry is parallel
  to per-target install logic. After this lands, the destBase
  paths in both should match. Coordination point flagged.

## Validation

- `go test ./...` passes including all new project + global
  smoke tests.
- `go vet ./...` clean.
- TOML-render unit test: canonical `agents/engineer.md` → TOML
  → re-parsed TOML → `developer_instructions` matches body,
  `name` matches stem, `description` matches frontmatter.
- Manual smoke per target:
  - **Codex:** `hero install project . --target codex`. `.codex/agents/engineer.toml` exists, parses as TOML, `developer_instructions` matches body. `.agents/skills/spec-format/SKILL.md` exists. `.codex/agents/engineer.md` does NOT exist after install (cleanup). Verify in real Codex CLI: `codex` recognizes the agent (manual check, optional).
  - **Copilot:** `hero install project . --target copilot`. `.github/skills/spec-format/SKILL.md` exists. `.github/prompts/agents/engineer.prompt.md` (or chosen layout) exists. `.github/copilot/{agents,commands,skills}` do NOT exist after install. `.github/copilot-instructions.md` exists. Verify in VS Code: open Copilot, the prompt files appear in the prompts UI.
  - **Claude (project + global):** unchanged behavior, regression guard.
  - **OpenCode (project + global):** unchanged behavior; `~/.config/opencode/` populated in global mode.
  - **Cursor:** unchanged behavior.
  - **Generic:** unchanged behavior.
- Multi-harness regression: install claude, codex, copilot,
  opencode, cursor sequentially in same project — all succeed
  without `--force`.

## Acceptance Criteria

- WHEN `hero install --target codex` runs (project or global)
  THE SYSTEM SHALL render each canonical agent as a TOML file at
  `.codex/agents/<n>.toml` (project) or `~/.codex/agents/<n>.toml`
  (global) with `name`, `description`, and
  `developer_instructions` fields populated from the canonical
  agent markdown.
- WHEN `hero install --target codex` runs THE SYSTEM SHALL
  install canonical skills at `.agents/skills/<n>/SKILL.md`
  (project) or `~/.agents/skills/<n>/SKILL.md` (global) via
  symlink-or-render to canonical `.hero/skills/`.
- WHEN `hero install --target codex` runs THE SYSTEM SHALL NOT
  write to `.codex/agents/<n>.md` (markdown form), to
  `.codex/commands/`, or to `~/.codex/prompts/`.
- IF `.codex/agents/<n>.md` exists from a prior Hero install AND
  is detectably Hero-authored THEN re-running `hero install
  --target codex` SHALL remove it.
- WHEN `hero install --target copilot` runs THE SYSTEM SHALL
  install canonical skills at `.github/skills/<n>/SKILL.md` AND
  render canonical agents/commands as `.prompt.md` files under
  `.github/prompts/`.
- WHEN `hero install --target copilot` runs THE SYSTEM SHALL
  NOT write to `.github/copilot/agents/`,
  `.github/copilot/commands/`, or `.github/copilot/skills/`.
- IF `.github/copilot/{agents,commands,skills}/` content exists
  from a prior install AND is detectably Hero-authored THEN
  re-running `hero install --target copilot` SHALL remove it.
- WHEN `hero install --target claude|opencode|cursor|generic`
  runs THE SYSTEM SHALL produce the same destination layout as
  today (verified by existing smoke tests, plus new global-mode
  coverage for claude and opencode).
- WHEN any target installs THE SYSTEM SHALL preserve the
  canonical-symlink architecture for cells where the target's
  format matches canonical (markdown → markdown). Render-at-install
  is used only where the format genuinely differs (Codex TOML,
  Copilot `.prompt.md`).
- WHEN HarnessContract validates a Codex agent file THE SYSTEM
  SHALL parse the file as TOML and assert
  `developer_instructions` is present and non-empty.
- THE SYSTEM SHALL emit per-target docstrings citing the
  authoritative source-code paths or doc URLs that justify each
  destination path.
- THE SYSTEM SHALL preserve the
  multi-harness-install-collision idempotency contract — running
  `hero install` twice for the same target produces zero new
  copies.
- WHEN `hero upgrade` runs against a project upgraded from the
  prior Hero release THE SYSTEM SHALL invoke each detected
  target's per-target installer (which includes cleanup of dead
  bytes from the prior release's incorrect install layout AND
  installation at the corrected paths in the corrected formats).
- WHEN `hero upgrade` encounters legacy Hero-installed files
  detectably matching prior canonical bytes (or symlinks to
  `.hero/`) THE SYSTEM SHALL remove them.
- WHEN `hero upgrade` encounters files in legacy install paths
  that do NOT match Hero-authored content (user-edited or
  unknown origin) THE SYSTEM SHALL leave them in place AND emit
  a warning naming the file and recommending manual review.
- WHEN a user runs `hero upgrade` after upgrading the Hero
  binary THE SYSTEM SHALL produce a single end state where every
  detected harness has files at the correct destination paths in
  the correct formats — no manual migration steps required for
  unmodified content.

## Followups

- **Copilot `.agent.md` proposed-API integration:** when VS Code's
  `chat.customAgents` API stabilizes, install Hero agents as
  proper Copilot custom agents (not just `.prompt.md`).
- **Copilot path-scoped instructions:** if Hero authors content
  shaped for `applyTo:` globs (e.g. language-specific
  guidelines), install at `.github/instructions/<n>.instructions.md`.
- **OpenCode commands `template:` audit:** verify Hero's
  canonical commands don't include frontmatter keys beyond
  OpenCode's strict schema (`template`, `description`, `agent`,
  `model`, `subtask`).
- **`installInstructionsMd` managed-region parity:** the
  copilot/generic instructions writer overwrites without
  preserving user edits. Should adopt `installManagedMarkdown`.
- **Cursor `.mdc` upgrade:** if Hero authors content with
  `globs:`/`alwaysApply:` semantics, switch Cursor install to
  `.mdc` files.
- **Real-harness validation hook:** open actual Codex CLI /
  VS Code Copilot / OpenCode in a Hero-installed fixture and
  verify each tool's introspection (e.g. `codex` shows the
  agent; Copilot prompts UI shows Hero prompts) — covered by
  parent initiative's child #6.
- **Per-target metadata registry consolidation:** between this
  spec's HarnessContract registry, the satellite materializer's
  `targetLayouts`, and per-target wiring (hooks/permissions),
  there are now multiple per-target metadata sources.
  Consolidation spec when patterns stabilize.
- **`/etc/codex/skills/` admin-scope install:** Codex's
  discovery walk includes `/etc/codex/skills/`. Out of scope;
  revisit if a real consumer asks.
