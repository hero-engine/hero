# Delivery Audit — mcp-tool-category-metadata

Cold audit by a fresh agent (no stake in the work), re-derived against the
uncommitted tree at base `ad1303f`. Two rounds: an initial HOLD on the AC-7
safety axis, then SHIP after the fix and an exhaustive missed-writer sweep.

<AUDIT_VERDICT>
verdict: SHIP
surface: noteworthy
confidence: high
</AUDIT_VERDICT>

<AUDIT_HEADLINE>
Delivered: mcp-tool-category-metadata — 9/9 acceptance criteria clean; the AC-7
safety blocker is resolved. SHIP with 2 non-blocking residuals.

Round 1 (HOLD): three read-classed tools (`hero_active`, `hero_contract`,
`hero_snapshot`) wrote to disk but advertised `readOnlyHint:true` — a conforming
harness could auto-call a state-writer believing it safe. The lead had caught 2
writers and missed 3.

Round 2 (SHIP): all three moved to `safetyMutate` (joining `hero_plan`,
`hero_error_pattern`); all five emit `readOnlyHint:false` on the wire with
`_meta` category/tier intact. New `TestConditionalWritersAreNotReadOnly` pins
them; falsified (reverting `hero_contract` → red naming the tool). An exhaustive
sweep of all 31 remaining read/analyze handlers — following delegated package
calls and every action/archive/batch branch, not body-grep — found no sixth
user-state writer.
</AUDIT_HEADLINE>

<AUDIT_HIGHLIGHTS>
- The blocker is genuinely fixed and complete; the read/analyze set is clean.
- Residual (non-blocking): `TestConditionalWritersAreNotReadOnly` is a pinned
  allowlist of five names, not a structural check. Flipping a listed tool back
  to read fails (verified), but adding a 6th conditional writer without
  appending it fails silently. One-line-someday follow-up (derive the writer set
  from a write-capability probe); not a reason to hold. Tracked below.
- Axis nuance (not a defect): ~10 read-classed tools do `os.MkdirAll` + SQLite
  cache/graph init via `index.Open`/`graph.Open` on a cold workspace. That is
  idempotent derived-cache materialization, conventionally read-only; the five
  flagged writers (plan.md, spec frontmatter, active-session registry, snapshot
  archives, error-pattern knowledge) are categorically the right place to draw
  the readOnly boundary.
- Scoped delivery blast radius fully green: `go test ./internal/serve/...
  ./cmd/attention-conformance/...` → 22 packages ok; conformance bundle
  deterministic, SHA matches the constant. (Full `./...` stalls in the auditor's
  environment on unrelated CLI-install integration tests — the
  `cli-test-isolation-stray-workspace-boundary` issue — not this delivery. The
  lead's environment runs the full suite clean: 103 packages, 0 failures.)
</AUDIT_HIGHLIGHTS>

## Follow-up recorded — ADDRESSED

The residual (the writer guard was a pinned allowlist that couldn't catch a
future unlisted writer) has been closed by a structural guard:
`TestReadClassedToolsDoNotReachWritePrimitives`
(`internal/serve/mcp_tool_safety_reachability_test.go`). It builds a
name-resolved call graph over the whole first-party module and asserts no
read/analyze-classed tool transitively reaches `os.WriteFile`/`Create`/
`Remove`/`RemoveAll`/`Rename`. Every user-state write funnels through those
primitives; the index/graph cache uses `MkdirAll`+SQLite and never trips them,
so there are no cache false-positives (31 read/analyze tools verified clean).

Falsified against real code: reverting `hero_contract`, `hero_plan`,
`hero_snapshot`, `hero_active` to `safetyRead` each turns it RED, naming the
write path — e.g. `serve.toolActive -> active.Register -> active.Save ->
os.WriteFile` (proving cross-package, multi-hop delegation detection, the exact
thing body-grep missed). Reverts restore green.

Residual limitation (documented in the test): a write reached through a method
on a non-receiver type is not resolved without full type info, so the pinned
`TestConditionalWritersAreNotReadOnly` is kept as a belt-and-suspenders net. No
such path exists in the current tree.
