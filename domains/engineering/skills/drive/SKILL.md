---
name: drive
description: Arm and supervise an autonomous /drive run over an initiative — ensure the Goal opener, confirm on first arm, emit the run condition for your harness's loop/continuation mechanism, relay needs_me pauses, and resume. Skill, not agent — the autonomy boundary stays deterministic in needs_me.
metadata:
  audience: main-loop
  purpose: autonomous-initiative-execution
---

# drive — autonomous initiative execution

`/drive <initiative>` runs a whole initiative on autopilot. You are thin
orchestration + UX; the judgment lives in `hero goal` / the `needs_me`
predicate, and the loop lives in your harness's loop/continuation
mechanism, where one exists. Do **not** reimplement the loop or the
completion check, and do **not** delegate to a sub-agent — the
boundary must stay deterministic.

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
4. **Emit + hand off.** No shipped hook wires `hero goal <init> --check`
   into a per-turn loop yet — on every harness, the supervisor runs the
   check manually between turns: call `hero goal <init> --check` with
   `$HERO_DRIVE_INITIATIVE=<init>` set, and act on the verdict below.
   Where your harness offers its own loop/continuation mechanism, paste
   the emitted run condition into it as an additional guard.

## Per turn (relay only)

The supervisor calls `hero goal <init> --check` each turn (per the arming
step above) and relays the verdict:

- **continue** → act on the verdict's **`action`** (progressive design):
  - `action: deliver` → run `/deliver <next_spec>` using its kickoff.
  - `action: design` → run `/design <next_spec>` first — the child isn't
    designed yet (a stub, or declared-but-unscaffolded). Designing as you go is
    normal; **do not** hand an undesigned child to delivery. After it's
    designed, the next `--check` returns `deliver` for it.
  Then let the loop run.
- **pause** → stop; surface `pause.reason`. The pause/resume layer writes the
  full question to `NEXT.md`. Wait for the human; on their answer, re-arm and
  the run resumes from the same point (state is on disk). A genuine design
  **fork** surfaced while designing is a `DesignFork` pause — routine design is
  not.
- **done** → report completion; reached only when **every intended child**
  (including ones the initiative declared but hadn't scaffolded) is designed
  and verified. The initiative auto-completes when its last child verifies.

## Guardrails (never relax)

- Irreversible / outward-facing actions pause regardless of mode.
- Never run unbounded — the hard cap and initiative-boundary pauses fire
  regardless of mode.
- When unsure, pause. `needs_me` is conservative by construction.
