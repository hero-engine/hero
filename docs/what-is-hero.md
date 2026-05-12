# What Is Hero?

Hero gives your project a memory that AI coding tools can use.

## The Problem

AI coding assistants are smart in the moment and forgetful afterward.
You explain how your team handles errors, why a service was rewritten
last year, what the API contract looks like, what already failed when
someone tried this — and the next session you start, the model knows
none of it. So you explain it again. And again.

The result is a kind of expensive amnesia. Every session repeats the
same onboarding. Conventions drift because no one is enforcing them in
the moment. The same dead ends get re-explored. Decisions made in chat
months ago are unrecoverable. Across a team, this multiplies: nobody's
context is shared.

## What Hero Does

Hero captures the things a project actually accumulates — specs,
decisions, conventions, attempts, failures, acceptance criteria, recent
activity — into a folder in your repo called `.hero/`. That folder
becomes a structured corpus the AI tool can read on every session.

When you start a new session, Hero hands the model the right context up
front: what's in flight, what conventions apply, what's been tried, who
made what decision and why. When you finish, Hero captures what changed
so the next session begins where this one left off.

## How a Codebase Becomes Hero-Aware

The first time you point Hero at an existing project, you run a scan:

> "scan this codebase"

Hero detects the stack (languages, frameworks, build tools, CI, linters,
test runners), extracts code intelligence (symbols, packages, module
boundaries, dependencies), and seeds the knowledge base with starter
entries — context about how the project is laid out, candidate
conventions it picked up, rules it inferred from configuration. Those
entries are stubs; you and your team enrich them over time. But the
project starts with a real map of itself instead of a blank `.hero/`.

Scans are idempotent and incremental. You re-run after a big change
and Hero updates what shifted.

## The Two Habits Hero Encourages

Spec-driven engineering is the workflow shape Hero is built around:

- **Design before you build.** Before writing code for a feature, Hero
  produces a spec — a plain-markdown document with goals, approach, and
  acceptance criteria. You read it, push back, and approve it before
  any code exists. Misunderstandings get caught when they're cheap.
- **Diagnose before you fix.** Before writing a fix for a bug, Hero
  investigates the root cause and produces a diagnosis. You see what
  the model thinks is broken, and why, before it touches anything.

These aren't bureaucratic checkpoints. The spec is half a page of
markdown that takes a couple of minutes to read. The payoff is that
you stop discovering that the model misunderstood the task in code
review, and you get an artifact future sessions can refer back to.

## You Don't Memorize Commands

Hero installs as slash commands inside your AI tool — `/design`,
`/diagnose`, `/deliver`, and so on. You can use those if you want to
be explicit, but you don't have to. You just say what you want:

- "fix the login timeout bug" → Hero routes to `/diagnose`
- "add CSV export to the reports page" → routes to `/design`
- "implement the auth spec" → routes to `/deliver`
- "review my PR" → routes to `/review`
- "what's blocking the billing migration?" → asks the corpus

The routing table is part of the project context, so the model knows
which workflow matches which kind of ask. The slash commands are
there for when you know exactly what you want; natural language is
there for when you don't.

## What's in the Corpus

A Hero workspace contains a few kinds of things, all as files you can
read and commit to git:

- **Specs** — designs for features, diagnoses for bugs, decisions
  about architecture
- **Knowledge** — conventions ("how we name tests"), notes ("Brian
  explained the auth quirk"), rules ("never put secrets in env files
  checked into git")
- **History** — what work has been started, finished, abandoned;
  who's working on what; what acceptance criteria passed

Hero also maintains a local graph and search index over all of this,
so it can answer questions like "why does this file exist," "what's
blocked," and "what conventions apply to the code I'm editing right
now."

## What Hero Doesn't Try to Be

- **Not a replacement for your AI tool.** Hero plugs into the tools
  you already use — Claude Code, Cursor, OpenCode, Codex, Copilot, and
  any tool that speaks MCP. The model still does the thinking. Hero
  curates what the model sees.
- **Not a cloud service.** The corpus lives on your machine in your
  repo. Nothing leaves unless you set up a team server (optional).
- **Not a heavy framework.** Hero is a single Go binary. The corpus
  is plain markdown. You can read everything Hero writes with `cat`.

## Who It's For

Engineers who:

- Use AI coding tools day-to-day and feel the friction of session
  amnesia
- Work in codebases with non-obvious conventions, history, or
  domain rules that a model can't infer from the code alone
- Want a way to share context with teammates without writing long
  onboarding documents that nobody reads

If you've ever caught yourself thinking "I just explained this last
week," Hero is probably for you.

## Next

- [Why Hero](why-hero.md) — a deeper, more technical look at how
  Hero's pieces fit together and how it differs from prompt files,
  rule files, and other context tools
- [Installation](getting-started/installation.md) — install Hero and
  set up your first project
