# Delivery audit — personal-focus-core

**Audited:** `git diff HEAD -- . ':(exclude).hero/planning/initiatives/durable-attention/spec.md'`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: explicit add persists one private exact-prompt item — `internal/attention/focus/store.go:49-84,199-230`, `internal/attention/focus/service.go:57-65,152-179`; `TestStorePersistsPrivateItemWithContentRevision`, `TestFocusCLIEndToEnd`, and `TestFocusCLIAutoBindsRegisteredCurrentWorkspace` assert persistence, positive revision, `0600` mode, exact multiline prompt, requested state, and project binding.
- [✓] AC-2: lifecycle moves enforce the supplied revision and return the updated item — `internal/attention/focus/store.go:144-171`, `internal/attention/focus/service.go:120-138`; `TestServiceLifecycleReplayAndListing` and `TestFocusCLIEndToEnd` assert authoritative lifecycle/revision results and stale rejection.
- [✓] AC-3: replaying the same transition is idempotent — `internal/attention/focus/service.go:130-136`; `TestServiceLifecycleReplayAndListing` asserts unchanged revision and `updated_at`.
- [✓] AC-4: identical source-key replay returns the original item — `internal/attention/focus/store.go:87-121`, `internal/attention/focus/service.go:68-83`; `TestStoreCreateOrGetIsIdempotentAndDetectsConflict` and `TestServiceCreateOrGetRequiresSourceAndIsExact` assert stable identity without duplication.
- [✓] AC-5: changed payload under the same source key conflicts without mutation — `internal/attention/focus/store.go:101-107`; the store and service CreateOrGet tests assert `ErrIdempotencyConflict` and preservation of the original prompt.
- [✓] AC-6: missing and unbound projects remain visible and cannot fall back on launch — `internal/attention/focus/project.go:85-121`, `internal/attention/focus/service.go:140-149,182-187`; `TestServiceLaunchIntentRequiresResolvedProjectAndDoesNotMutate` and `TestFocusCLIUnboundItemStaysVisibleButCannotLaunch` assert missing visibility and structured launch failure.
- [✓] AC-7: resolvable launch returns exact prompt and absolute target without mutation — `internal/attention/focus/service.go:140-149`; service and Cobra end-to-end tests assert exact prompt/path/revision and unchanged lifecycle/revision.
- [✓] AC-8: concurrent mutations use exclusive compare-and-replace and atomic durable writes — `internal/attention/focus/lock.go:13-38`, `internal/attention/focus/store.go:144-171,199-234`; `TestStoreConcurrentReplaceDoesNotLoseUpdate` asserts exactly one success and one stale result across two stores, with the race test reported passing.
- [✓] AC-9: lists are deterministic, filtered, and hide done by default — `internal/attention/focus/store.go:174-196`, `internal/attention/focus/service.go:96-118`; `TestServiceListIsDeterministic`, `TestServiceLifecycleReplayAndListing`, and `TestFocusCLIEndToEnd` assert order, explicit done/all selection, and default exclusion.
- [✓] AC-10: Focus does not mutate adjacent work systems — the diff confines writes to the injected Focus state root and adds only Focus CLI/docs wiring; no spec, Intake, tracker, harness-task, scheduling, assignment, estimate, or priority integration is present. The full repository suite reportedly passes.

## Changes
- [✓] Add private atomic Focus store, lock, and tests — added `internal/attention/focus/store.go`, `lock.go`, and `store_test.go` with private permissions, canonical content revision, traversal rejection, atomic replacement, idempotency, and two-store concurrency assertions.
- [✓] Add service validation, lifecycle, provenance, resolution, and launch — added `internal/attention/focus/service.go` and `service_test.go` with boundary, lifecycle, replay, conflict, listing, missing-project, and non-mutating launch coverage.
- [✓] Add registry and peering identity adapter — added `internal/attention/focus/project.go` and `project_test.go`; peer ID is canonical across registry and configured-peer resolution, current registered workspaces auto-bind, and missing references carry no target path.
- [✓] Add and register complete Focus CLI — added `internal/cli/focus.go`, `focus_test.go`, and root registration for add/list/show/move/done/launch, stdin/file prompts, human/JSON output, current-workspace binding, stale errors, and safe launch behavior.
- [✓] Add Focus command/reference documentation and distinguish its role — added `web/docs/src/cli/focus.md` and the MkDocs navigation entry; the page documents every command, privacy/revision/launch semantics, auto-binding, and separation from specs, Intake, trackers, and harness todos.
- [✓] Test source-derived CreateOrGet seam — store and service tests require typed provenance/key, assert exact replay, assert conflict on changed payload, and verify original-item preservation.

## Audit notes
- No Completion Ledger rows were missing, partial, skipped, blocked, or downgraded.
- Reported validation passed: focused Focus package and CLI tests, race detector, `go vet`, full repository tests, strict MkDocs build, `git diff --check`, and 10/10 `hero drift` coverage. The audit reviewed the test bodies and did not rerun experiments.
