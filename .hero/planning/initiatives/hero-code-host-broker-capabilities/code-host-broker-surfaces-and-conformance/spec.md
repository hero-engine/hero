---
title: "Code-host broker surfaces and conformance"
slug: code-host-broker-surfaces-and-conformance
type: feature
status: planning
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
   - Propagate signals/context cancellation and preserve post-dispatch
     reconciliation.
   - Keep credentials and mutation content out of argv and diagnostics.
3. Add twenty operation-specific MCP definitions and handlers.
   - Derive or exhaustively validate names, schemas, effects, consent,
     read-only/destructive/idempotent/open-world annotations, and dispatch
     against the v1 registry.
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
- **AC-2:** WHEN `hero code-host broker <operation>` runs THE SYSTEM SHALL accept one bounded JSON request from stdin, reject operation/path mismatch or trailing payloads, and emit exactly one v1 JSON response.
- **AC-3:** IF argv contains a token, authorization header, mutation body, review text, or credential material THEN THE SYSTEM SHALL reject the CLI invocation before broker dispatch.
- **AC-4:** THE SYSTEM SHALL expose exactly one MCP tool for each of the twenty v1 operations and no generic tool capable of changing effect class at runtime.
- **AC-5:** THE SYSTEM SHALL match every MCP tool's input schema, effect, consent, read-only/destructive/idempotent/open-world annotations, and handler dispatch to the authoritative operation-policy registry.
- **AC-6:** WHEN a mutation reaches CLI or MCP without user intent, required consent, idempotency key, capability revision, or observation revision THE SYSTEM SHALL reject it before provider dispatch even if client permission checks were bypassed.
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
- Deep-compare in-process, CLI, and MCP cases for every operation and normalized
  error/reconciliation state.
- Send cancellation before and after dispatch through CLI signals and MCP
  request contexts.
- Run credential/body canaries and maximum-input/output cases.
- Build a release-shaped binary and execute `hero code-host contract` plus
  representative fake operations.
- Run `go test ./internal/cli ./internal/serve ./internal/codehost/...`,
  `go test ./...`, and `go vet ./...`.
