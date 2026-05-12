# Project Structure

Hero has three important surfaces:

| Surface | Purpose |
|---|---|
| Go source | The `hero` binary, daemon, MCP server, graph, and CLI. |
| Domain content | Agent, command, and skill definitions installed into harnesses. |
| `.hero/` workspace | Per-project corpus: specs, knowledge, graph state, handoff state, events. |

---

## Repository Layout

```text
hero/
├── cmd/hero/                 # CLI entrypoint
├── internal/                 # Go implementation packages
│   ├── cli/                  # Cobra command wiring
│   ├── serve/                # HTTP daemon + MCP stdio server
│   ├── graph/                # SQLite graph substrate
│   ├── retrieval/            # BM25/TF-IDF retrieval
│   ├── scan/                 # master ingest and stack detection
│   ├── traversal/            # why/blocked queries
│   ├── spec/                 # spec parsing and lifecycle
│   └── ...
├── agents/                   # currently installed engineering agents
├── commands/                 # currently installed slash commands
├── skills/                   # currently installed skills
├── core/                     # core domain pack content
├── domains/engineering/      # Hero Code domain pack source
├── domains/sales/            # scaffolded Hero Sales domain pack
├── cloud/                    # team server / cloud backend
├── docs/                     # documentation
└── .hero/                    # this repo's Hero workspace
```

The top-level `agents/`, `commands/`, and `skills/` directories are the
installed engineering pack. `core/` and `domains/` hold the domain-pack
source split that newer install/upgrade paths use.

---

## Workspace Layout

```text
.hero/
├── mission.md                 # project charter and first principles
├── NEXT.md                    # shared projected handoff in solo mode
├── QUEUE.md                   # ready-work queue snapshot
├── planning/
│   ├── features/
│   ├── bugs/
│   └── initiatives/
├── specs/                     # completed specs
├── knowledge/
│   ├── context/
│   ├── notes/
│   └── rules/
├── next/                      # per-user and local handoff projections
├── smoke/                     # per-feature smoke metadata
├── events.log                 # cross-session activity feed source
├── graph.db                   # generated graph store
├── index.db                   # generated search index
└── hero.json                  # project config
```

---

## What Goes in Git

| Path | Git? | Notes |
|---|---|---|
| `.hero/mission.md` | Yes | Highest-priority project charter. |
| `.hero/planning/` | Yes | Active specs and initiatives. |
| `.hero/specs/` | Yes | Completed specs as project history. |
| `.hero/knowledge/` | Yes | Conventions, notes, rules, context. |
| `.hero/QUEUE.md` | Yes | Pre-rendered ready queue for cold-start harnesses. |
| `.hero/events.log` | Yes | Meaningful cross-session events. |
| `.hero/index.db` | No | Generated search index. |
| `.hero/graph.db` | No | Generated graph store. |
| `.hero/hero.local.json` | No | Local overlay and secrets. |
| `.hero/next/*.local.md` | No | Per-machine handoff projection. |

`hero init` manages the usual ignore entries.

---

## Spec Locations

| Type | Active location | Completed location |
|---|---|---|
| Feature | `.hero/planning/features/<slug>/spec.md` | `.hero/specs/<slug>/spec.md` |
| Bug | `.hero/planning/bugs/<slug>/spec.md` | `.hero/specs/<slug>/spec.md` |
| Initiative | `.hero/planning/initiatives/<slug>/spec.md` | `.hero/specs/<slug>/spec.md` |
| Note/context/rule | `.hero/knowledge/<type>/<slug>/spec.md` | Stays in knowledge. |

Current lifecycle statuses are `planning`, `in-review`, `delivering`,
and `completed`, with knowledge-specific statuses such as `active`,
`accepted`, or `draft` where appropriate.

CLI helpers:

```bash
hero spec new csv-export
hero spec claim csv-export
hero spec verify csv-export
hero spec complete .hero/planning/features/csv-export/spec.md
hero list --ready
hero queue --format kickoff
```
