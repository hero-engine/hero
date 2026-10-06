# Delivery audit — hero-spec-handoff-tools

**Audited:** `git diff 3f75cfd6...df250e73`, specifically commit 74e2ef3d. That covers `internal/serve/mcp_tools_read_contract.go` (`toolSpec`, `toolHandoff`, `specBody`, `frontmatterValue`, `specRelations`, `specACs`), `internal/nextdoc/path.go`, `internal/cli/next.go`, `mcp_dispatch.go`, `mcp_tools_def.go` and the tests.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: `hero_spec` returns `{item, body, relations, acs}`, where `item` is the shared WorkItem with `next` and the body has no frontmatter.
  - Tested by `TestToolSpecReturnsContractShape`.
  - Real `hero mcp` calls (auditor, clean df250e73 binary, copy of this repo's `.hero`): `mail-b7ca19966ac5041e6ff604dd` → feature / in_progress / Continue, and `hero-read-contract` → initiative / in_progress / Drive with 7 children.
- [✓] AC-2: relations and ACs.
  - **Relations:** refs carry `missing` for absent targets, and empty lists serialize as `[]`. The same test covers both.
  - **ACs:** the test only covers the `unknown` path. The auditor exercised the pass path for real: `hero_spec acceptance-criteria-graph` returns AC-1 `pass` (from a `passing` Criterion node) and the rest `unknown`. The fail path (`failing`/`regressed` → `fail`) has no test, but it is symmetric code.
- [✓] AC-3: clear errors. `TestToolSpecRejectsNonWorkAndUnknown` checks "not a work spec" and "no spec with slug".
- [✓] AC-4: `hero_handoff` returns the same file `hero next` shows, through the shared `nextdoc.HandoffPath`. `TestToolHandoff` covers three cases: no briefing (`{"markdown":"","updated_at":null}`), plain frontmatter, and a managed block before the frontmatter. The real call returned 1830 characters with `updated_at` 2026-09-30T23:30:05Z.
- [✓] AC-5: read-only, with no writes under the watch globs.
  - `tools/list` advertises `readOnlyHint: true`.
  - **Auditor check:** I took an mtime and size snapshot of `.hero/planning`, `specs`, `knowledge`, `next`, `NEXT.md` and `hero.json` before and after real calls to `hero_spec` (5 slugs), `hero_handoff` and `hero_work` (10×). The snapshots are identical.
  - **Side effects:** the only file written is `.hero/index.db`, from `ensureFreshIndex` → `index.RefreshIfStale`. `graph.Open` can create or migrate `.hero/graph.db`. Both are runtime files outside the globs, and the contract allows re-indexing when stale.

## Changes
- [✓] Tools and registration (`safetyRead`). The tool count is 68 after hero_work; the ledger's "65→67" was this commit's intermediate step.
- [✓] `internal/nextdoc/path.go`. `internal/cli/next.go` now delegates `nextUserSlug` and `resolveNextPath` to it, and the CLI tests pass.
- [✓] Tests: `mcp_tools_read_contract_test.go`, plus the tool count and name list in `mcp_test.go`.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- **Inconsistent output, contract-level:** `hero_spec hero-domains` lists 12+ children under `relations.children`, because it also counts specs whose `parent:` points at it. Its `item.next` is still `Compose` ("no children") with lane `none`, because the model's rule counts only declared children (`spec.DeclaredChildren`). One response therefore contradicts itself. The contract defines "designed" by declared children, so this is not a delivery defect here. `read-contract-v1` should decide whether reverse parent edges count. There are 6 Compose items on the real corpus.
- **Does not match `hero next` exactly:** `hero next` appends the gitignored per-machine local state file. `hero_handoff` returns only the briefing file. That is probably intended, since it is per-machine, but the contract says "what `hero next` prints". Note it in the contract.
- **Child roster includes non-work specs:** `specRelations` gathers every spec whose parent is the target. That includes notes and other knowledge types.
- **Minor:**
  - `specBody` does not strip CRLF (`---\r\n`) frontmatter.
  - `frontmatterValue` takes the first `---` pair, so a Markdown horizontal rule inside a managed block before the frontmatter would mislead it.
- **Inherited:** `item.verify` and `item.next` carry work-item-model's fail-open sign-off defect (see its HOLD).
- **Re-run (auditor):** at a clean df250e73 worktree, the serve, nextdoc and cli tests pass, and `go vet` is clean.
