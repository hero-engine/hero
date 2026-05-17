---
description: Define success metrics for a PRD or initiative — current → target, leading-not-lagging, with named baselines.
---
Route this metrics request to the `metrics-analyst` agent (P1). In v1, falls through to `pm-delivery-lead` loading the `metrics-design` skill directly.

## Required argument

A PRD or initiative slug. Without a slug, ask which artifact to define metrics for — don't infer from session context. Metrics anchor success criteria; they belong to a specific bet.

## What lands

The analyst writes (or refines) a `## Metrics` section on the spec with each metric in the shape:

```
- **<metric name>** — <one-line description>
  - Current: <baseline value, dated, source>
  - Target: <target value, deadline>
  - Type: leading | lagging
  - Source: <where the number comes from — analytics event, query, survey>
```

Rules the agent enforces:

- **Leading where possible.** Lagging metrics (revenue, retention, NPS) are valid but should be paired with at least one leading indicator the team can move week-over-week.
- **Baselines are named, not vibes.** "Current: 12% conversion (Q1 2026, mixpanel funnel `signup → activated`)" — not "Current: low."
- **Targets have deadlines.** A target without a date isn't a target.
- **No vanity metrics.** If a metric can only go up (page views, signups without qualification), the analyst flags it and asks for a quality dimension.

## Retrospective hook

After the metrics land, the analyst sets the spec's `shipped_at` retrospective hook. This wires Principle #5 (learn from what shipped) — when the child specs flip to `completed` (engineering close-out via `owner_history`), `/retro` knows which metrics to evaluate against.

## Output

- Updated spec with the `## Metrics` section.
- A one-line log naming the metrics added.
- The retrospective hook fields populated in frontmatter (`metrics_baseline_at`, `metrics_target_due_at`).

Request: $ARGUMENTS
