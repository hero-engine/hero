# Delivery audit — sept-review-cleanup

**Audited:** `git diff e7d31917...167d0268` (commit 167d0268; regenerated `.hero/NEXT.md`/`SNAPSHOT.md` ignored)
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1 every target flags row + lists missing paths (capped 10) at full counts — `internal/cli/doctor.go:224-250` (row flag via `inv.Incomplete()`, `doctorMissingPathLimit`); `TestDoctorNamesMissingArtifactsForEveryTargetDespiteFullCounts` iterates all 8 `inventoryTargetNames` and asserts row `!`, path, WARNING and verdict; `TestDoctorCapsMissingArtifactListing` asserts 10 paths + `… and 3 more`; real CLI output in `doctor-equal-count-repro.log` (`codex !` at 86/86, path named).
- [✓] AC-2 verdict and table never disagree — row flag, WARNING count and `doctorVerdict` (`doctor.go:369-379`) all use `TargetInventory.Incomplete()` (`internal/install/inventory.go:107`); both NEEDS REPAIR and NEEDS NEWER HERO branch off that same count. Tests assert verdict + marker together.
- [✓] AC-3 aha adapter ops fail with a named broker-only error — `internal/tracker/tracker.go` `ErrAhaAdapterNotImplemented`, `case "aha"` in `New`; `TestNew_AhaIsBrokerOnly` covers `New` and `NewWithJiraConfig`, asserts `errors.Is` and absence of "unknown tracker type".
- [✓] AC-4 connect aha states the limit; broker unaffected — `internal/cli/connect.go:409-411`; `TestNonInteractiveConnectAhaStatesBrokerOnlyLimit`. Broker path untouched by the diff (no change under `internal/serve` or broker code); `TestBrokerAhaRequestInjectsCredentialAndReturnsOnlySafeHeaders` passes (`go-tests.log`, and re-run here: `internal/serve` ok). Note: JSON-mode connect output does not carry the notice; AC-4 does not require it.
- [✓] AC-5 Mail vs Peering paragraphs identical in CLAUDE.md and AGENTS.md — `internal/install/agents_md.go:681` and `domains/engineering/routing.md:52`; `TestRoutingGuidanceReachesAllHarnessNativeRoots` asserts both markers across targets. Both paragraphs present at `CLAUDE.md:67,222` and `AGENTS.md:67,222`.
- [✓] AC-6 checked-in root files match the branch binary — auditor rebuilt 167d0268 in a scratch worktree, reproduced the local install set (`.codex/agents`, `.agents`), ran `hero upgrade`: `CLAUDE.md`/`AGENTS.md` byte-identical. DeepSeek roster line present (`CLAUDE.md:79`). (Without the untracked `.codex/agents` the Codex section drops out — expected environment dependence, not a defect of this delivery.)
- [~] AC-7 interactive `dsh --profile web --patch '<abs path>'` on install completion **or doctor activation guidance** — install path delivered: `internal/install/mcp_deepseek.go:27`, used at `target_deepseek.go:192` and `satellite.go:412`; exact-string test in `deepseek_test.go` incl. single-quote path; docs updated; native `web`+`headless` composition PASS (`native-compatibility-*.log`); harness confirms `--profile` is required (`apps/cli/src/args.ts:178`) and `web` is a shipped template (`packages/boot/app-boot/src/profile.ts:183`). **Doctor path not delivered:** `internal/cli/doctor.go:297` still prints `Launch dsh from the intended workspace with --patch /absolute/path/to/.dsh/hero.cordis.patch.yml.` — no `--profile web`, which the pinned `dsh` rejects. No test asserts the doctor wording.
- [✓] AC-8 DeepSeek file set rendered once per install — `planDeepSeek` (`target_deepseek.go:133-150`) called once in `install.Run` (`install.go:159-165`) and passed to `runDeepSeek(opts, plan)` (`install.go:194`); `TestDeepSeekInstallRendersFileSetOnce` counts content reads against a single `deepseekFiles` pass. Existing `TestDeepSeek*` pass unchanged.

## Changes
- [✓] 1. `doctor.go` + `doctor_test.go` — see AC-1/2.
- [✓] 2. `tracker.go` (+test), `connect.go` (+test) — see AC-3/4.
- [✓] 3. `agents_md.go` + parity tests; regenerate `CLAUDE.md`/`AGENTS.md` — also `domains/engineering/routing.md` and `domains/engineering/AGENTS.md`; regeneration reproduced by auditor.
- [~] 4. `mcp_deepseek.go`, **doctor footnote**, `README.md`, `MCP-SETUP.md`, `web/docs/.../mcp-setup.md` — launch function, both setup docs and compatibility script updated; README has no launch command (verified, `README.md:67-68`); doctor footnote not updated (ledger: "already says --patch generically" — but that generic form is the exact bare-`--patch` invocation the spec says fails).
- [✓] 5. single computation of the DeepSeek file set — `install.go`, `target_deepseek.go`. Design item 5 also asked `registerMCPDeepSeek` to reuse overlay bytes from `Run`; it still re-renders the overlay (`mcp_deepseek.go:56-61`). Ledger discloses this; the overlay is a small YAML render and AC-8's test is satisfied, so accepted as a disclosed design deviation.

## Open items (if any)
- None recorded as PARTIAL/SKIPPED/BLOCKED in the ledger. AC-7 and Changes #4 are marked DONE but are partial (see above).

## Audit notes
- **AC-7 downgraded DONE → partial.** `hero doctor`'s DeepSeek activation footnote (`internal/cli/doctor.go:297`) still tells users to launch with a bare `--patch`, which fails with `error: --profile <name> is required` on the pinned harness. Fix: print `install.DeepSeekLaunchCommand(<abs overlay path>)` (or at least `dsh --profile web --patch …`) and assert it in `TestDoctorDeepSeek*`. Small, but it is explicitly in AC-7 and Changes #4.
- Preflight-before-mutation ordering preserved: `planDeepSeek` (render + checksums + `preflightDeepSeek`) runs before `cleanupLegacyCanonicalSymlinks` in `install.Run`. `runDeepSeek` has one caller, always with a non-nil plan.
- Harness-changes-cover-all-targets tripwire: install changes are DeepSeek-only branches; doctor change applies to all 8 targets and is tested for each. Full suite green (`go-tests.log`); auditor re-ran `./internal/cli ./internal/install ./internal/tracker ./internal/serve -count=1`: all ok.
- Nit: `doctorMissingPathLimit` was inserted between `buildInventorySection`'s doc comment and the func (`doctor.go:187-193`), so godoc now attaches that comment to the const.
- Nit: committed `.hero/version.json` records `v0.34.2-6-ge7d31917-dirty`.
