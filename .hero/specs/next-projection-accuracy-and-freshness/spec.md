---
title: "NEXT.md projection recommends blocked and type-restricted work; QUEUE.md goes stale between commits"
type: bug
status: completed
priority: P1
severity: moderate
size: small
domain: engineering
---

## Goal

Make the "## Next" section of NEXT.md recommend work the user can actually start, surface high-priority bugs alongside features, and keep QUEUE.md as fresh as NEXT.md across all harnesses.

## Kickoff

Fix three bugs in the NEXT.md/QUEUE.md projection pipeline. In `internal/projection/projection.go`, `openFeaturesByPriority` (line 200) recommends blocked specs and excludes bugs — rename to `readyWorkByPriority`, add a NOT EXISTS clause against unmet dependency edges, widen the type filter to Feature + Bug, and implement two-slot logic (top feature + top P0/P1 bug). In `internal/cli/checkpoint.go`, add a `RenderQueueSnapshot` call inside `writeCheckpoint()` with content-hash gating so QUEUE.md refreshes on every checkpoint without causing constant dirtiness. Run `go test ./internal/projection/ ./internal/cli/` to verify.

## Problem

Three related bugs degrade the handoff projection's usefulness:

1. **Blocked specs appear in "## Next"**: `openFeaturesByPriority()` selects by priority alone. A P0 feature blocked on an unfinished dependency wins the "Next" slot, then appears again under "## Blocked on" — the file contradicts itself. The user gets a `/deliver` prompt for work they can't start.

2. **Only Feature nodes qualify for "Next"**: The `WHERE type = 'Feature'` filter means P0 bugs never surface as recommended next work, even when they're the most urgent ready item. The queue MCP tool has no such restriction, creating a "queue says X, NEXT says Y" divergence.

3. **QUEUE.md goes stale between commits**: The Stop hook calls `hero next checkpoint` but not `hero queue write`. QUEUE.md only refreshes on pre-commit (where both commands run). Sessions in hookless harnesses (opencode, cursor, copilot, codex, generic, grok) see multi-day-stale queue files.

## Design

### Fix 1: Exclude blocked specs from "## Next"

In `openFeaturesByPriority` (to be renamed), add a `NOT EXISTS` subquery that excludes nodes with an outgoing `depends_on` or `blocks` edge to a non-completed/non-accepted target — exactly the same join logic `blockedFeatures()` already uses, inverted:

```sql
AND NOT EXISTS (
    SELECT 1 FROM edges e
    JOIN nodes b ON e.to_id = b.id AND b.valid_to IS NULL
    WHERE e.from_id = nodes.id
      AND e.type IN ('depends_on', 'blocks')
      AND e.valid_to IS NULL
      AND COALESCE(json_extract(b.props, '$.status'), '')
          NOT IN ('completed', 'accepted')
)
```

### Fix 2: Two-slot "## Next" with Feature + Bug

Rename `openFeaturesByPriority` → `readyWorkByPriority`. Change the type filter from `type = 'Feature'` to `type IN ('Feature', 'Bug', 'Enhancement')`.

The renderer in `NextMD()` populates two slots:

- **Slot 1** (always): Top ready item by priority (any deliverable type). Shows the `/deliver <slug>` prompt.
- **Slot 2** (conditional): If slot 1 is a feature, also show the top ready P0/P1 bug (if one exists). If slot 1 is already a bug, show the top ready feature instead. This gives a "planned work + urgent interrupt" view without low-priority items muddying the signal.

If both slots resolve to the same item, show only one. The `/deliver` prompt always points at slot 1.

### Fix 3: QUEUE.md refresh in checkpoint (harness-agnostic)

Add a queue-write step to `writeCheckpoint()` in `checkpoint.go`, after the NEXT.md projection. Call `RenderQueueSnapshot(heroDir)` (already exported from `queue.go`) and write with content-hash gating.

QUEUE.md's `Generated:` timestamp lives inside an HTML comment header. The content-hash comparison must normalize this timestamp (same pattern as `normalizeUpdatedFrontmatter` for NEXT.md's `updated:` field) to avoid writes when only the timestamp changed.

This makes every checkpoint trigger (Stop hook, PreCompact hook, pre-commit hook, post-merge hook, manual `hero next checkpoint`) also refresh QUEUE.md. Harness-agnostic — no Claude-specific hook changes needed.

### Non-changes

- The `updated:` frontmatter field in NEXT.md stays. `writeProjectedFileIfSemanticChanged` already normalizes it before comparison, preventing timestamp-only dirtiness. The existing mechanism works.
- The pre-commit hook's explicit `hero queue write -q` call can remain as a belt-and-suspenders alongside the new checkpoint-internal queue write. Both are idempotent.
- No changes to hook installer or `.claude/settings.json` — the fix lives in checkpoint itself.

## Acceptance Criteria

- AC-1: WHEN a spec has an unresolved `depends_on` edge THE SYSTEM SHALL exclude it from the "## Next" section of NEXT.md
- AC-2: WHEN a P0 or P1 bug is ready (unblocked, not completed/superseded) THE SYSTEM SHALL show it as a second suggestion in "## Next" alongside the top ready feature
- AC-3: WHEN the top ready item by priority is a bug THE SYSTEM SHALL show it in slot 1 and show the top ready feature in slot 2 (if one exists)
- AC-4: WHEN no ready items exist for a slot THE SYSTEM SHALL omit that slot rather than showing a blocked or low-priority fallback
- AC-5: WHEN `hero next checkpoint` runs THE SYSTEM SHALL also refresh QUEUE.md with content-hash gating
- AC-6: IF QUEUE.md content has not changed (ignoring the Generated timestamp) THEN THE SYSTEM SHALL skip the write to avoid dirtying the working tree
- AC-7: THE SYSTEM SHALL rename `openFeaturesByPriority` to `readyWorkByPriority` to reflect the widened type scope

## Verification

```bash
go test ./internal/projection/ ./internal/cli/ -run "TestNextMD|TestQueue|TestCheckpoint"
```

Falsification: revert the blocked-exclusion change and confirm a test fails with a blocked spec appearing in "## Next".
