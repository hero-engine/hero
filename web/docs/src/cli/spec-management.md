# Spec Management

Spec lifecycle commands live under `hero spec`.

## Create

```bash
hero spec new csv-export
hero spec new login-crash --type bug
hero design csv-export
hero diagnose login-crash
```

`hero design` is a top-level alias for `hero spec new`, and `hero
diagnose` is a top-level alias for `hero spec diagnose` — the
bug-investigation front door that classifies root cause and produces a
fix spec.

## Claim and Coordinate

```bash
hero spec claim csv-export --agent codex
hero spec claim csv-export --release
hero spec claim csv-export --complete
hero spec claims
hero graph csv-export
hero graph csv-export --format mermaid
```

Claims are advisory and visible in status/search output. They are not a
distributed lock.

## Deliver and verify

```bash
hero spec deliver csv-export
hero deliver csv-export
hero spec verify csv-export
hero spec score csv-export
hero diff .hero/planning/features/csv-export/spec.md
hero drift csv-export
hero drift --in-flight
```

`hero deliver` is a top-level alias for `hero spec deliver`.

`hero spec verify <slug>` is the normal evidence-backed close. It checks the
Completion Ledger, cold delivery audit, acceptance-criterion test mapping, and
build/tests. When the hard gates pass, it marks the spec complete and archives
it. Do not follow it with `hero spec complete` and do not hand-edit completed
status.

## Acceptance Criteria and Contracts

```bash
hero ac list csv-export
hero ac status --feature csv-export
hero ac history csv-export:AC-2
hero coverage csv-export
hero spec contract status csv-export
hero spec contract link csv-export 1 e2e/export.spec.ts::streams_large_exports
hero spec contract check csv-export
hero spec lint csv-export
```

Acceptance criteria are graph nodes. Contract links connect criteria to
tests and make regressions visible.

## Plans, Mocks, Demos

```bash
hero spec plan csv-export
hero spec mock list
hero spec mock serve csv-export
hero spec demo record csv-export
hero spec demo list
hero spec demo show csv-export
hero spec demo clean csv-export
```

Plans are normally written by agents through the `hero_plan` MCP tool or
by delivery dry-runs.

## Sizing

```bash
hero size csv-export                 # print the declared size ("(unset)" if absent)
hero size csv-export large           # set the declared size
hero size --ack giant csv-export     # acknowledge an oversized spec (suppresses the nudge)
hero size --check                    # scan all specs for declared-vs-computed drift (CI-friendly)
```

The size ladder — `trivial / small / medium / large / x-large / giant` —
is shared across feature, bug, enhancement, epic, and initiative specs.
`hero size --check` exits non-zero when any leaf spec's declared size
drifts from its computed size, so it slots into CI; add `--summary` for a
single workspace-wide line.

## Superseding

```bash
hero supersede old-auth --by new-auth --reason "replaced by OAuth design"
hero supersede --list                # show current supersede chains
hero supersede --scan                # detect candidate pairs; writes a report, mutates nothing
hero supersede --unset old-auth      # clear superseded_by (only if set in error)
```

`hero supersede` wires the soft-archive genealogy between two specs: it
stamps `superseded_by:` on the old spec and the inverse `supersedes:` on
the new one, then reindexes so retrieval de-weights the superseded spec.
It refuses to create cycles or chain into an already-superseded target.

## Graph Writes

Beyond the read-only `hero graph <slug>` traversal, the graph accepts
direct node and edge writes — used by agents and automation to record
relationships the frontmatter parser wouldn't otherwise capture:

```bash
hero graph node add ...      # upsert a node into the graph
hero graph edge add ...      # write an edge between two existing nodes
hero graph reingest <slug>   # re-populate a subgraph from its source of truth
hero graph stats             # node and edge counts
```

Prefer declaring relationships in spec frontmatter (`parent:`,
`depends-on:`, `relations:`) where possible; the write CLI is for cases
that live outside a spec's own frontmatter.

## Listing and Queue

```bash
hero list --ready --sort priority
hero list --status delivering
hero list --horizon now,next
hero queue --format kickoff
```

Use `hero queue` when you want the curated ready-now list with
paste-ready kickoff prompts.

## Tasks

Tasks are the "next thing to do" sub-element of a spec — an additive peer
of acceptance criteria. Where acceptance criteria flip on test evidence,
tasks flip on human action, and the two live side-by-side.

```bash
hero task add csv-export "wire the export button"   # add to the spec's ## Tasks
hero task list csv-export                            # list a spec's tasks
hero task start csv-export T-1                        # flip to doing (stamps started)
hero task done csv-export T-1                         # flip to done (stamps done)
hero task history csv-export:T-1                      # timeline of status flips
hero task status --feature csv-export                # completion-rate rollup
```

## Pipeline

`hero pipeline` shows work specs grouped by their stage in the
import → diagnose → deliver flow — a birds-eye view of the batch:

```bash
hero pipeline                     # all specs grouped by stage
hero pipeline --type bug          # bugs only
hero pipeline --run diagnose      # async-diagnose all imported bugs
hero pipeline --run deliver       # async-deliver all approved specs
hero pipeline --run all           # diagnose then deliver
```

Stages: **Imported** (not yet investigated), **Diagnosed**, **Approved**
(ready for delivery), **Delivering**, **Completed**, and **Blocked**
(missing info or quality issues). The `--run` modes drive the batch
through the async agent runtime.

## Learned Templates

`hero templates` analyzes your completed specs to discover the section
structures, criteria density, and frontmatter patterns you actually use
per spec type — so new specs match your house style rather than a generic
scaffold.

```bash
hero templates              # summary table of discovered patterns
hero templates show bug     # full detail for one spec type
hero templates refresh      # re-analyze the corpus and write pattern files
```
