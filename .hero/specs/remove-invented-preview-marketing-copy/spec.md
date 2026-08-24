---
title: "Remove Invented Preview and Internal Launch Copy from Public Marketing"
slug: remove-invented-preview-marketing-copy
type: bug
status: completed
domain: engineering
size: medium
priority: critical
severity: high
root_cause_class: design
created: 2026-08-24
tags: [marketing, public-copy, landing, documentation, truth]
parent: hero-marketing
delivery_method: manual
completed_at: 2026-08-24T01:49:26Z
---

# Remove Invented Preview and Internal Launch Copy from Public Marketing

## Summary

Hero's public copy incorrectly labels the shipped memory-and-delivery product
model as a preview pending an invented cross-tool proof requirement. The landing
page also renders repository-visibility controls and unresolved build
placeholders as customer-facing marketing content. Both defects block the
public launch.

**Classification:** design defect, with a secondary enforcement gap.

## Background

Hero's product model is two connected systems: durable project memory informs
delivery, and delivery leaves decisions, evidence, corrections, and a current
handoff for later sessions. The memory system and delivery system are shipped.
An internal planning thread introduced a separate continuity demonstration and
then incorrectly promoted it into a prerequisite for describing that existing
product model.

The public landing source separately exposed release-readiness implementation
details: repository gating language, build revision fields, and generation
timestamps. Those details belong in repository documentation and
machine-readable deployment metadata, not in the visual marketing page.

## Analysis

The canonical claim registry marks `product-two-system-loop` as Class D,
preview, and contingent on independent public proof. The positioning authority
inherits that classification, and README, hosted docs, release notes, and the
landing page repeat it. Release-candidate tests then require the incorrect copy,
turning an unsupported planning decision into an enforced public narrative.

The landing page also has a visual `trust-strip` that describes repository
visibility gates and renders `BUILD_TIME_SOURCE_REVISION` and
`BUILD_TIME_GENERATED_AT`. The source revision already has appropriate
machine-readable homes in a head meta tag and `/revision.json`.

## Root Cause

The public truth baseline treated a proposed demonstration as the evidentiary
authority for Hero's shipped product architecture instead of checking the
implemented memory, retrieval, handoff, and verified-delivery paths. Public
surfaces then inherited that unsupported qualifier. No public-copy guard rejects
internal proof-gate language or visible build placeholders on the marketing
surface.

## Source

- `.hero/specs/hero-public-truth-baseline/public-claim-registry.yaml`
- `.hero/marketing/positioning.md`
- `web/landing/site/index.html`
- `web/docs/src/`
- `README.md`
- `docs/releases/v0.34.0-candidate.md`
- `scripts/test_release_candidate.py`
- `web/landing/scripts/landing_build.py`
- `internal/cli/public_docs_check.go`

## Fix Direction

1. Restore `product-two-system-loop` to a shipped, implementation-backed claim.
2. State the memory/delivery loop directly on public surfaces without a preview,
   proof, or perfection disclaimer.
3. Remove the landing page's preview banner, repository-boundary trust strip,
   and visible artifact-revision footer.
4. Preserve revision provenance in the head meta tag and `/revision.json` only.
5. Add regression checks that reject the invented proof language and unresolved
   build placeholders in visible landing content while preserving legitimate
   preview labels for genuinely preview capabilities such as headless runtime.

## Problem Statement

Public visitors are shown unsupported caveats about Hero's core value and
internal operational controls that do not help them understand or evaluate the
product. The resulting copy is both factually wrong and inappropriate for a
marketing site.

## Environment Details

- Affected release candidate: v0.34.0
- Affected source: local `main` before public visibility and release publication
- Public launch state: paused; no repository visibility, tag, release, or
  deployment mutation may proceed until this fix verifies

## Code Flow

1. Claim registry defines the product claim and evidence class.
2. Positioning authority translates the claim into reusable public language.
3. README, docs, release notes, and landing source consume that language.
4. Landing and release-candidate checks validate required and prohibited copy.
5. Deployment builds substitute revision metadata and publish static artifacts.

## Key Files

- `.hero/specs/hero-public-truth-baseline/public-claim-registry.yaml`
- `.hero/marketing/positioning.md`
- `web/landing/site/index.html`
- `web/landing/scripts/landing_build.py`
- `internal/cli/public_docs_check.go`
- `internal/cli/public_docs_check_test.go`

## Secondary Defects

- The visual landing page exposes unresolved build-time placeholders.
- Release-candidate tests require the incorrect preview copy.
- Public-copy checks do not distinguish machine-readable provenance from visible
  marketing content.

## Changes

1. Correct the canonical product claim and positioning authority.
2. Refresh every affected public narrative surface.
3. Remove internal launch/provenance UI from the landing page.
4. Replace tests that require the bad copy with positive and negative regression
   coverage for the corrected message.

## Acceptance Criteria

- WHEN a public surface explains Hero's memory-and-delivery model, THE SYSTEM SHALL describe the reinforcing loop as a shipped product model without a preview, proof, demonstration, or perfection qualifier.
- THE SYSTEM SHALL remove the preview outcome banner, repository-boundary trust
  strip, and visible artifact-revision footer from the landing page.
- THE SYSTEM SHALL retain source revision metadata in the landing head meta tag
  and `/revision.json` without rendering build placeholders as customer-facing
  content.
- THE SYSTEM SHALL update the claim registry, positioning authority, README,
  hosted docs, and v0.34 candidate notes so they cannot reintroduce the invented
  qualification.
- WHEN invented continuity-proof copy or visible unresolved build placeholders are added to a public marketing surface, THE SYSTEM SHALL fail the public-copy validation gate.
- THE SYSTEM SHALL preserve accurate preview labels for genuinely preview
  capabilities, including the headless agent runtime.

## Validation

- Run the landing strict build and landing tests.
- Run hosted documentation strict build and tests.
- Run public documentation checks and their Go tests.
- Run release-candidate copy tests.
- Search all public narrative sources for the removed phrases and unresolved
  visible placeholders.
- Render and inspect the landing page at desktop and narrow widths.

## Boundaries

- Do not redesign the landing page beyond removing the defective content.
- Do not remove machine-readable revision provenance.
- Do not change legitimate availability labels unrelated to the core
  memory-and-delivery product model.
- Do not make the repository public, publish v0.34, deploy the corrected sites,
  or change DNS until this fix passes its delivery gates.

## Kickoff

Begin with the canonical claim registry and positioning authority. Replace the
invented preview classification with implementation-backed shipped evidence,
then update all downstream surfaces. Remove the landing trust strip and visible
revision footer rather than rewriting them. Add guardrails that reject the
specific failure mode, run the public-site and release checks, visually inspect
the landing page, and complete the cold audit and verification before resuming
the launch initiative.

## Recap

This fix restores Hero's actual product message—project memory plus verified
delivery—and keeps internal launch controls out of the customer experience.

## Completion Ledger

| AC | Status | Evidence |
|---|---|---|
| AC-1 (shipped reinforcing loop without invented qualifier) | DONE | Claim registry now classifies `product-two-system-loop` as Class A/shipped; README, hosted docs, release notes, and landing use the corrected product model; public-source search finds no removed narrative copy. |
| AC-2 (remove landing banner and internal trust UI) | DONE | `web/landing/site/index.html` no longer contains `.preview-note`, `.trust-strip`, repository-boundary copy, or the artifact-revision footer; rendered landing inspected at the system and get-started sections. |
| AC-3 (machine-readable revision only) | DONE | Head `hero-source-revision` meta tag and `web/landing/site/revision.json` remain; landing build and artifact checks pass without visual build metadata. |
| AC-4 (correct every authority and downstream surface) | DONE | Updated claim registry, positioning authority, README, hosted docs, core concepts, and v0.34 candidate notes; `go run ./cmd/hero docs check --public --invocations` reports no issues. |
| AC-5 (regression guards) | DONE | Added forbidden-copy checks in `landing_build.py` and `public_docs_check.go`; the public contract now scans the positioning authority, claim registry, and v0.34 candidate notes for the invented qualifiers, with Python and Go mutation coverage. |
| AC-6 (preserve legitimate preview labels) | DONE | Headless runtime remains labeled Preview in the landing, hosted docs, claim registry, release notes, and release-candidate tests. |
| Validation | DONE | `go test ./...`; landing 8 tests plus source/build/artifact checks; docs 28 tests plus strict MkDocs build and link check; release candidate 14 tests; public docs contract and invocation validation all pass. |

- [x] exercise-the-feature: built the real landing artifact from the corrected source, opened it in a browser, and inspected the memory/delivery and get-started sections to confirm the invented banner and internal trust strip are absent.

## Delivered Changes

| # | Change | Status |
|---|---|---|
| 1 | Correct canonical claim registry and positioning authority | DONE |
| 2 | Refresh README, hosted docs, concepts, and v0.34 candidate notes | DONE |
| 3 | Remove landing preview banner, repository trust strip, and visual artifact revision | DONE |
| 4 | Add public-copy and visible-provenance regression checks | DONE |

## Delivery Audit

**Audited:** complete working-tree diff (`git diff`), cold re-audit
**Verdict:** SHIP
**Surface:** clean
**Confidence:** high

### Acceptance Criteria

- [✓] AC-1 — The claim registry classifies `product-two-system-loop` as Class A/shipped, and the changed README, hosted-docs, release-note, positioning, and landing copy describe the reinforcing model without the removed preview/proof/perfection qualifier (`.hero/specs/hero-public-truth-baseline/public-claim-registry.yaml:47`, `README.md:6`, `web/landing/site/index.html:380`).
- [✓] AC-2 — The landing source no longer contains `.preview-note`, `.trust-strip`, the repository-boundary block, or the artifact-revision footer; the remaining footer contains only `Hero Engine` (`web/landing/site/index.html:380`, `web/landing/site/index.html:492`, `web/landing/site/index.html:525`).
- [✓] AC-3 — The head retains `hero-source-revision` and `web/landing/site/revision.json` retains the full machine-readable provenance contract, while no build placeholder remains in visible landing markup (`web/landing/site/index.html:19`, `web/landing/site/revision.json`).
- [✓] AC-4 — The diff updates the claim registry, positioning authority, README, affected hosted-docs pages, and v0.34 candidate notes; the current public narrative search contains no removed qualifier on those surfaces.
- [✓] AC-5 — Invented continuity qualifiers are rejected across public narrative surfaces and the three public-truth authorities through `inventedContinuityRules` and `publicTruthAuthorityPaths`; mutation coverage exercises every authority, release-candidate tests reject the removed phrases, and visible landing provenance is independently rejected (`internal/cli/public_docs_check.go:86`, `internal/cli/public_docs_check.go:91`, `internal/cli/public_docs_check_test.go:97`, `scripts/test_release_candidate.py:159`).
- [✓] AC-6 — Legitimate headless-runtime preview labels remain in README, hosted docs, positioning authority, claim registry, and v0.34 candidate notes (`README.md:180`, `web/docs/src/index.md:41`, `docs/releases/v0.34.0-candidate.md:52`).

### Changes

- [✓] Correct the canonical product claim and positioning authority — both are changed to Class A/shipped language with implementation evidence.
- [✓] Refresh every affected public narrative surface — all named current surfaces in the diff carry the corrected message.
- [✓] Remove internal launch/provenance UI from the landing page — the banner, trust strip, visible revision link, timestamp, and artifact-revision footer are removed while head and JSON provenance remain.
- [✓] Replace bad-copy tests with positive and negative regression coverage — landing checks reject invented qualifiers and visible provenance, Go mutation tests cover public narratives plus every authority path, and release-candidate tests assert the removed phrases stay absent.

### Validation Evidence

- `go test ./...` — PASS.
- Landing validation — PASS (8 tests, strict build, and artifact checks).
- Hosted-documentation validation — PASS (28 tests, strict build, and link check).
- Release-candidate validation — PASS (14 tests).
- Public-documentation contract and invocation validation — PASS.
