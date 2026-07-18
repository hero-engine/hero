---
name: pm-preset-detection
description: Read `hero.json`'s `pm.presets` config and apply the right authoring rules per artifact type. Analog to engineering's stack-detection.
metadata:
  audience: prd-author, story-writer, roadmap-curator, pm-delivery-lead
  purpose: operational
---

## What I do

Read `hero.json`'s `pm.presets` configuration and decide which preset-specific authoring behavior applies for the current artifact. PM workspaces run one of four delivery presets and one of two roadmap presets (plus an optional milestone overlay); the active combination drives which frontmatter fields are required, which sections are populated, and which dashboards render. Authoring agents must read the preset before populating fields — hardcoding sprint assumptions in a cycle-preset workspace produces broken artifacts.

This skill is the PM analog of engineering's `stack-detection`: load it first, ask it what mode you're in, then load the methodology-specific skills accordingly.

### Methodology preset vs vocabulary preset — they are independent

Hero has **two independent preset systems**:

1. **Methodology preset** (`pm.presets` in `hero.json`) — this skill. Drives
   lifecycle overlays, preset-conditional frontmatter fields, and view
   configurations. Affects *behavior* (which fields to populate, which
   templates to use, which dashboards to render).

2. **Vocabulary preset** (`vocabulary` and `vocabulary_overrides` in
   `hero.json`) — handled by Hero's vocabulary resolver. Drives
   *display names only* — the canonical type/kind pair `spec, kind:
   feature` is rendered as "Story" under `agile-scrum`, "Scope" under
   `shape-up`, "Card" under `kanban`, "Feature" under `default`. Does
   not affect frontmatter or behavior.

The two compose orthogonally. A team can run `delivery: cycle` (Shape Up
methodology) with `vocabulary: agile-scrum` (calls features "Stories"
without forcing sprint cadence), or `delivery: sprint` with `vocabulary:
shape-up` (calls features "Scopes" but with story points and sprint
cadence). Most teams use the natural pairing (cycle + shape-up, sprint +
agile-scrum), but the systems do not require it.

**This skill governs methodology preset only.** When you need to render
a type or kind name for the user, read it from Hero's vocabulary
resolver (configured via `hero.json`) — do not hardcode "Story" or
"Feature" in agent output.

## When to use me

Load this skill when:

- any PM authoring agent starts work on a `feature`, `epic`, `initiative`, `prd`, or `pitch`
- `roadmap-curator` is grouping the Roadmap board (grouping depends on roadmap preset)
- `pm-delivery-lead` needs the math model for the active preset (planners `capacity-planner` / `cycle-planner`)
- `pm-delivery-lead` is orchestrating a refinement pass and needs to decide which authoring agent to call

## The preset schema

`hero.json` carries:

```json
{
  "pm": {
    "presets": {
      "roadmap": "horizon",
      "delivery": "continuous",
      "overlay": null
    }
  }
}
```

Field values:

| Field | Values | Meaning |
|---|---|---|
| `roadmap` | `"horizon"` \| `"q3-2026"` (or any quarter string) | How the Roadmap board is grouped. `horizon` = now/next/later; quarter strings = time-based buckets. |
| `delivery` | `"continuous"` \| `"sprint"` \| `"cycle"` \| `"phased"` | How delivery is shaped. `continuous` = flow; `sprint` = Scrum-style; `cycle` = Shape Up; `phased` = waterfall-ish discovery/design/build/launch. |
| `overlay` | `null` \| `"release"` \| `"milestone"` | Optional cross-cutting structure layered on top. `release` = group child work by named release; `milestone` = group by named milestone date. |

## Default when `pm.presets` is missing entirely

A workspace without `pm.presets` in `hero.json` is treated as:

- `roadmap: "horizon"`
- `delivery: "continuous"`
- `overlay: null`

The sane defaults. A PM picking up Hero for the first time gets a horizon roadmap and continuous-flow delivery without configuring anything. Switching to sprint / cycle / phased is an explicit choice.

If `pm.presets` is partially populated (e.g. only `delivery` is set), the missing fields fall back to the defaults — `roadmap: "horizon"`, `overlay: null`. Never throw on a missing preset key; degrade to the default.

## Per-preset field expectations

### `spec` (canonical type; PM-common `kind: feature`)

Always present (from the base spec type):
- `title`, `type`, `kind`, `status`, `priority`, `owner`

Preset-conditional:

| Preset | Required fields | Notes |
|---|---|---|
| `sprint` | `sprint`, `points` | `points` is content (Hero-authoritative); `sprint` is org-state (tracker-authoritative). |
| `cycle` | `cycle`, `hill_position` | `hill_position` ∈ {unknown, uphill, top, downhill, done}. Drives the hill chart. |
| `phased` | `release`, `phase` | `phase` ∈ {discovery, design, build, launch, post-launch}. |
| `continuous` | `wip_age` (computed, not authored) | Populated by the dashboard from `in-flight` timestamp. |

`story-writer` reads the preset, then prompts for the right fields. Under `continuous`, it does **not** prompt for `points` — there are no sprints. Under `sprint`, it does **not** prompt for `hill_position` — no hill chart.

### `initiative` spec

Always present:
- `horizon` (when `roadmap: "horizon"` is active — almost always)

Preset-conditional:

| Preset | Required fields | Notes |
|---|---|---|
| `cycle` (delivery) | `appetite` | ∈ {small, big}. Drives the betting table. |
| `phased` (delivery) | `target_release` | The release the initiative is committed to. Org-state. |
| `roadmap: <quarter>` | `horizon` set to the quarter string | The board groups by quarter, not by now/next/later. |

`product-strategist` and `roadmap-curator` read the preset to decide which fields to populate and how to group the Roadmap board.

### `epic` spec

| Preset (delivery) | Required fields | Notes |
|---|---|---|
| `sprint` | `velocity_target` | Estimated points across the epic for capacity planning. |
| `cycle` | `is_bet`, `appetite` | If `is_bet: true`, the epic is one of the cycle's bets and carries `appetite`. |
| `phased` | `release` | The release the epic is committed to. |
| `continuous` | (none) | Epics are coarse containers; no preset-specific fields. |

`pm-delivery-lead` reads the preset to decide which fields to populate.

## "Switching is a config edit, not a migration" — preserve fields when their preset toggles off

Critical rule: when a workspace switches presets (e.g. cycle → sprint), the previously-populated preset-conditional fields are **preserved** on existing artifacts. They become inactive (not required, not rendered) but they are not deleted.

Why:

- The team may switch back later; deleting destroys history.
- Past artifacts retain their authoring context — a story shipped under cycle preset still shows its `hill_position` history even after the workspace moves to sprint.
- Switching is a low-cost operation by design; data loss makes it high-cost.

The rule for authoring agents: **only populate fields for the active preset. Do not delete fields from the inactive preset.** A `story-writer` under sprint preset writing a new story does not author `hill_position`; a `story-writer` under sprint preset editing an old cycle-authored story does not strip its `hill_position`.

## How authoring agents query the preset

Before populating any preset-conditional field, the authoring agent calls `pm-preset-detection`:

```
preset = read_pm_presets()
# returns { roadmap: ..., delivery: ..., overlay: ... }

if preset.delivery == "sprint":
    prompt_for("points")
    prompt_for("sprint")  # may auto-fill from tracker
elif preset.delivery == "cycle":
    prompt_for("hill_position")
    prompt_for("cycle")
elif preset.delivery == "phased":
    prompt_for("release")
    prompt_for("phase")
# continuous → no preset-specific prompts
```

The pattern is the same for `initiative` (check `preset.delivery` for `appetite` / `target_release`; check `preset.roadmap` for `horizon` shape) and `epic` (check `preset.delivery` for the velocity / bet / release fields).

`pm-delivery-lead` reads the preset once per session and passes it to delegated authoring agents — they don't all need to re-read independently.

## Overlay handling

When `overlay: "release"`, every story / epic / initiative gets an additional `release` field regardless of delivery preset. The Roadmap board, the Refinement queue, and the Cycle / Sprint boards all add a "Release" grouping affordance.

When `overlay: "milestone"`, similarly, every artifact gets a `milestone` field and an additional grouping. Milestones are date-anchored (e.g. "EU launch Q3"); releases are version-anchored (e.g. "v2.4").

Authoring agents prompt for the overlay field after the preset-conditional prompts. The overlay is *additive* — it does not replace the delivery preset's fields.

## Tracker fronting and preset fields

Per the tracker-fronting decision (`.hero/knowledge/decisions/tracker-fronting-and-local-first.md`), each field carries a classification:

- **content** fields (Hero-authoritative): `points`, `hill_position`, `appetite`, `is_bet`, `velocity_target`, `phase` (content half)
- **org-state** fields (tracker-authoritative): `sprint`, `cycle`, `release`, `target_release`, `phase` (state half)

`pm-preset-detection` returns each field's classification along with its required/optional status. Authoring agents writing content fields write locally; writing org-state fields proposes a value that the tracker integration syncs. The user does not see a sync gesture.

When a workspace switches presets, the conflict-policy split still holds — `points` written under sprint stays Hero-authoritative even after the workspace switches to cycle.

## When the preset is ambiguous (e.g. partial migration)

A workspace mid-migration may have both `sprint`-era and `cycle`-era stories in active circulation. The active `pm.presets.delivery` governs **new authoring**; existing artifacts retain their authored fields.

If `pm-delivery-lead` encounters a story authored under a different preset than the current active one, it does not rewrite. It surfaces the mismatch as informational: "This story has `points: 5` but the active preset is `cycle`. Continue editing? (The `points` field will be preserved but not actively used.)"

## Anti-patterns

- **Hardcoding sprint assumptions in agents.** A `story-writer` that always prompts for `points` is broken under cycle / continuous presets.
- **Forcing estimation when the active preset doesn't require it.** Continuous-flow workspaces do not estimate; an agent that demands points wastes the user's time and produces meaningless data.
- **Deleting fields from the inactive preset on switch.** Destroys history; makes switching back high-cost.
- **Treating the preset as immutable for the session.** The user can edit `hero.json` mid-session; the agent should re-read on next authoring action, not cache forever.
- **Throwing when `pm.presets` is missing.** Degrade to the default (`horizon` + `continuous` + no overlay).
- **Mixing org-state and content fields without classification awareness.** Per the tracker-fronting decision, classification drives the write path. Confusing them produces sync conflicts.
- **Authoring overlay fields as if they replace delivery-preset fields.** Overlay is additive. A story under sprint + release overlay has both `points` and `release`.
- **Cross-preset auto-conversion.** Switching cycle → sprint does not auto-translate `appetite: big` to `points: 13`. The estimates are not commensurable; let the team re-estimate if needed.

## PM lifecycle vocabulary → engine statuses

PM process language (drafting, refined, ready, shipped, …) is **vocabulary**,
not a separate status machine. Every PM lifecycle word maps onto exactly one
engine status. **This table is the single source for status vocabulary; agents
cite it, they don't restate it.**

| PM lifecycle term | Engine status | Applies to |
|---|---|---|
| `drafting`, `drafted`, `shaping`, `refining` | `planning` | work specs |
| `refined`, `ready` (reviewed, handoff-eligible) | `in-review` | work specs |
| engineering has claimed the spec | `delivering` | work specs |
| `shipped` | `completed` | work specs |
| `dropped` | `superseded` (work specs) / `rejected` (intake) | work specs / intake |
| initiative `candidate` | `planning` | initiatives |
| initiative `committed` | `delivering` | initiatives |
| initiative `shipped` | `completed` | initiatives |

The canonical engine statuses are `planning`, `in-review`, `delivering`,
`completed`, `regressed` (work specs); `triaged`, `promoted`, `rejected`,
`merged` (intake); `superseded` (shared). PM agents write these values on disk —
the terms in the left column describe PM *process*, they are never a status
value in frontmatter.

**`handed_off` / `handed_back` are not part of this mapping.** They are
cross-repo peering statuses (the `hero peer` / `hero handoff <spec> <alias>`
boundary between sibling repos), not the pm→engineering owner flip. The
pm→engineering handoff is an `owner:` change — `hero spec set-owner <slug>
engineering` — **not** a status change: the spec stays `in-review` until
engineering claims it and flips it to `delivering`.
