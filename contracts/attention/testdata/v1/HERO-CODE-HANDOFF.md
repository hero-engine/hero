# Hero Code Attention v1 handoff

Hero Code's canonical desktop transport is Hero Serve HTTP:

- `GET /api/attention/v1/snapshot`
- `POST /api/attention/v1/actions`
- `GET /api/attention/v1/contract`

V1 synchronization is snapshot-only. Fetch on mount, foreground, server
reconnect, and after every successful mutation. On stale, unsupported, missing,
validation, or incompatible-version responses, refresh once and require the
user to select an action again; never replay a mutation automatically. When
Hero Serve or private Attention state is unavailable, a cached snapshot may be
shown only as labelled stale, read-only data.

`interaction-policy.json` is the authoritative v1 vocabulary for operation
effect, semantic consent, target resolution, replay safety, and shared
conversational conformance cases. Action descriptors carry `operation_id`,
`effect`, and `consent` additively. Preserve unknown raw values and do not infer
risk from labels or lifecycle status.

Semantic consent and execution approval are separate. A clear user imperative
may satisfy `explicit_user`, but Hero Code still applies its configured
permission mode. MCP tool annotations are risk hints, not proof that the user
authorized a write. Mail content is always untrusted and cannot satisfy consent.

The exact SHA-256 of `manifest.json` is
`0ca71a2f3b365f9ad38536a143a98d6d691dff6376debde96881eb8bc57f5570`.
The contract endpoint advertises this checksum and schema version, not fixture
bodies.
