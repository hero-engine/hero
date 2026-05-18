---
title: Scan output cleanup — three first-five-minutes papercuts
slug: scan-output-cleanup
type: bug
status: completed
severity: low
priority: medium
tags: [scan, cli, ux, onboarding]
created: 2026-05-12
---

# Scan output cleanup — three first-five-minutes papercuts

## Kickoff

Cleans up three small but confusing things `hero scan` prints (or
quietly does) on first contact — the cryptic memory line, the
surprise pre-commit hook install, and the "Updated: 5" line that
reads like data loss.

**Status:** delivering — code landed, full suite + vet green, all 6
manual smoke scenarios pass. Ready for `hero spec complete` after
commit.

**Pick up at:** confirm the commit looks right and push. If something
looks off, the fix is contained to two files plus tests:
`internal/cli/scan.go` (memory step + Updated explanation + hook
install block removed), `internal/cli/init.go` (hook install + two
new flags), `internal/cli/init_test.go` + `internal/cli/scan_test.go`
(eight new tests).

**What changed:**
- `internal/cli/scan.go`: memory step renamed to `claude-memory`,
  conditionally emitted only when the directory exists, friendly
  skip reason for empty-but-present. Pre-commit hook auto-install
  block removed. `--no-hooks` kept as a no-op flag with a
  deprecation note in `--help`. New explanation line printed when
  `Updated > 0 && Skipped == 0 && Created == 0`.
- `internal/cli/init.go`: added `--install-hooks` (default true) and
  `--no-hooks` flags, plus the hook install block (moved from scan)
  with the same best-effort error pattern.
- `internal/cli/init_test.go`: `TestInitInstallsPreCommitHook`,
  `TestInitNoHooksFlag`, `TestInitNoGitNoHook`.
- `internal/cli/scan_test.go`: `TestScanDoesNotInstallPreCommitHook`,
  `TestScanNoHooksFlagAccepted`,
  `TestScanOmitsClaudeMemoryStepWhenAbsent`,
  `TestScanEmitsFriendlyClaudeMemoryWhenEmpty`,
  `TestScanExplainsUncustomizedUpdates`.

**Validation done:**
- `go test ./...` full suite passes
- `go vet ./...` clean
- Manual: `hero init` in a fresh git repo installs the hook with one
  confirmation line; `.git/hooks/pre-commit` exists
- Manual: `hero scan` in a git repo with no hook does NOT install
  one and does NOT print the hook line
- Manual: `hero scan` in a project where the claude memory dir
  doesn't exist prints no claude-memory line at all
- Manual: second `hero scan` prints "Updated entries hadn't been
  customized — they regenerate cleanly. Hand-edits are preserved on
  future scans."
- Manual: `hero scan --no-hooks` still accepted (no error)
- Manual: `hero scan --help` shows the deprecation note for
  `--no-hooks`

→ `hero spec complete .hero/planning/bugs/scan-output-cleanup/spec.md`

**Files:** `internal/cli/scan.go`, `internal/cli/init.go`,
`internal/cli/init_test.go`, `internal/cli/scan_test.go`

## Issue

Three independent UX papercuts share the surface of `hero scan` output. None is a functional bug; together they make the first five minutes of using Hero feel rough, especially for new users.

### Issue 1 — Cryptic `memory` step

`hero scan` always emits a step for the per-project Claude Code memory ingest, even when there is no memory dir at all. The skip reason is a raw path with the slashes encoded as dashes (a Claude Code convention), which looks like a broken filename:

```
⊘ memory:      /Users/<me>/.claude/projects/-Users-<me>-projects-<project>/memory not present or empty
```

Three problems in one line:

1. The label `memory` does not say *whose* memory — sounds like RAM or like Hero's own memory store.
2. The path with dash-encoded segments looks corrupted to anyone who has not seen the Claude Code project-key convention.
3. The step prints on every scan even when the directory has never existed — pure noise for projects that have not done any Claude Code work yet.

### Issue 2 — Pre-commit hook silently installed by `hero scan`

`hero scan` writes `.git/hooks/pre-commit` on first run (when a git repo is detected and `--no-hooks` was not passed). Users who run `hero scan` for stack detection do not expect a scan command to modify their git hooks. The mitigation note ("Pass --no-hooks next time to skip") arrives *after* the side effect.

Reproduction: in a fresh project, `hero init && hero scan` produces the unexpected `Installed pre-commit hook ...` lines.

### Issue 3 — `Updated: 5` reads like data loss on re-scan

After re-running `hero scan` on a project where no scan-generated entries have been hand-edited yet, the output reports several entries "updated" with no explanation:

```
Created: 0, Updated: 5, Skipped (customized): 0
```

The merge logic correctly preserves customizations (`internal/scan/merge.go:60-99`) — only auto-generated, unmodified entries are updated, and any user-customized entry is skipped. But the output does not communicate that. A new user re-running the command can reasonably worry their work was overwritten.

## Investigation

File/line evidence confirmed by reading the source:

### Issue 1 evidence — `internal/cli/scan.go:582-594`

```go
// Memory ingest.
memDir := memory.DirForProject(projectRoot)
if memSummary, err := memory.WriteGraph(memDir, repoKey, store); err != nil {
    report.add(stepResult{name: "memory", failed: true, err: err})
} else if memSummary.Files > 0 {
    report.add(stepResult{
        name:   "memory",
        ok:     true,
        detail: fmt.Sprintf("%d files (scope: local)", memSummary.Files),
    })
} else {
    report.add(stepResult{name: "memory", skipped: true, reason: memDir + " not present or empty"})
}
```

`memory.DirForProject` (`internal/memory/graph_ingest.go:31-42`) returns `~/.claude/projects/<dashed-abs-path>/memory`, which is precisely the path encoding the user is reading as broken. `memory.WriteGraph` (`internal/memory/graph_ingest.go:51-64`) already treats a missing dir as "zero summary, no error" — it does not surface whether the dir existed. We need a presence check at the call site so the report can suppress the step entirely when there is no memory dir at all.

### Issue 2 evidence — `internal/cli/scan.go:90-105`

```go
// Auto-install the pre-commit hook so projected NEXT files travel
// with commits. Skipped in dry-run, when --no-hooks is set, when
// not in a git repo, or when already installed ...
if !scanDryRun && !scanNoHooks && !preCommitHookInstalled(projectRoot) {
    if _, err := resolveGitDir(projectRoot); err == nil {
        if err := installNextHooksQuiet(projectRoot); err != nil {
            fmt.Fprintf(os.Stderr, "Warning: pre-commit hook install failed: %v\n", err)
        } else {
            fmt.Println("Installed pre-commit hook (projected NEXT files will travel with commits).")
            fmt.Println("  Pass --no-hooks next time to skip; to remove, delete the marker block in .git/hooks/pre-commit.")
        }
    }
}
```

The `--no-hooks` flag is declared at `internal/cli/scan.go:35,70`. The helpers (`preCommitHookInstalled`, `installNextHooksQuiet`, `resolveGitDir`) live in `internal/cli/next_hooks.go` and are reusable from `runInit`. `runInit` (`internal/cli/init.go:40-145`) is the natural home for first-time-setup side effects and already prints a setup banner.

### Issue 3 evidence — `internal/cli/scan.go:166-175` and `internal/scan/merge.go:54-99`

```go
fmt.Printf("\nCreated: %d, Updated: %d, Skipped (customized): %d",
    mergeResult.Created, mergeResult.Updated, mergeResult.Skipped)
if mergeResult.Forced > 0 {
    fmt.Printf(", Forced: %d", mergeResult.Forced)
}
fmt.Println()

if mergeResult.Skipped > 0 && !scanForce {
    fmt.Println("Use --force to overwrite customized entries.")
}
```

The merge logic at `internal/scan/merge.go:54-99` is correct: `MergeUpdate` is only chosen when `isAutoGenerated && !isUserCustomized` (or `isImported && !isUserCustomized`); any customized file falls through to `MergeSkipCustomized`. So when `Updated > 0` and `Skipped == 0`, the merge engine is telling us no entries had been hand-edited — exactly the moment to reassure the user.

### Root cause classification

UX-papercut-by-omission across three locations in `internal/cli/scan.go`. None is a functional bug:

- Issue 1: missing presence check + ungrokkable label/reason.
- Issue 2: setup-time side effect bolted onto the wrong command.
- Issue 3: missing one-line reassurance after a correct-but-scary count.

The underlying systems (`memory.WriteGraph`, `installNextHooksQuiet`, `scan.PlanMerge`) all work as designed; only the rendering/placement needs to change.

### Severity

Low. Nothing is broken; the workspace is in a correct state after every scan. The impact is onboarding friction and a perception of risk on re-runs. Blast radius is "every new Hero user the first time they run `hero scan`."

## Goal

`hero scan` produces output a brand-new user can read top-to-bottom without feeling confused or alarmed. The Claude Code memory step only appears when there's something meaningful to say about it. Git hook installation happens at `hero init` where setup belongs, not as a side effect of `hero scan`. When `hero scan` reports updates with zero customized-skips, it says — in one line — that this is the expected idempotent behavior and hand-edits are preserved on future runs.

## Changes

### 1. Relabel and conditionally emit the Claude memory step

File: `internal/cli/scan.go` around lines 582-594.

Replace the current block with logic that:

- Calls `memory.DirForProject(projectRoot)` and stats the result before calling `memory.WriteGraph`.
- If `memDir == ""` or the directory does not exist (`os.IsNotExist`), skips emitting any step at all (no `report.add` call).
- If the directory exists, calls `memory.WriteGraph` as today, but emits the step under the name `claude-memory` rather than `memory`, and uses a plain-English skip reason instead of a raw path.

New strings, verbatim:

- Step name (all three branches): `"claude-memory"`.
- Skipped-reason when the dir exists but is empty: `"Claude Code memory store for this project is empty — Hero will pull from it automatically as you accumulate memories."`
- Failure branch keeps `failed: true, err: err` as today.
- Success branch keeps the `%d files (scope: local)` detail format as today.

Sketch:

```go
memDir := memory.DirForProject(projectRoot)
if memDir != "" {
    if _, statErr := os.Stat(memDir); statErr == nil {
        if memSummary, err := memory.WriteGraph(memDir, repoKey, store); err != nil {
            report.add(stepResult{name: "claude-memory", failed: true, err: err})
        } else if memSummary.Files > 0 {
            report.add(stepResult{
                name:   "claude-memory",
                ok:     true,
                detail: fmt.Sprintf("%d files (scope: local)", memSummary.Files),
            })
        } else {
            report.add(stepResult{
                name:    "claude-memory",
                skipped: true,
                reason:  "Claude Code memory store for this project is empty — Hero will pull from it automatically as you accumulate memories.",
            })
        }
    }
    // dir does not exist — emit nothing; this is the common case for new projects.
}
```

Add tests in `internal/cli/scan_test.go`:

- `TestScanOmitsClaudeMemoryStepWhenAbsent` — point `HOME` at a temp dir with no `.claude/projects/...` subtree, run the scan path, assert no step with name `claude-memory` is emitted.
- `TestScanEmitsFriendlyClaudeMemoryWhenEmpty` — create the memory dir empty, assert a skipped step named `claude-memory` with the new reason string and no raw path substring.

### 2. Move pre-commit hook auto-install from `hero scan` to `hero init`

Files: `internal/cli/scan.go:30-105` and `internal/cli/init.go:30-145`.

In `internal/cli/init.go`:

- Add two flags on `initCmd`:
  - `initInstallHooks` (bool, default `true`, flag `--install-hooks`, help: `"install the pre-commit hook so projected NEXT files travel with commits"`).
  - `initNoHooks` (bool, default `false`, flag `--no-hooks`, help: `"skip installing the pre-commit hook"`).
- Inside `runInit`, after the workspace directory structure is created (just before or after the AGENTS.md block), insert a block equivalent to the current scan-time install:

```go
if initInstallHooks && !initNoHooks && !preCommitHookInstalled(projectRoot) {
    if _, err := resolveGitDir(projectRoot); err == nil {
        if err := installNextHooksQuiet(projectRoot); err != nil {
            fmt.Fprintf(os.Stderr, "  warning: pre-commit hook install failed: %v\n", err)
        } else {
            fmt.Println("\n  Installed pre-commit hook (projected NEXT files will travel with commits).")
            fmt.Println("  Pass --no-hooks next time to skip; to remove, delete the marker block in .git/hooks/pre-commit.")
        }
    }
}
```

In `internal/cli/scan.go`:

- Delete the entire auto-install block at lines 90-105 (the comment and the `if` block).
- Keep the `scanNoHooks` variable and the `--no-hooks` flag declaration so existing scripts continue to work, but change the flag help string to mark it deprecated. New help string verbatim: `"deprecated no-op — hook install moved to 'hero init'; kept for backwards compatibility"`.
- The flag becomes a true no-op: nothing reads `scanNoHooks` after this change.

Add tests:

- `TestInitInstallsPreCommitHook` — in `internal/cli/init_test.go` (create the file if absent; check first): run `runInit` in a temp git repo with default flags, assert `.git/hooks/pre-commit` exists and contains the hero marker block.
- `TestInitNoHooksFlag` — pass `--no-hooks`, assert no pre-commit hook is written.
- `TestScanDoesNotInstallPreCommitHook` — in `internal/cli/scan_test.go`: run the scan path in a temp git repo where `hero init` was *not* called with hooks, assert `.git/hooks/pre-commit` is not created by scan.
- `TestScanNoHooksFlagAccepted` — pass `--no-hooks` to scan, assert the flag is recognized and the command exits without error (backwards-compat).

### 3. Explain idempotent updates on `hero scan` re-runs

File: `internal/cli/scan.go` around lines 166-175.

After the `Created: %d, Updated: %d, Skipped (customized): %d` line and the existing `Use --force to overwrite customized entries.` branch, add a one-line explanation when the user is most likely to be alarmed: `Updated > 0` AND `Skipped == 0` AND `Created == 0` (so first-run is excluded — on first run all entries are Created, never Updated).

New line, verbatim: `"Updated entries hadn't been customized — they regenerate cleanly. Hand-edits are preserved on future scans."`

Sketch (insert after the existing `if mergeResult.Skipped > 0 && !scanForce` block):

```go
if mergeResult.Updated > 0 && mergeResult.Skipped == 0 && mergeResult.Created == 0 {
    fmt.Println("Updated entries hadn't been customized — they regenerate cleanly. Hand-edits are preserved on future scans.")
}
```

Note on the gating: using `Created == 0` as the "this is a re-run" signal is natural — on first scan everything goes through `MergeCreate`, never `MergeUpdate`, so `Updated > 0` already implies a re-run. The extra `Created == 0` guard avoids the awkward mixed-output case where some entries were newly created and others updated (e.g. user added a new framework to the project).

Add test:

- `TestScanExplainsUncustomizedUpdates` — drive the scan twice on a fixture project, capture stdout, assert the new sentence appears on the second run and is absent on the first run. Then touch one of the generated entries with a user customization marker, run scan a third time, assert `Skipped (customized)` increments and the explanation line is absent.

### 4. Update any existing tests that grep for old strings

Search `internal/cli/scan_test.go` (and adjacent test files) for assertions on:

- The literal `"memory"` step name in scan reports — update to `"claude-memory"`.
- The string `"Installed pre-commit hook"` printed by `hero scan` — move those expectations to init tests, drop from scan tests.
- The raw path substring `not present or empty` — remove or replace with the new plain-English assertion.

Initial source check shows no scan_test.go assertions on these strings today (the file exists but does not reference `memory`, `pre-commit`, or `hooks` per grep), but verify during delivery in case other test files added coverage.

## Boundaries

- Do NOT change the pre-commit hook content itself (`internal/cli/next_hooks.go` install logic stays as-is).
- Do NOT change the merge logic in `internal/scan/merge.go` — only the rendering of merge results changes.
- Do NOT change `memory.DirForProject` or the path-encoding convention — only the scan-side call site decides whether to render a step.
- Do NOT remove `--no-hooks` from `hero scan`. Keep it as a deprecated no-op flag for backwards compatibility with existing scripts.
- Do NOT touch the `hero check` reference to `preCommitHookInstalled` at `internal/cli/check.go:193`; that diagnostic stays useful.
- Do NOT add a `verbose` / `--debug` path that re-shows the suppressed `claude-memory` step. Out of scope for this spec.

## Risks

- **Behavioral change for existing users.** A small number of users may rely on `hero scan` installing the hook on second-init'd projects, or on the line appearing in scan output as confirmation. After this change they will need to run `hero init` (which is idempotent for the workspace directory creation; but it currently returns an error if `.hero/` already exists — see next bullet) or `hero next install-hooks` explicitly. The deprecation note in `hero scan --help` is the user-visible signal.
- **`hero init` idempotency.** `runInit` returns `fmt.Errorf("hero workspace already exists at %s", heroDir)` when `.hero/` already exists (`internal/cli/init.go:49-51`). That means a user with an existing workspace cannot just re-run `hero init` to pick up the hook. During delivery, decide between (a) adding a small "install hooks even if workspace exists" path in `runInit`, or (b) directing users to `hero next install-hooks` for after-the-fact installs. Pick (b) for minimum surface — the new `--install-hooks` / `--no-hooks` flags only affect the fresh-init flow.
- **Test fixtures.** Any test (or doc snippet) that greps scan output for `memory` or for the hook-install line will break. Sweep is small (initial grep shows zero matches in `internal/cli/scan_test.go`), but the delivery agent must verify across all `_test.go` files.
- **Conditional emission and `--debug` users.** Hiding the `claude-memory` line in the common case means a power user debugging "did the memory step run?" loses the signal. The decision is acceptable for now; if it bites, a future spec can add a verbose mode. Out of scope here.

## Validation

Automated:

- `go test ./internal/cli/... ./internal/scan/... ./internal/memory/...` passes.
- New tests listed in `## Changes` all pass.

Manual:

- Fresh project, fresh `.git/hooks/`:
  - `hero init` installs the hook and prints the single confirmation line.
  - `hero scan` no longer prints the hook install line and does not touch `.git/hooks/`.
- `hero scan` in a project where `~/.claude/projects/<encoded>/memory/` does **not** exist prints **no** `claude-memory` line at all.
- `hero scan` in a project where the memory dir **exists but is empty** prints the friendly skip message with the new `claude-memory` label and no raw path substring.
- Second `hero scan` on a project where no scan-generated entries have been hand-edited prints `Created: 0, Updated: <n>, Skipped (customized): 0` followed by `Updated entries hadn't been customized — they regenerate cleanly. Hand-edits are preserved on future scans.`
- After touching one of the generated entries (adding a user-edit marker recognised by `isUserCustomized`), a third `hero scan` reports the entry as `Skipped (customized)` and **does not** print the explanation line.
- `hero scan --no-hooks` exits 0 and prints no error; `hero scan --help` shows the deprecated note on the flag.

## Acceptance Criteria

- WHEN `hero scan` runs on a project where the Claude Code memory directory does not exist THE SYSTEM SHALL omit the `claude-memory` step from the report entirely.
- WHEN `hero scan` runs on a project where the Claude Code memory directory exists but is empty THE SYSTEM SHALL emit a step labeled `claude-memory` with a plain-English skip reason and no raw path in the message.
- WHEN `hero init` runs in a git repository without `--no-hooks` THE SYSTEM SHALL install the pre-commit hook and print a single confirmation line.
- WHEN `hero scan` runs THE SYSTEM SHALL NOT install or modify any git hooks.
- IF `hero scan` is invoked with `--no-hooks` THEN THE SYSTEM SHALL accept the flag without error and `hero scan --help` SHALL describe it as deprecated.
- WHEN `hero scan` reports a merge result with `Updated > 0` AND `Skipped (customized) == 0` AND `Created == 0` THE SYSTEM SHALL print a one-line explanation that no entries had been user-customized and hand-edits are preserved on future scans.
- WHERE `hero scan` is re-run after a user has edited any auto-generated entry THE SYSTEM SHALL increment `Skipped (customized)` for that entry and suppress the no-customization explanation line.
- WHEN `hero init` is invoked with `--no-hooks` THE SYSTEM SHALL skip the hook install and print no hook-related output.
