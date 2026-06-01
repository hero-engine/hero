---
title: Hero binary upgrades silently strand installations in stale layout
slug: upgrade-strands-install-layout
type: bug
status: completed
severity: high
root_cause_class: design
created: 2026-05-31
tags: [install, upgrade, ux, drift]
completed_at: 2026-05-31T16:17:40Z
---
# Hero binary upgrades silently strand installations in stale layout

## Problem

After a Hero binary upgrade, the on-disk install layout (`.claude/`,
`.opencode/`, `.cursor/`, etc.) is **not** reconciled until the user
manually re-runs `hero install`. When a major-version install
contract change has happened in between (P2 symlink layout → v0.14+
render-direct), the old artifacts left in place can be actively
broken — pointing at directories that no longer exist or no longer
get materialized.

**Concrete manifestation in paperboy**
(`~/projects/personal/repository/paperboy`):

- `hero install` was last run on **2026-05-12** with a `dev` build
  that used the P2 symlink layout. Symlinks were created at:
  - `.claude/agents -> ../.hero/agents`
  - `.claude/commands -> ../.hero/commands`
  - `.claude/skills -> ../.hero/skills`
  - …same trio under `.opencode/`.
- The binary was later upgraded to **v0.14.1**.
- v0.14+ explicitly does NOT use symlinks
  ([internal/install/target_claude.go:18-21](internal/install/target_claude.go))
  and stopped materializing `.hero/{agents,commands,skills}/` as a
  canonical mirror.
- Since `hero install` was never re-run, the symlinks still pointed
  at `.hero/agents`/`.hero/commands`/`.hero/skills` — directories
  that no longer exist.
- Result: every Skill and Agent lookup in any Claude Code session
  opened in paperboy failed silently. `/design`, `/hero`,
  `feature-delivery-lead`, every Hero command/agent: unreachable.

The drift was **visible in state** but never surfaced:
`paperboy/.hero/install-state.json` had
`"hero_version": "v0.14.1-dirty"` at the top while
`"last_install": { "version": "dev", "timestamp": "2026-05-12T…" }`.
Nothing reads or warns on that mismatch.

This violates Hero's core promise — *every session starts as smart
as the last one ended*. Instead, sessions in upgraded-but-not-reinstalled
projects start dumber, with no surface available, and the user has
no signal that the cause is a missed `hero install`.

## Steps to Reproduce

1. In a fresh test project, install hero from an older build (≤v0.13)
   that still uses the P2 symlink layout:
   ```
   cd /tmp/repro && git init && hero install project .  # using older binary
   ```
   Confirm `.claude/agents -> ../.hero/agents` exists as a symlink.
2. Replace the `hero` binary with v0.14.1 (or current `main`).
   Do **not** re-run `hero install`.
3. Open a Claude Code session in `/tmp/repro` and try to use a Hero
   slash command (`/design`) or invoke a Hero agent
   (`feature-delivery-lead`). Both fail to resolve.
4. `cat .hero/install-state.json` shows `hero_version` ≠
   `last_install.version`, but `hero status` / `hero check` are silent.

Fix is a manual `hero install <mode> <path> --migrate` which fires
[cleanupLegacyCanonicalSymlinks](internal/install/cleanup.go#L114)
and renders fresh content.

## Expected Behavior

At minimum, Hero should **detect and surface** the version drift —
loudly, in any command that already inspects workspace state. Ideal
behavior: drift is auto-repaired (cleanup runs + content re-rendered)
the first time a command runs after the binary changes, with a clear
one-line announcement of what happened.

The user must never silently end up with a broken install just
because they updated the binary the normal way (homebrew, `go install`,
download).

## Investigation

### Categorization

| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — silently corrupts every Claude Code/Cursor/opencode session in any project where the user upgraded the `hero` binary but didn't re-run `hero install`. Directly violates the "every session starts as smart as the last one ended" mission. |
| **Ease of Fix** | moderate — detection logic is small (lstat + readlink + version compare). Wiring into `hero check` is straightforward. The judgment calls are (a) whether to also fire on every `hero` command and (b) whether to auto-repair vs nag. |
| **Caused by our codebase?** | Yes — the cleanup path only runs inside `install.Run` (Project mode). No other code path reconciles legacy layout when the binary changes. |
| **Needs more research?** | No — full code flow confirmed; paperboy on-disk evidence (pre- and post-fix `version.json`) reproduces the path. |

### Background

When a Hero binary upgrade crosses a major install-layout boundary (most
recently P2 symlink layout → v0.14 render-direct), any project where
`hero install` is not re-run retains the *old* layout artifacts. The
v0.14 cleanup code that knows how to remove the legacy symlinks
(`cleanupLegacyCanonicalSymlinks`) is gated behind one entry point —
`install.Run` in project mode — so it never fires on routine `hero`
invocations, `hero check`, `hero status`, or `hero upgrade`.

The user's harness (Claude Code) reads `.claude/agents/`,
`.claude/commands/`, `.claude/skills/` directly. After the binary
upgrade those paths are dangling symlinks pointing at
`.hero/{agents,commands,skills}/` — directories v0.14+ no longer
materializes. Every Skill, Agent, and slash command in the project
silently fails to resolve.

### Analysis

There are **three** independent gaps:

1. **Version-drift warning** (`version.Mismatch`) does fire on `hero`
   commands but the suggested remediation is wrong: it tells the user to
   run `hero upgrade`. `hero upgrade` calls `version.StampUpgrade` and
   re-renders content — it **never calls
   `cleanupLegacyCanonicalSymlinks`**. So even a compliant user
   following the warning would still have broken symlinks afterward.
2. **Broken-symlink detection** at root harness paths (e.g.
   `.claude/agents -> ../.hero/agents`) is implemented for satellite
   reconciliation (`install.Repair` → `DriftBrokenSymlink`) but the
   reconciler only walks `satellites.local.json` — root-level harness
   dirs are never inspected. A project with no satellites (the common
   case) has zero coverage.
3. **The cleanup list itself is incomplete.** [`harnessKindPaths` in
   cleanup.go:116-133](internal/install/cleanup.go#L116-L133) lists
   `.codex/skills` but **not `.codex/agents` or `.codex/commands`**.
   Codex installs in the P2 era wrote symlinks for all three kinds
   (confirmed in `hero-code/.codex/`: dangling `agents -> ../.hero/agents`
   and `commands -> ../.hero/commands` still present, with claude side
   already cleaned). So even when `hero install` IS re-run, the codex
   agents/commands symlinks survive. This is a flat-out cleanup bug
   independent of the drift-detection gap.

The fourth compounding factor: the harness session itself (Claude Code
loading skills/agents) never invokes any Hero code. The warning, even
when it fires, is only visible if the user happens to run a `hero …`
shell command in that project.

### Confirmed reproductions in production

- **paperboy** (`~/projects/personal/repository/paperboy`) — full
  failure: `.claude/{agents,commands,skills}`, `.opencode/{agents,commands,skills}`
  all dangling. Fixed in this session by running `hero install project . --migrate`.
- **hero-code** (`~/projects/hero-engine/repository/hero-code`) — partial
  failure: `.claude/*` real dirs (cleanup did run for claude at some
  point), but `.codex/agents` and `.codex/commands` still dangling
  because they were never in the cleanup list. **This is our own dev
  repo. The bug has been sitting in our own working tree for weeks.**

### Testing gap

[`TestVerify_LegacyCleanup_RemovesCanonicalMirrorAndSymlinks`](internal/install/verification_test.go#L225)
*does* exist and *does* assert that legacy symlinks are cleaned. But:

- It only seeds `.claude/agents` (one path).
- It only runs install for `TargetClaude` (one target).
- It only asserts `.claude/agents` was cleaned (one path).

There is no enumeration test that says "for every target Hero ever
installed symlinks for × every kind we ever wrote, cleanup removes
it." That's why the `.codex/agents` / `.codex/commands` hole sat in
the cleanup list undetected — the test was never going to fail.

Smoke tests run in fresh tmpdirs without legacy artifacts seeded, so
they don't catch it either.

**Required new tests:**

- Enumeration test: parameterize over every entry that *should* be in
  `harnessKindPaths`, seed each as a dangling symlink, run install,
  assert each is removed. Should be table-driven from a single
  authoritative list shared with the cleanup code.
- End-to-end upgrade-stranding test: install with a "P2-era" fixture
  layout (all known symlinks present), do NOT run install again, run
  `hero check` and assert drift is reported. Then run `hero install`
  and assert full cleanup.
- Codex-specific regression: seed `.codex/agents`, `.codex/commands`,
  `.codex/skills` as dangling symlinks, run install for `TargetCodex`,
  assert all three are removed.

### Root Cause

`cleanupLegacyCanonicalSymlinks` is reachable from exactly one call
site — `install.Run` at `internal/install/install.go:142`, gated on
`opts.Mode == ModeProject`. Nothing else in the codebase checks for or
removes the P2 symlink artifacts. Combined with:

- `hero check`'s satellite reconciler (`reportSatelliteDrift` →
  `install.Repair`) iterating only `local.Satellites`, never the root
  project's `.claude/`, `.opencode/`, `.cursor/`, `.codex/`, `.ai/`,
  `.github/copilot/`, `.agents/` harness dirs;
- `version.Mismatch`'s remediation text suggesting `hero upgrade`, which
  does not invoke the cleanup path;
- the harness (Claude Code) never reading Hero state at session start,
  so any nag printed by Hero CLI is invisible from inside the failing
  session;

…the result is that a binary upgrade silently strands the project until
the user reruns `hero install`.

### Source

- `internal/install/install.go:137-145` — sole call site for
  `cleanupLegacyCanonicalSymlinks`.
- `internal/install/cleanup.go:114-169` — the cleanup helper itself,
  including the `harnessKindPaths` list that defines exactly which root
  paths are at risk.
- `internal/install/satellite_repair.go:80-168` — `Repair` walks only
  satellite entries; broken-symlink detection at root never happens.
- `internal/cli/check.go:325-331` — `reportSatelliteDrift` is the only
  install-layout signal `hero check` surfaces.
- `internal/version/version.go:192-223` — `Mismatch` fires correctly
  but suggests the wrong remediation (`hero upgrade` not `hero install
  --migrate`).
- `internal/cli/upgrade.go:53-201` — `runUpgrade` re-renders content and
  bumps the stamp but never calls `cleanupLegacyCanonicalSymlinks`.

### Fix Direction

Chosen: combine **(4) cheap broken-symlink detector** (fires every
`hero` command and inside `hero check`) with **(1) version-drift
surfacing in `hero check`** that names the correct remediation
(`hero install <mode> . --migrate`). Defer (3) auto-repair until we
have telemetry on false-positive rate. Defer (2) loud nag on every
command — the broken-symlink detector subsumes the "loud" need while
limiting noise to the actually-broken case.

Justification:
- (4) is the highest-signal check available: a dangling symlink in a
  managed harness path is unambiguous evidence of strand. Costs are
  a handful of `os.Lstat` + `os.Readlink` calls (paths are already
  enumerated in `cleanup.go`).
- (1) extends `hero check` so workspace health audits and dashboards
  catch the drift even in projects where symlinks were already cleaned
  (e.g. a downstream `--force` install left state inconsistent).
- Skipping (3) and avoiding constant nag keeps the change surgical: one
  helper, one `hero check` row, one preamble line; ~100 LOC.
- Fixing the wrong-remediation text in `version.Mismatch` is the
  cheapest correctness win in the whole flow.

---

## Code Flow (End to End)

How the bug manifests:

1. `cmd/hero/main.go:14-26` — user invokes old `hero install` (e.g.
   v0.13). Build-time `version` is "v0.13.x" (or "dev").
2. `internal/install/install.go:137-200` — `install.Run` runs the P2
   path: creates `.claude/agents -> ../.hero/agents`,
   `.claude/commands -> ../.hero/commands`,
   `.claude/skills -> ../.hero/skills` (plus opencode/cursor variants),
   and materializes `.hero/{agents,commands,skills}/` as the
   canonical mirror.
3. `internal/install/state.go:138-179` — `RecordTargetInstall` stamps
   `install-state.json` with `Mode: "rendered"` and `HeroVersion:
   "v0.13.x"`.
4. `internal/version/version.go:105-128` — `StampInstall` writes
   `version.json` with `HeroVersion: "v0.13.x"` and `LastInstall.Version:
   "v0.13.x"`.
5. **User upgrades binary to v0.14.1** via brew/`go install`/manual
   download. `.hero/`, `.claude/`, etc. are untouched.
6. `internal/cli/root.go:37-66` — on the next `hero <anything>` call,
   `PersistentPreRun` fires `version.Mismatch(heroDir, "v0.14.1")`.
7. `internal/version/version.go:192-223` — `Mismatch` correctly returns
   `"workspace was created with v0.13.x, binary is v0.14.1 — run 'hero
   upgrade' to update"`. Printed to stderr.
8. User runs `hero upgrade`. `internal/cli/upgrade.go:53-201` —
   `runUpgrade` re-renders content via `upgradeTarget` and calls
   `version.StampUpgrade`. **Never invokes
   `cleanupLegacyCanonicalSymlinks`.** Symlinks remain dangling.
9. User opens Claude Code in the project. The harness walks
   `.claude/agents/`, hits `../.hero/agents` (which v0.14 never
   materialized), readdir returns ENOENT, every Agent/Skill/Command
   silently fails to load.
10. User runs `hero check`. `internal/cli/check.go:325-331` →
    `reportSatelliteDrift` → `install.Repair{..., DryRun: true}`.
    `internal/install/satellite_repair.go:63-226` — loops over
    `local.Satellites`. Paperboy has no satellites declared.
    `len(Findings) == 0`. Returns "all clear."
11. User has no further signal until they happen to run
    `hero install <mode> . --migrate` (or someone else recognizes the
    symptom from outside the harness).

Where the fix lands:

A. `internal/install/cleanup.go` — extract a pure `DetectLegacyDrift`
   helper that returns the list of dangling/legacy paths *without*
   removing them.
B. `internal/cli/check.go:runCheck` — add a `legacy-install-drift` row
   that calls `install.DetectLegacyDrift(projectRoot)` and reports
   broken symlinks plus binary↔state version mismatch, with the
   correct remediation string.
C. `internal/cli/root.go:PersistentPreRun` — if the cheap check finds
   any dangling root-harness symlinks, append a one-line warning after
   the existing `version.Mismatch` message naming the correct command.
D. `internal/version/version.go:Mismatch` — the suggested remediation
   becomes `'hero install <mode> . --migrate'` (or makes both commands
   explicit) to ensure the warning's prescribed action actually removes
   legacy artifacts.

---

## Key Files

### Install layer
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/install/install.go` | 137-145 | Sole call site for `cleanupLegacyCanonicalSymlinks`, the gate that makes the bug possible. |
| `internal/install/cleanup.go` | 105-169 | Defines `harnessKindPaths` — the canonical list of paths to probe. Extract `DetectLegacyDrift` reusing this list. |
| `internal/install/satellite_repair.go` | 19, 130-154 | Existing `DriftBrokenSymlink` model — pattern to follow for the new detector's finding type. Walks satellites only, not root. |
| `internal/install/state.go` | 26-72, 92-179 | `InstallState` + `TargetState.HeroVersion` are the source of truth for "what version last touched this target." |

### CLI surfaces
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/check.go` | 95-356, 407-421 | Health-check entry point and existing `reportSatelliteDrift`. New `reportInstallDrift` goes here. |
| `internal/cli/root.go` | 37-66 | `PersistentPreRun` already hosts `version.Mismatch`. Cheap symlink probe fits next to it. |
| `internal/cli/status.go` | 200-238 | Status footer renders `Hero vX (workspace vY — upgrade available)`. Should add `legacy install layout — run 'hero install ... --migrate'` when drift detected. |
| `internal/cli/upgrade.go` | 53-201 | `runUpgrade` is what `version.Mismatch` currently points users at — and what doesn't fix the problem. Either it gains a cleanup call or `Mismatch`'s suggestion text changes. |
| `internal/version/version.go` | 192-223 | `Mismatch` — fix the remediation string. |

---

## Delivery Status

| Step | Description | Shipped in | Status |
|------|-------------|------------|--------|
| §0 | Add `.codex/{agents,commands}` to cleanup list; hoist to `legacyHarnessKindPaths()` | v0.14.2 | ✅ shipped |
| §1 | `DetectLegacyDrift` read-only probe in cleanup.go | v0.14.3 | ✅ shipped |
| §2 | `install-drift` row in `hero check` | v0.14.3 | ✅ shipped |
| §3 | Preamble warning in `PersistentPreRun` | v0.14.3 | ✅ shipped |
| §4 | Fix `version.Mismatch` remediation text | v0.14.3 | ✅ shipped |
| §5 | Enumeration test over `legacyHarnessKindPaths` | v0.14.2 | ✅ shipped |
| §6 | Derive cleanup list from target metadata | — | deferred — structural improvement, no user-visible bug |
| §7 | `hero status` header note | — | deferred — redundant with §3 preamble |

**Correction to the original Code Flow analysis:** step 8 of the
"How the bug manifests" walkthrough claimed `hero upgrade` does not
invoke `cleanupLegacyCanonicalSymlinks`. That was wrong — `upgradeTarget`
calls `install.Run` ([internal/cli/upgrade.go:249](internal/cli/upgrade.go#L249))
which fires cleanup unconditionally for project-mode installs. The
real residual gap (now closed by §3) was "user upgrades binary and
never runs any `hero` command in the project at all" — between the
binary swap and the first `hero ...` invocation, the harness would
silently fail to load skills. The preamble probe in §3 means the
strand is announced loudly the first time the user runs any hero
command.

## Suggested Fix Approach

### 0. **URGENT** — Complete the `harnessKindPaths` cleanup list

**Why:** This is the highest-impact, smallest-change fix. The list
in `cleanup.go:116-133` is missing `.codex/agents` and `.codex/commands`.
hero-code (our own dev repo) demonstrates the consequence today.
Without this fix, every other detection/repair surface added below
will still leave codex installs in a broken state.

**Change** — add the two missing entries to `harnessKindPaths`:
```go
filepath.Join(projectDir, ".codex", "agents"),
filepath.Join(projectDir, ".codex", "commands"),
filepath.Join(projectDir, ".codex", "skills"),  // existing
```

Verify against the actual targets in `internal/install/target_*.go`
that nothing else is missing — write the test in step §5 first, watch
it fail, then fix the list. **Treat the path list as derived from
target metadata, not hardcoded** — see step §6.

### 1. Add `DetectLegacyDrift` in `internal/install/cleanup.go`

**Why:** Provide a pure (read-only) probe that returns the same set of
paths `cleanupLegacyCanonicalSymlinks` would act on, so `hero check`
and root `PersistentPreRun` can surface them without performing
cleanup. Reuse the existing `harnessKindPaths` list verbatim — single
source of truth for "which root paths are at risk."

**Before** (cleanup.go currently only mutates):
```go
func cleanupLegacyCanonicalSymlinks(opts Options, projectDir string) error {
    harnessKindPaths := []string{
        filepath.Join(projectDir, ".claude", "agents"),
        // …
    }
    for _, hp := range harnessKindPaths {
        info, err := os.Lstat(hp)
        // … remove if symlink
    }
    // also remove .hero/{agents,commands,skills}/
}
```

**After** (split path-list out, add a read-only sibling):
```go
// legacyHarnessKindPaths returns the absolute paths under projectDir
// that the P2→render-direct migration removes if present.
func legacyHarnessKindPaths(projectDir string) []string {
    return []string{
        filepath.Join(projectDir, ".claude", "agents"),
        filepath.Join(projectDir, ".claude", "commands"),
        filepath.Join(projectDir, ".claude", "skills"),
        // … same list as today, hoisted out of cleanupLegacyCanonicalSymlinks
    }
}

// LegacyDriftFinding describes one stale install artifact on disk.
type LegacyDriftFinding struct {
    Path   string // absolute path to the dangling symlink or legacy dir
    Kind   string // "broken_symlink" | "legacy_canonical_dir"
    Target string // readlink target for symlinks, empty for dirs
}

// DetectLegacyDrift returns dangling harness-dir symlinks pointing at
// .hero/, plus any leftover .hero/{agents,commands,skills}/ mirror
// dirs. Read-only — does not mutate the filesystem. Safe to call on
// any project; returns nil/empty for clean installs.
func DetectLegacyDrift(projectDir string) []LegacyDriftFinding {
    var out []LegacyDriftFinding
    for _, hp := range legacyHarnessKindPaths(projectDir) {
        info, err := os.Lstat(hp)
        if err != nil || info.Mode()&os.ModeSymlink == 0 {
            continue
        }
        target, _ := os.Readlink(hp)
        // Only flag symlinks Hero plausibly authored (point into .hero/).
        if !looksLikeHeroSymlinkTarget(target) {
            continue
        }
        // Resolve relative target and check existence.
        resolved := target
        if !filepath.IsAbs(resolved) {
            resolved = filepath.Join(filepath.Dir(hp), target)
        }
        if _, err := os.Stat(resolved); err == nil {
            continue // symlink resolves cleanly — not drift
        }
        out = append(out, LegacyDriftFinding{Path: hp, Kind: "broken_symlink", Target: target})
    }
    for _, kind := range []string{"agents", "commands", "skills"} {
        dir := filepath.Join(projectDir, ".hero", kind)
        if info, err := os.Lstat(dir); err == nil && info.IsDir() {
            out = append(out, LegacyDriftFinding{Path: dir, Kind: "legacy_canonical_dir"})
        }
    }
    return out
}
```

And refactor `cleanupLegacyCanonicalSymlinks` to call
`legacyHarnessKindPaths(projectDir)` so the list stays in one place.

### 2. Surface drift in `hero check` (`internal/cli/check.go`)

**Why:** `hero check` is the canonical health surface; adding a row
gives both human readers and the dashboard's `health.json` cache the
signal.

**After** — add inside `runCheck`, near the satellite drift block:
```go
// Legacy install-layout drift — broken symlinks from the P2→render-direct
// migration and stranded .hero/{agents,commands,skills}/ mirror dirs.
findings := install.DetectLegacyDrift(projectRoot)
if len(findings) == 0 {
    addRow("install-drift", "pass", "no legacy install drift")
} else {
    issues += len(findings)
    fmt.Printf("Legacy install drift (%d):\n", len(findings))
    for _, f := range findings {
        switch f.Kind {
        case "broken_symlink":
            fmt.Printf("  [broken_symlink]      %s → %s (target missing)\n", f.Path, f.Target)
        case "legacy_canonical_dir":
            fmt.Printf("  [legacy_canonical_dir] %s\n", f.Path)
        }
    }
    fmt.Println("  Run 'hero install project . --migrate' to remove and re-render.")
    fmt.Println()
    addRow("install-drift", "fail",
        fmt.Sprintf("%d legacy install artifact(s); run 'hero install project . --migrate'", len(findings)))
}
```

### 3. One-line warning in `PersistentPreRun` (`internal/cli/root.go:37-66`)

**Why:** Strand cases need to scream loudly enough that the user notices
on their next `hero` command, even when they bypassed `hero check`.

**After** — add immediately after the existing `version.Mismatch` block:
```go
binaryVersion := rootCmd.Version
if msg := version.Mismatch(heroDir, binaryVersion); msg != "" {
    fmt.Fprintf(os.Stderr, "hero: %s\n", msg)
}

if findings := install.DetectLegacyDrift(projectRoot); len(findings) > 0 {
    fmt.Fprintf(os.Stderr,
        "hero: %d legacy install artifact(s) detected (broken symlinks / stranded canonical dirs). "+
            "Run 'hero install project . --migrate' to repair — until then Claude Code skills/agents will fail to load.\n",
        len(findings))
}
```

### 4. Fix the remediation string in `version.Mismatch` (`internal/version/version.go:215-218`)

**Why:** Today's warning sends users to `hero upgrade`, which doesn't
run cleanup. Either point at `--migrate` directly or recommend both.

**Before**:
```go
return fmt.Sprintf("workspace was created with v%s, binary is v%s — run 'hero upgrade' to update",
    wsVer, binVer)
```

**After**:
```go
return fmt.Sprintf("workspace was created with v%s, binary is v%s — run 'hero install project . --migrate' (re-renders + clears legacy layout) or 'hero upgrade' (content only)",
    wsVer, binVer)
```

### 5. Enumeration test — close the testing hole that hid item §0

**Why:** A 6-row table-driven test would have caught the
`.codex/agents` / `.codex/commands` omission the moment it was made.
The current test seeds one path and checks one path — that's why a
literal cleanup-list typo escaped CI.

**New test** in `internal/install/cleanup_test.go`:
```go
// TestCleanupLegacy_RemovesEveryKnownHarnessPath seeds a dangling
// legacy symlink at EVERY path the cleanup list claims to cover,
// runs install, and asserts each one is removed. Catches regressions
// where a target's directory shape is added/changed but the cleanup
// list isn't updated in lockstep.
func TestCleanupLegacy_RemovesEveryKnownHarnessPath(t *testing.T) {
    for _, path := range legacyHarnessKindPaths(".") {
        t.Run(path, func(t *testing.T) {
            // seed a dangling symlink at this relative path under tmpdir,
            // run install for whichever target owns this path,
            // assert symlink removed and a real dir is in its place.
        })
    }
}
```

### 6. Derive `harnessKindPaths` from target metadata, not hardcoded list

**Why:** The root cause of §0 is that the cleanup list is a
hand-maintained mirror of what each target's install writes. Every
time a target's directory shape changes, two files must change in
lockstep, and one of them silently can't be tested without §5.
Replace with a single source of truth — e.g. each `target_*.go`
exposes its legacy paths, and `cleanup.go` aggregates from
`AllTargets()`. That way "what cleanup handles" is structurally
identical to "what install wrote." This is the durable fix; §0 is
the patch.

### 7. (Optional, low-cost) Status header note in `internal/cli/status.go:220-228`

Add one line after the existing `versionStr` when
`install.DetectLegacyDrift` returns non-empty:
```
legacy install layout detected — run 'hero install project . --migrate'
```
Keeps `hero status` honest about workspace condition.

---

## Test Plan

### Existing test review

| File | Coverage |
|------|----------|
| `internal/install/install_test.go` | exercises `Run`'s happy path; some cleanup coverage via fixture setup. |
| `internal/install/satellite_repair_test.go:61` | `TestRepairFixesBrokenSymlink` — pattern to mirror for root-level detector tests. |
| `internal/install/migration_test.go` | covers P2 → render-direct migration via `install.Run`. Confirms cleanup works *when install fires*. Doesn't cover the "user never re-runs install" path. |
| `internal/install/harness_smoke_test.go` | exercises `install-state.json` round-trip. |
| `internal/cli/check_test.go` + `check_json_test.go` | exercise `hero check` rows including JSON output. Add a `install-drift` row assertion. |

### New tests

1. **`internal/install/cleanup_test.go::TestDetectLegacyDrift_BrokenSymlinks`**
   - Setup: tmpdir with `.claude/agents` symlink to `../.hero/agents`,
     no `.hero/agents` directory on disk.
   - Expect: one `LegacyDriftFinding{Kind: "broken_symlink"}`.

2. **`internal/install/cleanup_test.go::TestDetectLegacyDrift_LegacyCanonicalDir`**
   - Setup: tmpdir with `.hero/agents/` directory present.
   - Expect: `LegacyDriftFinding{Kind: "legacy_canonical_dir"}` for each
     of agents/commands/skills present.

3. **`internal/install/cleanup_test.go::TestDetectLegacyDrift_CleanProjectReturnsEmpty`**
   - Setup: render-direct layout (no `.hero/{agents,commands,skills}/`,
     no harness symlinks).
   - Expect: empty slice.

4. **`internal/install/cleanup_test.go::TestDetectLegacyDrift_ResolvedSymlinkIsNotDrift`**
   - Setup: `.claude/agents -> ../.hero/agents` AND `.hero/agents/` exists.
   - Expect: no `broken_symlink` finding for that path (still flags the
     `legacy_canonical_dir`, which is desired).

5. **`internal/cli/check_test.go::TestCheck_ReportsLegacyInstallDrift`**
   - Setup: tmpdir with broken `.claude/agents` symlink + minimal `.hero/`
     workspace.
   - Run `runCheck`; assert stdout contains `Legacy install drift` and
     `health.json` contains `{name: "install-drift", status: "fail"}`.

6. **`internal/cli/root_test.go::TestPersistentPreRun_WarnsOnLegacyDrift`**
   (new file if absent): subprocess-style harness, verify stderr line
   on a synthetic broken-symlink project.

7. **`internal/version/version_test.go::TestMismatch_RemediationMentionsMigrate`**
   - Set workspace `HeroVersion: "v0.13.0"`, binary `v0.14.1`.
   - Expect returned string contains `--migrate`.

### Regression scope

- Confirm `cleanupLegacyCanonicalSymlinks` still works after
  `legacyHarnessKindPaths` is hoisted out (refactor-only change to the
  call site).
- Verify `PersistentPreRun` early-exit for `name == "install"` still
  applies — we don't want the drift warning to fire when the user is
  *running* `hero install` to fix the drift. Existing skip list at
  `root.go:40-42` already covers `install`, `upgrade`, `init`, `mcp`,
  `version`, `help`.
- Dashboard `health.json` consumers
  (`internal/serve/projectpage/data/health.go`) — adding a row is
  additive; no schema change.
- Confirm `hero check --json` smoke tests still pass with the new row
  present.

---

## Notes

- The `single-source-install-p4-verify` spec (status: completed) plans
  a `hero verify-install` command in this exact space. It does not
  exist in `internal/cli/` today — appears to never have been
  implemented or to have been removed. Worth surfacing in retro; this
  bug is effectively the operational consequence of that gap.
- The user's manual `hero install project . --migrate` worked
  end-to-end in paperboy (111 files re-rendered, symlinks removed).
  Pre-fix `version.json` showed `hero_version: "v0.7.0-143-g8790641-dirty"`
  vs the v0.14.1 binary — `Mismatch` *was* emitting the upgrade
  warning, but the suggested action (`hero upgrade`) wouldn't have
  cleared the symlinks.
- `hero upgrade` arguably should also run
  `cleanupLegacyCanonicalSymlinks` when it detects the source `Mode`
  was P2. Out of scope for this surgical fix but worth a follow-up
  spec if drift recurs after this lands.

---

## Recap

`cleanupLegacyCanonicalSymlinks` runs only inside `install.Run` —
nothing else reconciles the P2→render-direct layout flip. After a
binary upgrade users have dangling `.claude/agents → ../.hero/agents`
symlinks that v0.14+ never materializes, and no Hero surface
(`status`, `check`, `upgrade`, the per-command preamble) reports the
drift correctly. Fix: a read-only `DetectLegacyDrift` helper sourced
from the existing path list, surfaced as a row in `hero check` and a
one-line warning in `root.PersistentPreRun`, plus a correction to
`version.Mismatch`'s remediation text so it names the command that
actually fixes the problem.

## Kickoff

Restores Hero's "every session starts smart" promise after a binary
upgrade by detecting stranded P2 install layout (dangling harness
symlinks + leftover `.hero/{agents,commands,skills}/`) and surfacing
it loudly instead of letting Claude Code sessions silently break.

**Status:** planning — diagnosis complete; no code changes yet.

**Pick up at:** add `install.DetectLegacyDrift(projectDir)` in
`internal/install/cleanup.go` (reuse `harnessKindPaths`, return
`[]LegacyDriftFinding`). Then wire one call in `runCheck`
(`internal/cli/check.go`) and one in `PersistentPreRun`
(`internal/cli/root.go:37-66`), and fix `version.Mismatch`'s
remediation string at `internal/version/version.go:215-222`.

→ `.hero/planning/bugs/upgrade-strands-install-layout/spec.md`

**Files:** `internal/install/cleanup.go:114`,
`internal/cli/check.go:325`, `internal/cli/root.go:57`,
`internal/version/version.go:215`, `internal/install/satellite_repair.go:19`
**Skip:** auto-running cleanup from `PersistentPreRun` (option 3) —
nag-and-let-user-decide first; revisit once detection has shipped.
Also skip routing `hero upgrade` through cleanup in this spec — that's
a follow-up.
