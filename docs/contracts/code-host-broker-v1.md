# Code-Host Broker v1

`code-host-broker/v1` is Hero's provider-neutral pull-request lifecycle
contract. A local client can discover, inspect, and mutate pull requests while
Hero alone resolves the configured code-host connection and credential.
Credentials, authorization headers, provider tokens, and unredacted provider
bodies are never contract values.

The canonical Go types, validators, operation-policy registry, and consumer
fixture live under `contracts/codehostbroker/`. This contract is independent
of tracker selection: a workspace may use Jira or Linear for work tracking and
GitHub for code hosting. GitHub may advertise both capabilities, but a tracker
connection is not implicitly a code-host connection.

## Version and compatibility

Every request and response carries the literal version
`code-host-broker/v1`. V1 evolves additively:

- consumers ignore unknown fields and unknown advertised capability names;
- brokers reject an unknown requested operation;
- brokers and consumers fail closed on an incompatible major version; and
- removing, renaming, or changing the meaning or policy of an existing field
  or operation requires `code-host-broker/v2`.

The fixture includes unknown top-level additive fields so independent
decoders can prove that behavior.

## Operations and authoritative policy

The operation registry is the single authority for effects, consent, replay,
freshness, reconciliation, and bounds. Surfaces and clients must not infer
these properties from operation names.

| Operation | Effect | Consent | Unique target | Idempotency | Fresh observation | Reconciliation |
|---|---|---|---:|---:|---:|---:|
| `capabilities` | `read` | `none` | no | no | no | no |
| `list_pull_requests` | `read` | `none` | no | no | no | no |
| `search_pull_requests` | `read` | `none` | no | no | no | no |
| `get_pull_request` | `read` | `none` | yes | no | no | no |
| `get_commits` | `read` | `none` | yes | no | no | no |
| `get_diff` | `read` | `none` | yes | no | no | no |
| `get_checks` | `read` | `none` | yes | no | no | no |
| `get_reviews` | `read` | `none` | yes | no | no | no |
| `get_comments` | `read` | `none` | yes | no | no | no |
| `get_merge_readiness` | `read` | `none` | yes | no | no | no |
| `create_pull_request` | `external_write` | `explicit_user` | yes | yes | yes | yes |
| `comment` | `external_write` | `explicit_user` | yes | yes | yes | yes |
| `submit_review` | `external_write` | `explicit_user` | yes | yes | yes | yes |
| `approve` | `external_write` | `explicit_user` | yes | yes | yes | yes |
| `request_changes` | `external_write` | `explicit_user` | yes | yes | yes | yes |
| `mark_ready` | `external_write` | `explicit_user` | yes | yes | yes | yes |
| `retarget` | `external_write` | `explicit_user` | yes | yes | yes | yes |
| `close` | `external_write` | `explicit_user` | yes | yes | yes | yes |
| `reopen` | `external_write` | `explicit_user` | yes | yes | yes | yes |
| `merge` | `commitment` | `explicit_acceptance` | yes | yes | yes | yes |

Every registry policy is marked replay-safe because a mutation may only be
replayed with the same stable idempotency and reconciliation material. That
does not authorize a blind provider retry or a new key after an ambiguous
outcome.

Connection capabilities and runtime operation capabilities are different:
`capabilities: [code-host]` makes a connection eligible for this broker;
the `capabilities` operation advertises which of the twenty operations its
selected provider adapter currently supports. Unsupported operations fail
closed.

## Repository-qualified identity

A repository identity contains `host`, provider repository ID when known,
`owner`, `name`, and canonical `full_name`. `full_name` must equal
`owner/name`. A pull-request identity is the tuple:

1. stable `connection_id`;
2. complete repository identity;
3. opaque provider PR identity; and
4. positive repository-local PR number.

A PR number, URL, or branch alone is never a mutation target. The request's
connection and repository must exactly match the PR identity.

Base and head refs each carry their own complete repository identity, ref
name, and commit SHA. This preserves fork topology. A force push changes the
head SHA and therefore invalidates operation-specific observation material;
the client refreshes and makes a new explicit decision rather than applying a
write to stale code.

## Requests

All requests contain `version`, `operation`, `connection_id`, and
`repository`. PR-specific operations also require `pull_request`.
Collection operations may carry bounded `query`, `order`, `limit`, and an
opaque `cursor`.

Every mutation additionally requires:

- `intent_source: "user"`;
- consent exactly matching the advertised policy;
- a stable `idempotency_key`;
- the advertised `capability_revision`;
- an operation-specific `observation_revision`;
- a stable `reconciliation_key`; and
- the operation's typed bounded payload.

PR creation has no pre-existing PR identity, so its observation revision binds
the selected repository plus base and head refs. Other mutations bind the
repository-qualified PR and relevant mutable inputs. Comment and review
payloads include the expected head SHA. Retarget includes current and new base
refs. Merge includes the expected head SHA, observed base ref, and one of the
advertised `merge`, `squash`, or `rebase` methods.

Payload decoding fails on unknown fields. Additive compatibility applies to
the versioned envelope and advertised capabilities, not to silently accepting
misspelled mutation inputs.

## Response envelope

Every response contains:

- `version`, `operation`, `provider`, `connection_id`, and `repository`;
- the exact authoritative operation `policy` and `bounds`;
- `capability_revision`, `observation_revision`, and RFC 3339
  `observed_at`;
- explicit `freshness`, `completeness`, and `truncated` states;
- rate-limit metadata with its own observation time;
- bounded pagination and section-level `partial_failures`;
- a typed JSON `result`, or one normalized `error`, never both;
- duration; and
- for mutations, a safe receipt and reconciliation outcome.

Freshness is one of `current`, `stale`, `unknown`, or `unavailable`.
Completeness is one of `complete`, `partial`, `truncated`, or `unavailable`.
Availability inside checks and merge readiness is one of `available`,
`partial`, `unavailable`, or `unknown`. Clients must preserve these states:
unknown or unavailable data is not an empty successful result.

Partial responses retain every successful section and list bounded failures by
section. For example, checks may be returned while branch-protection data is
forbidden. A diff can be explicitly truncated while still returning its
bounded files and hunks.

## Pagination, revisions, and rate limits

Cursors are opaque to clients. Their fingerprint binds contract version,
provider, connection, repository scope, operation, normalized query, ordering,
and provider position. Reuse with any different binding returns
`cursor_mismatch`; clients never edit or transplant cursors.

Capability revisions change when supported operations, permissions, or merge
methods change. Observation revisions change when operation-relevant mutable
state changes, including head SHA, base SHA/ref, PR state, update marker, or
permissions. A stale revision returns `stale_observation` or
`capability_changed` with `refresh_then_retry`.

Rate-limit metadata records resource, limit, remaining count, reset time,
retry-after seconds, and the time Hero observed it. A rate-limited result uses
`rate_limited` and `retry_after`; absence of provider rate-limit information is
represented as unknown metadata, not unlimited capacity.

## Idempotency, reconciliation, and ambiguous results

Hero persists or derives stable operation and reconciliation material without
including request bodies or credentials. A successful or recovered mutation
reports one of:

| Status | Meaning |
|---|---|
| `applied` | this attempt was proven to apply the requested effect |
| `replayed` | the same idempotency key returned its prior proven result |
| `reconciled_applied` | a follow-up provider read proved the effect applied |
| `externally_completed` | the desired state already existed due to another actor |
| `not_applied` | reconciliation proved no effect occurred |
| `in_progress` | the provider accepted work that has not reached a terminal state |
| `ambiguous` | available evidence cannot prove whether the effect occurred |

Cancellation stops local work promptly but does not assert that an in-flight
provider mutation was undone. Likewise, timeout or connection loss after
dispatch is `ambiguous_result` with `reconcile`. The client must reconcile
using the same stable key; it must not create a new key and replay the write.
Merge queues commonly return `in_progress`, and a later read or reconciliation
may prove `applied` or `externally_completed`.

## Normalized errors and retry guidance

Retry guidance is a closed enum: `none`, `same_key`,
`refresh_then_retry`, `retry_after`, or `reconcile`.

| Retry guidance | Error codes |
|---|---|
| `refresh_then_retry` | `stale_observation`, `capability_changed`, `conflict` |
| `same_key` | `idempotency_conflict`, `operation_in_progress` |
| `retry_after` | `rate_limited` |
| `reconcile` | `ambiguous_result` |
| `none` | `invalid_input`, `incompatible_version`, `connection_not_found`, `code_host_role_missing`, `wrong_connection_capability`, `credential_unavailable`, `unauthorized`, `forbidden`, `unsupported_provider`, `unsupported_operation`, `not_found`, `provider_unavailable`, `provider_error`, `partial_failure`, `input_too_large`, `output_too_large`, `cancelled`, `encoding_error` |

Messages and fields are safe, bounded summaries. They never contain raw
provider response bodies, authorization headers, credentials, token-derived
values, or mutation text bodies.

## Bounds

The v1 registry publishes and validators enforce these maximums:

| Material | Maximum |
|---|---:|
| Repository scopes | 100 |
| Page size | 100 |
| Items per result | 100 |
| Text | 64 KiB |
| Mutation/body text | 256 KiB |
| Diff bytes | 2 MiB |
| Diff files | 300 |
| Diff hunks | 2,000 |
| Partial failures | 20 |
| Error detail | 4 KiB |
| Duration | 120,000 ms |
| Redirects | 3 |
| Mutation journal entries | 10,000 |
| Idempotency material | 512 bytes |

Adapters may return less. They must represent truncation or partial truth
explicitly and must not return an unbounded raw provider payload.

## Canonical consumer fixture

The canonical fixture is:

`contracts/codehostbroker/testdata/v1/consumer-fixture.json`

Its SHA-256 digest is published beside it in
`consumer-fixture.sha256`. Regenerate both deterministically with:

```bash
go generate ./contracts/codehostbroker
```

The fixture covers all twenty operations, every operation policy, forked refs,
pagination, partial checks, truncated diffs, all seven reconciliation states,
all normalized errors, and unknown additive fields. Hero and Hero Code's Swift
decoder consume the same committed bytes and digest.

## Security and ownership boundary

This package defines types and validation only. It performs no configuration
lookup, credential resolution, network I/O, CLI dispatch, MCP registration, or
durable journaling. Later Hero-owned broker layers resolve credentials inside
Hero and expose only bounded contract values to in-process, CLI, and MCP
consumers. Swift never persists or receives a provider credential.

GitHub Projects issue importing remains a tracker capability and is not part of
this PR lifecycle API.
