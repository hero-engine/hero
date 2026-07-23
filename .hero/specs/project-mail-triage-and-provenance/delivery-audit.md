# Delivery audit — project-mail-triage-and-provenance

**Audited:** `git diff --cached`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: read, acknowledge, and dismiss atomically update only revisioned receipts — `internal/attention/mail/triage.go:51`; `TestTriageReceiptActionsAreRevisionedIdempotentAndOrthogonal`
- [✓] AC-2: promotion reuses Intake/roadmap authority with typed Mail provenance — `internal/attention/mail/promotion.go:20`; `internal/intake/service.go:66`; `TestCapturePersistsTypedMailSourceMetadata`
- [✓] AC-3: partial and exact promotion retries do not duplicate work — `internal/attention/mail/promotion.go:61`; `TestPromotionResumesAfterEveryStepAndWritesBodyFreeProvenance`
- [✓] AC-4: promotion returns artifact, project, source, and navigation identity — `internal/attention/mail/triage.go:40`; `internal/attention/mail/promotion.go:153`; `TestPromotionResumesAfterEveryStepAndWritesBodyFreeProvenance`
- [✓] AC-5: Add to Today is idempotent and orthogonal to other receipt state — `internal/attention/mail/promotion.go:162`; `TestAddToTodayIsIdempotentAndDoesNotAlterOtherReceiptFields`
- [✓] AC-6: why traverses artifact → Intake → Mail without body metadata — `internal/spec/graph_ingest.go:175`; `TestPromotionResumesAfterEveryStepAndWritesBodyFreeProvenance`
- [✓] AC-7: resume and status preserve existing JSON fields and add bounded deterministic unread summaries — `internal/cli/brief.go:137`; `internal/cli/status.go:288`; `TestUnreadSummaryIsBoundedOldestFirst`; `TestMailCLIJSONCommandsAndErrors`
- [✓] AC-8: project-scoped MCP Mail tools delegate to the shared service and shape successes/failures — `internal/serve/mcp_tools_mail.go:51`; `TestMCPMailToolsAdvertiseAndReturnStructuredFailures`; `TestMCPMailDefinitionsAndDispatch`
- [✓] AC-9: stale, missing, and unsupported requests do not mutate and stale errors return current state — `internal/attention/mail/store.go:248`; `internal/attention/mail/triage.go:58`; `TestTriageReceiptActionsAreRevisionedIdempotentAndOrthogonal`; `TestMCPMailToolsAdvertiseAndReturnStructuredFailures`
- [✓] AC-10: Mail content remains data and work creation stays behind Intake/Focus authorities — `internal/intake/service.go:173`; `internal/attention/mail/promotion.go:85`; `internal/attention/mail/promotion.go:197`; `TestCaptureMailSourceDoesNotPrefixMatchOrInterpretTitle`

## Changes
- [✓] Extract Intake storage authority from Cobra — new `internal/intake/service.go` and `repository.go`; `internal/cli/intake.go` reduced to adapter behavior with characterization coverage retained.
- [✓] Add revisioned receipt persistence and triage service — `contracts/attention/mail.go`, `internal/attention/mail/store.go`, and new `triage.go`.
- [✓] Add promotion, Focus, graph, and feed adapters — new `internal/attention/mail/promotion.go`, graph ingestion, and resumable feed-event handling.
- [✓] Add Mail CLI triage commands and shared action paths — `internal/cli/mail.go`; `TestMailCLIJSONCommandsAndErrors`.
- [✓] Extend resume/status aggregation — `internal/cli/brief.go`, `internal/cli/status.go`, and bounded summary coverage.
- [✓] Add MCP definitions, dispatch, and handlers — `internal/serve/mcp_tools_def.go`, `mcp_dispatch.go`, and new `mcp_tools_mail.go`.
- [✓] Extend graph/event vocabulary and why coverage — `internal/feed/feed.go`, `internal/graph/edge.go`, `internal/spec/graph_ingest.go`, and `internal/traversal/why.go`.
- [✓] Deduplicate canonical Mail sources in capture guidance and implementation — `internal/intake/service.go`, `core/skills/auto-knowledge-capture/SKILL.md`, and `TestAllTargetsInstallMailSourceDedupGuidance`.

## Audit notes
- No performative DONE rows, partial items, skips, blockers, or out-of-scope drift found.
- Provided evidence reports `go test ./...`, focused `go vet`, `git diff --check`, CLI/MCP exercises, and six-target propagation all passed.
