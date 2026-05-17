---
title: Express Checkout — Apple Pay and Google Pay on Cart and Product Pages
type: feature
kind: new
status: completed
priority: P2
tags: [checkout, payments, conversion, mobile]
created: 2026-02-14
owner: engineering
relations:
  - target: q3-conversion-lift
    kind: parent
---

## Goal

Surface Apple Pay and Google Pay buttons on the cart page and on
product detail pages for eligible devices, so mobile users can complete
purchase without entering shipping or payment details manually.

## User Value

Shaves the entire checkout funnel down to a biometric prompt for the
~38 % of mobile traffic on iOS Safari and Android Chrome with a
provisioned wallet. Reduced friction is most acute on PDP impulse
purchases under $50.

## Acceptance Criteria

- Apple Pay button renders on cart and PDP for Safari on iOS/macOS
  when the user has a provisioned card and the merchant identifier is
  configured.
- Google Pay button renders on cart and PDP for Chrome and Edge when
  the Payment Request API reports a usable instrument.
- Tapping either button collects shipping + billing from the wallet,
  posts directly to the order service, and lands on the existing
  order-confirmation page.
- Server-side validation rejects any wallet token that fails the
  payment processor's verification, and surfaces a clear retry UI.
- Conversion analytics fire `checkout_started` and `order_placed`
  events with `method ∈ {apple_pay, google_pay}` so the funnel
  dashboard reports express vs. standard separately.

## Notes

Shipped 2026-03-30. The mobile-conversion dashboard showed a 2.1-point
lift on iOS in the first two weeks; Android instrumentation is still
settling. Retrospective notes captured in the team's shared notes —
the headline lesson was that wallet eligibility detection is async and
the buttons need a skeleton placeholder to avoid layout shift.
