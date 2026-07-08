---
name: pitch-writing-shape-up
description: Shape Up pitch authoring discipline — appetite as budget, fat-marker sketches, rabbit holes as named traps, no-gos as scope defense, cooldown as first-class state.
metadata:
  audience: prd-author, pm-reviewer, pm-delivery-lead
  purpose: pitch-writing
---

## What I do

Provide the authoring discipline for Shape Up pitches — the pitch-shaped variant of a PRD that's the default under cycle preset. This skill packages the rules from Ryan Singer's *Shape Up* (Basecamp, 2019) into the patterns that survive the move from "raw idea" → "shaped pitch" → "bet" → "build."

## When to use me

Load this skill when:

- authoring a pitch (P1; v1: `prd-author` in pitch shape)
- reviewing a pitch before a betting table (`pm-reviewer`, `pm-delivery-lead`)
- coaching a team adopting Shape Up for the first time
- distinguishing a pitch from a generic PRD when a team asks "is this a pitch?"

## Why Shape Up

Shape Up is Basecamp's product development method, documented in *Shape Up: Stop Running in Circles and Ship Work That Matters* (Ryan Singer, 2019). Its mechanics differ from Scrum and Kanban in load-bearing ways: fixed time, variable scope; no backlog; 6-week cycles + 2-week cooldown; betting tables instead of grooming; hill charts instead of burndowns.

The pitch is the artifact that survives the betting table — the thing the team commits to ship within Appetite or kill cleanly. Pitch quality is the single biggest input to whether a cycle ships its bets without burning the team out.

## The five required sections

A Shape Up pitch has five sections. None are optional.

```
1. Problem
2. Appetite
3. Solution
4. Rabbit Holes
5. No-Gos
```

Hero PM adds two optional sections (Linked stories, Risks) for graph-integration reasons — but the five above are the canonical Shape Up shape.

## Appetite — the canonical rule

**Appetite is the budget, not the estimate.**

This is *the* Shape Up rule. Memorize it. Everything else about pitches follows from it.

- **Estimate** says: "this work will take 6 weeks." (You're predicting cost.)
- **Appetite** says: "this problem is worth 6 weeks." (You're naming what you're willing to spend.)

The difference is direction of control. Estimates flow from work → cost ("how long will it take?"). Appetite flows from cost → work ("we're willing to spend 6 weeks; what can we build?"). Appetite makes scope the variable; estimates make time the variable.

### Two appetite values

Shape Up uses two:

- **Small batch** — 1-2 weeks. For tightly-scoped bets where the value is clear and the risk is low.
- **Big batch** — 6 weeks. For meatier work where the team needs the full cycle to ship.

Anything that doesn't fit in 6 weeks is either too big to shape (push back to discovery / break into multiple bets) or evidence the bet should be re-scoped.

### Writing the Appetite section

One paragraph. Name the value and the rationale.

```
## Appetite
Small batch — 1-2 weeks.

Customer interviews suggest the export workflow is the friction
point; CSV is the universally-asked format. We don't yet know
whether users want column customization, scheduling, or just
a one-click dump. Ship the one-click dump; learn before we
commit a Big batch to the configurability question.
```

What fails: missing Appetite; Appetite stated as estimate ("4 weeks"); Appetite that contradicts the Solution complexity ("Small batch" with a Solution requiring six new screens).

## Fat-marker sketches over wireframes

Solution sections use **fat-marker sketches** — drawings done with a thick marker so detail is impossible. The point is to communicate shape, not pixels.

### Why fat-marker

- Detail invites premature commitment. A polished wireframe locks in choices the team should make during build.
- Fat-marker sketches force the author to omit what they don't yet know.
- They're cheap to redraw — the team iterates fast in shaping.

### What replaces wireframes in a pitch

- **Fat-marker sketches** — hand-drawn (or hand-drawn-looking digital), low-fidelity, deliberately rough.
- **Breadboards** — text-based diagrams of UI elements and their connections, no layout. "Login [button] → Dashboard [page]. Dashboard has 'Export' [affordance] → opens Export [modal] → 'Download' [button]."
- **Named flows** — prose-described user paths through the system.

### What does NOT belong in Solution

- Pixel-perfect mockups (those are mockup-brief artifacts, not pitch content)
- Tech architecture (engineering decides during build)
- API contracts (engineering)
- Database schema (engineering)
- Implementation library / framework choices (engineering)

## Breadboards

A breadboard names the system's parts and connections without committing to layout. Useful when the UI is unfamiliar or when interactions are non-trivial.

**Example breadboard for an export pitch:**

```
Dashboard [page]
  Export [button] -> opens Export Modal

Export Modal [overlay]
  Date Range [field]
  Format [dropdown: CSV | JSON]
  Email Delivery [checkbox]
  Download [button] -> triggers job, closes modal
  
Job Status [notification area]
  Pending [state] -> shows progress
  Complete [state] -> shows download link
  Failed [state] -> shows error + retry
```

The breadboard captures *what's there* and *what connects to what*. It does not capture *where they sit on the screen*. That's deliberate.

## Rabbit Holes — named traps with the specific scenario

**Rabbit Holes are specific traps with explicit avoidance decisions.** Not generic risks. Not reassurance.

### What passes

```
## Rabbit Holes

- **Don't build configurable rate limits.** Customer requests mention
  "configurable" but the actual rate they care about is one we can
  hard-code. Picking one rate ships in days; configurable is a week.

- **Skip the multi-tenant case.** All current customers are single-
  tenant; multi-tenant introduces an auth complication that doesn't
  earn its weight this cycle. Defer.

- **Don't redesign the existing settings screen.** The export setting
  sits naturally next to the existing notifications setting; reuse
  the layout. Redesigning the screen is a separate bet.
```

Each Rabbit Hole has:

- The specific scenario.
- The avoidance decision.
- The rationale (one sentence).

### What fails

- "Performance might be tricky." (Generic. Not a trap with an avoidance decision.)
- "We'll keep an eye on edge cases." (Reassurance. Not a decision.)
- "Risk: users may not adopt." (That's a risk, belongs in Risks. Not a Rabbit Hole.)

### How to find Rabbit Holes

When you draft the Solution, ask: *what's the part of this that would inflate scope if I let it?* That's a candidate Rabbit Hole. Common categories:

- **Configurability** — the team's instinct is to make things configurable; usually one hard-coded choice is fine.
- **Edge cases** — there's always one edge case that takes 30% of the time; name it and decide whether it's in scope.
- **Adjacent improvements** — building X invites cleaning up Y nearby; resist unless Y is in scope.
- **Generality** — building for one customer is fast; building for "all customers" is slow. Name the customer.
- **New patterns** — adding a UI pattern (modal? drawer? side panel?) the team hasn't used before. Reuse if possible.

## No-Gos — scope defense

**No-Gos are work explicitly excluded from this appetite.** They are the difference between a bet that ships in Appetite and one that creeps.

### What passes

```
## No-Gos

- No mobile app changes this cycle. Web-only.
- No new admin UI; admin uses the existing CLI for export config.
- Not handling the API access path — UI export only.
- No internationalization of the export format; English column
  headers only.
- No bulk export across multiple accounts.
```

### What fails

- Empty No-Gos. (The single most common failure mode. Without explicit exclusions, the team assumes scope and inevitably creeps.)
- No-Gos that read like Rabbit Holes. (Rabbit Holes are *cuts inside the appetite*; No-Gos are *whole capabilities excluded from it*.)
- No-Gos that contradict the Solution. (If the Solution mentions mobile, the No-Gos can't exclude mobile.)
- Generic No-Gos. ("No scope creep" isn't a No-Go; it's a wish.)

### Finding No-Gos

Same instinct as Rabbit Holes but at a coarser scale. Ask: *what would a stakeholder reasonably assume is in scope, that isn't?* Common categories:

- **Adjacent platforms** — web-only? API-only? Mobile?
- **Adjacent users** — end-user only? Admin? Partner?
- **Adjacent data** — single record? Bulk?
- **Adjacent flows** — happy path only? Recovery? Migration?
- **Adjacent quality** — i18n? Accessibility above baseline? Performance beyond stated targets?

## Cooldown as a first-class state

After every 6-week cycle, Shape Up reserves 2 weeks of cooldown. Cooldown is not optional and not for catching up on cycle work.

### What cooldown is for

- Bug fixes, especially the long-tail issues that don't justify a pitch on their own.
- Engineering-led refactoring and infrastructure work.
- Exploration — engineers and designers chase ideas they've been sitting on.
- The shaping work that produces next cycle's pitches.

### What cooldown is NOT for

- Finishing cycle work that didn't ship. (If a bet doesn't ship in Appetite, it's killed — not rolled into cooldown. The team explicitly decides whether to re-pitch next cycle.)
- "Catching up" — the cycle's done; everything that didn't ship is in the past.
- Starting new bets. (New bets go through the next betting table.)

Pitches must respect Cooldown. A pitch that requires "we'll polish it in cooldown" violates Appetite — the work isn't actually fitting.

## The betting table

The betting table is a 90-minute meeting at the end of each cycle that decides what to bet on next.

### Mechanics

- **Fixed inputs** — the pitches that were shaped during this cycle (and any that survived from previous cycles). No new ideas surfaced at the table.
- **Stakeholders** — small group (Basecamp uses founders + heads of product/design/engineering). The team that has to ship the work has a say.
- **Decisions** — for each pitch: bet, kill, or shape further. A bet means commitment to next cycle.
- **No backlog** — pitches not bet on don't go on a list; they either re-surface next cycle or they don't. The author has to re-champion.

### Implication for pitch authors

Your pitch competes with others at the table. The bar isn't "is it shaped" — it's "is it shaped *and* worth this cycle's slot?" Common reasons a shaped pitch loses:

- The bet isn't grounded — Problem and Evidence are weak relative to other pitches.
- The Appetite is too big for the value (Big batch on a Small batch bet).
- The Rabbit Holes signal too much risk left.
- A more time-sensitive pitch (competitive, contractual, regulatory) takes the slot.

Re-pitching after a loss is normal. Treat the table's feedback as the input to the next draft.

## The pitch lifecycle

```
raw idea → shaping → shaped pitch → bet → build → ship-or-kill
                                              ↓
                                          cooldown
                                              ↓
                                   next cycle's bets
```

- **Shaping** happens *outside* the cycle, often by the PM + a senior designer/engineer. Not a public process; pitches aren't shared until they're ready for the betting table.
- **Bet** is the betting table's commitment. From this point, the team can hill-chart and the pitch is real.
- **Build** uses the full cycle. Teams use hill charts to surface "are we still uphill on unknowns" vs "are we downhill on execution."
- **Ship-or-kill** is the cycle's end. If the team can't see downhill clearly, the bet is killed and the team moves to cooldown.

## Anti-patterns to refuse

When the reviewer (P1; v1: `prd-author` in pitch shape) or `pm-reviewer` is asked to advance a pitch, refuse if:

- Appetite section is empty, says "TBD," or names a non-standard duration.
- Solution contains implementation details (architecture, schema, API contracts).
- Solution contains pixel-perfect mockups (pitch is fat-marker; that's mockup-brief work).
- Rabbit Holes section is empty or contains only generic risks.
- No-Gos section is empty.
- Pitch describes a six-week-plus scope. (Doesn't fit Big batch; isn't a pitch.)
- Pitch describes work that will "spill into cooldown." (Violates Cooldown.)
- Pitch is being authored without a clear champion who'll defend it at the betting table.

## Cross-references

- `prd-structure` — pitch shape is the default PRD template under cycle preset; this skill goes deeper.
- `prd-anti-patterns` — Shape-Up-specific failures (#2 empty No-Gos, #5 missing Appetite, #10 Rabbit-Hole-as-risk) are covered there.
- `shape-up-cadence` (P1, ships v1.5) — the 6+2 rhythm, betting table mechanics, cycle scheduling.
- `hill-chart-reasoning` (P1, ships v1.5) — hill charts as unknowns-remaining visualization (not progress bars).
- `cycle-planning` — capacity and commit logistics under cycle preset.
- PM domain mission — principle #3 (tradeoffs visible) is the philosophical root of Rabbit Holes and No-Gos.
- Prior art: Ryan Singer, *Shape Up* (Basecamp, 2019) — read it if you haven't.
