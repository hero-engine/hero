# Delivery audit — cross-project-mail-read-contract

**Audited:** `git diff 76ed88ee6d8fed562b067074a456be30c99b60cc...8088f22`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — `internal/attention/mailquery/service.go:55` builds exact scoped counts and bounded pages from body-free summaries; `internal/serve/api_attention_mail_test.go:59` asserts the HTTP list omits body content.
- [✓] AC-2 — `internal/attention/mailquery/service.go:162` resolves detail by `(project_peer_id, message_id)` after exact registry resolution; `internal/attention/mailquery/service_test.go:68` and `:132` assert exact project/thread scope and duplicate-ID isolation.
- [✓] AC-3 — `internal/attention/mailquery/service.go:101` sorts parsed instants with stable peer/message tie-breakers and `:118` binds opaque cursors to filters plus metadata revision; `internal/attention/mailquery/service_test.go:68` and `cursor_test.go:5` assert ordering, continuation, filter binding, and stale invalidation.
- [✓] AC-4 — `contracts/attention/mailread/contract.go:46` defines normalized receipt, identity, activity, and action metadata for summary/detail responses; `contracts/attention/mailread/validate_test.go:41` asserts receipt, unread, activity, and action-revision invariants.
- [✓] AC-5 — `internal/attention/mailquery/service.go:88` filters only on the stored thread ID within the already-resolved project; `internal/attention/mailquery/service_test.go:105` asserts the exact project-scoped thread page.
- [✓] AC-6 — `internal/attention/mailquery/service.go:182` calls source-owned `Show(messageID, false)` and returns the full envelope; `internal/serve/api_attention_mail_test.go:59` asserts an exact 65,536-byte HTTP round trip, final-byte canary, and absent receipt.
- [✓] AC-7 — `internal/attention/mail/capabilities.go:25` derives the six canonical descriptors from central operation policy and owns the source mapping; `internal/attention/mail/capabilities_test.go:10` pins IDs, policy fields, revision, idempotency, schemas, and mapping.
- [✓] AC-8 — `internal/attention/mailquery/service.go:203` validates and maps canonical actions before one source-service dispatch and `:247` delegates typed replies with thread and idempotency guards; service and HTTP replay tests at `internal/attention/mailquery/service_test.go:181` and `internal/serve/api_attention_mail_test.go:95` assert authoritative delegation, replay, conflict, and stale behavior.
- [✓] AC-9 — list uses source `Inbox` and detail uses `Show(..., false)` with no body interpretation or mutation path in `internal/attention/mailquery/service.go:79` and `:182`; `internal/attention/mailquery/service_test.go:132` and `internal/serve/api_attention_mail_test.go:59` assert reads create no receipt.
- [✓] AC-10 — validators and source-error translation return typed contract errors in `contracts/attention/mailread/validate.go:14` and `internal/attention/mailquery/service.go:470`; registry conflict, unavailable mailbox, missing identity, bad requests, stale cursor, and unavailable-versus-empty are asserted in `internal/attention/mailquery/service_test.go:160`, `:233`, `:244`, and `internal/serve/api_attention_mail_test.go:139`, `:156`, `:198`.
- [✓] AC-11 — projection reuses canonical policy without changing legacy MCP code in `internal/attention/projection/actions.go:24`; `internal/serve/mcp_tools_mail_test.go:48` pins legacy bodies and `read` action behavior, while `contracts/attention/mailread/conformance_test.go:60` pins both released Attention artifact hashes.
- [✓] AC-12 — `contracts/attention/mailread/bundle.go:64` builds and validates the standalone vendorable bundle; `contracts/attention/mailread/conformance_test.go:17`, `:60`, and `:136` assert all 19 fixture categories, deterministic checksums, inert unknowns, composite ordering, canonical actions, and maximum body, while `internal/serve/api_attention_mail_test.go:156` pins runtime manifest parity.

## Changes

- [✓] Add independent Mail-read DTOs, validators, schemas, and tests — `contracts/attention/mailread/contract.go`, `validate.go`, `schema/v1/`, `validate_test.go`, and `conformance_test.go` are new in the audited diff and implement the v1 wire contract and validation suite.
- [✓] Centralize canonical Mail capabilities and preserve projection/MCP semantics — `internal/attention/mail/capabilities.go` adds the shared policy-derived builder and source mapping; `internal/attention/projection/actions.go` consumes it; `internal/serve/mcp_tools_mail_test.go` pins legacy behavior.
- [✓] Add registry-backed source-owned query facade and cursors — `internal/attention/mailquery/service.go`, `cursor.go`, and their tests implement registered discovery, conflict handling, composite reads, filtering, paging, revisions, and source delegation without a second store.
- [✓] Add typed Hero Serve Mail adapters and process-global wiring — `internal/serve/api_attention_mail.go`, `api.go`, `api_attention.go`, and `server.go` add and mount all five global routes with bounded decoding and structured status mapping; `api_attention_mail_test.go` exercises the HTTP boundary.
- [✓] Publish standalone vendorable golden conformance bundle — `contracts/attention/mailread/bundle.go`, `cmd/mailread-bundle/main.go`, `testdata/v1/`, and `conformance/v1/` add the deterministic independently hashed schema/fixture bundle and consumer validation.
- [✓] Add cross-boundary regressions and consumer guidance — `internal/serve/api_attention_mail_test.go`, both Attention `HERO-CODE-HANDOFF.md` sources, `docs/serve.md`, and `.hero/knowledge/decisions/hero-code-mail-reads-use-http/spec.md` pin HTTP/MCP boundaries and document HTTP-only consumption.

## Open items (if any)

- None.

## Audit notes

- No Completion Ledger rows were downgraded. All 12 acceptance criteria and all six Changes items have implementation and assertion evidence in the audited diff.
- Provided validation evidence reports focused package tests, uncached `go test ./... -count=1`, formatting/diff checks, the exact maximum-body HTTP exercise, and the 300-message scan benchmark all passing.
