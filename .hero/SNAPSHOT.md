# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-07-18T18:50:22Z · projected from 555 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | 2d ago | satellite-corpus-integration |
| docs | maturing | web/docs/ | 5d ago | — |
| domains/chat | maturing | domains/chat/ | 1h ago | — |
| domains/engineering | maturing | domains/engineering/ | 6d ago | — |
| domains/pm | maturing | domains/pm/ | 18h ago | — |
| domains/sales | maturing | domains/sales/ | 9d ago | — |
| landing | building | web/landing/ | 6d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 2d ago | agent-outposts |
| (unassigned) | — | — | — | 220 specs without surface |

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
- **"Hero-in-Hero-Code Parity — Fix Hero Workflow Integration in the Desktop App"** (surface: —) — 0/8 specs done
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

- **"PM Pack Completion — Author the Deferred Roster and Reframe Around Critics + Corpus-Grounding"** (surface: domains/pm) — 10/10 specs done · COMPLETED 2026-07-18
- **"Knowledge Surfacing — Everything Captured Is Retrievable and Fed at the Right Time"** (surface: core) — 4/4 specs done · COMPLETED 2026-07-10
- **Hero Surface Polish — Ongoing Quality Pass on the Web Companion** (surface: serve) — 5/5 specs done · COMPLETED 2026-07-10

## Recently completed (last 14 days)

- **(unassigned)** — chat-slim-to-basic-research-seed, pm-foundation-delivery, pm-pack-completion, discovery-framing-coverage-skills, exec-narrative-and-evidence-synthesis, story-detail-and-intake-scrubber-backing, prd-editor-comms-backing, experiment-stage-and-metric-rca
- **domains/chat** — chat-canonical-research
- **domains/pm** — story-queue-planning-backing, remaining-roles-scrubbers-and-launch, competitive-and-market-grounding

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **serve** — `hero-team-server` (P1, delivering)
3. **core** — `satellite-corpus-integration` (high, delivering)
4. **serve** — `agent-outposts` (medium, delivering)
5. **serve** — `retrieval-contradiction-detection` (—, delivering)

## Open risks & blockers

- **Blocked specs (13):** `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `timely-briefs` (waits on retrieval-contradiction-detection); `wire-checks-to-boundaries` (waits on spec-contract-enums-unified).
- **Stale-in-flight (2):** `hero-landing-page` (61d), `hero-team-server` (60d).
- **Aged open bugs (9):** `install-target-emits-both-claude-and-agents-md` (open 64d), `next-project-file-conflict-not-regenerated` (open 45d), `desktop-sidebar-mcp-not-running` (open 44d), `hihcp-agent-loop-error-recovery` (open 39d), `hihcp-agents-md-harness-agnostic` (open 39d), `hihcp-mcp-auto-reconnect` (open 39d), `hihcp-mcp-first-turn-readiness` (open 39d), `hihcp-permission-bridge-validation` (open 39d), `hihcp-rgignore` (open 39d).
- **Unassigned specs (220) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 197/417 (47%)
- Projection generation: 1ms · Source nodes: 555

