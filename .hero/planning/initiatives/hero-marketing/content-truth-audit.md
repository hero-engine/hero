# Hero Public Content Truth Audit and Delivery Plan

Date: 2026-08-04  
Initiative: `hero-marketing`  
Scope: public repository/docs, deployed docs and landing, product-output examples, Apache-2.0 readiness, v0.34 release readiness, and automated public-truth checks

## Executive finding

Hero's public story is simultaneously too small and not reliable enough. It over-indexes on specs and roster scale while under-explaining the shipped system that reduces supervision across AI sessions. Worse, several onboarding instructions and architectural claims are false even though the current structural checks pass.

Recommended core:

> Hero carries your project's intent, decisions, and evidence from one AI session to the next, so you spend less time re-explaining and supervising.

- Category: operating layer for AI-assisted engineering.
- Outcome: AI sessions inherit the project and finish against evidence.
- Specs: mechanism and forcing function, not the headline category.
- “Correct your AI once”: candidate claim requiring a repeatable proof, not an unconditional promise.

## Evidence classes

| Class | Authority | Suitable use |
|---|---|---|
| A | Executable implementation contract, runtime registry/output, decoding test, CLI help, or passing isolated exercise | Supports `shipped` claims and executable examples. |
| B | Delivered/verified spec evidence, production capture tied to a revision, or repeatable product demo | Supports outcome/proof claims when prerequisites are stated. |
| C | Public or repository documentation without independent executable confirmation | Claim under review; never sole authority for product behavior. |
| D | Planning spec, private/sibling work, inference, or unresolved stakeholder statement | `preview`/`planned`/unknown only; cannot support `shipped`. |

Availability labels:

- `shipped`: present in the public product and supported by Class A/B evidence.
- `optional`: shipped but requires explicit setup, integration, provider, or feature enablement.
- `preview`: usable in a bounded form, but readiness/support limits must be stated.
- `planned`: design or implementation intent, not publicly available behavior.

## P0 falsehoods requiring correction

| Public claim/instruction | Finding | Correction authority |
|---|---|---|
| `hero install satellites list` / `hero install satellites add` | Commands do not exist; positional words enter the interactive workflow rather than performing the documented operation. | Current Cobra command tree and `hero ... --help` (A). |
| bare `hero install --repair` | Invalid because install requires the `project` or `global` target. | CLI help/error contract (A). |
| nested `.hero` workspaces for monorepo subtrees | Current architecture is one root `.hero` corpus with thin satellite harness trees; nested `hero init` instructions are dangerous. | `internal/cli/install_satellites.go` and isolated install exercise (A). |
| `hero spec verify` followed by `hero spec complete` | Verify already runs closing gates, completes, and archives; normal docs must not prescribe a second completion step. | `internal/cli/verify.go` and exercised delivery close (A/B). |
| configuration “Full Example” | Multiple field shapes/values cannot decode under the current model. | `internal/config/config.go` plus a production-decoder fixture (A). |
| Go 1.21+ prerequisite | `go.mod` requires Go 1.26.4. | `go.mod` (A). |
| `hero verify-install` | Root command does not exist; intent maps to `hero doctor` or `hero check`. | CLI tree/help (A). |

The baseline child must turn every P0 row into a correction packet naming affected surfaces, authoritative replacement, owner child, and executable validation. Root/hosted remediation then changes public content.

## P1 drift and misleading claims

- Landing source/live story says v0.9 while `v0.33.0` is the latest tag and v0.34.0 is the target release.
- Hosted release notes stop at v0.24.1.
- The deployed docs are stale as of June 11 while newer pages exist in source; current green source builds do not prove deployment freshness.
- `heroengine.ai` does not resolve, and public source links are dead while the repository remains private.
- Documentation dependencies are loosely bounded and warn about future incompatibility with MkDocs 2.0.
- Canonical observed inventory is 29 workflow commands, 35 agents, 57 skills, and 82 runtime MCP tools, yet root and hosted pages disagree. Mutable counts should appear only in generated reference surfaces.
- Landing presents fictional `hero status` output with a nonexistent `diagnosing` status as if it were captured product output.
- “Slash commands inside your AI tool” is wrong for Codex, which consumes Hero workflows as skills. Harness copy must describe native surfaces for all seven supported install targets.
- Root/docs repository trees still show a local `cloud/` tree that moved out of this repository.
- Peering copy implies one cross-repo graph or synchronous peer execution; current semantics are one graph per project plus manifests, asynchronous Project Mail, advisory/spec-out calls, and explicit handoff/promotion.
- PM and Sales are real domain packs but lack sufficient release-level evidence for blanket production-readiness claims.

## Structural validation gap

The following checks pass despite the false content above:

- `mkdocs build --strict`
- `hero docs check`
- `hero docs check --invocations`

They prove structure and selected shallow invariants, not product truth. The drift-guard child must extend validation across hosted docs, landing metadata/output, required arguments, nested command semantics, config decoding, MCP inventory, deployment revision, and production crawl results.

## Under-marketed shipped or meaningful abilities

Every capability below still needs claim-level prerequisites and maturity evidence before publication:

- Cold delivery audit plus `hero spec verify` evidence gates.
- Continuity of intent, corrections, decisions, failures, and next actions across sessions and tools.
- Attention, Mail, Focus, and bounded suggestion/action semantics.
- `hero serve` project intelligence UI: Now, Rollup, Work, Knowledge, Agents, People, chat, and command palette surfaces.
- Twenty-one guarded code-host MCP operations, subject to provider/setup and public-readiness confirmation.
- Tracker evidence loading and bounded mutations.
- Cross-repo Project Mail, advisory/spec-out calls, and explicit handoff/receiver promotion.
- Headless, approval-aware runtime behavior, with availability boundaries.
- PM and Sales domain packs, maturity-labeled and secondary to the engineering wedge.

## Claim registry seed

The baseline child expands this into an exhaustive registry covering every public sentence that asserts behavior, availability, architecture, compatibility, count, version, or prerequisite.

| Claim family | Initial state | Evidence needed before publication |
|---|---|---|
| Cross-session/project continuity | shipped, scope to prove | Repeatable two-tool cold-resume demo plus artifact provenance. |
| Audit + verify close | shipped | Actual gate output and verified spec archive evidence. |
| Attention/Mail/Focus | shipped/optional by surface | Runtime tool contract, CLI/MCP exercise, and setup needs. |
| `hero serve` project intelligence | shipped | Real screenshot/output tied to revision and supported startup path. |
| Code-host broker operations | optional; public maturity open | Registry-derived inventory, provider matrix, credential/approval boundaries, exercised operations. |
| Tracker evidence/mutations | optional | Supported adapter matrix and exercised read/write contracts. |
| Cross-repo peering | shipped with setup | One-graph-per-project architecture and Mail/handoff exercise. |
| Headless runtime | maturity open | Supported execution path, approval behavior, and environment prerequisites. |
| PM and Sales packs | maturity open | Release-level smoke journey for each pack. |
| Cloud/team/outposts | unknown/preview/planned until resolved | Public access path, ownership, support level, and sibling-repo evidence. |
| Hero licensing/open-source posture | Apache-2.0 candidate; publication prohibited pending gate | Sole-owner authorization, third-party dependency/asset inventory, explicit final mutation approval, and repository `LICENSE`. |
| Sprout licensing | cleared as a separate MIT dependency | Hero now pins `github.com/bdwheeler/sprout` at immutable licensed revision `v0.1.1-0.20260822024445-cd3f0c4a2208`; the exact module archive includes the MIT license and is content-identical to `v0.1.0` outside licensing metadata. See `hero-licensing-boundary-and-provenance`. |
| `hero-code` / `hero-cloud` | proprietary | Boundary validation proving neither repository is included in Hero's grant or public claims. |

## Delivery scope

1. Establish the exhaustive public-truth baseline and P0 correction packet.
2. Record sole-owner authorization and inventory the exact Apache boundary, dependencies, assets, and notices.
3. Rewrite positioning around supervision reduction, continuity, trust, and the public/proprietary boundary.
4. Repair root and hosted docs, restore deployed-doc revision parity, and bound docs dependencies.
5. Refresh the landing message, restore `heroengine.ai`, and gate source links while the repository is private.
6. Complete the public-repository exposure audit and minimum contribution/security/conduct/issue/support surfaces.
7. Produce repeatable continuity proof and install source/deployment/production drift guards.
8. Prepare v0.34 artifacts, notes, SBOM/notices, checksums, and install evidence without publication.
9. Run separately approved Apache grant and repository visibility/v0.34 launch gates.

## Deferred

- Launch campaigns, Product Hunt/HN plans, newsletter/social cadence, and broad community growth beyond minimum public-repository policies and support/issue surfaces.
- Telemetry/analytics backend and privacy policy.
- Pricing, enterprise collateral, broad competitor pages, and domain-platform marketing.
- Broad visual brand overhaul.
- Any unconditional cloud/team/outposts or code-host-readiness promise lacking evidence.

## Open questions

1. Which account/workflow owns docs and landing deployment, and what restores `heroengine.ai` DNS?
2. Which cloud, team, outpost, domain-pack, and code-host capabilities are publicly supportable at v0.34?
3. When the readiness packets are green, will the owner explicitly approve the Apache grant and, separately, public visibility/v0.34 publication?

Resolved in the initiative:

- The owner authorized preparation of the sole-owned Hero repository for an Apache-2.0 grant. The exact repository, proprietary-product, Sprout, dependency, asset, and notice boundaries are recorded by `hero-licensing-boundary-and-provenance`; the grant itself remains a later explicit gate.
- AI-native engineers and hands-on technical leads in long-lived codebases are the lead audience. Engineering leads, platform teams, and multi-repository maintainers are secondary; focused PM, QA, and Sales setups are maturity-labeled expansion paths.
- Mutable roster counts stay out of positioning and narrative pages. If needed at all, they belong only on generated reference surfaces.

## Completion evidence expected

- Exhaustive claim registry with evidence class, availability label, affected surfaces, owner, and last verification.
- Executed clean-workspace quickstarts and production-decoder config fixture.
- Real or explicitly illustrative product output tied to a revision.
- Derived command/agent/skill/MCP inventories where counts remain necessary.
- Root, hosted-docs, and landing scans plus link/build/accessibility checks.
- Production crawl tied to deployed revision with zero unresolved P0/P1 claims.
- Cleared owner-authorization/dependency/asset packet and exact Hero/Sprout/proprietary boundary.
- v0.34 release candidate with reproducible artifacts, checksums, SBOM/notices, and clean-install evidence.
- Recorded explicit approvals for the Apache grant and later public visibility/launch mutations.
