---
title: Restore Hero Feed CLI — Activity Feed Reader Regression
slug: restore-hero-feed-cli
type: bug
status: completed
priority: P1
severity: medium
tags: [feed, cli, regression, events]
created: 2026-05-04
relations:
  - target: team-activity-feed
    kind: regression-of
mission_alignment: |
  The team activity feed is a direct support for sessions starting aware of
  recent work. Removing the CLI reader left agents with write-side events and
  MCP access, but no simple terminal surface for "what happened recently?"
  Restoring `hero feed` brings the live coordination surface back without
  changing recap's separate git/spec digest role.
principles_check: |
  Serves #1 by making the documented command work again and #3 by restoring
  a fast startup check for recent cross-session activity. Keeps the fix
  surgical: restore the reader that was deleted before its claimed recap
  replacement existed.
horizon: now
---

## Symptom

`hero feed --since 1h` fails with `unknown command "feed"` even though
AGENTS.md, README.md, the completed `team-activity-feed` spec, and MCP tool
definitions still describe a feed surface.

## Root Cause

Commit `ac2fe63` deleted `internal/cli/feed.go` and said feed was folded into
`hero recap`. Current recap code does not read `.hero/events.log` or use
`internal/feed`, so the CLI reader disappeared while the backend and MCP tool
survived.

## Acceptance Criteria

**AC-1:** `hero feed` is registered as a top-level command again and reads
`.hero/events.log` through the existing `internal/feed` package.
✅ **passing** — verified by `go test ./internal/cli ./internal/feed
./internal/install` and `go run ./cmd/hero feed --limit 1`.

**AC-2:** `hero feed --type decision_made --slug <slug>` filters events by
type and slug.
✅ **passing** — verified by `TestFeedCmdFiltersByTypeAndSlug` in
`go test ./internal/cli`.

**AC-3:** `hero feed --format json` emits machine-readable JSON for matching
events.
✅ **passing** — verified by `TestFeedCmdJSON` in `go test ./internal/cli`.

**AC-4:** `hero recap` remains unchanged as the git/spec digest surface.
✅ **passing** — implementation only restores `internal/cli/feed.go`, registers
`feedCmd`, and adds feed tests; recap code was not changed.

## Kickoff

Restore the deleted `hero feed` CLI reader using the existing `internal/feed`
backend. Register it at the root command, reset its flags in CLI tests, and
add focused tests for default output, filters, and JSON output. Do not merge
feed into recap in this fix.
