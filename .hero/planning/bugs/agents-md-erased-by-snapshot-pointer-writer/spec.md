---
title: "AGENTS.md silently erased by the snapshot pointer writer — five of six harnesses started cold for two months"
slug: agents-md-erased-by-snapshot-pointer-writer
type: bug
status: delivering
severity: critical
priority: P0
domain: engineering
root_cause_class: code
created: 2026-07-15
tags: [agents-md, install, snapshot, managed-region, harness, regression]
relations:
  - target: codex-install-broken
    kind: related
  - target: agents-md-managed-region-consolidation
    kind: related
supersedes: [codex-install-broken]
---

# AGENTS.md silently erased by the snapshot pointer writer

## Kickoff

The snapshot pointer writer was deleting the whole AGENTS.md managed region every time
you ran `hero snapshot --project`. Fixed — the writer can no longer touch install-managed
files at all.

**Status:** delivering — fix is implemented, all suites green, uncommitted in the working tree.

**Pick up at:** commit the fix, then repair the repo's own AGENTS.md, which is still the
7-line stub on disk: run `make install` (auto-syncs the codex sibling) or
`hero install project . --target codex`, and confirm the file comes back at ~232 lines.
Then correct the `codex-install-broken` record — it is marked `completed` while its
Evidence section still describes today's repo.

→ `.hero/planning/bugs/agents-md-erased-by-snapshot-pointer-writer/spec.md`

**Files:** `internal/snapshot/pointers.go:17-53`, `internal/install/agents_md.go:214-220`,
`internal/install/harness_smoke_test.go:330`, `.hero/specs/codex-install-broken/spec.md:262-277`

**Skip:** don't make `managed.Writer.Write` merge with sections already on disk — section
IDs are not emitted into rendered output and cannot be recovered from it. Don't re-add an
`agentsPath` parameter guarded by an if; a param that silently does nothing is the same trap.

## Problem

`hero snapshot --project` and `hero next checkpoint` silently reduced the project-root
`AGENTS.md` from 232 lines to a 7-line stub, deleting all Hero install content — routing
table, workflow instructions, rules, CLI reference, everything but the snapshot pointer.

Any harness that reads `AGENTS.md` — codex, opencode, cursor, copilot, generic: **five of
six install targets** — started completely cold, as a vanilla coding agent with no
awareness of Hero. Only `claude` escaped, and only by accident (see below).

The bug produced zero symptoms. It ate exactly one file, and it was the one file a
Claude-dogfooding team never opens.

## Root cause

**Confirmed and reproduced.** Code defect, introduced by refactor.

`internal/snapshot/pointers.go` `EnsurePointer()` called `writePointerOnly()` on
`AGENTS.md`. `writePointerOnly` composes a `managed.Writer` whose `Sections` list contains
**only** the snapshot-pointer section:

```go
writer := managed.Writer{
    File:     path,
    Sections: []managed.SectionContributor{contrib},
}
```

`managed.Writer.Write` (`internal/managed/region.go`) renders the managed region from its
own section list alone and splices it in wholesale. It does **not** merge with sections
already present in the file — it cannot, because section IDs are not emitted into rendered
output and so cannot be recovered from it.

Therefore a pointer-only write **replaced** the entire `hero:managed` region, deleting
every install-contributed section.

## Why it was harmless when written, then became destructive

Two commits, the same day, by design and by accident:

| Commit | Date | What it did |
|---|---|---|
| `5110b1a` | 2026-05-18 | `feat(snapshot): land project-snapshot` — introduced the pointer writer. It owned its **own separate marker block** (`<!-- >>> hero snapshot pointer (managed) >>> -->`), appended at end of file. It physically could not touch install's block. Correct and safe. |
| `933a6a3` | 2026-05-18 | `refactor(managed): consolidate AGENTS.md/CLAUDE.md/NEXT.md writers into one orchestrator` — merged the two blocks into one region. The pointer writer kept its "write my block" behavior, which now meant **"write the WHOLE block."** |

The refactor had already taught install to emit the pointer as one of its own sections —
`defaultSections` in `internal/install/agents_md.go:214-220` includes
`snapshot.NewPointerSection`. It did the new half and left the old standalone writer
standing. This fix finishes that job.

## Evidence

`AGENTS.md` was 310 lines on 2026-05-12 (`982742d`). Git history shows it filled and eaten
**twice**, each time within a single day:

- `e7304a2` (May 31, 120 lines) → `969a18e` (May 31, 7 lines)
- `afe9553` (Jun 9, 150 lines, titled *"target-aware AGENTS.md bridge Codex workflow
  execution gap"*) → `5af2e46` (Jun 9, 7 lines)

**Reproduced** with old vs. fixed binary against an identical fixture: `hero snapshot
--project` took AGENTS.md 232 → 7 lines on the old binary, 232 → 232 on the fixed one.

**CLAUDE.md was spared only by accident.** The pointer path hardcoded the literal string
`"AGENTS.md"` (`agentsMDPathOrEmpty` in `internal/cli/snapshot.go`) and never resolved
`nativeInstructionFile(target)`. CLAUDE.md is proven equally vulnerable — with the eraser
reintroduced, the new guard test fails on CLAUDE.md at 19433 → 147 bytes.

## Why it survived two months

This is the point of the spec. The bug was live for two months, through a critical/P0
investigation that looked directly at it and closed without seeing it.

**The test that should have caught it asserted at the one moment the file is guaranteed
correct.** `TestHarness_SmokeCodex` *did* assert AGENTS.md content — "Running Hero
Workflows in Codex", the managed markers, a command-deliver reference. It is a good test.
It was structurally incapable of catching this, because it asserts **immediately after
install** — the single moment the file cannot be wrong, since install just wrote it. The
eraser fires on the *next* command. The test suite modeled install as a terminal state.
Nothing modeled the file *living in the repo* alongside other commands.

**The manual check had the identical blind spot.** `codex-install-broken`'s Completion
Ledger evidence column literally reads *"verified in AGENTS.md after install"*.

**The prior investigation misdiagnosed it, and its wrong conclusion is why this recurred.**
`codex-install-broken` (June 9, critical/P0, status `completed`) diagnosed this as "Codex
install is broken." Install was never broken — it renders 232 correct lines on the first
try. An empty file points suspicion at the writer that should FILL it, not at an unrelated
snapshot helper that EMPTIES it. That spec's Evidence section contains a **verbatim copy of
the 7-line stub that is still in the repo today**: it closed while the bug was live. The
false "fixed" signal is the single most load-bearing cause of the recurrence.

**That spec skipped every detection mechanism it designed.** Of six fixes, the three that
shipped all WRITE content; the three that were cut all DETECT problems — Fix 4 (`hero
check` advisories) marked "P3 priority", Fix 6 (auto-detect stale managed regions) marked
"lower priority". Fix 6 describes this exact failure.

**Camouflage.** The bug only ate the one file the team never reads. There were no symptoms
to notice.

## Fix

Implemented; all suites green, build clean.

### `internal/snapshot/pointers.go`
- `EnsurePointer(nextPath string)` — the `agentsPath` parameter was **removed entirely**
  rather than guarded. A parameter that silently does nothing is the same trap in a new
  costume; removing it makes the constraint unrepresentable rather than merely discouraged.
- The doc comment records the constraint and the incident, so the next person to reach for
  this writer reads why it must not point at an install-managed file.

### `internal/snapshot/projector.go`
- `ProjectOptions.AgentsMDPath` field removed.

### `internal/cli/snapshot.go`
- `AgentsMDPath` wiring removed, along with the now-orphaned `agentsMDPathOrEmpty` helper —
  the source of the accidental CLAUDE.md reprieve.

### `internal/cli/checkpoint.go`, `internal/serve/mcp_tools_snapshot.go`
- Call sites updated for the new signature.

## Test plan

### Existing test review
- `internal/install/harness_smoke_test.go` — `TestHarness_SmokeCodex` asserts AGENTS.md
  content but only at install time. Left as-is; it is correct for what it covers.
- `internal/snapshot/pointers_test.go` — legacy tests exercised the pointer writer against
  AGENTS.md. Retargeted to NEXT.md, the only file it now writes.

### Tests added
- `internal/snapshot/pointers_test.go:81` — `TestEnsurePointer_DoesNotWriteInstallManagedFiles`.
  The guard at the unit boundary.
- `internal/snapshot/pointers_test.go:50` — `TestWritePointerOnly_ReplacesEntireManagedRegion`.
  Deliberately **characterizes the destructive semantics** rather than hiding them. The
  replace-not-merge behavior is correct for NEXT.md and load-bearing; pinning it in a test
  keeps the sharp edge visible, so any future move to merge-semantics is a conscious
  decision with a red test in front of it.
- `internal/install/harness_smoke_test.go:330` —
  `TestHarness_InstalledContentSurvivesOrdinaryCommands`. **The durable guard.**
  Table-driven over all six targets (claude→CLAUDE.md; codex/opencode/cursor/copilot/generic
  →AGENTS.md) per the `harness-changes-cover-all-targets` tripwire — covering one target is
  how the blast radius got misjudged the first time. It installs, runs `snapshot.Project`
  (an *ordinary command*, not an install step), and asserts the root file is byte-identical
  and still contains the shared-body line "Finish the closing gate before yielding".
  **Verified to go RED when the eraser is reintroduced.** CI runs
  `go test -race -count=1 ./...`, so it gates the build.

The class of test that was missing — and now exists — is *"installed content survives
ordinary use,"* as distinct from *"install produces correct content."*

### Regression scope
- NEXT.md pointer insertion, including the legacy hand-authored-line short-circuit — covered
  by the retargeted `pointers_test.go` suite.
- `hero snapshot --project`, `hero next checkpoint`, and the `hero_snapshot` MCP tool all
  drive the changed signature; all three call sites compile and are exercised by suite.

## Outstanding

Recorded here, not done in this pass:

1. **The repo's own AGENTS.md is still the 7-line stub on disk.** Needs
   `hero install project . --target codex`, or `make install` (confirmed it auto-syncs:
   *"auto-sync siblings: [codex]"*).
2. **`codex-install-broken` is marked `completed` but its Evidence section describes the
   current repo.** Its record needs correcting. That false "fixed" signal is the single most
   load-bearing cause of this recurrence — leaving it standing invites a third round.
3. **Consider reviving its skipped Fix 4 / Fix 6** — detection, not writing: a `hero check`
   advisory when a root instruction file's managed region is missing expected sections.

## Changes

### `internal/snapshot/pointers.go`
- `EnsurePointer` signature reduced to `EnsurePointer(nextPath string)`; `agentsPath`
  parameter removed. Doc comment records the constraint and the two erasure incidents.

### `internal/snapshot/projector.go`
- `ProjectOptions.AgentsMDPath` removed.

### `internal/cli/snapshot.go`
- `AgentsMDPath` wiring and the orphaned `agentsMDPathOrEmpty` helper removed.

### `internal/cli/checkpoint.go`, `internal/serve/mcp_tools_snapshot.go`
- Call sites updated.

### `internal/snapshot/pointers_test.go`
- Added `TestEnsurePointer_DoesNotWriteInstallManagedFiles` and
  `TestWritePointerOnly_ReplacesEntireManagedRegion`; legacy tests retargeted to NEXT.md.

### `internal/install/harness_smoke_test.go`
- Added `TestHarness_InstalledContentSurvivesOrdinaryCommands`, table-driven over all six
  targets.

## Completion Ledger

| Item | Status | Evidence |
|---|---|---|
| Root cause confirmed | DONE | Reproduced old vs. fixed binary on identical fixture: `hero snapshot --project` took AGENTS.md 232 → 7 on old, 232 → 232 on fixed |
| Eraser removed at the source | DONE | `EnsurePointer(nextPath string)` in pointers.go:34 — `agentsPath` param gone, not guarded; `grep -rn AgentsMDPath --include='*.go'` returns zero hits |
| Dead wiring removed | DONE | `ProjectOptions.AgentsMDPath` and `agentsMDPathOrEmpty` deleted; call sites in checkpoint.go, snapshot.go, mcp_tools_snapshot.go updated |
| Unit guard | DONE | `TestEnsurePointer_DoesNotWriteInstallManagedFiles` (pointers_test.go:81) |
| Sharp edge characterized | DONE | `TestWritePointerOnly_ReplacesEntireManagedRegion` (pointers_test.go:50) pins replace-not-merge so a future change is deliberate |
| Durable cross-target guard | DONE | `TestHarness_InstalledContentSurvivesOrdinaryCommands` (harness_smoke_test.go:330) — all six targets, asserts survival of `snapshot.Project`; **verified RED when eraser reintroduced** |
| CLAUDE.md vulnerability proven | DONE | With eraser reintroduced, guard fails on CLAUDE.md at 19433 → 147 bytes — the reprieve was an accident of a hardcoded string, not a design |
| All tests pass | DONE | `go test -race -count=1 ./...` green; build clean |
| Repo's own AGENTS.md repaired | OUTSTANDING | Still the 7-line stub on disk; needs `make install` or `hero install project . --target codex` |
| `codex-install-broken` record corrected | OUTSTANDING | Marked `completed` while its Evidence describes the live bug; the false-fixed signal that caused this recurrence |
| Detection advisories (its Fix 4 / Fix 6) | OUTSTANDING | `hero check` advisory for managed regions missing expected sections — deliberately deferred, recorded so it isn't silently re-skipped a third time |

### Exercise-the-feature check

- [x] Exercised: built old and fixed binaries and ran `hero snapshot --project` against an
      identical 232-line AGENTS.md fixture. Old binary: 232 → 7 lines. Fixed binary:
      232 → 232, byte-identical. The guard test was then confirmed to go red when the
      eraser was reintroduced, so the regression cannot land silently again.

**Needs more research?** → No. Root cause confirmed, reproduced, fixed, and guarded.
