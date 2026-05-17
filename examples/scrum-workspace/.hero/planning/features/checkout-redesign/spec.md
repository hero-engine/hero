---
title: Checkout Redesign — Single-Page Flow With Saved Addresses
type: feature
kind: ux
status: planning
priority: P1
tags: [checkout, conversion, web, ux]
created: 2026-04-22
owner: pm
---

## Goal

Replace the four-step checkout funnel (cart → shipping → payment → review)
with a single-page checkout that reveals payment after a valid shipping
address is selected. Cuts steps in half for returning users with a saved
address and removes the page-load gap where mobile users currently drop.

## User Value

Returning customers with a saved address reach the pay button in a single
scroll. Guest users still see a linear sequence but never lose context
between steps. Recovers the 8 % drop measured between `/cart` and
`/checkout/shipping` on mobile in March.

## Acceptance Criteria

- Returning users with at least one saved address see the checkout page
  with shipping pre-selected and the payment section visible without a
  navigation event.
- Guest users see shipping fields first; payment becomes interactive
  once a valid shipping address is entered (validated client-side and
  server-side).
- The order review summary is fixed to the right column on viewports
  ≥ 1024 px and collapses to a bottom drawer below that.
- The funnel-step analytics event `checkout_step_viewed` continues to
  fire with `step ∈ {shipping, payment, review}` so the existing
  conversion dashboard keeps working.
- Page weight stays under 180 KB gzipped for the initial render.

## Boundaries

- Apple Pay / Google Pay express-checkout buttons are a separate spec
  (`express-checkout`) and out of scope here.
- The post-purchase confirmation page is unchanged.
- No changes to the cart page itself in this story.

## Risks

- Address validation latency on the server can stall the payment-reveal
  transition. Need to budget the validation call at <250 ms p95 or move
  to optimistic reveal with rollback.
- Saved-address surfacing must respect the active session's geo —
  surfacing an out-of-country address could confuse tax calculation
  downstream.

## Dependencies

- Saved-address service must expose a per-session "default for region"
  query; currently only returns most-recent.
