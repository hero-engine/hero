---
name: objection-handling
description: Common objection patterns and proven responses tailored to Hero and AI-powered tooling. Loaded by deal-strategist.
metadata:
  audience: deal-strategist
  purpose: objection-playbook
---

## The Objection Framework

Objections are almost never what they appear to be. The stated objection
is the surface; the real concern is deeper.

### Step 1: Acknowledge

Never argue with or dismiss an objection. First, acknowledge it genuinely:

> "That's a really important question — I hear this from a lot of [similar
> companies]."

> "That's fair. We should address that directly."

Acknowledgment is not agreement. It is respect.

### Step 2: Clarify the real concern

Ask a question to surface the underlying concern before responding:

> "When you say [price is too high], is it a budget availability issue, a
> question of whether the ROI justifies the investment, or something else?"

> "What specifically worries you about [AI making decisions]?"

The clarifying question often reveals that the objection you prepared for
is not the actual objection.

### Step 3: Respond to the real concern

Address the root concern — not the surface words — with:
- A specific story from a similar customer
- A data point that reframes the concern
- A logical argument that directly addresses the fear
- A demonstration (if applicable)

### Step 4: Confirm resolution

Ask whether the response addressed the concern:

> "Does that address what you were worried about, or is there more to it?"

Never assume an objection is closed because you responded to it.

---

## Common Objection Categories

### Price / Budget Objections

**"It's too expensive."**

Real concern: Either genuine budget constraint, or the ROI isn't clear.

Response framework:
1. Clarify: "Is the issue budget availability, or are you not sure the
   investment pays off?"
2. If ROI: reframe around the cost of the problem — "What's the current
   cost of [the manual process / the inefficiency / the risk]?"
3. If budget: "Let's talk about phasing. What could we start with that
   delivers immediate value within your current budget?"
4. Proof: share a customer who had the same concern and the outcome they
   saw — in their terms (time saved, revenue recovered, risk eliminated).

**"We don't have budget this year."**

Real concern: This is often a soft no — they're not saying no to the value,
they're saying yes to avoiding the risk of saying no.

Response:
> "I understand. Budget cycles are real. If the value were clear, would
> there be a path to finding budget? Sometimes teams find it in existing
> contracts, discretionary spend, or Q4 year-end funds."

If they genuinely have no path: ask to be on the Q1 list and put a meeting
in the calendar now for early next quarter. Do not walk away — stay in the
deal.

---

### Timing Objections

**"Not the right time right now."**

Real concern: Something is competing for attention — competing initiative,
organizational change, upcoming event.

Response:
1. Get specific about the timeline: "When do you see things settling down?
   What would need to be true for this to move to the top of the list?"
2. Surface the cost of waiting: "What's the impact on your team for the
   next 6 months if you don't address this?"
3. If there's a true blocker: pause the deal and set a specific re-engage
   date. Don't let it linger in the pipeline as "nurture."

**"Let's revisit next quarter."**

Response:
> "Happy to do that. To make sure the next conversation is productive,
> would you be open to spending 30 minutes now to document what would
> make this a 'yes' next quarter? That way we both know exactly what
> to prepare."

This converts a deferral into a commitment.

---

### Competitive Objections

**"We're looking at [Competitor] and they're cheaper."**

Real concern: Price anchoring from a competitor or unclear differentiation.

Response:
1. Don't panic or discount immediately.
2. Understand the comparison: "What about their approach are you most
   interested in?"
3. Reframe total cost: "A lower price with higher implementation cost,
   more maintenance burden, or lower adoption rates often costs more in
   the end. Can we look at total cost of ownership?"
4. Isolate the real difference: "If price were equal, what would be most
   important to you?"
5. Pull in proof: reference customers who evaluated both and chose us.

**"We already use [incumbent tool]."**

Real concern: Switching cost, inertia, existing relationships.

Response:
> "How is that working for you today? What would you change about it if
> you could?"

Let them articulate the pain themselves. Don't attack the competitor —
let the buyer's own frustration do the work.

---

### AI and Product-Specific Objections

**"We're not sure we trust AI making decisions about our deals."**

Real concern: Autonomy, accuracy, or fear of losing human judgment.

Response:
> "Hero doesn't make decisions — it surfaces structured analysis so you
> make better decisions faster. Every output is a draft your team reviews
> and acts on. The judgment stays with you."

Proof: walk through a specific workflow — how the output is reviewed before
anything happens.

**"Our reps won't use another tool."**

Real concern: Adoption friction; skepticism about yet another system.

Response:
> "That's exactly why we designed Hero to live alongside the tools reps
> already use — it doesn't replace the CRM, it makes the time spent in
> the CRM more valuable. The reps who adopt it fastest are usually the
> ones who hate busywork most."

Offer: set up a pilot with 2–3 engaged reps. Let them become the internal
advocates. Tool adoption through peer influence is far more effective than
mandate.

**"We don't want AI to have access to our deal data."**

Real concern: Data security, competitive sensitivity, privacy.

Response:
> "Completely understandable. Let me walk you through exactly how the
> data flows and what security controls exist."

Be specific about: data residency, access controls, encryption, logging,
SOC2 compliance, model training data policies (does customer data train
the model? No — it stays in the customer's workspace).

Prepare: the security review checklist and data processing addendum
before the conversation.

---

### ROI and Value Objections

**"We don't know if this will work for us."**

Real concern: Risk aversion; lack of social proof from similar companies.

Response:
1. Reference a similar customer: "[Company in same industry] had the same
   concern. Here's what they saw in the first 90 days."
2. Offer a structured proof period: "What would you need to see in a
   30-day pilot to be confident? Let's define that success criteria now."
3. Reduce the initial risk: "What's the smallest scope we could start with
   that would let you prove value without a large commitment?"

**"We can build this ourselves."**

Real concern: Engineering team is capable and cost seems avoidable.

Response:
> "Your team absolutely could build something like this. The question is
> what that costs in engineering time, and what those engineers aren't
> building for your core product while they're maintaining a sales tool.
> What's an engineering week worth at your company?"

Avoid: directly comparing your product to what they'd build. Compare the
opportunity cost.

---

## Capturing Objection Intel

After each significant objection interaction:

1. Note what worked and what didn't in the deal spec
2. If a novel objection or novel response pattern appeared, capture it:
   ```
   hero note "<objection> response"
   ```
3. If the same objection pattern appears across multiple deals, it should
   graduate to a `.hero/knowledge/objections/<slug>.md` knowledge entry

The objection knowledge base compounds over time — every won deal's
handling approach becomes available to every future rep.
