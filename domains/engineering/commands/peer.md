---
description: Cross-repo peering — ask a sibling Hero a question, have it design a spec, or hand off work. Routes to `hero peer call` / `hero handoff` / `hero peer list` / `hero peer show` with trail discipline.
---
Front door for cross-repo peering: picks the right interaction mode from
the user's intent and runs the CLI command with trail-discipline flags
filled in.

Before doing anything, **load `skills/cross-repo-peering/SKILL.md`** —
the decision tree, prompt-composition rules, and anti-patterns live
there. Don't compose a peer call without it.

**Steps:**

1. Run `hero peer list` first to confirm peers are registered. If empty,
   tell the user to register first with
   `hero admin repos add <alias> <path>` and stop. Don't try to invoke
   a peer call against an unregistered alias.

2. Parse the user's intent into one of five sub-actions:

   | Intent signal | Sub-action |
   |---|---|
   | "ask <peer>", "check with <peer>", "what's <peer>'s X" | advisory peer call |
   | "have <peer> design", "let <peer> handle the design of" | spec-out peer call |
   | "hand off to <peer>", "drop on <peer>'s queue", "transfer to <peer>" | async handoff |
   | "pick up", "accept", "peer finished" | handoff accept |
   | "list peers", "what peers", "show <peer>", "what does <peer> expose" | discovery (list / show) |

   If two sub-actions are plausible (common between advisory and spec-out;
   common between handoff and spec-out), ask one focused clarifying
   question. Don't guess.

3. **For advisory peer calls** (`--mode=advisory`):

   - Identify the active spec from session context; attach via
     `--related-spec <slug>` if there is one.
   - Draft `--reason` and compose the prompt as a focused brief per the
     skill's "Composing a good peer-call prompt" section. Show both to
     the user inline before dispatching. Default budgets (20 turns /
     50,000 tokens) are usually fine.

4. **For spec-out peer calls** (`--mode=spec-out`):

   - Same trail discipline (`--related-spec`, `--reason`); shape the
     prompt as a `/design` kickoff per the skill. Default budget is
     larger (50 turns / 150,000 tokens).
   - Confirm with the user before dispatching — spec-out writes a spec on
     the peer's side.

5. **For async handoff**:

   - Pre-condition: need a local spec to hand off — suggest the active
     one, or ask which if there isn't one.
   - `hero handoff <local-slug> <peer-alias> --reason "<why>" --type <type>`
     scaffolds a spec on the peer's side from the local spec's
     frontmatter + body (override receiver title/type with `--title` /
     `--type`).
   - After it returns, surface: the receiver-side slug (may differ on
     collision — Hero appends `-2`, `-3`...), that the trail entry
     wrote, and that the local spec moved to `handed_off` (peer now owns
     the next move).

6. **For handoff accept** (a `handed_back` spec needs picking up): run
   `hero handoff accept <slug>`, then prompt the user for the next status
   (usually verify the symptom, then mark complete).

7. **For discovery** (list / show): `hero peer list` for the table,
   `hero peer show <alias>` for one peer's manifest + in-flight
   handoffs. Read-only; no confirmation needed.

8. **Always surface the result to the user**, not raw command output —
   summarize (don't dump) for advisory, confirm the peer-side slug and
   status move for spec-out/handoff, render plainly for list/show.

9. **Trail discipline check** before reporting "done": `hero handoff
   status` shows the new entry, and the active spec's frontmatter
   (`status:`, trail block) is updated.

**What NOT to do:** see the skill's Anti-patterns section (don't call a
peer for a question you can answer locally, don't paraphrase the user
verbatim, don't skip `--related-spec`). One addition: `--mode=full`
(full-delivery peer call) is v2+, not available — offer spec-out as the
closest option if asked.

**What to say:** "ask peer", "ask hero-code about", "check with sibling",
"have hero-cloud design", "hand off to", "drop on peer's queue",
"transfer to", "list peers", "what peers do we have", "what does peer
expose", "pick up the handoff".
