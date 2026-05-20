---
title: Dashboard Drops Sprint as Default — Rolling Windows + Themes Are the Lead
type: decision
status: proposed
created: 2026-05-19
tags: [dashboard, serve, work-page, sprint, rolling-windows, decision]
relations:
  - target: hero-serve-dashboard-redesign
    kind: decided-in
---

# Dashboard Drops Sprint as Default — Rolling Windows + Themes Are the Lead

## Decision

The `hero serve` Work page no longer defaults to a "This sprint" tab.
Default is **"This week"** — a rolling Mon–Sun activity window that
populates automatically with zero configuration. Sprint UI renders
only when `sprint:` is explicitly configured in `hero.json`, and even
then sits alongside "This week" rather than replacing it.

Now page follows the same pattern: rolling activity-feed window
(default 7 days, switchable to today / week / month / all). Themes —
auto-detected work clusters from the rolling window via extended
`knowledge-flywheel` pattern detection — render in both Now and Work
when ≥3 related items reach the confidence threshold.

## Why

Sprint orthodoxy assumes a team, planned capacity windows, a backlog
you pull from, and ceremony. Hero's primary user is a solo
continuous-flow operator without a tracker. Forcing them through
sprint UI to read their own activity inverts the value: they spend
their first 30 seconds on the dashboard reading "no sprint configured"
instead of seeing what just happened.

Rolling windows + themes flip the default. The solo operator gets
real signal immediately. The team operator who opts into sprints gets
their planned-cadence view back via configuration. Everyone wins; the
solo operator wins by default.

## Consequences

- Existing sprint UI code paths stay — gated on `hasSprintConfig`.
- Both Now and Work pages need a rolling-window data layer
  (activity feed, in-flight strip, themes).
- `knowledge-flywheel` cluster detection extends to work clusters
  (file-path prefix + decision tag) with a conservative threshold
  (≥3 related items).
- Migration risk for current sprint users is low — their tab
  reappears the moment `sprint:` is configured.
