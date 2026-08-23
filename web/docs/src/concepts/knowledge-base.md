# Project memory

Hero's project memory preserves the context later AI sessions need: intent,
decisions, corrections, conventions, evidence, failures, and current state.
The durable corpus belongs to the project, not to one model conversation.

## What is durable

```text
.hero/
├── planning/          # active specs and initiatives
├── specs/             # completed work and delivery evidence
├── knowledge/
│   ├── conventions/   # recurring implementation patterns
│   ├── decisions/     # choices and rejected alternatives
│   ├── rules/         # hard constraints and tripwires
│   ├── context/       # architecture and domain background
│   ├── notes/         # corrections, observations, and brainstorms
│   └── explainers/    # synthesized current behavior
├── NEXT.md            # projected session handoff
├── QUEUE.md           # ready-work projection
└── SNAPSHOT.md        # project-shape projection
```

The markdown corpus and committed projections are inspectable and reviewable.
`graph.db` and `index.db` are derived local state. User-global Focus prompts,
credentials, sessions, and local overlays are not committed project memory.

## Capture

| Need | Workflow |
|---|---|
| Preserve a correction or observation | `/note` or `hero note <slug>` |
| Record why a choice won | `/decide` |
| Codify an established pattern | `/convention` |
| Save the current pickup point | `/handoff` or `hero next checkpoint` |
| Preserve delivery evidence | `/deliver` followed by `hero spec verify <slug>` |

Auto-capture can propose or persist novel learning at major workflow boundaries
when `knowledge.auto_capture` is enabled. It does not justify inventing facts;
captured knowledge should cite the code, decision, or evidence that supports it.

## Structure and retrieval

Hero builds graph, full-text, and semantic indexes over the corpus. Use:

```bash
hero search "authentication retry"
hero ask "why do we use an outbox?"
hero relevant --files internal/auth/session.go
hero why auth-retry
```

Retrieval should be bounded to the work. `hero relevant` is useful before
delivery because it returns governing conventions, decisions, past work, and
known risks without pasting the whole knowledge base into the prompt.

## Corrections and decisions

When project truth changes, preserve the replacement and its reason. Use a new
decision, an explicit correction note, or `hero supersede <old> --by <new>` for
spec genealogy. This keeps old context searchable without letting retrieval
treat it as current authority.

## Cross-session and cross-tool use

Supported harnesses receive native installed instructions that load Hero
context. The CLI remains available when a harness does not expose the same
interactive surface. Committed corpus and handoff projections can cross machine
or tool boundaries; private and machine-local state does not.

The memory system itself is **shipped** and requires a Hero workspace plus a
supported harness or CLI. The repeatable end-to-end cross-tool loop remains
**preview** pending its public proof.

Continue with [Continuity across sessions and tools](continuity.md), then see
[Verified delivery](core-loop.md) for the execution path that consumes and
enriches project memory.
