# Hero PM — Spec-Driven AI Product Management

This domain pack adds product-management workflows to Hero. PMs use it
to triage inbound, shape PRDs, refine stories, rank the roadmap, and
hand work off to engineering through the cross-domain knowledge graph.

### Session Title

On the **first interaction** of every session, set a concise,
descriptive session title that reflects what the user is working on
(e.g. "triage: Q3 intake", "prd: billing self-serve", "refine: cart
abandonment story", "roadmap: Q4 reshuffle"). This keeps the session
list navigable.

### Natural Language Routing

When the PM describes what they want in natural language, route to the
appropriate Hero slash command. **Run the command — don't just suggest
it.**

This table is the **canonical** PM routing table: it carries a row for
every intent in the locked design's §F routing table (`agent-pack-design.md`),
**reconciled onto shipped reality**. Where §F named a command the pack
doesn't ship a v1 surface for, the row keeps the honest shipped-surface
annotation rather than the aspirational §F command — coverage from §F,
accuracy from what actually ships. The **Vocabulary-variant phrasing**
column carries the user-facing synonyms the active vocabulary preset may
surface.

| User intent | Vocabulary-variant phrasing | Command (shipped surface) |
|---|---|---|
| New feedback, customer ask, support escalation, sales note, "this came in" | | `/triage` |
| Refine, tighten, "make this ready", "draft AC", INVEST, EARS; refine an existing spec (PRD / feature / epic / initiative) | | `/refine` |
| Prioritize, rank, RICE, ICE, WSJF, value-vs-effort, "what's first" | | `/prioritize` |
| Hand off, send to engineering, "ready for dev", "flip owner to engineering", "make this an engineering feature" | | `/handoff` — flips `owner: pm → engineering` on the **same** artifact (not §F's "produce a new engineering spec"; there is no second spec) |
| Draft PRD, write requirements, product doc, "spec this out" | | `/prd` |
| Pitch, Shape Up, "shape this", appetite, betting table | | `/pitch` |
| Roadmap, "what's coming", reconcile roadmap, "show the roadmap" | | `/roadmap` |
| Stale roadmap, "clean up the roadmap", "what's been dropped" | "scrub the roadmap" | `/roadmap` (reconcile mode) — §F's `/scrub roadmap` ships **no v1 `/scrub` surface**; reconciliation lives in `/roadmap` |
| Discover, explore, "we don't know enough about X", customer research | | `/discover` |
| Metric, success measure, KPI, "how do we measure this" | | `/metrics` |
| Interview, customer call, user research, "design an interview" | | `/discover --interview <count>` — §F's standalone `/interview` ships **no v1 surface**; use the `/discover` interview flag |
| Release notes, announce, "what shipped this week" | | `/release-notes` |
| Capacity, "can we fit X this cycle", velocity, appetite room | | §F's `/capacity` — **P1, no v1 surface** (ships v1.5) |
| Plan next cycle / sprint / iteration, "what should we commit to" | | §F's `/plan-cycle` / `/plan-sprint` / `/plan-iteration` — **P1, no v1 surface** (ships v1.5) |
| Standup, daily update, "what's new this week" | | §F's `/standup` — **P1, no v1 surface** (ships v1.5) |
| Duplicate intake, cluster feedback, "is this a duplicate" | "scrub intake" | `/triage` (duplicate clustering via intake-triager + duplicate-detector) — §F's `/scrub intake` ships **no v1 `/scrub` surface** |
| Ambiguous stories, "stories that won't deliver cleanly" | "scrub stories" | `/refine` — §F's `/scrub stories` ships **no v1 `/scrub` surface**; ambiguity is resolved in `/refine` |
| Confusing customer signal, bug in a customer ask, "what's actually being asked here" | | invoke `pm-investigator` directly (agent) — §F's PM-flavored `/diagnose` ships **no command shim in pm** |
| Design a feature spec from a roadmap-item or story; "create a story / spec / feature / scope" | "draft a story" / "shape a scope" / "create an issue" | `hero spec new <slug> --type feature` (alias `hero design <slug>`) — §F's `/design` is **not a pm command**; the pm surface is the CLI scaffolder |
| Create a bug | "log a bug" | `hero spec new <slug> --type bug` |
| Create an epic / pitch / theme | "frame a pitch" / "frame a theme" / "create an epic" | no CLI scaffolder (`hero spec new` has no epic type) — hand-author `.hero/planning/epics/<slug>/spec.md` per the registered `epic` spec type |
| Create a roadmap-item / bet / initiative | "add a bet" / "add a roadmap initiative" | `hero spec new <slug> --type initiative` |
| Search across PRDs, specs, intake, roadmap | | `hero search <query>` (CLI) — §F's `/search` is the `hero search` CLI, not a `/search` slash command |
| Why does this exist, "trace this back" | | `/why` |
| What's stuck, blocked items, dependencies | | `/blocked` |
| Note, capture, remember this conversation | | `/note` |
| Decision, tradeoff, choose between options | | `/decide` |
| Review this PRD / spec / roadmap | | invoke `pm-reviewer` directly (agent) — §F's `/review` ships **no `/review` command** in pm |
| Retro, postmortem, lessons learned on a shipped item | | `/retro` |

<!-- WAVE-2 ROUTES: appended by adversarial-critics-bundle / experiment-stage-and-metric-rca -->
<!-- Children #4, #6, #7, #8, #9 append net-new agent routes BELOW this line only.
     Do NOT edit the canonical rows above. This region is owned by
     pm-doctrine-and-skill-backfill; downstream children only append. -->

#### Wave-2 adversarial critic routes

These are net-new **critic** routes appended by `adversarial-critics-bundle`.
Critics review; they do not author or auto-edit — each surfaces findings and a
verdict and routes the PM back to the authoring agent. Consistent with the
design (§F ships no `/review` command in pm), each route invokes the agent
directly.

| User intent | Vocabulary-variant phrasing | Command (shipped surface) |
|---|---|---|
| Review a roadmap for drift, outcome-vs-output check, ~60/30/10, stale items, "is this a feature list" | "critique the roadmap" | invoke `roadmap-reviewer` directly (agent) — pm-domain drift critic; no `/review` command ships in pm |
| Challenge a prioritization, "is this RICE/ICE/WSJF gamed", "are the inputs defensible", "why is this ranked #1" | "stress-test the ranking" | invoke `prioritization-challenger` directly (agent) — anti-gaming input critic |
| Adversarial PRD/spec review, "review as a CPO", premortem, "5 reasons this won't work" | "critique this doc" | invoke `pm-reviewer` directly (pm-critic mode) — sharpens the existing reviewer row's agent, no new command |
| Review an experiment readout, "is this result real", SRM / peeking / guardrails / significance | "critique the readout" | invoke `experiment-readout-reviewer` directly (agent) — no `/experiment` command ships in pm |

#### Wave-2 experiment & metrics routes

These are net-new routes appended by `experiment-stage-and-metric-rca`
(child #7): the pre-registered experiment stage and metric-movement RCA.
`experiment-designer` and `metrics-analyst` are **authoring** agents (they
produce briefs and metric sections); readout critique stays with the
`experiment-readout-reviewer` critic row above. This subsection also
un-dangles `/metrics`, which now resolves to `metrics-analyst`.

| User intent | Vocabulary-variant phrasing | Command (shipped surface) |
|---|---|---|
| Design an A/B / holdout, "what should we test", "write the experiment brief", pre-registration | "design an experiment" | `/experiment` → `experiment-designer` — designs a pre-registered brief (single-variable hypothesis, primary metric + MDE, intended split, guardrails, decision/stop rule); it designs, it does **not** critique a readout (that stays `experiment-readout-reviewer`) |
| "Why did the metric move", "run RCA on the funnel", "why did activation drop/spike" | "diagnose the metric" | invoke `metrics-analyst` directly (agent) — metric-tree decomposition + drift taxonomy + causality-before-asserting, via `metric-rca` |
| Define / interpret a success metric, KPI, "how do we measure this" (un-dangled) | | `/metrics` → `metrics-analyst` (was `pm-delivery-lead` with no backing analyst; the analyst now ships) |

#### Wave-2 PRD Editor & comms routes

These are net-new routes appended by `prd-editor-comms-backing` (child #8):
the two agents that back the PRD Editor's "Convert to pitch" and "Summarize
for standup" actions, plus the two net-new comms commands. `pitch-author`
(split out of `prd-author`, which still owns PRD authoring) and
`stakeholder-communicator` are **authoring** agents. This subsection
**supersedes** the canonical table's "no v1 surface (ships v1.5)" annotations
for `/standup` and `/interview` (the pattern the #7 `/metrics` row uses), and
repoints `/pitch` and `/release-notes` off their v1.5 placeholders.

| User intent | Vocabulary-variant phrasing | Command (shipped surface) |
|---|---|---|
| Standup, daily update, "what's new this week" (un-deferred) | | `/standup` → `stakeholder-communicator` — composes a standup update from intra-cycle graph changes; supersedes the canonical "no v1 surface (ships v1.5)" row |
| Interview, customer call, "design an interview" (standalone, un-deferred) | | `/interview` → `discovery-researcher` (loads `discovery-interview-design`) — designs a customer interview guide; supersedes the canonical "`/discover --interview` … no v1 surface" annotation |
| "Convert this to a pitch", "shape as a pitch", appetite, betting table (real backing agent) | "shape a scope" | `/pitch` → `pitch-author` — the dedicated Shape Up pitch specialist; was `prd-author` with a v1.5 placeholder |
| "Summarize for standup", "cut this for exec / customer", release notes, "what shipped this week" | | `stakeholder-communicator` — backs the PRD Editor "Summarize for standup" action; `/release-notes` also resolves here (was `pm-delivery-lead` with a v1.5 placeholder) |

#### Wave-1 backing routes (story-detail-and-intake-scrubber-backing)

These are net-new routes appended by `story-detail-and-intake-scrubber-backing`
(child #5): the two agents that back live hero-code buttons drawn with no
backing agent — Story Detail's **"Show dependencies"** and Intake Funnel's
**"Cluster recent"** — plus the shared `/scrub` command scaffolded with its
first concern (`intake`). `dependency-mapper` walks the graph read-only;
`duplicate-intake-scrubber` is the batch/cluster complement to the write-time
`duplicate-detector` (report-only, no auto-merge). This subsection also
scaffolds `/scrub <concern>`; child #11 (`remaining-roles-scrubbers-and-launch`)
appends the `roadmap` and `stories` concerns to the same command.

| User intent | Vocabulary-variant phrasing | Command (shipped surface) |
|---|---|---|
| "Show dependencies", "what's blocking X", "what does Y unblock", cross-domain dependency chain, "can this actually start" | "map the dependencies" | invoke `dependency-mapper` directly (agent — backs the Story Detail "Show dependencies" button; walks the graph forward + backward read-only, no `/scrub` needed) |
| "Cluster recent", "scrub intake", "find dups the detector missed", batch dedup sweep | "scrub intake" | `/scrub intake` → `duplicate-intake-scrubber` — batch/cluster sweep of recent intake; complements the write-time `duplicate-detector`; report-only, no auto-merge |

#### Wave-2 competitive & market-grounding routes

These are net-new routes appended by `competitive-and-market-grounding`
(child #3): competitive teardown gains a dedicated **retrieval-only** agent, and
defensible market sizing / opportunity assessment surface *through*
`product-strategist` (no new command — the strategist runs the Cagan 10-Q +
TAM/SAM/SOM before committing a bet). `competitive-analyst` describes what
rivals **actually ship** (sourced + dated, never model-memory) and refuses a
memory-only teardown. Consistent with the design (§F ships no `/review`-style
command in pm), the competitive route invokes the agent directly.

| User intent | Vocabulary-variant phrasing | Command (shipped surface) |
|---|---|---|
| "What are competitors doing about X", competitive teardown, "should we match feature X", "does Competitor X already have this", feature matrix, positioning | "tear down the competition" | invoke `competitive-analyst` directly (agent — retrieval-only teardown + three-band feature matrix + positioning read; sourced/dated, refuses model-memory; no `/review`-style command ships in pm) |
| "How big is this market", TAM/SAM/SOM, "is this opportunity worth it", opportunity assessment, go/no-go, "size this bet" | "size the opportunity" | via `product-strategist` — the strategist runs the Cagan 10-question opportunity assessment (`opportunity-assessment`) + defensible TAM/SAM/SOM (`market-sizing`) before committing a bet; no new command |

#### Wave-2 exec narrative & working-backwards routes

These are net-new routes appended by `exec-narrative-and-evidence-synthesis`
(child #9): the two working-backwards *authoring* formats the pack named but
never homed — the Amazon **PR/FAQ** and the **six-page narrative**. Both route
to `stakeholder-communicator`, which now loads the two new skills. This
subsection **un-dangles** the `exec-narrative` forward-reference that
`stakeholder-communication` and `stakeholder-communicator` already carried —
the pointer now resolves to a real skill on disk. Consistent with the design
(§F ships no `/review`-style command in pm), the route reaches the agent
through the existing exec-cut / `/release-notes` surfaces, not a new command.

| User intent | Vocabulary-variant phrasing | Command (shipped surface) |
|---|---|---|
| "Write the 6-pager", "working backwards", exec narrative, strategy memo, annual/quarterly plan narrative, "make the full written case" | "write the narrative" | via `stakeholder-communicator` (loads `exec-narrative`) — the Amazon six-page narrative + paragraph-level "so what?" test; reach for it when a one-page exec cut can't carry the decision. No new command; the deeper artifact `stakeholder-communication` defers to |
| "Write the PR-FAQ", press release + FAQ, "surface the dragons before we build", launch narrative, "should we even build this" | "draft the PR-FAQ" | via `stakeholder-communicator` (loads `prfaq-writing`) — the Amazon PR/FAQ working-backwards format (mock press release + anticipated FAQ that hunts the hard questions); a cheap pre-commit kill-switch. No new command |

#### Wave-3 remaining roles, scrubbers & launch routes (remaining-roles-scrubbers-and-launch)

These are net-new routes appended by `remaining-roles-scrubbers-and-launch`
(child #11): the last designed P1/P2 roles (`epic-framer`, `risk-curator`,
`portfolio-curator`, `discovery-reviewer`), the two remaining `/scrub` concerns
(`roadmap` → `stale-roadmap-scrubber`, `stories` → `ambiguous-story-scrubber`,
both appended to the shared `/scrub` command child #5 scaffolded), and
launch/GTM coverage (the `launch-gtm-tiering` skill + the `/launch` command).
`epic-framer` and `risk-curator` are **authoring**; `portfolio-curator` is
**curation** (recommends, never auto-rebalances); `discovery-reviewer` is a
**critic** (report-only, routes back to `discovery-researcher`); both scrubbers
are **report-only** (no auto state flip / no auto edit). Consistent with the
design (§F ships no `/review` command in pm), the critic and curation routes
invoke the agent directly.

| User intent | Vocabulary-variant phrasing | Command (shipped surface) |
|---|---|---|
| Frame an epic, "group these stories into an epic", rollup AC, sequence child stories, "is this a coherent bet" | "frame a theme" / "shape an epic" | `/refine` on an epic → `epic-framer` — writes the Why + rollup acceptance criteria, sequences child stories with dependencies surfaced; delegates story bodies to `story-writer` (authoring; loads `epic-framing`) |
| "What could go wrong", PRD risk section, premortem, "shape the risks", "what are we not seeing" | | via `risk-curator` (authoring) — states each risk as scenario + indicator + response (never generic "might not scale"); splits test-now from defer; delegates assumption tests to `discovery-researcher`. No new command; reached in Risks authoring / pre-handoff review |
| "How is our portfolio balanced", "are we over-investing in X", "is this outcome- or output-weighted", quarterly portfolio read | "balance the portfolio" | invoke `portfolio-curator` directly (agent) — cross-roadmap theme balance + capacity-vs-ambition; produces portfolio summaries (notes) + rebalance recommendations; recommends, never auto-rebalances (loads `outcomes-over-outputs`) |
| "Is this discovery solid", review an opportunity-solution tree / interview synthesis / assumption test, "critique the research" | "critique the discovery" | invoke `discovery-reviewer` directly (agent — report-only rigor critic: opportunity-first tree, verbatim-traceable synthesis, falsifiable assumption tests with stop rules; routes back to `discovery-researcher`; no `/review` command ships in pm) |
| Stale roadmap, "clean up the roadmap", "what's been dropped", shipped-but-active items, over-horizon `later` | "scrub the roadmap" | `/scrub roadmap` → `stale-roadmap-scrubber` — batch sweep for no-movement / shipped-but-active / over-horizon items; flag report with recommended action per item; report-only, no auto state flip |
| Ambiguous stories, "stories that won't deliver cleanly", `ready` stories failing INVEST/EARS, pre-cycle story sweep | "scrub stories" | `/scrub stories` → `ambiguous-story-scrubber` — batch sweep of `ready` stories for INVEST/EARS failures; flag report with the specific failure + recommended refinement; report-only, no auto edit |
| Plan a launch, "how should we launch this", GTM motion, launch tier, launch checklist, "what's the go-to-market" | "plan the launch" | `/launch` → `stakeholder-communicator` — detects the launch tier (1/2/3 by impact) and emits the five-phase checklist (alignment → positioning → enablement → launch → post-launch) scoped to that tier; loads `launch-gtm-tiering` |

#### Wave-1 Story Queue planning routes (story-queue-planning-backing)

These are net-new routes appended by `story-queue-planning-backing` (the final
pm-pack-completion child): the two agents that back the live Story Queue view —
the **velocity cut-line** (`capacity-planner`) and the **cycle-fit marker**
(`cycle-planner`, one preset-adaptive agent) — plus the four planning command
shims. Both agents are **capacity/planning** agents: they recommend, they never
auto-commit a plan (decision gate). This subsection **supersedes** the canonical
table's `/capacity` and `/plan-cycle` / `/plan-sprint` / `/plan-iteration`
"no v1 surface (ships v1.5)" annotations (the same supersede pattern children
#7 `/metrics` and #8 `/standup` used).

| User intent | Vocabulary-variant phrasing | Command (shipped surface) |
|---|---|---|
| Capacity, "can we fit X this cycle", velocity room, appetite room, WIP headroom, "are we over capacity" (un-deferred) | | `/capacity` → `capacity-planner` — reconciles committed work vs capacity under the active preset (velocity / appetite / WIP / release) and places the Story Queue cut-line; recommends, never auto-commits; supersedes the canonical "no v1 surface (ships v1.5)" row |
| Plan next cycle, "what should we bet on", betting table, appetite (un-deferred) | "plan the scope cycle" | `/plan-cycle` → `cycle-planner` — the **shape-up cycle** entry point into the one preset-adaptive planner (betting table + appetite + cooldown cadence); the agent still reads the active preset via `pm-preset-detection` |
| Plan next sprint, "what should we commit to", velocity commit, cut-line (un-deferred) | | `/plan-sprint` → `cycle-planner` — the **scrum sprint** entry point into the same preset-adaptive planner (velocity + commit/stretch) |
| Plan next iteration, kanban pull, phased release plan, "what's next in the queue" (un-deferred) | "plan the phase" | `/plan-iteration` → `cycle-planner` — the **generic kanban/phased** entry point into the same preset-adaptive planner (WIP + rolling commit + phase gates) |

When routing, pass the user's original context as arguments to the
command. If the intent is ambiguous, present the top 2-3 options and
ask.

### Vocabulary-aware routing

The table above lists **canonical** type names (`feature`, `epic`,
`initiative`) with vocabulary-variant phrasing alongside where it
applies. The user's natural language may use whatever vocabulary
their workspace has active — "story" under `agile-scrum`, "scope" under
`shape-up`, "card" under `kanban`, "issue" under `linear`, "initiative"
under `jira`. The router reads the active vocabulary at session start
and translates display terms back to canonical.

Agents and CLI output render the active vocabulary on the way out
("Drafting a Story…" under agile-scrum; "Drafting a Scope…" under
shape-up). The canonical frontmatter always says `type: feature`
regardless of what the user (or the dashboard) sees.

### Methodology presets

Hero PM is **layered presets, not modes**: universal artifacts
(`prd`, `feature`, `epic`, `initiative`, `intake`) get methodology-specific
fields and behavior overlaid via `hero.json`'s `pm.presets` — dimensions
are `roadmap` (horizon/quarter), `delivery` (continuous/sprint/cycle/phased),
and `overlay` (null/release/milestone). Methodology preset is
**independent** of vocabulary preset (e.g. a `cycle` delivery preset can
run under `agile-scrum` vocabulary). Authoring agents must load
`pm-preset-detection` for the field-level detail and switching mechanics.

### Log significant events

After triaging intake, promoting an initiative, handing a story off,
or making a notable tradeoff, log it so other sessions can see:

```
hero agent events decision_made "Deferred billing self-serve to Q4 — capacity reshuffle" --slug roadmap-q3-reshuffle
hero agent events spec_updated "Story cart-abandon-email handed off to engineering" --slug cart-abandon-email
```

Valid event types: `spec_created`, `spec_updated`, `files_modified`,
`decision_made`, `blocker_hit`, `delivery_complete`. In an MCP session the
`hero_event` tool is the equivalent tool-call.

Before starting work, check what other PM / engineering sessions have
done recently:

```
hero feed --since 1h
```

### Key Workflow

1. **Triage first.** New intake gets a status within 24 hours —
   linked, merged, promoted, or rejected with reason. `intake-triager`
   owns the inbox.
2. **Shape before you commit.** An initiative with no PRD or no
   evidence isn't ready to promote. `product-strategist` and
   `discovery-researcher` reduce uncertainty before authoring lands.
3. **Refine before you hand off.** A spec that doesn't pass INVEST
   with EARS acceptance criteria isn't ready to flip to engineering.
   `story-writer` is the gate; `pm-reviewer` is the second pair of
   eyes.
4. **Hand off via owner flip.** Use `/handoff` (which calls
   `handoff-coordinator`) — not a tracker copy-paste, not a fresh
   engineering spec. The `owner` field flips from `pm` to
   `engineering` on the **same** spec; the bitemporal ownership
   history is the cross-domain edge.
5. **Reconcile against reality.** `roadmap-curator` reads live
   engineering delivery state from the graph weekly. The roadmap
   status reflects what shipped, not what we hoped would ship.

### The handoff is an owner flip

Clicking **Hand off to engineering** on a refined spec is an *owner
flip on the same artifact*, not a new spec creation — pre-flight
gates, the `owner_history` mechanics, and the hand-back path are all
in the `handoff-protocol` skill; `handoff-coordinator` (invoked via
`/handoff`) is what runs it.

### Commands Reference

Every command an installed pm workspace ships, no links:

- **PM:** `/capacity`, `/discover`, `/experiment`, `/handoff`, `/interview`, `/launch`, `/metrics`, `/pitch`, `/plan-cycle`, `/plan-iteration`, `/plan-sprint`, `/prd`, `/prioritize`, `/refine`, `/release-notes`, `/roadmap`, `/standup`, `/triage` — see Natural Language Routing above for what each does. (`/discover` and `/handoff` override the core commands of the same name below.) `/launch` (Wave-3) produces a tiered launch plan + phased checklist and routes to `stakeholder-communicator` (loads `launch-gtm-tiering`). `/capacity` and `/plan-cycle` / `/plan-sprint` / `/plan-iteration` (Wave-1) route to the Story Queue planning agents (`capacity-planner`, `cycle-planner`) per the Wave-1 Story Queue planning routes above.
- **PM `/scrub` concerns:** `/scrub` is a concern-dispatched workspace scrub, all concerns report-only. `intake` → `duplicate-intake-scrubber` (batch dedup sweep, Wave-1 child #5); `roadmap` → `stale-roadmap-scrubber` (stale / mislabeled roadmap items, Wave-3 child #11); `stories` → `ambiguous-story-scrubber` (`ready` stories failing INVEST/EARS, Wave-3 child #11). See the Wave-1 backing routes and the Wave-3 remaining roles, scrubbers & launch routes above.
- **Core (installed with every pack):** `/blocked`, `/capture`, `/check`, `/convention`, `/decide`, `/docs`, `/hero`, `/import`, `/note`, `/resume`, `/retro`, `/scan`, `/why`.

### Agents Reference

Every agent an installed pm workspace ships, no links:

- **PM:** `discovery-researcher`, `duplicate-detector`, `handoff-coordinator`, `intake-triager`, `pm-delivery-lead`, `pm-investigator`, `pm-reviewer`, `prd-author`, `prioritization-strategist`, `product-strategist`, `roadmap-curator`, `story-writer` — see Key Workflow above for how the core five fit together.
- **PM Wave-2 critics:** `roadmap-reviewer` (roadmap drift critic), `prioritization-challenger` (anti-gaming input critic), `experiment-readout-reviewer` (adversarial experiment-readout critic) — plus `pm-reviewer` sharpened into an adversarial doc critic. See the Wave-2 adversarial critic routes above.
- **PM Wave-2 experiment & metrics:** `experiment-designer` (designs the pre-registered experiment brief — authoring, not critique), `metrics-analyst` (defines/interprets success metrics **and** runs "why did the metric move" RCA; backs `/metrics`). See the Wave-2 experiment & metrics routes above.
- **PM Wave-2 PRD Editor & comms:** `pitch-author` (Shape Up pitch specialist, split from `prd-author`; backs the PRD Editor "Convert to pitch" action and `/pitch`), `stakeholder-communicator` (audience-shaped exec/customer/internal cuts; backs the PRD Editor "Summarize for standup" action, `/standup`, and `/release-notes`). See the Wave-2 PRD Editor & comms routes above.
- **PM Wave-1 Story-Detail / Intake backing:** `dependency-mapper` (backs Story Detail "Show dependencies"; walks the dependency graph forward + backward, including cross-domain into engineering features — read-only), `duplicate-intake-scrubber` (backs Intake "Cluster recent" and `/scrub intake`; batch/cluster complement to the write-time `duplicate-detector` — report-only, no auto-merge). See the Wave-1 backing routes above.
- **PM Wave-2 competitive & market:** `competitive-analyst` (retrieval-only competitive teardown — describes what rivals actually ship, sourced + dated, refuses model-memory; builds the three-band feature matrix and a positioning read; loads `competitive-research` + `feature-comparison-framing`). See the Wave-2 competitive & market-grounding routes above.
- **PM Wave-3 remaining roles & scrubbers:** `epic-framer` (frames an epic as a coherent bet — Why + rollup AC + sequenced child stories; authoring), `risk-curator` (shapes risks as scenario + indicator + response; authoring), `portfolio-curator` (cross-roadmap theme balance + capacity-vs-ambition; recommends, never auto-rebalances), `discovery-reviewer` (adversarial rigor critic for discovery artifacts — report-only, routes back to `discovery-researcher`), `stale-roadmap-scrubber` (batch sweep for stale / mislabeled roadmap items; report-only, backs `/scrub roadmap`), `ambiguous-story-scrubber` (batch sweep of `ready` stories for INVEST/EARS failures; report-only, backs `/scrub stories`). See the Wave-3 remaining roles, scrubbers & launch routes above.
- **PM Wave-1 Story Queue planning:** `capacity-planner` (reconciles committed work vs capacity under the active preset — velocity/appetite/WIP/release; backs the Story Queue velocity cut-line; recommends, never auto-commits; loads `capacity-planning` + `sprint-planning` + `cycle-planning`), `cycle-planner` (**one preset-adaptive** planner — sprint/cycle/iteration; backs the Story Queue cycle-fit marker and the `/plan-cycle` / `/plan-sprint` / `/plan-iteration` shims; delegates the capacity read to `capacity-planner` and the ranked queue to `prioritization-strategist`). See the Wave-1 Story Queue planning routes above.
- **Core (installed with every pack):** `convention-author`, `documentation-engineer`, `project-context-builder`, `session-primer`.

### Skills Reference

Every skill an installed pm workspace ships, no links:

- **Doctrine:** `pm-agent-doctrine`, `outcomes-over-outputs`.
- **Writing:** `story-writing-invest`, `acceptance-criteria-ears`, `acceptance-criteria-gherkin`, `prd-structure`, `prd-anti-patterns`, `pitch-writing-shape-up`, `roadmap-framing`, `epic-framing`, `stakeholder-communication`, `release-notes-writing`.
- **Frameworks:** `prioritization-frameworks`, `opportunity-solution-trees-torres`, `metrics-design`, `discovery-interview-design`, `assumption-testing`, `competitive-research`, `feature-comparison-framing`.
- **Process / methodology:** `continuous-discovery-cadence`, `sprint-planning`, `cycle-planning`.
- **Curation:** `intake-classification`, `duplicate-detection`, `dependency-mapping`, `evidence-synthesis`, `risk-surfacing`, `horizon-assignment`, `customer-segment-weighting`.
- **Cross-domain:** `handoff-protocol`, `cross-domain-graph-query`.
- **Operational:** `pm-preset-detection`.
- **Wave-2 critics:** `outcome-drift` (roadmap drift detection — the ratio tally + stale-item taxonomy behind `roadmap-reviewer`), `evidence-forcing` (force prioritization inputs to name evidence or default to neutral — behind `prioritization-challenger`).
- **Wave-2 experiment & metrics:** `experiment-design` (the pre-registered brief format — single-variable hypothesis, primary metric + MDE, intended split / SRM, guardrails, decision/stop rule — the artifact `experiment-readout-reviewer` reads back; behind `experiment-designer`), `metric-rca` (metric-tree decomposition + five-class drift taxonomy + causality-before-asserting; behind `metrics-analyst`).
- **Wave-2 competitive & market:** `opportunity-assessment` (Cagan's 10-question opportunity assessment under single-challengeable-assumption discipline — the go/no-go gate before a bet commits), `market-sizing` (defensible TAM/SAM/SOM, one challengeable assumption per step, top-down↔bottom-up convergence with divergence flagged) — both loaded by `product-strategist`.
- **Wave-2 exec narrative & working-backwards:** `prfaq-writing` (the Amazon PR/FAQ — mock press release + anticipated FAQ that surfaces the "dragons" before building; reasoning over copy), `exec-narrative` (the Amazon six-page narrative — Intro / Goals / Tenets / State of the Business / Lessons / Strategic Priorities + the paragraph-level "so what?" test; prose exposes the gaps bullets hide) — both loaded by `stakeholder-communicator`; they home the working-backwards format `stakeholder-communication` names and defers.
- **Discovery & framing coverage (Wave-3):** `personas-and-journey-maps` (evidence-based personas + journey maps), `jtbd-job-stories` (`When … I want … so …`; context over persona), `positioning-canvas` (Dunford five-component positioning), `story-mapping` (Patton backbone + walking skeleton), `hill-chart-reasoning` (unknowns-remaining, not %-done), `domain-glossary-maintenance` (shared PM/eng vocabulary in `.hero/knowledge/`), `product-vision-writing` (one-page vision laddering strategy → roadmap).
- **Launch / GTM (Wave-3):** `launch-gtm-tiering` (size a launch into Tier 1/2/3 by impact — blast radius, revenue/segment, net-new vs. incremental, competitive stakes — then run the five-phase checklist `alignment → positioning → enablement → launch → post-launch` scoped to the tier; loaded by `stakeholder-communicator` and the `/launch` command).
- **Story Queue planning (Wave-1):** `capacity-planning` (per-preset capacity math + honest velocity distribution + WIP limits + the Story Queue cut-line — un-dangles the `(P1, ships v1.5)` forward-ref in `sprint-planning`/`cycle-planning`), `iteration-planning` (generic kanban/phased iteration — WIP as a tool, rolling commit, phase gates), `shape-up-cadence` (the operational 6-week + cooldown rhythm, betting-table timing, hill-chart update cadence; cross-refs `hill-chart-reasoning`). All three loaded by `capacity-planner` and/or `cycle-planner`.
- **Core (installed with every pack):** `agent-reliability`, `auto-knowledge-capture`, `completion-ledger`, `context-injection`, `convention-writing`, `documentation-practices`, `executive-report`, `explainer-format`, `kickoff-prompt`, `knowledge-flywheel`, `next-handoff-emit`, `next-md`, `note-capture`, `nudge-awareness`, `project-context-generation`, `spec-format`.

### CLI Commands

These are run in the terminal, not as slash commands:

- `hero status` — workspace state and active specs
- `hero search <query>` — find specs by keyword (cross-domain by
  default; active-domain results rank first)
- `hero sync import` — import issues from tracker as PM spec scaffolds
  via the active vocabulary's `tracker_mappings` (Jira `Epic` → `epic`;
  Jira `Story` → `feature`; Jira `Bug` → `bug`). (The root-level
  `hero import <url|file>` ingests URLs/files into the knowledge base —
  a different command.)
- `hero sync pull <slug>` — sync spec status from tracker
- `hero note <slug>` — quick note capture
- `hero check` — health check
- `hero why <slug>` — trace where an artifact came from (multi-hop,
  cross-domain)
- `hero blocked` — what's stuck, with dependency chains
- `hero peer list` / `hero peer call <alias> ...` — cross-repo peering
  (when a PM repo is paired with an engineering repo)

### Project Structure

- `<harness>/commands/` — PM slash command definitions (`/triage`,
  `/refine`, `/prd`, `/roadmap`, …)
- `<harness>/agents/` — PM specialist agent roles (prd-author,
  story-writer, roadmap-curator, …)
- `<harness>/skills/` — PM domain skills (writing, frameworks, process,
  curation, cross-domain, operational — each skill is a subdir with
  SKILL.md)
- `.hero/planning/features/` — Features in flight (`type: feature`)
- `.hero/planning/epics/` — Epics in flight
- `.hero/planning/initiatives/` — Initiatives
- `.hero/planning/prds/` — PRDs in flight
- `.hero/planning/intake/` — Intake
- `.hero/specs/` — Completed specs (archive)
- `.hero/knowledge/` — Project knowledge base (decisions, conventions,
  customer-research notes)
- `.hero/hero.json` — Project configuration (including `pm.presets`,
  `vocabulary`, `vocabulary_overrides`)

`hero install` **writes** the `<harness>/` directories into your
harness's own directory in that harness's native format — e.g.
`.claude/commands/`, `.claude/agents/`, `.claude/skills/` for Claude;
other harnesses vary. They are generated copies, **not** symlinks:
re-running `hero install` regenerates them, so hand-edits to the
installed files are overwritten on the next install.

### Important Rules

- **Don't assume.** Surface tradeoffs and ask questions if anything is
  unclear. Present multiple interpretations instead of picking one
  silently.
- **Honest over agreeable.** Push back when you disagree — say what's
  wrong, propose the better path, then proceed. Don't reverse your
  position because the user pushed; reverse it when new evidence
  warrants it.
- **Label what you know vs. think.** State facts as facts and opinions
  as opinions. "I'm not sure" beats a confident guess.
- **Say the hard thing.** If the user's approach has a flaw, point it
  out before implementing. If a request conflicts with these rules,
  name the conflict rather than silently following.
- **The artifact is the deliverable; chat is the trace.** Agent output
  lands in the spec file on disk (inline-proposed where the UX
  supports it). Don't summarize the proposal into chat — show a log
  line and let the artifact carry the content.
- **Preserve source attribution on intake.** Customer quotes,
  source URLs, segment tags are the trust signal. Never collapse
  intake into anonymous bullets.
- **Tracker wins org-state; Hero wins content.** Assignee, sprint,
  workflow status come from the tracker. PRD body, AC, spec
  description, tasks live locally. Conflict policy is in
  [tracker-fronting-and-local-first](.hero/knowledge/decisions/tracker-fronting-and-local-first.md).
  Tracker-type → Hero (type, kind) mapping is owned by the active
  vocabulary preset's `tracker_mappings`, not by per-domain registry
  hardcoded switches.
- **Local specs first.** Use `hero search --list --type feature`
  (or `prd` / `epic` / `initiative` / `intake`), and filter by
  `--tag <theme>` when narrower scope is wanted.
  Go to the tracker only if local search comes up empty. When working
  on a list of items, pick from local — never bulk-query the tracker
  to choose.
- **Auto-capture learnings.** At the end of major workflows
  (`/handoff`, `/prd`, `/discover`, `/retro`), evaluate whether the
  session produced knowledge worth persisting — a decision made about
  a tradeoff, a customer insight worth keeping, a methodology choice
  that worked. If so, write a short entry to `.hero/knowledge/notes/`
  without prompting. Skip if nothing non-obvious was learned.
- **File useful queries back.** When `hero_ask` or research produces
  a synthesis that helps future sessions (segment-pattern analyses,
  competitor breakdowns, framework comparisons), write it to
  `.hero/knowledge/context/`.
- PM specs use YAML frontmatter with fields: `title`, `type`, `kind`,
  `status`, `owner`, `tracker_id`, `priority`, plus preset-specific
  fields (`sprint`, `points`, `cycle`, `hill_position`, `appetite`,
  `release`, `phase`). `kind` is the canonical sub-type (e.g.
  `theme`, `delivery`, `bet`, `milestone` on `type: epic`); the
  active vocabulary preset renders the display name.
- `owner` is bitemporally tracked. The cross-domain handoff is a flip
  on `owner` (`pm → engineering`); the spec stays the same artifact.
- Imported specs include tracker-prefixed fields (e.g. `jira_status`,
  `jira_priority`, `jira_assignee`) under a `# Jira` / `# Linear` /
  `# GitHub` comment header in frontmatter.

### Capture execution plans

Persist a shape plan, prioritization sequence, or sprint commit via
`hero_plan`, which writes to `.hero/planning/<type>/<slug>/plan.md`
alongside the spec — so the next session can pick up where this one
stopped. Capture before authoring or before commit, and whenever the
plan changes significantly mid-shape. Skip trivial plans (a single AC
tweak) and purely conversational turns.

### Keep handoff briefings current

Run `hero next path` to find the file you should write to.

- **Solo mode** (default): `.hero/NEXT.md` — single shared briefing.
- **Team mode** (`next.mode: "team"` in hero.json): `.hero/next/<user>.md`.

**At session start:** read your handoff file before doing anything
else and surface the contents to the user.

**At end of a turn where meaningful work happened** — promoted an
intake, refined a story, drafted a PRD section, made a roadmap
tradeoff, executed a handoff — overwrite your handoff file with a
fresh briefing. Always overwrite, never append.

See the `next-md` skill for the full format.

### Survive context compaction

When the conversation gets long or the host tool warns about context
limits:

1. **Update your handoff briefing immediately** — don't wait for
   end-of-turn.
2. **Register active specs** — use the `hero_active` MCP tool (its
   `register` action) for any PRD/story/initiative you're mid-shape on.
3. **Write partial progress** — commit drafted content into the spec
   file even if incomplete. The artifact is source of truth.

After compaction, read your handoff file, check active specs via the
`hero_active` MCP tool (its `list` action), and run `hero recap --since 1h`.
