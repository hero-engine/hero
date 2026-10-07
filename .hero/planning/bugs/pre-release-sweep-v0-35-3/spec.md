---
title: "Pre-release sweep for v0.35.3 — close every open audit note before tagging"
slug: pre-release-sweep-v0-35-3
type: bug
status: delivering
priority: P1
severity: moderate
root_cause_class: process
domain: engineering
size: small
created: 2026-10-07
tags: [release, why, read-contract, audit-sweep]
---

# Pre-release sweep for v0.35.3

## Goal

Ship v0.35.3 with no known defects. Every note raised by a cold audit since v0.35.0 is either fixed here (with a test that fails on the old code) or listed below as by-design with the reason.

## Kickoff

v0.35.2 shipped while an audit-flagged defect (`hero why --edges` start-node tie) was still open. New rule: before tagging, sweep every audit note from the release line. This spec fixes:
- the `--edges` tie;
- `open_followups` skipping mtime-fallback items;
- the literal `.hero` path in the archive exemption;
- the four audit test gaps.

Verify with `go test ./...`.

## Root Cause

Process. Audit highlights were treated as out-of-scope follow-ups and released around, instead of being given their own fix before tagging.

## Fixes

1. **`hero why --edges`** used an unordered `LIMIT 1`, so a promoted intake could beat its spec. It now resolves through the exported `traversal.ResolveTarget`, the same query `hero why` uses, so the two always agree.
2. **`open_followups`** compared against `completed_at` only. An item that is recently done through the mtime fallback never collected follow-ups. It now uses the same completion time the lane uses (`completionTime`).
3. **The archive exemption** matched the literal `/.hero/specs/`. It is now folder-agnostic. `isArchivedPath` takes as the hero folder the nearest ancestor that is a workspace (it holds `hero.json` or is named `.hero`) and whose child on the path is `planning/` or `specs/`. It is archived when that child is `specs/`. A repo beneath a `planning`/`specs` directory, a spec folder with either name, a `hero.json` further up, or a stray `hero.json` inside a spec folder cannot misplace it. (Two audits caught weaker versions: substring matching, then a farther `hero.json` beating a nearer `.hero`.)
4. **Test gaps:**
   - `hero_spec` CRLF frontmatter;
   - a knowledge note pointing at an initiative is not a child;
   - recorded `pass`/`fail` AC states;
   - initiative `verify: null` at the tool level;
   - `hero_work` no-write check across all six watch globs.
5. **Audit follow-ups:**
   - `Lane` calls `completionTime` instead of inlining a copy.
   - `hero why --edges` returns `ResolveTarget`'s error unchanged, so real DB errors are no longer reported as "no node".
   - `--edges` now has the same global-node fallback as `hero why`. That is intended, since the two must agree.
   - The no-write test seeds every watched glob, catches modification and deletion, and compares content. The sweep test checks every tool result.

## By design (not defects; documented, no change)

- **`hero serve` shutdown:** a request that arrives after shutdown begins is refused. That is inherent to stopping.
- **Audit staleness clock:** it uses committer timestamps (`%ct`), so a rebase that rewrites dates or clock skew can reorder commits. The mtime check still applies to uncommitted work.
- **Same-commit pair:** a spec and audit committed together are accepted as one reviewed unit. This was an explicit decision in `followup-audit-staleness-git`.
- **Signer lookup:** `spec.KnownSigners` reads git history per read-contract call. Measured at 0.2–0.3 s per `hero_work` on 513 specs, within the 2 s budget.
- **Handoff:** the handed-off reason is the generic "handed off to a peer", because specs do not record the alias. `hero_handoff` excludes per-machine local notes. Both are decided in `read-contract-v1`.
- **Aha!** connections are raw-request only. This is a documented limitation since v0.35.0 (`ErrAhaAdapterNotImplemented`), not a defect.

## Acceptance Criteria

- **AC-1:** WHEN a promoted intake and its spec share a slug THE SYSTEM SHALL start `hero why --edges` from the spec, using the same resolution as `hero why`.
- **AC-2:** WHEN a recently done item has no `completed_at` THE SYSTEM SHALL still report its open follow-ups, using the lane's completion time.
- **AC-3:** THE SYSTEM SHALL treat a spec as archived for audit staleness regardless of the hero folder's name.
- **AC-4:** THE SYSTEM SHALL have tests for each audit test gap listed in Fixes item 4.
- **AC-5:** THE SYSTEM SHALL close every first-audit note: one completion-time function, real errors from `--edges`, and no-write and sweep tests that cannot pass vacuously.

## Changes

1. `internal/traversal/why.go` (`ResolveTarget` exported), `why_federation_test.go`; `internal/cli/brief.go` (`runWhyEdges`), `internal/cli/why_edges_test.go`.
2. `internal/workmodel/polish.go` (`completionTime`), `polish_test.go`.
3. `internal/spec/audit.go` (`isArchivedPath`), `audit_git_test.go`.
4. `internal/serve/mcp_tools_read_contract_test.go` (`TestReadContractSweepGaps`; `TestToolWorkWritesNothing` now covers all six globs).
5. `internal/workmodel/model.go` (`Lane` uses `completionTime`); `internal/cli/brief.go` (error pass-through); stronger serve tests.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: --edges uses why's resolution | DONE | `TestWhyEdgesPromotedSpecWinsSlugTie` (the intake is inserted first and ingested later). It fails on the old code with "started from the intake" |
| 2 | AC-2: open_followups mtime fallback | DONE | `TestOpenFollowupsUseLaneCompletionFallback` fails on the old code |
| 3 | AC-3: folder-agnostic archive | DONE | `TestArchivedPathIgnoresFolderName` (real directories with `hero.json`) covers: a custom folder; a repo under a `planning` ancestor and one under a `specs` ancestor; an archived spec folder named `planning`; a planning spec folder named `specs`; a `.hero` with no `hero.json`; a path with no workspace. Cases also cover a `.hero` with no `hero.json` under a directory that holds one, and a stray `hero.json` inside an archived spec folder. The literal check, the substring version and the first anchor each fail cases |
| 4 | AC-4: audit test gaps | DONE | `TestReadContractSweepGaps` covers CRLF, the note-not-child case, pass/fail ACs and initiative verify null. `TestToolWorkWritesNothing` covers all six globs |
| 5 | AC-5: first-audit notes closed | DONE | `Lane` → `completionTime`; `--edges` returns `ResolveTarget`'s error. The no-write test seeds `NEXT.md`/`hero.json`/`next/`/`specs/` and checks deletion and content. The sweep test checks `IsError` and unmarshal, and asserts `crlf` is a child |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | why --edges | DONE | One shared resolver |
| 2 | polish | DONE | — |
| 3 | audit archive path | DONE | — |
| 4 | serve tests | DONE | — |
| 5 | audit follow-ups | DONE | — |

### Exercise-the-feature check

- [x] Each fix has a test driving the real component (graph store, `runWhyEdges`, the workmodel corpus, real MCP tool calls). The `--edges` and follow-up fixes are falsified against the old code.

### Excellence Bar self-check

- [x] Yes. Every audit note since v0.35.0 is accounted for: fixed, or by-design with the reason recorded.
