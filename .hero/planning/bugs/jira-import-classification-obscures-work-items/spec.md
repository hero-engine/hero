---
title: "Jira import classification makes successfully imported work appear missing"
slug: jira-import-classification-obscures-work-items
type: bug
status: superseded
domain: engineering
root_cause_class: design
severity: medium
priority: high
size: medium
created: 2026-07-20
tags: [jira, tracker, import, classification, visibility, hero-code]
superseded_by: jira-import-per-type-filter-and-body-parity
---

# Jira import classification makes successfully imported work appear missing

> Superseded by `jira-import-per-type-filter-and-body-parity`. The original count match was coincidental: later Jira evidence showed more than 1,000 open Bugs, while Hero silently capped fetches and the Hero Code importer bypassed configurable per-type JQL.

## Goal

Make Jira-to-Hero type classification explicit, configurable, and visible in import results so a successful broad import cannot look like hundreds of missing items when a consumer opens a type-specific view.

## Kickoff

Fix Hero's opaque Jira issue-type classification after Morpheus imported 508 active issues but exposed only 132 in Hero Code's Bugs view.

**Status:** planning — root cause confirmed in Hero Core; no source changes yet.

**Pick up at:** define configurable tracker issue-type mappings, then add an import classification summary and regression coverage for Jira Security and Chore types.

→ `.hero/planning/bugs/jira-import-classification-obscures-work-items/spec.md`

**Files:** `internal/cli/sync_import.go`, `internal/cli/sync_import_test.go`, `internal/config/config.go`, `internal/config/config_test.go`
**Skip:** changing Hero Code's Bugs view or importing completed Jira work by default; neither caused the observed count split.

## Summary

### Categorization

| Attribute | Assessment |
|---|---|
| **Criticality** | medium — imported work is present, but users reasonably conclude it was dropped |
| **Ease of Fix** | moderate — configuration, classification, reporting, and relocation must agree |
| **Caused by our codebase?** | Yes — Hero Core owns the mapping and generated `type:` frontmatter |
| **Needs more research?** | No — local Morpheus counts and both code paths establish the root cause |

### Background

After the tracker-import changes were exercised in Morpheus, Hero Code showed roughly 128 bugs while roughly 390 more Jira items appeared absent. The Morpheus Hero corpus actually contains 508 active imported Jira work specs: 132 bugs, 359 features, and 17 initiatives.

### Analysis

The apparent missing set is the type-classification remainder. Imported Jira types are 226 Story, 132 Bug, 107 Subtask, 18 Chore, 17 Epic, and 8 Security. Hero maps only `Bug` and `Defect` to `type: bug`; Stories and Subtasks map to features, Epics map to initiatives, and unknown types such as Security and Chore default to features. Hero Code's Bugs view correctly filters to `type == .bug`.

### Root Cause

Hero's Jira issue-type mapping is hard-coded and opaque. The import summary reports aggregate fetched/imported counts but does not show the native-type-to-Hero-type classification, and projects cannot declare that a native Jira type such as Security should be treated as a Hero bug. A successful import therefore looks incomplete when inspected through a type-specific consumer.

### Source

`internal/cli/sync_import.go` owns `inferSpecType`, placement, relocation, and summary output. `internal/config/config.go` has import filters but no native issue-type mapping. Hero Code reads the resulting Hero `type:` faithfully and its Bug Board filters to `.bug`.

### Fix Direction

Add optional per-integration/import configuration mapping native tracker issue-type names to Hero spec types, retain the existing mapping as the compatibility default, and print a native-type → Hero-type count summary for broad imports. Apply the same resolver to creation, filtering, relocation, and inventory reporting. Do not broaden the default Jira query to completed work.

## Problem Statement

Morpheus's broad Jira import fetched and persisted 508 active issues. A Bugs view then displayed only the 132 specs whose Jira type was literally Bug. The remaining 376 items were present under features and initiatives, closely matching the user's estimate of “another 390” missing from the view. No record-loss evidence was found.

The configured Hero Code peer advisory could not run because its Claude backend is not logged in, and the current chat transcript was not present in the legacy local `sessions.db`; those are evidence gaps. They do not prevent diagnosis because the on-disk Morpheus corpus accounts for the full count split and the producer/consumer code paths agree.

## Environment Details

- Morpheus workspace: `/Users/developer/projects/hpe/repository/morpheus`
- Jira project: `MORPH`
- Imported active work specs: 508
- Hero classification: 132 bug, 359 feature, 17 initiative
- Native Jira types: 226 Story, 132 Bug, 107 Subtask, 18 Chore, 17 Epic, 8 Security

## Root Cause Analysis

Confirmed findings:

1. `inferSpecType` maps only Bug/Defect to bug and defaults unknown Jira types to feature.
2. Generated specs persist that result as first-class `type:` frontmatter and are placed under the matching planning directory.
3. Hero Code's Bug Board filters the local store to `type == .bug`; it does not discard imported Jira records.
4. Every observed active Jira item is accounted for by the 508 local specs.
5. Jira's separate `ListIssues` default uses `statusCategory != Done`. That is an intentional active-work boundary and does not explain the observed 132/376 classification split.

Root-cause class: **design**. Fixed aliases are reasonable defaults, but treating them as an invisible, non-configurable universal taxonomy makes correctly imported work appear lost.

## Code Flow (End to End)

1. `internal/cli/sync_import.go:120-190` — resolves filters and fetches Jira issues.
2. `internal/tracker/jira.go:927-952` — broad Jira listing fetches active issues and includes native `issuetype`.
3. `internal/cli/sync_import.go:205-225` — calls `inferSpecType` for each fetched issue and applies any Hero type filter.
4. `internal/cli/sync_import.go:367-381` — hard-coded native issue-type mapping returns bug, initiative, feature, or feature-by-default.
5. `internal/cli/sync_import.go:225-285` — creates or relocates the spec using the inferred Hero type.
6. `packages/hero-swift/Sources/HeroSharedApplication/State/SpecStore.swift:493` — Hero Code loads planning specs by their Hero directories/frontmatter.
7. `packages/hero-swift/Sources/HeroSharedApplication/Content/BugBoardProjection.swift:827` — the Bug Board selects only specs whose Hero type is bug.

## Key Files

### Hero Core import

| File | Lines | Relevance |
|---|---:|---|
| `internal/cli/sync_import.go` | 120–290, 338–381 | fetch routing, classification, filtering, placement, and summary |
| `internal/cli/sync_import_test.go` | 50–90 | tests that pin the current fixed mapping |
| `internal/config/config.go` | 690–960 | import configuration and precedence |
| `internal/tracker/jira.go` | 927–1050 | active-item JQL and native type retrieval |

### Hero Code consumer evidence

| File | Lines | Relevance |
|---|---:|---|
| `packages/hero-swift/Sources/HeroSharedApplication/Content/BugBoardProjection.swift` | 827 | filters local specs to Hero bug type |
| `packages/hero-swift/Sources/HeroSharedApplication/Content/BugBoardView.swift` | 841–852 | renders bug-only rows/counts |

## Secondary Defects

- The installed `hero` binary used from Morpheus did not recognize the newer layered integration configuration, while current source does. This is version drift, not the count-split root cause; `hero doctor` should be used before live CLI reproduction.
- The peer-call backend failure prevents Hero-to-Hero advisory inspection until Claude is authenticated.

## Changes

1. Extend import configuration with a native issue-type mapping, scoped to the selected tracker connection/import configuration, from case-insensitive native names to supported Hero work types.
2. Replace `inferSpecType`'s direct switch with a resolver that applies configured mappings first and compatibility defaults second.
3. Use the same resolver for type filtering, creation, existing-spec relocation, type counts, and inventory reporting.
4. Print a bounded classification summary after fetch/import showing native Jira type, resulting Hero type, and count; include a clear line when an unknown native type used the default.
5. Add unit and command-level tests covering default compatibility, Security/Chore overrides, case-insensitivity, invalid target types, relocation, scoped `bugs` imports, and summary output.
6. Document the mapping in import help/config examples without changing Hero Code's type-specific views.

## Acceptance Criteria

- AC-1: WHEN an import has no custom issue-type mapping, THE SYSTEM SHALL preserve the existing Bug/Defect, Epic/Initiative, feature-alias, and unknown-to-feature behavior.
- AC-2: WHEN a configured native issue-type mapping matches case-insensitively, THE SYSTEM SHALL classify the issue using the configured Hero type.
- AC-3: IF a configured mapping targets an unsupported Hero type, THEN THE SYSTEM SHALL reject the configuration with a field-specific error before fetching issues.
- AC-4: WHEN import classification completes, THE SYSTEM SHALL report bounded native-type → Hero-type counts so every fetched issue is accounted for.
- AC-5: WHEN an unmapped native type falls back to the compatibility default, THE SYSTEM SHALL identify that fallback in the classification summary.
- AC-6: WHEN a scoped bugs import is run, THE SYSTEM SHALL apply the resolved mapping before deciding whether an issue belongs in the bug result.
- AC-7: WHEN a previously linked issue's resolved type changes, THE SYSTEM SHALL relocate the spec using the same mapping without duplicating or losing its tracker link.
- AC-8: THE SYSTEM SHALL preserve Jira's default active-work query and SHALL NOT make completed issues part of the default import.
- AC-9: THE SYSTEM SHALL require no Hero Code change for correctly classified specs to appear in its existing type-specific views.

## Validation

- Unit-test mapping resolution and config validation.
- Exercise `hero sync import --dry-run` against a fixture containing Bug, Story, Security, Chore, and Epic issues and assert the complete classification ledger.
- Run focused CLI/config/tracker tests and `go test ./internal/cli ./internal/config ./internal/tracker`.
- Re-run a Morpheus dry-run with its desired mapping and confirm fetched totals equal the sum of reported Hero type counts.

## Boundaries

- Do not change Hero Code's Bugs view to mix features or initiatives into a bug-specific surface.
- Do not change Jira's default `statusCategory != Done` active-work boundary in this fix.
- Do not mutate existing specs merely by loading configuration; relocation occurs only during an explicit import/refresh path.
- Do not add project-specific Jira type names to global hard-coded aliases.
- Do not redesign the broader Hero spec-type model.

## Notes

The useful product distinction is “missing from import” versus “present under another Hero type.” Import output should make that distinction obvious without requiring filesystem inspection.

## Recap

The original classification finding remains a secondary visibility concern, but it does not explain the missing Jira Bugs. This diagnosis is superseded by the corrected import-filter, truncation, and body-parity diagnosis.

## Investigation History

### Round 1 — Initial diagnosis
- **Date**: 2026-07-20T20:30:00Z
- **Agent**: debug-investigator
- **Root cause**: Hard-coded Jira-to-Hero type classification made imported work appear absent from the Bugs view.
- **Confidence**: Medium
- **Key evidence**: 508 local imported specs split into 132 bugs, 359 features, and 17 initiatives.

### Round 2 — Challenged (reject)
- **Date**: 2026-07-20T21:15:00Z
- **Challenged by**: engineer
- **Feedback**: "there are over 1000 bugs on jira" and "we used to have configurable JQL filters for stories and bugs for people to fine tune it"
- **Revised root cause**: The new importer bypasses per-type configured JQL, broad Core imports silently truncate, and summary-only fetches omit descriptions.
- **What changed**: The 508 local records are accumulated partial state, not a complete active Jira corpus; the apparent classification remainder was coincidence.
- **Confidence**: High
