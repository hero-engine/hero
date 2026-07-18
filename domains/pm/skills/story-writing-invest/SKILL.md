---
name: story-writing-invest
description: INVEST shape for user stories — Independent, Negotiable, Valuable, Estimable, Small, Testable. The bar every story must clear before handoff to engineering.
metadata:
  audience: story-writer, pm-delivery-lead, pm-reviewer
  purpose: story-authoring
---

## What I do

Provide the INVEST framework as the quality bar for the dev-ready PM atom that hands off into engineering. Under the unified type model the canonical type is `spec` (with `kind: feature` by default); the active vocabulary preset renders it as "Story" (agile-scrum), "Scope" (shape-up), "Card" (kanban), etc. **INVEST discipline applies regardless of the vocabulary** — the displayed name does not change the quality bar. This skill is named for the historical phrase ("INVEST stories"); the substance applies to any `spec, kind: feature` artifact.

INVEST is the shape that lets a spec survive negotiation with engineering, fit one delivery cycle, and produce an unambiguous "done" signal.

## When to use me

Load this skill when:

- authoring a new `spec` (any vocabulary — Story / Scope / Card / Issue / Feature)
- refining a spec before flipping its `status` to `ready`
- reviewing a spec prior to the owner-flip handoff (`pm-reviewer`, `handoff-coordinator`)
- splitting an oversized spec (`Small` failure)
- triaging an intake that's been promoted to a feature

## The six adjectives

INVEST originated with Bill Wake (2003). It is the most widely cited story-quality bar in product management. Each letter is independently testable; a story that fails any one of them is not ready.

### I — Independent

A story can be delivered without waiting on another in-flight story. Independence is about *delivery sequencing*, not "this story has zero relationship to others."

- **Passes:** "User can export their cart as CSV." Ships on its own; nothing else has to land first.
- **Fails:** "Wire the export button to the new export service." Depends on a sibling story that builds the service. Either merge them or sequence them explicitly with a hard dependency.

**Fix:** If two stories must ship together, they're one story. If one truly depends on the other, mark the dependency in the `Dependencies` section and accept that the dependent story can't go `ready` until the blocker ships.

### N — Negotiable

The story leaves room for engineering judgment on the *how*. It states the *what* and the *done* line — not the implementation.

- **Passes:** "User can recover their password without contacting support."
- **Fails:** "Add a `POST /password-reset` endpoint that sends a Mailgun template with a 24-hour JWT token."

**Fix:** Strip implementation language. If the team needs a specific implementation (security constraint, contract with an external system), put that in `Out of Scope` or `Dependencies`, not the story body.

### V — Valuable

The story names user or business value clearly. "Valuable" means a stakeholder cares about the outcome — not that engineering cares about the work.

- **Passes:** "User can resume a partially-completed signup so we stop losing 14% of conversions at step 3."
- **Fails:** "Refactor the signup state machine." (Engineering value, not user value. That's a `chore`.)
- **Fails:** "Because the PRD says so." (No actual user named.)

**Fix:** Use the canonical *As a {role}, I want {capability}, so that {value}* shape when value is unclear — but don't fetishize the format. A description paragraph is fine if value is named.

### E — Estimable

Engineering can size the work without further discovery. If the team can't say "this is roughly a week" or "this is more than the cycle's appetite," the story isn't estimable.

- **Fails:** "Add machine learning to the search results." (What model? What signal? What evaluation?)
- **Fails when:** the AC section says "TBD"; the technical approach requires research; a dependency is uncosted.

**Fix:** If estimation requires discovery, the story is premature — promote it back to an initiative or trigger a discovery spike before flipping `ready`.

### S — Small

The story fits one delivery cycle. Definitions:

- Under **sprint** preset: ≤ ~half the sprint's velocity (so it can't dominate).
- Under **cycle** preset: fits inside the assigned appetite (small = 1-2 weeks; big = 6 weeks).
- Under **continuous flow**: fits inside the team's WIP age budget (typically < 1 week).

A story that spans two cycles is an epic in disguise. See "Splitting non-S stories" below.

### T — Testable

Acceptance criteria flip green or red unambiguously. "Works well" is not testable; "responds within 200ms at p95 under 1k rps" is.

This is where INVEST meets EARS (see the `acceptance-criteria-ears` skill). EARS clauses are designed to be unambiguously testable; freeform bullets are fine when they're equally precise. The bar isn't EARS — the bar is *can a reviewer point at the artifact and say "we built it" or "we didn't"*?

## Sizing heuristics

Rough sizing without estimation theater:

- **One story** = one PR (or one tight series of PRs by one engineer in one cycle).
- **One developer-week** is the rule-of-thumb upper bound for a single story.
- If three or more engineers would touch it in parallel, it's an epic.
- If the AC section has more than ~7 criteria, you're probably looking at multiple stories.
- If the description paragraph needs more than one paragraph, it's an epic.

## Splitting a non-S story

A story that fails Small needs to be split. Common splits (in priority order):

1. **By user flow step.** "User can complete signup" → "User can submit signup form" + "User can verify email" + "User can choose plan."
2. **By data variation.** "User can export reports" → "User can export the last 30 days" + "User can export an arbitrary date range" + "User can export multiple report types."
3. **By rule complexity.** "Pricing supports promotions" → "Pricing applies a single fixed-percent promo" + "Pricing stacks multiple promos with precedence rules."
4. **By acceptance criterion.** When one AC bullet is itself a story's worth of work, split it out.
5. **By happy path vs edge case.** Ship happy path first; edge cases follow as separate stories with explicit triggers.

Avoid splitting by *technical layer* ("the backend story" + "the frontend story"). That violates Independent and creates the "wire it up" anti-pattern.

## INVEST and EARS — how they relate

INVEST is the *story shape*. EARS is the *acceptance criteria shape*. They serve different purposes and you need both:

- INVEST without EARS → a well-shaped story with vague AC. Fails T.
- EARS without INVEST → tight criteria attached to a story that's too large, has no value statement, or prescribes implementation. The criteria are testable but the story is unshippable.

Load `acceptance-criteria-ears` alongside this skill when writing or reviewing stories.

## `kind` typing — feature / bug / chore / refactor / perf / infra / security / ux

Under the unified type model, the PM-common work types are `feature`,
`bug`, `chore`. Engineering-originated work may use additional types
from the registered spec types. Set at draft time; drives priority
defaults and review rigor.

- **feature** — new user-facing capability. Full INVEST + EARS bar. Goes through `pm-reviewer` before the owner-flip handoff.
- **bug** — broken behavior the user experiences. AC describes the *correct* behavior (not "fix the bug"). Reproduction steps belong in the description; the AC is the post-fix expectation. INVEST applies but Negotiable is looser — sometimes the only valid fix is one specific thing.
- **chore** — engineering-internal work with no direct user value (refactor, dependency upgrade, internal tooling). User-value test (V) is waived; the value statement names the engineering or operational benefit. AC may be lighter — "tests still pass" is sometimes sufficient. Chores authored by PM are rare; most are engineering-originated and never go through the PM authoring path.
- **refactor / perf / infra / security / ux** — engineering-originated shapes; PM rarely authors these but may inherit them when engineering hands a spec back with `handed_back_reason` asking for product clarification.

If you can't decide between *feature* and *chore*, the question is *who asked for this*. Customer-asked or PM-asked → feature. Engineering-asked → chore (or one of the engineering-originated kinds).

## Common smells

- **The "wire it up" story** — "Hook the new endpoint to the dashboard." Almost always means two stories were split badly. Recombine or sequence with explicit dependency.
- **The technical task masquerade** — "Add a feature flag for X." That's a chore inside the actual story, not a story on its own.
- **The epic in story clothing** — Description paragraph keeps growing; AC list keeps growing; sizing keeps slipping. Promote to an epic (`epic-framer` frames it).
- **The implementation-as-value** — Value statement is "so that the system uses the new architecture." No actual user is named. Reframe or downgrade to chore.
- **The everything-bagel AC** — AC list mixes happy path, edge cases, security requirements, and performance targets. Split by AC, or move non-functional requirements to a linked PRD.
- **The spec without a `kind`** — Defaults shouldn't be implicit. Force the kind decision at draft time.
- **The "TBD" story** — Status `ready` with TBD sections. Fails Estimable by definition. Block the status flip.
- **The "as a user" story with no user** — Roles like "user," "system," "admin" carry no information when undifferentiated. Name the actual segment ("returning shopper," "first-time visitor," "ops on-call").

## Anti-patterns to refuse

When `story-writer` or `pm-reviewer` is asked to flip a story to `ready`, refuse if:

- AC section is empty or `TBD`.
- Story prescribes implementation in the description body.
- Spec has no `kind`.
- Story has no clear user value (and isn't typed as `chore`).
- Story exceeds the active preset's small-story budget.
- Story has unresolved hard dependencies on stories not yet `ready`.

These aren't suggestions. A story that violates these will cost more friction at handoff than it saves by being flipped early.
