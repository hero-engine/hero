---
name: acceptance-criteria-ears
description: EARS (Easy Approach to Requirements Syntax) patterns for writing acceptance criteria that survive handoff to engineering without ambiguity.
metadata:
  audience: story-writer, prd-author, handoff-coordinator, pm-reviewer
  purpose: ac-authoring
---

## What I do

Provide the five EARS clause patterns and the rules for using them as the default shape of acceptance criteria across PM artifacts. EARS produces criteria that are unambiguously testable — they flip green or red without interpretation. Under the unified type model the same spec carries PM → engineering, so EARS AC survives the owner-flip handoff without any rewriting.

## When to use me

Load this skill when:

- authoring or refining the `Acceptance Criteria` section of a `spec` (any vocabulary — Story / Scope / Card / Issue / Feature)
- writing AC inside a `prd` spec (when the PRD includes AC at all)
- packaging a spec for the owner-flip handoff (`handoff-coordinator`) — AC tightness is what makes the handoff useful
- reviewing PM artifacts (`pm-reviewer`) — vague AC is the most common blocking finding

## Why EARS

EARS was introduced by Alistair Mavin (Rolls-Royce, 2009) as a constrained-natural-language pattern for requirements. The constraint is the point — fitting a sentence to a clause shape forces the author to name *when* the behavior fires, *what state* triggers it, and *what the system does in response*. Vague predicates ("when appropriate," "if needed," "for the relevant user") don't survive the template.

Hero adopts EARS as the cross-domain AC default because:

- engineering's `/design` and `/deliver` commands already produce and consume EARS-shaped specs — using EARS in PM means zero translation when `owner` flips to engineering (it's the same spec)
- the five patterns cover the majority of real-world AC shapes without forcing
- the format is dense and testable; one EARS clause typically maps to one test case
- it's not a methodology — it's a sentence template, so it doesn't lock the team into Scrum, Shape Up, or anything else

## The five patterns

### Ubiquitous

Always-true behavior. No trigger; the system simply must.

```
THE SYSTEM SHALL <behavior>
```

**Examples:**
- `THE SYSTEM SHALL log every authentication attempt with timestamp, user-id, and outcome.`
- `THE SYSTEM SHALL retain audit records for at least 7 years.`
- `THE SYSTEM SHALL display the current logged-in user's email in the header.`

**Use for:** invariants, compliance requirements, omnipresent UI elements, logging/audit baselines.

### Event-driven (WHEN)

Behavior triggered by a discrete event.

```
WHEN <trigger event> THE SYSTEM SHALL <behavior>
```

**Examples:**
- `WHEN a user submits an empty cart THE SYSTEM SHALL display an "add items to continue" message.`
- `WHEN an order's payment is confirmed THE SYSTEM SHALL send an order-confirmation email within 60 seconds.`
- `WHEN a user clicks the export button THE SYSTEM SHALL begin a CSV download.`

**Use for:** the vast majority of interactive behavior. If you can name the trigger as a verb-phrase event, use WHEN.

### State-driven (WHILE)

Behavior that persists while a condition is true.

```
WHILE <state> THE SYSTEM SHALL <behavior>
```

**Examples:**
- `WHILE the user's session has unsaved changes THE SYSTEM SHALL show an unsaved-changes indicator in the header.`
- `WHILE a deploy is in progress THE SYSTEM SHALL block new deploy initiations.`
- `WHILE the user has read-only permission THE SYSTEM SHALL disable all edit controls.`

**Use for:** modal states, persistent UI affordances, mode-locked behavior. Distinguish from WHEN: WHILE is a duration, WHEN is a moment.

### Unwanted-behavior (IF / THEN)

Error and edge-case handling — what the system does when something goes wrong or violates an expected condition.

```
IF <unwanted trigger> THEN THE SYSTEM SHALL <behavior>
```

**Examples:**
- `IF the payment provider returns a 5xx error THEN THE SYSTEM SHALL retry up to 3 times before surfacing the error to the user.`
- `IF a user attempts to access a resource they don't own THEN THE SYSTEM SHALL return a 403 and log the attempt.`
- `IF the uploaded file exceeds 10MB THEN THE SYSTEM SHALL reject the upload with a clear size-limit message.`

**Use for:** validation failures, error recovery, security boundary violations, rate limits. IF/THEN is what makes negative-path testing explicit; without it, most teams ship the happy path and discover edge cases in production.

### Optional-feature (WHERE)

Behavior gated on a feature flag, plan tier, or environment.

```
WHERE <feature/condition> IS ENABLED THE SYSTEM SHALL <behavior>
```

**Examples:**
- `WHERE one-click-checkout IS ENABLED THE SYSTEM SHALL bypass the cart review step.`
- `WHERE the user IS ON the enterprise plan THE SYSTEM SHALL expose the SSO configuration page.`
- `WHERE the build IS staging THE SYSTEM SHALL display the "non-production" banner.`

**Use for:** flagged behavior, plan-tier gates, environment-conditional UI. WHERE makes the gate explicit at the AC level — testers and reviewers know what flag state to set.

## When to fall back to freeform bullets

EARS is the default. It is not mandatory. Freeform is the right shape when:

- The criterion is a **non-functional requirement** with a numerical target — "Search results return in under 200ms at p95 under 1k rps." Forcing this into WHEN/SHALL adds nothing.
- The criterion is a **referenced standard** — "Conforms to WCAG 2.1 AA." Wrapping in EARS dilutes the reference.
- The criterion is a **data-shape constraint** — "Export columns: order_id, customer_email, total_cents, currency, created_at." A table or list is clearer than five WHEN clauses.
- The criterion describes **content** — "Email subject reads 'Your order has shipped'." EARS is verbose for static-content checks.

Mix freely. A story with three EARS clauses, a freeform numerical target, and a data-shape list is normal and correct.

## What "testable" means

The single test for any AC bullet — EARS or freeform — is *can a reviewer point at the system and say "we built this" or "we didn't"*?

Failing examples:

- "Works well for most users." (Who's "most"? What's "well"?)
- "Performs reasonably under load." (Define load. Define reasonable.)
- "When appropriate, the system should consider notifying the user." (Three weasel words: appropriate, should, consider.)

Passing rewrites:

- "P95 latency under 200ms with 1,000 concurrent users."
- "When an order's status changes to `shipped` THE SYSTEM SHALL email the customer within 60 seconds."

## Story-scope AC vs PRD-scope AC

AC at different artifact scopes serves different purposes.

**Story-scope AC:**
- Granular, behavior-level.
- Maps 1:1 (or close) to engineering test cases.
- Bounded by the story's INVEST `Small` constraint — typically 3-7 bullets.
- Default shape: EARS clauses.

**PRD-scope AC:**
- Outcome-level or epic-level.
- Aggregates across child stories.
- Often references story AC rather than restating them.
- Sometimes higher-level success criteria ("X workflow completable in under 5 clicks").
- Lives in the PRD's `Acceptance Criteria` section (ten-section template) or rolled into `Solution` / `Goals` (pitch template).

PRD AC that duplicates child-spec AC is wasted; PRD AC that *only* lists "see linked specs" is also wasted. PRD AC is the rollup test — what's true when all the child specs have shipped.

## Writing AC that survives handoff

The `handoff-coordinator` agent flips `owner: pm → engineering` on the same spec. Tight AC is what makes the handoff useful — vague AC means engineering has to re-derive intent from the description, which is the exact friction the handoff is supposed to remove. There is no re-authoring step on the engineering side; the AC PM wrote is the AC engineering delivers against.

Pre-handoff AC checklist:

- Every bullet has an explicit trigger or state (or is ubiquitous).
- Every bullet's behavior is observable (the tester can point at the result).
- Every error path the user can reach has an IF/THEN clause.
- Every feature-flagged behavior has a WHERE clause.
- No bullet says "as appropriate," "when needed," "if possible," "should consider."
- No bullet says "click the X button" (prescribes UI implementation) unless the UI is locked by a sibling design spec.

The `pm-reviewer` runs this checklist as a blocking gate on the `refined → ready` transition.

## Anti-patterns

- **Forcing EARS where it doesn't fit.** A numerical p99 target wrapped in `WHEN load is high THE SYSTEM SHALL be fast` is worse than `P99 latency < 300ms at 500 rps`. Use freeform.
- **Vague predicates inside EARS.** `WHEN appropriate THE SYSTEM SHALL notify the user` — the EARS shape is right; the trigger is meaningless. The pattern doesn't save vague content.
- **Implementation-shaped AC.** `WHEN the user clicks the green button in the modal's top-right corner THE SYSTEM SHALL POST to /api/v2/users/:id/preferences with {notify: true}` — that's a test of a specific implementation, not of behavior. Engineering owns the *how*.
- **Missing IF clauses.** AC section lists only WHEN bullets — only the happy path. The PRD ships, engineering builds it, QA finds the negative paths in staging. Every WHEN should be considered for a matching IF.
- **EARS theater.** Every bullet starts with WHEN even when it shouldn't, because someone enforced "all AC must be EARS." The pattern is a tool, not a religion.
- **Gherkin / EARS mixing inside one story.** Pick one shape per story. (`acceptance-criteria-gherkin` covers the Gherkin shape.)
- **AC that restates the description.** "User can export CSV. WHEN a user exports CSV THE SYSTEM SHALL allow CSV export." Tautology. The AC must add specificity the description doesn't have.

## Cross-references

- `story-writing-invest` — INVEST is the story shape; EARS is the AC shape. Both are required.
- `prd-structure` — PRD templates that house AC sections.
- `handoff-protocol` — the owner-flip protocol. Under the unified type model the same spec carries through; tight AC here means a frictionless transition without re-authoring.
- Engineering's `/design` and `/deliver` commands — already produce and consume EARS-shaped specs. PM AC must be EARS-compatible because the engineering side reads the exact AC PM authored.
