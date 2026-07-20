---
title: "hero serve auto-refresh bypasses the bulk importer"
slug: serve-auto-refresh-bypasses-bulk-import
type: bug
status: completed
domain: engineering
severity: high
root_cause_class: design
priority: high
tags: [tracker, import, serve, performance]
created: 2026-07-20
delivery_method: manual
completed_at: 2026-07-20T23:52:59Z
---

# hero serve auto-refresh bypasses the bulk importer

## Issue

When `import.auto_refresh` is enabled, `hero serve` claims to periodically
re-import and synchronize tracker work. Instead, its private refresher discovers
every linked local work spec and calls `Tracker.GetIssue` once per spec. On a
project with thousands of Jira issues, every interval can therefore generate
thousands of API requests while still failing to discover new tracker issues.

The explicit `hero sync import --refresh` path already has the correct contract:
it uses configured bulk searches (including independent `import.by_type` queries),
creates newly discovered specs, and refreshes linked specs only from the issues
returned by those bulk calls. Deep comments, attachments, changelog, and full
field loading remain an explicit single-ticket `hero sync evidence` operation.

## Investigation

### Categorization

| Attribute | Assessment |
|---|---|
| **Criticality** | high — the request count scales linearly with the linked inventory and can overwhelm Jira or hit rate limits |
| **Ease of Fix** | moderate — the private refresher can delegate to the canonical importer without duplicating tracker logic |
| **Caused by our codebase?** | Yes — Core owns both divergent paths |
| **Needs more research?** | No — the call graph and mismatch are directly observable in source |

### Analysis

`Server.Run` starts `StartImportRefresher` for each configured project. The
refresher loads the tracker, discovers local specs, and loops through each open
linked work spec. The loop invokes `GetIssue` before applying status and
assignment changes. It neither uses `Search`/`ListIssues` nor invokes the import
logic that creates missing specs.

By contrast, `runSyncImport` resolves the configured bulk filter plan, including
per-type queries and limits, and its `--refresh` reconciliation builds an
in-memory issue map from those same bulk results. That code explicitly forbids a
fallback to `GetIssue` for records outside the bulk result.

### Root cause

Core has two implementations behind the phrase "import refresh." The CLI path is
the canonical bulk importer; the server path is an older status-polling loop
implemented independently. The duplicate server implementation drifted from the
import contract and made request count proportional to local spec count.

### Severity

High. The option is off by default, but enabling it on a large tracker creates an
unbounded periodic N+1 request pattern. The workaround is to disable
`auto_refresh` and invoke `hero sync import --refresh` explicitly.

### Code flow

1. `internal/serve/server.go` starts one refresher per project during server startup.
2. `internal/serve/refresh.go` runs immediately and then at the configured interval.
3. The old refresh implementation discovers local specs and invokes
   `Tracker.GetIssue` inside the spec loop.
4. `internal/cli/sync_import.go` separately implements the correct configured
   bulk query, discovery, deduplication, and bulk-only refresh behavior.

### Secondary defects

- The server auto-refresh does not discover new issues despite being documented
  as automatic tracker refresh.
- The private implementation initializes trackers differently from the CLI and
  can therefore drift on Jira-specific configuration and connection selection.

## Goal

Make `hero serve` auto-refresh execute the canonical bulk import-and-refresh
workflow. One refresh cycle must discover new issues and update eligible linked
specs using only the configured bulk calls. It must never make one API call per
ticket. Deep single-ticket evidence retrieval remains isolated to an explicit
model/user request.

## Changes

1. Replace the private tracker/spec loop in `internal/serve/refresh.go` with a
   cancellable invocation of the running Hero executable's canonical
   `sync import --refresh --no-report` command in the target project directory.
   - Reuse the exact installed executable via `os.Executable`, not a potentially
     stale `hero` from `PATH`.
   - Keep refresh cycles serialized and stop the child import when the server
     context is cancelled.
   - Publish the existing index event only after a successful bulk refresh.
   - Update the constructor call in `internal/serve/server.go` after removing the
     obsolete server-only spec-directory dependency.
2. Rewrite `internal/serve/refresh_test.go` around an injected bulk-import runner.
   - Assert one canonical bulk command per refresh cycle.
   - Assert the command arguments, project working directory, error handling,
     and event behavior.
   - Enforce structurally that the server refresher has no tracker dependency or
     `GetIssue` path.
3. Update `web/docs/src/configuration/hero-json.md` and
   `web/docs/src/configuration/tracker-setup.md` to state that `auto_refresh`
   runs the same bulk import workflow as `hero sync import --refresh`, honors
   configured filters/per-type limits, and does not load deep ticket evidence.

## Boundaries

- Do not add `GetIssue`, evidence, comments, attachments, or changelog calls to
  bulk import or background refresh.
- Do not add a second query planner or duplicate import implementation in the
  server package.
- Do not change the explicit `hero sync evidence` deep-ticket workflow.
- Do not change Hero Code lifecycle timing or consent behavior.
- Terminal issues excluded by user JQL remain excluded; this work does not invent
  an additional reconciliation query outside the configured import corpus.

## Risks

- A subprocess adds process startup cost per interval, but avoids package cycles,
  PATH drift, mutable CLI global state, and a second importer. The minimum
  five-minute interval keeps that cost bounded.
- Import output must be captured so a large import does not spam the daemon log;
  failures still need enough output to diagnose configuration/authentication
  problems.
- Cancellation must terminate an in-flight child process during server shutdown.

## Validation

- Focused `internal/serve` tests cover command construction, one-call delegation,
  success events, failures, and cancellation-aware execution.
- A source-level regression assertion confirms `internal/serve/refresh.go` does
  not import the tracker package or reference `GetIssue`.
- Existing `internal/cli` import tests confirm configured bulk/per-type behavior
  and the no-`GetIssue` refresh contract.
- Run `go test ./internal/serve ./internal/cli ./internal/config` and the full
  `go test ./...` suite.

## Acceptance Criteria

- WHEN `hero serve` auto-refresh runs THE SYSTEM SHALL invoke the canonical
  `sync import --refresh --no-report` workflow exactly once for that cycle.
- WHEN configured tracker filters include multiple `import.by_type` entries THE
  SYSTEM SHALL rely on the canonical importer so each type is fetched through its
  independent bulk query and new matching issues can be imported.
- THE SYSTEM SHALL NOT call `Tracker.GetIssue` or any per-ticket API from the
  server auto-refresh or bulk import path.
- WHEN the server stops during an import THE SYSTEM SHALL cancel the in-flight
  bulk import process and stop future refreshes.
- WHEN a bulk refresh succeeds THE SYSTEM SHALL publish an index-rebuilt event;
  when it fails THE SYSTEM SHALL report the failure without publishing success.
- THE SYSTEM SHALL document that full ticket evidence, including comments and
  attachments, is loaded only by the explicit deep-evidence operation and never
  by background sync.

## Kickoff

Fixes Core's hidden periodic N+1 tracker polling by routing `hero serve`
auto-refresh through the canonical bulk importer.

**Status:** delivering — implementation and validation complete; closing gates in progress.

**Pick up at:** cold-audit the completed diff and run
`hero spec verify serve-auto-refresh-bypasses-bulk-import`.

→ `.hero/planning/bugs/serve-auto-refresh-bypasses-bulk-import/spec.md`

**Files:** `internal/serve/refresh.go`, `internal/serve/refresh_test.go`,
`web/docs/src/configuration/hero-json.md`,
`web/docs/src/configuration/tracker-setup.md`

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Auto-refresh invokes canonical bulk workflow once per cycle | DONE | `internal/serve/refresh.go:75` serializes cycles; `internal/serve/refresh.go:131` constructs exactly `sync import --refresh --no-report`; asserted by `TestNewBulkImportCommandUsesCanonicalBulkRefresh` and `TestImportRefresherRefreshRunsOneBulkImportAndPublishes`. |
| 2 | Per-type configured imports use independent bulk queries and discover new work | DONE | `internal/serve/refresh.go:115` delegates to the canonical importer instead of copying its planner; existing `internal/cli/sync_import_bytype_test.go` covers independent query union/dedup. |
| 3 | No `GetIssue` or per-ticket API in background/bulk import | DONE | The old tracker/spec loop was removed. `TestServerAutoRefreshHasNoPerTicketTrackerPath` rejects tracker imports, `GetIssue`, and local-spec iteration; existing CLI refresh tests assert zero `GetIssue` calls. |
| 4 | Server stop cancels in-flight import and future cycles | DONE | `internal/serve/refresh.go:54` shares one cancelable context with `exec.CommandContext`; `TestImportRefresherRunStopsAndCancelsInFlightImport` verifies cancellation and loop exit. |
| 5 | Success publishes; failure reports without success event | DONE | `internal/serve/refresh.go:95` publishes only after a nil command error; success/failure tests verify both branches and bounded failure output. |
| 6 | Docs isolate full-ticket evidence from background sync | DONE | `web/docs/src/configuration/tracker-setup.md:150` and `web/docs/src/configuration/hero-json.md:198` document the bulk-only contract and explicit `hero sync evidence <spec-slug>` path. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Replace private server loop with canonical cancellable bulk import | DONE | `internal/serve/refresh.go` now self-invokes the exact running binary; `internal/serve/server.go:373` uses the simplified constructor. |
| 2 | Rewrite refresher tests around injected bulk runner | DONE | `internal/serve/refresh_test.go` covers command, single delegation, events, failure, cancellation, log bounds, and the no-per-ticket source guard. |
| 3 | Clarify auto-refresh and evidence documentation | DONE | Both tracker configuration guides describe filters, per-type bulk queries, limits, discovery, and deep-evidence separation. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised through the focused refresher tests (`go test ./internal/serve ./internal/cli ./internal/config`), the race-enabled server test (`go test -race ./internal/serve`), and the complete repository suite (`go test ./...`); all passed.

### Excellence Bar self-check

- [x] Yes — the duplicate N+1 implementation is deleted, the canonical importer remains the single source of truth, cancellation and failure behavior are explicit, and regression tests enforce the bulk-only boundary.
