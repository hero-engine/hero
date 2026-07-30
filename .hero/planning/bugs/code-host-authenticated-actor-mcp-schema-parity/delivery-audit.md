# Delivery audit — code-host-authenticated-actor-mcp-schema-parity

**Audited:** `git diff 4b98d4c..defa3ede0775f92209ab18fd76d351a397e4aad2`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: authenticated-actor MCP input is repository-only — `internal/serve/mcp_tools_code_host.go:125` gives `OperationGetAuthenticatedActor` a base-fields-only branch; `TestCodeHostMCPAuthenticatedActorIsRepositoryScoped` asserts the exact required fields and absence of `pull_request`.
- [✓] AC-2: the canonical repository-only actor request dispatches unchanged and returns the canonical response — `TestCodeHostMCPCanonicalFixtureParityAllOperations` invokes the canonical fixture through the registered MCP handler, requires exactly one broker request, compares all request contract fields, and compares the transported response.
- [✓] AC-3: PR-scoped and other operation shapes remain intact — the production change is a single actor-specific switch case; the new regression confirms `get_pull_request` still accepts and requires `pull_request`, and the repository-wide suite passed.
- [✓] AC-4: focused MCP and repository-wide Go test prerequisites pass — focused actor/canonical parity tests and `go test ./... -count=1` passed; `go vet ./...` and spec lint also passed. A GoReleaser snapshot at the exact audited commit reran its `go test ./...` before hook successfully.

## Changes
- [✓] Classify authenticated-actor lookup as repository-only — `internal/serve/mcp_tools_code_host.go:125`.
- [✓] Add operation-shape regression coverage — `internal/serve/mcp_tools_code_host_test.go:121`.
- [✓] Qualify focused, package, repository, and snapshot builds — focused and repository-wide tests, vet, lint, and `goreleaser release --snapshot --clean` passed at `defa3ede0775f92209ab18fd76d351a397e4aad2`; GoReleaser built Darwin, Linux, and Windows archives for amd64/arm64 plus checksums, Homebrew formula, and Scoop manifest.
