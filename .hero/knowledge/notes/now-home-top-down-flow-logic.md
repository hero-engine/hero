---
title: Now home top-down flow logic — section order, tabbed metrics, methodology-aware first tab
type: note
status: active
created: 2026-05-17
tags: [hero-serve, now-home, ui, information-architecture, decision-history]
relates-to:
  - hero-now-home
  - hero-surface-shell
  - hero-surface-architecture
---

## What this captures

The information-architecture reasoning behind the Now home's section
order, the tabbed metric strip pattern, and the methodology-aware
first tab. The first round of mocks ordered sections badly (chat
input buried at the bottom past three passive sections); the second
round fixed it by walking through the actual day-to-day flow.

## The day-to-day flow Now serves

A user opens hero serve in a tab. In sequence:

1. **Scan** — am I OK, what's red, what's running, what's waiting?
2. **Triage** — anything waiting on me?
3. **Act** — continue what I was doing OR launch something new
4. **Browse** — passive context, what's happened, deeper trends

Anything not in service of that sequence is decoration.

## The resulting section order

```
1. Page hero            — title + one-line status headline
2. Tabbed metric strip  — at-a-glance numbers, tabs for different lenses
3. Needs your input     — duty before discretion
4. Quick launch         — the chat doorway, above the fold
5. On your plate        — resume the active spec
6. Your agents          — live + today's runs
7. Since you were here  — passive feed
8. Footer
```

Two non-obvious calls in this order:

### Why metrics come second, not later

The previous mock had a "By the numbers" stat strip at the bottom of
the page — dressing, basically. The user pushed: "some stats and
metrics belong closer to the top." Right: an at-a-glance pulse is
part of the "scan" step. It answers "am I OK" before duty or
discretion. Moving the metric strip to position #2 (and deleting
the bottom strip entirely so there's only one) makes the page work.

### Why inbox comes before chat

Both belong above the fold (the chat doorway can't be buried). But
inbox first because **duty before discretion**: things waiting on
you outrank open-ended "what do you want to do." Once the inbox is
empty (or scanned), the eye lands naturally on the big chat input.

## The tabbed metric strip pattern

The metric strip uses **text-link tabs** (same visual idiom as the
top nav — hero-blue underline on active) that swap the row of 4
tiles beneath them. Vanilla JS toggling DOM-hidden panes; no SPA.

This pattern showed up because of a real ambiguity: which metrics
matter most to an individual? Sprint progress? Personal throughput?
Hero's value (autonomy, hours saved)?

We argued: for the **Now** home (personal), ROI metrics are too
abstract day-to-day. Sprint progress and personal throughput are
way more immediate ("where am I in the sprint? what did I ship this
week?"). But ROI is still nice to glance at.

Three tabs solved it:

| Tab | Default? | What it answers |
|---|---|---|
| **This sprint** | yes | Where are we in the current iteration? Am I on track? |
| **My week** | | What have I personally shipped lately? |
| **Hero ROI** | | What value is Hero delivering? |

The user can click between them to find the lens they want. Default
to "This sprint" because it's the most universally "where am I right
now" reading.

This pattern got reused: Work home has `This sprint / Throughput /
Quality` tabs. Agents home has `Right now / Today / Health (7d)`
tabs. People-ROI has `Money / Throughput / Quality` tabs. The
tabbed-metric-strip became a shared shell fragment, callable from
any home page.

## Methodology-aware first tab

Different teams use different methodologies. A solo Shape Up team
doesn't have "sprints" — they have "cycles." A kanban team has
neither. The `This sprint` tab label hard-codes one assumption.

Solution: detect methodology from the workspace (`hero.json` setting
or project type heuristic) and swap the first tab's label + tile
content accordingly:

| Methodology | First tab label | What it shows |
|---|---|---|
| Scrum / sprint | `This sprint` | Sprint progress, days remaining, at-risk, your slice |
| Shape Up | `This cycle` | Cycle progress, weeks remaining, hill-chart status |
| Kanban | `This week` | Throughput last 7 days, WIP, age of oldest in-flight |
| Solo | `This week` | Specs shipped, commits, hours active |

Same physical position. Different semantics. The page works for any
team without forcing them to mentally translate.

## What to remember

- Section order is a product of day-to-day flow, not visual balance.
- Stats go near the top because they're part of "scan," not because
  they look nice in a strip.
- Tabbed metric strips beat picking one set of metrics — let the
  user pick the lens.
- Don't hard-code methodology vocabulary. Detect it and adapt.
- Above-the-fold reachability matters: at a typical laptop viewport
  (~780px usable), the page hero + metric strip + first 1-2 inbox
  rows + start of chat input should all be visible before scrolling.
