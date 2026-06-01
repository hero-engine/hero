# Delivery audit — multi-spec-design-routing

**Audited:** `git diff HEAD -- domains/ .hero/planning/features/multi-spec-design-routing/`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Fire routing nudge on explicit multi-deliverable phrasing — `domains/engineering/skills/spec-composition/SKILL.md:63-72` (Trigger #1, patterns "design X, Y, and Z" / "and also" / "plus") + lead pre-flight wiring at `feature-delivery-lead.md:47` and `platform-delivery-lead.md:45` (fire "before writing any individual spec").
- [✓] Fire routing nudge on lead-identified sub-deliverables — `spec-composition/SKILL.md:73-78` (Trigger #2) + `commands/design.md:10` ("you spot independent sub-deliverables during clarification").
- [✓] Fire routing nudge on rolled-up midpoint-sum ≥ `large` — `spec-composition/SKILL.md:80-86` (Trigger #3, cites `internal/snapshot/rollup.go` for midpoint-sum mechanic).
- [✓] Quote canonical phrasing from skill rather than improvising — `spec-composition/SKILL.md:43-54` (Canonical phrasing) + per-trigger variants at lines 98-123; agent pre-flights direct leads to "fire the routing nudge **from the skill**" rather than author wording inline.
- [✓] Pivot to `/compose` with original `/design` request as initiative input, no flat sibling scaffolded — `spec-composition/SKILL.md:143-146` ("hands off cleanly into the existing `/compose` flow with the user's original `/design` request as the initiative input — it does not simulate `/compose` or scaffold flat siblings").
- [✓] Proceed with siblings on decline and suppress further routing nudges that session — `spec-composition/SKILL.md:137-142` (record choice, one decline covers whole session) + Suppression rule #2 at lines 194-198.
- [✓] Suppress when `/design` was launched from a parent `/compose` flow — `spec-composition/SKILL.md:188-191` (Suppression rule #1).
- [✓] When both routing and sizing nudges would fire, routing fires first; sizing fires per-child only if user declines — `spec-composition/SKILL.md:151-181` (Precedence section, explicit "accept routing" vs "decline routing" branches) + cross-reference at `spec-sizing/SKILL.md:263-272`.
- [✓] No re-fire of routing nudge on subsequent specs after explicit opt-in to siblings — `spec-composition/SKILL.md:194-198` (Suppression rule #2: "One decline covers the whole session — do not re-litigate on each sibling").
- [✓] Author `## Triggers` if not yet present (cooperative ownership) — pre-delivery file from `e15f37d` had a placeholder note in `## Triggers`; post-delivery it is real content at lines 56-88. Placeholder replaced in-place, not appended.
- [✓] Document advisory "user always wins" stance matching `spec-sizing` — `spec-composition/SKILL.md:129-149` (`## Stance` section explicitly says `word-for-word on the override behavior` and `friction with an off-ramp, not a gate`).
- [✓] Document precedence rule (routing before sizing) in `spec-composition` — `spec-composition/SKILL.md:151-181` (`## Precedence` section).
- [✓] Update both delivery-lead agents to load `spec-composition` in design pre-flight — `feature-delivery-lead.md:47` (one-paragraph insert after existing `spec-format`/`spec-sizing` load line) + `platform-delivery-lead.md:45` (matching insert, with platform-specific framing).
- [✓] Update `commands/design.md` with paragraph parallel to `spec-sizing` reference — `commands/design.md:10` (paragraph inserted right after the existing `spec-sizing` paragraph at line 8).
- [✓] Add "see also" cross-reference from `spec-sizing` to `spec-composition` — `spec-sizing/SKILL.md:263-272` (extends existing bullet in "Composing with related skills"; mentions routing-fires-first, sizing-per-child, "One nudge per moment").

## Changes

- [✓] `domains/engineering/skills/spec-composition/SKILL.md` — Triggers (any-of, three triggers), Phrasing (three trigger-variant blocks), Stance, Precedence, Suppression sections all present at canonical paths. Goal and Canonical phrasing from sibling #1 left intact verbatim; pre-existing `## User always wins` section consolidated into the fuller `## Stance` section (no duplicate).
- [✓] `domains/engineering/agents/feature-delivery-lead.md` — one paragraph inserted at line 47, additive to existing `spec-format`/`spec-sizing` load instructions.
- [✓] `domains/engineering/agents/platform-delivery-lead.md` — matching paragraph at line 45 with platform-specific framing (migrations/refactors/scaling routinely produce multi-spec scopes).
- [✓] `domains/engineering/commands/design.md` — paragraph at line 10, parallel structure to the existing `spec-sizing` paragraph at line 8.
- [✓] `domains/engineering/skills/spec-sizing/SKILL.md` — existing `spec-composition` bullet in "Composing with related skills" extended from a brief reference into a routing/sizing precedence pointer.

## Open items

None. Engineer's flagged judgment call (one-paragraph inserts in delivery leads instead of strict one-sentence per the spec's Changes wording) is a discoverability win, not a scope expansion — the inserts say exactly what the spec says they should say and stay within the "no restructure of the existing pre-flight" boundary.

## Audit notes

- **Cooperative ownership held cleanly.** Sibling #1's pre-delivery file (`git show e15f37d`) had Goal, Canonical phrasing, and a placeholder `## Triggers` note. Post-delivery file preserves Goal and Canonical phrasing verbatim, replaces the placeholder with real Triggers content, and absorbs sibling #1's brief `## User always wins` section into the fuller `## Stance` section. No duplicate stance, no leftover placeholder.
- **Any-of vs AND is unambiguous.** Section heading ("any of the following is true"), preamble ("single trigger is enough"), and footer ("Any-of. One trigger is sufficient.") all triple-affirm.
- **Precedence is consistent across both skills.** Routing fires first; sizing stands down; sizing resurfaces per-child either way. `spec-sizing` cross-reference uses identical framing.
- **No `.claude/` writes.** Confirmed via `git diff --name-only`. All five edits at canonical paths.
- **Stop conditions for the nudge are present and adequate.** Three Suppression rules (in-`/compose`, post-decline session, precedence with sizing) plus conservative-bias guidance on trigger #2 ("Lean toward *not* firing if uncertain") and trigger #3 fallback when size data isn't available.
- **Cross-references valid.** `internal/snapshot/rollup.go` cited in Trigger #3 exists on disk. Bidirectional skill cross-refs (`spec-composition` ↔ `spec-sizing`) in place.
- **Status flip is correct.** Spec moved from `ready` to `delivering`, not prematurely to `completed`. Audit is happening at the right moment.
- **Prose-only delivery shape matches the spec.** No code changes expected or made; the lead's reasoning is the implementation surface, as the spec's Changes section states.
