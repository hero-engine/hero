# Hero PM — Commands

Slash commands in this directory are PM workflows, loaded by the
domain command loader when the active domain in `hero.json` is `pm`.

See the [agent-pack design](../../../.hero/planning/features/hero-pm/agent-pack-design.md)
§E for the origin command design and prior-art attribution. The
`pm-pack-completion` initiative shipped the full set: **19 PM-specific
commands** live here, plus the reused cross-domain / core commands
listed at the bottom.

Each command file follows the engineering pack's shape: YAML
frontmatter (`description` only) then a markdown body that scopes the
workflow and names the delegated agent(s).

### Authoring & refinement
| File | Routes to | Purpose |
|---|---|---|
| `refine.md` | `pm-delivery-lead` → `story-writer` / `prd-author` / `epic-framer` / `product-strategist` (by spec `type`) | Refine a PM artifact (feature, PRD, epic, initiative) toward delivery readiness |
| `prd.md` | `prd-author` | Draft a new PRD or refine an existing one — preset-aware (pitch under cycle, ten-section under sprint/flow) |
| `pitch.md` | `pitch-author` | Draft a Shape Up pitch — Problem, Appetite, Solution sketch, Rabbit holes, No-Gos |

### Discovery & framing
| File | Routes to | Purpose |
|---|---|---|
| `discover.md` | `discovery-researcher` | Continuous-discovery kickoff or check-in — opportunity solution trees, interview design, assumption tests |
| `interview.md` | `discovery-researcher` | Design a customer interview guide — open, story-based, non-leading questions with a synthesis plan |

### Prioritization
| File | Routes to | Purpose |
|---|---|---|
| `prioritize.md` | `prioritization-strategist` | Rank a set of initiatives or specs using a framework (value-vs-effort, RICE, ICE, WSJF) |

### Planning & cadence
| File | Routes to | Purpose |
|---|---|---|
| `capacity.md` | `capacity-planner` | Reconcile committed work against capacity under the active preset and place the Story Queue cut-line — a proposal, never an auto-commit |
| `plan-sprint.md` | `cycle-planner` | Scrum sprint entry point into the one preset-adaptive planner (velocity + commit/stretch); recommends, never auto-commits |
| `plan-cycle.md` | `cycle-planner` | Shape Up cycle entry point into the one preset-adaptive planner (betting table + appetite + cooldown); recommends, never auto-commits |
| `plan-iteration.md` | `cycle-planner` | Kanban/phased entry point into the one preset-adaptive planner (WIP + rolling commit + phase gates); recommends, never auto-commits |

### Experiments & metrics
| File | Routes to | Purpose |
|---|---|---|
| `experiment.md` | `experiment-designer` | Design a pre-registered experiment brief — single-variable hypothesis, primary metric + MDE, duration, guardrails, decision/stop rule, locked before launch |
| `metrics.md` | `metrics-analyst` | Define success metrics for a PRD or initiative (current → target, leading-not-lagging), or run metric-movement RCA |

### Triage & curation
| File | Routes to | Purpose |
|---|---|---|
| `triage.md` | `intake-triager` (+ `duplicate-detector`, `pm-investigator` when needed) | Process inbound intakes into a triaged status — linked, merged, promoted, or rejected with reason |
| `scrub.md` | concern-dispatched: `intake` → `duplicate-intake-scrubber`, `roadmap` → `stale-roadmap-scrubber`, `stories` → `ambiguous-story-scrubber` | Sweep a workspace concern for accumulated quality issues — all three concerns report-only |

### Roadmap
| File | Routes to | Purpose |
|---|---|---|
| `roadmap.md` | `roadmap-curator` (on `--reconcile`; navigate-only otherwise, no agent) | Open or reconcile the roadmap — navigate the board, drill into an item, or reconcile against live engineering delivery |

### Launch & communication
| File | Routes to | Purpose |
|---|---|---|
| `launch.md` | `stakeholder-communicator` (per-phase owners where they differ) | Produce a tiered launch plan + five-phase checklist for a roadmap-item or PRD, scoped to the detected launch tier |
| `standup.md` | `stakeholder-communicator` | Compose a standup update from intra-cycle graph changes — what moved since the last standup, cut for the internal team |
| `release-notes.md` | `stakeholder-communicator` | Draft customer-facing (or internal) release notes for shipped items — pulls shipped status from the cross-domain graph |
| `handoff.md` | `handoff-coordinator` | Hand a refined spec off to engineering — flips `owner: pm → engineering` on the same artifact. No new spec is created — the cross-domain owner-flip **brand interaction** |

### Reused (cross-domain / core)
- `/why` — multi-hop trace across the spec hierarchy + bitemporal `owner_history` rows that show cross-domain ownership transitions
- `hero search` (CLI; no pack ships a `/search` command) — cross-domain search (results render `owner` so the PM/engineering boundary is visible)
- `/note` — note capture
- `/decide` — architectural/product decision capture (ADR-shaped)
- `/blocked` — surface items that can't move forward across the dependency tree
- `/retro` — post-delivery retrospective
- `/deliver` — engineering-pack command — runs on the engineering side after the owner flip, not in a pm install; runs against the same spec PM authored
