---
title: "GitLab Tracker Support — Issues/Epics/Milestones/Iterations Round-Trip"
slug: gitlab-tracker-support
type: feature
status: planning
priority: high
horizon: next
tags: [tracker, gitlab, sync, round-trip]
created: 2026-06-27
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  originator_slug: pm-workbench-tracker-validation
  call_id: 18bd0775ec35d74042d241cd94a2fd37
  mode: spec-out
  at: 2026-06-27T19:40:30Z
  at_commit: f1d4c1b
  reason: "GitLab adapter is one of two prerequisites for the originator's tracker round-trip harness."
relations:
  - target: tracker-fixtures
    kind: parent
  - target: mock-tracker-server
    kind: sibling
---

# GitLab Tracker Support — Issues/Epics/Milestones/Iterations Round-Trip

## Provenance

Designed in response to `hero peer call --mode=spec-out` from peer
`hero-code` (peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`),
related to originator spec `pm-workbench-tracker-validation`.

## Goal

Make GitLab a first-class Hero tracker on the same footing as
github/jira/linear: it imports, it round-trips field-level pushes and
pulls, it participates in `hero spec set-owner`, and it satisfies every
`Tracker` interface contract the existing adapters do.

## Approach

Mirror `github.go` and `github_fields.go` as the closest structural
template — both speak REST, both have simple flat issue + label
shapes, and both have a parent-namespace + project addressing model
(GitHub `owner/repo`, GitLab `namespace/project` or numeric project
ID). Where GitLab's shape is richer (Epics, Iterations) we lean on
the existing convention rather than inventing one: tracker-specific
fields land in the `gitlab_*` frontmatter namespace, parent linkage
goes through existing parent/child relations.

### Files Added

- `internal/tracker/gitlab.go` — REST client, implements `Tracker`
- `internal/tracker/gitlab_fields.go` — implements `UpdateFields` /
  `GetFields` with `doWithRetry` + `classifyHTTPError`
- `internal/tracker/gitlab_test.go` — table-driven request/response
  fixtures using `httptest.Server`
- `internal/tracker/gitlab_fields_test.go` — diff/push/pull round-trip
  fixtures
- `internal/tracker/gitlab_sprint.go` — GitLab Iteration loader,
  implements the relevant slice of `SprintLoader`

### Files Touched (Wiring)

- `internal/tracker/tracker.go` — add `case "gitlab"` to `New()` and
  `NewWithJiraConfig()` (falls through to `New()` for non-jira types,
  same as linear/github today)
- `internal/tracker/sprint.go` — add `case "gitlab"` to
  `NewSprintLoader`
- `internal/tracker/size_mapping.go` — add `case "gitlab"` to
  `defaultSizeMapping`, returning `defaultGitLabSizeMapping()` (label
  prefix `workflow::size/`, mirroring GitLab's scoped-label
  convention)
- `internal/cli/sync_import.go` line 738 — add `"gitlab_"` to the
  prefix list (round-trip tracker-prefixed frontmatter preservation)
- `internal/cli/sprint.go` `detectTrackerPrefix` line 413 — add
  `"gitlab_"` to the prefix list
- `internal/cli/connect.go` — add `case "gitlab"` to the switch and
  a `runConnectGitLab` helper (token + base_url prompt; defaults to
  `https://gitlab.com`)
- `internal/config/config.go` — extend the `Type` docstring on
  `TrackerConfig` to include `gitlab`. No new struct fields:
  `BaseURL`, `Project`, `Token`/`TokenEnv` cover everything GitLab
  needs.

### Identity and Mappings

| GitLab concept | Hero concept | Frontmatter | Spec type |
|---|---|---|---|
| Issue (project-scoped) | story / bug | `gitlab_id`, `gitlab_iid`, `gitlab_url` | story or bug (from labels / issue_type) |
| Epic (group-scoped, Premium) | initiative / epic | `gitlab_epic_id` | initiative or epic |
| Milestone | release | `gitlab_milestone` | release |
| Iteration (group, Premium) | cycle_index | `gitlab_iteration` | (frontmatter only) |
| Status | spec status | `gitlab_status` | — |
| Priority label (`priority::high`) | spec priority | `gitlab_priority` | — |
| Assignee (username) | claimed_by | `gitlab_assignee` | — |
| Labels | spec tags | (full list under `gitlab_labels`) | — |

**Stable external ID.** Hero's existing convention keys imported specs
by lowercased tracker ID. For GitLab issues use the global ID
(`<project_path>#<iid>`) because per-project IIDs collide across
projects in a group. Epics use `<group_path>&<id>`. The CLI
`gitlab_id` field always holds the global form; the human-friendly
`gitlab_iid` is a convenience for display only.

**Issue type detection.** GitLab has an explicit `issue_type` field
(`issue`, `incident`, `test_case`, `task`). Map `incident` → bug,
`task` → story, `issue` → story unless a label matches the bug-label
convention (`type::bug`, `kind/bug`, or workspace-configured
`type_label_prefix`), then bug. Defaults match `github.go`'s
label-driven type inference.

**Size mapping.** GitLab Premium has native "weight" (integer). When
available, prefer weight over labels. When not, fall through to the
scoped-label convention `workflow::size/<tier>`, parallel to GitHub's
`size/<tier>`. The size_mapping path already supports both numeric and
label-prefix modes — pass `Field: "weight"` (numeric) or
`Field: "workflow::size/"` (label) at construction based on a
`tracker.gitlab.size_field` config knob.

### Field-Level Round-Trip

`UpdateFields` writes via `PUT /api/v4/projects/:id/issues/:iid` with
the same diff shape as the github adapter:

| Canonical field | GitLab field | Notes |
|---|---|---|
| `title` | `title` | Direct |
| `description` | `description` | Markdown round-trips |
| `labels` | `labels` | GitLab `PUT` accepts comma-separated `labels=`; merge non-hero labels via GET-then-PUT, exactly like github mergeLabels |
| `points` | `weight` | Premium only; warn-and-skip on 403 |
| `priority` | (label rotation: `priority::*`) | Read-then-rotate, same shape as size labels |

`GetFields` reads `GET /api/v4/projects/:id/issues/:iid` and returns
the same canonical keys. Unknown / unwritable fields are silently
omitted, matching the github adapter's contract.

### Error Classification

Reuse `doWithRetry` and `classifyHTTPError("gitlab", …)` — GitLab uses
standard HTTP status codes and standard `Retry-After` semantics, so no
adapter-specific retry logic. The classifier already maps 401/403 →
auth, 429 → rate-limited, others → opaque, which is exactly what the
CLI's field-error exit-code mapping wants.

### Pagination

GitLab uses `Link` headers and `?per_page=100&page=N`. `ListIssues`
follows `Link: rel="next"` until exhausted, capped at the existing
`defaultPullLimit = 100` for scan-time pulls. Bulk-import paths
(`hero sync import`) follow links unbounded.

## Acceptance Criteria

1. **AC-1: `Tracker` interface** — `gitLab` type satisfies
   `tracker.Tracker` (compile-time enforced). `New(cfg)` with
   `Type: "gitlab"` returns a working adapter; missing `BaseURL`
   returns a clear error at construction.
2. **AC-2: Import** — `hero sync import` against a real GitLab project
   produces spec scaffolds with correct type (story/bug/initiative/
   epic/release), correct `gitlab_*` frontmatter, and correct
   parent/child relations (issues → epic, issues → milestone).
3. **AC-3: Field push** — `hero sync push --field title`,
   `--field description`, `--field labels` round-trip cleanly:
   modifying the local spec then pushing results in the GitLab issue
   reflecting the change.
4. **AC-4: Field pull** — `hero sync pull --field title` overwrites
   local title with the server's title. After a server-side edit,
   pull updates the local spec; `sync` reports drift in dry-run.
5. **AC-5: Status sync** — completing/superseding a spec closes the
   GitLab issue via `PUT state_event=close`; reopening reopens.
6. **AC-6: `hero spec set-owner`** — setting an owner on a
   GitLab-imported spec writes the owner-history block and (if the
   adapter advertises assignee support) PUTs the GitLab assignee.
7. **AC-7: Frontmatter prefix round-trip** — `detectTrackerPrefix`
   returns `"gitlab"` for any `gitlab_*` frontmatter field; the
   sync_import preservation loop preserves arbitrary `gitlab_*`
   fields across re-import.
8. **AC-8: Premium-feature degradation** — when Epics or Iterations
   return 403/404 (open-source / unlicensed instance), import
   continues without them and emits one notice line per category, not
   one per request.
9. **AC-9: Sprint loading** — `hero sprint load <iteration-id>`
   against a GitLab project with Iterations enabled returns the
   expected `SprintItem` list.
10. **AC-10: Tests** — unit tests cover happy path, 401, 403, 404,
   429 (single retry honored), label-merge preservation, pagination
   exhaustion, premium-degradation. All driven by `httptest.Server`,
   no live network calls.

## Open Questions

- **GraphQL vs REST?** GitLab's GraphQL surface is more uniform than
  REST (Epics, Iterations, Weights all in one query), but mixing
  REST + GraphQL within one adapter complicates retries and
  pagination. **Decision: REST-only for v1**, matching the github
  adapter's surface area. If a future capability (e.g. nested epic
  trees) demands it, add GraphQL behind a feature flag.
- **Self-managed quirks.** Some self-managed GitLab instances disable
  the `Link` header. Fallback: if no `Link` header is present on page
  1, assume single page and stop. Document the limitation in the
  adapter docstring.

## Completion Ledger

| AC | File | Status |
|----|------|--------|
| AC-1 | `internal/tracker/gitlab.go`, `internal/tracker/tracker.go` | ✅ `newGitLab` rejects empty `base_url`; compile-time `var _ Tracker = (*gitLab)(nil)`; `New()` wires `case "gitlab"`. Test: `TestNewGitLab_*`. |
| AC-2 | `internal/tracker/gitlab.go`, `internal/cli/sync_import.go` | ✅ `toIssue` maps issue_type→story/bug, embeds epic/milestone/iteration refs; `sync_import` writes `gitlab_*` frontmatter generically (prefix from `Name()`). Standalone epic→initiative/epic via `ListEpics`. Test: `TestGitLab_GetIssue`, `TestGitLab_IssueTypeMapping`. |
| AC-3 | `internal/tracker/gitlab_fields.go` | ✅ `UpdateFields` PUTs title/description; labels GET-then-PUT merge. Test: `TestGitLab_UpdateFields_TitleDescription`, `..._LabelMergePreservation`. |
| AC-4 | `internal/tracker/gitlab_fields.go`, `internal/cli/pull.go` | ✅ `GetFields` reads title/description/labels/weight (drives the existing diff/pull path; `pull.go` is tracker-agnostic). Test: `TestGitLab_GetFields`. |
| AC-5 | `internal/tracker/gitlab.go` (`UpdateStatus`) | ✅ note + `state_event=close` on completed/superseded, `reopen` otherwise. Test: `TestGitLab_UpdateStatus_Completed`. |
| AC-6 | `internal/tracker/gitlab_fields.go` | ✅ assignee surfaced via `toIssue` (`gitlab_assignee`); owner-history block is the tracker-agnostic `spec set-owner` path. Note: assignee write-back is not advertised in v1 (parity with github — labels/title/description only). |
| AC-7 | `internal/cli/sprint.go`, `internal/cli/sync_import.go` | ✅ `"gitlab_"` added to both prefix lists; `detectTrackerPrefix` returns `"gitlab"`. |
| AC-8 | `internal/tracker/gitlab.go`, `internal/tracker/gitlab_sprint.go` | ✅ `ListEpics` 403/404 → `(nil,false,nil)` (one notice, continue); iterations 403/404 → clear error. Test: `TestGitLab_ListEpics_PremiumDegradation`, `TestGitLabSprint_PremiumDegradation`. |
| AC-9 | `internal/tracker/gitlab_sprint.go`, `internal/tracker/sprint.go` | ✅ `gitlabSprintLoader` over group Iterations; `NewSprintLoader` wires `case "gitlab"`. Test: `TestGitLabSprint_LoadIteration`. |
| AC-10 | `internal/tracker/gitlab_test.go`, `internal/tracker/gitlab_fields_test.go` | ✅ happy path, 401/403, 404, 429-single-retry, label-merge, pagination-to-exhaustion + single-page fallback, premium degradation. All `httptest.Server`, no live network. |

## Dependencies

- None on Hero side. The `mock-tracker-server` sibling depends on
  this spec (so its `gitlab` mode has a wire shape to mirror), but
  this spec does not depend on it.
