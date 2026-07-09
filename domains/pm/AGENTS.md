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

| User intent | Vocabulary-variant phrasing | Command |
|---|---|---|
| New feedback, customer ask, support escalation, sales note, "this came in" | | `/triage` |
| Refine, tighten, "make this ready", "draft AC", INVEST, EARS; refine an existing spec (PRD / feature / epic / initiative); ambiguous specs that "won't deliver cleanly" | | `/refine` |
| Prioritize, rank, RICE, ICE, WSJF, value-vs-effort, "what's first" | | `/prioritize` |
| Hand off, send to engineering, "ready for dev", "flip owner to engineering" | | `/handoff` (flips `owner: pm → engineering` on the same artifact) |
| Create a story / spec / feature / pitch / scope | "draft a story" / "shape a scope" / "create an issue" | `hero spec new <slug> --type feature` (alias `hero design <slug>`) |
| Create a bug | "log a bug" | `hero spec new <slug> --type bug` |
| Create an epic / pitch / theme | "frame a pitch" / "frame a theme" / "create an epic" | no CLI scaffolder (`hero spec new` has no epic type) — hand-author `.hero/planning/epics/<slug>/spec.md` per the registered `epic` spec type |
| Create a roadmap-item / bet / initiative | "add a bet" / "add a roadmap initiative" | `hero spec new <slug> --type initiative` |
| Draft PRD, write requirements, product doc, "spec this out" | | `/prd` |
| Pitch, Shape Up, "shape this", appetite, betting table | | `/pitch` |
| Roadmap, "what's coming", reconcile roadmap, "show the roadmap" | | `/roadmap` |
| Stale roadmap, "clean up the roadmap", "what's been dropped" | | `/roadmap` (reconcile mode) |
| Discover, explore, "we don't know enough about X", customer research | | `/discover` |
| Metric, success measure, KPI, "how do we measure this" | | `/metrics` |
| Interview, customer call, user research, "design an interview" | | `/discover --interview <count>` |
| Release notes, announce, "what shipped this week" | | `/release-notes` |
| Capacity, cycle/sprint/iteration planning, standup / weekly update | | (P1, ships v1.5 — no v1 surface) |
| Duplicate intake, cluster feedback, "is this a duplicate" | | `/triage` (duplicate clustering via intake-triager + duplicate-detector) |
| Confusing customer signal, "what's actually being asked here" | | invoke `pm-investigator` directly (agent — no command shim ships with pm) |
| Search across PRDs, specs, intake, roadmap | | `hero search <query>` (CLI) |
| Why does this exist, "trace this back" | | `/why` |
| What's stuck, blocked items, dependencies | | `/blocked` |
| Note, capture, remember this conversation | | `/note` |
| Decision, tradeoff, choose between options | | `/decide` |
| Review this PRD / spec / roadmap | | invoke `pm-reviewer` directly (agent — no `/review` command ships with pm) |
| Retro, postmortem, lessons learned on a shipped item | | `/retro` |

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

- **PM:** `/discover`, `/handoff`, `/metrics`, `/pitch`, `/prd`, `/prioritize`, `/refine`, `/release-notes`, `/roadmap`, `/triage` — see Natural Language Routing above for what each does. (`/discover` and `/handoff` override the core commands of the same name below.)
- **Core (installed with every pack):** `/blocked`, `/capture`, `/check`, `/convention`, `/decide`, `/docs`, `/hero`, `/import`, `/note`, `/resume`, `/retro`, `/scan`, `/why`.

### Agents Reference

Every agent an installed pm workspace ships, no links:

- **PM:** `discovery-researcher`, `duplicate-detector`, `handoff-coordinator`, `intake-triager`, `pm-delivery-lead`, `pm-investigator`, `pm-reviewer`, `prd-author`, `prioritization-strategist`, `product-strategist`, `roadmap-curator`, `story-writer` — see Key Workflow above for how the core five fit together.
- **Core (installed with every pack):** `convention-author`, `documentation-engineer`, `project-context-builder`, `session-primer`.

### Skills Reference

Every skill an installed pm workspace ships, no links:

- **Writing:** `story-writing-invest`, `acceptance-criteria-ears`, `prd-structure`, `prd-anti-patterns`, `pitch-writing-shape-up`, `roadmap-framing`.
- **Frameworks:** `prioritization-frameworks`, `opportunity-solution-trees-torres`, `metrics-design`.
- **Process / methodology:** `continuous-discovery-cadence`, `sprint-planning`, `cycle-planning`.
- **Curation:** `intake-classification`, `duplicate-detection`, `dependency-mapping`, `evidence-synthesis`.
- **Cross-domain:** `handoff-protocol`, `cross-domain-graph-query`.
- **Operational:** `pm-preset-detection`.
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
