---
title: Codex Trust Nudge — One-Time Hero CLI Approval Guidance
slug: codex-trust-nudge
type: feature
status: completed
priority: P1
tags: [install, codex, permissions, trust, ux]
created: 2026-05-04
mission_alignment: |
  Hero's mission depends on the harness being able to call Hero without
  turning every session into permission paperwork. This feature keeps the
  boundary honest — Codex grants permissions, not Hero — while making the
  happy path obvious at install time so fresh projects can move from
  "installed" to "usable" with one broad approval instead of repeated prompts.
principles_check: |
  Serves #1 (it just works) by making the one-time permission step visible
  when it matters, and #2 (natural language is the interface; tools are the
  escape hatch) by giving the exact prompt a user can give Codex. Avoids
  pretending Hero can mutate Codex's permission store directly.
horizon: now
---

## Goal

When a user installs Hero for Codex, Hero prints one clear optional next step:
ask Codex to run `hero status` with persistent approval for the `hero` command
prefix. Users can also re-display that guidance later with `hero trust codex`.

## Scope

- Add a Codex-specific trust nudge to successful `hero install --target codex`.
- Add `hero trust codex` as the manual way to show the same guidance later.
- Do not attempt to edit Codex's permission store or imply Hero can grant host
  permissions itself.

## Acceptance Criteria

**AC-1:** After a successful non-dry-run Codex install, the CLI prints an
optional permission section that says Codex owns the approval and gives the
exact prompt: ask Codex to run `hero status` and request persistent approval
for the `hero` command prefix.
✅ **passing** — verified by
`go test ./internal/cli -run 'TestTrust|TestInstallCodexPrintsTrustHint'`
and `go test ./internal/cli`.

**AC-2:** `hero trust codex` prints the same Codex permission guidance without
requiring a Hero workspace.
✅ **passing** — verified by
`go test ./internal/cli -run 'TestTrust|TestInstallCodexPrintsTrustHint'`
and `go test ./internal/cli`.

**AC-3:** Unsupported trust targets fail clearly and list the supported target.
✅ **passing** — verified by
`go test ./internal/cli -run 'TestTrust|TestInstallCodexPrintsTrustHint'`
and `go test ./internal/cli`.

## Kickoff

Deliver `codex-trust-nudge`. Keep the implementation narrow: add an install
completion nudge for `--target codex`, add a `hero trust codex` command that
prints the same text, and test both surfaces. Do not claim Hero can grant
Codex permissions directly; the wording should say Codex owns the approval.
