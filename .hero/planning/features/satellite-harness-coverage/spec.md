---
title: "Satellite Harness Coverage — Per-Target Markers for OpenCode, Cursor, Generic"
slug: satellite-harness-coverage
type: feature
status: planning
priority: medium
horizon: now
tags: [monorepo, satellites, harness-integration]
relations:
  - target: monorepo-satellite-installs
    kind: parent
created: 2026-05-12
---

# Satellite Harness Coverage — Per-Target Markers for OpenCode, Cursor, Generic

## Problem

The satellite materializer's `targetLayouts` registry — the table that says "for each harness, here's the subdir to symlink and the marker file to write" — only fully covers Claude (`.claude/` + `CLAUDE.md`) and Codex (`.codex/` + `AGENTS.md`). The rest are incomplete:

- **OpenCode** has its `.opencode/` subdir registered but `MarkerFile` is empty. OpenCode actually uses `AGENTS.md` at the project root the same way Codex does — so an OpenCode-only user opens a satellite chat, the symlinks resolve correctly, but no satellite marker file appears in the folder. The model has no in-context signal that it's in a satellite of a larger workspace, only that the agents/commands/skills exist.
- **Cursor** isn't in the registry at all. A Cursor-using developer opening a chat in a subfolder gets nothing — no symlinks, no marker. Cursor's content lives at `.cursor/rules/{agents,commands,skills}/`, which is the same shape the satellite mechanism handles for everyone else.
- **Generic** isn't in the registry either. The generic target uses `.ai/{agents,commands,skills}/` plus an `AGENTS.md` at the root — same structure as OpenCode/Codex, just under `.ai/` instead of `.opencode/` or `.codex/`. Anyone using Hero with a tool that doesn't have a dedicated target falls back to generic and gets no satellite support.
- **Copilot** is registered for symlinks (`.github/copilot/`) but with no marker. This is probably correct because GitHub Copilot's instruction file (`.github/copilot-instructions.md`) lives at the repo root and Copilot's behavior is org/repo-scoped rather than cwd-relative — so a per-folder marker has nowhere meaningful to attach. But this is a guess, and worth confirming rather than leaving silently.

The user-visible failure mode is silent partial coverage: OpenCode/Cursor/generic users hit the "your harness ought to know it's in a satellite" gap that the previous specs fixed for Claude/Codex.

## Goal

Close the registry gaps so every harness target Hero supports has the same satellite affordances:

- OpenCode and Generic write `AGENTS.md` as the per-folder marker (matching Codex's convention since they all converge on AGENTS.md).
- Cursor gets registered with `.cursor/rules/` as its subdir, no per-folder marker (Cursor's rules walk up from cwd, no convention for a satellite-scoped instruction file).
- Copilot's status is documented explicitly (registered for symlinks, no marker — by intent, not omission) so future review doesn't re-discover the gap and treat it as a bug.

**Mission-fit.** A teammate using OpenCode in a satellite-equipped monorepo deserves the same in-harness experience as a Claude Code user. Right now they don't get it. This is a small, mechanical correction — but without it, the satellite story is "Claude users only," which contradicts the "harness-agnostic Hero" framing.

## Design

### 1. Update `targetLayouts`

The current registry in `internal/install/satellite.go`:

```go
var targetLayouts = []TargetLayout{
    {Target: TargetClaude,   SubDir: ".claude",                            MarkerFile: "CLAUDE.md"},
    {Target: TargetCodex,    SubDir: ".codex",                             MarkerFile: "AGENTS.md"},
    {Target: TargetOpenCode, SubDir: ".opencode",                          MarkerFile: ""},
    {Target: TargetCopilot,  SubDir: filepath.Join(".github", "copilot"),  MarkerFile: ""},
}
```

Becomes:

```go
var targetLayouts = []TargetLayout{
    {Target: TargetClaude,   SubDir: ".claude",                            MarkerFile: "CLAUDE.md"},
    {Target: TargetCodex,    SubDir: ".codex",                             MarkerFile: "AGENTS.md"},
    {Target: TargetOpenCode, SubDir: ".opencode",                          MarkerFile: "AGENTS.md"},
    {Target: TargetCursor,   SubDir: filepath.Join(".cursor", "rules"),    MarkerFile: ""},
    {Target: TargetCopilot,  SubDir: filepath.Join(".github", "copilot"),  MarkerFile: ""},
    {Target: TargetGeneric,  SubDir: ".ai",                                MarkerFile: "AGENTS.md"},
}
```

### 2. Multi-target AGENTS.md write idempotency

When a workspace has both Codex and OpenCode (or Codex and Generic) installed at root, `Materialize` will iterate through all targets that match `DetectInstalledTargets`. Each one with `MarkerFile: "AGENTS.md"` will try to write `AGENTS.md`. The existing `writeMarkerFile` helper:

- Skips if the file exists and is *not* hero-managed (no `<!-- hero:satellite -->` marker), unless `--force`.
- Replaces if the file exists and *is* hero-managed.
- Creates if absent.

Because every target generates the same `<!-- hero:satellite -->` content (per `perTargetMarker`), the second target's write is idempotent against the first's. The content is identical except for the `Target:` line — which we want to be the *primary* target, not the *last-written* target.

To avoid that flap, change the write order: when multiple targets share a `MarkerFile`, only the first one in the iteration writes it. Track written marker paths in `Materialize` and skip duplicates. This is a 5-line change (a `map[string]bool` of written paths).

### 3. Per-target marker content for shared files

When OpenCode and Codex both write `AGENTS.md`, the marker says `Target: codex` (or `opencode`, depending on order) — confusing because both are active. Adjust the marker generator to take a *list* of targets when called from a marker-collision path, and render `Targets: codex, opencode` in the body. Keep the single-target shape for solo installs.

This is a small refactor of `perTargetMarker` — instead of accepting `target Target`, accept `targets []Target`. The single-target callers become `[]Target{target}`.

### 4. Document Copilot's marker absence

Add a short comment in `targetLayouts` next to the Copilot entry explaining *why* `MarkerFile` is empty: Copilot's instruction file lives at `.github/copilot-instructions.md` at the repo root, and Copilot's discovery is org/repo-scoped, so a per-folder marker has no read path. Without the comment, the next contributor will think it's an oversight.

### 5. Detection still works

`DetectInstalledTargets` walks `targetLayouts` and checks if each target's SubDir has populated agents/commands/skills. With Cursor and Generic added, both become detectable — so a fresh install of Cursor or Generic at the workspace root will offer satellite materialization for them in the walkthrough automatically.

### Design decisions

**Why match OpenCode and Generic to AGENTS.md instead of giving each a unique file?** Because Codex, OpenCode, and Generic *all* converge on AGENTS.md at the project root in their root-install paths. Diverging at the satellite layer would invent a problem — three slightly different files saying the same thing. AGENTS.md is the convergence point; one marker, all three consumers happy.

**Why pick the first iteration order to write the AGENTS.md marker, instead of "last writer wins"?** Because the iteration order is deterministic (the registry is a slice in declared order: Codex before OpenCode before Generic). Predictable beats arbitrary. If you want to change which target's metadata gets recorded as primary, reorder the registry — that's a one-line PR with a clear rationale.

**Why does Cursor have no marker file?** Cursor reads `.cursor/rules/` walking up from cwd, but unlike Claude Code's `CLAUDE.md`, Cursor doesn't have a per-folder instruction-file convention that the model picks up automatically. There's no place to put "you are in a satellite" content that Cursor would actually surface. Empty is the honest answer; symlinks alone are enough for the cwd discovery to find the right rules.

**Why not extend the marker to include a list of all installed targets in the metadata for the satellite folder?** The `.hero-satellite` JSON marker already serves that role — it's machine-readable and the workspace locator reads it. The per-target markdown markers are *for the model's prompt context*, not for tooling. Two layers, two purposes; conflating them would muddy the design.

**Why ship Copilot's "documented absence" instead of writing a marker file there too?** Because writing `.github/CLAUDE.md` (or any marker under `.github/`) inside a subfolder is wrong — `.github/` is conventionally repo-root only, and putting a copilot-flavored file in a subfolder's `.github/` would make Copilot think there's a separate sub-repo. The right answer is "Copilot satellites get content symlinks but no marker, by design," and to write that down so it's not a question.

## Acceptance Criteria

- THE SYSTEM SHALL set `MarkerFile` to `"AGENTS.md"` for the OpenCode target in `targetLayouts`.
- THE SYSTEM SHALL include a `TargetCursor` entry in `targetLayouts` with `SubDir = filepath.Join(".cursor", "rules")` and an empty `MarkerFile`.
- THE SYSTEM SHALL include a `TargetGeneric` entry in `targetLayouts` with `SubDir = ".ai"` and `MarkerFile = "AGENTS.md"`.
- WHEN `Materialize` runs against a satellite folder for two or more targets that share a `MarkerFile` value THE SYSTEM SHALL write that marker file exactly once.
- THE SYSTEM SHALL render `Target:` (or equivalent) in the per-target marker as the comma-separated list of targets that share the file when multiple share, and as the single target name when only one applies.
- THE SYSTEM SHALL include a code comment near the Copilot entry in `targetLayouts` explaining that `MarkerFile` is intentionally empty because Copilot's discovery is repo-root-scoped.
- THE SYSTEM SHALL detect Cursor and Generic root installs via `DetectInstalledTargets` so the satellite walkthrough offers materialization when those targets are present.

## Changes

### Modified files
- `internal/install/satellite.go` — extend `targetLayouts`, add written-marker dedup in `Materialize`, take a target list in `perTargetMarker`, comment Copilot's empty marker.
- `internal/install/satellite_test.go` — add tests for: OpenCode-only install writing AGENTS.md; Codex+OpenCode dual install writing AGENTS.md exactly once with both targets named; Cursor and Generic showing up in `DetectInstalledTargets`.

## Phasing

Single phase — the change is mechanical and atomic. No partial state worth shipping separately.

## Kickoff

Resume by reading the spec at `.hero/planning/features/satellite-harness-coverage/spec.md`. Closes the per-harness coverage gap from the satellite arc. Mostly a registry edit (`targetLayouts`) plus a small refactor of `perTargetMarker` to accept multiple targets and a written-paths dedup in `Materialize`. Parent spec: monorepo-satellite-installs. Verified gap during example codebase migration (OpenCode+Claude both installed at root, only Claude got the marker).
