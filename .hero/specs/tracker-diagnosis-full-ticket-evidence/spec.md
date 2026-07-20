---
title: "Tracker diagnosis cannot load complete ticket evidence"
slug: tracker-diagnosis-full-ticket-evidence
type: bug
status: completed
domain: engineering
root_cause_class: design
severity: high
priority: high
size: medium
created: 2026-07-20
tags: [tracker, jira, diagnosis, evidence, attachments, comments]
relations:
  - target: tracker-source-evidence-preflight
    kind: related
  - target: jira-import-per-type-filter-and-body-parity
    kind: related
delivery_method: manual
completed_at: 2026-07-20T22:31:15Z
---

# Tracker diagnosis cannot load complete ticket evidence

## Goal

Give diagnostic agents a credential-safe, on-demand way to retrieve a complete tracker ticket—including every field, full description, paginated comments, attachment metadata and local screenshot files—without adding heavy detail calls to bulk sync.

## Kickoff

Adds a read-only full-ticket evidence preflight after MORPH-14171 diagnosis falsely concluded Jira itself had no description.

**Status:** delivering — implementation and live MORPH-14171 evidence retrieval are complete; closing audit and verify remain.

**Pick up at:** cold-audit the staged evidence API/CLI/workflow diff and run `hero spec verify` after the audit ships.

→ `.hero/planning/bugs/tracker-diagnosis-full-ticket-evidence/spec.md`

**Files:** `internal/tracker/tracker.go`, `internal/tracker/jira.go`, `internal/cli/sync_evidence.go`, `domains/engineering/commands/diagnose.md`
**Skip:** reusing this one-ticket endpoint inside import or refresh loops.

## Problem

MORPH-14171's local imported spec was nearly empty while Jira displayed extensive environment and reproduction details. The diagnosis workflow saw the scaffold, lacked a credentialed deep-fetch helper, and confidently claimed Jira's body was empty. It then asked the user to manually inspect/paste Jira content. That blocks reliable diagnosis and exposes a provenance problem: local absence was treated as verified remote absence.

The model must not receive Jira credentials. Bulk sync should stay lightweight and paginated. A separate Hero-owned read-only call is therefore required for the single issue under investigation.

## Root Cause

The tracker interface exposed only a normalized single-issue summary. It had no explicit evidence capability, no raw-field envelope, no complete comment pagination, no authenticated attachment download, and no diagnosis workflow rule distinguishing a local omission from a verified empty remote field.

## Changes

1. Add an optional tracker evidence capability that returns normalized fields, every raw issue field with Jira field names, changelog, all paginated comments, attachment metadata, retrieval timestamp, and explicit omissions.
2. Implement Jira full-ticket evidence with Hero's configured credential and same-origin authenticated attachment downloads; never expose the credential in arguments or output.
3. Add `hero sync evidence <spec-slug>` to resolve the linked tracker ID, fetch one ticket on demand, download attachments into ignored `.hero/cache/tracker-evidence/<slug>/attachments`, and emit structured JSON.
4. Update every installed diagnose workflow to run the helper for tracker-backed bugs, inspect downloaded screenshots, and forbid remote-content claims unless the helper succeeded.
5. Add tests for all-field preservation, comment pagination, safe attachment download, explicit attachment omissions, and rendered harness workflow parity.

## Acceptance Criteria

- AC-1: WHEN a tracker-backed diagnosis requests evidence, THE SYSTEM SHALL fetch exactly the linked issue through Hero's configured credential without exposing that credential to the model.
- AC-2: WHEN Jira returns issue fields, THE SYSTEM SHALL preserve all raw fields, their human-readable names, the normalized description, and changelog in the evidence envelope.
- AC-3: WHEN Jira comments span multiple pages, THE SYSTEM SHALL fetch them to exhaustion and preserve both readable text and raw bodies.
- AC-4: WHEN Jira includes attachments, THE SYSTEM SHALL return their metadata and download same-origin content to local inspectable paths by default.
- AC-5: IF an attachment cannot be downloaded, THEN THE SYSTEM SHALL report an explicit omission while retaining the rest of the ticket evidence.
- AC-6: IF an attachment URL is outside the configured Jira origin, THEN THE SYSTEM SHALL reject it without sending Jira authentication.
- AC-7: THE SYSTEM SHALL keep full-ticket evidence on demand and SHALL NOT call it from bulk import or refresh.
- AC-8: WHEN a diagnostic workflow has only an empty local scaffold, THE SYSTEM SHALL NOT claim the remote tracker field is empty unless evidence retrieval verified it.
- AC-9: WHEN Hero installs workflows for any supported harness, THE SYSTEM SHALL include the same evidence preflight and provenance constraint.

## Validation

- Use an HTTP fixture with raw custom fields, ADF description, two comment pages, changelog, and a screenshot attachment.
- Verify the screenshot is downloaded with a local path and a lookalike host is rejected.
- Verify a failed attachment is represented in `omissions` without failing the evidence envelope.
- Verify all generated harness diagnose instructions contain the evidence call and remote-absence rule.
- Run focused tracker, CLI, install, and full Go tests.

## Boundaries

- The command is read-only with respect to Jira and local specs.
- Attachments are ephemeral cache artifacts, not committed project data.
- Provider support is explicit; unsupported trackers return an honest error rather than a lossy substitute.
- Do not perform one full-ticket request per issue during import, refresh, preview, or inventory.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | One linked issue uses Hero credential without disclosure | DONE | `hero sync evidence <slug>` resolves one `tracker_id` and invokes the configured adapter; live MORPH-14171 retrieval succeeded without credentials in argv/output. |
| 2 | Preserve all raw fields, names, description, changelog | DONE | `IssueEvidence` and Jira `fields=*all&expand=names,changelog`; the live envelope returned the complete description and named custom-field corpus. |
| 3 | Paginate all comments with text and raw bodies | DONE | `fetchAllComments` walks to `total`; `TestJira_GetIssueEvidence_FullFieldsCommentsAndAttachment` covers two pages and the live ticket returned all comments. |
| 4 | Attachment metadata and same-origin local downloads | DONE | Jira attachment metadata plus `downloadEvidenceAttachments`; fixture test writes an inspectable local image path. |
| 5 | Failed attachment becomes explicit omission | DONE | `TestDownloadEvidenceAttachments_WritesInspectableFilesAndReportsOmissions` proves partial evidence remains usable. |
| 6 | Reject off-origin attachment URL | DONE | URL origin comparison precedes auth; `TestJira_GetIssueEvidence_FullFieldsCommentsAndAttachment` rejects a lookalike host. |
| 7 | Evidence remains on-demand, never bulk | DONE | Separate optional `EvidenceTracker`/`sync evidence` command; bulk refresh regression tests assert zero `GetIssue` fan-out and have no evidence calls. |
| 8 | Empty local scaffold cannot prove remote absence | DONE | Diagnose workflow explicitly forbids that inference; live MORPH-14171 proved the local-empty/remote-populated case. |
| 9 | All installed harnesses get evidence preflight | DONE | `TestHarness_DiagnosePullsCredentialSafeTrackerDescriptionForAllTargets` covers OpenCode, Cursor, Claude, Copilot, Codex, and generic installs. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Optional complete evidence model | DONE | `internal/tracker/tracker.go` adds lossless raw fields, names, changelog, comments, attachments, and omissions without widening the base Tracker contract. |
| 2 | Jira full-ticket and secure attachment implementation | DONE | `internal/tracker/jira.go` performs one full issue request, comment pagination, and same-origin authenticated downloads. |
| 3 | `hero sync evidence` command and ignored cache | DONE | `internal/cli/sync_evidence.go` emits structured JSON and writes 0600 attachment files under already-ignored `.hero/cache`. |
| 4 | Diagnose workflow provenance rule | DONE | `domains/engineering/commands/diagnose.md` requires the evidence preflight, screenshot inspection, and honest empty/omission/error distinctions. |
| 5 | Evidence and harness regression coverage | DONE | Tracker, CLI attachment, origin-security, and six-target install tests pass. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `/private/tmp/hero-vnext sync evidence morph-14171-instance-state-changed-to-unknown-when` fetched live Jira all-fields data, the complete missing description, field-name map, changelog, and all comments. Jira reported no attachments for this ticket; the attachment fixture exercised download and omission behavior.

### Excellence Bar self-check

- [x] Yes — credentials remain inside Hero, the output is lossless and provenance-aware, attachment auth is origin-checked, and the capability is structurally separate from bulk sync.
