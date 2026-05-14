---
title: Launch Readiness — Telemetry, Deploy, and Public-Use Polish
type: initiative
status: planning
priority: P1
tags: [launch, telemetry, deploy, observability, public]
created: 2026-04-27
relations:
  - target: hero-cloud
    kind: child
  - target: pre-launch-hardening
    kind: sibling
  - target: hero-marketing
    kind: related
horizon: someday
---

## Goal

Bridge the gap between "the product is good enough that we'd use it
ourselves" and "the product is good enough that we'd hand it to a
team we don't know." Tactically: stand up production infrastructure,
turn on opt-in telemetry, and absorb whatever lessons fall out of
dogfooding the polished core before we deploy.

## Why this is its own initiative (not pre-launch-hardening)

Pre-launch-hardening was scoped around making federation safe to ship:
fix the conflict noise, lock down tenant isolation, unify search.
Those landed. Telemetry and production deploy were originally part of
that sprint but got moved here on 2026-04-27 — the call was to keep
polishing the core product first rather than rush a deploy of
something that still has rough edges.

We pick this up after a focused round of product polish (separate
initiative or ad-hoc spec collection) when the core experience is
something we'd be proud to show off without caveats.

## Contents

| # | Spec | Notes |
|---|---|---|
| 1 | [`hero-telemetry`](../../features/hero-telemetry/spec.md) | Opt-in usage analytics + feedback channel. Spec already written. Trust-sensitive — strict consent UX, minimal payload, public schema. |
| 2 | Production deploy of `hero-cloud` to `cloud.heroengine.ai` (or chosen subdomain) | Domain `heroengine.ai` resolved 2026-05-14 (see `.hero/knowledge/decisions/domain-name.md`); `teamhero.cloud` parked. Needs: TLS, secrets management (not env vars in plaintext), CockroachDB managed instance (Cockroach Cloud or self-hosted on Fly/Railway), CI deploy pipeline, healthchecks, basic monitoring. |
| 3 | Onboarding documentation polish | Walkthrough that takes a fresh user from `brew install hero` to syncing with their team in under 5 minutes. |

## Pre-conditions for picking this up

- The core product polish work (separate scope) lands and we've
  dogfooded it for at least a few days without major frustration
- Pre-launch-hardening sprint complete (✅ done 2026-04-27)
- A real first-team candidate identified — someone we can hand the
  binary to without it being a science experiment

## Out of initial scope

- Marketing site, landing page, blog content (in
  [`hero-marketing`](../hero-marketing/spec.md))
- Killer features that haven't been specced as files yet (in
  [`hero-killer-features`](../hero-killer-features/spec.md))
- Self-serve billing / pricing pages (separate when we have pricing)
