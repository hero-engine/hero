# Delivery audit — codex-agents-wholesale-wipe-destroys-user-files

**Audited:** `git diff HEAD -- internal/install/` (working tree; `cleanup_test.go` intent-to-add)
**Verdict:** SHIP
**Surface:** noteworthy (1 documented residual)

Cold audit of the option-(c) fix that stops `hero install`/`upgrade` from wholesale-wiping
`.codex/agents/` (a live Codex loader dir) and destroying user-authored `.toml` agents. Every
claim below was verified against source on disk, the test suite (`-race`), and an end-to-end
repro with a freshly built binary — not trusted from the ledger.

## Acceptance criteria

- [✓] **AC-1 — user `.toml` agent survives byte-identical (load-bearing).** `removeLegacyDirMatching`
  (cleanup.go:64-67) skips entries where `shouldRemove(name)==false`; codex passes the `.md`-only
  predicate (target_codex.go:67). Empirically confirmed: planted `my-custom.toml` survived a re-install
  byte-identical (sha `db2521eb…` before and after). Tests: `TestRunCodex_PreservesUserAuthoredTomlAgent`,
  `TestPruneStaleFiles_NeverRemovesUserFile[codex]` (plants a `.toml`, asserts byte-equal survival).
- [✓] **AC-2 — legacy `.md` dead-bytes removed.** `isLegacyCodexAgentMarkdown = HasSuffix(name, ".md")`
  (target_codex.go:138-140). Repro: install printed `cleanup .codex/agents/engineer.md (removed dead
  bytes)` and the file was gone. Tests: `TestHarness_SmokeCodex`, `TestRunCodexProject`,
  `TestRemoveLegacyDirMatching_MdScopedPreservesOthers`.
- [✓] **AC-3 — dropped Hero `.toml` pruned via `pruneStaleFiles`, not the wipe.**
  `TestPruneStaleFiles_RemovesDroppedAgent[codex]` now runs with `pruneActor:true` and asserts the
  prune-only stderr string `removed — dropped from product` fires for codex plus disk+manifest removal —
  a non-vacuous assertion that the manifest prune (not the cleanup) is the remover.
- [✓] **AC-4 — current canonical agents still render to `.toml`.** Repro: `engineer.toml`/`reviewer.toml`
  present after re-install. Tests: smoke + survival test assert `mustBeRegularFile`.
- [✓] **AC-5 — `.codex/commands` stays wholesale.** target_codex.go:72 unchanged (`removeLegacyDir` =
  nil predicate). `TestRunCodex_CommandsDirWholesaleRemoved` plants a `.toml` there and asserts the whole
  dir is removed (proving it is NOT `.md`-scoped).
- [✓] **AC-6 — copilot + no live-loader `os.RemoveAll`.** Call-site grep confirms only 4 sites:
  `.codex/agents` (predicate, the fix), `.codex/commands`, copilot `.github/copilot/*` (target_copilot.go:62),
  and `.hero/` mirror (cleanup.go:202) — the last three all nil = byte-identical wholesale.
  `TestRemoveLegacyDirMatching_NilPredicateWholesale` proves the nil path removes files of every extension
  plus subdirs plus the dir itself.
- [✓] **AC-7 — fresh install (dir absent) no-ops.** `os.Lstat` → `os.IsNotExist` returns nil
  (cleanup.go:52-54). `TestRemoveLegacyDirMatching_MissingDirNoOp`.
- [✓] **AC-8 — scoped cleanup leaves non-`.md` entries → dir not rmdir'd.** The empty-dir removal
  (cleanup.go:73-87) re-reads the dir *after* the scoped pass and only `os.Remove`s when `len(leftover)==0`.
  Repro: `.codex/agents` survived with `my-custom.toml` inside. Tests: survival test asserts
  `mustBeDirectory`; `TestRemoveLegacyDirMatching_MdScopedPreservesOthers`.

## Changes

- [✓] **cleanup.go** — `removeLegacyDir` is now a nil-predicate wrapper over new
  `removeLegacyDirMatching`; per-entry removal gated on the predicate; walk/Lstat/ReadDir/symlink/empty-dir
  logic byte-identical for the nil path. Confirmed in diff (cleanup.go:36-72).
- [✓] **target_codex.go** — `.codex/agents` call switched to `removeLegacyDirMatching(..., isLegacyCodexAgentMarkdown)`;
  `strings` import + predicate func added; `.codex/commands` untouched; header comment corrected to state `.md`-scoped.
- [✓] **prune_test.go** — codex `pruneActor` flipped false→true; added per-target `userFile` field (codex = `.toml`,
  others `.md`); stale carve-out comments rewritten. Both codex subtests exercise the manifest prune non-vacuously.
- [✓] **cleanup_test.go** — survival test + commands-wholesale test + 5 `removeLegacyDirMatching` unit tests
  (`.md`-scoped, empty-dir rmdir, nil wholesale, missing-dir no-op). All pass under `-race`.

## Open items

- **Documented residual (acceptable, not a blocker):** a user file that is itself a `.md` inside
  `.codex/agents` (e.g. a stray `README.md` or `notes.md`) is still removed — it is indistinguishable by
  name from the pre-`.toml` dead-bytes and is inert as an agent (Codex ignores `.md` there). Documented in
  both the code (target_codex.go:130-137) and the spec's Residual section, with the stricter stem-match
  variant explicitly considered and rejected as over-engineering. The load-bearing data-loss vector
  (live `.toml` agents, subdirs, any non-`.md` file) is fully protected. No realistic Codex agent workflow
  puts a live `.md` in this dir; the only collateral is a hand-parked note/README, which is a narrow and
  clearly-disclosed edge.

## Audit notes

- **Build:** `go build ./...` clean.
- **Tests:** `go test -race -count=1 ./internal/install/` green (3.7s). All 8 named codex/prune/cleanup tests
  pass; both prune tests exercise codex subtests with real disk+manifest+stderr assertions (verified non-vacuous
  by reading bodies at prune_test.go:380-459).
- **Empirical proof:** built binary → `hero init` + `install --target codex`, planted user `my-custom.toml`
  (sha `db2521eb…`) + legacy `engineer.md`, re-ran `install --force`. Output was scoped
  (`cleanup .codex/agents/engineer.md (removed dead bytes)` — no dir wipe); user `.toml` survived byte-identical,
  `engineer.md` removed, `engineer.toml` present (sha `9598b073…`), dir intact.
- **Scope:** diff is confined to the four spec-named files under `internal/install/`. No drift.
- opsrunner subprocess-timeout flake (noted elsewhere) is out of scope; the primary package is green.
