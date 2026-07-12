# Delivery audit — init-gitignore-missing-machine-local-artifacts

**Audited:** `git diff HEAD` (working tree) + `git diff --cached HEAD` (index)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria (from Suggested Fix Approach + CRITICAL guardrail)

- [✓] Add `.hero/cache/` to `managedGitignoreEntries` with matching section comment — `internal/cli/init.go:455-457` (`# Regenerable cross-language cache (rewritten every hero invocation)`)
- [✓] Add `.hero/sessions/` to `managedGitignoreEntries` with matching section comment — `internal/cli/init.go:458-460` (`# Per-session ref store (ephemeral, session-scoped)`)
- [✓] Add `.hero/install-state.json` to `managedGitignoreEntries` with matching section comment — `internal/cli/init.go:461-463` (`# Per-machine install state (host capabilities, informational)`)
- [✓] Comments/blank-line grouping match existing style — new groups mirror the `# Per-machine satellite manifest` block form exactly (blank line + comment + entry)
- [✓] `.hero/events.log` NOT added to the ignore list — absent from the slice; still git-tracked (`git ls-files .hero/events.log` returns it)
- [✓] Regression fence prevents future `events.log` addition — `init_gitignore_test.go:41-46` asserts the generated block does not contain `events.log`
- [✓] Untrack the 3 leaked `refs.db` files (staged deletion-from-index, working files retained) — `git ls-files .hero/sessions/` is empty; all 3 staged as `Bin N -> 0 bytes`; working-tree files present on disk (49152 / 69632 / 32768 bytes)

## Changes

- [✓] `internal/cli/init.go` — 9 lines added to `managedGitignoreEntries`; no control-flow touched, data-only
- [✓] `internal/cli/init_gitignore_test.go` — 3 entries added to `CreatesWhenMissing` expected list; events.log guardrail fence added; rollout assertions added to `RefreshesUpdatedEntries`
- [✓] Index: 3 `refs.db` files staged for deletion-from-index only

## Test Plan coverage

- [✓] Step 2 (required): 3 new entries added to `CreatesWhenMissing` expected slice — `init_gitignore_test.go:35-37`
- [✓] Step 2 optional (defensive): `RefreshesUpdatedEntries` now asserts all 3 machine-local entries appear after refresh — `init_gitignore_test.go:127-135` — encodes the zero-migration rollout claim
- [✓] Step 3 optional (guardrail): events.log-absent assertion — `init_gitignore_test.go:41-46`
- [✓] `go test ./internal/cli/ -run TestEnsureManagedGitignoreBlock -v` → 4/4 PASS
- [✓] `go build ./...` → clean (exit 0)

## Open items

None.

## Audit notes

- Diff is tightly scoped to the two files named in the spec. No scope creep in source.
- The engineer implemented both *optional* Test Plan items (#2 rollout assertions, #3 events.log fence), not just the minimum required extension. Above the floor, no gold-plating — both directly encode spec constraints.
- `.hero/NEXT.md`, `.hero/SNAPSHOT.md`, `.hero/next/chet-bellows.md` changes are Hero's projected handoff files, out of scope for this fix — not delivery defects.
- Every ledger row maps to concrete on-disk evidence. No performative DONE marks found.
