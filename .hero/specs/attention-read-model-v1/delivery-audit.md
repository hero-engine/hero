# Delivery audit — attention-read-model-v1

**Audited:** `git diff --cached`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 projects active Mail, Today Focus, and pending suggestions with stable consumer fields and actions — `internal/attention/projection/service.go:48`, `internal/attention/projection/service.go:98`, `internal/attention/projection/service.go:121`, `internal/attention/projection/service.go:141`; `TestSnapshotOrderingRevisionAndCounts` and `TestSnapshotDoesNotAdvertiseProjectBoundActionsForUnboundRows`.
- [✓] AC-2 keeps snapshot revision stable when only generation time changes — `internal/attention/projection/service.go:78`, `internal/attention/projection/service.go:181`; `TestSnapshotOrderingRevisionAndCounts`.
- [✓] AC-3 applies deterministic group, activity-direction, missing-last, and source-ID ordering — `internal/attention/projection/ordering.go:10`; `TestSnapshotOrderingRevisionAndCounts`.
- [✓] AC-4 validates version, identity, advertised capability, revision, input, and idempotency before delegation — `internal/attention/projection/service.go:200`, `internal/attention/projection/actions.go:45`, `contracts/attention/validate.go:226`; `TestDispatchValidatesAndReturnsAuthoritativeInvalidation` and `TestDispatchRejectsIncompatibleVersionAndUnknownInput`.
- [✓] AC-5 returns authoritative source state, row/invalidation, snapshot revision, navigation, or launch results — `internal/attention/projection/service.go:244`, `internal/attention/projection/service.go:264`; `TestDispatchValidatesAndReturnsAuthoritativeInvalidation`, `TestDispatchDelegatesFocusAndSuggestionCapabilities`, and checked-in action-result fixtures.
- [✓] AC-6 returns structured stale, unsupported, missing, invalid, incompatible, and unavailable errors without replay — `internal/attention/projection/service.go:204`, `internal/attention/projection/service.go:222`, `internal/attention/projection/service.go:235`, `internal/attention/projection/service.go:322`, `internal/serve/api_attention.go:85`; projection dispatch tests and the v1 error fixtures.
- [✓] AC-7 serves the global snapshot before project routing without a selected project — `internal/serve/api.go:107`, `internal/serve/server.go:168`, `internal/serve/server.go:213`; `TestAttentionRouteWinsWithoutSelectedProject` and `TestAttentionUnavailableIsStructuredAndNotEmptySnapshot`.
- [✓] AC-8 keeps CLI, MCP, and HTTP on the same projection service and contract records — `internal/cli/attention.go:18`, `internal/serve/mcp_tools_attention.go:13`, `internal/serve/api_attention.go:10`; `TestAttentionTodayJSONMatchesProjectionService` and `TestAttentionHTTPAndMCPReturnTheSameProjectionRecords`.
- [✓] AC-9 converges through authoritative snapshot refresh without an event-stream dependency — successful dispatch re-snapshots at `internal/attention/projection/service.go:248`; global Attention routes are independent of `/api/events` in `internal/serve/api.go`; the on-disk exercise records a successful snapshot refresh path.
- [✓] AC-10 decodes the complete golden set with additive unknown values and pinned documented shapes — `contracts/attention/contract_test.go:21` validates 19 manifest entries, schemas, checksums, and DTO decoding; `TestForwardCompatibleRawValues` asserts unknown source/action/style preservation.
- [✓] AC-11 exposes only snapshot plus advertised-capability dispatch, with no generic write surface — `internal/attention/projection/service.go:228`, `internal/serve/api.go:110`, `internal/cli/attention.go:18`, `internal/serve/mcp_tools_def.go:10`; no generic Attention mutation appears in the staged surface.

## Changes

- [✓] Add projection service, ordering, actions, registry Mail adapter, and focused tests — added under `internal/attention/projection/`.
- [✓] Add snapshot, action, and contract HTTP handlers — `internal/serve/api_attention.go` with `internal/serve/api_attention_test.go`.
- [✓] Mount global routes before project routing and wire registry/global stores — `internal/serve/api.go:107`, `internal/serve/server.go:168`, `internal/serve/server.go:213`.
- [✓] Add and register Attention CLI over the projection service — `internal/cli/attention.go`, `internal/cli/root.go`, and `internal/cli/attention_test.go`.
- [✓] Add MCP definitions, dispatch, and handlers over the projection service — `internal/serve/mcp_tools_attention.go`, `internal/serve/mcp_tools_def.go`, `internal/serve/mcp_dispatch.go`, and MCP inventory coverage.
- [✓] Complete and pin the v1 fixture set — `contracts/attention/testdata/v1/manifest.json` lists 19 schema-checked, checksum-verified fixtures.
- [✓] Add the Hero Code handoff with the exact manifest checksum — `contracts/attention/testdata/v1/HERO-CODE-HANDOFF.md:3`.
- [✓] Update Serve and CLI/MCP documentation — `docs/serve.md` and `README.md`.

## Audit notes

- No Completion Ledger rows are PARTIAL, SKIPPED, or BLOCKED.
- The staged contract and Focus-service edits support the named projection and compatibility work; no unrelated scope drift was found.
- Test evidence supplied with the ledger reports focused package tests and `go test ./...` passing. The on-disk ledger also records a successful `hero attention today --json` exercise. This cold audit inspected those artifacts and test bodies; it did not rerun experiments.
