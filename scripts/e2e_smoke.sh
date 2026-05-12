#!/usr/bin/env bash
# e2e_smoke.sh — exercise hero end-to-end against a public repo.
#
# Goal: a repeatable take-it-for-a-spin that simulates a new user
# onboarding to an unfamiliar codebase with hero, then doing the kind
# of discovery/planning/pre-flight work hero is supposed to make easy.
#
# Outputs:
#   - per-step timing + exit code
#   - a markdown observation log under tmp/e2e-smoke/<repo>-<timestamp>/
#   - the cloned target repo left in tmp/e2e-smoke/<repo>/ for poking
#
# Usage:
#   scripts/e2e_smoke.sh                       # default: go-task/task
#   scripts/e2e_smoke.sh <github-owner/repo>   # any public Go/JS/Py repo
#
# Re-run safe: nukes the temp clone each time (set KEEP=1 to preserve).

set -uo pipefail

REPO_SLUG="${1:-go-task/task}"
HERO_BIN="${HERO_BIN:-hero}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP_BASE="${ROOT}/tmp/e2e-smoke"
RUN_TS="$(date -u +%Y%m%dT%H%M%SZ)"
SAFE_SLUG="${REPO_SLUG//\//-}"
RUN_DIR="${TMP_BASE}/${SAFE_SLUG}-${RUN_TS}"
CLONE_DIR="${TMP_BASE}/${SAFE_SLUG}"
LOG="${RUN_DIR}/observations.md"

mkdir -p "${RUN_DIR}"

# --- Logging helpers ---------------------------------------------------

# Append a section header to the observation log.
section() {
  printf '\n## %s\n\n' "$1" | tee -a "${LOG}" >&2
}

# Run a hero command, time it, capture exit code + stdout/stderr,
# append a markdown summary to the log. Args: <label> <cmd...>
run_step() {
  local label="$1"; shift
  local out_file="${RUN_DIR}/${label// /_}.txt"
  printf '\n### %s\n' "$label" >> "${LOG}"
  printf '\n```\n$ %s\n```\n' "$*" >> "${LOG}"

  local start_ns end_ns elapsed_ms
  start_ns=$(perl -MTime::HiRes=time -e 'printf "%d\n", time*1e9')
  ( cd "${CLONE_DIR}" && "$@" ) > "${out_file}" 2>&1
  local rc=$?
  end_ns=$(perl -MTime::HiRes=time -e 'printf "%d\n", time*1e9')
  elapsed_ms=$(( (end_ns - start_ns) / 1000000 ))

  local lines bytes
  lines=$(wc -l < "${out_file}" | tr -d ' ')
  bytes=$(wc -c < "${out_file}" | tr -d ' ')

  printf '\n- exit code: **%d**  \n- elapsed: **%dms**  \n- output: **%s lines / %s bytes**\n' \
    "${rc}" "${elapsed_ms}" "${lines}" "${bytes}" >> "${LOG}"

  if [ "${rc}" -ne 0 ]; then
    printf '\n**FAILED** — first 30 lines of output:\n\n```\n' >> "${LOG}"
    head -30 "${out_file}" >> "${LOG}"
    printf '```\n' >> "${LOG}"
  else
    printf '\n<details><summary>output (first 20 lines)</summary>\n\n```\n' >> "${LOG}"
    head -20 "${out_file}" >> "${LOG}"
    printf '```\n</details>\n' >> "${LOG}"
  fi

  printf '[%s] %s — rc=%d %dms %sL\n' "$(date -u +%H:%M:%S)" "${label}" "${rc}" "${elapsed_ms}" "${lines}" >&2
  return "${rc}"
}

# --- Setup -------------------------------------------------------------

cat > "${LOG}" <<EOF
# Hero E2E Smoke Test — ${REPO_SLUG}

- **Run timestamp:** ${RUN_TS}
- **Hero binary:** $(command -v "${HERO_BIN}")
- **Hero version:** $("${HERO_BIN}" --version 2>&1)
- **Target repo:** https://github.com/${REPO_SLUG}
- **Clone dir:** \`${CLONE_DIR}\`
- **Run artifacts:** \`${RUN_DIR}\`

This run exercises the workflows a new user would touch in their first
session with hero in an unfamiliar codebase. Each step records its
exit code, wall-clock time, output volume, and a snippet of output.
The observation section at the bottom is filled in by the operator
after the run.
EOF

section "Setup — clone target repo"

if [ -d "${CLONE_DIR}/.git" ] && [ -z "${KEEP:-}" ]; then
  echo "Removing previous clone at ${CLONE_DIR}" | tee -a "${LOG}" >&2
  rm -rf "${CLONE_DIR}"
fi

if [ ! -d "${CLONE_DIR}/.git" ]; then
  echo "Cloning https://github.com/${REPO_SLUG} → ${CLONE_DIR}" | tee -a "${LOG}" >&2
  git clone --depth=50 "https://github.com/${REPO_SLUG}.git" "${CLONE_DIR}" 2>&1 | tail -3 | tee -a "${LOG}" >&2
fi

# --- Step 1: onboarding ------------------------------------------------

section "Onboarding — init + scan + status"

run_step "init" "${HERO_BIN}" init
run_step "scan" "${HERO_BIN}" scan
run_step "status" "${HERO_BIN}" status
run_step "dashboard" "${HERO_BIN}" dashboard

# --- Step 2: discovery -------------------------------------------------

section "Discovery — search + ask + relevant"

# Pick a probably-meaningful search term from the README (heuristic).
SEARCH_TERM=$(grep -h -m1 -oE '\b[A-Z][a-zA-Z]{4,}\b' "${CLONE_DIR}/README.md" 2>/dev/null | head -1 || echo "command")
echo "Auto-picked search term: ${SEARCH_TERM}" >> "${LOG}"

run_step "search-by-term" "${HERO_BIN}" search "${SEARCH_TERM}"
run_step "ask-architecture" "${HERO_BIN}" ask "what is the overall architecture of this project"

# Pick a real file to ask about. Prefer the largest .go file as a proxy
# for a hot file someone might want to understand.
HOT_FILE=$(cd "${CLONE_DIR}" && find . -name '*.go' -not -path './vendor/*' -not -path './.git/*' \
  -exec wc -l {} + 2>/dev/null | sort -rn | head -2 | tail -1 | awk '{print $2}')
HOT_FILE="${HOT_FILE:-./README.md}"
echo "Auto-picked hot file: ${HOT_FILE}" >> "${LOG}"

run_step "relevant-hot-file" "${HERO_BIN}" relevant --files "${HOT_FILE}"

# --- Step 3: planning --------------------------------------------------

section "Planning — design a feature, view the spec graph"

run_step "design-fake-feature" "${HERO_BIN}" spec new "experimental-cli-flag" --type feature
run_step "graph-stats" "${HERO_BIN}" graph stats
run_step "graph-fake-feature" "${HERO_BIN}" graph experimental-cli-flag

# --- Step 4: pre-flight ------------------------------------------------

section "Pre-flight — conflicts + impact"

run_step "check-conflicts" "${HERO_BIN}" check conflicts experimental-cli-flag
run_step "impact-hot-file" "${HERO_BIN}" impact "${HOT_FILE}"

# --- Step 5: meta ------------------------------------------------------

section "Meta — what does hero say about itself"

run_step "blocked" "${HERO_BIN}" blocked
run_step "suggest" "${HERO_BIN}" suggest
run_step "next" "${HERO_BIN}" next

# --- Wrap up -----------------------------------------------------------

section "Operator observations (fill in after reviewing artifacts)"
cat >> "${LOG}" <<'OBS'
**Onboarding ease (1-5):** _

**Output usefulness (1-5):** _

**Friction points / surprises:**
-

**Polish targets surfaced:**
-

**Things hero did well:**
-
OBS

echo "" >&2
echo "==> done. Observation log: ${LOG}" >&2
echo "==> clone preserved at: ${CLONE_DIR} (set KEEP=1 to keep across runs)" >&2
