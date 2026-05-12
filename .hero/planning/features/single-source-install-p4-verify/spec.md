---
title: "Single-Source Install P4 — hero verify-install for ongoing drift detection"
type: feature
status: completed
status_verified: "2026-05-12 by go test ./internal/install/... -count=1 — 6 new TestVerify_* tests pass. Dogfooded on hero (caught .opencode/ rendered drift before P3 ran on it) + example codebase (post-migrate: ✓ no issues found)."
priority: P1
tags: [install, verify, drift, audit, health, ci]
created: 2026-05-12
relations:
  - target: single-source-install
    kind: parent
  - target: single-source-install-p3-migrate
    kind: follows
horizon: now
---

## Goal

A read-only command — `hero verify-install [path]` — that audits the
install state on disk and reports drift, broken symlinks, security
issues, and unexpected layouts. Designed to run on demand, on CI,
and inside `hero check`. Required for rendered-copy mode (Windows
without symlinks, Cline) to be safe: drift becomes visible and
fixable instead of silently corrosive.

## Problem

Once `hero install` lands, install state can rot:

- Rendered-copy mode (Windows, Cline) doesn't auto-update — a user
  edits a file in the harness dir while canonical evolves elsewhere.
- Symlinks can break if canonical moves or is removed.
- User-modified symlinks could point at arbitrary filesystem
  locations (security boundary if a malicious config slips in).
- Legacy state (regular directories left over from pre-P2 installs)
  produces drift risk.
- Mixed-mode installs across targets are easy to introduce
  accidentally (some symlinked, some rendered).

Hero needs an inspection tool to surface these conditions without
making changes — so users can see the state, decide what to do, and
either run `--migrate` or live with it explicitly.

## Design

### Command surface

```
hero verify-install                 # audits the cwd
hero verify-install <path>          # audits a specific project root
```

Exit codes:
- `0` no errors (warnings and info are allowed; user decides
  whether they're worth acting on)
- `1` one or more error-severity issues — install needs attention
- `2` argument/usage error

Read-only by contract — no flag exists to mutate state. Recovery
runs through `hero install --migrate` or `--force`.

### Check categories

| code | severity | what it means |
|---|---|---|
| `broken_symlink` | error | symlink target doesn't exist |
| `symlink_escape` | error | symlink target resolves outside the project root |
| `missing_canonical` | error | configured external content path doesn't exist on disk |
| `unexpected_file_type` | error | content dir is neither a directory nor symlink |
| `stat_failed` | error | filesystem error reading a content dir |
| `expected_symlink` | warning | harness content dir is a regular dir; P2 default is symlink-to-canonical |
| `drifted_rendered` | warning | rendered file differs from canonical (per-file, with hash diff) |
| `wrong_symlink_target` | warning | symlink points somewhere other than the resolved canonical |
| `mixed_mode` | info | symlink and rendered modes coexist across targets |

### Output

Human-readable text by default. Issues grouped by severity (errors
first, then warnings, then info); per-issue lines include code,
path, message, and an optional multi-line `detail` block with
remediation hints.

A `✓ no issues found.` summary when everything is clean. An
explicit "Result:" footer pointing at `hero install --migrate`
when errors are present.

### Integration points (deferred to follow-on work)

- `hero check`: include a verify-install pass in the workspace
  health check. Surfacing drift inline with other check signals.
- JSON output (`--format json`) for a Hero-native client consumption (P5).
- `--strict` mode that exits non-zero on warnings, for CI use.

These are explicit out-of-scope for P4 — easy to add later, not
needed for the initial command to deliver value.

## Acceptance Criteria

- WHEN `hero verify-install <path>` runs against a freshly-installed
  P2-symlink project THE SYSTEM SHALL report `✓ no issues found.`
  and exit 0
- WHEN a harness content directory exists as a regular dir (legacy
  pre-P2 install) THE SYSTEM SHALL report an `expected_symlink`
  warning identifying the dir and pointing the user at
  `hero install --migrate`
- WHEN a content-dir symlink points at a non-existent target THE
  SYSTEM SHALL report a `broken_symlink` error and exit 1
- WHEN a content-dir symlink target resolves outside the project
  root THE SYSTEM SHALL report a `symlink_escape` error and exit 1
- WHEN rendered-mode files differ from their canonical counterparts
  THE SYSTEM SHALL report per-file `drifted_rendered` warnings,
  each including the short hash of both copies for diagnostics
- WHEN a rendered-mode file has no counterpart in canonical at all
  THE SYSTEM SHALL report a `drifted_rendered` warning noting the
  orphan
- WHEN the configured external content path (from hero.json's
  `content.<kind>_path`) doesn't exist on disk THE SYSTEM SHALL
  report a `missing_canonical` error and exit 1
- WHEN harness install modes differ across detected targets THE
  SYSTEM SHALL report a `mixed_mode` info note (not error, not
  warning)
- THE SYSTEM SHALL make zero filesystem modifications during verify
  — read-only by contract
- THE SYSTEM SHALL render a `Result:` footer pointing at the
  recovery command (`hero install --migrate`) whenever any
  error-severity issue is reported

## Changes

- `internal/install/verify.go` (new) — RunVerify entry point,
  VerificationReport / VerificationIssue types, per-check helpers
  (verifyContentDir, verifySymlink, verifyRenderedDir,
  compareRenderedAgainstCanonical), StringReport renderer
- `internal/cli/verify_install.go` (new) — `verify-install` cobra
  command, hooked into rootCmd; exits 1 on errors
- `internal/install/verify_test.go` (new) — six tests covering
  clean symlink install, regular-dir reports
  expected_symlink, broken-symlink reports error, symlink-escape
  reports error, drifted rendered files report per-file, no
  targets detected returns clean

## Boundaries

- **Not in scope:** writing changes from verify. Verify is
  inspection-only; recovery is `hero install --migrate`.
- **Not in scope:** JSON / NDJSON output. Hero-code's UI
  orchestration story is P5; verify gets JSON output then.
- **Not in scope:** `--strict` warnings-as-errors mode. Easy
  follow-on if real CI usage demands it.
- **Not in scope:** auto-fix. Same rationale: orthogonal command,
  composable with verify.
- **Not in scope:** inclusion in `hero check`. Will be added as a
  small follow-on once verify's output shape is settled in
  practice.

## Mission Fit

> "Does this make the next agent session start smarter than the
> last one ended — and does it raise the floor for everyone?"

Yes. Rendered-mode and mixed-mode installs are silent corrosion
risks: an agent reading from a stale rendered copy ends a session
having delivered against the wrong content. Verify makes that
condition visible, with actionable next steps. The floor rises for
anyone on a platform where symlinks are restricted (Windows mixed
teams, certain CI environments) or running a symlink-broken harness
(Cline) — they get the same correctness guarantees as symlink-mode
users, via active drift detection.
