# Delivery audit — hero-spec-handoff-tools

**Audited:** commit 74e2ef3d plus the later changes in `git diff df250e73 8597d000 -- internal/serve/mcp_tools_read_contract.go`: signers passed through, `specRelations` built on `Corpus.Children`, and CRLF handling in `specBody`. This is re-confirmation round 4, built and run in a clean worktree at 8597d000.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: `hero_spec` returns `{item, body, relations, acs}`. Tested by `TestToolSpecReturnsContractShape`. In the auditor's full export, all 513 `hero_spec` items are byte-identical to their `hero_work` items, including `next` and the signer-resolved `verify`.
- [✓] AC-2: relations and ACs.
  - Children now come from `c.Children(target)`: declared children plus work specs naming it as parent. Knowledge and notes entries are excluded, because only work specs are indexed as children.
  - Real corpus: for every initiative, `relations.children` count equals `progress.total`. `hero-domains` has 20 children and Drive, so the round-3 contradiction is gone.
  - Declared but absent children report `status: missing`, as the contract specifies.
  - The pass path for ACs was confirmed for real in round 3: `acceptance-criteria-graph` AC-1 is `pass`.
- [✓] AC-3: clear errors. Tested by `TestToolSpecRejectsNonWorkAndUnknown`.
- [✓] AC-4: `hero_handoff`. Tested by `TestToolHandoff`. The round-3 amendment now defines the reply as the handoff file without per-machine local notes, which matches the code.
- [✓] AC-5: read-only, nothing written under the watch globs. The auditor snapshotted mtimes and sizes under all six globs before and after a full export (515 tool calls), and they are identical. Each call writes only the runtime index.

## Changes
- [✓] Tools and registration (`safetyRead`, `readOnlyHint: true`).
- [✓] `internal/nextdoc/path.go`, with the CLI delegating to it.
- [✓] Tests.
- [✓] Round 4: `Signers: s.ledgerSigners()` in `toolSpec`, `Corpus.Children` in `specRelations`, and `specBody` normalizing CRLF line endings.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- The round-3 notes are resolved: the children inconsistency, non-work children, CRLF frontmatter, and the local-notes wording, which is now in the contract.
- **Untested:** the suite has no case for CRLF frontmatter or for a knowledge entry pointing at an initiative. I verified both by reading the code. On the real corpus there are no non-work children apart from declared-but-missing slugs.
- **Behaviour change:** `specBody` now returns LF line endings for CRLF files. That is acceptable for Markdown display.
- **Re-run (auditor):** at 8597d000, the `serve`, `nextdoc` and `cli` tests pass, and `go vet` is clean. `hero_spec` takes 0.20–0.25 s end to end.
