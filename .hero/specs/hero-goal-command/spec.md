---
title: "`hero goal` — emit the run condition, `--check` the per-turn verdict, Stop-hook contract"
slug: hero-goal-command
type: feature
status: completed
priority: high
horizon: now
tags: [drive, goal, cli, mcp, stop-hook, harness, verify]
created: 2026-06-27
relations:
  - target: drive-autonomous-initiative-execution
    kind: parent
  - target: initiative-goal-section
    kind: depends-on
  - target: needs-me-predicate
    kind: depends-on
delivery_method: manual
completed_at: 2026-06-27T23:44:48Z
---

# `hero goal` — emit the run condition, `--check` the per-turn verdict, Stop-hook contract

## Goal

Ship `hero goal <initiative>` (CLI + MCP) as the bridge between Hero and the
harness loop. It does exactly two jobs — **emit** the paste-ready run
condition, and **judge** the run one turn at a time via `--check` — plus the
documented **Stop-hook contract** that lets the harness call the judge after
every turn. It is emphatically **not** a loop driver: the harness `/goal`
owns the turn-after-turn execution and the completion evaluator; `hero goal`
is the authoritative thing that loop consults.

## Kickoff

Add a `hero goal` command ([internal/cli/](../../../../../internal/cli/)) and a
matching MCP tool ([internal/serve/mcp_dispatch.go](../../../../../internal/serve/mcp_dispatch.go)).
`hero goal <init>` (or `--emit`) prints the run condition from the
initiative's `## Goal` ([initiative-goal-section](../initiative-goal-section/spec.md)).
`hero goal <init> --check` returns JSON `{verdict: continue|pause|done, ...}`
by ANDing `hero verify` over the children with `NeedsMe()`
([needs-me-predicate](../needs-me-predicate/spec.md)). Write the Stop-hook
contract doc + a reference hook script so Claude Code calls `--check` each
turn. Keep `--check` I/O plain JSON so other harnesses (Codex) need only a
thin adapter. No `/drive` command here — that's `drive-command-routing`.

## Problem

The harness loop can run turns, but it has no authoritative, deterministic
way to know — per turn — whether to continue, stop for the human, or
declare the initiative done. The harness's own evaluator reads the
*transcript* (a vibe-check, weakest link). Hero has the real signals
(`verify` gates, `needs_me`) but no single command that combines them into a
loop-consumable verdict, and no defined seam for the harness to call it.

## Design

### Subcommands / modes

```
hero goal <init>            # emit: print the run condition (paste into /goal)
hero goal <init> --emit     # explicit alias for the above
hero goal <init> --check    # judge: emit one-turn verdict as JSON
hero goal <init> --dry-run  # show the next 3 transitions --check WOULD take
```

### `--check` verdict (the contract)

```json
{
  "verdict": "continue" | "pause" | "done",
  "initiative": "drive-autonomous-initiative-execution",
  "next_spec": "needs-me-predicate",          // when continue
  "kickoff": "…paste-ready child Kickoff…",     // when continue
  "pause": {                                     // when pause
    "category": "DesignFork",
    "reason": "…",
    "question_ref": ".hero/next/<user>.md"      // see drive-pause-resume
  },
  "remaining": ["drive-pause-resume", "…"],
  "completed": ["initiative-goal-section"]
}
```

Logic:

1. If the just-finished child's `hero verify` is FAIL → consult `NeedsMe`
   (likely `VerifyStuck` after N) → `pause` or `continue` (rework).
2. If all children `hero verify` PASS → `done`.
3. Else pick the next ready child; run `NeedsMe(next, ctx, mode)`. `proceed`
   → `continue` with that child's `## Kickoff` as the turn prompt; `pause`
   → `pause` with category/reason.
4. Hard-cap and irreversible guardrails are enforced by `NeedsMe`; `--check`
   never overrides them.

`--check` is **stateless per call** — it derives everything from disk
(spec statuses, verify results, the run-ledger written by
`drive-pause-resume`). That is what makes it safe to call from a hook and
safe across context resets.

### Stop-hook contract (Claude Code)

Ship `skills/drive/` (or hook reference) with a Stop hook that, while a
Drive run is armed for `<init>`, runs `hero goal <init> --check` after each
turn and:

- `continue` → re-inject `next_spec`'s kickoff, let the loop proceed.
- `pause` → stop the loop, surface the question (file at `question_ref`).
- `done` → clear the goal, report completion.

The contract is harness-agnostic: any harness that can run a post-turn hook
and read JSON can drive Hero. Codex's `/goal` integration is a thin adapter
over the same `--check` JSON (tracked as R2 on the initiative).

## Acceptance Criteria

- WHEN `hero goal <init>` is run, THE SYSTEM SHALL print the initiative's run
  condition suitable to paste into the harness `/goal`.
- WHEN `hero goal <init> --check` runs and all children pass `hero verify`,
  THE SYSTEM SHALL return `verdict: done`.
- WHEN `--check` runs and the next ready child is proceed-eligible per
  `NeedsMe`, THE SYSTEM SHALL return `verdict: continue` with that child's
  Kickoff.
- IF `NeedsMe` returns a pause for the next transition, THEN `--check` SHALL
  return `verdict: pause` with the category and reason.
- THE SYSTEM SHALL derive every `--check` verdict from on-disk state only
  (no in-memory run session required), so a cold process produces the same
  verdict.
- THE SYSTEM SHALL NOT execute agent turns or evaluate completion from the
  transcript — those remain the harness's responsibility.

## Test Plan

- Unit: verdict computation across fixtures (all-pass→done, ready-next→
  continue, pause categories→pause, verify-fail→rework/continue then
  VerifyStuck→pause).
- Determinism: same on-disk fixture yields identical `--check` JSON across
  repeated/cold invocations.
- Integration: scripted Stop-hook against a fixture initiative drives
  continue→continue→pause→(answer)→continue→done.
- MCP parity: the MCP tool returns the same verdict shape as the CLI.

## Risks

- **R2 — harness coupling.** Mitigated by JSON-only `--check`; Claude Code
  hook is reference, Codex adapter is thin.
- **Stale verify results** — `--check` trusts the last `hero verify`. Define
  whether `--check` re-runs verify or trusts cached results (default: trust
  cache, with a `--reverify` escalation) to keep per-turn latency low.
- **Emit/derive coupling** — `--emit` must match the condition
  `initiative-goal-section` materializes; share one code path.

## Changes

- `internal/drive/check.go` — `Check()` (per-turn verdict from on-disk
  child verify-status + `NeedsMe`), `DryRun()` (preview), `Children()` and
  dependency-readiness helpers.
- `internal/cli/goal.go` + `root.go` — `hero goal` command: emit / `--check`
  / `--dry-run`, initiative guard.
- `internal/serve/{mcp_tools.go,mcp_dispatch.go,mcp_tools_def.go}` —
  `hero_goal` MCP tool with the same verdict shape; test-count bump.
- `scripts/drive/stop-hook.sh` + `stop-hook-contract.md` — reference
  Claude Code Stop hook and the harness-agnostic contract.
- Tests: `internal/drive/check_test.go`, `internal/cli/goal_test.go`,
  `internal/serve/mcp_test.go`.

**v1 signal scope (settled with the user):** `--check` operates at
spec-transition granularity; verdict is driven by mode + child verify-status
+ dependency-readiness. Richer `needs_me` signals (readiness score,
design-fork, irreversible-action, verify-stuck counts) await their detectors
/ the run-ledger in later specs and default to "unknown → safe" meanwhile.
Irreversible-action safety lives at the per-turn layer (base safety rules +
harness), not in the between-spec verdict — documented in the contract.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `hero goal <init>` prints the run condition | DONE | `goal.go` emit; `TestGoalEmit`; exercised live (printed objective + derived condition for the real initiative) |
| 2 | `--check` all children pass → `done` | DONE | `drive.Check`; `TestCheckDoneWhenAllChildrenCompleted` |
| 3 | `--check` ready next child → `continue` + Kickoff | DONE | `TestGoalCheckContinue`, `TestCheckContinueGuided`; live `--check` returned the real progress |
| 4 | `NeedsMe` pause → `pause` with category/reason | DONE | `TestCheckPauseSupervised`, `TestCheckBlockedWhenDepsUnmet` |
| 5 | Verdict derived from on-disk state only (cold-stable) | DONE | `Check`/`DryRun` are pure over discovered specs; live run read the two archived completed children correctly |
| 6 | Does NOT run turns or judge completion from transcript | DONE | no such code; loop/eval is the harness's; asserted in `stop-hook-contract.md` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `internal/drive/check.go` verdict engine | DONE | Check/DryRun/Children + helpers |
| 2 | `hero goal` CLI (emit/check/dry-run) + registration | DONE | `goal.go`, `root.go` |
| 3 | `hero_goal` MCP tool (parity) | DONE | handler + dispatch + def; `TestMCP_ToolGoal_CheckParity` |
| 4 | Stop-hook reference + contract | DONE | `scripts/drive/stop-hook.sh`, `stop-hook-contract.md` |
| 5 | Tests across drive/cli/serve | DONE | 8 drive + 3 cli + 1 mcp, all passing |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: built the binary and ran all three modes against the real workspace. `hero goal drive-autonomous-initiative-execution` emitted the objective + derived condition; `--check` returned `completed: [initiative-goal-section, needs-me-predicate]` / `remaining: [the other four]` with a supervised-mode pause on `hero-goal-command`; `--dry-run 3` previewed correctly; and `hero goal needs-me-predicate` (a feature) was rejected with the `/deliver` pointer.

### Excellence Bar self-check

- [x] yes — thin CLI/MCP over a pure, testable verdict engine; honest v1 signal scope agreed with the user (no fake detectors); harness-agnostic JSON contract; reference hook + doc; CLI and MCP return identical shapes. No scope creep into loop-driving or transcript-judging.
