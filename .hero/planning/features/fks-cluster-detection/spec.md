---
title: "Feature Cluster Detection — Infer Explainer-Worthy Spec Clusters"
type: feature
status: planning
slug: fks-cluster-detection
domain: engineering
parent: feature-knowledge-synthesis
priority: medium
size: large
created: 2026-06-23
tags: [knowledge, synthesis, clustering, graph, detection, explainer]
kind: new
relations:
  - target: fks-on-demand-synthesizer
    kind: depends-on
---

# Feature Cluster Detection

## Goal

Find the feature boundaries Hero is *not* told about. Given the spec graph,
detect coherent clusters of **completed** specs that together constitute one
shipped feature, score each by confidence, and emit them as candidates for
synthesis — so the originating pain ("a feature shipped across many specs and
nobody made a how-it-works doc") is caught automatically instead of requiring a
human to name the slugs to `hero synthesize`. Detection only: this spec emits
candidates; it does not synthesize (that's #2, which it calls) and does not
prompt the user (that's #4).

## Kickoff

Build cluster detection for the feature-knowledge-synthesis initiative. Given
all specs, produce candidate clusters of completed specs that look like one
shipped feature, each with a confidence score, the signals that fired, and a
suggested out-slug. Two sources: (a) an initiative/epic at `completed` → its
child set is a zero-false-positive cluster; (b) inferred connected components
over the relation graph + file-overlap, above a confidence threshold. **Graph
signals dominate; time-proximity is a weak tiebreaker only** (see Problem).
Gate on completeness — never propose a cluster with open members. Output feeds
#4 (trust handshake); synthesis itself is `hero synthesize` (#2). Likely a new
`internal/synthesize` (or sibling) detector + a `hero synthesize --detect`
surface. Depends on #2.

## Problem

`hero synthesize` (#2) requires the caller to name the cluster. That's fine for
a human who knows a feature shipped, but the initiative's real value is Hero
*noticing*. The hard part is drawing the boundary without a human — and the
naive approach is actively wrong:

**Time-window clustering does not work.** Delivering #2 proved this directly:
synthesizing `cold-start-trust-hardening` pulled in same-day commits from two
*unrelated* features (`feature-knowledge-synthesis`, `team-mode-cloud-
coordination`) because they shared a calendar day. On an active repo, "specs
completed near each other in time" is a poor proxy for "specs that are one
feature." Detection must lean on **structural** signals — what's actually linked
and what touches the same code — and treat time as a weak tiebreaker at most.

## Design

### Candidate sources

1. **Explicit (zero false positives).** When an initiative or epic reaches
   `completed`, its child set *is* a cluster. This is the highest-confidence
   source and should always be offered. (Hooking the completion transition to
   fire detection is in scope; the user-facing prompt is #4.)
2. **Inferred.** Connected components over the relation graph (parent/child,
   depends-on, related, blocks) intersected with **file-overlap** (specs whose
   `FilesTouched` intersect), among completed specs. Each component is a
   candidate.

### Signals & scoring (structural first)

Score each inferred candidate by a weighted combination, strongest to weakest:

| Signal | Weight | Why |
|---|---|---|
| Shared parent / initiative | highest | An explicit grouping the author already made |
| Relation edges among members | high | Specs that reference each other |
| Co-touched files | high | One feature tends to touch one code region |
| Shared tags | medium | Weak author-supplied grouping |
| Time proximity | **low** | Demoted — same-window ≠ same feature (see Problem) |
| Same author | low | Noisy on small teams |

Emit candidates above a confidence threshold, each carrying the score **and the
signals that fired** (explainability — #4 shows the user *why* a cluster was
proposed).

### Completeness gate

Never propose a cluster that still has open members — an explainer describes a
feature *as it exists now*, so a half-shipped feature would synthesize a lie.
Require all (or a high fraction of) cluster members to be `completed`.

### Output

A list of `Candidate{ Slugs, SuggestedOutSlug, Confidence, Signals }`. The
suggested out-slug follows #2's rule (initiative slug if present, else the
dominant member). Candidates are the input to #4; nothing is written or prompted
here.

### Dedup against existing explainers

Skip clusters that already have a current explainer (`synthesized_from` covers
them and `last_synthesized` is recent) — that's #5's amendment territory, not a
fresh candidate.

## Acceptance Criteria

- WHEN an initiative or epic reaches `completed` THE SYSTEM SHALL emit its
  completed child set as a high-confidence cluster candidate.
- THE SYSTEM SHALL infer candidate clusters from the relation graph and
  file-overlap among completed specs, scored by the structural signals above.
- THE SYSTEM SHALL weight structural signals (shared parent, relations,
  co-touched files) above time-proximity, which SHALL be at most a weak
  tiebreaker.
- IF a candidate cluster contains any non-`completed` spec THEN THE SYSTEM SHALL
  NOT emit it (completeness gate).
- THE SYSTEM SHALL attach to each candidate its confidence score and the signals
  that fired, so a downstream surface can explain the proposal.
- WHERE a cluster is already covered by a current explainer THE SYSTEM SHALL omit
  it from candidates (defer to amendment, #5).
- THE SYSTEM SHALL emit candidates only — it SHALL NOT synthesize or prompt.

## Boundaries / Out of scope

- No synthesis — candidates are handed to `hero synthesize` (#2).
- No user-facing prompt / trust handshake / autonomy setting — that's #4.
- No amendment of existing explainers — that's #5.
- No new clustering ML; this is weighted graph/heuristic scoring.

## Dependencies

- **Depends on `fks-on-demand-synthesizer` (#2)** — the engine candidates feed.
- Consumes the relation graph and `FilesTouched`; coordinate with how
  `internal/spec` exposes edges and touched files.
- Output consumed by `fks-trust-handshake` (#4).

## Notes

- The time-window limitation is a *delivered observation*, not a hypothesis —
  recorded so #3 doesn't repeat #2's same-day conflation.
