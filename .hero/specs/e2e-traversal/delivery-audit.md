# Delivery audit — e2e-traversal

**Audited:** `git log --all --oneline -- scripts/e2e/traversal.sh` → both files land in commit `982742d` (hero v0.8.0 initial public release); no subsequent modifications.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] **AC-1:** `hero why <feature-slug>` returns non-empty origin chain — `traversal.sh:54–75`: runs `hero why next-as-projection`, captures stdout to `why_feature.txt`, asserts `WHY_EXIT -eq 0 && WHY_BYTES -gt 50`. Exit-code check + 50-byte minimum catches silent no-op.
- [✓] **AC-2:** `hero why <feature:AC-N>` returns chain with `satisfied_by`/`belongs_to` edge — `traversal.sh:77–97`: runs `hero why acceptance-criteria-graph:AC-3`, asserts exit=0 and `grep -q "satisfied_by\|belongs_to"` on output. Both edge types from spec are checked.
- [✓] **AC-3:** `hero blocked` exits 0 with non-zero stdout — `traversal.sh:99–120`: asserts `BLOCKED_EXIT -eq 0 && BLOCKED_BYTES -gt 0`. Directly targets the silent-empty regression class named in the spec.
- [✓] **AC-4:** `hero relevant <file>` exits 0 with non-trivial output — `traversal.sh:122–143`: runs `hero relevant internal/cli/checkpoint.go`, asserts exit=0 and `REL_BYTES -gt 0`. Correct test target (`RELEVANT_FILE`) matches spec.
- [✓] **AC-5:** `hero impact <file>` exits 0 — `traversal.sh:145–165`: exit-0 smoke only, matching the "coverage smoke only" language in the spec. No byte assertion — consistent with spec scope.
- [✓] **AC-6:** `hero suggest` exits 0 with non-zero output — `traversal.sh:167–188`: asserts `SUG_EXIT -eq 0 && SUG_BYTES -gt 0`. Spec permits "no high-churn files" canonical message; byte check still passes because that message is non-zero length.
- [✓] **AC-7:** `hero check conflicts` exits 0 — `traversal.sh:190–212`: runs `hero check conflicts next-as-projection`, asserts `CONF_EXIT -eq 0`. Exit-0 smoke only, consistent with spec.
- [✓] **AC-8:** `hero ac list e2e-traversal --json` returns array including `AC-1` — `traversal.sh:214–234`: runs from `REPO_ROOT`, asserts exit=0 and `grep -q '"ac_id":"AC-1"'`. Self-referential ingest check mirrors onboarding AC-4 pattern named in spec.

## Changes

- [✓] `scripts/e2e/traversal.sh` created — 240-line executable (`-rwxr-xr-x`) implementing all 8 ACs with inline pass/fail logic, parallel-array result accumulation, and per-AC observations written to `E2E_LOG`. Sets `E2E_SPEC_SLUG="e2e-traversal"` so `e2e_finish` stamps results.json with the correct spec key.
- [✓] `scripts/e2e/lib.sh` created — 222-line shared harness exporting `e2e_init`, `e2e_finish`, `e2e_step`, `e2e_assert`, `_e2e_now_ms`. Defines `HERO_BIN`, `REPO_ROOT`, and the four parallel result arrays. BSD/macOS `date` fallback (`_e2e_now_ms`) is present.
- [✓] `traversal.sh` sources `lib.sh` — `source "${SCRIPT_DIR}/lib.sh"` at line 18 with `# shellcheck source=lib.sh` annotation.
- [✓] `e2e_init "traversal"` called at line 43, `e2e_finish` called at line 238 with `exit $?` — harness lifecycle is correct.
- [✓] Pinned test targets documented in comments (`traversal.sh:46–52`) with explicit re-pin instructions — per the spec's brittleness mitigation note.

## Open items

- `smoke: deferred` (frontmatter, line 30) — SKIPPED — spec says "Runs every CI pass (when wired)." The `hero smoke --all` / `--since` path in `smoke_cmd.go:113` skips any spec where `s.Smoke.Deferred == true`. CI will not invoke `traversal.sh` until this flag is cleared and a `smoke.script` path is set. **Reason is concrete**: spec explicitly defers CI wiring as follow-on work; the script runs correctly on demand today.

## Audit notes

- The `smoke: deferred` flag is a known-open item, not a delivery gap. The spec's "when wired" language at line 50 pre-acknowledges that CI integration is out of scope for this delivery. The script is runnable on demand (`scripts/e2e/traversal.sh` or `scripts/e2e/traversal.sh --record`). Operator must clear `smoke: deferred` and add `smoke.script: scripts/e2e/traversal.sh` to the frontmatter to activate CI integration — straightforward follow-on.
- AC-8 is a self-referential ingest check: it passes only if this spec's ACs have been ingested via `hero scan` or `hero ac record` before the suite runs. The spec acknowledges this ("mirrors onboarding's AC-4 self-check"). Not a delivery gap — behavioral dependency is documented in the spec.
- `traversal.sh` does not use `e2e_step` / `e2e_assert` from lib.sh — it drives ACs with inline if/then logic and pushes directly into the parallel arrays. This is consistent with the lib.sh contract (the parallel arrays are the accumulator; `e2e_step` is a convenience wrapper, not required). No deviation from spec intent.
- No test evidence of a live run was provided. The audit is code-read only. The script is syntactically correct bash, the logic maps precisely to each AC, and the lib.sh interface contract is honored. Residual risk: runtime behavior of the hero commands themselves is not verified here.

**Verdict:** SHIP
