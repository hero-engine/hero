# Delivery audit — read-contract-conformance

**Audited:** `git diff 8597d000 00cc6eeb` together with the original commit 06e17116. The files in scope are `internal/serve/read_contract_schema_test.go`, `testdata/read_contract_v1.golden`, `scripts/export-read-contract-fixture.py`, `web/docs/src/cli/server-and-mcp.md` and `slugRank`. This is re-audit round 2, built and run in a clean worktree at 00cc6eeb.
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: removing or retyping a field recorded in the golden fails the suite, and new fields are allowed.
  - **Golden growth:** 151 → 182 lines, purely additive (`--numstat`: 31 added, 0 removed).
  - **Round-1 gaps closed:** every shape from the HOLD is now recorded:
    - `items[].verify.audit:string` and `item.verify.audit:string`
    - `items[].size:string`
    - `items[].next:null` and `item.next:null`
    - `items[].verify:null` and `item.verify:null`
    - `hero_handoff.updated_at:null`, alongside `:string`
    - element shapes for `relations.children[]` and `relations.blocks[]`
  - **Explore item:** `suggested[].slug:null` comes from a separate thin-backlog `hero_work` call.
  - **Falsified (auditor):** in the scratch worktree only, I made `Size` nil and dropped the audit verdict. `TestReadContractV1SchemaIsAdditiveOnly` then failed, naming exactly these paths:
    - `items[].size:string` and `item.size:string`
    - `items[].verify.audit:string` and `item.verify.audit:string`
    - `items[].next:null` and `item.next:null`

    I restored the file afterwards, leaving the scratch tree clean.
- [✓] AC-2: fixture export.
  - **Auditor run:** `scripts/export-read-contract-fixture.py` against a real `hero mcp` built at 00cc6eeb wrote 513 `hero_spec` files, `hero_work.json`, `hero_handoff.json` and a manifest with `project_commit 00cc6eeb…` and revision `9eaee9317b197733`.
  - Nothing was written under the watch globs.
  - My first attempt failed with ENOSPC on the host disk. That was an environment problem; the rerun succeeded.
- [✓] AC-3: docs.
  - The section now says the tools never write under `watch_globs` and that only runtime index and graph files may refresh, which matches what I observed.
  - `docs-build.log` shows `mkdocs build --strict` completing.

## Changes
- [✓] Golden schema test and golden. The fixture is enriched: a verified, sized, audited item; a decision child; a `blocks` edge; initiative children; a handoff call before NEXT.md exists; and a thin-backlog call.
- [✓] Export script. The manifest key is now `project_commit`, and the docstring says it is the exported project's commit.
- [✓] Docs.
- [✓] `slugRank`. Unchanged since round 1.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- None. All round-1 findings are resolved.
- **Re-run (auditor) at 00cc6eeb:** `go test ./internal/serve -run 'ReadContractV1|ToolWork|ToolSpec|ToolHandoff'` passes, and `go vet ./internal/serve` is clean.
