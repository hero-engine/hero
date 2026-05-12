#!/usr/bin/env bash
# scripts/e2e/onboarding.sh — Onboarding area smoke.
#
# Spec:    .hero/planning/features/e2e-onboarding/spec.md
# Goal:    Prove `hero init` + `hero scan` + `hero status` produce a
#          populated, queryable workspace from a clean repo.
#
# Usage:
#   scripts/e2e/onboarding.sh           # run, leave results in tmp/
#   scripts/e2e/onboarding.sh --record  # also call `hero ac record`
#   HERO_BIN=/path/to/hero scripts/e2e/onboarding.sh

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

E2E_SPEC_SLUG="e2e-onboarding"

# A throwaway sandbox repo. Never reuse — every run starts clean so
# AC-1 actually exercises a fresh `hero init`.
SANDBOX_PARENT="${REPO_ROOT}/tmp/e2e/onboarding-sandboxes"
SANDBOX_TS="$(date -u +%Y%m%dT%H%M%SZ)"
WORK_DIR="${SANDBOX_PARENT}/sandbox-${SANDBOX_TS}"

mkdir -p "${WORK_DIR}"
git -C "${WORK_DIR}" init -q

# Drop a tiny readme so `hero scan` has something to work with —
# without it the stack analyzer gets very thin output.
cat > "${WORK_DIR}/README.md" <<'README'
# E2E Onboarding Sandbox

Synthetic sandbox repo for the e2e-onboarding suite. Created and
nuked per run.
README

cat > "${WORK_DIR}/main.go" <<'GO'
package main

import "fmt"

func main() {
	fmt.Println("hello from the e2e sandbox")
}
GO

git -C "${WORK_DIR}" add -A >/dev/null
git -C "${WORK_DIR}" -c user.name=e2e -c user.email=e2e@example.com \
  commit -q -m "init: e2e sandbox bootstrap"

e2e_init "onboarding"
export WORK_DIR

# --- AC-1: hero init ------------------------------------------------

INIT_OUT="${E2E_RUN_DIR}/init_stdout.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" init ) > "${INIT_OUT}" 2>&1
INIT_EXIT=$?

if [[ "${INIT_EXIT}" -eq 0 && -f "${WORK_DIR}/.hero/hero.json" && -f "${WORK_DIR}/AGENTS.md" ]]; then
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("init exit=0; .hero/hero.json + AGENTS.md present")
  printf '  pass AC-1 — init bootstrapped workspace\n' >&2
else
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("init exit=${INIT_EXIT}; missing files")
  printf '  fail AC-1 — init exit=%d or files missing\n' "${INIT_EXIT}" >&2
fi
{
  printf '\n#### AC-1\n\n_Assertion:_ `hero init` exits 0 + .hero/hero.json + AGENTS.md exist\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${INIT_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-2: hero scan ------------------------------------------------

# Use a separate captured output file because the assertion needs to
# grep what scan printed.
SCAN_OUT="${E2E_RUN_DIR}/scan_stdout.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" scan ) > "${SCAN_OUT}" 2>&1
SCAN_EXIT=$?

# Replay through e2e_step shape: register the AC manually since the
# assertion needs the captured output, not the captured stdout from
# inside e2e_step.
if [[ "${SCAN_EXIT}" -eq 0 ]] && grep -q "Graph ingest summary" "${SCAN_OUT}"; then
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exited 0 and printed Graph ingest summary")
  printf '  pass AC-2 — scan exit=%d, summary present\n' "${SCAN_EXIT}" >&2
else
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("scan exit=${SCAN_EXIT} or summary absent")
  printf '  fail AC-2 — scan exit=%d or no summary\n' "${SCAN_EXIT}" >&2
fi
{
  printf '\n#### AC-2\n\n_Assertion:_ scan exit 0 + "Graph ingest summary" in output\n\n'
  printf '\`\`\`\n'
  head -c 2000 "${SCAN_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-3: hero status produces output -------------------------------

STATUS_OUT="${E2E_RUN_DIR}/status_stdout.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" status ) > "${STATUS_OUT}" 2>&1
STATUS_EXIT=$?
STATUS_BYTES=$(wc -c < "${STATUS_OUT}" | tr -d ' ')

if [[ "${STATUS_EXIT}" -eq 0 && "${STATUS_BYTES}" -gt 0 ]]; then
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("status exit=0, ${STATUS_BYTES} bytes")
  printf '  pass AC-3 — status %d bytes\n' "${STATUS_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("status exit=${STATUS_EXIT}, ${STATUS_BYTES} bytes")
  printf '  fail AC-3 — status exit=%d %d bytes\n' "${STATUS_EXIT}" "${STATUS_BYTES}" >&2
fi
{
  printf '\n#### AC-3\n\n_Assertion:_ status exit 0 + non-zero bytes (catches silent-no-op)\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${STATUS_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-4: AC graph reflects this spec's ingest ----------------------

# This step queries the *outer* hero repo (not the sandbox) because
# that's where this spec was scanned and its Criterion nodes live.
# We deliberately point at REPO_ROOT, not WORK_DIR.
AC_OUT="${E2E_RUN_DIR}/ac_list_stdout.txt"
( cd "${REPO_ROOT}" && "${HERO_BIN}" ac list e2e-onboarding --json ) > "${AC_OUT}" 2>&1
AC_EXIT=$?

if [[ "${AC_EXIT}" -eq 0 ]] && grep -q '"ac_id":"AC-1"' "${AC_OUT}"; then
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac list found AC-1 for e2e-onboarding")
  printf '  pass AC-4 — ac list found AC-1\n' >&2
else
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac list exit=${AC_EXIT} or AC-1 not found")
  printf '  fail AC-4 — ac list exit=%d or AC-1 absent\n' "${AC_EXIT}" >&2
fi
{
  printf '\n#### AC-4\n\n_Assertion:_ `hero ac list e2e-onboarding --json` returns AC-1\n\n'
  printf '\`\`\`\n'
  head -c 1500 "${AC_OUT}"
  printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-5: scan idempotency ------------------------------------------

STATS_BEFORE=$(cd "${WORK_DIR}" && "${HERO_BIN}" graph stats 2>/dev/null | grep -oE 'Current nodes \([0-9]+\)' | head -1 || echo "")
( cd "${WORK_DIR}" && "${HERO_BIN}" scan >/dev/null 2>&1 )
STATS_AFTER=$(cd "${WORK_DIR}" && "${HERO_BIN}" graph stats 2>/dev/null | grep -oE 'Current nodes \([0-9]+\)' | head -1 || echo "")

if [[ -n "${STATS_BEFORE}" && "${STATS_BEFORE}" == "${STATS_AFTER}" ]]; then
  E2E_AC_IDS+=("AC-5"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("idempotent: ${STATS_BEFORE}")
  printf '  pass AC-5 — idempotent: %s\n' "${STATS_BEFORE}" >&2
else
  E2E_AC_IDS+=("AC-5"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("counts differ: before=${STATS_BEFORE} after=${STATS_AFTER}")
  printf '  fail AC-5 — drift: %s → %s\n' "${STATS_BEFORE}" "${STATS_AFTER}" >&2
fi
{
  printf '\n#### AC-5\n\n_Assertion:_ second `hero scan` produces identical node total\n\n'
  printf -- '- Before: %s\n- After:  %s\n' "${STATS_BEFORE}" "${STATS_AFTER}"
} >> "${E2E_LOG}"

# --- finalize -------------------------------------------------------

e2e_finish
exit $?
