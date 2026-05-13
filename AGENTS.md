# Hero — Spec-Driven AI Engineering

<!-- hero:managed-start v=dev -->
## Hero — Spec-Driven AI Engineering

This project uses **Hero** for spec-driven engineering workflows. Hero manages specs, integrates with work trackers (Jira, GitHub, Linear), and provides structured workflows via slash commands.

### Session Title

On the **first interaction** of every session, set a concise, descriptive session title that reflects what the user is working on (e.g. "design: auth flow", "fix: cart total rounding", "deliver: export-csv"). This keeps the session list navigable.

### Natural Language Routing

When the user describes what they want in natural language, route to the appropriate Hero slash command. **Run the command — don't just suggest it.**

| User intent | Command |
|---|---|
| Bug, error, broken, fix, investigate, diagnose | `/diagnose` |
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
| Check, health, validate workspace | `/check` |
| Sprint, iteration, load sprint | `/sprint` |
| Import, pull issues, fetch from tracker, sync issues | `/import` |

When routing, pass the user's original context as arguments to the command. If the intent is ambiguous, present the top 2-3 options and ask.

### Key Workflow

1. **Design first**: Use `/design` to create a spec before building anything
2. **Deliver from spec**: Use `/deliver` to implement from an approved spec
3. **Debug with specs**: Use `/diagnose` to investigate bugs and produce fix specs
4. **Never work on closed items**: Commands like `/diagnose` and `/deliver` check if the tracker issue is still open before starting work

### CLI Commands

These are run in the terminal, not as slash commands:
- `hero status` — workspace state and active specs
- `hero search <query>` — find specs by keyword
- `hero import` — import issues from tracker as spec scaffolds
- `hero pull <slug>` — sync spec status from tracker
- `hero note <slug>` — quick note capture
- `hero check` — health check

### Project Structure

- `commands/` — Slash command definitions (workflows like /design, /deliver, /diagnose)
- `agents/` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)
- `skills/` — Domain-specific knowledge and patterns
- `.hero/planning/` — Active specs being worked on
- `.hero/specs/` — Completed specs (archive)
- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)
- `hero.json` — Project configuration

### Important Rules

- **Don't assume.** Surface tradeoffs and ask questions if anything is unclear. Present multiple interpretations instead of picking one silently.
- **Simplicity first.** Write the minimum code that solves the problem. No speculative features, no unnecessary abstractions, and no error handling for impossible scenarios.
- **Surgical changes.** Touch only what is strictly required. Do not "improve" nearby code or refactor unrelated sections. Match the existing style perfectly.
- **Verify before reporting done.** Define clear success criteria for every task. Run tests or validation scripts and iterate until the criteria are met before reporting completion.
- **Local specs first.** When asked to work on bugs, features, or any tracked items, ALWAYS check what's already imported locally before querying the tracker. Use `hero search --list --type <type>` to find local specs. Only go to the tracker if the local search comes up empty. When working on multiple items (e.g. "diagnose 10 bugs"), select from locally imported specs — never bulk-query the tracker to pick work items.
- Always check spec status before doing work — don't investigate closed bugs or deliver completed specs
- When a tracker is configured, sync status with `hero pull` before starting work
- **Hero handoff travels with commits.** When committing, stage any modified `.hero/NEXT.md` and `.hero/next/*.md` alongside your code changes. These are projected handoff files — if they don't travel with the commit, the next session (possibly on another machine) starts cold. `hero next install-hooks` installs a pre-commit hook that automates this; the rule is your backstop when the hook isn't installed.
- Capture novel learnings to `.hero/knowledge/` at the end of major workflows
- Specs use YAML frontmatter with fields: title, type, status, tracker_id, priority, severity
- Imported specs include tracker-prefixed fields (e.g. jira_status, jira_priority, jira_assignee) under a # Jira/GitHub/Linear comment header
<!-- hero:managed-end -->

This project uses **Hero** for spec-driven engineering workflows. Hero manages specs, integrates with work trackers (Jira, GitHub, Linear), and provides structured workflows via slash commands.

## ⚡ Read these FIRST every session (until `project-charter` ships auto-injection)

The mission and current recovery state are not yet auto-injected into your context. Until `project-charter` lands (see `.hero/planning/features/project-charter/spec.md`), the **first thing you do every session in this repo** is read these in order:

1. **[`.hero/mission.md`](.hero/mission.md)** — Hero's locked charter. *Sidekick brain for AI-augmented knowledge work.* Three modes, five principles, locked vocabulary, six anti-patterns, mission-fit test. **Highest-priority context.**
2. **[`.hero/NEXT.md`](.hero/NEXT.md)** — what's open right now, what's next, what we tried, what's blocked. The persistent cross-session briefing.
3. **[`.hero/knowledge/notes/recovery-strategy-conversation/spec.md`](.hero/knowledge/notes/recovery-strategy-conversation/spec.md)** — captures all 14 strategic moves from the 2026-04-28 recovery session in user's voice. **Read this before the initiative spec** — the meta-reasoning isn't in the strict-form artifacts.
4. **[`.hero/planning/initiatives/get-back-on-track/spec.md`](.hero/planning/initiatives/get-back-on-track/spec.md)** — the active recovery initiative coordinating 9 child features.
5. *Optional but useful:* [`.hero/knowledge/notes/v2-delivery-audit-2026-04-28/spec.md`](.hero/knowledge/notes/v2-delivery-audit-2026-04-28/spec.md) (audit findings — verify against code; the audit got several things wrong, corrections noted in NEXT.md).

If you skip these, you will re-derive the work and probably drift. The whole mission of Hero (especially principle #3 — *sessions start omniscient*) is making this read unnecessary in the future. Until then, it is the load-bearing manual step.

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
| Handoff, checkpoint, save session, save state, switching tools, force refresh NEXT | `/handoff` |
| Resume, pick up where we left off, pickup where we left off, continue, let's continue, load NEXT | `/resume` |
| Scan, detect, onboard, stack analysis | `/scan` |
| Scrub, clean, dead code, duplication, weak types, slop | `/scrub` |
| Check, health, validate workspace | `/check` |
| Sprint, iteration, load sprint | `/sprint` |
| Import, pull issues, fetch from tracker, sync issues | `/import` |
| Why does X exist, where did Y come from, history of Z, origin chain, traceback | `/why <target>` |
| What's blocked, what's stuck, what's open and waiting, dependency chain, failing ACs | `/blocked` |

When routing, pass the user's original context as arguments to the command. If the intent is ambiguous, present the top 2-3 options and ask.

## Capture handoff state as you work

If `next.projected` is enabled in `.hero/hero.json` (run
`hero next migrate-to-projection` to opt in), the agent half of
NEXT.md is no longer hand-written — it's projected from graph
events. See [skills/next-handoff-emit.md](skills/next-handoff-emit.md)
for the cadence. Three commands carry the load:

```
hero next ask "<verbatim or one-sentence paraphrase of user's prompt>"
hero next suggest "<paste-ready next prompt, in user's voice>"
hero next reflection "<one-line lesson worth carrying forward>"
```

Fire `ask` when the user directs new work, `suggest` at the end of a
meaningful work unit, `reflection` when a non-obvious lesson surfaces.
Skip when nothing meaningful happened. Reads: `hero next suggest`,
`hero next ask`, `hero next reflection` (no args).

## Cold-start prompts and the ready queue

Every spec carries a `## Kickoff` section — a paste-ready cold-start
prompt for picking that work back up in a fresh session. Format and
quality bar are in [skills/kickoff-prompt.md](skills/kickoff-prompt.md).
Authors: `/design` writes the kickoff at scaffold time; `/deliver` and
`/diagnose` rewrite it at status flips. Hand-edit anytime — the spec
is source of truth.

When the user asks **"give me a prompt for a new session"** or
**"what should I work on?"**:

- For a specific spec → call `hero_kickoff <slug>` (MCP) or
  `hero list <slug> --format kickoff` (CLI).
- For the ranked ready queue → call `hero_queue` (MCP) or read
  `.hero/QUEUE.md` (the pre-rendered snapshot the pre-commit hook
  keeps current). `.hero/QUEUE.md` is the surface for harnesses that
  can't pop a terminal at session start (Claude Code).
- Only hand-author a kickoff if no spec covers the request — and if
  you do, drop it as a new spec via `/design` so it joins the queue.

`hero queue` is curated (ready specs, priority sort, kickoff format).
`hero list` is the power-user query (filter by type/status/horizon/
tag/ready/blocked/pinned/mine/stale, sort by recency/status/alpha/
priority, format text/json/table/kickoff).

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

## Keep NEXT.md current

NEXT.md is the single living artifact that lets a fresh session pick up
where the last one left off. It has two halves:

- **Machine half** — auto-written every turn by `hero next checkpoint`
  (run from a host-tool Stop hook). Branch, recent commits, dirty files,
  hot files. You don't write this; ignore the `<!-- BEGIN HERO MACHINE
  STATE -->` block.
- **Agent half** — Last user ask, Just finished, Next, Blocked on, Tried
  and failed, Context. *You* write this.

Run `hero next path` to find the file. Solo mode → `.hero/NEXT.md`.
Team mode → `.hero/next/<user>.md`.

**At session start:** read your NEXT file and surface it to the user.

**Update the agent half when** (and only when):

1. The user said something the next session needs to know — quote it
   verbatim into the **Last user ask** section
2. The intent shifted — what we're trying to do changed
3. An approach was tried and failed — record it under **Tried and failed**
4. You're about to switch tools or context feels close to full — force a
   full refresh (the `/handoff` slash command does exactly this)

You do **not** need to update the agent half just because files were
edited or a commit was made — the machine half captures that. Skip
conversational turns entirely.

**Always overwrite, never append.** Hard cap 60 lines for the agent half.

See the `next-md` skill for the full format and quality bar.

## Survive context compaction

When the conversation is getting long or the host tool warns about
context limits:

1. **Run `/handoff`** — force-refresh both halves of NEXT.md now. The
   Stop hook normally handles the machine half, but `/handoff` runs it
   immediately and reminds you to refresh the agent half too.
2. **Commit WIP** — if mid-implementation, commit what you have so the
   next session sees it in `git log`.

After compaction, the host tool reloads AGENTS.md. First action: read
NEXT.md, then `hero recap --since 1h` to rebuild context.

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
