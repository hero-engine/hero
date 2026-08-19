---
title: "Hero Public Truth — Product Story and Documentation Repair"
slug: hero-marketing
type: initiative
status: planning
domain: engineering
size: x-large
priority: critical
horizon: now
created: 2026-04-25
tags: [marketing, positioning, documentation, public-truth, continuity, trust]
child:
  - hero-public-truth-baseline
  - hero-positioning
  - hero-root-docs-remediation
  - hero-hosted-docs-remediation
  - hero-continuity-proof-demo
  - hero-landing-message-refresh
  - hero-public-docs-drift-guard
relations:
  - target: hero-self-consistency
    kind: related
  - target: generated-command-refs-validated
    kind: related
---

# Hero Public Truth — Product Story and Documentation Repair

## Vision

Hero tells one public story that is both differentiated and demonstrably true: it carries a project's intent, decisions, corrections, and delivery evidence from one AI session to the next, so engineers spend less time re-explaining and supervising. Public onboarding works exactly as written, shipped capabilities are easy to discover, maturity labels prevent roadmap work from masquerading as product, and automated checks catch factual drift before deployment.

## Goal

Replace the stale launch-and-growth premise of this existing initiative with a bounded truth-repair program. Establish an evidence-backed claim baseline, rewrite positioning around reduced supervision through durable project intelligence, repair root and hosted documentation, prove continuity in a repeatable two-tool demonstration, refresh the landing message only after the proof exists, and close with production-aware drift guards.

## Kickoff

Rebuilds Hero's public story around session continuity and evidence-backed completion while repairing false onboarding and documentation claims.

**Status:** planning — the stale marketing program has been recomposed into seven sequenced truth-repair children; no public content has been changed.

**Pick up at:** design and deliver `hero-public-truth-baseline`, including the P0 correction packet and maturity registry every later child consumes.

→ `.hero/planning/initiatives/hero-marketing/hero-public-truth-baseline/spec.md`

**Files:** `.hero/planning/initiatives/hero-marketing/content-truth-audit.md`, `README.md`, `web/docs/src/`, `web/landing/site/index.html`
**Skip:** launch campaigns, telemetry, pricing, community growth, and a broad brand overhaul.

## Product direction

### Core message

> Hero carries your project's intent, decisions, and evidence from one AI session to the next, so you spend less time re-explaining and supervising.

- **Category:** operating layer for AI-assisted engineering.
- **Outcome:** AI sessions inherit the project and finish against evidence.
- **Mechanism:** specs, decisions, conventions, project intelligence, and delivery evidence become durable context that supported harnesses can retrieve and act against.
- **Wedge:** engineering leads; PM and Sales expansion must be described only with verified maturity labels.
- **Candidate, not promise:** “Correct your AI once” remains a claim to test and prove. It is prohibited as an absolute statement until the continuity demonstration and claim baseline support it.

Specs remain a forcing function for trustworthy work; they are not the headline category. Roster size, raw command counts, and “spec-driven” framing are reference detail rather than primary differentiation.

## Why now

The current public surfaces contain operational falsehoods and material drift while structural checks remain green. Known P0 failures include nonexistent satellite commands, invalid repair syntax, a false nested-workspace architecture, an obsolete verify-then-complete sequence, an undecodable configuration example, an impossible Go prerequisite, and the dead `hero verify-install` command. P1 drift includes stale release/version claims, contradictory counts, fictional command output, inaccurate Codex slash-command language, stale repository trees, and wrong one-graph peering semantics.

At the same time, the public story underplays shipped or meaningful abilities: cold delivery audit plus verification evidence, cross-session/tool continuity, Attention/Mail/Focus, the `hero serve` project intelligence UI, guarded code-host operations, tracker evidence and mutations, cross-repo Project Mail/advisory/spec-out/handoff, and approval-aware headless execution. PM and Sales domain packs may be mentioned only with maturity caveats until release-level proof exists.

The durable audit and claim taxonomy live in [`content-truth-audit.md`](content-truth-audit.md).

## Child specs and phases

| Phase | Child | Priority | Size | Depends on | Outcome |
|---:|---|---|---|---|---|
| 1 | [`hero-public-truth-baseline`](hero-public-truth-baseline/spec.md) | critical | medium | — | Claim/capability inventory, evidence authority, maturity labels, and a P0 correction packet. |
| 1 | [`hero-positioning`](hero-positioning/spec.md) | high | medium | baseline | Supervision/continuity/trust message hierarchy, audiences, proof pillars, vocabulary, and prohibited claims. |
| 2 | [`hero-root-docs-remediation`](hero-root-docs-remediation/spec.md) | high | medium | baseline + positioning | Executable root onboarding and accurate configuration, harness, peering, and team guidance. |
| 2 | [`hero-hosted-docs-remediation`](hero-hosted-docs-remediation/spec.md) | high | large | baseline + positioning | Page-by-page hosted-doc factual repair and discoverability for differentiated capabilities. |
| 2 | [`hero-continuity-proof-demo`](hero-continuity-proof-demo/spec.md) | high | medium | baseline + positioning | Repeatable two-tool cold-resume proof with a surviving correction and evidence-backed completion. |
| 3 | [`hero-landing-message-refresh`](hero-landing-message-refresh/spec.md) | high | medium | root docs + hosted docs + demo | Current, truthful landing message using real or explicitly illustrative product evidence. |
| 4 | [`hero-public-docs-drift-guard`](hero-public-docs-drift-guard/spec.md) | medium | medium | root docs + hosted docs + landing | Derived assertions, executable examples, deployment freshness, and a zero-unresolved-claims production crawl. |

The critical path is:

```text
hero-public-truth-baseline
  → hero-positioning
      → hero-root-docs-remediation ─┐
      → hero-hosted-docs-remediation├→ hero-landing-message-refresh
      → hero-continuity-proof-demo ─┘
          → hero-public-docs-drift-guard
```

Phase 2 children may proceed in parallel only when the reciprocal overlap guards below are respected. If `hero-hosted-docs-remediation` grows beyond a coherent page-by-page repair, `/split` it into factual remediation first and capability discoverability second.

## In-flight overlap watch

Every seam below has reciprocal `conflicts-with` relations on the named children. Do not deliver each pair concurrently without first partitioning the listed assets.

| Child A | Child B | Shared seam |
|---|---|---|
| `hero-public-truth-baseline` | `hero-public-docs-drift-guard` | claim registry, derived counts/version authority, and freshness contract |
| `hero-root-docs-remediation` | `hero-public-docs-drift-guard` | root markdown scans, quickstarts, counts, and command assertions |
| `hero-hosted-docs-remediation` | `hero-public-docs-drift-guard` | `web/docs/src` scanning, navigation, releases, and generated references |
| `hero-continuity-proof-demo` | `hero-landing-message-refresh` | demonstration fixtures, captures, embeds, and proof claims |
| `hero-hosted-docs-remediation` | `hero-landing-message-refresh` | `web/` metadata, terminology, navigation links, and destination URLs |

## Existing child disposition

These dispositions preserve history without pretending stale lifecycle state is accurate. No historical item is being marked complete by this recomposition.

| Existing slug | Disposition |
|---|---|
| `hero-positioning` | Relocated from `.hero/planning/features/hero-positioning/spec.md` and rewritten in place under this initiative. Its narrative/ICP/messaging/comparison intent is preserved; its “spec layer” headline and count-based proof are retired. |
| `hero-landing-page` | Historical landing delivery with stale `delivering` status and a v0.9 premise. Reconcile its delivery evidence/status separately, then use `hero supersede hero-landing-page --by hero-landing-message-refresh` if genealogy is still appropriate. Its content scope is replaced here. |
| `hero-docs-site` | Historical hosting/site delivery whose `planning` status and greenfield premise no longer describe reality. Reconcile separately, then formally supersede with `hero-hosted-docs-remediation` if warranted. Its content scope is replaced here. |
| `hero-distribution` | Most stated gaps now exist (binaries/releases/install paths). Reconcile separately and move any remaining release hardening to a release/platform initiative; it is not a child of this truth-repair program. |
| `hero-demo-content` | Broad asset-catalog premise is replaced by the narrow continuity proof. Reconcile the old spec, then formally supersede it with `hero-continuity-proof-demo` rather than hand-editing genealogy. |
| `hero-launch-playbook` | Deferred until truthful surfaces and a small external cohort validate the message. Re-author later without coordinated voting/comment tactics. |
| `hero-content-engine` | Removed from this initiative; ongoing publishing is a separate growth program. |
| `hero-community` | Removed from this initiative; community/support operations are not a prerequisite for truthful docs. |
| `hero-telemetry` | Removed from this initiative; product analytics and privacy policy require their own product decision. |

Several historical specs still declare `hero-marketing` as parent. Their frontmatter and stale lifecycle state are intentionally untouched here; before `/drive hero-marketing`, reconcile those inbound relations with the supported lifecycle/supersession commands so the executable child set matches the seven-child roster above.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL provide one evidence-backed public claim registry with `shipped`, `optional`, `preview`, and `planned` availability labels.
- **AC-2:** WHEN a user follows a public onboarding path THE SYSTEM SHALL execute the documented commands and configuration against the current product contract.
- **AC-3:** WHEN Hero is described on a root doc, hosted docs page, or landing surface THE SYSTEM SHALL use the supervision/continuity/trust hierarchy and cite or link to appropriate proof.
- **AC-4:** IF a claim depends on unverified cloud, team, outpost, domain-pack, licensing, or code-host readiness THEN THE SYSTEM SHALL label the uncertainty and prohibit an unconditional shipped claim.
- **AC-5:** WHEN a public build is validated THE SYSTEM SHALL detect command, count, version, configuration, harness, peering, link, and deployment-freshness drift across all public surfaces.
- **AC-6:** WHEN the initiative closes THE SYSTEM SHALL produce a production crawl with zero unresolved P0/P1 claims and a revision marker tying deployed content to reviewed source.

## Boundaries

- No launch campaign, Product Hunt/HN calendar, newsletter, social cadence, or community program.
- No telemetry backend, pricing, competitor microsites, enterprise sales collateral, or domain-platform marketing.
- No broad visual brand overhaul; only assets necessary to make repaired surfaces coherent and truthful.
- No public claim that cloud/team/outposts or all 21 code-host operations are production-ready until evidence resolves their maturity and prerequisites.
- No changing product behavior merely to preserve old docs; docs follow the supported product contract.
- No implementation of README, docs, landing, site, or code during initiative composition.

## Non-Goals

- Re-launching Hero, growing an audience, pricing the product, or choosing a new visual identity.
- Making unfinished cloud, team, outpost, domain-pack, or integration work appear market-ready.
- Preserving stale documentation behavior through compatibility code instead of documenting the supported contract.

## Verification

- Lint and score the initiative and all seven children; index the corpus and confirm parent, dependency, related, and reciprocal `conflicts-with` edges resolve uniquely.
- Require each delivery child to reconcile its assigned claim-registry rows and run its executable/build validation before completion.
- Close only after strict source builds, executable examples, link/accessibility checks, deployment revision verification, and a production crawl report zero unresolved P0/P1 claims.

## Risks and open decisions

- **Licensing:** no repository `LICENSE` exists; “open source” and licensing claims remain prohibited until posture is decided.
- **Maturity:** public readiness of cloud/team/outposts, PM/Sales packs, and all 21 code-host operations needs explicit evidence.
- **Audience:** an individual AI-native engineer versus a 5–50-person engineering lead must be chosen as the lead audience during positioning.
- **Release channel:** confirm whether the newest repository tag is the public channel and how deployed surfaces derive it.
- **Counts:** avoid mutable roster counts outside generated reference pages; counts are weak differentiation and maintenance liabilities.
- **False confidence:** `mkdocs --strict`, docs checks, and invocation checks currently pass despite false content, so green structure checks cannot be treated as truth evidence.

## Progress

- 2026-04-25 — Original broad positioning/distribution/launch initiative created.
- 2026-08-04 — Recomposed in place into a truthful public product story and documentation repair program. Seven child stubs and a durable content-truth audit were authored; no public content was changed.
