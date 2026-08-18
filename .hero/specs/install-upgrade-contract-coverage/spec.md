---
title: Install + Upgrade Contract Coverage — Prove Every Target Works Every Time
slug: install-upgrade-contract-coverage
type: initiative
status: completed
priority: P0
severity: high
tags: [install, upgrade, testing, foundation, multi-harness, ci]
created: 2026-05-12
horizon: now
completed_at: 2026-08-18T03:44:33Z
---

## Goal

Make `hero install` and `hero upgrade` provably correct on every
supported target, every release, by enforcing per-target contracts
on the installed output rather than just file-existence smoke.

Concrete bar: a Hero release SHALL NOT pass CI if any of the six
install targets (claude, opencode, cursor, codex, copilot, generic)
produces output that the consuming harness cannot register or
consume. Today only one target × one content kind has that
guarantee — claude × agents, just added in the
`claude-subagent-frontmatter-registration` fix. The other 17
target-by-kind cells are uncovered.

## Why this is its own initiative

Install is the worst surface to ship bugs on. A user who hits a
broken install on first contact never recovers — they uninstall,
move on, and Hero loses them. The
`claude-subagent-frontmatter-registration` bug shipped silently for
the entire history of the agents/ directory because file-existence
tests passed: the files landed, they just weren't consumable. That
class of bug is structurally invisible to the test suite as it
exists.

Closing one cell at a time is the wrong shape — it's whack-a-mole
and the next contributor adding a new target or content kind has
no scaffolding to plug into. The right move is one foundational
piece (the contract registry primitive) and then sweep every cell
with the new tooling. That's an initiative, not a feature.

Adjacent recent work this builds on:
- `multi-harness-install-collision` — established canonical content
  materialization with idempotency.
- `claude-subagent-frontmatter-registration` — closed the first
  contract cell and added the canonical-source contract test in
  `content_test.go`.
- `monorepo-satellite-installs` — adds satellite installs, which
  this initiative must extend coverage to.

## Audit (state of the world today)

| Target   | Smoke test | Agent contract | Command contract | Skill contract | Hooks/MCP contract |
|----------|------------|----------------|------------------|----------------|---------------------|
| claude   | yes        | **yes (just added)** | none           | none           | partial (`wireClaudeHooks`, `wireClaudePermissions` have unit tests) |
| opencode | yes        | none           | none             | none           | partial (`opencode.json` MCP tests exist) |
| cursor   | yes        | none           | none             | none           | none |
| codex    | **none**   | none           | none             | none           | partial (`wireCodexHooks` unit tests) |
| copilot  | **none**   | none           | none             | none           | none |
| generic  | **none**   | none           | none             | none           | none |

Upgrade-path coverage today: scattered satellite tests plus
`TestRunIdempotentReinstall` and `TestRunIdempotentAcrossTargets`
from the multi-harness collision fix. **No end-to-end "install
at version N-1, upgrade to N, assert resulting tree satisfies
all current contracts AND user edits preserved"** test exists.

CI gate today: `go test ./...` runs in CI but there is no
documented release-gate policy that ties install/upgrade test
failures to release blocking. (Verify and document as part of
this initiative.)

## Contents

Five sequenced child specs. (1) is foundational — everything else
depends on it. (2)/(3)/(5) are parallelizable after (1). (4) lands
last so it asserts contracts on every target.

### 1. install-contract-registry-foundation (FIRST — blocking)

**Problem.** No primitive exists for "this target consumes content
kind X in shape Y; prove the installed output fits." Per-target
tests reinvent the assertion ad-hoc, so most don't bother.

**Build.**
- Define a `HarnessContract` shape per target × kind. Probably:
  `type HarnessContract struct { RequiredFrontmatter []string; FilenamePattern string; ContentValidator func([]byte) error }`.
- Each `internal/install/target_*.go` declares a
  `Contracts() map[ContentKind]HarnessContract`.
- Add `(*installHarness).mustSatisfyContract(target Target, kind ContentKind)` that walks the installed destination dir for that kind and runs each file through the declared contract.
- Land Claude × commands and Claude × skills as the FIRST consumers
  to prove the primitive — these are almost certainly latent-bug cells.
- Hard-fail at registration if any (target, kind) returns nil
  contract — forces every new target/kind to declare its shape.

**Done when.** `TestHarness_SmokeClaude` calls
`mustSatisfyContract(TargetClaude, KindAgents)`,
`mustSatisfyContract(TargetClaude, KindCommands)`,
`mustSatisfyContract(TargetClaude, KindSkills)` and all three pass.

**Stub for /design.** Already specced enough above to drop into
`/design install-contract-registry-foundation` and produce a fix
spec.

---

### 2. install-smoke-coverage-codex-copilot-generic (parallel after #1)

**Problem.** Three of six install targets have **zero** end-to-end
smoke tests. We have no proof `hero install --target codex` (or
copilot, or generic) produces consumable output, or even runs to
completion without panicking, on a clean fixture project.

**Build.**
- Add `TestHarness_SmokeCodex`, `TestHarness_SmokeCopilot`,
  `TestHarness_SmokeGeneric` mirroring the existing
  `TestHarness_SmokeClaude` shape.
- Each test uses `mustSatisfyContract` from #1 for every content
  kind that target installs.
- Verify each target's hooks/config wiring (codex_hooks, copilot
  config, generic AGENTS.md) lands and parses.

**Done when.** All 6 targets have a `TestHarness_Smoke<X>` and each
satisfies its declared contracts.

**Stub for /design.** Drop into
`/design install-smoke-coverage-codex-copilot-generic` after #1
lands.

---

### 3. install-contract-coverage-opencode-cursor (parallel after #1)

**Problem.** OpenCode and Cursor have smoke tests but no
contract assertions. OpenCode has its own `opencode.json` MCP
schema; Cursor has rule-frontmatter requirements. Both are latent
bugs waiting for a registry/parser change in those tools.

**Build.**
- Declare `Contracts()` for OpenCode (agent name-from-filename,
  required `description:`, opencode.json MCP block schema) and
  Cursor (rule frontmatter, .mdc shape).
- Wire `mustSatisfyContract` calls into existing
  `TestHarness_SmokeOpenCode` and `TestHarness_SmokeCursor`.
- Spot-check live OpenCode and Cursor docs to make sure the
  declared contracts match what those tools actually require —
  do not author contracts from memory.

**Done when.** Both smoke tests exercise full per-kind contract
assertions.

**Stub for /design.** Drop into
`/design install-contract-coverage-opencode-cursor` after #1.

---

### 4. install-upgrade-path-coverage (after #1, ideally after #2/#3)

**Problem.** Upgrade flows (`hero install --migrate`,
`hero upgrade`, satellite repair) have no end-to-end coverage that
asserts the resulting tree still passes contracts AND that
user-edited managed regions are preserved. The bug class:
`hero upgrade` ships, but a previously-installed user's tree
silently degrades because the new release expects a contract the
old install doesn't satisfy yet.

**Build.**
- New test file `internal/install/upgrade_test.go` (or extend
  `migrate_test.go`).
- Test fixtures simulating an installed Hero workspace at the
  previous minor version (or with a pinned old contract shape).
- End-to-end:
  1. Stand up a "previous install" tree with realistic content +
     user edits in managed regions.
  2. Run `hero install --migrate` (or the upgrade entry point).
  3. Assert the resulting tree passes every (target × kind)
     contract from #1/#2/#3.
  4. Assert user-edited content outside managed regions is
     byte-identical.
  5. Assert managed regions inside files like AGENTS.md /
     CLAUDE.md got refreshed without losing user-authored sections
     outside the markers.
- Cover satellite repair: install satellites, drift one, repair,
  assert contracts hold post-repair.
- Cover idempotency: upgrade twice, second run is a no-op.

**Done when.** Upgrade scenarios for all 6 targets are covered;
managed-region preservation is asserted; satellite repair is
covered.

**Stub for /design.** Drop into
`/design install-upgrade-path-coverage` after #1 and ideally #2/#3.

---

### 5. install-ci-gate-policy (parallelizable, low effort)

**Problem.** No documented policy that ties install/upgrade test
failures to release blocking. Today's coverage is implicit
("`go test ./...` runs in CI") and easy to bypass without anyone
noticing.

**Build.**
- Audit current CI config (GitHub Actions / whatever the gate is).
  Verify install/upgrade tests run on every PR and on every release
  tag.
- Document the policy in `internal/install/README.md` (create if
  missing) or in the project context: "install/upgrade test
  failures block release; do not bypass."
- If today's CI doesn't already gate releases on install tests,
  fix that.
- Stretch: tag install/upgrade tests with a build tag or naming
  convention so they can be run as a focused subset in pre-release
  smoke (`go test -run 'Install|Upgrade|Harness' ./internal/install/...`).

**Done when.** CI gate is verified, documented, and enforced.

**Stub for /design.** Drop into
`/design install-ci-gate-policy`. Independent of #1-#4 — can run
in parallel with any of them.

---

### 6. install-real-harness-validation (stretch / lower priority)

**Problem.** Even with full per-target contract coverage, the unit
tests can't prove the consuming tool (Claude Code, OpenCode,
Cursor) actually loads what we wrote. The contract is our model of
what they want; reality may diverge.

**Build.**
- `make verify-install` (or `hero verify install --target X`) that
  installs into a fixture project, then launches the target tool
  in a headless / scriptable mode and asserts the agent registry
  is populated.
- Probably uses each tool's MCP debug surface or CLI introspection
  (Claude Code has `/agents` listing; OpenCode has CLI listing;
  Cursor's rule listing surface is unclear — investigate first).
- Manual / scheduled rather than per-PR — these are slow and
  flaky-prone. Run on release-candidate tags.

**Done when.** A scheduled job (or manual `make verify-install`
target) exercises real-harness loading for at least Claude Code
and OpenCode and surfaces failures to the team.

**Stub for /design.** Drop into
`/design install-real-harness-validation`. Lowest priority — defer
until #1-#5 land. Treat as a separate initiative rather than a
blocking child if it grows large.

## Recommended delivery order

1. **install-contract-registry-foundation** (blocks everything else)
2. **install-ci-gate-policy** (parallel — independent, small,
   protects every later landing)
3. **install-smoke-coverage-codex-copilot-generic** (after #1, in
   parallel with #4 if appetite allows; closes the largest gap)
4. **install-contract-coverage-opencode-cursor** (after #1, in
   parallel with #3)
5. **install-upgrade-path-coverage** (after #1, ideally after #3
   and #4 so it asserts contracts on every target)
6. **install-real-harness-validation** (stretch; after #1-#5 or
   spun out as its own initiative)

Critical path: #1 → #5. Estimated foundation work (#1) is 1-2
days. Per-cell sweep (#2 + #3) is another 1-2 days in parallel.
Upgrade coverage (#4) is the largest piece — likely 2-3 days
because of the fixture work. CI policy (#2 in the order above) is
a few hours.

## Cross-cutting concerns and shared risks

- **Contract drift between Hero's model and the consuming tool's
  reality.** Mitigation: cite the source for each declared
  contract (link to Claude Code docs, OpenCode README, Cursor
  rules docs) in the contract definition. When a tool changes its
  schema, the cited source goes stale and a future audit catches
  it. Stretch goal #6 (real-harness validation) is the ultimate
  defense.
- **Test runtime.** Six targets × multiple kinds × multiple
  scenarios will balloon test time. Mitigation: keep the harness
  fixture small (current `seedSource` is the right scale); use
  `t.Parallel()` where safe.
- **Symlink fallback path.** Every assertion has to work whether
  the install rendered to symlinks or to copies (linkOrRenderDir).
  The contract test reads bytes off the destination file, which is
  agnostic to which path produced it — verify in #1.
- **Satellite + canonical interaction.** `monorepo-satellite-installs`
  is in flight. Coordinate with that initiative — the contract
  primitive from #1 should be reusable for satellite assertions
  in #4.
- **Embedded source vs disk source.** Contract tests must run
  against both the in-binary embedded FS (caught by
  `content_test.go`) AND the on-disk installed output (caught by
  the new `mustSatisfyContract`). Both layers stay; they catch
  different classes of failure.
- **OpenCode/Crush vs Claude Code frontmatter conflict.**
  `claude-subagent-frontmatter-registration` proved we can satisfy
  both with one source file. Future contracts must keep that
  property — never declare a contract that requires removing a
  field another target needs.

## Acceptance Criteria (initiative-level)

- WHEN any of the 6 install targets installs into a clean project
  THE SYSTEM SHALL produce output that satisfies a declared
  per-(target, kind) contract.
- WHEN a new install target is added in `internal/install/target_*.go`
  THE SYSTEM SHALL fail tests until that target declares contracts
  for every content kind it installs.
- WHEN `hero upgrade` or `hero install --migrate` runs against a
  previously-installed tree THE SYSTEM SHALL preserve user-edited
  content outside managed regions byte-identical AND produce a
  resulting tree that passes every current contract.
- WHEN install or upgrade tests fail THE SYSTEM SHALL block
  release at the CI gate (per documented policy).
- THE SYSTEM SHALL maintain the single-source canonical-symlink
  architecture established by
  `multi-harness-install-collision` and
  `claude-subagent-frontmatter-registration`. No per-target
  content translation introduced.

## Out of scope

- Validating Hero's slash-command behavior end-to-end inside each
  harness (separate concern — covered by per-feature smoke
  initiative).
- Performance optimization of install/upgrade (covered separately
  if it becomes a bottleneck).
- New install targets (Aider, Continue, Cody, etc.) — once #1 is
  in, adding a target requires declaring contracts; this
  initiative is about coverage of the existing six.

## Kickoff

Foundation initiative for proving install + upgrade work on every
target, every release.

**Status:** planning — ready to design child #1.

**Pick up at:**
1. Run `/design install-contract-registry-foundation` to produce
   the foundation child spec. That spec defines the
   `HarnessContract` shape, the `Contracts()` method per target,
   and the `mustSatisfyContract` harness primitive. Lands Claude ×
   commands + Claude × skills as the first consumers.
2. While #1 is in design, optionally `/design install-ci-gate-policy`
   in parallel — independent and small.
3. After #1 lands, fan out: `/design install-smoke-coverage-codex-copilot-generic`
   and `/design install-contract-coverage-opencode-cursor` in
   parallel.
4. Then `/design install-upgrade-path-coverage`.
5. Defer #6 (real-harness validation) until #1-#5 have landed and
   it's clear how much investment it warrants.

**Why this matters now:** the
`claude-subagent-frontmatter-registration` fix today closed one
cell of an 18-cell coverage matrix. The remaining 17 cells are
the same class of latent bug — install ships, files land, the
consuming tool silently rejects them, the user gets degraded
behavior with no error to report. This is the worst surface to
ship bugs on, and the matrix won't close itself one-spec-at-a-time
without the contract registry primitive in #1.

**Coordination:** check in with `monorepo-satellite-installs`
before #4 so satellite-aware upgrade tests reuse the contract
primitive rather than building a parallel one.
