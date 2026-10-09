# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-10-09T18:22:14Z · projected from 700 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | maturing | cmd/, internal/ | 2d ago | hero-runner |
| docs | maturing | web/docs/ | 88d ago | — |
| domains/chat | maturing | domains/chat/ | 36d ago | — |
| domains/engineering | maturing | domains/engineering/ | 2d ago | — |
| domains/pm | maturing | domains/pm/ | 36d ago | — |
| domains/qa | concept | domains/qa/ | — | — |
| domains/sales | maturing | domains/sales/ | 49d ago | — |
| landing | building | web/landing/ | 47d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 12m ago | next-projection-reads-stale-spec-graph |
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
- **"Hero Doesn't Lie — Self-Consistency Between Generated Guidance, Hero's Own Writes, and Hero's Actual Contract"** (surface: core) — 1/5 specs done
- **Hero Surface Architecture — One Surface, Every Layer, Every Role** (surface: serve) — 8/9 specs done
- **Hero Team Experience — Complete Multi-Developer Workflow** (surface: —) — 0/1 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **"Retrieval Quality — Reranking, Expansion & Feedback Loop"** (surface: —) — 0/0 specs done

### Recently completed initiatives

- **"Hero read contract — hero_work, hero_spec, hero_handoff for every Hero client"** (surface: core, domains/engineering, serve) — 6/7 specs done · COMPLETED 2026-10-07
- **"Hero v0.34 Public Release Readiness"** (surface: serve) — 12/12 specs done · COMPLETED 2026-08-24
- **Install + Upgrade Contract Coverage — Prove Every Target Works Every Time** (surface: domains/engineering) — 1/1 specs done · COMPLETED 2026-08-18

## Recently completed (last 14 days)

- **(unassigned)** — hero-read-contract
- **core** — why-intake-slug-tie, followup-epic-parity, followup-audit-staleness-git, work-item-model
- **serve** — symlinked-hero-dir-walks-nothing, pre-release-sweep-v0-35-3, followup-serve-shutdown, read-contract-conformance, polish-and-suggested, hero-work-tool, hero-spec-handoff-tools

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **hero-core** — `mail-b7ca19966ac5041e6ff604dd` (critical, delivering)
3. **hero-core** — `mail-thread-foreground-read-action` (high, delivering)
4. **serve** — `next-projection-reads-stale-spec-graph` (P1, delivering)
5. **serve** — `agent-outposts` (medium, delivering)

## Open risks & blockers

- **Blocked specs (11):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-docs-site); `hero-landing-page` (waits on hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection); `wire-checks-to-boundaries` (waits on spec-contract-enums-unified).
- **Stale-in-flight (6):** `retrieval-contradiction-detection` (91d), `agent-outposts` (89d), `team-connect` (89d), `hero-landing-page` (47d), `mail-b7ca19966ac5041e6ff604dd` (36d), `mail-thread-foreground-read-action` (36d).
- **Aged open bugs (14):** `install-target-emits-both-claude-and-agents-md` (open 147d), `next-project-file-conflict-not-regenerated` (open 128d), `desktop-sidebar-mcp-not-running` (open 127d), `hihcp-permission-bridge-validation` (open 122d), `hihcp-mcp-auto-reconnect` (open 122d), `hihcp-mcp-first-turn-readiness` (open 122d), `hihcp-agents-md-harness-agnostic` (open 122d), `hihcp-rgignore` (open 122d), `hihcp-agent-loop-error-recovery` (open 122d), `jira-connection-onboarding-misleads-agents` (open 87d), `tracker-backed-diagnosis-publication-contract-broken` (open 82d), `jira-import-classification-obscures-work-items` (open 81d), `tracker-semantic-priority-field-mapping` (open 81d), `graph-unpartitioned-writers-duplicate-nodes` (open 76d).
- **Unassigned specs (259) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 10 (inferred: 10 · overrides applied: 0)
- Specs covered: 239/498 (47%)
- Projection generation: 2ms · Source nodes: 700

