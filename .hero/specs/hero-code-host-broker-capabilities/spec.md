---
title: "Code Host Broker Capabilities — Hero-owned PR lifecycle boundary"
slug: hero-code-host-broker-capabilities
type: initiative
status: completed
domain: engineering
priority: critical
size: giant
autonomy: autonomous
created: 2026-07-27
child:
  - code-host-integration-capability-model
  - code-host-broker-v1-contract
  - github-code-host-read-adapter
  - github-pull-request-create-broker
  - github-pull-request-collaboration-broker
  - github-pull-request-state-transition-broker
  - github-pull-request-merge-broker
  - code-host-broker-surfaces-and-conformance
relates-to:
  - integration-config-uses-stable-ids
  - layered-integration-configuration
  - brokered-tracker-agent-access
  - pull-request-lifecycle-workbench
tags: [code-host, pull-request, broker, github, credentials, hero-code]
completed_at: 2026-07-28T05:01:07Z
---

# Code Host Broker Capabilities — Hero-owned PR lifecycle boundary

## Vision

Make code hosting a first-class Hero integration capability, parallel to but
distinct from issue tracking. Hero Desktop and model-driven clients can
discover, create, inspect, review, and merge pull requests through one
credential-safe, provider-neutral boundary. GitHub is the first implementation;
the contract does not assume that a workspace's delivery tracker is also its
code host.

The same stable integration connection and credential may serve multiple roles.
A GitHub or GitLab connection may be selected for both `delivery` and
`code-host`; Jira and Linear remain valid delivery trackers but cannot satisfy
the code-host role. This initiative adds no second credential store and returns
no secret to Swift, MCP, model context, argv, logs, fixtures, or persistence.

## Goal

Deliver `code-host-broker/v1` from configuration through released consumer
surfaces:

1. provider capability declarations and unambiguous `code-host` role selection;
2. a frozen repository-qualified PR contract and cross-language fixture;
3. bounded GitHub read operations with pagination, freshness, rate limits, and
   partial-result truth;
4. independently deliverable creation, collaboration, state-transition, and
   merge operations with operation-specific stale-state, idempotency, and
   ambiguous-outcome recovery; and
5. one in-process implementation surfaced through JSON CLI and typed MCP tools
   that Hero Code can validate against a released Hero binary.

## Kickoff

Build Hero's credential-safe code-host boundary for the Hero Code PR lifecycle
workbench, using the existing connection and broker foundations without
extending tracker issue APIs.

**Status:** planning — composition and all eight child designs are complete.

**Pick up at:** deliver the integration capability model first, then follow the
strict dependency chain through contract, reads, mutations, and surfaces.

→ `/drive hero-code-host-broker-capabilities`

**Files:** `internal/config/integrations.go`, `contracts/trackerbroker/contract.go`, `internal/tracker/broker.go`, `internal/serve/mcp_tools_def.go`
**Skip:** do not add PR operations to `internal/tracker`, reuse GitHub Projects issue DTOs, or introduce another credential store.

## Problem

Hero owns stable integration connections, layered local credentials, secret
redaction, and a proven `tracker-broker/v1` boundary. It does not own a
code-host domain. Hero Code therefore invokes `gh` directly for its narrow PR
creation path, inheriting ambient credentials and subprocess semantics while
remaining unable to list, review, reconcile, or safely merge PRs.

Putting PR operations into the tracker broker would encode a false product
relationship. GitHub and GitLab can provide both issues and pull/merge requests,
but Jira and Linear cannot provide repository hosting. A workspace commonly
tracks delivery in Jira while hosting code on GitHub. Provider identity,
integration connection, and advertised capability must therefore remain
separate concepts.

## Architecture

### One integration system, multiple capabilities

`integrations.connections` remains the only workspace connection and credential
model. A central provider declaration answers which semantic capabilities a
provider kind can satisfy. Each connection declares the subset it actually
serves through `capabilities`. `github` and `gitlab` may declare `tracker`,
`code-host`, or both; `jira` and `linear` may declare `tracker`; `confluence`
may declare `docs`. Existing connections with no field infer their legacy
capability (`tracker` for non-Confluence and `docs` for Confluence), so an
existing GitHub tracker does not silently become a code host. Role validation
uses connection capabilities, while runtime broker capabilities advertise only
operations implemented by the current binary.

`roles.code-host` selects the connection for code-host work. An explicit
`connection_id` may override it. Code-host resolution never falls back to
`roles.delivery`, `integrations.default`, provider name, ambient CLI login, or
map order.

### A separate code-host package and contract

PR domain types live in `contracts/codehostbroker` and implementation lives in
`internal/codehost`. The tracker broker remains issue-shaped under
`internal/tracker`; narrow generic security helpers may be extracted for reuse,
but tracker DTOs, search semantics, and GitHub Projects import types never enter
the code-host API.

### One execution path, three client surfaces

The in-process broker is authoritative. `hero code-host broker` and MCP tools
are thin adapters over it. Every surface returns the same
`code-host-broker/v1` envelope, capability policy, error codes, bounds,
freshness, pagination, rate-limit metadata, and reconciliation outcome.

### Cross-repo contract authority

Hero owns the canonical Go types, schemas, fixture bundle, and released-binary
emission. Hero Code owns its Swift decoder and UI state. The checked-in fixture
in Hero is the wire authority; the copied Swift test fixture must match it
byte-for-byte or by a published digest, and both repositories test unknown
additive fields.

## Sequenced specs

| Wave | Child | Priority | Size | Depends on | Outcome |
|---|---|---|---|---|---|
| 1 | `code-host-integration-capability-model` | critical | medium | — | `code-host` role, provider capability validation, shared credential resolution |
| 2 | `code-host-broker-v1-contract` | critical | large | capability model | Frozen identity, operation, policy, error, metadata, and fixture contract |
| 3 | `github-code-host-read-adapter` | high | large | v1 contract | In-process GitHub reads, fake adapter, pagination/freshness/rate-limit behavior |
| 4 | `github-pull-request-create-broker` | high | medium | read adapter | Exact repo/head/base creation with duplicate and ambiguous-response recovery |
| 5 | `github-pull-request-collaboration-broker` | high | medium | creation | Comment and review submissions with receipt-aware reconciliation |
| 6 | `github-pull-request-state-transition-broker` | high | medium | collaboration | Mark-ready, retarget, close, and reopen with fresh-state guards |
| 7 | `github-pull-request-merge-broker` | high | medium | state transitions | Merge/queue semantics guarded by head, checks, reviews, protection, and methods |
| 8 | `code-host-broker-surfaces-and-conformance` | high | medium | merge | JSON CLI, MCP tools, released fixture, docs, and Hero Code conformance handoff |

## Dependency order

```text
code-host-integration-capability-model
└── code-host-broker-v1-contract
    └── github-code-host-read-adapter
        └── github-pull-request-create-broker
            └── github-pull-request-collaboration-broker
                └── github-pull-request-state-transition-broker
                    └── github-pull-request-merge-broker
                        └── code-host-broker-surfaces-and-conformance
```

This order is intentionally serial. Adjacent implementation children touch the
same GitHub transport, broker dispatch, fixture inventory, fake provider, or
mutation/reconciliation core. The named seams carry reciprocal
`conflicts-with` relations so the `/drive` judge also protects them if a hard
dependency is edited later:

- `github-code-host-read-adapter` ↔ `github-pull-request-create-broker`:
  GitHub transport, observation revision, broker dispatch, and fake provider.
- `github-pull-request-create-broker` ↔
  `github-pull-request-collaboration-broker`: idempotency journal, mutation
  dispatch, receipts, and ambiguous-response recovery.
- `github-pull-request-collaboration-broker` ↔
  `github-pull-request-state-transition-broker`: mutation preconditions,
  permission policies, receipt/reconciliation state, and fake routes.
- `github-pull-request-state-transition-broker` ↔
  `github-pull-request-merge-broker`: fresh-state validation, capability
  revision, branch protection, mergeability, and dispatch.

## Cross-repo dependency order

1. Hero Code's `pull-request-domain-and-host-contract` may proceed against the
   designed field vocabulary and fixture path, but it must treat Hero's
   `code-host-broker/v1` fixture as the final wire authority.
2. Hero delivers `code-host-integration-capability-model`, then
   `code-host-broker-v1-contract`.
3. After the contract child lands, Hero Code may lock its Swift decoder and
   capability probe against the emitted fixture while Hero delivers the read
   and mutation children.
4. Hero delivers `code-host-broker-surfaces-and-conformance` and cuts a binary
   containing `hero code-host contract`.
5. Hero Code runs its decoder suite against that binary before completing its
   `hero-code-host-broker-capabilities` coordination child.
6. Hero Code's sync/index, creation, review, and merge children then consume the
   broker without direct GitHub credentials or `gh` fallback.

## Cross-cutting contracts

- Canonical PR identity always includes connection, repository, and provider PR
  identity; PR number alone is never sufficient.
- Every response carries authoritative operation policy, capability revision,
  observation/freshness state, bounds, and normalized error behavior.
- Every mutation requires explicit user intent, an idempotency key, a fresh
  observation revision, and operation-specific reconciliation material.
- Merge is classified as a commitment requiring explicit acceptance. Other PR
  mutations are external writes requiring an explicit user action.
- Cancellation before dispatch produces no effect. Cancellation or transport
  loss after dispatch enters ambiguous-outcome reconciliation and never causes
  a blind second provider write.
- Provider responses may be partial, stale, rate-limited, or unavailable; none
  of those states is translated to an empty list or a closed PR.
- All request, item, page, diff, text, error, duration, redirect, and journal
  retention bounds are explicit and testable.

## Acceptance Criteria

- **AC-1:** WHEN a workspace tracks delivery in Jira or Linear and binds `roles.code-host` to GitHub THE SYSTEM SHALL resolve PR operations only through the GitHub connection.
- **AC-2:** WHEN one GitHub or GitLab connection is bound to both `delivery` and `code-host` THE SYSTEM SHALL reuse the same stable connection and credential without duplicating either.
- **AC-3:** IF Jira, Linear, or Confluence is bound to `roles.code-host` THEN THE SYSTEM SHALL reject the configuration with the exact role, connection, provider, and supported capabilities.
- **AC-4:** THE SYSTEM SHALL expose one additive `code-host-broker/v1` contract across in-process, CLI, and MCP surfaces without importing tracker issue or GitHub Projects payloads.
- **AC-5:** WHEN the initiative is complete THE SYSTEM SHALL cover every required read and mutation operation through a fake GitHub adapter, shared contract fixture, and released-binary conformance exercise.
- **AC-6:** THE SYSTEM SHALL keep credentials inside Hero and exclude them from Swift, model context, argv, stdout/stderr, logs, errors, fixtures, persistence, and reconciliation artifacts.
- **AC-7:** WHEN a write is retried, stale, externally completed, partially failed, cancelled after dispatch, or returned ambiguously THE SYSTEM SHALL produce one typed replay/reconciliation outcome without blindly duplicating the provider effect.
- **AC-8:** WHEN Hero Code consumes the released fixture THE SYSTEM SHALL provide an exact digest and additive-decoding contract sufficient for the Swift decoder to reject incompatible versions and ignore unknown additive fields.

## Boundaries

- Do not extend `tracker-broker/v1`, the GitHub Issues adapter, or
  `GitHubProjectsClient` into PR lifecycle work.
- Do not add OAuth, Keychain storage, a second credential file, credential
  export, token-return APIs, or direct Swift credential access.
- Do not add a GitLab merge-request adapter in v1. GitLab is valid in the
  provider capability model, but runtime operations remain explicitly
  unavailable until its adapter ships.
- Do not add org-wide implicit repository enumeration, webhooks, background
  polling, local PR persistence, or Hero Code UI. Those belong to downstream
  Hero Code children.
- Do not make `gh` the production transport. Existing tracker CLI brokerage
  remains intact, but this code-host boundary uses the typed provider adapter.

## Risks

- A capability registry can become aspirational. Static provider-domain
  eligibility and runtime implemented operations must remain separate.
- GitHub REST and GraphQL expose different PR features. The adapter must return
  partial/unavailable metadata where permissions or APIs cannot prove
  readiness instead of inventing a normalized answer.
- GitHub does not provide native idempotency for most PR writes. The broker
  needs both a private replay journal and provider read-back reconciliation;
  either alone leaves a crash window.
- Fork heads, force pushes, repository transfers, dismissed reviews, branch
  protection, and merge queues invalidate branch-name or boolean models.
- MCP permission annotations are risk metadata, not authorization. The broker
  must validate user intent, required consent, revision, and idempotency at the
  service boundary too.

## Validation

- Lint every child and verify all parent/child/dependency/conflict relations.
- Test configuration and credential behavior with Jira-delivery/GitHub-host,
  shared GitHub dual-role, invalid Jira code-host, multiple GitHub connections,
  GitHub Enterprise origins, and missing credentials.
- Validate the contract and fixture in Go and through a Swift decoder supplied
  by Hero Code.
- Run fake-GitHub tests for every read, write, permission, pagination,
  rate-limit, fork, force-push, stale, duplicate, partial, cancellation,
  ambiguous, and externally-completed scenario.
- Run focused packages, `go test -race ./...`, `go vet ./...`,
  `hero docs check`, and the released-binary consumer exercise.

## Progress

Composition and all eight child designs are complete. No implementation has
started. Drive should begin with `code-host-integration-capability-model`.
