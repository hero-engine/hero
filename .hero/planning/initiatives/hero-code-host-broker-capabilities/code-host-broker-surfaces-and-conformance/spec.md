---
title: "Code-host broker surfaces and conformance"
slug: code-host-broker-surfaces-and-conformance
type: feature
status: delivering
domain: engineering
priority: high
size: medium
created: 2026-07-27
parent: hero-code-host-broker-capabilities
depends-on:
  - github-pull-request-merge-broker
relates-to:
  - pull-request-lifecycle-workbench
tags: [code-host, cli, mcp, fixtures, conformance, hero-code]
delivery_method: manual
---

# Code-host broker surfaces and conformance

## Context

The in-process broker owns the contract and GitHub behavior, but Hero Code and
chat-driven model loops need stable process and MCP surfaces. Existing Hero
broker commands demonstrate bounded JSON stdin/stdout, while Attention tools
demonstrate static permission annotations and service-side intent checks. A
single generic MCP “broker operation” tool cannot truthfully advertise one
read/write/destructive policy for operations ranging from list to merge.

Hero Code also needs a deterministic way to prove its Swift decoder matches the
released Hero binary rather than a hand-copied design draft.

## Goal

Expose the same `code-host-broker/v1` implementation through a bounded JSON
CLI and operation-specific MCP tools, publish the canonical fixture and digest,
test transport parity and permission metadata, and hand Hero Code an exact
released-binary conformance procedure.

## Kickoff

Finish the Hero-owned boundary by making every operation safely callable from
Hero Desktop and model chat loops without returning credentials or weakening
service-side policy.

**Status:** planning — CLI broker, MCP registry/dispatch, permission
annotations, request-context, fixture, and Hero Code decoder paths are mapped.

**Pick up at:** add the CLI contract/dispatch adapters, then generate the MCP
inventory from the authoritative operation-policy registry.

→ `/deliver code-host-broker-surfaces-and-conformance`

**Files:** `internal/cli/code_host_broker.go`, `internal/serve/mcp_tools_code_host.go`, `contracts/codehostbroker/HERO-CODE-HANDOFF.md`
**Skip:** do not expose a token, arbitrary authenticated request, generic mixed-effect MCP tool, or `gh` fallback.

## Problem

If CLI and MCP each translate requests independently, they will drift on
effects, consent, cancellation, pagination, errors, bounds, and output shapes.
If the desktop launches a command that depends on ambient PATH credentials, it
recreates the direct-GitHub problem. If a generic MCP tool advertises read-only
metadata but accepts merge, client permission UI becomes misleading even when
the service rejects misuse.

## Approach

Keep the in-process broker authoritative. Add:

- `hero code-host contract`, which emits the exact canonical v1 fixture bundle,
  operation policies, bounds, and SHA-256 digest without resolving credentials;
- `hero code-host broker <operation>`, which accepts exactly one bounded JSON
  request on stdin, writes exactly one v1 JSON response to stdout, sends
  diagnostics only to bounded stderr, and never accepts credentials or bodies
  in argv.

Mutation-specific CLI and MCP surfaces expose an explicit non-mutating prepare
phase. Preparation calls the operation's existing provider preflight and
returns a bounded v1 prepared-request envelope containing the exact capability
and observation revisions that the user or client is accepting. Execution is a
separate call with that prepared request. Surfaces never silently prepare and
execute in one step, because doing so would make freshness checks and explicit
acceptance performative. The mutation tool retains its conservative maximum
write or commitment annotation even when invoked in prepare mode.

Both commands use process signals to cancel the broker context and return the
same typed cancellation/reconciliation state. No CLI path invokes `gh`, emits
provider raw bodies, or introduces a second implementation.

Expose one MCP tool per v1 operation so its immutable annotations can match the
authoritative policy. Tool definitions and dispatch inventory are generated or
validated against the operation registry:

- ten read/discovery tools are read-only, non-destructive, and require no
  consent;
- nine non-merge mutation tools are external writes requiring explicit user
  intent;
- merge is a commitment/destructive external effect requiring explicit
  acceptance.

Every MCP request uses the caller request context, enforces the same bounds and
contract validation, and calls the same broker method as CLI. The service
requires `intent_source: user`, revisions, idempotency, and consent material
even if a client ignores annotations. Tool results return bounded structured
content and no duplicated prose payload.

Hero remains fixture authority at
`contracts/codehostbroker/testdata/v1/consumer-fixture.json`. Add
`contracts/codehostbroker/HERO-CODE-HANDOFF.md` with the fixture digest,
supported binary/version probe, copy/update procedure, Swift fixture
destination, compatibility rules, and exact conformance commands. Hero Code's
copy at
`packages/hero-swift/Tests/HeroSharedApplicationTests/Fixtures/CodeHost/code-host-broker-v1.json`
must match the published digest and test unknown additive fields.

## Changes

1. Register the `hero code-host` CLI group with `contract` and bounded
   `broker <operation>` commands.
2. Add one thin CLI adapter over the in-process broker.
   - Read bounded stdin, reject trailing/multiple payloads, validate the
     requested operation against the path, and emit one response.
   - Add `--prepare` for mutation operations only; return the typed prepared
     request and require a separate execution call with those exact revisions.
   - Propagate signals/context cancellation and preserve post-dispatch
     reconciliation.
   - Keep credentials and mutation content out of argv and diagnostics.
3. Add twenty operation-specific MCP definitions and handlers.
   - Derive or exhaustively validate names, schemas, effects, consent,
     read-only/destructive/idempotent/open-world annotations, and dispatch
     against the v1 registry.
   - Accept `prepare: true` only on mutation-specific tools and return the same
     prepared-request envelope as CLI without provider mutation.
   - Use request-scoped cancellation and the same service policy checks.
4. Add transport parity tests.
   - Run every success, partial, error, stale, rate-limited, cancelled, replay,
     externally completed, and ambiguous fixture case through in-process, CLI,
     and MCP.
   - Deep-compare canonical response JSON after transport-only metadata is
     removed.
5. Add credential/body canaries through argv, stdin, MCP input, stdout, stderr,
   logs, errors, receipts, and fixture generation.
6. Publish the fixture, digest, contract docs index, and Hero Code handoff
   procedure.
7. Build a release-shaped Hero binary and run `code-host contract` plus a fake
   broker exercise against it so packaging/registration gaps are caught.

## Acceptance Criteria

- **AC-1:** WHEN `hero code-host contract` runs THE SYSTEM SHALL emit the canonical `code-host-broker/v1` fixture, operation policies, bounds, and published digest without resolving a credential.
- **AC-2:** WHEN `hero code-host broker <operation>` runs THE SYSTEM SHALL accept one bounded JSON request from stdin, reject operation/path mismatch or trailing payloads, support explicit `--prepare` for mutations, and emit exactly one bounded v1 response.
- **AC-3:** IF argv contains a token, authorization header, mutation body, review text, or credential material THEN THE SYSTEM SHALL reject the CLI invocation before broker dispatch.
- **AC-4:** THE SYSTEM SHALL expose exactly one MCP tool for each of the twenty v1 operations, allow `prepare: true` only on mutation-specific tools, and expose no generic tool capable of changing effect class at runtime.
- **AC-5:** THE SYSTEM SHALL match every MCP tool's input schema, effect, consent, read-only/destructive/idempotent/open-world annotations, preparation behavior, and handler dispatch to the authoritative operation-policy registry.
- **AC-6:** WHEN a mutation reaches CLI or MCP without user intent, required consent, idempotency key, capability revision, or observation revision THE SYSTEM SHALL either return a non-mutating prepared-request envelope when explicitly requested or reject it before provider dispatch; execution SHALL never prepare implicitly even if client permission checks were bypassed.
- **AC-7:** WHEN a CLI signal or MCP request context is cancelled THE SYSTEM SHALL propagate cancellation to the broker and preserve pre-dispatch no-effect versus post-dispatch reconciliation semantics.
- **AC-8:** WHEN any canonical fixture case runs through in-process, CLI, and MCP surfaces THE SYSTEM SHALL produce semantically identical bounded envelopes and normalized errors.
- **AC-9:** WHEN output, diagnostics, tool content, error detail, or input exceeds its declared bound THE SYSTEM SHALL return the contract's typed bound error or truncation metadata without emitting unbounded provider data.
- **AC-10:** THE SYSTEM SHALL keep credentials out of Swift, MCP results, model context, argv, stdout, stderr, logs, errors, fixtures, receipts, and reconciliation artifacts under canary tests.
- **AC-11:** THE SYSTEM SHALL publish a deterministic fixture and SHA-256 digest plus an exact Hero Code copy/decode procedure and additive-field compatibility test.
- **AC-12:** WHEN a release-shaped Hero binary is built THE SYSTEM SHALL successfully run its contract emission and representative fake read/write/reconciliation operations through the installed command registry.
- **AC-13:** THE SYSTEM SHALL document that Jira/Linear tracker selection is independent from GitHub code-host selection and that one explicitly dual-capability GitHub connection may serve both roles.
- **AC-14:** THE SYSTEM SHALL leave the existing tracker broker, GitHub Projects importer, and issue mock behavior unchanged while adding the separate code-host surfaces.

## Boundaries

- No Hero Code Swift implementation, UI, local PR cache, sync scheduler, or
  decoder changes in this repository.
- No credentials returned by `contract`, broker CLI, MCP, fixtures, or docs.
- No arbitrary authenticated HTTP, GraphQL, shell, `gh`, or provider-payload
  passthrough tool.
- No additional code-host behavior beyond adapting the completed in-process
  broker.

## Risks

- Twenty MCP tools expand the public inventory. Registry-derived conformance
  tests are necessary to prevent permission and schema drift.
- MCP annotations vary by client and are not authorization. Broker validation
  remains the security boundary.
- Fixture copies can drift across repositories. Digest checks and
  released-binary emission make drift visible but require Hero Code CI to
  enforce them.
- CLI stderr can accidentally reveal provider errors. Normalize and bound
  diagnostics before writing them.

## Validation

- Inventory-test all twenty operations across registry, contract, CLI command,
  MCP definition, handler dispatch, and fixture coverage.
- Prove every mutation prepares without provider writes and executes only the
  exact separately submitted prepared request; prove reads reject prepare mode.
- Deep-compare in-process, CLI, and MCP cases for every operation and normalized
  error/reconciliation state.
- Send cancellation before and after dispatch through CLI signals and MCP
  request contexts.
- Run credential/body canaries and maximum-input/output cases.
- Build a release-shaped binary and execute `hero code-host contract` plus
  representative fake operations.
- Run `go test ./internal/cli ./internal/serve ./internal/codehost/...`,
  `go test ./...`, and `go vet ./...`.

## Completion Ledger

Delivered the public `code-host-broker/v1` process and MCP boundary over the
existing in-process broker. Mutation preparation is explicit, non-mutating,
and shared by both transports. The canonical fixture digest is
`9159b224be3c9f1dee9072b2135e01364c91c2d97800d98ee554a997d4d7f6ff`.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Contract command emits fixture, registry, bounds, and digest without credentials | DONE | `internal/cli/code_host_broker.go:49`, `TestCodeHostContractEmitsFixturePoliciesBoundsAndDigestWithoutWorkspace` — exact fixture string and SHA are emitted without a workspace |
| 2 | Bounded exact-one CLI request/response with explicit mutation prepare | DONE | `internal/cli/code_host_broker.go:83`, `TestCodeHostBrokerRejectsMismatchTrailingUnknownAndOversizedInputBeforeProvider`, `TestCodeHostBrokerPrepareInputFailuresUsePreparationEnvelope` |
| 3 | Reject token/header/body/review content in argv before dispatch | DONE | `TestCodeHostBrokerRejectsArgvContentBeforeBrokerConstruction` proves positional and flag canaries never construct the broker or echo values |
| 4 | Exactly twenty operation-specific MCP tools, mutation prepare only, no generic tool | DONE | `internal/serve/mcp_tools_code_host.go:33`, `TestCodeHostMCPInventoryAndPoliciesMatchRegistry` |
| 5 | MCP schemas, annotations, metadata, and dispatch match registry | DONE | `internal/serve/mcp_tools_code_host.go:84`, `internal/serve/mcp_dispatch.go:92`, `TestCodeHostMCPNestedSchemasAreClosedAndOperationSpecific` |
| 6 | Missing mutation policy material rejects or explicitly prepares without a write | DONE | `internal/codehost/prepare.go:10`, `contracts/codehostbroker/validate.go:277`, CLI/MCP explicit-prepare tests, and existing broker policy tests |
| 7 | CLI and MCP propagate cancellation and retain broker reconciliation semantics | DONE | CLI uses `cmd.Context()`; MCP uses `s.ctx`; `TestCodeHostBrokerPrepareRejectsReadsAndContextCancellationPropagates` and `TestCodeHostMCPUsesServerCancellationContext` pass |
| 8 | In-process, CLI, and MCP transports preserve canonical successes and errors | DONE | all twenty cases and every canonical normalized error pass `TestCodeHostBrokerMatchesEveryCanonicalFixtureCaseWithoutTranslation`, `TestCodeHostBrokerPreservesEveryCanonicalError`, `TestCodeHostMCPCanonicalFixtureParityAllOperations`, and `TestCodeHostMCPCanonicalErrorParity` |
| 9 | Bounds return typed safe errors without unbounded output | DONE | 1 MiB transport readers, contract validators, `TestCodeHostMCPInputBoundReturnsTypedSafeEnvelope`, and CLI oversized-input coverage |
| 10 | Credentials remain outside argv, output, diagnostics, MCP content, fixtures, and receipts | DONE | CLI and MCP credential/body canaries pass; fixture mutation text remains `[redacted]`; provider credential tests remain green |
| 11 | Deterministic fixture/digest and exact Hero Code decoder handoff | DONE | `contracts/codehostbroker/fixture.go:122`, generated fixture/digest, `contracts/codehostbroker/HERO-CODE-HANDOFF.md`, and contract tests |
| 12 | Release-shaped binary exercises registration, read, prepared write, and reconciliation | DONE | `TestReleasedShapeHeroBinaryExercisesContractReadAndPreparedWrite` builds `cmd/hero` and runs contract/capabilities/comment against the fake GitHub adapter |
| 13 | Tracker and code-host role independence is documented | DONE | `docs/contracts/code-host-broker-v1.md` and the Hero Code handoff explicitly separate Jira/Linear tracker selection from GitHub code hosting and allow explicit dual-capability GitHub |
| 14 | Tracker broker, Projects importer, and issue mocks remain unchanged | DONE | code-host commands and tools are separately registered; repository-wide tests pass with no tracker/importer/mock implementation changes |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Register `hero code-host` contract and broker commands | DONE | `internal/cli/root.go`, `internal/cli/code_host_broker.go` |
| 2 | Add bounded CLI adapter and explicit prepare flow | DONE | exact-one decoder, path binding, typed errors, `Broker.Prepare`, cancellation, and argv rejection are covered in `internal/cli/code_host_broker_test.go` |
| 3 | Add twenty operation-specific MCP definitions and handlers | DONE | registry-derived inventory, closed nested schemas, conservative annotations, forced operation identity, and shared broker live in `internal/serve/mcp_tools_code_host.go` |
| 4 | Add transport parity tests | DONE | every operation, reconciliation status, partial/truncated response, and normalized error is preserved through CLI and MCP |
| 5 | Add credential/body canaries across public surfaces | DONE | CLI argv/stdout and MCP input/content bounds and non-reflection tests pass alongside provider-boundary canaries |
| 6 | Publish fixture, digest, docs index, and Hero Code handoff | DONE | generated fixture SHA, contract reference, docs index, and `HERO-CODE-HANDOFF.md` are synchronized |
| 7 | Exercise a release-shaped binary | DONE | compiled binary contract/read/prepare/write/reconciliation test passes |

### Exercise-the-feature check

- [x] Built the real `cmd/hero` binary and exercised `hero code-host contract`,
  a fake GitHub capabilities read, explicit comment preparation, separate
  comment execution, and the returned reconciliation envelope.

### Validation evidence

- `go test ./contracts/codehostbroker ./internal/codehost ./internal/cli ./internal/serve -count=1`
- `go test ./... -count=1`
- `go test -race ./contracts/codehostbroker ./internal/codehost ./internal/cli ./internal/serve -run 'TestCodeHost|TestReleasedShapeHeroBinary' -count=1`
- `go vet ./...`
- `go generate ./contracts/codehostbroker`
- `hero spec lint code-host-broker-surfaces-and-conformance`
- `git diff --check`

### Excellence Bar self-check

Yes. The implementation keeps one credential boundary and one operation
registry, makes permission metadata truthful per operation, binds mutation
execution to an explicit preparation step, preserves the contract byte-for-byte
through CLI and MCP, and proves the released command registry instead of only
testing package internals.
