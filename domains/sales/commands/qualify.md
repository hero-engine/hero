---
description: Score and qualify a deal using the configured framework (MEDDPICC by default). Writes structured findings into the deal spec.
---
Route this qualification request to the `qualification-analyst` agent.

**Before starting**, load the `deal-qualification` skill. It contains the
MEDDPICC scoring rubric, dimension definitions, red flag checklist, and the
output template the analyst must write to the deal spec.

**Resolve the deal spec** from the argument (slug or company name):
1. If a slug is provided, read `.hero/planning/deals/<slug>/spec.md` directly.
2. If a company name is provided, run `hero search "<company>"` and resolve
   to a deal spec. If none exists, create a new deal spec scaffold at
   `.hero/planning/deals/<slug>/spec.md` before proceeding.
3. Check `status` — if the deal is `won` or `lost`, report: "This deal is
   already closed. Run `/debrief <slug>` for post-close analysis."

**Load the qualification framework** from `.hero/hero.json` under
`qualification.framework`. Default to `meddpicc` if not set.

**Delegate to `qualification-analyst`** with:
- The full deal spec path
- The framework to use
- Any context provided by the user (discovery call notes, CRM data, etc.)

The agent will:
1. Read the existing deal spec to understand what's already known
2. Score each dimension of the framework (see skill for rubric)
3. Identify gaps — dimensions that are unknown or weak
4. Flag red flags that signal qualification failure
5. Write a structured `## Qualification` section into the deal spec
6. Update `meddpicc_score` and `probability` in frontmatter
7. Produce a next-action recommendation: "Next: close gap on Economic Buyer"

**After the analyst completes**, check that the deal spec on disk was
updated. If not, prompt the analyst to write findings to disk.

**If a CRM is configured**, sync the updated probability and stage back:
```
hero sync push <slug>
```

**Auto-capture** novel qualification patterns to `.hero/knowledge/` if
`knowledge.auto_capture` is enabled — e.g., "Deals in fintech without a
named Economic Buyer at Discovery close at 12% vs. 34% average."

**Always emit a brief summary** of the qualification score, key gaps,
and the single most important next action.

---

## Qualification Session Title

Set the session title to: `qualify: <company> (<score>/100)` after scoring.

---

## Batch Qualification

When asked to qualify multiple deals (e.g. "qualify all prospects"):

1. Run `hero search --type deal --status prospect` to get the list.
2. Present the list to the user for confirmation before proceeding.
3. Run qualification sequentially — complete each deal fully before the next.
4. Produce a summary table when all are done:

| Deal | Score | Biggest Gap | Recommendation |
|---|---|---|---|
| Acme Corp | 42/100 | Economic Buyer | Qualify out or find EB |
| BigCo | 67/100 | Metrics | Book metrics call |

---

Deal: $ARGUMENTS
