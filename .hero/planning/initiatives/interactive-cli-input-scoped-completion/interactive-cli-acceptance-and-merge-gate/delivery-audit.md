# Delivery audit — interactive-cli-acceptance-and-merge-gate

**Audited:** repair `git diff 6870da2...107272f`, integrated result `git diff main...107272f`, and final-gate artifact history `git diff 8581c07...107272f`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] **AC-1 — every production hunk maps to a parent AC and owning child** — `scope-provenance.md`, `acceptance-evidence.md`, the final-child ledger, and parent Progress now consistently distinguish 229 production-behavior hunks across 27 `internal/cli` files from four test-only portable PTY hunks in `internal/ptytest`. The production hunk table remains complete.
- [✓] **AC-2 — boundary-excluded production work is absent** — the integrated production diff contains only the scoped prompt/TTY, setup/connect, six-target, and frozen-selector work. `git diff 6870da2...107272f -- internal` is empty, so the repair introduces no production or test change.
- [✓] **AC-3 — every donor change has a durable disposition** — every donor commit group remains represented in `donor-branch-disposition.md`. The repaired `c191f6d` destination resolves exactly at `c191f6d:.hero/planning/features/cli-test-isolation-stray-workspace-boundary/spec.md` under `git cat-file`; no row now relies on the previously nonexistent path.
- [✓] **AC-4 — donor branch is retained while donor-only work remains** — `design/interactive-cli-input` remains present and explicitly non-deletable. The reconciliation distinguishes ported work, deliberate drops, and donor-retained work with exact commit/path evidence.
- [✓] **AC-5 — prompt-policy matrix covers terminal, pipe, JSON, NEVER-PROMPT, and secrets** — `acceptance-evidence.md` maps the prompt package, baseline, JSON, NEVER-PROMPT, open-writer pipe, and protected-terminal behavior to named tests. The archived prompt/TTY child and its SHIP audit corroborate the implementation evidence.
- [✓] **AC-6 — connect equivalence reaches the effective resolver** — the evidence map names paired interactive/flag persistence, role routing, JSON, live-pipe, and `ResolveCodeHostConnection` tests; the archived setup/connect child has a clean SHIP audit.
- [✓] **AC-7 — all six targets have install/uninstall evidence** — the six-target picker contract and original four plus real Copilot/generic uninstall round trips cover target parity, removal, and adjacent/shared-file preservation.
- [✓] **AC-8 — frozen selector targets have the required matrix** — the archived selector child and evidence map cover the frozen adopter set, 0/1/25/26/250 candidate boundaries, outside-first-25 selection, cancellation, invalid input, explicit values, non-TTY, and JSON paths.
- [✓] **AC-9 — repository/platform/Hero validation passes** — the supplied evidence records the full suite, affected CLI/prompt race suites, vet, native build, Windows build and prompt compile, portable protected-console seam, focused live-pipe/setup/uninstall checks, diff hygiene, Hero lint 12/12, score 95/A, clean drift, and index. Repair checks also report lint 12/12, 95/A, clean drift/index, and clean diff checks.
- [~] **AC-10 — fully DONE ledger and fresh SHIP audit precede verification** — substantive acceptance and Changes rows are supported, and this report supplies the required fresh SHIP before either verification command. The row remains a concrete temporal gate in the audited artifact because it cannot cite an audit that did not yet exist; root must disposition it explicitly before verification without claiming that verification already ran.
- [✓] **AC-11 — production defects return to owning children** — neither the final-gate artifact commit nor the bounded repair changes `internal/`; this audit found no production defect to route back to a child.
- [✓] **AC-12 — parent/child progress, audit, and code state agree** — the parent Specs table now reads completed/completed/completed/delivering. Each archived implementation child has `status: completed`, a post-verification Kickoff, and an explicit link to its archived SHIP audit; the final child alone remains delivering pending the external closing gates.

## Changes

- [✓] **1 — complete scope-provenance and donor-disposition maps** — the provenance map accounts for 229 production hunks plus four separately classified test-only PTY hunks, and the corrected `c191f6d` path resolves exactly.
- [✓] **2 — build the initiative-wide test ledger** — `acceptance-evidence.md` maps every parent/final-gate behavior family to named test and validation evidence.
- [✓] **3 — run the clean-successor validation matrix** — the supplied normal/race/vet/build/Windows/seam/live-pipe/diff/Hero result set is complete and internally consistent; the repair changed no implementation file.
- [✓] **4 — produce the ledger and commission a cold audit** — the prior independent HOLD report existed and drove bounded truth repairs; this replacement report is the fresh independent re-audit and returns SHIP.
- [~] **5 — reconcile progress and run both Hero verify commands** — progress is reconciled and the fresh SHIP now exists. The child-first and parent-second verification commands remain intentionally unrun because they are the external closing operations that follow this audit.

## Open items

- **AC-10 — `BLOCKED`, concrete temporal gate.** Root should not falsely mark the row `DONE` as though its precondition existed at commit `107272f`. Before verification, append a delivery-lead sign-off such as `[signed-off] Fresh SHIP audit exists at interactive-cli-acceptance-and-merge-gate/delivery-audit.md; this temporal row could not cite its own future audit.` The ledger contract expressly permits a signed-off `BLOCKED` row at Gate 1.
- **Changes 5 — `BLOCKED`, concrete sequential closing gate.** Before verification, append `[signed-off] Progress reconciliation is complete; child verify followed by parent verify are external closing operations and are intentionally not preclaimed.` Then run child verification first and parent verification second. This preserves the truth that neither command had run when the ledger was audited and avoids a self-referential `DONE` claim.

## Audit notes

- All prior HOLD findings are closed by the bounded artifact repair: count terminology is consistent, the exact donor destination resolves, the parent status table is reconciled, and all three archived child Kickoffs are post-verification and audit-linked.
- The repair is artifact-only: `git diff 6870da2...107272f -- internal` is empty, and both the repair and integrated diff checks are clean.
- This SHIP is `noteworthy` only because AC-10 and Changes 5 are honest sequential gate rows requiring the explicit sign-off-and-verify procedure above; no implementation or evidence defect remains.
