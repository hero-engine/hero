#!/usr/bin/env bash
# scripts/e2e/traversal.sh — Traversal area smoke.
#
# Spec:    .hero/planning/features/e2e-traversal/spec.md
# Goal:    Prove the v2 traversal verbs (why / blocked / impact /
#          relevant / suggest / check conflicts) all run, return
#          non-trivial output, and the AC graph picks up this spec.
#
# Usage:
#   scripts/e2e/traversal.sh           # run, leave results in tmp/
#   scripts/e2e/traversal.sh --record  # also call `hero ac record`
#   HERO_BIN=/path/to/hero scripts/e2e/traversal.sh

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"

# --- argument parsing -----------------------------------------------

E2E_RECORD=0
for arg in "$@"; do
  case "${arg}" in
    --record) E2E_RECORD=1 ;;
    -h|--help)
      sed -n '2,15p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) printf 'unknown arg: %s\n' "${arg}" >&2; exit 2 ;;
  esac
done
export E2E_RECORD

# --- per-suite setup ------------------------------------------------

E2E_SPEC_SLUG="e2e-traversal"

# Run against the outer hero repo. Traversal needs a populated graph;
# a sandbox would have nothing to traverse.
WORK_DIR="${REPO_ROOT}"

e2e_init "traversal"
export WORK_DIR

# Pinned test targets. Re-pin if these get renamed / removed:
#   - WHY_FEATURE: any feature with a Feature→Initiative parent edge
#   - WHY_AC: any AC with a satisfied_by edge to a real Commit
#   - RELEVANT_FILE: any file with participates_in edges
WHY_FEATURE="next-as-projection"
WHY_AC="acceptance-criteria-graph:AC-3"
RELEVANT_FILE="internal/cli/checkpoint.go"

# --- AC-1: hero why <feature> ---------------------------------------

WHY_OUT="${E2E_RUN_DIR}/why_feature.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" why "${WHY_FEATURE}" ) > "${WHY_OUT}" 2>&1
WHY_EXIT=$?
WHY_BYTES=$(wc -c < "${WHY_OUT}" | tr -d ' ')

if [[ "${WHY_EXIT}" -eq 0 && "${WHY_BYTES}" -gt 50 ]]; then
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero why ${WHY_FEATURE} → ${WHY_BYTES} bytes")
  printf '  pass AC-1 — hero why %s (%d bytes)\n' "${WHY_FEATURE}" "${WHY_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero why exit=${WHY_EXIT}, ${WHY_BYTES} bytes")
  printf '  fail AC-1 — hero why exit=%d, %d bytes\n' "${WHY_EXIT}" "${WHY_BYTES}" >&2
fi
{
  printf '\n#### AC-1\n\n_Assertion:_ `hero why %s` exits 0 + >50 bytes (feature has known origin)\n\n' "${WHY_FEATURE}"
  printf '\`\`\`\n'
  head -c 1500 "${WHY_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-2: hero why <feature:AC-N> ----------------------------------

WHY_AC_OUT="${E2E_RUN_DIR}/why_ac.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" why "${WHY_AC}" ) > "${WHY_AC_OUT}" 2>&1
WHY_AC_EXIT=$?

if [[ "${WHY_AC_EXIT}" -eq 0 ]] && grep -q "satisfied_by\|belongs_to" "${WHY_AC_OUT}"; then
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero why ${WHY_AC} returned origin chain")
  printf '  pass AC-2 — hero why %s\n' "${WHY_AC}" >&2
else
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero why ${WHY_AC} exit=${WHY_AC_EXIT}, no edge type in output")
  printf '  fail AC-2 — hero why %s exit=%d\n' "${WHY_AC}" "${WHY_AC_EXIT}" >&2
fi
{
  printf '\n#### AC-2\n\n_Assertion:_ `hero why %s` returns chain with satisfied_by/belongs_to edge\n\n' "${WHY_AC}"
  printf '\`\`\`\n'
  head -c 1500 "${WHY_AC_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-3: hero blocked ---------------------------------------------

BLOCKED_OUT="${E2E_RUN_DIR}/blocked.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" blocked ) > "${BLOCKED_OUT}" 2>&1
BLOCKED_EXIT=$?
BLOCKED_BYTES=$(wc -c < "${BLOCKED_OUT}" | tr -d ' ')

if [[ "${BLOCKED_EXIT}" -eq 0 && "${BLOCKED_BYTES}" -gt 0 ]]; then
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero blocked → ${BLOCKED_BYTES} bytes")
  printf '  pass AC-3 — hero blocked (%d bytes)\n' "${BLOCKED_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero blocked exit=${BLOCKED_EXIT}, ${BLOCKED_BYTES} bytes")
  printf '  fail AC-3 — hero blocked exit=%d, %d bytes\n' "${BLOCKED_EXIT}" "${BLOCKED_BYTES}" >&2
fi
{
  printf '\n#### AC-3\n\n_Assertion:_ `hero blocked` exits 0 + non-zero stdout (catches silent no-op)\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${BLOCKED_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-4: hero relevant <file> -------------------------------------

REL_OUT="${E2E_RUN_DIR}/relevant.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" relevant "${RELEVANT_FILE}" ) > "${REL_OUT}" 2>&1
REL_EXIT=$?
REL_BYTES=$(wc -c < "${REL_OUT}" | tr -d ' ')

if [[ "${REL_EXIT}" -eq 0 && "${REL_BYTES}" -gt 0 ]]; then
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero relevant ${RELEVANT_FILE} → ${REL_BYTES} bytes")
  printf '  pass AC-4 — hero relevant %s (%d bytes)\n' "${RELEVANT_FILE}" "${REL_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero relevant exit=${REL_EXIT}, ${REL_BYTES} bytes")
  printf '  fail AC-4 — hero relevant exit=%d, %d bytes\n' "${REL_EXIT}" "${REL_BYTES}" >&2
fi
{
  printf '\n#### AC-4\n\n_Assertion:_ `hero relevant %s` exits 0 + non-zero stdout\n\n' "${RELEVANT_FILE}"
  printf '\`\`\`\n'
  head -c 1500 "${REL_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-5: hero impact <file> ---------------------------------------

IMP_OUT="${E2E_RUN_DIR}/impact.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" impact "${RELEVANT_FILE}" ) > "${IMP_OUT}" 2>&1
IMP_EXIT=$?

if [[ "${IMP_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-5"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero impact ${RELEVANT_FILE} exit=0")
  printf '  pass AC-5 — hero impact exit=0\n' >&2
else
  E2E_AC_IDS+=("AC-5"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero impact exit=${IMP_EXIT}")
  printf '  fail AC-5 — hero impact exit=%d\n' "${IMP_EXIT}" >&2
fi
{
  printf '\n#### AC-5\n\n_Assertion:_ `hero impact %s` exits 0 (smoke only)\n\n' "${RELEVANT_FILE}"
  printf '\`\`\`\n'
  head -c 1500 "${IMP_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-6: hero suggest ---------------------------------------------

SUG_OUT="${E2E_RUN_DIR}/suggest.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" suggest ) > "${SUG_OUT}" 2>&1
SUG_EXIT=$?
SUG_BYTES=$(wc -c < "${SUG_OUT}" | tr -d ' ')

if [[ "${SUG_EXIT}" -eq 0 && "${SUG_BYTES}" -gt 0 ]]; then
  E2E_AC_IDS+=("AC-6"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero suggest → ${SUG_BYTES} bytes")
  printf '  pass AC-6 — hero suggest (%d bytes)\n' "${SUG_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-6"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero suggest exit=${SUG_EXIT}, ${SUG_BYTES} bytes")
  printf '  fail AC-6 — hero suggest exit=%d, %d bytes\n' "${SUG_EXIT}" "${SUG_BYTES}" >&2
fi
{
  printf '\n#### AC-6\n\n_Assertion:_ `hero suggest` exits 0 + non-zero stdout\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${SUG_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-7: hero check conflicts -------------------------------------

CONF_OUT="${E2E_RUN_DIR}/conflicts.txt"
# `hero check conflicts <spec-slug>` requires a slug — pass a known
# in-flight one so the verb is exercised even when no conflicts exist.
( cd "${WORK_DIR}" && "${HERO_BIN}" check conflicts "${WHY_FEATURE}" ) > "${CONF_OUT}" 2>&1
CONF_EXIT=$?

if [[ "${CONF_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-7"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero check conflicts exit=0")
  printf '  pass AC-7 — hero check conflicts exit=0\n' >&2
else
  E2E_AC_IDS+=("AC-7"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero check conflicts exit=${CONF_EXIT}")
  printf '  fail AC-7 — hero check conflicts exit=%d\n' "${CONF_EXIT}" >&2
fi
{
  printf '\n#### AC-7\n\n_Assertion:_ `hero check conflicts` exits 0 (smoke)\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${CONF_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-8: AC graph reflects this spec's ingest ---------------------

AC_OUT="${E2E_RUN_DIR}/ac_list.txt"
( cd "${REPO_ROOT}" && "${HERO_BIN}" ac list e2e-traversal --json ) > "${AC_OUT}" 2>&1
AC_EXIT=$?

if [[ "${AC_EXIT}" -eq 0 ]] && grep -q '"ac_id":"AC-1"' "${AC_OUT}"; then
  E2E_AC_IDS+=("AC-8"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac list found AC-1 for e2e-traversal")
  printf '  pass AC-8 — ac list found AC-1\n' >&2
else
  E2E_AC_IDS+=("AC-8"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac list exit=${AC_EXIT} or AC-1 not found")
  printf '  fail AC-8 — ac list exit=%d or AC-1 absent\n' "${AC_EXIT}" >&2
fi
{
  printf '\n#### AC-8\n\n_Assertion:_ `hero ac list e2e-traversal --json` returns AC-1\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${AC_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- finalize -------------------------------------------------------

e2e_finish
exit $?
