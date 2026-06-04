#!/usr/bin/env bash
# scripts/e2e/handoff.sh — Handoff continuity area smoke.
#
# Spec:    .hero/planning/features/e2e-handoff-continuity/spec.md
# Goal:    Prove the handoff "magic" survives a real cross-machine
#          leap with the real `hero` binary: machine A captures
#          context and commits; machine B is a fresh `git clone` of A
#          (so ONLY committed files travel — graph.db is gitignored and
#          does NOT come along), and B reconstructs A's ask + suggestion
#          via `hero next ingest` + the `hero next` read surface.
#
# This is the full-fidelity sibling of the in-process Go guardrail in
# internal/cli/handoff_continuity_test.go (the always-on `go test ./...`
# tripwire). Here we drive the actual binary across two sandbox repos.
#
# Usage:
#   scripts/e2e/handoff.sh           # run, leave results in tmp/
#   scripts/e2e/handoff.sh --record  # also call `hero ac record`
#   HERO_BIN=/path/to/hero scripts/e2e/handoff.sh

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
      sed -n '2,19p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) printf 'unknown arg: %s\n' "${arg}" >&2; exit 2 ;;
  esac
done
export E2E_RECORD

# --- per-suite setup ------------------------------------------------

E2E_SPEC_SLUG="e2e-handoff-continuity"

# Two sandbox repos, A and B, under the run dir. A captures + commits;
# B is a fresh clone of A. Everything happens in the sandbox — the
# outer repo is never touched.
SANDBOX_ROOT="${REPO_ROOT}/tmp/e2e/handoff-sandbox-$(date -u +%Y%m%dT%H%M%SZ)"
MACHINE_A="${SANDBOX_ROOT}/machine-a"
MACHINE_B="${SANDBOX_ROOT}/machine-b"
USER_SLUG="alice"
HANDOFF_FILE=".hero/next/${USER_SLUG}.md"

# Distinct, greppable text so an assertion proves the exact value
# reconstructed on B, not a coincidental substring.
ASK_TEXT="MACHINE_A_ASK where did we leave off on cross-machine handoff"
SUGGEST_TEXT="MACHINE_A_SUGGESTION land the cross-machine continuity guardrail"

# WORK_DIR for the lib defaults; individual steps cd explicitly.
WORK_DIR="${SANDBOX_ROOT}"
mkdir -p "${SANDBOX_ROOT}"

e2e_init "handoff-continuity"
export WORK_DIR

# set_default_agent <repo-dir> <slug> — pin tracking.default_agent so
# nextUserSlug is deterministic (and A's path matches B's path). No jq
# dependency; python3 merges the field.
set_default_agent() {
  local repo="$1" slug="$2"
  python3 - "$repo/.hero/hero.json" "$slug" <<'PY'
import json, sys
path, slug = sys.argv[1], sys.argv[2]
with open(path) as f:
    cfg = json.load(f)
cfg.setdefault("tracking", {})["default_agent"] = "human/" + slug
with open(path, "w") as f:
    json.dump(cfg, f, indent=2)
PY
}

# --- AC-1: cross-machine reconstruction (the core) ------------------
#
# A: init a git repo, init hero, capture ask + suggestion, checkpoint
# (projects .hero/next/<user>.md), commit. B: fresh clone (graph.db
# gitignored → does NOT travel), ingest, then read back A's context.

A_LOG="${E2E_RUN_DIR}/machine-a.txt"
{
  git init -q "${MACHINE_A}"
  cd "${MACHINE_A}"
  git config user.email "alice@example.com"
  git config user.name "alice"
  "${HERO_BIN}" init --no-agents --no-hooks
  set_default_agent "${MACHINE_A}" "${USER_SLUG}"
  "${HERO_BIN}" next ask "${ASK_TEXT}"
  "${HERO_BIN}" next suggest "${SUGGEST_TEXT}"
  "${HERO_BIN}" next checkpoint -q
  git add -A
  git commit -q -m "machine A: capture handoff context"
} > "${A_LOG}" 2>&1
A_EXIT=$?

# graph.db must NOT be tracked (it's the whole point — it can't travel).
GRAPH_TRACKED="$(cd "${MACHINE_A}" && git ls-files .hero/graph.db | head -1)"
HANDOFF_TRACKED="$(cd "${MACHINE_A}" && git ls-files "${HANDOFF_FILE}" | head -1)"

B_LOG="${E2E_RUN_DIR}/machine-b.txt"
{
  # Fresh clone — only committed files come across. graph.db is
  # gitignored on A so it is absent from the clone by construction.
  git clone -q "${MACHINE_A}" "${MACHINE_B}"
  cd "${MACHINE_B}"
  git config user.email "alice@example.com"
  git config user.name "alice"
  # Prove B starts with no graph before ingest.
  test ! -e .hero/graph.db && echo "B_NO_GRAPH_BEFORE_INGEST=1" || echo "B_NO_GRAPH_BEFORE_INGEST=0"
  "${HERO_BIN}" next ingest
} > "${B_LOG}" 2>&1
B_EXIT=$?

# Read back A's ask + suggestion on B via the real `hero next` surface.
B_ASK="$(cd "${MACHINE_B}" && "${HERO_BIN}" next ask)"
B_SUGGEST="$(cd "${MACHINE_B}" && "${HERO_BIN}" next suggest)"

AC1_OK=1
[[ "${A_EXIT}" -eq 0 ]] || AC1_OK=0
[[ "${B_EXIT}" -eq 0 ]] || AC1_OK=0
[[ -n "${HANDOFF_TRACKED}" ]] || AC1_OK=0                       # file travels
[[ -z "${GRAPH_TRACKED}" ]] || AC1_OK=0                          # graph does not
echo "${B_ASK}" | grep -qF "${ASK_TEXT}" || AC1_OK=0
echo "${B_SUGGEST}" | grep -qF "${SUGGEST_TEXT}" || AC1_OK=0

if [[ "${AC1_OK}" -eq 1 ]]; then
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("B reconstructed A's ask+suggestion from committed file (graph.db did not travel)")
  printf '  pass AC-1 — cross-machine reconstruction\n' >&2
else
  E2E_AC_IDS+=("AC-1"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("A_EXIT=${A_EXIT} B_EXIT=${B_EXIT} handoff_tracked=${HANDOFF_TRACKED:-NONE} graph_tracked=${GRAPH_TRACKED:-NONE}")
  printf '  fail AC-1 — A_EXIT=%d B_EXIT=%d handoff_tracked=%s graph_tracked=%s\n' \
    "${A_EXIT}" "${B_EXIT}" "${HANDOFF_TRACKED:-NONE}" "${GRAPH_TRACKED:-NONE}" >&2
fi
{
  printf '\n#### AC-1\n\n_Assertion:_ B (fresh clone, no graph.db) reconstructs A'\''s ask + suggestion after `hero next ingest`\n\n'
  printf '\`\`\`\nB next ask:\n%s\n\nB next suggest:\n%s\n\`\`\`\n' "${B_ASK}" "${B_SUGGEST}"
} >> "${E2E_LOG}"

# --- AC-2: same-machine fresh session (resume runs on A) ------------
#
# On A's own populated graph, the start-of-turn load (`hero resume`)
# runs clean. The per-user read surface still surfaces the ask.

RESUME_OUT="${E2E_RUN_DIR}/resume-a.txt"
( cd "${MACHINE_A}" && "${HERO_BIN}" resume --budget 800 ) > "${RESUME_OUT}" 2>&1
RESUME_EXIT=$?
RESUME_BYTES=$(wc -c < "${RESUME_OUT}" | tr -d ' ')
A_ASK="$(cd "${MACHINE_A}" && "${HERO_BIN}" next ask)"

if [[ "${RESUME_EXIT}" -eq 0 && "${RESUME_BYTES}" -gt 0 ]] && echo "${A_ASK}" | grep -qF "${ASK_TEXT}"; then
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("hero resume on A → ${RESUME_BYTES} bytes; ask surfaced")
  printf '  pass AC-2 — same-machine resume + ask (%d bytes)\n' "${RESUME_BYTES}" >&2
else
  E2E_AC_IDS+=("AC-2"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("resume exit=${RESUME_EXIT} bytes=${RESUME_BYTES}")
  printf '  fail AC-2 — resume exit=%d bytes=%d\n' "${RESUME_EXIT}" "${RESUME_BYTES}" >&2
fi
{
  printf '\n#### AC-2\n\n_Assertion:_ `hero resume` runs clean on A + `hero next ask` surfaces the ask\n\n'
  printf '\`\`\`\n'; head -c 1200 "${RESUME_OUT}"; printf '\n\`\`\`\n'
} >> "${E2E_LOG}"

# --- AC-3: travel-eligibility (gitignore semantics) -----------------
#
# In A's repo: the per-user file is tracked / not ignored, while
# *.local.md and graph.db are ignored. Uses real `git check-ignore`.

A3_OK=1
( cd "${MACHINE_A}" && git check-ignore -q "${HANDOFF_FILE}" ) && A3_OK=0        # ignored → bad
( cd "${MACHINE_A}" && git check-ignore -q ".hero/next/${USER_SLUG}.local.md" ) || A3_OK=0
( cd "${MACHINE_A}" && git check-ignore -q ".hero/graph.db" ) || A3_OK=0

if [[ "${A3_OK}" -eq 1 ]]; then
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("${HANDOFF_FILE} not ignored; *.local.md + graph.db ignored")
  printf '  pass AC-3 — travel eligibility\n' >&2
else
  E2E_AC_IDS+=("AC-3"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("gitignore semantics wrong for handoff/local/graph paths")
  printf '  fail AC-3 — gitignore semantics\n' >&2
fi
{
  printf '\n#### AC-3\n\n_Assertion:_ `%s` not gitignored; `*.local.md` + `graph.db` gitignored (git check-ignore)\n' "${HANDOFF_FILE}"
} >> "${E2E_LOG}"

# --- AC-4: idempotence on B -----------------------------------------
#
# ingest → checkpoint → ingest on B does not duplicate the reflection
# or drop the suggestion. We compare the projected file modulo the
# `updated:` timestamp line and count the suggestion occurrences.

B_FILE="${MACHINE_B}/${HANDOFF_FILE}"
( cd "${MACHINE_B}" && "${HERO_BIN}" next checkpoint -q ) >> "${B_LOG}" 2>&1
FIRST_NORM="$(grep -v '^updated:' "${B_FILE}" 2>/dev/null)"
( cd "${MACHINE_B}" && "${HERO_BIN}" next ingest ) >> "${B_LOG}" 2>&1
( cd "${MACHINE_B}" && "${HERO_BIN}" next checkpoint -q ) >> "${B_LOG}" 2>&1
SECOND_NORM="$(grep -v '^updated:' "${B_FILE}" 2>/dev/null)"
SUGGEST_COUNT="$(grep -cF "${SUGGEST_TEXT}" "${B_FILE}" 2>/dev/null || echo 0)"

A4_OK=1
[[ "${FIRST_NORM}" == "${SECOND_NORM}" ]] || A4_OK=0
[[ "${SUGGEST_COUNT}" -eq 1 ]] || A4_OK=0

if [[ "${A4_OK}" -eq 1 ]]; then
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("pass"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("B round-trip byte-stable; suggestion appears exactly once")
  printf '  pass AC-4 — idempotence on B\n' >&2
else
  E2E_AC_IDS+=("AC-4"); E2E_STATUSES+=("fail"); E2E_DURATIONS+=("0")
  E2E_DETAILS+=("byte_stable=$([[ "${FIRST_NORM}" == "${SECOND_NORM}" ]] && echo 1 || echo 0) suggest_count=${SUGGEST_COUNT}")
  printf '  fail AC-4 — byte_stable=%s suggest_count=%s\n' \
    "$([[ "${FIRST_NORM}" == "${SECOND_NORM}" ]] && echo 1 || echo 0)" "${SUGGEST_COUNT}" >&2
fi
{
  printf '\n#### AC-4\n\n_Assertion:_ ingest→checkpoint→ingest→checkpoint on B is byte-stable (mod `updated:`) + suggestion count == 1\n'
} >> "${E2E_LOG}"

# --- finalize -------------------------------------------------------

e2e_finish
exit $?
