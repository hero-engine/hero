#!/usr/bin/env bash
# scripts/e2e/discovery.sh — Discovery area smoke.
#
# Spec:    .hero/planning/features/e2e-discovery/spec.md
# Goal:    Prove the five discovery verbs (search / ask / recap /
#          next / resume) all run, return non-trivial output, and
#          the AC graph picks up this spec.
#
# Usage:
#   scripts/e2e/discovery.sh           # run, leave results in tmp/
#   scripts/e2e/discovery.sh --record  # also call `hero ac record`
#   HERO_BIN=/path/to/hero scripts/e2e/discovery.sh

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

E2E_SPEC_SLUG="e2e-discovery"

# Run against the outer hero repo. Discovery needs a populated graph;
# a sandbox would have nothing to find.
WORK_DIR="${REPO_ROOT}"

e2e_init "discovery"
export WORK_DIR

# Pinned test inputs. Re-pin if these ever return 0 bytes:
#   - SEARCH_TERM: any high-recall term in the graph
#   - ASK_QUESTION: a question the knowledge base should answer
SEARCH_TERM="graph"
ASK_QUESTION="what is hero"

# --- AC-1: hero search <term> ---------------------------------------

SEARCH_OUT="${E2E_RUN_DIR}/search.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" search "${SEARCH_TERM}" ) > "${SEARCH_OUT}" 2>&1
SEARCH_EXIT=$?
SEARCH_BYTES=$(wc -c < "${SEARCH_OUT}" | tr -d ' ')

if [[ "${SEARCH_EXIT}" -eq 0 && "${SEARCH_BYTES}" -gt 100 ]]; then
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero search ${SEARCH_TERM} → ${SEARCH_BYTES} bytes")
  printf '  pass AC-1 — hero search %s (%d bytes)\n' "${SEARCH_TERM}" "${SEARCH_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero search exit=${SEARCH_EXIT}, ${SEARCH_BYTES} bytes")
  printf '  fail AC-1 — hero search exit=%d, %d bytes\n' "${SEARCH_EXIT}" "${SEARCH_BYTES}" >&2
fi
{
  printf '\n#### AC-1\n\n_Assertion:_ `hero search %s` exits 0 + >100 bytes\n\n' "${SEARCH_TERM}"
  printf '\`\`\`\n'
  head -c 1500 "${SEARCH_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-2: hero ask <question> --------------------------------------

ASK_OUT="${E2E_RUN_DIR}/ask.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" ask "${ASK_QUESTION}" ) > "${ASK_OUT}" 2>&1
ASK_EXIT=$?
ASK_BYTES=$(wc -c < "${ASK_OUT}" | tr -d ' ')

if [[ "${ASK_EXIT}" -eq 0 && "${ASK_BYTES}" -gt 100 ]]; then
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ask → ${ASK_BYTES} bytes")
  printf '  pass AC-2 — hero ask (%d bytes)\n' "${ASK_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ask exit=${ASK_EXIT}, ${ASK_BYTES} bytes")
  printf '  fail AC-2 — hero ask exit=%d, %d bytes\n' "${ASK_EXIT}" "${ASK_BYTES}" >&2
fi
{
  printf '\n#### AC-2\n\n_Assertion:_ `hero ask "%s"` exits 0 + >100 bytes (canonical pre-polish silent-no-op)\n\n' "${ASK_QUESTION}"
  printf '\`\`\`\n'
  head -c 1500 "${ASK_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-3: hero recap -----------------------------------------------

RECAP_OUT="${E2E_RUN_DIR}/recap.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" recap --since 7d ) > "${RECAP_OUT}" 2>&1
RECAP_EXIT=$?
RECAP_BYTES=$(wc -c < "${RECAP_OUT}" | tr -d ' ')

if [[ "${RECAP_EXIT}" -eq 0 && "${RECAP_BYTES}" -gt 0 ]]; then
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero recap --since 7d → ${RECAP_BYTES} bytes")
  printf '  pass AC-3 — hero recap (%d bytes)\n' "${RECAP_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero recap exit=${RECAP_EXIT}, ${RECAP_BYTES} bytes")
  printf '  fail AC-3 — hero recap exit=%d, %d bytes\n' "${RECAP_EXIT}" "${RECAP_BYTES}" >&2
fi
{
  printf '\n#### AC-3\n\n_Assertion:_ `hero recap --since 7d` exits 0 + non-zero stdout\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${RECAP_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-4: hero next ------------------------------------------------

NEXT_OUT="${E2E_RUN_DIR}/next.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" next ) > "${NEXT_OUT}" 2>&1
NEXT_EXIT=$?
NEXT_BYTES=$(wc -c < "${NEXT_OUT}" | tr -d ' ')

if [[ "${NEXT_EXIT}" -eq 0 && "${NEXT_BYTES}" -gt 0 ]]; then
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero next → ${NEXT_BYTES} bytes")
  printf '  pass AC-4 — hero next (%d bytes)\n' "${NEXT_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero next exit=${NEXT_EXIT}, ${NEXT_BYTES} bytes")
  printf '  fail AC-4 — hero next exit=%d, %d bytes\n' "${NEXT_EXIT}" "${NEXT_BYTES}" >&2
fi
{
  printf '\n#### AC-4\n\n_Assertion:_ `hero next` exits 0 + non-zero stdout\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${NEXT_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-5: hero resume ----------------------------------------------

RESUME_OUT="${E2E_RUN_DIR}/resume.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" resume --budget 500 ) > "${RESUME_OUT}" 2>&1
RESUME_EXIT=$?
RESUME_BYTES=$(wc -c < "${RESUME_OUT}" | tr -d ' ')

if [[ "${RESUME_EXIT}" -eq 0 && "${RESUME_BYTES}" -gt 0 ]]; then
  E2E_AC_IDS+=("AC-5"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero resume --budget 500 → ${RESUME_BYTES} bytes")
  printf '  pass AC-5 — hero resume (%d bytes)\n' "${RESUME_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-5"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero resume exit=${RESUME_EXIT}, ${RESUME_BYTES} bytes")
  printf '  fail AC-5 — hero resume exit=%d, %d bytes\n' "${RESUME_EXIT}" "${RESUME_BYTES}" >&2
fi
{
  printf '\n#### AC-5\n\n_Assertion:_ `hero resume --budget 500` exits 0 + non-zero stdout\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${RESUME_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-6: AC graph reflects this spec's ingest ---------------------

AC_OUT="${E2E_RUN_DIR}/ac_list.txt"
( cd "${REPO_ROOT}" && "${HERO_BIN}" ac list e2e-discovery --json ) > "${AC_OUT}" 2>&1
AC_EXIT=$?

if [[ "${AC_EXIT}" -eq 0 ]] && grep -q '"ac_id":"AC-1"' "${AC_OUT}"; then
  E2E_AC_IDS+=("AC-6"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac list found AC-1 for e2e-discovery")
  printf '  pass AC-6 — ac list found AC-1\n' >&2
else
  E2E_AC_IDS+=("AC-6"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac list exit=${AC_EXIT} or AC-1 not found")
  printf '  fail AC-6 — ac list exit=%d or AC-1 absent\n' "${AC_EXIT}" >&2
fi
{
  printf '\n#### AC-6\n\n_Assertion:_ `hero ac list e2e-discovery --json` returns AC-1\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${AC_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- finalize -------------------------------------------------------

e2e_finish
exit $?
