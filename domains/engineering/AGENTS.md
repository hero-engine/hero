# Hero — Spec-Driven AI Engineering

This project uses **Hero** for spec-driven engineering workflows. Hero manages specs, integrates with work trackers (Jira, GitHub, Linear), and provides structured workflows via slash commands.

## Session Title

On the **first interaction** of every session, set a concise, descriptive session title that reflects what the user is working on (e.g. "design: auth flow", "fix: cart total rounding", "deliver: export-csv"). This keeps the session list navigable.

## Natural Language Routing

When the user describes what they want in natural language, route to the appropriate Hero slash command. **Run the command — don't just suggest it.**

| User intent | Command |
|---|---|
| Bug, error, broken, fix, investigate, diagnose | `/diagnose` |
| Challenge, revise, wrong root cause, re-diagnose, "this analysis is off", "also consider" | `/challenge` |
| New feature, build, design, add, plan | `/design` |
| Implement, deliver, ship, code, execute | `/deliver` |
| Review, PR, pull request, code review | `/review` |
| Break down, decompose, epic, sequence | `/compose` |
| Convention, pattern, standard, style | `/convention` |
| Decision, tradeoff, compare, choose, ADR | `/decide` |
| Explore, brainstorm, roadmap, ideate | `/discover` |
| Document, docs, explain, write docs | `/docs` |
| Release, deploy, version, ship | `/release` |
| Retro, postmortem, lessons learned | `/retro` |
| Note, capture, remember, save thought | `/note` |
| Scan, detect, onboard, stack analysis | `/scan` |
| Scrub, clean, dead code, duplication, weak types, slop | `/scrub` |
| Check, health, validate workspace | `/check` |
| Sprint, iteration, load sprint | `/sprint` |
| Import, pull issues, fetch from tracker, sync issues | `/import` |

When routing, pass the user's original context as arguments to the command. If the intent is ambiguous, present the top 2-3 options and ask.

**Vocabulary-aware routing.** When the workspace declares a `vocabulary:` or `methodology:` in `hero.json`, the user may speak in that dialect — "create a story" under `agile-scrum`, "shape a scope" under `shape-up`, "log a card" under `kanban`. Translate display terms back to canonical types before routing: `story` / `scope` / `card` all canonicalize to `feature`, so `hero new feature` is the right call. The on-disk frontmatter stays canonical (`type: feature`) regardless of how the user (or the dashboard) sees it. The active dialect is summarized in the "Active workspace dialect" section of this file when one is configured; engineering / default workspaces see no extra section and the canonical names are the user-facing names.

## Log significant events

After creating or updating a spec, modifying files, making a notable design
decision, or hitting a blocker, log it so other sessions can see:

```
hero event decision_made "Chose streaming CSV over buffered" --slug csv-export
hero event blocker_hit "Auth middleware rejects test tokens" --slug csv-export
```

Before starting work, check what other agents have done recently:

```
hero feed --since 1h
```

## Key Workflow

1. **Design first**: Use `/design` to create a spec before building anything
2. **Deliver from spec**: Use `/deliver` to implement from an approved spec
3. **Debug with specs**: Use `/diagnose` to investigate bugs and produce fix specs
4. **Never work on closed items**: Commands like `/diagnose` and `/deliver` check if the tracker issue is still open before starting work

## CLI Commands

These are run in the terminal, not as slash commands:
- `hero status` — workspace state and active specs
- `hero search <query>` — find specs by keyword
- `hero import` — import issues from tracker as spec scaffolds
- `hero sync pull <slug>` — sync spec status from tracker
- `hero note <slug>` — quick note capture
- `hero check` — health check

## Project Structure

- `commands/` — Slash command definitions (workflows like /design, /deliver, /diagnose)
- `agents/` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)
- `skills/` — Domain-specific knowledge and patterns
- `.hero/planning/` — Active specs being worked on
- `.hero/specs/` — Completed specs (archive)
- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)
- `hero.json` — Project configuration

## Important Rules

- **Don't assume.** Surface tradeoffs and ask questions if anything is unclear. Present multiple interpretations instead of picking one silently.
- **Simplicity first.** Write the minimum code that solves the problem. No speculative features, no unnecessary abstractions, and no error handling for impossible scenarios.
- **Surgical changes.** Touch only what is strictly required. Do not "improve" nearby code or refactor unrelated sections. Match the existing style perfectly.
- **Verify before reporting done.** Define clear success criteria for every task. Run tests or validation scripts and iterate until the criteria are met before reporting completion.
- **Local specs first.** When asked to work on bugs, features, or any tracked items, ALWAYS check what's already imported locally before querying the tracker. Use `hero search --list --type <type>` to find local specs. Only go to the tracker if the local search comes up empty. When working on multiple items (e.g. "diagnose 10 bugs"), select from locally imported specs — never bulk-query the tracker to pick work items.
- Always check spec status before doing work — don't investigate closed bugs or deliver completed specs
- When a tracker is configured, sync status with `hero sync pull` before starting work
- **Auto-capture learnings.** At the end of major workflows (`/deliver`,
  `/diagnose`, `/design`, `/retro`), evaluate whether the session produced
  knowledge worth persisting — design decisions made, debugging techniques
  that worked, conventions discovered or reinforced, surprising findings.
  If so, write a short entry to `.hero/knowledge/notes/` without prompting.
  Skip if nothing non-obvious was learned. This is enabled by default via
  `knowledge.auto_capture` in `hero.json`.
- **File useful queries back.** When `hero_ask` or research produces a
  synthesis that would help future sessions (architecture explanations,
  debugging playbooks, integration guides), write it to
  `.hero/knowledge/context/` as a knowledge entry. Every exploration
  should add up — ephemeral Q&A becomes permanent institutional memory.
- Specs use YAML frontmatter with fields: title, type, status, tracker_id, priority, severity
- Imported specs include tracker-prefixed fields (e.g. jira_status, jira_priority, jira_assignee) under a `# Jira` / `# Github` / `# Linear` comment header in frontmatter

## Keep handoff briefings current

Run `hero next path` to find the file you should write to. This resolves
based on the project's mode:

- **Solo mode** (default): `.hero/NEXT.md` — single shared briefing.
- **Team mode** (`next.mode: "team"` in hero.json): `.hero/next/<user>.md` —
  your personal briefing. Also update the one-liner for your name in the
  shared `.hero/NEXT.md` so teammates see what you're working on at a glance.

**At session start:** read your handoff file (via `hero next path`) before
doing anything else and surface the contents to the user. In team mode,
also check `.hero/NEXT.md` for team updates.

**At end of a turn where meaningful work happened** — finished a spec section,
landed a code change, made a design decision, or chose what to do next —
overwrite your handoff file with a fresh briefing. Always overwrite, never
append. Skip when the turn was purely conversational or exploratory.

In team mode, also update your one-liner in `.hero/NEXT.md` and optionally
add a team update entry if the work affects others.

See the `next-md` skill for the full format, quality bar, and shared-file
conventions.

## Survive context compaction

When you sense the conversation is getting long or the host tool warns about
context limits, take these steps to preserve continuity:

1. **Update your handoff briefing immediately** — don't wait for end-of-turn.
   Write the current state to your NEXT file so a post-compaction session
   can resume.
2. **Register active specs** — run `hero active register <session-id> <slug>`
   for any spec you're mid-delivery on. After compaction, the active session
   registry tells you what you were working on.
3. **Write partial progress** — if you're mid-implementation, commit what you
   have (even WIP) and note the stopping point in the handoff briefing.

After compaction, the host tool will reload AGENTS.md. Your first action
should be: read your handoff file, check `hero active list`, and run
`hero recap --since 1h` to rebuild context.

## Capture execution plans

When you generate an execution plan, implementation plan, or delivery plan —
whether via plan mode, a thinking step, or any other mechanism — persist it
as a Hero artifact so it survives the session and is visible to the team.

**How:** call `hero_plan` with the spec slug and plan content. This writes
the plan to `.hero/planning/features/<slug>/plan.md` (or `bugs/<slug>/plan.md`)
alongside the spec. If no spec exists yet, create a lightweight one first
via `/design`.

**Why:** plans generated inside a session vanish when the session ends.
Writing them to `.hero/` makes them searchable, visible in the dashboard,
and available to the next agent that picks up the work. It also lets
`hero drift` check whether the implementation matches the plan.

**When to capture:**
- Before starting implementation (the initial plan)
- When the plan changes significantly mid-delivery (overwrite with updated plan)
- When plan mode or a thinking step produces a structured execution strategy

**When NOT to capture:**
- Trivial plans ("read file, edit line, commit") — not worth persisting
- Plans for purely conversational tasks
