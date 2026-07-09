# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-07-08T23:43:56Z · projected from 388 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | 5h ago | flat-named-spec-discovery |
| docs | concept | web/docs/ | — | — |
| domains/chat | concept | domains/chat/ | — | — |
| domains/engineering | maturing | domains/engineering/ | 2m ago | routing-file-completeness |
| domains/pm | maturing | domains/pm/ | 2d ago | — |
| domains/sales | maturing | domains/sales/ | 1d ago | — |
| landing | building | web/landing/ | 23d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 10d ago | agent-outposts |
| (unassigned) | — | — | — | 188 specs without surface |

_Run `hero snapshot assign` to bucket unassigned specs._

> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.

## Active initiatives

- **"Cold-Start Trust Hardening — Fail Loud, Never Mislead, at First Use"** (surface: core) — 2/2 specs done
- **"Concurrent-Session Branching & Worktree Isolation"** (surface: —) — 0/0 specs done
- **Content Remediation — Audit Follow-Through Across Packs, Gates, and Harnesses** (surface: core, domains/engineering) — 3/4 specs done
- **"Context Engine v2 — Fix and Optimize hero-code Desktop Context Curation"** (surface: —) — 4/8 specs done
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
- **"Knowledge Surfacing — Everything Captured Is Retrievable and Fed at the Right Time"** (surface: —) — 3/4 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **"Retrieval Quality — Reranking, Expansion & Feedback Loop"** (surface: —) — 0/0 specs done
- **Single-Source Install — One Canonical Tree, Every Harness Reads It** (surface: core) — 5/6 specs done

### Recently completed initiatives

- **"Tracker Fixtures — GitLab Parity and Offline Mock Server for Round-Trip Validation"** (surface: —) — 2/2 specs done · COMPLETED 2026-07-02
- **"Drive — Autonomous Initiative Execution with a Human-Boundary Predicate"** (surface: core, domains/engineering, serve) — 6/6 specs done · COMPLETED 2026-06-28
- **"Feature Knowledge Synthesis — Auto-Generated 'How It Works' Entries from Shipped Work"** (surface: —) — 5/5 specs done · COMPLETED 2026-06-23

## Recently completed (last 14 days)

- **(unassigned)** — knowledge-context-injection, knowledge-content-retrieval, inline-flow-relations-dropped, hero-content-audit, cev2-corpus-reconcile-to-v3, team-oauth
- **core** — delivery-gate-consistency
- **domains/engineering** — harness-agnosticism-sweep, core-commands-domain-neutral
- **domains/pm** — pm-pack-phantom-surfaces, content-dedup-resync
- **domains/sales** — sales-pack-reality-sync

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **core** — `flat-named-spec-discovery` (high, delivering)
3. **serve** — `hero-team-server` (P1, delivering)
4. **serve** — `agent-outposts` (medium, delivering)
5. **(unassigned)** — `team-connect` (—, delivering)

## Open risks & blockers

- **Blocked specs (12):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection).
- **Stale-in-flight (3):** `agent-outposts` (23d), `hero-landing-page` (23d), `hero-team-server` (23d).
- **Aged open bugs (8):** `next-project-file-conflict-not-regenerated` (open 35d), `desktop-sidebar-mcp-not-running` (open 34d), `hihcp-agent-loop-error-recovery` (open 29d), `hihcp-agents-md-harness-agnostic` (open 29d), `hihcp-mcp-auto-reconnect` (open 29d), `hihcp-mcp-first-turn-readiness` (open 29d), `hihcp-permission-bridge-validation` (open 29d), `hihcp-rgignore` (open 29d).
- **Unassigned specs (188) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 175/363 (48%)
- Projection generation: 1ms · Source nodes: 388

