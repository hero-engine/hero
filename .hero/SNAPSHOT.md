# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-08-23T00:26:22Z · projected from 662 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | maturing | cmd/, internal/ | 1d ago | hero-runner |
| docs | maturing | web/docs/ | 41d ago | — |
| domains/chat | maturing | domains/chat/ | 35d ago | — |
| domains/engineering | maturing | domains/engineering/ | 3d ago | — |
| domains/pm | maturing | domains/pm/ | 36d ago | — |
| domains/qa | concept | domains/qa/ | — | — |
| domains/sales | maturing | domains/sales/ | 1d ago | — |
| landing | building | web/landing/ | 42d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 1d ago | agent-outposts |
| (unassigned) | — | — | — | 253 specs without surface |

_Run `hero snapshot assign` to bucket unassigned specs._

> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.

## Active initiatives

- **"Always-On Runtime"** (surface: serve) — 0/1 specs done
- **"Cold-Start Trust Hardening — Fail Loud, Never Mislead, at First Use"** (surface: core) — 2/2 specs done
- **"Concurrent-Session Branching & Worktree Isolation"** (surface: —) — 0/0 specs done
- **"Context Engine v2 — Fix and Optimize hero-code Desktop Context Curation"** (surface: —) — 4/8 specs done
- **Environment Awareness — CI/Deployment/Runtime Visibility** (surface: —) — 0/0 specs done
- **Get Back on Track — Mission-First V2 Recovery** (surface: core) — 7/11 specs done
- **Hero Domains — Platform Architecture for Non-Engineering Verticals** (surface: core, domains/engineering, domains/pm, domains/sales) — 16/20 specs done
- **"Hero-in-Hero-Code Parity — Fix Hero Workflow Integration in the Desktop App"** (surface: —) — 0/9 specs done
- **Hero Killer Features — Agent Effectiveness, Team Power, Living Specs** (surface: core, serve) — 10/11 specs done
- **"Hero v0.34 Public Readiness — Truth, Licensing, and Launch"** (surface: core, landing, serve) — 0/20 specs done; in flight: hero-landing-page
- **Hero Platform — Headless Execution, Team Automation, and Shared Visibility** (surface: core, serve) — 3/8 specs done
- **"Hero Doesn't Lie — Self-Consistency Between Generated Guidance, Hero's Own Writes, and Hero's Actual Contract"** (surface: core) — 0/5 specs done
- **Hero Surface Architecture — One Surface, Every Layer, Every Role** (surface: serve) — 8/9 specs done
- **Hero Team Experience — Complete Multi-Developer Workflow** (surface: —) — 0/1 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **"Retrieval Quality — Reranking, Expansion & Feedback Loop"** (surface: —) — 0/0 specs done

### Recently completed initiatives

- **Install + Upgrade Contract Coverage — Prove Every Target Works Every Time** (surface: domains/engineering) — 1/1 specs done · COMPLETED 2026-08-18
- **Single-Source Install — One Canonical Tree, Every Harness Reads It** (surface: core) — 6/7 specs done · COMPLETED 2026-08-14
- **"Interactive CLI Input — Scoped Completion"** (surface: core) — 4/4 specs done · COMPLETED 2026-08-03

## Recently completed (last 14 days)

- **(unassigned)** — dual-mode-pm-qa-capability-packs, qa-public-pack, pm-public-pack, install-upgrade-contract-coverage, single-source-install
- **core** — node-index-repo-identity-collision, hero-desktop-release-artifact-contract
- **domains/engineering** — grok-build-harness-target
- **serve** — cross-project-mail-read-contract, mcp-tool-category-metadata

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **serve** — `agent-outposts` (medium, delivering)
3. **serve** — `retrieval-contradiction-detection` (—, delivering)
4. **(unassigned)** — `team-connect` (—, delivering)
5. **(unassigned)** — `always-on-runtime` (P0, planning)

## Open risks & blockers

- **Blocked specs (24):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-apache-license-grant-gate` (waits on hero-v034-release-prep, hero-licensing-boundary-and-provenance, hero-public-repo-readiness); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-continuity-proof-demo` (waits on hero-public-truth-baseline, hero-positioning, hero-public-repo-readiness); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-hosted-docs-remediation` (waits on hero-public-truth-baseline, hero-positioning); `hero-landing-message-refresh` (waits on hero-root-docs-remediation, hero-hosted-docs-remediation); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-licensing-boundary-and-provenance` (waits on hero-public-truth-baseline); `hero-positioning` (waits on hero-public-truth-baseline, hero-licensing-boundary-and-provenance); `hero-public-docs-drift-guard` (waits on hero-root-docs-remediation, hero-hosted-docs-remediation, hero-landing-message-refresh, hero-public-repo-readiness, hero-continuity-proof-demo); `hero-public-repo-readiness` (waits on hero-landing-message-refresh, hero-licensing-boundary-and-provenance); `hero-public-visibility-launch-gate` (waits on hero-apache-license-grant-gate, hero-v034-release-prep); `hero-root-docs-remediation` (waits on hero-public-truth-baseline, hero-positioning); `hero-team-server` (waits on hero-runner); `hero-v034-release-prep` (waits on hero-public-docs-drift-guard); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection); `wire-checks-to-boundaries` (waits on spec-contract-enums-unified).
- **Stale-in-flight (4):** `hero-landing-page` (96d), `retrieval-contradiction-detection` (43d), `agent-outposts` (42d), `team-connect` (42d).
- **Aged open bugs (16):** `install-target-emits-both-claude-and-agents-md` (open 100d), `next-project-file-conflict-not-regenerated` (open 81d), `desktop-sidebar-mcp-not-running` (open 80d), `hihcp-agents-md-harness-agnostic` (open 75d), `hihcp-mcp-auto-reconnect` (open 75d), `hihcp-mcp-first-turn-readiness` (open 75d), `hihcp-permission-bridge-validation` (open 75d), `hihcp-agent-loop-error-recovery` (open 75d), `hihcp-rgignore` (open 75d), `jira-connection-onboarding-misleads-agents` (open 40d), `resume-emits-dead-recall-command` (open 40d), `tracker-backed-diagnosis-publication-contract-broken` (open 35d), `tracker-semantic-priority-field-mapping` (open 34d), `jira-import-classification-obscures-work-items` (open 34d), `ledger-signoff-substring-match-fails-open` (open 29d), `graph-unpartitioned-writers-duplicate-nodes` (open 29d).
- **Unassigned specs (253) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 10 (inferred: 10 · overrides applied: 0)
- Specs covered: 217/470 (46%)
- Projection generation: 1ms · Source nodes: 662

