---
title: Roadmap Review — Command, Agent, and Skill for On-Demand Shape Detection
slug: roadmap-review
type: feature
domain: engineering
status: completed
size: medium
priority: P1
tags: [roadmap, sizing, command, agent, skill]
created: 2026-06-01
relations:
  - target: roadmap-shape
    kind: parent
---

## Goal

Introduce `/roadmap-review`, the `roadmap-reviewer` agent, and the
`roadmap-review` skill so a user can run one command, get a
**prioritized triage of sizing drift across the planning corpus**, and
resolve each item conversationally — the agent executes the resolution
CLI itself, doesn't punt the keystrokes back to the user. Sizing is the
only active lens in v1; horizons / releases / sprint-shape are named
scaffolding placeholders the agent **refuses to act on** rather than
improvising. A session ends when high-impact drift is exhausted, the
user says stop, or 5 items have been resolved (whichever comes first),
and a brief record lands in `.hero/knowledge/` so cadence is visible
over time.

## Context

This is the first child of the [`roadmap-shape`](../../initiatives/roadmap-shape/spec.md)
initiative. Three sibling specs (#2 ambient surfacing, #3 actionable
output, #4 multi-spec design routing) reference its command name,
agent name, and skill structure — so this has to land first and lock
those names.

The stub captures the decisions already made in composition: command
name `/roadmap-review`, agent name `roadmap-reviewer`, skill name
`roadmap-review`, sizing-only v1, Lenses extension model. This design
pass honors all of those and is not re-litigating them.

The capability rides on top of the size mechanic shipped in
`spec-size-and-promotion-nudge` — `hero size`, `hero size --check`,
`hero estimate`, the `spec-sizing` skill, and `hero_warnings`. None
of that machinery changes; `/roadmap-review` is the *interactive
triage loop* over its output.

## Approach

### Three artifacts, one shared loop

The capability is three files that compose:

1. **The skill** (`roadmap-review`) — carries the doctrine: what the
   Lenses model is, how the active sizing lens prioritizes drifts, the
   conversational phrasing per resolution option, and the stop
   condition. The skill is what makes the agent opinionated.
2. **The agent** (`roadmap-reviewer`) — loads the skill, surveys the
   workspace, prioritizes findings, walks them one at a time, and
   **executes resolutions itself**. It does not produce a wall of
   prose; it produces a triage loop.
3. **The command** (`/roadmap-review`) — thin router. Delegates to the
   agent and passes the user's optional focus argument through.

### The interactive triage loop

Every session follows the same shape:

```
load skill → survey workspace → prioritize findings → walk one at a time
  → for each: surface finding, propose resolution, get confirm,
    execute CLI, advance → stop on exhaustion / user halt / N=5 cap
  → capture session record to .hero/knowledge/
```

The agent's defining behavior is the **walk-one-at-a-time** rhythm.
No bulk operations, no "here are all your drifts, please fix them" —
the agent's value is conversation, not enumeration.

### Survey query set (locked)

On invocation the agent calls these — in this order — and merges the
results into a single working list. All run read-only.

| Source | Tool | Purpose |
|---|---|---|
| Declared-vs-computed drift across the corpus | `hero size --check` | Primary sizing drift signal |
| Aggregated workspace warnings (includes drift, dedup against above) | `mcp__hero__hero_warnings` | Catches drift surfaced through the warnings pipeline; deduped against `hero size --check` |
| All planning-status work specs with `size:` frontmatter | `mcp__hero__hero_list` filtered to `status: planning,delivering` | Provides corpus shape for prioritization (counts, tier mix) |
| Topical clusters of related specs | `mcp__hero__hero_search` over recent spec titles + tag overlap | Detects the "multiple related specs" condition (see Cross-cutting #4) |

If `hero size --check` is empty AND no related-spec clusters surface,
the agent reports "no shape concerns — workspace is healthy" and
exits without entering the loop.

### Prioritization rules (active sizing lens)

The skill names the order. The agent does not re-sort.

1. **`giant` without `size_ack`** — highest. Month+ specs that haven't
   been consciously acknowledged are the loudest shape failure.
2. **`x-large` features / bugs / enhancements** — multi-week leaf work.
   Almost always wants a `/split`.
3. **Container drift** (epic/initiative declared smaller than rolled-up
   children) — the parent is lying about its scope.
4. **Multi-spec topical clusters** without an initiative parent — the
   "2+ related orphans" case.
5. **`large` features / bugs / enhancements** — soft. Surface only if
   nothing higher remains.

`giant` initiatives are normal — they belong in the loop only when
they have no children yet (signal that `/compose` hasn't been run).

### Resolution options + conversational phrasing

The skill defines four resolution options. For each, the agent has a
canonical script: *what it says, what the user can answer, what CLI
it fires on confirm.* The agent never tells the user "run X" — it
runs X itself.

| Resolution | When the agent offers it | On confirm |
|---|---|---|
| **Acknowledge** | `giant` leaf where the user explicitly wants single-pass delivery | `hero size --ack giant <slug>` (or write `size_ack: giant` to frontmatter if no CLI flag exists yet) |
| **`/compose`** | `x-large`/`giant` that should become an initiative with phased children | Invoke the `/compose` flow against the slug — agent hands off, doesn't simulate |
| **`/split`** | `large`/`x-large` leaf that belongs as 2–3 sibling specs | Invoke the `/split` flow against the slug — agent hands off |
| **Re-horizon** *(sizing-lens version: bump declared tier)* | Drift where declared < computed; user agrees the spec really is the bigger tier | `hero size <slug> <new-tier>` |

The skill carries the paste-ready phrasing for each option — the agent
quotes it rather than improvising, the same pattern `spec-sizing` uses
for its nudge text. Example phrasing the skill commits to:

- Acknowledge: *"This `giant` spec is intentional? I can stamp
  `size_ack: giant` so it stops asking at design time — mid-delivery
  checks continue. Confirm?"*
- `/compose`: *"This is `x-large` — recommended path is `/compose` into
  3–5 phased children. Want me to kick that off now?"*
- `/split`: *"This is `large` across two subsystems — `/split` into a
  pair of sibling specs. Run it?"*
- Re-horizon: *"Declared `medium`, computed `large`. Bump to `large`?
  (Y/n)"*

### "Execute on confirm" pattern

When the user says "yes" / "do it" / "go" / "acknowledge it," the
agent **runs the CLI itself**. The user does not see "now run
`hero size foo large`" — they see the agent run it, surface the
output, and advance to the next item. This is the agent's job; punting
the keystrokes back defeats the loop.

For resolutions that invoke another slash command (`/compose`,
`/split`), the agent triggers the workflow and yields — once that
workflow returns, the user can re-fire `/roadmap-review` to continue
the triage.

### Stop condition

A session ends on the first of:

1. **No high-impact drift remains** — priorities 1–4 above are
   exhausted. The agent reports the clean state and exits.
2. **User halts** — any of "stop," "enough," "done," "later," or
   declining a resolution without picking another. The agent does not
   ask "are you sure"; it accepts and exits.
3. **Five items resolved in this session** — soft cap to keep momentum.
   The agent reports the cap, summarizes what was resolved, and
   suggests re-running tomorrow.

The cap is in the skill, not hard-coded in the agent — so it's tunable
by editing the skill without touching the agent prompt.

### Session record (knowledge capture)

On exit (any reason), the agent writes a short note to
`.hero/knowledge/roadmap-review-sessions/{YYYY-MM-DD}-{HHMM}.md` via
the same conventions `note-capture` uses. Format:

```markdown
---
type: note
created: 2026-06-01T14:32:00Z
tags: [roadmap-review, sizing]
---

# Roadmap review — 2026-06-01 14:32

**Findings surveyed:** 7 (5 sizing-drift, 2 missing-parent clusters)
**Walked:** 4
**Resolved:** 3 (1 `/split`, 1 ack, 1 size bump)
**Deferred:** 1 (user said "later")
**Exit reason:** user halt

## Items
- `agent-outposts` (giant, no ack) → user said "later"
- `master-ingest-restore` (x-large) → split into 2 children via `/split`
- `e2e-discovery` (declared medium, computed large) → bumped to large
- `hero-sales` + `hero-team-server` cluster → user acknowledged, no action
```

Over time the directory becomes the dogfood signal for *"is the user
running this on cadence, and what trends are surfacing"* — feeds the
nudge-fatigue tuning in child #2.

### Skill structure (Lenses)

The `## Lenses` section is the extensibility seam. Today:

```markdown
## Lenses

### Active

#### Sizing
- Prioritization rules: [the 5-item ordering above]
- Resolution options + phrasing: [the four scripts above]
- Stop condition: [exhaustion / halt / N=5 cap]

### Scaffolding (not implemented — future work)

#### Horizons
Placeholder. v1 does not act on horizon drift. If the user asks the
agent to triage horizons, the agent responds:
> "Horizon-lens triage isn't implemented yet — that's deferred to
> future lens work. I can only act on sizing drift in this version."

#### Releases
Placeholder. Same refusal pattern.

#### Sprint-shape
Placeholder. Same refusal pattern.
```

Adding a new lens later = move it from Scaffolding to Active and fill
in the four sub-sections (prioritization, resolution options, phrasing,
stop condition). No agent prompt changes required.

### `/roadmap-review` vs `hero check` (one sentence each, twice)

The initiative's Cross-cutting concerns require this distinction land
in the spec and the docs. Canonical wording:

> `hero check` is **workspace hygiene** — a one-shot report of stale
> specs, missing fields, convention drift, and warning counts.
> `/roadmap-review` is **roadmap shape** — an interactive triage that
> walks each high-impact drift, proposes a resolution, and executes the
> resolution CLI on confirm. Both surface drift; only `/roadmap-review`
> resolves it.

This sentence lives in the `roadmap-review` skill (Doctrine section)
and the agent prompt. README updates are out of scope for this spec.

### "Multiple related specs" canonical phrasing — Option C

The stub asked us to pick: phrasing lives in `spec-sizing` (Option A),
in `roadmap-review` (Option B), or in a new shared skill
`spec-composition` (Option C). **We pick C.**

Rationale: the condition ("2+ specs are topically related and have no
initiative parent") is fundamentally about **composition discipline** —
when to lift a cluster into an initiative — not about sizing per se.
A `medium` cluster of three orphans is a composition problem; sizing
doesn't model it. And child #4 (`multi-spec-design-routing`) catches
the same condition at `/design`-time, which is also a composition
moment. Locating the phrasing in `spec-sizing` would force `spec-sizing`
to grow scope it doesn't otherwise want; locating it in `roadmap-review`
would force #4 to cross-reference a sibling.

Therefore a new skill `spec-composition` carries the canonical
detection heuristic + phrasing. Both `roadmap-review` (this spec) and
`multi-spec-design-routing` (child #4) cross-reference it. The skill
ships with this spec because we're the first consumer; #4 will
reference it without re-authoring.

The canonical phrasing (lives in `spec-composition`):

> *"`<slug-a>`, `<slug-b>`, and `<slug-c>` look topically related and
> none of them have an initiative parent. If they're one body of work,
> `/compose` lifts them into a shared initiative. If they're genuinely
> independent, no action needed. Want me to `/compose` them?"*

## Acceptance Criteria

- WHEN the user runs `/roadmap-review` THE SYSTEM SHALL invoke the
  `roadmap-reviewer` agent and load the `roadmap-review` skill before
  any survey calls.
- WHEN the agent starts a session THE SYSTEM SHALL call `hero size --check`,
  `mcp__hero__hero_warnings`, `mcp__hero__hero_list`, and
  `mcp__hero__hero_search` to assemble the survey set.
- WHEN survey results contain drift THE SYSTEM SHALL prioritize findings
  using the 5-item ordering (unacknowledged `giant` first, `x-large`
  leaves, container drift, multi-spec orphan clusters, `large` leaves).
- WHEN a finding is presented THE SYSTEM SHALL walk one item at a time
  — surface the finding, propose exactly one resolution option, accept
  the user's response, advance.
- WHEN the user confirms an Acknowledge resolution THE SYSTEM SHALL run
  the size-ack CLI itself (write `size_ack: giant` to frontmatter via
  `hero size --ack` or direct frontmatter edit) without asking the user
  to run it.
- WHEN the user confirms a `/compose` or `/split` resolution THE SYSTEM
  SHALL invoke the matching slash command flow and yield, not simulate
  or describe it.
- WHEN the user confirms a Re-horizon resolution THE SYSTEM SHALL run
  `hero size <slug> <new-tier>` itself.
- WHEN survey results are empty THE SYSTEM SHALL report "no shape
  concerns — workspace is healthy" and exit without entering the loop.
- WHEN the high-impact priority queue (rules 1–4) is exhausted THE
  SYSTEM SHALL exit the loop and write the session record.
- WHEN the user says "stop," "enough," "done," or "later" THE SYSTEM
  SHALL exit the loop without further confirmation.
- WHEN five items have been resolved in one session THE SYSTEM SHALL
  report the cap, summarize, and exit.
- WHEN any session ends THE SYSTEM SHALL write a session record to
  `.hero/knowledge/roadmap-review-sessions/{YYYY-MM-DD}-{HHMM}.md`
  using the format in Approach.
- IF the user asks the agent to triage a non-sizing lens (horizons,
  releases, sprint-shape) THEN THE SYSTEM SHALL refuse with the
  scaffolded phrase: *"<lens>-lens triage isn't implemented yet — that's
  deferred to future lens work. I can only act on sizing drift in this
  version."*
- IF the agent improvises a resolution outside the four canonical
  options (Acknowledge, `/compose`, `/split`, Re-horizon) THEN THE
  SYSTEM SHALL be considered out of spec and fail review.
- THE SYSTEM SHALL document the `hero check` vs `/roadmap-review`
  distinction in the `roadmap-review` skill body using the canonical
  wording in Approach.
- THE SYSTEM SHALL carry the "multiple related specs" canonical
  phrasing in the new `spec-composition` skill, with cross-references
  from `roadmap-review` and (later) `multi-spec-design-routing`.
- WHERE the workspace has a configured tracker THE SYSTEM SHALL NOT
  write to the tracker during a roadmap-review session — read-local-only
  is a hard boundary.

## Files to Touch

Canonical paths (under `domains/engineering/`); harness directories
(`.claude/`) are views and update via existing sync.

1. **`domains/engineering/skills/roadmap-review/SKILL.md`** (new)
   - Doctrine: capability summary, `hero check` vs `/roadmap-review`
     distinction, walk-one-at-a-time rhythm.
   - `## Lenses` section with Active (Sizing fleshed out) and
     Scaffolding (Horizons / Releases / Sprint-shape placeholders +
     the refusal phrase).
   - Prioritization rules (the 5-item ordering).
   - Resolution options + paste-ready phrasing for each.
   - Stop condition (exhaustion / halt / N=5 cap).
   - Session record format + path.
   - Cross-reference to `spec-composition` for multi-spec phrasing.

2. **`domains/engineering/skills/spec-composition/SKILL.md`** (new)
   - "Multiple related specs" detection heuristic.
   - Canonical phrasing for the `/compose` recommendation when a
     cluster of orphan related specs is detected.
   - Used by `roadmap-review` (now) and `multi-spec-design-routing`
     (child #4, future).

3. **`domains/engineering/agents/roadmap-reviewer.md`** (new)
   - Frontmatter: `name: roadmap-reviewer`, `mode: subagent`,
     `role: review`, `temperature: 0.1`, modeled on `pr-reviewer.md`.
   - Prompt: load `roadmap-review` skill → run survey set →
     prioritize → walk one at a time → execute resolutions on confirm
     → stop on conditions → write session record.
   - Rules: refuse non-sizing lenses with the skill's scaffolded
     phrase; never punt CLI keystrokes back to the user.
   - Closing output: list of items resolved, deferred, refused.

4. **`domains/engineering/commands/roadmap-review.md`** (new)
   - Frontmatter `description:` short — "On-demand triage of
     roadmap-shape drift across the planning corpus. Walks one finding
     at a time and resolves on confirm."
   - Body: thin router. Delegate to `roadmap-reviewer`, pass
     `$ARGUMENTS` through as optional focus (e.g., a single slug to
     scope to).
   - Match the brevity of `commands/review.md` (~15 lines).

5. **`domains/engineering/skills/spec-sizing/SKILL.md`** (small edit)
   - Add a short "Composing with related skills" entry pointing to
     `roadmap-review` as the interactive triage surface for the drift
     it surfaces, and to `spec-composition` for the multi-spec case.
   - Do not move sizing phrasing into `roadmap-review`; the size
     ladder remains the source of truth.

6. **No Go surfaces required.** The agent calls existing CLIs
   (`hero size`, `hero size --check`) and existing MCP tools
   (`hero_warnings`, `hero_list`, `hero_search`). If `hero size --ack`
   does not yet exist as a CLI flag, the agent writes `size_ack: giant`
   to spec frontmatter via direct file edit — this is an acceptable
   degradation and worth a follow-up spec, not a blocker for v1.

## Mockups

Not produced for this spec — the surface is conversational, not visual.

## Boundaries

Explicitly out of scope for v1:

- **Implementation of horizons / releases / sprint-shape lenses.**
  Scaffolding placeholders only. The agent refuses to act on them.
- **Auto-running `/roadmap-review` on a schedule.** Manual invocation
  only in v1.
- **Web UI / dashboard surfacing.** The triage loop is CLI/agent only.
- **Bulk operations** — no "acknowledge all `giant`," no "bump all
  drift." One item at a time is the design.
- **Tracker writes.** Read-local-only is a hard boundary. No labels,
  no comments, no ticket creation against Jira / GitHub / Linear.
- **Ambient surfacing** — that's child #2 (`roadmap-review-ambient-surfacing`).
  This spec ships the capability; child #2 wires it into NEXT.md,
  hero_pulse, and the delivery-lead pre-flight.
- **New `hero check` categories.** The `hero check` vs `/roadmap-review`
  boundary is a one-line distinction; we do not modify `hero check`.
- **Redesigning `/compose` or `/split`.** The agent invokes them as
  black boxes.

## Cross-cutting

**Locked names** (initiative-wide; three siblings depend on them):

- Command: `/roadmap-review`
- Agent: `roadmap-reviewer`
- Skill: `roadmap-review`

**Shared "multiple related specs" phrasing — Option C decision.**
We pick Option C: phrasing lives in a new `spec-composition` skill.
Rationale documented in Approach. `roadmap-review` cross-references
`spec-composition`; child #4 (`multi-spec-design-routing`) will
cross-reference it on its own delivery. `spec-sizing` does *not*
absorb this — it stays focused on the size ladder.

**`hero check` vs `/roadmap-review` distinction.** Canonical sentence
in Approach lives in the `roadmap-review` skill body. Docs/README
updates are out of scope for this spec.

**Cross-references between skills:**

- `roadmap-review` ↔ `spec-sizing` — skill bodies cross-reference; the
  ladder lives in `spec-sizing`, the triage loop lives in `roadmap-review`.
- `roadmap-review` ↔ `spec-composition` — multi-spec detection +
  phrasing lives in `spec-composition`.
- `spec-sizing` ← `roadmap-review` — small "Composing with related
  skills" addition in `spec-sizing` pointing at `roadmap-review`.

## Risks

- **Naming churn.** Locked, but if `/roadmap-review` doesn't stick in
  practice (users call it `/triage` or `/shape`), three sibling specs
  rewrite. *Mitigation:* the locked names are validated in this spec;
  if a rename surfaces during dogfood, coordinate across the initiative
  before any rename lands.
- **Lens scaffolding rot.** Placeholders for horizons / releases /
  sprint-shape with no owner become dead weight in `SKILL.md`.
  *Mitigation:* the placeholder bodies are 1–2 lines (refusal phrase
  only) — low surface to rot. If by the time #2 ships the
  scaffolding still has no follow-up initiative, drop it from the
  skill and revisit when a real lens lands.
- **Session drag.** A user with 12 drifts could enter a long loop and
  bail mid-way. *Mitigation:* the N=5 cap + halt-on-any-stop-word
  + walk-one-at-a-time rhythm. The cap is in the skill, not the agent
  — tunable.
- **Overlap confusion with `hero check`.** Users may not know which
  to run. *Mitigation:* the canonical one-sentence distinction in
  the skill body and the agent prompt. Both surface drift; only
  `/roadmap-review` resolves it.
- **Agent improvises a resolution outside the four canonical options.**
  *Mitigation:* the skill commits to four options with paste-ready
  phrasing; the agent prompt is explicit that any other resolution is
  out of spec. Caught in acceptance criteria.
- **Dependency on missing `hero size --ack` flag.** If the flag doesn't
  exist yet, the agent falls back to direct frontmatter edit. *Mitigation:*
  acceptable degradation; the CLI flag is a follow-up spec, not a blocker.

## Validation

- A fresh session running `/roadmap-review` on the current workspace
  (which has known size drift across multiple `delivering` specs)
  surfaces a prioritized triage list, walks the top item, executes the
  user's chosen resolution via CLI, and advances.
- A run on a workspace with zero drift reports "no shape concerns"
  and exits without entering the loop.
- A run where the user types "stop" after one item exits cleanly and
  writes a session record reflecting the one-item walk.
- A run where the user asks the agent to "triage horizons" returns the
  scaffolded refusal phrase and does not improvise.
- The session record lands under
  `.hero/knowledge/roadmap-review-sessions/` with frontmatter and the
  documented structure.
- Cross-spec check: child #4 (`multi-spec-design-routing`) can be
  designed against the `spec-composition` skill without re-authoring
  the multi-spec phrasing.
- `hero check` and `/roadmap-review` produce non-overlapping outputs
  on the same workspace (check reports hygiene counts; roadmap-review
  reports an interactive triage queue).

## Kickoff

`roadmap-review` — DELIVERED. All 17 ACs DONE; clean SHIP audit. Four
files landed at canonical paths: `domains/engineering/skills/{roadmap-review,spec-composition}/SKILL.md`,
`domains/engineering/agents/roadmap-reviewer.md`,
`domains/engineering/commands/roadmap-review.md`. Sizing-only lens in
v1; horizons/releases/sprint-shape are scaffolded placeholders with
the agent's refusal phrase. Session record format established with
`drift_count_at_exit:` frontmatter (sibling #2 reads it for the
24h stop-nagging rule). One residual: `hero size --ack <tier> <slug>`
CLI flag doesn't exist — agent uses direct frontmatter edit as
fallback per the spec's documented Risk. To pick up the residual:
add `--ack` flag to `internal/cli/size.go`, mirroring the `<slug> <tier>`
positional form, writing `size_ack:` not `size:`.

→ Next: `/deliver size-drift-actionable-output` (trivial, independent)
or `/deliver multi-spec-design-routing` (extends `spec-composition`
with Triggers/Phrasing).
