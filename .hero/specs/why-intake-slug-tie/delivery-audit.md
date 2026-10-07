# Delivery audit — why-intake-slug-tie

**Audited:** `git show 7f98fd8b` (clean detached worktree at 7f98fd8b; parent `7f98fd8b~1` for the regression check)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: a promoted intake and its spec share a slug, `hero why` resolves to the spec, and the chain includes the derived_from and mail_source hops. Evidence: `internal/traversal/why.go:122` adds `(type = 'Intake') ASC` ahead of `ingested_at`. `TestWhy_PromotedSpecWinsSlugTieWithItsIntake` (`internal/traversal/why_test.go:301`) ingests the intake one second after the spec and asserts the start node is `Feature` and that both hops are present. The auditor re-ran it against the parent `why.go` and it fails with `start node = Intake, want the promoted Feature`. At 7f98fd8b it passes.
- [✓] AC-2: same-key ties resolve deterministically. Evidence: `id DESC` is now the last sort key (`why.go:122`), so equal `(repo, intake, ingested_at)` rows can no longer be picked arbitrarily. This is structural; no test isolates the equal-timestamp, non-Intake case.

## Changes
- [✓] `internal/traversal/why.go`: `resolveTarget` ORDER BY becomes `(repo = ?) DESC, (type = 'Intake') ASC, ingested_at DESC, id DESC`, and the doc comment is updated.
- [✓] `internal/traversal/why_test.go`: adds `TestWhy_PromotedSpecWinsSlugTieWithItsIntake`, which uses the real graph store and `traversal.Why`.

## Open items
- None. Every ledger row is DONE.

## Audit notes
- **Checks the auditor re-ran:**
  - `go vet ./internal/traversal` is clean.
  - `go test ./internal/traversal` passes.
  - Every `-run Why` test in `./internal/...` passes under `-race`. This includes the `internal/cli` tests `TestWhyAndGraphAgreeOnResolution`, `TestWhySurvivesSiblingRepoIngest`, `TestWhyResolvesSpecCreatedSinceLastIngest` and `TestWhyReconcileStaysWithinBudgetAtCorpusScale`.
  - `TestPromotionResumesAfterEveryStepAndWritesBodyFreeProvenance` (`internal/attention/mail/triage_test.go`) passes 20/20 with `-race -count=20`. The ledger claims 5/5.
  - The supplied `go-tests.log` has no FAIL lines, and `vet.log` is empty.
- **The original flake did not reproduce on the parent.** The same 20x `-race` run also passed 20/20 against the parent `why.go`. The test's causal link is still sound: it calls `traversal.Why(..., result.Artifact.Slug, 4)` and asserts the derived_from and mail_source hops (`triage_test.go:199-209`). The fix is proven by the deterministic unit test, not by observing the flaky test stop flaking.
- **Callers.** `traversal.Why` has two production callers: `hero why` (`internal/cli/brief.go:474`) and the MCP `hero_why` tool (`internal/serve/mcp_tools.go:3141`). Both take a bare key, with no type qualifier. For both, the spec is the right start: its chain walks derived_from to the intake and then mail_source to the mail, so the intake still appears in the trace. A user who wants the intake as the *root* can no longer get it from `hero why <slug>` once a non-Intake node shares that key. No caller or test depends on that today. An intake that has no same-key spec still resolves to itself.
- **Other node types that share keys.** Only `Intake` is demoted. For every other pair of types, the order is unchanged except where `ingested_at` ties exactly. Before, those ties were arbitrary; now the higher id wins. MailSource keys (for example `mail_<id>`) and commit SHAs are unlikely to collide with spec slugs.
  - Repo-scoped rows still beat global rows, and that comparison comes before the Intake rank. A repo-scoped Intake therefore still beats a global (repo-less) node with the same key, such as a mission or person. This matches the prior "repo first" intent.
- **Follow-up, out of scope: `hero why --edges` has the same tie.** `runWhyEdges` (`internal/cli/brief.go:495-498`) resolves with `WHERE key = ? AND repo = ? ... LIMIT 1` and no ORDER BY. On a promoted slug it can still pick the intake or the spec nondeterministically. This also makes the "how do I reach the intake" answer unreliable. A follow-up is recommended.
- The diff matches the spec's two named files plus the spec and evidence artifacts. There is no scope drift.
