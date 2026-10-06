# Delivery audit — work-item-model

**Audited:** `git diff df250e73 8597d000`, which covers `internal/workmodel/model.go`, `next.go`, `revision.go`, `internal/spec/signers.go`, `internal/cli/verify.go`, the tests, and the round-3 amendment to `read-contract-v1`. This is re-audit round 4, built and run in a clean worktree at 8597d000.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: one item per work spec, in path order. A real `hero_work` call at 8597d000 returned 513 items with no duplicate slugs. `NewCorpus` now ranks colliding slugs: work specs first, then other specs, then intakes.
- [✓] AC-2: lanes.
  - Initiative "designed", "started" and progress now use `Corpus.Children`: declared children plus *work* specs whose `parent` names the initiative (model.go:136-160), as the round-3 amendment says.
  - `TestRound3AuditCases` checks reverse-parent children: progress 1/2, `in_progress`, Drive.
  - Real corpus: `hero-domains` is in_progress / Drive with progress 17/20. Compose now appears only for `environment-awareness` and `retrieval-quality`.
- [✓] AC-3: verify state matches Gate 1.
  - `ledgerAllDone` now calls `ResolveSigners` with `Options.Signers` (model.go:408-415). A nil signer set fails closed.
  - Gate 1's `knownSigners` (`internal/cli/verify.go:713`) now delegates to the new `spec.KnownSigners`, so the gate and the model share one rule.
  - Both tools pass `s.ledgerSigners()`.
  - Tests in `TestRound3AuditCases`:

    | Case | Result |
    |---|---|
    | Known signer | passed |
    | Free-text signer | partial, Verify offered |
    | Non-DONE row | partial |
    | Unfinished spec with a stale SHIP audit | not_run, audit null |
    | Nil signers | partial |

    The last three close the round-2 minor gaps.
  - Real corpus: `token-efficiency-pass` is now `partial` with audit `ship`.
- [✓] AC-4: normalization and progress. No change apart from the wider set of children.
- [✓] AC-5: revision.
  - An initiative's revision now includes its reverse-parent children (revision.go:30).
  - Signer resolution feeds the verify state, which is part of the derived input.
  - Ten real calls returned one revision.
  - Every `hero_spec` item in a full 513-item fixture export is byte-identical to its `hero_work` item.

## Changes
- [✓] `internal/workmodel/model.go`: `Options.Signers`, `Corpus.Children` and `slugRank`. `Designed` now takes the corpus.
- [✓] `internal/spec/signers.go` (new): `KnownSigners`.
- [✓] `internal/cli/verify.go`: `knownSigners` delegates to `KnownSigners`.
- [✓] Tests: `TestRound3AuditCases`.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- All round-3 findings are resolved.
- **Re-run (auditor), at a clean 8597d000 worktree:** the `workmodel`, `serve`, `spec`, `nextdoc`, `install` and `cli` tests pass. `go vet` is clean, and `gofmt` reports nothing in the changed files.
- **Minor, latent:** for `epic`, `NextFor` uses `Corpus.Children` (Compose or Drive), but `Designed` falls into the feature branch (needs Changes and ACs). Progress is null and the revision skips children. Lane and next step can therefore disagree for an epic. There are no epics in this corpus.
- **Cost:** `spec.KnownSigners` runs `git log` over the full history on every read. Measured end to end, `hero_work` took 0.19–0.25 s and `hero_spec` 0.20–0.25 s, both well within budget. Exporting all 513 specs takes about 95 s.
