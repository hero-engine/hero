---
title: Contracts Package Import Discipline
type: convention
status: active
created: 2026-05-15
scope:
  - "contracts/**"
tags: [contracts, boundaries, cross-repo, hero-cloud, oss]
---

# Contracts Package Import Discipline

## The rule

> **`contracts/...` is a leaf.** No package under `github.com/hero-engine/hero/contracts/...` may import any other in-repo path. Stdlib and third-party imports are allowed (and should be kept minimal — prefer stdlib only). Other `contracts/...` paths may freely import each other.
>
> **The read direction is unrestricted.** Any package outside `contracts/` may import `contracts/...` shapes freely. Internal code consuming contracts is the whole point.

Enforced by `contracts/contracts_boundary_test.go`. Adding a forbidden import to any file under `contracts/...` fails this test with a message naming the offending package and import.

## Why it exists

The contracts package is the **OSS-eligible boundary** between hero (CLI + local engine) and hero-cloud (server, separate repo at `github.com/hero-engine/hero-cloud`). hero-cloud depends on hero via Go `replace` plus a `hero.ref` pin, and it consumes `contracts/` directly.

If `contracts/` ever imports `internal/` (or any other non-contracts in-repo path):

- hero-cloud either drags the entire `internal/` tree into its build (defeating the boundary) or fails to compile (breaking the seam).
- The day `contracts/` ships under an OSS license while the rest of hero does not, one stray import collapses the licensing story.
- The "vocabulary lives here, implementation lives over there" split — which is what makes the governance model auditable — silently rots.

The boundary is the contract. Keeping `contracts/` a leaf is what keeps the contract a contract.

## What goes in `contracts/` — and what doesn't

**Belongs here:**

- Wire and graph shapes (`Node`, `Event`, `AgentToken`, `AuditEvent`, `PolicyNode`).
- Vocabulary types and enums (`Classification`, `SubjectType`, `Purpose`, `PrincipalKind`).
- Interface signatures that mark a seam (`Retriever`, `Principal`).
- Minimal, pure helpers on those types — total functions, no I/O, no side effects, no logger, no config. The bar is "could this be a method on a primitive." See `Classification.Compare` and `governance.Max` in `contracts/governance/classification.go` for the shape.
- Shape-version constants and bump rules (`ContractsVersion`, `ServerMinContractsVersion`, `PeeringContractsVersion`).

**Does not belong here:**

- Anything that opens a database, file, or socket.
- Anything that constructs an LLM call, hits a network, or reads env vars.
- Anything that takes a `*log.Logger` or emits telemetry.
- Constructors that return real implementations of the interfaces declared here — those live in hero-cloud (server-side) or in hero's `internal/` (CLI-side).
- Anything that depends on `internal/graph`, `internal/spec`, `internal/index`, or any other in-repo package. If you need a type the internal layer already defines, the contracts package gets its **own wire shape** and the internal layer maps to/from it.

## Current layout (canonical example)

```
contracts/
├── contracts_boundary_test.go   # the enforcement test
├── event.go                     # Event envelope + EventKind constants
├── node.go                      # Node, NodeID, Kind (wire-level node)
├── version.go                   # ContractsVersion, ServerMinContractsVersion
├── governance/
│   ├── agent_token.go           # AgentToken claim shape
│   ├── audit.go                 # AuditEvent shape
│   ├── classification.go        # Classification enum + Compare/Max helpers
│   ├── policy.go                # PolicyNode, Rule shapes
│   ├── principal.go             # Principal interface, PrincipalKind
│   ├── purpose.go               # Purpose enum
│   ├── retriever.go             # Retriever interface, Query, NodeDecision
│   └── subject.go               # Subject, SubjectType
└── peering/
    └── version.go               # PeeringContractsVersion (independent bump)
```

A reader landing here should be able to skim every file in 10 minutes. If a file in this tree starts to look like a "service," it's drifted — extract the implementation, leave the shape.

## How to add something to `contracts/`

1. **Confirm it's vocabulary, not implementation.** Type, constant, interface signature, or pure helper on those. If it has behavior beyond shape-level helpers, it doesn't belong here.
2. **Confirm a cross-repo consumer exists or is imminent.** hero-cloud (or another future external consumer) must actually need the shape. Don't speculatively pre-stage types here.
3. **Add the type.** Place it in the smallest sensible subpackage (`contracts/` for top-level wire shapes, `contracts/governance/` for the governance vocabulary, `contracts/peering/` for peer-handshake shapes).
4. **Run `go test ./contracts/...`** — the boundary test must stay green.
5. **Decide whether `ContractsVersion` bumps.** Any breaking change to an exported type, field, or method signature is a bump. Adding a new field with a zero-value-safe default is not a bump. Renaming or removing is.

## How to remove something from `contracts/`

Removal is a breaking change to every external consumer.

1. Bump the relevant `ContractsVersion` (or `PeeringContractsVersion`) and document the removal in its bump rule comment.
2. Coordinate with hero-cloud: the removal lands here, and the next `hero.ref` pin bump in hero-cloud is the adoption point.
3. The old shape may stay deprecated until the next hero-cloud pin lands; don't delete-and-bump in the same change unless hero-cloud is ready to receive it.

## Anti-patterns

### Anti-pattern 1 — putting a real implementation behind a contracts interface

```go
// contracts/governance/retriever.go — WRONG
import "github.com/hero-engine/hero/internal/graph"

func NewRetriever(g *graph.Store) Retriever { /* ... */ }
```

Why it's wrong: `NewRetriever` is enforcement, not vocabulary. The import drags `internal/graph` into every consumer of `contracts/`, including hero-cloud, which has no business knowing about hero's local graph store. The constructor and implementation belong in hero-cloud (server-side) or in hero's own `internal/` (CLI-side). `contracts/` only declares the interface.

### Anti-pattern 2 — reusing an internal type instead of defining a wire shape

```go
// contracts/policy.go — WRONG
import "github.com/hero-engine/hero/internal/graph"

type PolicyNode struct {
    Node graph.Node // reuses the internal struct
    // ...
}
```

Why it's wrong: it couples the wire shape to whatever the internal graph package happens to look like today. The internal layer is free to refactor; the contracts layer is not. The contracts package must define its own minimal `PolicyNode` (see `contracts/governance/policy.go`) and the internal graph layer maps between its node type and the contracts type at the boundary.

### Anti-pattern 3 — adding a logger or env-var read to a "helper"

```go
// contracts/governance/classification.go — WRONG
import (
    "log"
    "os"
)

func (c Classification) Compare(b Classification) int {
    if os.Getenv("HERO_DEBUG_CLASS") != "" {
        log.Printf("comparing %d vs %d", c, b)
    }
    return int(c) - int(b)
}
```

Why it's wrong: `Compare` was a pure shape helper. Adding env-var reads and logging makes it side-effecting and pulls observability concerns across the boundary. Helpers in `contracts/` stay total and pure — if you want to debug, do it in the calling code.

## Enforcement

- `contracts/contracts_boundary_test.go` walks `go list -deps -json ./...`, filters to packages under `github.com/hero-engine/hero/contracts`, and fails on any import that starts with `github.com/hero-engine/hero` but is not itself under `contracts/`. The failure names the offending package, directory, and import path.
- Run as a normal Go test: `go test ./contracts/...`.
- This is the only enforcement; there is no separate lint rule. Don't disable the test.

## Sibling convention (cloud side)

The mirror rule lives in **hero-cloud** as part of its `cross-repo-workflow` convention: nothing under `cloud/` or `cmd/hero-cloud/` may import `internal/` or any other in-repo non-contracts path from this repo. That convention covers the cloud-side discipline; this one covers the hero-side leaf rule. The two together keep the seam clean in both directions.

## Exceptions

None. The boundary test is absolute. If you genuinely need to share something across the seam, it belongs in `contracts/` — make it a vocabulary type and put the implementation on the correct side.
