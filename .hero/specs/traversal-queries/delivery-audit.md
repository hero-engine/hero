---
audit_date: 2026-06-09
spec: traversal-queries
verdict: SHIP
auditor: claude-sonnet-4-6 (cold audit)
---

# Delivery Audit — traversal-queries

**Spec:** `.hero/planning/features/traversal-queries/spec.md`
**Audit date:** 2026-06-09
**Verdict:** SHIP

---

## AC-by-AC findings

### AC-1 — `hero why <slug>` recursive CTE, depth 4 default, `--depth` flag
**PASS**

- `internal/traversal/why.go:81` — `Why()` entry point, delegates to `resolveTarget` + `walkOrigins`.
- `walkOrigins` (line 145) executes a `WITH RECURSIVE chain(…)` CTE over `edges`, bounded by `c.depth < maxDepth`.
- `const DefaultDepth = 4` (line 72).
- CLI: `internal/cli/brief.go:434` — `whyCmd.Flags().IntVar(&whyDepth, "depth", traversal.DefaultDepth, …)`.
- `runWhy` (line 439) passes `whyDepth` through to `traversal.Why()`.
- `TestWhy_TwoHopChain` (why_test.go:12) — seeds Criterion → Feature → Initiative chain and asserts 2-hop result.
- `TestWhy_DepthBoundsRecursion` (why_test.go:38) — verifies `Why(…, depth=2)` stops at 2 hops on a 3-deep chain.

### AC-2 — `hero why <file-path>` chain through commits to spec
**SKIPPED [signed-off]**

Documented deferral at `why.go:79`: "Path/SHA disambiguation arrives in a follow-up." Spec ledger (line 241) records the decision. The skip is intentional and captured; no gap.

### AC-3 — `hero why <feature:AC-N>` returns origin chain of an AC
**PASS**

- `resolveTarget` (why.go:108) does exact-key matching — colon-form keys like `feat-x:AC-1` are first-class node keys; no special-casing needed as long as nodes are seeded with that key.
- `TestWhy_TwoHopChain` (why_test.go:12) seeds `feat-x:AC-1` as target and asserts the chain traverses Criterion → Feature → Initiative.
- `TestWhy_BoundaryAwareHandoff` (why_test.go:193) further confirms cross-domain `derived_from` hop resolution.

### AC-4 — `hero blocked` returns dependency tree of open features
**PASS**

- `runBlocked` (brief.go:552): queries `nodes f JOIN edges e ON e.type IN ('depends_on','blocks') JOIN nodes b` for open features, renders tree.
- Filters completed/superseded features and completed/accepted blockers correctly (line 583–583).
- `acFailuresByParent` result merged into output at lines 643–664.

### AC-5 — `hero blocked` joins failing/regressed Criterion nodes
**PASS**

- `acFailuresByParent` (brief.go:675): queries Criterion nodes with `status IN ('failing','regressed')`, groups by `props.parent`.
- Inline merge into per-feature output (lines 643–648): ACs printed as `failing AC \`key\` (status)`.
- Standalone failing-AC entries rendered for features with no dep-blockers (lines 652–664).

### AC-6 — Recursive CTE bounded at maxDepth with cycle detection
**PASS**

- `walkOrigins` CTE uses `AND c.depth < ?` bound (line 165), with `maxDepth` as the final arg.
- `SELECT DISTINCT` on `(chain.depth, chain.edge_type, n.type, n.key, …)` in the outer SELECT prevents duplicate hop emission.
- `TestWhy_BreaksCycles` (why_test.go:79): seeds `a → b → a` cycle, runs with `depth=6`, asserts `len(chains) <= 12`. Call returns without infinite recursion.

Note: The cycle bound relies on the depth cap, not a visited-set guard. On a graph dense enough that DISTINCT doesn't collapse rows, cycles are only guaranteed to terminate within `2 * maxDepth` hops. The test confirms termination and bounds the output; this is acceptable for the current depth ranges.

### AC-7 — `hero resume`/`hero next` includes "Currently blocked:" section automatically
**PASS**

- `digest.go:190–196`: `blockedSection(store, opts, plan.BlockedOn)` called unconditionally in `Generate()` (section 5 of 8).
- `digest.go:604`: `blockedSection` returns `BriefSection{Title: "Blocked on"}` and emits `_nothing blocked_` when the query returns empty — never omits the call.
- `digest_test.go:89`: asserts `"## Blocked on"` appears in `Generate()` output.

### AC-8 — Queries use indexes; depth-4 traversal < 200ms
**PASS**

- `TestWhy_DepthFourUnder200ms` (why_test.go:102): seeds 6-node chain, warm-runs once, then averages 10 cold runs against a 200ms budget.
- Spec records avg ~1ms in-process (commit ae91f4c EXPLAIN QUERY PLAN confirms `idx_edges_from(from_id, type)` and `idx_nodes_current(type, key) WHERE valid_to IS NULL` index coverage).

### AC-9 — `hero_why` and `hero_blocked` MCP tools registered
**PASS**

- `mcp_dispatch.go:48–49`: `"hero_why": s.toolWhy` and `"hero_blocked": s.toolBlocked` in dispatch table.
- `mcp_tools.go:3021` (`toolWhy`) and `mcp_tools.go:3067` (`toolBlocked`): full implementations present.
- `mcp_tools_def.go:479` and `mcp_tools_def.go:492`: tool definitions in the published tools list.
- `mcp_expand.go:202–204`: tools appear in expand routing.
- `mcp_test.go:286–287`: test asserts both tool names present in registered set.

---

## Build and test evidence

Stated in spec exercise-the-feature check:
- `go build ./...` clean.
- `go test ./internal/traversal/... ./internal/digest/...` pass.
- Named tests confirmed present and correct in source: `TestWhy_TwoHopChain`, `TestWhy_BreaksCycles`, `TestWhy_DepthFourUnder200ms`, `TestWhy_DepthBoundsRecursion`, `TestWhy_BoundaryAwareHandoff`, `TestWhy_WalksSupersedesEdge`, `TestMarkdown_RendersHopsIndented`.
- `digest_test.go:89` asserts `"## Blocked on"` in digest output.

---

## Noteworthy

**Cycle detection approach.** `walkOrigins` relies on SQLite's DISTINCT + depth cap rather than a visited-set. For small graphs (depth ≤ 6) this is fine and matches the spec's "cycle test terminates within depth bound" language. If the graph grows densely and users pass `--depth 16+`, DISTINCT alone may not prevent combinatorial expansion. Not a ship blocker — noted for the `forward-traversal` follow-up.

**AC-2 signed-off skip.** The deferral is clean: comment at why.go:79, spec ledger entry, explicit `SKIPPED [signed-off]` status. No ambiguity.

---

## Summary

9 ACs audited. 8 PASS, 1 SKIPPED [signed-off]. All implementation surfaces are present and correctly wired. Spec is ready to archive.
