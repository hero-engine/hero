---
title: "Tripwire System — Forbidden-Option Guardrails for Model Sessions"
slug: tripwire-system
type: feature
status: completed
priority: high
horizon: now
tags: [knowledge, context-injection, drift-prevention]
relations:
  - target: architectural-drift-detection
    kind: related
completed_at: 2026-06-09T18:38:23Z
---

# Tripwire System — Forbidden-Option Guardrails for Model Sessions

## Problem

Models drift from project first principles during long sessions even when the right context is loaded at session start. The failure mode is specific and repeatable:

1. Session starts. Hero primes context: mission, active work, conventions, rules.
2. Model reads everything, operates correctly for the first phase of work.
3. Model hits a dead end (something doesn't work, a dependency fails, a design assumption breaks).
4. Model enters "options analysis" mode — brainstorms alternatives.
5. Model proposes an option that directly contradicts the project's reason for existing.

The context was *available* — it was loaded at prime time. The problem is **retrieval timing**: by the time the model needs to check constraints (step 4), the prime context has decayed through compaction or simply been overwhelmed by hours of conversation. There is no mechanism to re-inject constraints at the moment the model is about to make a decision.

Real example from project-example codebase-mlx: the project exists to replace a Python/PyO3 wrapper with native Rust. Two hours into a session, the model hit a model-loading failure, brainstormed alternatives, and proposed… PyO3. The literal thing being replaced. The mission statement, the v0.2 status doc, and the auto-memory all said "the MLX C++ kernels are fine; replace the Python wrapper." The model read all of it at session start and still produced PyO3 as an option.

This is not a content problem — the knowledge exists. It's a **delivery timing** problem. Hero loads context at session start and on-demand when the model asks. But at the critical moment (dead-end → brainstorm), nothing re-injects the constraints that matter most.

### Why rules and conventions aren't enough

Rules describe **how to do things** — coding standards, workflow requirements, CI conventions. They're scoped to files and surfaced during `hero_context` calls. They answer "how should I approach this?"

The gap is **what must never be done** — explicit forbidden options with the reasoning attached. These need different retrieval behavior (always present, not just on file-scope match), different prominence (harder to miss than a list of coding rules), and different semantics (a violated tripwire isn't a style issue — it's a project-direction failure).

### Why this is a Hero problem, not a CLAUDE.md problem

CLAUDE.md is tool-specific (Claude Code only). Hero serves every harness — Claude Code, Cursor, opencode, Aider, CI agents. A drift-prevention mechanism that only works in one tool is a workaround, not a solution. Tripwires belong in the Hero knowledge layer where they compound across sessions, sync across team members, and work in any harness.

## Goal

Prevent models from proposing options that violate project first principles, by introducing a knowledge type and injection mechanism that keeps forbidden-option constraints present at the exact moments where drift happens — not just at session start.

## Design

Three components, each addressing a different part of the failure chain:

### 1. Tripwire Knowledge Type

A new spec type: `tripwire`. Semantically distinct from rules — a tripwire says "this specific thing must never be proposed, and here's why."

**Storage**: `.hero/knowledge/tripwires/<slug>/spec.md`

**Frontmatter**:
```yaml
---
title: Do Not Use PyO3
type: tripwire
status: active          # active | superseded
scope: ["*.rs", "*.py"] # optional file-scope (like rules)
triggers: [pyo3, "python bindings", "python wrapper", "python ffi"]
severity: critical       # critical | high — how bad is violating this
created: 2026-04-29
---
```

The `triggers` field is unique to tripwires. It's a list of keywords/phrases that cause this tripwire to surface in retrieval when a query contains a matching term. Unlike `scope` (file-path globs), `triggers` match against query text.

**Required sections**:
- `## Constraint` — what is forbidden, stated directly
- `## Why` — why it's forbidden — the reasoning the model needs to not re-derive a different conclusion
- `## Instead` — what to do instead (the positive direction)

**Example**:
```markdown
---
title: Do Not Use PyO3
type: tripwire
status: active
triggers: [pyo3, "python bindings", "python wrapper", "python runtime"]
severity: critical
created: 2026-04-29
---

# Do Not Use PyO3

## Constraint

Do not propose, evaluate, or implement PyO3 as a solution to any
problem in this project. Do not treat it as an "option" in any
analysis.

## Why

This project exists specifically to replace the PyO3/Python wrapper
with native Rust calling the MLX C++ API directly. PyO3 is the thing
being replaced. Proposing it as an option is proposing we undo the
project.

## Instead

Use the MLX C++ kernels directly via Rust FFI. The C++ API is the
asset; the Python layer is the liability. If a model won't load via
the C++ path, that's a v0.2 bug to fix, not a reason to reintroduce
Python.
```

### 2. Context Injection — Always Present, Re-Injectable

Tripwires get three injection points, each addressing a different failure window:

#### A. Prime-time injection (session start)

In `BuildPrimeContext()`, add a `## Tripwires` section **before** conventions and decisions. Always included — not gated by `includeKnowledge`. Rendered with the full Constraint text (not just title), because the constraint text is short and high-signal.

```
## Tripwires (Do Not Violate)
- **no-pyo3** [critical]: Do not propose, evaluate, or implement PyO3
  as a solution. This project exists to replace the Python wrapper.
  Instead: use MLX C++ kernels directly via Rust FFI.
- **no-mlx-lm-dep** [high]: Do not treat oxideai/mlx-lm as a
  load-bearing dependency. It was a v0.1 bootstrap, not an
  architectural choice. Instead: own model definitions directly.
```

This is the first line of defense. The model sees tripwires before any work begins. But it's not sufficient alone — it decays over long sessions.

#### B. hero_context injection (on-demand, file-scoped)

When `hero_context` is called with file paths, include matching tripwires alongside rules. Two matching criteria:

- **Scope match**: tripwires with `scope` globs matching the queried files
- **Global**: tripwires with no scope or `scope: ["*"]` are always included

Tripwires appear in a dedicated `## Tripwires` section in the context output, separate from rules, with higher visual prominence.

#### C. hero_anchor tool (explicit re-anchoring)

New MCP tool: `hero_anchor`. Designed to be called at decision moments — when the model is about to propose alternatives or has hit a dead end.

**Input**: Optional `context` string describing what the model is deciding. Used to match relevant tripwires by trigger keywords. If omitted, returns all tripwires.

**Output**:
1. Project mission statement (from `.hero/mission.md`, truncated to the Mission section)
2. All active tripwires with full Constraint + Why + Instead text
3. Any trigger-matched tripwires highlighted at the top if `context` was provided

**Tool description** (what the model sees in the tool list):
> Re-anchor on project first principles. Call this BEFORE proposing architectural alternatives, when hitting a dead end, or when brainstorming solutions. Returns project mission and all active tripwires (forbidden options). Prevents drift from first principles.

The tool description is the behavioral nudge — it tells the model *when* to call it, not just what it does.

### 3. Retrieval Integration — Trigger-Based Surfacing

When `hero_ask` or `hero_search` processes a query:

1. Tokenize the query text
2. Check each active tripwire's `triggers` list for keyword overlap
3. If any triggers match, include the matching tripwire(s) in results with maximum priority (above the standard type-boost)

This means asking Hero "should we use PyO3?" or "what about python bindings?" will surface the relevant tripwire before any other result. The model doesn't have to know to call `hero_anchor` — any query touching a trigger keyword will surface the guardrail.

Implementation: in the retrieval layer, before running the standard BM25/FTS5 query, do a fast keyword scan against all active tripwire triggers. Matching tripwires are prepended to results. This is O(tripwires × query_tokens) — negligible for realistic tripwire counts (projects typically have 3-15).

### 4. Skill Integration — Automatic Re-Anchoring

Modify the prompt templates for these skills to include an explicit anchor step:

- **`/decide`**: Before evaluating options, call `hero_anchor` with the decision context. Verify no proposed option violates a tripwire.
- **`/deliver`**: Before proposing implementation approach, call `hero_anchor`. Check that the design doesn't introduce a forbidden dependency or pattern.
- **`/diagnose`**: Before proposing fixes, call `hero_anchor`. Verify the fix direction doesn't violate constraints.
- **`/design`**: Before finalizing the design, call `hero_anchor`. Check all proposed components against tripwires.

The skill prompts should frame this as a hard gate, not a suggestion:
> "Before proceeding, call hero_anchor with your current context. If any proposed option appears in the tripwire list, eliminate it — do not present it as an option with caveats."

### 5. Graph Integration

Tripwires become nodes in the knowledge graph:

- **Node type**: `Tripwire` (new, alongside existing Feature, Bug, Decision, Convention, Rule, etc.)
- **Edges**: `constrains` → features/specs this tripwire is relevant to
- **Bitemporal**: track when tripwires were created and superseded, so the graph records the history of what was forbidden and when
- **Federation**: tripwires follow the same scoping as other knowledge — `local` (per-dev), `team` (per-repo), `unit` (cross-repo)

### 6. CLI Surface

- **`hero tripwire add`**: Interactive creation of a new tripwire. Prompts for constraint, why, instead, triggers, scope.
- **`hero tripwire list`**: List all active tripwires with trigger keywords.
- **`hero tripwire check <text>`**: Check whether a piece of text (e.g., a proposal) matches any tripwire triggers. Returns matching tripwires. Useful for CI integration or manual checking.
- **`hero anchor`**: CLI equivalent of `hero_anchor` MCP tool. Prints mission + tripwires.

### Design decisions

**Why a new type instead of a tag on rules?**
Rules and tripwires have different retrieval behavior (rules are file-scoped on demand; tripwires are always-on with keyword triggers), different injection behavior (rules are part of context; tripwires are part of prime), and different semantics (rules guide; tripwires forbid). A tag would mean overloading the rule type with two incompatible behaviors.

**Why include full constraint text in prime context?**
Tripwires are short (1-3 sentences) and high-signal. Including just the title ("no-pyo3") loses the critical context — *why* it's forbidden. The model needs the reasoning to avoid re-deriving a different conclusion. The token cost is minimal: 10 tripwires × ~50 tokens each = ~500 tokens in prime context.

**Why triggers instead of just full-text search?**
FTS5 would find tripwires that mention "PyO3" when someone searches for "PyO3." But the boost would compete with other results mentioning the same term. Triggers are an explicit author-provided signal that means "if this keyword appears, this tripwire MUST surface, above everything else." It's a hard priority override, not a relevance score.

**Why not scan model output for trigger keywords?**
MCP tools can't see the model's output before it's sent. Hero provides context to the model; it doesn't filter the model's responses. The mechanism has to be about better input at the right time, not output validation. This is a fundamental constraint of the MCP architecture.

## Acceptance Criteria

- WHEN a `tripwire` type spec is created in `.hero/knowledge/tripwires/` THE SYSTEM SHALL parse it with `triggers`, `scope`, and `severity` frontmatter fields and include it in spec discovery.
- WHEN a session starts THE SYSTEM SHALL include all active tripwires in prime context with full Constraint text, rendered before conventions and decisions, regardless of `includeKnowledge` flag.
- WHEN `hero_context` is called with file paths THE SYSTEM SHALL include tripwires whose `scope` matches the queried files, in a dedicated Tripwires section separate from rules.
- WHEN `hero_anchor` is called THE SYSTEM SHALL return the project mission statement plus all active tripwires with full Constraint, Why, and Instead text.
- WHEN `hero_anchor` is called with a `context` string THE SYSTEM SHALL highlight tripwires whose triggers match keywords in that context.
- WHEN `hero_ask` or `hero_search` query text matches a tripwire's trigger keyword THE SYSTEM SHALL include the matching tripwire in results with maximum priority, above standard type-boost ranking.
- WHEN a tripwire is superseded THE SYSTEM SHALL stop including it in prime context and retrieval results but preserve it in graph history.
- THE SYSTEM SHALL expose `hero tripwire list` and `hero tripwire check <text>` CLI commands.
- THE SYSTEM SHALL register `hero_anchor` as an MCP tool with a description that tells the model when to call it (before proposing alternatives, when hitting a dead end).

## Changes

### New files
- `internal/spec/spec.go` — add `TypeTripwire Type = "tripwire"` constant
- `.hero/knowledge/tripwires/` — directory for tripwire specs

### Modified files
- `internal/spec/spec.go` — add `Triggers []string` field to Spec struct, parse from frontmatter, add `TypeTripwire`, update `typeFromPath`/`statusFromPath` for tripwires path, update `IsKnowledge()` to include tripwires
- `internal/serve/prime.go` — add tripwire section to `BuildPrimeContext()` and `BuildPrimeContextJSON()`, always included (not gated by `includeKnowledge`)
- `internal/serve/mcp.go` — register `hero_anchor` tool, inject tripwires in `toolContext()` alongside rules, add trigger-matching in `toolAsk`/`toolSearch`
- `internal/index/index.go` — add `FindTripwiresForFiles()` (like `FindRulesForFiles()`), add `FindTripwiresByTrigger(query string)` for keyword matching
- `internal/retrieval/retrieval.go` — prepend trigger-matched tripwires to retrieval results before standard ranking
- `internal/graph/schema.go` — add `Tripwire` node type
- `internal/graph/ingest.go` — ingest tripwire specs as graph nodes with trigger metadata
- `internal/context/format.go` — add tripwire formatting for all output formats
- `internal/context/truncate.go` — tripwires get highest priority in truncation (never dropped)
- `internal/cli/` — add `tripwire` subcommand (list, add, check) and `anchor` command
- Skill prompt templates (`/decide`, `/deliver`, `/diagnose`, `/design`) — add anchor-before-brainstorm step

## Completion Ledger

| # | Item | Status | Evidence |
|---|------|--------|----------|
| AC-1 | Tripwire type spec parsed with triggers/scope/severity | DONE | `TypeTripwire` in `internal/spec/spec.go:25`; `Triggers []string` field at line 136; `typeFromPath` detects `/tripwires/` path (line 818); `IsKnowledge()` includes tripwires (line 1133). |
| AC-2 | Session start includes all active tripwires in prime context (before conventions, not gated) | DONE | `internal/serve/prime.go:76-81` — `## Tripwires (Do Not Violate)` section added for all `status=active` tripwires, not gated by `includeKnowledge`. |
| AC-3 | `hero_context` with file paths includes scope-matched tripwires | DONE | `internal/index/index.go:745` — `FindTripwiresForFiles()` filters by `scope` globs; called at line 1527 in the context builder. |
| AC-4 | `hero_anchor` returns mission + all active tripwires with full Constraint/Why/Instead | DONE | MCP tool registered at `internal/serve/mcp_tools_def.go:162`; `toolAnchor` in `mcp_tools.go:887` returns mission + full tripwire blocks. CLI `hero anchor` in `internal/cli/anchor.go`. |
| AC-5 | `hero_anchor` with `context` highlights trigger-matched tripwires | DONE | `mcp_tools.go:887` — calls `FindTripwiresByTrigger(ctx)`, renders matched tripwires in `### ⚠ Relevant to your current context` section first. |
| AC-6 | `hero_ask`/`hero_search` query matching a trigger prepends tripwire warning | DONE | `tripwireWarning()` at `mcp_tools.go:930`; called in `toolSearch` at line 254 and `toolAsk` at line 845. Matching tripwires prepended to results with `## TRIPWIRE WARNING` block. |
| AC-7 | Superseded tripwires excluded from prime context and retrieval | DONE | All three index functions (`FindAllTripwires`, `FindTripwiresByTrigger`, `FindTripwiresForFiles`) filter by `status = 'active'`. Superseded entries remain in DB but are never returned. |
| AC-8 | `hero tripwire list` and `hero tripwire check <text>` CLI commands | DONE | `internal/cli/tripwire.go` — `tripwireListCmd` and `tripwireCheckCmd` registered under `tripwireCmd`. |
| AC-9 | `hero_anchor` MCP tool registered with "when to call it" description | DONE | `internal/serve/mcp_tools_def.go:162-168` — description reads "Call this BEFORE proposing architectural alternatives, when hitting a dead end, or when brainstorming solutions." |

### Exercise-the-feature check

- [x] Exercised: `go build ./...` clean. All tests in `internal/cli`, `internal/index`, `internal/spec`, `internal/serve` pass. `hero tripwire list` CLI path exercised via `runTripwireList` (tripwire.go). `hero_anchor` MCP dispatch wired in `mcp_dispatch.go:38`. `tripwireWarning` verified in `toolSearch` and `toolAsk`.

## Phasing

### Phase 1: Type + Prime injection
Add the tripwire type, parse it, include it in prime context. This alone addresses the "decay over long sessions" problem for session-start context. Models see tripwires prominently at session start.

### Phase 2: hero_anchor + hero_context integration
Add the re-anchoring tool and file-scoped injection. This gives models an explicit way to re-check constraints at decision moments, and ensures tripwires surface during normal context queries.

### Phase 3: Retrieval + trigger matching
Add trigger-based surfacing in hero_ask/hero_search. This makes tripwires surface even when the model doesn't explicitly re-anchor — any query touching a trigger keyword will pull the guardrail into results.

### Phase 4: Skill integration + CLI
Modify skill prompts to auto-anchor before brainstorming. Add CLI commands for managing tripwires. This is the final behavioral layer that makes re-anchoring automatic rather than opt-in.
