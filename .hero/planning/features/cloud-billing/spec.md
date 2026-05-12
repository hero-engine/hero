---
title: Cloud Billing and Subscription Management
type: feature
status: planning
tags: [cloud, billing, stripe]
created: 2026-04-12
parent: hero-cloud
depends-on: [cloud-auth, cloud-api]
horizon: next
smoke: deferred
---

## Goal

Integrate Stripe for subscription billing. Org owners can subscribe to Team
or Enterprise tiers, manage seats, and view invoices. The billing system
gates cloud features by tier.

## Design

### Tiers and Gating

| Feature | Free (CLI) | Team ($15-30/seat/mo) | Enterprise ($50-100/seat/mo) |
|---|---|---|---|
| CLI + local MCP | yes | yes | yes |
| Cloud sync | - | yes | yes |
| Cloud dashboard | - | yes | yes |
| Cloud MCP | - | yes | yes |
| Cross-repo search | - | yes | yes |
| SSO/SAML | - | - | yes |
| Audit log | - | - | yes |
| SLA | - | - | yes |
| Priority support | - | - | yes |

### Stripe Integration

- Products: hero-team, hero-enterprise
- Pricing: per-seat, monthly billing
- Webhook handling for subscription lifecycle (created, updated, cancelled, payment failed)
- Customer portal for self-service (update payment, cancel, view invoices)
- Proration on seat changes

### Seat Counting

A "seat" = an org member with admin or member role. Viewers don't consume seats.
Seat count is synced to Stripe on member add/remove.

### Trial

14-day free trial of Team tier, no credit card required.
Trial includes full cloud features. Downgrade to CLI-only on expiration.

## Changes

- Cloud service: `billing/` package — Stripe integration
- Cloud service: `billing/webhook.go` — Stripe webhook handler
- Cloud service: `middleware/tier.go` — feature gating by subscription tier
- Cloud dashboard: billing settings page

## Acceptance Criteria

- Org owners can subscribe to Team or Enterprise tier
- Seat count syncs with Stripe on member changes
- Feature gating enforced: non-subscribers get 403 on cloud endpoints
- Stripe webhooks handle subscription lifecycle correctly
- 14-day trial available without credit card
- Customer portal accessible for self-service billing management
- Downgrade to free tier preserves org and data (just gates cloud features)
