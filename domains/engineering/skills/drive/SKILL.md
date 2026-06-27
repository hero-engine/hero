---
name: drive
description: Arm and supervise an autonomous /drive run over an initiative — ensure the Goal opener, confirm on first arm, emit the condition for the harness /goal, relay needs_me pauses, and resume. Skill, not agent — the autonomy boundary stays deterministic in needs_me.
metadata:
  audience: main-loop
  purpose: autonomous-initiative-execution
---

# drive — autonomous initiative execution

`/drive <initiative>` runs a whole initiative on autopilot. You are thin
orchestration + UX; the judgment lives in `hero goal` / the `needs_me`
predicate, and the loop lives in the harness `/goal`. Do **not** reimplement
the loop or the completion check, and do **not** delegate to a sub-agent —
the boundary must stay deterministic.

## Arming a run

1. **Resolve** the initiative (slug or fuzzy title). If it resolves to a
   non-initiative spec, decline: "That's a single spec — use `/deliver`."
2. **Ensure a `## Goal` run-opener.** If missing, author the canonical
   default (every child verifies, or a `needs_me` pause is raised).
3. **Confirm on first arm.** Show the run condition (`hero goal <init>
   --emit`), the `autonomy:` mode (supervised / guided / autonomous), and the
   guardrails (irreversible actions always pause; hard cap; dry-run
   available). Require a go-ahead. Offer `hero goal <init> --dry-run 3` so
   the user can preview the next transitions before committing.
4. **Emit + hand off.** Paste the emitted condition into the harness `/goal`,
   and ensure the Stop hook (`scripts/drive/stop-hook.sh` → `hero goal <init>
   --check`) is armed with `$HERO_DRIVE_INITIATIVE=<init>`.

## Per turn (relay only)

The harness runs the turn; the Stop hook calls `hero goal <init> --check`:

- **continue** → deliver `next_spec` using its kickoff, then let the loop run.
- **pause** → stop; surface `pause.reason`. The pause/resume layer writes the
  full question to `NEXT.md`. Wait for the human; on their answer, re-arm and
  the run resumes from the same point (state is on disk).
- **done** → report completion; the initiative auto-completes when its last
  child verifies.

## Guardrails (never relax)

- Irreversible / outward-facing actions pause regardless of mode.
- Never run unbounded — the hard cap and initiative-boundary pauses fire
  regardless of mode.
- When unsure, pause. `needs_me` is conservative by construction.
