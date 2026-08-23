---
title: "Hero Landing Message and Proof Refresh"
slug: hero-landing-message-refresh
type: feature
status: completed
domain: engineering
size: medium
priority: high
horizon: now
created: 2026-08-04
tags: [landing-page, positioning, messaging, proof]
parent: hero-marketing
depends-on: [hero-root-docs-remediation, hero-hosted-docs-remediation]
relations:
  - target: hero-continuity-proof-demo
    kind: conflicts-with
  - target: hero-hosted-docs-remediation
    kind: conflicts-with
supersedes: [hero-landing-page]
delivery_method: manual
completed_at: 2026-08-23T19:26:36Z
---

# Hero Landing Message and Proof Refresh

## Goal

Replace the stale v0.9/spec-tool-scale landing story with a clear two-system message: durable project memory is the headline, while specs and specialized agents form the connected delivery system. Prepare a validated, revision-identifiable `heroengine.ai` artifact and gated deployment path for the later visibility/launch owner without deploying, changing DNS, or overstating the still-unproven continuity outcome.

## Kickoff

Refreshes the real landing source around durable project memory and evidence-gated delivery, then prepares a revision-verifiable artifact for the later public launch gate.

**Status:** in-review — refreshed source, social asset, revision-stamped artifact pipeline, and landing regressions pass locally; deployment and DNS remain untouched.

**Pick up at:** cold-audit the implementation and Completion Ledger, then run the verification gate; leave deployment, DNS, redirects, and anonymous source activation to the visibility/launch gate.

→ `.hero/planning/initiatives/hero-marketing/hero-landing-message-refresh/spec.md`

**Files:** `web/landing/site/index.html`, `web/landing/scripts/landing_build.py`, `.github/workflows/landing.yml`, `.hero/marketing/positioning.md`
**Skip:** broad brand redesign, campaign calendar, pricing, or unsupported roadmap promises.

## Changes

1. Lead with durable project memory across sessions, tools, and agents; immediately explain the spec-and-agent delivery system that uses that memory to implement, cold-audit, and verify work.
   - Show the reinforcing loop: memory informs delivery, verified delivery enriches memory, and the next session starts smarter.
   - Make the distinction understandable without prior knowledge and prevent the page from reading like another spec kit.
2. Remove hardcoded v0.9 and mutable roster-scale proof; stamp the built artifact with its exact source commit, deterministic source-tree digest, dirty/clean state, and generation time while leaving release facts to the public release authority.
3. Replace fictional `hero status` output with revision-tied evidence or clearly label any abbreviated workflow output illustrative.
4. Separate `available`, `optional`, `preview`, and `planned` capabilities; correct harness-native workflow, peering, cloud/team/outpost, domain, and code-host language.
5. Route each proof pillar to current hosted documentation and publish a visible, machine-readable source revision marker in the built artifact.
6. Prepare canonical metadata and a build-tested, manually approved Cloudflare deployment path that mirrors hosted-doc revision handling; keep public source links gated and assign the first deployment, DNS, redirects, and production parity proof to `hero-public-visibility-launch-gate`.
7. Add landing-specific regression coverage for messaging claims, availability qualifiers, HTML/accessibility structure, responsive behavior, links, assets, and build metadata.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL present Hero as two connected systems in one: durable project memory as the primary promise and a spec-and-agent delivery system as the execution layer that finishes against evidence.
- **AC-2:** WHEN a product outcome is claimed THE SYSTEM SHALL link it to real evidence or explicitly label the output illustrative.
- **AC-3:** THE SYSTEM SHALL NOT present hardcoded stale version/count claims, universal slash commands, one cross-repo graph, or unverified roadmap work as shipped.
- **AC-4:** WHEN a capability requires setup or is not generally available THE SYSTEM SHALL show the approved availability label and prerequisites.
- **AC-5:** THE SYSTEM SHALL link every proof pillar to an accurate docs destination and identify the exact source bytes of each built landing artifact.
- **AC-6:** THE SYSTEM SHALL pass local rendered-artifact HTML structure, responsive CSS, accessibility, asset, link, claim, build, and HTTP smoke checks; production DNS, redirect, browser accessibility, and live smoke proof SHALL remain owned by `hero-public-visibility-launch-gate`.
- **AC-7:** WHEN the landing page is ready for repository audit THE SYSTEM SHALL produce a validated, revision-identifiable artifact and a launch-gated deployment path; `hero-public-visibility-launch-gate` SHALL own enabling deployment, DNS/redirect verification, production revision parity, and anonymous source-link activation.
- **AC-8:** WHEN a new reader reviews the primary landing hierarchy THE SYSTEM SHALL understand Hero's memory distinction before encountering detailed workflow mechanics and SHALL NOT reasonably classify Hero as merely another spec kit.

## Boundaries

- No pricing, enterprise collateral, competitor pages, launch campaign, or broad visual identity replacement.
- No new product capability added solely to support copy.
- No license or repository visibility mutation, and no implication that `hero-code` or `hero-cloud` is open source.

## Validation

- Compare every landing assertion with the claim registry and positioning authority.
- Validate illustrative/evidence labels, exact source commit/digest markers, links, HTML structure, responsive CSS, accessibility signals, assets, claims, build output, local artifact HTTP responses, Hero lint/score, and index refresh.
- Leave production deployment, DNS/redirect checks, browser accessibility proof, and live smoke evidence to `hero-public-visibility-launch-gate`.

## Completion Ledger

Implementation kept the existing static HTML/Cloudflare architecture and dark blue brand identity. Supporting changes cover `web/landing/README.md`, `.github/workflows/landing.yml`, `web/landing/wrangler.toml`, `site/{hero-logo.svg,favicon.svg,og-image.png,robots.txt,sitemap.xml,revision.json}`, plus the shared canonical logo copies in `web/docs/src/assets/{logo.svg,favicon.svg}` and `internal/serve/shell/static/favicon.svg`. Fresh validation ran eight Python standard-library regressions, tracked-source validation, a dirty-worktree revision-stamped build, built-artifact validation, local HTTP exercise of the page/revision/logo/social/sitemap assets, workflow YAML parsing, Hero drift/score, and `git diff --check`. No deployment, DNS, visibility, license, verify, archive, or commit action was performed.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Present durable memory first and verified delivery second | DONE | `web/landing/site/index.html:317` leads with the canonical “Your project remembers. Your agents deliver.” tagline; `:356` establishes the two-system hierarchy before delivery mechanics. |
| 2 | Link outcomes to evidence or label output illustrative | DONE | `web/landing/site/index.html:329` — abbreviated CLI panel is explicitly marked illustrative; proof pillars link to current hosted docs. |
| 3 | Remove stale versions/counts/universal slash/graph/roadmap claims | DONE | `web/landing/scripts/landing_build.py:36` and `:254` — claim regression rejects stale release, license, fictional output, universal-workflow, and unsupported roadmap language; page uses one-graph-per-project boundaries. |
| 4 | Show availability and prerequisites | DONE | `web/landing/site/index.html:441` — shipped, optional, preview, and planned capabilities carry adjacent prerequisites; headless copy includes the no-pause/resume boundary at `:466`. |
| 5 | Link proof pillars and identify exact artifact source | DONE | `web/landing/site/index.html:19`, `web/landing/site/revision.json`, and `web/landing/scripts/landing_build.py` publish and cross-check commit, exact source-tree digest, dirty state, composite revision, and current docs destinations. |
| 6 | Pass local rendered-artifact checks; defer production proof | DONE | `web/landing/scripts/landing_build.py` validates HTML structure, focus/reduced-motion/responsive rules, tracked docs links, canonical logo/social assets, claims, exact source identity, and local artifact HTTP responses. No production, DNS, redirect, or browser-accessibility proof is claimed; those remain with the launch gate. |
| 7 | Produce artifact and launch-gated deployment path | DONE | `.github/workflows/landing.yml:43`, `:49`, `:53`, and `:67` build/validate/upload before a manual+approved deploy; `web/landing/wrangler.toml:5` serves only `dist/`. DNS and production launch remain explicitly assigned to the later gate. |
| 8 | Make memory distinction clear before workflow mechanics | DONE | `web/landing/site/index.html:317` states the memory promise in the first viewport; `:356` explicitly says Hero is not another spec kit before the delivery section at `:410`. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Lead with memory, then connected verified delivery | DONE | `web/landing/site/index.html:317-434` — new hero, two-system model, bounded reinforcing loop, and completion gates. |
| 2 | Remove v0.9/count proof and derive exact build identity | DONE | `web/landing/scripts/landing_build.py` records an exact 40-character commit plus SHA-256 source-tree digest; dirty local builds use `<commit>+worktree:<digest>`, while explicit CI identity rejects non-HEAD or dirty source. |
| 3 | Replace fictional output or label illustrative | DONE | `web/landing/site/index.html:329` — output is intentionally abbreviated and visibly labeled illustrative. |
| 4 | Separate shipped/optional/preview/planned capabilities | DONE | `web/landing/site/index.html:441-472` — maturity and prerequisites are adjacent, including corrected headless and peering boundaries. |
| 5 | Route proof to docs and publish revision marker | DONE | Proof links target tracked hosted-doc pages; `site/revision.json` and rendered HTML share the verified composite revision while machine metadata carries commit/digest/dirty fields. |
| 6 | Prepare canonical and launch-gated deployment path | DONE | `README.md`, workflow, Wrangler config, robots/sitemap, canonical logo/favicon copies, and social metadata are aligned; deployment still requires manual dispatch plus `LANDING_LAUNCH_APPROVED=true`, and source links remain gated. |
| 7 | Add landing-specific regression coverage | DONE | Eight tests cover structure/claims/assets/links, canonical brand hashes, clean and dirty identities, arbitrary explicit revision rejection, exact source-digest mismatch, and unresolved-placeholder failure. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: built `web/landing/dist` from the dirty delivery worktree with revision `b514e8083b16edbe0e1cf6d464d4362b15189ad9+worktree:2cd139d1f00f7ab8a2e21fbd9f36460faae0cb0735546a0e69dc9a4d83bbfc9c`, served it locally, and received HTTP 200 for `/`, `/revision.json`, `/hero-logo.svg`, `/og-image.png`, and `/sitemap.xml`; rendered HTML showed the canonical tagline, illustrative label, composite revision, and build timestamp. This is local artifact smoke, not production proof.

### Excellence Bar self-check

- [x] Yes — the landing now has a clear product hierarchy, bounded evidence language, accessible/responsive static implementation, and a fail-closed revision/deployment path without changing the established visual identity or crossing the launch gate.
