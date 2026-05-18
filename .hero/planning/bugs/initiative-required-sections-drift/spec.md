---
title: initiative spec-type's required-sections YAML disagrees with its prose docs
type: bug
status: planning
severity: low
priority: P2
created: 2026-05-17
tags: [spec-types, content, drift, initiative]
relations:
  - target: hero-code-handover-pack
    kind: surfaced-by
---

# initiative spec-type's required-sections YAML disagrees with its prose docs

## Problem

`core/spec-types/initiative.md` carries an internal inconsistency between its YAML frontmatter and its prose body:

- YAML `sections.required:` declares `[Goal]`.
- Prose section docs describe required sections as `[Bet, Evidence, Tradeoffs]`.

Authors reading the prose to author an initiative spec will write the prose-required sections; the registry validator (which reads the YAML) will accept (or reject) on a different basis. Surfaced by the C5 agent (sample scrum workspace work item, `hero-code-handover-pack`) when authoring `examples/scrum-workspace/.hero/planning/initiatives/q3-conversion-lift/spec.md` — the agent included sections satisfying both readers but flagged the inconsistency.

## Steps to Reproduce

```
grep -A 4 "^sections:" core/spec-types/initiative.md
# Shows: required: [Goal]

grep -B 1 -A 8 "## Required" core/spec-types/initiative.md  # or similar prose
# Shows prose describing Bet / Evidence / Tradeoffs as required
```

## Expected Behavior

YAML and prose agree on the required sections. The YAML is the registry-authoritative source (B2 spec-type-registry parses it); prose must match.

## Root Cause

The initiative type file was authored across two passes:

1. Original PM-pack version at `domains/pm/spec-types/roadmap-item.md` carried the `[Bet, Evidence, Tradeoffs]` prose framing (Shape Up–flavored).
2. A1 of `pm-foundation-delivery` (Track A content authoring) authored the canonical `core/spec-types/initiative.md` and pinned the YAML at `[Goal]` (more methodology-neutral) but did not update the prose body inherited from the PM version.

Result: the prose still reads as Shape Up–shaped (Bet/Evidence/Tradeoffs) while the registry sees `[Goal]` only.

## Fix

Pick a single source of truth and align the other.

**Recommend keeping YAML at `[Goal]`** (methodology-neutral; methodology profiles can layer required sections later via a `sections_overrides:` block — same pattern as `lifecycle_overrides`). Update the prose to describe `Goal` as required; demote Bet / Evidence / Tradeoffs to **suggested** sections (optional) with a note that Shape Up workspaces typically require them.

Touch:
- `core/spec-types/initiative.md` — update prose `## Sections` description.
- Optional: if any in-tree initiative specs were authored against the prose framing (with `Bet` / `Evidence` / `Tradeoffs` but no `Goal`), add a `Goal` section to those specs. Likely zero — most initiatives in `.hero/planning/initiatives/` followed the YAML.

## Acceptance Criteria

- THE SYSTEM SHALL keep `core/spec-types/initiative.md`'s `sections.required:` YAML at `[Goal]`.
- THE SYSTEM SHALL update the prose body to describe `Goal` as the single required section.
- THE SYSTEM SHALL describe `Bet`, `Evidence`, `Tradeoffs` as suggested / optional sections in the prose, with a note about Shape Up alignment.
- THE SYSTEM SHALL leave `internal/spectypes/` and the JSON cache export unchanged — the bug is content, not code.
- WHEN a spec author reads `core/spec-types/initiative.md`, THE SYSTEM SHALL present a single coherent answer to "what sections are required for an initiative?"

## Boundaries

- **Not** changing the YAML's `required:` field — this is the registry contract.
- **Not** authoring `sections_overrides:` in methodology profiles. That's a separate feature if/when needed.
- **Not** touching other spec-type files. This is one-file drift.
- **Not** migrating existing initiative specs unless any are actually missing the `Goal` section (verify; likely zero).

## Validation

- After fix, `grep -A 4 "^sections:" core/spec-types/initiative.md` and the prose reading both name `Goal` as the (sole) required section.
- `go test ./internal/spectypes/...` clean (no code touched; tests should be unaffected).
- Walk `.hero/planning/initiatives/*/spec.md` and confirm every existing initiative has a `## Goal` section. If any don't, add one before declaring done.

## Kickoff

> Read `.hero/planning/bugs/initiative-required-sections-drift/spec.md` (this file) and `core/spec-types/initiative.md`. Update the prose body to describe `Goal` as the sole required section; demote `Bet`, `Evidence`, `Tradeoffs` to suggested/optional with a Shape Up reference. Walk `.hero/planning/initiatives/*/spec.md` and confirm each has a `## Goal` section (likely all do — verify, don't assume). Run `go test ./internal/spectypes/...` clean (no Go changes expected). Report what shipped and whether any existing initiatives needed a `Goal` section added, under 200 words.
