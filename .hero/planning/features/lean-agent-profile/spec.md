---
title: Lean Agent Profile — Opt-In Trim for Top-Tier Models
type: feature
status: planning
tags: [agents, mcp, profile, dx, future]
created: 2026-04-22
relations:
  - target: agent-cold-start
    kind: related
horizon: next
smoke: deferred
---

## Goal

Let users running Hero against a top-tier model (Opus 4.7, GPT-5, similar)
opt into a lean `AGENTS.md` template and a smaller MCP tool surface, without
changing any defaults. Capable models save context budget; smaller models
keep the full safety net and discoverability they rely on.

## Why later, not now

The dogfooding review on 2026-04-22 confirmed that the full `AGENTS.md`
("Important Rules", 17-row routing table, project structure block) and the
full 15-tool MCP surface are *load-bearing* for smaller / local models
(Haiku, Sonnet 4.6, Qwen, Llama). Trimming defaults is a regression for
that audience. Opt-in lean is the right shape but not urgent — top-tier
model users can already work productively with the verbose setup; the
savings are nice-to-have, not unlocking new capability.

Defer until at least one of:
- A user explicitly asks for it
- Hero has telemetry showing top-tier-model sessions burning meaningful
  context on the verbose template
- The CLI ships another feature that meaningfully expands `AGENTS.md`

## Design (sketch — fill in when delivering)

### Profile switch

`hero.json`:

```json
{
  "agents": {
    "profile": "standard"
  },
  "serve": {
    "tool_filter": {
      "profile": "standard"
    }
  }
}
```

Both default to `"standard"` — current behavior, byte-identical. `"lean"`
is opt-in.

### Lean `AGENTS.md`

Target ~40 lines. Specific cuts (all things a top-tier model already
internalizes from its system prompt):

- Drop "Important Rules": "Don't assume", "Simplicity first", "Surgical
  changes", "Verify before reporting done"
- Compress routing table from 17 rows to 8 (collapse `/scrub`, `/scan`,
  `/check` and similar low-frequency intents)
- Drop "Project Structure" block

Keep: session title rule, compressed routing table, key workflow,
Hero-specific rules ("local specs first", "check status before working",
"capture novel learnings"), frontmatter conventions, NEXT.md instruction
block.

### Lean MCP profile

Expose 5 tools instead of 15:

| Default-exposed | Why |
|---|---|
| `hero_context` | Bootstrap context for the file currently being edited |
| `hero_read_spec` | Full spec by slug — replaces grep+read for spec corpus |
| `hero_search` | FTS5 search across the corpus |
| `hero_ask` | Extractive Q&A — fast knowledge lookup |
| `hero_claim` | Mark/release work — only mutating tool kept lean |

Hidden under lean: `hero_status`, `hero_check`, `hero_nudge`, `hero_list`,
`hero_knowledge`, `hero_pulse`, `hero_skill_run`, `hero_velocity`,
`hero_test_generate`, `hero_demo_record`. All still callable when profile
is `"standard"` or `"full"`.

### Install UX

```
hero install project . --target opencode --profile lean
hero install project . --target opencode                  # standard (default)
```

`--profile` flag passes through to both the `AGENTS.md` template selection
and the `tool_filter.profile` write.

## Changes (sketch)

- `internal/install/install.go` — `--profile` flag, template selection
- `internal/install/templates/AGENTS-lean.md` — new lean template
- `internal/serve/mcp_filter.go` — named `lean` profile alongside existing
  `default`/`full` mechanism
- `internal/config/config.go` — `AgentsConfig.Profile`, validate against
  known set

## Acceptance Criteria

- WHEN `hero install project ... --profile lean` runs THE SYSTEM SHALL write the lean `AGENTS.md` template and set `agents.profile` and `serve.tool_filter.profile` to `"lean"`
- WHEN `hero install project ...` runs without `--profile` THE SYSTEM SHALL write the current standard `AGENTS.md` byte-identical to today
- WHEN the MCP server starts with `serve.tool_filter.profile = "lean"` THE SYSTEM SHALL expose only the 5 named lean tools
- WHEN `agents.profile` is set to an unknown value THE SYSTEM SHALL log a warning, fall back to `"standard"`, and continue
- THE SYSTEM SHALL leave existing projects' `AGENTS.md` and `tool_filter` config untouched on `hero install` re-runs without `--profile`
- THE SYSTEM SHALL preserve all 15 MCP tools as callable when profile is `"standard"` or `"full"`

## Boundaries

- Does **not** change any defaults — lean is strictly opt-in
- Does **not** auto-detect model capability — the user picks the profile
- Does **not** remove any `AGENTS.md` content from the standard template
- Does **not** delete any MCP tools — only filters which are exposed
- Does **not** ship until there is concrete demand or telemetry justification
