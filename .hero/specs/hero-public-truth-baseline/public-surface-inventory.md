# Public surface inventory

Verified: 2026-08-23 at revision `75ea3cb1`

This inventory closes the surface side of the claim registry: every current root guide, hosted documentation page, and landing artifact has a downstream owner and a registry disposition. Asset-only files are listed separately because they do not assert behavior.

## Root documentation

| Surface | Primary claim IDs | Owner | State |
|---|---|---|---|
| `README.md` | product-memory-system, product-delivery-system, product-two-system-loop, install-satellite-subcommands, install-bare-repair, verify-closes-delivery, go-prerequisite, verify-install-command, installed-command-count, installed-agent-count, installed-skill-count, runtime-mcp-tool-count, harness-native-workflows, repository-layout-cloud, cold-audit-and-verify, apache-license-status, proprietary-repository-boundary | `hero-root-docs-remediation` | P0/P1 correction required |
| `GETTING-STARTED.md` | install-satellite-subcommands, install-bare-repair, satellite-workspace-architecture, verify-closes-delivery, public-config-example, harness-native-workflows, supported-install-targets, engineering-default-pack, optional-domain-packs | `hero-root-docs-remediation` | P0/P1 correction required |
| `MCP-SETUP.md` | runtime-mcp-tool-count, harness-native-workflows, attention-mail-focus, hero-serve-intelligence, code-host-operations | `hero-root-docs-remediation` | audit and count removal required |
| `CROSS-REPO-PEERING.md` | cross-repo-peering, proprietary-repository-boundary | `hero-root-docs-remediation` | architecture wording audit required |
| `TEAM-SERVER.md` | headless-runtime, proprietary-repository-boundary | `hero-root-docs-remediation` | availability boundary required |

## Hosted documentation

| Surface | Primary claim IDs | Owner | State |
|---|---|---|---|
| `web/docs/src/index.md` | product-memory-system, product-two-system-loop, installed-command-count, installed-agent-count, installed-skill-count, runtime-mcp-tool-count, current-release-version | `hero-hosted-docs-remediation` | message and inventory correction required |
| `web/docs/src/what-is-hero.md` | product-memory-system, product-delivery-system, product-two-system-loop, harness-native-workflows, pluggable-spec-systems | `hero-hosted-docs-remediation` | memory-first rewrite required |
| `web/docs/src/why-hero.md` | product-memory-system, product-two-system-loop, installed-command-count, installed-agent-count, installed-skill-count, harness-native-workflows | `hero-hosted-docs-remediation` | memory-first rewrite and count removal required |
| `web/docs/src/project-structure.md` | satellite-workspace-architecture, verify-closes-delivery, repository-layout-cloud | `hero-hosted-docs-remediation` | P0/P1 correction required |
| `web/docs/src/agents/index.md` | installed-agent-count, engineering-default-pack, optional-domain-packs | `hero-hosted-docs-remediation` | derive or remove count; explain composition |
| `web/docs/src/commands/index.md` | installed-command-count, harness-native-workflows, verify-closes-delivery | `hero-hosted-docs-remediation` | harness terminology and closing path correction required |
| `web/docs/src/concepts/agents-and-skills.md` | product-delivery-system, installed-agent-count, installed-skill-count, engineering-default-pack, optional-domain-packs | `hero-hosted-docs-remediation` | composition and maturity audit required |
| `web/docs/src/concepts/core-loop.md` | product-memory-system, product-delivery-system, product-two-system-loop, cold-audit-and-verify | `hero-hosted-docs-remediation` | two-system loop rewrite required |
| `web/docs/src/concepts/cross-repo.md` | cross-repo-peering | `hero-hosted-docs-remediation` | one-graph-per-project wording required |
| `web/docs/src/concepts/knowledge-base.md` | product-memory-system, product-two-system-loop | `hero-hosted-docs-remediation` | expand into primary product-memory path |
| `web/docs/src/concepts/specs.md` | product-delivery-system, product-two-system-loop, verify-closes-delivery | `hero-hosted-docs-remediation` | position specs as mechanism, not product category |
| `web/docs/src/configuration/hero-json.md` | public-config-example, tracker-evidence-and-mutations, hero-serve-intelligence | `hero-hosted-docs-remediation` | decoder-backed replacement required |
| `web/docs/src/configuration/mcp-setup.md` | runtime-mcp-tool-count, hero-serve-intelligence, harness-native-workflows | `hero-hosted-docs-remediation` | runtime registry and prerequisite audit required |
| `web/docs/src/configuration/tracker-setup.md` | tracker-evidence-and-mutations | `hero-hosted-docs-remediation` | provider/settings evidence audit required |
| `web/docs/src/getting-started/first-workflow.md` | product-delivery-system, verify-closes-delivery, cold-audit-and-verify | `hero-hosted-docs-remediation` | execute against current CLI |
| `web/docs/src/getting-started/installation.md` | current-release-version, go-prerequisite, supported-install-targets | `hero-hosted-docs-remediation` | release/source-build split required |
| `web/docs/src/getting-started/project-setup.md` | install-satellite-subcommands, satellite-workspace-architecture, harness-native-workflows, engineering-default-pack | `hero-hosted-docs-remediation` | P0 correction required |
| `web/docs/src/cli/focus.md` | attention-mail-focus | `hero-hosted-docs-remediation` | bounded/private semantics audit required |
| `web/docs/src/cli/import.md` | tracker-evidence-and-mutations | `hero-hosted-docs-remediation` | current flags/config audit required |
| `web/docs/src/cli/mail.md` | attention-mail-focus, cross-repo-peering | `hero-hosted-docs-remediation` | untrusted-data and consent boundaries required |
| `web/docs/src/cli/overview.md` | current-release-version, installed-command-count, installed-agent-count, installed-skill-count, headless-runtime, code-host-operations | `hero-hosted-docs-remediation` | version/count/availability correction required |
| `web/docs/src/cli/peering.md` | cross-repo-peering | `hero-hosted-docs-remediation` | current async semantics audit required |
| `web/docs/src/cli/search-and-context.md` | product-memory-system | `hero-hosted-docs-remediation` | retrieval/capture distinction required |
| `web/docs/src/cli/server-and-mcp.md` | hero-serve-intelligence, runtime-mcp-tool-count, code-host-operations | `hero-hosted-docs-remediation` | current registry and setup boundaries required |
| `web/docs/src/cli/spec-management.md` | product-delivery-system, verify-closes-delivery, cold-audit-and-verify | `hero-hosted-docs-remediation` | P0 closing-path correction required |
| `web/docs/src/cli/testing-and-demos.md` | cold-audit-and-verify, landing-product-output | `hero-hosted-docs-remediation` | real-vs-illustrative evidence contract required |
| `web/docs/src/cli/tracker-integration.md` | tracker-evidence-and-mutations | `hero-hosted-docs-remediation` | supported provider/consent audit required |
| `web/docs/src/releases/index.md` | current-release-version | `hero-v034-release-prep` | generated v0.34 notes/current release authority required |
| `web/docs/src/serve/homes.md` | hero-serve-intelligence | `hero-hosted-docs-remediation` | current dashboard route audit required |
| `web/docs/src/serve/mcp-tool-metadata.md` | runtime-mcp-tool-count, hero-serve-intelligence | `hero-hosted-docs-remediation` | metadata/current-registry audit required |
| `web/docs/src/workflows/delivery-and-debugging.md` | product-delivery-system, cold-audit-and-verify, verify-closes-delivery | `hero-hosted-docs-remediation` | execute current workflow examples |
| `web/docs/src/workflows/discovery-and-design.md` | product-delivery-system | `hero-hosted-docs-remediation` | current workflow audit required |
| `web/docs/src/workflows/knowledge-and-standards.md` | product-memory-system, product-two-system-loop | `hero-hosted-docs-remediation` | capture/retrieval loop required |
| `web/docs/src/workflows/review-and-quality.md` | product-delivery-system, cold-audit-and-verify | `hero-hosted-docs-remediation` | current gate/agent audit required |
| `web/docs/src/workflows/sprint-and-planning.md` | product-delivery-system | `hero-hosted-docs-remediation` | current command/availability audit required |

## Landing and repository exposure

| Surface | Primary claim IDs | Owner | State |
|---|---|---|---|
| `web/landing/site/index.html` | product-memory-system, product-delivery-system, product-two-system-loop, current-release-version, harness-native-workflows, landing-product-output, apache-license-status, canonical-domain-availability | `hero-landing-message-refresh` | memory-first rewrite, real evidence, DNS/deploy required |
| `web/landing/site/robots.txt` | canonical-domain-availability | `hero-landing-message-refresh` | verify production behavior |
| Repository description/topics/homepage | product-memory-system, product-two-system-loop, apache-license-status, proprietary-repository-boundary, canonical-domain-availability | `hero-public-repo-readiness` | update only at visibility gate |
| Repository source links in docs/site | apache-license-status, canonical-domain-availability | `hero-public-visibility-launch-gate` | keep hidden/gated until anonymous access exists |

## Non-claim assets

These files make no behavioral claims but remain in the later third-party asset and accessibility inventories:

- `web/docs/src/assets/favicon.svg`
- `web/docs/src/assets/logo.svg`
- `web/docs/src/stylesheets/brand.css`
- `web/landing/site/favicon.svg`
- `web/landing/site/og-image.svg`

## Completeness check

- Root guides inventoried: 5.
- Hosted Markdown pages inventoried: 35.
- Landing/exposure surfaces inventoried: 4.
- Non-claim web assets inventoried: 5.
- Every row has an owner and at least one registry claim ID.

Run this source-list check before downstream delivery to detect a newly added page that lacks a row:

```text
rg --files web/docs/src web/landing/site | sort
```
