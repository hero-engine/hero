# Delivery audit — mockup-renderer-selection-swiftui-bias

**Audited:** uncommitted working-tree diff (`git diff` + untracked `internal/cli/mock_detect.go`, `internal/cli/mock_detect_test.go`)
**Spec:** `.hero/planning/bugs/mockup-renderer-selection-swiftui-bias/spec.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] **AC1** — Swift project → SwiftUI renderer. `internal/cli/mock_detect.go:143-145` selects `swiftui` when `swiftDetected && toolchainOK`. Verified by `TestMockDetect_PackageSwiftAtRoot` (`mock_detect_test.go:76-102`), `TestMockDetect_XcodeprojAtRoot` (`:104-125`), `TestMockDetect_OnlySwiftFiles_WeakSignal` (`:127-149`).
- [✓] **AC2** — JSON output has all 8 fields. `detectOutput` struct (`mock_detect.go:27-36`) declares `renderer`, `reason`, `signals`, `toolchain_ok`, `toolchain_path`, `config_override`, `explicit_flag`, `conflict`. Single-line shape asserted by `TestMockDetect_JSONIsSingleLine` (`mock_detect_test.go:292-306`).
- [✓] **AC3** — Announce line before generation. `domains/engineering/commands/mock.md:35-43` mandates the one-line `Renderer: {renderer} — reason: {reason} — swiftc: {toolchain_path or "unavailable"}` announce before any generation. Parallel block in `domains/engineering/agents/ui-designer.md:51-55`.
- [✓] **AC4** — `--renderer=html` on Swift → conflict populated. `mock_detect.go:127-129` sets `Conflict` when `ExplicitFlag == "html" && swiftDetected`. Verified by `TestMockDetect_ExplicitHtmlOnSwift_PopulatesConflict` (`mock_detect_test.go:191-210`). Halt instruction lives in `mock.md:23-33` and `ui-designer.md:41-49`.
- [✓] **AC5** — Swift signals + missing `swiftc` → HTML fallback with explicit reason. `mock_detect.go:146-148` returns `html` with reason `"Swift signals detected but swiftc not found — falling back to HTML"`. Verified by `TestMockDetect_SwiftSignalsWithoutToolchain_FallsBack` (`mock_detect_test.go:249-265`).
- [✓] **AC6** — `hero.json` `mockups.renderer` honored as override (but not over explicit flag). Precedence in `mock_detect.go:121-152` is: explicit flag → config override → auto-detect. `TestMockDetect_ConfigOverrideHtmlOnSwiftProject` (`mock_detect_test.go:151-170`) asserts config wins over auto-detect. `TestMockDetect_ConfigOverrideSwiftuiWithoutToolchain_FallsBackToHTML` (`:172-189`) asserts the config+missing-toolchain fallback path. `MockupsConfig` struct round-trips through Load — `config_test.go:921-988` (3 cases: swiftui, html, unset).
- [✓] **AC7** — Selection grounded in CLI, not LLM. The 4-step algorithm has been deleted from both `mock.md` (replaced at lines 6-50 with "Do not derive the renderer yourself. The CLI does it.") and `ui-designer.md` (replaced at lines 31-55 with "You do not pick the renderer."). Diff confirms hard removal, not addition-alongside.
- [✓] **AC8** — `mock.md` and `ui-designer.md` call CLI verbatim with no inline algorithm. Both files now contain only the CLI dispatch + announce + halt-on-conflict instructions. The Swift-signal trigger list is gone from `mock.md`. `ui-designer.md` retains only a single sentence telling agents not to re-derive ("Do not re-derive the choice from Swift signals — the algorithm lives in Go for exactly this reason." at line 33).

## Changes (delivered files in spec)

- [✓] `internal/cli/mock_detect.go` — new file, 326 lines, registers `mockDetectCmd` under `mockCmd` (`init()` at `:64-67`).
- [✓] `internal/cli/mock_detect_test.go` — new file, 337 lines, 13 tests; all pass.
- [✓] `internal/cli/mock.go` — `mockCmd.Long` updated to cross-reference `detect` subcommand (`:24-34`). `go run ./cmd/hero spec mock --help` confirms `detect` appears under Available Commands.
- [✓] `internal/cli/helpers_test.go` — `resetFlags()` resets `mockDetectRenderer` (line 285).
- [✓] `internal/config/config.go` — `Mockups *MockupsConfig` field added to `Config` (`:40`); `MockupsConfig` struct defined at `:1069-1077`.
- [✓] `internal/config/config_test.go` — 3 round-trip tests at `:921-988`; all pass.
- [✓] `domains/engineering/commands/mock.md` — 4-step algorithm removed, replaced with CLI dispatch + announce + halt-on-conflict.
- [✓] `domains/engineering/agents/ui-designer.md` — parallel rewrite of Renderer Selection block.
- [✓] `domains/engineering/skills/swiftui-mockup-renderer/SKILL.md` — "When to use" updated to point at `hero spec mock detect`.
- [✓] `domains/engineering/skills/html-mockup-generation/SKILL.md` — "When to use" added, points at `hero spec mock detect`.

## Open items

None. The engineer's ledger called out two informational items:

- Spec Change #7 (hero.json schema docs) — N/A: no canonical schema doc exists in repo; behavior is documented by `MockupsConfig` struct doc-comment (`config.go:1069-1077`), three round-trip tests, the `mockDetectCmd.Long` help, and the prompt updates. Concrete rationale, not a soft skip.
- Spec wrote `hero mock detect` but actual command path is `hero spec mock detect` (because `mockCmd` is mounted under `specCmd`). Engineer used the correct path consistently in prompts, tests, and help text. Verified — no stale `hero mock detect` strings remain in shipped files.

## Audit notes

- **Carve-outs respected.** The four Follow-ups (iPhone sizing on Mac, iOS-only colors, silent compile fallback, target platform detection) were not touched. The SwiftUI capture pipeline in `ui-designer.md:109-136` is unchanged in this diff.
- **Risks addressed in code.** The monorepo-Swift-detection risk from the spec's Risks section is documented in `mock_detect.go:157-170` with the explicit "next person can revisit if a small apps/ios/ tree under a primarily-Go monorepo causes false positives — see the Risks section of the spec" note. Matches engineer's stated plan.
- **`snapshot.ScanRepo` reused as spec required.** `mock_detect.go:221` calls `snapshot.ScanRepo(root)` to drive the monorepo container walk.
- **C8 cross-reference is bidirectional.** `mockCmd.Long` references `hero spec mock detect --help` (mock.go:29-32). `mockDetectCmd.Long` references `/mock` (mock_detect.go:52-56). Spec C8 also suggested the CLI subcommand "print one example" — the Long block is descriptive rather than including a concrete `hero spec mock detect --renderer=swiftui` invocation example, but the help is informative enough that callers can act on it. Not flagged.
- **Risk #1 (Cobra subcommand collision) mitigated.** Verified by running `go run ./cmd/hero spec mock --help` — `detect` is registered as a subcommand while bare flags (`--list`, `--open`, `--serve`) still dispatch via `runMock`. No swallow-the-arg behavior observed.
- **Test evidence reproduced.** Re-ran `go test ./internal/cli/... ./internal/config/...` cold; both packages pass. The 13 detect tests + 3 config round-trip tests run green individually under `-count=1 -run 'MockDetect|Mockups'`.
- **Out-of-scope churn is hero-managed projection files only.** AGENTS.md (managed block), QUEUE.md, SNAPSHOT.md, NEXT.md, peer-manifest.yaml, version.json all updated. These are agent/tool-projected artifacts that update during normal workflow runs — not engineer-authored scope drift.
- **One new knowledge file** (`.hero/knowledge/decisions/move-llm-judgment-to-cli-when-failure-costs-asymmetric.md`) was added — appropriate for the "capture learnings" rule and consistent with the decision this delivery encodes. Not scope drift.
