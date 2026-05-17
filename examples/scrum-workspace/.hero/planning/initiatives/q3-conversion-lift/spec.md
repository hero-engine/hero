---
title: Q3 Conversion Lift — Push Mobile Checkout Conversion +3 Points
type: initiative
status: planning
priority: P1
tags: [conversion, mobile, checkout, q3]
created: 2026-04-01
owner: pm
horizon: q3-2026
relations:
  - target: cart-abandonment-emails
    kind: child
  - target: express-checkout
    kind: child
---

## Goal

Lift mobile checkout completion rate from 38.2 % (Q1 2026 baseline) to
41.0 % or higher over Q3, measured as orders / `checkout_started`
events on mobile devices. Hold or improve desktop conversion as a
non-regression guard.

## Outcome

A measurable +2.8 point absolute lift on mobile mobile, sustained over
the final four weeks of the quarter, with no greater than a -0.5 point
movement on desktop.

## Bets

- **Express checkout** — wallet buttons remove the funnel for the
  highest-intent traffic segment (already shipped in March; tracking
  the lift over the full quarter).
- **Abandonment recovery** — two-touch lifecycle emails recapture the
  warm-then-cold abandonment cohort.
- **Single-page checkout** — collapse the four-step funnel; included
  in this initiative if the express + abandonment bets land short of
  target by mid-quarter.

## Risks

- The abandonment-email touch could cannibalize organic returns from
  users who would have come back anyway. Need a holdback group from
  day one.
- iOS Safari WebKit rollouts repeatedly break Apple Pay button
  eligibility. Owner: payments team monitors Safari beta channels.
- Marketing's seasonal promo cadence in Q3 (back-to-school, Labor Day)
  injects funnel noise that can mask or fake a lift. Lock the
  measurement window to non-promo weeks for the headline reporting.

## Notes

Q1 baseline numbers pulled from the analytics warehouse on 2026-03-25.
A/B holdback infrastructure for the abandonment-email cohort needs
provisioning before that bet can ship — owner: data eng.
