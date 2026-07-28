# Delivery audit — code-host-broker-surfaces-and-conformance

**Audited:** `git diff e313f0ac9ea3b8400dfd3548323cd4d0ce1c35c3..ff6624d`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — `hero code-host contract` emits the embedded fixture, all twenty ordered policies, shared bounds, and a verified digest without loading a workspace or credential — `internal/cli/code_host_broker.go:49`, `TestCodeHostContractEmitsFixturePoliciesBoundsAndDigestWithoutWorkspace`
- [✓] AC-2 — the broker command binds the path operation, decodes exactly one bounded object, rejects trailing or unknown input, supports mutation-only preparation, and emits one typed envelope — `internal/cli/code_host_broker.go:83`, `TestCodeHostBrokerRejectsMismatchTrailingUnknownAndOversizedInputBeforeProvider`, `TestCodeHostBrokerPrepareInputFailuresUsePreparationEnvelope`
- [✓] AC-3 — token, header, body, review, and positional canaries are rejected before broker construction and are not reflected in output or diagnostics — `TestCodeHostBrokerRejectsArgvContentBeforeBrokerConstruction`
- [✓] AC-4 — the authoritative registry produces exactly twenty operation-specific MCP tools, only mutation tools accept preparation, and no generic mixed-effect tool is registered — `internal/serve/mcp_tools_code_host.go:34`, `TestCodeHostMCPInventoryAndPoliciesMatchRegistry`
- [✓] AC-5 — operation identity, closed nested schemas, effects, consent, and MCP annotations are derived from or exhaustively checked against the operation registry; dispatch fixes the operation from the tool name — `internal/serve/mcp_tools_code_host.go:94`, `internal/serve/mcp_dispatch.go:109`, MCP inventory/schema tests
- [✓] AC-6 — preparation is an explicit non-writing call that returns only revisions, while execution retains broker-side intent, consent, idempotency, capability, observation, and reconciliation validation — `internal/codehost/prepare.go:9`, `TestCodeHostBrokerPrepareThenExecuteIsExplicitAndDoesNotEchoPayload`, `TestCodeHostMCPPreparationModeIsExplicitAndClosed`
- [✓] AC-7 — CLI uses the command context; MCP registers a context per JSON-RPC request ID, processes `notifications/cancelled` while the provider call is in flight, serializes concurrent responses, and cancels/waits for outstanding calls on EOF — `internal/serve/mcp_lifecycle.go:110`, `internal/serve/mcp_lifecycle.go:197`, `internal/serve/mcp_lifecycle.go:337`, `TestCodeHostMCPRequestCancellationNotificationCancelsExactCall`; existing broker tests cover pre- and post-dispatch reconciliation
- [✓] AC-8 — all twenty canonical success envelopes, including stale/partial/truncated and all reconciliation states, plus every normalized error, are preserved through CLI and MCP adapters — CLI and MCP canonical parity tests; `TestCanonicalFixtureCoversAndValidatesEveryOperation`
- [✓] AC-9 — arguments retain a 1 MiB bound inside a finite 2 MiB JSON-RPC envelope, the real stdio loop returns typed `input_too_large`, and provider/output/error bounds remain enforced by the shared contract and broker — `internal/serve/mcp_lifecycle.go:21`, `TestCodeHostMCPRunReturnsTypedErrorBeyondArgumentBound`, contract and broker bound tests
- [✓] AC-10 — credentials remain inside the existing Hero connection resolver; argv, CLI output, MCP content, fixture, provider errors, receipts, and debug logs have canary coverage, and debug logging records only method/status/byte metadata — `internal/serve/mcp_lifecycle.go:98`, `internal/serve/mcp_lifecycle.go:337`, `TestCodeHostMCPDebugLogDoesNotRecordArgumentsOrResults`
- [✓] AC-11 — fixture bytes are generator-stable at SHA-256 `e66a8d5643dce518db66a5e20b2a39be1ac5766b464f2f1244c05ff6a8b43edb`; the handoff gives the exact Hero Code destination, newline-safe copy/hash commands, additive compatibility expectations, and exact Swift test command — `contracts/codehostbroker/HERO-CODE-HANDOFF.md:141`, fixture determinism and unknown-field tests
- [✓] AC-12 — a release-shaped `cmd/hero` binary exercises command registration, contract emission, a fake capabilities read, explicit comment preparation/execution, and applied reconciliation — `TestReleasedShapeHeroBinaryExercisesContractReadAndPreparedWrite`
- [✓] AC-13 — the contract separates Jira/Linear tracker selection from GitHub code-host selection and permits one explicitly dual-capability GitHub connection — `docs/contracts/code-host-broker-v1.md:9`
- [✓] AC-14 — the delivery diff adds separate code-host commands/tools without modifying tracker broker, GitHub Projects importer, or issue-mock implementation; the full repository suite passes — audited name-status diff and `go test ./... -count=1`

## Changes

- [✓] Register `hero code-host contract` and `broker <operation>` — `internal/cli/root.go:143`, `internal/cli/code_host_broker.go:40`
- [✓] Add the bounded CLI adapter and explicit prepare/execute flow — `internal/cli/code_host_broker.go:83`, `internal/codehost/prepare.go:9`, CLI command and binary tests
- [✓] Add twenty operation-specific MCP definitions and handlers — `internal/serve/mcp_tools_code_host.go:34`, `internal/serve/mcp_tools_def.go:735`, request-context dispatch in `internal/serve/mcp_dispatch.go:109`
- [✓] Add transport parity tests — all operation fixtures, normalized errors, stale state, preparation, request cancellation, and real stdio bounds are exercised in CLI/MCP tests
- [✓] Add credential/body canaries across public surfaces — CLI argv/stdout, MCP content/debug log, fixture, provider errors, and reconciliation artifacts are covered
- [✓] Publish fixture, digest, docs index, and Hero Code handoff — generated `testdata/v1` artifacts, `docs/contracts/README.md`, contract reference, and executable handoff procedure agree on the current digest
- [✓] Exercise a release-shaped binary — `TestReleasedShapeHeroBinaryExercisesContractReadAndPreparedWrite` and the cold-audit binary contract/hash exercise both pass

## Audit notes

- Focused tests, race-focused code-host tests, the full repository suite, `go vet ./...`, generator stability, and `git diff --check` passed.
- The release-built contract emitted version `code-host-broker/v1`, twenty policies, and fixture digest `e66a8d5643dce518db66a5e20b2a39be1ac5766b464f2f1244c05ff6a8b43edb`; the documented newline-safe extraction reproduced that digest exactly.
