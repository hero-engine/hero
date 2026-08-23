---
title: "Hero Cross-Tool Continuity Proof"
slug: hero-continuity-proof-demo
type: feature
status: planning
domain: engineering
size: medium
priority: high
horizon: now
created: 2026-08-04
tags: [demo, continuity, evidence, trust]
parent: hero-marketing
depends-on: [hero-public-truth-baseline, hero-positioning, hero-public-repo-readiness]
relations:
  - target: hero-landing-message-refresh
    kind: conflicts-with
supersedes: [hero-demo-content]
---

# Hero Cross-Tool Continuity Proof

## Goal

Produce one repeatable two-tool demonstration proving that project intent, a rejected approach, a correction, partial progress, and the next action survive a cold session boundary and close against evidence.

## Kickoff

Creates the must-win proof behind Hero's supervision-reduction claim.

**Status:** planning — blocked on approved claims, positioning, and a public-safe repository fixture.

**Pick up at:** define a deterministic fixture and evidence rubric before recording any output or assets.

→ `.hero/planning/initiatives/hero-marketing/hero-continuity-proof-demo/spec.md`

**Files:** `.hero/NEXT.md`, `.hero/events.log`, `.hero/planning/`, `internal/cli/verify.go`, repeatable fixture/scripts, evidence transcript, real captures, and landing/docs-ready assets
**Skip:** a broad GIF/video asset catalog or synthetic output presented as a capture.

## Changes

1. Define a realistic engineering task in tool A with a rejected approach, durable correction/constraint, decision, and partial delivery state.
2. Start a cold session in a different supported tool and issue only the agreed continuation prompt; measure whether Hero retrieves the right context without manual re-priming.
3. Complete the task through acceptance evidence, cold delivery audit, and `hero spec verify`.
4. Capture inputs, outputs, artifact provenance, environment, prerequisites, revision, and known limits in a reproducible demo package.
5. Produce landing/docs assets from real output or label any illustrative simplification explicitly.

The fixture SHALL preserve the tool-A state through Hero's normal `.hero/events.log`, spec, and projected handoff surfaces; the closing evidence SHALL exercise the current `internal/cli/verify.go` contract rather than a narrated substitute.

## Acceptance Criteria

- **AC-1:** WHEN a cold tool-B session receives only the continuation prompt for the prepared tool-A state THE SYSTEM SHALL retrieve the rejected approach, correction, decision, partial progress, and next action.
- **AC-2:** WHEN tool B continues the task THE SYSTEM SHALL avoid the rejected approach without manual re-priming.
- **AC-3:** WHEN the demonstration completes THE SYSTEM SHALL include acceptance, test/build, cold-audit, and verify evidence tied to a source revision.
- **AC-4:** THE SYSTEM SHALL provide a repeatable fixture, transcript, prerequisites, limitations, and provenance sufficient for independent replay.
- **AC-5:** IF output is simplified for presentation THEN THE SYSTEM SHALL label it illustrative and preserve a link to the real evidence.

## Boundaries

- No absolute “Correct your AI once” promise based on one demonstration.
- No fictional CLI status, hidden manual context injection, or hand-edited evidence.
- No unrelated brand or launch content.

## Validation

- Replay the demonstration from a clean fixture with two supported harnesses and compare expected evidence checkpoints.
- Run the documented acceptance/test/build/audit/verify gates plus Hero lint/score and index refresh.
