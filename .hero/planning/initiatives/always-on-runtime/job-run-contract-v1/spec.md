---
title: "Job and Run Contract v1"
slug: job-run-contract-v1
type: feature
status: planning
domain: engineering
size: medium
priority: P0
parent: always-on-runtime
created: 2026-07-13
tags: [contracts, runtime, jobs, events, compatibility]
relates-to: [hero-runner, hero-team-server, always-on-runner-authority]
---

# Job and Run Contract v1

## Goal

Publish a versioned, executor-neutral contract for submitting, observing and controlling Hero jobs so `hero serve`, the Go compatibility runner, hero-code workers, and Hero Cloud share one lifecycle vocabulary without sharing internal storage or inference implementations.

## Kickoff

Adds the stable v1 job/event boundary consumed by Hero, hero-code, and Hero Cloud.

**Status:** planning — design complete; implementation waits on the runner-authority decision.

**Pick up at:** define `contracts/runtime` types, transition validation, JSON Schema, and golden fixtures before adapting existing stores.

→ `.hero/planning/initiatives/always-on-runtime/job-run-contract-v1/spec.md`

**Files:** `contracts/version.go`, `contracts/contracts_boundary_test.go`, `internal/serve/jobs.go`, `internal/runner/runner.go`, `../hero-cloud/cloud/internal/seam_smoke.go`

## Problem

`internal/serve.Job` and `internal/runner.JobRecord` currently encode different slices of the same run. Status strings are duplicated, worker ownership is ephemeral, logs are not cursor-addressable, and clients cannot reliably reconnect or distinguish a retried submission from a new job. Extending either internal struct across repository boundaries would leak storage and executor decisions into every consumer.

## Approach

Create a new leaf package, `contracts/runtime`, following the existing `contracts/governance` and `contracts/peering` boundary pattern. It contains data shapes and pure validation only—no database, network, provider, shell, filesystem, or internal-package imports.

### Contract shapes

- `Version` and `CurrentVersion` with explicit compatibility validation.
- `JobRequest`: version, idempotency key, repository identity, workflow/command request, submitter, execution requirements, approval policy, budget limits and metadata.
- `JobSnapshot`: job identity, current state, attempt number, timestamps, target/worker references, last event sequence and terminal outcome reference.
- `JobEvent`: globally stable event ID, job ID, per-job monotonic sequence, attempt, timestamp, kind and versioned JSON payload.
- `JobOutcome`: terminal status, summary, artifact references, usage/cost and structured error—never raw credentials.
- `ControlMessage`: cancel, approve, reject and answer-input operations with an idempotency key and expected state/version.
- `WorkerCapability` and `ExecutionRequirement`: provider/tool/runtime labels and target constraints without endpoints or secrets.

Canonical states are `queued`, `leased`, `running`, `awaiting_input`, `awaiting_approval`, `completed`, `failed`, `cancelled`, `budget_exceeded`, `turn_limit`, and `worker_lost`. A pure transition table rejects illegal regressions and terminal-state overwrites.

### Compatibility

- JSON field names and enum values are the public v1 wire format.
- Readers ignore unknown additive fields but reject unsupported major versions.
- Golden JSON fixtures live under `contracts/runtime/testdata/v1/` and are consumed by Cloud and hero-code tests.
- A checked-in JSON Schema documents the contract for non-Go clients.
- `internal/serve` and `internal/runner` receive explicit translators; neither internal type becomes the wire contract.

### Migration

Use expand-contract sequencing:

1. Add the contract package and fixtures with no runtime behavior change.
2. Add translators and parity tests against current job/runner records.
3. Make APIs emit v1 alongside current response shapes where compatibility requires it.
4. Migrate consumers, observe, then retire duplicate lifecycle semantics in separate specs.

## Changes

1. Add `contracts/runtime/version.go`, `job.go`, `event.go`, `state.go`, `control.go`, and `worker.go`.
   - Keep the package dependency-free beyond the standard library.
   - Add constructors/validators for required identifiers, versions, sequence values and legal transitions.
2. Add `contracts/runtime/schema/job-run-v1.schema.json` and golden fixtures under `contracts/runtime/testdata/v1/`.
   - Include active, approval/input, successful, failed, cancelled and unknown-additive-field fixtures.
3. Extend `contracts/contracts_boundary_test.go` to forbid imports from `internal/`, provider, database, HTTP and execution packages.
4. Add translation helpers and tests beside `internal/serve/jobs.go` and `internal/runner/runner.go`.
   - Preserve current storage and behavior; prove lossless translation for all representable fields.
   - Surface any current state that cannot map cleanly instead of silently coercing it.
5. Extend the hero-cloud seam canary to compile against representative runtime contract symbols once the sibling pin includes this change.
6. Document versioning and additive evolution in `docs/contracts/job-run-v1.md`.

## Acceptance Criteria

- THE SYSTEM SHALL publish versioned `JobRequest`, `JobSnapshot`, `JobEvent`, `JobOutcome`, `ControlMessage`, `WorkerCapability`, and `ExecutionRequirement` shapes in `contracts/runtime`.
- THE SYSTEM SHALL define one canonical set of job states including queued, leased, running, awaiting-input, awaiting-approval, completed, failed, cancelled, budget-exceeded, turn-limit, and worker-lost.
- WHEN a state transition is requested THE SYSTEM SHALL accept only transitions present in the v1 transition table and reject terminal-state overwrites.
- WHEN the same submission idempotency key is reused for the same repository and workflow identity THE SYSTEM SHALL allow consumers to resolve it as the same logical job.
- THE SYSTEM SHALL assign every lifecycle event a stable event ID and a strictly increasing per-job sequence number.
- WHEN a v1 reader receives unknown additive JSON fields THE SYSTEM SHALL preserve compatibility rather than reject the payload.
- IF a reader receives an unsupported major contract version THEN THE SYSTEM SHALL return a typed compatibility error.
- THE SYSTEM SHALL represent worker and target requirements without embedding credentials, tokens, provider keys, filesystem secrets, or transport endpoints.
- WHEN existing `serve.Job` and `runner.JobRecord` values are translated THE SYSTEM SHALL preserve every representable lifecycle, usage, timing, result and error field.
- IF an existing internal state cannot map to v1 THEN THE SYSTEM SHALL fail the translation explicitly and identify the unmapped value.
- THE SYSTEM SHALL check in a JSON Schema and golden v1 fixtures usable by Go, Rust, Swift, and Cloud consumer tests.
- THE SYSTEM SHALL keep `contracts/runtime` free of imports from internal runtime, storage, transport, provider, filesystem and tool-execution packages.

## Boundaries

- No queue or database implementation.
- No worker registration transport, leases, heartbeat loop or retry executor.
- No scheduler or trigger implementation.
- No UI, SSE, webhook or notification adapter.
- No inference, provider selection, tool execution, transcript format or credential broker.
- No removal of current internal job structs in this change.

## Risks

- Encoding current implementation quirks would freeze the wrong abstraction. Mitigation: contract only portable lifecycle facts and translate explicitly.
- Swift/Rust consumers can drift from Go. Mitigation: schema plus shared golden fixtures are mandatory compatibility tests.
- Additive evolution can still break strict decoders. Mitigation: fixtures include unknown fields and consumer guidance requires tolerant decoding.
- Dual-shape rollout can create ambiguity. Mitigation: emit explicit contract version and document the retirement gate for legacy responses.

## Validation

- Run `go test ./contracts/...` including boundary, schema, fixture and transition-table tests.
- Run focused translator tests in `internal/serve` and `internal/runner`.
- Run `go test ./...` to catch import cycles and cross-package regressions.
- In hero-cloud, compile and test the updated cross-repo seam against the pinned Hero revision.
- Validate every golden fixture against the checked-in JSON Schema and round-trip it without semantic loss.

