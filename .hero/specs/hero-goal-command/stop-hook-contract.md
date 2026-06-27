# `/drive` Stop-hook contract

The harness `/goal` loop drives turns; Hero judges them. The seam is a
**Stop hook** that runs after each turn and calls `hero goal <init> --check`.
Reference implementation: [`scripts/drive/stop-hook.sh`](../../../../scripts/drive/stop-hook.sh).

## The call

```
hero goal <initiative> --check   →   JSON verdict (stdout)
```

```json
{
  "verdict": "continue" | "pause" | "done",
  "initiative": "<slug>",
  "next_spec": "<slug>",        // continue/pause: the child in question
  "kickoff": "<text>",          // continue: the child's ## Kickoff (turn prompt)
  "pause": { "category": "...", "reason": "..." },   // pause only
  "remaining": ["..."],
  "completed": ["..."]
}
```

The verdict is computed from **on-disk state only** (child verify-status ANDed
with the `needs_me` boundary), so it is identical across a cold process — safe
to call from a hook and safe across context resets.

## The hook's job (relay, don't decide)

| Verdict | Harness action |
|---|---|
| `continue` | Block the stop, run another turn; re-inject `kickoff` as the turn prompt. |
| `pause` | Allow the stop; surface `pause.reason` to the human (the pause/resume layer writes the full question to `NEXT.md`). |
| `done` | Allow the stop; report completion. |

The armed initiative is passed via `$HERO_DRIVE_INITIATIVE` (set by the
`/drive` skill, spec `drive-command-routing`).

## Boundaries (what this is NOT)

- Hero does **not** run turns or evaluate completion from the transcript —
  the harness `/goal` owns the loop and its evaluator.
- Irreversible/outward-facing actions are governed turn-by-turn by the base
  safety rules and the harness, *inside* a delivery — not by this between-spec
  verdict. `--check` operates at spec-transition granularity.

## Harness-agnostic

`--check` emits plain JSON; the MCP tool `hero_goal` returns the same shape.
Any harness that can run a post-turn hook (or call the MCP tool) and read JSON
can drive Hero. The Claude Code script above is the reference; a Codex `/goal`
adapter is a thin wrapper over the same contract.
