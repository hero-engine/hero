#!/usr/bin/env bash
# scripts/e2e/lib.sh — shared harness for per-area e2e suites.
#
# Each suite at scripts/e2e/<area>.sh sources this lib, calls
# `e2e_init <area-slug>` to set up the run dir, then drives ACs via
# `e2e_step <ac-id> <command...>`. At the end `e2e_finish` writes
# results.json + observations.md and ingests the run via
# `hero ac record` if --record was passed.
#
# Output layout per run:
#   tmp/e2e/<area>-<UTC-timestamp>/
#     results.json          — feeds `hero ac record`
#     observations.md       — eyeball pass for the operator
#     <ac-id>.txt           — captured stdout/stderr per step
#
# Conventions every suite must honor:
#   - HERO_BIN env var (default: hero) — binary under test
#   - All steps run in the suite's WORK_DIR (the project being
#     onboarded / scanned / etc.) unless the suite explicitly cds out
#   - One e2e_step per AC; the AC id is the source of truth for
#     mapping back to the graph

set -uo pipefail

E2E_LIB_VERSION=1

HERO_BIN="${HERO_BIN:-hero}"
SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${SCRIPT_ROOT}/.." && pwd)"

# Result accumulator — array of JSON objects, joined at finish time.
# Bash 3 (macOS default) doesn't have associative arrays we want here,
# so we keep a parallel-arrays setup with explicit indices.
declare -a E2E_AC_IDS=()
declare -a E2E_STATUSES=()
declare -a E2E_DURATIONS=()
declare -a E2E_DETAILS=()

# e2e_init <area-slug>
#   Sets up the run dir, opens the observation log, exports
#   E2E_AREA / E2E_RUN_DIR / E2E_LOG. Must be called before any
#   e2e_step.
e2e_init() {
  local area="$1"
  local ts
  ts="$(date -u +%Y%m%dT%H%M%SZ)"

  E2E_AREA="${area}"
  E2E_RUN_TS="${ts}"
  E2E_RUN_DIR="${REPO_ROOT}/tmp/e2e/${area}-${ts}"
  E2E_LOG="${E2E_RUN_DIR}/observations.md"
  E2E_RESULTS="${E2E_RUN_DIR}/results.json"

  mkdir -p "${E2E_RUN_DIR}"

  cat > "${E2E_LOG}" <<EOF
# e2e/${area} — ${ts}

- **Hero binary:** \`${HERO_BIN}\`
- **Working dir:** \`${WORK_DIR:-<unset>}\`
- **Spec:** \`.hero/planning/features/e2e-${area}/spec.md\`

## Step results
EOF
}

# e2e_section <heading>
#   Append a markdown section header to the observation log.
e2e_section() {
  printf '\n### %s\n' "$1" >> "${E2E_LOG}"
}

# e2e_step <ac-id> <command...>
#   Run command, time it, capture exit code + output, record result
#   for the named AC. Status:
#     pass — exit 0
#     fail — non-zero exit
#   The output goes to <ac-id>.txt and a summary appears in the log.
e2e_step() {
  local ac_id="$1"; shift
  local out_file="${E2E_RUN_DIR}/${ac_id}.txt"

  printf '\n#### %s\n\n```\n$ %s\n```\n' "${ac_id}" "$*" >> "${E2E_LOG}"

  local start_ms end_ms elapsed_ms exit_code
  start_ms=$(_e2e_now_ms)
  ( cd "${WORK_DIR:-${REPO_ROOT}}" && "$@" ) > "${out_file}" 2>&1
  exit_code=$?
  end_ms=$(_e2e_now_ms)
  elapsed_ms=$((end_ms - start_ms))

  local status="pass"
  if [[ "${exit_code}" -ne 0 ]]; then
    status="fail"
  fi

  E2E_AC_IDS+=("${ac_id}")
  E2E_STATUSES+=("${status}")
  E2E_DURATIONS+=("${elapsed_ms}")
  E2E_DETAILS+=("exit=${exit_code}")

  printf '%s in %dms (exit %d)\n\n' "${status}" "${elapsed_ms}" "${exit_code}" >> "${E2E_LOG}"
  printf '<details><summary>output</summary>\n\n```\n' >> "${E2E_LOG}"
  # Truncate captured output to keep the log small but useful.
  head -c 2000 "${out_file}" >> "${E2E_LOG}"
  printf '\n```\n</details>\n' >> "${E2E_LOG}"

  # Echo a compact line to stderr for live progress.
  printf '  %s %s — %dms\n' "${status}" "${ac_id}" "${elapsed_ms}" >&2
}

# e2e_assert <ac-id> <description> <bash-expression>
#   Like e2e_step but the command is an inline bash test (e.g.
#   '[[ -f .hero/spec.md ]]'). The description is captured in the log
#   so the assertion's intent is recorded.
e2e_assert() {
  local ac_id="$1"; shift
  local desc="$1"; shift
  local expr="$*"

  printf '\n#### %s\n\n_Assertion:_ %s\n\n```\n%s\n```\n' "${ac_id}" "${desc}" "${expr}" >> "${E2E_LOG}"

  local start_ms end_ms elapsed_ms
  start_ms=$(_e2e_now_ms)
  if ( cd "${WORK_DIR:-${REPO_ROOT}}" && eval "${expr}" ); then
    local status="pass"
  else
    local status="fail"
  fi
  end_ms=$(_e2e_now_ms)
  elapsed_ms=$((end_ms - start_ms))

  E2E_AC_IDS+=("${ac_id}")
  E2E_STATUSES+=("${status}")
  E2E_DURATIONS+=("${elapsed_ms}")
  E2E_DETAILS+=("assertion: ${desc}")

  printf '%s in %dms\n' "${status}" "${elapsed_ms}" >> "${E2E_LOG}"
  printf '  %s %s — %dms\n' "${status}" "${ac_id}" "${elapsed_ms}" >&2
}

# e2e_finish
#   Write results.json. Optionally call `hero ac record` if --record
#   is passed (or if E2E_RECORD=1 in env).
e2e_finish() {
  local sha
  sha="$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo)"

  # Build results.json manually — no jq dependency.
  {
    printf '['
    local i count
    count=${#E2E_AC_IDS[@]}
    for ((i=0; i<count; i++)); do
      local ac_id="${E2E_AC_IDS[$i]}"
      local status="${E2E_STATUSES[$i]}"
      local elapsed="${E2E_DURATIONS[$i]}"
      printf '%s\n  {' "$([[ $i -eq 0 ]] && echo "" || echo ",")"
      printf '"ac":"%s:%s",' "${E2E_SPEC_SLUG:-e2e-${E2E_AREA}}" "${ac_id}"
      printf '"status":"%s",' "${status}"
      printf '"ts":"%s",' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
      printf '"sha":"%s",' "${sha}"
      printf '"duration_ms":%s,' "${elapsed}"
      printf '"run_id":"%s-%s"' "${E2E_AREA}" "${E2E_RUN_TS}"
      printf '}'
    done
    printf '\n]\n'
  } > "${E2E_RESULTS}"

  # Tally + log summary.
  local pass=0 fail=0 i count
  count=${#E2E_STATUSES[@]}
  for ((i=0; i<count; i++)); do
    if [[ "${E2E_STATUSES[$i]}" == "pass" ]]; then
      pass=$((pass+1))
    else
      fail=$((fail+1))
    fi
  done

  cat >> "${E2E_LOG}" <<EOF

## Summary

- **Pass:** ${pass}
- **Fail:** ${fail}
- **Results JSON:** \`${E2E_RESULTS}\`
- **Commit:** \`${sha}\`
EOF

  printf '\n' >&2
  printf 'e2e/%s — %d pass, %d fail (%d total)\n' "${E2E_AREA}" "${pass}" "${fail}" "$((pass+fail))" >&2
  printf 'Results: %s\n' "${E2E_RESULTS}" >&2
  printf 'Log:     %s\n' "${E2E_LOG}" >&2

  # Optional graph ingest. Honor either flag (--record on the script
  # cmdline — caller must parse and set E2E_RECORD=1) or env var.
  if [[ "${E2E_RECORD:-0}" == "1" ]]; then
    printf '\nIngesting results into AC graph...\n' >&2
    "${HERO_BIN}" ac record "${E2E_RESULTS}"
  fi

  # Exit code: non-zero if any AC failed (so CI can short-circuit).
  if [[ "${fail}" -gt 0 ]]; then
    return 1
  fi
  return 0
}

# _e2e_now_ms — millisecond timestamp. GNU date supports `%s%3N`;
# BSD/macOS date emits the literal `%3N` and breaks bash arithmetic.
# Detect numeric-only output, otherwise fall back to perl.
_e2e_now_ms() {
  local raw
  raw="$(date +%s%3N 2>/dev/null)"
  if [[ "${raw}" =~ ^[0-9]+$ ]]; then
    printf '%s\n' "${raw}"
    return
  fi
  perl -MTime::HiRes=time -e 'printf "%d\n", time*1000'
}
