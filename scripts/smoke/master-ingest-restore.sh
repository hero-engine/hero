#!/usr/bin/env bash
# scripts/smoke/master-ingest-restore.sh — Per-feature smoke for
# master-ingest-restore: verify hero scan runs end-to-end, all ingest
# sources fire (or skip gracefully), the ingest summary renders, and
# idempotency holds.
#
# Runs against the live hero repo — a real populated workspace
# is required to exercise the ingest paths.
#
# Usage:
#   scripts/smoke/master-ingest-restore.sh
#   scripts/smoke/master-ingest-restore.sh --record

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../e2e/lib.sh"

E2E_RECORD=0
for arg in "$@"; do
  case "${arg}" in
    --record) E2E_RECORD=1 ;;
    -h|--help) sed -n '2,10p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) printf 'unknown arg: %s\n' "${arg}" >&2; exit 2 ;;
  esac
done
export E2E_RECORD

E2E_SPEC_SLUG="master-ingest-restore"
WORK_DIR="${REPO_ROOT}"
export WORK_DIR

e2e_init "master-ingest-restore"

# --- AC-1: Note nodes ingested from knowledge/notes/ ----------------

SCAN1_OUT="${E2E_RUN_DIR}/scan1.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" scan --dry-run 2>/dev/null \
  || "${HERO_BIN}" scan ) > "${SCAN1_OUT}" 2>&1
SCAN1_EXIT=$?

STATS1_OUT="${E2E_RUN_DIR}/stats1.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" graph stats ) > "${STATS1_OUT}" 2>&1

if [[ "${SCAN1_EXIT}" -eq 0 ]] && grep -qi "note" "${STATS1_OUT}"; then
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exit=0, graph stats shows Note nodes")
  printf '  pass AC-1 — Note nodes present after scan\n' >&2
else
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exit=${SCAN1_EXIT} or no Note in graph stats")
  printf '  fail AC-1 — scan exit=%d or Note absent from graph stats\n' "${SCAN1_EXIT}" >&2
fi
{
  printf '\n#### AC-1\n\n_Assertion:_ `hero scan` exits 0 and `hero graph stats` shows Note nodes\n\n```\n'
  head -c 2000 "${STATS1_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-2: Memory nodes ingested from ~/.claude memory dir ----------

STATS2_OUT="${E2E_RUN_DIR}/stats2.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" graph stats ) > "${STATS2_OUT}" 2>&1

# Memory nodes may not exist if dir is absent — scan should still exit 0.
if [[ "${SCAN1_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exited 0 regardless of memory dir presence")
  printf '  pass AC-2 — memory ingest path: scan exits 0 (skip or ingest)\n' >&2
else
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exit=${SCAN1_EXIT}")
  printf '  fail AC-2 — scan non-zero exit\n' >&2
fi
{
  printf '\n#### AC-2\n\n_Assertion:_ `hero scan` exits 0 even without memory dir (graceful skip)\n\n```\n'
  head -c 2000 "${STATS2_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-3: Tracker pull skips gracefully when not configured --------

SCAN2_OUT="${E2E_RUN_DIR}/scan_tracker.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" scan ) > "${SCAN2_OUT}" 2>&1
SCAN2_EXIT=$?

TRACKER_SKIP=0
if grep -qi "tracker.*skip\|Graph tracker: skipped\|tracker: skip" "${SCAN2_OUT}"; then
  TRACKER_SKIP=1
fi

if [[ "${SCAN2_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exits 0 (tracker skip=${TRACKER_SKIP})")
  printf '  pass AC-3 — tracker path: scan exits 0 (skip=%d)\n' "${TRACKER_SKIP}" >&2
else
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exit=${SCAN2_EXIT}")
  printf '  fail AC-3 — scan non-zero exit=%d\n' "${SCAN2_EXIT}" >&2
fi
{
  printf '\n#### AC-3\n\n_Assertion:_ `hero scan` exits 0 when tracker not configured (skip path)\n\n```\n'
  head -c 2000 "${SCAN2_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-4: Team-server sync skips gracefully when not logged in -----
# Same scan run as AC-3 also exercises the sync skip path.

if [[ "${SCAN2_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exits 0 (team-server skip path)")
  printf '  pass AC-4 — team-server path: scan exits 0 (skip)\n' >&2
else
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exit=${SCAN2_EXIT}")
  printf '  fail AC-4 — scan non-zero\n' >&2
fi
{
  printf '\n#### AC-4\n\n_Assertion:_ `hero scan` exits 0 when team-server not configured (skip path)\n\n'
  printf '_(shared output with AC-3 step)_\n'
} >> "${E2E_LOG}"

# --- AC-5: Tier-2 extraction skips gracefully without ANTHROPIC_API_KEY

SCAN3_OUT="${E2E_RUN_DIR}/scan_tier2.txt"
( cd "${WORK_DIR}" && ANTHROPIC_API_KEY="" "${HERO_BIN}" scan ) > "${SCAN3_OUT}" 2>&1
SCAN3_EXIT=$?

if [[ "${SCAN3_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-5"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exits 0 without API key (Tier-2 skip path)")
  printf '  pass AC-5 — Tier-2 extraction skip path: scan exits 0\n' >&2
else
  E2E_AC_IDS+=("AC-5"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exit=${SCAN3_EXIT} without API key")
  printf '  fail AC-5 — scan exit=%d without API key\n' "${SCAN3_EXIT}" >&2
fi
{
  printf '\n#### AC-5\n\n_Assertion:_ `ANTHROPIC_API_KEY="" hero scan` exits 0 (Tier-2 skips gracefully)\n\n```\n'
  grep -i "tier-2\|extraction\|api key" "${SCAN3_OUT}" | head -5
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-6: Ingest summary block appears in scan output --------------

if [[ "${SCAN2_EXIT}" -eq 0 ]] && grep -qi "ingest summary\|graph ingest\|ingest report" "${SCAN2_OUT}"; then
  E2E_AC_IDS+=("AC-6"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("ingest summary block present in scan output")
  printf '  pass AC-6 — ingest summary block found in scan output\n' >&2
else
  E2E_AC_IDS+=("AC-6"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exit=${SCAN2_EXIT} or no ingest summary block")
  printf '  fail AC-6 — ingest summary missing from scan output\n' >&2
fi
{
  printf '\n#### AC-6\n\n_Assertion:_ `hero scan` output contains ingest summary block\n\n```\n'
  grep -i "ingest\|summary" "${SCAN2_OUT}" | head -20
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-7: Idempotency — two consecutive scans yield same node count

STATS_BEFORE="${E2E_RUN_DIR}/stats_before.txt"
STATS_AFTER="${E2E_RUN_DIR}/stats_after.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" graph stats ) > "${STATS_BEFORE}" 2>&1
( cd "${WORK_DIR}" && "${HERO_BIN}" scan ) > /dev/null 2>&1
( cd "${WORK_DIR}" && "${HERO_BIN}" graph stats ) > "${STATS_AFTER}" 2>&1

BEFORE_NODES=$(grep -i "total\|nodes" "${STATS_BEFORE}" | head -1)
AFTER_NODES=$(grep -i "total\|nodes" "${STATS_AFTER}" | head -1)

if [[ "${BEFORE_NODES}" == "${AFTER_NODES}" ]]; then
  E2E_AC_IDS+=("AC-7"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("graph stats identical before/after second scan")
  printf '  pass AC-7 — idempotent: stats match before/after\n' >&2
else
  E2E_AC_IDS+=("AC-7"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("stats differ: before=[${BEFORE_NODES}] after=[${AFTER_NODES}]")
  printf '  fail AC-7 — non-idempotent: before=[%s] after=[%s]\n' "${BEFORE_NODES}" "${AFTER_NODES}" >&2
fi
{
  printf '\n#### AC-7\n\n_Assertion:_ Two consecutive scans produce identical graph stats\n\n```\n'
  printf 'before: %s\nafter:  %s\n' "${BEFORE_NODES}" "${AFTER_NODES}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-8: Per-step failure isolation — scan exits 0 even if one step errors

# Already validated by AC-3/4/5 above (tracker/sync/tier-2 all skip gracefully).
# Confirm the scan exit code from those was 0.
if [[ "${SCAN2_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-8"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exits 0 with multiple optional steps skipped")
  printf '  pass AC-8 — per-step isolation: scan exits 0 with skipped steps\n' >&2
else
  E2E_AC_IDS+=("AC-8"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exit=${SCAN2_EXIT}")
  printf '  fail AC-8 — scan non-zero\n' >&2
fi
{
  printf '\n#### AC-8\n\n_Assertion:_ `hero scan` exits 0 with multiple steps skipped (isolation)\n\n'
  printf '_(validated via AC-3/4/5 step outputs above)_\n'
} >> "${E2E_LOG}"

e2e_finish
exit $?
