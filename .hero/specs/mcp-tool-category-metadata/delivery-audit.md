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

## Follow-up recorded

The safety guard's allowlist limitation is a real but low-priority hardening:
today it regresses-proof the five known conditional-writers but cannot catch a
future unlisted one. A structural replacement (probe each read-classed handler
for a write path, or derive the writer set) is the durable fix. Out of scope for
this delivery; surfaced to the user.
