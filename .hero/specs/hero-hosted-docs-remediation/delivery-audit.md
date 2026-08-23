# Delivery audit — hero-hosted-docs-remediation

**Audited:** `git diff HEAD -- .github/workflows/docs.yml requirements-docs.txt web/docs .hero/planning/initiatives/hero-marketing/hero-hosted-docs-remediation/spec.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Resolve every hosted-doc P0/P1 claim — `web/docs/src/about/truth-reconciliation.md:7` accounts for all 13 registry rows owned by this child; affected pages remove the obsolete satellite, topology, closing, configuration, inventory, harness, peering, and domain-readiness forms.
- [✓] Validate documented commands and configuration — `web/docs/scripts/test_docs_build.py:187` compares the published JSON block with the production-decoder fixture; the provided evidence reports the focused decoder, satellite, verify, Markdown-invocation, and MCP registry tests passed.
- [✓] State capability availability, prerequisites, action boundaries, and evidence — `web/docs/src/reference/capability-status.md:11` supplies all four fields for the required capability families.
- [✓] Make differentiated capabilities discoverable — `web/docs/mkdocs.yml:69` exposes Project Memory, Verified Delivery, continuity, verification, Attention/Mail/Focus, Serve, peering, code-host, and tracker paths; the strict build reported no omitted navigation pages.
- [✓] Derive release/version and identify revision/freshness — `.github/workflows/docs.yml:47` regenerates releases and `:53` generates build metadata; `web/docs/scripts/docs_build.py:47` derives release, revision, and timestamp, while tracked surfaces use explicit non-identity placeholders. The workflow-order regression at `web/docs/scripts/test_docs_build.py:49` enforces tests before metadata before build.
- [✓] Build strictly with valid navigation, links, anchors, and reconciled claims — the workflow-equivalent sequence passed 28 unit tests, derived 61 releases with `v0.33.0` first, generated exact metadata, built strictly, sanitized, syntax-checked JavaScript, and checked internal/external links. `web/docs/scripts/test_docs_build.py:153` proves definitive 404/410/5xx responses fail, and `:170` tests the disclosed 401/403/429 boundary.
- [✓] Use bounded dependencies and a revision-verifiable deployment path — `requirements-docs.txt:3` pins MkDocs/Material; `.github/workflows/docs.yml:43` now tests placeholders before metadata generation at `:53`, builds at `:59`, validates the artifact, uploads a revision-named artifact, and gates Cloudflare Pages deployment with `${{ github.sha }}` at `:88`.
- [✓] Distinguish and link memory from spec-and-agent delivery — `web/docs/src/index.md:8`, `web/docs/src/concepts/knowledge-base.md:1`, `web/docs/src/concepts/continuity.md:32`, and `web/docs/src/concepts/core-loop.md:1` present separate linked paths and bound the reinforcing loop as preview.

## Changes

- [✓] Audit and repair architecture, configuration, closing gates, harnesses, peering, layout, inventory, and packs — concrete corrections appear across `web/docs/src/getting-started/project-setup.md`, `web/docs/src/project-structure.md`, `web/docs/src/cli/spec-management.md`, `web/docs/src/concepts/cross-repo.md`, and `web/docs/src/concepts/agents-and-skills.md`.
- [✓] Derive release framing and reconcile Focus/Mail/Releases navigation — release generation is wired at `.github/workflows/docs.yml:47`; navigation is present at `web/docs/mkdocs.yml:69`.
- [✓] Add evidence-backed memory and delivery capability coverage — new capability status, continuity, Attention, and code-host pages exist and are linked by the two primary navigation paths.
- [✓] Rewrite server/MCP and CLI references from implementation — server/MCP, tool metadata, overview, tracker, peering, and spec references use current capability groups and explicit setup/consent boundaries; focused registry and invocation tests passed.
- [✓] Bound docs dependencies, repair gated deployment, and expose revision marker — exact pins, placeholder-safe workflow ordering, build-time metadata, sanitizer verification, JavaScript syntax checking, Cloudflare Pages gating, and `/revision.json` are present; the workflow-equivalent sequence passed from restored placeholders.
- [✓] Align terminology, links, metadata, availability, and repository boundary — `web/docs/src/index.md:68`, `web/docs/src/project-structure.md:27`, and `web/docs/src/reference/capability-status.md:3` state the bounded Hero/Sprout/proprietary and maturity contracts; link checks passed.

## Audit notes

- None.
