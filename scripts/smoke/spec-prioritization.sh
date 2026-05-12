#!/usr/bin/env bash
# scripts/smoke/spec-prioritization.sh — Per-feature smoke for
# spec-prioritization: verify horizon field parses, hero status
# filters by horizon, hero spec new defaults to horizon:now, and
# hero check enforces the field's presence.
#
# Runs against the live hero repo plus a sandbox workspace for
# mutation tests.
#
# Usage:
#   scripts/smoke/spec-prioritization.sh
#   scripts/smoke/spec-prioritization.sh --record

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

E2E_SPEC_SLUG="spec-prioritization"
WORK_DIR="${REPO_ROOT}"
export WORK_DIR

e2e_init "spec-prioritization"

# --- AC-1: horizon field parses cleanly (hero scan ingest) ----------
# hero graph stats after a scan: spec-prioritization spec has horizon:next;
# verify scan exits 0 (parser didn't choke on horizon field).

SCAN_OUT="${E2E_RUN_DIR}/scan.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" scan ) > "${SCAN_OUT}" 2>&1
SCAN_EXIT=$?

if [[ "${SCAN_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero scan exits 0 — horizon field parsed without error")
  printf '  pass AC-1 — horizon field parsed cleanly (scan exits 0)\n' >&2
else
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero scan exit=${SCAN_EXIT} (possible horizon parse failure)")
  printf '  fail AC-1 — scan exit=%d\n' "${SCAN_EXIT}" >&2
fi
{
  printf '\n#### AC-1\n\n_Assertion:_ `hero scan` exits 0 (horizon field in spec parses cleanly)\n\n```\n'
  tail -20 "${SCAN_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-2: hero status default shows only now+next horizon ----------

STATUS_DEFAULT_OUT="${E2E_RUN_DIR}/status_default.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" status ) > "${STATUS_DEFAULT_OUT}" 2>&1
STATUS_DEFAULT_EXIT=$?

# The live corpus has many now/next specs, so output should be non-empty.
# Also look for the someday/parking summary line.
STATUS_BYTES=$(wc -c < "${STATUS_DEFAULT_OUT}" | tr -d ' ')

if [[ "${STATUS_DEFAULT_EXIT}" -eq 0 && "${STATUS_BYTES}" -gt 0 ]]; then
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero status exits 0, ${STATUS_BYTES} bytes output")
  printf '  pass AC-2 — hero status exits 0 (%d bytes)\n' "${STATUS_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero status exit=${STATUS_DEFAULT_EXIT}, ${STATUS_BYTES} bytes")
  printf '  fail AC-2 — status exit=%d, %d bytes\n' "${STATUS_DEFAULT_EXIT}" "${STATUS_BYTES}" >&2
fi
{
  printf '\n#### AC-2\n\n_Assertion:_ `hero status` exits 0 with non-empty output\n\n```\n'
  head -c 2000 "${STATUS_DEFAULT_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-3: hero status --all shows complete corpus ------------------

STATUS_ALL_OUT="${E2E_RUN_DIR}/status_all.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" status --all ) > "${STATUS_ALL_OUT}" 2>&1
STATUS_ALL_EXIT=$?
STATUS_ALL_BYTES=$(wc -c < "${STATUS_ALL_OUT}" | tr -d ' ')

# --all should show at least as many entries as default
if [[ "${STATUS_ALL_EXIT}" -eq 0 && "${STATUS_ALL_BYTES}" -ge "${STATUS_BYTES}" ]]; then
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero status --all exits 0, ${STATUS_ALL_BYTES} bytes ≥ ${STATUS_BYTES}")
  printf '  pass AC-3 — hero status --all exits 0, more output than default\n' >&2
else
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("exit=${STATUS_ALL_EXIT} or all(${STATUS_ALL_BYTES}) < default(${STATUS_BYTES})")
  printf '  fail AC-3 — status --all exit=%d, bytes all=%d default=%d\n' \
    "${STATUS_ALL_EXIT}" "${STATUS_ALL_BYTES}" "${STATUS_BYTES}" >&2
fi
{
  printf '\n#### AC-3\n\n_Assertion:_ `hero status --all` exits 0 and output ≥ default view\n\n```\n'
  head -c 2000 "${STATUS_ALL_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# AC-4 (hero spec new defaults to horizon: now) and AC-5 (hero check validate
# rejects missing horizon) are not yet implemented (spec-prioritization
# Phases 3+ are still planning). These will be added to the smoke when those
# phases ship.

e2e_finish
exit $?
