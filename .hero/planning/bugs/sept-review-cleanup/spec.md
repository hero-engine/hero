---
title: "September delivery review cleanup — doctor missing paths, Aha gating, managed-block drift, DeepSeek launch guidance"
slug: sept-review-cleanup
type: bug
status: delivering
priority: P2
severity: moderate
root_cause_class: code
domain: engineering
size: small
created: 2026-09-30
tags: [doctor, install, deepseek, aha, tracker, instructions, cleanup]
relations:
  - target: codex-command-workflow-surface-split-brain
    kind: related
  - target: deepseek-harness-install-target
    kind: related
  - target: stale-connect-provider-usage-expectation
    kind: related
delivery_method: manual
---

# September delivery review cleanup

## Goal

Close the defects and rough edges found by the 2026-09-30 review of the
September deliveries (branch `chore/sept-local-deliveries`, commits
`58b352e1`, `88228bd8`, `ad44dc7e`) so every shipped surface reports and
behaves consistently across all eight install targets.

## Kickoff

Five review findings from the September deliveries, all fixed on branch `chore/sept-local-deliveries`:
- doctor flags and lists missing paths for all targets
- Aha is broker-only
- the Mail/Peering paragraphs are in the managed block
- DeepSeek launches with `dsh --profile web --patch`
- the DeepSeek file set is built once per install

**Status:** implementation, full suite, docs build and native DeepSeek checks are green. Closing gates: cold audit, then `hero spec verify sept-review-cleanup`.

**Pick up at:** if verify has not passed, run the audit against `git diff 9ee65dac...HEAD -- ':!.hero'`, then verify.

**Files:** `internal/cli/doctor.go`, `internal/tracker/tracker.go`, `internal/cli/connect.go`, `internal/install/{agents_md,mcp_deepseek,target_deepseek,install}.go`, `domains/engineering/routing.md`

## Problem

The review ran build, vet and the full uncached suite, and all of them pass. Reading the code turned up these gaps:

1. **Doctor hides equal-count missing artifacts for seven of eight targets.**
   `TargetInventory.Incomplete()` counts `Missing` paths, so `doctorVerdict`
   returns `NEEDS REPAIR`. But the table marker and WARNING count use only
   `kindShort` (`doctor.go:221`), and missing paths are printed only for
   DeepSeek (`doctor.go:233`). Reproduced with a Codex inventory at full counts
   (`Agents 3/3`, `Skills 10/10`) and
   `Missing: [.agents/skills/command-decide/SKILL.md]`. The output was
   `NEEDS REPAIR` with no `!` marker, no WARNING, and no path. That is exactly
   the Candy scenario the Codex split-brain fix targeted, and the
   `harness-changes-cover-all-targets` tripwire applies.
2. **Aha! is connectable but not usable as a tracker.** `hero connect aha`
   stores a connection with `CapabilityTracker`, and the broker injects Bearer
   auth. But `tracker.New` has no `aha` case, so every adapter-backed path
   (import, sync, sprint, size mapping) fails with the generic
   `unknown tracker type: "aha"`. No spec covers the Aha work, and
   `hero-pm` / `pm-public-pack` explicitly defer roadmap providers.
3. **Hand-added managed-block text is lost on regeneration.** Commit
   `b04ae23e` added two paragraphs to `CLAUDE.md` (Project Mail as transport vs
   Peering as semantic layer). They are not in the `agents_md.go` template,
   so `hero install` / `upgrade` deletes them, and `AGENTS.md` targets never
   received them. The checked-in `CLAUDE.md` / `AGENTS.md` were also written
   by a pre-DeepSeek binary, so they lack the eight-target roster line.
4. **The DeepSeek "launch" guidance runs a paid headless prompt.**
   `DeepSeekLaunchCommand` (`internal/install/mcp_deepseek.go:26`) and both
   MCP setup docs print
   `dsh --profile headless --patch '<path>' 'Resume this Hero workspace'`.
   That is a one-shot model invocation, not a session launch.
5. **Redundant work.** The DeepSeek file set (rendering every skill plus the
   overlay) is built in `install.Run`'s preflight, again in `runDeepSeek`, and
   the overlay again in `registerMCPDeepSeek`.

## Design

1. **Doctor.** In `buildInventorySection`, mark a row incomplete when
   `inv.Incomplete()` is true, for every target. Render
   `  ! <target> missing: <path>` for every inventory's `Missing`, capped at
   10 per target plus `… and N more`. Keep the DeepSeek activation footnote
   unchanged. Table cells keep the per-kind `!` marker. Add a trailing `!`
   to the target name when counts are full but paths are missing, so the row
   is visibly flagged.
2. **Aha: broker-only.** Keep `hero connect aha`, config validation and
   broker auth. Add an `aha` case to `tracker.New` / `NewWithJiraConfig`
   that returns a typed, actionable error:
   `aha connections support raw tracker requests (hero_tracker_request) only;
   issue import/sync is not implemented for Aha!`. The connect success output
   for `aha` states the same limit. Do not add an adapter. Aha issue sync
   needs its own feature spec.
3. **Managed block.** Add the two paragraphs to the shared managed-block
   renderer in `internal/install/agents_md.go`. The Mail/Peering sentence goes
   after the peering CLI bullets, and the routing note goes before the
   Attention table. Both root files then get the same text. Regenerate the
   repo's `CLAUDE.md` and `AGENTS.md` with the branch binary
   (`go run ./cmd/hero upgrade`) and commit them. `domains/engineering/AGENTS.md`
   parity tests must stay green.
4. **DeepSeek launch.** Make `DeepSeekLaunchCommand` return
   `dsh --profile web --patch '<path>'`, keeping the POSIX quoting. The
   pinned `dsh` rejects a missing `--profile`, and `web` is its shipped
   interactive profile. Update `MCP-SETUP.md` and
   `web/docs/src/configuration/mcp-setup.md` to show the interactive form
   first, and describe headless only as the scripted one-shot path. Extend
   `scripts/deepseek-compatibility.mjs` to compose both the web and headless
   profiles with the generated overlay.
5. **Dedupe.** Compute `deepseekFiles` and prior checksums once. `install.Run`
   runs preflight, then passes the computed set to `runDeepSeek`, for example
   through an unexported field on `Options` or a small struct.
   `registerMCPDeepSeek` reuses the overlay bytes when called from `Run`.
   Standalone MCP registration keeps its current behavior.

## Acceptance Criteria

- **AC-1:** WHEN any installed target (all eight) has a missing Hero-owned artifact while its kind counts are full THE SYSTEM SHALL mark that target's doctor row, include it in the WARNING count, and print each missing path (capped at 10 with a remainder count).
- **AC-2:** WHEN doctor reports `NEEDS REPAIR` or `NEEDS NEWER HERO` THE SYSTEM SHALL also show at least one flagged row in the inventory table, so the verdict and the table never disagree.
- **AC-3:** IF an adapter-backed tracker operation is invoked on an `aha` connection THEN THE SYSTEM SHALL fail with an error naming Aha! and stating that only raw tracker requests are supported, rather than `unknown tracker type`.
- **AC-4:** WHEN `hero connect aha` succeeds THE SYSTEM SHALL state that the connection supports raw tracker requests only; broker requests against Aha SHALL keep working.
- **AC-5:** WHEN `hero install` or `hero upgrade` renders the managed block for any target THE SYSTEM SHALL include the Project Mail vs Peering paragraphs, identically in `CLAUDE.md` and `AGENTS.md`.
- **AC-6:** The repo's checked-in `CLAUDE.md` and `AGENTS.md` SHALL match what the branch binary generates, including the DeepSeek roster line.
- **AC-7:** WHEN a DeepSeek install completes, or doctor shows DeepSeek activation guidance, THE SYSTEM SHALL present an interactive `dsh --profile web --patch '<absolute path>'` launch command, with no prompt argument and not the headless profile. (dsh requires `--profile`. At the pinned harness commit the shipped profiles are acp, web, headless and sdk; web is the interactive one.)
- **AC-8:** WHEN `hero install --target deepseek` runs THE SYSTEM SHALL render the DeepSeek file set once per invocation, and all existing DeepSeek ownership, collision, dry-run and pruning tests SHALL pass unchanged.

## Boundaries

- No Aha issue adapter, field mapping, or sprint support.
- No change to doctor's schema-verdict logic or to the upgrade downgrade guard.
- No changes to DeepSeek file ownership or checksum semantics.
- Out of scope: the `hero-team-server` status change (`planning → delivering`, uncommitted, not yet confirmed with the user) and pruning `/private/tmp` worktrees (local housekeeping).

## Validation

- Doctor: extend `TestBuildDoctorReport` with a table over all eight targets, each at full counts with one `Missing` path. Assert the flagged row, the WARNING and the path. Add one case with more than 10 missing paths to check the cap.
- Aha: unit test that `tracker.New` with `Type: "aha"` returns the typed error. Extend the connect test to assert the broker-only notice. The existing `TestBrokerAhaRequestInjectsCredentialAndReturnsOnlySafeHeaders` must still pass.
- Managed block: the routing-guidance and content-parity tests assert the paragraphs for Claude and one `AGENTS.md` target. `git diff --exit-code CLAUDE.md AGENTS.md` after `go run ./cmd/hero upgrade`.
- DeepSeek: update `DeepSeekLaunchCommand` expectations, including a path with a single quote. Run the docs tests plus strict MkDocs build. Optionally re-run `scripts/deepseek-compatibility.mjs` against the pinned harness.
- Full: `go test ./... -count=1`, `go vet ./...`, `make build`.

## Changes

1. `internal/cli/doctor.go` + `doctor_test.go` — flag rows and list missing paths for all targets (AC-1, AC-2).
2. `internal/tracker/tracker.go` (+ test), `internal/cli/connect.go` (+ test) — Aha broker-only error and notice (AC-3, AC-4).
3. `internal/install/agents_md.go` (+ parity tests), regenerate `CLAUDE.md` / `AGENTS.md` (AC-5, AC-6).
4. `internal/install/mcp_deepseek.go`, doctor footnote, `README.md`, `MCP-SETUP.md`, `web/docs/src/configuration/mcp-setup.md` (AC-7).
5. `internal/install/install.go`, `target_deepseek.go`, `mcp_deepseek.go` — single computation of the DeepSeek file set (AC-8).

## Completion Ledger

Implemented inline with the implementation-principles, go-stack and testing-and-validation skills. Evidence logs are in this spec directory. Base is `e7d31917`.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: every target flags its row and lists missing paths (capped) despite full counts | DONE | `doctor.go` `buildInventorySection` uses `inv.Incomplete()` for all targets, `doctorMissingPathLimit`=10; `TestDoctorNamesMissingArtifactsForEveryTargetDespiteFullCounts` (8 subtests), `TestDoctorCapsMissingArtifactListing`; real CLI repro `doctor-equal-count-repro.log` (codex 86/86, `codex !`, path named) |
| 2 | AC-2: a NEEDS REPAIR / NEEDS NEWER HERO verdict always has a flagged row | DONE | Row flag and WARNING count use the same `Incomplete()` predicate as `doctorVerdict`; the same tests assert WARNING, row marker and verdict together |
| 3 | AC-3: adapter-backed ops on aha fail naming the broker-only limit | DONE | `tracker.go` `ErrAhaAdapterNotImplemented`, `case "aha"` in `New` (and via `NewWithJiraConfig`); `TestNew_AhaIsBrokerOnly` |
| 4 | AC-4: connect aha states the limit; broker keeps working | DONE | `connect.go` prints the notice after the text-mode success line; `TestNonInteractiveConnectAhaStatesBrokerOnlyLimit`; `TestBrokerAhaRequestInjectsCredentialAndReturnsOnlySafeHeaders` still passes |
| 5 | AC-5: Mail vs Peering paragraphs render identically into CLAUDE.md and AGENTS.md | DONE | `agents_md.go` (after the peer bullets) and `domains/engineering/routing.md` (before the Attention table), with concrete `hero_mail_*` tool names so `TestCanonicalRoutingReferencesResolveAgainstRealSurfaces` passes; `TestRoutingGuidanceReachesAllHarnessNativeRoots` asserts both markers across all 8 targets |
| 6 | AC-6: checked-in CLAUDE.md/AGENTS.md match the branch binary | DONE | Regenerated with `./hero upgrade` (v0.34.2-6-ge7d31917); a second upgrade was a no-op and `CLAUDE.md` byte-identical; pack parity `TestEngineeringPackBodyMatchesGoFallback` passes |
| 7 | AC-7: interactive `dsh --profile web --patch '<path>'`, no prompt, not headless | DONE | `mcp_deepseek.go` `DeepSeekLaunchCommand`; exact-string assertion in `deepseek_test.go`; install-output and satellite guidance use the same function; docs updated; native `web` + `headless` composition PASS for engineering/pm/qa (`native-compatibility-*.log`) |
| 8 | AC-8: DeepSeek file set rendered once per install; ownership tests unchanged | DONE | `planDeepSeek` in `target_deepseek.go`, computed once in `install.Run` and passed to `runDeepSeek`; `TestDeepSeekInstallRendersFileSetOnce` (falsified: old code 4 reads vs 2 for one pass); all existing `TestDeepSeek*` pass |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | doctor.go + tests | DONE | See AC-1/2 |
| 2 | tracker.go, connect.go + tests | DONE | See AC-3/4 |
| 3 | agents_md.go + parity tests; regenerate root files | DONE | Also `domains/engineering/routing.md` and the regenerated `domains/engineering/AGENTS.md` pack file |
| 4 | DeepSeek launch command + docs | DONE | `mcp_deepseek.go`, `MCP-SETUP.md`, `web/docs/src/configuration/mcp-setup.md`, `scripts/deepseek-compatibility.mjs` (now composes web + headless); README has no launch command; doctor footnote already says "--patch" generically |
| 5 | Single computation of DeepSeek file set | DONE | `install.go`, `target_deepseek.go`; `registerMCPDeepSeek` keeps its cheap overlay-only preflight for standalone MCP registration |

### Exercise-the-feature check

- [x] Real CLI runs:
  - `hero doctor` on a temp Codex install with `command-decide` deleted, plus a stray `command-stray`: shows `codex !` at 86/86 and names the missing path.
  - `./hero upgrade` regenerates the root files, and a second run is a no-op.
  - `node scripts/deepseek-compatibility.mjs` against pinned harness `477b4f42` passes for engineering, pm and qa: web and headless composition, MCP initialize/list (86 tools), `hero_status`, zero model calls.

### Excellence Bar self-check

- [x] Yes. Each finding has a regression test that fails on the old behavior. The launch-command change was checked against the real harness source, which exposed and corrected two wrong assumptions (a bare `--patch` fails without `--profile`, and `tui` is not a shipped profile).

## Final validation

`go test ./... -count=1` exit 0 (`go-tests.log`); `go vet ./...` clean (`vet.log`); `make build` (`build.log`); docs unit tests (`docs-tests.log`) and `mkdocs build --strict` (`docs-build.log`) pass; `git diff --check` clean.
