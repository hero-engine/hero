# Delivery audit — tracker-source-reconciliation

**Audited:** canonical Jira ADF conversion, all description read surfaces, pagination regression, and completion ledger
**Verdict:** SHIP
**Surface:** standard

## Acceptance criteria

- [✓] Search/ListIssues, the paths used by bulk imports, receive the same canonical Markdown as item evidence.
- [✓] Nested lists, semantic nodes, panels, code blocks, links, media fallbacks, and malformed sibling recovery are fixture-covered.
- [✓] Paginated sprint reads preserve the same description conversion on every page.
- [✓] No historical refresh writeback was introduced; non-placeholder local authored content stays owned locally.
- [✓] The full CLI/tracker regression suite passes.

## Open items

None.

## Tests

- `go test ./internal/tracker ./internal/cli` — pass.
- `go test ./...` — pass.
