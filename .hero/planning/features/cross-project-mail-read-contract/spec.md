---
title: "Cross-Project Mail Read Contract — Paged Metadata and Full Detail"
slug: cross-project-mail-read-contract
type: feature
status: in-review
domain: engineering
priority: high
size: medium
horizon: now
created: 2026-08-16
parent: durable-attention
depends-on:
  - project-mail-core
  - project-mail-triage-and-provenance
  - attention-read-model-v1
  - attention-mcp-action-tools
related:
  - attention-contract-bundle-publication
tags: [mail, attention, hero-code, http, pagination, contracts]
delivery_method: manual
---

# Cross-Project Mail Read Contract — Paged Metadata and Full Detail

## Context

Hero Code's `mail-sidebar-and-direct-detail` spec is blocked on an authoritative
Mail read contract. The released `hero_mail_list` MCP tool is workspace-scoped,
unbounded, and returns full bodies; its action descriptors use the legacy
`read` vocabulary and omit the richer Attention policy fields. The bounded
`hero_attention_snapshot` projection is metadata-only, but intentionally
contains only active unread rows and cannot enumerate Mail history.

Hero permits a 65,536-byte Mail body, while Hero Code's generic MCP result
normalization caps text near 50 KB. A valid `hero_mail_show` response can
therefore be truncated before typed decoding. The existing Hero Code Attention
handoff already declares Hero Serve HTTP as the canonical desktop transport,
so this spec resolves the conflict in favor of an additive typed HTTP Mail
surface. Existing MCP tools remain model-facing compatibility surfaces and do
not change shape or semantics.

## Goal

Publish one versioned, fixture-backed Hero Serve contract that lists
non-dismissed Mail across registered projects as bounded metadata pages, reads
one exact full message without mutation or truncation, and dispatches the
advertised canonical Mail actions and replies through the existing source-owned
services using stable composite identity.

## Kickoff

Adds Hero Code's typed Mail transport: paged cross-project metadata, exact full
detail, canonical actions, and replies over Hero Serve HTTP.

**Status:** in-review — implementation, standalone conformance bundle, docs,
AC-linked tests, and the Completion Ledger are complete; all Go tests pass.

**Pick up at:** run the cold delivery audit, then close with
`hero spec verify cross-project-mail-read-contract --skip-tests`.

→ `.hero/planning/features/cross-project-mail-read-contract/spec.md`

**Files:** `contracts/attention/mailread/contract.go`, `internal/attention/mailquery/service.go`, `internal/attention/mail/capabilities.go`, `internal/serve/api_attention_mail.go`, `contracts/attention/mailread/conformance/v1/manifest.json`
**Skip:** changing legacy `hero_mail_list/show` output or routing Hero Code full bodies through MCP.

## Problem

The current surfaces force a native consumer to choose between incomplete
metadata and an unsafe payload path:

- Direct MCP list results include every full body and have no paging or global
  project enumeration.
- Message IDs are stored beneath recipient peer IDs, but current global lookup
  can search by message ID alone. That is not a safe cross-project target.
- Direct Mail advertises `read`; Attention advertises `mark_read` plus
  `operation_id`, effect, consent, input schema, required revision, and
  idempotency. Hero Code must not maintain its own translation table.
- Receipt fields exist, but no direct consumer contract states unread or a
  canonical activity instant.
- The exact-hash Attention v1 bundle is already released. Adding Mail fixtures
  to that bundle in place would create an unrelated compatibility failure for
  clients pinned to its current hash.

## Design

### Canonical desktop transport and compatibility

Hero Code consumes the new contract through Hero Serve HTTP, alongside its
existing Attention HTTP client. Add these user-global routes before the generic
project router:

- `GET /api/attention/v1/mail/messages`
- `GET /api/attention/v1/mail/messages/{message_id}?project_peer_id=...`
- `POST /api/attention/v1/mail/actions`
- `POST /api/attention/v1/mail/replies`
- `GET /api/attention/v1/mail/contract`

The list and detail handlers return typed JSON directly from shared Go DTOs;
they never pass through MCP text normalization. Detail returns the complete
validated envelope, including every byte of a valid 65,536-byte body. It never
silently truncates: encoding or size failures are structured errors.

Do not change the existing `hero_mail_list`, `hero_mail_show`, or
`hero_mail_action` request/response shapes, including their legacy `read`
action. Do not make Hero Code combine HTTP reads with MCP mutations. The new
HTTP action and reply adapters provide the minimal coherent desktop transport
and delegate to the same Mail service methods used by MCP.

### Versioned Mail-read contract

Add package `contracts/attention/mailread` with v1 request and response DTOs:

- `ListRequest`: optional exact `project_peer_id`, `thread_id`, and
  `unread_only` filters plus `limit` and opaque `cursor`;
- `ListResponse`: `schema_version`, opaque content `revision`, exact scoped
  total/unread counts, returned items, page metadata, next cursor, or one
  `attention.ContractError`;
- `MessageSummary`: authoritative project reference, message/thread/reply IDs,
  sender and recipient, subject, kind, created/activity timestamps, explicit
  unread state, normalized receipt view, receipt revision, and canonical
  `attention.ActionDescriptor` values; it has no body or body-derived summary;
- `DetailResponse`: one full `attention.MailEnvelope` plus the same project,
  receipt, unread, activity, and action metadata as its list item;
- `ActionRequest`/`ActionResponse`: composite target, canonical action ID,
  expected receipt revision, idempotency key, typed action input, resulting
  authoritative receipt/navigation, or structured error;
- `ReplyRequest`/`ReplyResponse`: composite target, expected thread ID, exact
  body/optional subject and kind, idempotency key, and authoritative
  `MailDelivery` or structured error.

Publish a separate immutable Mail-read conformance bundle and manifest beneath
`contracts/attention/mailread/conformance/v1`. Its discovery route advertises
its own schema version, bundle version, and manifest SHA-256. The released
Attention v1 fixture and bundle hashes remain unchanged. Hero Code pins both
contracts independently and treats an absent or incompatible Mail-read bundle
as unavailable rather than falling back to storage or legacy MCP decoding.

### Identity and source ownership

Every list item carries the recipient project's stable `peer_id`; current
registry slug and display name are optional routing/presentation metadata, not
identity. Detail, action, and reply require the composite target
`(project_peer_id, message_id)`. Thread scope is likewise
`(project_peer_id, thread_id)`; neither message nor thread IDs are newly claimed
to be globally unique. A request containing `thread_id` without
`project_peer_id` is invalid rather than a global thread search.

Add `internal/attention/mailquery.Service` as a read/dispatch facade over the
machine registry and the existing `mail.Service` instances. It may enumerate
only registered projects, resolves an exact peer ID before touching a mailbox,
and delegates list/show/action/reply to the service that owns that recipient
box. It does not parse files in handlers, copy envelopes into another store, or
write a projection cache. An arbitrary unregistered peer ID returns `missing`.

If two registry entries resolve to the same canonical project path and peer ID,
choose the lexicographically smallest registry slug for presentation. If
different canonical paths claim one peer ID, fail the read as `unavailable`
with the conflicting peer ID instead of choosing a mailbox silently. In v1,
any registry/config/mailbox failure fails the whole scoped query; partial data
is never represented as a complete empty or successful page.

### Paging, ordering, receipts, and threads

List defaults to 20 items and accepts 1 through 100. It enumerates
non-dismissed messages, applies project/thread/unread filters, and returns exact
counts for that scope. Sort by parsed instants and stable identity:

1. immutable envelope `created_at` exposed as `activity_at`, descending;
2. recipient `peer_id` ascending;
3. message ID ascending.

For message pagination, canonical `activity_at` is the immutable envelope
delivery instant in `created_at`, parsed as an RFC3339Nano instant rather than
compared as text. Receipt mutations never reorder the message list. A client
may derive a thread leaf's activity as the maximum delivery instant among the
inbound messages it has authoritatively enumerated, but the contract makes no
claim of complete bidirectional thread reconstruction. Malformed timestamps
fail closed through the existing validators.

The normalized receipt view exposes revision, `unread`, read/acknowledged/
dismissed timestamps, promoted artifact reference, and Focus item ID, but not
internal action request hashes or idempotency journals. A missing receipt is
revision zero and unread. Reads never create a receipt.

Cursors are opaque, versioned, and bound to the exact filters plus the full
metadata revision. The revision hashes the ordered summary identity, activity,
receipt revision, unread state, and action IDs. Continuation recomputes that
revision; if delivery or receipt/capability state changed between pages, return
`stale` and require a fresh first page. This avoids offset duplicates/skips and
keeps the cursor implementation replaceable by a metadata index later.

List items expose authoritative `thread_id` and `in_reply_to` without grouping
or subject inference. A client can page a thread by supplying exact
`project_peer_id + thread_id`, then fetch each full message individually.
Detail does not return an unbounded aggregate thread or claim that outbound
delivery receipts contain bodies.

### Canonical capabilities and mutations

Move Mail capability construction into
`internal/attention/mail/capabilities.go` and have both the new query service
and `internal/attention/projection/actions.go` use it. Each advertised
descriptor includes canonical ID, label, operation ID, effect, consent, input
schema, required receipt revision, and idempotency requirement from the
existing interaction policy. Unknown future descriptors remain decodable and
inert.

The new contract advertises and accepts only this mapping:

| Consumer action ID | Operation ID | Existing source operation |
|---|---|---|
| `mark_read` | `mail.mark_read` | `mail.ActionRead` (`read`) |
| `acknowledge` | `mail.acknowledge` | `mail.ActionAcknowledge` |
| `dismiss` | `mail.dismiss` | `mail.ActionDismiss` |
| `promote` | `mail.promote` | `mail.ActionPromote` |
| `add_to_today` | `mail.add_to_today` | `mail.ActionAddToToday` |
| `reply` | `mail.reply` | `mail.Service.Reply` |

Legacy MCP continues to advertise and accept `read`; the alias never appears
in the new DTOs. The HTTP adapters validate that the requested capability is
currently advertised, translate through the single mapping above, and then
delegate. Opening/listing/showing a message does not mark it read. Reply uses
its dedicated typed request because it carries body content and thread guards;
it is not squeezed into generic action input.

### Rollout and rollback

Ship in contract-first order: DTO/schema/fixtures, capability builder, query
service, HTTP adapters, then documentation/handoff. Hero Code can feature-detect
the Mail-read contract endpoint and pin its hash before enabling the sidebar.
There is no data or schema migration. Rollback removes the additive routes and
contract advertisement without altering envelopes or receipts; a newer Hero
Code must surface Mail as unavailable and must not fall back to legacy MCP or
local files.

## Changes

1. Add the independent Mail-read contract package and validators.
   - Add `contracts/attention/mailread/contract.go`, `validate.go`, and focused
     tests for list/detail/action/reply DTOs, composite identity, limits,
     cursors, timestamps, and unknown additive fields.
   - Add JSON Schemas under `contracts/attention/mailread/schema/v1` without
     modifying existing Attention v1 schemas or fixture hashes.
2. Centralize canonical Mail capabilities and source mapping.
   - Add `internal/attention/mail/capabilities.go` and tests deriving rich
     descriptors from `contracts/attention/interaction.go` policies.
   - Update `internal/attention/projection/actions.go` to consume the shared
     builder while preserving existing Attention row semantics.
   - Leave `internal/serve/mcp_tools_mail.go` legacy descriptor/output code
     unchanged except for tests proving it remains compatible.
3. Add the registry-backed query facade.
   - Add `internal/attention/mailquery/service.go`, `cursor.go`, and tests for
     registered-project discovery, peer-ID conflicts, composite lookup,
     receipt normalization, activity calculation, sorting, filtering, counts,
     revision-bound keyset cursors, and source delegation.
   - Reuse `internal/attention/mail/store.go` and `service.go`; add no second
     database, persisted index, or handler-owned storage access.
4. Add typed Hero Serve HTTP adapters.
   - Add `internal/serve/api_attention_mail.go` for list, exact detail,
     canonical action, reply, and contract discovery handlers.
   - Wire a process-global Mail query factory through `internal/serve/api.go`
     and `server.go`, mounting routes before `/api/` project routing.
   - Enforce existing local/private Attention access posture and request body
     bounds while allowing the full valid Mail envelope in responses.
5. Publish a standalone golden conformance bundle.
   - Add list-page, detail, receipt-state, threaded-page, canonical-actions,
     action/reply success and failure, errors, unknown-additive, max-body, and
     cross-project fixtures under `contracts/attention/mailread/testdata/v1`.
   - Add the immutable vendorable bundle under
     `contracts/attention/mailread/conformance/v1`, its builder/checksum tests,
     and an exact manifest hash advertised by the Mail contract endpoint.
   - Include duplicate message IDs in different projects and equal activity
     timestamps to pin composite identity and ordering.
6. Add cross-boundary regressions and consumer guidance.
   - Extend `internal/serve/api_attention_test.go` or add
     `api_attention_mail_test.go` for HTTP/service semantic parity,
     unavailable-versus-empty behavior, wrong-project lookup, stale cursors,
     and mutation non-replay.
   - Update `contracts/attention/conformance/v1/HERO-CODE-HANDOFF.md` only with
     an additive pointer to the independently hashed Mail-read bundle; do not
     change the existing Attention bundle identity.
   - Mirror that additive pointer in the generator source at
     `contracts/attention/testdata/v1/HERO-CODE-HANDOFF.md` so the checked-in
     Attention bundle remains reproducible without changing its manifest hash.
   - Update `docs/serve.md` with the HTTP-only Hero Code Mail transport and
     explicit legacy MCP compatibility statement.

## Acceptance Criteria

- **AC-1:** WHEN `/api/attention/v1/mail/messages` is requested without a project filter, THE SYSTEM SHALL return a bounded metadata-only page across every uniquely registered recipient peer ID with exact scoped total/unread counts and no message body.
- **AC-2:** WHEN a list is filtered by `project_peer_id` or `thread_id`, THE SYSTEM SHALL scope by exact stable composite identity and SHALL NOT resolve a mailbox from display name, stale envelope alias, current workspace, or message ID alone.
- **AC-3:** WHEN Mail summaries are paged, THE SYSTEM SHALL expose immutable envelope delivery time as canonical `activity_at`, order by its parsed instant then peer ID and message ID, and use a filter- and revision-bound opaque cursor that returns `stale` instead of duplicating or skipping across a changed result set.
- **AC-4:** WHEN a summary or detail is returned, THE SYSTEM SHALL expose explicit unread state, normalized receipt revision/state, canonical activity time, message ID, thread ID, reply target, sender, recipient project identity, and currently applicable rich action descriptors.
- **AC-5:** WHEN a thread filter is used, THE SYSTEM SHALL return only messages carrying that authoritative project-scoped thread ID and SHALL NOT infer membership from subject, body, sender, or display text.
- **AC-6:** WHEN exact detail is requested with a registered `project_peer_id` and message ID, THE SYSTEM SHALL return the complete validated envelope without changing its receipt; a valid 65,536-byte body SHALL round-trip byte-for-byte with its final-byte canary intact and SHALL NOT be silently truncated.
- **AC-7:** WHEN Mail capabilities are advertised, THE SYSTEM SHALL use only canonical `mark_read`, `acknowledge`, `dismiss`, `promote`, `add_to_today`, and `reply` IDs with labels, operation IDs, effects, consent, input schemas, required receipt revisions, and idempotency requirements derived from the central interaction policy.
- **AC-8:** WHEN a canonical Mail action or reply is submitted over HTTP, THE SYSTEM SHALL resolve its composite target, validate the currently advertised capability/revision/input/idempotency, translate through the exact source mapping, and delegate once to the existing Mail service authority.
- **AC-9:** WHILE list or detail is read, THE SYSTEM SHALL NOT write a receipt, mark Mail read, execute body content, launch a model, create work, or mutate a project tree.
- **AC-10:** IF a request has an invalid/incompatible contract, unregistered or conflicting project identity, missing message, stale cursor/revision, unsupported action, or unavailable registry/state/service THEN THE SYSTEM SHALL return the matching structured error and SHALL NOT represent it as an empty page, partial success, or retried mutation.
- **AC-11:** WHILE the new HTTP contract is enabled, THE SYSTEM SHALL preserve the released request/response and legacy `read` action behavior of `hero_mail_list`, `hero_mail_show`, and `hero_mail_action` and SHALL keep the existing Attention v1 bundle hashes unchanged.
- **AC-12:** WHEN the standalone Mail-read golden bundle is validated, THE SYSTEM SHALL pin list/detail/receipt/thread/action/reply/error/cross-project/max-body behavior, tolerate unknown additive fields and raw identifiers as inert data, and advertise the exact matching manifest hash at runtime.

## Boundaries

- No Hero Code/SwiftUI implementation or peer-repository mutation.
- No change to Mail envelope/receipt storage, no migration, no copied read
  database, and no metadata index in this release.
- No breaking or repurposing of `hero_mail_list`, `hero_mail_show`,
  `hero_mail_action`, or the existing Attention snapshot/action contracts.
- No Mail send endpoint, generic Attention write endpoint, event stream, push
  notification, cloud transport, attachment handling, or model runtime.
- No automatic mark-read on list, detail, visibility, scroll, or reconnect.
- No subject-based thread inference and no claim of complete bidirectional
  thread history from outbound delivery receipts.
- No arbitrary peer-ID mailbox probing; only registered local projects are in
  scope.

## Risks

- **Filesystem scan cost:** v1 still reads and validates authoritative
  envelopes/receipts before paging. Measure a representative multi-project
  mailbox. The opaque cursor preserves a later source-maintained metadata index
  without committing to one now.
- **Receipt-state invalidation:** read/acknowledge does not reorder immutable
  message activity, but it changes unread, revision, or capabilities while a
  client pages. Revision-bound cursors deliberately invalidate instead of
  returning mixed receipt state.
- **Registry identity conflicts:** copied projects can share a peer ID. Failing
  closed is less convenient than choosing one silently but prevents cross-
  project disclosure or mutation.
- **Contract hash rollout:** the current Attention bundle hash is an existing
  consumer pin. A separate Mail-read bundle avoids invalidating it; release
  checks must still prove runtime discovery and vendored artifacts agree.
- **Sensitive/untrusted content:** full detail may contain secrets or prompt
  injection. The route retains existing private local access, returns inert
  data, and never logs or executes the body.
- **Rollback:** rollback is code-only because no state shape changes. Removing
  the routes makes the capability unavailable; clients must not interpret that
  as empty Mail or activate a legacy decoding fallback.

## Validation

- `go test ./contracts/attention/mailread/... ./internal/attention/mail/... ./internal/attention/mailquery/... ./internal/attention/projection/... ./internal/serve/...`
- `go test ./...`
- Golden schema/DTO/manifest checksum tests for every Mail-read fixture and
  unknown additive fields/identifiers.
- HTTP test with an exact 65,536-byte UTF-8 body and trailing canary proving
  byte-for-byte detail decoding with no MCP normalization path.
- Cross-project tests with duplicate message IDs, exact project/thread filters,
  duplicate peer-ID conflicts, stale registry entries, equal timestamps, and
  empty versus unavailable state.
- Pagination tests for first/last pages, limits, filter-bound cursors, receipt
  state invalidation without activity reorder, and deterministic replay against
  unchanged state.
- Action/reply tests for every canonical descriptor and source mapping, stale
  revision, idempotent replay, conflicting replay, unsupported/unknown action,
  and no mutation during list/show.
- Compatibility tests recording the existing direct MCP list/show/action JSON
  shapes and existing Attention v1 fixture/bundle hashes unchanged.
- Representative mailbox benchmark or bounded performance test documenting
  full-scan latency and allocation before considering an index.

## Completion Ledger

Implemented the independent Go `mailread/v1` contract, source-owned registry
facade, canonical capabilities, Hero Serve HTTP adapters, vendorable golden
bundle, compatibility documentation, and cross-boundary regressions. Loaded
the Go, API-contract, reliability, context-injection, testing, kickoff, and
Completion Ledger guidance. Focused affected-package tests, the uncached full
Go suite, formatting, diff checks, and a 300-message scan benchmark all pass.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Return bounded metadata-only cross-project pages with exact counts | DONE | `internal/attention/mailquery/service.go:55` and `internal/serve/api_attention_mail_test.go:59` — bounded list pages omit bodies and expose exact total/unread counts. |
| 2 | Scope project/thread filters by exact stable composite identity | DONE | `internal/attention/mailquery/service_test.go:68` and `internal/attention/mailquery/service_test.go:132` — exact peer/thread filters and duplicate message IDs resolve only through the registered peer. |
| 3 | Order by parsed immutable activity and use revision-bound opaque cursors | DONE | `internal/attention/mailquery/service_test.go:68` and `internal/attention/mailquery/cursor_test.go:5` — parsed timestamp ordering, stable tie-breakers, filter binding, deterministic continuation, and stale revision behavior are exercised. |
| 4 | Expose normalized receipt, unread, identity, activity, and rich actions | DONE | `contracts/attention/mailread/validate_test.go:41` and `internal/attention/mailquery/service.go:375` — summary/detail metadata and normalized receipt state validate against the current project and receipt revision. |
| 5 | Filter threads only by authoritative project-scoped thread ID | DONE | `internal/attention/mailquery/service_test.go:68` — the thread page includes only exact stored thread IDs under the requested peer. |
| 6 | Return exact nonmutating detail including a 65,536-byte body | DONE | `internal/serve/api_attention_mail_test.go:59` and `contracts/attention/mailread/conformance_test.go:136` — the maximum valid body round-trips byte-for-byte with its final canary and no receipt is created. |
| 7 | Advertise only canonical rich Mail capabilities from central policy | DONE | `internal/attention/mail/capabilities_test.go:10` — all six canonical IDs, operation/effect/consent/input/revision/idempotency fields, and exact source mappings are pinned. |
| 8 | Validate and delegate canonical action/reply mutations exactly once | DONE | `internal/attention/mailquery/service_test.go:181` and `internal/serve/api_attention_mail_test.go:95` — source delegation, stale revision, action/reply replay, and conflicting idempotency keys are covered. |
| 9 | Keep list/detail reads inert and nonmutating | DONE | `internal/attention/mailquery/service_test.go:132` and `internal/serve/api_attention_mail_test.go:59` — list/detail leave the receipt absent and treat body content only as returned data. |
| 10 | Return structured errors without empty/partial/retried failure semantics | DONE | `internal/attention/mailquery/service_test.go:160`, `internal/attention/mailquery/service_test.go:233`, `internal/attention/mailquery/service_test.go:244`, and `internal/serve/api_attention_mail_test.go:156` — conflicts, stale registry/mailbox state, missing/wrong identity, validation, stale cursor, and unavailable-versus-empty are distinct. |
| 11 | Preserve legacy MCP shapes/read action and Attention v1 hashes | DONE | `internal/serve/mcp_tools_mail_test.go:48` and `contracts/attention/mailread/conformance_test.go:60` — legacy body/descriptor output remains `read`, while both released Attention manifests retain their exact hashes. |
| 12 | Pin and advertise the standalone golden Mail-read bundle | DONE | `contracts/attention/mailread/conformance_test.go:17`, `contracts/attention/mailread/conformance_test.go:60`, and `internal/serve/api_attention_mail_test.go:156` — 19 fixture categories, schema/DTO/checksum validation, unknown inert fields, maximum body, and runtime manifest parity pass. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add independent Mail-read DTOs, validators, schemas, and tests | DONE | `contracts/attention/mailread/contract.go`, `validate.go`, `schema/v1/`, and focused validator tests implement the v1 wire contract. |
| 2 | Centralize canonical Mail capabilities and preserve projection/MCP semantics | DONE | `internal/attention/mail/capabilities.go`, `internal/attention/projection/actions.go`, and `internal/serve/mcp_tools_mail_test.go` share policy-derived descriptors while pinning legacy behavior. |
| 3 | Add registry-backed source-owned query facade and cursors | DONE | `internal/attention/mailquery/service.go`, `cursor.go`, tests, and benchmark cover discovery, identity conflicts, reads, dispatch, ordering, filtering, counts, and stale continuation. |
| 4 | Add typed Hero Serve Mail adapters and process-global wiring | DONE | `internal/serve/api_attention_mail.go`, `api.go`, `server.go`, `api_attention.go`, and HTTP tests add all five private user-global routes with bounded requests and structured statuses. |
| 5 | Publish standalone vendorable golden conformance bundle | DONE | `contracts/attention/mailread/bundle.go`, `cmd/mailread-bundle/`, `testdata/v1/`, `conformance/v1/`, and conformance tests publish and pin the independently hashed bundle. |
| 6 | Add cross-boundary regressions and Hero Code guidance | DONE | `internal/serve/api_attention_mail_test.go`, both Attention handoff sources, and `docs/serve.md` cover HTTP parity, compatibility, and HTTP-only consumer guidance. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `go test ./internal/serve -count=1` sent real HTTP list/detail/action/reply/contract requests, including an exact 65,536-byte body with final-byte canary; `go test ./... -count=1` also passed.

### Excellence Bar self-check

- [x] Yes — the implementation is source-owned, additive, independently versioned, compatibility-pinned, fail-closed, exercised at HTTP boundaries, and fully covered by the approved acceptance criteria.
