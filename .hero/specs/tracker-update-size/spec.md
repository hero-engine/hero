---
title: "Tracker UpdateSize — close the size push loop"
slug: tracker-update-size
type: feature
domain: engineering
status: completed
size: small
relates-to: [spec-size-and-promotion-nudge]
tags: [tracker, sync, size]
completed_at: 2026-06-01T15:43:51Z
---

# Tracker UpdateSize — close the size push loop

## Context

The just-completed `spec-size-and-promotion-nudge` spec (now archived at
`.hero/specs/spec-size-and-promotion-nudge/spec.md`) added a living `size:`
frontmatter field and wired it into tracker `CreateIssue` payloads across
Jira / Linear / GitHub. Slice 5 also landed `PlanSizePush` in
`internal/tracker/size_mapping.go`, which classifies a pending push as
`SizeSyncPushToTracker` (clean push), `SizeSyncConflict` (human-set value
on the tracker side that maps to a different tier), or `SizeSyncNoop`
(both sides agree, or no mapping configured). The planner is fully tested
in `internal/tracker/size_mapping_test.go`.

The gap: no adapter actually *writes* the size on update. `runSync` in
`internal/cli/sync.go` lines 115–123 inspects the planner's output but
only surfaces it as a stderr warning ("manual push required — sync spec
does not write size fields"). This spec closes that loop by adding an
`UpdateSize` method to the `Tracker` interface, implementing it on each
real adapter, and consuming the planner's `SizeSyncPushToTracker` action
inside `runSync` to call it.

Parent spec reference: see slice-5 Approach and AC#16 in
`.hero/specs/spec-size-and-promotion-nudge/spec.md`.

## Goal

`hero sync` (and any future code path that holds a `Tracker` and a spec)
can push a local `size:` value to the tracker for issues where the
tracker side is empty, by calling `tracker.UpdateSize(issueID, localTier)`.
The non-destructive contract from slice 5 is preserved: conflicts still
warn and stop, agreements still no-op. Each of the three real adapters
writes the size in the shape that adapter already uses on create (Jira
numeric custom field, Linear `estimate`, GitHub `size/<tier>` label).
A `Tracker` that does not implement `UpdateSize` returns a sentinel
error so callers can degrade gracefully.

## Approach

**Additive, non-breaking interface extension.** Add `UpdateSize(issueID,
localTier string) error` to the `Tracker` interface in
`internal/tracker/tracker.go`. The three real adapters (`jira.go`,
`linear.go`, `github.go`) implement it. Mirror the field-resolution
logic already used in each adapter's `CreateIssue`: read
`configuredSizeMapping` for the field/prefix, call `MapSize(localTier)`
for the value, then issue the appropriate API call.

**New sentinel error.** Add `ErrSizeUpdateNotSupported` to the tracker
package (alongside any existing sentinels) so callers can distinguish
"adapter doesn't write sizes" from real network/auth errors. Real
adapters never return this; it exists for future stub trackers and for
tests that want to assert behavior on degraded adapters.

**Consume the planner, don't reimplement it.** `PlanSizePush` already
classifies the situation. `runSync` switches on its `Action`:
`SizeSyncPushToTracker` calls `UpdateSize`; `SizeSyncConflict` keeps the
existing warn-and-stop; `SizeSyncNoop` stays silent. The non-destructive
guarantee lives in the planner, not in this code path.

**GitHub label rotation is the fiddly piece.** Unlike Jira (single
custom field) or Linear (single `estimate` field), GitHub stores size
as a label and the issue carries other labels we must not clobber —
especially `hero:*` labels. The implementation must (a) fetch current
labels (a `GET /repos/{owner}/{repo}/issues/{number}` is acceptable;
prefer the simpler one extra round-trip over caller-passed labels),
(b) strip any label whose name starts with the configured size prefix
(`size/` by default), (c) add the mapped `size/<tier>` label, and
(d) PATCH the issue with the merged label set. All non-size labels —
including but not limited to `hero:*` — are preserved verbatim.

**Linear soft failure.** Linear's `estimate` field is only present when
a team has estimation enabled. If the GraphQL mutation returns a
"field not enabled" / "estimation disabled" error, log it as a warning
and return nil rather than failing the whole sync. Real network/auth
errors still propagate.

**Test harness reuse.** The httptest harness in
`internal/tracker/create_issue_size_test.go` already covers per-adapter
payload-shape assertions. New tests live in
`internal/tracker/update_size_test.go` (separate file for clarity, but
same harness style). One sync-level test in `internal/cli` exercises
the full `runSync` → `UpdateSize` path with a stubbed tracker.

## Acceptance Criteria

- THE SYSTEM SHALL expose `UpdateSize(issueID, localTier string) error` on
  the `Tracker` interface in `internal/tracker/tracker.go`.
- THE SYSTEM SHALL define `ErrSizeUpdateNotSupported` as an exported
  sentinel error in the `tracker` package.
- WHEN `UpdateSize` is called on a tracker without a real implementation
  THE SYSTEM SHALL return `ErrSizeUpdateNotSupported`.
- WHEN the Jira adapter's `UpdateSize` is invoked with a mapped tier THE
  SYSTEM SHALL issue `PUT /rest/api/2/issue/{key}` with a body of the
  form `{"fields": {"<configured_field>": <numeric_points>}}` where the
  field name comes from `configuredSizeMapping.field` and the value
  comes from `MapSize(localTier)`.
- WHEN the Linear adapter's `UpdateSize` is invoked with a mapped tier
  THE SYSTEM SHALL issue an `issueUpdate(id, input: { estimate })`
  GraphQL mutation with the mapped numeric value.
- IF the Linear GraphQL mutation rejects the update because the team
  has estimation disabled THEN THE SYSTEM SHALL log a single-line
  warning to stderr and return nil.
- WHEN the GitHub adapter's `UpdateSize` is invoked THE SYSTEM SHALL
  read the issue's current labels, strip any label whose name starts
  with the configured size prefix, add the mapped `size/<tier>` label,
  and PATCH the issue with the merged set.
- THE SYSTEM SHALL preserve all non-size labels on the GitHub issue
  during `UpdateSize`, including any `hero:*` labels and unrelated
  user labels.
- IF the tracker API returns a non-success status for `UpdateSize`
  (other than the Linear estimation-disabled case) THEN THE SYSTEM
  SHALL return the error to the caller without swallowing it.
- WHEN `runSync` processes a spec whose `PlanSizePush` result is
  `SizeSyncPushToTracker` THE SYSTEM SHALL call
  `t.UpdateSize(s.TrackerID, s.Size)` and surface success on stdout.
- WHEN `runSync` processes a spec whose `PlanSizePush` result is
  `SizeSyncConflict` THE SYSTEM SHALL retain the existing warn-only
  behavior and SHALL NOT call `UpdateSize`.
- WHEN `runSync` processes a spec whose `PlanSizePush` result is
  `SizeSyncNoop` THE SYSTEM SHALL NOT call `UpdateSize`.

## Changes

1. **Extend the `Tracker` interface** — `internal/tracker/tracker.go`.
   - Add `UpdateSize(issueID, localTier string) error` to the interface
     declaration alongside `UpdateStatus`.
   - Add the doc comment explaining the contract: writes the mapped
     value to the tracker's size field, returns
     `ErrSizeUpdateNotSupported` for adapters that don't implement it,
     propagates real errors otherwise.

2. **Add the sentinel error** — `internal/tracker/tracker.go` (or a new
   `errors.go` in the same package if cleaner).
   - `var ErrSizeUpdateNotSupported = errors.New("tracker: UpdateSize not supported by this adapter")`.

3. **Jira adapter** — `internal/tracker/jira.go`.
   - Add `func (j *jiraTracker) UpdateSize(issueID, localTier string) error`.
   - Resolve the field name from `j.configuredSizeMapping.Field` (mirror
     the lookup logic already used in `CreateIssue`).
   - Call `j.MapSize(localTier)` for the numeric value; propagate the
     error if the tier is unknown or no mapping is configured.
   - Issue `PUT {baseURL}/rest/api/2/issue/{issueID}` with body
     `{"fields": {<field>: <points>}}`. Parse the numeric value the same
     way create does — do not double-encode as a string.
   - Standard auth + non-2xx → error handling matches the patterns in
     existing `jira.go` methods.

4. **Linear adapter** — `internal/tracker/linear.go`.
   - Add `func (l *linearTracker) UpdateSize(issueID, localTier string) error`.
   - GraphQL mutation:
     `mutation IssueUpdate($id: String!, $estimate: Float!) {
        issueUpdate(id: $id, input: { estimate: $estimate }) {
          success
        }
      }`
     with `$estimate` populated from `l.MapSize(localTier)` parsed to a
     number.
   - On a GraphQL error whose message indicates estimation is disabled
     (substring match on "estimate" + "not enabled" / "disabled" — check
     Linear's actual error shape during implementation), log
     `"Warning: Linear team has estimation disabled; skipping size update for <issueID>"` to stderr and return nil.

5. **GitHub adapter** — `internal/tracker/github.go`.
   - Add `func (g *githubTracker) UpdateSize(issueID, localTier string) error`.
   - `GET /repos/{owner}/{repo}/issues/{number}` to fetch current labels.
   - Determine the size-label prefix from
     `g.configuredSizeMapping.LabelPrefix` (mirror the create path's
     lookup, default `"size/"`).
   - Build a new label slice: keep every existing label whose name does
     not start with the prefix; append the mapped `<prefix><tier>` from
     `g.MapSize(localTier)`.
   - `PATCH /repos/{owner}/{repo}/issues/{number}` with body
     `{"labels": [<merged>]}`.

6. **Wire `runSync` to call `UpdateSize`** — `internal/cli/sync.go`,
   lines 115–123.
   - Replace the warn-only `SizeSyncPushToTracker` branch with a call
     to `t.UpdateSize(s.TrackerID, s.Size)`. On success, print
     `"Updated %s size for issue %s → %s\n"`. On error, return the
     wrapped error (do not swallow).
   - Keep `SizeSyncConflict` warn-only; do not call `UpdateSize`.
   - Default branch (`SizeSyncNoop`) stays silent.

7. **Per-adapter tests** — `internal/tracker/update_size_test.go` (new
   file, adjacent to `create_issue_size_test.go`).
   - `TestJiraUpdateSize_CleanPush_EmitsExpectedPayload` — httptest
     server asserts PUT path + body `{"fields": {<field>: <points>}}`.
   - `TestJiraUpdateSize_PropagatesError` — server returns 500, error
     bubbles up.
   - `TestLinearUpdateSize_CleanPush_EmitsExpectedMutation` — asserts
     GraphQL mutation body shape and `estimate` variable.
   - `TestLinearUpdateSize_EstimationDisabled_LogsAndReturnsNil` —
     server returns the estimation-disabled error; assert nil and
     stderr contains the warning.
   - `TestGitHubUpdateSize_StripsOldSizeLabelKeepsOthers` — issue starts
     with `["bug", "hero:active", "size/small"]`, PATCH body must equal
     `["bug", "hero:active", "size/large"]` (order is acceptable
     unordered; assert as a set).
   - `TestGitHubUpdateSize_PreservesNonSizeLabelsWhenNoOldSizeLabel` —
     issue starts with `["bug", "hero:active"]`, PATCH body must equal
     `["bug", "hero:active", "size/large"]`.

8. **One sync-level test** — extend `internal/cli/sync_test.go` (or
   create it if the file doesn't exist; mirror an existing CLI test
   file's harness style).
   - `TestRunSync_CleanPush_CallsUpdateSize` — given a spec with
     `size: large` and a stub tracker whose `GetIssue` returns an
     issue with no size set, `runSync` invokes `UpdateSize` with the
     spec's tier. Use a fake tracker that records calls.

### Files actually touched (delivered)

- `internal/tracker/tracker.go` — added `UpdateSize` to the `Tracker`
  interface and the `ErrSizeUpdateNotSupported` sentinel.
- `internal/tracker/jira.go` — `(*jira).UpdateSize`, PUT against
  `/rest/api/3/issue/{key}` (v3, matching the rest of the file).
- `internal/tracker/linear.go` — `(*linear).UpdateSize` +
  `isLinearEstimationDisabled` substring matcher for the soft-fail.
- `internal/tracker/github.go` — `(*gitHub).UpdateSize` (GET + PATCH
  label rotation). Also widened `GetIssue` to populate `issue.Labels`
  so `PlanSizePush` can detect existing `size/<tier>` labels and
  return `SizeSyncConflict` instead of silently classifying as
  push-to-tracker.
- `internal/cli/sync.go` — replaced the warn-only
  `SizeSyncPushToTracker` branch with a real `UpdateSize` call,
  graceful skip on `ErrSizeUpdateNotSupported`.
- `internal/tracker/update_size_test.go` — new file: 9 tests across
  Jira / Linear / GitHub (clean push, error propagation, estimation
  disabled, label rotation, non-size label preservation).
- `internal/cli/sync_test.go` — added
  `TestRunSync_CleanPush_CallsUpdateSize` and
  `TestRunSync_Conflict_DoesNotCallUpdateSize` (the non-destructive
  contract regression test).

## Boundaries

- **No new tracker types.** Only Jira / Linear / GitHub gain a real
  implementation. Any future tracker adapter is responsible for its
  own `UpdateSize` (or returning `ErrSizeUpdateNotSupported`).
- **No bulk size-push CLI.** "Push all drifted sizes at once" is a
  separate workflow concern. This spec only wires the existing
  per-spec `hero sync` path.
- **No container-tier mapping push.** The parent spec's
  `container_field` config exists but isn't pushed today on create
  either. Whole container direction is a future spec.
- **No reverse-direction writes.** Pulling size from tracker to local
  is already covered by `PlanSizePull` / `SizeSyncSeedLocal`; that
  path is not changed here.
- **No mid-delivery auto-bump.** Drift detection still surfaces; the
  human decides whether to bump declared `size:`.

## Risks

- **Jira custom-field IDs vary per workspace.** The config-driven
  `size_mapping.field` is the single source of truth; do not
  hard-code a field name. If the field name in config is wrong, Jira
  returns a 400 — propagate the error verbatim so the user sees it.
- **GitHub label rotation requires a read-then-write.** That's one
  extra API call per push. Acceptable; the alternative (PATCH with
  only the new size label) would clobber unrelated labels and is
  unsafe. Document the round-trip in the function comment.
- **Linear estimation-disabled detection is brittle.** The substring
  match on the error message is the only signal Linear gives. During
  implementation, confirm the exact error shape; if it's structured
  (an error code) prefer that over substring matching. The test
  should pin down whatever shape is chosen.
- **Non-destructive contract integrity.** The conflict branch must
  continue to warn and stop. Add an explicit assertion in the
  sync-level test that `UpdateSize` is NOT called on conflict.

## Validation

- `go test ./internal/tracker/... ./internal/cli/...` passes.
- New tests fail when the relevant adapter logic is reverted (each
  test must actually exercise the new code path).
- Manual smoke: against a Jira sandbox, run `hero sync push` on a
  spec with `size: large` and an issue whose size field is empty.
  Confirm the field is set; confirm a second run is a no-op.
- The conflict path is unchanged: a spec at `size: large` against an
  issue at `size: medium` (mapped) still warns and does not write.

## Kickoff

You're picking up `tracker-update-size`. Parent spec is archived at
`.hero/specs/spec-size-and-promotion-nudge/spec.md`; read its slice-5
Approach for context. The planner (`PlanSizePush` in
`internal/tracker/size_mapping.go`) is done and tested — your job is to
consume it. Add `UpdateSize` to the `Tracker` interface, implement it on
Jira / Linear / GitHub (mirror each adapter's existing `CreateIssue`
size-write shape), and replace the warn-only branch in
`internal/cli/sync.go` (lines 115–123) with a real call. Reuse the
httptest harness from `internal/tracker/create_issue_size_test.go` for
per-adapter tests. The tricky bit is GitHub label rotation — preserve
non-size labels including `hero:*`. Linear estimation-disabled is a
soft warning, not a hard error.
