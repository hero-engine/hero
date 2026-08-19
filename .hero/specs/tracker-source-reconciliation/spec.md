---
title: "Tracker import fidelity and render-safe reconciliation"
slug: tracker-source-reconciliation
type: feature
status: completed
priority: high
size: small
tags: [tracker, import, source-fidelity]
completed_at: 2026-07-22T20:30:26Z
created: 2026-07-22
---

# Tracker source reconciliation

## Kickoff

Verify source fidelity for new imports and keep historical reconciliation non-destructive. Full item evidence remains a separate private snapshot; consumers blend it with the local spec at render time. Preserve every Hero-authored workflow section and keep legacy imported specs safe through the existing untouched-placeholder repair rule.

## Goal

Make newly imported specs useful before card hydration without allowing a tracker refresh to overwrite diagnosis or delivery work.

## Design

- Verify that Jira bulk search requests and preserves the canonical description through ADF conversion and generated specs.
- Keep the full item evidence snapshot separate from the local spec; do not copy comments, changelogs, attachments, or refreshed source text into authored workflow sections.
- Preserve Kickoff, Root Cause, Investigation, Fix, Changes, completion ledgers, and audit notes byte-for-byte.
- Legacy specs retain the existing conservative placeholder-only repair path. Partial non-placeholder descriptions are not rewritten implicitly.
- `hero sync import --dry-run` performs no writes.

## Acceptance Criteria

- New imports contain the complete canonical tracker description in Problem/Goal.
- Search-to-spec tests cover nested Jira ADF structures rather than only injecting an already-normalized description.
- Historical refresh never replaces a non-placeholder Problem/Goal.
- Authored workflow content survives refresh unchanged.
- Dry-run performs no filesystem writes.
- Tests cover fresh import, safe replacement, legacy repair, preservation, and idempotence.

## Completion Ledger

Delivered 2026-07-22 by verifying and extending the canonical Jira ADF read path used by bulk import, item reads, evidence, and sprint loading. Historical non-placeholder specs remain intentionally untouched; Hero Code performs read-only presentation reconciliation.

### Acceptance Criteria

| Criterion | Status | Evidence |
|---|---|---|
| Complete source description is available to new imports | DONE | `TestJiraADFMarkdown_IsIdenticalAcrossReadSurfaces` proves Search/ListItems receive the canonical MORPH-297 Markdown. |
| Search conversion covers nested Jira ADF | DONE | MORPH-297 fixture includes lists/code/semantic nodes and is exercised through live HTTP-shaped search responses. |
| Historical authored content is not overwritten | DONE | No refresh/writeback path was added; evidence remains private and the consumer reconciles immutable snapshots. |
| Dry run/idempotence behavior remains intact | DONE | Existing import/CLI suite passes unchanged. |
| Sprint pagination preserves source conversion | DONE | Jira ADF test now verifies both paginated sprint results use the same canonical Markdown. |

### Exercise-the-feature check

- [x] The MORPH-297 ADF fixture was exercised across GetIssue, ListIssues, Search, GetFields, evidence, and paginated sprint loading.
- [x] Full `go test ./...` passed.
