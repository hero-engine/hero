---
title: "Spec titles keep their YAML quotes in hero_list and hero list"
slug: spec-title-keeps-yaml-quotes
type: bug
status: completed
priority: P2
severity: low
root_cause_class: code
domain: engineering
size: trivial
created: 2026-10-06
tags: [spec, parser, mcp, peer-request]
delivery_method: manual
completed_at: 2026-10-06T23:08:18Z
---

# Spec titles keep their YAML quotes

## Goal

Return spec titles without their YAML quotes in every Hero surface.

## Kickoff

Reported by peer hero-harness (Mail `mail_2a0ea87cdef292b91b0035db`): `hero_list` returns `"Title"` with the quotes. The frontmatter parser (`internal/spec/spec.go`, `case "title"`) stores the raw scalar. Decode quoted titles as YAML scalars. Verify with `go test ./internal/spec -run TestParseTitleUnquotesYAMLScalars`.

## Problem

`title: "Ten graph writers omit Repo, …"` was listed by `hero list` and `hero_list` as `"Ten graph writers omit Repo, …"`, quotes included. Many specs quote their titles, which YAML requires when a title contains `: `. Clients then show the quotes or have to strip them.

## Root Cause

`spec.Parse` uses a hand-written `key: value` frontmatter reader and assigns `s.Title = val` verbatim. Quoted YAML scalars are never decoded.

## Fix

`unquoteYAMLScalar` decodes a value that starts with `"` or `'` through yaml.v3, which handles escapes and doubled `''`. Anything unquoted or undecodable passes through unchanged. Only `title` is decoded; no writer re-serializes a parsed title.

## Acceptance Criteria

- **AC-1:** WHEN a spec title is a double- or single-quoted YAML scalar THE SYSTEM SHALL expose the decoded value (no surrounding quotes; escapes and doubled single quotes resolved) to `hero list`, `hero_list`, and every `spec.Spec` consumer.
- **AC-2:** IF a title is unquoted or not a valid quoted scalar THEN THE SYSTEM SHALL keep it exactly as written.

## Changes

1. `internal/spec/spec.go`: `unquoteYAMLScalar` applied to `title`.
2. `internal/spec/title_unquote_test.go`: table test.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: quoted titles decoded for every consumer | DONE | `spec.go` `unquoteYAMLScalar` in `spec.Parse`, which every surface uses. `TestParseTitleUnquotesYAMLScalars` covers double quotes with `: `, single with `''`, escaped inner quotes, and `''` inside double quotes; 4 of its cases fail on the old parser. Real `hero list` output in `real-exercise.log` |
| 2 | AC-2: unquoted or undecodable titles unchanged | DONE | Same test: plain title with inner quotes, and an unterminated `"` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | spec.go unquote | DONE | Title only; no writer re-serializes a parsed title (checked all `title:` writers) |
| 2 | table test | DONE | `internal/spec/title_unquote_test.go` |

### Exercise-the-feature check

- [x] `./hero list --type bug` shows quoted-title specs without quotes (`real-exercise.log`). `hero_list` shares the same `spec.Discover` path.

### Excellence Bar self-check

- [x] Yes. A minimal fix at the parser, with YAML-correct decoding rather than quote stripping.

## Final validation

`go test ./... -count=1` and `go vet ./...` pass (`go-tests.log`, `vet.log`).
