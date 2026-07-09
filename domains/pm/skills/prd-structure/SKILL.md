---
name: prd-structure
description: Canonical PRD templates and section-by-section quality bar — pitch-shaped (default under cycle preset) and ten-section (default under sprint, continuous, and phased presets).
metadata:
  audience: prd-author, pm-reviewer
  purpose: prd-authoring
---

## What I do

Provide the two canonical PRD templates Hero PM ships with, the rules for choosing between them, and the section-by-section quality bar each must clear. PRDs are the largest PM authoring artifact and the single most reviewed artifact in the workflow — getting the structure right is the difference between a PRD that ships and one that loops in review.

## When to use me

Load this skill when:

- drafting a new PRD from an initiative (`prd-author`; pitch authoring is P1)
- choosing between pitch shape and ten-section shape based on the active preset
- reviewing a PRD for completeness (`pm-reviewer`)
- switching template mid-draft (rare, but allowed)
- inheriting an external PRD draft and refactoring it into Hero shape

Cross-reference `prd-anti-patterns` for the smell catalog — this skill says how to structure a PRD; the anti-patterns skill says what makes one unusable.

## The two templates

### Pitch template (Shape Up default — cycle preset)

```
# {Title}

## Problem
What's broken or missing. Concrete, customer-grounded.

## Appetite
Small (1-2 weeks) or Big (6 weeks). The budget, not the estimate.

## Solution
Fat-marker sketches. Breadboards. What we'd build, at a level the
team can react to without overcommitting.

## Rabbit Holes
Specific traps to avoid. "Don't build configurable X." "Skip the
edge case where Y."

## No-Gos
Work explicitly excluded from this appetite. Defends against scope
creep.

## Linked stories
Child stories that decompose this pitch. Filled in during /refine.

## Risks
What we'd discover too late if we didn't probe now.
```

Required sections: Problem, Appetite, Solution, Rabbit Holes, No-Gos.

### Ten-section template (sprint, continuous, phased default)

```
# {Title}

## Problem
## Goals & Success Metrics
## Users & Personas
## Solution
## User Flows
## Acceptance Criteria
## Out of Scope
## Risks
## Open Questions
## Timeline
```

Required sections: all ten (sections can be marked "N/A — {reason}" but must not be silently omitted).

## Choosing between templates

The choice is driven by the active methodology preset in `hero.json`:

| Preset | Default template | Rationale |
|---|---|---|
| **cycle** (Shape Up) | pitch | Appetite is load-bearing; cooldown demands tightness; Rabbit Holes and No-Gos are how Shape Up defends scope. |
| **sprint** (Scrum) | ten-section | Timeline and AC are required by sprint planning rituals; Goals & Metrics drive sprint goals. |
| **continuous** (Kanban / flow) | ten-section | Open Questions and Risks support rolling commitment; Timeline is approximate but still expected. |
| **phased** (waterfall-ish) | ten-section | Timeline and Milestones are first-class; phase-gates need explicit Out of Scope. |

The preset detection is automatic via `pm-preset-detection`. Authors can override per-PRD if the methodology choice doesn't match the work — but flag the override in the spec so reviewers know it's deliberate.

### Switching mid-draft

Acceptable. The two templates share most of their content; the move is one of compression or expansion, not a rewrite.

- **Ten-section → pitch:** Compress Goals & Success Metrics + Users & Personas + User Flows into Solution. Lift Timeline rationale into Appetite. Promote Out of Scope to No-Gos. Demote AC to a Linked stories reference.
- **Pitch → ten-section:** Expand Solution into Solution + User Flows. Add Goals & Success Metrics (from Problem framing). Add Users & Personas. Convert No-Gos into Out of Scope plus an explicit Timeline.

## Section-by-section quality bar

The bar both templates share. Cycle-preset-only sections (Appetite, Rabbit Holes, No-Gos) and ten-section-only sections (Goals & Success Metrics, User Flows, Timeline) are called out.

### Problem (both)

The customer-grounded thing that's broken or missing. **What "passes":** a single paragraph naming the user, the situation, the friction, and (where available) the evidence — intake quotes, support ticket counts, win-loss notes. **What "fails":** abstract problem statements ("users want better X"); restated solutions ("we don't have a CSV export"); aspirational framing ("we should be world-class at Y").

### Appetite (pitch only)

The *budget*, not the estimate; missing Appetite blocks `draft → review`. See `pitch-writing-shape-up` for the full quality bar and passing/failing examples.

### Goals & Success Metrics (ten-section only)

What the PRD wants to move and by how much. **What "passes":** named metric, baseline, target, observation method. **What "fails":** "increase engagement" with no definition; targets without baselines; metrics that can only be measured a quarter after launch. Cross-reference `metrics-design` for the deeper bar.

### Users & Personas (ten-section only)

Who the change is for. **What "passes":** named segment with the relevant context for this PRD ("returning shoppers who abandon at the cart step"). **What "fails":** generic "users"; persona dumps copied from a brand book; personas that don't match the segments Goals & Metrics targets.

### Solution (both)

What we'd build, at the level the team can react to without overcommitting. **What "passes" (pitch):** fat-marker sketches, breadboards, named flows, named omissions. **What "passes" (ten-section):** a clearer description, possibly with embedded mockup references; still no implementation prescription. **What "fails:"** wireframes detailed enough to be a spec (that's `mockup-brief.md`'s job); implementation details ("we'll use Redis with a 5-minute TTL"); solutions that have no relationship to the Problem.

### User Flows (ten-section only)

Step-by-step walk through the user's journey. **What "passes":** ordered steps with branch points named; happy path + 1-2 critical error/edge paths. **What "fails":** flowcharts so detailed they're a UI spec; one-line "user does the thing" descriptions.

### Acceptance Criteria (ten-section only)

The PRD-scope AC — outcome-level, aggregates across child stories. **What "passes":** EARS-shaped (see `acceptance-criteria-ears`) bullets describing the system's testable post-shipment behavior. **What "fails":** restated story AC; "see linked stories" with nothing else; vague success language ("users will love it").

### Rabbit Holes (pitch only)

Specific traps the team must avoid, each with a named scenario and an avoidance decision — not a generic risk list. See `pitch-writing-shape-up` for passing/failing examples.

### No-Gos (pitch only)

Work explicitly excluded from this appetite. The scope-defense section; empty No-Gos is the most common failure mode. See `pitch-writing-shape-up` for passing/failing examples.

### Out of Scope (ten-section only)

The ten-section analog of No-Gos. Same bar — name what's excluded so engineering doesn't assume it's in.

### Risks (both)

What we'd discover too late if we didn't probe. **What "passes":** specific scenario + what would trigger it + what we'd do. **What "fails":** abstract risk lists ("technical complexity," "user adoption"); risk lists that read as reassurance ("we don't anticipate any major issues").

### Open Questions (ten-section only)

Things the team hasn't decided yet, with a note on who needs to weigh in and by when. **What "passes":** "Should the export include archived items? — needs ops input by sprint planning." **What "fails":** open questions that are silently load-bearing in the Solution; questions with no owner; questions that should have been resolved before the PRD was written.

### Timeline (ten-section only — required under phased preset)

Approximate milestones and dates. Tighter under phased; looser under sprint/continuous. **What "passes":** "Discovery complete by W4; first story handed off W6; expected ship W10." **What "fails":** missing Timeline (blocks `draft → review` under phased); Timeline that contradicts Goals & Metrics measurement windows; precise dates with no margin.

### Linked stories (pitch; implicit in ten-section)

Child stories that decompose the PRD. Populated during `/refine` — empty at draft time is fine. **What "passes":** named story slugs/refs with one-line summaries. **What "fails":** orphaned stories not linked back to the PRD; PRD claimed `approved` with no child stories under cycle preset (the bet hasn't been refined).

## The five-adjective test

See `pitch-writing-shape-up` for the definition (Clarity, Structure, Flexibility, Actionability, Stakeholder focus). `pm-reviewer` runs it on every PRD — pitch or ten-section — before `review → approved`; failure on any one is a blocking finding.

## PRD vs story scope — what belongs where

PRDs frame; stories implement.

| Belongs in PRD | Belongs in linked stories |
|---|---|
| Problem framing, user segment, evidence | "User submits empty cart" — single observable behavior |
| Goals, success metrics, baseline & target | AC for one cohesive behavior |
| Solution sketches, breadboards, flow diagrams | Implementation-adjacent details (still no *how*) |
| Risks at the bet level | Per-story dependencies |
| Out of Scope / No-Gos for the whole bet | Per-story Out of Scope |

PRDs that bundle story content inline (a common ChatPRD-template failure mode) lose the handoff atom — engineering can't pull "the next story" from a wall of PRD prose. The PRD references stories by slug; the stories carry the AC.

## Cross-references

- `prd-anti-patterns` — the smell catalog. Required reading alongside this skill.
- `pitch-writing-shape-up` — deeper guidance on Appetite, Rabbit Holes, No-Gos, and fat-marker sketches.
- `acceptance-criteria-ears` — AC format for PRD-scope and story-scope AC.
- `story-writing-invest` — what the child stories look like.
- `metrics-design` — the Goals & Success Metrics deeper bar.
- `pm-preset-detection` — selects the default template.
