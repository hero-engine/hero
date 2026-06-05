# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-06-05T21:02:44Z · projected from 306 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | 1d ago | hero-local-merge-missing-dialect-fields |
| docs | concept | web/docs/ | — | — |
| domains/chat | concept | domains/chat/ | — | — |
| domains/engineering | building | domains/engineering/ | 4d ago | spec-types-cache-frontmatter-empty |
| domains/pm | concept | domains/pm/ | 18d ago | — |
| domains/sales | concept | domains/sales/ | — | — |
| landing | building | web/landing/ | 4d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 3d ago | vocabulary-resolve-misses-methodology-derivation |
| (unassigned) | — | — | — | 141 specs without surface |

_Run `hero snapshot assign` to bucket unassigned specs._

> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.

## Active initiatives

- **"Concurrent-Session Branching & Worktree Isolation"** (surface: —) — 0/0 specs done
- **Environment Awareness — CI/Deployment/Runtime Visibility** (surface: —) — 0/0 specs done
- **Get Back on Track — Mission-First V2 Recovery** (surface: core) — 3/11 specs done; in flight: master-ingest-restore, per-feature-smoke-coverage, spec-status-integrity, traversal-queries
- **Hero Domains — Platform Architecture for Non-Engineering Verticals** (surface: core, domains/engineering, domains/pm, serve) — 10/16 specs done; in flight: hero-sales
- **Hero Killer Features — Agent Effectiveness, Team Power, Living Specs** (surface: core, serve) — 10/11 specs done
- **Hero Marketing — Positioning, Distribution, and Launch** (surface: landing) — 0/9 specs done; in flight: hero-landing-page
- **Hero Platform — Headless Execution, Team Automation, and Shared Visibility** (surface: core, serve) — 2/7 specs done; in flight: hero-team-server
- **Hero Surface Architecture — One Surface, Every Layer, Every Role** (surface: serve) — 7/9 specs done
- **Hero Surface Polish — Ongoing Quality Pass on the Web Companion** (surface: serve) — 5/5 specs done
- **Hero Team Experience — Complete Multi-Developer Workflow** (surface: —) — 0/0 specs done
- **Install + Upgrade Contract Coverage — Prove Every Target Works Every Time** (surface: —) — 0/0 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **Single-Source Install — One Canonical Tree, Every Harness Reads It** (surface: core) — 5/6 specs done

### Recently completed initiatives

- **Roadmap Shape — Detect, Surface, and Resolve Spec-Shape Drift** (surface: serve) — 4/4 specs done · COMPLETED 2026-06-01
- **hero serve Project Section — Per-Project Info, Utilities, and Operations Page** (surface: serve) — 5/5 specs done · COMPLETED 2026-05-20
- **Hero v2 — System Design Initiative** (surface: core, serve) — 8/8 specs done · COMPLETED 2026-05-18

## Recently completed (last 14 days)

- **(unassigned)** — handoff-captures-session-intent, resume-brief-missing-project-context, cross-machine-handoff-slug-mismatch, resume-brief-surfaces-handoff, next-unconditional-commit-staging, e2e-handoff-continuity, next-snapshot-file-missing-merge-attribute, next-merge-driver-not-portable, next-projection-gate-punts-migration-to-user, next-team-mode-per-user-handoff-unmaintained, hero-mcp-orphan-no-parent-liveness
- **core** — next-auto-emit-user-ask

## Next up across surfaces

1. **(unassigned)** — `e2e-discovery` (P0, delivering)
2. **(unassigned)** — `e2e-traversal` (P0, delivering)
3. **landing** — `hero-landing-page` (P0, delivering)
4. **serve** — `hero-sales` (P0, delivering)
5. **(unassigned)** — `master-ingest-restore` (P0, delivering)

## Open risks & blockers

- **Blocked specs (14):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `e2e-traversal` (waits on traversal-queries); `embedded-inference` (waits on master-ingest-restore); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `timely-briefs` (waits on retrieval-contradiction-detection); `traversal-queries` (waits on master-ingest-restore); `unified-retrieval-layer` (waits on master-ingest-restore).
- **Stale-in-flight (21):** `hero-landing-page` (18d), `spec-status-integrity` (18d), `vocabulary-resolve-misses-methodology-derivation` (18d), `unified-search` (18d), `unified-retrieval-layer` (18d), `cross-repo-peering` (18d), `e2e-discovery` (18d), `e2e-traversal` (18d), `e2e-validation` (18d), `graph-conflict-detection` (18d), `spec-types-cache-frontmatter-empty` (18d), `tripwire-system` (18d), `hero-local-merge-missing-dialect-fields` (18d), `master-ingest-restore` (18d), `monorepo-satellite-installs` (18d), `per-feature-smoke-coverage` (18d), `traversal-queries` (18d), `hero-team-server` (17d), `hero-sales` (17d), `agent-outposts` (17d), `compact-handoff-test-coverage` (15d).
- **Aged open bugs (3):** `hero-workspace-not-self-describing` (open 24d), `scan-enrichment-unbounded-loop` (open 24d), `scan-test-detection-misses-spock-vitest` (open 24d).
- **Unassigned specs (141) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 143/284 (50%)
- Projection generation: 0ms · Source nodes: 306

