---
description: Kanban overview of all open deals by stage with ARR totals, health signals, and next actions due.
---
Load the `pipeline-management` skill for stage definitions, exit criteria,
and hygiene rules before rendering.

**Gather all open deals**:
```
hero search --type deal --status "prospect,qualifying,demo,proposal,negotiation"
```

**Produce a pipeline board** organized as a kanban view by stage. For each
stage, show:

- **Stage header** with total deal count and total ARR
- **Each deal card** showing:
  - Company name and deal title
  - ARR
  - Days in current stage
  - MEDDPICC score (color-coded: red <40, yellow 40–59, green 60+)
  - Next action due (from deal spec)
  - Owner
  - Staleness flag if no activity in 14+ days

**Format**:

```
## Prospecting  (3 deals · $340K ARR)
─────────────────────────────────────
Acme Corp               $120K  ·  Day 8   ·  Score: 24  ·  [At Risk]
  Next: Initial discovery call — jane.smith  ·  Due: tomorrow

Startup Co              $80K   ·  Day 3   ·  Score: —   ·
  Next: Send intro email — john.doe  ·  Due: today

...

## Qualifying   (5 deals · $820K ARR)
─────────────────────────────────────
...
```

**Pipeline summary** at the top:

| Metric | Value |
|---|---|
| Total open pipeline | $X.XM |
| Weighted forecast | $X.XM |
| At-risk deals | N |
| Stale deals (14+ days) | N |
| Deals closing this week | N |
| Deals closing this month | N |

**Hygiene alerts** below the board — any deal that violates exit criteria
for its stage (see `pipeline-management` skill):

- Missing Economic Buyer at Proposal stage
- No next action set
- Close date in the past
- Single contact only (not multi-threaded)

**After displaying** the pipeline, surface the top 3 deals that need
attention today: highest ARR deals with an overdue next action or a
slippage signal.

---

## Flags

- `--rep <name>` — filter to one rep's pipeline
- `--stage <stage>` — show only that stage
- `--at-risk` — show only deals with risk signals
- `--stale` — show only deals with no activity in 14+ days
- `--week` — deals with close dates in the next 7 days

---

## Session Title

Set the session title to: `pipeline: overview`

---

$ARGUMENTS
