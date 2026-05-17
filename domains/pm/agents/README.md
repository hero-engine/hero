# Hero PM — Agents

Agents in this directory are the PM specialist roster. They are loaded
by the domain agent loader (per primitive #3,
`domain-routing-and-agents`) when the active domain in `hero.json` is
`pm`.

See the vertical charter at [`../mission.md`](../mission.md) for the
mission, and the [agent-pack design](../../../.hero/planning/features/hero-pm/agent-pack-design.md)
for the full 27-agent roster with priorities and prior-art
attribution. The 12 P0 agents shipped in v1:

| File | Tier | Role |
|---|---|---|
| `pm-delivery-lead.md` | Coordination | Orchestrates PM agents for refine / handoff / shape |
| `pm-investigator.md` | Coordination | Investigates ambiguous signals; "what's actually being asked" |
| `product-strategist.md` | Strategic | Frames roadmap bets in outcomes and tradeoffs |
| `discovery-researcher.md` | Strategic | Continuous-discovery research (Torres) |
| `prd-author.md` | Authoring | Drafts and refines PRDs (pitch- or ten-section-shaped) |
| `story-writer.md` | Authoring | Drafts and refines features (`type: feature`) with INVEST + EARS. Vocabulary-aware display ("Story Writer" / "Scope Author" / "Spec Writer"). |
| `roadmap-curator.md` | Authoring | Maintains the roadmap board against live delivery |
| `intake-triager.md` | Triage | Triages inbound feedback into intakes |
| `duplicate-detector.md` | Triage | Detects near-duplicate intake / specs / initiatives |
| `prioritization-strategist.md` | Prioritization | Applies RICE / ICE / WSJF / value-vs-effort |
| `handoff-coordinator.md` | Coordination-delivery | Executes the PM → engineering owner flip on the same spec — **brand interaction** (no separate engineering spec is created) |
| `pm-reviewer.md` | Review | Reviews PM artifacts pre-handoff and pre-promotion |

Each agent file follows the engineering pack's shape: YAML frontmatter
(`name`, `description`, `mode`, `temperature`, `color`, `permission`)
then a markdown body with role, when-invoked, workflow guidance, and
delegation rules.

P1 / P2 agents are listed in §C of the pack design and ship in v1.5+.
