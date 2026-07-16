# Delivery audit — install-integrity-self-check

**Audited:** `git diff HEAD` (uncommitted, intent-to-add staged) on branch `fix/agents-md-erased-by-snapshot-pointer-writer`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC-1 missing sections → fail naming target/file/sections — `internal/install/integrity.go` section-presence check; `TestCheckIntegrity_DetectsGuttedRegion` (all six targets, asserts kind/file/sections/repair cmd). Reproduced live: scratch fixture, codex install, region gutted to the incident stub → `fail`, "missing section(s): Hero — Spec-Driven AI Engineering, Hero Binary & MCP Surface".
- [✓] AC-2 body differs, sections present → warn stale — `TestCheckIntegrity_DetectsStaleBody` (six targets, asserts `IntegrityStale`, empty `MissingSections`). Reproduced live: one-phrase in-region mutation → `warn` stale.
- [~] AC-3 byte-identical body → "no install-integrity row" — CheckIntegrity returns zero findings (tests assert this, run twice for determinism) and text output is silent; **but health.json emits a `pass` row** (`internal/cli/check.go` `reportInstallIntegrity`). Verified live on this repo: text silent, `{"name":"install-integrity","status":"pass",...}` in `.hero/cache/health.json`, both runs. Judgment: satisfies the AC's intent (no noise, no false positive — the AC exists to protect false-positive discipline) and matches the house pattern — every check category emits a row via `addRow` (e.g. `orphan-instruction-files` emits `pass`); suppressing it would make health.json unable to distinguish "checked and healthy" from "not checked". Recommend amending AC-3's wording to "no fail/warn row" rather than changing code. Documented in the ledger (openly cites the pass row) though not labeled a deviation in "As delivered".
- [✓] AC-4 exact repair command — `TestCheckIntegrity_DetectsGuttedRegion` asserts the exact string. Live round-trip: ran the printed `hero install project . --target codex` from the advisory → file restored (274 lines), check went silent.
- [✓] AC-5 route via `nativeInstructionFile`, all six targets — `integrity.go` calls `nativeInstructionFile(t)` (no hardcoded filename anywhere in the file); `integrityTargets` table pins claude→CLAUDE.md, other five→AGENTS.md; `f.File` asserted per target across all damage tests.
- [✓] AC-6 not in union → silent even when file exists — `TestCheckIntegrity_SilentOnNeverInstalledTarget` (no-install case; claude-installed-with-user-AGENTS.md case where the file is never even inspected for a region).
- [✓] AC-7 content outside markers never flagged, never modified — `TestCheckIntegrity_IgnoresUserContentOutsideMarkers` (prose above and below, zero findings, byte-identical after). Structural: only `FindManagedRegion(...).Body` is ever read.
- [✓] AC-8 `v=` stamp drift, identical body → silent — `TestCheckIntegrity_IgnoresVersionStampDrift` (v=dev → v=v99.0.0, six targets). Comparison is body-vs-body; markers excluded by construction.
- [✓] AC-9 check writes nothing — mechanism verified: `TestCheckIntegrity_WritesNothing` snapshots every regular file's mtime (UnixNano) + full bytes under the fixture tree, runs CheckIntegrity **on a damaged fixture** (so the finding-producing path executes), and compares count, presence, mtime, and content after. Sound. Corroborated: `integrity.go` contains no write call — grep for `WriteFile|.Write(|Create|OpenFile|Mkdir|Remove|Rename` matches only the "READ-ONLY BY CONSTRUCTION" comment; imports are fmt/os/path/filepath/strings/managed and the only os call is `os.ReadFile`. Live: repeated `hero check` runs on the damaged fixture kept reporting (nothing self-repaired).
- [✓] AC-10 file exists, no region → fail — `TestCheckIntegrity_MissingRegion` (six targets, markers stripped entirely).

## Changes

- [✓] `internal/install/integrity.go` (new, 227 lines) — `CheckIntegrity`, `IntegrityFinding`, `IntegrityKind`, `UnionTargets`, group-by-file + any-match comparison, section-presence via line-anchored H2 match (`bodyHasSectionHeading` correctly rejects prefix-of-longer-heading and prose matches).
- [✓] `internal/cli/check.go` — `reportInstallIntegrity` (Options mirroring install: domain from hero.json defaulted to engineering, domain pack overlaid on core via `hero.OverlayFS(domainFS, hero.CoreFS())`) + one call in `runCheck` after the orphan-instruction-files block; damaged→fail, stale-only→warn, pass row otherwise; degrades to a `warn` row (not a crash) on domain-resolution or check error.
- [✓] `internal/cli/upgrade.go` — `unionTargets` body replaced with a delegate to `install.UnionTargets`; the moved implementation is byte-equivalent logic (same dedupe, same first-seen order).
- [✓] `internal/install/integrity_test.go` (new, 462 lines) — ten test functions: the nine planned plus `TestCheckIntegrity_RenderBodyIsDeterministic` (guards the oracle's core assumption, per the spec's Risks section) and `TestCheckIntegrity_MultiTargetSharedAgentsMdIsSilent`.
- [✓] `internal/install/harness_smoke_test.go` — `TestHarness_InstalledContentSurvivesOrdinaryCommands` now also runs `CheckIntegrity` after each ordinary command and requires zero findings.
- [~] `docs/` — SKIPPED. Verified: `rg -l "satellite-drift|orphan-instruction-files" docs/` exits 1 (no matches); check categories are not enumerated in docs. The skip was pre-authorized by the spec's own Changes item 5 ("skip this item rather than inventing a new doc surface"). Concrete, not soft.

## Open items

- Changes item 5 (docs) — SKIPPED — categories not enumerated in docs; spec pre-authorized the skip — **concrete** (independently verified).
- Copilot fresh-clone inference gap — documented `t.Skipf` in `TestCheckIntegrity_FreshCloneWithoutInstallState` — pre-existing `targetLayouts` probe of legacy `.github/copilot/`, shared with `hero upgrade`'s detection; not introduced by this change — **concrete**; follow-up spec recommended in ledger.

## Audit notes

**Deviation 1 — any-match grouping (lead flagged): verified as necessary, with a demonstrated but inherent false-negative window.**
- The rationale is real, not asserted: `install.Run` auto-syncs sibling targets *after* the primary (`internal/install/install.go:198-201`), so on a codex+opencode project the final AGENTS.md body is codex's rendering when opencode was the primary (sibling codex wrote last) and opencode's rendering when codex was the primary. Confirmed live: installing opencode over a codex install left the codex rendering on disk. Per-target strict equality would therefore fail on one ordering of every healthy multi-target install.
- `TestCheckIntegrity_MultiTargetSharedAgentsMdIsSilent` runs both orderings, which (via the sibling-last mechanics above) lands *both* final bodies — each any-match branch is genuinely exercised, and strict equality would fail at least one ordering. Sufficient as a false-positive guard.
- The false-negative window is live: I installed codex+opencode in a scratch fixture, hand-deleted exactly the Codex-only `### Running Hero Workflows in Codex` subsection from inside the managed region → `hero check` silent (body now equals the opencode rendering). Control: any other in-region mutation → `warn` stale. **However**, that damaged state is byte-identical to a legitimately reachable healthy state (opencode primary with codex sibling ordering, or `--only-target`), so no content oracle can distinguish them — the hole is inherent to last-writer-wins on the shared file, not introduced by the check design. The window is narrow: only damage that exactly reproduces another *installed* target's full rendering passes; gutting, hand-edits, and stale content are all caught. A future fix would require persisting the last writer in install-state.json — follow-up spec material, not a defect here.
- **AC-1's "for each installed target" framing is technically weakened to per-file**, but a per-file finding still names a target whose repair command regenerates the file (and auto-sync refreshes siblings), so the user-facing contract holds.

**Deviation 2 — AC-3 pass row: satisfies intent; amend the AC, not the code.** See AC-3 above.

**Ledger honesty spot-checks:** the ledger openly states the eraser itself was not reintroduced (equivalent on-disk damage written directly) — acceptable equivalence, since the oracle inspects on-disk state and is indifferent to how damage arrived; the validation section's "reintroduce the eraser" steps were about producing that exact damage state, which the fixture reproduces byte-for-byte (the 7-line stub). No performative DONE rows found; every claim I tested reproduced.

**Test evidence (run by auditor, not trusted from ledger):** `go test -race -count=1 ./internal/install/ ./internal/cli/` → ok (2.9s / 55.7s); `go test -count=1 ./internal/managed/` → ok; `go build ./cmd/hero` clean. Live behavior verified on this repo (pass, twice, text-silent + health.json row) and on a scratch fixture (damage → fail with sections + repair cmd; repair → silent; union without install-state.json → still detected; stale → warn).
