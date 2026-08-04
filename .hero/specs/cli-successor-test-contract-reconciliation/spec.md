---
title: "Interactive CLI successor tests contradict the delivered contract"
slug: cli-successor-test-contract-reconciliation
type: bug
status: completed
domain: engineering
size: medium
priority: critical
severity: high
root_cause_class: design
created: 2026-08-03
tags: [cli, tests, prompts, merge-gate, regression]
relates-to:
  - interactive-cli-input-scoped-completion
delivery_method: manual
completed_at: 2026-08-04T05:56:21Z
---

# Interactive CLI successor tests contradict the delivered contract

## Issue

The completed `interactive-cli-input-scoped-completion` initiative is not
merge-ready: `go test -count=1 -timeout 10m ./...` fails in
`internal/cli`. The branch's acceptance evidence records that exact command as
passing, but a fresh run outside the network-restricted sandbox deterministically
fails 26 test nodes. There is no configured tracker.

The failure was discovered during the user's explicit pre-merge gate on
`codex/interactive-cli-input-scoped-completion`. Production code must not be
changed merely to satisfy predecessor tests: the delivered behavior in the
failing areas matches the successor specs.

## Summary

### Categorization

| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — blocks the requested merge and invalidates the recorded full-suite evidence |
| **Ease of Fix** | moderate — production is sound, but several test contracts, four golden fixtures, and archived evidence must be reconciled deliberately |
| **Caused by our codebase?** | Yes — selectively ported tests retained predecessor assumptions that later successor children intentionally replaced |
| **Needs more research?** | No — the failure set is deterministic outside the sandbox and every failing assertion traces to an explicit successor contract |

### Background

The successor was composed from several children delivered in sequence. The
prompt child imported donor tests that described pre-gate stream behavior. The
setup and selector children then changed the intended contract without updating
all inherited assertions. One donor test also references a release-note file
that the scoped successor explicitly chose not to introduce because `main` has
no release-note surface.

### Analysis

Four independent stale-test groups make the suite red:

1. `prompt_adoption_test.go` drives `skill save` and `promptNextStatus` through
   non-terminal `bytes.Buffer` streams while expecting terminal prompts and
   answers. Production now gates those prompts on `prompt.IsInputTTY`.
2. Four connect golden fixtures still expect the prompt child's generic
   non-terminal error and old help, while the setup child requires the
   provider's first missing-field error and updated flag help.
3. `selector_test.go` requires exactly two consumers of a generic
   `collectFields` descriptor, while the setup child expressly rejected
   `promptfield.go`/skill coupling and delivered zero generic consumers.
4. `prompt_policy_test.go` opens
   `docs/release-notes/breaking-changes.md`, a donor-only surface absent from
   both `main` and the successor. Its comment cites donor AC-14, which is not an
   acceptance criterion in the recomposed prompt child.

### Root Cause

This is an integration-design failure, not a production prompt defect and not
filesystem shared-state leakage. Each child validated its local behavior, but
the successor had no enforced ownership rule requiring a later child to update
earlier child tests and golden fixtures when it intentionally superseded their
contract. The final gate then recorded a full-suite PASS that current git
history cannot support: all contradictory tests existed before the evidence
commit, and no later commit repairs them.

### Source

- `internal/cli/prompt_adoption_test.go` — stale non-TTY expectations for
  `runSkillSave` and `promptNextStatus`.
- `internal/cli/testdata/prompt_baseline/connect_github_{repo,secret}_prompt.{closed,pipe}.txt`
  — stale connect errors and help bytes.
- `internal/cli/selector_test.go` — stale two-consumer descriptor assertion.
- `internal/cli/prompt_policy_test.go` — stale dependency on a donor-only docs
  path.
- `.hero/specs/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md`
  — unsupported PASS record.

### Fix Direction

Reconcile tests, fixtures, and evidence to the already-approved successor
contract. Do not relax the TTY gates, reintroduce the generic field descriptor,
or change connect's provider-specific early failure. Keep the donor
`cli-test-isolation-stray-workspace-boundary` enhancement separate.

---

## Problem Statement

### Reproduction

From `/private/tmp/hero-interactive-cli-scoped`:

```bash
go test -json -count=1 -timeout 10m ./internal/cli \
  | jq -r 'select(.Action == "fail") | [.Package, (.Test // "PACKAGE")] | @tsv'
```

Outside the network-restricted sandbox this consistently reports:

- five `TestSkillSave*` failures;
- the table and subtests under `TestPromptNextStatusAnswersFromTheCommandStream`,
  plus the non-TTY-default and unknown-answer tests;
- four connect subtests under `TestPromptSiteBaseline`;
- `TestSanctionedBreaksAreDocumented`;
- `TestFieldDescriptorStillHasExactlyTwoConsumers`;
- the `internal/cli` package failure.

Representative errors:

```text
runSkillSave: skill save requires an attached terminal
answer "2\n" => "delivering", want "in-review"
baseline drift ... want "interactive connect requires an attached terminal" ... got "repository is required"
read ../../docs/release-notes/breaking-changes.md: no such file or directory
collectFields consumers = [], want exactly [connect.go,skill.go]
```

The complete isolated package run outside the sandbox took about 79 seconds.
The same run inside the sandbox can stop earlier at callback/listener tests
because the sandbox denies local socket binds; those failures are environmental
and are not evidence about the CLI contract.

## Environment Details

- Branch: `codex/interactive-cli-input-scoped-completion`
- Worktree: `/private/tmp/hero-interactive-cli-scoped`
- Host: macOS, Go test temp paths under
  `/private/var/folders/.../T`
- `/private/tmp/.hero` and `/tmp/.hero` were absent during reproduction.
- The deterministic contract failures reproduce in an isolated
  `./internal/cli` invocation outside the sandbox.
- No tracker integration is configured.

---

## Root Cause Analysis

### Confirmed findings

1. **Skill save tests exercise the wrong stream class.**
   `internal/cli/skill.go:205-225` returns
   `skill save requires an attached terminal` before rendering a prompt when
   `prompt.IsInputTTY(cmd.InOrStdin())` is false. The failing tests at
   `prompt_adoption_test.go:271-405` use `newStreamCmd`, whose input is a
   `strings.Reader`, and assert the predecessor's piped-form behavior. The
   successor prompt spec requires missing ordinary input to fail promptly on a
   non-TTY and drive real terminal prompts through Cobra streams.

2. **Handoff tests exercise the wrong stream class.**
   `internal/cli/handoff.go:368-373` intentionally returns `delivering` without
   writing a menu when input is non-terminal. Tests at
   `prompt_adoption_test.go:515-594` use `newStreamCmd` yet expect the menu and
   parse aliases. The existing PTY test proves terminal answers still work.

3. **Connect fixtures predate the setup child's sanctioned correction.**
   `interactive-setup-and-connect-closure` AC-3 requires the provider's first
   missing-field error before output or read. `connect.go:595-607` implements
   that with `firstMissingConnectField`; focused tests in
   `connect_writer_unification_test.go:341-383` already expect
   `repository is required`. The four golden files still contain the prompt
   child's generic pre-setup error and pre-setup `--project` help.

4. **The descriptor assertion is the opposite of the approved boundary.**
   `selector_test.go:1730-1754` expects `connect.go,skill.go` to call generic
   `collectFields`. No `promptfield.go` exists and there are zero consumers.
   The setup spec's Context, Kickoff, Approach, and Boundaries explicitly say
   not to port generic promptfield/skill coupling; connect owns a private
   `collectConnectFields` collector instead.

5. **The release-note dependency was ported without its surface or requirement.**
   `prompt_policy_test.go:208-235` reads a file that exists only on
   `design/interactive-cli-input`. `main` and the successor have no
   `docs/release-notes/` directory. The setup spec says to update that surface
   only if the successor carries it from `main`, and the recomposed prompt spec
   has 13 criteria rather than the donor test's cited AC-14.

6. **The final validation record is not trustworthy.**
   The contradictory prompt tests entered at `2767452`, the descriptor test at
   `268214c`, and the setup production contract at `73fac3a`. The final evidence
   commit `6870da2` and later Hero-only commits contain no repair. Therefore the
   recorded `go test -count=1 -timeout 10m ./... | PASS` result cannot describe
   the checked-in tree.

### Why the stray-workspace diagnosis is not the current cause

The retained donor enhancement correctly describes a separate latent hazard:
`workspace.LocateFromCWD()` calls unbounded `Locate(cwd)`, and
`newTestEnvEmpty` gives the walk no stop boundary. However:

- no `/private/tmp/.hero` or `/tmp/.hero` existed during this reproduction;
- direct function tests fail without workspace lookup being involved;
- the docs and source-inspection tests fail without executing a CLI command;
- the same fixed failure set reproduces in the isolated package outside the
  sandbox.

Materializing that enhancement later remains worthwhile, but it cannot make
these assertions correct and must not be used as the merge-blocker fix.

### Root-cause classification

`design`: the initiative lacked a cross-child test-contract reconciliation
rule, allowing later approved behavior to contradict earlier child assertions.
Stale test code is the symptom. Severity is `high` because the full suite is red
and the closing evidence falsely says it is green; no released user data or
security boundary is currently at risk because the branch has not merged.

---

## Code Flow (End to End)

1. `internal/cli/prompt_adoption_test.go:271` constructs a non-terminal Cobra
   stream and calls `runSkillSave`.
2. `internal/cli/skill.go:205` resolves the workspace and creates the skills
   directory.
3. `internal/cli/skill.go:223` classifies the Cobra input with
   `prompt.IsInputTTY`; the reader is not a terminal.
4. `internal/cli/skill.go:224` returns the approved non-TTY error before either
   label, contradicting the predecessor assertion.
5. `internal/cli/prompt_adoption_test.go:515` similarly sends aliases through a
   non-terminal stream to `promptNextStatus`.
6. `internal/cli/handoff.go:369` classifies the stream and returns the
   `delivering` default without rendering or reading; alias assertions therefore
   cannot observe `in-review`.
7. `internal/cli/prompt_baseline_test.go:263` builds the current binary and
   launches each connect case in an isolated subprocess.
8. `internal/cli/connect.go:595` invokes `firstMissingConnectField` before any
   output/read on a non-TTY.
9. `internal/cli/connect.go:641` returns `repository is required`; the four
   predecessor golden files expect the superseded generic gate and old help.
10. `internal/cli/selector_test.go:1730` scans production `.go` files for a
    generic `collectFields(` call and finds none, which is the approved design
    but the test treats it as failure.
11. `internal/cli/prompt_policy_test.go:215` tries to read the intentionally
    absent donor-only release-note path and fails before checking content.
12. Go aggregates these deterministic failures and returns a failing
    `internal/cli` package, so the repository-wide suite cannot pass.

---

## Key Files

### Delivered prompt behavior

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/skill.go` | 205-245 | Enforces terminal-only `skill save` and strict terminal line reads |
| `internal/cli/handoff.go` | 357-397 | Defaults silently on non-TTY and parses choices only at a terminal |
| `internal/cli/connect.go` | 483-660 | Private connect field collector and provider-specific pre-I/O missing-field error |

### Contradictory tests and evidence

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/prompt_adoption_test.go` | 265-405, 515-594 | Asserts predecessor stream behavior after terminal gates landed |
| `internal/cli/prompt_baseline_test.go` | 263-326 | Compares current binary to stale connect goldens |
| `internal/cli/selector_test.go` | 1701-1754 | Requires rejected generic descriptor coupling |
| `internal/cli/prompt_policy_test.go` | 208-235 | References absent donor-only docs and stale AC numbering |
| `.hero/specs/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md` | Validation record | Claims the failing tree passed the full suite |

### Separate latent workspace-isolation issue

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/workspace/locate.go` | 56-70, 85-163 | Has `WithStopAt` but `LocateFromCWD` does not use a test boundary |
| `internal/cli/helpers_test.go` | 26-110 | Changes process cwd to temp dirs without bounding parent discovery |
| `design/interactive-cli-input:.hero/planning/features/cli-test-isolation-stray-workspace-boundary/spec.md` | full file | Accurate donor diagnosis of a different environment-dependent failure mode |

---

## Secondary Defects

1. **Unsupported validation evidence.** The final gate's PASS record and
   completion status were accepted despite deterministic failures already in
   the tree. The repair must append a correction and fresh command evidence; it
   must not silently rewrite the historical claim.
2. **Sandbox-only listener failures obscure useful output.** Callback and HTTP
   listener tests fail when local binds are prohibited. Those tests pass when
   run outside the sandbox and should remain part of the full suite; do not
   weaken or skip them in production CI to accommodate this harness.
3. **Unbounded test workspace discovery remains latent.** The donor enhancement
   is valid but not causal here. It should be materialized separately after the
   merge blocker is resolved.

---

## Load-Bearing Claims

- **read:** `runSkillSave` rejects non-TTY input before rendering either field.
- **read:** `promptNextStatus` silently returns `delivering` on non-TTY input.
- **read:** setup AC-3 owns connect's provider-specific pre-I/O failure.
- **read:** the successor explicitly rejects generic promptfield/skill coupling.
- **read:** neither `main` nor the successor contains `docs/release-notes/`.
- **read:** `/private/tmp/.hero` and `/tmp/.hero` were absent during reproduction.
- **read:** the isolated package outside the sandbox produces the deterministic
  failure list above.
- **assumed:** remote CI permits loopback listener binds; verify this through the
  normal CI run rather than inferring it from the desktop sandbox.

---

## Suggested Fix Approach

The project anchor reports no tripwire against this test-only reconciliation.
No harness-generated content is involved. The fix must preserve all six-target
behavior already delivered and must not reintroduce a Claude-only surface.

### 1. Reconcile terminal-policy tests in `internal/cli/prompt_adoption_test.go`

**Block:** the `TestSkillSave*` and `TestPromptNextStatus*` groups.

**Before (current non-TTY stream asserted as a successful form):**

```go
cmd, out := newStreamCmd("adopted-skill\nDeliberately Not The Slug\n")
if err := runSkillSave(cmd, nil); err != nil {
	t.Fatalf("runSkillSave: %v", err)
}
```

**After (successful form uses the existing real-PTY helper):**

```go
cmd, out := newPTYStreamCmd(t, "adopted-skill\nDeliberately Not The Slug\n")
if err := runSkillSave(cmd, nil); err != nil {
	t.Fatalf("runSkillSave: %v", err)
}
```

Move the empty/unterminated field cases that are still meaningful onto a PTY.
Replace the old piped-form assertion with the actual non-TTY contract:

```go
cmd, out := newStreamCmd("adopted-skill\nDeliberately Not The Slug\n")
err := runSkillSave(cmd, nil)
if err == nil || err.Error() != "skill save requires an attached terminal" {
	t.Fatalf("error = %v, want the attached-terminal refusal", err)
}
if out.String() != "" {
	t.Fatalf("non-TTY skill save prompted: %q", out.String())
}
```

For `promptNextStatus`, run alias/unknown-answer parsing through
`newPTYStreamCmd`; keep a separate `newStreamCmd` case that expects
`StatusDelivering` and empty output regardless of bytes waiting on the pipe.

**Why:** this tests both sides of the delivered policy instead of rolling
production back to insecure/blocking stream prompts.

### 2. Refresh only the four superseded connect golden fixtures

**Files:**

- `internal/cli/testdata/prompt_baseline/connect_github_repo_prompt.closed.txt`
- `internal/cli/testdata/prompt_baseline/connect_github_repo_prompt.pipe.txt`
- `internal/cli/testdata/prompt_baseline/connect_github_secret_prompt.closed.txt`
- `internal/cli/testdata/prompt_baseline/connect_github_secret_prompt.pipe.txt`

**Before:**

```text
Error: interactive connect requires an attached terminal; supply --integration-id, --project, and --token-stdin for automation
--project string          project identifier (used with --remove)
```

**After:**

```text
Error: repository is required
--project string          project identifier (required for flag-driven connect; used with --remove)
```

Regenerate only these four cases with the focused `-update-baseline` command,
then inspect the complete diff. Do not bulk-regenerate unrelated fixtures.

**Why:** setup AC-3 intentionally superseded the prompt child's generic error,
and the help text is an explicit setup-child deliverable.

### 3. Make the descriptor guard enforce the successor boundary

**File/block:** `internal/cli/selector_test.go`,
`TestFieldDescriptorStillHasExactlyTwoConsumers`.

**Before:**

```go
want := "connect.go,skill.go"
if strings.Join(consumers, ",") != want {
	t.Errorf("collectFields consumers = %v, want exactly [%s]. The descriptor is capped at two; "+
		"a third is the hard-stop drift alarm for this initiative.", consumers, want)
}
```

**After:**

```go
if len(consumers) != 0 {
	t.Errorf("generic collectFields consumers = %v, want none; connect owns collectConnectFields and skill prompts directly", consumers)
}
if _, err := os.Stat("promptfield.go"); !os.IsNotExist(err) {
	t.Errorf("promptfield.go must remain absent; stat error = %v", err)
}
```

Rename the test to describe the zero-generic-descriptor contract and update its
comments. Keep the existing selector direct-primitive guard.

**Why:** the current assertion enforces the donor design that the successor
explicitly rejected.

### 4. Remove the invalid donor-only release-note dependency

**File/block:** `internal/cli/prompt_policy_test.go`,
`TestSanctionedBreaksAreDocumented`.

**Before:**

```go
path := filepath.Join("..", "..", "docs", "release-notes", "breaking-changes.md")
notes, err := os.ReadFile(path)
if err != nil {
	t.Fatalf("read %s: %v", path, err)
}
```

**After:** delete `TestSanctionedBreaksAreDocumented`; no replacement source
inspection test is needed. The positive behavior tests in
`prompt_sanctioned_breaks_test.go` and setup help tests remain the executable
contract. If Hero adopts a canonical release-note surface later, design that
surface separately and add coverage there rather than resurrecting a donor-only
path.

**Why:** the successor spec conditionally excluded this absent surface, and the
test's cited AC-14 no longer exists. Copying the donor's 330-line changelog
would import unrelated release history solely to satisfy a stale test.

### 5. Correct the closing evidence without rewriting history

**Files/blocks:**

- `.hero/specs/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md`
- this bug's eventual Completion Ledger and delivery audit

**Before:**

```markdown
| go test -count=1 -timeout 10m ./... | PASS — full repository suite. |
```

**After:** append a dated correction to the archived evidence identifying this
bug, the invalid prior result, and the fresh passing command/commit after the
repair. Do not erase the historical row. The bug delivery's own ledger and
cold audit become the authoritative merge gate.

**Why:** a silently rewritten audit trail would hide the process failure that
allowed a red suite to be called green.

---

## Goal

Restore a truthful merge gate without changing shipped CLI behavior: all stale
prompt, connect-baseline, descriptor, and donor-doc assertions match the scoped
successor contract; the isolated CLI package and full repository suite pass
outside the network-restricted sandbox; and the archived acceptance record
contains an explicit correction backed by fresh command output.

## Changes

1. Update terminal-policy adoption tests in
   `internal/cli/prompt_adoption_test.go` to use real PTYs for interactive
   answers and to assert silent, prompt-free defaults/refusals on non-TTY input.
2. Regenerate and review only the four GitHub connect closed/pipe golden
   fixtures superseded by setup AC-3 and the new help contract.
3. Replace the selector child's two-generic-consumer assertion with a guard
   requiring no generic descriptor and preserving direct primitives/private
   connect collection.
4. Delete the donor-only release-note path assertion while retaining executable
   sanctioned-break and help coverage.
5. Append a dated correction to the archived acceptance evidence. Use this
   bug's normal cold-audit and `hero spec verify` workflow as the external
   closing gate rather than preclaiming those operations in the implementation.

## Acceptance Criteria

- **AC-1:** WHEN `skill save` is driven through a real Cobra input terminal THE SYSTEM SHALL prompt for and persist both fields, including empty and unterminated-input guards where applicable.
- **AC-2:** WHILE `skill save` input is non-terminal THE SYSTEM SHALL return the attached-terminal refusal without rendering a field prompt or writing a skill file.
- **AC-3:** WHEN `promptNextStatus` receives a terminal answer THE SYSTEM SHALL parse all supported aliases, and WHILE input is non-terminal THE SYSTEM SHALL return `delivering` without output or input consumption.
- **AC-4:** WHEN the four GitHub connect closed/pipe baseline cases run THE SYSTEM SHALL match the setup child's provider-first missing-field error and current help bytes without prompting or consuming the pipe.
- **AC-5:** THE SYSTEM SHALL have zero generic `collectFields` consumers and SHALL keep `promptfield.go` absent while retaining direct selector primitives and private connect collection.
- **AC-6:** THE SYSTEM SHALL NOT require a `docs/release-notes/` path that is absent from `main` and explicitly excluded by the scoped successor design.
- **AC-7:** WHEN `go test -count=1 -timeout 10m ./internal/cli` runs outside a network-restricted sandbox THE SYSTEM SHALL pass with no failing test nodes.
- **AC-8:** WHEN `go test -count=1 -timeout 10m ./...` runs in the normal development/CI environment THE SYSTEM SHALL pass, followed by affected race tests, vet, native build, and Windows build evidence.
- **AC-9:** THE SYSTEM SHALL append a dated correction to the false full-suite PASS record and SHALL preserve the historical row for auditability.
- **AC-10:** THE SYSTEM SHALL leave production `.go` files unchanged unless a newly failing behavior test proves a separate production defect and that defect is routed through a new diagnosis.

## Boundaries

- Do not remove or relax the `skill save`, handoff, connect, JSON, secret, or
  live-pipe terminal policies.
- Do not reintroduce `promptfield.go`, generic form/schema infrastructure, or
  skill/connect coupling.
- Do not bulk-regenerate prompt baselines.
- Do not import the donor's complete `breaking-changes.md`; this repository has
  no approved release-note surface on `main`.
- Do not weaken loopback-listener tests to pass inside the desktop sandbox; run
  them in the normal environment.
- Do not fold `cli-test-isolation-stray-workspace-boundary` into this repair.
  It is a valid separate hardening item, not the current cause.
- Do not merge or push until this bug is delivered, cold-audited, verified, and
  the full suite is freshly green.

## Risks

1. PTY capture is asynchronous. Use the existing `waitForOutput` helper where
   output is read from the PTY master; do not replace it with sleeps.
2. Golden regeneration can normalize an unintended production change. Limit
   `-update-baseline` to the four named connect cases and inspect every byte.
3. Deleting the stale docs test could accidentally delete behavioral coverage.
   Keep `prompt_sanctioned_breaks_test.go` and connect/help coverage green.
4. A sandbox-only bind failure can be misclassified as a product regression.
   Run final normal tests outside the sandbox and record both environment and
   command.
5. Another stale cross-child assertion may surface after these known groups are
   repaired. Treat any new failure as evidence, not as permission to bulk-update
   fixtures.

## Test Plan

### Existing test review

- `internal/cli/prompt_adoption_test.go` already has PTY helpers and passing
  positive terminal tests for both affected functions.
- `internal/cli/prompt_sanctioned_breaks_test.go` directly proves the intended
  password and install-target compatibility corrections.
- `internal/cli/connect_writer_unification_test.go` proves provider-first
  non-TTY failure for closed and live pipes.
- `internal/cli/prompt_setup_commands_test.go` proves help, interactive setup,
  JSON suppression, and private direct primitives.
- `internal/cli/prompt_baseline_test.go` provides the exact byte-level fixture
  harness and focused update flag.

### Test changes needed

1. Convert successful/validation `skill save` cases to PTY input; add one
   explicit non-TTY refusal/no-mutation case.
2. Convert handoff alias/error cases to PTY input; change the non-TTY test to
   require silent defaulting and non-consumption.
3. Refresh only the four named connect fixtures.
4. Change the descriptor source guard from exactly two generic consumers to
   zero and an absent `promptfield.go`.
5. Remove the stale donor-doc test, keeping behavior/help coverage.

### Regression scope

- Prompt rendering and stream ownership across Unix PTYs.
- Non-TTY liveness and no-read guarantees.
- Connect help and provider-specific missing-field errors.
- Selector/setup architecture boundaries.
- Windows prompt build compatibility.
- Full repository tests, including listener tests in an environment that
  permits loopback binds.

## Validation

Run in this order:

```bash
go test -count=1 ./internal/cli -run 'TestSkillSave|TestPromptNextStatus|TestPromptSiteBaseline/connect_github_(repo|secret)_prompt|Test.*Generic.*Descriptor|TestSanctioned'
go test -count=1 -timeout 10m ./internal/cli
go test -race -count=1 ./internal/cli ./internal/cli/prompt
go test -count=1 -timeout 10m ./...
go vet ./...
go build ./...
GOOS=windows GOARCH=amd64 go build ./...
git diff --check
hero spec lint cli-successor-test-contract-reconciliation
hero spec score cli-successor-test-contract-reconciliation
```

The full package/repository and listener-bearing checks must run outside the
network-restricted desktop sandbox. Record actual output in the bug Completion
Ledger before the cold audit. Then run the audit and
`hero spec verify cli-successor-test-contract-reconciliation`; do not merge on
a skip-tests verification.

## Notes

The donor stray-workspace spec remains a sound follow-up. Its proposed
`HERO_WORKSPACE_BOUNDARY` mechanism should be re-reviewed as an independent
enhancement after this blocker is closed, because it changes workspace-location
plumbing and test harness behavior beyond the present test-contract repair.

---

## Recap

The branch is blocked by stale and incompletely ported tests, not by broken
interactive production behavior and not by a current stray `/private/tmp/.hero`.
Reconcile those tests to the successor specs, correct the unsupported validation
record, and require a fresh full green run before merge.

## Kickoff

Reconciles stale interactive-CLI tests with the scoped successor contract; production prompt behavior stays unchanged.

**Status:** completed — Hero verification passed every gate, including the
non-skipped build and 103-package test run, with no production change.

**Pick up at:** no delivery work remains; this archived repair is the merge
evidence for the scoped interactive CLI successor.

→ `.hero/specs/cli-successor-test-contract-reconciliation/spec.md`

**Files:** `internal/cli/prompt_adoption_test.go`, `internal/cli/prompt_baseline_test.go`, `internal/cli/selector_test.go`, `internal/cli/prompt_policy_test.go`
**Skip:** production rollback, generic promptfield resurrection, bulk golden regeneration, and stray-workspace hardening.

## Completion Ledger

Stack detected: Go. The repair changes tests, four generated golden fixtures,
and archived evidence only; production `.go` files are unchanged.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Real-terminal skill save prompts and persists both fields | DONE | The positive, empty-name, and unterminated-name/title cases use real PTYs; `TestSkillSaveUnterminatedTerminalInputWritesNothing` also proves terminal EOF leaves no file. |
| 2 | Non-terminal skill save refuses silently without mutation | DONE | `TestSkillSaveNonTTYFailsFastAndWritesNothing` proves the exact refusal, no output, no input consumption, and no file. |
| 3 | Terminal handoff aliases parse; non-TTY defaults silently | DONE | `TestPromptNextStatusAnswersAtATerminal`, the PTY error test, and the non-TTY no-read test pass. |
| 4 | Four GitHub connect baselines match setup AC-3 | DONE | Only the named repo/secret closed/pipe fixtures were regenerated and pass focused byte comparisons. |
| 5 | Generic descriptor remains absent | DONE | `TestGenericFieldDescriptorRemainsAbsent` requires zero `collectFields` consumers and no `promptfield.go`; direct selector/setup guards remain. |
| 6 | No absent donor-only release-note dependency | DONE | The stale docs-path assertion is removed; sanctioned behavior and help tests remain and pass. |
| 7 | Isolated CLI package passes | DONE | `go test -count=1 -timeout 10m ./internal/cli` passed outside the network sandbox in 83.292s after the terminal EOF assertions were added. |
| 8 | Full suite, race, vet, native, and Windows gates pass | DONE | After `d67b06c`, the full suite passed with `internal/cli` in 131.405s; affected race, vet, native build, Windows build, and Windows prompt test compile also exited zero. |
| 9 | Archived false PASS has an explicit correction | DONE | `acceptance-evidence.md` retains the historical row and appends the dated diagnosis plus fresh results. |
| 10 | Production Go files remain unchanged | DONE | The repair touches `_test.go`, four text fixtures, Hero spec/evidence, and generated projections only. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Reconcile skill-save and handoff terminal tests | DONE | PTY and non-TTY tests now exercise the delivered stream classes and focused tests pass. |
| 2 | Refresh four superseded connect fixtures | DONE | Focused generation changed only the four named golden files; their complete diff was reviewed. |
| 3 | Enforce zero generic descriptor | DONE | Stale two-consumer assertions/comments now encode private connect collection and direct primitives. |
| 4 | Remove invalid donor release-note assertion | DONE | The nonexistent, excluded docs surface is no longer a test dependency; behavioral coverage remains. |
| 5 | Append a dated correction to closing evidence | DONE | Archived evidence preserves the unsupported historical row and records the fresh green commands from this repair. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised through real PTYs, non-terminal
  streams with unread-byte assertions, focused subprocess goldens, the complete
  CLI package, and the full repository suite.

### Excellence Bar self-check

- [x] The repair makes the merge evidence truthful without changing approved
  production behavior, bulk-normalizing fixtures, or reviving rejected
  abstractions.

### Delivered artifacts

- `internal/cli/prompt_adoption_test.go` — reconciled terminal/non-terminal contracts.
- `internal/cli/prompt_policy_test.go` — removed the invalid donor-doc dependency.
- `internal/cli/prompt_setup_commands_test.go` and `internal/cli/selector_test.go` — corrected architecture guards.
- `internal/cli/testdata/prompt_baseline/connect_github_*` — four scoped golden updates.
- `.hero/specs/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md` — explicit historical correction and fresh validation evidence.
