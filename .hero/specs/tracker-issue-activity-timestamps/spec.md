---
title: "Tracker Issue Activity Timestamps and Jira Broad-Import Field Parity"
slug: tracker-issue-activity-timestamps
type: feature
status: completed
domain: engineering
size: medium
priority: high
tags: [tracker, import, timestamps, jira, hero-code]
created: 2026-07-20
delivery_method: manual
completed_at: 2026-07-20T18:10:23Z
---

# Tracker Issue Activity Timestamps and Jira Broad-Import Field Parity

## Context

Hero's provider-neutral `tracker.Issue` carries `CreatedAt` but not an update timestamp. Jira `Search` requests `created`, while the broad `ListIssues` path omits it, and no Jira read path requests or parses the native `updated` field. GitHub, GitLab, and Linear expose native creation/update timestamps but do not project update time consistently into `Issue`.

This creates two observable failures. A broad `hero sync import` can stamp the local import date into `created:` instead of the real Jira creation date, and downstream consumers such as Hero Code have no provider-owned activity timestamp to distinguish tracker activity from a spec file's local modification time. The existing Jira pagination and custom-field discovery are correct and must remain unchanged.

## Goal

Make tracker creation and update metadata a consistent provider-owned contract across Jira, GitHub, GitLab, and Linear. Every successful provider read should populate the available native timestamps, initial import should persist real tracker evidence without inventing fallback activity, and refresh should update the same fields through the existing shared mapping path. Hero Code should receive one stable canonical field for tracker activity without depending on provider-specific keys or local file metadata.

## Kickoff

Adds native tracker update time across all four adapters and makes broad Jira imports preserve real creation/activity timestamps.

**Status:** completed — implementation, full-suite validation, cold audit, and Hero verification all passed.

**Pick up at:** use the compatibility contract when wiring consumers: read only `tracker_updated_at`, and treat missing or invalid values as unknown.

→ `.hero/specs/tracker-issue-activity-timestamps/spec.md`

**Files:** `internal/tracker/tracker.go:36`, `internal/tracker/jira.go:627`, `internal/cli/sync_import.go:1252`, `internal/spec/spec.go:209`, `internal/cli/sync_push_diff.go:45`
**Skip:** do not alter Jira pagination, custom-field discovery, Severity mapping, or Hero Code UI behavior.

## Problem

The data loss occurs at two boundaries. Provider adapters do not expose a common update timestamp, and Jira's broad list query does not request even the existing creation timestamp. The shared import mapper therefore has no trustworthy activity value to persist and currently fills a missing creation date with `time.Now()`. Refresh inherits the same gap because it reuses broad-list results when available. Local spec mtime then becomes the only apparent freshness signal even though it records Hero edits, not tracker activity.

## Compatibility Contract

The frontmatter contract is additive:

| Field | Ownership | Format | Meaning |
|---|---|---|---|
| `created` | tracker-owned on imported specs | `YYYY-MM-DD` | Existing Hero creation date, derived from a valid provider `CreatedAt`; preserve the provider timestamp's calendar date. |
| `tracker_updated_at` | tracker-owned | UTC RFC3339 with provider precision retained (`RFC3339Nano`) | Canonical instant of the tracker's latest native issue update. This is the field Hero Code should consume. |
| `<provider>_updated_at` | tracker-owned evidence | Exact validated provider string | Provider-specific evidence, e.g. `jira_updated_at`, `github_updated_at`, `gitlab_updated_at`, or `linear_updated_at`. Not the cross-provider consumer contract. |

`tracker_updated_at` normalizes offsets to UTC while preserving the same instant and available fractional-second precision. For example, Jira `2026-07-20T10:15:30.123-0600` becomes `2026-07-20T16:15:30.123Z`; `jira_updated_at` retains the original validated string.

Missing, empty, or malformed provider timestamps mean **unknown**. Initial import omits the corresponding fields, including `created`; it must not stamp import time. Refresh does not erase a previously valid value when the provider omits or malforms the field, and it must not substitute refresh time or file mtime. A later valid refresh writes or corrects the fields. Existing specs without the new keys remain valid.

Hero Code should parse only `tracker_updated_at` as tracker activity. If absent or invalid, activity is unknown; it must not fall back to `created`, the spec file's mtime, import time, or refresh time. Provider-prefixed keys are diagnostic evidence only. Initial import and refresh use the same normalization and ownership rules.

## Approach

Keep `tracker.Issue` as the narrow adapter-to-sync boundary: add `UpdatedAt` beside `CreatedAt`, leaving both as provider-native strings. Adapters should project the exact native strings and should not normalize them independently. This keeps provider decoding simple and makes one persistence boundary responsible for validation and normalization.

Extend the existing `specFieldsFromIssue` mapping so initial import and `refreshImportedSpecs` share timestamp behavior. A small timestamp parser should accept the native formats already returned by the four adapters (RFC3339/RFC3339Nano, Jira's offset-without-colon form, and date-only values for `CreatedAt`). Only valid values produce desired frontmatter fields. Canonical update time is formatted once as UTC `RFC3339Nano`; provider-prefixed evidence retains the validated native string.

Keep Jira's three issue read surfaces on the same activity-field baseline by adding `created,updated` to the field selections used by `ListIssues`, `Search`, and `GetIssue`, then parsing both centrally in `parseIssueRaw`. Do not change `searchIssues`, `nextPageToken`, page sizing, or custom-field discovery.

Treat all timestamp fields as tracker-owned organizational state. They may be updated by import/refresh but never pushed to a tracker as user-authored content. Extend spec/frontmatter parsing enough for stable comparison and GitLab prefix parity so refresh is idempotent and does not rewrite unchanged timestamp fields.

## Changes

1. Extend the provider-neutral issue contract in `internal/tracker/tracker.go`.
   - Add `UpdatedAt string` beside `CreatedAt` and document both as provider-native timestamp values.
   - Do not add a new tracker interface method or provider capability flag.

2. Make Jira activity metadata consistent in `internal/tracker/jira.go`.
   - Request both `created` and `updated` from `GetIssue`, `Search`, and broad `ListIssues`.
   - Populate `Issue.CreatedAt` and `Issue.UpdatedAt` in `parseIssueRaw` so all three paths share parsing.
   - Preserve existing requested assignee, priority, severity/custom fields, reporter/description behavior, limits, JQL construction, pagination, and deduplication.

3. Populate native update timestamps in the other adapters.
   - `internal/tracker/github.go`: decode `created_at` and `updated_at` in `GetIssue`, list endpoint results, and search endpoint results, projecting both into `Issue`.
   - `internal/tracker/gitlab.go`: add `updated_at` to `gitLabIssue` and project it through the existing `toIssue` path used by `GetIssue`, `ListIssues`, and `Search`.
   - `internal/tracker/linear.go`: request and project `createdAt` and `updatedAt` in both `GetIssue` and the shared `Search`/`ListIssues` query.
   - Leave each provider's filtering, ordering, pagination, type inference, priority, labels, and custom fields untouched.

4. Persist timestamps through the shared import mapping in `internal/cli/sync_import.go`.
   - Centralize validation of the supported tracker timestamp formats.
   - For valid `CreatedAt`, keep writing `created: YYYY-MM-DD` from the provider timestamp; remove the initial-import fallback to `time.Now()`.
   - For valid `UpdatedAt`, produce canonical `tracker_updated_at` in UTC `RFC3339Nano` and exact validated `<provider>_updated_at` evidence.
   - Emit those fields from `generateImportedSpec` on initial import and update them from `refreshImportedSpecs` through `specFieldsFromIssue`.
   - Treat missing/malformed timestamps as absent desired values so refresh preserves previously valid evidence rather than clearing or replacing it.

5. Preserve parser and ownership semantics in `internal/spec/spec.go`, `internal/cli/sync_import.go`, and `internal/cli/sync_push_diff.go`.
   - Parse `tracker_updated_at` and the four supported provider-prefixed update keys sufficiently for exact, idempotent refresh comparison; include the existing GitLab provider namespace in the tracker metadata path.
   - Classify canonical and provider-prefixed timestamp keys as tracker-owned organizational state so field-level push refuses them.
   - Keep timestamp writes targeted: do not reorder or rewrite unrelated frontmatter and do not change assignee, priority, severity, custom-field, or status ownership.

6. Add focused regression coverage.
   - Extend `internal/tracker/tracker_test.go`, `internal/tracker/gitlab_test.go`, and `internal/tracker/linear_search_test.go` with native timestamp fixtures and request-field assertions.
   - Prove Jira `ListIssues`, `Search`, and `GetIssue` all request and return the same `CreatedAt`/`UpdatedAt` values without changing pagination or custom-field behavior.
   - Extend `internal/cli/sync_import_test.go` for initial import, shared refresh, exact provider evidence, UTC normalization, fractional precision, malformed/missing omission, no clock/mtime fallback, and idempotent second refresh.
   - Add parser/field-classification tests where needed to prove the new keys remain readable and non-pushable.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL expose provider-native `CreatedAt` and `UpdatedAt` values on the provider-neutral `tracker.Issue`
- **AC-2:** WHEN Jira returns an issue through `ListIssues`, `Search`, or `GetIssue` THE SYSTEM SHALL request and project both native `created` and `updated` fields consistently
- **AC-3:** WHEN GitHub, GitLab, or Linear returns an issue through its list/search or single-issue path THE SYSTEM SHALL project the provider's native creation and update timestamps into `tracker.Issue`
- **AC-4:** WHEN an initially imported issue has valid creation and update timestamps THE SYSTEM SHALL write the provider-derived `created` date, canonical UTC `tracker_updated_at`, and exact `<provider>_updated_at` evidence
- **AC-5:** WHEN a linked tracker issue's valid update timestamp changes THE SYSTEM SHALL update canonical and provider-prefixed timestamp fields through the existing shared refresh path
- **AC-6:** IF a provider creation or update timestamp is missing, empty, or malformed THEN THE SYSTEM SHALL omit that desired value without substituting import time, refresh time, or local file mtime
- **AC-7:** WHILE refreshing a spec whose provider response lacks a valid timestamp THE SYSTEM SHALL preserve any previously valid persisted timestamp rather than clear or replace it
- **AC-8:** THE SYSTEM SHALL normalize `tracker_updated_at` to UTC RFC3339 while preserving the source instant and available fractional-second precision
- **AC-9:** THE SYSTEM SHALL retain `<provider>_updated_at` as the exact validated native timestamp string and classify all canonical/provider timestamp fields as tracker-owned non-pushable state
- **AC-10:** WHILE importing or refreshing timestamp metadata THE SYSTEM SHALL preserve existing assignee, priority, severity/custom-field, pagination, deduplication, status, and field-ownership behavior
- **AC-11:** WHEN the same valid tracker timestamp is refreshed twice THE SYSTEM SHALL leave the spec unchanged on the second refresh
- **AC-12:** IF `tracker_updated_at` is absent on an existing spec THEN THE SYSTEM SHALL treat tracker activity as unknown and remain backward-compatible without backfill or migration

## Boundaries

- Do not change Jira pagination, `nextPageToken` handling, page sizes, limits, or JQL construction.
- Do not rebuild Jira custom-field discovery, Severity support, priority mapping, or field caches.
- Do not add Jira changelog/history retrieval, assignment history, or "time since assignment."
- Do not add Bug Bash UI, stale filters, sorting, or date-display behavior in Hero Code. Hero Code owns those consumers.
- Do not infer tracker activity from comments, status transitions, local spec edits, import/refresh execution, git history, or filesystem metadata.
- Do not add a migration or eagerly backfill existing specs. Their missing canonical field means unknown until a valid tracker refresh supplies it.
- Do not add a provider-neutral `tracker_created_at` field in this change; the established `created` field remains the creation-date contract for imported specs.

## Risks

- Jira timestamps commonly use an RFC3339-like numeric offset without a colon. A parser limited to Go's strict `time.RFC3339` would silently drop valid Jira activity, so fixtures must cover Jira's native representation and fractional seconds.
- Canonical UTC normalization and exact provider evidence intentionally produce different strings for offset timestamps. Tests should compare instants for the canonical field and bytes for the provider-prefixed field.
- The refresh loop currently compares parsed spec fields rather than arbitrary raw keys. If the new keys are not individually readable, one divergent key can cause perpetual rewrites or hide drift.
- Removing the initial `time.Now()` fallback makes `created` absent when a provider gives no valid creation time. This is intentional: `created` is optional, and consumers must not reinterpret the spec parser's mtime fallback as tracker evidence.
- Adding update fields to provider queries can expose weak fixtures that return partial wire shapes. Tests should assert requested fields without broadening the production contract or introducing live-network dependencies.

## Validation

- Run focused adapter tests for Jira, GitHub, GitLab, and Linear with `httptest.Server`/GraphQL fixtures, covering single-issue and list/search paths.
- Assert Jira's `fields` query parameter contains `created` and `updated` for `ListIssues`, `Search`, and `GetIssue`, while existing pagination and custom-field tests continue to pass unchanged.
- Run sync-import tests proving initial import and refresh write the compatibility contract, exact native evidence survives, canonical UTC output preserves precision, malformed/missing values produce no fallback, and a second refresh is byte-stable.
- Run spec parser and field-classification tests for all four provider prefixes and the canonical key.
- Run `go test ./internal/tracker ./internal/spec ./internal/cli`, then `go test ./...`.

## Completion Ledger

Implemented the provider-owned activity timestamp contract in Go. The provider adapters retain exact native strings, while the shared import/refresh boundary validates and normalizes persisted metadata. Loaded `agent-reliability`, `implementation-principles`, `testing-and-validation`, `go-stack`, and `completion-ledger`.

Validation performed:

- `go test ./internal/tracker ./internal/spec ./internal/cli` — PASS.
- `go test ./...` — PASS.
- Uncached focused adapter/import/parser exercise with `go test -count=1` — PASS for Jira, GitHub, GitLab, Linear, initial generation, refresh, omission/preservation, ownership, parser compatibility, and idempotency.
- `git diff --check -- <delivery files>` — PASS.

Risks/follow-ups: none within this contract. Hero Code must consume only `tracker_updated_at`; provider-prefixed evidence remains diagnostic.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Expose provider-native `CreatedAt` and `UpdatedAt` on `tracker.Issue` | DONE | `internal/tracker/tracker.go:36` — both native timestamp strings are part of the shared issue contract. |
| 2 | Jira List/Search/Get request and project both timestamps | DONE | `internal/tracker/jira.go:627`, `internal/tracker/jira.go:789`, `internal/tracker/jira.go:948` — all three paths share `parseIssueRaw`; `TestJira_GetIssue` and `TestJira_ListSearchActivityTimestampParity` assert fields and values. |
| 3 | GitHub, GitLab, and Linear project timestamps on all paths | DONE | `internal/tracker/github.go:234`, `internal/tracker/gitlab.go:106`, `internal/tracker/linear.go:280` — single/list/search wire shapes project both fields; Linear `ListIssues` delegates to the tested shared `Search` path. |
| 4 | Initial import persists real creation date, canonical update, and exact evidence | DONE | `internal/cli/sync_import.go:1252`, `internal/cli/sync_import.go:1327` — `TestGenerateImportedSpec_TrackerActivityTimestamps` exercises Jira offset normalization and exact evidence. |
| 5 | Refresh updates canonical and provider timestamps | DONE | `internal/cli/sync_import.go:751`, `internal/cli/sync_import_test.go:358` — the existing shared refresh mapper updates both fields from a changed native timestamp. |
| 6 | Missing/malformed timestamps omit values without fallback | DONE | `internal/cli/sync_import.go:1291`, `internal/cli/sync_import_test.go:310` — invalid values produce no desired fields; initial generation omits `created` and update keys. |
| 7 | Refresh preserves previously valid values when new data is invalid | DONE | `internal/cli/sync_import_test.go:358` — the malformed/missing third refresh is byte-stable and reports zero updates. |
| 8 | Canonical update is UTC RFC3339 with fractional precision | DONE | `internal/cli/sync_import.go:1258` — UTC `RFC3339Nano`; tests preserve six- and nine-digit source precision and the same instant. |
| 9 | Exact provider evidence and non-pushable ownership | DONE | `internal/cli/sync_import.go:1261`, `internal/cli/sync_push_diff.go:45` — all four evidence keys remain exact and canonical/provider keys classify as org-state. |
| 10 | Preserve existing tracker mapping, pagination, dedup, status, and ownership behavior | DONE | Changes are additive at existing projection points; unchanged focused suites and full `go test ./...` pass, including Jira/GitLab pagination and custom-field coverage. |
| 11 | Repeated refresh is idempotent | DONE | `internal/cli/sync_import_test.go:358` — the second valid refresh produces zero updates and byte-identical content. |
| 12 | Existing specs without update metadata remain compatible/unknown | DONE | `internal/spec/spec_test.go:118` — a legacy imported spec parses with empty canonical/native activity values and requires no migration. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Extend provider-neutral issue contract | DONE | `internal/tracker/tracker.go:36` — added documented provider-native `UpdatedAt` beside `CreatedAt`; no interface/capability change. |
| 2 | Make Jira activity metadata consistent | DONE | `internal/tracker/jira.go:627`, `internal/tracker/jira.go:789`, `internal/tracker/jira.go:948` — added fields without changing JQL, limits, pagination, or discovery. |
| 3 | Populate GitHub, GitLab, and Linear native updates | DONE | `internal/tracker/github.go`, `internal/tracker/gitlab.go`, `internal/tracker/linear.go` — all single and shared list/search decoding paths carry creation/update strings. |
| 4 | Persist timestamps through shared import mapping | DONE | `internal/cli/sync_import.go:837`, `internal/cli/sync_import.go:1252`, `internal/cli/sync_import.go:1327` — centralized parsing, removed clock fallback, and wired initial/refresh persistence. |
| 5 | Preserve parser and ownership semantics | DONE | `internal/spec/spec.go:209`, `internal/spec/spec.go:541`, `internal/spec/spec.go:687`, `internal/cli/sync_push_diff.go:45` — canonical/evidence parsing, GitLab namespace, idempotent comparison, and non-pushability. |
| 6 | Add focused regression coverage | DONE | `internal/tracker/tracker_test.go`, `internal/tracker/gitlab_test.go`, `internal/tracker/linear_search_test.go`, `internal/cli/sync_import_test.go`, `internal/spec/spec_test.go` cover all required paths and edge cases. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: uncached focused tests drove provider HTTP/GraphQL responses through adapters and initial import/refresh persistence, observing real `created`, normalized `tracker_updated_at`, exact provider evidence, preservation, and byte-stable repeat refresh.

### Excellence Bar self-check

- [x] Honest answer to "would a senior engineer who cares about this codebase be proud to ship this?" — yes. The implementation is additive, keeps provider decoding native, centralizes validation once, preserves established pagination/discovery behavior, and has focused plus full-suite evidence.
