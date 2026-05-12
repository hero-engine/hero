---
title: Architecture Overview
type: context
status: active
created: 2026-04-29
tags: [imported, architecture]
---

## Project Structure

- `commands/` — Slash command definitions (workflows like /design, /deliver, /diagnose)
- `agents/` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)
- `skills/` — Domain-specific knowledge and patterns
- `.hero/planning/` — Active specs being worked on
- `.hero/specs/` — Completed specs (archive)
- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)
- `hero.json` — Project configuration

## Design a feature

/design add CSV export
```

## Core Workflows

| Workflow | Commands | Purpose |
|---|---|---|
| **Build** | `/discover` → `/design` → `/deliver` | Explore, spec, implement |
| **Fix** | `/diagnose` → `/deliver` | Investigate, spec fix, implement |
| **Maintain** | `/convention`, `/scrub`, `/review` | Standards, cleanup, review |
| **Plan** | `/compose`, `/sprint`, `/import` | Break down, prioritize, sync |

## Next Steps

- [Installation](getting-started/installation.md) — Get Hero on your machine
- [Project Setup](getting-started/project-setup.md) — Initialize your first workspace
- [First Workflow](getting-started/first-workflow.md) — Walk through designing and delivering a feature

## Project Structure

Hero has two structural layers: the **Go application source** (the CLI itself) and the **`.hero/` directory** (per-project workspace managed by Hero).

---

## Repository Layout

```
hero/
├── cmd/hero/              # CLI entrypoint (main.go)
├── internal/              # Go packages (not importable externally)
│   ├── config/            # hero.json parsing and validation
│   ├── tracker/           # GitHub, Jira, Linear integrations
│   ├── spec/              # Spec CRUD, frontmatter, status management
│   ├── knowledge/         # Knowledge base operations
│   ├── mcp/               # MCP JSON-RPC server
│   ├── serve/             # HTTP API server
│   └── ...
├── agents/                # Agent definitions (markdown)
├── commands/              # Slash command definitions (markdown)
├── skills/                # Domain-specific knowledge (markdown)
├── docs/                  # Documentation site (MkDocs)
├── .hero/                 # Hero workspace (specs, knowledge, reports)
├── hero.json              # Project configuration
├── go.mod / go.sum        # Go module dependencies
├── Makefile               # Build targets
├── .goreleaser.yaml       # Release automation
└── mkdocs.yml             # Documentation site config
```

## Key directories

**`cmd/hero/`** — The CLI entrypoint. Parses commands, loads config, and dispatches to internal packages.

**`internal/`** — All Go business logic. Follows standard Go project layout — packages here are private to the module.

**`agents/`** — Markdown files defining agent roles, tool permissions, and workflow instructions. Loaded at runtime by command workflows. See [Agents Reference](agents/index.md).

**`commands/`** — Markdown files defining slash command workflows. Each file has YAML frontmatter (description) and a markdown body with orchestration logic. See [Commands Reference](commands/index.md).

**`skills/`** — Domain knowledge files covering technology stacks (Go, Python, React, etc.), engineering practices (testing, security, API design), and Hero-specific workflows. Agents reference skills for context-specific guidance.

---

## `.hero/` Directory

The `.hero/` directory is Hero's per-project workspace. It stores specs, knowledge, and reports.

```
.hero/
├── planning/              # Active specs being worked on
│   ├── features/          # Feature specs
│   │   └── user-avatars/
│   │       └── spec.md
│   ├── bugs/              # Bug investigation/fix specs
│   ├── chores/            # Chore specs
│   └── initiatives/       # Multi-spec initiatives
│       └── hero-cloud/
│           └── spec.md
├── specs/                 # Completed specs (archive)
│   ├── context-pipe/
│   │   └── spec.md
│   └── ...
├── knowledge/             # Project knowledge base
│   ├── context/           # Project context docs
│   │   └── project-overview/
│   │       └── spec.md
│   ├── conventions/       # Convention specs
│   ├── decisions/         # Decision records (ADRs)
│   └── notes/             # Captured notes
├── reports/               # Generated reports (scrub, check, retro)
└── hero.json              # Alt config location (if not at repo root)
```

## What goes in git

| Path | Git? | Notes |
|---|---|---|
| `.hero/planning/` | **Yes** | Active specs are shared with the team |
| `.hero/specs/` | **Yes** | Completed specs serve as project history |
| `.hero/knowledge/` | **Yes** | Conventions, decisions, and context are team knowledge |
| `.hero/reports/` | Optional | Reports are regenerable; commit if useful for history |
| `hero.json` | **Yes** | Shared project configuration |
| `agents/` | **Yes** | Agent definitions are part of the project |
| `commands/` | **Yes** | Command definitions are part of the project |
| `skills/` | **Yes** | Skills are part of the project |

---

## Spec Types and Locations

Specs follow a consistent format: a directory named by slug containing a `spec.md` with YAML frontmatter.

```yaml
---
title: User Avatar Upload
type: feature
status: in_progress
tracker_id: PROJ-142
priority: high
---

<!-- Imported from: AGENTS.md, docs/index.md, docs/project-structure.md -->
