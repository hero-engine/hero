---
name: "source-command-handoff"
description: "Force-refresh NEXT.md before switching tools or hitting context limits."
---

# source-command-handoff

Use this skill when the user asks to run the migrated source command `handoff`.

## Command Template

The host-tool Stop hook normally keeps the machine half of NEXT.md fresh
on every turn. Use this command when you want to force a full refresh —
typically before switching tools, when context feels close to full, or
when stepping away.

**Steps:**

1. Run `hero next checkpoint` to refresh the machine half right now
   (branch, recent commits, dirty files, hot files).
2. Then rewrite the agent half of NEXT.md if anything in it is stale:
   - **Last user ask** — quote the user's most recent meaningful turn
   - **Just finished** — what shipped (only the interesting parts; the
     machine half lists commits)
   - **Next** — the concrete next step + a runnable `→` pointer
   - **Blocked on** — actual errors, if any
   - **Tried and failed** — rejected approaches, if any
   - **Context to carry forward** — non-obvious things the next session
     needs that aren't in `git log` or the spec corpus

3. Confirm to the user that NEXT.md is current and ready for handoff.

See `skills/next-md.md` for the full format.

**What to say:** "handoff", "save checkpoint", "save session", "save state".
