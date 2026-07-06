# Delivery audit — hero-content-audit

**Audited:** untracked new files under `.hero/planning/features/hero-content-audit/` at HEAD `bc86ad9` (findings-only delivery; zero-edit AC checked via `git status --porcelain core/ domains/`)
**Verdict:** SHIP
**Surface:** clean

> Re-audit. The first pass returned HOLD on three concerns (one false S1, finding-count drift, coverage-split mismatch). All three were fixed and independently re-verified; details in "Re-audit resolution" below.

## Acceptance criteria

- [✓] AC1 — `inventory.md`, `rubric.md`, `audit-report.md` exist in `.hero/planning/features/hero-content-audit/` (plus `dup-map.txt` and four findings files).
- [✓] AC2 — Coverage is genuinely complete. Verified programmatically, not by trusting the tables: extracted all 227 inventory paths and diffed against the union of the four findings files' coverage/claims tables — **exact match, zero gaps, zero extras** (58 agents + 72 commands + 94 skills + 3 AGENTS.md). Every file has a per-file verdict. The report's Coverage table (14/42/2, 31/39/2, 25/67/2, 0/3, totals 70/151/6) matches the per-file verdict tables exactly (independently recounted).
- [✓] AC3 — Dedup/drift matrix is real and classified. `findings-skills.md §(b)`: all 13 core↔eng skill pairs (8 identical by md5 — md5s independently re-computed and confirmed — 5 classified accidental-drift with commit-level evidence: `92c94aa` fork origin, `d7fd9e9`, `0cfe403`, `4a62fea` one-sided landings). `findings-commands.md §D`: 17 core↔eng command pairs (9 identical, 8 accidental-drift) **plus** cross-domain pairs (core+pm, core+chat) classified intentional specialization. 4 agent pairs in `findings-agents.md`. Nothing left "unclear".
- [✓] AC4 — Staleness citations re-verified cold against the binary and `internal/` source: 11 S1 evidence targets sampled, 10 held exactly; the 1 false finding (executive-report `hero sprint status --week` — the flag exists, `internal/cli/pulse.go:33`) was **withdrawn by the engineer after the first audit pass**, with withdrawal notes in both `findings-skills.md` and `audit-report.md` and the flag moved to the verified-real list. All remaining sampled S1s cite referencing file + concrete evidence, and the citations survive re-verification.
- [✓] AC5 — `audit-report.md` has a ranked Top-S1 table (14 rows post-withdrawal, renumbered cleanly, blast-radius column present) and 10 proposed follow-up specs each with slug, one-line scope, type, and size, plus a sequencing note.
- [✓] AC6 — `git status --porcelain core/ domains/` returns empty (run by this auditor, re-run after the re-audit fixes). Only new untracked files under the spec folder.

## Changes (phases)

- [✓] P1 inventory — 227 rows, stamped `bc86ad9` (= HEAD, dated 2026-07-01), with path/surface/domain/words/frontmatter-keys columns.
- [✓] P2 rubric — 7 dimensions matching the spec's list, empirical p50/p90 word bands per surface, finding format, S1–S3 severity scale.
- [✓] P3 per-surface passes — four findings files, each with coverage table, methodology note, findings, and verified-clean lists. Path-level coverage confirmed (see AC2).
- [✓] P4 synthesis — 5 themes, deduped ranked S1 table, clean-checks section, 10 sized follow-ups. Declared totals (120 raw findings: 31 S1 · 54 S2 · 35 S3) match an independent heading recount across the four findings files exactly (agents 6/12/7, commands 8/14/7 incl. F9 graded S1→S2, skills 10/19/16, routing 7/9/5).

## Open items

None.

## Re-audit resolution (first pass was HOLD)

1. **False S1 (blocker) — fixed.** The executive-report finding claiming `hero sprint status --week` doesn't exist was withdrawn: heading removed from `findings-skills.md` (header now 10/19/16 = 45 with a withdrawal note citing `internal/cli/pulse.go:33`), `hero sprint status --week/--since/--md` added to the verified-real list, the dup-matrix row's "(shared S1 flag bug)" parenthetical removed, and the Top-S1 table row dropped (now 14 rows) with a withdrawal note under the table. Re-verified: no stale reference to the withdrawn claim remains.
2. **Finding-count drift — fixed.** Report now states 120 (31/54/35); auditor's independent recount of `### [S…]` headings across all four files gives exactly 31/54/35. `findings-commands.md` header corrected to 8/14/7 with F9's S1→S2 grading made explicit.
3. **Coverage-split mismatch — fixed.** Report Coverage table now taken from the per-file verdict tables (ground truth); every row and the new totals row (70/151/6, files 227/227) matches the auditor's independent counts, with an explicit note that "flagged" includes dedup-matrix-only flags.

## Audit notes

- **Re-verified S1 evidence that held (10/11 sampled, pre-withdrawal):** `.hero/conventions/` vs `ConventionsDir` = `knowledge/conventions` (`internal/config/config.go:1686`); core `import.md` `--preset/--jql` vs actual `hero import` (knowledge ingest) / `hero sync import` (tracker); `session-primer` `hero status --delivering/--claimed` (real flags: `--all`, `--horizon` only); `hero event` unknown command (real: `hero agent events`); spec-format drift incl. both md5s and the legacy `status: superseded` line at `domains/engineering/skills/spec-format/SKILL.md:294` vs core's `hero supersede` genealogy; context-injection eng copy missing "Handling superseded specs" with git history exactly as cited (eng untouched since `92c94aa`, core got `d7fd9e9`); `hero queue` has no `--owner/--status`; `hero search --themes`, `hero note --type`, `hero new`, `hero read-spec`, `hero forecast`, top-level `hero pulse` all nonexistent as claimed.
- The withdrawn finding is documented transparently inside the deliverable itself (withdrawal notes in `findings-skills.md` header and under the `audit-report.md` Top-S1 table), so downstream consumers of the report see the correction, not a silent edit.
