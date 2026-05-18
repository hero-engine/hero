---
title: Skills as Git-Native Reusable Commands — hero skill save / run
slug: skills-git-native
type: feature
status: completed
milestone: v0.3
tags: [skills, git, reusable, commands, workflows, automation]
created: 2026-04-13
relations:
  - target: hero-context-pipe
    kind: related
  - target: agent-contribution-tracking
    kind: related
horizon: now
---

## Goal

Let engineers capture, share, and reuse multi-step agent workflows as versioned, git-native "skills" — named command sequences that combine Hero context with shell commands and agent instructions, stored in the repo and runnable with `hero skill run <name>`.

## Problem

Agent-assisted engineering produces successful patterns: a specific sequence of context pulls, prompts, and shell commands that reliably accomplishes a task (e.g., "add a new API endpoint", "write a migration", "draft a decision doc from a GitHub issue"). These patterns live in engineers' heads or Slack threads — they're not captured, versioned, or reusable.

Linear's equivalent is saved views and template workflows — the team accumulates institutional knowledge in the tool, not in individual memory. Hero needs the same: skills as first-class artifacts that the team can share, improve, and run consistently.

## Design

### Skill Format

A skill is a markdown file stored in `.hero/skills/<name>.md`:

```markdown
---
title: Add New API Endpoint
slug: add-api-endpoint
version: 1
tags: [api, codegen, gin]
author: chet-bellows
created: 2026-04-13
---

## Steps

1. `hero context internal/api/` — load API conventions into context
2. `hero ask "what are the naming rules for gin handlers?"` — get handler naming conventions
3. Run: `touch internal/api/{{name}}_handler.go`
4. Prompt agent: "Using the conventions above, implement a Gin handler for {{endpoint}} following our existing patterns. See context for error handling rules."
5. `hero triage internal/api/{{name}}_handler.go` — check for convention violations after generation

## Parameters

- `name` — Go-safe filename for the new handler (e.g., `user_profile`)
- `endpoint` — HTTP route description (e.g., `GET /api/users/:id/profile`)

## Notes

Check that the handler is registered in `internal/api/router.go` after generation.
```

Skills use `{{parameter}}` template syntax for runtime substitution.

### CLI Interface

```
hero skill list                          # list all available skills
hero skill show <name>                   # show a skill's steps and parameters
hero skill run <name> [--param key=val]  # run a skill interactively
hero skill save                          # capture a skill from the current session
hero skill edit <name>                   # open skill in $EDITOR
hero skill rm <name>                     # remove a skill
```

**`hero skill run`** executes skill steps in order:
- Shell command steps (`Run:`) are executed directly
- Hero command steps (`hero context`, `hero ask`, etc.) are executed and output is streamed
- Agent prompt steps (`Prompt agent:`) are printed to stdout for the agent to pick up (or piped via `hero context pipe`)
- Parameters are interpolated at runtime

**Interactive parameter prompting:**
```
$ hero skill run add-api-endpoint
  name: user_profile
  endpoint: GET /api/users/:id/profile
Running skill: Add New API Endpoint
...
```

**Non-interactive (all params via flags):**
```
hero skill run add-api-endpoint --param name=user_profile --param "endpoint=GET /api/users/:id/profile"
```

### `hero skill save`

Capture a skill from the current session's command history:

```
hero skill save
  Skill name: add-api-endpoint
  Title: Add New API Endpoint
  Commands from this session will be included.
  Review and edit in .hero/skills/add-api-endpoint.md? [Y/n]
```

Hero reconstructs the session's `hero` commands and shell commands into a skill template, opens it in `$EDITOR` for the engineer to annotate with `Prompt agent:` steps and `{{parameter}}` slots, then saves to `.hero/skills/`.

Since skills are in `.hero/` (which is committed), `git commit` is the sharing mechanism. No cloud sync needed.

### Skill Discovery

`hero skill list` shows:
```
Available skills (3):

  add-api-endpoint     Add New API Endpoint      [api, codegen, gin]
  write-migration      Write a DB Migration       [db, migration]
  draft-decision       Draft a Decision Doc       [docs, decisions]

Run `hero skill show <name>` for details.
```

Skills in subdirectories are supported: `hero skill run team/add-api-endpoint` resolves to `.hero/skills/team/add-api-endpoint.md`.

### MCP Exposure

Skills are listed as resources via MCP, and `hero skill run` is exposed as a tool:

```json
{
  "name": "hero_skill_run",
  "description": "Run a named skill workflow",
  "inputSchema": {
    "type": "object",
    "properties": {
      "skill": { "type": "string" },
      "params": { "type": "object", "additionalProperties": { "type": "string" } }
    },
    "required": ["skill"]
  }
}
```

This allows agents to invoke skills by name without knowing the steps — the skill encapsulates the workflow.

### Git-Native Sharing

Skills are committed with the repo. Teams share skills via PR. Skills are versioned (the `version` frontmatter field increments on edit). `hero skill list` shows the git log of recent changes to `.hero/skills/`:

```
hero skill log add-api-endpoint
```

## Changes

- `internal/skills/skills.go` — skill loading, parameter interpolation, step parsing
- `internal/skills/runner.go` — step execution (shell, hero commands, prompt output)
- `internal/skills/capture.go` — `hero skill save` session reconstruction
- `internal/cli/skill.go` — `hero skill` command group (list, show, run, save, edit, rm, log)
- `internal/serve/mcp.go` — add `hero_skill_run` tool and skills resource listing
- `.hero/skills/` — new directory, committed to repo

## Acceptance Criteria

- Skills are stored as markdown in `.hero/skills/<name>.md` with standard frontmatter
- `hero skill run <name>` executes all steps in order with parameter interpolation
- `hero skill run` prompts for missing parameters interactively
- `--param key=val` flags suppress interactive prompting for named parameters
- `hero skill save` reconstructs a session into an editable skill template
- `hero skill list` lists all `.hero/skills/` files with title and tags
- `hero skill show <name>` prints the full skill markdown
- Skills are git-committed with the repo and shareable via normal git workflow
- `hero_skill_run` MCP tool works end-to-end
- Unknown `{{parameter}}` slots without defaults return a clear error, not a silent empty substitution

## Boundaries

- Does **not** run LLM calls — `Prompt agent:` steps output text to stdout for the agent to act on
- Does **not** enforce skill correctness — skills are trusted artifacts authored by the team
- Does **not** implement skill versioning beyond the `version` frontmatter field and git log
- Shell command execution uses the user's shell (`$SHELL`) — no sandboxing in v1
