---
title: "Timely Briefs — Scheduled Synthesis Surfaces Over the Hero Graph"
type: feature
status: planning
priority: medium
horizon: now
tags: [briefs, synthesis, retrieval, html, scheduling, knowledge]
relations:
  - target: retrieval-contradiction-detection
    kind: depends-on
  - target: activity-digest
    kind: builds-on
  - target: hero-pulse
    kind: builds-on
  - target: html-report
    kind: related
  - target: synthesis-maintenance
    kind: related
---

## Problem

Hero already captures and structures everything that happens during work — feed
events, knowledge entries, decisions, conventions, spec status changes, commits,
MCP sessions. The substrate is rich. But the substrate is *passive*: it answers
questions when asked. Nothing in Hero proactively tells the human "here is what
changed, here is what connects, here is what is worth thinking about today"
without the human first sitting down, opening the harness, and typing a query.

`hero recap` answers "what happened since yesterday?" descriptively — grouped
commits by spec, status transitions, new knowledge entries. `hero pulse` gives
sprint-level narrative. Both are pull-mode and descriptive. Neither does the
interpretive work: surfacing non-obvious connections between recent captures and
older notes, naming the pattern the corpus is implicitly converging on,
spotlighting contradictions the contradiction detector found this week, or
asking the one question worth sitting with.

The gap: a **scheduled, interpretive synthesis surface** that runs without being
asked, reads broadly across the graph, produces a small number of high-signal
findings, and renders them in a form the human will actually read. Markdown
specs over 100 lines do not get read. A daily wall of recap text does not get
read either. The brief has to be short, visual, and ritualized.

## Goal

Ship `hero brief` — one engine, three cadences (daily, weekly, on-demand
pulse) — that reads the Hero graph over a configurable window, runs a small
fixed set of synthesis prompts against it, and emits a self-contained HTML
artifact as the primary deliverable, with the structured findings also
persisted as a markdown knowledge entry so the brief itself joins the corpus.

**Mission-fit.** This is the human-facing dual of "make the next agent session
start smarter than the last one ended." The brief makes sure the human starts
the day knowing what changed, what connects, and what is worth thinking about
— without having to ask. Floor-raising because the brief surfaces connections
a senior dev would notice naturally over coffee and a junior would miss
entirely. The brief itself becomes corpus, so future briefs (and future agent
sessions) can reference past briefs as part of the rolling synthesis.

## Design

### One engine, three cadences

The brief engine is a single code path parameterized by:

- **window** — duration (`24h`, `7d`, `Nh`, `Nd`) or absolute date range
- **template** — which synthesis prompt to apply (`daily`, `weekly`, `pulse`,
  or any user-defined template under `.hero/briefs/templates/`)
- **trigger** — `schedule` (came from a scheduled task), `manual` (user typed
  `hero brief`), or `composed` (called from another command)

Three cadences are just configured invocations:

| Cadence | Default trigger | Default window | Default template |
|---|---|---|---|
| Morning brief | scheduled, 06:00 local weekdays | 24h | `daily` |
| Weekly synthesis | scheduled, Monday 06:00 local | 7d | `weekly` |
| On-demand pulse | `hero brief [--window=Nh] [--template=X]` | 24h | `daily` |

### Inputs (everything in the graph touched in the window)

The engine pulls a unified ranked candidate set from the window:

1. **Feed events** — all events in `.hero/events.log` with timestamp in window.
2. **Spec status changes** — specs whose `status` flipped in window (via the
   bitemporal `valid_from` on the status node).
3. **New / modified knowledge entries, decisions, conventions** — via
   `valid_from` in window.
4. **Commits** — `git log --since` filtered to commits whose touched files
   appear in the `files_touched` index (reuse `hero recap` mapping).
5. **MCP sessions** — sessions with `last_active` in window from the session
   index.
6. **Contradictions surfaced in window** — call `contradict.CheckRecent(window)`
   to get the set of warnings raised since the last brief.

All six sources are merged, deduplicated by `(type, key)`, and ranked using
the existing unified retrieval ranker (BM25 + recency + node degree). The top
N (default 50, configurable) become the candidate set fed to the synthesis
template.

### Output structure (per cadence)

The three default templates produce different shapes; the engine is agnostic
to the shape, it just renders whatever the template produces.

**`daily` template output:**

```
{
  "connections": [
    { "title": "...", "narrative": "...", "node_a": <ref>, "node_b": <ref>, "hops": 1 },
    { ... },
    { ... }
  ],
  "pattern": { "title": "...", "narrative": "..." },
  "question": "..."
}
```

**`weekly` template output:**

```
{
  "thesis": { "title": "...", "narrative": "..." },
  "contradictions": [<contradict.Warning>, ...],
  "knowledge_gaps": [{ "topic": "...", "narrative": "..." }, ...],
  "action": { "title": "...", "narrative": "..." }
}
```

The `contradictions` array reuses the `Warning` struct from
`internal/contradict` — do not duplicate or paraphrase contradiction signals.

### Skip-when-empty threshold

Before invoking the LLM, the engine evaluates the candidate set against a
threshold: at least **3 distinct ranked entities** AND at least **1 of**
{spec status change, new knowledge entry, commit touching a tracked spec,
contradiction warning}. If the threshold is not met, the engine logs
`brief.skipped` to the feed and writes nothing. No "quiet day" placeholder.
The threshold is configurable per cadence under `briefs.<cadence>.threshold`.

### Output: HTML primary, markdown secondary

**Primary deliverable** — a self-contained HTML file at
`.hero/briefs/<YYYY-MM-DD>-<cadence>.html`. Self-contained means inline CSS,
inline SVG, no external assets — same constraint as `hero report`.

The HTML includes:

- Header with cadence name, window, generated-at timestamp.
- Each finding rendered as a card with the narrative.
- A **connection subgraph SVG** for daily briefs: nodes are graph entities
  (spec, knowledge entry, decision, etc.), edges are the relations the brief
  surfaced. Small (≤ 12 nodes), inline, no library dependency. The SVG is the
  unlock vs markdown — being able to show "this idea from March is one hop
  from this spec delivered yesterday."
- A **contradictions panel** for weekly briefs, side-by-side rendering of
  conflicting nodes per warning.
- Footer with the engine version, template name, and a link to the persisted
  knowledge entry.

**Secondary persistence** — the structured findings (the JSON shape above)
plus a short prose summary are written as a markdown knowledge entry to
`.hero/knowledge/briefs/<YYYY-MM-DD>-<cadence>.md` with frontmatter:

```yaml
---
type: knowledge
kind: brief
cadence: daily
window_start: 2026-05-09T06:00:00-07:00
window_end:   2026-05-10T06:00:00-07:00
generated_at: 2026-05-10T06:00:12-07:00
referenced_nodes:
  - { type: spec, key: timely-briefs }
  - { type: decision, key: api-auth-strategy-v2 }
  - ...
tags: [brief, brief/daily]
---
```

The markdown form is what joins the graph and gets indexed for retrieval.
Future briefs can reference past briefs ("the pattern named in last Monday's
synthesis is now showing up in three more specs"). The HTML is what the
human reads.

### Scheduling

Use **`hero automations`** (the existing scheduled-task infrastructure) as
the scheduling backend. Rationale:

- It already exists and is workspace-scoped — survives across machines if
  the workspace is checked into git.
- It degrades gracefully: if the machine was asleep at 06:00, the next
  invocation when the daemon wakes runs the brief over the *actual* elapsed
  window, not a stale 24h. The engine accepts a `--catchup` flag that
  expands the window to "since last successful brief of this cadence."
- No external cron dependency, no per-OS launch agent dance, no new daemon.

OS cron / launchd / systemd timers are explicitly **not** a v1 dependency.
Users who want OS-level scheduling can wire `hero brief --template=daily` into
their own cron; the engine works the same regardless of trigger.

Scheduled briefs are configured in `hero.json`:

```json
{
  "briefs": {
    "daily":  { "enabled": true,  "schedule": "0 6 * * MON-FRI", "window": "24h", "template": "daily" },
    "weekly": { "enabled": true,  "schedule": "0 6 * * MON",     "window": "7d",  "template": "weekly" },
    "templates_dir": ".hero/briefs/templates"
  }
}
```

Disabled by default on a fresh workspace. `hero brief --enable daily` flips
it on and creates the automation entry.

### Prompt template surface

Templates live in `.hero/briefs/templates/<name>.md` and are user-tunable.
Hero ships built-ins for `daily.md`, `weekly.md`, `pulse.md` (the latter two
templates default to the same as `daily` until the user differentiates).

A template is a markdown file with a frontmatter block declaring the
expected output schema and a body that is the prompt itself. Variables
available to every template:

- `{{ .Window }}` — the window descriptor ("last 24 hours", "May 3 – May 10")
- `{{ .CandidateNodes }}` — the ranked candidate list (rendered as a
  digestible block, capped at the configured top-N)
- `{{ .Contradictions }}` — `contradict.Warning` set raised in window
- `{{ .PriorBriefs }}` — last 3 briefs of this cadence (for continuity:
  "did the question I asked Monday get answered?")
- `{{ .Mission }}` — workspace `mission.md` contents

If the template body is missing, the engine falls back to the built-in for
that name. Editing the built-in is a simple `cp` from the embedded asset
into the templates dir.

### Relationship to write-through synthesis maintenance

Write-through synthesis maintenance (separate spec) is the **push** side:
every write into the graph triggers cheap integration ops (edge updates,
contradiction flags) and proposes (but does not auto-apply) expensive ones
(merge, supersede, rewrite summaries). Maintenance keeps the substrate
coherent so retrieval has good material.

Briefs are the **pull** side: scheduled reads of the substrate that surface
synthesis to the human. The brief engine **does not** perform any
maintenance ops. It does not edit existing knowledge entries, it does not
propose merges, it does not rewrite summaries. It reads, synthesizes, and
emits one new artifact (the brief). If the brief notices a maintenance
opportunity (e.g. two knowledge entries clearly say the same thing), it can
*surface* that as a finding ("two entries on retry-with-backoff appear to be
duplicates — consider merging") but the merge itself is the maintenance
spec's job.

### CLI surface

```
hero brief                                # run pulse, default window=24h, template=daily
hero brief --window=7d --template=weekly  # arbitrary
hero brief --catchup                      # expand window to since-last-successful
hero brief --dry-run                      # show what would be in the candidate set, skip LLM
hero brief --enable daily                 # turn on the daily automation
hero brief --disable weekly               # turn off the weekly automation
hero brief --list                         # show last N briefs with paths
hero brief --open                         # opens the most recent brief HTML in default browser
```

### MCP tool — `hero_brief`

```json
{
  "name": "hero_brief",
  "description": "Generate a synthesis brief over a window of the Hero graph",
  "inputSchema": {
    "type": "object",
    "properties": {
      "window":   { "type": "string", "description": "Duration (24h, 7d) or ISO date range. Default: 24h" },
      "template": { "type": "string", "description": "Template name. Default: daily" },
      "format":   { "type": "string", "enum": ["html_path", "json", "markdown"], "description": "Default: html_path" }
    }
  }
}
```

`format=html_path` returns the path to the generated HTML; `json` returns
the structured findings; `markdown` returns the persisted knowledge entry.
This lets agents request briefs from inside other workflows (e.g. a sprint
planner agent could call `hero_brief --window=7d --format=json` to ground
its planning in the past week's synthesis).

## Changes

- `internal/brief/brief.go` — `Engine` struct, `Run(ctx, opts) (Result, error)`,
  candidate selection, ranking, threshold check, template invocation,
  artifact write
- `internal/brief/templates.go` — embed built-in `daily.md`, `weekly.md`,
  `pulse.md`; resolution order (workspace dir then embedded fallback)
- `internal/brief/render_html.go` — self-contained HTML rendering, including
  inline SVG subgraph generation
- `internal/brief/render_markdown.go` — knowledge entry serialization
- `internal/brief/brief_test.go` — engine, threshold, template fallback,
  HTML self-containment, markdown frontmatter
- `internal/cli/brief.go` — `hero brief` command + flags
- `internal/cli/root.go` — register `briefCmd`
- `internal/serve/mcp_tools.go` — register `hero_brief` MCP tool
- `internal/automations/` — wire scheduled invocation of the brief engine via
  the existing automation runner (no new scheduling primitive)
- `internal/contradict/contradict.go` — add `CheckRecent(window) []Warning`
  helper if not already present
- `hero.json` schema — add `briefs` section
- `.hero/briefs/templates/` — created on first `hero brief --enable`

## Acceptance Criteria

- WHEN `hero brief` runs with no arguments THE SYSTEM SHALL produce a brief
  over the last 24 hours using the `daily` template
- WHEN `hero brief --window=7d --template=weekly` runs THE SYSTEM SHALL
  produce a weekly synthesis over the last 7 days
- WHEN the candidate set in the window contains fewer than 3 distinct ranked
  entities OR none of {spec status change, new knowledge entry, commit
  touching a tracked spec, contradiction warning} THE SYSTEM SHALL skip
  generation and log `brief.skipped` to the feed
- WHEN a brief is generated THE SYSTEM SHALL write a self-contained HTML
  file to `.hero/briefs/<YYYY-MM-DD>-<cadence>.html` with no external asset
  dependencies
- WHEN a brief is generated THE SYSTEM SHALL write a markdown knowledge entry
  to `.hero/knowledge/briefs/<YYYY-MM-DD>-<cadence>.md` with frontmatter
  including `cadence`, `window_start`, `window_end`, `generated_at`, and
  `referenced_nodes`
- WHEN a `daily` template brief is generated THE SYSTEM SHALL include an inline
  SVG subgraph of at most 12 nodes showing the relations between the surfaced
  connections
- WHEN a `weekly` template brief is generated THE SYSTEM SHALL include a
  contradictions section populated by `contradict.CheckRecent(window)` and
  SHALL render conflicting nodes side-by-side
- WHEN `hero brief --catchup` runs THE SYSTEM SHALL expand the window to
  begin at the timestamp of the last successful brief of the same cadence
- WHEN `hero brief --dry-run` runs THE SYSTEM SHALL print the candidate set
  and skip any LLM invocation
- WHEN a template file exists at `.hero/briefs/templates/<name>.md` THE
  SYSTEM SHALL prefer it over the embedded built-in of the same name
- WHEN a template file does not exist THE SYSTEM SHALL fall back to the
  embedded built-in
- WHEN a scheduled brief runs and the workspace machine was asleep past the
  scheduled time THE SYSTEM SHALL run the brief over the actual elapsed
  window since the last successful run of that cadence
- WHEN `hero_brief` MCP tool is invoked with `format=json` THE SYSTEM SHALL
  return the structured findings without writing the HTML artifact
- THE SYSTEM SHALL NOT mutate any existing graph node, knowledge entry,
  decision, convention, or spec as part of generating a brief
- THE SYSTEM SHALL include the previous 3 briefs of the same cadence in the
  template context as `{{ .PriorBriefs }}` so the synthesis can reference
  continuity

## Boundaries

- Does **not** perform synthesis maintenance (merging entries, rewriting
  summaries, applying supersession edges) — that is the
  `write-through-synthesis-maintenance` spec's job. Briefs may *surface*
  maintenance opportunities as findings but never act on them. See
  `synthesis-maintenance` spec for the push side.
- Does **not** ship email, Slack, push, or any non-local delivery in v1 —
  HTML file at a stable path only. The user can open it manually or wire
  their own delivery.
- Does **not** ship a relevance-feedback loop (thumbs up/down, mark as
  noise) in v1 — the ranker has nothing to learn from yet. Defer until
  briefs prove useful enough that the user wants to tune them.
- Does **not** federate across cloud / cross-repo — single-workspace only.
  Cloud-federated briefs are downstream of `cloud-mcp`.
- Does **not** inject the brief into the next agent session's context. That
  is interesting and adjacent (and arguably more on-mission), but it is a
  separate spec — this one is about surfacing to the human side. The
  knowledge-entry persistence does mean briefs become eligible for normal
  retrieval, which is the seam.
- Does **not** replace `hero recap` or `hero pulse` — those remain the
  descriptive layer; briefs are the interpretive layer. Briefs may *call*
  recap/pulse internally as part of candidate selection.
- Does **not** require a new scheduling daemon or OS-level cron entry —
  uses existing `hero automations` infrastructure.
- Does **not** generate briefs on workspaces with `briefs.daily.enabled =
  false` (the default for fresh workspaces) — opt-in via `hero brief --enable`.
