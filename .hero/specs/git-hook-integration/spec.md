---
title: Git Hook Integration — Branch Creation Drives Spec Status Changes
slug: git-hook-integration
type: feature
status: completed
milestone: v0.3
tags: [git, hooks, automation, status, workflow, lifecycle]
created: 2026-04-13
relations:
  - target: agent-contribution-tracking
    kind: related
  - target: hero-pulse
    kind: related
  - target: sprint-from-tracker
    kind: related
horizon: now
---

## Goal

Connect git events to Hero spec lifecycle transitions automatically — when an engineer creates a branch named after a spec, Hero marks that spec as `delivering`; when they merge, Hero marks it `done`. No manual status updates, no drift between what's in Hero and what's actually happening in the repo.

## Problem

Hero spec statuses drift from reality. Engineers start work on a spec, don't update the status, and Hero shows it as `planning` while it's actually 80% implemented. This makes `hero status`, `hero pulse`, and `hero context` less accurate — they're working from stale signals.

The root cause: status updates require deliberate action. Git events (branch create, push, merge) are already happening naturally. Hooking into git makes status updates automatic and zero-friction.

## Design

### Install / Uninstall

```
hero hooks install      # install Hero git hooks into .git/hooks/
hero hooks uninstall    # remove Hero git hooks
hero hooks status       # show which hooks are installed and active
```

Hero installs lightweight shell scripts into `.git/hooks/` that call `hero hook <event>`. The hooks are non-blocking — if Hero is unavailable, the git operation proceeds normally.

### Supported Hooks

| Git Hook | Trigger | Hero Action |
|---|---|---|
| `post-checkout` | Branch created or switched | Spec status → `delivering` if branch name matches slug |
| `post-merge` | Merge commit completed | Spec status → `done` if branch name matched a spec |
| `post-commit` | Any commit | Record commit SHA + timestamp against matched spec |
| `prepare-commit-msg` | Before commit editor opens | Inject spec slug into commit message template |

### Branch Name Matching

Hero maps branch names to spec slugs using configurable patterns:

```json
{
  "hooks": {
    "branch_patterns": [
      "feat/{{slug}}",
      "feature/{{slug}}",
      "{{slug}}",
      "fix/{{slug}}"
    ],
    "slug_transform": "kebab"
  }
}
```

Examples:
- `feat/hero-ask` → matches slug `hero-ask`
- `feature/spec-triage-intake` → matches slug `spec-triage` (longest prefix match)
- `hero-ask` → direct slug match

If no pattern matches, the hook is a no-op. No false positive status transitions.

### Status Transition Rules

Transitions are gated — Hero only applies them if they make sense:

| Current Status | Git Event | New Status | Condition |
|---|---|---|---|
| `planning` | branch created | `delivering` | branch matches slug |
| `delivering` | branch merged to main | `done` | merge target is main/master |
| `done` | branch created | `delivering` | re-opens spec (with warning) |
| `delivering` | branch deleted without merge | `planning` | configurable: `hooks.revert_on_delete` |

Transitions are logged to `.hero/events.log` with timestamp, branch, and triggering hook.

### Commit Message Injection (`prepare-commit-msg`)

When on a branch that matches a spec slug, Hero prepends the spec slug to the commit message template:

```
[hero-ask] 
# Please enter the commit message for your changes.
# Branch: feat/hero-ask
# Spec: hero-ask (delivering) — Semantic Query Against Knowledge and Specs
```

This is opt-in via config:
```json
{
  "hooks": {
    "inject_commit_prefix": true
  }
}
```

### Conflict with Manual Status

If a spec's status was manually set to something that would be overridden by a hook, Hero warns but applies the transition:

```
hero: auto-advancing hero-ask from planning → delivering (branch feat/hero-ask created)
      Override manual status? [Y/n]
```

In non-interactive environments (CI, no TTY), `--no-prompt` applies the transition silently.

### Hook Script Format

The installed hook is a minimal shell script:

```sh
#!/bin/sh
# Hero git hook — post-checkout
# Installed by: hero hooks install
# Remove with: hero hooks uninstall
hero hook post-checkout "$@" 2>/dev/null || true
```

The `|| true` ensures git never fails due to Hero. Hero's hook handler is in `internal/hooks/`.

## Changes

- `internal/hooks/hooks.go` — hook event handler, branch pattern matching, status transition logic
- `internal/hooks/install.go` — hook script generation, `.git/hooks/` install/uninstall
- `internal/cli/hooks.go` — `hero hooks` command group (install, uninstall, status)
- `internal/cli/hook.go` — `hero hook <event>` internal command (called by hook scripts)
- `internal/config/config.go` — `hooks` config section

## Acceptance Criteria

- `hero hooks install` writes Hero hook scripts to `.git/hooks/` without overwriting existing non-Hero hooks (appends a call if hook already exists)
- `hero hooks uninstall` cleanly removes Hero's additions, leaving other hook content intact
- Creating a branch matching a spec slug transitions that spec to `delivering`
- Merging a matching branch to main transitions the spec to `done`
- No git operation ever fails due to Hero — hooks always exit 0
- Branch patterns are configurable; default patterns cover `feat/`, `feature/`, and bare slug
- Transition events are logged to `.hero/events.log`
- `hero hooks status` shows installed hooks and their Hero version
- `prepare-commit-msg` injection is opt-in and does not corrupt non-matching commits

## Boundaries

- Does **not** push status changes to remote tracker — that's `sprint-from-tracker` integration
- Does **not** require a network connection — all transitions are local
- Does **not** support bare repos or worktrees in v1
- Hook scripts call `hero hook` — they do not embed logic directly; uninstalling Hero cleanly removes all behavior
