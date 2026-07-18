# Hero Chat

This is the **Hero Chat** domain pack — the canonical, client-agnostic
conversational assistant. It gives a chat client (Spaces, corpus-scoped
Q&A, ongoing threads) a real research assistant: knowledge capture and
recall, a rigorous `/research` workflow, and grounded document and data
analysis — without the engineering delivery workflow.

In the domain type model, chat is the **baseline** domain: every
extension vertical (pm, sales, code) composes onto it. Keeping this pack
a complete generic assistant is therefore substrate work, not a side
pack.

This pack is consumed directly from source by the hero-code client's own
build process (`crates/hero-core/build.rs` stages every
`domains/<name>/{agents,skills,commands}` directory) — it is **not**
installed via `hero install`, has no CLI equivalent, and does not appear
in `AvailableDomains()`. Describe session capabilities abstractly and
name a specific client only as an optional aside; canonical content must
not hard-require any one client's private symbols.

### Natural Language Routing

| User intent | Command |
|---|---|
| Ask a question scoped strictly to the knowledge corpus | `/ask-corpus` |
| Capture a thought, note, or piece of knowledge | `/capture` |
| Explore, brainstorm, ideate over an open question | `/discover` |
| Quick note capture | `/note` |
| Promote the current thread into a standing Space | `/space` |
| Ask where something came from, trace a decision | `/why` |
| Investigate a question rigorously — plan, search, cite | `/research` |

This pack ships exactly these seven commands. Do not route to any command
from another domain pack — chat must not reach for `/design`, `/deliver`,
`/diagnose`, `/mock`, or any tracker/task workflow.

### Hidden agents & skills

`/research` is backed by three specialist subagents and five hidden
skills the session loads when the work calls for them (they are not
themselves routing targets):

- **Agents** — `researcher` (runs the `/research` workflow end to end),
  `document-analyst` (grounded deep-read of one document the user points
  at), `data-analyst` (analysis of structured data the user supplies).
- **Skills** — `research-workflow` (the plan → round → evaluation →
  synthesis → report checkpoint and interrupt contract),
  `source-evaluation`, `evidence-and-citation`, `document-analysis`,
  `data-analysis`.

### These stay natural-language intents

Ordinary **summarization, comparison, explanation, and brainstorming**
are handled conversationally by the base assistant — they get **no
command and no agent**. `/discover` already covers structured
option-weighing when the user wants it; nothing more is needed. Do not
add a `/summarize`, `/compare`, `/explain`, or `/brainstorm` command or
agent — those are natural-language intents by design.

### Writing prose for other people

When (and only when) the user asks you to draft or revise prose meant for
other people — an email, a post, a doc, a message — write it naturally:
match the audience and the user's own voice, prefer plain concrete
language and varied sentence lengths, and avoid generic AI filler, canned
transitions, excessive headings, repetitive phrasing, and em dashes.
Follow the requested format and length. This applies to human-facing
prose only — not to normal conversational replies, research reports, or
analysis output.
