---
title: "Read contract v1 — Hero-side semantics for hero_work / hero_spec / hero_handoff"
slug: read-contract-v1
type: decision
status: accepted
priority: critical
domain: engineering
created: 2026-10-06
parent: hero-read-contract
---

# Read contract v1

## Kickoff

This decision records the Hero-side semantics for the read contract that hero-harness drafted. The shapes are theirs (`../hero-harness/.hero/planning/initiatives/hero-harness-mvp/mvp-3a-hero-service/spec.md`); the meaning is Hero's, and is defined below. It was decided with Hero defaults on 2026-10-06 and sent to hero-harness as the v1 proposal. After v1, changes are additive only.

## Decision

**Shapes.** Adopt the requester's shapes for `HeroWork`, `WorkItem`, `NextStep`, `PolishItem`, `SuggestedItem`, `hero_spec` and `hero_handoff` unchanged, with the semantics below. `schema_version: 1`. All three tools are `readOnlyHint: true`, take the project from the server's root, re-index when stale, write nothing, and return JSON.

### Work specs and lanes

`items` holds every spec of type `feature`, `bug`, `enhancement`, `initiative` or `epic`, plus `decision` specs that are children of an initiative. Knowledge types are never items.

**Designed.**
- feature/enhancement: has a `## Changes` section and at least one acceptance criterion.
- bug: has a `## Root Cause` (or `## Root Cause Analysis`) section and a `## Changes` / `## Fix` / `## Suggested Fix Approach` section.
- initiative: has at least one child: declared in frontmatter or a children table, or a work spec naming it as `parent`.
- decision: has a `## Decision` section.

**Unmet dependency.** An outgoing `depends-on` or `blocks` edge to a target that is not finished (`spec.IsFinished`). This is the same rule `internal/projection` uses for NEXT.md.

| Lane | Rule (first match wins) |
|---|---|
| `recently_done` | Finished, with `completed_at` (or file mtime when absent) within `recent_days` (default 14) |
| `in_progress` | Status `delivering`, `in-review`, `regressed`, `handed_off`, `awaiting_peer` or `handed_back`; or an initiative with ≥1 child finished or in progress and not itself finished |
| `ready` | Status `planning` / `proposed`, designed, and no unmet dependency |
| `designed` | Status `planning` / `proposed`, designed, with an unmet dependency |
| `none` | Everything else: undesigned stubs, older finished work, superseded/rejected/merged |

`progress` for initiatives is children finished / children total, using the same children as "designed" (declared plus work specs that name it as parent). It is `null` for other types.

`priority` / `severity` normalization:
- `critical`: p0, critical, blocker, highest.
- `high`: p1, high, major.
- `medium`: p2, medium, moderate, normal.
- `low`: p3, p4, low, lowest, minor, trivial.
- Anything else: `null`.

`depends-on`, `depends_on` and `blocks` edges all count as dependencies.

### Verify state

There is one source of truth: files on disk. It never comes from `events.log`, so reads cannot depend on write-only logs.

`verify.audit` is the verdict of the spec's own audit report (`spec.FindAuditReport`, which validates the slug and staleness): `ship`, `hold`, or `null`. For a **finished** spec the staleness check is ignored, because `hero spec verify` rewrites the status after the audit, so a finished spec is always newer than its report. A slug mismatch never counts.

`verify.state`:
- `passed`: finished, audit `ship`, and every ledger row `DONE`.
- `partial`: finished, but the audit is missing or not SHIP, or the ledger is missing or has rows that `hero spec verify` would not accept. It accepts DONE, or SKIPPED/BLOCKED with a structured sign-off.
- `failed`: audit `hold`, or status `regressed`.
- `not_run`: not finished (an unfinished spec may still report `audit: "ship"` from an earlier round; the verify gate has not run).

`verify` is `null` for types that don't verify: decisions, and containers (initiatives and epics), which finish through their children.

### Next step: one primary action

These rules are kept from the requester:
- Exactly one primary action, never Diagnose and Deliver together.
- `Delivered` shows only when `verify.state = passed`.
- A finished item with a weak or missing verify offers **Verify**, never Deliver.

| Type | State | action / label / command | phase (label, state) | enabled / reason |
|---|---|---|---|---|
| feature, enhancement | not designed | design / Design / `/design <slug>` | Planning, ready | true |
| feature, enhancement | designed, unmet dependency | deliver / Deliver / `/deliver <slug>` | Planning, waiting | false, "waits on <slug>[, <slug>]" |
| feature, enhancement | ready | deliver / Deliver / `/deliver <slug>` | Ready, ready | true |
| any work type | delivering, in-review | deliver / Continue / `/deliver <slug>` | Delivering, active (attention if audit `hold`) | true |
| bug | not diagnosed | diagnose / Diagnose / `/diagnose <slug>` | Reported, ready | true |
| bug | diagnosed (per "Designed") | deliver / Fix / `/deliver <slug>` | Diagnosed, ready or waiting | as feature |
| any work type | regressed | diagnose / Diagnose / `/diagnose <slug>` | Regressed, attention | true |
| initiative | no children | design / Compose / `/compose <slug>` | Planning, ready | true |
| initiative | has unfinished children | drive / Drive / `/drive <slug>` | Driving (if in_progress) or Planning, active or ready | true |
| decision | not accepted | design / Decide / `/decide <slug>` | Proposed, ready | true |
| any work type | handed_off, awaiting_peer | deliver / Deliver / `/deliver <slug>` | With peer, waiting | false, "handed off to a peer" (specs do not record the peer alias today) |
| any work type | finished, verify `passed` | `next: null` (clients show **Delivered** from `verify.state = passed`) | — | — |
| feature, bug, enhancement | finished within `recently_done`, verify not `passed` | verify / Verify / `/verify <slug>` | Delivered?, attention | true |
| feature, bug, enhancement | finished before the window, verify not `passed` | `next: null` (historical work; not re-verified) | — | — |
| any | superseded, rejected, merged, accepted decision | `next: null` | Closed, done | — |

`/verify <slug>` is a new thin slash workflow, shipped to every harness target by `next-step-engine`. It runs the cold delivery audit if it is missing or stale, then `hero spec verify <slug>`.

**Extras** never contain the other primary action of the pair (Diagnose ⇄ Deliver):
- Design stage: `Split` (`/split <slug>`).
- Ready or Delivering: `Check against the code` (`/review <slug>`).
- Bug diagnosed: `Challenge diagnosis` (`/challenge <slug>`).

### Revisions and watch globs

- **Item revision:** the first 16 hex characters of the SHA-256 of the spec file bytes, plus each related spec's `slug:status` (sorted), plus the audit verdict and its file mtime, plus the derived lane and verify state. The derived part is included because time alone can move an item out of `recently_done`.
- **Global revision:** the first 16 hex characters of the SHA-256 of the sorted item revisions, polish, suggested, and `hero_version`. The whole corpus is recomputed per read in v1. Budget: p95 under 2 s on this repo (~400 specs), measured by `hero-work-tool`. Incremental computation only if the budget is missed.
- **`watch_globs`:** `.hero/planning/**`, `.hero/specs/**`, `.hero/knowledge/**`, `.hero/NEXT.md`, `.hero/next/**`, `.hero/hero.json`. They never include Hero's runtime files (index/graph DBs, `events.log`, `cache/`, `sessions/`, pidfiles, `QUEUE.md`, `SNAPSHOT.md`).

### Polish (v1)

Polish covers `recently_done` items only.
- `weak_verify`: `verify.state` is not `passed`.
- `bug_against_recent`: an unfinished bug with any relation to the item.
- `open_followups`: an unfinished spec created on or after the item's completion, with any relation to it.
- **`rated_worse` is omitted in v1.** Hero has no rating source. The kind stays reserved and is added only once a source exists.

### Suggested (v1)

- Hero owns the threshold. **The backlog is thin when fewer than 3 items are in `ready` or `in_progress`.**
- Up to 3 picks with `source: queue`: the highest-priority `ready` items, then undesigned stubs.
- When the backlog is thin, the list ends with `{ slug: null, title: "Explore what's next", source: "queue", next: discover / Explore / "/discover" }`.
- `snapshot`, `note` and `pulse` sources are reserved for later, additive additions.

### `hero_spec`

- `body`: the Markdown without frontmatter.
- `relations`: from frontmatter edges.
- `acs`: the parsed acceptance criteria, each `pass`/`fail` from the latest recorded acceptance results in the graph, else `unknown`.

### `hero_handoff`

Returns the handoff file `hero next` shows (team mode: `.hero/next/<user>.md`, else `.hero/NEXT.md`). Amended 2026-10-09 by `next-projection-reads-stale-spec-graph` (additive): `markdown` is the briefing body only, without the managed snapshot block or frontmatter, and `updated_at` and `repo` come from the frontmatter. Per-machine local notes, which `hero next` appends, are excluded: they are not shared state.

### Versioning

- `schema_version` bumps only for breaking changes. v1 allows additive fields and new enum members in `polish.kind` / `suggested.source` only.
- Clients detect missing tools via `tools/list` (`tool_missing`).

## Consequences

- One shared model (`work-item-model`) and one rule table (`next-step-engine`) feed all three tools. `hero list` / `hero_queue` should later converge on it.
- `/verify` becomes a new slash workflow for all eight harness targets, so the `harness-changes-cover-all-targets` tripwire applies.
- hero-harness can build against these semantics now.

## Amendments

- **2026-10-06 (work-item-model cold audit):**
  - Explicit priority/severity synonyms.
  - `depends_on` edges count.
  - Staleness is ignored for finished specs' audits; on the real corpus it would otherwise reject 164 valid audits.
  - The "unknown AC results → partial" clause is dropped; it conflicted with "files on disk only".
  - The revision includes the derived lane and verify state.
  - Verify is offered only within `recently_done`. Otherwise 258 historical specs, which predate audits, would show Verify.
- **2026-10-06 (next-step-engine and work-item-model cold audit, round 2):**
  - `verify.passed` accepts signed-off SKIPPED/BLOCKED ledger rows, matching `hero spec verify` Gate 1, so `/verify` never dead-ends.
  - The diagnosed-bug extra uses action `challenge`.
  - The handed-off reason is generic.
  - A promoted intake never shadows the spec that shares its slug.
- **2026-10-06 (cold audit, round 3):**
  - Ledger sign-offs count toward `passed` only when the signer resolves against `hero spec verify` Gate 1's known signers (`spec.KnownSigners`: git authors plus `ledger.signers`). With no signer set, no sign-off counts.
  - Initiative children include work specs whose `parent` names the initiative.
  - `hero_handoff` excludes per-machine local notes.
- **2026-10-07 (follow-up `followup-epic-parity`, additive within v1):**
  - Epics are modelled exactly like initiatives (children, designed, progress, lane, next step, decision children).
  - Containers (initiative/epic) carry `verify: null`, since they are never audited themselves. Previously they reported `not_run`/`partial`, which the golden schema already allows as `null`.
