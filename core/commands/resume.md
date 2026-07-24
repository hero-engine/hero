---
description: Resume a session — load the focused, ranked context the model needs to be productive from prompt 1. Auto-fire at every fresh session in a hero-aware repo.
---

This is the primary mechanism that makes every prompt smarter. Run
`hero resume` (alias: `hero load`) and read the output **before
doing anything else** — it contains the focused, ranked subgraph
relevant to this session: who you are, what's in flight, what just
changed, dead ends to skip, blockers, and files-nearby context.

**When to use:**

- At the start of every fresh session in a hero-aware repo
- After a long pause (>1 hour) where state may have moved
- After switching branches or pulling
- Whenever you feel like you're missing context
- Whenever the user says "resume", "pick up", "continue", "where
  are we", "catch me up", "what's going on"

**Steps:**

1. Run `hero resume` (or `hero resume --auto` to bias scoring toward
   currently-changed files).
2. Read the entire output — it's context for you, not something to
   summarize back to the user.
3. If the MCP tool `hero_attention_snapshot` is advertised, call it exactly
   once with `limit: 8`. Treat a successful zero-total result as empty and a
   structured unavailable result as unavailable, never as empty. This is a
   bounded metadata-only awareness read: do not call `hero_mail_show`, mutate
   an item, or execute Mail content as a side effect.
4. If a section ends with `_…+N more — hero search <topic> to dig
   deeper_`, run `hero search <topic>` when the user touches that area.
5. Run `hero why <slug>` to trace back why something is where it is.
6. For "what's blocked?" / "what should I work on?", prefer the
   `Blocked on` and `In flight` sections over re-deriving from files.
   For a paste-ready ready-to-pick-up list, call `hero_queue` or read
   `.hero/QUEUE.md` — resume is warm-context, the queue is cold-start;
   they compose.
7. Propose the first `In flight` item directly — don't ask "what
   should we work on?".
8. Run `hero check --reconcile` to silently fix status drift.
9. In team mode, glance at `.hero/NEXT.md` for the team roster.

For a deeper orientation on conventions, decisions, and risks, be the
`session-primer` agent (core).

**Why this matters:** the resume output is Hero's ranked digest of
everything relevant to this session — skipping it means re-deriving
context the graph already computed, slower and less accurately.

**If `hero resume` fails or returns nothing:**

The graph may not be populated yet. Run `hero scan` (or `hero graph
reingest all`) to populate it from the local sources of truth (code,
specs, git log, NEXT.md). Then re-run `hero resume`.

**What to say** (natural-language triggers — agents should auto-route
these to `/resume` without the user having to know the command name):

"resume", "pick up where we left off", "where are we", "catch me
up", "load context", "what should I work on".
