# Delivery audit — roadmap-review

**Audited:** working tree vs HEAD (4 new files + 1 edit, no commit yet)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] `/roadmap-review` invokes `roadmap-reviewer` agent and loads `roadmap-review` skill before survey — `domains/engineering/commands/roadmap-review.md:4-11` (router), `domains/engineering/agents/roadmap-reviewer.md:23-32` ("Load before substantial work" section names both skills)
- [✓] Agent calls `hero size --check`, `hero_warnings`, `hero_list`, `hero_search` on survey — `agents/roadmap-reviewer.md:54-64` (survey table, in order)
- [✓] Prioritizes by 5-item ordering — `skills/roadmap-review/SKILL.md:66-83`, `agents/roadmap-reviewer.md:74-87`
- [✓] Walks one item at a time, proposes exactly one resolution — `skills/roadmap-review/SKILL.md:38-48`, `agents/roadmap-reviewer.md:89-106`
- [✓] Acknowledge runs CLI itself (`hero size --ack` with frontmatter-edit fallback) — `agents/roadmap-reviewer.md:112`
- [✓] `/compose` and `/split` invoke flow + hand off, do not simulate — `agents/roadmap-reviewer.md:113-114`
- [✓] Re-horizon runs `hero size <slug> <new-tier>` itself — `agents/roadmap-reviewer.md:115`
- [✓] Empty survey reports verbatim and exits — `agents/roadmap-reviewer.md:66-71`
- [✓] Exhaustion of priorities 1–4 exits and writes record — `agents/roadmap-reviewer.md:131-133, 156-180`
- [✓] Halt words ("stop," "enough," "done," "later") exit without re-asking — `agents/roadmap-reviewer.md:99-103, 134-135`
- [✓] N=5 cap reports, summarizes, exits — `agents/roadmap-reviewer.md:136-138`; cap defined in skill `skills/roadmap-review/SKILL.md:123-128` (tunable from skill)
- [✓] Session record written to `.hero/knowledge/roadmap-review-sessions/{YYYY-MM-DD}-{HHMM}.md` — `agents/roadmap-reviewer.md:157-180`, format in `skills/roadmap-review/SKILL.md:150-189`
- [✓] Refuses non-sizing lens with scaffolded phrase verbatim — `skills/roadmap-review/SKILL.md:130-148`, `agents/roadmap-reviewer.md:143-153`
- [✓] Improvised resolutions are out of spec — `skills/roadmap-review/SKILL.md:86-87, 205`, `agents/roadmap-reviewer.md:195-196`
- [✓] `hero check` vs `/roadmap-review` distinction in skill body with canonical wording — `skills/roadmap-review/SKILL.md:26-36`, also quoted in agent at `agents/roadmap-reviewer.md:34-43`
- [✓] "Multiple related specs" canonical phrasing in `spec-composition`, cross-referenced (not duplicated) by `roadmap-review` — canonical sentence appears ONLY in `skills/spec-composition/SKILL.md:44-54`; `roadmap-review` skill cross-refs at `:77-78, 197-199`; agent cross-refs at `agents/roadmap-reviewer.md:121-125`. Verified by grep: no copy of the sentence elsewhere.
- [✓] No tracker writes — hard rule named in `agents/roadmap-reviewer.md:193-194` and `skills/roadmap-review/SKILL.md:210`

## Changes

- [✓] `domains/engineering/skills/roadmap-review/SKILL.md` (new) — doctrine, Lenses (Sizing active, Horizons/Releases/Sprint-shape scaffolded), 5-item prioritization, four resolution scripts with paste-ready phrasing, stop condition, session record format including `drift_count_at_exit:`, `hero check` distinction, cross-refs to `spec-sizing` / `spec-composition` / `note-capture`. All present.
- [✓] `domains/engineering/skills/spec-composition/SKILL.md` (new) — canonical "multiple related specs" sentence, composition-discipline framing, cooperative-ownership note for `multi-spec-design-routing` to extend `## Triggers` later. Phrasing lives here only.
- [✓] `domains/engineering/agents/roadmap-reviewer.md` (new) — frontmatter `name: roadmap-reviewer`, `mode: subagent`, `role: review`, `temperature: 0.1` (matches `pr-reviewer.md` pattern per spec). Body covers load-skills, survey, prioritize, walk-one-at-a-time, execute-CLI-on-confirm, four resolution mappings, stop conditions, refusal phrase, mandatory session record with `drift_count_at_exit:`. Hard rules section explicit on no-bulk / execute-on-confirm / no-tracker-writes / no-improvised-resolutions.
- [✓] `domains/engineering/commands/roadmap-review.md` (new) — thin router (17 lines), passes `$ARGUMENTS` as focus slug, mentions sizing-only v1 + scaffolded refusal.
- [✓] `domains/engineering/skills/spec-sizing/SKILL.md` (edit, +7 lines) — adds `roadmap-review` and `spec-composition` cross-refs to the related-skills tail. Does not move sizing phrasing out; size ladder remains source of truth.

## Open items

None. The spec's own Risks section flags the `hero size --ack` flag may not exist yet; the agent prompt handles this explicitly with a frontmatter-edit fallback (`agents/roadmap-reviewer.md:112`). This is an acceptable degradation called out in the spec at `Files to Touch #6` and in Risks — not a defect.

## Audit notes

- **Canonical paths verified.** All four new files + the edit are under `domains/engineering/`; no stray writes to `.claude/`. The Slice-4 convention violation from the parent initiative was not repeated.
- **No duplication of the canonical multi-spec sentence.** Grep for "topically related" finds three occurrences and all are in `spec-composition` (canonical sentence + two framing references). `roadmap-review` skill and the agent both cross-reference, neither copies. Option C honored.
- **Lenses scaffolding stays scaffolded.** Horizons / Releases / Sprint-shape sections at `skills/roadmap-review/SKILL.md:138-148` are each exactly 1–2 lines with only a placeholder marker. The refusal phrase is named once at the section head (`:132-136`) and reused. No real prioritization rules leaked into the placeholders.
- **Walk-one-at-a-time enforced in both surfaces.** Skill `:38-48` and agent `:89-106` both teach the rhythm; agent's "Hard rules" at `:184-198` re-asserts it. The only mentions of "tell user to run X" are in anti-pattern callouts (skill `:46`, agent `:104-105`), correctly inverted.
- **`drift_count_at_exit` named explicitly in three places.** Skill body `:158, 168, 185` plus agent `:71, 164, 176, 198`. Sibling spec #2's read path is preserved.
- **Cross-references resolve.** `roadmap-review` ↔ `spec-sizing` ↔ `spec-composition` form a clean triangle: skill bodies reference each other by name, no broken pointers, no stale "see X" without the X existing.
- **Prose quality.** No hedging language ("consider whether you might want to"), no contradictions between skill and agent on the four resolutions, no missing stop conditions. The agent prompt is tight (~220 lines) and the skill body is doctrine-shaped, not rule-list-shaped. Closing-output template at agent `:201-218` is a concrete receipt format, not vague "summarize what happened."
- **No tests / no Go code is correct for this delivery.** The spec explicitly says "No Go surfaces required" at `Files to Touch #6`. Prose artifacts (skill + agent + command) are not unit-testable in the conventional sense; the spec's Validation section names runtime checks (running `/roadmap-review` on the live workspace) that are out of scope for the audit. No evidence is missing — there is no evidence to be missing.
- **Spec frontmatter edit caught in diff.** `.hero/planning/features/roadmap-review/spec.md` shows a 2-byte change (likely status transition); not a content edit. Not flagged.
