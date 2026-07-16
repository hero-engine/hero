# Delivery audit — agents-md-erased-by-snapshot-pointer-writer

**Audited:** `git show fa3b339` (scoped to `internal/snapshot/`, `internal/cli/snapshot.go`, `internal/cli/checkpoint.go`, `internal/serve/mcp_tools_snapshot.go`, `internal/install/harness_smoke_test.go`, `AGENTS.md`) plus current HEAD (`8f89724`) for test coverage and outstanding-item follow-through.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria / ledger rows

- [✓] Eraser removed at the source — `internal/snapshot/pointers.go:34` is `func EnsurePointer(nextPath string) error`; the `agentsPath` parameter is gone, not guarded. Doc comment (pointers.go:17-33) records the constraint and both erasure incidents. Fresh `grep -rn "AgentsMDPath\|agentsMDPathOrEmpty" --include='*.go'` over the whole tree: zero hits.
- [✓] Dead wiring removed — `ProjectOptions.AgentsMDPath` and `agentsMDPathOrEmpty` absent from the tree; `internal/cli/snapshot.go` (-23 lines), `internal/cli/checkpoint.go` (-20), `internal/serve/mcp_tools_snapshot.go` (-1) all touched in fa3b339. The only remaining `EnsurePointer` call site is `internal/snapshot/projector.go:119`, targeting NEXT.md only.
- [✓] Unit guard — `TestEnsurePointer_DoesNotWriteInstallManagedFiles` (internal/snapshot/pointers_test.go:81). Read the body: seeds realistic install-managed AGENTS.md *and* CLAUDE.md, runs `EnsurePointer`, asserts both files byte-identical. Real assertion, not an import-and-smile test. PASS under `-race -count=1`.
- [✓] Sharp edge characterized — `TestWritePointerOnly_ReplacesEntireManagedRegion` (pointers_test.go:50). Asserts the foreign section is *gone* after `writePointerOnly` — genuinely pins replace-not-merge semantics with a failure message directing a future merge-author to re-evaluate the restriction. PASS.
- [✓] Durable cross-target guard — `TestHarness_InstalledContentSurvivesOrdinaryCommands` (internal/install/harness_smoke_test.go:330). Verified table-driven over all six targets (claude→CLAUDE.md; codex/opencode/cursor/copilot/generic→AGENTS.md), runs `snapshot.Project` between install and assert, requires byte-identity plus the shared-body line "Finish the closing gate before yielding". On HEAD it additionally asserts `CheckIntegrity` reports clean (extension from install-integrity-self-check). PASS.
- [✓] "Verified RED when eraser reintroduced" — **independently reproduced, not taken on faith.** In a scratchpad clone, reintroduced a `writePointerOnly(<root>/AGENTS.md, ...)` call in `Project`; the guard test failed on all five AGENTS.md targets with `size 19433 -> 147` — the exact byte figures the ledger claims — and `CheckIntegrity` flagged the missing sections too. Clone deleted after the run.
- [✓] All tests pass — ran `go test -race -count=1 ./internal/snapshot/ ./internal/install/` myself: both `ok` (1.3s / 3.0s). Named guard tests re-run with `-v`: PASS.
- [✓] Root cause confirmed — cannot re-run the old-binary reproduction, but the mechanism is directly evidenced: `writePointerOnly` (pointers.go:45-53) composes a single-section `managed.Writer`, and the characterization test proves region replacement. The 19433→147 reproduction above is the same failure mode live.

## Outstanding rows (ledger) — all three subsequently satisfied

- Repo's own AGENTS.md repaired — **now done.** `AGENTS.md` on disk and at HEAD is 232 lines, restored inside fa3b339 itself (+227 lines); contains the full install doctrine ("Finish the closing gate before yielding", "Running Hero Workflows in Codex", agents/skills/CLI reference).
- `codex-install-broken` record corrected — **now done.** `.hero/specs/codex-install-broken/spec.md` frontmatter carries `superseded_by: agents-md-erased-by-snapshot-pointer-writer` plus a full `superseded_reason` explaining the misdiagnosis; superseding spec declares `supersedes: [codex-install-broken]`.
- Detection advisories (Fix 4 / Fix 6) revived — **now done.** `.hero/specs/install-integrity-self-check/spec.md` is `status: completed`, `completed_at: 2026-07-16T07:04:09Z`, archived with its own delivery-audit.md; its `CheckIntegrity` oracle is wired into the guard test at harness_smoke_test.go:408.

## Audit notes

- **No performative rows.** Every DONE row has diff or on-disk evidence; the strongest claim (guard goes red) was reproduced independently and matched byte-for-byte.
- **Ledger is stale in the honest direction:** the three OUTSTANDING rows are all satisfied as of HEAD but still read OUTSTANDING, and the Kickoff still says "uncommitted in the working tree." Spec status is `delivering`. Flip the three rows (with evidence pointers) before `hero spec verify`.
- **Mixed commit:** fa3b339 also carries unrelated work (skill-dir pruning, `.agents/skills` untracking). Known and disclosed in the commit message; excluded from this audit's scope. Noted only so the delivery record is honest about the commit not being single-purpose.
- Spec cites `harness_smoke_test.go:330` for the guard test — still accurate on HEAD despite later extension.
