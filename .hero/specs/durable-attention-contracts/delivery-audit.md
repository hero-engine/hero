# Delivery audit — durable-attention-contracts

**Audited:** `git diff 0816a56...HEAD`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: portable v1 records use opaque IDs, UTC timestamps, typed references, and no paths — DTOs in `contracts/attention/{mail,focus,suggestion,projection}.go`; UTC enforcement and schema exercise in `contracts/attention/validate.go:275` and `TestExactUTCTimestampsAcrossRecordsAndSchemas`.
- [✓] AC-2: additive fields and raw future source/action/style values remain compatible — `TestForwardCompatibleRawValues` decodes `testdata/v1/unknown-fields.json` and asserts the raw values.
- [✓] AC-3: invalid writes return stable structured errors before storage exists — validators in `contracts/attention/validate.go:19-208`; `TestValidationContract` and `TestEveryWriteContractHasBoundaryValidation` assert codes and fields.
- [✓] AC-4: state-root precedence, prohibited locations, and private permissions — `internal/attention/state/root.go:26-90`; `TestResolvePrecedenceAndLocations`, `TestResolveRejectsRepositoryAndSensitiveRoots`, and `TestEnsurePrivatePermissionsAndSeparateStores`.
- [✓] AC-5: action descriptors expose raw ID, display metadata, input schema, revision precondition, and idempotency — `contracts/attention/action.go:14-30`; descriptor fixture coverage through `TestGoldenFixturesSchemasDTOsAndChecksums`.
- [✓] AC-6: action results carry authoritative success forms or one of six stable errors — `contracts/attention/action.go:5-11,43-55`, `ValidateActionResult`, and `TestActionResultRequiresExactlyOneOutcome`.
- [✓] AC-7: all eight manifest fixtures validate against executable schemas, decode to DTOs, and match SHA-256 — `TestGoldenFixturesSchemasDTOsAndChecksums`.
- [✓] AC-8: one global HTTP boundary is documented and streaming is optional — `web/docs/src/cli/server-and-mcp.md:149-159`.
- [✓] AC-9: Mail and Focus have distinct mutation/lifecycle DTOs while sharing only the projection — `contracts/attention/mail.go`, `focus.go`, and `projection.go`; no generic mutable endpoint is documented at `web/docs/src/cli/server-and-mcp.md:151-158`.
- [✓] AC-10: attention contracts remain leaf-only — standard-library imports in `contracts/attention`; repository boundary enforcement in `contracts/contracts_boundary_test.go` and passing `go test ./contracts/...` evidence.

## Changes
- [✓] Add schema version and compatibility policy — `contracts/attention/version.go`.
- [✓] Add separate Mail, Focus, suggestion, and projection DTOs — `contracts/attention/{mail,focus,suggestion,projection}.go`.
- [✓] Add action/result/navigation/error contracts — `contracts/attention/action.go`.
- [✓] Add structural boundary validation — `contracts/attention/validate.go`.
- [✓] Add versioned schemas, fixtures, and checksum manifest — `contracts/attention/schema/v1/` and `contracts/attention/testdata/v1/`.
- [✓] Enforce leaf boundary, schema parity, fixture checksums, and forward decoding — `contracts/contracts_boundary_test.go` and `contracts/attention/contract_test.go`.
- [✓] Add shared private state-root resolver and tests — `internal/attention/state/root.go` and `root_test.go`.
- [✓] Document the desktop HTTP boundary — `web/docs/src/cli/server-and-mcp.md:149-159`.

## Open items

None.

## Audit notes

None. The supplied focused tests pass; the repository-wide suite passes with one unrelated non-hermetic live-session-registry test excluded. `hero drift durable-attention-contracts --since 0816a56 --format json` reports all 10 criteria addressed with no signals.
