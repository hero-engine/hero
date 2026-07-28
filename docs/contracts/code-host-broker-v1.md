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

All requests contain `version`, `operation`, expected `provider`,
`connection_id`, and `repository`. Hero verifies the expected provider against
the resolved connection. PR-specific operations also require `pull_request`.
Collection operations may carry bounded `query`, `order`, `limit`, and an
opaque `cursor`. `repositories` is an optional additional repository scope;
the primary `repository` is always part of the scope.

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
- redirect and mutation-journal entry counts; and
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

### Field and nullability catalog

Fields described as optional use `omitempty` and may be absent. Pointer fields
are nullable. All other fields are emitted; an operation validator decides
whether a zero value is legal.

- `RepositoryIdentity`: required `host`, `owner`, `name`, `full_name`;
  optional `provider_id`.
- `RefIdentity`: required `repository`, `name`, `sha`.
- `PullRequestIdentity`: required `connection_id`, `repository`,
  `provider_id`, positive `number`.
- `Actor`: required `login`; optional `provider_id`, `display`.
- `PullRequest`: required `identity`, `title`, `url`, `state`, `draft`,
  `author`, `base`, `head`; optional `body`, `created_at`, `updated_at`,
  `merged_at`.
- `Commit`: required `sha`, `message`, `author`; optional `authored_at`,
  `url`.
- `DiffHunk`: required `header`, `patch`.
- `DiffFile`: required `path`, `status`, non-negative `additions`,
  non-negative `deletions`, `hunks`, `truncated`.
- `Check`: required `name`, `status`, `availability`; optional
  `provider_id`, `conclusion`, `url`.
- `Review`: required `provider_id`, `author`, `state`, `head_sha`; optional
  `body`, `submitted_at`.
- `Comment`: required `provider_id`, `author`, `body`; optional `url`,
  `created_at`, `updated_at`.
- `MergeReadiness`: required `state`, six availability fields (`checks`,
  `reviews`, `branch_protection`, `permissions`, `mergeability`, `queue`),
  and bounded `reasons`.
- `RateLimit`: required `observed_at`; optional/zero-when-unknown `resource`,
  `limit`, `remaining`, `reset_at`, `retry_after_seconds`. Numeric values are
  non-negative and remaining cannot exceed a known limit.
- `Page`: required bounded `limit` and `count`; optional `next_cursor`.
  Absence of `next_cursor` is the terminal page.
- `PartialFailure`: required `section`, normalized `code`, bounded safe
  `message`.
- `Receipt`: required safe `operation_id`; optional
  `provider_receipt_id`, `target_revision`.
- `Reconciliation`: required `status` and safe `key`; it has no free-form body
  field.
- `ContractError`: required normalized `code`, bounded safe `message`, exact
  `retry`; optional `field`, `retry_at`.
- `Bounds`: required maxima for `repository_scopes`, `page_size`, `items`,
  `text_bytes`, `body_bytes`, `diff_bytes`, `diff_files`, `diff_hunks`,
  `partial_failures`, `error_detail_bytes`, `duration_ms`, `redirects`,
  `journal_entries`, and `idempotency_bytes`.
- `OperationPolicy`: required `operation`, `effect`, `consent`,
  `requires_unique_target`, `requires_idempotency`,
  `requires_fresh_observation`, `requires_reconciliation`, `replay_safe`,
  and `bounds`.
- `Capability`: required `policy`, `available`; optional `reason`.
- `Response`: required `version`, `operation`, `provider`, `connection_id`,
  `repository`, `policy`, both revisions, `observed_at`, `freshness`,
  `rate_limit`, `bounds`, `completeness`, `partial_failures`, `result`,
  `truncated`, `duration_ms`, `redirects`, `journal_entries`, and nullable
  `error`; optional/nullable `page`, `receipt`, and `reconciliation`.
  Successful mutations require both receipt and reconciliation. Error
  responses carry JSON `null` as result.
- `Request`: required `version`, `operation`, `provider`, `connection_id`,
  `repository`; optional `repositories`, `pull_request`, `intent_source`,
  `consent`, `idempotency_key`, both revisions, `reconciliation_key`,
  `query`, `order`, `limit`, `cursor`, and `payload`. Policy makes the
  mutation fields and typed payload mandatory.
- `CreatePullRequestPayload`: required `base`, `head`, `title`, `draft`;
  optional `body`.
- `CommentPayload`: required `expected_head_sha`, `body`.
- `ReviewPayload`: required `expected_head_sha`; optional `body`.
- `RetargetPayload`: required `expected_head_sha`, `current_base`, `new_base`.
- `LifecyclePayload`: required `expected_head_sha`.
- `MergePayload`: required `expected_head_sha`, `observed_base`, `method`;
  optional `commit_title`, `commit_message`.
- `CursorMaterial`: required `version`, `provider`, `connection_id`,
  normalized complete repository identities (host, provider repository ID,
  namespace, name, and full name), `operation`, normalized `query`, `order`,
  and provider `position`. `CursorEnvelope` contains that material plus
  required `fingerprint`.
- `RevisionMaterial`: required `connection_id`, `repository`; optional
  `pull_request`, `base`, `head`, `state`, `updated_at`, `permissions`.
- Fixture-only `FixtureCase`: required `name`, `request`, `response`.
  `ConsumerFixtureBundle`: required `version`, advertised `operations`,
  `cases`, normalized `errors`, and `future_additive`.

Success result schemas are operation-specific and strictly decoded:
`CapabilitiesResult.capabilities`, `PullRequestsResult.pull_requests`,
`PullRequest`, `CommitsResult.commits`, `DiffResult.files`,
`ChecksResult.checks`, `ReviewsResult.reviews`, `CommentsResult.comments`,
`MergeReadiness`, or `MutationResult` with required `pull_request` and
`outcome`, an optional typed `actor` for actor-attributed mutations, and
optional `invalidated_operations` naming read evidence consumers must discard.
Hero's producer-side validator rejects non-canonical adapter results;
independent consumers ignore unknown additive result fields and unknown
advertised capability entries while preserving all known values.

## Pagination, revisions, and rate limits

Cursors are opaque base64url-encoded envelopes. Their embedded SHA-256
fingerprint binds contract version, provider, connection, normalized
repository scope, operation, normalized query, ordering, and provider
position. Both returned and requested cursors are decoded and fingerprint
validated; a request additionally compares scope, operation, query, and order.
Reuse with any different binding returns `cursor_mismatch`; clients never edit
or transplant cursors.

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
| Repository scopes, including the primary repository | 100 |
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
`consumer-fixture.sha256`. The current digest is
`f35203f1a3656b4d591e30def627d6d9d9cf4f2c688b61b45dca225fd4f92cd3`.
Regenerate both deterministically with:

```bash
go generate ./contracts/codehostbroker
```

The fixture covers all twenty operations, every operation policy, forked refs,
non-terminal and terminal pagination, all availability and completeness
states, partial checks, truncated diffs, all seven reconciliation states, all
normalized errors, an unknown advertised operation, and unknown additive
fields. Mutation text fields contain only the literal `[redacted]` sentinel,
never user content. Hero and Hero Code's Swift decoder consume the same
committed bytes and digest.

## Security and ownership boundary

This package defines types and validation only. It performs no configuration
lookup, credential resolution, network I/O, CLI dispatch, MCP registration, or
durable journaling. Later Hero-owned broker layers resolve credentials inside
Hero and expose only bounded contract values to in-process, CLI, and MCP
consumers. Swift never persists or receives a provider credential.

GitHub Projects issue importing remains a tracker capability and is not part of
this PR lifecycle API.
