---
description: Win/loss analysis on a closed deal. Captures learnings into the knowledge base so every future deal benefits.
---
Route this debrief to the `deal-strategist` agent, which coordinates the
win/loss analysis and knowledge capture.

**Resolve the deal spec** from the argument:
1. Read the deal spec at `.hero/planning/deals/<slug>/spec.md`.
2. Check `status` — if not `won` or `lost`, confirm with the user: "This deal
   is still open (status: <status>). Close it in the CRM first, then run
   debrief."
3. Accept `--won` or `--lost` flags to set the outcome if the spec hasn't
   been updated yet.

**Detect the outcome** from `status` field or `--won`/`--lost` flag.

**Conduct the debrief**. The agent will read the full deal spec history and
produce a structured analysis written to the spec under `## Debrief`:

### Win Debrief Format (--won)

1. **Why we won** — the decisive factors (price, relationship, product, timing,
   champion strength, competitive positioning)
2. **What moved the deal forward** — key moments, meetings, content, or
   interactions that accelerated progress
3. **Champion effectiveness** — how the champion showed up; what enabled them
4. **Objections we overcame** — which objections came up and what responses worked
5. **Competitive dynamics** — if competitive, what we did that worked; any
   traps they walked into
6. **Repeatable patterns** — what we should do on every similar deal
7. **What nearly lost it** — moments of risk; what we'd do differently

### Loss Debrief Format (--lost)

1. **Why we lost** — the real reason (be honest; not "price" if it was
   something else)
2. **When we knew we were losing** — early signals we missed or ignored
3. **What we could have done differently** — specific moments where a
   different action might have changed the outcome
4. **Qualification failure?** — should we have disqualified earlier? At what
   stage did this become unwinnable?
5. **Competitive dynamics** — how the competitor beat us; what they said about
   us; what we can learn
6. **What the buyer told us** — verbatim or close to it; what they cited
7. **Playbook gaps** — what's missing from our playbooks that would have helped

**Extract learnings** and write them to the knowledge base:

1. **Objection responses** — any objection handling that worked or failed goes
   to `.hero/knowledge/objections/`
2. **Playbook updates** — if a pattern emerged that should change how we run
   this motion, update the relevant playbook
3. **Battlecard updates** — if competitive intel was learned, update the
   relevant battlecard
4. **Persona insights** — if a new buyer behavior or persona pattern emerged,
   capture it

**Update the deal spec** frontmatter:
```yaml
status: won    # or lost
close_date: 2026-07-15   # actual close date
```

**Archive the deal spec** by running:
```
hero spec complete <slug>
```

**Emit a debrief summary** — a 4-sentence summary suitable for a team Slack
update or CRM close note:
- What happened
- Why we won/lost
- Key learning
- What we're changing

---

## Flags

- `--won` — mark as won and run win debrief
- `--lost` — mark as lost and run loss debrief
- `--quick` — abbreviated debrief (5 questions, no knowledge capture)

---

## Session Title

Set the session title to: `debrief: <company> (won/lost)`

---

Deal: $ARGUMENTS
