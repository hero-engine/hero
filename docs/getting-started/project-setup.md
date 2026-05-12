# Project Setup

## Initialize the Workspace

From your project root:

```bash
hero init
```

This creates the `.hero/` directory structure:

```
.hero/
├── planning/      # Active specs being worked on
├── specs/         # Completed specs (archive)
├── knowledge/     # Conventions, decisions, context
└── hero.json      # Project configuration
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

!!! tip
    Use `--dry-run` to preview what files will be created or modified before committing to the install.

    ```bash
    hero install project . --target opencode --dry-run
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
| `.hero/hero.json` | Yes |
| `.hero/index.db` | No — local index, regenerated automatically |
| `.hero/hero.local.json` | No — local overrides (tokens, personal prefs) |

!!! note
    Your `.gitignore` should include:
    ```
    .hero/index.db
    .hero/hero.local.json
    ```

## Next Steps

- [First Workflow](first-workflow.md) — Design and deliver your first feature with Hero
