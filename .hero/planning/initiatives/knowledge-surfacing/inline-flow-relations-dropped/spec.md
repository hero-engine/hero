---
title: "Inline-flow YAML relations are silently dropped by the frontmatter parser"
slug: inline-flow-relations-dropped
type: bug
status: completed
priority: P1
severity: high
size: small
domain: engineering
created: 2026-07-07
completed_at: 2026-07-07T00:00:00Z
delivery_method: drive
tags: [parser, frontmatter, relations, silent-data-loss, drive, verify]
relations:
  - target: knowledge-surfacing
    kind: parent
  - target: flat-named-spec-discovery
    kind: related
---

# Inline-flow YAML relations are silently dropped by the frontmatter parser

## Goal

Make the hand-rolled frontmatter parser accept **inline-flow** relation entries
(`- { target: x, kind: parent }`) as well as the block style it already
supports, so declared relations are never silently discarded.

## Kickoff

Hero's frontmatter parser (`internal/spec/spec.go`, `parseRelationsBlock` /
`applyRelField`) only parses block-style relations. An inline-flow entry —
`- { target: content-remediation, kind: parent }` — is `strings.Cut` at the
first `:`, yielding key `"{ target"`, which matches neither `target` nor `kind`,
so the relation parses to **nothing** and is dropped with no error. 4 specs in
this repo are affected; their parent/related edges are invisible. Fix:
`parseRelationsBlock` should detect a `{…}` payload after `- ` and parse it as an
inline map (strip braces, split on `,`, apply each `k: v`). Start:
`internal/spec/spec.go:710` (`parseRelationsBlock`), `:861` (`applyRelField`).

## Problem

`applyRelField` (internal/spec/spec.go:861) does `k, v, _ := strings.Cut(line, ":")`.
For an inline-flow list entry the line (after trimming `- `) is
`{ target: content-remediation, kind: parent }`, so `k = "{ target"` and the
`switch k` falls through — `Target` and `Kind` stay empty, and
`parseRelationsBlock` skips the entry because `current.Target == ""`.

Inline flow is valid YAML and is used across the repo (the `/design` templates
emit it), so this is silent, ongoing data loss on a load-bearing field.

**Demonstrated impact** (live, against this repo):
- 4 specs declare relations in inline flow: `sales-pack-reality-sync`,
  `pm-pack-phantom-surfaces`, `content-remediation/token-efficiency-pass`,
  `content-remediation/core-commands-domain-neutral`.
- All four declare `- { target: content-remediation, kind: parent }`. Because
  the edge is dropped, `hero goal content-remediation --check` reports
  `completed children: None` — the initiative cannot see 4 of its shipped
  children and can never reach `done`. Drive/verify completion tracking is
  broken for any initiative whose children use inline flow.
- This is the same failure that made the `knowledge-surfacing` children
  invisible until their relations were rewritten to block style.

## Fix

In `parseRelationsBlock`, when the text after `- ` begins with `{`, parse it as
an inline map instead of a single `k: v`:
- Strip a leading `{` and trailing `}`.
- Split on `,` into `key: value` pairs (relation values are simple slugs/enums;
  no nested commas or quotes in practice — keep the split simple, trim spaces).
- Apply each pair via `applyRelField`.

Block style stays exactly as-is. Keep the parser tolerant: a malformed inline
entry contributes what it can rather than erroring.

## Acceptance Criteria

- WHEN a spec declares `relations:` with an inline-flow entry
  `- { target: <slug>, kind: <kind> }`, THE SYSTEM SHALL parse a relation with
  that target and kind — verified by a parser unit test asserting both fields.
- WHEN a spec mixes inline-flow and block-style relation entries, THE SYSTEM
  SHALL parse all of them.
- WHEN `hero goal content-remediation --check` runs after re-indexing, THE
  SYSTEM SHALL count the 4 inline-flow children as resolved (they appear in
  `completed`/`remaining`, not silently absent).
- THE SYSTEM SHALL preserve existing block-style parsing behavior unchanged (no
  regression in the spec-parser test suite).

## Validation

- Unit test in `internal/spec` for inline-flow, block-style, and mixed
  relations blocks.
- Regression: full `go test ./...` green.
- Live: after fix + `hero index`, `hero goal content-remediation --check` shows
  the previously-invisible children resolved.

## Out of scope

- Migrating existing specs from inline flow to block style — unnecessary once the
  parser accepts both. (Optional cleanup, not required.)
- Adopting a full YAML library for frontmatter — a targeted parser fix is
  sufficient and keeps the dependency-light frontmatter path.
