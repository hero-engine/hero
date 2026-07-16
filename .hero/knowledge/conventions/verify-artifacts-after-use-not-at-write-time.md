---
type: convention
status: draft
scope: ["internal/install/**", "internal/snapshot/**", "internal/managed/**", "**/*_test.go"]
tags: [testing, install, managed-region, regression, verification]
relations:
  - target: agents-md-erased-by-snapshot-pointer-writer
    kind: introduced-by
  - target: install-integrity-self-check
    kind: related
---

# Verify Generated Artifacts After Ordinary Use, Not At Write Time

## Pattern

When a generated artifact must stay correct over time, assert on it
**after an ordinary command runs against the repo** — not immediately
after the code that writes it.

A check that runs right after the writer is checking the one moment the
artifact is guaranteed correct. It proves the writer works. It cannot
prove the artifact *survives*, and survival is what users experience.

## Why this rule exists

The snapshot pointer writer silently replaced the whole `hero:managed`
region of AGENTS.md with a pointer-only section, deleting every install
section — 232 lines to a 7-line stub — on any ordinary
`hero snapshot --project` / `hero next checkpoint` run. Five of six
harnesses started completely cold. It ran for two months and was found
by accident.

**There was already a test asserting AGENTS.md content.**
`TestHarness_SmokeCodex` checked the managed markers, the
"Running Hero Workflows in Codex" section, and the `command-deliver`
reference. It was a good test. It was structurally incapable of catching
this, because it asserted immediately after install and the eraser fired
on the *next* command.

The manual check had the identical blind spot. `codex-install-broken`'s
completion ledger recorded its evidence as, verbatim:
*"verified in AGENTS.md after install."*

So the failure wasn't missing coverage. It was coverage pointed at the
wrong moment. The tests modeled install as a terminal state; nothing
modeled living in the repo.

## Rule

> **Install is not a terminal state.** If a test's assertion would still
> pass when a later writer destroys the artifact, it is testing the
> writer, not the artifact.

Structure such a test as: **write → run an ordinary command → assert.**
The middle step is the whole test. See
`TestHarness_InstalledContentSurvivesOrdinaryCommands`
(internal/install/harness_smoke_test.go).

## Applying it

- **Assert the invariant, not the culprit.** A guard aimed at the one
  eraser you found assumes the next one lives in the same function. Assert
  "the root file is byte-identical after an ordinary command" and any
  writer fails it, from any package, including ones not written yet.
- **Cover every target.** The original bug was filed as "Codex install is
  broken" but hit five of six harnesses; claude escaped only because the
  eraser hardcoded the literal string `"AGENTS.md"` and never resolved
  `nativeInstructionFile(target)`. See the
  `harness-changes-cover-all-targets` tripwire.
- **Prove the guard bites.** Reintroduce the bug and watch the test go
  red. A green test proved nothing here for two months; a guard that has
  never failed is a guess.
- **Characterize destructive semantics on purpose.** `writePointerOnly`
  legitimately replaces a whole region — correct for NEXT.md, catastrophic
  for an install-managed file. `TestWritePointerOnly_ReplacesEntireManagedRegion`
  pins that behavior so the sharp edge is visible and a future change to it
  is a conscious decision rather than a silent one.

## Related anti-pattern: skipped items in a closed spec

`codex-install-broken` designed the detection for this exact failure and
cut it: Fix 4 (`hero check` advisories) as "P3 priority", Fix 6
(auto-detect stale managed regions) as "lower priority". Every fix that
shipped **wrote** content; every fix cut **detected** problems. The spec
then closed while the bug was live — its Evidence section is a verbatim
copy of the stub that stayed in the repo for another five weeks.

A row in a ledger is a note. Only a spec is work: skipped rows in a
completed spec don't appear in `hero queue`, `hero list`, or any search
for open work. If a follow-up matters, it needs its own spec before the
parent closes. See [[install-integrity-self-check]], which exists to
revive exactly those two cut items.
