# Delivery audit — polish-and-suggested

**Audited:** commit db89aea1, including the reason-wording fix, at 8597d000. That covers `internal/workmodel/polish.go`, `polish_test.go`, and the `toolWork` wiring in `internal/serve/mcp_tools_read_contract.go`. Built and run in a clean worktree at 8597d000.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: polish only on recently_done items.
  - **Kinds** (`Polish`, polish.go):
    - `weak_verify` when verify is not passed; its next step is the item's own.
    - `open_followups`: unfinished, related in either direction, not a bug, and created on or after the completion day.
    - `bug_against_recent`: unfinished related bugs.
    - `rated_worse` is never emitted.
  - **Test:** `TestPolishKinds` covers all three kinds, the exclusion of an older related spec, `rated_worse` being absent, and recent-only.
  - **Real corpus** (auditor's fixture export): 5 `weak_verify` entries, each with `/verify <slug>`.
- [✓] AC-2: at most 3 queue picks.
  - **Rule:** ready items by priority, then path; then undesigned stubs whose next action is design or diagnose. The reason is derived from the pick's next step.
  - **Test:** `TestSuggested`.
  - **Real corpus:**

    | Pick | Reason |
    |---|---|
    | `tracker-backed-diagnosis-publication-contract-broken` | "ready to fix, critical priority" |
    | `always-on-runtime` | "ready to drive, critical priority" |
    | `machine-checkable-acs` | "ready to deliver, high priority" |

    Each reason matches the pick's command.
- [✓] AC-3: Explore when thin.
  - When fewer than 3 items are ready or in progress, the list ends with `{slug: null, title: "Explore what's next", source: queue, next: discover / Explore / "/discover"}`.
  - `TestSuggested` checks it appears when thin and is absent when not.
  - `TestToolWorkShape` asserts it through the tool.
  - The golden records `suggested[].slug:null`.

## Changes
- [✓] `internal/workmodel/polish.go` and `polish_test.go`.
- [✓] Tool wiring: `toolWork` fills `Polish(items, specs)` and `Suggested(items)`. `serve` aliases the types. The global `workRevision` hashes both lists.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- **Real corpus not exercised, contract rule:** when a finished initiative or epic is recently done and its verify is not passed, it gets a `weak_verify` entry with `next: null`, because finished initiatives have no next step. That matches the contract ("verify.state is not passed") but gives the client no action. No such item exists in this corpus.
- **Edge case:** an item that is recently_done only through the mtime fallback (no `completed_at`) never collects `open_followups`, because the "created on or after completion" check needs `CompletedAt`.
- `Polish` builds its own `NewCorpus` and scans all items for each recently_done item (O(recent × n)). That is negligible at 16 recent items.
- **Re-run (auditor):** at 8597d000, the `workmodel`, `serve` and `cli` tests pass, and `go vet` is clean.
