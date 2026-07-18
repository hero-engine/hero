# Hero PM — Skills

Skills in this directory are PM-domain skill packages, loaded by
agents on demand. Each skill is a directory containing a `SKILL.md`
with YAML frontmatter (`name`, `description`, `compatibility`,
`metadata`) and a markdown body.

See the [agent-pack design](../../../.hero/planning/features/hero-pm/agent-pack-design.md)
§D for the origin skill library and prior-art attribution. The
`pm-pack-completion` initiative shipped the full library: **51 skills**
now live here, grouped below by concern.

### Writing (10)
- `story-writing-invest/` — INVEST shape for user stories, the bar every story clears before handoff
- `acceptance-criteria-ears/` — EARS patterns for AC that survive handoff without ambiguity
- `acceptance-criteria-gherkin/` — Given/When/Then as the alternate AC shape when a team prefers it over EARS
- `jtbd-job-stories/` — the Jobs-to-be-Done job-story shape and how it differs from an INVEST user story
- `story-mapping/` — Jeff Patton's story map — activity backbone, walking skeleton, release slices
- `epic-framing/` — frame an epic as a coherent bet, not a bag of stories
- `prd-structure/` — canonical PRD templates (pitch-shaped + ten-section) with a section-by-section quality bar
- `prd-anti-patterns/` — the PRD smells that make a document unusable, and what to write instead
- `pitch-writing-shape-up/` — Shape Up pitch authoring — appetite as budget, rabbit holes, no-gos, cooldown
- `risk-surfacing/` — name risks as scenario + indicator + response so a Risks section is decision-useful

### Discovery & framing (7)
- `opportunity-solution-trees-torres/` — Teresa Torres' OST — outcome → opportunities → solutions → assumption tests
- `continuous-discovery-cadence/` — Torres' weekly discovery rhythm — three touchpoints as habit, not project
- `discovery-interview-design/` — interviews that generate opportunity-space signal, not polite agreement
- `assumption-testing/` — fast Torres-style assumption tests with pre-registered pass/fail criteria
- `personas-and-journey-maps/` — evidence-based personas and the journey grid that feeds an OST
- `evidence-synthesis/` — turn raw signals into a roadmap evidence trail that survives "how do you know?"
- `opportunity-assessment/` — Cagan's 10-question assessment under single-challengeable-assumption discipline

### Strategy & narrative (4)
- `product-vision-writing/` — the one-page product vision that ladders strategy → roadmap
- `exec-narrative/` — Amazon's six-page narrative plus the "so what?" pressure-test — prose over slides
- `prfaq-writing/` — Amazon PR/FAQ working-backwards to surface the dragons while the bet is cheap to change
- `market-sizing/` — defensible TAM/SAM/SOM, top-down and bottom-up, every step on one challengeable assumption

### Competitive & positioning (3)
- `competitive-research/` — retrieval-augmented competitive teardown, never model-memory; every claim dated and sourced
- `feature-comparison-framing/` — a competitive matrix that drives a decision, not a checkbox arms race
- `positioning-canvas/` — April Dunford's five-component canvas run in order, so positioning is derived not asserted

### Prioritization (3)
- `prioritization-frameworks/` — RICE, ICE, WSJF, value-vs-effort — formulas, fit, and how soft inputs make scores lie
- `evidence-forcing/` — force every prioritization input to name evidence or default to neutral, then recompute
- `customer-segment-weighting/` — weight reach/impact by segment economics, recorded once and always disclosed

### Metrics & experiments (3)
- `metrics-design/` — leading vs lagging, observable, named baseline and target before commit
- `metric-rca/` — "why did the metric move" — metric-tree decomposition, drift taxonomy, causality-before-asserting
- `experiment-design/` — the pre-registered brief the readout-reviewer reads back

### Planning & cadence (6)
- `capacity-planning/` — per-preset capacity math and how the Story Queue cut-line is drawn
- `sprint-planning/` — sprint commit, velocity reading, cut-line decisions under the sprint preset
- `cycle-planning/` — Shape Up cycle planning — build+cooldown rhythm, betting table, appetite, hill chart
- `iteration-planning/` — the generic kanban/phased iteration shape — WIP limits, rolling commitment, phase gates
- `shape-up-cadence/` — the repeating operational rhythm of the cycle preset (distinct from cycle-planning mechanics)
- `hill-chart-reasoning/` — Basecamp's hill chart read correctly — unknowns-remaining, not percent-done

### Roadmap curation (4)
- `roadmap-framing/` — how to write initiatives that read honestly — bet-shaped, evidence-grounded, horizon-reconciled
- `horizon-assignment/` — the now/next/later discipline that keeps a roadmap honest and catches aspirational drift
- `outcomes-over-outputs/` — the outcome ladder and the ~60/30/10 framing ratio for reading a roadmap honestly
- `outcome-drift/` — the ratio tally + stale-item taxonomy behind the roadmap drift critic

### Curation & intake (3)
- `intake-classification/` — classify inbound by theme, segment, source quality, and signal strength
- `duplicate-detection/` — overlap signals beyond lexical similarity; never auto-merge
- `dependency-mapping/` — walk the dependency graph forward/backward, hard blockers vs soft sequencing

### Launch & communication (3)
- `launch-gtm-tiering/` — size a launch into Tier 1/2/3 and run the five-phase checklist scoped to the tier
- `release-notes-writing/` — the two release-note shapes, customer-facing and internal
- `stakeholder-communication/` — audience-shaped messaging cut for exec, customer, engineering, and sales

### Cross-domain (3)
- `handoff-protocol/` — the PM → engineering owner-flip protocol and the bitemporal ownership history it writes
- `cross-domain-graph-query/` — walking the graph across domain boundaries; the graph, not the tracker, is the truth surface
- `domain-glossary-maintenance/` — keep a shared PM/eng vocabulary alive as a knowledge entry

### Operational & doctrine (2)
- `pm-agent-doctrine/` — the pack-wide discipline every PM agent shares — corpus-grounded, human-gated, compare-don't-replace
- `pm-preset-detection/` — read `hero.json` `pm.presets` and apply the right authoring rules per artifact type
