---
title: Hero Trust Global Scope — Apply Hero Allowlist to User-Level Claude Settings
type: feature
status: completed
priority: medium
tags: [cli, trust, claude, permissions, install]
created: 2026-05-14
completed: 2026-05-15
relates-to: [claude-trust-permission-allowlist]
---

## Goal

`hero trust claude` accepts an optional `<project|global>` positional
scope argument, defaulting to `project` (current behavior unchanged).
When invoked as `hero trust claude global`, the `Bash(hero:*)` entry is
written to the user-level settings file (`~/.claude/settings.json`)
instead of the project-local one, so a single command stops Claude Code
prompt-fatigue across every project on the machine — including projects
where `hero install` was never run. The merge stays idempotent and
preserves user-added entries. Same shape for Codex is out of scope (see
Boundaries) because Codex owns its own approval state.

## Kickoff

Shipped. `hero trust claude` now takes an optional `<project|global>`
positional scope. `hero trust claude global` writes `Bash(hero:*)` into
`~/.claude/settings.json`, ending Claude Code prompt fatigue across
every project on the machine — including ones where `hero install` was
never run.

**Status:** completed.

**What landed:**
- `internal/install/claude_hooks.go`: `EnsureClaudeHeroAllowlist(mode, projectDir)` — routes through existing `claudeSettingsPath(opts)`; rejects empty `projectDir` in project mode.
- `internal/cli/trust.go`: positional scope arg, `RangeArgs(1,2)`, `Use` updated. Global path renders with `~` substitution. Codex with explicit scope prints a no-effect note.
- Tests: 5 new CLI tests, 5 new install-package tests; existing tests unchanged.

→ `.hero/specs/hero-trust-global-scope/spec.md`

## Context

The parent spec `claude-trust-permission-allowlist`
(`.hero/specs/claude-trust-permission-allowlist/spec.md`) shipped
`hero trust claude` as a project-only command: it writes `Bash(hero:*)`
into the cwd's `.claude/settings.json`. That fixed Claude Code prompt
fatigue *for projects where `hero install --target claude` has run*.

The remaining pain: the user works across many projects, most of which
have never had `hero install` run against them. Claude Code still
prompts on every `hero` invocation in those projects. Today the only
workaround is `hero install global` (which already wires
`~/.claude/settings.json` via `wireClaudePermissions` — see
`internal/install/target_claude.go:54`). But install is a heavy hammer
when all the user wants is to trust the `hero` command prefix globally.

This spec adds the lightweight one-liner: `hero trust claude global`.

The existing infrastructure makes this nearly free:

- `internal/install/claude_hooks.go:272-288` already has
  `claudeSettingsPath(opts)` resolving project vs. global to the right
  path. The internal `wireClaudePermissions` already honors that.
- `EnsureClaudeHeroAllowlist` (lines 149-162) is the only thing locked
  to project mode — it hard-codes `Options{Mode: ModeProject, TargetDir: projectDir}`.
- `internal/cli/install.go:98-110` shows the positional-arg pattern to
  mirror: parse `args[0]` as `install.Mode`, reject anything that isn't
  `project` or `global`.

## Problem

1. Users hit prompt fatigue across projects where they haven't run
   `hero install --target claude`. A per-project trust command doesn't
   solve this — there can be dozens of projects.
2. `hero install global` does write the allowlist to
   `~/.claude/settings.json`, but it also installs the full set of
   user-level Hero affordances (agents, skills, hooks). Users who just
   want the prompt-fatigue fix don't want the rest of the install
   footprint as a prerequisite.
3. The fix is *almost* in the codebase already — `wireClaudePermissions`
   handles global mode correctly; only the CLI surface and the
   exported helper are locked to project scope.

## Design

### Positional vs. flag (open question 1)

**Positional**, matching `hero install <project|global> [path]`. Users
who have already learned `hero install global` will guess
`hero trust claude global` correctly. The positional form also future-proofs
the CLI for additional scopes (e.g. workspace) without flag bloat.

`hero trust claude --global` is rejected. The cobra `Use` string becomes
`trust <codex|claude> [project|global]`.

### Discoverability (open question 2)

`hero trust claude` (no scope) stays silent on the global option. The
output today is already a two-liner; adding "tip: try `hero trust claude
global` to apply to all projects" would clutter the success path for
the common case where project-scope is what the user wanted. Discovery
happens via `--help` and the `Use` string update.

If the user expresses prompt-fatigue across projects later, that's
better surfaced from a different surface — e.g. a one-shot hint when
the agent detects repeated `hero trust claude` runs across different
project roots. Out of scope here.

### Codex global support (open question 3)

**Skip.** Codex doesn't expose a Hero-writable user-level permission
file in the same way. `hero trust codex` is *already* instructional —
it prints a hint asking the user to ask Codex for persistent approval.
That guidance works equally well across any scope; the Codex agent
itself decides whether the approval is project- or user-scoped.

`hero trust codex global` and `hero trust codex project` are both
accepted by the parser (so the positional scope argument is uniform
across targets and doesn't error out as "unsupported"), but they print
the same hint and warn that the scope argument has no effect for Codex.
The warning prevents silent misbehavior; the parser uniformity prevents
"`hero trust codex global` errored but `hero trust claude global`
worked, why?" surprises.

### Interaction with `hero install global` (open question 4)

**No change.** `hero install global` already calls
`wireClaudePermissions` with `ModeGlobal` and writes the allowlist
entry today (verified at `internal/install/target_claude.go:54`).
`hero trust claude global` becomes the lightweight alias that does
*only* the permission wiring without the rest of install.

If the user runs `hero install global` first and then
`hero trust claude global`, the latter reports "already present" —
that's the existing idempotent path. No new logic needed.

### Function signature

Rather than splitting into `EnsureClaudeHeroAllowlistProject` /
`EnsureClaudeHeroAllowlistGlobal`, take a `Mode` parameter:

```go
func EnsureClaudeHeroAllowlist(mode Mode, projectDir string) (added bool, path string, err error)
```

- `mode == ModeProject` requires `projectDir`. Error otherwise.
- `mode == ModeGlobal` ignores `projectDir` (matching how
  `claudeSettingsPath` already behaves).
- Any other mode is rejected — same behavior `claudeSettingsPath`
  already gives.

The single-signature form keeps the call site clean (`trust.go`
switches on the parsed `Mode` once, passes it through) and matches the
existing `Options{Mode, TargetDir}` style used elsewhere in the install
package.

### Error contract

- Unknown scope arg ("workspace", "foo"): cobra-level error, "scope
  must be 'project' or 'global', got %q" — same wording as
  `internal/cli/install.go:100-101`.
- Project scope with no resolvable project root: same error as today —
  "could not resolve project root; run from inside a hero workspace".
- Global scope when `os.UserHomeDir()` fails: wrapped error from the
  install layer; CLI surfaces it as-is.

### Output messages

Project scope (unchanged):
- `Claude Code: added Bash(hero:*) to <project>/.claude/settings.json`
- `Claude Code: Bash(hero:*) already present in <project>/.claude/settings.json.`

Global scope (new):
- `Claude Code (global): added Bash(hero:*) to ~/.claude/settings.json`
- `Claude Code (global): Bash(hero:*) already present in ~/.claude/settings.json.`

The leading `(global)` makes the scope explicit in terminal scrollback.
Render the path with `~` substitution if possible (match what other
hero commands do), otherwise the absolute path is fine.

## Acceptance Criteria

- WHEN the user runs `hero trust claude` with no scope arg THE SYSTEM
  SHALL behave exactly as it does today — write to the project's
  `.claude/settings.json` and report project-scoped messages.
- WHEN the user runs `hero trust claude project` THE SYSTEM SHALL
  behave identically to `hero trust claude` with no scope arg.
- WHEN the user runs `hero trust claude global` THE SYSTEM SHALL ensure
  `permissions.allow` in `~/.claude/settings.json` contains
  `"Bash(hero:*)"` and SHALL report a global-scope success message.
- WHEN the user runs `hero trust claude global` and `~/.claude/settings.json`
  already contains `"Bash(hero:*)"` THE SYSTEM SHALL report "already
  present" and SHALL NOT add a duplicate entry.
- IF `~/.claude/settings.json` contains other allowlist entries or
  unrelated settings keys THEN THE SYSTEM SHALL preserve them
  byte-equivalent (other keys) and in original order (allow entries).
- IF `~/.claude/settings.json` does not exist when `hero trust claude
  global` runs THEN THE SYSTEM SHALL create it (and the parent `.claude`
  directory) with `"Bash(hero:*)"` in `permissions.allow` and no
  unrelated keys.
- IF the user passes an unknown scope (e.g. `hero trust claude workspace`)
  THEN THE SYSTEM SHALL exit non-zero with an error message stating
  scope must be 'project' or 'global'.
- WHEN the user runs `hero trust codex global` or `hero trust codex
  project` THE SYSTEM SHALL print the existing codex hint and SHALL
  print a one-line note that the scope argument has no effect for
  Codex.
- THE SYSTEM SHALL keep `hero trust codex` (no scope) output unchanged.
- THE SYSTEM SHALL update the cobra `Use` string to
  `trust <codex|claude> [project|global]` so `--help` documents the
  new argument.

## Files Affected

1. `internal/cli/trust.go`
   - Update `trustCmd.Use` to `"trust <codex|claude> [project|global]"`.
   - Change `Args` from `cobra.ExactArgs(1)` to
     `cobra.RangeArgs(1, 2)`.
   - In `runTrust`, parse the optional `args[1]` as `install.Mode`,
     defaulting to `install.ModeProject` when absent. Reject any value
     that isn't `project` or `global` with the same wording as
     `runInstall`.
   - `applyClaudeTrust(mode)` now takes a `Mode`. When project, resolve
     `findProjectRoot()` (today's path). When global, skip the project
     resolver and pass an empty `projectDir` through to
     `EnsureClaudeHeroAllowlist`.
   - Adjust the success/idempotent messages to include `(global)`
     when applicable. Render `~/.claude/settings.json` with a leading
     `~` when the path lives under `os.UserHomeDir()`.
   - `printCodexTrustHint(scope)` accepts the parsed scope; when scope
     is non-default, append a one-line note: "Note: scope `<scope>`
     has no effect for Codex; Codex owns its own approval state."

2. `internal/install/claude_hooks.go`
   - Change `EnsureClaudeHeroAllowlist(projectDir string)` to
     `EnsureClaudeHeroAllowlist(mode Mode, projectDir string)`.
   - Build `Options{Mode: mode, TargetDir: projectDir}` and pass
     through to the existing `claudeSettingsPath(opts)` +
     `wireClaudePermissions(opts, nil)` flow — no internal logic
     changes needed; the helper already handles `ModeGlobal`.
   - Validation: when `mode == ModeProject` and `projectDir == ""`,
     return a clear error. When `mode` is neither `ModeProject` nor
     `ModeGlobal`, return an error matching `claudeSettingsPath`'s
     "unknown mode" wording.

3. `internal/install/claude_hooks_test.go`
   - Add `TestEnsureClaudeHeroAllowlist_GlobalScope`: redirect
     `HOME` to a `t.TempDir()` (or pass an `Options` override if the
     test harness supports it), call `EnsureClaudeHeroAllowlist(ModeGlobal, "")`,
     and assert `<tmpHome>/.claude/settings.json` contains
     `Bash(hero:*)` and no project-local `.claude` directory was
     created.
   - Add `TestEnsureClaudeHeroAllowlist_GlobalIdempotent`: seed
     `<tmpHome>/.claude/settings.json` with a user entry plus
     `Bash(hero:*)`, run twice, assert exactly one Hero entry, user
     entry preserved.
   - Add `TestEnsureClaudeHeroAllowlist_GlobalPreservesOtherKeys`:
     seed `<tmpHome>/.claude/settings.json` with `model`, `hooks`,
     and another allowlist entry; assert the wiring touches only the
     allow array.
   - Add `TestEnsureClaudeHeroAllowlist_ProjectAndGlobalIndependent`:
     run global, then project (with a separate tempDir), then global
     again — assert both files exist, both contain `Bash(hero:*)`,
     and the second global call reports no change.
   - Existing `TestEnsureClaudeHeroAllowlistIdempotent` updated to
     pass `ModeProject` explicitly (signature change).

4. `internal/cli/trust_test.go`
   - Existing tests (`TestTrustClaudeAppliesAllowlist`,
     `TestTrustClaudeIdempotent`) continue to assert project-scope
     behavior with the no-arg form (no changes needed).
   - Add `TestTrustClaudeProjectExplicit`: identical to
     `TestTrustClaudeAppliesAllowlist` but pass `"project"` as the
     second arg; assert same outcome.
   - Add `TestTrustClaudeGlobal`: redirect `HOME` to env temp dir, run
     `runCmd("trust", "claude", "global")`, assert
     `<HOME>/.claude/settings.json` contains `Bash(hero:*)` and stdout
     contains `Claude Code (global): added`.
   - Add `TestTrustClaudeGlobalIdempotent`: pre-seed
     `<HOME>/.claude/settings.json` with the entry plus a user entry,
     run global, assert "already present" output and user entry
     preserved.
   - Add `TestTrustClaudeUnknownScope`: assert
     `runCmd("trust", "claude", "workspace")` fails with a "scope must
     be 'project' or 'global'" error.
   - Add `TestTrustCodexWithScopeNote`: run
     `runCmd("trust", "codex", "global")` and assert the codex hint
     plus the no-effect note both appear in output.
   - Helper: extend `readClaudeAllowlist` (or add a sibling) to read
     from an arbitrary settings path so both project and global tests
     can use it.

## Out of Scope

- Codex global permission wiring. Codex owns approval state; Hero
  cannot write a meaningful user-level Codex permission file. If the
  Codex model exposes a Hero-writable user surface in the future, a
  separate spec adds that.
- Generalized cross-tool trust helper. The shape stays close to
  `wireClaudePermissions` / `EnsureClaudeHeroAllowlist` until a second
  concrete consumer (opencode, cursor, copilot) lands.
- Workspace or sub-folder scope. Today's two modes mirror what
  `hero install` exposes; additional scopes wait for a use case.
- Changing the no-scope output to advertise the global form. Discovery
  happens via `--help`; the success message stays terse.
- Removing or altering existing `permissions.allow` entries during
  the merge. The contract from the parent spec (append-if-missing,
  no reorder) is preserved.
- Refactoring `printCodexTrustHint` beyond accepting the optional
  scope and emitting the note. The codex flow remains
  instructional-only.

## Risks

- **Stale `~/.claude/settings.json` schema.** If Anthropic changes the
  Claude Code settings schema between when the user last ran Claude
  Code and now, the parse can fail. Mitigation: the existing
  `wireClaudePermissions` returns a wrapped parse error and refuses to
  overwrite — this is correct; surface it cleanly through trust.
- **Race with Claude Code writing settings.json.** If Claude Code is
  open and rewrites settings.json between our read and our write, our
  write wins and may drop a concurrent change. Mitigation: documented
  risk; same as `hero install global` already has today. No file lock
  introduced here.
- **Test harness `HOME` redirection.** `newTestEnvEmpty` may or may
  not redirect `$HOME`. Need to verify before relying on it for global
  tests; if it doesn't, wrap the global tests with explicit
  `t.Setenv("HOME", ...)`. Check
  `internal/cli/testing.go` (or wherever `newTestEnvEmpty` lives)
  during implementation.
- **Codex scope note as breaking change.** Existing automation that
  greps `hero trust codex` output for an exact string match could
  break when the optional scope note is appended. Mitigation: only
  emit the note when a scope arg is explicitly passed — the no-arg
  invocation (today's contract) is byte-equivalent.
- **`Use` string rendering in older cobra.** Verify cobra renders
  the new optional-positional bracket form correctly in `--help`. If
  not, fall back to `trust <codex|claude> [scope]` and document
  values in `Short`/`Long`.
