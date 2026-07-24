# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-07-24T04:19:25Z · projected from 588 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | <1m ago | satellite-corpus-integration |
| docs | maturing | web/docs/ | 11d ago | — |
| domains/chat | maturing | domains/chat/ | 5d ago | — |
| domains/engineering | maturing | domains/engineering/ | <1m ago | — |
| domains/pm | maturing | domains/pm/ | 6d ago | — |
| domains/sales | maturing | domains/sales/ | 15d ago | — |
| landing | building | web/landing/ | 12d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | <1m ago | agent-outposts |
| (unassigned) | — | — | — | 236 specs without surface |

_Run `hero snapshot assign` to bucket unassigned specs._

> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.

## Active initiatives

- **"Always-On Runtime"** (surface: serve) — 0/1 specs done
- **"Cold-Start Trust Hardening — Fail Loud, Never Mislead, at First Use"** (surface: core) — 2/2 specs done
- **"Concurrent-Session Branching & Worktree Isolation"** (surface: —) — 0/0 specs done
- **"Context Engine v2 — Fix and Optimize hero-code Desktop Context Curation"** (surface: —) — 4/8 specs done
- **Environment Awareness — CI/Deployment/Runtime Visibility** (surface: —) — 0/0 specs done
- **Get Back on Track — Mission-First V2 Recovery** (surface: core) — 7/11 specs done
- **Hero Domains — Platform Architecture for Non-Engineering Verticals** (surface: core, domains/engineering, domains/pm, domains/sales) — 13/17 specs done
- **"Hero-in-Hero-Code Parity — Fix Hero Workflow Integration in the Desktop App"** (surface: —) — 0/9 specs done
- **Hero Killer Features — Agent Effectiveness, Team Power, Living Specs** (surface: core, serve) — 10/11 specs done
- **Hero Marketing — Positioning, Distribution, and Launch** (surface: landing) — 0/9 specs done; in flight: hero-landing-page
- **Hero Platform — Headless Execution, Team Automation, and Shared Visibility** (surface: core, serve) — 2/8 specs done; in flight: hero-team-server
- **"Hero Doesn't Lie — Self-Consistency Between Generated Guidance, Hero's Own Writes, and Hero's Actual Contract"** (surface: core) — 0/5 specs done
- **Hero Surface Architecture — One Surface, Every Layer, Every Role** (surface: serve) — 8/9 specs done
- **Hero Team Experience — Complete Multi-Developer Workflow** (surface: —) — 0/1 specs done
- **Install + Upgrade Contract Coverage — Prove Every Target Works Every Time** (surface: —) — 0/0 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **"Retrieval Quality — Reranking, Expansion & Feedback Loop"** (surface: —) — 0/0 specs done
- **Single-Source Install — One Canonical Tree, Every Harness Reads It** (surface: core) — 6/7 specs done

### Recently completed initiatives

- **"Conversational Attention Operability — Safe Chat-Loop Access to Mail and Focus"** (surface: core, domains/engineering) — 6/6 specs done · COMPLETED 2026-07-24
- **"Durable Attention — Project Mail, Personal Focus, and Explicit Promotion"** (surface: core, domains/engineering, serve) — 7/7 specs done · COMPLETED 2026-07-23
- **"Tracker Source Fidelity — Canonical Jira Markdown and Lazy Evidence Provenance"** (surface: hero-engine-tracker) — 2/2 specs done · COMPLETED 2026-07-22

## Recently completed (last 14 days)

- **(unassigned)** — conversational-attention-operability, attention-contract-bundle-publication, attention-conversational-routes, attention-lifecycle-read-awareness, attention-mcp-action-tools, durable-attention
- **core** — attention-interaction-consent-contract
- **domains/engineering** — portable-routing-rules, peering-over-project-mail
- **serve** — attention-read-model-v1, project-mail-triage-and-provenance, hero-idea-primitive-core

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **serve** — `hero-team-server` (P1, delivering)
3. **core** — `satellite-corpus-integration` (high, delivering)
4. **serve** — `agent-outposts` (medium, delivering)
5. **serve** — `retrieval-contradiction-detection` (—, delivering)

## Open risks & blockers

- **Blocked specs (13):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection); `wire-checks-to-boundaries` (waits on spec-contract-enums-unified).
- **Stale-in-flight (2):** `hero-landing-page` (66d), `hero-team-server` (66d).
- **Aged open bugs (9):** `install-target-emits-both-claude-and-agents-md` (open 70d), `next-project-file-conflict-not-regenerated` (open 51d), `desktop-sidebar-mcp-not-running` (open 50d), `hihcp-agent-loop-error-recovery` (open 45d), `hihcp-agents-md-harness-agnostic` (open 45d), `hihcp-mcp-auto-reconnect` (open 45d), `hihcp-mcp-first-turn-readiness` (open 45d), `hihcp-permission-bridge-validation` (open 45d), `hihcp-rgignore` (open 45d).
- **Unassigned specs (236) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 211/447 (47%)
- Projection generation: 1ms · Source nodes: 588

