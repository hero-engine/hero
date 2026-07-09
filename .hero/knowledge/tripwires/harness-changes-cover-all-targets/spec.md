---
title: Harness-Facing Changes Must Cover All Install Targets
type: tripwire
status: active
triggers: [
  "CLAUDE.md", "AGENTS.md", "harness", "install target", "instruction file",
  "slash command", "routing", "agents", "skills", "commands",
  "opencode", "cursor", "codex", "copilot", "generic", "hook", "PreCompact", "Stop hook"
]
scope: [
  "CLAUDE.md", "AGENTS.md",
  "internal/cli/install*.go",
  "domains/**/commands/*.md",
  ".agents/skills/**", ".hero/skills/**",
  ".claude/**", ".opencode/**", ".cursor/**"
]
severity: high
---
# Harness-Facing Changes Must Cover All Install Targets

## Constraint

Do not implement or deliver a harness-facing change (instruction-file content,
routing guidance, slash commands, agents, skills, or anything `hero install`
propagates) that handles only **one** harness — most commonly Claude/`CLAUDE.md`.
Hero installs to **six** targets: `opencode | cursor | claude | copilot | codex |
generic`. Every harness-facing change must account for all of them, or be
explicitly scoped with the reason the others are excluded.

## Why

Claude Code is the conspicuous outlier — it reads `CLAUDE.md` and has end-of-
session hooks (Stop/PreCompact); the other five read `AGENTS.md` and have **no
equivalent hook**. A change authored against `CLAUDE.md` or a Claude-only hook
silently breaks coverage for opencode, cursor, copilot, codex, and generic, and
the gap is invisible until someone runs Hero in those harnesses. This recurs
often enough to need an always-on guardrail. See
[[harness-instruction-file-survey]] and the `--target` set in
`internal/cli/install.go`.

## Instead

- **Author once, harness-agnostic.** Put the authoritative content in `.hero/`
  (knowledge/convention/skill), then let `hero install` propagation render it to
  each target.
- **AGENTS.md is the canonical instruction surface** (native to ~9 harnesses);
  `CLAUDE.md` is the Claude-only *view*, fed by the `CLAUDE.md → AGENTS.md`
  symlink / `@AGENTS.md` import `hero install` already wires.
- **Make AGENTS.md guidance self-contained and imperative** so a hookless harness
  can act on instruction alone. Treat the Claude Stop/PreCompact hook as an
  enhancement layered on top — never the mechanism. Never gate core behavior on a
  Claude-only hook.
- **Verify per-target propagation** before calling the change done — confirm the
  guidance lands in each target's instruction surface.
