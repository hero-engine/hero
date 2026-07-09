---
name: continuous-discovery-cadence
description: Teresa Torres-style weekly discovery rhythm — three touchpoints, opportunity revisits, and assumption tests as habit, not project.
compatibility: opencode
metadata:
  audience: discovery-researcher, pm-investigator, product-strategist
  purpose: process-guidance
---
## What I do

Provide the weekly rhythm that keeps continuous discovery alive — what cadence to commit to, what artifacts survive across weeks, how to operationalize when the team only has an hour, and how findings flow from interviews into the OST and onward to authored specs. Source: Torres, *Continuous Discovery Habits*.

## When to use me

- Setting up a discovery practice from zero.
- Triaging "we should do some research before we build" into actual recurring habits.
- Pushing back on episodic discovery ("we'll do a study before Q3") in favor of weekly habits.
- Establishing how a `pm-investigator` or `discovery-researcher` operates session-to-session.
- Connecting intake (inbound signals) into the discovery loop without letting intake drown the loop.

## The core cadence

Torres' weekly minimum:

- **3 customer touchpoints per week** — interviews, usability sessions, customer calls where the PM is listening for opportunity signal (not running a demo).
- **1 opportunity revisited per week** — the active branch of the OST gets new evidence or gets re-ranked.
- **1 assumption tested per week** — a small, fast test (prototype, concierge, fake door, paper sketch) resolves an assumption.

The number 3 matters less than the recurrence. A team that does 3 touchpoints every week for a quarter learns more than a team that does 30 in a single month and then nothing for six. The rhythm is what produces insight; bursts produce reports.

## What "continuous" actually means

Episodic discovery has a start date and an end date — "we'll do a research sprint in May, then we'll be ready to build." Continuous discovery has neither. It's a recurring slot on the calendar, defended like a standup or a sprint review.

The distinction matters because the bets you'll make six months from now are shaped by the evidence you gather today. Episodic discovery answers a current question; continuous discovery builds the evidence base that lets you answer questions you haven't asked yet.

## The minimum viable cadence (1 hour/week)

Most PMs don't have unlimited time. The 1-hour-per-week version:

- **20 minutes:** one customer touchpoint (a single recorded customer call, a single usability session, a 20-minute interview).
- **20 minutes:** synthesize — what did this tell us, which OST opportunity does it touch, what new evidence to add.
- **20 minutes:** revisit the OST — what changed, what's the next assumption to test, what's the next touchpoint to schedule.

This is the floor. A team running the 1-hour version produces less than the Torres recommended cadence but produces *more than zero*, which is what most teams running episodic discovery actually deliver.

When the PM gets more time, expand the touchpoints first, then the synthesis depth, then the assumption tests. Don't add elaborate output artifacts (research decks, formal reports) until the cadence is sustainable — the cadence is the deliverable.

## Building the recruitment loop

The single biggest failure mode in continuous discovery is "I couldn't get any customers this week." Without recruitment infrastructure, the cadence dies in week three.

The recruitment loop:

1. **Always-on signup.** A page or in-app surface where willing customers can volunteer for research. Low-friction (email + 1 question).
2. **Standing scheduler.** Calendar link, fixed weekly slots (e.g. Tuesday/Thursday 2-4pm). Volunteers self-book.
3. **Auto-thank-you and rotation.** Volunteers get acknowledgement, a small thank-you (gift card / credit), and aren't asked again for 90 days.
4. **Sales/CS as a feed.** Sales calls and support escalations are recruitment opportunities. A weekly handoff from sales/CS surfaces customers worth talking to.
5. **A backlog of customers to talk to, not a question to ask.** The PM should never be in the position of "I need to talk to someone this week and don't have anyone." The pipeline is full enough that the question is "which conversation is most useful this week," not "is there anyone."

The recruitment loop is platform work. It pays for itself in the second month and compounds from there. Spend a week building it up front.

## What happens in a touchpoint

Touchpoints are **not demos**, **not user testing of a specific design**, and **not surveys**. They are conversations to surface opportunities — user needs, pain points, current behaviors, workarounds.

The default structure (Torres' interview shape):

1. **Recent specific experience.** "Tell me about the last time you [tried to do X]." Not "do you ever," not "would you," not hypothetical. A specific past event.
2. **Walk it through.** Step-by-step what happened, what they were thinking, what they did before/after, where they got stuck.
3. **Surfacing pain.** Where did the friction live? What did they wish was different?
4. **No leading.** Don't pitch a solution. Don't validate a hypothesis. The point is to surface what they say unprompted.

A good touchpoint produces 1–3 opportunity statements that get added to the OST (in the user's voice, with attribution). A touchpoint that doesn't produce any new opportunity signal is still useful — it confirms the existing tree.

## Synthesis — turning conversations into opportunities

Raw interview notes are not opportunities. The synthesis step:

1. **Capture verbatim quotes.** When the customer said something striking, record it word-for-word with attribution.
2. **Cluster across touchpoints.** When the same pain shows up in 3 of 8 recent conversations, it's an opportunity. When it shows up once, it's a signal — track it but don't act yet.
3. **Phrase opportunities in the user's voice.** "I can't tell which conversations I've already responded to" — not "users want better conversation tracking."
4. **Trace evidence.** Each opportunity carries a count and attribution (`6/12 interviews Oct 2026`).

Synthesis happens *every week*, not at the end of a research project. A backlog of unsynthesized interviews is a sign the cadence is breaking down.

## The OST is the artifact that survives

Individual interview notes age out. Slide decks get archived and never re-opened. The Opportunity Solution Tree is what persists — the same tree refreshes weekly with new evidence, gets re-pruned, gets re-ranked.

Each week's discovery output flows into the existing tree:

- **New opportunities** appear or get reinforced with new evidence.
- **Solutions** get killed when assumption tests fail, get promoted when tests pass.
- **The active branch** may shift if new evidence outranks the current target.

The tree is **the** discovery artifact. The interview notes feed it; the OST is what other agents and other sessions read.

See `opportunity-solution-trees-torres` for the tree's structure.

## How findings flow into authored specs

Continuous discovery is upstream of authoring, not parallel to it. The flow:

```
Intake / customer signals → discovery touchpoints → OST opportunities
        ↓
    OST solutions (with assumption tests)
        ↓
    Validated solution → Evidence section of initiative
        ↓
    PRD authored → stories decomposed → handoff
```

When `prd-author` or `product-strategist` starts a spec, they pull from the OST's `Evidence` for the relevant outcome. The PRD inherits the opportunity framing, the test results, and the customer attributions. The PM didn't invent the bet — discovery surfaced it.

This is why discovery-without-authoring is wasted (insight that never reaches a spec dies) and authoring-without-discovery is dangerous (specs grounded in nothing).

## Relationship to intake

Intake (the inbound feedback stream) and discovery (outbound research) are different. Intake is reactive — it captures what comes in. Discovery is proactive — it goes looking for what isn't being said.

The two are connected:

- Intake **feeds** discovery: clusters of similar intake become opportunities worth investigating.
- Discovery **shapes** the OST: the OST then orders which intake clusters get attention.
- Neither replaces the other. An intake-only team responds to whoever shouts loudest. A discovery-only team misses what existing customers are telling them.

The `intake-triager` and `intake-classification` skills handle the intake side. This skill handles the discovery side. The two meet at the OST.

## Cadence failure modes and how to recover

- **No touchpoints this week.** Check the recruitment loop. If it's empty, that's the problem — fix recruitment before resuming.
- **Touchpoints but no synthesis.** Notes pile up unread. Add 20 minutes/week of synthesis as a separate calendar block; don't try to do it ad hoc.
- **Synthesis but no OST update.** Insights captured but never integrated. Make the OST refresh part of synthesis itself, same session.
- **OST updates but no assumption tests.** Tree grows; nothing gets validated. The team is doing exploration without conclusion. Force one assumption test per week, however small.
- **Assumption tests but nothing reaches specs.** Discovery is happening; authoring isn't pulling from it. Have `prd-author` and `product-strategist` cite OST evidence in every Bet they write.

The cadence rarely dies all at once. It erodes through one of these failure modes. Watch for the first sign and patch immediately.

## Anti-patterns

- **Episodic discovery.** "We'll do a research sprint in May." Builds nothing durable. Insight dies between bursts.
- **Discovery without synthesis.** Touchpoint notes pile up unread. The conversations didn't happen, effectively.
- **Tests that block forward motion.** Assumption tests that take a cycle to run aren't tests — they're projects. Shrink the test.
- **Recruitment-when-you-need-it.** Scrambling for customers when a touchpoint is due. Build the standing pipeline instead.
- **Interviews as demos.** "Let me show you what we're building and get your feedback." This is not discovery — it's solution validation, and it's heavily biased.
- **Hypothetical questions.** "Would you use a feature that did X?" Customers are bad at predicting their own behavior. Ask about specific past experiences instead.
- **One-and-done synthesis.** A research deck written in March that nobody reads in April. The OST is the living artifact; reports are not.
- **Discovery without authoring.** Insight that never shapes a spec. The loop must close.

## Cross-references

- `opportunity-solution-trees-torres` — the artifact discovery feeds and refines weekly.
- `discovery-interview-design` (P1, ships v1.5) — how to run the individual touchpoint.
- `assumption-testing` (P1, ships v1.5) — how to design tests that resolve in days.
- `evidence-synthesis` — clustering raw interview notes into opportunity statements.
- `intake-classification` — the inbound side that feeds discovery (and gets shaped by it).
- `metrics-design` — outcomes at the top of the OST follow these rules.
- `initiative` and `prd` spec types — the `Evidence` sections where discovery output lands.
- PM principle #1 (decide what's worth building) — continuous discovery is the operating mechanism.
