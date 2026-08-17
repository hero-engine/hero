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

`direct-actions.json` publishes canonical typed requests for `hero_mail_send`,
`hero_mail_reply`, and `hero_focus_create`. Expected product failures use the
same versioned `ActionResult.error` envelope as advertised row actions; success
puts the authoritative Mail delivery or Focus item in `ActionResult.source`.

`conversational-routes.json` is the complete model-facing route corpus. Its
cases pin trusted-user, model-originated, untrusted-Mail, ambiguity, peering,
stale, unavailable, retry, and unknown-additive behavior to canonical
operations, tools or advertised actions, effects, consent classes, and exact
mutation counts.

The HTTP snapshot remains the full native projection. The MCP
`hero_attention_snapshot` adapter returns a compact metadata-only window from
that same authority: 8 rows by default, an explicit maximum of 20, full source
counts/revision, no row bodies, and no Mail summaries. Its additive `window`
object distinguishes `current` from a successful `empty` read; structured
`unavailable` is never an empty snapshot. Full Mail content remains an explicit
`hero_mail_show` read.

The exact SHA-256 of the canonical fixture inventory `manifest.json` is
`059b5418cc2005d506b0ca718df1ed25109ed022e385c7b16bca3e3c4a0d8e07`.
Within the vendorable conformance bundle this file is
`fixtures/manifest.json`. The contract endpoint advertises this fixture
checksum and schema version, not fixture bodies.

## Independent Mail-read contract

Cross-project Mail history, exact full detail, canonical Mail mutations, and
typed replies use the separate `contracts/attention/mailread/conformance/v1`
bundle over Hero Serve HTTP. Pin it independently from this Attention bundle:

- Mail-read bundle version: 1
- Mail-read schema version: 1
- Mail-read bundle manifest SHA-256: `69cc93c0a4566d1fd4f9678ead9d2fef3c7c9655559d2c55eb94551f0facb69e`

An absent or incompatible Mail-read bundle means Mail is unavailable. Do not
fall back to local storage or legacy MCP body decoding.
