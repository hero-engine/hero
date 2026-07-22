# Delivery audit — tracker-project-snapshot-contract

**Audited:** contract, Jira snapshot loader, CLI boundary, focused tests, and full Go suite
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] The contract is versioned and provider-neutral.
- [✓] Active/future sprint identity, dates, membership, native/normalized status, owner, rank, and URL are represented.
- [✓] Board, sprint, and issue collection paginate; no first-page truncation is hidden.
- [✓] Local slug joins are read-only and happen at the CLI boundary.
- [✓] The schedule loader requests only four issue fields and does not bulk-hydrate item evidence.

## Open items

None for the explicit-refresh v1. Incremental cursor semantics are reserved for later and are not claimed.

## Tests

- `go test ./internal/tracker ./internal/cli` — pass.
- `go test ./...` — pass.
