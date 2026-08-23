---
title: "Project Mail thread lifecycle contract is HTTP-only and unavailable to bundled clients"
slug: project-mail-thread-lifecycle-mcp-parity
type: bug
status: completed
domain: engineering
size: medium
priority: critical
severity: high
root_cause_class: design
created: 2026-08-23
relations:
  - target: mail-b7ca19966ac5041e6ff604dd
    kind: related
  - target: mail-da2727fd11615a9cafa5125c
    kind: related
tags: [mail, attention, mcp, contract, desktop]
completed_at: 2026-08-23T16:22:35Z
---

# Project Mail thread lifecycle contract is HTTP-only and unavailable to bundled clients

## Summary

### Categorization

| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — the bundled desktop transport cannot use any delivered Project Mail thread lifecycle operation; the HTTP workaround is unavailable/forbidden for the embedded client, but there is no data loss, security breach, or whole-service outage |
| **Ease of Fix** | moderate — the authority already exists, but four MCP schemas/handlers, exact typed-error parity, tool metadata, tests, and the generated Attention MCP inventory must move together |
| **Caused by our codebase?** | Yes — Hero's MCP registry and dispatcher omit the already-delivered thread surface |
| **Needs more research?** | No — the missing definitions/handlers and the authoritative service boundary are both directly visible in this branch |

### Background

The completed `mail-b7ca19966ac5041e6ff604dd` and
`mail-da2727fd11615a9cafa5125c` deliveries added source-owned lifecycle state,
thread projection, pagination, and HTTP endpoints. The bundled desktop client,
however, launches Hero as an MCP child rather than a localhost HTTP daemon. The
MCP surface still exposes only legacy message-oriented Mail tools, so the
desktop cannot consume the new authoritative thread model.

### Analysis

`mailquery.Service` already owns all required list, detail, and action behavior,
including registry resolution, composite identity, classification, counts,
cursor validation, revision checks, idempotency, and structured contract
errors. HTTP is a thin adapter over that service. MCP has no equivalent adapter:
its tool inventory and handler map stop at legacy single-project message reads
and triage actions.

### Root Cause

The lifecycle/projection design treated Hero Serve HTTP as the only typed
consumer transport and omitted MCP parity from its acceptance and qualification
surface. This is classified as `design`: the feature interaction between the
new lifecycle contract and the embedded MCP-only client was not included, so
the implementation correctly delivered the planned HTTP adapter while leaving
the actual bundled-client path incapable of using it.

### Source

The authority is in `contracts/attention/mailthread` and
`internal/attention/mailquery`. The HTTP adapter is in
`internal/serve/api_attention_mail.go`. The missing transport surface is the MCP
definition/dispatch/handler path in `internal/serve/mcp_tools_def.go`,
`internal/serve/mcp_dispatch.go`, and `internal/serve/mcp_tools_mail.go`.

### Fix Direction

Add four MCP-only transport adapters that decode the already-published argument
shapes, obtain the registry-backed `mailquery.Service`, invoke its existing
methods, and marshal its existing versioned response envelopes unchanged. Do
not reconstruct threads, classify buckets, generate cursors, translate lifecycle
policy, or route through localhost HTTP.

## Goal

Expose authoritative Project Mail thread list, detail, lifecycle action, and
contract negotiation through bundled Hero MCP so embedded clients can consume
the delivered lifecycle model without starting `hero serve`, using localhost
HTTP, or reimplementing any policy locally.

## Kickoff

Adds the missing MCP transport for Hero-owned Project Mail thread lifecycle operations.

**Status:** planning — diagnosis confirmed the registry and dispatch gap; no source fix has landed.

**Pick up at:** add the four exact tool definitions and direct handlers over `mailquery.Service`, then prove HTTP/MCP envelope parity and regenerate the Attention MCP inventory.

→ `/deliver project-mail-thread-lifecycle-mcp-parity`

**Files:** `internal/serve/mcp_tools_def.go`, `internal/serve/mcp_dispatch.go`, `internal/serve/mcp.go`, `internal/attention/mailquery/service.go`, `contracts/attention/mailthread/contract.go`
**Skip:** do not call localhost HTTP, flatten the portable action request, or duplicate lifecycle/projection policy in MCP.

## Problem Statement

### Observed behavior

The HTTP server exposes `GET /api/attention/v1/mail/threads`,
`GET /api/attention/v1/mail/threads/{thread_id}`,
`POST /api/attention/v1/mail/thread-actions`, and
`GET /api/attention/v1/mail/thread-contract`. In the same build,
`MCPServer.toolDefinitions()` and `MCPServer.toolHandlers()` expose only the
legacy `hero_mail_list`, `hero_mail_show`, `hero_mail_send`,
`hero_mail_reply`, and `hero_mail_action` message tools. There is no MCP name a
client can call for thread buckets, composite-identity detail, authoritative
cursors, lifecycle state/actions, or thread-contract identity.

The legacy tools are not substitutes. `toolMailList` calls a project-local
`mail.Service.Inbox` and returns message bodies plus legacy receipt/triage
actions. It does not call the registry-backed `mailquery.Service.Threads`, and
therefore cannot return one row per `(project_peer_id, thread_id)`, lifecycle
buckets, thread counts, paging cursors, or thread revisions.

### Expected behavior

The bundled MCP server advertises and dispatches four additive thread tools
whose outputs are the exact `mailthread` response envelopes already returned by
HTTP. Existing message tools remain byte/behavior compatible.

### Minimal reproduction

1. Construct `serve.NewMCPServer(...)` on this branch and inspect
   `toolDefinitions()` or the MCP `tools/list` response.
2. Observe that no `hero_mail_thread_*` tool is advertised; the current test at
   `internal/serve/mcp_tools_mail_test.go:238` checks only the five legacy Mail
   tools.
3. Inspect `toolHandlers()` and observe the same absence, so a direct
   `tools/call` for any thread operation resolves as an unknown tool.
4. Compare with `API.Handler()` at `internal/serve/api.go:129-140`, where all
   four HTTP thread routes are registered and working.

## Environment Details

- Branch: `codex/project-mail-lifecycle`
- Investigated commit: `4d5aa4c1380064969bd130eb8ec0e90ff60d046e`
- Active Hero domain: `engineering`
- Go implementation; no tracker ID or external runtime evidence is required.
- The failure is transport-dependent: `hero serve` HTTP has the capability;
  bundled MCP does not.

## Root Cause Analysis

### Confirmed observations

1. `contracts/attention/mailthread/contract.go:144-245` defines the versioned
   list, detail, action, and contract response types used by consumers.
2. `internal/attention/mailquery/thread_projection.go:24-102` implements
   authoritative list filtering, classification, counts, deterministic order,
   and stale cursor rejection; `:104-128` implements exact thread detail.
3. `internal/attention/mailquery/service.go:35-53` validates and delegates
   thread actions to the source-owned Mail service, preserving structured
   validation, missing, unavailable, stale, and idempotency errors.
4. `internal/serve/api_attention_mail.go:15-84` is a thin HTTP adapter over
   those methods and the immutable `mailthread.ContractResponse` identity.
5. `internal/serve/mcp_tools_def.go:55-119` declares only legacy message Mail
   tools; no thread tool exists.
6. `internal/serve/mcp_dispatch.go:26-101` maps only those legacy names; no
   thread handler is callable.
7. `internal/serve/mcp.go:34-69` has an Attention projection factory seam but
   no `mailquery.Service` factory seam for MCP, while `internal/serve/api.go:50-62`
   explicitly carries the registry-backed Mail query factory/interface.
8. `internal/serve/mcp_tools_mail.go:137-188` proves the legacy list/show/action
   path calls the single-project `mail.Service`, not `mailquery.Service`.

### Load-bearing claim list

| Claim | Grounding |
|-------|-----------|
| HTTP thread lifecycle is implemented | **read** — `api.go`, `api_attention_mail.go`, and their tests |
| Thread authority already exists outside the transport | **read** — `mailquery.Service`, `mail.Service.ThreadAction`, and contract validators |
| MCP cannot advertise or dispatch thread operations | **read** — exhaustive literal definition and handler registries |
| Legacy Mail MCP tools cannot provide equivalent semantics | **read** — their concrete service calls and response structs differ |
| The smallest correct fix is an MCP adapter over `mailquery.Service` | **read/inference** — this exactly mirrors the working HTTP sibling and introduces no new authority |

### Why this is definitive

The failure occurs before lifecycle code can execute: MCP has neither a tool
definition nor a dispatcher entry for these operations. No data shape, race,
environment variable, or source-state condition can make an unadvertised and
unregistered tool callable. The working HTTP sibling already proves the
service methods and envelopes exist, narrowing the fault to the MCP adapter.

## MCP Tool Contract

These names and arguments are fixed so Hero, the desktop consumer, and this
spec remain aligned. Inputs preserve the portable contract's nesting. The MCP
wire advertises `thread_revision` as a decimal int64 string because MCP
arguments decode through `float64`; the handler parses it with the existing
`int64Arg` helper and constructs the exact numeric `mailthread.ActionRequest`
internally without losing 63-bit revision precision.

| Tool | Arguments | Delegation | Exact result |
|------|-----------|------------|--------------|
| `hero_mail_thread_list` | optional `project_peer_id`; optional `bucket` (`needs_attention`, `updates`, `history`); optional `lifecycle` (`open`, `resolved`, `archived`); optional integer `limit` (`0..100`); optional `cursor` | construct `mailthread.ThreadListRequest` with `SchemaVersion: 1`, then call `mailquery.Service.Threads(...)` | `mailthread.ThreadListResponse` |
| `hero_mail_thread_show` | required `project_peer_id`; required `thread_id` | `mailquery.Service.ThreadDetail(project_peer_id, thread_id)` | `mailthread.ThreadDetailResponse` |
| `hero_mail_thread_action` | required numeric `schema_version: 1`; required nested `identity: {project_peer_id, thread_id}`; required `action_id`; required decimal int64 string `thread_revision`; required `idempotency_key`; optional object `input` | parse `thread_revision` with `int64Arg`, then call `mailquery.Service.ThreadAction(mailthread.ActionRequest)` | `mailthread.ActionResponse` |
| `hero_mail_thread_contract` | `{}` | construct the same immutable identity as the HTTP handler from `mailthread.SchemaVersion`, `BundleVersion`, `ConformanceManifestSHA256`, and `Compatibility` | `mailthread.ContractResponse` |

`input` is passed as JSON to the existing validator. `resolve` requires
`reason` and `source` and may carry `outcome`, `source_id`, and
`grace_class`; other actions accept no input fields. The MCP tool schema must
not duplicate an action-ID enum: callers dispatch only descriptors advertised
by the authoritative thread response, and `mailthread.ValidateActionRequest`
remains the closed-world executable policy.

## Code Flow (End to End)

1. `internal/serve/mcp_lifecycle.go` receives MCP `tools/list` and `tools/call`.
2. `internal/serve/mcp_tools_def.go:14-119` builds the advertised Attention/Mail
   inventory. Today it stops at legacy message tools, which is the first
   divergence from the working HTTP path.
3. `internal/serve/mcp_dispatch.go:26-101` resolves a tool name to a handler.
   Today a `hero_mail_thread_*` call has no mapping and cannot reach domain code.
4. The fix's MCP handlers decode only transport arguments and obtain the same
   registry-backed service boundary used by HTTP, using the MCP server's private
   Attention state root plus `projectregistry.Load()` and
   `mailquery.NewService(...)`.
5. `internal/attention/mailquery/thread_projection.go:24-102` performs thread
   list aggregation/filtering/counts/cursor logic, or `:104-128` performs exact
   composite-identity detail.
6. For mutations, `internal/attention/mailquery/service.go:35-53` validates the
   portable request, resolves the authoritative project source, and delegates
   to `internal/attention/mail/thread.go:229-334` for revisioned/idempotent
   receipt and lifecycle mutation.
7. The handler marshals the returned `mailthread.ThreadListResponse`,
   `ThreadDetailResponse`, `ActionResponse`, or `ContractResponse` directly;
   product failures remain structured envelopes, not MCP transport errors.
8. `internal/attention/conformance/builder.go:112-167` derives the published
   Attention `mcp-tools.json` inventory from runtime definitions, so adding the
   tools requires regenerating that bundle and updating its compiled manifest
   hash. The independent Mail-thread lifecycle checksum does not change merely
   because a new transport adapter exists.

## Key Files

### Authoritative contract and services

| File | Lines | Relevance |
|------|-------|-----------|
| `contracts/attention/mailthread/contract.go` | 144-245 | Portable list/detail/action/contract request and response types |
| `contracts/attention/mailthread/validate.go` | 216-440 | Existing closed-world validation and structured compatibility rules |
| `internal/attention/mailquery/thread_projection.go` | 24-128 | Authoritative list and detail implementation |
| `internal/attention/mailquery/service.go` | 35-53, 299-367 | Authoritative action delegation and registry source resolution |
| `internal/attention/mail/thread.go` | 229-365 | Source-owned thread mutation and advertised capabilities |

### Working HTTP adapter

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/api.go` | 50-62, 129-140 | Mail query service seam and four registered routes |
| `internal/serve/api_attention_mail.go` | 15-84 | Thin list/detail/action/contract HTTP adapter to mirror |
| `internal/serve/api_attention_mail_test.go` | 15-320 | Existing endpoint, error, contract, and public-surface coverage |

### Missing MCP adapter and generated inventory

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_tools_def.go` | 14-119, 938-953 | Canonical tool definitions and generated Attention/Mail inventory selector |
| `internal/serve/mcp_dispatch.go` | 26-101 | Tool-name-to-handler map |
| `internal/serve/mcp.go` | 34-69 | MCP service/factory seams |
| `internal/serve/mcp_tools_mail.go` | 115-188 | Legacy local Mail implementation; useful compatibility pattern but not the thread authority |
| `internal/attention/conformance/builder.go` | 112-167 | Regenerates pinned `mcp-tools.json`, manifest, and handoff hash |

## Secondary Defects

The portable `action-request.schema.json` in both
`contracts/attention/mailthread/schema/v1/` and the checked-in conformance copy
enumerates only `resolve`, `reopen`, `archive`, and `restore`, while
`ValidateActionRequest`, `ValidateCapabilitySet`, `ThreadCapabilities`, and the
runtime action service also accept/advertise `mark_read` and `mark_unread`.
This is confirmed schema/runtime drift. It is deliberately outside this minimum
transport-parity fix: runtime capability descriptors and validators remain the
authority, and the new MCP action tool schema must not copy that incomplete
enum. Correct the portable schema separately unless exact MCP dispatch tests
prove it blocks this adapter.

## Notes

The anchor check found no applicable tripwire. This is not a harness-specific
instruction change; it is a server capability shared by every MCP client. The
fix must preserve MCP annotations accurately: list/show/contract are read-only;
thread action is a revisioned state write requiring the advertised action and
explicit user consent.

## Recap

Project Mail thread lifecycle is functional behind HTTP but unreachable from
the MCP-only bundled client because the MCP definition and dispatcher registries
never added the thread surface. Add four thin adapters over
`mailquery.Service`, preserve the exact portable envelopes, and regenerate the
published MCP inventory; do not create a second lifecycle authority.

## Suggested Fix Approach

### 1. Advertise the four exact tools

**File/block:** `internal/serve/mcp_tools_def.go`, Attention/Mail definitions in
`MCPServer.toolDefinitions`.

**Before (current source):**

```go
{
    Name:        "hero_mail_list",
    Category:    CategoryAttentionAndMail,
    Tier:        TierDeferrable,
    Description: "List Project Mail for this workspace with receipt state and advertised triage actions.",
    InputSchema: InputSchema{Type: "object", Properties: map[string]PropSchema{
        "unread": {Type: "boolean", Description: "Return unread messages only"},
    }},
},
{
    Name:        "hero_mail_show",
    Category:    CategoryAttentionAndMail,
    Tier:        TierDeferrable,
    Description: "Read one Project Mail message in this workspace without mutating receipt state.",
    InputSchema: InputSchema{Type: "object", Properties: map[string]PropSchema{
        "message_id": {Type: "string", Description: "Stable Mail message ID"},
    }, Required: []string{"message_id"}},
},
```

No thread definitions follow these legacy tools.

**After:** add `hero_mail_thread_list`, `hero_mail_thread_show`,
`hero_mail_thread_action`, and `hero_mail_thread_contract` with the exact schemas
in `## MCP Tool Contract`. Close nested objects where supported, keep action ID
open at the transport schema, and set explicit MCP metadata so reads are marked
read-only while the action is a replay-safe state write.

**Why:** `tools/list` is the model/client capability contract. A handler alone
is unreachable and an incorrectly annotated read can be withheld or over-gated
by conforming clients.

### 2. Register and implement thin handlers

**Files/blocks:** `internal/serve/mcp_dispatch.go` in `toolHandlers`;
`internal/serve/mcp.go` MCP service seams; the existing
`internal/serve/mcp_tools_mail.go` handler file.

**Before (current source):**

```go
"hero_mail_list":          s.toolMailList,
"hero_mail_show":          s.toolMailShow,
"hero_attention_contract": s.toolAttentionContract,
// ...
"hero_mail_send":          s.toolMailSend,
"hero_mail_reply":         s.toolMailReply,
"hero_mail_action":        s.toolMailAction,
```

`MCPServer` has no registry-backed Mail query factory:

```go
attentionStateRoot string
attentionResolver  focus.ProjectResolver
attentionService   func() (*projection.Service, error)
```

**After:** register all four exact names, add a testable lazy
`mailquery.Service` factory seam parallel to `attentionService`, and implement
handlers with this shape:

```go
service, err := s.mailThreadQueryService()
if err != nil {
    return marshalMailThread(unavailableMailThreadResponse(err))
}
return marshalMailThread(service.Threads(request)) // or ThreadDetail / ThreadAction
```

The contract handler returns the same constants used at
`api_attention_mail.go:84`. The list handler pins `SchemaVersion: 1` internally,
matching the HTTP GET adapter rather than advertising a redundant list argument.
Action decoding must preserve its required numeric `schema_version`, parse
decimal-string `thread_revision` losslessly, and keep nested `identity`/`input`;
malformed shapes return a structured
`attention.ContractError`, and service-level failures remain
inside the existing versioned response rather than as an MCP protocol failure.

**Why:** this follows the working HTTP sibling structurally and keeps registry
resolution, cursor logic, classification, revisions, idempotency, and lifecycle
rules in their current authorities.

### 3. Pin the additive runtime inventory and guidance

**Files/blocks:** generated `contracts/attention/conformance/v1/mcp-tools.json`,
`manifest.json`, and `HERO-CODE-HANDOFF.md`;
`internal/serve/api_attention.go` compiled Attention bundle hash;
`contracts/attention/mailthread/testdata/v1/HERO-CODE-HANDOFF.md` plus its
generated copy only for the transport-name mapping; `docs/serve.md` for the
embedded transport statement.

**Before (current source/artifacts):** the generated Attention tool inventory
contains only legacy `hero_mail_list` and `hero_mail_show` reads, and
`docs/serve.md:52-56` describes legacy MCP as compatibility-only while directing
Hero Code to typed HTTP.

**After:** regenerate the Attention bundle from
`serve.AttentionToolDefinitions()`, update only its derived manifest/hash, and
document the exact four tool-to-contract mappings. Keep
`mailthread.ConformanceManifestSHA256` unchanged unless an actual portable
contract artifact changes; do not hand-invent derived hashes.

**Why:** the checked-in inventory is a build-enforced consumer contract. Adding
runtime definitions without regeneration makes the conformance suite fail and
leaves consumers pinned to an incomplete surface.

### 4. Add focused regression and parity coverage

**Files/blocks:** focused additions to `internal/serve/mcp_tools_mail_test.go`;
inventory expectations in that file and
`internal/serve/mcp_test.go`; conformance tests already driven by runtime tool
definitions.

**Before (current source):**

```go
for _, name := range []string{
    "hero_mail_list", "hero_mail_show", "hero_mail_send",
    "hero_mail_reply", "hero_mail_action",
} {
    // assert definition + handler
}
```

There is no test that performs a thread call through MCP or compares its JSON
with the HTTP/service envelope.

**After:** cover all four definitions and direct dispatcher calls; compare
decoded MCP results with direct `mailquery.Service`/HTTP results from the same
state for current list/detail/contract and action success/error cases; assert
stale cursor, stale revision, idempotency replay/conflict, invalid nested input,
unavailable registry/state, annotations, and legacy-tool compatibility.

**Why:** the original regression existed because qualification never asserted
the actual bundled transport. Direct MCP dispatch plus cross-transport envelope
parity closes that gap.

## Acceptance Criteria

- **AC-1:** WHEN `hero_mail_thread_list` receives a valid portable list request THE SYSTEM SHALL return the authoritative `mailthread.ThreadListResponse` with exact buckets, counts, paging cursor, and structured errors from `mailquery.Service`.
- **AC-2:** WHEN `hero_mail_thread_show` receives one exact `(project_peer_id, thread_id)` identity THE SYSTEM SHALL return the authoritative `mailthread.ThreadDetailResponse` without reconstructing thread state in the transport layer.
- **AC-3:** WHEN `hero_mail_thread_action` receives the existing nested `mailthread.ActionRequest` shape for an advertised action THE SYSTEM SHALL enforce the authoritative revision, idempotency, and closed-world validation contract and return `mailthread.ActionResponse`.
- **AC-4:** WHEN `hero_mail_thread_contract` is called THE SYSTEM SHALL return the same `mailthread.ContractResponse` identity exposed over HTTP.
- **AC-5:** THE SYSTEM SHALL advertise and dispatch exactly `hero_mail_thread_list`, `hero_mail_thread_show`, `hero_mail_thread_action`, and `hero_mail_thread_contract` with accurate read/write annotations.
- **AC-6:** THE SYSTEM SHALL preserve the five legacy message-level Mail tools and their existing request/response behavior.
- **AC-7:** THE SYSTEM SHALL prove direct MCP dispatch and decoded HTTP/MCP envelope parity for list, show, action, contract, paging, validation, stale, idempotency, and unavailable paths.
- **AC-8:** THE SYSTEM SHALL regenerate the published Attention MCP tool inventory and its derived manifest/hash without changing the independent Mail-thread lifecycle checksum unless a portable contract artifact changes.

## Changes

1. Add the four exact definitions in `internal/serve/mcp_tools_def.go` and mappings in `internal/serve/mcp_dispatch.go`.
2. Add a lazy/testable registry-backed Mail query service seam in `internal/serve/mcp.go` and thin handlers in the existing `internal/serve/mcp_tools_mail.go`.
3. Add focused definition, metadata, direct-dispatch, lifecycle, paging, error, compatibility, and HTTP/MCP parity coverage in `internal/serve/mcp_tools_mail_test.go` and `internal/serve/mcp_test.go`.
4. Regenerate `contracts/attention/conformance/v1/mcp-tools.json`, `contracts/attention/conformance/v1/manifest.json`, and `contracts/attention/conformance/v1/HERO-CODE-HANDOFF.md`; update the compiled Attention bundle hash and embedded-transport guidance required by those generated mappings.

## Boundaries

- Do not move lifecycle policy, thread aggregation, cursor generation, or receipt semantics into the MCP layer.
- Do not remove or change existing HTTP endpoints or legacy Mail MCP tools.
- Do not require `hero serve` for embedded/bundled clients.
- Do not flatten `identity`/`input`, expose `thread_revision` as an MCP JSON number, or invent lifecycle policy in a second consumer DTO; the decimal string is only a precision-safe wire adapter to the existing numeric contract field.
- Do not modify the portable lifecycle action enum/schema drift in this minimum transport fix unless an exact dispatch test proves it blocks MCP parity.

## Risks

- Hand-written schemas can drift from the contract types; tests must bind decoded MCP output to the current versioned envelopes.
- Action argument translation must preserve exact revisions and stable idempotency keys.
- Prefix-based MCP metadata defaults would mark new Mail reads as generic advertised actions unless the definitions explicitly carry correct read-only annotations.
- Regenerating the main Attention tool inventory changes its derived bundle hash; failing to update the compiled HTTP/MCP Attention contract identity will break conformance tests.
- The confirmed portable action-request schema enumeration drift can confuse schema-only consumers, but changing it here would broaden a bounded transport fix and alter the independent contract bundle.

## Test Plan

### Existing test review

| Test file | Existing coverage |
|-----------|-------------------|
| `contracts/attention/mailthread/contract_test.go` | lifecycle/read orthogonality and strict action validation |
| `contracts/attention/mailthread/conformance_test.go` | pinned list/detail/action/contract fixtures and bundle determinism |
| `internal/attention/mailquery/service_test.go` | registry-backed Mail query behavior and structured failures |
| `internal/attention/mailquery/thread_projection_test.go` | buckets, counts, paging, stale cursors, composite identity, and detail |
| `internal/serve/api_attention_mail_test.go` | working HTTP endpoints, contract discovery, and public route behavior |
| `internal/serve/mcp_tools_mail_test.go` | legacy Mail tool shapes and current definition/handler inventory |
| `internal/serve/mcp_test.go` | complete advertised MCP tool-name guard |
| `internal/attention/conformance/builder_test.go` | runtime-derived MCP inventory and checked-in bundle parity |

### Test changes needed

1. Extend `internal/serve/mcp_tools_mail_test.go` with a registry/state fixture
   shared by direct `mailquery.Service`, HTTP, and MCP paths.
2. Assert exact input schemas, required fields, the absence of a list
   `schema_version` argument, action's required numeric `schema_version`, nested
   `identity`, decimal-string `thread_revision`, action input object preservation,
   and read/write annotations for all four tools.
3. Invoke handlers through `toolHandlers()` rather than only calling helper
   methods; decode and validate every returned `mailthread` envelope.
4. Compare HTTP and MCP decoded responses for identical list, show, action, and
   contract requests. JSON whitespace/status codes may differ; typed envelope
   content may not.
5. Cover stale cursor, stale thread revision, same-key replay,
   idempotency conflict, missing project/thread, malformed nested input,
   incompatible schema version, and unavailable registry/state root.
6. Extend global inventory guards and regenerate the canonical Attention bundle;
   run its deterministic checked-in bundle test.
7. Re-run legacy Mail MCP tests unchanged to prove additive compatibility.

### Regression scope

- MCP tool filtering/progressive disclosure and annotations.
- Legacy message Mail tools and Attention snapshot/action behavior.
- HTTP Mail thread endpoints and contract discovery.
- Registry-backed multi-project identity and private state-root resolution.
- Generated Attention conformance inventory/hash and downstream consumers.

## Validation

Run focused checks first, then the repository gates:

```sh
go test -race ./contracts/attention/mailthread ./internal/attention/mail ./internal/attention/mailquery ./internal/attention/conformance ./internal/serve
go test ./...
go vet ./...
git diff --check
```

Manual protocol check: launch the built `hero mcp`, perform MCP initialize and
`tools/list`, assert all four exact names and annotations, then call each tool
against a temporary registered two-project Mail fixture without starting
`hero serve` or connecting to localhost.

## Delivery Validation

- `go test ./internal/serve -run 'TestMCPMail(Thread|Definitions)' -count=1` — passed.
- `go test ./internal/serve -count=1` — passed.
- `go test -race ./internal/serve ./internal/attention/conformance ./contracts/attention/mailread -count=1` — passed.
- `go run ./cmd/attention-conformance --check` — passed with manifest `2c29a1c6e04c3504969736494d0759f566a982d01c59ab7d8552c751a64b31fa`.
- `go test ./...` — passed; slowest package was `internal/cli` at 85.228 seconds.
- `go vet ./...` — passed.
- `git diff --check` — passed.
- Actual stdio `go run ./cmd/hero mcp --project-root <fixture>` initialization and `hero_mail_thread_contract` dispatch returned the pinned Mail-thread v1 contract without starting `hero serve`.
- HOLD remediation: `go test ./internal/attention/conformance ./contracts/attention/mailread ./contracts/attention/mailthread -count=1`, `go run ./cmd/attention-conformance --check`, and `git diff --check` passed after pinning the four MCP mappings in canonical/generated guidance and `docs/serve.md`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Evidence |
|---|---|---|---|
| AC-1 | Thread list returns authoritative buckets, counts, cursor, and errors | DONE | `toolMailThreadList` delegates to `mailquery.Service.Threads`; `TestMCPMailThreadDirectDispatchMatchesHTTP` and `TestMCPMailThreadPagingAndStructuredErrors` cover serialized parity, paging, stale cursors, validation, and unavailable envelopes. |
| AC-2 | Thread show returns authoritative composite-identity detail | DONE | `toolMailThreadShow` delegates to `mailquery.Service.ThreadDetail`; direct JSON-RPC HTTP/MCP parity verifies the exact versioned detail envelope and message body. |
| AC-3 | Thread action preserves portable validation, revisions, and idempotency | DONE | `toolMailThreadAction` parses precision-safe decimal revisions into the existing nested `mailthread.ActionRequest`; `TestMCPMailThreadActionPreservesRevisionInputAndIdempotency` covers resolve input, replay, idempotency conflict, and stale revision. |
| AC-4 | Thread contract matches HTTP | DONE | `toolMailThreadContract` uses the canonical Mail-thread constants; parity and `mailthread.ValidateContractResponse` pass, and the real stdio MCP exercise returned the pinned v1 identity. |
| AC-5 | Exact four tools are advertised and dispatched with accurate metadata | DONE | `TestMCPMailDefinitionsAndDispatch`, `TestMCP_ToolsList`, and the generated `mcp-tools.json` assert all four exact names, nested schemas, read-only metadata for list/show/contract, and write metadata for action. |
| AC-6 | Five legacy Mail tools remain compatible | DONE | Definition/handler coverage asserts all five legacy names alongside the four additions; the full `internal/serve` and repository suites pass unchanged legacy behavior. |
| AC-7 | Direct MCP and HTTP/MCP parity cover success and failure paths | DONE | Real JSON-RPC `tools/call` tests cover list, show, action, and contract; focused tests cover continuation/stale cursors, validation/unavailable envelopes, lifecycle input, replay/conflict, and stale revision. |
| AC-8 | Published Attention inventory and derived hashes are regenerated without changing the Mail-thread checksum | DONE | Deterministic regeneration changed only the main Attention inventory/manifest and its compiled/test pins to `2c29a1c6...b31fa`; the canonical/generated Attention handoff and Serve docs now pin the four MCP mappings. Generator and focused conformance tests pass while `mailthread.ConformanceManifestSHA256` remains `5438ed2d...abcd5a`. |

### Changes

| # | Changes item (abbreviated) | Status | Evidence |
|---|---|---|---|
| 1 | Add four exact definitions and dispatcher mappings | DONE | `internal/serve/mcp_tools_def.go`, `internal/serve/mcp_dispatch.go`, and the global inventory test contain the additive surface. |
| 2 | Add registry-backed query seam and thin handlers | DONE | `internal/serve/mcp.go` carries the injectable factory; `internal/serve/mcp_tools_mail.go` adapts arguments and returns existing `mailthread` envelopes without local policy. |
| 3 | Add metadata, direct-dispatch, parity, paging, lifecycle, and error tests | DONE | `internal/serve/mcp_tools_mail_test.go` and `internal/serve/mcp_test.go` provide the focused regression matrix; focused, race, and full suites pass. |
| 4 | Regenerate and pin the consumer-facing Attention MCP mapping | DONE | Updated the canonical source and generated `HERO-CODE-HANDOFF.md`, `docs/serve.md`, generated Attention inventory/manifest, compiled/test checksum pins, and a three-surface drift assertion in `internal/attention/conformance/builder_test.go`; deterministic and focused conformance checks pass. |

### Exercise-the-feature check

- [x] An actual stdio `hero mcp` child initialized and returned the pinned Mail-thread v1 contract through `hero_mail_thread_contract` without an HTTP daemon; stateful list/show/action behavior was exercised through real JSON-RPC `tools/call` dispatch against the same authority as HTTP.

### Excellence Bar self-check

Yes. The change is additive and transport-only, preserves full 64-bit revisions, proves exact authoritative envelope parity, retains legacy tools, and passes focused race plus repository-wide validation.
