---
title: "hero_spec and hero_handoff MCP tools"
slug: hero-spec-handoff-tools
type: feature
status: completed
priority: high
domain: engineering
size: small
created: 2026-10-06
parent: hero-read-contract
depends-on: [next-step-engine]
relations:
  - target: hero-work-tool
    kind: conflicts-with
completed_at: 2026-10-06T23:50:03Z
---

# hero_spec and hero_handoff

## Goal

Serve one work spec and the handoff briefing over MCP as read contract v1 JSON, so clients render Hero's meaning without parsing `.hero` files.

## Kickoff

Two read-only MCP tools:
- **`hero_spec {slug}`** → `{item, body, relations, acs}`. The item comes from `workmodel` with `next`. The body is Markdown without frontmatter. Relations are refs with `missing` for absent targets. ACs come from the graph's latest results (`acceptance.ListBySpec`), else `unknown`.
- **`hero_handoff {}`** → `{markdown, updated_at}`, using the same file `hero next` shows, via a shared `nextdoc.HandoffPath`.

Test with `go test ./internal/serve -run 'ToolSpec|ToolHandoff|ReadContract'`.

## Design

- `internal/serve/mcp_tools_read_contract.go`:
  - `toolSpec` re-indexes when stale, discovers specs, and builds the item via `workmodel.BuildOne` plus `NextFor`.
  - Relations: parent from the parent edge; children from declared children plus specs pointing at it; `depends_on`, `blocks`, and `related` (related / relates-to / supersedes / conflicts-with). Empty lists are `[]`.
  - Non-work specs and unknown slugs return clear errors.
- `toolHandoff` reads `nextdoc.HandoffPath(heroDir, cfg)`. That rule (team mode: `next/<user>.md`, else `NEXT.md`) moved from `internal/cli/next.go` into `internal/nextdoc/path.go`, and the CLI now calls it too.
- Both tools are registered `safetyRead`, so they advertise `readOnlyHint: true`.

## Acceptance Criteria

- **AC-1:** WHEN `hero_spec` is called for a work spec THE SYSTEM SHALL return `{item, body, relations, acs}`, where `item` is the shared `WorkItem` with `next`, and `body` is the Markdown without frontmatter.
- **AC-2:** THE SYSTEM SHALL report relations as `{parent, children, depends_on, blocks, related}` of `{slug,title,type,status}` refs (status `missing` for absent targets, `[]` for empty lists), and ACs as `{id,text,state}` with pass/fail from recorded results, else unknown.
- **AC-3:** IF the slug is unknown or not a work spec THEN THE SYSTEM SHALL return an error naming the reason.
- **AC-4:** WHEN `hero_handoff` is called THE SYSTEM SHALL return the briefing `hero next` shows as `{markdown, updated_at}`, with `updated_at` from its frontmatter even when a managed block precedes it, and an empty `markdown` and null `updated_at` when there is none.
- **AC-5:** THE SYSTEM SHALL register both tools read-only (`readOnlyHint: true`), and calling them SHALL write nothing under the contract's watch globs.

## Changes

1. `internal/serve/mcp_tools_read_contract.go` (new), `mcp_dispatch.go`, `mcp_tools_def.go`.
2. `internal/nextdoc/path.go` (new); `internal/cli/next.go` delegates to it.
3. Tests: `internal/serve/mcp_tools_read_contract_test.go`; tool count and list in `mcp_test.go`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: hero_spec shape | DONE | `TestToolSpecReturnsContractShape`: decoded title, lane `designed`, next disabled with "waits on base", and the body without frontmatter. Real `hero mcp` call on this repo (`real-exercise.log`) |
| 2 | AC-2: relations and ACs | DONE | Same test: parent, depends_on status, a missing related target, `[]` for empty lists, initiative children, 2 ACs `unknown` |
| 3 | AC-3: clear errors | DONE | `TestToolSpecRejectsNonWorkAndUnknown` |
| 4 | AC-4: hero_handoff | DONE | `TestToolHandoff`: no briefing, frontmatter at the top, and a managed block before the frontmatter (a bug found in the real exercise and fixed). Real call returns this repo's NEXT.md with `updated_at` |
| 5 | AC-5: read-only, no writes | DONE | `TestReadContractToolsAreReadOnlyAndWriteNothing`: tree snapshot before and after, plus `readOnlyHint` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | tools + registration | DONE | `safetyRead` classification |
| 2 | nextdoc.HandoffPath | DONE | `hero next` behavior unchanged (CLI tests pass) |
| 3 | tests | DONE | Tool count 65→67 plus the expected-name list |

### Exercise-the-feature check

- [x] Ran a real `hero mcp` over stdio JSON-RPC on this repo: `hero_spec hero-work-tool` and `hero_handoff` (`real-exercise.log`).

### Excellence Bar self-check

- [x] Yes. Thin tools over the shared model, with one handoff-path rule now shared with the CLI.
