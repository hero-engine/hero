# Spec Management

Spec lifecycle commands live under `hero spec`.

## Create

```bash
hero spec new csv-export
hero spec new login-crash --type bug
hero design csv-export
```

`hero design` is a top-level alias for `hero spec new`.

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

## Deliver, Verify, Complete

```bash
hero spec deliver csv-export
hero deliver csv-export
hero spec verify csv-export
hero spec score csv-export
hero diff .hero/planning/features/csv-export/spec.md
hero drift csv-export
hero drift --in-flight
hero spec complete .hero/planning/features/csv-export/spec.md
```

`hero deliver` is a top-level alias for `hero spec deliver`.

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

## Listing and Queue

```bash
hero list --ready --sort priority
hero list --status delivering
hero list --horizon now,next
hero queue --format kickoff
```

Use `hero queue` when you want the curated ready-now list with
paste-ready kickoff prompts.
