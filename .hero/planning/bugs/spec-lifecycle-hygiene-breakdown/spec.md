---
title: Spec lifecycle hygiene breakdown — five concurrent failures of Hero's own contracts
slug: spec-lifecycle-hygiene-breakdown
type: bug
status: planning
severity: high
priority: P0
root_cause_class: design
created: 2026-05-18
tags: [meta, lifecycle, dogfood, governance, hygiene]
---

# Spec lifecycle hygiene breakdown

## Kickoff

You are picking up a meta-bug: Hero is failing to enforce its own spec-lifecycle contracts. Five symptoms surfaced together in `hero check` on 2026-05-18 (120 issues total): 11 completed-but-not-moved specs, 43 kickoff-missing specs (12 of them currently delivering), 19 specs simultaneously in `delivering` (no WIP limit anywhere), 0/125 status-truthfulness verifications (the verifier ships but no specs have AC graph nodes), and `hero next install-hooks` is run by `hero init` but not by `hero install` or `hero scan`. Open this spec, then read `internal/cli/check.go:55-265`, `internal/cli/complete.go`, `internal/cli/deliver.go:99-134`, `internal/integrity/status.go`, `internal/cli/next_hooks.go:262-372`, and `commands/deliver.md:114`. Pick up at: classify each symptom, then propose child fix-specs in `## Proposed Child Specs` — do NOT create the child specs.

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — every downstream agent decision inherits whatever the spec corpus says; if specs lie or accumulate WIP, the whole "smart cold-start" promise breaks |
| **Ease of Fix** | moderate — each symptom is independently small (one gate or one wired call site), but five concurrent fixes need sequencing to avoid regressing the others |
| **Caused by our codebase?** | Yes — every gap is in Hero itself |
| **Needs more research?** | No — code paths fully traced; child specs can be opened directly |

### Background

`hero check` on 2026-05-18 surfaced 120 issues across five distinct lifecycle dimensions: status-drift accumulation, kickoff-section absence on in-flight work, runaway delivering WIP, no acceptance-criteria backing on completed specs, and a known-good git hook that isn't installed on the most common setup paths. All five are facets of one underlying issue: **Hero detects its lifecycle violations but does not prevent them**, so they accumulate between every `hero check --reconcile` invocation.

### Analysis

The spec lifecycle has the shape `created → planning → delivering → completed → archived (specs/)`. Each transition has a contract Hero documents in skills/commands and a code path that *can* enforce it. The pattern in every symptom: the enforcement is either advisory (warned about but allowed), the prevention path is in a one-off CLI (`hero spec complete`) that depends on the model remembering to invoke it, or the path simply doesn't exist (no WIP awareness, no kickoff gate at status-flip).

### Root Cause

Hero's lifecycle contracts are enforced **only retroactively** by `hero check`. There are no preventive guardrails at the transition points themselves. `hero check --reconcile` cleans up after the fact, but every flow that *starts* the next status (`/deliver`, `hero deliver`, `hero design`, `hero import`) writes the new status without checking the prior contract has been satisfied. The verifier (`internal/integrity/status.go`) exists and is wired into `hero check status`, but its substrate — the acceptance-criteria graph — is empty for the existing corpus, so it returns "unverifiable" on 100% of completed specs.

### Source

- `internal/cli/complete.go` — the missing-link mover (works correctly when invoked, but nothing auto-calls it)
- `internal/cli/deliver.go:99-134` — flips status to `delivering` with no kickoff/WIP check
- `internal/cli/check.go:217-232` — detects missing kickoff but doesn't gate
- `internal/integrity/status.go:91-121` — verifier returns `unverifiable` when `Total == 0`
- `internal/cli/init.go:154-158` — only path that installs the pre-commit hook; `hero install` (the harness-materialization command) and `hero scan` do not

### Fix Direction

Convert detection-only checks into transition-time gates. Specifically: (1) `/deliver` should auto-call `hero spec complete` when status flips to `completed`, (2) kickoff should be a hard precondition for entering `delivering`, (3) a WIP counter should warn (not block) when `delivering > N`, (4) `hero install` and the pre-commit-hook-installed check on `hero scan` should self-heal, (5) leave the AC-graph backfill to the `spec-status-integrity` feature already in flight.

---

## Problem Statement

`hero check` output captured on 2026-05-18:

- **11 specs** with `status: completed` still living under `.hero/planning/` (should be under `.hero/specs/`).
- **43 specs** missing the `## Kickoff` section, **12 of those** with `status: delivering` — the most-active set is least-pick-up-able from a cold session.
- **19 specs** with `status: delivering` simultaneously. No tooling surfaces this as a problem.
- **125 completed specs** evaluated for status-truthfulness; **0 verified, 125 unverifiable** (none have acceptance-criteria graph nodes).
- **Pre-commit hook** flagged not-installed on common setup paths (`hero install`, `hero scan`), even though `hero init` does install it.

The reporter's mission framing: "make the next agent session start smarter than the last one ended." Each of these five failures degrades that promise — kickoff-less delivering specs strand context, lying completed specs poison context-injection, runaway WIP makes "what should I pick up?" unanswerable.

## Environment Details

- Project: Hero itself (dogfooding).
- Branch: `main`.
- Go binary: built from this tree.
- Tracker: not configured for this spec (internal/meta).

---

## Root Cause Analysis

### Symptom 1 — Status drift (11 completed specs stuck in `planning/`)

**Code path that should fire:** `internal/cli/complete.go` implements `hero spec complete <path>` (formerly the `complete` verb). It does three things idempotently: sets `status: completed` in frontmatter, moves the slug dir from `planning/<type>/<slug>/` to `specs/<slug>/`, and re-indexes. Both `moveToSpecs` (line 126) and the runComplete top-level (line 31) are idempotent — they self-no-op when already done.

**Where it should be invoked:** `commands/deliver.md:114` says "When delivery is complete and verified, run `hero spec complete <spec-path>` to archive the spec." But this is **a documented step in a markdown skill, not a code call**. The Go-level deliver paths (`runManualDeliver`, `runAsyncDeliver`, `runDeliverBatch`) flip `status: delivering` (lines 277, 313, 363) but never call the completion mover. Async job completion does write `completed` to frontmatter (verified via `internal/async/`) but does **not** call `moveToSpecs`.

**Confirmed root cause (Symptom 1):** *design gap*. The "complete-and-move" verb exists; the lifecycle-flipping code paths just don't call it. The model is expected to remember to type `hero spec complete <path>` at the end of every delivery — and forgets, repeatedly.

### Symptom 2 — Missing kickoff sections (43, including 12 delivering)

**Detection:** `internal/cli/check.go:331-354 (missingKickoffSpecs)` discovers them. They're also excluded from `hero queue` (`internal/cli/queue.go:38`).

**What writes the kickoff:** Per `commands/design.md:26` and `commands/diagnose.md:16`, `/design` and `/diagnose` are supposed to author the `## Kickoff` section between `## Goal` and `## Problem`. Per `commands/deliver.md:104-109`, `/deliver` should rewrite it on status flip. None of these are code-enforced — they're skill instructions to the model.

**What bypasses kickoff:** `hero import` (`internal/cli/import.go`) scaffolds specs from tracker issues with no kickoff body. The model is supposed to add one before `/deliver`, but there is no gate that prevents `hero deliver` from running on a kickoff-less spec.

**Confirmed root cause (Symptom 2):** *design gap*. The contract ("non-completed work spec carries a `## Kickoff`") is documented and detected, but there is **no hard gate at the planning→delivering transition** that refuses to flip status without one. The fact that 12 of the 43 are currently `delivering` is the smoking gun: the transition fired without the precondition holding. (Also, `hero import` scaffolds without kickoff at all.)

### Symptom 3 — 19 specs in `delivering` simultaneously

**Search for WIP gate:** `grep -rn "WIP\|wip_limit\|MaxInFlight\|max_in_flight" internal/` returns **no matches**. `internal/cli/pulse.go:18` mentions "in-flight" but only for sprint reporting (`hero pulse`), with no advisory or hard limit. `internal/cli/deliver.go:99-134` validates current spec status (rejects double-deliver of the same spec) but never asks "how many specs are *already* delivering?"

**Confirmed root cause (Symptom 3):** *design gap*. There is no WIP-limit concept in the codebase. The contract "finish one delivering spec before starting another" exists only as soft guidance in chats; there is no code asking the question.

### Symptom 4 — 0/125 verified, 125 unverifiable

**Verifier code:** `internal/integrity/status.go:91-121 (CheckCompletedSpecs)`. It iterates completed specs, calls `verifySpec` (line 137), which fetches Criterion nodes from the graph via `acceptance.ListBySpec(store, s.Slug)`. **If `Total == 0`, verdict is `VerdictUnverifiable`** (line 148-151).

**Wiring:** `internal/cli/check.go:237-246` calls `statusTruthfulnessSummary` which calls `buildStatusReport` (`internal/cli/check_status.go:205`) which calls `integrity.CheckCompletedSpecs`. Wired correctly, runs every `hero check`.

**Why all 125 unverifiable:** The verifier depends on Criterion nodes existing in the graph. These come from the `acceptance-criteria-graph` feature (which `spec-status-integrity` lists as `depends-on`). The corpus does not yet have these nodes for completed specs — most predate the AC graph, and the backfill / Phase-2-3 work of `spec-status-integrity` is still `delivering` (per `.hero/planning/features/spec-status-integrity/spec.md:5`).

**Confirmed root cause (Symptom 4):** *dependency / data gap*. The verifier is fully built, fully wired, and operating correctly given empty inputs. The "0 verified" number is **not** a Hero code bug; it's the in-flight AC-graph backfill not yet covering the historical corpus. Blocked by `acceptance-criteria-graph` + `spec-status-integrity` Phase 2+.

### Symptom 5 — Pre-commit hook not auto-installed by `hero install` / `hero scan`

**Where the hook installer lives:** `internal/cli/next_hooks.go:30-114 (nextInstallHooksCmd)` and `installNextHooksQuiet` (line 352). Idempotent, marker-delimited.

**Where it's invoked automatically:**
- `internal/cli/init.go:154-158` — `hero init` calls `installNextHooksQuiet` when not already installed and `--no-hooks` is not set. `--install-hooks` defaults to true.
- `internal/cli/upgrade.go` via `refreshHooksIfPresent` (next_hooks.go:319) — refreshes if already present, **respects user removal** if absent.
- Nowhere else.

**What `hero install` does:** `internal/cli/install.go:63` — installs harness assets (Claude/Codex/Opencode/etc) to a target tool. Different verb from `hero init`. It does not touch git hooks.

**What `hero scan` does:** `internal/cli/scan.go:39` — analyzes codebase, generates knowledge entries. No hook install.

**Confirmed root cause (Symptom 5):** *design gap*, with a deliberately-respected user-removal escape hatch in `refreshHooksIfPresent`. `hero init` does install the hook, but `hero init` is a one-time call. Real-world friction: a teammate who clones the repo and runs `hero install` (the verb they're more likely to discover via `hero --help`) gets a no-hook setup, with `hero check` later telling them to run `hero next install-hooks`. The "first-time setup happens via `hero init`" assumption is not robust.

---

## Code Flow (End to End)

### Symptom 1 — completion → move

1. User invokes `/deliver` → routes to `feature-delivery-lead` agent → agent runs `runManualDeliver` or `runAsyncDeliver` (`internal/cli/deliver.go:306, 333`).
2. Spec frontmatter is mutated to `status: delivering` (lines 313, 363) and re-indexed.
3. Implementation work happens (manual or async).
4. On completion, model is *supposed* to run `hero spec complete <path>` per `commands/deliver.md:114`. **Frequently forgotten.**
5. If skipped: spec ends up with `status: completed` (via agent edit or batch flow at `internal/cli/deliver.go:277`) but still under `.hero/planning/<type>/<slug>/`.
6. `internal/reconcile/reconcile.go:72-86` later detects "completed but in planning/" and reports a finding; `internal/cli/check.go:144-153` calls `moveToSpecs` when `--reconcile` is passed.

### Symptom 2 — kickoff at status flip

1. `hero deliver --manual <slug>` (`internal/cli/deliver.go:53`) → looks up target spec → validates only that current status is not `completed` or already-`delivering` (lines 100-120).
2. **No check** that `target.Kickoff()` is non-empty.
3. Writes `status: delivering` (line 313).
4. `hero check` later detects via `missingKickoffSpecs` and advises.

### Symptom 3 — WIP awareness

1. Same as Symptom 2 entry path.
2. **No query of "how many specs already have `status: delivering`"** before the write.

### Symptom 4 — verifier

1. `hero check` → `runCheck` → `statusTruthfulnessSummary` (`check.go:237`) → `buildStatusReport` (`check_status.go:205`) → opens graph store, discovers specs, calls `integrity.CheckCompletedSpecs`.
2. For each completed spec, `verifySpec` (`status.go:137`) calls `acceptance.ListBySpec(store, s.Slug)`.
3. AC graph nodes don't exist for historical specs → `len(criteria) == 0` → `VerdictUnverifiable`.

### Symptom 5 — hook install

1. `hero init` (`internal/cli/init.go:154-158`) → `installNextHooksQuiet`.
2. `hero install <target>` (`internal/cli/install.go`) → harness materialization → **no hook install**.
3. `hero scan` (`internal/cli/scan.go`) → codebase analysis → **no hook install**.
4. `hero check` (`internal/cli/check.go:192-215`) detects missing hook, prints advisory.

---

## Key Files

### Lifecycle transition code
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/complete.go` | 1–233 | The "set completed + move to specs/" verb. Idempotent. Not auto-called by any other path. |
| `internal/cli/deliver.go` | 99–134, 277, 313, 363 | Status-flip write sites. No kickoff/WIP precondition. |
| `internal/cli/init.go` | 154–158 | Only auto-installer of pre-commit hook. |
| `internal/cli/install.go` | 63 | Harness install — does not touch hooks. |
| `internal/cli/scan.go` | 39 | Scan/analyze — does not touch hooks. |

### Detection-only code (where every fix needs to be promoted to enforcement)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/check.go` | 134–179 | Status-drift detection (reconcile) + auto-fix gated behind `--reconcile`. |
| `internal/cli/check.go` | 217–232 | Missing-kickoff detection (advisory only). |
| `internal/cli/check.go` | 192–215 | Missing-hook detection (advisory only). |
| `internal/reconcile/reconcile.go` | 21–86 | The reconciliation engine; correct, well-scoped, only fires on demand. |

### Verifier (Symptom 4)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/integrity/status.go` | 91–176 | Built and correct. Returns `unverifiable` when AC graph has no Criterion nodes for the spec. |
| `internal/cli/check_status.go` | 205–226 | Wires the verifier into `hero check`. Works as designed. |
| `.hero/planning/features/spec-status-integrity/spec.md` | 1–80 | Status: delivering. Phases 2-3 (AC graph backfill, pre-commit gate, auto-downgrade) outstanding. |

### Hook installer (Symptom 5)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/next_hooks.go` | 78–113, 262–372 | The install verb + helpers. `refreshHooksIfPresent` deliberately respects user removal (line 322–327). |

### Contracts in commands/skills
| File | Lines | Relevance |
|------|-------|-----------|
| `commands/deliver.md` | 114 | "Run `hero spec complete <path>`" — text instruction, not enforced. |
| `commands/design.md` | 26 | "Always include a `## Kickoff` section" — text instruction. |
| `commands/diagnose.md` | 16 | Same kickoff text instruction. |
| `skills/kickoff-prompt/SKILL.md` | 1–50 | Canonical kickoff format. |

---

## Secondary Defects

- **Async job completion** (`internal/async/`, not fully traced here — flagged for the child spec on Symptom 1): the async deliver loop writes `status: completed` from a background process but does not invoke the move logic, even though the move is a pure file operation that could run from the background process safely. Confirm and either include in Symptom 1's fix or open a follow-up.
- **`hero import` scaffolding lacks kickoff template** (`internal/cli/import.go`): imported specs land kickoff-less. Symptom 2's fix should include either (a) scaffolding a `## Kickoff` placeholder at import time, or (b) explicitly flagging imported specs as needing a kickoff before delivery.
- **`refreshHooksIfPresent` respects user removal** intentionally (`internal/cli/next_hooks.go:322-327`) — Symptom 5's fix must preserve this opt-out, not blanket-reinstall.
- **`hero check`'s "stale-hook" detection** (`internal/cli/check.go:206-213`) already auto-detects content drift; the missing-install case (line 194-200) is the only one with no auto-fix counterpart, even though installation is also idempotent. Pure inconsistency.

---

## Notes

This bug is a single-author dogfood report — the very tool reports its own corpus as 120-issues unhealthy. The mood matters: every symptom here is something Hero already *knows about* and has language for. The fix is not "build new diagnostics," it's "promote the diagnostics into preventive gates at the transition where the violation enters the system." The shape of every child spec below is the same: "this contract has detection — add enforcement at the transition site."

The framing question from the prompt ("at what lifecycle transitions does Hero fail to enforce its own contracts?") yields a tidy table:

| Transition | Contract | Enforcement code path | Observed gap |
|---|---|---|---|
| created → planning | spec has `## Kickoff` | none (skill-only) | imported & hand-created specs missing it |
| planning → delivering | kickoff present, WIP < N | `internal/cli/deliver.go:99-134` (status check only) | no kickoff gate, no WIP awareness |
| delivering → completed | move to `specs/`, ACs pass | `internal/cli/complete.go` (manual invocation) | not auto-called by `/deliver` or async runner |
| completed (verified) | ACs in graph, all passing | `internal/integrity/status.go` (works) | empty inputs (AC backfill in flight) |
| any setup path | hooks installed | `internal/cli/init.go:154-158` only | `hero install` / `hero scan` skip it |

---

## Recap

Hero detects all five lifecycle violations but prevents none of them. `hero check` is the janitor; there is no usher at the door. Each symptom is independently a small fix at a known code site, but they share the same shape — promote a soft warning to a transition-time gate — so they're best delivered as five sibling child specs rather than one monolithic fix.

---

## Suggested Fix Approach

Per-symptom fix direction (high level — full plans live in the child specs):

### Symptom 1 — Auto-complete-and-move on delivery finish
Hook `internal/cli/complete.go`'s `runComplete` into the tail of every `/deliver` success path. Either (a) call `runComplete` from the async job runner once the model writes `status: completed`, or (b) wrap `commands/deliver.md`'s final step in a code-enforced "if status flipped to completed, auto-move" check via a post-`/deliver` shell call. Preferred: a code-level hook in `internal/async/` (background-safe) and a CLI-level wrapper for manual flows.

### Symptom 2 — Kickoff is a hard precondition for `delivering`
Add a check at `internal/cli/deliver.go:120` (before status-flip writes): if `target.Kickoff()` is empty, refuse and instruct the user to add one (or auto-scaffold a placeholder + remind the model to fill it). Also patch `hero import` to scaffold a `## Kickoff` stub.

### Symptom 3 — WIP advisory (warn, don't block)
Add a "how many specs currently `delivering`" query at the top of `runDeliver` (`internal/cli/deliver.go:53`). If above a threshold (5? configurable), print a warning listing the in-flight specs and recommend finishing one first. **Soft gate, not hard** — the user may have valid reasons.

### Symptom 4 — No code change; track via `spec-status-integrity`
The verifier works. The 0/125 reflects the AC-graph backfill being in flight. Confirm `spec-status-integrity` Phase 2+ covers the historical-corpus backfill; if not, add it. No new spec needed beyond a note on the existing one.

### Symptom 5 — Self-heal hook install in `hero install` and add a `--repair` hint to `hero check`
At the tail of `internal/cli/install.go`'s install path, if a git repo is present and no managed block exists (and the user hasn't opted out), call `installNextHooksQuiet`. Preserve the user-removal opt-out per the existing `refreshHooksIfPresent` semantics — only install when no markers exist *and* there's no opt-out marker. Optionally surface the same install from `hero scan` for the case where the user runs `hero scan` first.

---

## Test Plan

### Existing test review
- `internal/cli/check_test.go` covers status-drift detection and kickoff-coverage warning. New tests for gating will extend these patterns.
- `internal/cli/complete_test.go` covers the mover. The interesting new tests are *integration* tests that confirm the mover runs from the deliver completion path.
- `internal/cli/next_hooks_test.go` covers the hook install in isolation. New tests need to assert install fires from the broader install path.
- `internal/integrity/status_test.go` — already complete; no changes needed for Symptom 4.

### Test changes needed
- Symptom 1: integration test that runs a full deliver→complete cycle and asserts the spec ends up under `specs/<slug>/spec.md` without a manual `hero spec complete` call.
- Symptom 2: unit test that `runDeliver` errors on a kickoff-less spec; integration test that `hero import` scaffolds a kickoff stub.
- Symptom 3: unit test that `runDeliver` emits a WIP warning to stderr when ≥N specs are already delivering; does NOT error.
- Symptom 5: integration test that `hero install <target>` against a fresh git repo installs the pre-commit hook; also that it skips when a `# >>> hero next hooks (managed) >>>` marker is explicitly removed.

### Regression scope
- WIP advisory (Symptom 3) must not regress async/batch flows that intentionally enqueue many specs (`runDeliverBatch`). Suppress the warning under `--batch`.
- Kickoff gate (Symptom 2) must not break tests that scaffold throwaway specs (`runDeliverBatch` again, plus fixture-driven tests). Provide a `--no-kickoff-gate` test-only flag or auto-skip when running under `go test`.
- Hook install self-heal (Symptom 5) must respect the explicit-removal opt-out — `refreshHooksIfPresent`'s semantics are the canonical reference.

---

## Proposed Child Specs

One per symptom, except Symptom 4 (folded into existing in-flight spec). Naming follows the existing bug-spec convention.

1. **`deliver-auto-completes-and-moves-spec`** (bug — design gap)
   *Auto-call `hero spec complete` at the end of every `/deliver` success path, including the async job runner.*
   Files: `internal/cli/deliver.go`, `internal/async/runner.go` (verify path), `commands/deliver.md`.
   Severity: medium. Closes Symptom 1.

2. **`deliver-requires-kickoff`** (bug — design gap)
   *Block planning→delivering when `## Kickoff` is absent; auto-scaffold a placeholder during `hero import`.*
   Files: `internal/cli/deliver.go:99-134`, `internal/cli/import.go`, `commands/deliver.md`.
   Severity: high (12 in-flight specs are currently un-pickup-able from cold). Closes Symptom 2.

3. **`deliver-warns-on-wip-overflow`** (feature or bug — design gap)
   *Soft advisory when `delivering` count ≥ threshold (configurable, default 5). Lists in-flight specs and recommends finishing one. No hard block.*
   Files: `internal/cli/deliver.go`, `internal/config/config.go` (add `team.max_in_flight`).
   Severity: medium. Closes Symptom 3.

4. **No new spec — track via `spec-status-integrity`**
   *Confirm Phase 2+ of `spec-status-integrity` covers historical AC-graph backfill. If not, file a follow-on.* Closes Symptom 4.

5. **`install-self-heals-pre-commit-hook`** (bug — design gap)
   *`hero install` (and optionally `hero scan`) installs the pre-commit hook on first run when no managed block exists, preserving the user-removal opt-out semantics from `refreshHooksIfPresent`.*
   Files: `internal/cli/install.go`, `internal/cli/scan.go` (optional), `internal/cli/next_hooks.go` (reuse `installNextHooksQuiet`).
   Severity: medium. Closes Symptom 5.

---

Needs more research? → No
