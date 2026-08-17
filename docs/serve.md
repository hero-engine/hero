# Hero Serve

Hero Serve hosts project-scoped APIs under `/api/{project}/...` and the
user-global Attention v1 API under `/api/attention/v1`. Attention routes are
registered before the generic project router, so they work before a project is
opened and `attention` is never interpreted as a project slug.

## Attention v1

`GET /api/attention/v1/snapshot` returns unread active Project Mail, Today
Focus, and pending unexpired deferred-work suggestions. The response is a full
active snapshot: v1 has no pagination or event cursor.

`POST /api/attention/v1/actions` accepts only a row ID, an advertised action
ID, its source revision, an idempotency key when required, and action-specific
input. It delegates to Mail, Focus, or Suggestion ownership and returns
authoritative source state plus the new snapshot revision. There is no generic
Attention write endpoint.

`GET /api/attention/v1/contract` advertises schema version 1 and the exact
fixture-manifest checksum.

Clients fetch on mount, foreground, reconnect, and after successful mutation.
A stale, unsupported, missing, validation, or incompatible-version result
causes a refresh and user re-selection, never automatic mutation replay.
Attention does not depend on `/api/events`. An unavailable store or server is
reported as `unavailable`, never as an empty snapshot; clients may retain an
explicitly labelled stale snapshot in read-only form.

The projection validates a Focus row revision immediately before asking the
Focus service for a launch intent. Launch is read-only; a concurrent change is
resolved by the mandatory post-action snapshot refresh rather than replay.

## Mail-read v1

Hero Code uses the additive user-global Hero Serve HTTP Mail transport:

- `GET /api/attention/v1/mail/messages` returns bounded metadata-only pages.
- `GET /api/attention/v1/mail/messages/{message_id}?project_peer_id=...`
  returns one complete validated envelope without changing its receipt.
- `POST /api/attention/v1/mail/actions` dispatches a currently advertised
  canonical action against `(project_peer_id, message_id)`.
- `POST /api/attention/v1/mail/replies` sends a typed, project-scoped reply.
- `GET /api/attention/v1/mail/contract` advertises the independently pinned
  Mail-read schema, bundle version, and manifest SHA-256.

Message identity is the composite `(project_peer_id, message_id)`, and thread
scope is `(project_peer_id, thread_id)`. The list never includes bodies. Exact
detail is typed JSON written directly by Hero Serve, so a valid 65,536-byte
body does not pass through MCP text normalization or silent truncation.

The existing `hero_mail_list`, `hero_mail_show`, and `hero_mail_action` tools
remain model-facing compatibility surfaces. Their request/response shapes and
legacy `read` action are unchanged. Hero Code must not combine HTTP reads with
MCP mutations or fall back to local Mail files or legacy MCP decoding when the
Mail-read contract is unavailable or incompatible.
