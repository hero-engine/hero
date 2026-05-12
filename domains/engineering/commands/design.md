---
description: Produce a spec for a feature, enhancement, or platform change.
---
Route this feature or enhancement request to the appropriate delivery lead for design.

Determine whether this is product feature work or platform/migration work:
- Product features, user-facing enhancements, and new capabilities → delegate to `feature-delivery-lead`
- Migrations, refactors, platform changes, scaling work → delegate to `platform-delivery-lead`

The delivery lead will produce a spec document and save it to the hero planning folder.

**Before proposing a design**, call `hero_anchor` with the feature context to load project mission and active tripwires. If any design direction conflicts with a tripwire, eliminate it before proceeding — do not present forbidden options as alternatives.

The spec's `## Acceptance Criteria` section should prefer EARS (Easy Approach to Requirements Syntax) patterns where they fit. These give downstream tools (`hero test generate`, `hero spec lint`, drift detection) structured input:

- `WHEN <event> THE SYSTEM SHALL <behavior>` — event-driven
- `WHILE <state> THE SYSTEM SHALL <behavior>` — state-driven
- `IF <trigger> THEN THE SYSTEM SHALL <behavior>` — unwanted/guard condition
- `WHERE <feature> IS ENABLED THE SYSTEM SHALL <behavior>` — optional feature
- `THE SYSTEM SHALL <behavior>` — ubiquitous

Freeform bullets remain valid for criteria that genuinely don't fit any pattern. Mix as needed; do not force EARS onto criteria where it reads awkwardly.

After saving the spec, if `knowledge.auto_capture` is enabled in `hero.json` (default: on), silently review the design session for novel learnings — architectural decisions made, constraints discovered, technology choices with rationale. If anything meets the capture threshold, write it to `.hero/knowledge/` and run `hero index`. Don't prompt — just capture and briefly mention what was saved. See the `auto-knowledge-capture` skill for details.

Request: $ARGUMENTS
