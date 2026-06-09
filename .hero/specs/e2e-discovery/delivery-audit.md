# Delivery audit — e2e-discovery

**Audited:** commit `982742d` (hero v0.8.0 — initial public release)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC-1: `hero search <term>` exits 0 and emits >100 bytes for known-populated term — `scripts/e2e/discovery.sh:54–73`. Runs `hero search graph`, captures stdout, asserts `SEARCH_EXIT -eq 0 && SEARCH_BYTES -gt 100`. Verified: 3327 bytes on first run.
- [~] AC-2: `hero ask <question>` exits 0 and emits >100 bytes — `scripts/e2e/discovery.sh:77–96`. Check correctly implemented: runs `hero ask "what is hero"`, asserts `ASK_EXIT -eq 0 && ASK_BYTES -gt 100`. **Known failing at runtime** due to unified retrieval layer not setting `Path` on graph-node results (see `internal/cli/ask.go:99-103`). Spec explicitly marks this AC "❌ failing" with root cause documented; tracked separately. The harness correctly surfaces this regression — the AC is not a soft skip.
- [✓] AC-3: `hero recap --since 7d` exits 0 with non-zero stdout — `scripts/e2e/discovery.sh:100–119`. Asserts exit=0 and `RECAP_BYTES -gt 0`. Verified: 26108 bytes.
- [✓] AC-4: `hero next` exits 0 with non-zero stdout — `scripts/e2e/discovery.sh:122–142`. Asserts exit=0 and `NEXT_BYTES -gt 0`. Verified: 4008 bytes.
- [✓] AC-5: `hero resume --budget 500` exits 0 with non-zero stdout — `scripts/e2e/discovery.sh:145–165`. Asserts exit=0 and `RESUME_BYTES -gt 0`. Verified: 3504 bytes.
- [✓] AC-6: `hero ac list e2e-discovery --json` returns array including AC-1 — `scripts/e2e/discovery.sh:168–187`. Runs against `REPO_ROOT`, greps for `"ac_id":"AC-1"`. Verified: 5/6 ACs flipped to passing on first `--record` run.

## Changes

- [✓] `scripts/e2e/discovery.sh` created — new file, mode 100755, added in commit `982742d`
- [✓] `scripts/e2e/lib.sh` sourced — exists and provides `e2e_init`, `e2e_finish`, shared scaffolding; also added in commit `982742d`

## Open items

- AC-2 `hero ask` fails at runtime — KNOWN — root cause: `internal/cli/ask.go:99-103` passage extraction requires `r.Path`, which unified retrieval (Phase B) does not set on graph-node results. Spec explicitly marks this "❌ failing" and states the AC is intentionally red until the underlying bug is fixed. Reason is concrete — names the file, lines, and mechanism. This spec's delivery is the harness, not the fix.

## Audit notes

- AC-2 is marked `~` (partial/known-failing), not `✗`. The assertion code is correctly written and executes correctly — the failure is an expected runtime signal from a real upstream bug, not a missing or dishonest claim. The spec documentation is clear and honest: the AC is intentionally red, the root cause is named, and the separate tracking is confirmed.
- discovery.sh does **not** use `e2e_step` / `e2e_assert` helpers from lib.sh — it uses the parallel-arrays accumulator pattern directly (capturing output files, computing byte counts inline). This is a deliberate pattern difference from traversal/validation suites (which needed exit-code-only checks); discovery needed byte-count assertions that `e2e_step` does not provide. No gap — the `e2e_init` and `e2e_finish` contract is fully honored.
- Script is executable (`-rwxr-xr-x`), sources lib.sh via `${SCRIPT_DIR}/lib.sh`, calls `e2e_init "discovery"` at line 43 and `e2e_finish` at line 191. All four structural requirements verified.
- Byte-count thresholds: AC-1 uses `>100` (high-recall term guard); AC-3/4/5 use `>0` (non-empty guard). AC-1's stricter threshold is intentional and documented in spec.

**Verdict:** SHIP
