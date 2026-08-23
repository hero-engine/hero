---
title: "Hero v0.34 Public Release Readiness"
slug: hero-marketing
type: initiative
status: planning
domain: engineering
size: giant
priority: critical
horizon: now
autonomy: autonomous
created: 2026-04-25
tags: [public-readiness, positioning, documentation, apache-2, release, trust]
child:
  - hero-public-truth-baseline
  - hero-licensing-boundary-and-provenance
  - hero-positioning
  - hero-root-docs-remediation
  - hero-hosted-docs-remediation
  - hero-landing-message-refresh
  - hero-public-repo-readiness
  - hero-public-docs-drift-guard
  - hero-v034-release-prep
  - hero-apache-license-grant-gate
  - hero-public-visibility-launch-gate
relations:
  - target: hero-self-consistency
    kind: related
  - target: generated-command-refs-validated
    kind: related
---

# Hero v0.34 Public Release Readiness

## Vision

Hero v0.34 presents one accurate, supportable public product: the `hero` CLI and this repository explain what is shipped, install and operate as documented, expose current hosted docs and landing pages, carry the minimum collaboration and security surfaces expected of a public repository, and are ready for an explicitly approved Apache-2.0 grant and public visibility change.

The public boundary is narrow. The `hero` repository is the candidate Apache-2.0 work. Sprout is a separately owned, public MIT-licensed dependency. `hero-code` and `hero-cloud` remain proprietary and must not be relicensed, copied into this repository, or represented as part of Hero's open-source grant.

## Goal

Deliver the complete v0.34 public-readiness path: an inventory of the shipped product, memory-first positioning, current root and hosted documentation, a current public site, repository-readiness materials, drift guards, and a clean release candidate. The owner has authorized preparing this repository for an Apache-2.0 grant. Adding the license, changing repository visibility, and publishing the release remain explicit final mutations rather than prep work.

## Current reality

- The latest release tag is `v0.33.0`; this initiative targets `v0.34.0`.
- The repository has no `LICENSE`. Hero's sole owner has stated that he owns the repository content and authorizes preparing it for an Apache-2.0 grant; the remaining license review concerns third-party dependencies and bundled assets, not contributor consent.
- Only the `hero` CLI/repository is in this Apache-2.0 licensing scope. Sprout is a separately owned public MIT project. `hero-code` and `hero-cloud` remain proprietary by product choice.
- The deployed docs are stale as of June 11 even though newer pages exist in source. A green local build does not prove the public site is current.
- `heroengine.ai` does not currently resolve. Domain and deployment restoration are release blockers.
- Documentation dependencies are loosely bounded and the build warns about future incompatibility with MkDocs 2.0.
- Public source links are dead while the repository is private. They must remain hidden or clearly unavailable until the visibility gate succeeds.

The durable claim audit and evidence taxonomy live in [`content-truth-audit.md`](content-truth-audit.md).

## Product direction

### Core message

> Hero carries your project's intent, decisions, and evidence from one AI session to the next, so you spend less time re-explaining and supervising.

- **Category:** project memory and delivery system for AI-assisted engineering.
- **Primary promise:** durable project memory. Decisions, context, corrections, conventions, evidence, and current state survive sessions, tools, and agents.
- **Execution system:** specs and specialized agents turn that memory into structured work, implementation, cold audit, and verified completion.
- **Outcome:** every AI session can inherit the project, act with context, and leave the project smarter for the next session.
- **Wedge:** engineering leads; PM and Sales expansion must use verified maturity labels.
- **Candidate, not promise:** “Correct your AI once” remains prohibited as an absolute statement unless it is independently validated as a carefully bounded claim. That validation is not a v0.34 release dependency.

Hero is two connected systems in one: project memory is the differentiated headline, and the spec-and-agent delivery loop is the execution layer built on it. Each delivery records more useful memory; that memory improves every later delivery. Specs remain a forcing function for trustworthy work, not the headline category. Public copy must not make Hero read like another spec kit.

## Child specs and phases

| Phase | Child | Priority | Size | Depends on | Outcome |
|---:|---|---|---|---|---|
| 1 | [`hero-public-truth-baseline`](hero-public-truth-baseline/spec.md) | critical | medium | — | Shipped-surface inventory, maturity labels, P0 corrections, and current public-state baseline. |
| 1 | [`hero-licensing-boundary-and-provenance`](hero-licensing-boundary-and-provenance/spec.md) | critical | small | baseline | Owner authorization, exact repository boundary, and third-party dependency, asset, and notice inventory. |
| 2 | [`hero-positioning`](hero-positioning/spec.md) | high | medium | shipped inventory + licensing inventory | Approved message hierarchy and precise public/proprietary boundary language. |
| 2 | [`hero-root-docs-remediation`](hero-root-docs-remediation/spec.md) | high | medium | baseline + positioning | Executable root onboarding and accurate product, repository, and licensing-boundary guidance. |
| 2 | [`hero-hosted-docs-remediation`](hero-hosted-docs-remediation/spec.md) | high | large | baseline + positioning | Factual page repair, capability discoverability, bounded docs dependencies, and deployment restoration. |
| 3 | [`hero-landing-message-refresh`](hero-landing-message-refresh/spec.md) | high | medium | root docs + hosted docs | Current truthful landing source, working destinations, and restored `heroengine.ai` DNS/deployment. |
| 3 | [`hero-public-repo-readiness`](hero-public-repo-readiness/spec.md) | high | medium | landing + licensing inventory | Public-exposure audit plus minimum contribution, security, conduct, issue, and support surfaces. |
| 4 | [`hero-public-docs-drift-guard`](hero-public-docs-drift-guard/spec.md) | medium | medium | root + hosted + landing + repo readiness | Derived assertions, pinned docs compatibility, deployment freshness, and production crawls. |
| 4 | [`hero-v034-release-prep`](hero-v034-release-prep/spec.md) | high | medium | drift guard | Reconciled v0.34 notes, artifacts, SBOM/notices, checksums, install proof, and launch checklist without publication. |
| 5 | [`hero-apache-license-grant-gate`](hero-apache-license-grant-gate/spec.md) | critical | small | release prep + licensing inventory + repo readiness | Explicitly approved Apache-2.0 grant for this repository only. |
| 5 | [`hero-public-visibility-launch-gate`](hero-public-visibility-launch-gate/spec.md) | critical | medium | Apache grant + release prep | Explicitly approved visibility flip, anonymous verification, and v0.34 launch/publication. |

The hard dependency chain is:

```text
hero-public-truth-baseline
  → hero-licensing-boundary-and-provenance
    → hero-positioning
        → {hero-root-docs-remediation, hero-hosted-docs-remediation}
          → hero-landing-message-refresh
            → hero-public-repo-readiness
              → hero-public-docs-drift-guard
                → hero-v034-release-prep
                  → hero-apache-license-grant-gate
                    → hero-public-visibility-launch-gate
```

Root and hosted documentation may proceed in parallel after positioning. All other arrows are hard order. Phase 5 contains human-controlled mutations and cannot be entered merely because its dependencies are green.

## In-flight overlap watch

Every seam below has reciprocal `conflicts-with` relations on the named children. Do not deliver each pair concurrently without first partitioning the listed assets.

| Child A | Child B | Shared seam |
|---|---|---|
| `hero-public-truth-baseline` | `hero-public-docs-drift-guard` | claim registry, derived counts/version authority, and freshness contract |
| `hero-root-docs-remediation` | `hero-public-docs-drift-guard` | root markdown scans, quickstarts, counts, and command assertions |
| `hero-hosted-docs-remediation` | `hero-public-docs-drift-guard` | `web/docs/src` scanning, navigation, releases, and generated references |
| `hero-hosted-docs-remediation` | `hero-landing-message-refresh` | `web/` metadata, terminology, navigation links, and destination URLs |
| `hero-root-docs-remediation` | `hero-public-repo-readiness` | root README support/contribution links and public-repository guidance |

## Existing spec disposition

These dispositions preserve genealogy without mutating out-of-scope historical specs during composition.

| Existing slug | Disposition |
|---|---|
| `hero-positioning` | Preserved under this initiative; it consumes the explicit Hero/Sprout/proprietary repository boundary established by the licensing-inventory child. |
| `hero-landing-page` | Superseded by `hero-landing-message-refresh`; the stale v0.9 scope is closed. |
| `hero-docs-site` | Superseded by `hero-hosted-docs-remediation`, which owns source remediation and deployed-site restoration. |
| `hero-distribution` | Superseded by `hero-v034-release-prep`; broad distribution work is not revived. |
| `hero-demo-content` | Deferred outside this release initiative as an optional, clearly illustrative interactive-install terminal animation. |
| `hero-launch-playbook` | Broad campaign work remains deferred. The final gate performs only the approved repository/release launch and verification. |
| `hero-content-engine` | Remains outside this initiative; ongoing publishing is a separate growth program. |
| `hero-community` | Broad community growth remains deferred; the minimum public repository policies and issue/support surfaces belong to `hero-public-repo-readiness`. |
| `hero-telemetry` | Remains outside this initiative; analytics and privacy policy require a separate product decision. |

The four deferred growth specs no longer declare `hero-marketing` as parent. The four replaced specs carry explicit supersession edges to their v0.34 successors.

## Cross-cutting requirements

- Public behavior, version, architecture, availability, and licensing claims require executable or source evidence, not repetition across docs.
- The legal boundary must remain exact: Apache-2.0 can cover this `hero` repository only after approval; Sprout remains separately MIT-licensed; `hero-code` and `hero-cloud` remain proprietary.
- Repository scans must include source, generated artifacts, dependencies, vendored material, docs assets, fonts, screenshots, and logos. Owner authorization is recorded; third-party material remains subject to license review.
- Public URLs are part of the product contract. Source links, docs, DNS, anonymous clone, release artifacts, and install commands must be verified from an unauthenticated environment.
- License and public exposure gates are fail-closed. Missing approval, unresolved third-party licensing, missing license text, non-resolving DNS, stale deployment, or dead anonymous links halt the gate.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL provide one evidence-backed public claim registry with `shipped`, `optional`, `preview`, and `planned` availability labels.
- **AC-2:** THE SYSTEM SHALL record the owner's authorization and identify every third-party dependency, bundled asset, and repository included in or excluded from the proposed Apache-2.0 grant.
- **AC-3:** WHEN a user follows a public onboarding path THE SYSTEM SHALL execute the documented commands and configuration against the v0.34 product contract.
- **AC-4:** WHEN Hero is described on root docs, hosted docs, or the landing surface THE SYSTEM SHALL lead with durable project memory, explain the spec-and-agent delivery system as the connected execution layer, and state the exact public/proprietary repository boundary.
- **AC-5:** IF content suggests `hero-code`, `hero-cloud`, or another proprietary repository is Apache-licensed or included in the public grant THEN THE SYSTEM SHALL fail public-readiness validation.
- **AC-6:** WHEN the hosted docs or landing source changes THE SYSTEM SHALL verify deployed revision parity, resolvable DNS, live destinations, and compatibility-bounded documentation dependencies.
- **AC-7:** WHEN the v0.34 release candidate is prepared THE SYSTEM SHALL provide reproducible artifacts, checksums, SBOM/notices, install evidence, release notes, and a launch checklist without creating a public release.
- **AC-8:** IF explicit approval to add Apache-2.0 has not been recorded THEN THE SYSTEM SHALL leave this repository without a license mutation.
- **AC-9:** IF explicit approval to make the repository public has not been recorded THEN THE SYSTEM SHALL leave repository visibility unchanged and SHALL NOT publish the v0.34 launch.
- **AC-10:** WHEN the final launch is approved THE SYSTEM SHALL verify anonymous clone, install, docs, landing, source links, issue/support surfaces, and v0.34 artifacts from an unauthenticated environment.

## Boundaries

- Do not license, publish, copy, or imply an Apache-2.0 grant for `hero-code` or `hero-cloud`; both remain proprietary.
- Do not modify or relicense Sprout from this repository. Its MIT grant remains separate; only a licensed Sprout module release and Hero dependency bump are in scope as an external prerequisite.
- Do not add `LICENSE`, change repository visibility, publish a GitHub release/tag, or announce launch without the explicit approval required by the corresponding final gate.
- No launch campaign, Product Hunt/HN calendar, newsletter, social cadence, analytics backend, pricing, enterprise collateral, or broad community-growth program.
- No broad visual brand overhaul; only changes needed for a coherent, truthful, accessible public surface.
- No changing product behavior merely to preserve stale docs. Documentation follows the supported contract.
- No history rewrite or contributor-rights investigation. The owner's authorization is authoritative for Hero-owned content; unresolved third-party licensing remains a blocker to surface.

## Verification

- Lint and score the initiative and every child; confirm all parent/dependency/related edges resolve and every named overlap has reciprocal `conflicts-with` relations.
- Reconcile each public claim, dependency, and bundled asset before entering either final gate.
- Require strict source builds, bounded docs dependencies, executable examples, artifact/SBOM checks, link/accessibility checks, deployment revision verification, DNS resolution, and a production crawl.
- Exercise both final gates in fail-closed mode before seeking approval, then record the approving human and exact authorized mutation when each gate runs.

## Risks and open decisions

- **Owner authorization:** the sole owner has authorized Apache-2.0 preparation for this repository; record that decision plainly and do not reopen contributor-rights analysis.
- **Third-party material:** source dependencies may be compatible while bundled docs/media/font assets are not; asset licensing must be reviewed separately.
- **Deployment ownership:** source contains newer docs than the June 11 live deployment, so the deploy trigger, hosting account, and revision marker must be proven.
- **DNS:** `heroengine.ai` currently does not resolve; registrar/hosting access may require explicit external coordination.
- **Dependency drift:** loose documentation bounds and the MkDocs 2.0 warning can turn a future rebuild into a breaking deploy unless resolved before v0.34.
- **Sprout release metadata:** Sprout is a separate public MIT project; verify the exact version consumed by Hero carries the expected license metadata during third-party inventory.
- **Exposure:** making a repository public reveals the full reachable history, issues/settings permitted by the host, and all tracked assets; scan results must be resolved before approval, not after.

## Progress

- 2026-04-25 — Original broad positioning/distribution/launch initiative created.
- 2026-08-04 — Recomposed into seven evidence-led public-truth children.
- 2026-08-21 — Expanded in place into the v0.34 public-readiness plan with bounded licensing, repository, release, Apache grant, and visibility gates; no repository license, visibility, public site, docs, code, or release state was changed.
- 2026-08-23 — Owner authorization recorded, autonomous preparation armed, and public positioning sharpened to lead with durable project memory plus its connected spec-and-agent delivery system.
- 2026-08-23 — Removed the cross-tool continuity demonstration from the initiative and release critical path. Optional synthetic install animation work remains separate and cannot block v0.34.
