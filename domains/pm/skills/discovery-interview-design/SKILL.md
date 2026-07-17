---
name: discovery-interview-design
description: Design customer interviews that generate opportunity-space signal instead of polite agreement — non-leading questions about specific past experience, a 5–10/week cadence, and structured synthesis that survives the "how do you know?" challenge.
metadata:
  audience: discovery-researcher, and the deferred pm-delivery-lead loader
  purpose: framework-guidance
---

## What I do

Supply the interview-design discipline behind continuous discovery. Most product interviews generate garbage signal — leading questions produce agreement, hypothetical questions produce speculation, and a single interview before building produces false confidence. Teresa Torres' method inverts all three: ask about *specific past experience*, never lead, run a *steady cadence* rather than a pre-launch burst, and synthesize structurally so the findings trace back to what people actually said. When `discovery-researcher` designs an interview guide, this is the method.

## When to use me

- designing an interview guide before a discovery round
- `/discover --interview` produces an interview plan
- an assumption test's method is "5-user interview" and you need the guide
- coaching a team whose interviews keep producing "yes people love it" with no actionable signal
- setting up a weekly discovery cadence

## Story-based, not opinion-based

The foundational move: **ask what people did, not what they would do.** Past behavior is evidence; speculation about future behavior is noise dressed as data.

| Opinion-based (noise) | Story-based (signal) |
|---|---|
| "Would you use a CSV export?" | "Tell me about the last time you needed to get data out of the product. Walk me through it." |
| "Do you find onboarding confusing?" | "Think back to your first week. Where did you get stuck?" |
| "Would you pay for this?" | "When you evaluated tools last, what made you pick the one you did?" |
| "Is speed important to you?" | "Tell me about a time the product was too slow to be useful. What did you do next?" |

Why past-experience questions work: they anchor on a real event the person actually lived, which surfaces the messy details (workarounds, emotions, who else was involved) that speculation smooths over. "Would you use X?" invites the person to be helpful and agreeable — the answer is almost always a polite yes that predicts nothing.

## Never lead

A leading question hands the interviewee the answer you want and rewards them for agreeing:

- ❌ "Don't you think it'd be helpful if export were faster?"
- ❌ "Wouldn't you love a one-click option?"
- ❌ "So the current flow is frustrating, right?"

Each of these will get a yes that means nothing. Replace with a neutral, open probe rooted in their experience: *"What happened the last time you exported? What was that like?"* Let the friction come from them, unprompted — friction they volunteer is signal; friction you suggested is an echo.

Watch for subtler leads too: praising the current product before asking about it, describing your solution and then asking if they'd want it, or nodding hard at the answers you like (bias by body language). Stay curious and flat.

## Cadence — a habit, not a project

Torres' cadence discipline: a discovery-active team interviews **5–10 users per week, continuously.** Not "we'll run interviews before Q3." The point is a steady trickle of contact with reality so findings arrive in time to act on, and so the team never argues from stale or imagined evidence.

- **5 users is the unit for a given segment** — after ~5, you hit diminishing returns on the *same* segment. Spend the 6th on a *different* segment (a churned user, a never-converted prospect) rather than a 6th happy customer.
- **Weekly slots, pre-booked** — recruiting is the bottleneck; standing weekly slots keep the pipeline full so discovery never starves into a burst.
- **Interview → synthesize → refresh the tree, every week** — the cadence includes synthesis and opportunity-solution-tree updates, not just the calls.

## Structured synthesis

An interview you don't synthesize is a conversation you had. Synthesis is *extraction*, not transcription. After each round, pull out:

- **Confirmed assumptions** — with the specific quote/moment that confirmed them.
- **Disconfirmed assumptions** — with the evidence. This is the high-value finding; a round that disconfirms a bet just saved a build.
- **New opportunities** — unmet needs or friction the person named that weren't in your tree. These often reshape the roadmap.
- **New questions** — synthesis usually reveals the next layer of uncertainty to test.

Keep **traceability to verbatim**: every synthesized theme links back to the actual words behind it, so a reader can judge whether your synthesis is fair (this is the compare-don't-replace discipline from `pm-agent-doctrine`). And surface **outliers**, not just the tidy pattern — the one user who wanted the opposite is often where the real insight hides.

## Sample composition

Who you talk to determines what you learn. The default failure is recruiting only current, happy users — they confirm what you already believe. Deliberately mix:

- current users who hit the friction you're studying
- users who **churned** (they'll tell you what broke)
- prospects who **never converted** (they'll tell you why not)
- users of a **competitor** (they'll tell you the alternative's shape)

## Anti-patterns

- **"Would you use it?" questions.** Generate polite agreement; predict nothing. Ask about a specific past experience instead.
- **Leading questions.** "Wouldn't it be great if…" rewards agreement. Stay neutral; let friction come unprompted.
- **Single-interview-then-build.** One conversation is an anecdote, not evidence. Run the cadence.
- **Happy-path sampling.** Interviewing only users who already love the product confirms the wrong things.
- **Synthesis as transcription.** Pasting notes into the spec isn't synthesis — extract confirmed/disconfirmed assumptions and new opportunities.
- **Episodic discovery.** A pre-launch interview burst arrives too late to change the plan. Weekly or not at all.
- **Burying the outlier.** Reporting only the confirming pattern sells a conclusion; the dissenter is often the finding.

## Cross-references

- `continuous-discovery-cadence` — the weekly rhythm this cadence plugs into.
- `assumption-testing` — interviews are one test method; pre-register the pass/fail before running the guide.
- `opportunity-solution-trees-torres` — interviews feed the opportunity space; synthesis refreshes the tree.
- `evidence-synthesis` — weighting and attribution mechanics for turning interview output into a defensible evidence trail.
- `pm-agent-doctrine` — never fabricate a quote; compare-don't-replace when synthesizing; ground themes in verbatim.
- Prior art: Teresa Torres, *Continuous Discovery Habits*; producttalk.org interview method.
