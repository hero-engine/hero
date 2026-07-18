# Delivery audit — pm-foundation-delivery

**Audited:** current working tree @ `main` (retroactive close, re-verified 2026-07-18)
**Verdict:** SHIP
**Surface:** noteworthy

This is a **retroactive** close of a historical sprint (deliverables shipped
May 2026; status left stale). This audit independently re-verifies the
Completion Ledger's 9 criteria against the current tree — every claim was
checked on disk, not trusted from the ledger. All nine confirmed.

## Acceptance criteria (ledger's 9 sprint criteria)

- [✓] **1. Nine canonical spec types** — `core/spec-types/` holds exactly `initiative, prd, epic, feature, bug, chore, intake, release, sprint` (`ls` → 9 files, all present).
- [✓] **2. Five methodology profiles** — `core/methodologies/` holds `scrum.yaml, kanban.yaml, shape-up.yaml, waterfall.yaml, scrumban.yaml` (5 files present).
- [✓] **3. `internal/vocabulary|methodology|tasks|spectypes` Go pkgs + build** — all four dirs present with Go sources (vocabulary: 5 files; methodology: 5; tasks: 7; spectypes: 6). `go build ./...` → exit 0, clean.
- [✓] **4. schema 1.1 cache export** — `.hero/cache/spec-types.json` present (66 KB); `"schema_version": "1.1"`; `types` collection has 11 entries (9 canonical + engineering `decision`/`convention`), all nine canonical names grep-confirmed present.
- [✓] **5. `hero task` CLI ships** — `./hero task --help` (binary, 57 MB) resolves with `add|list|start|done|history|status` subcommands, exit 0. Help text confirms "additive peer of acceptance criteria," AC untouched.
- [✓] **6. Inline-propose Go side** — `internal/propose/` has 6 Go files (`envelope.go`, `shim.go`, `store.go` + tests); `docs/contracts/inline-propose-v1.md` present (9.4 KB).
- [✓] **7. PM pack aligned to canonical names** — `domains/pm/spec-types/` holds `intake.md` + `prd.md` (+ `README.md`); **no `story.md`, `roadmap-item.md`, or `epic.md`** — collapsed as designed.
- [✓] **8. `domain-plugin-architecture` finishing touches** — sibling spec `.hero/specs/domain-plugin-architecture/spec.md` is `status: completed` (archived).
- [✓] **9. hero-code consumes the three contracts** — Handoff Trail record in spec (line 278): `2026-05-17T21:42:55Z — out → hero-code`, mode advisory, peer_id `ad027c2f...`, result_ref present, reason cites the three stabilized contracts.

**Sanity sweep:** `go test ./...` → **0 FAIL** (86 `ok` packages). Matches ledger claim.

## Changes

- [✓] A1–A3 content authoring — canonical type files + methodology profiles + pm-pack alignment all on disk (criteria 1, 2, 7).
- [✓] B1 `domain-plugin-architecture` cutover — sibling `completed` (criterion 8).
- [✓] B2 `internal/spectypes/` + schema 1.1 export — package (6 files incl. `export.go`, `parity_test.go`) + cache present (criteria 3, 4).
- [✓] B3 `internal/methodology/` + B4 `internal/tasks/` + `hero task` — packages present, tests green, CLI resolves (criteria 3, 5).
- [✓] B5 inline-propose Go + published contract — `internal/propose/` + `docs/contracts/inline-propose-v1.md` (criterion 6).
- [✓] Engineering corpus unchanged — registry ships `internal/spectypes/parity_test.go`; full suite 0 FAIL.

## Open items

None. No PARTIAL / SKIPPED / BLOCKED rows in the ledger; none introduced by this audit.

## Audit notes

- **Nature of the close.** This is a bookkeeping close, not a re-delivery. The end-to-end feature exercise happened in-session across May 2026 (per checklist + handoff record); this audit only re-verifies artifacts still exist and the tree still builds/tests clean. That is exactly what a retroactive close can honestly claim — no live end-to-end re-exercise was performed, and the ledger is candid about that.
- **Frontmatter status is `delivering`, not `planning`.** The spec's `status:` field (line 5) reads `delivering`; the task brief and ledger describe it as stale at `planning`. Immaterial to the evidence, but the orchestrator flipping this to `completed` should note the starting state was `delivering`.
- **AC#7 minor wording.** Ledger says the PM spec-types dir "holds only intake.md + prd.md"; it also contains `README.md`. Harmless — the substantive claim (no `story`/`roadmap-item`/`epic`) is confirmed. README retention is expected per work item A3.
- No scope drift observed in the checked artifacts; all evidence maps to spec-named locations.
