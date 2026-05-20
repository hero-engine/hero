---
type: convention
status: draft
scope:
  - internal/serve/pages/now/**
  - internal/serve/pages/work/**
  - internal/serve/shell/templates/**
tags: [dashboard, ui, activity-feed, empty-state, hero-serve]
relations:
  - target: hero-serve-dashboard-redesign
    kind: drawn-from
---

# Dashboard Pages Lead with Activity, Not Configuration

## Pattern

Every `hero serve` dashboard page leads with the *substance of recent
work*, not with a configuration prompt. The above-the-fold content on
any page must answer "what happened, and what do I look at next?" —
not "what did you forget to configure?"

## When to apply

When designing or modifying any top-level dashboard page handler or
template under `internal/serve/pages/<page>/`. Applies to Now, Work,
Knowledge, People, Agents, and any future top-level page.

## How

- The first content section on the page is an activity feed, in-flight
  strip, or equivalent live-signal surface — populated by default from
  graph events with a sensible rolling window (typically 7 days).
- Configuration prompts, install panels, and setup CTAs are
  **state-aware components**: they render only when the relevant
  capability is unconfigured, and they sit *below the fold* or
  in a secondary slot, never as the page lead.
- Empty states are acceptable for genuinely-empty projects only. A
  project with recent graph activity must never render an
  empty-headline page.
- Chat-input and command-bar widgets get **≤15% of vertical real
  estate** on a default-density screen. They are tools, not headlines.
- Tabbed metric strips show **rolling-window counts** (touched this
  week, shipped this week, etc.) that always have values without
  configuration. Sprint-shaped metrics are opt-in.

## Examples

- Now page activity feed in
  `internal/serve/pages/now/templates/activity.html` rendering graph
  events from the last 7 days as the first section above the fold.
- Work page "This week" tab as the default landing, with rolling-
  window tiles for touched/shipped/started/stale counts.

## Anti-patterns

- Headline reading "no agent running · 19h ago" or "no active sprint
  configured · configure sprint in hero.json" on a project with active
  work in the last 24 hours.
- Chat input plus install-prompt panel plus quick-command chips
  consuming the majority of the first viewport.
- A four-tile metric strip where three of four tiles read `—` or `0`
  on a heavy-activity day because the metrics are sprint-shaped and
  no sprint is configured.

## Exceptions

- Genuinely-empty projects (post-`hero scan` with no further activity)
  may show empty states as headlines. The fix is to seed real
  activity, not to hide the empty state.
- Pages that are fundamentally configuration surfaces (e.g. a future
  Settings page) are exempt — their job *is* configuration.
