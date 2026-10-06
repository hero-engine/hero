# Delivery audit — hero-work-tool

**Audited:** `git diff 3f75cfd6...df250e73`, specifically commit df250e73. That covers `toolWork`, `workRevision`, `WatchGlobs`, `HeroWork`/`PolishItem`/`SuggestedItem` in `internal/serve/mcp_tools_read_contract.go`, the registration, and the tests.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: the HeroWork shape.
  - `TestToolWorkShape` checks the header, `watch_globs`, the lanes, the next command, that knowledge specs are excluded, and `"polish":[]` / `"suggested":[]`.
  - A real call returned `schema_version` 1, 513 items, and empty `polish`/`suggested`.
  - Filling `polish` and `suggested` is out of scope here, as the Kickoff states. It belongs to the sibling `polish-and-suggested`.
- [✓] AC-2: the revision is stable when content is unchanged and moves when it changes.
  - `TestToolWorkRevision` shows the same revision an hour later and a different one after a spec edit.
  - `workRevision` hashes the schema version, `hero_version`, each `slug=item revision` in path order, and polish/suggested. It does not hash `generated_at`.
  - Ten real consecutive calls all returned `03f29485ba755708`.
- [✓] AC-3: `recent_days` validation. `TestToolWorkRecentDays` rejects 0, -1, 1.5 and "7", and accepts 30.
- [✓] AC-4: read-only, with no writes under the watch globs.
  - `readOnlyHint: true` is advertised in `tools/list`.
  - **Auditor check:** a snapshot of everything under `WatchGlobs` was identical before and after 10 real calls. Only `.hero/index.db` was written, by the stale-index refresh. That is a runtime file outside the globs.
  - `TestToolWorkWritesNothing` covers only the four directories, not `NEXT.md` or `hero.json`. The auditor's real-run check covers all six globs.
- [✓] AC-5: latency on the real repo, with a stable revision.
  - The ledger's evidence was indirect: it cited workmodel probe timings and an untimed `real-exercise.log`.
  - **Auditor measurement:** end-to-end `tools/call hero_work` through a real `hero mcp` stdio session, with a clean df250e73 binary on a copy of this repo's `.hero`. Times were 0.40 s for the first call (index refresh), then 0.24–0.28 s across 9 more. That is well under the 2 s p95 budget.

## Changes
- [✓] `internal/serve/mcp_tools_read_contract.go`: `HeroWork`, `PolishItem`, `SuggestedItem`, `WatchGlobs` (all six contract globs, no runtime files), `toolWork`, `workRevision`.
- [✓] Registration: `hero_work` is in `mcp_dispatch.go` (`safetyRead`) and `mcp_tools_def.go`, and the tool count is 68 in `mcp_test.go`.
- [✓] Tests: the 4 `TestToolWork*` tests.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- **Inherited:** items carry work-item-model's fail-open sign-off defect. For example, `token-efficiency-pass` is reported verify `passed` with `next: null`, although Gate 1 would reject its sign-offs (see the work-item-model HOLD).
- **Contract detail, harmless:** `read-contract-v1` says the global revision covers "sorted item revisions". The implementation hashes `slug=revision` in path order, which is deterministic and equivalent in effect.
- **Evidence gap, now closed:** `tests.log` and `real-exercise.log` carry no timings. The latency AC now rests on the auditor's measurement above.
- **Re-run (auditor):** at a clean df250e73 worktree, the workmodel, serve, nextdoc, install and cli tests pass, and `go vet` is clean.
