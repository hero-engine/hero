---
title: Scan Pluggability — Per-Domain `hero scan` Implementations
type: feature
status: planning
priority: P0
tags: [platform, domains, scan, ingestion, refactor]
created: 2026-05-15
relations:
  - target: hero-domains
    kind: parent
depends-on:
  - domain-plugin-architecture
  - spec-type-registry
horizon: next
smoke: deferred
---

> **Status: awaiting platform primitives.** This stub is a `/design`-ready
> brief, not a complete design. Implementation is blocked on
> `domain-plugin-architecture` (the pack layout) and `spec-type-registry`
> (so PM-domain scanners have target types to write into). `/design
> scan-pluggability` can resolve the open questions below in parallel
> with those primitives.

## Kickoff

Make `hero scan` domain-aware. Today scan is a code scanner: detect languages, frameworks, test runners, write engineering-shaped knowledge. PM needs a totally different scan: import a roadmap doc, parse tracker epics, ingest OKRs. Generalize `hero scan` to dispatch to the active pack's scanner; engineering's code scan becomes the reference impl under `domains/engineering/scan/`. PM-specific scanners ship in `hero-pm`, not here.

**Status:** planning — stub written 2026-05-15. Blocked on `domain-plugin-architecture` and `spec-type-registry`.

**Pick up at:** Run `/design scan-pluggability`. First decision: scan output schema — do all domains emit the same node/edge types into the graph, or domain-typed nodes? Then design the dispatch surface (`hero scan` reads active pack and runs its scanner).

→ `/design scan-pluggability`

**Files:** .hero/planning/features/scan-pluggability/spec.md, .hero/planning/initiatives/hero-domains/spec.md, internal/scan/, internal/index/
**Skip:** Designing PM scanners (roadmap-doc parser, tracker-epic ingester) — those live in `hero-pm`. This spec only ships the dispatch shape and engineering's reference scanner.

## Goal

Generalize `hero scan` so the active domain pack decides what scanning
means. Engineering's current code-scan (language detection, framework
detection, test runner detection, knowledge base entries) becomes the
reference implementation under `domains/engineering/scan/`. The CLI
`hero scan` looks up the active pack and dispatches to its scanner.

PM-domain scanners (roadmap-doc parsing, tracker-epic import, OKR
ingest) will be designed inside `hero-pm`; this spec defines the
plug-in shape they implement and ships nothing PM-specific.

## Why now

PM onboarding is fundamentally different. Asking `hero scan` to detect
Go modules in a PM project is wrong; the right scan is "import the
roadmap doc you already have, parse tracker epics into stories, ingest
OKRs if they exist." Until scan is pluggable, PM onboarding either
runs the wrong scan or skips scan entirely — both bad.

This spec depends on `spec-type-registry` because a PM scanner needs
to emit `epic`, `story`, `roadmap-item` typed specs; without the
registry, those types don't exist.

## Scope outline

The design pass should cover:

1. **Scanner interface.** A Go interface (or manifest + handler
   lookup) that a domain pack implements. Inputs: project root,
   config. Outputs: discovered specs, knowledge entries, graph
   nodes/edges, and a structured report.
2. **Dispatch.** `hero scan` reads the active domain from `hero.json`
   and invokes that pack's scanner. Pre-scan dependency checks
   (workspace exists, etc.) stay in core.
3. **Output schema.** What scanners emit. The big open question:
   one shared node/edge schema across all domains, or each domain
   emits domain-typed nodes?
4. **Engineering reference impl.** Move today's `internal/scan/`
   code-scanning logic into `domains/engineering/scan/`. Behavior
   bit-identical.
5. **Shared scan infrastructure.** Some scan plumbing is generic
   (writing to the knowledge base, progress reporting, lint of
   produced specs). That stays in core; only the domain-specific
   detection logic moves into the pack.
6. **Report format.** Today `hero scan` produces a report and
   summary. Keep that contract; let pack scanners populate it.

## Touchpoints (sketch — confirm during design)

- `internal/scan/` — most logic moves to
  `domains/engineering/scan/`; the dispatch shell stays
- `internal/cli/scan.go` — invokes active pack's scanner
- `domains/engineering/scan/` — new directory, reference impl
- `internal/index/` — scan output writes through here, no change
  expected
- `internal/spec/` — depends on `spec-type-registry` so scanners can
  emit non-engineering types

## Unknowns for design pass

1. **Output schema — shared or domain-typed.** Do all scanners emit
   the same graph node and edge types (e.g. generic `Component`,
   `Dependency`, `Document`)? Or does each domain emit
   domain-typed nodes (`RoadmapItem`, `Epic`, `Component`,
   `Dependency`)? Domain-typed is more honest but couples scan to
   the spec-type registry more tightly.
2. **Scanner manifest vs Go code.** Same shape as
   `spec-type-registry`'s question — declarative manifest, compiled
   Go interface, or both?
3. **Scan composition.** Can a project run multiple scanners
   (engineering + PM in a coexistence world)? Probably yes
   eventually, but v1 is single-domain. Confirm.
4. **Progress and cancellation.** Scan today produces progress
   output. The interface should preserve that for any scanner the
   pack ships, not just engineering's.
5. **Failure handling.** If a pack scanner crashes mid-scan, what
   state does the workspace end in? Today's scan has its own answer;
   carry it forward.

## Boundaries

- **Not** designing PM scanners — those land in `hero-pm`.
- **Not** changing the knowledge base schema (depends on the answer
  to unknown #1).
- **Not** changing scan triggering (CLI command, post-clone hook) —
  only the implementation surface.
- **Not** introducing third-party / out-of-tree scanners. Scanners
  ship with their pack, in-tree.

## Risks

- **Output schema decision propagates.** If domain-typed nodes win,
  the knowledge graph needs to know about every domain's node types
  ahead of time. If shared-types win, scan loses some fidelity.
  This decision affects `domain-scoped-knowledge-graph` (item #6).
- **Engineering reference parity.** Moving `internal/scan/` into a
  pack must preserve current behavior bit-for-bit. Diff the scan
  output of a real project before/after.
- **Pack-shipped scanners are user-visible early.** PM users run
  scan during onboarding; a flaky scanner is a bad first impression.
  Plan testing scope explicitly.
