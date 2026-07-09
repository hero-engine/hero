# Hero PM — Skills

Skills in this directory are PM-domain skill packages, loaded by
agents on demand. Each skill is a directory containing a `SKILL.md`
with YAML frontmatter (`name`, `description`, `compatibility`,
`metadata`) and a markdown body.

See the [agent-pack design](../../../.hero/planning/features/hero-pm/agent-pack-design.md)
§D for the full 32-skill library and prior-art attribution. The v1
P0 skill set (~18 skills) shipped here:

### Writing (6)
- `story-writing-invest/` — INVEST shape for stories
- `acceptance-criteria-ears/` — EARS patterns for AC
- `prd-structure/` — pitch-shaped + ten-section PRD templates
- `prd-anti-patterns/` — the smells that make a PRD unusable
- `pitch-writing-shape-up/` — Shape Up pitch authoring
- `roadmap-framing/` — how to write an initiative that reads honestly

### Frameworks (3)
- `prioritization-frameworks/` — RICE, ICE, WSJF, value-vs-effort
- `opportunity-solution-trees-torres/` — Torres' OST mapping
- `metrics-design/` — leading metrics, baselines, target-naming

### Process / methodology (3)
- `continuous-discovery-cadence/` — Torres' weekly habits
- `sprint-planning/` — sprint commit, velocity reading
- `cycle-planning/` — Shape Up cycle / cooldown rhythm

### Curation (4)
- `intake-classification/` — classifying inbound by theme / segment
- `duplicate-detection/` — overlap signals beyond similarity score
- `dependency-mapping/` — hard blockers vs soft sequencing
- `evidence-synthesis/` — turning quotes / data into roadmap evidence

### Cross-domain (2)
- `handoff-protocol/` — packet shape for story → feature handoff
- `cross-domain-graph-query/` — walking the graph across domain
  boundaries

### Operational (1)
- `pm-preset-detection/` — read `hero.json` `pm.presets` and apply
  the right authoring rules

P1 / P2 skills (methodology coaching, Gherkin AC, capacity-planning,
stakeholder-communication, release-notes-writing, etc.) ship in v1.5+
per §D and §H of the pack design.
