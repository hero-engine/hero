# Delivery audit — mail-3ab437af53c66997e94a0268

**Audited:** `git diff f99496bad6710256a17cd99a24462aa338effd19..631608a`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC-1 — Jira broker attachment GETs force `redirect=false` and retain the bounded response path. `Broker.Request` invokes the narrow normalizer only after safe URL construction, and the regression asserts an overriding caller value becomes `false` and the body is returned (`internal/tracker/broker.go:349`, `internal/tracker/broker_test.go:378`).
- [✓] AC-2 — Unexpected broker cross-origin redirects remain fail-closed. The existing redirect callback rejects before dispatch, while the focused test asserts `unsafe_redirect` and a zero media-host call count (`internal/tracker/broker.go:369`, `internal/tracker/broker_test.go:418`).
- [✓] AC-3 — Typed Jira attachment downloads normalize the provider URL, preserve request-context cancellation, and return exact bytes. The implementation builds the request with the caller context; the focused regression asserts `redirect=false` and exact body bytes, and the existing cancellation test covers an in-flight attachment download (`internal/tracker/jira.go:833`, `internal/tracker/tracker_test.go:813`, `internal/tracker/jira_evidence_context_test.go:12`).
- [✓] AC-4 — Typed initial and redirected cross-origin URLs are rejected without target contact. The origin check occurs before request construction, the redirect check runs before credentials are reapplied, and both test shapes assert the media target remains at zero calls (`internal/tracker/jira.go:842`, `internal/tracker/jira.go:852`, `internal/tracker/tracker_test.go:821`).
- [✓] AC-5 — Same-origin redirects remain bounded and authenticated only after origin validation. The typed callback checks the shared five-redirect limit, then same-origin, then reapplies Jira headers; tests assert redirected Basic auth and the exact redirect-call bound. Existing broker tests cover the same-origin authenticated path and broker limit (`internal/tracker/jira.go:852`, `internal/tracker/tracker_test.go:834`, `internal/tracker/tracker_test.go:841`, `internal/tracker/broker_test.go:313`, `internal/tracker/broker_test.go:477`).
- [✓] AC-6 — Provider, method, path, and existing safety behavior remain isolated. New coverage asserts non-Jira, non-GET, and unrelated Jira requests preserve the caller query; existing regressions cover header rejection, redaction, cancellation, output bounds, and non-idempotent redirect rejection (`internal/tracker/broker_test.go:427`, `internal/tracker/broker_test.go:453`, `internal/tracker/broker_test.go:489`).

## Changes

- [✓] Shared Jira attachment URL normalization and typed redirect policy — `internal/tracker/jira.go` adds an exact v3 attachment-content predicate, replaces any caller redirect value, clones the HTTP client, and installs bounded same-origin redirect enforcement with stable sentinel errors (`internal/tracker/jira.go:831`, `internal/tracker/jira.go:879`).
- [✓] Broker normalization before authenticated dispatch — `internal/tracker/broker.go` applies the shared normalizer only to resolved Jira GET requests and leaves the existing authenticated transport and response/error contract in place (`internal/tracker/broker.go:349`).
- [✓] Security-boundary and compatibility regressions — `internal/tracker/broker_test.go` and `internal/tracker/tracker_test.go` exercise successful bodies, forced cross-origin redirects, zero target contact, same-origin auth, redirect limits, and provider/method/path isolation.

## Audit notes

- The two focused regressions and complete `internal/tracker` suite passed, and both `internal/tracker` and `internal/cli` test binaries compiled. `go vet ./...`, spec lint (6/6), and `git diff --check` also passed.
- Direct runtime execution of the larger CLI and full-repository test binaries was unavailable because macOS blocked newly linked binaries before Go runtime startup. This does not undermine the six acceptance criteria: no CLI code changed, the CLI/MCP adapters share the exercised `Broker.Request` path, and the complete changed package passed. It remains a transparent host-level validation limitation rather than a delivery blocker.
- The diff is scoped to the four named tracker files plus the received handoff/spec and Hero-managed projections.
