# Delivery audit — github-pull-request-create-broker

**Audited:** `git diff 8a6497f...HEAD`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: require the complete creation target, explicit draft, user intent, idempotency key, and revisions — `internal/codehost/mutations.go:27` rejects an omitted draft; the v1 request validator supplies the remaining required target and mutation-policy checks; `TestCreateRequiresExplicitDraftAndPolicyMaterial` asserts rejection before any create attempt.
- [✓] AC-2: preserve owner-qualified fork identity — `internal/codehost/github_create.go:193` queries and filters the repository-qualified base/head target, and `internal/codehost/github_create.go:223` sends `owner:ref`; `TestCreateExternallyCompletedForkAndAmbiguousRecovery` asserts the configured fork identity.
- [✓] AC-3: reject permission, scope, ref, capability, and observation failures before write — `internal/codehost/github_create.go:109`, `internal/codehost/github_create.go:124`, and `internal/codehost/github_create.go:302` implement the pre-dispatch gates; `TestCreateRejectsStalePermissionScopeAndCancellationBeforeDispatch` asserts typed failures with zero create attempts.
- [✓] AC-4: same key and payload perform at most one provider create — `internal/codehost/idempotency.go:93` takes the cross-platform journal lock around ownership and execution, while `internal/codehost/github_create.go:287` reconciles an existing entry before dispatch; `TestCreateDuplicateConcurrentAndConflictingRetries` covers sequential, concurrent, canonical-identity, and rejected-write retries with one attempt.
- [✓] AC-5: different canonical payload under the same key conflicts without another write — `internal/codehost/mutations.go:50` hashes canonical target and payload material, `internal/codehost/mutations.go:83` scopes the key by provider connection, and `internal/codehost/github_create.go:257` rejects digest mismatch; the title and cross-repository regressions assert `idempotency_conflict` and one total create attempt.
- [✓] AC-6: exact existing PR returns `externally_completed` — `internal/codehost/github_create.go:193` performs exact repository/ref lookup and `internal/codehost/github_create.go:336` requires a single fully satisfying match; the external-completion scenario asserts zero create attempts.
- [✓] AC-7: lost or cancelled applied writes reconcile to `reconciled_applied` — `internal/codehost/github_create.go:381` performs detached bounded read-back after ambiguous dispatch; `TestCreateReconcilesLostAndCancelledResponses` proves both malformed-response and post-apply cancellation paths with one attempt.
- [✓] AC-8: unprovable outcomes remain ambiguous and do not redispatch — `internal/codehost/github_create.go:394` persists ambiguous state and `internal/codehost/github_create.go:426` keeps dispatched/ambiguous replays in reconciliation; the ambiguous scenario repeats the same key and asserts `ambiguous_result` with one attempt.
- [✓] AC-9: cancellation respects the dispatch boundary — `internal/codehost/github_create.go:311` records pre-dispatch cancellation as `not_applied`, while post-dispatch uncertainty uses the detached recovery path; the cancellation tests assert zero effects before dispatch and reconciliation after an applied write.
- [✓] AC-10: journal is private, atomic, locked, and content/credential-free — `internal/codehost/idempotency.go:64` limits persisted target and receipt material, `internal/codehost/idempotency.go:93` enforces 0700/0600 plus file locking, and `internal/codehost/idempotency.go:143` uses the atomic writer; `assertJournalPrivateAndRedacted` checks permissions and title/body/credential canaries.
- [✓] AC-11: retention removes only expired terminal entries — `internal/codehost/idempotency.go:155` prunes only states classified terminal at `internal/codehost/idempotency.go:179`, and `internal/codehost/github_create.go:263` fails closed when retained records fill capacity; `TestCreateJournalCrashRecoveryRetentionAndRedaction` preserves year-old ambiguous/in-progress entries.
- [✓] AC-12: return the v1 mutation policy, receipt, revisions, rate metadata, bounds, and outcome — `internal/codehost/broker.go` maps mutation metadata into the common response and bounds mutation results; `TestCreatePullRequestSuccessAndContractConformance` validates the full response contract and asserts external-write consent, receipt, reconciliation, rate limits, journal count, and normalized outcome.

## Changes

- [✓] Add a durable mutation journal with bounded retention and locking — `internal/codehost/idempotency.go` adds the versioned digest-only document, private directory/file modes, shared file lock, atomic save, age pruning, and entry bound.
- [✓] Add creation validation and preflight observation — `internal/codehost/mutations.go:27` adds strict creation decoding/canonicalization, and `internal/codehost/github_create.go:40` adds provider-backed preparation and revision generation.
- [✓] Add exact existing-PR reconciliation — `internal/codehost/github_create.go:193` queries the owner-qualified base/head target and `internal/codehost/github_create.go:408` safely resolves replay, applied, and ambiguous journal states.
- [✓] Add one typed GitHub create and normalized receipt — `internal/codehost/github_create.go:223` emits one typed REST create request and validates the normalized result; `internal/codehost/mutations.go:106` constructs the v1 mutation result and receipt.
- [✓] Integrate cancellation and ambiguous recovery — `internal/codehost/github_create.go:311` handles pre-dispatch cancellation, and `internal/codehost/github_create.go:354` journals dispatch before the write and reconciles uncertain responses without cancellation inheritance.
- [✓] Extend the deterministic fake — `internal/mockcodehost/server.go` adds repository permission/ref reads, typed create handling, attempt accounting, external completion, lost/cancelled response, ambiguous read-back, fork, and stale-head controls.
- [✓] Add recovery, redaction, retention, and conformance tests — `internal/codehost/github_create_test.go` exercises success, validation, duplicate/concurrent/conflicting keys, recovery, cancellation, fork identity, crash states, redaction, permissions, retention, and contract validity.

## Audit notes

- None.
