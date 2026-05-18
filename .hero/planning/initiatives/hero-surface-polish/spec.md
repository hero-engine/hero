---
title: Hero Surface Polish — Ongoing Quality Pass on the Web Companion
slug: hero-surface-polish
type: initiative
status: planning
tags: [serve, surface, polish, ongoing]
created: 2026-05-18
relations:
  - target: hero-surface-architecture
    kind: parent
horizon: now
---

## Vision

The [Hero Surface Architecture](../hero-surface-architecture/spec.md)
initiative shipped the full set of five top-level homes plus the
shell, chat dispatcher, and ⌘K overlay. The plumbing is in place; the
surface is *navigable*. It is not yet *delightful*.

This initiative is the durable shelf for the polish work that lives
between "structurally complete" and "feels like a finished product." It
will accumulate child specs over time as we triage what users actually
hit when they open `hero serve` and as we close gaps the original specs
deferred.

## What lives here

- Bug fixes against the surface that don't warrant their own home's
  spec history (cross-cutting issues, broken sub-routes, data fetcher
  misses, layout glitches).
- Default-state alignment — what `/home`, `/home/sub-tab` actually
  render and how much breathing room they get.
- Empty-state quality — what shows up when there's no data, no
  adapter, no sprint configured.
- Data-fetcher event vocabulary expansion as the event log grows.
- Performance + pagination on long-tail views (Work firehose, Recent
  feeds).
- Accessibility, keyboard navigation, focus management.

## What does NOT live here

- New homes or sub-views (those get their own specs).
- Architectural changes to the shell, chat dispatcher, or rendering
  model (those go back to the parent initiative).
- Domain pack work (PM, QA, etc. — those are separate products).
- hero-code adapter wire-up (lives in the hero-code repo).

## Specs (children)

| Spec | Status | Purpose |
|---|---|---|
| [hero-surface-polish-v1](../../features/hero-surface-polish-v1/spec.md) | planning | First polish pass: broken sub-routes (23×), Now data-fetcher event vocabulary, Work firehose + dupes, Knowledge default view, People default view split |

Future child specs land here as they're triaged from running the
surface.

## How this initiative ends

It doesn't — it stays open as the standing shelf for polish work. Each
v1, v2, … child completes and archives; the initiative itself stays in
`planning` to absorb the next pass.
