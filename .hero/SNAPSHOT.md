# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-05-20T01:12:51Z · projected from 262 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | 8h ago | hero-local-merge-missing-dialect-fields |
| docs | concept | web/docs/ | — | — |
| domains/chat | concept | domains/chat/ | — | — |
| domains/engineering | building | domains/engineering/ | 8h ago | spec-types-cache-frontmatter-empty |
| domains/pm | concept | domains/pm/ | 1d ago | — |
| domains/sales | concept | domains/sales/ | — | — |
| landing | building | web/landing/ | 1d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 1m ago | vocabulary-resolve-misses-methodology-derivation |
| (unassigned) | — | — | — | 113 specs without surface |

_Run `hero snapshot assign` to bucket unassigned specs._

> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.

## Active initiatives

- **Environment Awareness — CI/Deployment/Runtime Visibility** (surface: —) — 0/0 specs done
- **Get Back on Track — Mission-First V2 Recovery** (surface: core) — 3/11 specs done; in flight: master-ingest-restore, per-feature-smoke-coverage, spec-status-integrity, traversal-queries
- **Hero Domains — Platform Architecture for Non-Engineering Verticals** (surface: core, domains/engineering, domains/pm, serve) — 10/16 specs done; in flight: hero-sales
- **Hero Killer Features — Agent Effectiveness, Team Power, Living Specs** (surface: core, serve) — 10/11 specs done
- **Hero Marketing — Positioning, Distribution, and Launch** (surface: landing) — 0/9 specs done; in flight: hero-landing-page
- **Hero Platform — Headless Execution, Team Automation, and Shared Visibility** (surface: core, serve) — 2/7 specs done; in flight: hero-team-server
- **hero serve Project Section — Per-Project Info, Utilities, and Operations Page** (surface: serve) — 0/5 specs done; in flight: hero-serve-project-section-mvp
- **Hero Surface Architecture — One Surface, Every Layer, Every Role** (surface: serve) — 7/9 specs done
- **Hero Surface Polish — Ongoing Quality Pass on the Web Companion** (surface: serve) — 5/5 specs done
- **Hero Team Experience — Complete Multi-Developer Workflow** (surface: —) — 0/0 specs done
- **Install + Upgrade Contract Coverage — Prove Every Target Works Every Time** (surface: —) — 0/0 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **Single-Source Install — One Canonical Tree, Every Harness Reads It** (surface: core) — 5/6 specs done
- **Competitor Parity — Borrow the Best Ideas from Competing Tools** (surface: —) — 0/0 specs done · COMPLETED 2026-05-18
- **Web Surfaces Restructure — Make Docs and Landing Peers Under web/** (surface: —) — 0/0 specs done · COMPLETED 2026-05-18
- **Hero v2 — System Design Initiative** (surface: core, serve) — 8/8 specs done · COMPLETED 2026-05-18

## Recently completed (last 14 days)

- **(unassigned)** — peer-call-findings-truncation-and-persistence, peer-call-result-yaml-int-strict-parse, pm-platform-delivery, hero-code-handover-pack, scan-pluggability, domain-routing-and-agents, domain-scoped-knowledge-graph
- **core** — install-core-domain-merge
- **domains/engineering** — contentfs-legacy-fallback-removal
- **serve** — hero-serve-dashboard-redesign, hero-serve-multi-project, dashboard-now-headline-misleading-when-empty

## Next up across surfaces

1. **(unassigned)** — `e2e-discovery` (P0, delivering)
2. **(unassigned)** — `e2e-traversal` (P0, delivering)
3. **landing** — `hero-landing-page` (P0, delivering)
4. **serve** — `hero-sales` (P0, delivering)
5. **(unassigned)** — `master-ingest-restore` (P0, delivering)

## Open risks & blockers

- **Blocked specs (17):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `e2e-traversal` (waits on traversal-queries); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-serve-project-section-aggregate` (waits on hero-serve-project-section-mvp); `hero-serve-project-section-destructive` (waits on hero-serve-project-section-mvp, hero-serve-project-section-aggregate, hero-serve-project-section-opsrunner); `hero-serve-project-section-healthcache` (waits on hero-serve-project-section-mvp); `hero-serve-project-section-opsrunner` (waits on hero-serve-project-section-mvp); `hero-team-server` (waits on hero-runner); `timely-briefs` (waits on retrieval-contradiction-detection); `traversal-queries` (waits on master-ingest-restore); `unified-retrieval-layer` (waits on master-ingest-restore).
- **Unassigned specs (113) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 128/241 (53%)
- Projection generation: 0ms · Source nodes: 262

