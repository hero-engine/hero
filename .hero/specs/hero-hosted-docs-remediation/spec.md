---
title: "Hero Hosted Documentation Truth and Capability Refresh"
slug: hero-hosted-docs-remediation
type: feature
status: completed
domain: engineering
size: large
priority: high
horizon: now
created: 2026-08-04
tags: [documentation, website, information-architecture, truth]
parent: hero-marketing
depends-on: [hero-public-truth-baseline, hero-positioning]
relations:
  - target: hero-public-docs-drift-guard
    kind: conflicts-with
  - target: hero-landing-message-refresh
    kind: conflicts-with
supersedes: [hero-docs-site]
delivery_method: manual
completed_at: 2026-08-23T17:23:06Z
---

# Hero Hosted Documentation Truth and Capability Refresh

## Goal

Audit every hosted documentation page against implementation evidence, correct false or stale content, organize the product around durable project memory plus its connected delivery system, bound documentation dependencies, and prepare a revision-verifiable deployment path that can replace the stale June 11 live site at the later visibility/launch gate.

## Kickoff

Prepares a structurally green but factually drifting docs site to become the authoritative product reference after the separate visibility/launch gate deploys and crawls it.

**Status:** delivering — hosted source remediation and validation are complete; the mandatory independent cold delivery audit is next.

**Pick up at:** cold-audit the on-disk spec, diff, Completion Ledger, and validation evidence. Do not deploy before the visibility/launch gate; live parity remains unproven until that later deploy and production crawl.

→ `.hero/planning/initiatives/hero-marketing/hero-hosted-docs-remediation/spec.md`

**Files:** `web/docs/src/`, `web/docs/mkdocs.yml`, documentation generation scripts and navigation
**Skip:** landing-page copy and broad visual rebranding.

## Changes

1. Perform a page-by-page claim audit; repair satellite architecture, configuration, closing gates, harness behavior, peering, repository layout, versions, counts, and domain-pack maturity.
2. Make release/current-version framing derived and current; reconcile locally present Focus/Mail/Releases coverage with deployed navigation.
3. Add discoverable, evidence-backed coverage for continuity, cold audit/verify, Attention/Mail/Focus, `hero serve`, guarded code-host operations, tracker evidence, peering/handoffs, and approval-aware runtime behavior.
   - Create a clear project-memory path covering capture, structure, retrieval, continuity, corrections, decisions, evidence, and cross-session/tool use.
   - Create a distinct delivery-system path covering design, specs, agents, implementation, cold audit, verification, and knowledge capture.
   - Explain how the two paths reinforce one another without positioning Hero as a generic spec kit.
4. Rewrite server/MCP and CLI references from current registries and actual subcommands; state provider/setup/approval prerequisites.
5. Bound MkDocs/theme/plugin dependencies away from the warned incompatible MkDocs 2.0 transition, repair the gated Cloudflare deployment automation, and generate a revision marker that the later visibility/launch gate can use to prove source/live parity.
6. Align terminology, links, metadata, availability labels, and the Hero/Sprout/proprietary repository boundary with positioning and the landing refresh.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL resolve every hosted-doc P0/P1 claim in the public claim registry.
- **AC-2:** WHEN hosted docs describe a command or configuration THE SYSTEM SHALL validate it against the current command tree or production decoder.
- **AC-3:** WHEN a differentiated capability is documented THE SYSTEM SHALL state availability, prerequisites, safe action boundaries, and evidence.
- **AC-4:** THE SYSTEM SHALL provide discoverable navigation for continuity, verification, Attention/Mail/Focus, `hero serve`, peering, code-host, and tracker capabilities.
- **AC-5:** WHEN release or version information appears THE SYSTEM SHALL derive it from the agreed public release authority and identify source revision/freshness.
- **AC-6:** THE SYSTEM SHALL build the docs in strict mode with valid links, navigation, anchors, and no unresolved assigned claims.
- **AC-7:** WHEN hosted docs deploy THE SYSTEM SHALL use compatibility-bounded dependencies and expose a revision proving the newer source, not the stale June 11 build, is live.
- **AC-8:** THE SYSTEM SHALL provide distinct, linked navigation for project memory and the spec-and-agent delivery system, with memory presented as Hero's primary differentiated value.

## Boundaries

- No claims based only on planning specs or sibling/private implementation.
- No broad tutorial/video library, landing rewrite, or competitor campaign.
- If the work becomes x-large, split factual remediation before capability discoverability.
- No license or visibility mutation and no open-source claim for `hero-code` or `hero-cloud`.
- No live documentation deployment; the visibility/launch gate owns publication and the production crawl.

## Validation

- Strict MkDocs build, internal/external link and anchor checks, executable example/config tests, registry reconciliation, Hero lint/score, and index refresh.

## Completion Ledger

Implemented the hosted documentation truth refresh as a memory-first and
evidence-bounded reference, plus a revision-verifiable Cloudflare Pages build
path. The documentation stack is MkDocs/Material with Python standard-library
build helpers; `documentation-practices`, `implementation-principles`,
`python-stack`, `testing-and-validation`, and the delivery/ledger instructions
were applied.

Validation performed: 28 documentation-tool unit tests, including an explicit
workflow-order assertion that placeholder tests precede metadata generation and
metadata generation precedes the strict build; strict MkDocs build;
internal and external link/anchor checks (definitive 404, 410, and 5xx responses
fail; authentication, authorization, and rate-limit responses remain bounded);
generated search-bundle and source-map sanitation with a post-build dangling-
reference check and Node.js syntax validation; decoder-backed public-config test;
satellite install tests;
verify/Markdown CLI drift tests; MCP `tools/list` tests; release-feed derivation;
YAML and Wrangler configuration checks; claim reconciliation; Hero spec lint,
score, and index refresh; and diff whitespace review. Live parity is deliberately
unproven: this child did not deploy. The later visibility/launch gate must run
the gated Cloudflare deployment and compare production `/revision.json` with the
approved source revision.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Resolve every hosted-doc P0/P1 claim | DONE | `web/docs/src/about/truth-reconciliation.md` accounts for every hosted-doc-owned registry row; `getting-started/project-setup.md`, `project-structure.md`, `cli/spec-management.md`, and `configuration/hero-json.md` correct the hosted P0 satellite, topology, closing, and decoder claims. Obsolete-form reconciliation returned `Hosted claim reconciliation OK`. |
| 2 | Validate commands and configuration against current implementation | DONE | `web/docs/scripts/test_docs_build.py` proves the published full JSON example equals `internal/config/testdata/public-hero.json`; the production decoder test, Markdown CLI invocation test, satellite tests, verify tests, and MCP registry tests passed. |
| 3 | State capability availability, prerequisites, safe boundaries, and evidence | DONE | `web/docs/src/reference/capability-status.md` gives those four fields for memory, delivery, Attention/Mail/Focus, Serve, trackers, code hosts, peering, and headless execution. |
| 4 | Make required differentiated capabilities discoverable | DONE | `web/docs/mkdocs.yml` exposes distinct Project Memory and Verified Delivery paths plus continuity, verification, Attention/Mail/Focus, Serve, peering, code-host, and tracker navigation. Strict build reported no omitted navigation pages. |
| 5 | Derive release/version and identify source revision/freshness | DONE | `web/docs/scripts/generate_release_notes.py` derived 61 published releases with `v0.33.0` first. Tracked `src/about/build.md` and `src/revision.json` contain explicit build-time placeholders; the workflow tests those placeholders before `docs_build.py metadata` replaces them with the exact release, full revision, and generated timestamp in the ephemeral build workspace, then runs the strict build. |
| 6 | Strict build with valid links/navigation/anchors and reconciled claims | DONE | `mkdocs build --strict`, `docs_build.py check-js`, `docs_build.py check --external`, claim assertions, output route checks, YAML parsing, and `git diff --check` passed. The external checker rejects definitive 404, 410, and 5xx responses while tolerating bounded authentication, authorization, and rate-limit responses; the workflow re-runs it on every deploy-gated path. |
| 7 | Use bounded dependencies and expose a deployment revision | DONE | `requirements-docs.txt` pins MkDocs 1.6.1 and Material 9.7.7; `wrangler.toml` declares the Pages output; `.github/workflows/docs.yml` builds, sanitizes, uploads a revision-named artifact, and gates Cloudflare deploy with the commit hash. Live deployment/crawl remains owned by the visibility/launch gate. |
| 8 | Distinguish and link memory from spec-and-agent delivery | DONE | `src/index.md`, `concepts/knowledge-base.md`, `concepts/continuity.md`, and `concepts/core-loop.md` lead with memory, define delivery separately, and bound their reinforcing cross-tool loop as preview. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Audit and repair architecture, config, gates, harnesses, peering, layout, inventory, and packs | DONE | Rewrote or corrected the affected hosted pages; removed stale counts and version-led product proof; corrected one-root satellite topology, verify-only closing, native harness rendering, asynchronous peering, repository boundaries, and domain maturity. |
| 2 | Derive release framing and reconcile Focus/Mail/Releases navigation | DONE | Release notes remain generated from the release authority; build information derives current tag/revision; the Project Memory nav includes Attention, Mail, and Focus, and the build verifies their output routes. |
| 3 | Add evidence-backed memory and delivery capability coverage | DONE | Added `reference/capability-status.md`, `concepts/continuity.md`, `cli/attention.md`, and `cli/code-host.md`; reorganized navigation into distinct memory and delivery paths with mutual links. |
| 4 | Rewrite server/MCP and CLI references from implementation | DONE | `cli/server-and-mcp.md`, `serve/mcp-tool-metadata.md`, `configuration/mcp-setup.md`, `cli/overview.md`, tracker, peering, and spec references now use capability groups/current commands and explicit provider/setup/approval boundaries rather than mutable inventories or proprietary surfaces. |
| 5 | Bound docs dependencies, repair gated deploy, and expose revision marker | DONE | Exact direct pins avoid MkDocs 2.0 drift; tracked revision surfaces are non-authoritative placeholders and CI generates exact values before building; `docs_build.py sanitize` removes unused reciprocal language bundles and source maps, strips their generated references, and verifies the artifact contains neither files nor dangling references; `docs_build.py check-js` validates every generated JavaScript file after rewriting; attributions are published; Cloudflare Pages replaces the mismatched GitHub Pages command; deploy and live-link validation remain gated; source/live proof uses the generated `/revision.json`. |
| 6 | Align terminology, links, metadata, availability, and repository boundary | DONE | Site metadata now says project memory and verified delivery; narrative pages follow positioning vocabulary; links/anchors pass; Hero Code/Cloud remain proprietary and Sprout separate MIT; no premature open-source claim appears. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end from restored placeholders in workflow order: ran tooling tests first; generated release notes and exact test revision/release/timestamp values; built the complete site with `mkdocs build --strict`; sanitized the artifact; syntax-checked every generated JavaScript file with Node.js; checked internal and live external links/anchors; verified expected output routes and generated `/revision.json`; proved the built artifact contains neither unused Lunr language, Wordcut, TinySegmenter, or source-map files nor references to them; and restored and rechecked the tracked placeholders afterward.

### Excellence Bar self-check

Yes — the docs now lead with the actual differentiated product, keep every
optional/preview capability beside its prerequisites and action boundary,
derive mutable facts, fail closed on incompatible generated assets, and provide
the later launch gate a concrete revision proof without pretending a deploy
occurred.
