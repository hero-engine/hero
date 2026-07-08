---
name: prd-anti-patterns
description: The PRD smells that make a document unusable — what they look like, why they fail, and what to write instead.
metadata:
  audience: prd-author, pm-reviewer
  purpose: prd-review
---

## What I do

Provide the concrete failure modes PRDs fall into — the smells `pm-reviewer` flags as blocking findings, and the patterns `prd-author` and `prd-author-scrubber` (P1, not yet shipped) actively prevent. This is the negative counterpart to `prd-structure`: that skill says what good looks like; this skill says what bad looks like, why it fails, and the fix.

## When to use me

Load this skill when:

- reviewing a PRD before it advances to `approved` (`pm-reviewer`)
- authoring a PRD and self-checking against known traps (`prd-author`; `pitch-author` ships v1.5)
- running the planned PRD scrubber (`prd-author-scrubber` — P1)
- sanity-checking an inherited or externally-drafted PRD

## The five-adjective bar (recap)

From ChatPRD / MetaGPT. Every PRD must pass:

- **Clarity** — readable in one pass by someone unfamiliar.
- **Structure** — every required section present and earning its place.
- **Flexibility** — leaves the *how* to engineering.
- **Actionability** — engineering can decompose into stories without back-and-forth.
- **Stakeholder focus** — exec, customer, engineering, sales each find what they need.

Each anti-pattern below maps to at least one failed adjective. The fix restores it.

## The anti-pattern catalog

### 1. The prescriptive PRD

**Pattern:** PRD describes implementation. "Use Redis for caching. Build a new microservice. Hit the `/users/v2/preferences` endpoint with a JWT-signed payload."

**Why it fails:** Violates Flexibility (the *N* in INVEST, scaled to PRD). Engineering's job is the *how*; the PRD's job is the *what* and *why*. Prescriptive PRDs short-circuit engineering judgment and ship worse implementations because the PM made architecture choices outside their domain.

**Fix:** Strip implementation language. If a specific tech choice is a hard constraint (security, compliance, contract with an external system), name it in `Out of Scope` or `Risks` with the *why*, not as a Solution detail. Under the unified type model, implementation details live in each child spec's `plan.md` (companion artifact authored by engineering after the owner flip) — not in the PRD body.

### 2. The empty-No-Gos PRD (cycle preset)

**Pattern:** Pitch template, No-Gos section present but blank or trivial ("No-Gos: TBD").

**Why it fails:** No-Gos are the scope-defense section. Under Shape Up, the team commits to ship within Appetite — but if scope can creep silently, the team blows the appetite and Cooldown vanishes. Empty No-Gos guarantee creep.

**Fix:** Every pitch has at least one No-Go. "No mobile app changes this cycle." "No new admin UI." "Not handling the multi-tenant case." If the author genuinely can't name an exclusion, the bet isn't shaped — push back to discovery. `pm-reviewer` blocks `draft → review` on empty No-Gos.

### 3. The intake-paraphrase PRD

**Pattern:** Problem section reads like a copy-paste of the originating intake or initiative — same customer quote, same paraphrased ask, no synthesis.

**Why it fails:** PRDs are the artifact where signal becomes *direction*. A PRD that paraphrases intake without synthesizing it hasn't done the work — the PM is acting as a forwarder, not an authorer. Engineering reads the PRD and is no better-informed than reading the intake itself.

**Fix:** Synthesis means: name the underlying opportunity (not the surface ask), aggregate evidence across multiple intakes (not just the loudest one), state what we'd *learn* by shipping (not just what we'd build). Reference `evidence-synthesis` and `opportunity-solution-trees-torres` skills. If the Problem section can be replaced by the intake's body without loss, the PRD hasn't earned its existence.

### 4. The vanity-metric PRD

**Pattern:** Goals & Success Metrics section names "engagement," "satisfaction," or "adoption" with no definition, no baseline, no observation method.

**Why it fails:** Vanity metrics can't fail. A target like "increase engagement" is always plausibly met; if it's not, the metric definition gets revised in the retrospective. The team learns nothing, and principle #5 (learn from what shipped) silently fails.

**Fix:** Every metric needs: named definition (the SQL or event you'd actually query), baseline (current value, with the measurement date), target (the value that defines success, with rationale for *why* that target), and observation method (where the data lives, who runs the query, on what cadence). Cross-reference `metrics-design` for the deeper bar. If metrics aren't ready, mark the section "N/A — metrics deferred to v1.5" with the rationale, rather than shipping vague counters.

### 5. The missing-Appetite pitch (cycle preset)

**Pattern:** Pitch template, Appetite section absent or set to a duration without rationale.

**Why it fails:** Appetite is *the* Shape Up constraint — it's the budget the team agrees to spend, full stop. Without Appetite, the team estimates inside the cycle, which is the trap Shape Up was built to prevent.

**Fix:** Pick small (1-2 weeks) or big (6 weeks). Name the rationale in one sentence ("Small — we don't yet know if customers will use this; ship a thin slice and learn"). `pm-reviewer` blocks `draft → review` on missing Appetite under cycle preset.

### 6. The missing-Timeline PRD (phased preset)

**Pattern:** Ten-section template under phased preset, Timeline section absent, vague ("TBD"), or with no internal milestones.

**Why it fails:** Phased delivery depends on date-bound milestones — discovery complete, first story ready, beta launch, GA. A PRD without Timeline can't be planned against a release.

**Fix:** Approximate dates are fine — name the milestones and a rough window for each. If dates aren't known, name what's blocking them ("Timeline pending platform team's availability — to confirm by W3"). Under phased preset, `pm-reviewer` blocks `draft → review` on missing Timeline.

### 7. The vague-success PRD

**Pattern:** Success criteria written as adjectives — "great UX," "scalable architecture," "delightful experience." Acceptance Criteria section reads as marketing copy.

**Why it fails:** Untestable. "Delightful" can't ship as green or red. The PRD enters delivery and engineering invents their own success criteria, often misaligned with what the PM intended.

**Fix:** Every success bullet is observable. Replace adjectives with measurements ("p95 latency < 300ms"), behaviors ("user completes signup in under 5 clicks"), or counts ("zero P0 bugs in the first week post-launch"). Cross-reference `acceptance-criteria-ears` for the AC shape.

### 8. The "TBD" final PRD

**Pattern:** PRD marked `approved` or moved to `delivered` with sections containing "TBD," "to be determined," "pending review," or placeholder text.

**Why it fails:** Approval implies the PRD is decision-bearing. TBD sections are unresolved decisions ratified by accident. Engineering reads TBD as "do whatever," ships something, and the disagreement surfaces in retrospective.

**Fix:** Resolve every TBD before approval, OR explicitly mark it as `Out of Scope` / `Open Question` with a named owner and timeline ("Open: pricing tier mapping — needs CRO input by W2"). `pm-reviewer` blocks `review → approved` on any unresolved TBD.

### 9. The over-long PRD that should have been an epic

**Pattern:** PRD exceeds 5-7 pages, has 12+ child stories, spans multiple bets or audiences.

**Why it fails:** PRDs frame one bet; epics group one bet's stories. A PRD with 12 stories crossing three user segments is actually three bets in a trench coat. Engineering can't plan against it; reviewers can't review it in one pass; metrics can't isolate one bet's impact from the others.

**Fix:** Decompose. Split the PRD into multiple bets, each its own initiative with its own PRD. Use the epic spec type for grouping related stories *within* one bet. If the original PRD truly is one cohesive bet but just has a lot of stories, consider whether the Solution is over-scoped — Shape Up's Appetite would force a cut.

### 10. The Rabbit-Hole-as-risk PRD (cycle preset)

**Pattern:** Rabbit Holes section reads like a generic risk list — "scaling might be hard," "users might not adopt," "edge cases could be complex."

**Why it fails:** Rabbit Holes are *specific traps with named avoidance decisions*. Generic risks belong in the Risks section. Rabbit Holes that read as risks signal the author didn't actually shape the work — they listed concerns instead of making cuts.

**Fix:** Each Rabbit Hole names: the specific scenario the team would otherwise sink time into, and the explicit decision to avoid it. "Don't build configurable rate-limiting — pick one rate and ship." "Skip the multi-tenant case — single-tenant only this cycle." If you can't name the avoidance decision, it's a risk, not a Rabbit Hole. Cross-reference `pitch-writing-shape-up`.

### 11. The stakeholder-list PRD

**Pattern:** Users & Personas section lists every conceivable stakeholder ("end users, admins, customer success, sales, partners, the integrations team") without distinguishing primary from secondary or what each needs from this PRD.

**Why it fails:** Universal scope = no scope. Engineering can't optimize for everyone simultaneously; success criteria can't be tied to "all stakeholders." The PRD will inevitably underserve the actual primary user.

**Fix:** Name the primary user segment (one — at most two), then list secondary stakeholders with the specific concern each has. "Primary: returning shoppers abandoning at cart. Secondary: ops (needs export visibility); sales (needs talking points for renewal calls)." If multiple primaries are truly equal, the PRD probably contains multiple bets — see anti-pattern #9.

### 12. The implementation-as-Solution PRD

**Pattern:** Solution section contains technical architecture, database schema, API contracts, or library choices.

**Why it fails:** Variant of #1 (prescriptive PRD), specifically at the Solution section. Solution should be fat-marker sketches (pitch) or user-shaped flows (ten-section) — not implementation. The boundary is: *what will the user experience* (PRD) vs *how will the system produce that experience* (engineering's `plan.md` companion to each child spec).

**Fix:** Replace technical descriptions with user-experience descriptions. If a tech detail is load-bearing for the bet (e.g., "must work offline" implies a specific architecture), state the user requirement, not the implementation. Engineering will design the architecture in the child spec's `plan.md` after the owner flip.

## How `pm-reviewer` uses this catalog

The `pm-reviewer` agent runs this catalog as a checklist on every PRD before `review → approved`:

- Anti-patterns 2, 5, 6, 8 are **blocking** — the spec cannot advance until they're resolved.
- Anti-patterns 1, 3, 4, 7, 10, 11, 12 are **strong findings** — they require either a fix or an explicit author response defending the choice.
- Anti-pattern 9 (over-long) is a **structural finding** — the reviewer recommends decomposition; the PM decides whether to accept the split.

Findings are written into the spec's `## Review` section with specific line/section references and severity, following the engineering `pr-review` shape.

## How `prd-author` self-checks

The `prd-author` agent runs a lighter version of the catalog *during* authoring — surfacing the smell before the artifact gets to review. The smells most worth catching during authoring (where the cost of fixing is lowest):

- **Anti-patterns 1 and 12** — prescriptive language slips in unconsciously. Surface a prompt: "this sentence describes implementation; is this a hard constraint or are you sketching?"
- **Anti-pattern 2** — propose at least one No-Go based on what the author *did* describe ("you mentioned X is in scope; would Y, Z be No-Gos?").
- **Anti-pattern 4** — when the author writes a metric, prompt for baseline and observation method.
- **Anti-pattern 8** — never let the author flip status `draft → review` with TBD content.

## Cross-references

- `prd-structure` — the positive counterpart. What good PRDs look like.
- `pitch-writing-shape-up` — Shape Up-specific guidance, especially for anti-patterns 2, 5, 10.
- `acceptance-criteria-ears` — the AC shape (relevant to anti-patterns 4, 7).
- `metrics-design` — the metric-quality bar (relevant to anti-pattern 4).
- `evidence-synthesis` and `opportunity-solution-trees-torres` — synthesis bar (relevant to anti-pattern 3).
- PM domain mission — principle #3 (tradeoffs visible) drives No-Gos / Out of Scope discipline; principle #5 (learn from what shipped) drives metric rigor.
