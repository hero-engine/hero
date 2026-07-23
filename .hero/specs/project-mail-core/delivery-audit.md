# Delivery audit — project-mail-core

**Audited:** `git diff --cached 42055dc`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: atomically persist one immutable envelope and one outbound receipt — `internal/attention/mail/store.go` uses locked, fsynced atomic creation for both records; `TestStorePrivateImmutableDeliveryAndReplay` asserts both files, private modes, and envelope immutability.
- [✓] AC-2: equivalent idempotency retry returns the original delivery without duplication — `Store.Deliver` normalizes generated identity fields before comparison; `TestServiceSendReplayConflictAndReceipts` and `TestStorePrivateImmutableDeliveryAndReplay` assert the original message ID and `created == false` on replay.
- [✓] AC-3: conflicting idempotency key or message ID fails without overwrite — `Store.Deliver` returns `ErrIdempotencyConflict` on content, sender/recipient, or existing-message mismatch; store and service tests assert the conflict and preserved body.
- [✓] AC-4: preserve root/reply identity and reject invalid reply targets — `Service.deliver` establishes root identity and `Service.Reply` retains the original thread; `TestServiceReplyPreservesThreadAndRejectsBadTargets` asserts reply identity plus missing and cross-thread rejection.
- [✓] AC-5: JSON inbox/show/reply/ack operations use contract records and structured errors — `internal/cli/mail.go` emits `MailDelivery`, `ListedMessage`, and `MailReceipt` values and stable coded errors; `TestMailCLIJSONCommandsAndErrors` exercises the complete command flow and asserts a nonzero structured `missing` error.
- [✓] AC-6: show and ack update a separate atomic receipt without rewriting the envelope — `Store.UpdateReceipt` locks and atomically replaces only receipt state; `TestServiceSendReplayConflictAndReceipts` and `TestStoreConcurrentReceiptUpdatesPreserveState` assert preserved acknowledgement/read state and unchanged envelopes.
- [✓] AC-7: resolution and identity failures precede mailbox writes — `Service.Send` resolves the configured path and validates the peer manifest and IDs before calling delivery; `TestServiceResolutionFailsBeforeWriting` asserts no partial mailbox state.
- [✓] AC-8: no model/server execution or recipient-repository writes — the implementation calls only config, peer-manifest, and injected state APIs; `TestServiceEndToEndDoesNotMutateProjects` delivers between committed projects and asserts both worktrees remain clean.
- [✓] AC-9: invalid limits and unsafe storage identifiers fail without partial delivery — contract/service validation rejects size and UTF-8 violations, while store validation rejects traversal and malformed outbound/receipt state; contract, store, and service tests assert rejection and absence of partial records.
- [✓] AC-10: two temporary projects remain byte-for-byte clean — `TestServiceEndToEndDoesNotMutateProjects` compares tracked-file bytes and asserts empty `git status --porcelain` output for both repositories.

## Changes
- [✓] Add mail store, lock, and store tests — added `internal/attention/mail/lock.go`, `store.go`, and `store_test.go` with private layout, locking, atomic writes, replay/conflict handling, receipts, and concurrency coverage.
- [✓] Add service and service tests — added `internal/attention/mail/service.go` and `service_test.go` for peer resolution, delivery, replies, inbox/show, acknowledgement, and fail-closed behavior.
- [✓] Reuse peering identity, resolve, and manifest APIs — `Service.Send` and `Service.Reply` use `Config.ResolveRepoPath`, configured peer metadata, and `peering.ReadPeerManifest`; no duplicate manifest parser was added.
- [✓] Add and register the mail CLI — added `internal/cli/mail.go` and `mail_test.go`; `internal/cli/root.go` registers send, inbox, show, reply, and ack under `hero mail`.
- [✓] Add only necessary JSON result/error structs — contract DTOs gained delivery and receipt fields; the CLI adds only its small structured error type and composes existing envelope/receipt DTOs.
- [✓] Add the two-project non-mutation test — `TestServiceEndToEndDoesNotMutateProjects` initializes and commits both projects, delivers through the injected global state root, and verifies clean worktrees.
- [✓] Update CLI reference and completion coverage — README and MkDocs navigation/reference document Project Mail; `TestMailCommandTreeFeedsDynamicShellCompletion` verifies all dynamic Cobra children.

## Open items (if any)
- None.

## Audit notes
- The supplied ledger has all ten acceptance-criterion rows and all seven Changes rows marked DONE, with a populated end-to-end exercise check.
- Supplied validation evidence reports focused tests, race tests, the unrestricted full Go suite, strict MkDocs build, drift 10/10, and `git diff --cached --check` all passing.
