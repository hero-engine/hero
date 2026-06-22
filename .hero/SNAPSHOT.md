# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-06-22T04:20:04Z · projected from 323 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | maturing | cmd/, internal/ | 3h ago | hero-workspace-not-self-describing |
| docs | concept | web/docs/ | — | — |
| domains/chat | concept | domains/chat/ | — | — |
| domains/engineering | maturing | domains/engineering/ | 10d ago | — |
| domains/pm | concept | domains/pm/ | 34d ago | — |
| domains/sales | maturing | domains/sales/ | 12d ago | — |
| landing | building | web/landing/ | 21d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 12d ago | agent-outposts |
| (unassigned) | — | — | — | 154 specs without surface |

_Run `hero snapshot assign` to bucket unassigned specs._

> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.

## Active initiatives

- **"Concurrent-Session Branching & Worktree Isolation"** (surface: —) — 0/0 specs done
- **"Context Engine v2 — Fix and Optimize hero-code Desktop Context Curation"** (surface: —) — 0/0 specs done
- **Environment Awareness — CI/Deployment/Runtime Visibility** (surface: —) — 0/0 specs done
- **Get Back on Track — Mission-First V2 Recovery** (surface: core) — 7/11 specs done
- **Hero Domains — Platform Architecture for Non-Engineering Verticals** (surface: core, domains/engineering, domains/pm, domains/sales) — 11/16 specs done
- **"Hero-in-Hero-Code Parity — Fix Hero Workflow Integration in the Desktop App"** (surface: —) — 0/8 specs done
- **Hero Killer Features — Agent Effectiveness, Team Power, Living Specs** (surface: core, serve) — 10/11 specs done
- **Hero Marketing — Positioning, Distribution, and Launch** (surface: landing) — 0/9 specs done; in flight: hero-landing-page
- **Hero Platform — Headless Execution, Team Automation, and Shared Visibility** (surface: core, serve) — 2/7 specs done; in flight: hero-team-server
- **Hero Surface Architecture — One Surface, Every Layer, Every Role** (surface: serve) — 7/9 specs done
- **Hero Surface Polish — Ongoing Quality Pass on the Web Companion** (surface: serve) — 5/5 specs done
- **Hero Team Experience — Complete Multi-Developer Workflow** (surface: —) — 0/0 specs done
- **Install + Upgrade Contract Coverage — Prove Every Target Works Every Time** (surface: —) — 0/0 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **"Retrieval Quality — Reranking, Expansion & Feedback Loop"** (surface: —) — 0/0 specs done
- **Single-Source Install — One Canonical Tree, Every Harness Reads It** (surface: core) — 5/6 specs done

### Recently completed initiatives

- **Roadmap Shape — Detect, Surface, and Resolve Spec-Shape Drift** (surface: serve) — 4/4 specs done · COMPLETED 2026-06-01
- **hero serve Project Section — Per-Project Info, Utilities, and Operations Page** (surface: serve) — 5/5 specs done · COMPLETED 2026-05-20
- **Hero v2 — System Design Initiative** (surface: core, serve) — 8/8 specs done · COMPLETED 2026-05-18

## Recently completed (last 14 days)

- **(unassigned)** — landing-docs-content-refresh, unified-retrieval-layer, e2e-discovery, e2e-traversal, e2e-validation, graph-conflict-detection
- **core** — verify-slug-resolution-hints, embedded-inference, cross-repo-peering
- **domains/engineering** — spec-completion-loop
- **domains/sales** — hero-sales
- **serve** — monorepo-satellite-installs

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **serve** — `hero-team-server` (P1, delivering)
3. **serve** — `agent-outposts` (medium, delivering)
4. **(unassigned)** — `context-engine-v2` (critical, planning)
5. **(unassigned)** — `core-vertical-layering` (P0, planning)

## Open risks & blockers

- **Blocked specs (10):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `timely-briefs` (waits on retrieval-contradiction-detection).
- **Stale-in-flight (3):** `agent-outposts` (34d), `hero-landing-page` (34d), `hero-team-server` (34d).
- **Aged open bugs (8):** `hero-workspace-not-self-describing` (open 41d), `scan-enrichment-unbounded-loop` (open 41d), `scan-test-detection-misses-spock-vitest` (open 41d), `initiative-required-sections-drift` (open 36d), `spec-lifecycle-hygiene-breakdown` (open 35d), `claim-matches-sentinel-collision` (open 34d), `install-target-emits-both-claude-and-agents-md` (open 34d), `hero-search-json-flag-silently-ignored` (open 29d).
- **Unassigned specs (154) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 147/301 (48%)
- Projection generation: 1ms · Source nodes: 323

