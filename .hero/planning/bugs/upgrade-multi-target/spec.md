---
title: hero upgrade only touches one tool when multiple are installed
type: bug
status: completed
priority: P1
severity: medium
tags: [upgrade, multi-target, install]
created: 2026-04-29
relations:
  - target: get-back-on-track
    kind: parent
horizon: now
smoke: deferred
---

## Captured

User report (2026-04-29):

> "i did find a bug on another project here - i have opencode and
> claude files using both - and hero upgrade only touched the
> opencode - no way to pick which to upgrade and upgrade doesn't do
> both."

## Root cause

[`internal/cli/upgrade.go:213`](../../../../internal/cli/upgrade.go:213)
`detectInstalledTarget` returns a single `install.Target` via an
ordered if/else over filesystem probes:

```go
if _, err := os.Stat(filepath.Join(projectRoot, ".opencode")); err == nil {
    return install.TargetOpenCode
}
if _, err := os.Stat(filepath.Join(projectRoot, ".cursor")); err == nil {
    return install.TargetCursor
}
if _, err := os.Stat(filepath.Join(projectRoot, ".claude")); err == nil {
    return install.TargetClaude
}
```

The first match wins. A repo with both `.opencode/` and `.claude/`
gets opencode-only upgrade. There's no `--target` flag and no loop
over multiple detected targets.

## Fix design

1. Replace `detectInstalledTarget(...) Target` with
   `detectInstalledTargets(...) []Target` that returns ALL hits.
2. Default `hero upgrade` walks every detected target and upgrades
   each in turn. The summary aggregates counts across targets.
3. Add `--target <name>` flag (multi-valued) so users can opt into
   single-target upgrades when needed (e.g. only refresh opencode
   without touching the in-flight claude install).
4. `version.json`'s `LastInstall` could grow into a list (or a map
   keyed by target) so the version-stamp signal stays accurate when
   multiple targets are installed.

## Acceptance criteria

**AC-1:** `hero upgrade` on a repo with both `.opencode/` and
`.claude/` upgrades both and reports separate counts per target.

**AC-2:** `hero upgrade --target claude` upgrades only the claude
target, leaving `.opencode/` untouched. Verified by checksum
comparison.

**AC-3:** `hero upgrade --target opencode --target claude` accepts
multiple values and upgrades both (the same as the default for the
two-installed case, just spelled explicitly).

**AC-4:** `hero upgrade --dry-run` lists changes per target rather
than the current single-target plan. No filesystem writes.

**AC-5:** Backward compat: `version.json` from a single-target
install still parses; the upgrade detects the existing target
correctly when only one is installed.

## Out of scope

- Changing `hero install` to install multiple targets in one shot
  (that's a separate bug if requested — distinct from upgrade).
- Cross-target customization-detection (a customized agent file in
  `.opencode/` shouldn't influence `.claude/` upgrade decisions —
  per-target is fine).

## Open questions

- When `--target` is omitted and multiple are installed: silently
  upgrade all, or print "detected: opencode, claude. Run with
  `--target` to narrow." Lean: silent default = all. Power-user
  flag for narrowing.
