# Search and Context

Hero exposes project memory through search, Q&A, file-aware relevance,
activity digests, and graph traversal.

## Index and Search

```bash
hero index
hero search "payment retry"
hero search --type bug "timeout"
hero search --file src/payments.go
hero search --list --type feature
```

`hero search` searches the unified corpus: specs, knowledge entries, and
code intelligence populated by `hero scan`.

## Ask

```bash
hero ask "How does the retry logic work?"
hero ask "What conventions exist for error handling?"
```

`hero ask` is extractive Q&A over the corpus using BM25/TF-IDF ranking.
It does not call an LLM.

## Relevant

```bash
hero relevant src/auth.go src/session.go
hero relevant --files src/auth.go,src/session.go
```

`hero relevant` is the current CLI command for file-aware context. It
surfaces conventions, past work, decisions, in-flight specs, and known
risks for the files you are touching.

The MCP tool is named `hero_nudge`; it returns the same style of
lightweight guidance to agents.

## Resume and Recap

```bash
hero resume
hero recap
hero recap --since 2d
hero next
hero next team
```

`hero resume` is the session warm-up command. `hero recap` groups recent
activity by spec. `hero next` renders the current handoff projection.

When `next.projected` is enabled in `hero.json`, NEXT.md is regenerated
from graph events rather than hand-written. Per-machine state lands in
`.hero/next/<user>.local.md` (gitignored); the shared handoff lives in
`.hero/NEXT.md` (solo mode) or `.hero/next/<user>.md` (team mode). A
SessionStart hook fires `hero next ingest` so context from other
machines flows into the local projection on every fresh session. The
pre-flight migration gate on `hero next checkpoint` keeps legacy
hand-written NEXT files from being silently overwritten — run
`hero next migrate-to-projection` once to opt in.

## Project Snapshot

```bash
hero snapshot                           # markdown rollup to stdout
hero snapshot --json                    # structured JSON
hero snapshot --section surfaces        # surfaces | initiatives | recent | next | risks | all
hero snapshot --project                 # rewrite .hero/SNAPSHOT.md + NEXT/AGENTS pointers
hero snapshot detect                    # show inferred surfaces with rationale
hero snapshot assign                    # walk unassigned specs and prompt for a surface
hero snapshot archive                   # write a timestamped archive into .hero/snapshots/
hero snapshot history                   # list archived snapshots newest-first
hero snapshot show <archive>            # render one archived snapshot
hero snapshot diff <a> <b>              # text diff between two archives (or vs `live`)
```

`hero snapshot` renders the project-shape rollup — surfaces, lifecycle
stages, initiatives, recent activity, next moves, and risks — from the
live graph. Surfaces are inferred from repo shape; an optional
`.hero/surfaces.yaml` overrides detection. The snapshot is discoverable
through a one-line pointer that lives in NEXT.md and AGENTS.md; it is
never auto-injected into a session. Archives are excluded from default
search and cold-start bundles.

## Graph Traversal

```bash
hero why csv-export
hero why csv-export:AC-2
hero blocked
hero graph stats
hero graph csv-export --format mermaid
```

`hero why` traces origin chains through the graph. `hero blocked` joins
feature dependencies with failing or regressed acceptance criteria.

## Status and Health

```bash
hero status
hero status --all
hero dashboard
hero check
hero docs check
```

`hero status` shows actionable work by horizon. Use `--all` to include
`someday` and `parking` work.
