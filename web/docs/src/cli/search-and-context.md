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
hero search --hybrid "retry logic for failed logins"
hero search --semantic "error handling conventions"
```

`hero search` searches the unified corpus: specs, knowledge entries, and
code intelligence populated by `hero scan`.

By default, search uses BM25/TF-IDF ranking. Two additional modes leverage
the built-in semantic embedding engine:

| Flag | Behavior |
|---|---|
| *(none)* | BM25 lexical search over the full-text index. |
| `--hybrid` | Fuses BM25 results with vector similarity via Reciprocal Rank Fusion. Best for natural-language queries where exact keyword matches may miss semantically related content. |
| `--semantic` | Vector-only search. Finds content by meaning, not keywords. |

Hybrid search is the recommended mode for exploratory queries. It
surfaces results that BM25 alone would miss (e.g., searching "login
failure backoff" finds a spec titled "Authentication Retry Logic").

## Embeddings

The embedding engine is built into the `hero` binary — no external model
download, no Python, no CGo. It runs in-process in microseconds.

```bash
hero embeddings status       # chunk counts, model info, index size
hero embeddings rebuild      # wipe and rebuild the vector index from scratch
```

`hero scan` automatically refreshes the embedding index alongside the
full-text index. Only chunks whose content changed are re-embedded
(content-hash invalidation). A refresh on an unchanged project completes
in under 100ms.

The embedding index covers five corpora: specs, knowledge, conventions,
graph events, and code symbols. Configure which corpora to embed via
`hero.json`:

```json
{
  "embeddings": {
    "enabled": true,
    "scope": ["spec", "knowledge", "convention", "event", "code"]
  }
}
```

## Ask

```bash
hero ask "How does the retry logic work?"
hero ask "What conventions exist for error handling?"
```

`hero ask` is extractive Q&A over the corpus. It uses BM25/TF-IDF
ranking and does not call an LLM.

## Capture: Notes and Intake

Two low-friction ways to get a thought into the graph before it's a spec:

```bash
hero note "thinking about the auth flow"     # quick knowledge-base note
hero note auth-ideas --from conversation.md   # import note content from a file
cat convo.txt | hero note piped-thoughts      # or pipe via stdin

hero intake "let users export to CSV"         # capture a pre-commitment idea
hero intake list                              # list intakes by status
hero intake promote csv-export                # promote an intake to a roadmap spec
hero intake reject stale-idea                 # terminal: reject
```

`hero note` captures brainstorms, conversation dumps, and
stream-of-consciousness thinking — anything not ready to be a spec.

`hero intake` is different: an **intake** is a pre-commitment idea or
inbound signal that lives in the spec graph (searchable,
provenance-linked) but is deliberately held **out** of committed-work
rollups — status, queue, velocity, snapshot — until you `promote` it to a
real spec. It's the inbox stage before anything becomes planned work. See
[Specs & Lifecycle](../concepts/specs.md#intake-the-pre-spec-stage).

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

## Impact Analysis

```bash
hero impact src/payments.go
hero impact src/payments.go src/session.go     # multiple files
hero impact src/payments.go --format           # JSON output
```

`hero impact` reports the blast radius of changing a file: which specs,
conventions, and decisions are affected. Use it before a refactor to see
what documentation and in-flight work touches the code you're about to
move. Cross-repository work uses explicit one-graph-per-project peering; a local
impact query does not silently merge sibling graphs.

## Coverage Suggestions

```bash
hero suggest                # top churn areas with no spec coverage
hero suggest --since 90d    # widen the churn window (default 30d)
hero suggest --top 20       # show more suggestions
```

`hero suggest` analyzes git churn to find files with heavy recent
activity but no spec coverage — the places where work is happening
undocumented — ranked by churn intensity. It's a fast way to spot where a
`/design` or `/scan` pass would pay off.

## Synthesis (Explainers)

```bash
hero synthesize cold-start-trust-hardening      # synthesize one cluster
hero synthesize feat-a feat-b feat-c            # synthesize across several specs
hero synthesize --detect                        # list explainer-worthy clusters
hero synthesize --auto                          # synthesize auto-eligible clusters
hero synthesize --stale                         # explainers whose cluster grew since last run
hero synthesize --set-mode review               # autonomy: auto | review | off
```

`hero synthesize` reads a cluster of related specs plus the git activity
across their delivery window and produces an **explainer** — a "how this
feature works, as it exists now" knowledge entry — at
`.hero/knowledge/explainers/<slug>/spec.md`. The CLI assembles the inputs
deterministically; prose is written by an LLM when `ANTHROPIC_API_KEY` is
set, otherwise a scaffold is emitted for an agent (via the
`hero_synthesize` MCP tool) or a human to complete. See
[Knowledge & Standards](../workflows/knowledge-and-standards.md) for where
explainers fit in the knowledge workflow.

## Activity Feed

```bash
hero feed                        # last 20 significant events, newest first
hero feed --since 1h             # events from the last hour
hero feed --type decision_made   # filter by event type
hero feed --slug csv-export      # filter by spec
```

`hero feed` is the cross-session activity feed — significant events
logged by every agent working in the repo. It's how you see what other
sessions (and other machines) have been doing.

## Reasoning Sessions

A **session** is a recorded reasoning log — the trail of events an agent
emitted while working. Sessions can be distilled into knowledge or
replayed to reconstruct how a decision was reached.

```bash
hero session list                # all sessions
hero session start               # begin a new reasoning session
hero session log <id>            # show events from a session
hero session replay <id>         # render a full session summary
hero session distill <id>        # suggest knowledge entries from a session
hero session prune --days 30     # prune sessions older than N days
```

## Extract Decisions & Concepts

```bash
hero extract              # extract from notes (default target)
hero extract specs        # extract from planning specs
hero extract all          # both
hero extract --dry-run    # preview without calling the LLM
```

`hero extract` reads hand-authored prose from notes and specs and pulls
structured **Decision** and **Concept** nodes into the knowledge graph, so
accumulated reasoning surfaces in later sessions. It's provider-agnostic
(defaults to Anthropic via `ANTHROPIC_API_KEY`) and idempotent —
unchanged sources are skipped by content hash, no LLM call.

## Guardrails: anchor & tripwire

Tripwires are forbidden-option guardrails — decisions the project has
ruled out — and `anchor` re-grounds a session on the project mission plus
any tripwires relevant to what you're deciding.

```bash
hero anchor                              # mission + all active tripwires
hero anchor "should we add a message queue?"   # highlight tripwires matching this context
hero tripwire list                       # list active tripwires
hero tripwire check "let's cache in Redis"     # does this text trip a guardrail?
```

## Stack & Code Scan

```bash
hero scan              # detect stack + generate knowledge stubs + code intelligence
hero scan --code       # code intelligence only (symbols, packages, deps)
hero scan --dry-run    # preview without writing
hero scan --force      # overwrite existing entries
```

`hero scan` detects the technology stack and seeds the knowledge base and
code-intelligence corpus that `hero search` and `hero relevant` query.
Generated knowledge entries are stubs — review and enrich them, or use the
`/scan` workflow to let an agent fill them in. Code-scan depth is set by
`code_scan.depth` in `hero.json` (`normal` / `deep` / `disabled`).

## Status and Health

```bash
hero status
hero status --all
hero dashboard
hero check
hero docs check
```

`hero status` shows actionable work by horizon. Use `--all` to include
`someday` and `parking` work. The default human view is a compact operational
briefing: all work in progress, up to ten priority-ranked upcoming items, up
to ten items waiting on peers, and the five newest completions that carry an
authoritative `completed_at` timestamp.

```text
Work: 3 in progress · 2 upcoming (1 ready, 1 blocked) · 1 waiting · 6 completed
Other: 4 intake · 17 knowledge · 6 hidden by horizon

In progress (3):
  returned-export                 feature     handed_back    Return Export
  fix-login                       bug         delivering     Fix Login
  review-billing                  feature     in-review      Review Billing

Upcoming (2):
  csv-export                      feature     planning       CSV Export
  retry-policy                    feature     planning       Retry Policy  [blocked]

Waiting (1):
  mobile-client                   feature     awaiting_peer  Mobile Client

Recently completed (6):
  audit-log                       feature     3h ago          Audit Log
  search-refresh                  bug         5h ago          Search Refresh
  team-signup                     feature     1 day ago       Team Signup
  retry-fix                       bug         2 days ago      Retry Fix
  docs-refresh                    feature     3 days ago      Docs Refresh
  … 1 more — `hero list --status completed --sort recency`
```

Completed, intake, and knowledge totals are workspace-wide and are not dumped
entry by entry. Use `hero list` for the full corpus:

```bash
hero list --status planning --sort priority
hero list --status handed_off,awaiting_peer --sort priority
hero list --status completed --sort recency
hero list --type intake
hero list --type convention,decision,rule,external,context,note
```

Use `hero status --json` for automation; its unbounded schema is independent
of the human view's row limits.
