---
title: Cart Abandonment Emails — Two-Touch Recovery Sequence
type: feature
kind: new
status: delivering
priority: P1
tags: [email, lifecycle, conversion, marketing]
created: 2026-04-08
owner: engineering
relations:
  - target: q3-conversion-lift
    kind: parent
---

## Goal

Send a recovery email one hour after a logged-in user abandons a cart
worth more than $25, and a second email 23 hours later if the cart is
still untouched. Mirrors the lifecycle pattern marketing has been
running manually via CSV exports since February.

## User Value

Catches the "got distracted" abandonment case while intent is still
warm, and gives a second window for the "thinking about it" case.
Early experiments from the manual CSV runs showed a 4 % open-to-purchase
rate on the one-hour touch.

## Acceptance Criteria

- A logged-in user who adds an item worth >= $25 and leaves the site
  without checking out receives email touch 1 at T+1h.
- If the cart still has the same items 23 hours after touch 1, the user
  receives email touch 2.
- Either email is suppressed if the user completes checkout, empties
  the cart, or has opted out of marketing email.
- Both emails render the current cart contents inline, with a single
  "Resume checkout" CTA that deep-links to `/cart` with the cart state
  hydrated from the saved session.

## Tasks

- [ ] T-1 — Wire the abandonment detector to the existing session
  inactivity event stream and emit a `cart_abandoned` event with cart
  snapshot. (todo)

## Risks

- The session inactivity stream currently fires for tab-close as well
  as true abandonment; we need to filter on `last_activity_at` >= 60 min
  to avoid emailing users who closed a tab and reopened in a new window.
- Cart contents at email-send time may differ from cart contents at
  detection time. Snapshot at detect, render snapshot in email, but
  resolve to live cart on click.

## Dependencies

- Marketing email transport (Postmark) — existing, no changes needed.
- Saved-session hydration on `/cart` — already used by the
  `checkout-redesign` work; assumed stable.
