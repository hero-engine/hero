# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-06-25T00:02:19Z · projected from 437 source nodes_
_Last refreshed: 2026-06-25T00:15:08Z · projected from 338 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | maturing | cmd/, internal/ | 1m ago | cli-sync-resilience |
| core | maturing | cmd/, internal/ | 5m ago | hero-runner |
| docs | concept | web/docs/ | — | — |
| domains/chat | concept | domains/chat/ | — | — |
| domains/engineering | maturing | domains/engineering/ | 12d ago | — |
| domains/pm | concept | domains/pm/ | 35d ago | — |
| domains/sales | maturing | domains/sales/ | 12d ago | — |
| landing | building | web/landing/ | 20d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 11m ago | agent-outposts |
| (unassigned) | — | — | — | 164 specs without surface |
| serve | building | internal/serve/ | 6h ago | agent-outposts |
| (unassigned) | — | — | — | 165 specs without surface |

_Run `hero snapshot assign` to bucket unassigned specs._

> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.

## Active initiatives

- **"Cold-Start Trust Hardening — Fail Loud, Never Mislead, at First Use"** (surface: core) — 2/2 specs done
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
- **Hero Team Experience — Complete Multi-Developer Workflow** (surface: —) — 0/1 specs done
- **Install + Upgrade Contract Coverage — Prove Every Target Works Every Time** (surface: —) — 0/0 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **"Retrieval Quality — Reranking, Expansion & Feedback Loop"** (surface: —) — 0/0 specs done
- **Single-Source Install — One Canonical Tree, Every Harness Reads It** (surface: core) — 5/6 specs done

### Recently completed initiatives

- **"Feature Knowledge Synthesis — Auto-Generated 'How It Works' Entries from Shipped Work"** (surface: —) — 5/5 specs done · COMPLETED 2026-06-23
- **Roadmap Shape — Detect, Surface, and Resolve Spec-Shape Drift** (surface: serve) — 4/4 specs done · COMPLETED 2026-06-01
- **hero serve Project Section — Per-Project Info, Utilities, and Operations Page** (surface: serve) — 5/5 specs done · COMPLETED 2026-05-20

## Recently completed (last 14 days)

- **(unassigned)** — cold-start-trust-hardening, cloud-cli-verify, fks-feature-knowledge-artifact, fks-trust-handshake, fks-cluster-detection, scan-enrichment-unbounded-loop, fks-living-doc-amendment
- **core** — mock-export-cli, knowledge-export-cli, cst-initiative-premature-autocomplete, cst-verify-lifecycle-scoping, scan-test-detection-misses-spock-vitest
- **(unassigned)** — cloud-cli-verify, fks-on-demand-synthesizer, spec-lifecycle-hygiene-breakdown, scan-enrichment-unbounded-loop, fks-cluster-detection, fks-feature-knowledge-artifact, fks-living-doc-amendment
- **core** — cli-sync-resilience, cst-initiative-premature-autocomplete, cst-verify-lifecycle-scoping, scan-test-detection-misses-spock-vitest
- **serve** — claim-matches-sentinel-collision

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **serve** — `hero-team-server` (P1, delivering)
3. **serve** — `agent-outposts` (medium, delivering)
4. **(unassigned)** — `team-connect` (—, delivering)
5. **(unassigned)** — `team-oauth` (—, delivering)

## Open risks & blockers

- **Blocked specs (12):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection).
- **Stale-in-flight (3):** `agent-outposts` (35d), `hero-landing-page` (35d), `hero-team-server` (35d).
- **Aged open bugs (2):** `next-project-file-conflict-not-regenerated` (open 22d), `desktop-sidebar-mcp-not-running` (open 21d).
- **Unassigned specs (164) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.
- **Aged open bugs (2):** `next-project-file-conflict-not-regenerated` (open 22d), `desktop-sidebar-mcp-not-running` (open 21d).
- **Unassigned specs (165) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 153/317 (48%)
- Projection generation: 1ms · Source nodes: 437
- Specs covered: 151/316 (47%)
- Projection generation: 0ms · Source nodes: 338

