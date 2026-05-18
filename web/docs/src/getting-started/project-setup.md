# Project Setup

## Initialize the Workspace

From your project root:

```bash
hero init
```

This creates the `.hero/` directory structure:

```
.hero/
├── mission.md      # project charter and first principles
├── NEXT.md         # shared projected handoff (solo mode)
├── QUEUE.md        # ready-work queue for cold starts
├── SNAPSHOT.md     # project-shape rollup (managed by `hero snapshot --project`)
├── planning/       # active specs (features/, bugs/, initiatives/)
├── specs/          # completed specs (archive)
├── knowledge/      # conventions, decisions, rules, context, notes, templates
├── next/           # per-user and local handoff projections
├── smoke/          # per-feature smoke metadata
├── events.log      # cross-session activity feed
├── graph.db        # generated graph store
├── index.db        # generated search index
└── hero.json       # project configuration
```

## Install into Your AI Tool

Hero provides its workflows as slash commands inside your AI coding tool. Install with:

=== "OpenCode"

    ```bash
    hero install project . --target opencode
    ```

=== "Cursor"

    ```bash
    hero install project . --target cursor
    ```

=== "Claude Code"

    ```bash
    hero install project . --target claude
    ```

=== "Codex"

    ```bash
    hero install project . --target codex
    ```

=== "GitHub Copilot"

    ```bash
    hero install project . --target copilot
    ```

=== "Generic MCP"

    ```bash
    hero install project . --target generic
    ```

!!! tip
    Use `--dry-run` to preview what files will be created or modified before committing to the install.

    ```bash
    hero install project . --target opencode --dry-run
    ```

### Domain packs

Hero ships content in layers: a shared `core/` pack and one or more
domain packs. The default is `engineering`. Other domains can be
scaffolded later.

```bash
hero install project . --target claude --domain engineering
hero domain               # show / switch active domain
```

### Monorepos

For monorepos where the AI tool runs from a subfolder, register each
subfolder as a satellite of the root install:

```bash
hero install project . --target cursor --workspace services/auth
hero install satellites list
hero install satellites add services/auth
hero install --repair     # verify symlinks/markers
```

### Reconciling existing installs

Multiple AI tools, drifted copies, post-upgrade resets:

```bash
hero install --migrate          # reconcile drifted copies across harnesses
hero verify-install             # audit install state
```

## Scan Your Project

Once installed, open your AI tool and run:

```
/scan
```

This detects your stack (languages, frameworks, tools) and seeds the knowledge base under `.hero/knowledge/`. It gives Hero context about your project so that workflows produce relevant, grounded output.

## Codify Conventions (optional)

If your team has specific patterns or standards, capture them early:

```
/convention
```

This walks you through documenting conventions (naming, architecture, testing patterns) and stores them in `.hero/knowledge/`. All subsequent Hero workflows will respect these conventions.

## What to Commit

Commit everything under `.hero/` **except**:

| Path | Commit? |
|---|---|
| `.hero/planning/` | Yes |
| `.hero/specs/` | Yes |
| `.hero/knowledge/` | Yes |
| `.hero/mission.md`, `NEXT.md`, `QUEUE.md`, `SNAPSHOT.md` | Yes |
| `.hero/hero.json` | Yes |
| `.hero/index.db`, `.hero/graph.db` | No — generated, rebuilt automatically |
| `.hero/events.log` | No — local activity feed |
| `.hero/next/*.local.md`, `.hero/sessions/` | No — per-machine state |
| `.hero/hero.local.json` | No — local overrides (tokens, personal prefs) |

!!! note
    Your `.gitignore` should include:
    ```
    .hero/index.db
    .hero/graph.db
    .hero/events.log
    .hero/hero.local.json
    .hero/next/*.local.md
    .hero/sessions/
    ```

## Next Steps

- [First Workflow](first-workflow.md) — Design and deliver your first feature with Hero
