---
description: ICP scoring, outreach strategy, and discovery angle for a new target. Turns a name or description into a ready-to-engage prospect brief.
---
Route this prospecting request to the `buyer-researcher` agent, with the
`deal-strategist` synthesizing the outreach strategy.

**Before starting**, load the `discovery-questioning` skill so the agent
can suggest strong discovery questions tailored to this prospect's profile.

**Parse the target** from the argument:
- A company name: `prospect "Acme Corp"`
- A description: `prospect "Series B fintech companies in payments"`
- A person: `prospect "Jane Smith CTO Acme Corp"`
- A market: `prospect "healthcare CIOs replacing homegrown systems"`

**Check for existing intel**:
```
hero search "<company or description>"
hero search --type prospect "<company>"
```
If a prospect or deal spec already exists, load it and extend — don't
start fresh.

**Load ICP definition** from `.hero/knowledge/` if it exists:
```
hero search --type knowledge "ICP"
hero search --type knowledge "ideal customer profile"
```
Use the ICP to score fit for this prospect.

**Delegate to `buyer-researcher`** to research the target. The agent will
produce:

### 1. ICP Fit Score

Score this prospect against the configured ICP across key dimensions:

| Dimension | Score (1–5) | Evidence |
|---|---|---|
| Company size fit | | |
| Industry fit | | |
| Tech stack alignment | | |
| Pain signal strength | | |
| Budget indicators | | |
| Timing / trigger | | |
| **Total fit** | **/30** | |

**Fit interpretation:**
- 24–30 — Strong ICP fit. Prioritize.
- 16–23 — Moderate fit. Research further before investing heavily.
- Below 16 — Weak fit. Low priority unless a strong trigger exists.

### 2. Company Profile

Key facts: size, industry, revenue, growth stage, tech stack, recent news.
(See full company research format in `research.md`.)

### 3. Buying Triggers Identified

Events that make this prospect ready to buy now:
- [ ] Recent funding / IPO
- [ ] New leadership (CTO, VP Eng, CPO hired in last 6 months)
- [ ] Competitive displacement (left a competitor they're unhappy with)
- [ ] Compliance or regulatory deadline
- [ ] Known pain in their public communications
- [ ] Hiring for roles that signal the problem we solve

### 4. Stakeholder Targets

Who to engage first, and why:

| Name | Title | Why target them | Entry angle |
|---|---|---|---|

### 5. Outreach Strategy

- **Channel** — email, LinkedIn, warm intro, event, partner referral
- **Hook** — the opening angle; what problem or trigger makes this timely
- **Sequence** — suggested 3-touch outreach sequence (email 1, follow-up,
  call) with key messages at each step
- **Personalization signals** — specific details from research to weave in

### 6. Discovery Questions

Top 5 questions to ask in the first call (drawn from `discovery-questioning`
skill), tailored to this prospect's profile and likely pain.

**Create a prospect spec** at `.hero/planning/deals/<slug>/spec.md`:

```yaml
---
title: <Company> — Prospect
type: deal
status: prospect
company: <company>
arr: <estimated ARR>
priority: <P0-P3 based on ICP score>
tags: [prospect, <industry>]
---
```

Include the research brief in the spec body under `## Research`.

**Surface to the user**:
1. ICP fit score and interpretation
2. Top 3 buying triggers found
3. Recommended first contact and opening angle
4. The 3 best discovery questions for this prospect

---

## Flags

- `--segment <description>` — prospect a market segment (returns a list of targets)
- `--icp-score-only` — just score fit, skip full research
- `--create-spec` — always create a prospect spec (default: create if ICP score ≥ 16)

---

## Session Title

Set the session title to: `prospect: <target>`

---

Target: $ARGUMENTS
