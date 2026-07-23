# Project Snapshot — hero

> Hero is the sidekick brain for AI-augmented knowledge work.

_Last refreshed: 2026-07-23T01:34:36Z · projected from 582 source nodes_

## Surfaces

| Surface | Stage | Path(s) | Last touched | Driver spec |
|---|---|---|---|---|
| core | building | cmd/, internal/ | <1m ago | satellite-corpus-integration |
| docs | maturing | web/docs/ | 10d ago | — |
| domains/chat | maturing | domains/chat/ | 4d ago | — |
| domains/engineering | maturing | domains/engineering/ | 1h ago | deferred-work-suggestion-contract |
| domains/pm | maturing | domains/pm/ | 5d ago | — |
| domains/sales | maturing | domains/sales/ | 14d ago | — |
| landing | building | web/landing/ | 11d ago | hero-landing-page |
| mcp | concept | internal/serve/mcp*.go | — | — |
| serve | building | internal/serve/ | 1h ago | agent-outposts |
| (unassigned) | — | — | — | 231 specs without surface |

_Run `hero snapshot assign` to bucket unassigned specs._

> **No release model declared.** Add `release_target:` to a spec or initiative, or configure tracker integration, to enable the initial-release rollup column.

## Active initiatives

- **"Always-On Runtime"** (surface: serve) — 0/1 specs done
- **"Cold-Start Trust Hardening — Fail Loud, Never Mislead, at First Use"** (surface: core) — 2/2 specs done
- **"Concurrent-Session Branching & Worktree Isolation"** (surface: —) — 0/0 specs done
- **"Context Engine v2 — Fix and Optimize hero-code Desktop Context Curation"** (surface: —) — 4/8 specs done
- **"Durable Attention — Project Mail, Personal Focus, and Explicit Promotion"** (surface: core, domains/engineering, serve) — 2/7 specs done
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

- **"Tracker Source Fidelity — Canonical Jira Markdown and Lazy Evidence Provenance"** (surface: hero-engine-tracker) — 2/2 specs done · COMPLETED 2026-07-22
- **"PM Pack Completion — Author the Deferred Roster and Reframe Around Critics + Corpus-Grounding"** (surface: domains/pm) — 10/10 specs done · COMPLETED 2026-07-18
- **"Knowledge Surfacing — Everything Captured Is Retrievable and Fed at the Right Time"** (surface: core) — 4/4 specs done · COMPLETED 2026-07-10

## Recently completed (last 14 days)

- **(unassigned)** — tracker-source-reconciliation, tracker-project-snapshot-contract, tracker-source-fidelity-and-evidence, brokered-tracker-agent-access, jira-import-per-type-filter-and-body-parity, tracker-diagnosis-full-ticket-evidence
- **core** — personal-focus-core, durable-attention-contracts, broker-issue-id-validator-raw-string-regression
- **hero-engine-tracker** — lazy-tracker-evidence-sidecar, jira-adf-description-fidelity-loss
- **serve** — serve-auto-refresh-bypasses-bulk-import

## Next up across surfaces

1. **landing** — `hero-landing-page` (P0, delivering)
2. **serve** — `hero-team-server` (P1, delivering)
3. **core** — `satellite-corpus-integration` (high, delivering)
4. **serve** — `agent-outposts` (medium, delivering)
5. **serve** — `retrieval-contradiction-detection` (—, delivering)

## Open risks & blockers

- **Blocked specs (16):** `attention-read-model-v1` (waits on project-mail-triage-and-provenance, deferred-work-suggestion-contract); `core-vertical-layering` (waits on project-charter); `e2e-area-suites` (waits on project-charter); `hero-community-edition` (waits on hero-governance); `hero-content-engine` (waits on hero-positioning, hero-docs-site); `hero-demo-content` (waits on hero-positioning); `hero-docs-site` (waits on hero-positioning); `hero-landing-page` (waits on hero-positioning, hero-distribution, hero-demo-content); `hero-launch-playbook` (waits on hero-positioning, hero-landing-page, hero-distribution, hero-demo-content); `hero-team-server` (waits on hero-runner); `hihcp-agent-loop-error-recovery` (waits on hihcp-mcp-first-turn-readiness, hihcp-mcp-auto-reconnect); `hihcp-agents-md-harness-agnostic` (waits on hihcp-skill-run-tool); `peering-over-project-mail` (waits on project-mail-triage-and-provenance); `project-mail-triage-and-provenance` (waits on project-mail-core, hero-idea-primitive-core); `timely-briefs` (waits on retrieval-contradiction-detection); `wire-checks-to-boundaries` (waits on spec-contract-enums-unified).
- **Stale-in-flight (2):** `hero-landing-page` (65d), `hero-team-server` (65d).
- **Aged open bugs (9):** `install-target-emits-both-claude-and-agents-md` (open 69d), `next-project-file-conflict-not-regenerated` (open 50d), `desktop-sidebar-mcp-not-running` (open 49d), `hihcp-agent-loop-error-recovery` (open 44d), `hihcp-agents-md-harness-agnostic` (open 44d), `hihcp-mcp-auto-reconnect` (open 44d), `hihcp-mcp-first-turn-readiness` (open 44d), `hihcp-permission-bridge-validation` (open 44d), `hihcp-rgignore` (open 44d).
- **Unassigned specs (231) — no `surface:` declared.** Run `hero snapshot assign` to bucket them.

## Snapshot health

- Surfaces detected: 9 (inferred: 9 · overrides applied: 0)
- Specs covered: 210/441 (47%)
- Projection generation: 1ms · Source nodes: 582

