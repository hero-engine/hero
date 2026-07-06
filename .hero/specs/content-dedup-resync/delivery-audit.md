# Delivery audit — content-dedup-resync

**Audited:** working tree vs `HEAD` (bc86ad9) — deletions staged, core edits + `content_parity_test.go` unstaged/untracked
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC1 — Zero engineering files shadowing core paths. Re-ran the overlap check myself: `comm -12` over full file listings of `core/` and `domains/engineering/` → 0 rows. Parity test passes for all three domains.
- [✓] AC2 — Engineering installs ship previously-duplicated files from core, six targets. Verified against the on-disk matrix: `engineering-claude/.claude/commands/import.md` and `.claude/skills/spec-format/SKILL.md` (supersede genealogy sentinel), `engineering-copilot/.github/prompts/commands/import.prompt.md` (rename), `engineering-codex/.agents/skills/command-import/SKILL.md` (rename), `engineering-opencode` + `engineering-generic` session-primer with the `hero list --status delivering` fix. **Exception verified as pre-existing, not a regression:** cursor installs zero skills — `internal/install/target_cursor.go:33` routes skills through `installFlat`, and `internal/install/content.go:43` skips directories, so nested `skills/<name>/SKILL.md` never installs; neither file is touched by this change, and the `baseline-cursor` (pre-change binary) install in the matrix also has 0 skill files and the identical 65-file total. Cursor's previously-duplicated commands/agents do install (e.g. `convention-author.md` present under `.cursor/rules/agents/`).
- [✓] AC3 — pm/sales get merged improvements. `pm-generic/.ai/commands/import.md` contains `hero sync import`; `sales-claude/.claude/skills/next-md/SKILL.md` and `next-handoff-emit/SKILL.md` contain the machine-state-only `.local.md` warning.
- [✓] AC4 — Core spec-format carries both sides. `git diff HEAD -- core/skills/spec-format/SKILL.md` removes zero lines (old core 100% retained, +75 insertions); diff vs the deleted eng copy shows exactly one replaced line — the legacy `supersedes` row teaching the deprecated `status: superseded` hand-edit — plus the genealogy rows/section from core and the required `/mock` scoping sentence on the Mockups section.
- [✓] AC5 — Unannotated shadow fails parity test. `content_parity_test.go:63` errors on any shadow without `core_fork:`; branch verified by reading; passing state verified by running `TestDomainPacks_NoUnannotatedCoreShadows` (PASS, all 3 domains). The fixture exercise itself was transient (not reproducible from disk) — accepted on code evidence.
- [✓] AC6 — Annotated byte-identical shadow fails. `content_parity_test.go:66-67` implements the AC as written (byte-equality check). See audit note on branch reachability.
- [✓] AC7 — Full suite + docs check. Auditor re-ran `go test ./...` → exit 0, no failures. **Exception verified as pre-existing:** `hero docs check` reports exactly 2 mismatches, both from `GETTING-STARTED.md` claims counted against root-level `agents/`/`skills/` directories that do not exist at HEAD (absent since the domains refactor — `docs_check.go:48-51` reads `<projectRoot>/agents` etc.); README/GETTING-STARTED are unmodified in this change, so the output is unaffected by the deletions.

## Changes

- [✓] C1 — Merge 14 forked pairs into core. Auditor diffed every new core master against `git show HEAD:` of its deleted engineering copy: **11/11 claimed verbatim ports are byte-identical** (capture, convention, note, decide, handoff, prime, resume, import, session-primer, next-md, next-handoff-emit); **agent-reliability** differs by exactly one sentence — the `engineer.md` cross-ref scoped to "the engineering pack's `engineer.md` … packs without an engineer agent account for the same items explicitly in prose" — no content lost; **spec-format** as per AC4; **context-injection** confirmed core-strictly-ahead (deleted eng copy is a pure subset — core adds only the "Handling superseded specs" section), so no core edit was needed. No engineering content line silently lost anywhere.
- [✓] C2 — pm annotations. `domains/pm/commands/discover.md:2` and `domains/pm/commands/handoff.md:2` carry `core_fork:` with concrete one-line reasons; parity test passes them.
- [✓] C3 — Delete all 34 duplicates. Staged `git rm`: 17 commands + 4 agents + 13 skill SKILL.md files under `domains/engineering/`. Remaining dirs non-empty (31 agents, 13 commands, 39 skills), so embed directives stay valid; build green.
- [✓] C4 — Parity test. `content_parity_test.go` walks `AvailableDomains()` vs `CoreFS()` over agents/commands/skills prefixes; three failure branches present: unannotated shadow (l.63), empty reason (l.65), annotated byte-identical (l.67). Run by auditor: PASS.
- [✓] C5 — Collateral refs. Auditor re-ran the grep for all 34 deleted paths across the repo (excluding `.git`, `.hero/`, `web/docs`): zero references. Additionally checked `web/docs` itself: also zero references — stronger than the ledger's claim.

## Open items

None. Both DONE-with-exception rows (AC2 cursor, AC7 docs check) are genuinely pre-existing conditions with concrete evidence, not soft skips; the cursor gap is filed as a background task per the ledger.

## Audit notes

- **Working tree carries unrelated concurrent work.** `internal/cli/install.go`, `internal/cli/helpers_test.go` (modified) and `internal/cli/install_json_test.go` (untracked) implement an `--json` install mode — a different feature, not named by this spec. Not a defect of this delivery, but the commit must be scoped to: the 34 staged deletions, the 13 `core/` edits, the 2 `domains/pm/` annotations, and `content_parity_test.go`.
- **AC6 branch is nearly unreachable in practice.** `bytes.Equal(domBytes, coreBytes)` compares the domain file *including* its `core_fork:` line, so it only fires when the core file also contains that identical line. A domain copy that differs from core *only* by its annotation line passes as an "intentional fork" — the AC as written (byte-identical) is satisfied, but the "pointless copy" spirit has this small gap. Cosmetic; worth a follow-up tweak (compare with the annotation line stripped), not a blocker.
- **Minor ledger label nit:** the spec Kickoff says "16 core edits"; the actual surface is 13 `core/` files + 2 `domains/pm/` files + 1 new test file. Content matches the merge-direction table exactly; only the count label is off.
- Fixture exercises claimed for AC5/AC6 were transient and cannot be re-verified from disk; accepted on direct code reading plus a live run of the test.
