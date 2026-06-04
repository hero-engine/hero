---
title: NEXT-projection migration gate punts migration to the user instead of doing it automatically
slug: next-projection-gate-punts-migration-to-user
type: bug
status: planning
severity: high
priority: P1
domain: engineering
created: 2026-06-03
origin: session
root_cause_class: design
tags: [next-md, projection, migration, checkpoint, stop-hook, ux, mission-fit]
relations:
  - target: next-as-projection
    kind: regression-of
  - target: next-as-projection-architecture
    kind: revises-decision-of
  - target: next-team-mode-per-user-handoff-unmaintained
    kind: relates-to
  - target: next-project-file-conflict-not-regenerated
    kind: relates-to
  - target: next-merge-driver-not-portable
    kind: relates-to
---

# NEXT-projection migration gate punts migration to the user instead of doing it automatically

> Session-originated bug (surfaced live, not from the tracker). No `tracker_id`.

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — the projection subsystem halts ordinary work and hands the user a raw CLI incantation to run a transition Hero is designed to do for them. Directly contradicts the mission ("inject context automatically, without anyone asking"). Not cosmetic. |
| **Ease of Fix** | moderate — the safe migration logic already exists and is non-interactive; the fix is to call it from the gate's trigger point instead of returning an error, plus a failure contract that preserves the existing no-clobber safety. The team-mode path-mismatch (below) adds care. |
| **Caused by our codebase?** | Yes — `internal/cli/checkpoint.go` pre-flight gate (AC-14) returns a user-facing error string instead of performing the migration. |
| **Needs more research?** | No — root cause confirmed against source, spec, and git history. Trigger-point recommendation and failure contract are specified below. |

### Background
In a Hero-managed repo that has NOT run `hero next migrate-to-projection`, the Stop/checkpoint hook fails every turn with:

> `unmigrated NEXT.md detected (legacy section header `## Just finished` contains hand-written content) — run `hero next migrate-to-projection` first`

The repo carries hand-authored NEXT.md content under legacy section headers (or legacy `<!-- BEGIN HERO MACHINE STATE -->` markers) and `next.projected` is still `false` in `hero.json`. The pre-flight gate refuses to write and tells the user to go run a CLI migration by hand. This is a leak of an internal transition step into consumer projects.

### Analysis
The gate (`checkpoint.go:126-133`) was added deliberately (commit `07a0403`, "pre-flight migration gate for hero next checkpoint (AC-14)") as a *safety* measure: the legacy write path would otherwise overwrite NEXT.md with a placeholder and lose hand-authored content. The gate's choice was to **refuse and punt to the user** rather than **migrate automatically**. But `hero next migrate-to-projection` (`next_migrate.go`) is already fully non-interactive and content-preserving: it captures the entire existing NEXT.md as a durable `Note` node ("next-as-projection migration; preserving pre-projection content"), extracts structured fields into `UserAsk`/`NextSuggestion` nodes, updates `.gitattributes`, and flips `next.projected = true`. Because the migration requires no human judgment and preserves content recoverably, there is no product reason it must be a manual step gated behind a warning.

### Root Cause
**Design / process defect — an over-cautious transition, not an unfinished one.** AC-14 was specified and shipped as a *refuse-and-direct-the-user* gate (delivery spec AC-14; architecture decision §5 and the "What this locks in" bullet: *"The migration gate must keep firing for unmigrated repos indefinitely. Removing it would silently wipe hand-authored content…"*). The reasoning conflated two different actions: (a) **silently wiping** content via the legacy placeholder write — genuinely destructive, must never happen; and (b) **silently migrating** content via the existing content-preserving migration path — safe, recoverable, and exactly what the mission wants. The design picked "refuse" because at the time the projection path was new and not yet trusted to own the file. That caution is now mis-targeted: the migration command it directs users to is the *same logic* that could run automatically, and it preserves content. The gate punts work back onto the human that Hero is designed to do for them.

### Source
- `internal/cli/checkpoint.go:119-133` — the pre-flight gate that returns the error.
- `internal/cli/checkpoint.go:410-496` — `detectUnmigratedNextMD` / `sectionHasRealContent` / `isItalicPlaceholder` (detection helpers).
- `internal/cli/next_migrate.go:41-140` — `runNextMigrateProjection` + `captureNextSnapshot` + `extractAskAndSuggestion` (the safe, reusable migration logic).
- `.hero/specs/next-as-projection/spec.md` (AC-14) and `.hero/specs/decisions/next-as-projection-architecture/spec.md` (§5, "What this locks in") — the design intent being revised.

### Fix Direction
At the point the gate currently fires, **perform the migration automatically and silently** instead of returning an error: run the existing migration logic (capture → ingest → flip `next.projected`), then continue the checkpoint as a migrated repo. Surface a message ONLY if the auto-migration actually fails — and on failure, fall back to the existing no-clobber behavior (do not overwrite hand-authored NEXT.md with a placeholder) with a clear, actionable message rather than a raw CLI incantation. Optionally also trigger the migration at `hero upgrade` so version-skewed workspaces transition proactively.

---

## Problem Statement

`hero next checkpoint` runs on every Stop hook on every assistant turn (locked-in consequence of next-as-projection). In a repo where:

- `next.projected == false` in `.hero/hero.json`, AND
- `.hero/NEXT.md` contains real hand-written content under a legacy header (`## Just finished`, `## Next`, `## Tried and failed`, `## Context to carry forward`) or a legacy `<!-- BEGIN HERO MACHINE STATE -->` marker,

…`writeCheckpoint()` returns the error:

```
unmigrated NEXT.md detected (legacy section header `## Just finished` contains hand-written content) — run `hero next migrate-to-projection` first
```

The Stop hook surfaces this to the user during ordinary work. The user is asked to run an internal CLI migration by hand — a transition that requires no human judgment and that Hero already knows how to perform losslessly.

### Reproduction (inferred — not run in this session)
1. In a repo with `next.projected` unset/false, write `.hero/NEXT.md` with a `## Just finished` section containing a non-placeholder line.
2. Run `hero next checkpoint` (or trigger a Stop hook).
3. Observe the non-zero exit and the "run `hero next migrate-to-projection` first" message.

This exact path is covered by existing tests `Test_writeCheckpoint_PreFlightGate_RefusesLegacyMarkers` and `…_RefusesLegacyHeaders` in `internal/cli/checkpoint_test.go` (they assert the *refusal* — the behavior we are changing).

## Environment Details
- This very workspace has `next.projected: true` (`.hero/hero.json:75-76`), so the gate does NOT fire here. The bug manifests in **consumer/unmigrated** projects.
- This workspace reports a version skew ("workspace created with v0.14.5, binary v0.15.3 — run hero upgrade"), which is relevant to trigger-point candidate (b): `hero upgrade` is a natural place to perform the transition proactively for workspaces created before projection existed.

---

## Root Cause Analysis

**Confirmed (read in this session):**

1. The gate is intentional, not unfinished. `git log -L 119,133:internal/cli/checkpoint.go` shows it landed whole in commit `07a0403` ("pre-flight migration gate for hero next checkpoint (AC-14)"). The comment states the rationale explicitly: *"Without this gate, the legacy write path silently rewrites the file with a placeholder, losing hand-authored sections that `hero next migrate-to-projection` would have ingested into the graph as durable nodes."* (`checkpoint.go:119-125`).

2. AC-14 (`.hero/specs/next-as-projection/spec.md:451-457`) specifies the refuse-and-direct behavior: *"SHALL exit non-zero … with a message directing the user to `hero next migrate-to-projection`, and MUST NOT overwrite the existing file in that state."* The architecture decision (`next-as-projection-architecture/spec.md:166-186`, and the locked-in consequence at lines 427-431) reaffirms it: *"The migration gate must keep firing for unmigrated repos indefinitely."*

3. The migration logic is safe and non-interactive (`next_migrate.go:41-117`):
   - It is **idempotent** — re-running detects the projected flag and is a no-op (`next_migrate.go:49-52`).
   - It **captures the full NEXT.md body as a durable `Note` node** before anything else (`captureNextSnapshot`, `next_migrate.go:122-140`), with `reason: "next-as-projection migration; preserving pre-projection content"`. Nothing is lost.
   - It **extracts structured fields** into `UserAsk` / `NextSuggestion` nodes (`next_migrate.go:78-97`).
   - It updates `.gitattributes` for the merge driver (`next_migrate.go:102-105`).
   - It flips `next.projected = true` in `.hero/hero.json` (`setNextProjected`, `next_migrate.go:282-310`).

**The defect:** the gate directs the user to run logic that is *already safe to run automatically*. The destructive thing the gate protects against is the **legacy placeholder write** (`checkpoint.go:156-167` — `nextBody = nextPlaceholder(...)` when the stripped body is empty), NOT the migration. Auto-migrating closes the gap the gate exists to protect while doing the work for the user.

**Hypothesis (not fully proven, low risk):** at the checkpoint trigger point, running the migration mutates state mid-Stop-hook — opens the graph, upserts nodes, writes `hero.json` and `.gitattributes`. `writeCheckpoint` already opens the graph (`writeProjectedNextMD`, `acFlipsSince`) and already writes files. The migration's writes are append/upsert, not destructive, and the flag flip is a single-value JSON write. No new re-entrancy is introduced beyond what checkpoint already does. The one place that needs care is the **team-mode path mismatch** (see Secondary Defects) — confirm the migration targets the same file the gate detected before treating them as interchangeable.

---

## Code Flow (End to End)

1. Host-tool Stop hook fires `hero next checkpoint --quiet` (wired by `internal/install/claude_hooks.go` / `codex_hooks.go`).
2. `internal/cli/checkpoint.go:82` `runNextCheckpoint` → `writeCheckpoint()`.
3. `internal/cli/checkpoint.go:109-117` — load config, resolve `nextPath` (`resolveNextPath`: `.hero/NEXT.md` solo, `.hero/next/<user>.md` team) and `localPath`.
4. `internal/cli/checkpoint.go:126` — `if !cfg.NextProjected()` (true for unmigrated repos).
5. `internal/cli/checkpoint.go:127` — `detectUnmigratedNextMD(nextPath)` returns a non-empty reason (`checkpoint.go:435-450`) because `sectionHasRealContent` (`checkpoint.go:457-485`) finds a non-placeholder line under a legacy header.
6. `internal/cli/checkpoint.go:128-131` — **returns the error string to the user. Checkpoint aborts. NEXT.md is left untouched and no migration happens.** ← bug manifests here.

Contrast: the safe path the user is told to run manually:

7. `internal/cli/next_migrate.go:41` `runNextMigrateProjection` → captures Note, ingests fields, updates `.gitattributes`, flips `next.projected = true` (`next_migrate.go:69-116`). After this, `cfg.NextProjected()` is true and step 4's branch is skipped on the next checkpoint, so `writeProjectedNextMD` (`checkpoint.go:288-312`) projects from the graph.

---

## Key Files

### Checkpoint / gate
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/checkpoint.go` | 108-133 | `writeCheckpoint` + the pre-flight gate that returns the error (the fix site). |
| `internal/cli/checkpoint.go` | 147-167 | The legacy write path — placeholder overwrite is the actual destructive behavior the gate guards. The failure fallback must keep this safe. |
| `internal/cli/checkpoint.go` | 410-496 | `legacyNextHeaders`, `detectUnmigratedNextMD`, `sectionHasRealContent`, `isItalicPlaceholder` — detection. |

### Migration (reusable logic)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/next_migrate.go` | 41-117 | `runNextMigrateProjection` — the migration to be invoked automatically. Needs a non-cobra entry point. |
| `internal/cli/next_migrate.go` | 122-140 | `captureNextSnapshot` — durable Note preservation (why auto-migration is safe). |
| `internal/cli/next_migrate.go` | 145-230 | `extractAskAndSuggestion` and helpers — structured ingest. |
| `internal/cli/next_migrate.go` | 282-310 | `setNextProjected` — single-value `hero.json` flip. |

### Config
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/config/config.go` | 1656-1668 | `NextMode()` / `NextProjected()` — the authoritative migration signal. |

### Spec / decision being revised
| File | Lines | Relevance |
|------|-------|-----------|
| `.hero/specs/next-as-projection/spec.md` | 451-457 | AC-14 — the requirement to revise. |
| `.hero/specs/decisions/next-as-projection-architecture/spec.md` | 166-186, 427-431 | §5 + locked-in consequence — the design decision to revise. |

---

## Secondary Defects

1. **Team-mode gate/migration path mismatch (pre-existing) — NOW OWNED BY A SIBLING SPEC.** In team mode, `resolveNextPath` returns `.hero/next/<user>.md` (`next.go:106-112`), so the gate at `checkpoint.go:127` inspects the per-user file, but `runNextMigrateProjection` hardcodes `.hero/NEXT.md` (`next_migrate.go:55`) — so even the manual migration would not capture a team user's flagged content. **This is no longer decided here.** It is owned and resolved by `next-team-mode-per-user-handoff-unmaintained` (resolution: make the migration mode-aware via `resolveNextPath`, decision (a)). **Delivery handoff:** this gate spec's auto-migration change (Change 2 below) must NOT edit `next_migrate.go:55` — the team-mode spec does that. To avoid a collision, sequence the team-mode spec first (or land them together), and have this spec's auto-migration call the migration entry point as the team-mode spec leaves it. In solo mode (the default, where this bug was reported) the gate and migration already agree on `.hero/NEXT.md`, so the auto-migration fix is correct and shippable independently for solo repos.

2. **`sectionHasRealContent` treats filler as real content (eager detection).** `nextPlaceholder()` emits italic placeholders that are correctly skipped, but real-world filler like "Nothing yet." / "No open features in this repo." is plain (non-italic) text and would count as "real content," making the gate fire on effectively-empty files. **Mostly mooted by the fix:** once auto-migration runs, an over-eager detection just means the (harmless, content-preserving) migration runs on a near-empty file — no user-visible halt. The captured Note would hold filler, which is benign. Recommend NOT tightening detection as part of this fix (out of scope); note it for a follow-up only if the captured-Note noise becomes a concern.

3. **`captureNextSnapshot` props map has a gofmt alignment quirk** (`next_migrate.go:125-130`, `"reason"` over-indented). Cosmetic; ignore.

---

## Notes

- The mission framing is load-bearing for severity. Hero's job is to inject context automatically "without anyone asking." A feature that halts and tells the human to run a CLI migration is the subsystem punting its own designed work back onto the user. That is why this is **high**, not low/cosmetic.
- The fix must NOT remove the no-clobber safety. The architecture decision's fear (silent content loss) stays valid for the *placeholder write path*; the fix replaces "refuse" with "migrate (which preserves), and on migration failure, fall back to the existing no-clobber refuse-but-don't-overwrite behavior."

---

## Root Cause Classification

- **Class:** `design` (an over-cautious transition baked into the AC-14 spec and the architecture decision), with a `process` flavor (the manual-migration step was treated as a permanent gate rather than a temporary bridge). Per `root-cause-classification`: not `code` (the code does exactly what AC-14 told it to), not `data`/`environment`/`dependency`.
- **Severity:** high.

---

## Acceptance Criteria

> **Why this section exists.** The original `next-as-projection` work encoded the "disposable, always-regenerate" contract only *partially* — `AC-2` covers hand-edits to `.hero/NEXT.md`, but the same guarantee for `.hero/next/<user>.md` lives only in prose (`spec.md:245-248`), and the no-op-write idempotency lives in a *separate* spec (`next-noop-writes`). Nothing in the AC set contradicted a permanent refuse-and-punt gate, which is how AC-14 shipped. These criteria write the contract down as enforceable invariants so the gate's behavior is pinned to it. The deep-research synthesis (this session) is the source.

**Behavioral contract — the disposable projection (the invariant the gate must serve):**

- **AC-A1 — Projected files are derived, never authoritative (steady state).** With `next.projected == true`, `.hero/NEXT.md` and `.hero/next/<user>.md` are total-rewritten from the graph on every checkpoint. A hand-edit to either file is forfeit on the next regeneration. *(Encodes for `<user>.md` what `next-as-projection` AC-2 encodes only for `NEXT.md`.)*
- **AC-A2 — One-time capture is the ONLY preservation path.** The single exception to AC-A1 is the legacy→projection transition: before the flag flips, the pre-projection file body is captured into the graph as a durable, recoverable `Note` (`captureNextSnapshot`). This exists because legacy-era prose had no graph home; after capture, AC-A1 resumes permanently. There is no other path that reads a projected file's content back as truth (the cross-machine `hero next ingest` round-trip is a separate, intentional mechanism, not a hand-edit-preservation path).
- **AC-A3 — Idempotent write (no git noise).** A regeneration whose content is identical to disk except the `updated:` frontmatter timestamp MUST NOT write the file. *(Already implemented by `writeProjectedFileIfSemanticChanged` / `normalizeUpdatedFrontmatter`; this AC pins it with a guard test so a future change can't silently reintroduce per-turn churn.)*

**The fix — auto-migrate instead of punt:**

- **AC-B1 — Silent auto-migration.** When `next.projected == false` and `detectUnmigratedNextMD` finds legacy content at checkpoint, Hero performs the migration automatically (capture → ingest → flip flag) and continues the checkpoint as a migrated repo. No user-facing instruction to run a CLI command is emitted on the success path. A `--quiet` checkpoint stays byte-silent.
- **AC-B2 — Failure is the only thing that speaks.** If auto-migration fails, `writeCheckpoint` surfaces an *actionable, human* message (not a raw `run hero next migrate-to-projection first` incantation), leaves `.hero/NEXT.md` byte-for-byte untouched (never the `nextPlaceholder` overwrite), and leaves `next.projected == false`. This preserves the exact no-clobber safety AC-14 was protecting.
- **AC-B3 — Idempotent transition.** Running checkpoint twice on a freshly-unmigrated repo migrates once; the second run takes the already-projected path with no second capture.
- **AC-B4 — Proactive transition at upgrade.** `hero upgrade` performs the same migration (guarded by `!NextProjected()`) so version-skewed workspaces transition without waiting for the next checkpoint.

**Scope boundaries (owned elsewhere — do NOT fix here):**

- Team-mode gate/migration path reconciliation → `next-team-mode-per-user-handoff-unmaintained`.
- `.hero/NEXT.md` conflicts not regenerated by the merge driver → `next-project-file-conflict-not-regenerated`.
- Merge driver not surviving fresh clones → `next-merge-driver-not-portable`.

The conflict-regeneration half of the disposability contract (AC-A1's "conflicts regenerate, never surface") is real intent but is currently *under-enforced* — its two enforcement gaps are the two sibling merge specs above. This spec asserts the contract; those specs make it hold everywhere.

---

## Suggested Fix Approach

### Change 1 — Extract a reusable, non-cobra migration entry point

**File:** `internal/cli/next_migrate.go`

**Why:** the migration body is currently welded to `runNextMigrateProjection(cmd, args)` and writes to `cmd.OutOrStdout()`. To call it from `writeCheckpoint`, factor the core into a function that takes the already-loaded `projectRoot`/`cfg` and returns an error, with output optional/suppressible.

**Before** (`next_migrate.go:41-52`, abbreviated):
```go
func runNextMigrateProjection(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	out := cmd.OutOrStdout()

	if cfg.NextProjected() {
		fmt.Fprintln(out, "Already migrated. next.projected is true in hero.json.")
		return nil
	}
	// … capture / ingest / gitattributes / flip …
}
```

**After** (shape — keep the cobra command as a thin wrapper):
```go
func runNextMigrateProjection(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	return migrateToProjection(projectRoot, cfg, cmd.OutOrStdout())
}

// migrateToProjection performs the legacy→projection transition.
// Idempotent (no-op when already projected). Content-preserving:
// captures the full NEXT.md body as a durable Note before flipping.
// Callable both from the CLI command and automatically from the
// checkpoint gate. `out` may be io.Discard for silent auto-migration.
func migrateToProjection(projectRoot string, cfg config.Config, out io.Writer) error {
	// (the existing body of runNextMigrateProjection, minus the
	//  config.Load it now receives, writing progress to `out`)
}
```

### Change 2 — Replace the refuse-gate with auto-migration + safe failure fallback

**File:** `internal/cli/checkpoint.go`

**Before** (`checkpoint.go:126-133`):
```go
	if !cfg.NextProjected() {
		if reason := detectUnmigratedNextMD(nextPath); reason != "" {
			return "", fmt.Errorf(
				"unmigrated NEXT.md detected (%s) — run `hero next migrate-to-projection` first",
				reason,
			)
		}
	}
```

**After:**
```go
	if !cfg.NextProjected() {
		if reason := detectUnmigratedNextMD(nextPath); reason != "" {
			// Auto-migrate silently: ingest legacy content into the
			// graph (durable, recoverable) and flip next.projected.
			// This is the transition `hero next migrate-to-projection`
			// performs — there is no human judgment required, so Hero
			// does it rather than punting to the user.
			if err := migrateToProjection(projectRoot, cfg, io.Discard); err != nil {
				// Migration failed. Preserve the original safety
				// contract: do NOT let the legacy placeholder write
				// clobber hand-authored content. Surface an actionable
				// message and leave NEXT.md untouched.
				return "", fmt.Errorf(
					"automatic NEXT.md migration failed (%s): %w — your NEXT.md was left untouched; "+
						"run `hero next migrate-to-projection` to retry or inspect the error",
					reason, err,
				)
			}
			// Reload config so the rest of writeCheckpoint sees
			// next.projected == true and takes the projection path.
			if reloaded, lerr := config.Load(projectRoot); lerr == nil {
				cfg = reloaded
			}
		}
	}
```

**Why:** turns the gate from "refuse and tell the user to run X" into "do X, silently." The failure branch keeps the exact no-clobber safety AC-14 was protecting (NEXT.md untouched; never the placeholder overwrite), but with an actionable message instead of a raw incantation in the success case. After a successful migration, the reloaded `cfg.NextProjected()` is true, so the existing projection write path (`checkpoint.go:147-154`) runs and NEXT.md is regenerated from the graph (now including the just-ingested content).

### Change 3 (team mode) — reconcile the gate/migration target

**File:** `internal/cli/next_migrate.go` (and/or the gate)

**Why:** Secondary Defect 1 — the gate inspects `resolveNextPath(...)` (per-user in team mode) but `migrateToProjection` reads `.hero/NEXT.md`. Two acceptable resolutions; pick during delivery:
- **(a)** Pass the gate's `nextPath` into `migrateToProjection` so it captures/ingests the exact file the gate flagged, OR
- **(b)** Scope auto-migration to solo mode (mirror the `writeUserHandoffFile` team-mode no-op at `checkpoint.go:321-323`) and leave team-mode migration as an explicit operator step, keeping a clear (non-raw) message in that narrow case.

Recommend **(a)** unless team-mode migration has additional structure the manual command never handled — verify before committing.

### Recommended trigger point

**Primary: (a) lazily, at the first projected checkpoint where the gate currently fires.** This is the most frequent trigger (every Stop hook), so it self-heals any unmigrated repo on the very next turn with zero user action — the strongest mission fit. The migration is idempotent, so repeated checkpoints after the first are no-ops on the flag check.

**Also do (b): `hero upgrade`.** Workspaces created before projection existed (the version-skew population this workspace itself reports) should transition proactively at upgrade rather than waiting for the next checkpoint. Low cost, removes a class of "first checkpoint after upgrade does extra work" surprise. Wire `migrateToProjection(projectRoot, cfg, out)` into the upgrade flow (`internal/cli/upgrade.go:runUpgrade`) guarded by `!cfg.NextProjected()`.

**Recommendation: both, primary = checkpoint.** Checkpoint is the guarantee; upgrade is the proactive nicety.

### Failure-mode contract (explicit)

On auto-migration failure at the checkpoint trigger:
1. NEXT.md MUST be left byte-for-byte untouched (never overwritten with `nextPlaceholder`).
2. `next.projected` MUST remain `false` (migration is transactional enough that a partial flip without capture doesn't strand the user — verify `setNextProjected` is the last step, which it is: `next_migrate.go:109`).
3. The user-facing message MUST be actionable and human ("automatic NEXT.md migration failed: <reason>; your NEXT.md was left untouched; run `hero next migrate-to-projection` to retry"), NOT a bare CLI incantation presented as the normal path.

---

## Test Plan

### Existing test review
| Test | File | Disposition |
|------|------|-------------|
| `Test_writeCheckpoint_PreFlightGate_RefusesLegacyMarkers` | `internal/cli/checkpoint_test.go:527` | **Invert** — assert auto-migration runs and succeeds (no error, `next.projected` flipped, Note captured) instead of asserting refusal. |
| `Test_writeCheckpoint_PreFlightGate_RefusesLegacyHeaders` | `internal/cli/checkpoint_test.go:564` | **Invert** — same. |
| `Test_writeCheckpoint_PreFlightGate_AllowsWhenMigrated` | `internal/cli/checkpoint_test.go:597` | Keep — already-migrated repos skip the gate; behavior unchanged. |
| `Test_writeCheckpoint_PreFlightGate_AllowsCleanUnmigrated` | `internal/cli/checkpoint_test.go:621` | Keep — clean/placeholder-only NEXT.md still passes through with no migration. |
| `TestExtractAskAndSuggestion_*`, `TestFirstQuoteOrText_*` | `internal/cli/next_migrate_test.go` | Keep — migration internals unchanged. |

### New tests
1. **Auto-migration on legacy headers** — set up unmigrated repo with `## Just finished` real content; call `writeCheckpoint`; assert: (a) no error, (b) `hero.json` `next.projected == true`, (c) a `Note` node with reason `next-as-projection migration; preserving pre-projection content` exists in the graph, (d) NEXT.md is now a graph projection (not the original hand text verbatim, but the captured content is recoverable from the Note).
2. **Auto-migration on legacy machine-state markers** — same with `<!-- BEGIN HERO MACHINE STATE -->` present.
3. **Silence** — auto-migration on a `--quiet` checkpoint emits nothing to stdout (the auto path uses `io.Discard`); assert no migration progress lines leak. (Extends `TestNextCheckpointQuietIsSilent`.)
4. **Migration-failure preserves content** — inject a failure (e.g. read-only `hero.json` so `setNextProjected` fails, or a graph-open error); assert: (a) `writeCheckpoint` returns an error whose message is the actionable human string (NOT the bare "run `hero next migrate-to-projection` first" incantation, and NOT a placeholder), (b) NEXT.md is byte-for-byte unchanged, (c) `next.projected` is still false.
5. **Idempotence** — run `writeCheckpoint` twice on the unmigrated repo; second run takes the already-projected path with no second Note capture.
6. **Team-mode path** — depending on the chosen resolution (Change 3): either assert the per-user file's legacy content is captured (option a), or assert solo-only scoping with a clear team-mode message (option b).

### Regression scope
- `writeProjectedNextMD` / `writeUserHandoffFile` / `projectSnapshot` run after the (now-successful) migration — confirm the post-migration projection path still produces a valid NEXT.md on the first turn.
- The CI drift gate (`.github/workflows/test.yml`) runs `hero next checkpoint --quiet` then `git diff --exit-code .hero/NEXT.md` — confirm auto-migration in CI on an unmigrated fixture doesn't cause spurious drift failures (CI repos in this codebase are already migrated; verify no fixture regresses).
- `.gitattributes` and `hero.json` are mutated by the migration mid-checkpoint — confirm the pre-commit auto-stage hook (`pre-commit-auto-stage-next`) stages these so the transition travels with the commit.

---

## Kickoff

You're picking up a fix for a confirmed `design` bug in Hero's NEXT-projection subsystem. The diagnosis is complete and lives in `.hero/planning/bugs/next-projection-gate-punts-migration-to-user/spec.md` — read it first.

**The bug:** in an unmigrated repo (`next.projected == false` with legacy NEXT.md content), `hero next checkpoint` refuses every Stop hook with `unmigrated NEXT.md detected (...) — run `hero next migrate-to-projection` first`. The pre-flight gate at `internal/cli/checkpoint.go:126-133` punts a safe, non-interactive migration back onto the user instead of doing it. This violates Hero's mission (inject context automatically, without anyone asking).

**The fix (do NOT just delete the gate):**
1. Factor the migration body out of `runNextMigrateProjection` (`internal/cli/next_migrate.go:41`) into a reusable `migrateToProjection(projectRoot, cfg, out io.Writer)` — keep the cobra command as a thin wrapper. Confirm it stays idempotent and that `captureNextSnapshot` still runs first (content preservation).
2. In `writeCheckpoint` (`checkpoint.go:126`), replace the `return "", fmt.Errorf(...)` with: call `migrateToProjection(projectRoot, cfg, io.Discard)`; on success reload config so the projection path runs; on failure return an *actionable human* error and leave NEXT.md untouched (never the placeholder write). The failure contract is spelled out in the spec's "Failure-mode contract" — honor all three points.
3. Resolve the team-mode path mismatch (Secondary Defect 1): the gate checks `resolveNextPath` (per-user in team mode) but the migration hardcodes `.hero/NEXT.md`. Decide between passing the gate's path in vs. solo-only scoping — see Change 3.
4. Also wire `migrateToProjection` into `hero upgrade` (`internal/cli/upgrade.go:runUpgrade`) guarded by `!cfg.NextProjected()` for proactive transition of version-skewed workspaces.

**Tests:** invert `Test_writeCheckpoint_PreFlightGate_Refuses*` (they assert refusal today — `checkpoint_test.go:527,564`), keep the `Allows*` tests, and add the six new tests listed in the spec's Test Plan — especially the migration-failure-preserves-content test (point 4) and the silence test.

**Do NOT** tighten `sectionHasRealContent` detection — the auto-migration approach moots the over-eager detection concern; it's flagged as out-of-scope follow-up only.

When done, update AC-14 in `.hero/specs/next-as-projection/spec.md:451-457` and §5 + the "What this locks in" bullet in `.hero/specs/decisions/next-as-projection-architecture/spec.md` to reflect "auto-migrate, don't refuse" — the old text says the gate must keep firing forever, which this fix supersedes.

---

## Recap
Hero's NEXT-projection pre-flight gate (`checkpoint.go:126-133`, AC-14) refuses every checkpoint in an unmigrated repo and tells the user to run `hero next migrate-to-projection` by hand — even though that migration is already non-interactive and content-preserving. It's a design/process defect (an over-cautious transition that conflated "silently wipe content" with "silently migrate content"). The fix is to call the existing migration logic automatically at the checkpoint trigger (and at `hero upgrade`), surfacing a message only on genuine failure while keeping the no-clobber safety. Severity: high — the subsystem punts its own designed work back onto the user, directly against Hero's mission.
