# Delivery audit — tracker-issue-activity-timestamps

**Audited:** `git diff -- internal/tracker/tracker.go internal/tracker/jira.go internal/tracker/github.go internal/tracker/gitlab.go internal/tracker/linear.go internal/tracker/tracker_test.go internal/tracker/gitlab_test.go internal/tracker/linear_search_test.go internal/cli/sync_import.go internal/cli/sync_import_test.go internal/cli/sync_push_diff.go internal/spec/spec.go internal/spec/spec_test.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Expose provider-native creation and update timestamps on `tracker.Issue` — `internal/tracker/tracker.go:36`.
- [✓] Request and project Jira timestamps on list, search, and get — `internal/tracker/jira.go:630`, `internal/tracker/jira.go:781`, `internal/tracker/jira.go:948`, `internal/tracker/jira.go:966`; `TestJira_GetIssue`, `TestJira_ListSearchActivityTimestampParity`.
- [✓] Project GitHub, GitLab, and Linear timestamps on single and list/search paths — `internal/tracker/github.go:236`, `internal/tracker/github.go:453`, `internal/tracker/github.go:533`, `internal/tracker/gitlab.go:108`, `internal/tracker/gitlab.go:134`, `internal/tracker/linear.go:280`, `internal/tracker/linear.go:405`, `internal/tracker/linear.go:515`; focused adapter tests assert native values on each shared path.
- [✓] Persist provider-derived creation date, canonical update time, and exact evidence on initial import — `internal/cli/sync_import.go:1250`, `internal/cli/sync_import.go:1311`; `TestGenerateImportedSpec_TrackerActivityTimestamps`.
- [✓] Update canonical and provider evidence through shared refresh — `internal/cli/sync_import.go:748`; `TestRefreshImportedSpecs_UpdatesPreservesAndIsIdempotent`.
- [✓] Omit missing or malformed values without clock or mtime fallback — `internal/cli/sync_import.go:1255`, `internal/cli/sync_import.go:1291`, `internal/cli/sync_import.go:1330`; `TestGenerateImportedSpec_InvalidOrMissingTimestampsOmitFields`, `TestCurrentSpecFieldValue_MtimeIsNotTrackerCreatedEvidence`.
- [✓] Preserve valid persisted timestamps when refresh data is invalid — desired fields contain only validated values at `internal/cli/sync_import.go:1255`, and refresh only walks present desired keys at `internal/cli/sync_import.go:756`; the third pass in `TestRefreshImportedSpecs_UpdatesPreservesAndIsIdempotent` asserts byte stability.
- [✓] Normalize canonical update time to UTC RFC3339Nano with source precision — `internal/cli/sync_import.go:1258`; import and provider-evidence tests cover offset normalization plus six- and nine-digit fractional precision.
- [✓] Retain exact provider evidence and refuse timestamp fields as pushable content — `internal/cli/sync_import.go:1261`, `internal/cli/sync_push_diff.go:45`; `TestSpecFieldsFromIssue_ProviderTimestampEvidence`, `TestTrackerTimestampFieldsAreNonPushable`.
- [✓] Preserve existing tracker behavior — timestamp changes are additive at existing projection/mapping points; provided focused and full `go test ./...` evidence passes, including existing pagination, custom-field, mapping, status, and ownership suites.
- [✓] Leave repeated valid refresh byte-stable — the second pass in `TestRefreshImportedSpecs_UpdatesPreservesAndIsIdempotent` asserts zero updates and identical content.
- [✓] Keep existing specs without activity metadata backward-compatible and unknown — `internal/spec/spec.go:541`, `internal/spec/spec.go:690`; `TestParseWithoutTrackerActivityTimestampRemainsCompatible`.

## Changes

- [✓] Extend provider-neutral issue contract — `internal/tracker/tracker.go:36` adds documented native fields without an interface change.
- [✓] Make Jira activity metadata consistent — `internal/tracker/jira.go:630`, `internal/tracker/jira.go:789`, `internal/tracker/jira.go:948`, and `internal/tracker/jira.go:966` use the existing shared parsing/search paths.
- [✓] Populate GitHub, GitLab, and Linear timestamps — all three adapters decode and project both values through their single and shared list/search paths.
- [✓] Persist timestamps through shared import mapping — `internal/cli/sync_import.go:1250` centralizes desired fields for initial generation and refresh; `internal/cli/sync_import.go:1318` removes the clock fallback.
- [✓] Preserve parser and ownership semantics — `internal/spec/spec.go:203`, `internal/spec/spec.go:541`, `internal/spec/spec.go:690`, `internal/cli/sync_import.go:837`, and `internal/cli/sync_push_diff.go:45` provide readable, comparable, non-pushable canonical and provider fields, including GitLab.
- [✓] Add focused regression coverage — modified adapter, import/refresh, parser, and field-classification tests assert provider parity, normalization, omission, preservation, precision, ownership, compatibility, and idempotency.

## Audit notes

- No ledger rows were downgraded. All 12 acceptance criteria and all 6 change items have corresponding code and assertion evidence in the scoped diff.
- Provided validation evidence reports the focused packages, uncached focused exercise, and full `go test ./...` suite passing; `git diff --check` also passed.
