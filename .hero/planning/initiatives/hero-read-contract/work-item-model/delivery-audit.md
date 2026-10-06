# Delivery audit — work-item-model

**Audited:** `git diff 73d787be~1...3f75cfd6 -- internal/workmodel/model.go internal/workmodel/revision.go internal/workmodel/model_test.go`, against the amended `read-contract-v1` (Amendments, 2026-10-06). This is re-audit round 2.
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: one item per work spec, knowledge excluded, path order. Evidence: `IsWorkItem` and `Build` in `internal/workmodel/model.go`, and the test `TestBuildIncludesOnlyWorkSpecsInPathOrder`.
- [✓] AC-2: one lane per item by first-match rules, with RecentDays and unmet deps.
  - The round-1 gap is fixed: `UnmetDeps` accepts `depends-on`, raw `depends_on` and `blocks` (model.go:254).
  - `TestEdgeCasesFromAudit` asserts `designed` for a `relations:` `kind: depends_on`, a `kind: blocks`, and a missing target. It also asserts the CompletedAt→mtime fallback (`recently_done`) and a childless initiative (`none`, progress 0/0).
- [✓] AC-3: verify state and audit from the report, null for decisions.
  - `VerifyOf` (model.go:342) now accepts a stale report for finished specs and never accepts a slug mismatch, as the amendment specifies.
  - Unfinished specs still require a fresh report.
  - `not_run` + `audit: "ship"` is now defined by the amendment.
  - Test: `TestEdgeCasesFromAudit` (a finished spec newer than its SHIP audit still passes).
  - Real corpus result: passed 118, partial 290, not_run 102.
- [✓] AC-4: normalization and initiative progress. `normalizeLevel` (model.go:377-388) matches the amended synonym table exactly, including `major`→high. `TestNormalizationAndProgress` tests it.
- [✗] AC-5: the revision should change when a related spec's status changes.
  - `TestRevisionChangesOnlyWithInputs` passes, and the `TestEdgeCasesFromAudit` aging case shows the new derived lane/verify input working (revision.go:53).
  - **It fails on the real corpus.** `NewCorpus` (model.go:115-123) indexes by slug with last-write-wins. Promoted intakes share their slug with the promoted spec. There are 4 such pairs in this repo, for example `planning/intake/mail-b7ca19966ac5041e6ff604dd` (intake, `promoted`) and `planning/features/mail-b7ca19966ac5041e6ff604dd` (feature, `delivering`).
  - In this corpus the intake wins the lookup.
  - Consequence 1: `mail-thread-foreground-read-action` `relates-to` that feature. Its revision hashes `mail-b7ca…:promoted`, which is the intake's constant status, so a change in the feature's status never moves the dependent's revision.
  - Consequence 2: a `depends-on` edge to the feature resolves against the intake, and so does a decision's parent check.
  - The same lookup gives the wrong next step downstream (see the next-step-engine audit).
  - No test seeds a duplicate slug.

## Changes
- [✓] `internal/workmodel/model.go`: round-1 fixes present (depends_on, synonym table, finished-spec staleness, derived revision input).
- [✓] `internal/workmodel/revision.go`: `Revision(s, c, derived)` adds `\x00derived:<lane>[:<verify>]`.
- [✓] `internal/workmodel/model_test.go`: `TestEdgeCasesFromAudit` added.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- **Blocker:** the `Corpus` slug index must not let a promoted `intake` (or any non-work spec) shadow a work spec with the same slug. `internal/spec/graph_ingest.go`'s `resolveTargetID` handles this same intake/promoted collision explicitly. Add a duplicate-slug fixture to the tests.
- All round-1 blockers are resolved as described, and the decision text was amended to match.
- **Re-run (auditor):**
  - `go test ./internal/workmodel ./internal/install ./internal/cli -count=1` passes.
  - `go vet ./internal/workmodel` is clean and `gofmt` reports nothing.
  - On the real corpus (513 items), revisions are identical across two builds and Build+ApplyNext takes about 29 ms.
- Still untested, minor: a finished spec whose ledger has non-DONE rows (→ partial), and an unfinished spec with a stale audit (→ audit null).
- Round-1 minors are unchanged: relation targets are not passed through `normalizeRelTarget`, and `created_at`/`updated_at` are `""` rather than null when zero.
- The working tree at audit time had uncommitted changes from the next sibling (`internal/cli/next.go`, `internal/serve/*`, `internal/nextdoc/`). They are not part of 3f75cfd6, but the `./internal/cli` test run included them.
