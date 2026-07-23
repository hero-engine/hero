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

The exact SHA-256 of `manifest.json` is
`f632733b4625c57983bdba98e4a9f58818f76ce571bf05c91722d8620b4697f6`.
The contract endpoint advertises this checksum and schema version, not fixture
bodies.
