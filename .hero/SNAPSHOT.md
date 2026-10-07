# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-10-06T23:43:42Z · projected from 691 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | 2m ago | work-item-model |
| docs | maturing | web/docs/ | 85d ago | — |
| domains/chat | maturing | domains/chat/ | 33d ago | — |
| domains/engineering | maturing | domains/engineering/ | <1m ago | — |
| domains/pm | maturing | domains/pm/ | 33d ago | — |
| domains/qa | concept | domains/qa/ | — | — |
| domains/sales | maturing | domains/sales/ | 46d ago | — |
| landing | building | web/landing/ | 44d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 2m ago | agent-outposts |
| (unassigned) | — | — | — | 259 specs without surface |

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
- **"Hero read contract — hero_work, hero_spec, hero_handoff for every Hero client"** (surface: core, domains/engineering, serve) — 1/7 specs done; in flight: hero-spec-handoff-tools, hero-work-tool, polish-and-suggested, read-contract-conformance, work-item-model
- **"Hero Doesn't Lie — Self-Consistency Between Generated Guidance, Hero's Own Writes, and Hero's Actual Contract"** (surface: core) — 1/5 specs done
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

- **(unassigned)** — next-projection-accuracy-and-freshness, ledger-signoff-substring-match-fails-open
- **core** — upgrade-refreshes-managed-gitignore, spec-title-keeps-yaml-quotes, deepseek-project-mcp-registration, sept-review-cleanup, resume-emits-dead-recall-command, stale-connect-provider-usage-expectation
- **domains/engineering** — next-step-engine, deepseek-harness-install-target
- **serve** — codex-command-workflow-surface-split-brain

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **hero-core** — `mail-b7ca19966ac5041e6ff604dd` (critical, delivering)
3. **core** — `work-item-model` (critical, delivering)
4. **serve** — `hero-spec-handoff-tools` (high, delivering)
5. **serve** — `hero-work-tool` (high, delivering)

## Open risks & blockers

- **Blocked specs (14):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-docs-site); `hero-landing-page` (waits on hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `polish-and-suggested` (waits on hero-work-tool); `read-contract-conformance` (waits on hero-spec-handoff-tools, hero-work-tool, polish-and-suggested); `timely-briefs` (waits on retrieval-contradiction-detection); `wire-checks-to-boundaries` (waits on spec-contract-enums-unified); `work-item-model` (waits on read-contract-v1).
- **Stale-in-flight (6):** `retrieval-contradiction-detection` (88d), `agent-outposts` (87d), `team-connect` (87d), `hero-landing-page` (44d), `mail-b7ca19966ac5041e6ff604dd` (33d), `mail-thread-foreground-read-action` (33d).
- **Aged open bugs (14):** `install-target-emits-both-claude-and-agents-md` (open 144d), `next-project-file-conflict-not-regenerated` (open 125d), `desktop-sidebar-mcp-not-running` (open 124d), `hihcp-permission-bridge-validation` (open 119d), `hihcp-mcp-auto-reconnect` (open 119d), `hihcp-mcp-first-turn-readiness` (open 119d), `hihcp-agents-md-harness-agnostic` (open 119d), `hihcp-rgignore` (open 119d), `hihcp-agent-loop-error-recovery` (open 119d), `jira-connection-onboarding-misleads-agents` (open 84d), `tracker-backed-diagnosis-publication-contract-broken` (open 79d), `jira-import-classification-obscures-work-items` (open 78d), `tracker-semantic-priority-field-mapping` (open 78d), `graph-unpartitioned-writers-duplicate-nodes` (open 73d).
- **Unassigned specs (259) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 10 (inferred: 10 · overrides applied: 0)
- Specs covered: 232/491 (47%)
- Projection generation: 1ms · Source nodes: 691

