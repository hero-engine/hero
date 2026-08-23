# Project structure

Hero has three important surfaces:

| Surface | Purpose |
|---|---|
| Go source | CLI, local daemon, MCP server, graph, retrieval, and integrations |
| Core and domain content | Agent, workflow, and skill definitions rendered into supported harnesses |
| `.hero/` workspace | Project memory: specs, knowledge, evidence, relationships, and handoff state |

## Repository layout

```text
hero/
├── cmd/hero/                 # CLI entry point
├── internal/                 # implementation packages
├── contracts/                # portable integration and Attention contracts
├── core/                     # shared agents, workflows, and skills
├── domains/engineering/      # default Engineering setup
├── domains/pm/               # optional focused PM setup
├── domains/qa/               # optional focused QA setup
├── domains/sales/            # optional focused Sales setup
├── web/                      # hosted docs and landing source
└── .hero/                    # this repository's Hero workspace
```

Hero Code and Hero Cloud are separate proprietary products and are not
directories in this repository. Sprout is a separate MIT-licensed dependency.

Installed harness files are generated copies, not source directories or
symlinks. Re-running `hero install` regenerates the managed content in the
target's native layout.

## Workspace layout

```text
.hero/
├── mission.md
├── planning/                 # active specs and initiatives
├── specs/                    # completed specs and delivery evidence
├── knowledge/                # decisions, corrections, conventions, context
├── NEXT.md                   # shared projected handoff
├── next/                     # user and local projections
├── QUEUE.md                  # ready-work projection
├── SNAPSHOT.md               # project-shape projection
├── peer-manifest.yaml        # optional peering surface
├── hero.json                 # shared non-secret configuration
├── hero.local.json           # private overlay; never commit
├── graph.db                  # derived local graph
└── index.db                  # derived local search index
```

## Monorepo topology

A monorepo has one root `.hero` corpus. `hero install satellites` creates thin
harness-native trees in selected subproject folders so sessions opened there
can reach the root content. It does not create nested project graphs or nested
Hero workspaces.

## Spec locations and closing path

| Type | Active | Completed |
|---|---|---|
| Feature | `.hero/planning/features/<slug>/spec.md` | `.hero/specs/<slug>/spec.md` |
| Bug | `.hero/planning/bugs/<slug>/spec.md` | `.hero/specs/<slug>/spec.md` |
| Initiative child | `.hero/planning/initiatives/<parent>/<slug>/spec.md` | `.hero/specs/<slug>/spec.md` |

```bash
hero spec new csv-export
hero spec claim csv-export
hero spec verify csv-export
```

`hero spec verify <slug>` checks the Completion Ledger, cold audit,
acceptance-criterion test coverage, and build/tests. When the hard gates pass,
it completes and archives the spec. `hero spec complete` is not the normal
evidence-backed close.

## Commit boundary

Commit the project corpus, including planning, completed specs, knowledge, and
the projected `NEXT.md`, `QUEUE.md`, and `SNAPSHOT.md` files. Do not commit
credentials, `.hero/hero.local.json`, sessions, per-machine `.local.md` files,
or generated graph/index databases. `hero init` manages the usual ignore rules.
