# Tracker Broker v1

`tracker-broker/v1` lets a local client use a configured Hero tracker
connection without receiving its credential. Hero resolves the stable
connection ID, injects authentication inside the HTTP adapter or child
process, and returns one bounded JSON envelope.

The canonical Go types and golden fixture live under
`contracts/trackerbroker/`. A released binary prints the exact fixture:

```bash
hero tracker contract
```

## Operations

The in-process service, CLI, and MCP tools use the same request and response
types:

| Operation | CLI input | MCP tool |
|---|---|---|
| `get_issue` | `hero tracker broker get_issue` | `hero_tracker_get_issue` |
| `search` | `hero tracker broker search` | `hero_tracker_search` |
| `request` | `hero tracker broker request` | `hero_tracker_request` |
| `cli` | `hero tracker broker cli` | `hero_tracker_cli` |

CLI broker commands read one JSON request object from stdin and write one JSON
response object to stdout. Credentials must never be included in the request.

All requests accept optional `connection_id`. Omission succeeds only when the
workspace has exactly one tracker connection. Delivery roles and
`integrations.default` do not resolve broker ambiguity.

## Response envelope

Every response includes `version`, `operation`, `effect`, `truncated`, and
`duration_ms`. Once selection succeeds it also includes `provider` and
`connection_id`. HTTP responses use `status_code` and `body`; CLI responses use
`exit_code`, `stdout`, and `stderr`; normalized operations use structured
`result`. Searches may return opaque `next_cursor`. Failures use an `error`
object with stable `code`, safe `message`, and `retryable`.

Effects are authoritative:

- `read`
- `write_idempotent`
- `write_non_idempotent`

Consumers must not infer effects from operation prose. Cursors are opaque and
bound to the provider, connection, and exact query.

## Security boundary

`request` accepts only a relative path resolved against the configured tracker
origin. It rejects caller authentication/cookie headers, absolute and
scheme-relative URLs, userinfo, fragments, host-confusion forms, and
cross-origin redirects. Authentication is added only after validation.

`cli` accepts only provider-declared bare executable identities. Hero executes
exact argv without a shell, strips ambient tracker credential variables, adds
the selected credential only to the child environment, and binds the provider
host through documented environment variables. Auth/config/token-display
commands, host/header overrides, shell escapes, and credential literals are
rejected before executable lookup. Jira currently advertises no CLI because
its supported CLI does not provide a documented non-persistent, child-only
token environment contract; Jira broad REST access is available through
`request`.

All provider output and errors are byte bounded and credential-redacted.
Non-idempotent operations execute once with no automatic retry. Cancellation
propagates to HTTP requests and child processes.

## Compatibility

The v1 contract is additive: consumers must ignore unknown fields and error
codes. Removing or renaming fields, changing effect semantics, weakening
connection selection, or changing cursor binding requires a new contract
version. Hero Code should validate its decoder against `hero tracker contract`
from the released binary before enabling brokered tools.
