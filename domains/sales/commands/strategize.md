---
description: Produce a deal plan with approach, stakeholder map, objection handling, and win criteria. The "spec" of a sales deal.
---
Route this deal strategy request to the `deal-strategist` agent.

**Before starting**, load:
- `deal-strategy` skill — multi-threading, champion development, economic
  buyer access, close plan structure
- `objection-handling` skill — common objections and response patterns
- `deal-qualification` skill — so the strategy respects the MEDDPICC state

**Resolve the deal spec** from the argument:
1. If a slug is provided, read `.hero/planning/deals/<slug>/spec.md`.
2. If a company name is provided, run `hero search "<company>"` to find it.
3. If no deal spec exists yet, create one before proceeding:
   ```
   # Create scaffold at .hero/planning/deals/<slug>/spec.md
   ```
4. If `meddpicc_score` is absent or below 40, warn: "This deal hasn't been
   qualified yet. Run `/qualify <slug>` first for a stronger strategy."
   Proceed anyway if the user insists.

**Anchor check**: call `hero_anchor` with deal context to check for
tripwires (discount floors, approval thresholds, restricted competitors).
Surface any violations before proposing strategy.

**Search for applicable playbooks and battlecards**:
```
hero search --type playbook "<segment>"     # e.g. "enterprise land"
hero search --type battlecard "<competitor>" # if competitive deal
```
Include relevant playbook guidance in the strategy.

**Delegate to `deal-strategist`** with:
- The full deal spec path
- Applicable playbooks and battlecards found above
- Any context provided by the user

The agent will produce and write to the deal spec a `## Deal Strategy`
section containing:

1. **Situation Summary** — where we are, what we know, what's at risk
2. **Approach** — the primary motion (land-and-expand, competitive displacement,
   greenfield, etc.) with rationale
3. **Stakeholder Map** — each stakeholder: name, title, role (champion/EB/user/
   blocker), sentiment (advocate/neutral/detractor), influence level, and
   the specific action needed to move them
4. **Threading Plan** — which additional stakeholders to engage and how;
   who can introduce us to the Economic Buyer
5. **Objection Playbook** — anticipated objections with specific responses
   drawn from the `objection-handling` skill and past deal history
6. **Win Criteria** — what "done" looks like from the buyer's perspective;
   what must be true for them to sign
7. **Risk Assessment** — top 3 risks to this deal closing (with mitigation)
8. **Close Plan** — week-by-week milestones from now to the target close date
9. **Next Actions** — the 3 most important things to do this week, assigned
   by owner

**After the strategist completes**, verify the deal spec on disk contains
the `## Deal Strategy` section. If missing, prompt the agent to write it.

**Update spec frontmatter** if the strategy changes the probability estimate:
```yaml
probability: 45   # updated from strategy assessment
```

**Auto-capture** novel strategic patterns to `.hero/knowledge/` — e.g.,
"CFO-first motion in healthcare deals shortens procurement cycle by ~3 weeks."

---

## Strategy Review

To get a second opinion on the strategy before executing it:

After `/strategize` completes, suggest: "Run `/review <slug>` to have the
deal reviewed for gaps before you start executing."

---

## Session Title

Set the session title to: `strategize: <company> (<stage>)`

---

Deal: $ARGUMENTS
