# Continuity across sessions and tools

Continuity is the operational path through Hero's project memory. It preserves
what a later session needs, then retrieves a bounded slice at the next start.

## Capture durable project truth

| Information | Capture path | Stored as |
|---|---|---|
| Current intent and next action | `/handoff`, `hero next checkpoint` | Projected NEXT state |
| A correction or observation | `/note`, `hero note <slug>` | Knowledge note |
| A durable technical choice | `/decide` | Decision entry |
| A recurring implementation pattern | `/convention` | Convention entry |
| Delivery evidence | `/deliver` and `hero spec verify <slug>` | Completed spec, ledger, audit, AC evidence |
| Why work exists | Spec frontmatter relations and recorded events | Traversable graph provenance |

Captured artifacts are files or graph-backed projections owned by the project.
Do not put credentials or private user Focus prompts in the repository corpus.

## Structure and retrieve

- `hero search <query>` finds literal or indexed matches.
- `hero ask "<question>"` answers from the local corpus.
- `hero relevant --files <paths...>` retrieves conventions, decisions, past
  work, and known risks related to specific files.
- `hero why <target>` traverses the provenance chain.
- `/resume` loads the ranked session context installed for the active harness.

Retrieval does not mean loading the whole corpus. The goal is the smallest
relevant context block that retains the governing decision and evidence.

## Cross-session and cross-tool boundary

Committed corpus and projected handoff files can travel with the repository.
Per-machine state and private Focus data do not. Supported harnesses consume
Hero through native installed surfaces, so the exact invocation differs by
tool even though the project artifacts remain the same.

The memory-and-delivery loop is the product model: later sessions retrieve the
project-owned decisions, evidence, corrections, and handoff state produced by
earlier work, regardless of which supported harness renders the workflow.

## Corrections and stale knowledge

Record a correction as a new authoritative note, decision, or superseding spec;
do not silently rewrite history when the old rationale matters. Use
`hero supersede <old> --by <new>` for spec genealogy, and run `hero index` after
manual corpus edits so retrieval sees the latest source.

Next: [Verified delivery](core-loop.md) explains how delivery consumes this
memory and produces new evidence for later sessions.
