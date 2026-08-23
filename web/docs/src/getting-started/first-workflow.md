# First Workflow

This walks through the core Hero loop: **design a feature, then deliver it**.

## 1. Design a Feature

In your AI tool, run:

```
/design add CSV export to the reports page
```

Hero's design workflow will:

1. Ask clarifying questions about scope and constraints
2. Produce a spec with acceptance criteria, technical approach, and edge cases
3. Save the spec to `.hero/planning/`

The spec is a markdown file with YAML frontmatter:

```yaml
---
title: "Add CSV Export to Reports Page"
type: feature
status: planning
priority: medium
---
```

Review the spec. The AI will surface tradeoffs and ask for your input — this is where you catch misunderstandings before any code is written.

!!! info
    The spec status moves through: `planning` → `in-review` → `delivering` → `completed`. You approve the spec before any implementation begins.

## 2. Deliver the Feature

Once you're satisfied with the spec, implement it:

```
/deliver add-csv-export
```

The delivery workflow reads the spec, follows your codified conventions, and implements the feature. It will:

- Write only the code required by the spec
- Run tests or validation if defined in the acceptance criteria
- Write a Completion Ledger against every acceptance criterion and Changes item
- Send the result through a fresh cold audit
- Run `hero spec verify <slug>` so the ledger, audit, test mapping, and build
  gates close and archive the spec only when the hard gates pass

!!! tip
    `/deliver` works from the spec, not from memory. If you want to change scope, update the spec first with `/design`, then deliver again.

## 3. Not Sure Which Command?

If you're not sure which workflow to use, just describe what you want in natural language:

```
/hero I need to fix the bug where totals round incorrectly
```

Hero routes your intent to the right command — in this case `/diagnose` — and passes along your context.

| What you say | Where it routes |
|---|---|
| "fix the login bug" | `/diagnose` |
| "add dark mode" | `/design` |
| "implement the auth spec" | `/deliver` |
| "review my PR" | `/review` |
| "break down the epic" | `/compose` |

## What's Next

You now know the core loop. From here:

- Use `/diagnose` to investigate bugs before fixing them
- Use `/convention` to codify team standards
- Use `/compose` to break epics into sequenced specs
- Run `hero status` in your terminal to see workspace state
