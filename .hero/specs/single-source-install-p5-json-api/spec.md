---
title: "Single-Source Install P5 — JSON ops API for programmatic consumers"
slug: single-source-install-p5-json-api
type: feature
status: completed
status_verified: "2026-05-12 by go test ./internal/install/... — JSON-output round-trip stability tests pass. Dogfooded: hero install/install --migrate/verify-install all emit clean structured JSON (no stdout leaks even from deep helpers — silenceStdout belt-and-suspenders covers it)."
priority: P1
tags: [install, json, api, a Hero-native client, ops, consumer]
created: 2026-05-12
relations:
  - target: single-source-install
    kind: parent
  - target: single-source-install-p4-verify
    kind: follows
  - target: bundle-hero-binary
    kind: enables
horizon: now
---

## Goal

Hero install, migrate, and verify-install all gain a `--json` flag
that emits a single structured JSON object on stdout instead of
human-readable text. The shape is stable, snake_case, and additive-
evolvable — a public ops API surface that a Hero-native client (and any future
Hero consumer that subprocess-invokes Hero) can parse without
scraping text.

## Problem

The bundle-hero-binary work in a Hero-native client positions a Hero-native client as the
operations UI for Hero: user clicks "Set up Hero," a Hero-native client
subprocess-invokes the bundled hero binary, parses the result, shows
progress + outcome. That promise requires hero to produce machine-
readable output. Text output works for humans but is fragile under
programmatic parsing (formatting changes break consumers).

Without JSON output, a Hero-native client would need to scrape stdout, parse
indented "agents -> .claude/agents (symlink -> ../.hero/agents)"
lines, and reconstruct semantics from prose. That's the wrong
boundary. The right boundary: hero exposes a structured ops API;
consumers build UI on top.

## Design

### Output shape — install

```json
{
  "target": "claude",
  "mode": "project",
  "target_dir": "/path/to/project",
  "hero_version": "v0.7.1",
  "result": {
    "copied": [
      {"source": "symlink->../.hero/agents", "dest": "/path/.claude/agents"}
    ],
    "merged": ["AGENTS.md", "CLAUDE.md"],
    "skipped": []
  },
  "duration_ms": 87
}
```

On failure, `error` is populated:

```json
{
  "target": "claude",
  "mode": "project",
  "result": null,
  "duration_ms": 12,
  "error": {
    "code": "install_failed",
    "message": "..."
  }
}
```

### Output shape — migrate

```json
{
  "report": {
    "detected_targets": ["claude", "opencode"],
    "promoted_files": {
      "agents": ["/path/.hero/agents/x.md"],
      "commands": [],
      "skills": []
    },
    "conflicts": [
      {
        "kind": "agents",
        "file": "engineer.md",
        "winner": "/path/.claude/agents/engineer.md",
        "candidates": [...]
      }
    ],
    "skipped_targets": [],
    "target_results": {
      "claude": {"copied": [...], "merged": [...], "skipped": []}
    },
    "dry_run": false,
    "errors": []
  },
  "duration_ms": 142
}
```

### Output shape — verify-install

```json
{
  "report": {
    "target_dir": "/path/to/project",
    "detected_targets": ["claude", "opencode"],
    "issues": [
      {
        "severity": "warning",
        "code": "expected_symlink",
        "path": ".claude/agents",
        "message": "regular dir; expected symlink",
        "detail": "re-run hero install --migrate to recover"
      }
    ],
    "clean": false
  },
  "duration_ms": 5
}
```

### Stdout cleanliness — belt-and-suspenders

Two layers ensure --json output has no leaked progress text:

1. **opts.Quiet** suppresses per-file `progressf` calls inside the
   install package at most progress-print sites.

2. **silenceStdout** wraps the install/migrate calls at the CLI
   layer, redirecting os.Stdout to a discarded pipe for the
   duration of the operation. Catches any prints that escape the
   Quiet gate (e.g., from helpers that don't have opts in scope).

After the operation returns, the original stdout is restored and
the JSON payload is the single thing written.

### Stable schema commitments

Field names are snake_case across all JSON payloads. The set of top-
level fields per output is the public contract:

- **install**: target, mode, target_dir, hero_version, result,
  duration_ms, error?
- **migrate**: report, duration_ms, error?
- **verify**: report, duration_ms, error?

Additive evolution is allowed (new optional fields). Removing or
renaming an existing field is a breaking change and would require a
new schema version. A round-trip stability test
(`json_output_test.go`) fails on any of these as a tripwire.

### Error codes

The JSONError.code field is a stable machine-readable string:

- `install_failed` — top-level install error
- `migrate_failed` — top-level migrate error
- `verify_failed` — top-level verify error

(Granular per-step codes can be added later as needs surface; the
top-level "something failed" suffices for the first release. The
human message in `error.message` carries the specific details.)

### Exit codes

- `0` success (warnings in verify still produce exit 0)
- `1` operation error (install failed, verify found error-severity
  issues)
- `2` argument/usage error

## Acceptance Criteria

- WHEN `hero install --json ...` runs successfully THE SYSTEM SHALL
  emit a single InstallJSONOutput JSON object on stdout containing
  target, mode, target_dir, hero_version, result, duration_ms; the
  `error` field SHALL be absent (omitempty)
- WHEN `hero install --json ...` fails THE SYSTEM SHALL emit the
  same JSON shape with `error.code = "install_failed"` and
  `error.message` populated, AND propagate a non-zero exit status
- WHEN `hero install --migrate --json ...` runs THE SYSTEM SHALL
  emit a MigrateJSONOutput on stdout with the full MigrationReport
  embedded plus duration_ms; under `--dry-run`, the report's
  `dry_run` field SHALL be true
- WHEN `hero verify-install --json ...` runs THE SYSTEM SHALL emit
  a VerifyJSONOutput on stdout with the full VerificationReport
  embedded; clean and dirty states are both represented in the
  `report.clean` boolean
- THE SYSTEM SHALL ensure stdout in `--json` mode contains ONLY the
  JSON payload (no progress text, no warnings, no header lines) so
  programmatic consumers can parse stdout directly with no
  pre-filtering
- THE SYSTEM SHALL use snake_case field names throughout the JSON
  output for consistency across the public API surface
- THE SYSTEM SHALL stamp `duration_ms` in every JSON output so
  consumers can show timing info / detect long-running ops

## Changes

- `internal/install/json_output.go` (new): InstallJSONOutput,
  MigrateJSONOutput, VerifyJSONOutput envelopes; JSONError;
  NewJSONError helper.
- `internal/install/install.go`: Options.Quiet field for
  progress-print suppression.
- `internal/install/log.go` (new): progressf helper that respects
  Options.Quiet.
- `internal/install/{linking,files,agents_md,target_opencode,
  claude_hooks,codex_hooks}.go`: progress-print sites routed
  through progressf.
- `internal/install/migrate.go`: json tags on MigrationReport,
  MigrationConflict, MigrationCandidate; field names snake_case.
- `internal/install/verify.go`: json tags on VerificationReport,
  VerificationIssue.
- `internal/install/install.go` Result, CopyAction: json tags
  added.
- `internal/cli/install.go`: emitJSON + silenceStdout helpers;
  `--json` flag; install + migrate paths emit JSON when set.
- `internal/cli/verify_install.go`: `--json` flag; emits
  VerifyJSONOutput.
- `internal/install/json_output_test.go` (new): round-trip
  stability tests — assert every documented field appears with
  the documented key name. These are the breaking-change
  tripwires.

## Boundaries

- **Not in scope:** NDJSON progress streaming. Most installs
  complete in well under a second; final-result output is
  sufficient for the bundle-hero-binary UI orchestration. If real
  use surfaces "I need to show a progress bar for a 30s install,"
  we add streaming then.
- **Not in scope:** YAML / other output formats. `--json` is the
  one structured format; `--format` can be generalized later if
  needed.
- **Not in scope:** machine-readable error CODE enumeration beyond
  the top-level "install_failed/migrate_failed/verify_failed". Per-
  step error codes can be added incrementally as consumers need to
  distinguish them.
- **Not in scope:** `hero status --json` or other commands. P5 is
  the install/migrate/verify trio. Other commands can adopt the
  same pattern (emitJSON + silenceStdout) when needed.
- **Not in scope:** breaking-change versioning of the JSON schema.
  First release is implicitly v1; if a future change requires
  breaking, we add a `schema_version` field to all outputs and bump
  it.

## Mission Fit

> "Does this make the next agent session start smarter than the
> last one ended — and does it raise the floor for everyone?"

Yes, indirectly. P5 itself is plumbing — JSON ops API for
programmatic consumers. But it unlocks the a Hero-native client UI work
(spec: `../a Hero-native client/.hero/planning/features/bundle-hero-binary/`)
which is the user-facing payoff: a polished native app where
"Set up Hero" is a button, not a CLI command. The floor rises for
every user who'd otherwise need to memorize Hero's CLI vocabulary
to take advantage of its workflow.
