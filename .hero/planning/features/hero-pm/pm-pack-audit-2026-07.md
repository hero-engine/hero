# PM Pack Audit & Inventory Refresh — 2026-07-17

**Trigger:** hero-code now embeds the Hero engine and serves agents/commands/skills
live per active domain. Running as PM, it reports the PM pack as "very light."

**Method:** three-way reconciliation —
1. **Designed-vs-shipped diff** against the locked `hero-pm` design (`agent-pack-design.md`, `spec.md`).
2. **hero-code surface map** — what the macOS + GTK consumers actually expose for PM and what they pull from the pack.
3. **External best-practice scan** — published PM agentic/prompt sets + skeptical practitioner corpus.

---

## Verdict

**The PM pack is not unbuilt — it is the v1 minimum-viable set the design deliberately shipped, and hero-code is now exercising the surfaces the design deferred.** Nothing is broken at the plumbing level (routing, vocabulary-awareness, methodology presets, and the `pm-pack-phantom-surfaces` reference fixes are all in). "Light" = three things:

1. **~Half the designed pack was never authored** (all P1/P2 items). The shipped roster is exactly the design's §H "minimum viable pack."
2. **Several *shipped* hero-code views have no backing agent** — buttons are drawn in the mockups with nothing behind them (see §Surface coverage).
3. **The design predates two things it should now answer to:** hero-code's real surfaces, and the external best-practice signal that the *differentiated* value is in critics/rigor-forcers, not the generators we prioritized in v1.

### Current state

| Surface | Designed (full) | Shipped (v1) | Missing |
|---|---|---|---|
| Agents | 27 (12 real P0 + 15 P1/P2) | 12 | 15 |
| Commands (PM-specific) | ~15 | 10 | 5 |
| Skills | 32 | 19 | 13 |

Engineering for contrast: 31 agents / 14 commands / 39 skills.

---

## The reframe (from the external scan — this reorders the build)

The consistent, well-sourced signal across published PM prompt packs, PM-SaaS AI features, and skeptical practitioner writing:

- **Generators are commoditized and *distrusted*.** Notion AI PRDs test ~70% right / ~30% generic-or-hallucinated; Dovetail's interview summaries fabricate quotes; generic-LLM PRDs "read like someone who read a lot of PRDs but never shipped." The failure mode is *confident, evidence-free output that looks like analysis and propagates into a roadmap, then gets cited in a QBR.*
- **The underserved, high-leverage capabilities are the "keep it honest" ones:** evidence-linked synthesis, roadmap-drift detection, anti-gaming prioritization critique, adversarial experiment readouts, metric-movement RCA.
- **Corpus grounding is the line between praised and gimmicky.** Every trusted feature (Linear triage, ChatPRD's MCP pull, Productboard clustering) grounds in the team's own feedback/calls/tracker/analytics. Every criticized one free-associates.
- **Suggest, don't decide.** Human-decision gates: marked, reversible, explainable, human-accountable. Never auto-decide prioritization or strategy.

**Implication for Hero:** our design maps generators→commands, mechanics→skills, and *reviewers* to agents — but frames the reviewers as passive quality gates. We should **elevate the critics into first-class adversarial agents** and make **corpus-grounding + decision-gate discipline a pack-wide doctrine skill**, not an afterthought. This is also consistent with our own note that non-engineering domains welcome heavier, more integrated assistance — but PM specifically wants that assistance pointed at *rigor*, not *authorship*.

---

## Wave plan

### Wave 0 — Wiring fixes (not content; unblock the consumer)

These are correctness bugs surfaced by hero-code embedding, not roster gaps.

| Item | Where | Fix |
|---|---|---|
| Stale manifest refs | hero-code `PMManifest.swift` declares agent/skill ids pointing at **engineering** agents (`product-ideator`, `design-reviewer`, `ui-designer`, `platform-delivery-lead`, `greenfield-architect`) | Repoint to the real PM roster (`product-strategist`, `pm-reviewer`, `roadmap-curator`…). Consumer-side; hand off to hero-code with the correct roster. |
| 6 referenced-but-undesigned skills | our `domains/pm/` agents load `outcomes-over-outputs`, `competitive-research`, `feature-comparison-framing`, `epic-framing`, `horizon-assignment`, `customer-segment-weighting` — none exist as files | Author them (see below). `outcomes-over-outputs` is the worst offender: 6 agents load it, and it's *the* spine framework per the external scan. Author it first. |
| `dashboard.md` orphan | on disk in `domains/pm/commands/`, no design entry | Reconcile — either document as a real command or remove. |
| GTK PM types | GTK boards recognize only engineering spec types; no `prd`/`story`/`roadmap-item`/`epic` renders first-class | Consumer-side (`gtk-m4-pm-surfaces`); flag that PM artifacts need vocab added or must express as `feature`/`initiative`. |

### Wave 1 — Author the deferred design items that back *live* hero-code surfaces

Every shipped view whose buttons currently have no backing agent. Highest urgency because the surface already exists in the consumer.

| Item | Type | Backs surface | Notes |
|---|---|---|---|
| `capacity-planner` | agent | Story Queue (velocity cut-line) | Story Queue view has **zero** backing agents today |
| `cycle-planner` | agent | Story Queue (cycle-fit marker) | one agent, preset-adaptive (sprint/cycle/iteration) |
| `dependency-mapper` | agent | Story Detail "Show dependencies" | |
| `stakeholder-communicator` | agent | PRD Editor "Summarize for standup" | |
| `pitch-author` | agent | PRD Editor "Convert to pitch" | split from prd-author |
| `duplicate-intake-scrubber` | agent | Intake Funnel "Cluster recent" | |
| `outcomes-over-outputs` | skill | roadmap/strategy/review views | load-bearing; author first |
| `capacity-planning`, `iteration-planning`, `shape-up-cadence` | skills | Story Queue / cycle planning | |
| `release-notes-writing`, `stakeholder-communication` | skills | PRD Editor / Handoff | |
| `/capacity`, `/plan-{cycle,sprint,iteration}`, `/scrub <concern>`, `/standup`, `/interview` | commands | Story Queue, Intake, cadence | the 5 deferred command files |

### Wave 2 — Differentiators the external scan ranks highest (mostly NET-NEW; not in our design)

Ordered by leverage × how-badly-served-today. These are what make the pack *worth switching to* rather than parity.

| Capability | Proposed surface(s) | Framework/artifact | In design? |
|---|---|---|---|
| **Roadmap-drift / outcome-vs-output signal memo** | sharpen `roadmap-reviewer` (P1) into a drift *critic*; new `outcome-drift` skill | outcome/output/input ratio (~60/30/10), stale-item flagging | partial (hygiene only) |
| **Anti-gaming prioritization critic** | new `prioritization-challenger` agent OR critic-mode on `prioritization-strategist`; new `evidence-forcing` skill | "Confidence needs named evidence or it's 50%" | **no** |
| **Evidence-linked interview synthesis** (compare-don't-replace, verbatim traceability, outlier-surfacing) | sharpen `discovery-researcher`; extend `evidence-synthesis` | Torres synthesize-then-compare; interview snapshot | partial |
| **Experiment brief + adversarial readout** | new `experiment-designer` + `experiment-readout-reviewer` agents; `experiment-design` skill; `/experiment` cmd | pre-registration, MDE, guardrails, SRM, no early-stopping | **no (whole stage absent)** |
| **"Why did the metric move" RCA** | `metrics-analyst` (P1) + new `metric-rca` skill | metric-tree decomposition, drift taxonomy | **no** |
| **Doc critic ("review as CPO")** | sharpen `pm-reviewer` into adversarial `pm-critic` | premortem, "5 reasons this won't work" | partial (passive gate) |
| **Exec "so what" / PR-FAQ working-backwards** | `stakeholder-communicator` + new `prfaq-writing` + `exec-narrative` skills | Amazon 6-pager / PR-FAQ | partial |
| **Defensible market sizing / opportunity assessment** | `product-strategist` + new `opportunity-assessment` (Cagan 10-Q) + `market-sizing` (TAM/SAM/SOM) skills | single-challengeable-assumption discipline | **no** |
| **Live-augmented competitive teardown** | `competitive-analyst` (P1) + `competitive-research` skill — **retrieval-augmented, never model-memory** | teardown + feature matrix + positioning | designed as role only |

### Wave 3 — Round out table-stakes coverage still missing, + remaining deferred roles

| Item | Type | Note |
|---|---|---|
| Personas & journey maps | skill (+ light command) | recurs across *every* published pack; we have none |
| JTBD job-stories | skill | `When [situation] I want [motivation] so [outcome]` — distinct from INVEST |
| April Dunford positioning canvas | skill | grounds positioning in real alternatives |
| Launch / GTM tiering | skill + `/launch` | tier 1/2/3, phased checklist |
| Story mapping | skill | nice-to-have; release slicing |
| `acceptance-criteria-gherkin` | skill | design P1 |
| `epic-framer`, `risk-curator`, `portfolio-curator`, `roadmap-reviewer` (base), `discovery-reviewer`, `stale-roadmap-scrubber`, `ambiguous-story-scrubber` | agents | remaining designed P1/P2 roles |
| `risk-surfacing`, `assumption-testing`, `discovery-interview-design`, `hill-chart-reasoning`, `domain-glossary-maintenance`, `product-vision-writing` | skills | remaining designed P1 skills |
| `okr-design` | skill | design P2; likely a future `strategy` domain, not PM |

---

## Cross-cutting doctrine to bake in (skill-level, applies to every agent)

1. **Corpus-grounding contract** — every PM agent cites the team's own feedback/calls/tracker/analytics; no free-association. (deanpeters' "evidence contracts + citations" pattern.)
2. **Decision gates** — suggestions are marked, reversible, explainable, human-accountable. Never auto-decide prioritization/strategy.
3. **Compare-don't-replace for synthesis** — the agent does its pass, the PM does theirs, they reconcile (protects against outsourcing judgment).

These belong in a `pm-agent-doctrine` skill (or folded into `pm-preset-detection` + `context-injection`) loaded by every authoring/critic agent.

---

## Surface coverage matrix (hero-code views → backing status)

| View | Backing agents | Status |
|---|---|---|
| Roadmap (default) | roadmap-curator, prioritization-strategist | ✅ shipped |
| Story Queue | capacity-planner, cycle-planner | ❌ **both missing** |
| PRD Editor | prd-author ✅; stakeholder-communicator ❌; pitch-author ❌ | ⚠️ partial |
| Story Detail | story-writer ✅, handoff-coordinator ✅, pm-reviewer ✅, duplicate-detector ✅; dependency-mapper ❌ | ⚠️ partial |
| Intake Funnel | intake-triager ✅, duplicate-detector ✅; duplicate-intake-scrubber ❌ | ⚠️ partial |
| Handoff Stream | handoff-coordinator ✅ | ✅ shipped |
| Chat | pm-delivery-lead ✅ | ✅ shipped |

---

## Recommended next step

This is initiative-sized (3 waves, ~15 agents + ~20 skills + 5 commands + net-new stages). Recommend `/compose` to sequence it as child specs, with Wave 0 + Wave 1 as the first deliverable sprint (unblocks hero-code), Wave 2 as the differentiation sprint, Wave 3 as coverage fill. The existing `hero-pm` design is the base; this audit is the refresh delta against it.

**Sources:** full external citation list in the research stream (Torres/producttalk, SVPG/Cagan, Amazon Working Backwards, Reforge, Lenny, deanpeters/product-manager-prompts, product-on-purpose/pm-skills, ChatPRD, BMAD, Intercom RICE, Mavin EARS, ProdPad Now-Next-Later, Amplitude NSM, Statsig/GrowthBook experimentation).
