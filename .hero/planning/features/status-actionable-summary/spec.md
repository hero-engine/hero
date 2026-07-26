---
title: "Actionable Status Summary — In Progress, Upcoming, and Recently Completed"
slug: status-actionable-summary
type: feature
status: planning
domain: engineering
size: medium
priority: medium
horizon: now
tags: [cli, status, workflow, usability]
created: 2026-07-25
---

# Actionable Status Summary — In Progress, Upcoming, and Recently Completed

## Context

`hero status` describes itself as the current state of the workspace and the
documentation says it shows actionable work by horizon. In a mature Hero
workspace, however, the human-readable command discovers the full corpus and
prints every completed work spec plus every convention, decision, rule,
external entry, context entry, note, and intake item line by line.

In this repository that means the useful operational state is followed by 321
completed specs and the rest of the knowledge corpus. The output answers “what
has Hero ever stored?” more strongly than “what is happening, what is next, and
what just finished?” It can also place non-completed files found under the
archive directory into active lifecycle groups, making stale archive
frontmatter look like current work.

Hero already has the primitives needed for a concise view:

- `Spec.CompletedAt` is the authoritative completion timestamp; 335 of the 346
  archived specs in this workspace already carry it.
- `spec.Select` and its priority sort define the queue ordering: pinned first,
  then lifecycle, horizon, and recency.
- `hero list` is the explicit, filterable corpus browser for full planning,
  completed, intake, and knowledge listings.
- `hero status --json` is a machine-readable surface and must remain stable for
  scripts and adapters.

## Goal

Make the default human-readable `hero status` a compact operational briefing.
It should lead with lifecycle counts, show all work genuinely in progress, show
a bounded and priority-ranked upcoming list, preserve waiting and failure
signals, and show only the five most recent timestamped completions. Full
history and reference material remain available through `hero list`, while
existing JSON and horizon-filter contracts remain compatible.

## Kickoff

Reworks human-readable `hero status` around counts, current work, ranked
upcoming work, and five recent completions instead of dumping the corpus.

**Status:** planning — output contract is designed; no implementation yet.

**Pick up at:** introduce a collected human status view in `status.go`, then
lock its grouping, bounds, ordering, and compatibility in focused tests.

→ `/deliver status-actionable-summary`

**Files:** `internal/cli/status.go`, `internal/cli/status_test.go`,
`internal/spec/select.go`, `web/docs/src/cli/search-and-context.md`
**Skip:** changing `status --json` or redefining `--all`; both remain compatible.

## Problem

The current human output has four distinct problems:

1. **Archive volume dominates current state.** Completed work and knowledge
   entries are rendered as exhaustive lists even though their counts are enough
   for an operational status check.
2. **Planning volume is also unbounded.** A workspace with dozens of active
   horizon items still produces a long `Planning` section with no canonical
   priority ranking or ready/blocked distinction.
3. **Lifecycle labels do not answer the user’s questions directly.**
   `Delivering`, `In Review`, `Handed Back`, `Planning`, and `Awaiting Peer`
   are accurate primitives, but the primary mental model is simpler:
   `In progress`, `Upcoming`, and `Waiting`.
4. **Archive inconsistencies can masquerade as active work.** Discovery scans
   both `.hero/planning/` and `.hero/specs/`; stale non-completed frontmatter
   under `.hero/specs/` currently enters an active group instead of being
   identified as an integrity problem.

## Approach

### Default information hierarchy

After any existing workspace/dialect/Mail preamble that is actually present,
render one compact count block before detailed work:

```text
Work: 6 in progress · 63 upcoming (54 ready, 9 blocked) · 9 waiting · 321 completed
Other: 4 intake · 47 knowledge · 56 hidden by horizon
```

Counts and lists use the same collected view so they cannot drift.

The detailed sections then answer the operational questions in this order:

1. **In progress** — all locally actionable active work:
   - `handed_back` first because the peer has returned the ball;
   - `delivering`;
   - `in-review`.
2. **Upcoming** — planning work in the selected horizon, bounded to ten rows.
   Dependency-ready items sort before blocked items. Within each partition,
   reuse Hero’s canonical priority ordering rather than introducing a second
   queue policy. Mark blocked rows explicitly.
3. **Waiting** — `handed_off` and `awaiting_peer`, bounded to ten rows with an
   omitted-count hint when necessary.
4. **Recently completed** — at most five work specs with a non-zero
   `CompletedAt`, newest first with slug, type, title, and relative completion
   age.

Keep smoke failures, active async jobs, connection health, version mismatch,
and peer-reconciliation notices visible. They are operational signals rather
than corpus listings.

### Counts and category semantics

- `in progress` counts `handed_back`, `delivering`, and `in-review` work in the
  selected active horizon.
- `upcoming` counts planning work in the selected horizon and separately
  computes ready versus blocked using the canonical dependency predicates.
- `waiting` counts `handed_off` and `awaiting_peer` work in the selected
  horizon.
- `completed` is the workspace-wide count of completed work and is not reduced
  by a horizon filter; horizon no longer has operational meaning after work is
  complete.
- `intake` and `knowledge` are workspace-wide counts. Neither is rendered
  entry-by-entry in the default human view.
- `hidden by horizon` keeps the existing open-work explanation and remains
  absent when `--all` or an explicit `--horizon` is used.

### Bounds and escape hatches

Use fixed defaults rather than adding configuration:

- Upcoming: ten rows.
- Waiting: ten rows.
- Recently completed: five rows.
- In progress: unbounded because every actively executing or locally returned
  item is material to the current state.

When a section is truncated, print the omitted count and the exact command for
the full view:

- Upcoming: `hero list --status planning --sort priority`
- Waiting: `hero list --status handed-off,awaiting-peer`
- Completed: `hero list --status completed --sort recency`
- Intake: `hero list --type intake`
- Knowledge: `hero list --type convention,decision,rule,external,context,note`

If a suggested command is not accepted by the current `hero list` parser,
adjust it to the nearest existing typed filter during delivery; do not emit a
command that Hero itself cannot run.

### Completion ordering

Only `completed_at` / `Spec.CompletedAt` qualifies a spec for the recent list.
Sort descending by that timestamp, then by slug for deterministic ties.
Historical completions without a timestamp still contribute to the completed
count but do not receive a fabricated “recent” position. Do not fall back to
filesystem modification time: checkout, archive movement, backfill, and
post-delivery edits make it an unreliable completion signal.

If there are completed specs but none with a timestamp, omit the item rows and
print the completed count plus the full-history command.

### Archive integrity

Classify active work from the planning tree, not merely from frontmatter:

- A non-completed work spec under `.hero/specs/` must not appear under
  `In progress`, `Upcoming`, or `Waiting`.
- A completed work spec still under `.hero/planning/` may contribute to
  completed history but is an archive inconsistency.
- When either mismatch exists, print one concise warning:
  `Archive inconsistencies: N — run hero check for details`.

This status change surfaces the mismatch but does not repair or rewrite specs.
`hero check` remains the authority for diagnosis and reconciliation.

### Compatibility

- Keep `hero status --json` byte-shape compatible: the existing top-level
  `workspace`, `hero_dir`, `specs`, and optional `mail` fields and per-spec
  fields remain unchanged.
- Keep `--all` meaning “include someday and parking work.” It expands open-work
  counts and the bounded Upcoming/Waiting candidates; it does not dump the
  completion archive or knowledge corpus.
- Keep `--horizon` filtering open work to the selected horizon. Completed,
  intake, and knowledge summary counts remain workspace-wide.
- Do not add a new `--completed` flag. `hero list --status completed` is the
  canonical archive browser.

## Changes

1. **Refactor human status collection in `internal/cli/status.go`.**
   - Introduce a small internal view model or helper that partitions discovered
     specs into in-progress, upcoming-ready, upcoming-blocked, waiting,
     completed, intake, knowledge, auto-context, and archive-inconsistency
     counts.
   - Keep collection separate from terminal rendering so counts, truncation,
     ordering, JSON compatibility, and tests are independently understandable.
   - Detect planning/archive location from each spec’s resolved path without
     changing global discovery semantics.

2. **Replace exhaustive corpus rendering in `internal/cli/status.go`.**
   - Render the top count block and the four operational sections in the order
     defined above.
   - Reuse `printSpecGroup` formatting where it remains useful, but add bounded
     rendering and omitted-count/query hints rather than duplicating ad hoc
     loops.
   - Remove default line-by-line output for the completed archive, knowledge
     types, and intake.
   - Preserve smoke failures, async jobs, connection health, peer completion,
     workspace context, dialect, Mail summary, and version reporting.

3. **Reuse canonical selection semantics from `internal/spec/select.go`.**
   - Use the existing dependency-ready/blocked predicates and priority sort.
   - Export or add the smallest focused helper only if `status.go` cannot reuse
     the canonical ordering through the current public selector.
   - Do not create a status-only ranking algorithm that can drift from
     `hero queue`.

4. **Expand `internal/cli/status_test.go`.**
   - Add focused fixtures for every category, more than ten upcoming/waiting
     entries, ready/blocked ordering, five-plus completions, missing
     `completed_at`, timestamp ties, empty state, horizon flags, knowledge and
     intake suppression, and archive-path inconsistencies.
   - Assert omitted-count hints contain valid, existing CLI invocations.
   - Retain existing smoke-failure and connection/async behavior coverage.

5. **Protect the JSON contract.**
   - Add or extend tests for `hero status --json` proving the schema and horizon
     filtering are unchanged by the human presentation refactor.
   - Ensure human-only bounds do not remove entries from JSON.

6. **Update command help and user documentation.**
   - Change the `statusCmd.Long` text in `internal/cli/status.go` to describe the
     compact default and explicit `hero list` escape hatches.
   - Update `web/docs/src/cli/search-and-context.md` and the concise status
     references in `README.md` if needed.
   - Include one representative output example so users understand
     `In progress`, `Upcoming`, `Waiting`, and `Recently completed`.

## Boundaries

- No changes to spec lifecycle states, persistence, graph ingestion, archive
  movement, or completion stamping.
- No automatic repair of stale archive frontmatter; only a count and
  `hero check` pointer.
- No change to `hero queue` ranking policy or `hero list` filtering semantics.
- No change to the `hero status --json` schema.
- No pagination, interactive terminal UI, configurable row limits, or new
  status flags in this spec.
- No harness-facing instruction or `hero install` change; the six-target
  propagation tripwire does not apply to this CLI-only presentation change.

## Risks

- **Count/list drift:** independently computed totals and lists would recreate
  ambiguity. Build both from one collected view and test exact counts.
- **Ranking drift:** copying `priorityLess` into the CLI would eventually make
  Status and Queue disagree. Reuse the canonical selector or expose a narrow
  helper.
- **Misleading historical recency:** using file modification time would make
  old work appear newly completed. Timestamped-only recent output is deliberate
  even when fewer than five rows are available.
- **Archived non-completed work:** hiding it without any signal could conceal a
  real reconciliation problem. The explicit inconsistency warning preserves
  visibility without claiming it is current.
- **Automation breakage:** terminal parsers may exist despite `--json`. The
  human output is intentionally changing, but JSON must remain stable and docs
  must direct automation to it.
- **Suggested-command drift:** escape-hatch text becomes harmful if it contains
  unsupported enum spellings. Invocation-validation tests must cover every
  emitted command.

## Acceptance Criteria

- **AC-1:** WHEN a user runs human-readable `hero status` THE SYSTEM SHALL print top-level counts for in-progress, upcoming-ready, upcoming-blocked, waiting, completed, intake, knowledge, and horizon-hidden work before detailed spec lists
- **AC-2:** WHEN active work exists THE SYSTEM SHALL render one `In progress` section containing all handed-back, delivering, and in-review planning-tree specs in that precedence order
- **AC-3:** WHEN planning work exists THE SYSTEM SHALL render at most ten `Upcoming` rows with dependency-ready work before blocked work and canonical Hero priority ordering within each partition
- **AC-4:** WHEN more than ten upcoming or waiting specs match THE SYSTEM SHALL print the omitted count and a runnable `hero list` command for the complete filtered view
- **AC-5:** WHEN completed work has authoritative completion timestamps THE SYSTEM SHALL render at most the five newest items under `Recently completed`, ordered by `CompletedAt` descending and slug ascending for timestamp ties
- **AC-6:** IF a completed spec lacks `CompletedAt` THEN THE SYSTEM SHALL include it in the completed total without fabricating a recent-list timestamp from file modification time
- **AC-7:** WHEN knowledge or intake entries exist THE SYSTEM SHALL summarize them by count without printing every entry in the default human status output
- **AC-8:** WHEN smoke failures, active async jobs, connection health, peer reconciliation, workspace scope, Mail state, or version mismatch information exists THE SYSTEM SHALL preserve those operational signals in the human status output
- **AC-9:** IF a non-completed work spec is found under `.hero/specs/` or a completed work spec remains under `.hero/planning/` THEN THE SYSTEM SHALL exclude the stale archive entry from active groups and print one archive-inconsistency count with a `hero check` pointer
- **AC-10:** WHERE `--all` OR `--horizon` IS USED THE SYSTEM SHALL apply that option to open-work counts and candidates while leaving workspace-wide completed, intake, and knowledge counts semantically unchanged
- **AC-11:** WHEN `hero status --json` is used THE SYSTEM SHALL preserve the existing JSON fields, unbounded spec collection, and horizon-filter behavior
- **AC-12:** WHEN no work or corpus entries exist THE SYSTEM SHALL render zero counts and a concise empty operational state without exhaustive empty sections
- **AC-13:** THE SYSTEM SHALL validate every CLI command emitted as an archive or full-list hint against Hero’s real command tree

## Validation

### Unit and command-output tests

Add table-driven tests in `internal/cli/status_test.go` covering:

- exact category counts from a mixed lifecycle fixture;
- all in-progress rows and their lifecycle precedence;
- ready-before-blocked Upcoming ordering and the ten-row cap;
- waiting cap and omitted-count text;
- five timestamped recent completions, deterministic ties, and missing
  timestamp exclusion;
- completed, intake, and knowledge counts without exhaustive rows;
- archive-path lifecycle mismatches and the `hero check` warning;
- empty workspace output;
- default, `--all`, and explicit `--horizon` behavior;
- retained smoke-failure, async, connection, Mail, and version signals;
- unchanged `--json` shape and lack of human truncation in JSON.

### Invocation validation

Resolve every emitted hint through the real Cobra command tree or the existing
markdown invocation validator. The test must fail if `hero list` does not
support the exact status/type/sort spellings printed to users.

### Manual exercise

1. Run `hero status` in this repository.
2. Confirm the first stable block is the count summary.
3. Confirm all current executing/review/returned work appears under
   `In progress`.
4. Confirm `Upcoming` contains no more than ten priority-ranked rows and marks
   blocked work.
5. Confirm only five recent completions appear and their dates match
   `completed_at`.
6. Confirm no full completed, knowledge, or intake corpus dump remains.
7. Run every printed `hero list` hint and confirm it returns the expected full
   view.
8. Run `hero status --json` before and after the change against the same
   fixture and compare schemas and item counts.
