---
name: jtbd-job-stories
description: The Jobs-to-be-Done job-story shape — When [situation], I want [motivation], so [outcome] — favoring context and motivation over persona, and how it differs from an INVEST user story.
metadata:
  audience: story-writer, discovery-researcher, product-strategist
  purpose: framework-guidance
---

## What I do

Give agents the **job-story** format from the Jobs-to-be-Done school (Klement / Intercom lineage) and the judgment for when to reach for it instead of the classic "As a [role]…" user story. A job story frames work around the **situation and motivation** that trigger a need, not around a demographic persona. It's a discovery-and-framing lens; it is *not* the delivery-ready spec bar. Knowing which one you're writing — and why — is the whole point.

## When to use me

- framing a need where the *trigger* matters more than *who* has it (`discovery-researcher`)
- the team keeps writing "As a user, I want…" and the role carries no information (`story-writer`)
- synthesizing interviews into needs before solutions exist (`product-strategist`)
- pushing back when a "persona" is doing no work in a story and could be anyone

## The job-story shape

```
When [situation], I want [motivation], so [outcome].
```

- **Situation** — the concrete context that triggers the job. Time, place, emotional and system state. This is the load-bearing clause; it's where personas hide most of what actually predicts behavior.
- **Motivation** — what the person wants to *do* in that moment, stated without an implementation.
- **Outcome** — the result that tells them the job is done. Distinct from the motivation — it names the *why*, the state they're trying to reach.

**Good:**

> When a data-pull request lands in my shared inbox mid-incident, I want to hand the requester a self-serve export, so I can stay focused on the incident instead of becoming a ticket queue.

**Weak:**

> As an ops user, I want an export button, so that I can export data.

The weak one names a role that could be anyone, an implementation ("button") as the motivation, and an outcome that just restates the motivation ("export… so I can export").

## Why context/motivation beats persona

"As a [persona]" invites you to smuggle in a demographic and call it insight. Two people in the *same* persona behave completely differently in different situations; two people in *different* personas behave identically in the *same* situation. The situation predicts the behavior; the persona often doesn't.

The discipline: when you catch yourself writing a persona into the role slot, ask "what *situation* is this persona a proxy for?" and write the situation instead. Personas still have a place — see `personas-and-journey-maps` — but as a research artifact anchoring *whose* outcome matters, not as the trigger inside a story.

### Harvesting job stories from research

Job stories aren't invented at a whiteboard; they're *lifted* from what users actually said. The raw material is the "When… I found myself…" moments in interviews and tickets. A user who says "I got the request while I was already firefighting and just wanted to hand it off" is handing you a situation and a motivation almost verbatim. The synthesis job is to strip the incidental detail and keep the trigger + want + why. A journey map (`personas-and-journey-maps`) is a dense source: every pain in the grid sits in a situation, which is the raw "When" clause of a job story.

### Force-ranking situations

Not every situation is worth building for. Once you have a set of job stories, rank them by **frequency × intensity** — how often the situation occurs and how much it hurts when it does. A rare-but-agonizing situation and a constant-but-mild one both deserve attention; a rare-and-mild one usually doesn't. This ranking is what turns a wall of job stories into a prioritized opportunity space.

## The four forces behind a switch

JTBD explains *why* a person hires or fires a product using four forces. A job story lives inside this tension, and naming the forces sharpens the situation clause:

- **Push** of the current situation — the pain making them look for something new ("the manual export keeps failing mid-incident").
- **Pull** of the new solution — the appeal of the thing they're considering ("self-serve export, no ticket").
- **Anxiety** about the new solution — what makes them hesitate ("will the exported data be correct?").
- **Habit** of the present — the inertia of the current way ("filing a ticket is annoying but familiar").

A solution only wins when push + pull *outweigh* anxiety + habit. When you write a job story, the situation encodes the push, the motivation encodes the pull — and the anxiety/habit forces are exactly what your solution's onboarding and trust cues have to overcome. Skipping them is why "obviously better" features fail to get adopted.

## Job stories vs INVEST user stories — when to use which

They answer different questions and live at different stages. A job story frames the *need* during discovery; an INVEST user story is the *delivery-ready* atom engineering builds against. See `story-writing-invest` for the delivery bar.

| | Job story (JTBD) | INVEST user story |
|---|---|---|
| **Stage** | Discovery / framing | Delivery hand-off |
| **Frames** | The situation + motivation behind a need | A shippable slice with a done line |
| **Role slot** | Situation ("When…"), not a persona | Named segment when value needs it |
| **Success** | Outcome the person is trying to reach | Testable acceptance criteria |
| **Answers** | *Why* does this need exist? | *What* do we build, and when is it done? |
| **Bar** | Traceable to research/evidence | Independent, Negotiable, Valuable, Estimable, Small, Testable |

The flow is: a job story surfaces a need in discovery → the need becomes an opportunity in an OST (`opportunity-solution-trees-torres`) → a chosen solution is authored as one or more INVEST stories. Don't hand a raw job story to engineering as a spec; don't force an INVEST story during discovery before the situation is understood. They're a relay, not rivals.

## Anti-patterns

- **Persona-smuggling.** "When a millennial power-user opens the app…" — the demographic is back, wearing a "When" clause. Strip it to the situation that actually triggers the job.
- **Solution-in-the-situation.** "When I click the export button…" bakes the answer into the trigger. The situation is the context *before* any solution exists; if the tool is in the "When," you've prejudged the build.
- **Outcome that restates the motivation.** "…I want to export data, so that I can export data." The outcome must name a *different* thing — the state they're reaching for — or the clause is dead weight.
- **The everyone-situation.** "When a user wants to do something…" — a situation true of all users at all times triggers nothing. Make it specific enough to exclude most moments.
- **Job story as delivery spec.** Handing a job story to engineering with no acceptance criteria. It framed the need; now write the INVEST story.
- **Motivation as feature list.** Three "I want" clauses stapled together. One job, one motivation; split the rest into their own stories.

## Cross-references

- `story-writing-invest` — the delivery-ready bar a job story graduates into; INVEST is *what/when-done*, job stories are *why*.
- `opportunity-solution-trees-torres` — job stories surface needs that become opportunities under an outcome.
- `personas-and-journey-maps` — personas as research anchors; the journey grid is a rich source of situations for job stories.
- `discovery-interview-design` — interviews are where real situations (not invented ones) come from.
- Prior art: Alan Klement (*When Coffee and Kale Compete*), the Intercom "Jobs-to-be-Done" job-story format, Clayton Christensen's JTBD theory.
