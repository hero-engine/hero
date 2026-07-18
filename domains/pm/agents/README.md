# Hero PM — Agents

Agents in this directory are the PM specialist roster. They are loaded
by the domain agent loader (per primitive #3,
`domain-routing-and-agents`) when the active domain in `hero.json` is
`pm`.

See the vertical charter at [`../mission.md`](../mission.md) for the
mission, and the [agent-pack design](../../../.hero/planning/features/hero-pm/agent-pack-design.md)
for the origin design, priorities, and prior-art attribution. The
`pm-pack-completion` initiative shipped the full roster: **30 agents**
now live here, listed below by role.

Each agent file follows the engineering pack's shape: YAML frontmatter
(`name`, `description`, `mode`, `temperature`, `color`, `permission`)
then a markdown body with role, when-invoked, workflow guidance, and
delegation rules.

### Coordination & delivery
| File | Role |
|---|---|
| `pm-delivery-lead.md` | Coordinate PM specialists to refine, prioritize, hand off, and ship product-management work |
| `pm-investigator.md` | Investigate ambiguous intake and vague asks to identify the underlying opportunity before authoring |
| `handoff-coordinator.md` | Execute the PM → engineering handoff as an owner flip on the same spec — **brand interaction** (no second spec is created) |

### Strategy & research
| File | Role |
|---|---|
| `product-strategist.md` | Frame roadmap-level bets in outcomes, opportunities, and tradeoffs — owns "why this and not that" |
| `discovery-researcher.md` | Design and synthesize customer research (Torres continuous discovery); writes findings into PRDs and initiatives |
| `competitive-analyst.md` | Retrieval-only competitive teardown — parity/differentiation/white-space matrix, positioning read; refuses training-data recollection |
| `metrics-analyst.md` | Define success metrics and run "why did the metric move" RCA — metric-tree decomposition, drift taxonomy, causality-before-asserting |
| `portfolio-curator.md` | Reconcile cross-roadmap theme balance and capacity-vs-ambition; recommends rebalances, never auto-rebalances |

### Authoring
| File | Role |
|---|---|
| `prd-author.md` | Produce and refine PRD specs — pitch-shaped under cycle preset, ten-section under sprint/continuous/phased |
| `pitch-author.md` | Shape a Shape Up pitch — appetite as budget, rabbit holes as named traps, no-gos as scope defense; refuses empty Appetite/No-Gos |
| `story-writer.md` | Produce and refine features (canonical `type: feature`) to INVEST shape with EARS acceptance criteria. Vocabulary-aware display ("Story Writer" / "Scope Author" / "Spec Writer"). The highest-volume PM authoring agent |
| `epic-framer.md` | Frame an epic as a coherent bet — the Why, the rollup acceptance criteria, sequenced children; delegates story bodies to story-writer |
| `roadmap-curator.md` | Maintain the roadmap board — horizon assignments, delivery-state reconciliation against live engineering reality, stale-item surfacing |
| `risk-curator.md` | Surface risks on PRDs, roadmap-items, and stories as scenario + indicator + response — never generic boilerplate |

### Triage & curation
| File | Role |
|---|---|
| `intake-triager.md` | Process inbound signals into triaged intakes — linked, merged, or rejected with reason — within 24 hours |
| `duplicate-detector.md` | Detect near-duplicate intakes, initiatives, and stories at write-time; ranked candidates with field-overlap evidence, never auto-merges |
| `dependency-mapper.md` | Surface dependencies across items, epics, and stories — including cross-domain chains into engineering; proposes, never auto-edits |

### Prioritization
| File | Role |
|---|---|
| `prioritization-strategist.md` | Apply prioritization frameworks (RICE / ICE / WSJF / value-vs-effort) to initiatives and stories |

### Planning & delivery cadence
| File | Role |
|---|---|
| `capacity-planner.md` | Reconcile committed work against team capacity under the active preset and place the Story Queue velocity cut-line; recommends, never auto-commits |
| `cycle-planner.md` | One preset-adaptive planner (sprint / cycle / iteration) that plans the next iteration and powers the Story Queue cycle-fit marker; recommends, never auto-commits |

### Experiments & metrics
| File | Role |
|---|---|
| `experiment-designer.md` | Design falsifiable experiments — the pre-registered brief fixing hypothesis, primary metric + MDE, duration, guardrails, decision/stop rule |

### Communication
| File | Role |
|---|---|
| `stakeholder-communicator.md` | Translate a PM artifact into audience-shaped cuts — exec, customer, internal — backing /standup and /release-notes |

### Review & critics (adversarial)
| File | Role |
|---|---|
| `pm-reviewer.md` | Review PM artifacts (PRDs, stories, epics, initiatives, intakes) for quality before they advance — analog to design-reviewer / pr-reviewer |
| `discovery-reviewer.md` | Adversarial rigor review of discovery artifacts — OSTs, interview synthesis, assumption tests; report-only, routes back to the author |
| `experiment-readout-reviewer.md` | Adversarial experiment-readout critic — argues the strongest false-positive case (SRM, peeking, guardrail regressions, multiple comparisons) against the pre-registered brief |
| `prioritization-challenger.md` | Anti-gaming prioritization critic — stress-tests RICE/ICE/WSJF inputs, defaults unsupported inputs to neutral, recomputes; interrogates, does not rank |
| `roadmap-reviewer.md` | Adversarial roadmap drift critic — audits for outcome-vs-output drift (~60/30/10), stale items, and reality-contradicting claims, grounded in delivery state |

### Scrubbers (report-only sweeps)
| File | Role |
|---|---|
| `duplicate-intake-scrubber.md` | Cluster a window of recent intake to surface near-duplicates the write-time detector missed; recommends a canonical survivor per cluster, no auto-merge |
| `stale-roadmap-scrubber.md` | Sweep the roadmap for items that haven't moved in N weeks, shipped items still marked active, and over-horizon `later` items; recommends archive / drop / refresh |
| `ambiguous-story-scrubber.md` | Sweep `ready` stories for INVEST / EARS failures — the ones that cause friction at handoff — and flag each with its specific failure and recommended refinement |

**Note (pack filename).** `story-writer.md` is the canonical pack
filename; a follow-up may rename it to `spec-writer.md`. Its display
name is vocabulary-aware (see the Authoring table); the canonical
frontmatter it authors always says `type: feature` (or `type: bug` /
`type: chore` / …) regardless of display name.
