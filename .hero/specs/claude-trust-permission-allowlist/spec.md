---
title: Claude Trust Permission Allowlist — Stop Per-Command Prompt Fatigue
slug: claude-trust-permission-allowlist
type: bug
status: completed
severity: medium
priority: high
tags: [install, trust, claude, permissions, onboarding]
created: 2026-05-12
completed_at: 2026-05-18T19:25:38Z
---

## Kickoff

Stops Claude Code prompt-fatigue after `hero install --target claude` by
writing `Bash(hero:*)` into `.claude/settings.json`'s `permissions.allow`,
and makes `hero trust claude` apply / report the same entry on demand.

**Status:** delivering — code landed, full suite + vet green, manual
smokes pass. Ready for `hero spec complete` after commit.

**Pick up at:** confirm the commit looks right and push. If something
looks off, the fix is contained to four files and trivially revertable:
`internal/install/claude_hooks.go` (new `wireClaudePermissions` +
`EnsureClaudeHeroAllowlist`), `internal/install/target_claude.go` (one
new call), `internal/cli/trust.go` (claude case), and the two test
files. No infrastructure or schema changes.

**What changed:**
- `internal/install/claude_hooks.go`: added `heroAllowlistEntry`
  constant, `wireClaudePermissions(opts, result) (added bool, err)`
  mirroring `wireClaudeHooks` shape (read → idempotent mutate → write),
  and the exported `EnsureClaudeHeroAllowlist(projectDir)` wrapper for
  the CLI.
- `internal/install/target_claude.go`: calls `wireClaudePermissions`
  right after `wireClaudeHooks` in `runClaude` with the same
  warning-on-error pattern.
- `internal/cli/trust.go`: `case "claude"` resolves project root via
  `findProjectRoot()` and calls `EnsureClaudeHeroAllowlist`; prints
  added/already-present message; default case now lists both
  supported targets.
- `internal/cli/trust_test.go`: replaced `TestTrustUnsupportedTarget`
  (which cemented the bug) with `TestTrustUnknownTarget` against
  `vscode`. Added `TestTrustClaudeAppliesAllowlist` and
  `TestTrustClaudeIdempotent`.
- `internal/install/claude_hooks_test.go`: added
  `TestWireClaudePermissions_FromEmpty`,
  `TestEnsureClaudeHeroAllowlistIdempotent` (3 calls, single entry,
  preserves user `Bash(*)`),
  `TestWireClaudePermissions_PreservesOtherKeys`, and
  `TestInstallClaudeWritesHeroAllowlist`.

**Validation done:**
- `go test ./...` full suite passes
- `go vet ./...` clean
- Manual: fresh `hero init && hero install --target claude` writes
  `Bash(hero:*)` to `.claude/settings.json` `permissions.allow`
- Manual: `hero trust claude` on the same project reports "already
  present"
- Manual: edit settings.json to drop the entry but leave a user entry
  (`Bash(other:thing)`); `hero trust claude` re-adds Hero's entry and
  preserves the user entry (`['Bash(other:thing)', 'Bash(hero:*)']`)
- Manual: `hero trust vscode` errors with both supported targets
  listed

→ `hero spec complete .hero/planning/bugs/claude-trust-permission-allowlist/spec.md`

**Files:** `internal/cli/trust.go`, `internal/cli/trust_test.go`,
`internal/install/claude_hooks.go`, `internal/install/target_claude.go`,
`internal/install/claude_hooks_test.go`
**Skip:** extending the trust flow to opencode/cursor/copilot — those
need their own investigation of each tool's permission model (call it
out as a follow-up if `hero trust opencode` is the next user surprise).

## Issue

After `hero install --target claude`, Claude Code prompts for permission every
time the agent runs a `hero` command — `hero status`, `hero list`, `hero pull`,
etc. The expected setting that would suppress these prompts (a `Bash(hero:*)`
entry under `permissions.allow` in `.claude/settings.json`) is never written.

The companion `hero trust claude` command — which by symmetry with `hero trust
codex` users expect to handle this — errors out:

```
$ hero trust claude
Error: unsupported trust target "claude"; supported targets: codex
```

**Reproduction:**

1. Fresh project directory.
2. `hero init && hero install --target claude`.
3. Open Claude Code in the project; ask the agent to run any `hero ...` command.
4. Observe: Claude Code prompts to approve `Bash(hero ...)` every time.
5. Run `hero trust claude` from a shell.
6. Observe: command errors out instead of fixing the allowlist.

**Asymmetry:** `hero install --target claude` *already* writes
`.claude/settings.json` for `Stop` + `PreCompact` hooks via `wireClaudeHooks`.
The install pathway has the file open and is mutating it — it just stops short
of adding the permissions entry. Codex is a deliberately different shape:
Codex owns the approval state, so Hero can only print a hint. Claude Code
stores its permissions in `.claude/settings.json`, which Hero already writes;
therefore Hero can and should manage it directly.

## Investigation

### Code path: `hero trust claude`

`internal/cli/trust.go` lines 9–24:

```go
var trustCmd = &cobra.Command{
	Use:   "trust <codex>",
	Short: "Show one-time harness permission guidance for Hero",
	Args:  cobra.ExactArgs(1),
	RunE:  runTrust,
}

func runTrust(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "codex":
		printCodexTrustHint()
		return nil
	default:
		return fmt.Errorf("unsupported trust target %q; supported targets: codex", args[0])
	}
}
```

The switch only handles `codex`; every other value (including `claude`)
falls into the error path.

### Codified gap: `TestTrustUnsupportedTarget`

`internal/cli/trust_test.go` lines 21–31 asserts the wrong contract — that
`hero trust claude` *should* fail:

```go
func TestTrustUnsupportedTarget(t *testing.T) {
	_ = newTestEnvEmpty(t)
	_, err := runCmd("trust", "claude")
	if err == nil {
		t.Fatal("trust claude should fail")
	}
	if !strings.Contains(err.Error(), `unsupported trust target "claude"; supported targets: codex`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

This test currently cements the bug. It needs to be repurposed against a
target that really is unsupported (e.g. `vscode`).

### Install-side gap: `wireClaudeHooks` covers hooks, not permissions

`internal/install/claude_hooks.go` `wireClaudeHooks` (lines 35–80):

- Reads `.claude/settings.json` if present (parses JSON, errors on
  malformed input).
- Treats missing file as empty map.
- Walks `claudeHookEvents` (`Stop`, `PreCompact`) and upserts a hero entry
  via `upsertHeroEntry`, which preserves user-owned entries.
- `MarshalIndent`s the map and writes it back, creating the directory if
  needed.

It never touches `settings["permissions"]`. The pattern (read → mutate one
slice idempotently → write) is exactly what the permissions fix needs.

Call site: `internal/install/target_claude.go` line 75 invokes
`wireClaudeHooks(opts, result)` near the end of `runClaude`. The
permissions wiring belongs right next to it.

### Settings format — confirmed from existing fixtures

`internal/install/claude_hooks_test.go` lines 80–92
(`TestWireClaudeHooks_PreservesUserContent`) and lines 142–154
(`TestUnwireClaudeHooks_RemovesHeroOnly`) both seed:

```json
{
  "permissions": {"allow": ["Bash(*)"]},
  ...
}
```

So `permissions.allow` is an array of strings under a top-level
`permissions` object. The Hero entry we need is `"Bash(hero:*)"`.
`TestWireClaudeHooks_PreservesUserContent` already proves the current code
preserves the `permissions` key untouched when wiring hooks — i.e. our
upsert just needs to read the existing array, append if missing, and write
back.

### Mode handling

`claudeSettingsPath` (lines 190–206) already resolves the right path for
both project mode (`<target>/.claude/settings.json`) and global mode
(`~/.claude/settings.json`). The permissions wiring can reuse it as-is.

### Root cause classification

**Missing-feature regression.** The install pathway grew to cover hooks
(commit history shows `wireClaudeHooks` was added to fix cross-session
handoffs) but the permissions entry was never extended in lockstep, even
though it lives in the same file the installer already writes. Meanwhile
the `trust` command was authored for codex's instructional-only shape and
the unhandled-target test (`TestTrustUnsupportedTarget`) cemented the
gap so it didn't surface during refactors. Neither file is buggy in
isolation; together they leave the user paying the same approval cost
every turn.

## Goal

Running either `hero install --target claude` or `hero trust claude` leaves
`.claude/settings.json`'s `permissions.allow` containing `"Bash(hero:*)"`,
without disturbing existing entries, and Claude Code no longer prompts for
each `hero` command.

## Changes

1. Add `wireClaudePermissions` in `internal/install/claude_hooks.go`.
   - Mirror the shape of `wireClaudeHooks`: resolve settings path via
     `claudeSettingsPath(opts)`; honor `opts.DryRun` with a progress line;
     load existing settings.json (treat missing file as empty map; return
     a wrapped error on malformed JSON).
   - Read `settings["permissions"]` as `map[string]interface{}`; if absent,
     create it. Read `permissions["allow"]` as `[]interface{}`; if absent,
     create it.
   - Idempotent append: scan the existing entries for the literal string
     `"Bash(hero:*)"`; if not found, append it. Never remove or reorder
     other entries.
   - Marshal with `json.MarshalIndent`, ensure parent dir exists, write
     `0o644`. Append a trailing newline (match `wireClaudeHooks`).
   - Append the settings path to `result.Merged` only if it wasn't already
     recorded by `wireClaudeHooks` in the same run — or, simpler, always
     append (the result is informational; duplicates are harmless). Pick
     whichever pattern the existing `Result` consumers already tolerate.
   - Return a small struct or a `bool` indicating whether the entry was
     newly added vs. already present. Needed so `hero trust claude` can
     report status; not used by the install path.

2. Call `wireClaudePermissions` from `runClaude` in
   `internal/install/target_claude.go`.
   - Invoke it after `wireClaudeHooks(opts, result)` (line 75 area). Use
     the same warning-on-error pattern (`fmt.Printf("  warning: could not
     wire claude permissions: %v\n", err)` rather than failing the whole
     install).

3. Add `EnsureClaudeHeroAllowlist(projectDir string) (added bool, path
   string, err error)` (or similar shaped exported helper) in the install
   package, building `Options{Mode: ModeProject, TargetDir: projectDir}`
   and invoking the internal wiring. Keeps the CLI layer free of
   filesystem detail. Global-mode trust is out of scope (see Boundaries).

4. Extend `internal/cli/trust.go` with `case "claude"`.
   - Resolve the project directory via the same helper the rest of the CLI
     uses (look for the pattern in nearby commands — likely `projectRoot`
     or `installRoot` in this package). Pass to `EnsureClaudeHeroAllowlist`.
   - Print one of two messages depending on the `added` return:
     - `"Claude Code: added Bash(hero:*) to <path> — prompts should stop after Claude Code reloads settings."`
     - `"Claude Code: Bash(hero:*) already present in <path>. No change needed."`
   - Leave `case "codex"` unchanged.
   - Default case continues to return `unsupported trust target %q; supported targets: codex, claude`.

5. Update the cobra `Use` string from `"trust <codex>"` to `"trust <codex|claude>"`.

6. Test changes:
   - `internal/cli/trust_test.go`:
     - **Replace** `TestTrustUnsupportedTarget` with `TestTrustUnknownTarget`
       that invokes `runCmd("trust", "vscode")` and asserts the error
       message lists both supported targets.
     - **Add** `TestTrustClaudeAppliesAllowlist`: create a temp project,
       run `runCmd("trust", "claude")`, assert
       `.claude/settings.json` ends up with `Bash(hero:*)` in
       `permissions.allow` and stdout contains the "added" message.
     - **Add** `TestTrustClaudeIdempotent`: pre-write a settings.json that
       already includes `Bash(hero:*)`, run `hero trust claude`, assert no
       duplicate entry and stdout contains the "already present" message.
   - `internal/install/claude_hooks_test.go`:
     - **Add** `TestInstallClaudeWritesHeroAllowlist`: run `wireClaudeHooks`
       + `wireClaudePermissions` on an empty dir and verify
       `permissions.allow` contains exactly `["Bash(hero:*)"]`.
     - **Add** `TestEnsureClaudeHeroAllowlistIdempotent`: call the wiring
       three times against a settings.json that also contains
       `"Bash(*)"`; assert the final `permissions.allow` is exactly
       `["Bash(*)", "Bash(hero:*)"]` (order preserved, no duplicates).
     - **Add** `TestWireClaudePermissions_PreservesOtherKeys`: seed
       settings.json with `model`, `hooks`, and other `permissions.allow`
       entries; confirm everything except the new entry is byte-equivalent
       after the wiring.

## Boundaries

- Do NOT change the `hero trust codex` hint or behavior. Codex still owns
  its own approval state.
- Do NOT extend trust/permissions wiring to `opencode`, `cursor`, or
  `copilot` in this spec. Each has its own settings shape and warrants
  separate diagnosis. Track follow-ups separately.
- Do NOT remove or reorder existing entries in `permissions.allow`. The
  only mutation is an append-if-missing of `"Bash(hero:*)"`.
- Do NOT introduce a cross-tool generalized permission helper. Keep the
  code shape close to `wireClaudeHooks` until a second concrete consumer
  arrives.
- Do NOT widen scope to global-mode trust (`~/.claude/settings.json`). The
  CLI helper is project-scoped to match the predominant `hero install`
  pattern; a global-mode follow-up can be a separate spec.
- Do NOT change the cobra command's `Args` validation beyond updating the
  `Use` string for help text.

## Risks

- **User has hand-edited `permissions.allow`.** Mitigation: read-modify-write
  the array; only append the Hero entry; preserve order and other entries
  byte-equivalent. The existing
  `TestWireClaudeHooks_PreservesUserContent` proves the surrounding code
  already preserves the `permissions` key when it's not the one being
  mutated — the new test must prove the same for entries inside the
  array.
- **Settings file does not exist.** Mitigation: mirror `wireClaudeHooks`
  behavior — treat missing file as an empty map, create the directory,
  write a new file with just the keys Hero added. The
  `TestWireClaudeHooks_CreatesSettingsFromScratch` pattern applies here.
- **Malformed JSON in settings.json.** Mitigation: return a wrapped error
  rather than overwriting the user's broken-but-intentional file.
  `wireClaudeHooks` already does exactly this; reuse the pattern.
- **Claude Code does not hot-reload settings.json.** Users may see
  prompts persist for one session. Surface this in the "added" message
  ("...prompts should stop after Claude Code reloads settings.") so the
  user knows to restart.
- **Other harness allowlist gaps.** Users who use both Claude Code and
  another harness (opencode, cursor, copilot) may notice the asymmetry
  ("why does Claude not prompt anymore but Cursor still does?"). Surface
  this as a follow-up spec rather than scope creep here.
- **Test churn.** Replacing `TestTrustUnsupportedTarget` flips the
  meaning of an existing assertion. Reviewers should confirm the
  replacement test still exercises the "truly unknown target" branch.

## Validation

- `go test ./internal/install/... ./internal/cli/...` passes, including
  the new and replaced tests.
- Manual smoke 1 — fresh install:
  1. `mktemp -d` and `cd` into it.
  2. `hero init && hero install --target claude`.
  3. `jq '.permissions.allow' .claude/settings.json` includes
     `"Bash(hero:*)"`.
- Manual smoke 2 — trust on existing install:
  1. From the same dir, remove the entry: `jq '.permissions.allow |= map(select(. != "Bash(hero:*)"))' .claude/settings.json > tmp && mv tmp .claude/settings.json`.
  2. `hero trust claude` — output reports the entry was added.
  3. Re-run `hero trust claude` — output reports no change needed; file
     unchanged (`shasum` matches).
- Manual smoke 3 — preserves user entries:
  1. Edit `.claude/settings.json`, add a custom entry like
     `"Bash(rg:*)"` to `permissions.allow`.
  2. `hero trust claude`. Custom entry remains; `Bash(hero:*)` present.
- Manual smoke 4 — Claude Code behavior:
  1. After reloading Claude Code on the project, ask the agent to run
     `hero status`. The agent should run it without a permission prompt.

## Acceptance Criteria

- WHEN `hero install --target claude` runs THE SYSTEM SHALL ensure
  `permissions.allow` in `.claude/settings.json` contains
  `"Bash(hero:*)"`.
- WHEN `hero trust claude` runs and the `"Bash(hero:*)"` entry is missing
  THE SYSTEM SHALL add it and report that it was applied.
- WHEN `hero trust claude` runs and the `"Bash(hero:*)"` entry is already
  present THE SYSTEM SHALL report that no change was needed and SHALL NOT
  add a duplicate entry.
- IF `.claude/settings.json` contains other allowlist entries THEN THE
  SYSTEM SHALL preserve them when adding the Hero entry, in their
  original order.
- IF `.claude/settings.json` does not exist when `hero trust claude` or
  `hero install --target claude` runs THEN THE SYSTEM SHALL create it
  with `"Bash(hero:*)"` in `permissions.allow` and no unrelated keys,
  mirroring `wireClaudeHooks` behavior.
- IF `.claude/settings.json` contains malformed JSON THEN THE SYSTEM
  SHALL return a wrapped parse error and SHALL NOT overwrite the file.
- THE SYSTEM SHALL keep `hero trust codex` behavior unchanged —
  instructional output only, no `.codex` or `.claude` settings mutation.
- THE SYSTEM SHALL reject truly unsupported trust targets (e.g.
  `vscode`) with an error listing all supported targets.
