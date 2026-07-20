# Delivery audit — jira-import-per-type-filter-and-body-parity

**Audited:** `7e3fc57` with the supplied exact `git diff --cached --` file set
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Per-type filters paginate independently — `internal/cli/sync_import.go:544` sends the configured limit to every type query and unions all deduplicated results without a global early exit; `TestFetchByTypeUnion_AppliesLimitPerType` asserts both configured searches receive `limit=2`.
- [✓] Jira continues beyond 500 — `internal/tracker/jira.go:1079` raises the per-query safety ceiling to 5,000 and `internal/tracker/jira.go:1200` paginates in bounded pages; `TestJira_SearchAllowsMoreThanLegacyFiveHundredCap` returns 600 issues across six pages.
- [✓] Explicit truncation is visible — Jira warns when a page token remains at the limit (`internal/tracker/jira.go:1267`), GitHub warns for either a next link or excess items in the current page (`internal/tracker/github.go:569`), Linear warns on `hasNextPage` (`internal/tracker/linear.go:575`), and GitLab warns for either condition (`internal/tracker/gitlab.go:535`). `TestGitHubAndLinear_ReportIncompleteLimitedResults`, `TestJira_Search_PaginationRespectsLimit`, and `TestGitLab_ListIssues_ReportsIncompleteLimit` assert the provider warning paths, including GitHub final-page truncation.
- [✓] Full bulk description reaches Problem/Goal — Jira list/search fields include `description` (`internal/tracker/jira.go:1095`, `internal/tracker/jira.go:1114`), provider projections retain full bodies, and `internal/cli/sync_import.go:1398` writes the complete trimmed description into the canonical section; `TestGenerateImportedSpec_PreservesFullDescriptionInProblem` asserts the full Problem body.
- [✓] Untouched placeholder repairs without detail fetch — `internal/cli/sync_import.go:773` replaces only the exact imported scaffold marker and updates the baseline; `TestRefreshImportedSpecs_RepairsUntouchedImportedProblem` asserts the repaired body, baseline tracker ID/body, and zero `GetIssue` calls.
- [✓] Authored local body survives refresh — replacement is gated on the exact untouched marker at `internal/cli/sync_import.go:773`; `TestRefreshImportedSpecs_PreservesAuthoredProblem` asserts authored text remains and remote text is not inserted.
- [✓] Missing bulk records never fan out — `internal/cli/sync_import.go:743` skips records absent from the bulk lookup without calling `GetIssue`; `TestRefreshImportedSpecs_DoesNotFetchMissingIssuesIndividually` supplies no bulk results and asserts zero detail calls.
- [✓] GitHub and Linear paginate — GitHub follows `Link` pagination at `internal/tracker/github.go:545` and Linear follows cursors at `internal/tracker/linear.go:547`; the named multi-page tests assert two requests and complete bodies.
- [✓] GitLab, GitHub, and Linear retain descriptions beyond 500 characters — provider projections assign full descriptions at `internal/tracker/gitlab.go:144`, `internal/tracker/github.go:487`, `internal/tracker/github.go:575`, and `internal/tracker/linear.go:652`; long-body tests cover all three providers.
- [✓] Jira bulk writes require an explicit transition cohort — `validateJiraBulkPush` runs before tracker initialization and rejects pushes without both `--status` and a configured transition (`internal/cli/sync.go:175`, `internal/cli/sync.go:232`); `TestValidateJiraBulkPush_RequiresBoundedTransitionCohort` covers dry-run, both rejection cases, and the allowed case.

## Changes
- [✓] Jira pagination, fields, response lifecycle, and warnings — the staged adapter raises the ceiling, includes reporter/description, closes every page response before continuing, and reports incomplete capped results.
- [✓] Per-type limit and bulk-only refresh — the staged coordinator removes global union termination and the refresh `GetIssue` fallback.
- [✓] Canonical full bodies with safe placeholder repair — generation writes complete Problem/Goal bodies; refresh changes only untouched scaffold markers and advances the sync baseline, with direct regression assertions.
- [✓] GitHub/Linear/GitLab pagination, lossless bodies, and visible truncation — all three staged adapters retain complete descriptions, follow provider pagination, and warn when an explicit limit omits known results.
- [✓] Fail-closed Jira bulk push — the staged guard requires one explicit Hero status cohort backed by a configured Jira transition before any write loop begins.

## Audit notes
- None.
