# Project Snapshot — hero-mail-main.PaKrG3

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-08-27T15:42:33Z · projected from 536 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | maturing | cmd/, internal/ | <1m ago | hero-runner |
| docs | maturing | web/docs/ | <1m ago | — |
| domains/chat | maturing | domains/chat/ | <1m ago | — |
| domains/engineering | maturing | domains/engineering/ | <1m ago | — |
| domains/pm | maturing | domains/pm/ | <1m ago | — |
| domains/qa | concept | domains/qa/ | — | — |
| domains/sales | maturing | domains/sales/ | <1m ago | — |
| landing | building | web/landing/ | <1m ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | <1m ago | agent-outposts |
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
- **Hero Platform — Headless Execution, Team Automation, and Shared Visibility** (surface: core, serve) — 3/8 specs done
- **"Hero Doesn't Lie — Self-Consistency Between Generated Guidance, Hero's Own Writes, and Hero's Actual Contract"** (surface: core) — 0/5 specs done
- **Hero Surface Architecture — One Surface, Every Layer, Every Role** (surface: serve) — 8/9 specs done
- **Hero Team Experience — Complete Multi-Developer Workflow** (surface: —) — 0/1 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **"Retrieval Quality — Reranking, Expansion & Feedback Loop"** (surface: —) — 0/0 specs done

### Recently completed initiatives

- **"Hero v0.34 Public Release Readiness"** (surface: serve) — 12/12 specs done · COMPLETED 2026-08-24
- **Install + Upgrade Contract Coverage — Prove Every Target Works Every Time** (surface: domains/engineering) — 1/1 specs done · COMPLETED 2026-08-18
- **Single-Source Install — One Canonical Tree, Every Harness Reads It** (surface: core) — 6/7 specs done · COMPLETED 2026-08-14

## Recently completed (last 14 days)

- **(unassigned)** — mock-tracker-server, gitlab-tracker-support, cev2-verbatim-turn-counting, cev2-protect-compaction-summaries, hero-marketing, hero-public-visibility-launch-gate, remove-invented-preview-marketing-copy, hero-apache-license-grant-gate, hero-v034-release-prep, hero-public-repo-readiness, hero-landing-message-refresh
- **serve** — hero-public-docs-drift-guard

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **hero-core** — `mail-b7ca19966ac5041e6ff604dd` (critical, delivering)
3. **hero-core** — `mail-thread-foreground-read-action` (high, delivering)
4. **serve** — `agent-outposts` (medium, delivering)
5. **serve** — `retrieval-contradiction-detection` (—, delivering)

## Open risks & blockers

- **Blocked specs (11):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-docs-site); `hero-landing-page` (waits on hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection); `wire-checks-to-boundaries` (waits on spec-contract-enums-unified).
- **Aged open bugs (16):** `install-target-emits-both-claude-and-agents-md` (open 104d), `next-project-file-conflict-not-regenerated` (open 85d), `desktop-sidebar-mcp-not-running` (open 84d), `hihcp-agents-md-harness-agnostic` (open 79d), `hihcp-mcp-auto-reconnect` (open 79d), `hihcp-mcp-first-turn-readiness` (open 79d), `hihcp-permission-bridge-validation` (open 79d), `hihcp-agent-loop-error-recovery` (open 79d), `hihcp-rgignore` (open 79d), `jira-connection-onboarding-misleads-agents` (open 44d), `resume-emits-dead-recall-command` (open 44d), `tracker-backed-diagnosis-publication-contract-broken` (open 39d), `tracker-semantic-priority-field-mapping` (open 38d), `jira-import-classification-obscures-work-items` (open 38d), `ledger-signoff-substring-match-fails-open` (open 33d), `graph-unpartitioned-writers-duplicate-nodes` (open 33d).
- **Unassigned specs (257) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 10 (inferred: 10 · overrides applied: 0)
- Specs covered: 219/476 (46%)
- Projection generation: 1ms · Source nodes: 536

