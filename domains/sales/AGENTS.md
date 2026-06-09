# Hero Sales — AI-Powered Revenue Workflow

This is the **Hero Sales** domain pack. It gives revenue teams the same
structured, AI-powered workflow Hero gives engineering teams: design before
you pitch, diagnose before you lose. Every deal starts as a spec, gets
strategized, then gets executed with full context.

The core manifesto (`.hero/mission.md`) applies unchanged. The shape of
the work changes; the system's job doesn't.

---

## Session Start

At the start of every session:

1. **Set a concise session title** reflecting the deal or task at hand (e.g.
   "strategize: Acme Corp enterprise", "qualify: new fintech lead", "forecast:
   Q3 pipeline"). This keeps the session list navigable for the whole team.

2. **Load deal context.** If a deal slug or company name was mentioned, find
   the spec immediately:
   ```
   hero search "<company name>"          # find the deal spec
   hero read-spec <slug>                  # load frontmatter and current state
   ```
   Do not start work before understanding where the deal currently stands:
   stage, MEDDPICC score, last activity, next action, close date, ARR.

3. **Check pipeline state.** Run `hero status` for the quick pipeline view —
   stage counts, at-risk deals, forecast summary. This orients you before
   any single-deal work.

4. **Orient to next-best-action.** Read `.hero/NEXT.md` for queued actions
   from the previous session. Check `hero queue` for deals waiting on action.
   A rep should never start a session wondering "what should I work on?" —
   Hero answers that.

5. **Load relevant playbooks.** Before strategizing or qualifying, search
   the knowledge base for applicable patterns:
   ```
   hero search --type playbook "<segment or motion>"
   hero search --type battlecard "<competitor>"
   ```

6. **Anchor check for large strategic moves.** Before proposing a deal
   strategy that involves significant pricing, competitive positioning, or
   executive engagement, call `hero_anchor` to verify no active tripwires
   apply (e.g., "do not discount below X", "executive engagement requires
   approval above $Y ARR").

---

## Natural Language Routing

When the user describes what they want, route to the appropriate command.
**Run the command — don't just suggest it.**

| User intent | Command |
|---|---|
| Qualify, score, assess fit, MEDDPICC, BANT | `/qualify` |
| Strategize, deal plan, approach, stakeholder map | `/strategize` |
| Forecast, pipeline numbers, commit, upside | `/forecast` |
| Pipeline overview, deal board, stage view | `/pipeline` |
| Research, company intel, buyer background | `/research` |
| Won/lost debrief, post-close analysis, lessons | `/debrief` |
| Prospect, ICP fit, outreach angle, new target | `/prospect` |
| Search deals, find spec, look up | `hero search` |
| Capture note, log activity, save thought | `/note` |
| Check workspace health | `/check` |

When routing, pass the user's full context as arguments. If intent is
ambiguous between `/qualify` and `/strategize`, ask: "Do you want to score
fit (qualify) or build a deal plan (strategize)?"

**Deal-slug resolution.** When the user names a company or deal without a
slug, search before asking:
```
hero search "<company name>"
```
Pass the resolved slug to the command. Only ask the user if search returns
no match.

---

## Commands Reference

| Command | What it does |
|---|---|
| [`/qualify`](commands/qualify.md) | Run MEDDPICC (or configured framework) — score deal and write findings to spec |
| [`/strategize`](commands/strategize.md) | Produce a full deal plan: approach, stakeholder map, objections, win criteria |
| [`/forecast`](commands/forecast.md) | Weighted pipeline forecast grouped by stage, rep, and time period |
| [`/pipeline`](commands/pipeline.md) | Kanban overview of all open deals by stage with ARR totals |
| [`/research`](commands/research.md) | Competitive intel, buyer background, company context |
| [`/debrief`](commands/debrief.md) | Win/loss analysis — capture learnings, update knowledge base |
| [`/prospect`](commands/prospect.md) | ICP scoring, outreach strategy, discovery angle for new targets |

---

## Agents Reference

| Agent | Role |
|---|---|
| [`deal-strategist`](agents/deal-strategist.md) | Coordinates the full deal plan — the delivery lead equivalent |
| [`qualification-analyst`](agents/qualification-analyst.md) | Runs structured qualification, produces scored deal brief |
| [`forecast-analyst`](agents/forecast-analyst.md) | Maintains pipeline accuracy, flags slippage, produces forecast |
| [`competitive-intel`](agents/competitive-intel.md) | Tracks competitive landscape, battlecards, win probability |
| [`buyer-researcher`](agents/buyer-researcher.md) | Researches prospects, identifies buying triggers, maps org |

---

## Skills Reference

| Skill | What it covers |
|---|---|
| [`deal-qualification`](skills/deal-qualification/SKILL.md) | MEDDPICC framework, scoring rubrics, red flags |
| [`deal-strategy`](skills/deal-strategy/SKILL.md) | Multi-threaded approach, champion development, economic buyer access |
| [`objection-handling`](skills/objection-handling/SKILL.md) | Common objection patterns and proven responses |
| [`pipeline-management`](skills/pipeline-management/SKILL.md) | Stage definitions, exit criteria, deal hygiene rules |
| [`forecast-methodology`](skills/forecast-methodology/SKILL.md) | Weighted pipeline, coverage ratio, commit vs. upside |
| [`competitive-positioning`](skills/competitive-positioning/SKILL.md) | Battlecard patterns, win/loss analysis |
| [`discovery-questioning`](skills/discovery-questioning/SKILL.md) | SPIN questioning — Situation, Problem, Implication, Need-payoff |

---

## Deal Spec Structure

All deals are tracked as specs following the schema in
[`spec-types/deal.yaml`](spec-types/deal.yaml). Key fields:

```yaml
---
title: Acme Corp — Enterprise Platform Deal
slug: acme-corp-enterprise
type: deal
status: qualifying          # prospect → qualifying → demo → proposal → negotiation → won → lost
company: Acme Corp
owner: jane.smith@company.com
arr: 120000
close_date: 2026-09-30
stage: Qualifying
meddpicc_score: 42
probability: 25
---
```

Deal specs live at `.hero/planning/deals/<slug>/spec.md`.

---

## Key CLI Commands (Sales)

```bash
# Pipeline state
hero status                           # pipeline by stage with ARR totals
hero pulse --week                     # weekly pipeline narrative
hero forecast                         # weighted forecast summary

# Work a deal
hero search "Acme Corp"              # find the deal spec
/qualify acme-corp-enterprise        # score with MEDDPICC
/strategize acme-corp-enterprise     # produce deal plan
/research "Acme Corp"                # buyer and company intel

# Knowledge management
hero search --type playbook          # find applicable playbooks
hero search --type battlecard        # find competitive positioning
/debrief acme-corp-enterprise --won  # capture win learnings

# Pipeline hygiene
hero queue                           # deals needing attention
hero search --match "stale"          # find stale deals
```

---

## Auto-Capture and Knowledge Flywheel

After every significant deal interaction, Hero captures what was learned:

- **Discovery calls** — key pain points, stakeholders mentioned, objections raised
- **Qualification sessions** — MEDDPICC findings, gaps identified
- **Won/lost debriefs** — patterns that correlate with outcomes
- **Objection responses** — what worked, what didn't

These land in `.hero/knowledge/` and improve every future deal. The more
you use Hero, the smarter it gets about your specific market, buyers, and
competitive landscape.

To review what's been captured:
```bash
hero search --type knowledge          # all knowledge entries
hero search --type playbook           # playbooks built from patterns
hero search --type battlecard         # competitive intel
```

---

## Surviving Context Compaction

Sales sessions can run long — multiple deal reviews in a single context
window. If the context compacts mid-session:

1. **Your deal state is in the spec files** — run `hero search "<company>"` to
   reload the deal spec. The spec has everything: stage, MEDDPICC score,
   stakeholder map, objections, next actions.
2. **Pipeline state is always current** — `hero status` reconstructs the
   pipeline view from spec frontmatter at any time.
3. **NEXT.md carries session intent** — read `.hero/NEXT.md` to see what you
   were working toward before compaction hit.
4. **Session handoff pattern** — before ending any session, run `hero next`
   to write a crisp handoff that the next session (possibly a different rep)
   can pick up cold.

---

## Domain Configuration

Hero Sales reads sales-specific config from `.hero/hero.json`:

```json
{
  "domain": "sales",
  "crm": {
    "type": "salesforce",
    "instance": "your-org.my.salesforce.com",
    "token_env": "SALESFORCE_TOKEN",
    "auto_sync": true
  },
  "qualification": {
    "framework": "meddpicc",
    "min_score": 60
  },
  "forecast": {
    "methodology": "weighted",
    "stages": {
      "Prospecting": 10,
      "Discovery": 20,
      "Evaluation": 40,
      "Proposal": 60,
      "Negotiation": 80,
      "Closed Won": 100,
      "Closed Lost": 0
    }
  },
  "pipeline": {
    "stale_days": 14,
    "hygiene_schedule": "weekly"
  }
}
```
