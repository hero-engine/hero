# Delivery audit — project-mail-thread-lifecycle-mcp-parity

**Audited:** working-tree diff at `4d5aa4c13800` using `git diff -- contracts/attention internal/attention/conformance internal/serve docs/serve.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: thread list returns authoritative buckets, counts, paging, and structured errors — `internal/serve/mcp_tools_mail.go:173` builds the portable request and delegates to `mailquery.Service.Threads`; `TestMCPMailThreadDirectDispatchMatchesHTTP` and `TestMCPMailThreadPagingAndStructuredErrors` assert JSON-RPC/HTTP parity, continuation paging, stale cursor, validation, and unavailable envelopes.
- [✓] AC-2: thread show returns authoritative composite-identity detail — `internal/serve/mcp_tools_mail.go:199` delegates directly to `ThreadDetail`; `internal/serve/mcp_tools_mail_test.go:321` compares the exact HTTP/MCP response and asserts the authoritative message body.
- [✓] AC-3: thread action preserves validation, revisions, and idempotency — `internal/serve/mcp_tools_mail.go:207` parses the decimal int64 revision, preserves nested identity/input, and delegates to `ThreadAction`; `TestMCPMailThreadActionPreservesRevisionInputAndIdempotency` asserts resolve input, replay, conflict, and stale revision.
- [✓] AC-4: thread contract matches HTTP — `internal/serve/mcp_tools_mail.go:240` constructs the response from canonical `mailthread` constants; `internal/serve/mcp_tools_mail_test.go:332` compares HTTP/MCP bytes and validates the contract. The ledger also supplies a successful real stdio MCP exercise.
- [✓] AC-5: exactly four additive thread tools are advertised and dispatched with accurate metadata — definitions are at `internal/serve/mcp_tools_def.go:72`, handlers are mapped at `internal/serve/mcp_dispatch.go:58`, and `contracts/attention/conformance/v1/mcp-tools.json:613` pins the generated action/read metadata and nested schemas. `TestMCPMailDefinitionsAndDispatch` and `TestMCP_ToolsList` assert the runtime inventory.
- [✓] AC-6: five legacy Mail tools remain compatible — the diff adds handlers without changing the legacy implementations; `internal/serve/mcp_tools_mail_test.go:55` asserts legacy body/action shapes and `:245` asserts all five legacy definitions/handlers remain beside the four additions. Supplied `internal/serve`, race, and repository-wide test evidence is green.
- [✓] AC-7: direct MCP dispatch and HTTP/MCP parity cover success and failure paths — `callTool` routes through real JSON-RPC `tools/call` at `internal/serve/mcp_test.go:657`; the three focused thread tests exercise all four tools plus paging, validation, unavailable, lifecycle mutation, replay, conflict, and stale paths.
- [✓] AC-8: the Attention inventory and derived hashes were regenerated while the independent Mail-thread checksum stayed fixed — the generated inventory contains all four tools, `go run ./cmd/attention-conformance --check` independently returned `2c29a1c6e04c3504969736494d0759f566a982d01c59ab7d8552c751a64b31fa`, and the compiled/test pins match. `contracts/attention/mailthread` has no working-tree diff and `contracts/attention/mailthread/contract.go:14` remains `5438ed2da3e91d988a896656f7bc42a878f46bb6f1aa79811061bfe744abcd5a`.

## Changes

- [✓] Add four exact definitions and dispatcher mappings — `internal/serve/mcp_tools_def.go:72` defines the four schemas and annotations; `internal/serve/mcp_dispatch.go:58` maps the three reads and `:83` maps the action.
- [✓] Add a registry-backed query seam and thin handlers — `internal/serve/mcp.go:70` adds the injectable factory; `internal/serve/mcp_tools_mail.go:173` adapts transport arguments and delegates list/show/action to `mailquery.Service` without duplicating lifecycle policy.
- [✓] Add metadata, direct-dispatch, parity, paging, lifecycle, error, and compatibility coverage — `internal/serve/mcp_tools_mail_test.go:245` contains the focused matrix and `internal/serve/mcp_test.go:409` pins the global runtime tool inventory.
- [✓] Regenerate and pin the consumer-facing Attention MCP mapping — canonical `contracts/attention/testdata/v1/HERO-CODE-HANDOFF.md:46`, generated `contracts/attention/conformance/v1/HERO-CODE-HANDOFF.md:46`, and `docs/serve.md:52` each name the four exact tool-to-response mappings and state bundled MCP needs no `hero serve` or HTTP daemon. `internal/attention/conformance/builder_test.go:41` asserts all five snippets across all three surfaces; the independently rerun focused test and generator check passed.

## Open items

- None.

## Audit notes

- None.
