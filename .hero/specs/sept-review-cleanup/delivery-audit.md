# Delivery audit — sept-review-cleanup

**Audited:** `git diff e7d31917...97baef55` (commits 167d0268 + 97baef55; `.hero` projections ignored)
**Verdict:** SHIP
**Surface:** noteworthy

Round 2. Round 1 (167d0268) was HOLD because the `hero doctor` DeepSeek footnote still showed a bare `--patch` launch. 97baef55 fixes that.

## Acceptance criteria
- [✓] AC-1 every target flags its row and lists missing paths (capped at 10) at full counts. Evidence: `internal/cli/doctor.go` `buildInventorySection` (row flag via `inv.Incomplete()`, `doctorMissingPathLimit`). `TestDoctorNamesMissingArtifactsForEveryTargetDespiteFullCounts` iterates all 8 `inventoryTargetNames` and asserts the row `!`, the path, the WARNING and the verdict. `TestDoctorCapsMissingArtifactListing` asserts 10 paths plus `… and 3 more`. Real CLI output is in `doctor-equal-count-repro.log`.
- [✓] AC-2 verdict and table never disagree. The row flag, the WARNING count and `doctorVerdict` (NEEDS REPAIR and NEEDS NEWER HERO branches) all derive from `TargetInventory.Incomplete()` (`internal/install/inventory.go:107`).
- [✓] AC-3 an aha adapter build fails with an error naming the broker-only limit. Evidence: `internal/tracker/tracker.go` `ErrAhaAdapterNotImplemented` and `case "aha"`. `TestNew_AhaIsBrokerOnly` covers `New` and `NewWithJiraConfig`.
- [✓] AC-4 `hero connect aha` states the limit and the broker is unaffected. Evidence: `internal/cli/connect.go` text-mode notice, `TestNonInteractiveConnectAhaStatesBrokerOnlyLimit`. The diff does not touch broker code, and `TestBrokerAhaRequestInjectsCredentialAndReturnsOnlySafeHeaders` passes.
- [✓] AC-5 the Mail vs Peering paragraphs render into every root. Evidence: `internal/install/agents_md.go` (after the peer bullets) and `domains/engineering/routing.md` (before the Attention table). `TestRoutingGuidanceReachesAllHarnessNativeRoots` asserts both markers. They are present at `CLAUDE.md:67,222` and `AGENTS.md:67,222`.
- [✓] AC-6 the checked-in root files match the branch binary. In round 1 the auditor rebuilt the branch in a scratch worktree with the local install set (`.codex/agents`, `.agents`) and ran `hero upgrade`: `CLAUDE.md` and `AGENTS.md` came out byte-identical. 97baef55 changes only `renderDeepSeekWorkflowSection`, which is emitted only when DeepSeek is installed. This repo has no `.dsh`, so the root files are unaffected. The DeepSeek roster line is present.
- [✓] AC-7 interactive `dsh --profile web --patch '<abs path>'` on install completion and in doctor guidance.
  - `DeepSeekLaunchCommand` (`internal/install/mcp_deepseek.go:27`) is used by the install output (`target_deepseek.go`), the satellite guidance (`satellite.go:412`) and now the doctor footnote (`doctor.go`, with the real workspace path).
  - Tests:
    - `deepseek_test.go` has an exact-string assertion that includes a single-quote path.
    - `TestDoctorDeepSeekNamesMissingArtifactsDespiteFullCounts` asserts `Launch from the intended workspace: dsh --profile web --patch '/repo/.dsh/hero.cordis.patch.yml'`.
    - The generated DeepSeek AGENTS.md section (`agents_md.go`) names `dsh --profile web --patch`, and `deepseek_test.go` asserts it.
  - `git grep` at 97baef55 finds no remaining bare `--patch` launch guidance. The only `--profile headless` mentions are the docs that describe the scripted one-shot path.
  - The pinned harness (477b4f42) requires `--profile` (`apps/cli/src/args.ts:178`) and ships `web` in `PROFILE_TEMPLATES` (`packages/boot/app-boot/src/profile.ts:183`).
  - Native web and headless composition pass (`native-compatibility-{engineering,pm,qa}.log`).
- [✓] AC-8 the DeepSeek file set is rendered once per install. `planDeepSeek` runs once in `install.Run`, before the legacy cleanup, and is passed to `runDeepSeek`. `TestDeepSeekInstallRendersFileSetOnce` covers it, and the existing `TestDeepSeek*` tests pass.

## Changes
- [✓] 1. `doctor.go` and `doctor_test.go`. See AC-1/2. The `doctorMissingPathLimit` doc-comment placement was fixed in 97baef55.
- [✓] 2. `tracker.go`, `connect.go` and their tests. See AC-3/4.
- [✓] 3. `agents_md.go` and parity tests, plus the regenerated root files. This also covers `domains/engineering/routing.md` and `domains/engineering/AGENTS.md`.
- [✓] 4. DeepSeek launch command and docs:
  - `mcp_deepseek.go`
  - the doctor footnote
  - the `agents_md.go` DeepSeek section
  - `MCP-SETUP.md` and `web/docs/src/configuration/mcp-setup.md`
  - `scripts/deepseek-compatibility.mjs`, which now composes both web and headless

  README has no launch command (`README.md:67-68` links to setup).
- [✓] 5. Single computation of the DeepSeek file set (`install.go`, `target_deepseek.go`). One deviation from Design item 5 is disclosed in the ledger: `registerMCPDeepSeek` still re-renders the small overlay rather than reusing the bytes from `Run`, because the same function also handles standalone MCP registration. AC-8 still holds and is tested.

## Open items (if any)
- None. The ledger has no PARTIAL, SKIPPED or BLOCKED rows.

## Audit notes
- Round-1 HOLD resolved. The ledger's AC-7 and Changes 4 rows now name the doctor and AGENTS.md sites and record the round-1 miss.
- Disclosed design deviation (Design item 5, `registerMCPDeepSeek` overlay re-render). It has no user-visible effect and AC-8 is satisfied.
- The preflight-before-mutation ordering is preserved in `install.Run`. `runDeepSeek` has a single caller, which always passes a non-nil plan.
- Harness-changes-cover-all-targets tripwire: install changes are confined to DeepSeek branches, and the doctor change is tested for all 8 targets.
- Tests:
  - The auditor re-ran `go test ./internal/cli ./internal/install ./internal/tracker -count=1` at 97baef55: all ok.
  - `go vet` on those packages is clean.
  - The refreshed `go-tests.log` shows the full suite passing.
  - Round 1 also ran `./internal/serve`, which passed. The coordinator reports an intermittent shutdown timeout at `internal/serve/server_test.go:184` that is unrelated to this diff; the auditor did not reproduce it.
- Nit: the doctor footnote derives the workspace as `filepath.Dir(info.heroDir)`. That is correct while `.hero` sits at the project root (the only layout shipped today). It would need revisiting if configurable workspace location ships.
- Nit: the committed `.hero/version.json` records a `-dirty` version string.
