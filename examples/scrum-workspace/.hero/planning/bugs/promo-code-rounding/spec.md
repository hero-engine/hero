---
title: Promo Code Rounding — Percentage Discount Off By One Cent On Even Subtotals
type: bug
kind: edge-case
status: planning
priority: P2
severity: minor
tags: [pricing, promo, rounding, cart]
created: 2026-05-04
owner: engineering
---

## Symptoms

Carts with an even-cent subtotal (e.g. $50.00, $128.00) and a 10 %
promo code applied display a discount that is one cent lower than the
expected `subtotal * 0.10`. The order total is correct on the order
summary but inconsistent with the line-item discount displayed on the
cart page.

Customer-visible example:
- Subtotal: $50.00
- Promo code `SAVE10` (10 % off)
- Displayed discount: $4.99 (expected $5.00)
- Cart total: $45.01 (expected $45.00)

## Reproduction

1. Add any item priced at $50.00 (e.g. SKU `MUG-CLASSIC-12OZ`) to an
   empty cart.
2. Apply promo code `SAVE10` at the cart page.
3. Observe the discount line reads `-$4.99` and the cart total reads
   `$45.01`.

Reproducible on production and staging. Reproducible across all
browsers — server-rendered total is wrong, not a frontend display
issue.

## Diagnosis

The discount calculation in `pricing.applyPercentDiscount` floors the
result of `subtotal_cents * percent / 100` before converting back to
dollars, instead of rounding to nearest. For inputs where the
multiplication is exact (any subtotal whose cent count is divisible by
the percent denominator), the floor is a no-op — but for inputs that
land exactly on an integer (e.g. `5000 * 10 / 100 = 500`), the
intermediate floating-point representation is `499.9999…` and floors
to `499`.

## Root Cause

Use of `math.floor(subtotal * percent)` on floating-point inputs.
Should be integer-cents arithmetic end-to-end or
`math.round(subtotal_cents * percent / 100)`.

## Acceptance Criteria

- A cart with subtotal $50.00 and a 10 % promo applies a $5.00
  discount, producing a $45.00 total.
- A cart with subtotal $128.00 and a 25 % promo applies a $32.00
  discount.
- Regression test added that exercises the failing inputs at the
  pricing-service unit-test layer.
- Spot-check on the most-used promo codes (top 10 by 30-day redemption)
  shows no other rounding mismatches.

## Risks

- Changing the rounding direction could shift one-cent values the
  other way on currently-correct carts. Need to dry-run the new
  calculator over a 7-day sample of historical orders and confirm
  net-zero or net-customer-positive impact before merging.

## Notes

Customer reports filed against three different SKUs since April 28 —
all on subtotals ending in `.00`. Support has been issuing one-cent
goodwill credits as a workaround.
