# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-07-30T18:09:02Z · projected from 612 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | 1h ago | satellite-corpus-integration |
| docs | maturing | web/docs/ | 17d ago | — |
| domains/chat | maturing | domains/chat/ | 12d ago | — |
| domains/engineering | maturing | domains/engineering/ | 6d ago | — |
| domains/pm | maturing | domains/pm/ | 12d ago | — |
| domains/sales | maturing | domains/sales/ | 21d ago | — |
| landing | building | web/landing/ | 18d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 27m ago | agent-outposts |
| (unassigned) | — | — | — | 239 specs without surface |

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

- **"Code Host Broker Capabilities — Hero-owned PR lifecycle boundary"** (surface: —) — 8/8 specs done · COMPLETED 2026-07-28
- **"Continuous Code-Index Freshness"** (surface: core) — 3/3 specs done · COMPLETED 2026-07-27
- **"Conversational Attention Operability — Safe Chat-Loop Access to Mail and Focus"** (surface: core, domains/engineering) — 6/6 specs done · COMPLETED 2026-07-24

## Recently completed (last 14 days)

- **(unassigned)** — hero-code-host-broker-capabilities, code-host-broker-surfaces-and-conformance, github-pull-request-merge-broker, github-pull-request-state-transition-broker, github-pull-request-collaboration-broker, github-pull-request-create-broker, github-code-host-read-adapter, code-host-broker-v1-contract, code-host-integration-capability-model, windows-file-lock-cross-build
- **core** — mail-3ab437af53c66997e94a0268
- **serve** — code-host-authenticated-actor-mcp-schema-parity

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **(unassigned)** — `code-host-cancel-after-apply-test-timing` (high, delivering)
3. **serve** — `hero-team-server` (P1, delivering)
4. **core** — `satellite-corpus-integration` (high, delivering)
5. **serve** — `agent-outposts` (medium, delivering)

## Open risks & blockers

- **Blocked specs (13):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection); `wire-checks-to-boundaries` (waits on spec-contract-enums-unified).
- **Stale-in-flight (6):** `hero-landing-page` (72d), `hero-team-server` (72d), `retrieval-contradiction-detection` (20d), `agent-outposts` (18d), `satellite-corpus-integration` (18d), `team-connect` (18d).
- **Aged open bugs (9):** `install-target-emits-both-claude-and-agents-md` (open 76d), `next-project-file-conflict-not-regenerated` (open 57d), `desktop-sidebar-mcp-not-running` (open 56d), `hihcp-agent-loop-error-recovery` (open 51d), `hihcp-agents-md-harness-agnostic` (open 51d), `hihcp-mcp-auto-reconnect` (open 51d), `hihcp-mcp-first-turn-readiness` (open 51d), `hihcp-permission-bridge-validation` (open 51d), `hihcp-rgignore` (open 51d).
- **Unassigned specs (239) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 206/445 (46%)
- Projection generation: 1ms · Source nodes: 612

