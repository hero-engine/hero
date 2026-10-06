# Delivery audit — stale-connect-provider-usage-expectation

**Audited:** `git diff -- internal/cli/prompt_setup_commands_test.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- ✓ AC-1: Expected non-terminal usage includes `aha`; pipe and closed stdin cases pass with existing assertions preserved — `internal/cli/prompt_setup_commands_test.go:133` matches runtime usage at `internal/cli/connect.go:103`. The one-line diff preserves timeout, nonzero exit, prompt suppression, and piped-input refusal assertions at test lines 143–159. `/tmp/hero-aha-before.log` shows both cases failing solely on the obsolete string; `/tmp/hero-aha-after.log` records the focused provider suite passing.

## Changes
- ✓ Update only `wantErr` to include `aha` between `gitlab` and `confluence` — the supplied diff changes exactly that constant at `internal/cli/prompt_setup_commands_test.go:133`.

## Audit notes
- No missing ledger rows, partial work, or scope drift in the audited diff.
- The reported command `go test -race ./internal/cli -run '^TestConnectProvider' -count=1` selects all seven existing provider tests, including the real CLI pipe/closed cases. Its passing output was inspected. `go build ./...` and `git diff --check` passed according to the supplied execution evidence. No experiments were rerun during this audit.
