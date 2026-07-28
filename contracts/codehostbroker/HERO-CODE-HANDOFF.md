# Hero Code code-host broker v1 handoff

Hero owns provider credentials, connection selection, authorization, bounded
provider calls, normalization, mutation journaling, and reconciliation. Hero
Code sends and receives only `code-host-broker/v1` JSON. It must never persist
or request a GitHub token and must not invoke `gh` or GitHub directly.

## Contract discovery

A released Hero binary emits the complete consumer contract without loading a
workspace or resolving credentials:

```bash
hero code-host contract
```

The command writes exactly one JSON object:

```json
{
  "version": "code-host-broker/v1",
  "sha256": "<canonical fixture sha256>",
  "bounds": {},
  "policies": [],
  "fixture": "<exact UTF-8 JSON bytes of the canonical fixture>"
}
```

`fixture` is a JSON string so its decoded UTF-8 bytes are byte-for-byte equal
to `contracts/codehostbroker/testdata/v1/consumer-fixture.json`. Hash those
decoded bytes and require an exact match with `sha256` before running decoder
fixtures. `policies` is the authoritative ordered operation registry and
`bounds` is the shared v1 bound set. Do not infer effect, consent, freshness,
idempotency, or reconciliation requirements from operation names.

The current canonical fixture SHA-256 is
`e66a8d5643dce518db66a5e20b2a39be1ac5766b464f2f1244c05ff6a8b43edb`.

## CLI transport

For every operation, write exactly one typed `Request` object to stdin and read
exactly one `Response` object from stdout:

```bash
hero code-host broker get_pull_request < request.json
```

The path operation and `request.operation` must match. The command rejects
unknown fields, trailing JSON values, and input over 1 MiB before constructing
the broker or reaching a provider. Provider tokens, authorization headers,
request bodies, review text, and other content fields are not CLI flags or
positional arguments.

Known-operation input failures are returned as normalized v1 JSON on stdout.
Process stderr is diagnostic only and is never a contract transport.
Cancellation of the caller context propagates into Hero's provider request.

## MCP transport for chat-driven use

Hero's MCP server advertises exactly twenty v1 tools, one for each
authoritative operation:

```text
hero_code_host_<operation>
```

For example, use `hero_code_host_get_pull_request`,
`hero_code_host_comment`, or `hero_code_host_merge`. There is no generic
mixed-effect `hero_code_host_broker` tool.

Pass the typed `Request` fields as flat top-level tool arguments. Do not pass
`operation`; the selected tool fixes it. The top-level input schema is closed,
as are all nested repository, pull-request, reference, and mutation-payload
schemas. Unknown fields fail before broker construction or provider dispatch.
Mutation tools alone advertise `prepare`; set `prepare: true` for the
non-writing preparation call, then omit it for the separate execution call
with the returned revisions applied to the original arguments.

Every tool advertises the contract version, fixed operation, authoritative
effect, and required consent in metadata. Its MCP annotations are conservative
policy hints: read tools are read-only, only `merge` is destructive,
registry-declared replay-safe operations are idempotent, and provider-backed
operations are open-world. An annotation is not proof of semantic consent and
does not replace Hero Code's permission mode.

A tool call returns exactly one MCP text content item. The text is exactly one
structured `code-host-broker/v1` JSON envelope: `Response` for a read or
execution and `PreparationResponse` for mutation preparation. Decode and
validate that JSON as the contract result; do not scrape prose or treat MCP
transport errors as provider results.

## Explicit mutation preparation

Mutations are always two separate calls. Preparation performs bounded,
non-mutating provider reads and never silently continues into execution:

```bash
hero code-host broker comment --prepare < request.json > preparation.json
```

`PreparationResponse` is:

```json
{
  "version": "code-host-broker/v1",
  "operation": "comment",
  "capability_revision": "<revision>",
  "observation_revision": "<revision>",
  "error": null
}
```

On failure, `error` is populated and both revision fields are absent. The
preparation response deliberately does not echo the request or mutation
payload. Apply the two successful revisions to the original typed request,
retain the same user intent, consent, target, idempotency key, reconciliation
key, and payload, then make a separate execution call:

```bash
hero code-host broker comment < prepared-request.json
```

Read operations reject `--prepare`. A preparation result is observation
material, not user authorization. Hero Code must still enforce the policy's
consent requirement and its own execution permission mode before the separate
mutation call. `merge` is a `commitment` requiring `explicit_acceptance`; the
other v1 mutations require `explicit_user`.

## Retry and reconciliation

Never create a new key merely because a response was lost. Retry only the exact
canonical mutation with the same idempotency and reconciliation keys.
`ambiguous_result` requires refresh/reconciliation, not a blind provider write.
Treat `externally_completed`, `reconciled_applied`, and `replayed` as distinct
successful outcomes. Unknown additive fields and unknown advertised
capabilities remain inert but decodable.

Queue submission is not supported by the GitHub v1 adapter. A queue-required
repository advertises merge unavailable and Hero performs no direct merge.

## Vendoring

Hero Code's release-pinned fixture destination is:

```text
packages/hero-swift/Tests/HeroSharedApplicationTests/Fixtures/CodeHost/code-host-broker-v1.json
```

From the Hero Code repository root, use the released or clean locally built
Hero binary to copy and verify the exact decoded bytes:

```bash
contract_tmp="$(mktemp)"
hero code-host contract > "$contract_tmp"
jq -jr '.fixture' "$contract_tmp" > packages/hero-swift/Tests/HeroSharedApplicationTests/Fixtures/CodeHost/code-host-broker-v1.json
expected="$(jq -r '.sha256' "$contract_tmp")"
actual="$(shasum -a 256 packages/hero-swift/Tests/HeroSharedApplicationTests/Fixtures/CodeHost/code-host-broker-v1.json | awk '{print $1}')"
test "$actual" = "$expected"
```

The `-j` flag is required: appending a newline changes the canonical digest.
Record the clean Hero commit or release that supplied the fixture, then run the
consumer conformance test exactly:

```bash
swift test --package-path packages/hero-swift --filter CodeHostBrokerClientTests
```

Run every fixture case, including preparation, stale-freshness, additive
unknown-field, error, and reconciliation cases, before enabling the broker UI.
