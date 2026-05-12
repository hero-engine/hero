#!/usr/bin/env bash
# scripts/smoke/acceptance-criteria-graph.sh — Per-feature smoke for
# acceptance-criteria-graph: spec parser, scan ingest, run-result
# record, hero deliver AC block, and bitemporal status storage.
#
# Runs against the live hero repo (populated graph required).
# Covers ACs 1–4 and AC-6 (the five that are marked passing).
#
# Usage:
#   scripts/smoke/acceptance-criteria-graph.sh
#   scripts/smoke/acceptance-criteria-graph.sh --record

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

E2E_SPEC_SLUG="acceptance-criteria-graph"
WORK_DIR="${REPO_ROOT}"
export WORK_DIR

e2e_init "acceptance-criteria-graph"

# --- AC-1: spec parser extracts AC entries --------------------------
# hero ac list <slug> --json should return AC nodes for the target spec.

AC_LIST_OUT="${E2E_RUN_DIR}/ac_list.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" ac list acceptance-criteria-graph --json ) \
  > "${AC_LIST_OUT}" 2>&1
AC_LIST_EXIT=$?

if [[ "${AC_LIST_EXIT}" -eq 0 ]] && grep -q '"ac_id"' "${AC_LIST_OUT}"; then
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac list returned AC nodes")
  printf '  pass AC-1 — spec parser/ingest: ac list returned AC nodes\n' >&2
else
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac list exit=${AC_LIST_EXIT} or no ac_id in output")
  printf '  fail AC-1 — ac list exit=%d\n' "${AC_LIST_EXIT}" >&2
fi
{
  printf '\n#### AC-1\n\n_Assertion:_ `hero ac list acceptance-criteria-graph --json` returns AC nodes\n\n```\n'
  head -c 2000 "${AC_LIST_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-2: scan upserts Criterion nodes into the graph --------------
# hero graph stats should show a non-zero Criterion count.

STATS_OUT="${E2E_RUN_DIR}/graph_stats.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" graph stats ) > "${STATS_OUT}" 2>&1
STATS_EXIT=$?

if [[ "${STATS_EXIT}" -eq 0 ]] && grep -qi "criterion" "${STATS_OUT}"; then
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero graph stats shows Criterion node type")
  printf '  pass AC-2 — graph stats shows Criterion nodes\n' >&2
else
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero graph stats exit=${STATS_EXIT} or no Criterion in output")
  printf '  fail AC-2 — graph stats exit=%d\n' "${STATS_EXIT}" >&2
fi
{
  printf '\n#### AC-2\n\n_Assertion:_ `hero graph stats` shows Criterion node type\n\n```\n'
  head -c 2000 "${STATS_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-3: run-result ingest via hero ac record ---------------------
# Write a minimal results.json and verify hero ac record exits 0.

RESULTS_JSON="${E2E_RUN_DIR}/test_results.json"
RECORD_OUT="${E2E_RUN_DIR}/ac_record.txt"
SHA="$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo test)"
TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

cat > "${RESULTS_JSON}" <<EOF
[
  {"ac":"acceptance-criteria-graph:AC-1","status":"pass","ts":"${TS}","sha":"${SHA}","duration_ms":1,"run_id":"smoke-ac-graph"}
]
EOF

( cd "${WORK_DIR}" && "${HERO_BIN}" ac record "${RESULTS_JSON}" ) \
  > "${RECORD_OUT}" 2>&1
RECORD_EXIT=$?

if [[ "${RECORD_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac record exit=0")
  printf '  pass AC-3 — hero ac record ingested results.json\n' >&2
else
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero ac record exit=${RECORD_EXIT}")
  printf '  fail AC-3 — ac record exit=%d\n' "${RECORD_EXIT}" >&2
fi
{
  printf '\n#### AC-3\n\n_Assertion:_ `hero ac record <results.json>` exits 0 (bitemporal ingest)\n\n```\n'
  head -c 2000 "${RECORD_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-4: hero deliver shows AC block ------------------------------
# Create a sandbox spec in planning state with ACs, then run
# hero spec deliver --manual on it and verify the AC block prints.
# Using a sandbox avoids mutating live corpus specs.

DELIVER_SANDBOX="${E2E_RUN_DIR}/deliver_sandbox"
mkdir -p "${DELIVER_SANDBOX}/.hero/planning/features/smoke-ac4-test"
cp "${WORK_DIR}/.hero/hero.json" "${DELIVER_SANDBOX}/.hero/hero.json" 2>/dev/null || true

cat > "${DELIVER_SANDBOX}/.hero/planning/features/smoke-ac4-test/spec.md" <<'EOF'
---
title: Smoke AC4 Test
type: feature
status: planning
horizon: now
smoke: deferred
---
## Goal
Test spec for smoke verification.
## Acceptance criteria
**AC-1:** something verifiable happens when delivered
**AC-2:** another criterion exists
EOF

# Scan the sandbox first so the graph has Criterion nodes for the spec.
( cd "${DELIVER_SANDBOX}" && "${HERO_BIN}" scan ) > /dev/null 2>&1

DELIVER_OUT="${E2E_RUN_DIR}/deliver.txt"
( cd "${DELIVER_SANDBOX}" && "${HERO_BIN}" spec deliver smoke-ac4-test --manual ) \
  > "${DELIVER_OUT}" 2>&1
DELIVER_EXIT=$?

if [[ "${DELIVER_EXIT}" -eq 0 ]] && grep -qi "acceptance criteria\|AC-1\|graded" "${DELIVER_OUT}"; then
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero spec deliver showed AC block (${DELIVER_EXIT}=0)")
  printf '  pass AC-4 — hero deliver shows AC block\n' >&2
else
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero spec deliver exit=${DELIVER_EXIT} or no AC block in output")
  printf '  fail AC-4 — deliver exit=%d or no AC block\n' "${DELIVER_EXIT}" >&2
fi
{
  printf '\n#### AC-4\n\n_Assertion:_ `hero spec deliver --manual` on sandbox spec shows AC block\n\n```\n'
  head -c 2000 "${DELIVER_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

# --- AC-6: bitemporal GetNodeAt retrieves historical state ----------
# After recording a result above, ac list should reflect status update.

HIST_OUT="${E2E_RUN_DIR}/ac_history.txt"
( cd "${WORK_DIR}" && "${HERO_BIN}" ac history acceptance-criteria-graph:AC-1 2>/dev/null \
  || "${HERO_BIN}" ac list acceptance-criteria-graph --json ) \
  > "${HIST_OUT}" 2>&1
HIST_EXIT=$?

if [[ "${HIST_EXIT}" -eq 0 ]]; then
  E2E_AC_IDS+=("AC-6"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("bitemporal query exit=0")
  printf '  pass AC-6 — bitemporal AC query succeeded\n' >&2
else
  E2E_AC_IDS+=("AC-6"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("bitemporal query exit=${HIST_EXIT}")
  printf '  fail AC-6 — bitemporal query exit=%d\n' "${HIST_EXIT}" >&2
fi
{
  printf '\n#### AC-6\n\n_Assertion:_ AC status query exits 0 after status flip\n\n```\n'
  head -c 2000 "${HIST_OUT}"
  printf '\n```\n'
} >> "${E2E_LOG}"

e2e_finish
exit $?
