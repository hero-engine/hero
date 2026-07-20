# Delivery audit — tracker-diagnosis-full-ticket-evidence

**Audited:** `git diff --cached -- .hero/planning/bugs/tracker-diagnosis-full-ticket-evidence/spec.md domains/engineering/commands/diagnose.md internal/cli/sync_evidence.go internal/cli/sync_evidence_test.go internal/install/harness_smoke_test.go internal/tracker/tracker.go internal/tracker/jira.go internal/tracker/tracker_test.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Fetch exactly the linked issue through Hero's credential without disclosure — `internal/cli/sync_evidence.go:33-60` resolves one spec `tracker_id` and passes only that ID to the configured adapter; `internal/tracker/jira.go:665-672` applies auth internally. The provided live MORPH-14171 exercise confirms the command worked without credentials in argv or output.
- [✓] Preserve raw fields, field names, normalized description, and changelog — `internal/tracker/tracker.go:20-31` defines the envelope and `internal/tracker/jira.go:665-704` requests `fields=*all&expand=names,changelog` and retains each payload; `TestJira_GetIssueEvidence_FullFieldsCommentsAndAttachment` verifies the all-field query, normalized description, raw-field corpus, and changelog.
- [✓] Exhaust comment pagination and preserve readable and raw bodies — `internal/tracker/jira.go:734-774` advances until Jira's total and assigns both `Text` and `RawBody`; `TestJira_GetIssueEvidence_FullFieldsCommentsAndAttachment` verifies two pages are fetched, and the provided live exercise returned all MORPH-14171 comments.
- [✓] Return attachment metadata and download same-origin content to inspectable paths — `internal/tracker/jira.go:706-721` retains metadata, `internal/cli/sync_evidence.go:75-96` writes 0600 files and records `local_path`, and the Jira/CLI fixture tests verify downloaded bytes and an inspectable path.
- [✓] Report failed attachment downloads as explicit omissions without losing ticket evidence — `internal/cli/sync_evidence.go:85-88` appends the failure and continues; `TestDownloadEvidenceAttachments_WritesInspectableFilesAndReportsOmissions` verifies one successful file and one explicit omission in the same envelope.
- [✓] Reject off-origin attachment URLs before adding authentication — `internal/tracker/jira.go:777-794` compares scheme and host before request construction and `setHeaders`; `TestJira_GetIssueEvidence_FullFieldsCommentsAndAttachment` rejects a lookalike host.
- [✓] Keep full-ticket evidence on demand and out of bulk import/refresh — `internal/tracker/tracker.go:54-60` makes evidence an optional interface and `internal/cli/sync_evidence.go:17-31` exposes it only as `sync evidence`; bulk-refresh regression tests report zero single-issue fan-out, and the provided full suite passed.
- [✓] Do not infer remote absence from an empty local scaffold — `domains/engineering/commands/diagnose.md:12-20` requires successful evidence preflight and distinguishes empty, omitted, authentication-failed, and helper-failed states; the provided MORPH-14171 exercise covered the local-empty/remote-populated case.
- [✓] Install the same preflight for every supported harness — `internal/install/harness_smoke_test.go:214-244` renders and checks OpenCode, Cursor, Claude, Copilot, Codex, and generic targets.

## Changes

- [✓] Optional complete evidence model — `internal/tracker/tracker.go:20-60` adds lossless evidence, comments, attachments, omissions, and a separate optional capability.
- [✓] Jira full-ticket and secure attachment implementation — `internal/tracker/jira.go:662-803` implements full issue retrieval, comment pagination, attachment metadata, and same-origin authenticated downloads.
- [✓] `hero sync evidence` command and ignored cache — `internal/cli/sync_evidence.go:17-98` resolves a spec, emits JSON, and writes protected attachment files under `.hero/cache/tracker-evidence`; `.gitignore:23` already excludes `.hero/cache/`.
- [✓] Diagnose workflow provenance rule — `domains/engineering/commands/diagnose.md:12-20` requires credential-safe evidence retrieval, screenshot inspection, and honest provenance distinctions.
- [✓] Evidence and harness regression coverage — `internal/tracker/tracker_test.go:681-763`, `internal/cli/sync_evidence_test.go:26-45`, and `internal/install/harness_smoke_test.go:214-244` cover full fields, pagination, attachment security/download/omissions, and all install targets.

## Audit notes

- No Completion Ledger rows were downgraded. The provided full `go test ./...`, focused tests, and live read-only MORPH-14171 exercise all passed.
