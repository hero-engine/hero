---
title: "Hero read contract — hero_work, hero_spec, hero_handoff for every Hero client"
slug: hero-read-contract
type: initiative
status: completed
priority: high
autonomy: autonomous
domain: engineering
size: large
created: 2026-10-06
tags: [mcp, api, read-contract, next-step, peer-request]
child:
  - read-contract-v1
  - work-item-model
  - next-step-engine
  - hero-spec-handoff-tools
  - hero-work-tool
  - polish-and-suggested
  - read-contract-conformance
completed_at: 2026-10-07T00:01:05Z
---

# Hero read contract

## Goal

Give every Hero client a stable, read-only MCP API for *the meaning of the
work*: what each spec is, which lane it sits in, and what to do next. Clients
show Hero's judgement instead of re-deriving it from files. The first consumer
is hero-harness MVP-3; the API is Hero's, for all clients.

## Kickoff

Initiative from peer hero-harness's spec-out request (Mail `mail_2a0ea87cdef292b91b0035db`; contract draft in `../hero-harness/.hero/planning/initiatives/hero-harness-mvp/mvp-3a-hero-service/spec.md`, "The read contract"). There are seven children in four waves. Start with `read-contract-v1`, which settles Hero-side semantics and confirms them with the peer. Then build the shared work-item model and the next-step engine before any tool.

→ `/design read-contract-v1`

## Background

hero-harness asked for three new read-only MCP tools:
- **`hero_work`:** every work spec with lane, progress, verify state and a next step, plus `polish` and `suggested`.
- **`hero_spec`:** one item with body, relations and AC states.
- **`hero_handoff`:** the `hero next` briefing.

It also asked Hero to keep rules learned the hard way:
- one primary action, never Diagnose and Deliver at once;
- `Delivered` only after a passed `hero spec verify`;
- a completed item with a weak or missing verify offers Verify, never Deliver.

The two bugs in the same request are already fixed:
- `spec-title-keeps-yaml-quotes`
- `mcp-pidfiles-leak-into-workspace`

What Hero already has:
- `internal/acceptance` (AC states);
- `internal/projection` (ready work, blocked edges);
- `internal/drive` (next-child judging);
- `hero_read_spec`, `hero_queue`, `hero_kickoff`;
- the `hero next` projection.

There is no per-spec next step, lane, or handoff over MCP, and no rating data.

## Hero-side decisions to settle (child `read-contract-v1`)

The requester drafted shapes; Hero owns the semantics. Open decisions:
1. **Lanes.** Exact definitions of `designed` vs `ready` (designed means a scored spec with Changes and ACs? ready means designed plus no unmet `depends-on`?), `in_progress` (delivering or in-review), `recently_done` (completed within `recent_days`), and `none`.
2. **Verify state.** The source of truth for `passed | partial | failed | not_run`: the last `hero spec verify` result, plus `--skip-tests` or `--force` meaning partial? The audit verdict comes from `delivery-audit.md`.
3. **Next-step table.** Type × status × verify → one primary action plus extras, with the requester's three rules. Also `enabled` and the `reason` ("waits on X").
4. **Revisions.** Per-item: the hash of the spec file plus its relation targets' statuses. Global: the hash of all item revisions plus polish and suggested. Is incremental computation needed for large corpora?
5. **`watch_globs`.** Hero publishes the set (planning, specs, knowledge, NEXT.md, next/, hero.json), never its own runtime files.
6. **Polish kinds.**
   - `weak_verify` and `bug_against_recent` are derivable.
   - `open_followups` needs a definition: Kickoff "follow-up" lines? `deferred-work-suggestions` Focus items?
   - `rated_worse` needs data Hero doesn't have. Drop it from v1, or define a rating source.
7. **Suggested and "backlog is thin".** The threshold, the sources (`queue`, `snapshot`, `note`, `pulse`), and the trailing Explore item.
8. **Versioning.** `schema_version: 1`, the additive-change policy, and how clients detect `tool_missing`.

## Children

| Wave | Child | Type | Priority | Depends on | Why this position |
|---|---|---|---|---|---|
| 1 | `read-contract-v1` | decision | critical | — | Every other child encodes its answers; the peer builds against it |
| 1 | `work-item-model` | feature | critical | read-contract-v1 | Shared `WorkItem` (lane, revision, verify, progress, relations) used by all three tools |
| 2 | `next-step-engine` | feature | high | work-item-model | Deterministic `NextStep` per item; the core of the request and the hardest rules |
| 3 | `hero-spec-handoff-tools` | feature | high | next-step-engine | Small surface, unblocks the peer's spec viewer early |
| 3 | `hero-work-tool` | feature | high | next-step-engine | The main read; items, revision, watch_globs |
| 4 | `polish-and-suggested` | feature | medium | hero-work-tool | Adds two lists on top of a working `hero_work` |
| 4 | `read-contract-conformance` | feature | low | hero-spec-handoff-tools, hero-work-tool, polish-and-suggested | Golden schema tests, fixture export for the peer, docs |

**In-flight overlap watch.** `hero-spec-handoff-tools` and `hero-work-tool` both
register tools in `internal/serve/mcp_tools_def.go` and `mcp_tools.go`, and both
render `WorkItem` JSON. Deliver them one at a time; they carry a reciprocal
`conflicts-with`.

## Cross-cutting concerns

- **Read-only and side-effect-free.** `readOnlyHint: true`. A read may re-index (`ensureFreshIndex`) but must not write anything under `watch_globs`, or clients loop.
- **One model.** All three tools render the same `WorkItem` from `work-item-model`. No second interpretation of lanes or next steps, and `hero list` / `hero_queue` should converge on it over time.
- **Harness-agnostic.** Commands are slash commands every target understands. The tripwire `harness-changes-cover-all-targets` applies to any wording in the guidance.
- **Performance.** `hero_work` on this repo's ~400 specs must stay well under the peer's 10 s timeout. Measure it.

## Risks

| Risk | Mitigation |
|---|---|
| Contract churn after the peer builds on it | `read-contract-v1` is confirmed over Mail before code; additive-only changes after v1 |
| Next-step rules disagree with `/drive`'s judge | `next-step-engine` reuses or aligns with `internal/drive` and is tested against it |
| Slow reads on large corpora | Revision short-circuit; measure; incremental only if needed |
| Undefined polish kinds (`rated_worse`) | Decided in `read-contract-v1`; v1 may omit them |

## Recommended delivery order

`read-contract-v1` → `work-item-model` → `next-step-engine` →
(`hero-spec-handoff-tools`, then `hero-work-tool`) → `polish-and-suggested` →
`read-contract-conformance`. Reply to the peer after `read-contract-v1` is
decided and again when each tool lands, so they can bump their Hero pin.
