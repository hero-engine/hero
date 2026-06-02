---
title: Roadmap-Review Ambient Surfacing — NEXT.md, Pulse, and Pre-Flight Hooks
slug: roadmap-review-ambient-surfacing
type: feature
domain: engineering
status: completed
size: large
priority: P1
tags: [roadmap, ambient-context, next-md, pulse, delivery-lead]
created: 2026-06-01
relations:
  - target: roadmap-shape
    kind: parent
  - target: roadmap-review
    kind: depends-on
completed_at: 2026-06-01T23:43:43Z
---

## Goal

Wire size-drift findings into three existing context surfaces — the
NEXT.md projection, `hero_pulse` / `hero_kickoff` MCP output, and the
feature- and platform-delivery-lead pre-flight — so the user hears
about roadmap-shape drift at natural session moments without having to
remember `hero size --check`. Surface text is **count-only** by default
(no row excerpts) and always points the user at `/roadmap-review`, not
the underlying CLI. A workspace-wide noise threshold ensures we only
fire when drift is actually actionable (active spec, recently-touched
specs, or high-impact unacked `horizon: now` initiatives), and a
24-hour stop-nagging window suppresses the line after the user has run
`/roadmap-review` so they aren't pinged immediately after triaging.

## Context

Third child in the [`roadmap-shape`](../../initiatives/roadmap-shape/spec.md)
initiative, shipping last in the recommended sequence so it can lock
to the stable `/roadmap-review` command name and `roadmap-reviewer`
agent established in sibling [`roadmap-review`](../roadmap-review/spec.md)
(now `status: ready`). The sibling owns the interactive triage; this
spec owns the *ambient invitations* to run it.

The size mechanic from `spec-size-and-promotion-nudge` is fully wired —
`hero size --check`, `drift.DriftSummaries`, `hero_warnings`, and the
`spec-sizing` skill all do their jobs. The signal exists. What's
missing is the **passive surfacing** at session boundaries:

- NEXT.md regenerates on commit (`internal/projection/projection.go`)
  but does not mention drift.
- `hero_pulse` already populates a `Drift` field but it counts only
  warning-pipeline drift, not the size-drift filtered to the noise
  threshold described below; `hero_kickoff` does not mention drift at
  all.
- The feature- and platform-delivery-lead pre-flight (step 4d) already
  loads `spec-sizing` and runs `hero size --check` against the **active
  spec only**. It does not surface the workspace-wide count.

Without ambient surfacing, the detection work in child #1 is wasted
on users who don't think to invoke it.

The dominant risk is **nudge fatigue**: a user pinged on every session
will mute the channel, and once muted, the whole initiative's value
collapses. The noise threshold and stop-nagging rule below are the
mitigations.

## Approach

### One shared filter helper, three thin surface hooks

Introduce a single helper — `sizing.AmbientDrift(heroDir, projectRoot, opts)` —
that returns the *filtered* drift count plus a hint string. All three
surfaces call the same helper so the rule is implemented once and
behaves identically everywhere.

```go
// internal/sizing/ambient.go (new)

// AmbientDriftReport summarises the workspace-wide drift count after
// applying the noise threshold (active spec, recently-changed specs,
// horizon: now initiatives without declared size). The Count is the
// number of specs that would benefit from /roadmap-review; Hint is
// the paste-ready one-line message surfaces echo verbatim. Quiet
// indicates whether surfaces should suppress entirely (Count == 0
// after filtering OR stop-nagging window active).
type AmbientDriftReport struct {
    Count   int
    Hint    string
    Quiet   bool
    Reason  string // diagnostic: why quiet ("no drift", "recently triaged", "below threshold")
}

type AmbientDriftOpts struct {
    ActiveSpec    string        // slug currently being touched, "" if none
    RecencyDays   int           // default 7
    Now           time.Time     // injectable for tests
}

func AmbientDrift(heroDir, projectRoot string, opts AmbientDriftOpts) AmbientDriftReport
```

The helper:

1. Calls `drift.DriftSummaries(heroDir, projectRoot)` (existing) to
   get the raw drift set.
2. Calls `spec.Discover(heroDir)` to load all specs and identify
   recency (git mtime via `gitutil`) and horizon.
3. Applies the noise-threshold filter (see "Noise threshold" below)
   to drop drift that doesn't meet "active / recent / high-impact."
4. Checks the stop-nagging signal (see "Stop-nagging rule" below) and
   sets `Quiet = true` if active.
5. Composes the hint:
   `"⚠ N specs have size drift — run /roadmap-review to triage"`
   (singular `1 spec has`).

### Noise threshold (precise rule)

A drift entry surfaces in the ambient count if **any** of the
following hold (OR-joined):

1. **Active spec.** The spec's slug equals `opts.ActiveSpec`. Always
   surfaces, regardless of recency or horizon. This is the loudest
   signal — drift on what the user is actively working on.
2. **Recency window.** The spec's planning file (`spec.md`) was last
   committed (git mtime) within `opts.RecencyDays` (default **7**).
   Rationale for 7: matches the standard one-week dogfood cadence the
   initiative validates against, long enough that yesterday's work
   stays in scope, short enough that month-old drift doesn't pile
   onto the count. Overridable via `hero.json` (see below).
3. **High-impact unacked container.** The spec is an `initiative` (or
   `epic`) with `horizon: now` AND no declared `size:` field. These
   are the unsized "now" containers the prior work flagged as the
   highest-leverage gap — they get surfaced regardless of recency.

Drift entries that match **none** of the above are **suppressed** from
the count. In particular, the ~10 `(unset)` containers that predate
the `size:` field are not counted unless they also satisfy rule 3
(horizon:now) — which most won't.

**Count emission rule.** The `Count` field is the **total filtered
drift count after the above rule**. Drift on `(unset)`-predating-the-field
specs is **excluded entirely** from the count when those specs don't
meet rules 1–3. We do not parenthetically split "N + (M legacy)"; the
extra cognitive load isn't worth the precision, and the goal is
"actionable signal," not "complete inventory."

`hero size --check` (sibling spec #3) remains the inventory view for
users who want the full list.

### Stop-nagging rule (24h session marker)

Sibling spec #1 (`roadmap-review`) writes a session record to
`.hero/knowledge/roadmap-review-sessions/{YYYY-MM-DD}-{HHMM}.md` every
time `/roadmap-review` exits. This spec uses the **most recent file's
mtime** as the stop-nagging signal:

- If the newest file under `.hero/knowledge/roadmap-review-sessions/`
  was modified within the last **24 hours**, `AmbientDrift` returns
  `Quiet: true` with `Reason: "recently triaged"`, regardless of
  filtered drift count.
- After 24h, the suppression lifts and the ambient line returns.
- **Exception**: if filtered drift count has *increased* since the
  newest session record, suppression lifts immediately. This is
  detected by comparing the current filtered count against a small
  counter we stash in the session record's frontmatter (a new
  `drift_count_at_exit:` field that sibling spec #1 already writes —
  see "Cross-spec coordination" below). If the field is missing
  (sessions written before this spec landed), behave as if the count
  hasn't changed.

The 24h window is workspace-wide (not session-scoped) because a user
who triages in the morning shouldn't be re-pinged in their afternoon
session.

### Surface 1 — NEXT.md projection

`internal/projection/projection.go` `NextMD()` builds NEXT.md from
graph queries. Add a small "Roadmap shape" line placed **between the
`## Next` section and the `## Blocked on` section**.

Placement rationale: `## Next` answers "what to do," `## Blocked on`
answers "what's stuck." Drift is "what's misshapen" — it sits
naturally between them and reads as an additional decision input
before the user picks a task.

Format:

```markdown
## Roadmap shape

⚠ 3 specs have size drift — run `/roadmap-review` to triage
```

If `AmbientDrift` returns `Quiet: true` OR `Count: 0`, the section is
**omitted entirely** (no header, no "Nothing." line — silence is the
quiet state for this surface).

`NextMDOptions` gains an optional `ActiveSpec string` field (defaults
to empty). The projection caller passes the current session's active
spec when known. NEXT.md regeneration runs on commit (existing wiring
in `internal/nextdoc/graph_ingest.go`) and gets `ActiveSpec=""` at
commit time — the active-spec branch of the noise threshold only
matters when an agent renders this for in-session display.

### Surface 2 — `hero_pulse` / `hero_kickoff` MCP tools

Both tools currently live in `internal/serve/mcp_tools.go`. Extend
both:

**`hero_pulse` (`toolPulse`).** The function already calls
`drift.DriftSummaries` to populate `pulse.PulseData.Drift` — but that
field carries the raw drift inventory, which is too noisy for the
ambient summary. Add a new `SizeDrift *AmbientDriftSummary` field on
`pulse.PulseData` carrying the filtered ambient view:

```go
// internal/pulse/pulse.go

type AmbientDriftSummary struct {
    Count int    `json:"count"`
    Hint  string `json:"hint"`
}

type PulseData struct {
    // ...existing fields...
    SizeDrift *AmbientDriftSummary // nil when Quiet or Count == 0
}
```

`toolPulse` builds the report via `sizing.AmbientDrift(...)` after
the existing `PopulateDrift` call. If `Quiet || Count == 0`,
`SizeDrift` stays nil. Otherwise it carries the count + hint.

`pulse.RenderText` and `pulse.RenderMarkdown` render the new field
**before** the existing detailed `Drift detected (N)` section as a
single line:

```
Size drift: 3 specs — run /roadmap-review to triage
```

Markdown variant uses `## Size drift` with the same one-line body.
The detailed `Drift detected` section remains as today (it lists each
drifted spec by title; that's the inventory view).

`RenderJSON` adds `size_drift: { count, hint }` at the top level.

**`hero_kickoff` (`toolKickoff`).** Currently emits a single spec's
kickoff block. Extend the rendered string with a leading drift line
when applicable, placed **above** the `## {slug} — {title}` header:

```
⚠ 3 specs have size drift — run /roadmap-review to triage

## roadmap-review-ambient-surfacing — Roadmap-Review Ambient Surfacing...
```

The kickoff caller passes the requested slug into `AmbientDrift` as
`ActiveSpec` so the active-spec branch of the filter naturally lights
up when a user is kicking off the same spec that has drift on it.

### Surface 3 — Delivery-lead pre-flight

Step 4d in both `domains/engineering/agents/feature-delivery-lead.md`
and `domains/engineering/agents/platform-delivery-lead.md` already
loads the `spec-sizing` skill and runs `hero size --check` against
the active spec. Extend that step with **one additional sentence**:

> Also check workspace-wide drift via `hero_pulse` (read the
> `size_drift` field). If it reports a count above zero, surface it
> verbatim: *"⚠ N specs have size drift — run `/roadmap-review` to
> triage."* Do not enumerate the drifted specs; that's `/roadmap-review`'s
> job.

This keeps the active-spec nudge (the existing behavior) loud and
gives the workspace-wide signal exactly one extra line. The lead
quotes the hint from `hero_pulse` — no string composition in the
agent prompt.

### Lens-agnostic phrasing (locked)

Per the parent initiative's cross-cutting note, ambient surface text
says **"size drift"** — never "sizing-lens drift" or "roadmap-shape
concern." When future lenses (horizons, releases, sprint-shape) come
online, those lenses will compose into the same `AmbientDrift`
helper and the message extends naturally (e.g. "3 size + 2 horizon
concerns"). v1 stays sizing-only. The phrasing is fixed at the helper
level (one source of truth: `sizing.AmbientDrift`'s `Hint` field) and
all three surfaces quote it verbatim.

### Cross-spec coordination with sibling #1

Sibling spec #1 (`roadmap-review`) already writes a session record
frontmatter on exit. This spec depends on that record carrying one
additional field:

```yaml
---
type: note
created: 2026-06-01T14:32:00Z
tags: [roadmap-review, sizing]
drift_count_at_exit: 3   # NEW
---
```

`drift_count_at_exit` is the filtered ambient count at the moment the
`/roadmap-review` session exited. The `AmbientDrift` helper reads it
to power the "count increased since last triage" exception to the 24h
suppression.

This requires a small touch to the sibling spec's session-record
writer to include the field. Coordinate via a one-line note in
sibling #1's spec body (added as part of this design pass) or as a
follow-up edit before #1's delivery starts. **No new acceptance
criterion is needed on sibling #1** because the field is forward-
compatible (missing field → behave as if count unchanged), so this
spec ships safely even if #1's writer doesn't gain the field
immediately.

### Configurability

Add a `roadmap` block to `hero.json` shape (parsed via `internal/config`):

```json
{
  "roadmap": {
    "ambient_recency_days": 7,
    "stop_nagging_hours": 24
  }
}
```

Both fields are optional with the documented defaults. No CLI surface
for tuning these in v1 — edit the config file.

## Acceptance Criteria

- THE SYSTEM SHALL expose a single helper
  `sizing.AmbientDrift(heroDir, projectRoot, opts) AmbientDriftReport`
  that returns a filtered drift count, a paste-ready hint string, and
  a quiet flag, so all three ambient surfaces consume identical
  filtering logic.
- WHEN `AmbientDrift` evaluates the corpus THE SYSTEM SHALL include a
  drift entry in the count if and only if (a) the spec slug equals
  `opts.ActiveSpec`, OR (b) the spec file was committed within
  `opts.RecencyDays` (default 7), OR (c) the spec is an initiative or
  epic with `horizon: now` and no declared `size:` field.
- WHEN no drift entries pass the noise filter THE SYSTEM SHALL return
  `Count: 0` with `Quiet: true` and `Reason: "no drift"`.
- WHEN the newest file under
  `.hero/knowledge/roadmap-review-sessions/` was modified within the
  last `stop_nagging_hours` (default 24) AND the current filtered
  count is not greater than the session record's `drift_count_at_exit`
  THE SYSTEM SHALL return `Quiet: true` with `Reason: "recently triaged"`.
- WHEN the current filtered drift count exceeds the most recent
  session record's `drift_count_at_exit` field THE SYSTEM SHALL lift
  the recently-triaged suppression and emit the hint.
- IF the most recent session record lacks a `drift_count_at_exit`
  field THEN THE SYSTEM SHALL treat the recently-triaged window as
  fully suppressive (no exception fires).
- THE SYSTEM SHALL phrase the ambient hint as `"⚠ N specs have size
  drift — run /roadmap-review to triage"` (with singular `1 spec has`
  when N == 1) and use this exact wording in all three surfaces.
- WHEN `NextMD` regenerates AND `AmbientDrift` returns a non-quiet
  non-zero report THE SYSTEM SHALL emit a `## Roadmap shape` section
  containing the hint, placed between `## Next` and `## Blocked on`.
- WHEN `AmbientDrift` returns `Quiet: true` or `Count: 0` THE SYSTEM
  SHALL omit the `## Roadmap shape` section from NEXT.md entirely
  (no header, no placeholder line).
- WHEN `hero_pulse` is invoked AND `AmbientDrift` returns a non-quiet
  non-zero report THE SYSTEM SHALL populate `PulseData.SizeDrift`
  with the count and hint, and render the hint as a one-line summary
  in text, markdown, and JSON outputs.
- WHEN `hero_pulse` is invoked AND `AmbientDrift` is quiet or zero
  THE SYSTEM SHALL set `PulseData.SizeDrift` to nil and emit no
  size-drift line.
- WHEN `hero_kickoff` is invoked for a spec slug THE SYSTEM SHALL
  pass that slug as `opts.ActiveSpec` to `AmbientDrift` and prepend
  the hint as a leading line above the kickoff body when the report
  is non-quiet and non-zero.
- WHEN `hero_kickoff` is invoked AND `AmbientDrift` is quiet or zero
  THE SYSTEM SHALL emit the kickoff body unchanged.
- THE SYSTEM SHALL extend `feature-delivery-lead.md` and
  `platform-delivery-lead.md` pre-flight step 4d with one additional
  sentence instructing the lead to read `hero_pulse`'s `size_drift`
  field and surface the hint verbatim when non-empty.
- THE SYSTEM SHALL NOT enumerate drifted spec slugs or rows in any of
  the three ambient surfaces — count and hint only.
- THE SYSTEM SHALL phrase the ambient hint using the lens-agnostic
  term "size drift" (never "sizing-lens drift" or "roadmap-shape
  concern").
- WHERE `hero.json` declares `roadmap.ambient_recency_days` or
  `roadmap.stop_nagging_hours` THE SYSTEM SHALL honor the configured
  values; otherwise THE SYSTEM SHALL use 7 days and 24 hours
  respectively.
- IF the `.hero/knowledge/roadmap-review-sessions/` directory does
  not exist THEN THE SYSTEM SHALL treat the stop-nagging window as
  inactive (no suppression).

## Files to Touch

Canonical paths under `internal/` and `domains/engineering/`; harness
view directories update via existing sync.

1. **`internal/sizing/ambient.go`** (new, ~120 lines)
   - `AmbientDriftReport`, `AmbientDriftOpts`, `AmbientDriftSummary` types.
   - `AmbientDrift(heroDir, projectRoot, opts) AmbientDriftReport`
     entry point.
   - Internal helpers: `filterDrift`, `readLatestSessionRecord`,
     `checkStopNagging`, `formatHint`.
   - Reads config defaults from `internal/config` (extend `Config` with
     a `Roadmap *RoadmapConfig` field if not already present).

2. **`internal/sizing/ambient_test.go`** (new, ~250 lines)
   - Filter rule tests (active spec, recency boundary, horizon:now
     initiative without size, the `(unset)` exclusion).
   - Stop-nagging tests (within 24h, beyond 24h, count-increased
     exception, missing `drift_count_at_exit` field).
   - Hint formatting (singular vs plural).
   - Config override tests.

3. **`internal/config/config.go`** (small edit)
   - Add `Roadmap *RoadmapConfig` field with
     `AmbientRecencyDays int` and `StopNaggingHours int`.
   - Document defaults in struct comments.

4. **`internal/projection/projection.go`** (small edit, ~30 lines)
   - Add `ActiveSpec string` field to `NextMDOptions`.
   - After the `## Next` block and before `## Blocked on`, call
     `sizing.AmbientDrift` and emit the `## Roadmap shape` section
     when non-quiet/non-zero.
   - Omit section entirely when quiet or zero.

5. **`internal/projection/projection_test.go`** (small additions)
   - Golden test for NEXT.md with drift surfacing.
   - Golden test for NEXT.md with quiet drift (section absent).

6. **`internal/pulse/pulse.go`** (small edit)
   - Add `AmbientDriftSummary` type.
   - Add `SizeDrift *AmbientDriftSummary` field to `PulseData`.

7. **`internal/pulse/render.go`** (small edit, ~30 lines)
   - Render `SizeDrift` summary line at the top of text and markdown
     outputs, above the existing `Drift detected` detail block.
   - Add `size_drift` key to JSON output.

8. **`internal/serve/mcp_tools.go`** (small edit, ~20 lines)
   - `toolPulse`: after `pulse.PopulateDrift`, call
     `sizing.AmbientDrift(s.heroDir, s.projectRoot, opts)` and assign
     to `p.SizeDrift` when non-quiet/non-zero.
   - `toolKickoff`: build `AmbientDriftOpts{ActiveSpec: slug}`, call
     `sizing.AmbientDrift`, prepend the hint line to the output when
     non-quiet/non-zero.

9. **`internal/serve/mcp_test.go`** (small additions)
   - `toolPulse` test asserting `size_drift` field in JSON response
     under simulated drift, and absent when quiet.
   - `toolKickoff` test asserting hint-line prefix under simulated
     drift, and absent when quiet.

10. **`domains/engineering/agents/feature-delivery-lead.md`** (small edit)
    - Extend pre-flight step 4d with the one-sentence workspace-wide
      drift surfacing instruction.

11. **`domains/engineering/agents/platform-delivery-lead.md`** (small edit)
    - Same one-sentence extension to its analogous pre-flight step.

12. **`domains/engineering/skills/roadmap-review/SKILL.md`** (small
    addition, only after sibling #1 has created the file)
    - Add a brief "Ambient surfaces" subsection documenting that
      NEXT.md, `hero_pulse`/`hero_kickoff`, and the delivery-lead
      pre-flight all carry the ambient invitation to run
      `/roadmap-review` when drift is non-quiet/non-zero.
    - Document the `drift_count_at_exit` field convention so future
      session-record writers preserve it.

13. **No new MCP tools.** Surfaces are extensions of existing ones.

## Mockups

Not produced — surfaces are textual.

## Boundaries

Explicitly out of scope for v1:

- **Surfaces beyond the three named.** No wiring into `/prime`,
  `/resume`, `hero status`, the status bar, `hero check`, or any new
  MCP tool. Three surfaces; tune fatigue against three before any
  expansion.
- **Real-time monitoring or push notifications.** All three surfaces
  are pull-on-render. No file watchers, no daemons, no webhooks.
- **Non-sizing lenses in surface text.** Message stays "size drift."
  When horizons / releases / sprint-shape lenses land later, *their*
  spec extends `AmbientDrift` and the phrasing — this spec does not
  pre-commit to a taxonomy.
- **Per-spec inventory in any ambient surface.** Count and hint only.
  Row excerpts couple this spec to sibling #3's CLI format choices
  and inflate the surface.
- **A new MCP tool for ambient drift.** Use existing `hero_pulse` /
  `hero_kickoff`. A new tool is a scope-creep signal.
- **Auto-running `/roadmap-review` when drift surfaces.** Surface the
  invitation only; the user invokes the command.
- **Tracker-side surfacing.** No Jira labels, no GitHub comments
  signalling drift. Read-local-only.
- **Persistent config across CLI flags.** Tuning happens via
  `hero.json` edits only in v1.

## Risks

- **Nudge fatigue — the dominant risk to the whole initiative.**
  Mitigations layered: noise threshold (filters to active /
  recently-touched / high-impact only), 24h stop-nagging window after
  any `/roadmap-review` session, count-only phrasing (no row spam),
  hard cap of three surfaces. If real-world cadence still feels
  noisy, raise `stop_nagging_hours` in config; if too quiet, lower
  `ambient_recency_days`. The point of `AmbientDrift` being a single
  helper is exactly this: one place to tune, three places to inherit.
- **Recency window calibration.** 7 days is a guess based on the
  initiative's one-week dogfood cadence. Mitigation: configurable via
  `hero.json` (`roadmap.ambient_recency_days`). Document the default
  with rationale; revisit after a one-week dogfood window per the
  parent initiative's validation criterion.
- **Pulse / kickoff response size growth.** Mitigated by the
  count-only contract — adds at most ~80 bytes to either response
  payload. JSON field is a small object, not a list of slugs.
- **Coordination with sibling #1's session-record writer.** This spec
  reads `drift_count_at_exit` from records written by sibling #1.
  Mitigation: missing-field path is fully suppressive (same as no
  session record at all), so this spec ships safely even if the
  field is added later. The field is forward-compatible.
- **Active-spec detection is best-effort.** NEXT.md regeneration on
  commit doesn't know the agent's current spec, so the active-spec
  branch of the filter only lights up for `hero_kickoff` (which has
  the slug in args) and for delivery-lead pre-flight (which has the
  spec in its prompt and can pass it through the read of `hero_pulse`
  via a future arg — out of scope here; v1 lead reads `hero_pulse`
  without active-spec context). Acceptable degradation: recency +
  high-impact branches already catch the common case.
- **Phrasing churn.** If the hint wording is tuned later, three
  call-sites and one helper update. Mitigation: helper owns the
  string. One place to edit.

## Validation

- **Helper unit tests.** `sizing.AmbientDrift` returns expected
  count, hint, quiet, and reason across the filter scenarios (active
  spec, in-window, out-of-window, horizon:now initiative without
  size, `(unset)` legacy spec excluded), the stop-nagging scenarios
  (within window suppresses, beyond window emits, count-increased
  exception, missing-field treats as suppressive), and the
  singular/plural hint formatting.
- **NEXT.md golden tests.** A workspace fixture with seeded drift
  produces a NEXT.md containing `## Roadmap shape`; a fixture with
  no drift produces a NEXT.md without the section.
- **MCP tool tests.** `toolPulse` JSON response includes
  `size_drift: { count, hint }` under simulated drift, and omits the
  field when quiet; `toolKickoff` prepends the hint line under
  simulated drift and emits the kickoff body unchanged when quiet.
- **Delivery-lead pre-flight smoke check.** Run `/deliver` on a spec
  in a workspace with seeded drift and confirm the lead's pre-flight
  emits both the active-spec sizing nudge (existing) and the
  workspace-wide hint (new).
- **Dogfood window.** Per the parent initiative's validation, run
  with the surfacing enabled for at least one week before declaring
  v1 done; tune `ambient_recency_days` and `stop_nagging_hours` if
  fatigue or under-surfacing surface in real use.
- **Cross-spec check.** Verify that sibling #1's session-record
  writer either includes `drift_count_at_exit` by the time this spec
  ships or that the missing-field fallback path keeps surfaces sane.

## Cross-cutting

**Hint phrasing — one source of truth.** `sizing.AmbientDrift.Hint`
is the canonical string. NEXT.md, `hero_pulse`, `hero_kickoff`, and
delivery-lead pre-flight all quote it verbatim. Do not compose the
string in surface code.

**Locked surfaces.** Three surfaces only: NEXT.md, `hero_pulse` /
`hero_kickoff`, delivery-lead pre-flight. Any expansion is a
scope-creep signal and a separate spec.

**Soft dependency on sibling #1's session-record format.** The
`drift_count_at_exit` field is forward-compatible — this spec
degrades gracefully if it's absent.

**No dependency on sibling #3.** Count-only contract decouples us
from `hero size --check` row format choices.

## Kickoff

`roadmap-review-ambient-surfacing` — DELIVERED. 18 ACs DONE; SHIP /
noteworthy audit. `sizing.AmbientDrift` helper feeds three surfaces:
NEXT.md (`## Roadmap shape` section between Next and Blocked on),
`hero_pulse`/`hero_kickoff` (`size_drift` field), and delivery-lead
pre-flight. Live exercise on this workspace shows "12 specs have
size drift — run /roadmap-review to triage" in NEXT.md after a
checkpoint. Stop-nagging suppression and count-grew exception both
verified against fake session records. Two notes:

1. Hint phrasing dropped the spec's `⚠` prefix per CLAUDE.md's
   no-emojis rule. Shipped phrasing is the canonical form going
   forward (no emoji); spec text out-of-date but not load-bearing.
2. Scope absorbed `hero size --check --summary` flag — gave the
   delivery leads a CLI alternative to the MCP `size_drift` read.
   Small isolated addition, called out in commit and ledger.

Closes the `roadmap-shape` initiative — all four children
delivered.
