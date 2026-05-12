# Why Hero

For engineers evaluating whether to adopt Hero. This is the deeper
read — what problem it actually solves, how it's built, and how it
differs from the smaller context tools you may already be using.

## The Problem, More Precisely

AI coding tools are, in effect, exceptionally well-read interns with
no memory. Every session they read the code in front of them, the
prompt you give them, and whatever instruction files you've pointed
them at — and that's it. They don't know:

- **What you've already tried.** Three sessions ago someone explored
  this exact refactor and abandoned it because of a subtle locking
  issue. That investigation is gone.
- **Why the code looks the way it does.** The reason `OrderService`
  has a weird retry loop is that the upstream API was unreliable in
  2024. That context lives in a Slack thread.
- **What your conventions are.** Not the ones in your style guide —
  the ones in your team's head. "We never throw raw HTTP errors past
  the handler layer." A new model session has no way to know.
- **Who decided what.** Last week's architecture call settled the
  question of sync vs async for billing events. That decision is in
  a calendar invite description.
- **What's blocked or in flight.** Three feature branches are
  half-done. Two are blocked on a third. The model has no idea.

The result: every session starts cold. You re-explain. You restate
constraints. Models repeat past mistakes because they don't know
they're past mistakes. And the cost compounds — small at first,
unbearable once a project has any real history.

## Why Existing Solutions Aren't Enough

Most teams reach for one of these first:

**CLAUDE.md / `AGENTS.md` / Cursor rule files.** A single markdown
file the model reads on every session. Works for stable, repo-wide
facts ("Run tests with `pnpm test`"). Doesn't work for anything that
changes session-to-session or anything too specific to load into
every prompt. Becomes either too short to be useful or too long to be
afforded.

**Hand-written context in the prompt.** "Remember last time you tried
X, it didn't work because Y." Fine for the current session,
disappears the next. Doesn't scale across teammates.

**Project wikis / Notion / Confluence.** The information might exist
there. The model doesn't read it. You'd have to manually paste in
relevant pages every session.

**Conversation history.** Some tools persist conversation history.
That helps within a single thread, but isn't structured, isn't
shared, isn't queryable, and gets dropped or summarized when context
fills up.

What's missing in all of these: **structure**, **lifecycle**, and
**retrieval**. A 5000-line CLAUDE.md isn't useful — the model needs
the *right* 200 lines for what it's doing right now.

## What Hero Adds

Hero is built around three ideas:

### 1. A Local Corpus, Not a Flat File

Hero stores project context as a structured corpus under `.hero/`:

```
.hero/
├── mission.md
├── planning/        specs in flight (features, bugs, initiatives)
├── specs/           completed specs (archive)
├── knowledge/
│   ├── conventions/
│   ├── decisions/
│   ├── rules/
│   ├── context/
│   └── notes/
├── events.log       cross-session activity feed
├── graph.db         generated dependency/relationship graph
└── index.db         generated full-text search index
```

Everything in `planning/`, `specs/`, and `knowledge/` is plain
markdown, committed to git, reviewable in PRs. The `graph.db` and
`index.db` are derived state — regenerated from the markdown.

This is the durable thing. Models come and go; the corpus stays.

### 2. A Spec Lifecycle as the Forcing Function

Capturing context only works if it gets captured at the right moment.
Hero uses the spec lifecycle as the trigger:

- `/discover` brainstorms → produces ideas, possibly a `discovery` spec
- `/design` produces a `feature` spec with goals, approach, acceptance
  criteria, and a `## Kickoff` section that future sessions can use to
  cold-start
- `/diagnose` produces a `bug` spec with the investigation, root cause,
  and fix plan
- `/decide` produces a `decision` spec (an ADR, essentially)
- `/convention` captures a coding convention by analyzing the existing
  codebase
- `/deliver` implements an approved spec, records what worked, captures
  learnings, marks AC results
- `/retro` runs a post-delivery retrospective against the original spec
- `/handoff` preserves session state so the next prompt starts warm

The result: by the time you finish delivering a feature, you've
naturally produced a design spec, an ADR if you made a hard call, a
captured convention if you established a pattern, and recorded AC
results — all without writing a "doc" as a separate activity.

### 3. Natural Language as the Primary Surface

A workflow tool only gets used if the friction to invoke it is below
the friction of just typing what you want. Hero ships ~27 slash
commands. Nobody memorizes 27 slash commands.

Hero solves this by making **natural language the default**. Each
install registers a routing table inside the AI tool's project
context, so the model knows that "fix the login bug" routes to
`/diagnose`, "add CSV export" routes to `/design`, "implement the
auth spec" routes to `/deliver`, "review my PR" routes to `/review`,
and so on. The user describes what they want; the model picks the
workflow and runs it.

The slash commands still exist for when you want to be explicit —
useful in scripts, automations, and when training new teammates. But
the everyday experience is conversational. The workflow shape stays
the same; only the syntax disappears.

This is a small thing that matters a lot. It's the difference
between Hero being something the team adopts and Hero being a tool
nobody remembers to use.

### 4. A Graph and a Retrieval Layer

Markdown alone isn't enough. Hero builds a graph over the corpus
(specs, files, decisions, conventions, AC results, tracker links) and
a full-text index over the content. That makes possible:

- `hero relevant src/auth/session.go` — given the files you're
  editing right now, what specs/decisions/conventions/notes apply?
- `hero why csv-export` — what chain of decisions, specs, and ideas
  led to this work item existing?
- `hero blocked` — what's stuck, on what dependency, for how long?
- `hero ask "what is our error response format?"` — natural-language
  question against the corpus
- `hero impact internal/auth/session.go` — what specs/work depend on
  this file?
- `hero coverage csv-export` — which AC have evidence, which don't?

The same retrieval layer feeds the AI tool's context. `/resume` at
the start of a session loads what's in flight, what conventions
apply, what's recently changed, what dead ends to avoid — chosen by
the graph and index, not pasted into a flat file.

## Bootstrapping from an Existing Codebase

The obvious question: "Do I have to write all this context from
scratch?" No. The first thing you do in any non-trivial project is
run a scan:

```bash
hero scan          # full scan: stack + code intelligence + knowledge stubs
hero scan --code   # just code intelligence
hero scan --dry-run
```

A scan does three things at once:

1. **Stack detection.** Identifies languages, frameworks, build
   tools, CI, linters, test runners, and common patterns. The
   detected stack is recorded in the knowledge base as `context`
   entries the model reads on every session.
2. **Code intelligence.** Extracts symbols, packages, module
   boundaries, dependency graphs, and import relationships into the
   graph store. This is what makes `hero relevant`, `hero impact`,
   and the auto-context features work on day one.
3. **Knowledge seeding.** Generates starter entries — candidate
   conventions, inferred rules, project context. They're explicitly
   *stubs* (clearly marked), meant to be reviewed and enriched by
   the team rather than treated as ground truth.

A scan with `code_scan.depth: deep` in `hero.json` adds LLM-generated
descriptions to symbols, which is slower but produces much richer
retrieval. The default `normal` depth is fast and structural-only.

Scans are idempotent and incremental — re-running after a refactor
updates what changed without trampling hand-edited knowledge. This is
the move that took Hero from "interesting in greenfield" to "useful in
an existing codebase with five years of history."

## Architected as Core + Domain Packs

Although everything above is framed in terms of AI *coding* tools,
the engine itself is not coding-specific. Hero is built in two
layers:

- **Hero Core** lives under `core/` and contains the domain-agnostic
  pieces: the corpus structure, graph store, retrieval layer, spec
  lifecycle plumbing, install machinery, MCP server, and the small
  set of commands that any vertical needs (handoff, resume, search,
  ask, etc.).
- **Domain packs** live under `domains/<name>/` and contain the
  vocabulary and workflows for a specific knowledge-work vertical:
  the slash commands, agents, skills, mission, and (where useful)
  custom spec types.

The shipped pack is `domains/engineering/` — the engineering
vertical (Hero Code). It's complete and in production use: 11
domain-specific slash commands, 30 specialist agents, 31 skills, and
the spec types (feature, bug, decision, initiative) the engineering
loop is built around.

A second pack, `domains/sales/`, exists as a scaffold today — its
`mission.md` and directory structure are checked in to triangulate
what belongs in core versus what belongs in a vertical. It is not
yet usable as a working domain. The point of having it sit in the
repo alongside engineering is that it forces the boundary to be
real, not aspirational.

What this means in practice:

- `hero install --domain engineering` installs the engineering pack
  on top of core. `hero install --domain sales` will install the
  sales pack on top of core when that pack is built out.
- New verticals (support, ops, research, writing, design) can be
  built as additional domain packs that reuse the core engine.
- The corpus shape — specs, knowledge, graph, retrieval — is the
  same regardless of vertical. Only the vocabulary changes.

If you're evaluating Hero today, you're almost certainly evaluating
the engineering pack. But the core/domain split is a real
architectural commitment, not a marketing line — it shapes how the
codebase is organized and how the install path works.

## How It Plugs In

Hero is harness-agnostic. The same corpus drives:

- **Claude Code** — via slash commands installed at `~/.claude/`,
  agents, skills, and an MCP server
- **Cursor** — via `.cursor/` rules, commands, and MCP
- **OpenCode** — via `.opencode/` and MCP
- **Codex** — via `.codex/`
- **GitHub Copilot** — via `.github/` instruction files
- **Generic MCP** — any tool that speaks the protocol

`hero install project . --target <tool>` translates Hero content into
each tool's expected format. The `--migrate` flag detects existing
harness setups and reconciles them. The `--workspace` flag handles
monorepos where the harness runs from a subfolder.

You can mix tools. The corpus is the same; only the surface changes.

## What's Inside the Box Today

The numbers move as the project does — current counts:

- **27 slash commands** for the spec lifecycle and supporting
  workflows
- **34 specialist agents** (feature-delivery-lead, debug-investigator,
  security-reviewer, scrubbers, architects, etc.)
- **45 skills** capturing domain patterns (stack-specific guidance,
  test strategy, security review, debugging methodology, etc.)
- **41 MCP tools** exposing the corpus to the AI tool

The agents and skills aren't magic — they're prompt patterns refined
across real deliveries. You can read them; they're files in the
`agents/` and `skills/` directories.

## The Cost

Honest accounting:

- **Discipline.** Hero only helps if you actually use specs. If you
  skip straight to "/just write the code," you'll get less out of
  it. The cost is mostly social — convincing the team that 3 minutes
  of spec review saves 30 minutes of code review.
- **Disk.** A typical workspace runs 10–100 MB depending on history.
  The corpus is small.
- **Setup.** First-time install is ~5 minutes. `hero init` then
  `hero install project . --target <tool>` for each AI tool you use.
- **Cognitive overhead.** A new convention. New commands. The
  payoff is that the model takes more of the load.

## When It's a Fit

- You use AI coding tools day-to-day and feel the session-amnesia
  pain
- Your project has non-obvious conventions, history, or domain rules
- You work with teammates and would benefit from shared context
- You prefer your project's context to live in your repo, in git,
  reviewable, rather than in a SaaS

## When It Isn't

- You're solo on a throwaway script
- You don't use AI coding tools
- You strongly prefer one big static prompt file and never want to
  think about it again

## Next

- [Installation](getting-started/installation.md) — get Hero running
  in your project
- [The Core Loop](concepts/core-loop.md) — the spec lifecycle in
  more detail
- [Knowledge Base](concepts/knowledge-base.md) — how the corpus is
  structured
- [Commands Reference](commands/index.md) — the full slash command
  surface
