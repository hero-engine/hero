# Hero Chat

This is the **Hero Chat** domain pack. It gives a conversational chat
client (Spaces, corpus-scoped Q&A, ongoing threads) a small set of
Hero-flavored commands for capturing, discovering, and querying
knowledge without the engineering delivery workflow.

This pack is consumed directly from source by the hero-code client's
own build process — it is not installed via `hero install` and has no
CLI equivalent. No current consumer renders this file's routing table
into a running session; it exists so that if chat is ever wired into
an instructions-file path, the routing is already correct rather than
falling back to the engineering pack's commands, none of which chat
ships.

### Natural Language Routing

| User intent | Command |
|---|---|
| Ask a question scoped to the knowledge corpus | `/ask-corpus` |
| Capture a thought, note, or piece of knowledge | `/capture` |
| Explore, brainstorm, ideate | `/discover` |
| Quick note capture | `/note` |
| Promote the current thread into a standing Space | `/space` |
| Ask where something came from, trace a decision | `/why` |

This pack ships exactly these six commands. Do not route to any
command from another domain pack — chat has no agents, no skills, and
no delivery workflow.
