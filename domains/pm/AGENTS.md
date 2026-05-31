# Hero PM — Spec-Driven AI Product Management

This domain pack adds product-management workflows to Hero. PMs use it
to triage inbound, shape PRDs, refine stories, rank the roadmap, and
hand work off to engineering through the cross-domain knowledge graph.

## Session Title

On the **first interaction** of every session, set a concise,
descriptive session title that reflects what the user is working on
(e.g. "triage: Q3 intake", "prd: billing self-serve", "refine: cart
abandonment story", "roadmap: Q4 reshuffle"). This keeps the session
list navigable.

## Natural Language Routing

When the PM describes what they want in natural language, route to the
appropriate Hero slash command. **Run the command — don't just suggest
it.**

| User intent | Command |
|---|---|
| New feedback, customer ask, support escalation, sales note, "this came in" | `/triage` |
| Refine, tighten, "make this ready", "draft AC", INVEST, EARS | `/refine` |
| Prioritize, rank, RICE, ICE, WSJF, value-vs-effort, "what's first" | `/prioritize` |
| Hand off, send to engineering, "ready for dev", "flip owner to engineering" | `/handoff` (flips `owner: pm → engineering` on the same artifact) |
| Create a story / spec / feature / pitch / scope (vocabulary-aware) | `hero new feature` (canonical; vocabulary renders the display term) |
| Create a bug | `hero new bug` |
| Create an epic / pitch / theme | `hero new epic` |
| Create a roadmap-item / bet / initiative | `hero new initiative` |
| Draft PRD, write requirements, product doc, "spec this out" | `/prd` |
| Pitch, Shape Up, "shape this", appetite, betting table | `/pitch` |
| Roadmap, "what's coming", reconcile roadmap, "show the roadmap" | `/roadmap` |
| Discover, explore, "we don't know enough about X", customer research | `/discover` |
| Metric, success measure, KPI, "how do we measure this" | `/metrics` |
| Interview, customer call, user research, "design an interview" | `/interview` |
| Release notes, announce, "what shipped this week" | `/release-notes` |
| Capacity, "can we fit X this cycle", velocity, appetite room | `/capacity` |
| Plan next cycle / sprint / iteration | `/plan-cycle` / `/plan-sprint` / `/plan-iteration` |
| Standup, daily update, "what's new this week" | `/standup` |
| Stale roadmap, "clean up the roadmap", "what's been dropped" | `/scrub roadmap` |
| Duplicate intake, cluster feedback, "is this a duplicate" | `/scrub intake` |
| Ambiguous specs / stories, "won't deliver cleanly" | `/scrub specs` |
| Confusing customer signal, "what's actually being asked here" | `/diagnose` (routes to `pm-investigator`) |
| Refine an existing spec (PRD / feature / epic / initiative) | `/refine` |
| Search across PRDs, specs, intake, roadmap | `/search` |
| Why does this exist, "trace this back" | `/why` |
| What's stuck, blocked items, dependencies | `/blocked` |
| Note, capture, remember this conversation | `/note` |
| Decision, tradeoff, choose between options | `/decide` |
| Review this PRD / spec / roadmap | `/review` (routed to `pm-reviewer`) |
| Retro, postmortem, lessons learned on a shipped item | `/retro` |

When routing, pass the user's original context as arguments to the
command. If the intent is ambiguous, present the top 2-3 options and
ask.

## Vocabulary-aware routing

The routing table above lists **canonical** type names (`feature`, `epic`,
`initiative`). The user's natural language may use whatever vocabulary
their workspace has active — "story" under `agile-scrum`, "scope" under
`shape-up`, "card" under `kanban`, "issue" under `linear`, "initiative"
under `jira`. The router reads the active vocabulary at session start and
translates display terms back to canonical:

| User says (vocabulary-dependent) | Canonical route |
|---|---|
| "draft a story" / "shape a scope" / "create an issue" | `hero new feature` |
| "log a bug" | `hero new bug` |
| "frame a pitch" / "frame a theme" / "create an epic" | `hero new epic` |
| "add a bet" / "add a roadmap initiative" | `hero new initiative` |

Agents and CLI output render the active vocabulary on the way out
("Drafting a Story…" under agile-scrum; "Drafting a Scope…" under
shape-up). The canonical frontmatter always says `type: feature`
regardless of what the user (or the dashboard) sees.

## Methodology presets

Hero PM is **layered presets, not modes**. Artifacts (`prd`, `feature`,
`epic`, `initiative`, `intake`) are universal under the unified
type model. Process layers overlay methodology-specific fields and
behavior onto them. Read the active preset from `hero.json` under
`pm.presets` before authoring:

- `roadmap`: `horizon` (Now/Next/Later) or a quarter
- `delivery`: `continuous` / `sprint` / `cycle` / `phased`
- `overlay`: `null` or `release`/`milestone`

Authoring agents (`prd-author`, `story-writer`, `pitch-author`) and
the `cycle-planner` agent must load `pm-preset-detection` and populate
the right preset-specific fields (`sprint`/`points`,
`cycle`/`hill_position`, `appetite`, `release`/`phase`). Switching a
preset is a config edit + dashboard reload — no data migration.

Methodology preset is **independent** of vocabulary preset. A team can
run a `cycle` delivery preset (Shape Up methodology) with `agile-scrum`
vocabulary (calls features "Stories"), or a `sprint` preset with
`shape-up` vocabulary (calls features "Scopes" but with story points
and sprint cadence). See `pm-preset-detection` and the vocabulary
preset system for details.

## Log significant events

After triaging intake, promoting an initiative, handing a story off,
or making a notable tradeoff, log it so other sessions can see:

```
hero event decision_made "Deferred billing self-serve to Q4 — capacity reshuffle" --slug roadmap-q3-reshuffle
hero event handoff "Story cart-abandon-email handed off to engineering" --slug cart-abandon-email
```

Before starting work, check what other PM / engineering sessions have
done recently:

```
hero feed --since 1h
```

## Key Workflow

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

## The handoff is an owner flip

Clicking **Hand off to engineering** on a refined spec is the platform
thesis in one click. Under the unified type model, the handoff is an
*owner flip on the same artifact* — not a new spec creation. The flow
must always:

1. Pre-flight check: spec is `status: ready` with EARS AC, populated
   Out of Scope, linked PRD context (warn if absent), linked
   initiative with rationale (warn if absent).
2. Flip the spec's `owner` field from `pm` to `engineering`. The
   bitemporal history (`owner_history`) is recorded automatically.
3. The Cross-domain Handoff stream row is sourced from
   `owner_history`; the spec appears in engineering's queue
   (`hero queue --owner engineering`) immediately.
4. Engineering's `engineer` agent picks the spec up via `/deliver
   <slug>`; status flips `ready → delivering`; `plan.md` is authored
   as a companion artifact in the same spec folder.

`handoff-coordinator` orchestrates the pre-flight and the flip. It does
**not** call `/design`, does **not** author an engineering spec, does
**not** write a separate `kind: handoff` graph edge — the ownership
history is the edge. Handoff is the boundary, and the spec carries
through unchanged.

**Hand-back path.** Engineering can flip `owner` back to `pm` with a
`handed_back_reason` when refinement reveals an under-specified
requirement.

## CLI Commands

These are run in the terminal, not as slash commands:

- `hero status` — workspace state and active specs
- `hero search <query>` — find specs by keyword (cross-domain by
  default; active-domain results rank first)
- `hero import` — import issues from tracker as PM spec scaffolds via
  the active vocabulary's `tracker_mappings` (Jira `Epic` → `epic`;
  Jira `Story` → `feature`; Jira `Bug` → `bug`)
- `hero sync pull <slug>` — sync spec status from tracker
- `hero note <slug>` — quick note capture
- `hero check` — health check
- `hero why <slug>` — trace where an artifact came from (multi-hop,
  cross-domain)
- `hero blocked` — what's stuck, with dependency chains
- `hero peer list` / `hero peer call <alias> ...` — cross-repo peering
  (when a PM repo is paired with an engineering repo)

## Project Structure

- `domains/pm/agents/` — PM specialist agents
- `domains/pm/skills/` — PM domain skills (writing, frameworks,
  process, curation, cross-domain, operational)
- `domains/pm/commands/` — PM slash commands
- `domains/pm/spec-types/` — PM-led spec-type schemas (`prd`,
  `intake`)
- `core/spec-types/` — shared cross-domain spec-type schemas
  (`feature`, `epic`, `initiative`) used by both PM and engineering
- `core/vocabularies/` — vocabulary preset files (`default`,
  `agile-scrum`, `shape-up`, `kanban`, `jira`, `linear`)
- `.hero/planning/features/` — Features in flight (replaces `stories/`;
  `type: feature`)
- `.hero/planning/epics/` — Epics in flight
- `.hero/planning/initiatives/` — Initiatives
- `.hero/planning/prds/` — PRDs in flight
- `.hero/planning/intake/` — Intake
- `.hero/knowledge/` — Project knowledge base (decisions,
  conventions, customer-research notes)
- `hero.json` — Project configuration (including `pm.presets`,
  `vocabulary`, `vocabulary_overrides`)

## Important Rules

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
  [tracker-fronting-and-local-first](../../.hero/knowledge/decisions/tracker-fronting-and-local-first.md).
  Tracker-type → Hero (type, kind) mapping is owned by the active
  vocabulary preset's `tracker_mappings` (in `core/vocabularies/*.yaml`),
  not by per-domain registry hardcoded switches.
- **Local specs first.** Use `hero search --list --type feature`
  (or `prd` / `epic` / `initiative` / `intake`), and filter by
  `--kind=…` when narrower scope is wanted.
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

## Keep handoff briefings current

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

## Survive context compaction

When the conversation gets long or the host tool warns about context
limits:

1. **Update your handoff briefing immediately** — don't wait for
   end-of-turn.
2. **Register active specs** — `hero active register <session-id>
   <slug>` for any PRD/story/initiative you're mid-shape on.
3. **Write partial progress** — commit drafted content into the spec
   file even if incomplete. The artifact is source of truth.

After compaction, read your handoff file, check `hero active list`,
and run `hero recap --since 1h`.

## Capture execution plans

When you generate a shape plan, prioritization sequence, or sprint
commit, persist it as a Hero artifact via `hero_plan`. This writes the
plan to `.hero/planning/<type>/<slug>/plan.md` alongside the spec.
Plans live with the artifact so the next session can pick up where
this one stopped.

**When to capture:**
- Before authoring (the initial PRD shape plan)
- Before commit (the sprint / cycle commit recommendation)
- When the plan changes significantly mid-shape

**When NOT to capture:**
- Trivial plans (refining a single AC bullet)
- Purely conversational turns
