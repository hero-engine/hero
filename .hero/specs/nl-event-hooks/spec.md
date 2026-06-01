---
title: Natural-Language Event Hooks — File-Level Triggers in Host Tool
slug: nl-event-hooks
type: feature
status: completed
tags: [hooks, claude-code, opencode, cursor, automation]
created: 2026-04-22
relations:
  - target: competitor-parity
    kind: parent
  - target: git-hook-integration
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Let users define natural-language hooks ("when I save a route handler,
regenerate the OpenAPI snippet") in a single `.hero/hooks/` directory, and have
`hero install` translate them into the host tool's native hook format
(Claude Code `PostToolUse`, OpenCode `tool.post`, Cursor rules where supported).

## Problem

Hero today has git lifecycle hooks (`hero hooks install` → branch ↔ spec
transitions). It does not have file-event hooks — the kind competitor markets as
"GitHub Actions for your IDE." Users who want "when I save X, do Y" today must
hand-edit `.claude/settings.json`, `opencode.json`, or `.cursorrules` per tool,
duplicating the same intent across surfaces.

## Design

### Hook authoring

A hook is a Markdown file in `.hero/hooks/`:

```markdown
---
name: regenerate-openapi-on-handler-save
event: file.save
match: "internal/api/handlers/**/*.go"
agent: api-engineer       # optional — pick which agent runs the prompt
mode: silent              # silent | confirm | foreground
---

The file `{{file}}` is an HTTP handler. Re-run the OpenAPI snippet generator
for this handler and update `docs/openapi.gen.yaml`. Do not modify any other
file. If the snippet is unchanged, exit silently.
```

Frontmatter fields:

| Field | Values | Default |
|---|---|---|
| `name` | unique slug | (required) |
| `event` | `file.save` \| `file.create` \| `file.delete` | (required) |
| `match` | glob (Go `filepath.Match` syntax via doublestar) | `**/*` |
| `agent` | agent slug from `.hero/agents/` or installed bundle | (none — runs as default agent) |
| `mode` | `silent` \| `confirm` \| `foreground` | `confirm` |

### Translation per host tool

`hero install project . --target claude` already writes `.mcp.json`. Extend it
to also write hook entries.

**Claude Code** — emit a `PostToolUse` matcher per hook into
`.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          { "type": "command", "command": "hero hook fire regenerate-openapi-on-handler-save --file ${file}" }
        ]
      }
    ]
  }
}
```

**OpenCode** — emit a `tool.post` entry in `opencode.json` calling the same
`hero hook fire` shim.

**Cursor** — Cursor doesn't have first-class file-event hooks; surface a
warning at install time and skip these hooks for the Cursor target.

### `hero hook fire` shim

```
hero hook fire <name> --file <path>
```

The shim:

1. Loads the hook by name from `.hero/hooks/`.
2. Checks the `match` glob against `--file`. Exits 0 silently if no match.
3. Renders the hook body with `{{file}}`, `{{relative_path}}`, `{{ext}}` substitution.
4. Writes the rendered prompt to stdout in the host tool's expected format
   (Claude Code `additionalContext`, OpenCode message payload).
5. Honors `mode`:
   - `silent` — emits prompt + suppresses tool UI surfacing
   - `confirm` — emits prompt + asks the host tool for user confirmation
   - `foreground` — emits prompt as a foreground task

### CLI surface

```
hero hook list                  # show all hooks + which targets they're installed for
hero hook show <name>           # print hook content + computed match
hero hook test <name> --file <path>   # dry-run: classify match, render prompt, no fire
hero hook install --target claude     # write hook entries to host tool config
hero hook uninstall --target claude   # remove hook entries
```

`hero install project ...` invokes `hero hook install` automatically as part of
the install flow when `.hero/hooks/` is non-empty.

### Safety

- Hooks never run shell commands — only emit prompts to the host tool.
- The host tool retains full permission control (Claude Code's permission
  modes, OpenCode's tool gating).
- `mode: silent` is opt-in; default is `confirm`.

## Changes

- `internal/hooks/nlhooks.go` — hook parser, matcher, renderer
- `internal/hooks/install_claude.go` — Claude Code settings.json emitter
- `internal/hooks/install_opencode.go` — OpenCode opencode.json emitter
- `internal/hooks/install_cursor.go` — Cursor warning + skip
- `internal/cli/hook.go` — `hero hook list|show|test|install|uninstall|fire`
- `internal/install/install.go` — call `hero hook install` if `.hero/hooks/` exists
- `commands/hero.md` — surface `/hero` routing for "set up a hook for X"
- `skills/agent-reliability.md` — note hook contract (never executes shell)

## Acceptance Criteria

- WHEN a user adds a Markdown file under `.hero/hooks/` with valid frontmatter THE SYSTEM SHALL recognize it via `hero hook list`
- WHEN `hero hook install --target claude` runs THE SYSTEM SHALL write a `PostToolUse` entry per hook into `.claude/settings.json`
- WHEN `hero hook install --target opencode` runs THE SYSTEM SHALL write a `tool.post` entry per hook into `opencode.json`
- WHEN `hero hook install --target cursor` runs THE SYSTEM SHALL warn that Cursor lacks file-event hooks and exit 0
- WHEN `hero hook fire <name> --file <path>` runs and the path matches the glob THE SYSTEM SHALL emit the rendered prompt on stdout
- WHEN `hero hook fire <name> --file <path>` runs and the path does not match THE SYSTEM SHALL exit 0 with no output
- WHEN `hero hook test <name> --file <path>` runs THE SYSTEM SHALL classify the match and print the rendered prompt without firing
- WHEN `hero install project ...` runs and `.hero/hooks/` is non-empty THE SYSTEM SHALL invoke `hero hook install` for the chosen target
- THE SYSTEM SHALL never execute shell commands as part of hook firing — only emit prompts

## Boundaries

- Does **not** run hooks itself — the host tool runs the prompt, Hero only emits it
- Does **not** support remote/HTTP hooks
- Does **not** add a daemon or file-watcher process — relies on host tool events
- Does **not** support every tool's hook format on day one — Cursor is best-effort
- Does **not** allow shell command bodies — prompts only
