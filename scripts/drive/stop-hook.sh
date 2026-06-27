#!/usr/bin/env bash
# Reference Claude Code Stop hook for `/drive` — the seam between the harness
# /goal loop and Hero's authoritative judgment. While a Drive run is armed
# for an initiative, this hook runs after every turn and calls
# `hero goal <init> --check`, then tells the harness whether to keep going.
#
# Contract (see stop-hook-contract.md beside the hero-goal-command spec):
#   hero goal <init> --check  ->  {"verdict":"continue|pause|done", ...}
#     continue -> re-inject next_spec's kickoff, keep the loop running
#     pause    -> stop the loop, surface pause.reason to the human
#     done     -> stop the loop, report completion
#
# Hero does NOT drive the loop or judge completion from the transcript — the
# harness owns that. This hook only relays Hero's verdict. The armed
# initiative is read from $HERO_DRIVE_INITIATIVE (set by the /drive skill).
set -euo pipefail

init="${HERO_DRIVE_INITIATIVE:-}"
if [[ -z "$init" ]]; then
  # No Drive run armed — allow the stop, do nothing.
  exit 0
fi

verdict_json="$(hero goal "$init" --check 2>/dev/null)" || exit 0
verdict="$(printf '%s' "$verdict_json" | sed -n 's/.*"verdict": *"\([a-z]*\)".*/\1/p' | head -1)"

case "$verdict" in
  continue)
    # Block the stop so the harness runs another turn; hand back the next
    # child's kickoff as the reason/continuation prompt.
    reason="$(printf '%s' "$verdict_json" | sed -n 's/.*"kickoff": *"\(.*\)".*/\1/p' | head -1)"
    printf '{"decision":"block","reason":%s}\n' "$(printf '%s' "${reason:-continue the run}" | json_escape 2>/dev/null || printf '"continue the run"')"
    ;;
  done)
    # Allow the stop; the run is complete.
    echo "Drive: initiative '$init' complete — all children verified." >&2
    exit 0
    ;;
  pause|*)
    # Allow the stop and surface the pause question (written to NEXT.md by
    # the pause/resume layer). Allowing the stop hands control back to you.
    echo "Drive: paused — see the question in .hero/NEXT.md (or per-user handoff)." >&2
    exit 0
    ;;
esac
