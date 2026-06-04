---
title: Team-mode NEXT handoff is half-built — per-user .hero/next/<user>.md is neither projected nor migrated
slug: next-team-mode-per-user-handoff-unmaintained
type: bug
status: completed
severity: high
priority: P1
domain: engineering
size: medium
created: 2026-06-03
origin: session
root_cause_class: design
tags: [next-md, projection, team-mode, checkpoint, migration, handoff, cross-machine]
relations:
  - target: next-projection-gate-punts-migration-to-user
    kind: relates-to
  - target: next-as-projection
    kind: relates-to
  - target: next-project-file-conflict-not-regenerated
    kind: relates-to
  - target: next-merge-driver-not-portable
    kind: relates-to
completed_at: 2026-06-04T03:02:44Z
---

# Team-mode NEXT handoff is half-built — per-user `.hero/next/<user>.md` is neither projected nor migrated

> Session-originated bug (surfaced live, not from the tracker). No `tracker_id`.

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high (for team-mode users) — in `next.mode: "team"` the primary handoff file `.hero/next/<user>.md` is the file the agent reads/writes (`resolveNextPath` returns it), yet checkpoint never projects it AND the migration can't capture it. The whole point of team mode — per-person briefings that travel across machines and people — silently doesn't work. Blast radius is gated by opt-in. |
| **Ease of Fix** | moderate — the per-user render (`projection.UserHandoffMD`) and the round-trip ingest (`handoff.IngestUserFile`) both already exist and are complete. The minimum fix is removing one early-return and parameterizing one hardcoded path. Full roster/updates rendering for the shared `.hero/NEXT.md` is additional scope a roster renderer must be built for — phase it. |
| **Caused by our codebase?** | Yes — `internal/cli/checkpoint.go` `writeUserHandoffFile` early-returns in team mode (a deferred "Phase 7"), and `internal/cli/next_migrate.go` `runNextMigrateProjection` hardcodes `.hero/NEXT.md` instead of the file the gate flagged. |
| **Needs more research?** | No — root cause confirmed against source. The only open *decision* (not research) is how far to take roster rendering; the spec phases it explicitly. |

### Background
Hero's NEXT subsystem supports two modes (`next.mode` in `hero.json`):

- **solo** (default): one shared `.hero/NEXT.md`.
- **team**: per-user `.hero/next/<user>.md` is each person's *personal* briefing; the shared `.hero/NEXT.md` becomes a **roster** ("## Working on" — one line per teammate) plus cross-cutting **updates** ("## Updates").

The skills describe this richly (`domains/engineering/skills/next-md/SKILL.md` §"Shared file (team mode only)"; `skills/next-handoff-emit/SKILL.md` §"cross-machine handoff"). The per-user file is also the **load-bearing federation medium**: `hero next ingest` walks `.hero/next/*.md` and re-ingests them into the local graph for cross-machine continuity. So there is a real, documented contract.

The code doesn't fulfill it. In team mode the per-user file is never projected on checkpoint, and the one-time legacy→projection migration captures the wrong file.

### Analysis
Two independent gaps, both in the checkpoint/migration wiring (not in the render or ingest layers, which are complete):

1. **Checkpoint never projects the per-user file in team mode.** `writeUserHandoffFile` (`checkpoint.go:321-324`) early-returns when `cfg.NextMode() == "team"`, with a comment that "the migration command (Phase 7) handles the team-mode switchover deliberately." Phase 7 was never wired. So in team mode the primary handoff file goes stale forever — checkpoint produces only `.hero/next/<user>.local.md` (machine state, gitignored) and never the durable `.hero/next/<user>.md`.

2. **Migration targets the wrong file in team mode.** The pre-flight gate (`checkpoint.go:126-133`) inspects `resolveNextPath(...)`, which in team mode returns `.hero/next/<user>.md` (`next.go:106-112`). But `runNextMigrateProjection` hardcodes `nextPath := filepath.Join(heroDir, "NEXT.md")` (`next_migrate.go:55`). So even if a team user runs the migration the gate directs them to, it captures and ingests the *shared* `.hero/NEXT.md`, not the per-user file the gate flagged. The gate and the migration point at different files.

### Root Cause
**Design / process defect — a deferred phase that was never delivered, plus a hardcoded path that predates team mode.**

- Gap 1 is a deliberate `return nil` placeholder for unbuilt work ("Phase 7"). The render it would call (`projection.UserHandoffMD`) is finished and tested; only the checkpoint wiring is missing. This is unfinished delivery, not a logic error.
- Gap 2 is a stale assumption: `runNextMigrateProjection` was written for the solo world where the only handoff file *is* `.hero/NEXT.md`. It hardcodes that path and never consulted `resolveNextPath`, so it silently diverged from the gate when team mode was added.

Both are `design`/`process`, not `code` — the individual functions do what they were written to do; the system contract (team mode == per-user primary handoff) was never finished.

### Source
- `internal/cli/checkpoint.go:314-324` — `writeUserHandoffFile`, the `if cfg.NextMode() == "team" { return nil }` early return (the Phase-7 punt). **Primary fix site.**
- `internal/cli/checkpoint.go:119-133` — the pre-flight gate, which uses `resolveNextPath` (per-user in team mode).
- `internal/cli/next_migrate.go:55` — `nextPath := filepath.Join(heroDir, "NEXT.md")` hardcode.
- `internal/cli/next.go:104-112` — `resolveNextPath` / `nextUserSlug` (the per-user path the gate flags but the migration ignores).
- `internal/projection/user_handoff.go:31-137` — `UserHandoffMD` (the per-user render — complete, do not rewrite).
- `internal/handoff/ingest.go:29-130` — `IngestUserFile` / `ParseUserHandoff` (the round-trip ingest — complete).
- `internal/projection/projection.go:74` — `NextMD` (the shared-file render — has NO roster/team awareness; full roster is additional scope).

### Fix Direction
Finish Phase 7. In `writeUserHandoffFile`, project the per-user file in team mode too (remove the early return). Reconcile the migration path so it captures whatever file the gate flagged (`resolveNextPath`), not a hardcoded `.hero/NEXT.md`. Decide the shared-file contract in team mode (roster + updates per the skill) and phase the roster renderer separately — `projection.NextMD` doesn't render a roster today and building one is meaningfully larger than the two-line minimum fix.

---

## Problem Statement

In a repo with `next.mode: "team"` in `hero.json`:

1. The agent reads and is told to write `.hero/next/<user>.md` (`hero next path` → `resolveNextPath` → per-user file).
2. On every Stop hook, `hero next checkpoint` runs `writeCheckpoint()`. It writes `.hero/next/<user>.local.md` (machine state, gitignored) and then calls `writeUserHandoffFile(...)`, which **returns immediately without writing** because `cfg.NextMode() == "team"`.
3. Result: `.hero/next/<user>.md` — the durable, committed, primary handoff — is **never regenerated from the graph**. It only exists if the user hand-wrote it or ran `hero next migrate` (the file-copy migration, distinct from `migrate-to-projection`). Once it exists, the graph nodes (`UserAsk`, `NextSuggestion`, `SessionReflection`) that should drive it are never reflected back into the file.
4. The cross-machine round-trip is broken at the source: `hero next ingest` (`next_handoff.go:108-145`) walks `.hero/next/*.md` to re-ingest per-user handoffs on a second machine, but checkpoint never *produces* those files in team mode, so there's nothing fresh to ingest.

Independently, if a team-mode repo has unmigrated legacy content and the projection gate fires:

5. The gate inspects `.hero/next/<user>.md` (via `resolveNextPath`) but `migrate-to-projection` captures and ingests `.hero/NEXT.md` (hardcoded). The user runs the migration the gate told them to and their per-user legacy content is **not** captured — the wrong file is preserved.

### Reproduction (inferred — not run in this session)
1. Set `{"next": {"mode": "team"}}` in `.hero/hero.json`; set a user (e.g. `tracking.defaultAgent: "human/alice"`).
2. Emit some handoff state: `hero next ask "..."`, `hero next suggest "..."`.
3. Run `hero next checkpoint`. Observe `.hero/next/alice.local.md` is written but `.hero/next/alice.md` is **not** created/updated from the graph.
4. (Migration path) With `next.projected: false` and legacy content in `.hero/next/alice.md`, trigger the gate; run `hero next migrate-to-projection`; observe it captures `.hero/NEXT.md` (often empty/absent), not `alice.md`.

---

## Environment Details
- This very workspace runs **solo** mode (`next.mode` unset → "solo"), so neither gap fires here. The bug manifests only in repos that opted into `next.mode: "team"`.
- The per-user render and ingest are already exercised by `internal/handoff/ingest_test.go` and `internal/cli/checkpoint_test.go` (`TestWriteUserHandoffFileSkipsUpdatedOnlyChange`) — but those tests run in **solo** config, so they never hit the team-mode early return. The gap is untested.

---

## Root Cause Analysis

**Confirmed (read in this session):**

1. **`writeUserHandoffFile` early-returns in team mode.** `checkpoint.go:321-324`:
   ```go
   func writeUserHandoffFile(projectRoot, heroDir string, cfg config.Config) error {
       if cfg.NextMode() == "team" {
           return nil
       }
       ...
   ```
   The doc comment (`checkpoint.go:314-320`) states the intent: "In team mode, `.hero/next/<user>.md` is the primary handoff file (resolveNextPath returns it) and currently holds agent-authored content. To avoid clobbering that during the projection rollout, this function is a no-op in team mode — the migration command (Phase 7) handles the team-mode switchover deliberately." Phase 7 is not present anywhere in the codebase. The function below the early return (`checkpoint.go:325-350`) is the complete, correct projection — it just never runs in team mode.

2. **The render it would call is complete.** `projection.UserHandoffMD` (`user_handoff.go:31-137`) renders frontmatter (`user`, `updated`, `repo`) + `## Last user ask`, `## Suggested next prompt`, `## Recent reflections`, `## Tried and failed`, `## Your recent activity` from the user-graph nodes. Stable input→output. No team-specific gap in the renderer.

3. **The ingest round-trip is complete.** `handoff.IngestUserFile` / `ParseUserHandoff` (`ingest.go:29-130`) invert `UserHandoffMD` and upsert `UserAsk`/`NextSuggestion`/`SessionReflection` back into the graph, deduped. `hero next ingest` (`next_handoff.go:108-145`) walks `.hero/next/*.md` (skipping `*.local.md`) and calls it per file. The federation medium works — it just has no fresh per-user file to ingest because checkpoint never writes one in team mode.

4. **The migration hardcodes the solo path.** `next_migrate.go:55`: `nextPath := filepath.Join(heroDir, "NEXT.md")`. It never calls `resolveNextPath`. So in team mode it captures/ingests `.hero/NEXT.md` while the gate (`checkpoint.go:127`, via `resolveNextPath` at `next.go:106-112`) flagged `.hero/next/<user>.md`. The gate and migration disagree on which file is "the handoff" in team mode.

5. **The shared-file renderer has no roster mode.** `projection.NextMD` (`projection.go:74-...`) renders `## Just finished` / `## Next` / etc. from the project graph with no awareness of `NextMode()` and no `## Working on` roster or `## Updates` section. The skill's team-mode shared-file contract (`next-md/SKILL.md:201-223`) is **entirely unimplemented** in the projector. Building it is additional scope (see Boundaries / Suggested Fix Approach Phase 2).

**No hypotheses outstanding** — all five points read directly from source.

---

## Code Flow (End to End) — team-mode checkpoint path

1. Host-tool Stop hook fires `hero next checkpoint --quiet`.
2. `checkpoint.go:82` `runNextCheckpoint` → `writeCheckpoint()`.
3. `checkpoint.go:109-117` — load config; `nextPath = resolveNextPath(heroDir, cfg)` → in team mode this is `.hero/next/<user>.md` (`next.go:107-110`); `localPath = .hero/next/<user>.local.md`.
4. `checkpoint.go:126-133` — pre-flight gate. If `!NextProjected()` and `detectUnmigratedNextMD(nextPath)` is non-empty, it currently *refuses* (the gate spec changes this to auto-migrate). Either way it inspects the **per-user** file in team mode.
5. `checkpoint.go:147-167` — NEXT.md write. `writeProjectedNextMD(nextPath, ...)` (projected) or the legacy preserve-and-strip path. **Note:** this path writes to `nextPath` which IS the per-user file in team mode — but it renders with `projection.NextMD` (project-wide content), not the per-user render. So in team mode the per-user file, if written here at all, gets *project* content, not the *personal* briefing. (See Secondary Defects.)
6. `checkpoint.go:174-184` — write `.hero/next/<user>.local.md` (machine block). This works in team mode.
7. `checkpoint.go:190` — `writeUserHandoffFile(projectRoot, heroDir, cfg)`.
8. `checkpoint.go:321-324` — **early-returns because team mode. The per-user personal briefing is never projected.** ← primary bug manifests here.
9. `checkpoint.go:201` — `projectSnapshot(...)` runs against `nextPath` (the per-user file in team mode) — also project-shape, unrelated to the personal briefing.

Migration mismatch flow:

10. Gate (step 4) flags `.hero/next/<user>.md`. User runs `hero next migrate-to-projection`.
11. `next_migrate.go:55` reads `.hero/NEXT.md` (hardcoded), captures *that* as the Note, ingests *that* file's fields — **not** the per-user file the gate flagged.

---

## Key Files

### Checkpoint (primary fix site)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/checkpoint.go` | 314-350 | `writeUserHandoffFile` — the team-mode early return to remove + the (complete) per-user projection below it. |
| `internal/cli/checkpoint.go` | 108-204 | `writeCheckpoint` — orchestration; shows the NEXT.md write path (`nextPath`) vs the per-user write path are separate concerns. |
| `internal/cli/checkpoint.go` | 119-133 | Pre-flight gate — uses `resolveNextPath`; the file the migration must agree with. |

### Path resolution
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/next.go` | 104-112 | `resolveNextPath` — per-user `.hero/next/<user>.md` in team mode, shared `.hero/NEXT.md` in solo. |
| `internal/cli/next.go` | 90-102 | `nextUserSlug` — the user slug source. |

### Migration (path-mismatch fix site)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/next_migrate.go` | 41-117 | `runNextMigrateProjection` — hardcodes `.hero/NEXT.md` at line 55; must use the gate's file in team mode. |
| `internal/cli/next_migrate.go` | 122-140 | `captureNextSnapshot` — what preserves the captured file (currently always the shared file). |

### Render + ingest (complete — do NOT rewrite)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/projection/user_handoff.go` | 31-137 | `UserHandoffMD` — the per-user personal-briefing render. Wire it; don't change it. |
| `internal/handoff/ingest.go` | 29-130 | `IngestUserFile` / `ParseUserHandoff` — round-trip ingest. Complete. |
| `internal/cli/next_handoff.go` | 95-146 | `hero next ingest` — walks `.hero/next/*.md`; the cross-machine consumer of the file checkpoint must produce. |

### Shared-file roster (Phase 2 scope — does not exist yet)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/projection/projection.go` | 28-74+ | `NextMDOptions` / `NextMD` — no `NextMode`/roster awareness. A roster renderer would live here or beside it. |

### Skills (the intended contract)
| File | Lines | Relevance |
|------|-------|-----------|
| `domains/engineering/skills/next-md/SKILL.md` | 57-62, 201-223 | Team-mode contract: per-user = personal briefing; shared = roster + updates. |
| `domains/engineering/skills/next-handoff-emit/SKILL.md` | 149-160 | Per-user file is the cross-machine federation medium. |

---

## Acceptance Criteria

- WHERE `next.mode` IS `team` THE SYSTEM SHALL project `.hero/next/<user>.md` from the user-graph nodes (`UserAsk`/`NextSuggestion`/`SessionReflection` + author-attributed activity) on every `hero next checkpoint`, identical in shape to the solo-mode per-user render.
- WHEN `hero next checkpoint` runs in team mode THE SYSTEM SHALL NOT leave `.hero/next/<user>.md` stale — a fresh graph state SHALL be reflected into the file (subject to the existing semantic-change skip that suppresses updated-only churn).
- WHEN `hero next migrate-to-projection` runs THE SYSTEM SHALL capture and ingest the file that `resolveNextPath` returns for the active mode (`.hero/next/<user>.md` in team mode), NOT a hardcoded `.hero/NEXT.md`.
- WHERE the projection gate flagged a file as unmigrated THE SYSTEM SHALL migrate that same file (gate target == migration target in both solo and team mode).
- WHERE `next.mode` IS `team` AND the shared-roster phase is delivered THE SYSTEM SHALL render the shared `.hero/NEXT.md` with a `## Working on` roster line for the current user (see Phase 2 — may ship as a follow-up).
- IF the per-user projection fails (graph open/render error) THEN THE SYSTEM SHALL keep the checkpoint non-fatal (warn to stderr, continue), preserving today's `writeUserHandoffFile` failure contract.

---

## Goal
In team mode, `hero next checkpoint` projects `.hero/next/<user>.md` from the graph the same way solo mode does, so each person's personal briefing stays current and travels across machines via `hero next ingest`; and `hero next migrate-to-projection` captures the same file the projection gate flags, so a team user's legacy content is preserved on migration. The shared `.hero/NEXT.md` roster/updates rendering is scoped as an explicit follow-up phase.

---

## Suggested Fix Approach

Phased. **Phase 1 is the load-bearing minimum** and resolves all but the last acceptance criterion. **Phase 2 (roster)** is genuinely larger and may ship separately.

### Phase 1 — project the per-user file in team mode + reconcile the migration path

**Delivered (2026-06-03):**
- `internal/cli/checkpoint.go` — Change 1 (removed the `cfg.NextMode() == "team"` early return in `writeUserHandoffFile`, updated doc comment) + Change 3 (introduced `sharedNextPath := filepath.Join(heroDir, nextFileName)`; routed the project-shape NEXT.md write block and the snapshot pointer to it so the project projection targets shared `.hero/NEXT.md` and the per-user projection owns `nextPath` — no double-write in team mode).
- `internal/cli/next_migrate.go` — Change 2 (replaced the hardcoded `.hero/NEXT.md` with `resolveNextPath(heroDir, cfg)`). Confirmed `extractAskAndSuggestion` round-trips a `UserHandoffMD`-shaped file (same `## Last user ask` / `## Suggested next prompt` headers + blockquote bodies) — reused it; `ParseUserHandoff` not needed.
- `internal/cli/checkpoint_test.go` — added `TestWriteUserHandoffFileTeamModeProjectsPerUserFile` (canonical regression), `Test_writeCheckpoint_TeamMode_WritesBothFiles` (Change 3 guard), `Test_writeCheckpoint_TeamMode_NonFatalOnProjectionError`.
- `internal/cli/next_migrate_test.go` — added `TestExtractAskAndSuggestion_RoundTripsProjectedShape`, `Test_runNextMigrateProjection_TeamModeCapturesPerUserFile`, `Test_runNextMigrateProjection_SoloModeCapturesSharedFile`.

Phase 2 (shared-file roster) deferred to a follow-up, as designed.

#### Change 1 — Remove the team-mode early return in `writeUserHandoffFile`

**File:** `internal/cli/checkpoint.go`

**Why:** the render below the early return (`UserHandoffMD` → `writeProjectedFileIfSemanticChanged`) is complete and correct. The only reason it doesn't run in team mode is the deferred-Phase-7 guard. Removing it makes the per-user file project in *both* modes — and `nextUserSlug` already resolves the user in both. The `writeProjectedFileIfSemanticChanged` call keeps the no-op-write suppression intact, so this does not churn the file every turn.

**Before** (`checkpoint.go:314-329`):
```go
// writeUserHandoffFile renders .hero/next/<user>.md from the graph.
//
// In team mode, .hero/next/<user>.md is the primary handoff file
// (resolveNextPath returns it) and currently holds agent-authored
// content. To avoid clobbering that during the projection rollout,
// this function is a no-op in team mode — the migration command
// (Phase 7) handles the team-mode switchover deliberately.
func writeUserHandoffFile(projectRoot, heroDir string, cfg config.Config) error {
	if cfg.NextMode() == "team" {
		return nil
	}
	user := nextUserSlug(cfg)
	if user == "" {
		return nil
	}
	userPath := filepath.Join(heroDir, nextDirName, user+".md")
```

**After:**
```go
// writeUserHandoffFile renders .hero/next/<user>.md from the graph in
// BOTH solo and team mode. In team mode this file is the primary
// handoff (resolveNextPath returns it); in solo mode it is the
// per-user companion to the shared NEXT.md. The render is total-
// rewrite from the user-graph nodes (UserAsk / NextSuggestion /
// SessionReflection) — projections always win, and the semantic-
// change guard below suppresses updated-only churn.
func writeUserHandoffFile(projectRoot, heroDir string, cfg config.Config) error {
	user := nextUserSlug(cfg)
	if user == "" {
		return nil
	}
	userPath := filepath.Join(heroDir, nextDirName, user+".md")
```

(The body from `graph.Open` through `writeProjectedFileIfSemanticChanged` is unchanged.)

**Note on safety:** the original comment feared "clobbering agent-authored content." That fear is the same one the `next-as-projection` migration already resolved for solo NEXT.md: the migration captures hand content as a durable `Note` before projection takes over. For team mode the equivalent protection is Change 2 (the migration must capture the per-user file). Sequence the delivery so the migration-path fix lands with — or before — the projection switch, so a team repo that still has hand-authored `.hero/next/<user>.md` content has it captured before the projector starts overwriting it. State this ordering in the PR.

#### Change 2 — Make `migrate-to-projection` capture the gate's file (mode-aware path)

**File:** `internal/cli/next_migrate.go`

**Why:** the gate inspects `resolveNextPath(heroDir, cfg)`; the migration must capture and ingest that exact file. Replacing the hardcode aligns gate target == migration target in both modes and ensures a team user's per-user legacy content is preserved.

**Before** (`next_migrate.go:54-56`):
```go
	heroDir := cfg.HeroDir(projectRoot)
	nextPath := filepath.Join(heroDir, "NEXT.md")
	body, err := os.ReadFile(nextPath)
```

**After:**
```go
	heroDir := cfg.HeroDir(projectRoot)
	// Capture the file the projection gate flags as unmigrated, which
	// is mode-dependent: shared .hero/NEXT.md in solo, per-user
	// .hero/next/<user>.md in team. Hardcoding NEXT.md silently
	// diverged from the gate once team mode existed.
	nextPath := resolveNextPath(heroDir, cfg)
	body, err := os.ReadFile(nextPath)
```

**Decision to confirm during delivery:** `extractAskAndSuggestion` parses the *legacy hand-written* section names ("Last user ask" / "Suggested next prompt" / "Proposed next ask"). A team user's existing `.hero/next/<user>.md` may already be in the *projected* shape (same section headers `UserHandoffMD` emits). Both paths use the same `## ` section names, so `extractAskAndSuggestion` should still find them — but verify against a realistic team-mode per-user fixture before assuming the extraction round-trips. If team per-user files need the richer `handoff.ParseUserHandoff` extractor instead of `extractAskAndSuggestion`, prefer reusing `ParseUserHandoff` (it already inverts `UserHandoffMD` and handles reflections). The `.gitattributes` directive update (`ensureNextMDMergeDirective`) already covers `.hero/next/*.md`, so no change there.

> **Handoff to / from the gate spec (`next-projection-gate-punts-migration-to-user`):** that spec lists this exact path mismatch as **"Secondary Defect 1 (flag, decide during fix)"** and recommends either (a) pass the gate's `nextPath` into the migration, or (b) scope auto-migration to solo only. **THIS spec owns and resolves that defect** — the decision is **(a): make the migration mode-aware via `resolveNextPath`** (Change 2 above). The gate spec's auto-migration work should therefore call into a `migrateToProjection(projectRoot, cfg, out)` that already reads `resolveNextPath` (i.e., land Change 2 here first, or coordinate so the extracted `migrateToProjection` helper picks up the mode-aware path). **The two specs must not both edit `next_migrate.go:55` independently** — Change 2 is the single source of truth for that line; the gate spec defers the path reconciliation here.

#### Change 3 — (optional, recommended) guard against double-writing `nextPath` in team mode

**File:** `internal/cli/checkpoint.go`

**Why (Secondary Defect):** in team mode `resolveNextPath` returns the per-user file, so the NEXT.md write block (`checkpoint.go:147-167`, via `writeProjectedNextMD`/legacy path) writes *project-shape* content to `.hero/next/<user>.md`, and then Change 1 immediately overwrites it with *personal-briefing* content. Two writers, last-wins, churn. Confirm during delivery whether `writeProjectedNextMD` should target `.hero/NEXT.md` (the shared file) in team mode rather than `nextPath`. Likely Phase-1 cleanup: in team mode, the "NEXT.md projection" should target the *shared* file and the "user handoff projection" targets the *per-user* file — they are different documents with different renderers. Smallest correct version: in team mode, point the shared-file projection at `filepath.Join(heroDir, nextFileName)` and leave the per-user projection (Change 1) to own `nextPath`. **Verify the interaction before finalizing** — this is the one place the two write paths overlap.

### Phase 2 — shared `.hero/NEXT.md` roster + updates (additional scope, may be a follow-up)

`projection.NextMD` has no roster mode. Delivering the skill's full team-mode shared-file contract (`## Working on` roster line per teammate, `## Updates` cross-cutting section) requires a new renderer (or a `NextMode`/roster branch in `NextMD`) that:

- enumerates known users (from `.hero/next/*.md` filenames and/or graph user nodes),
- renders one roster line per user (latest `NextSuggestion`/activity as the "what they're working on" one-liner),
- renders recent cross-cutting `Updates` (source TBD — likely `SessionReflection`s tagged team-affecting, or a new node kind).

This is meaningfully larger than Phase 1 and has open design questions (where does "affects others" live in the graph?). **Recommendation: ship Phase 1 alone first** — it restores the primary per-user handoff and fixes migration, which is the high-severity part. File Phase 2 as a follow-up feature spec (`next-team-mode-shared-roster-projection`) rather than blocking the bug fix on it. Until Phase 2 lands, the shared `.hero/NEXT.md` in team mode remains agent-authored/legacy-projected exactly as today — no regression, just the roster contract stays unimplemented as it already is.

---

## Boundaries

- **Not changing** `projection.UserHandoffMD` or `handoff.IngestUserFile`/`ParseUserHandoff` — they are complete and correct. This bug is checkpoint/migration wiring only.
- **Not building** the shared-file roster renderer in Phase 1 — that's Phase 2 / a follow-up feature. Phase 1 does not regress the shared file; it stays as-is in team mode.
- **Not touching** the `hero next migrate` file-copy command (`next.go:275-318`) — that's the distinct one-shot solo→team file move, unrelated to projection.
- **Defers to the gate spec** (`next-projection-gate-punts-migration-to-user`) for the auto-migrate-instead-of-refuse behavior. This spec only owns the *path* the migration targets, not the *trigger*. Coordinate so the extracted migration helper there uses the mode-aware path from Change 2.
- **Not tightening** `detectUnmigratedNextMD` / `sectionHasRealContent` detection — out of scope (also flagged out-of-scope by the gate spec).

---

## Risks
- **Clobbering existing hand-authored per-user content** when Change 1 flips the projector on for team repos. Mitigation: land Change 2 (migration captures the per-user file as a durable Note) first or together, and document the ordering. The `writeProjectedFileIfSemanticChanged` guard prevents churn but does NOT preserve pre-existing hand content — capture must precede projection.
- **Double-write churn on `nextPath`** in team mode (Change 3) — if not addressed, the per-user file may flip between project-shape and personal-briefing content within a single checkpoint. Verify the two write paths target distinct files in team mode.
- **`extractAskAndSuggestion` vs projected-shape per-user files** — confirm the migration extractor handles a per-user file already in `UserHandoffMD` shape; prefer `ParseUserHandoff` if not.
- **Cross-machine ingest now has live files** — once Phase 1 lands, `hero next ingest` will start ingesting freshly-projected per-user files on other machines. Confirm the round-trip (project → commit → pull → ingest → re-project) is idempotent (the ingest dedupes reflections and skips auto-derived suggestions; verify on a team fixture).

---

## Test Plan

### Existing test review
| Test | File | Disposition |
|------|------|-------------|
| `TestWriteUserHandoffFileSkipsUpdatedOnlyChange` | `internal/cli/checkpoint_test.go:166-196` | Keep — runs in solo config (no `next.mode`); still valid. Add a team-mode sibling (below). |
| `internal/handoff/ingest_test.go` (`IngestUserFile` round-trip) | — | Keep — render/ingest unchanged. |
| `internal/cli/next_migrate_test.go` (`TestExtractAskAndSuggestion_*`) | — | Keep; add a mode-aware path assertion (below). |

### New tests
1. **Team-mode checkpoint projects the per-user file** — set `cfg.Next = &config.NextConfig{Mode: "team"}` and a user; emit a `UserAsk`/`NextSuggestion` into the graph; call `writeUserHandoffFile` (and/or `writeCheckpoint`); assert `.hero/next/<user>.md` exists and contains the emitted ask/suggestion (the team-mode analogue of the solo test). This is the canonical regression test for the primary bug — it FAILS today (early return writes nothing).
2. **Team-mode checkpoint via full `writeCheckpoint`** — run the whole checkpoint in team config; assert both `.hero/next/<user>.local.md` (machine state) AND `.hero/next/<user>.md` (durable briefing) are written, and that the durable file is the personal-briefing render (has `## Last user ask` / `## Suggested next prompt`), not the project-shape `## Just finished` render. Guards Change 3 (no double-write contamination).
3. **Migration captures the gate's file in team mode** — set team mode + a per-user `.hero/next/<user>.md` with legacy content + an empty/absent `.hero/NEXT.md`; run `migrate-to-projection`; assert the captured `Note` body is the *per-user* file's content (not the shared file's), and the extracted `UserAsk`/`NextSuggestion` came from the per-user file.
4. **Migration captures the shared file in solo mode (regression)** — same test in solo config; assert it still captures `.hero/NEXT.md`. Guards Change 2 against breaking solo.
5. **Round-trip idempotence in team mode** — project → ingest (`hero next ingest`) → re-project; assert the second projection is byte-stable (no duplicate reflections, no auto-derived-suggestion corruption).
6. **Non-fatal failure** — force a graph-open error during the team-mode per-user projection; assert `writeCheckpoint` still returns nil (warn-and-continue contract preserved).

### Regression scope
- Solo mode: confirm `writeUserHandoffFile` behavior is unchanged for solo repos (Change 1 only removes the team gate; solo path is identical).
- CI drift gate: team-mode is not exercised in this repo's CI (solo), so no fixture should regress — but confirm no test fixture sets `next.mode: team` and then asserts the per-user file is absent.
- `hero next ingest` on a multi-user `.hero/next/` directory — confirm it still skips `*.local.md` and ingests only durable files (unchanged, but now there are durable files to ingest).

---

## Kickoff

You're fixing a confirmed `design` bug: in team mode (`next.mode: "team"`), Hero's per-user handoff `.hero/next/<user>.md` is neither projected on checkpoint nor captured by the migration. The render and ingest layers are already complete — this is checkpoint/migration *wiring*. Read `.hero/planning/bugs/next-team-mode-per-user-handoff-unmaintained/spec.md` first.

**Status:** completed — **Phase 1 delivered** on branch `fix/next-team-mode-per-user-handoff` (audit verdict SHIP). `writeUserHandoffFile` now projects the per-user file in team mode (`checkpoint.go`), `migrate-to-projection` uses the mode-aware `resolveNextPath` (`next_migrate.go:55` — single source of truth for that line), and the team-mode double-write is deconflicted via `sharedNextPath`. Six regression tests added; project→ingest→re-project confirmed idempotent.

**Pick up at (Phase 2 — the only remaining scope):** the shared `.hero/NEXT.md` roster + updates renderer. `projection.NextMD` still has no `NextMode`/roster awareness — file this as the follow-up feature spec `next-team-mode-shared-roster-projection` (open design question: where does "affects others" live in the graph?). Until then the shared file in team mode stays project-shape, no regression.

<details><summary>Phase 1 delivery instructions (done — kept for history)</summary>

**The two gaps:**
1. `internal/cli/checkpoint.go:321-324` — `writeUserHandoffFile` early-returns `nil` when `cfg.NextMode() == "team"` (deferred "Phase 7", never wired). The complete per-user projection sits right below it, dead in team mode.
2. `internal/cli/next_migrate.go:55` — `migrate-to-projection` hardcodes `.hero/NEXT.md`; the gate (`checkpoint.go:127`) flags `resolveNextPath` (per-user in team mode). They target different files.

**Pick up at (Phase 1 only — defer the roster):**
1. Remove the team-mode early return in `writeUserHandoffFile` (Change 1). The `writeProjectedFileIfSemanticChanged` guard already prevents churn.
2. Replace `next_migrate.go:55` hardcode with `resolveNextPath(heroDir, cfg)` (Change 2). Sequence this with/before Change 1 so a team repo's hand-authored per-user content is captured as a durable Note before the projector starts overwriting it.
3. Check Change 3: in team mode the NEXT.md projection (`checkpoint.go:147-167`) and the per-user projection both touch `nextPath`. Point the shared-file projection at `.hero/NEXT.md` so they don't double-write.

→ `internal/cli/checkpoint.go:314`, `internal/cli/next_migrate.go:55`, `internal/cli/next.go:104`, `internal/projection/user_handoff.go:31`

**Coordinate:** the gate spec `next-projection-gate-punts-migration-to-user` lists this path mismatch as its "Secondary Defect 1 (decide during fix)" and DEFERS it here. THIS spec owns `next_migrate.go:55`. When the gate spec extracts a `migrateToProjection` helper, it must use the mode-aware path from Change 2 — don't let both specs edit that line independently.

**Skip:** rewriting `projection.UserHandoffMD` or `handoff.IngestUserFile` — they're complete. Building the shared-file roster renderer — that's Phase 2 / a separate feature spec (`next-team-mode-shared-roster-projection`), not part of this fix.

**Tests:** add a team-mode `writeUserHandoffFile` test (it fails today), a team-mode full-`writeCheckpoint` test asserting both `<user>.md` and `<user>.local.md` are written with the right renders, and a migration test asserting the per-user file is captured in team mode (and the shared file still captured in solo).

</details>

---

## Recap
In `next.mode: "team"`, Hero's primary per-user handoff `.hero/next/<user>.md` is silently never maintained: `writeUserHandoffFile` early-returns on a deferred "Phase 7" (`checkpoint.go:321-324`), and `migrate-to-projection` hardcodes `.hero/NEXT.md` (`next_migrate.go:55`) instead of the per-user file the gate flags — so the file is neither projected nor migrated, breaking the documented cross-machine federation contract. The render (`UserHandoffMD`) and ingest (`IngestUserFile`) are already complete; the fix is two-line-minimum wiring (remove the early return, parameterize the path) plus a write-path deconfliction, with full shared-file roster rendering deferred to a Phase 2 follow-up. Severity: high for team-mode users (opt-in blast radius), `design`/`process` root cause.
