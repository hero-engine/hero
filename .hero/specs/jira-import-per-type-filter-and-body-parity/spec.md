---
title: "Tracker bulk import truncates work, drops bodies, and risks request fan-out"
slug: jira-import-per-type-filter-and-body-parity
type: bug
status: completed
domain: engineering
root_cause_class: code
severity: high
priority: high
size: large
created: 2026-07-20
tags: [tracker, import, pagination, scale, jira, github, linear, gitlab]
relations:
  - target: jira-import-classification-obscures-work-items
    kind: supersedes
  - target: tracker-diagnosis-full-ticket-evidence
    kind: related
delivery_method: manual
completed_at: 2026-07-20T22:39:46Z
---

# Tracker bulk import truncates work, drops bodies, and risks request fan-out

## Goal

Make tracker imports complete, lossless, and bounded: each configured work type gets its own pagination budget, bulk pages carry the fields needed to build useful specs, every provider reports truncation, and no refresh path fans out to one request per issue.

## Kickoff

Hardens bulk tracker import after Morpheus exposed a 500-item Jira clamp, missing descriptions, and a tempting per-ticket refresh fallback.

**Status:** delivering — implementation and live Morpheus dry-run are complete; closing audit and verify remain.

**Pick up at:** cold-audit the staged provider/import diff and run `hero spec verify` after the audit ships.

→ `.hero/planning/bugs/jira-import-per-type-filter-and-body-parity/spec.md`

**Files:** `internal/tracker/jira.go`, `internal/tracker/github.go`, `internal/tracker/linear.go`, `internal/tracker/gitlab.go`, `internal/cli/sync_import.go`
**Skip:** calling `GetIssue` for every imported ticket; deep evidence belongs to the on-demand evidence spec.

## Problem

Morpheus Jira shows more than 1,000 active Bugs when Work Type is Bug and Done, Rejected, and Cancelled are excluded, while Hero exposed only a partial corpus. Jira requests were silently clamped to 500, one broad query mixed all types, and later types could be starved by a global union limit. The broad Jira list also omitted descriptions, so imported specs contained empty placeholders even when Jira held detailed reproduction evidence.

A wider audit found the same failure class elsewhere: GitHub and Linear silently clamped requests to 100, GitLab stopped at a requested limit without telling the user, and GitHub/Linear/GitLab discarded body content after 500 characters. Refresh also contained a per-spec `GetIssue` fallback, which would turn a bulk sync into N network calls. Jira pagination deferred response-body closes inside its loop. Finally, `hero sync jira --push` could walk every linked spec and fall back to posting a status comment on each ticket.

## Root Cause

Tracker adapters treated API page size as a total-result limit, while the import coordinator treated one limit as a global union budget. Field selection and description truncation optimized for lightweight previews even though those same records generated canonical local specs. Bulk refresh did not enforce a bulk-only boundary, and bulk Jira push had no explicit cohort/transition guard.

## Changes

1. Update Jira import pagination and field selection to support at least 1,000 records per configured type, include full descriptions in bulk pages, close each page response promptly, and warn whenever an explicit cap leaves pages unread.
2. Apply the configured limit independently to every `import.by_type` query, union/deduplicate results, and keep refresh strictly bulk-only with zero per-ticket `GetIssue` fallbacks.
3. Preserve complete descriptions in generated Problem/Goal sections and repair only untouched import placeholders during refresh; never overwrite authored local content.
4. Add real pagination, lossless bodies, and visible truncation warnings to GitHub, Linear, and GitLab adapters instead of silently stopping at one page or 500 characters.
5. Fail closed for bulk Jira writes: `--push` must name one explicit Hero status cohort with a configured Jira transition, preventing unbounded comment-only storms.

## Acceptance Criteria

- AC-1: WHEN multiple work types define import filters, THE SYSTEM SHALL execute and paginate every type independently with the configured per-type limit.
- AC-2: WHEN a Jira query matches more than 500 issues, THE SYSTEM SHALL continue pagination beyond 500 up to the explicit safety limit.
- AC-3: IF any provider stops because an explicit result limit was reached while more results exist, THEN THE SYSTEM SHALL emit a visible incomplete-results warning.
- AC-4: WHEN a bulk tracker record has a description, THE SYSTEM SHALL preserve the complete normalized body in the imported Problem or Goal section.
- AC-5: WHEN refresh sees an untouched imported placeholder and its bulk record has a body, THE SYSTEM SHALL repair the body and baseline without a per-ticket fetch.
- AC-6: IF local Problem or Goal content was authored, THEN bulk refresh SHALL preserve it.
- AC-7: WHEN a linked local spec is absent from the bulk query result, THE SYSTEM SHALL skip it without calling `GetIssue`.
- AC-8: WHEN GitHub or Linear returns more than one API page, THE SYSTEM SHALL follow pagination until the requested limit or exhaustion.
- AC-9: WHEN GitLab, GitHub, or Linear returns a description longer than 500 characters, THE SYSTEM SHALL retain the full value.
- AC-10: IF a user requests bulk Jira `--push`, THEN THE SYSTEM SHALL require an explicit status cohort and configured transition rather than applying comment fallbacks across all specs.

## Validation

- Exercise Jira pagination above the former 500 cap and per-type union behavior.
- Assert bulk refresh invokes `GetIssue` zero times for both present and absent issues.
- Exercise multi-page GitHub and Linear fixtures plus GitLab pagination/limit fixtures.
- Assert long descriptions survive every provider projection and imported spec generation.
- Assert unsafe Jira bulk push shapes are rejected before tracker initialization.
- Run `go test ./internal/tracker ./internal/cli ./internal/install` and the repository-wide Go suite.

## Boundaries

- Do not use per-ticket full issue calls to fill bulk sync fields.
- Do not make project-specific status names global; project JQL owns terminal-state exclusions.
- Do not broaden Hero Code's Bugs view to disguise an incomplete producer corpus.
- Full comments, attachments, screenshots, changelog, and custom-field evidence are fetched only by the separate on-demand evidence helper.

## Investigation History

### Round 1 — Initial diagnosis
- **Date:** 2026-07-20T20:30:00Z
- **Agent:** debug-investigator
- **Root cause:** Hard-coded type classification appeared to explain the local count split.
- **Confidence:** Medium
- **Key evidence:** The partial local corpus split into bug, feature, and initiative specs.

### Round 2 — Challenged (reject)
- **Date:** 2026-07-20T21:15:00Z
- **Challenged by:** engineer
- **Feedback:** "there are over 1000 bugs on jira" and bulk sync must not call `GetIssue` for every ticket.
- **Revised root cause:** Silent provider caps, a global type-union budget, lossy bulk fields, and missing bulk-only boundaries produced an incomplete and misleading corpus.
- **What changed:** The apparent local count match was coincidental; Jira's filtered Bug inventory disproved it and the scale audit found the same class in other adapters.
- **Confidence:** High

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Per-type filters paginate independently | DONE | `fetchByTypeUnion` applies the configured limit to every query; `TestFetchByTypeUnion_AppliesLimitPerType` proves later types are not starved. |
| 2 | Jira continues beyond 500 | DONE | Jira's per-query ceiling is 5,000; `TestJira_SearchAllowsMoreThanLegacyFiveHundredCap` fetches 600. Live Morpheus dry-run fetched 4,836 unioned records. |
| 3 | Explicit truncation is visible | DONE | Jira, GitHub, Linear, and GitLab emit incomplete-result warnings when more pages remain at the limit; provider pagination tests exercise the limit paths. |
| 4 | Full bulk description reaches Problem/Goal | DONE | Jira list fields include description and all provider projections retain full bodies; `TestGenerateImportedSpec_PreservesFullDescriptionInProblem` asserts the canonical section. |
| 5 | Untouched placeholder repairs without detail fetch | DONE | `TestRefreshImportedSpecs_RepairsUntouchedImportedProblem` verifies body/baseline repair and zero `GetIssue` calls. |
| 6 | Authored local body survives refresh | DONE | `TestRefreshImportedSpecs_PreservesAuthoredProblem` preserves investigated content. |
| 7 | Missing bulk records never fan out | DONE | `TestRefreshImportedSpecs_DoesNotFetchMissingIssuesIndividually` asserts zero per-ticket calls. |
| 8 | GitHub and Linear paginate | DONE | `TestGitHub_ListIssues_PaginatesBeyondOneHundredAndPreservesBody` and `TestLinear_Search_PaginatesBeyondOneHundredAndPreservesDescription` exercise multiple pages. |
| 9 | Non-Jira providers preserve bodies over 500 chars | DONE | GitHub/Linear pagination tests and `TestGitLab_SearchProjectsActivityTimestamps` assert complete long descriptions. |
| 10 | Jira bulk writes require explicit transition cohort | DONE | `validateJiraBulkPush` fails closed; `TestValidateJiraBulkPush_RequiresBoundedTransitionCohort` covers unbounded and comment-only cases. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Jira pagination, fields, response lifecycle, warnings | DONE | `internal/tracker/jira.go` raises the per-type ceiling, includes description, closes pages promptly, and warns on truncation. |
| 2 | Per-type limit and bulk-only refresh | DONE | `internal/cli/sync_import.go` removes both early union termination and every refresh `GetIssue` fallback. |
| 3 | Canonical full bodies with safe placeholder repair | DONE | Generation writes full Problem/Goal bodies; refresh only replaces untouched scaffold markers and advances the baseline. |
| 4 | GitHub/Linear/GitLab pagination and lossless bodies | DONE | All three adapters now retain complete text; GitHub and Linear paginate beyond API page size and all warn on explicit truncation. |
| 5 | Fail-closed Jira bulk push | DONE | `internal/cli/sync.go` requires `--status` plus a configured transition before any cohort write. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `/private/tmp/hero-vnext sync import --dry-run --no-report` in Morpheus completed against live Jira, fetched 4,836 configured per-type records, and accounted for 2,281 bugs, 1,238 features, and 1,317 already-linked records without writing specs.

### Excellence Bar self-check

- [x] Yes — the corrected design is bulk-only, provider caps are honest, live Jira results disprove the old 128/508 assumption, and regression tests cover the exact N+1 mistake the user caught.
