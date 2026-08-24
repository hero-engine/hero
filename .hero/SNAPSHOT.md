# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-08-24T01:49:43Z · projected from 669 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | maturing | cmd/, internal/ | 4d ago | hero-runner |
| docs | maturing | web/docs/ | 42d ago | — |
| domains/chat | maturing | domains/chat/ | 36d ago | — |
| domains/engineering | maturing | domains/engineering/ | 4d ago | — |
| domains/pm | maturing | domains/pm/ | 37d ago | — |
| domains/qa | concept | domains/qa/ | — | — |
| domains/sales | maturing | domains/sales/ | 2d ago | — |
| landing | building | web/landing/ | 10h ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 4h ago | agent-outposts |
| (unassigned) | — | — | — | 257 specs without surface |

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
- **"Hero v0.34 Public Release Readiness"** (surface: serve) — 11/12 specs done
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

- **(unassigned)** — remove-invented-preview-marketing-copy, hero-apache-license-grant-gate, hero-v034-release-prep, hero-public-repo-readiness, hero-landing-message-refresh, hero-root-docs-remediation, hero-hosted-docs-remediation, hero-positioning, hero-licensing-boundary-and-provenance, hero-public-truth-baseline, dual-mode-pm-qa-capability-packs
- **serve** — hero-public-docs-drift-guard

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **serve** — `agent-outposts` (medium, delivering)
3. **serve** — `retrieval-contradiction-detection` (—, delivering)
4. **(unassigned)** — `team-connect` (—, delivering)
5. **(unassigned)** — `always-on-runtime` (P0, planning)

## Open risks & blockers

- **Blocked specs (11):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-docs-site); `hero-landing-page` (waits on hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection); `wire-checks-to-boundaries` (waits on spec-contract-enums-unified).
- **Stale-in-flight (3):** `retrieval-contradiction-detection` (44d), `agent-outposts` (43d), `team-connect` (43d).
- **Aged open bugs (16):** `install-target-emits-both-claude-and-agents-md` (open 101d), `next-project-file-conflict-not-regenerated` (open 82d), `desktop-sidebar-mcp-not-running` (open 81d), `hihcp-agents-md-harness-agnostic` (open 76d), `hihcp-mcp-auto-reconnect` (open 76d), `hihcp-mcp-first-turn-readiness` (open 76d), `hihcp-permission-bridge-validation` (open 76d), `hihcp-agent-loop-error-recovery` (open 76d), `hihcp-rgignore` (open 76d), `jira-connection-onboarding-misleads-agents` (open 41d), `resume-emits-dead-recall-command` (open 41d), `tracker-backed-diagnosis-publication-contract-broken` (open 36d), `tracker-semantic-priority-field-mapping` (open 35d), `jira-import-classification-obscures-work-items` (open 35d), `ledger-signoff-substring-match-fails-open` (open 30d), `graph-unpartitioned-writers-duplicate-nodes` (open 30d).
- **Unassigned specs (257) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 10 (inferred: 10 · overrides applied: 0)
- Specs covered: 216/473 (45%)
- Projection generation: 1ms · Source nodes: 669

