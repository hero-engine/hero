# Delivery audit — read-contract-conformance

**Audited:** commit 06e17116, at 8597d000. That covers `internal/serve/read_contract_schema_test.go`, `testdata/read_contract_v1.golden`, `scripts/export-read-contract-fixture.py`, `web/docs/src/cli/server-and-mcp.md`, and `slugRank` in `NewCorpus`. Built and run in a clean worktree at 8597d000.
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [~] AC-1: removing or retyping any field recorded in the v1 golden fails the suite.
  - **What works:** `TestReadContractV1SchemaIsAdditiveOnly` flattens all three replies into `path:type` entries and fails on any golden line that is no longer produced. `-update-read-contract` only adds lines, and the check runs first, so removals and renames are caught. The ledger's falsification (renaming `lane`) is consistent with how the test works.
  - **The gap:** the spec's Design says the fixture "exercises every shape". It does not. Several contract fields appear in the golden only as `:null`, or with no element shape at all, so retyping them passes:
    - `hero_work.items[].verify.audit` and `hero_spec.item.verify.audit`: only `:null` is recorded. No audited spec is in the fixture, so the `"ship"`/`"hold"` string type is unguarded.
    - `hero_work.items[].size`: only `:null`, never `:string`.
    - `hero_work.items[].next:null` is missing (finished or verified items), and so is `hero_work.items[].verify:null` (decisions).
    - `hero_handoff.updated_at:null` is missing (no briefing).
    - `hero_spec.relations.children[]` and `hero_spec.relations.blocks[]` have no element shape; only `:array` is recorded.
  - A client depending on any of these is not protected. The fix is small: add an audited spec, a sized item, a decision, a verified or finished item, a blocks edge, a child, and a no-NEXT.md handoff call to the fixture, then run `-update-read-contract`.
- [✓] AC-2: fixture export.
  - **Auditor run:** `scripts/export-read-contract-fixture.py --hero <8597d000 build> --project <8597d000 worktree> OUT` wrote `hero_work.json`, `hero_handoff.json`, 513 `hero_spec/*.json` files and `manifest.json` (`revision 6efdc906d18c54f1`, `source_commit 8597d000…`) in about 95 s.
  - It wrote nothing under the watch globs.
  - Every `hero_spec` item is identical to its `hero_work` item.
- [✓] AC-3: docs. The new "Read contract v1" section in `web/docs/src/cli/server-and-mcp.md` covers:
  - the three tools and their reply shapes
  - the lane, verify and next-step vocabulary
  - revision and watch-glob semantics
  - the additive-only policy and the golden file
  - the export script

  I did not re-run `mkdocs build --strict`.

## Changes
- [~] Golden schema test and golden. Present and working, but the coverage is incomplete (see AC-1).
- [✓] `scripts/export-read-contract-fixture.py`.
- [✓] Docs section.
- [✓] `slugRank` collision fix in `NewCorpus`. Its regression case is in `TestRound2AuditCases`. On the real corpus there are no duplicate items and no spec/work mismatches.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- **Blocker (AC-1):** close the golden's shape gaps listed above. Today `verify.audit`, `size`, a null `next`, a null `verify`, a null handoff `updated_at`, and the elements of the `children`/`blocks` relations can change type silently, which contradicts both the Design's "exercises every shape" and the spec's goal of locking v1.
- **Manifest:** `source_commit` comes from `git rev-parse HEAD` in `--project`, not from the Hero binary that produced the replies. Those differ when exporting another project. Consider also recording the binary's version or commit; `hero_version` is `"dev"` for local builds.
- **Docs wording:** the docs say the tools "write nothing", but each call may update `.hero/index.db` (and `graph.db`) as part of re-indexing. That is outside the watch globs and allowed by the contract, but the wording overstates it.
- **Re-run (auditor):** at 8597d000, the `serve`, `workmodel` and `cli` tests pass, and `go vet` is clean.
