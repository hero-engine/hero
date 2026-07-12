# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-07-12T17:06:33Z · projected from 404 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | 1h ago | satellite-corpus-integration |
| docs | concept | web/docs/ | — | — |
| domains/chat | concept | domains/chat/ | — | — |
| domains/engineering | maturing | domains/engineering/ | 21h ago | — |
| domains/pm | maturing | domains/pm/ | 3d ago | — |
| domains/sales | maturing | domains/sales/ | 3d ago | — |
| landing | building | web/landing/ | 21h ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 11h ago | agent-outposts |
| (unassigned) | — | — | — | 193 specs without surface |

_Run `hero snapshot assign` to bucket unassigned specs._

> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.

## Active initiatives

- **"Cold-Start Trust Hardening — Fail Loud, Never Mislead, at First Use"** (surface: core) — 2/2 specs done
- **"Concurrent-Session Branching & Worktree Isolation"** (surface: —) — 0/0 specs done
- **"Context Engine v2 — Fix and Optimize hero-code Desktop Context Curation"** (surface: —) — 4/8 specs done
- **Environment Awareness — CI/Deployment/Runtime Visibility** (surface: —) — 0/0 specs done
- **Get Back on Track — Mission-First V2 Recovery** (surface: core) — 7/11 specs done
- **Hero Domains — Platform Architecture for Non-Engineering Verticals** (surface: core, domains/engineering, domains/pm, domains/sales) — 11/16 specs done
- **"Hero-in-Hero-Code Parity — Fix Hero Workflow Integration in the Desktop App"** (surface: —) — 0/8 specs done
- **Hero Killer Features — Agent Effectiveness, Team Power, Living Specs** (surface: core, serve) — 10/11 specs done
- **Hero Marketing — Positioning, Distribution, and Launch** (surface: landing) — 0/9 specs done; in flight: hero-landing-page
- **Hero Platform — Headless Execution, Team Automation, and Shared Visibility** (surface: core, serve) — 2/7 specs done; in flight: hero-team-server
- **Hero Surface Architecture — One Surface, Every Layer, Every Role** (surface: serve) — 8/9 specs done
- **Hero Team Experience — Complete Multi-Developer Workflow** (surface: —) — 0/1 specs done
- **Install + Upgrade Contract Coverage — Prove Every Target Works Every Time** (surface: —) — 0/0 specs done
- **Launch Readiness — Telemetry, Deploy, and Public-Use Polish** (surface: —) — 0/0 specs done
- **Pre-Launch Hardening — Federation Polish, Security, Observability** (surface: —) — 0/0 specs done
- **"Retrieval Quality — Reranking, Expansion & Feedback Loop"** (surface: —) — 0/0 specs done
- **Single-Source Install — One Canonical Tree, Every Harness Reads It** (surface: core) — 5/6 specs done; in flight: single-source-install-p1-agents-md

### Recently completed initiatives

- **"Knowledge Surfacing — Everything Captured Is Retrievable and Fed at the Right Time"** (surface: core) — 4/4 specs done · COMPLETED 2026-07-10
- **Hero Surface Polish — Ongoing Quality Pass on the Web Companion** (surface: serve) — 5/5 specs done · COMPLETED 2026-07-10
- **Content Remediation — Audit Follow-Through Across Packs, Gates, and Harnesses** (surface: core, domains/engineering, domains/pm, domains/sales) — 8/8 specs done · COMPLETED 2026-07-10

## Recently completed (last 14 days)

- **(unassigned)** — incremental-scan-complete-result, incremental-scan-prunes-unchanged-packages, codescan-created-slug-frontmatter-followup, knowledge-surfacing, eventslog-churn-fix-tracked-not-dirty
- **core** — scan-clobbers-authored-created-slug-frontmatter, next-drift-gate-unwinnable, created-field-stamp-and-surface, install-json-mode-repair-migrate-parity, flat-tripwire-trigger-parity, flat-named-spec-discovery
- **serve** — opsrunner-keepalive-data-race

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **core** — `single-source-install-p1-agents-md` (P0, delivering)
3. **serve** — `hero-team-server` (P1, delivering)
4. **core** — `satellite-corpus-integration` (high, delivering)
5. **serve** — `agent-outposts` (medium, delivering)

## Open risks & blockers

- **Blocked specs (12):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection).
- **Stale-in-flight (2):** `hero-landing-page` (54d), `hero-team-server` (54d).
- **Aged open bugs (9):** `install-target-emits-both-claude-and-agents-md` (open 58d), `next-project-file-conflict-not-regenerated` (open 39d), `desktop-sidebar-mcp-not-running` (open 38d), `hihcp-agent-loop-error-recovery` (open 33d), `hihcp-agents-md-harness-agnostic` (open 33d), `hihcp-mcp-auto-reconnect` (open 33d), `hihcp-mcp-first-turn-readiness` (open 33d), `hihcp-permission-bridge-validation` (open 33d), `hihcp-rgignore` (open 33d).
- **Unassigned specs (193) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 182/375 (48%)
- Projection generation: 1ms · Source nodes: 404

