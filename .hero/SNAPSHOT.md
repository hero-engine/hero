# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-06-09T15:46:36Z · projected from 308 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | 3m ago | codex-install-broken |
| docs | concept | web/docs/ | — | — |
| domains/chat | concept | domains/chat/ | — | — |
| domains/engineering | building | domains/engineering/ | 2d ago | spec-types-cache-frontmatter-empty |
| domains/pm | concept | domains/pm/ | 21d ago | — |
| domains/sales | concept | domains/sales/ | — | — |
| landing | building | web/landing/ | 8d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 7d ago | vocabulary-resolve-misses-methodology-derivation |
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

1. **core** — `codex-install-broken` (P0, delivering)
2. **domains/engineering** — `delivery-gate-enforcement` (P0, delivering)
3. **(unassigned)** — `e2e-discovery` (P0, delivering)
4. **(unassigned)** — `e2e-traversal` (P0, delivering)
5. **landing** — `hero-landing-page` (P0, delivering)

## Open risks & blockers

- **Blocked specs (14):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `e2e-traversal` (waits on traversal-queries); `embedded-inference` (waits on master-ingest-restore); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `timely-briefs` (waits on retrieval-contradiction-detection); `traversal-queries` (waits on master-ingest-restore); `unified-retrieval-layer` (waits on master-ingest-restore).
- **Stale-in-flight (21):** `hero-landing-page` (21d), `hero-sales` (21d), `vocabulary-resolve-misses-methodology-derivation` (21d), `agent-outposts` (21d), `unified-search` (21d), `cross-repo-peering` (21d), `e2e-discovery` (21d), `e2e-traversal` (21d), `e2e-validation` (21d), `graph-conflict-detection` (21d), `spec-types-cache-frontmatter-empty` (21d), `hero-team-server` (21d), `hero-local-merge-missing-dialect-fields` (21d), `master-ingest-restore` (21d), `monorepo-satellite-installs` (21d), `per-feature-smoke-coverage` (21d), `spec-status-integrity` (21d), `traversal-queries` (21d), `tripwire-system` (21d), `unified-retrieval-layer` (21d), `compact-handoff-test-coverage` (19d).
- **Aged open bugs (10):** `hero-workspace-not-self-describing` (open 28d), `scan-enrichment-unbounded-loop` (open 28d), `scan-test-detection-misses-spock-vitest` (open 28d), `hero-local-merge-missing-dialect-fields` (open 23d), `initiative-required-sections-drift` (open 23d), `spec-types-cache-frontmatter-empty` (open 23d), `vocabulary-resolve-misses-methodology-derivation` (open 23d), `spec-lifecycle-hygiene-breakdown` (open 22d), `claim-matches-sentinel-collision` (open 21d), `install-target-emits-both-claude-and-agents-md` (open 21d).
- **Unassigned specs (141) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 145/286 (50%)
- Projection generation: 0ms · Source nodes: 308

