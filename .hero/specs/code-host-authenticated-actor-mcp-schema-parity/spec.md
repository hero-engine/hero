---
title: "Authenticated-actor MCP schema incorrectly requires a pull request"
slug: code-host-authenticated-actor-mcp-schema-parity
type: bug
status: completed
diagnosis_status: diagnosed
domain: engineering
priority: high
severity: high
size: small
root_cause_class: code
created: 2026-07-30
tags: [code-host, mcp, contract, release-blocker]
relations:
  - target: code-host-broker-surfaces-and-conformance
    kind: related
delivery_method: manual
completed_at: 2026-07-30T17:41:44Z
---

# Authenticated-actor MCP schema incorrectly requires a pull request

## Kickoff

Removes a false pull-request requirement from the repository-scoped
`get_authenticated_actor` MCP tool so the v2 contract fixture and runtime agree.

**Status:** delivering — implementation, cold audit, and release-prerequisite
validation pass; the Hero verification gate is next.

**Pick up at:** run the Hero verification gate.

→ `/deliver code-host-authenticated-actor-mcp-schema-parity`

**Files:** `internal/serve/mcp_tools_code_host.go`, `internal/serve/mcp_tools_code_host_test.go`
**Skip:** do not add a synthetic pull request or change the v2 fixture; actor identity is connection/repository scoped.

## Summary

The `v0.31.0` release candidate fails
`TestCodeHostMCPCanonicalFixtureParityAllOperations` for
`get_authenticated_actor`. The contract and canonical fixture correctly model
authenticated actor lookup as a repository-scoped read. The MCP input-schema
switch does not name the new operation, so it falls into the generic
pull-request operation branch and adds a required `pull_request` field.

The canonical fixture supplies the valid repository-only request. MCP rejects
it at schema validation and returns a typed input error instead of dispatching
the fixture response, producing the parity failure. GoReleaser runs
`go test ./...` in its `before` hook, making this a hard release blocker.

## Root Cause

`codeHostMCPInputSchema` groups operation shapes with a switch. Capabilities,
collections, paged PR children, create, and generic PR-scoped operations have
separate branches. `OperationGetAuthenticatedActor` was added to the contract,
policy registry, adapter, broker, and fixture, but not to the MCP schema
switch. The default branch therefore applies the wrong identity requirement.

This is an incomplete consumer update, not a contract or credential defect.
The handler dispatch, broker, GitHub adapter, and v2 fixture are correct.

## Evidence

1. `go test ./...` passes every package except `internal/serve`.
2. The focused failure is:
   `TestCodeHostMCPCanonicalFixtureParityAllOperations:
   get_authenticated_actor fixture parity mismatch`.
3. The v2 fixture request contains version, operation, provider, connection,
   and repository, with no pull request.
4. `codeHostMCPInputSchema` has no actor case and its default branch appends
   `pull_request` for every operation except create.
5. The Jira attachment delivery diff does not touch `internal/serve`,
   `internal/codehost`, or `contracts/codehostbroker`; the regression exists at
   the release baseline.

## Goal

Restore exact contract parity for authenticated-actor lookup without changing
the provider-neutral v2 contract, fixture, broker, or credential boundary.

## Acceptance Criteria

- **AC-1:** WHEN Hero advertises the `get_authenticated_actor` MCP tool THE SYSTEM SHALL require only the shared version, provider, connection, and repository identity fields and SHALL NOT require a pull-request identity.
- **AC-2:** WHEN a canonical repository-only authenticated-actor request reaches MCP THE SYSTEM SHALL dispatch the semantically identical broker request and return the canonical response without an input error.
- **AC-3:** THE SYSTEM SHALL preserve pull-request identity requirements for every PR-scoped read or mutation and SHALL preserve the existing create, collection, and capability operation shapes.
- **AC-4:** WHEN the release candidate is qualified THE SYSTEM SHALL pass the focused MCP contract tests and GoReleaser's repository-wide `go test ./...` prerequisite.

## Suggested Fix Approach

1. Add `OperationGetAuthenticatedActor` to the same repository-only branch as
   capabilities in `codeHostMCPInputSchema`.
2. Add focused schema assertions proving authenticated actor omits
   `pull_request` while a representative PR-scoped read still requires it.
3. Retain the canonical all-operation fixture-parity test as the end-to-end
   dispatch and response regression.
4. Run focused `internal/serve` tests, the complete package, repository-wide
   tests/vet, and a GoReleaser snapshot before tagging.

## Changes

1. Update `internal/serve/mcp_tools_code_host.go` to classify
   `get_authenticated_actor` as repository-only.
2. Extend `internal/serve/mcp_tools_code_host_test.go` with explicit
   operation-shape regression coverage.
3. Qualify the release with focused, package, repository, and snapshot builds.

## Boundaries

- Do not change `code-host-broker/v2`, canonical fixtures, or operation policy.
- Do not synthesize a pull request for actor lookup.
- Do not change credential resolution, return credentials, or broaden MCP
  output.
- Do not touch harness installation surfaces.

## Validation

- `go test ./internal/serve -run 'TestCodeHostMCP.*(Schema|FixtureParity)' -count=1`
- `go test ./internal/serve -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `goreleaser release --snapshot --clean`
- `hero spec lint code-host-authenticated-actor-mcp-schema-parity`

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Actor MCP schema is repository-only | DONE | `codeHostMCPInputSchema` now has an explicit actor branch; `TestCodeHostMCPAuthenticatedActorIsRepositoryScoped` verifies the exact required fields and absence of `pull_request` |
| 2 | Canonical actor fixture dispatches without input error | DONE | `TestCodeHostMCPCanonicalFixtureParityAllOperations` passes with the repository-only canonical request and response |
| 3 | PR-scoped and other operation shapes remain intact | DONE | The regression proves `get_pull_request` still requires `pull_request`; the complete MCP inventory and repository suite pass |
| 4 | Focused and repository-wide release test prerequisite passes | DONE | Focused actor/parity tests, `go test ./... -count=1`, and `go vet ./...` pass |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Classify actor lookup as repository-only | DONE | Added the explicit `OperationGetAuthenticatedActor` switch branch without changing capabilities or collection properties |
| 2 | Add operation-shape regression coverage | DONE | Added focused actor-versus-PR assertions in `internal/serve/mcp_tools_code_host_test.go` |
| 3 | Qualify focused, repository, and snapshot builds | DONE | Focused canonical parity, full repository tests/static analysis, and an exact-commit GoReleaser snapshot pass; the snapshot built all configured Darwin, Linux, and Windows targets plus package-manager metadata |

### Exercise-the-feature check

- [x] The canonical repository-only actor request was exercised through the
  existing all-operation MCP fixture parity test and returned the canonical
  broker response.

### Excellence Bar self-check

- [x] Yes — the fix is operation-specific, preserves every contract and
  credential boundary, and adds a direct regression at the schema seam that
  caused the release failure.
