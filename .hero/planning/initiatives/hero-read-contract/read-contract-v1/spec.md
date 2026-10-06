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
- initiative: has at least one declared child.
- decision: has a `## Decision` section.

**Unmet dependency.** An outgoing `depends-on` or `blocks` edge to a target that is not finished (`spec.IsFinished`). This is the same rule `internal/projection` uses for NEXT.md.

| Lane | Rule (first match wins) |
|---|---|
| `recently_done` | Finished, with `completed_at` (or file mtime when absent) within `recent_days` (default 14) |
| `in_progress` | Status `delivering`, `in-review`, `regressed`, `handed_off`, `awaiting_peer` or `handed_back`; or an initiative with ≥1 child finished or in progress and not itself finished |
| `ready` | Status `planning` / `proposed`, designed, and no unmet dependency |
| `designed` | Status `planning` / `proposed`, designed, with an unmet dependency |
| `none` | Everything else: undesigned stubs, older finished work, superseded/rejected/merged |

`progress` for initiatives is declared children finished / declared children total. It is `null` for other types.

### Verify state

There is one source of truth: files on disk. It never comes from `events.log`, so reads cannot depend on write-only logs.

`verify.audit` is the verdict of the spec's own audit report (`spec.FindAuditReport`, which validates the slug and staleness): `ship`, `hold`, or `null`.

`verify.state`:
- `passed`: finished, audit `ship`, and every ledger row `DONE`.
- `partial`: finished, but the audit is missing, or ledger rows are signed-off SKIPPED/BLOCKED, or some AC results are unknown.
- `failed`: audit `hold`, or status `regressed`.
- `not_run`: not finished, with no audit.

`verify` is `null` for types that don't verify (decision).

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
| any work type | handed_off, awaiting_peer | deliver / Deliver / `/deliver <slug>` | With peer, waiting | false, "with peer <alias>" when known |
| any work type | finished, verify `passed` | `next: null` | Delivered, done | — |
| feature, bug, enhancement | finished, verify not `passed` | verify / Verify / `/verify <slug>` | Delivered?, attention | true |
| any | superseded, rejected, merged, accepted decision | `next: null` | Closed, done | — |

`/verify <slug>` is a new thin slash workflow, shipped to every harness target by `next-step-engine`. It runs the cold delivery audit if it is missing or stale, then `hero spec verify <slug>`.

**Extras** never contain the other primary action of the pair (Diagnose ⇄ Deliver):
- Design stage: `Split` (`/split <slug>`).
- Ready or Delivering: `Check against the code` (`/review <slug>`).
- Bug diagnosed: `Challenge diagnosis` (`/challenge <slug>`).

### Revisions and watch globs

- **Item revision:** the first 16 hex characters of the SHA-256 of the spec file bytes, plus each related spec's `slug:status` (sorted), plus the audit verdict and its file mtime.
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

Returns the NEXT.md projection that `hero next` prints, as `markdown`, with `updated_at` from its frontmatter.

### Versioning

- `schema_version` bumps only for breaking changes. v1 allows additive fields and new enum members in `polish.kind` / `suggested.source` only.
- Clients detect missing tools via `tools/list` (`tool_missing`).

## Consequences

- One shared model (`work-item-model`) and one rule table (`next-step-engine`) feed all three tools. `hero list` / `hero_queue` should later converge on it.
- `/verify` becomes a new slash workflow for all eight harness targets, so the `harness-changes-cover-all-targets` tripwire applies.
- hero-harness can build against these semantics now.
