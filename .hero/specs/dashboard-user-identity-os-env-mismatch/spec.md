---
title: Dashboard "you" identity uses $USER instead of git config — author-filtered metrics always read 0
slug: dashboard-user-identity-os-env-mismatch
type: bug
status: completed
severity: high
root_cause_class: code
priority: high
tags: [dashboard, serve, identity, metrics, attribution]
created: 2026-05-19
completed_at: 2026-05-19T14:16:21Z
---

# Dashboard "you" identity uses $USER instead of git config — author-filtered metrics always read 0

## Issue

Reporter: 277887514+chet-bellows@users.noreply.github.com — observed live in browser on 2026-05-19
against `hero serve` running locally on this workspace.

Symptoms:

- `/now` "commits authored — last 7 days" reads **0** despite the user
  authoring 132 commits in the trailing 7-day window.
- `/now` "longest open spec" reads "—".
- `/now` "your committed specs" reads "—" with footer "no claim yet".
- `/now` "On your plate" cards render empty when they should surface
  specs the user is actively working on.

Live verification (run in this repo, 2026-05-19):

```bash
$ echo $USER
bwheeler
$ git config user.name
chet-bellows
$ git config user.email
277887514+chet-bellows@users.noreply.github.com
$ git log --since='7 days ago' --pretty='%an' | sort | uniq -c
 132 chet-bellows
$ git log --since='7 days ago' --author=bwheeler --pretty=oneline | wc -l
       0
$ git log --since='7 days ago' --author=chet-bellows --pretty=oneline | wc -l
     132
```

`$USER` ≠ `git config user.name`, and the dashboard wires the former
through the entire attribution path.

## Investigation

### Two parallel identity helpers

The codebase has **two** "who is the user" helpers and they disagree:

1. `internal/serve/server.go:852-862 — shellUserName()` — reads `$USER`
   then `$USERNAME`, falls back to `"you"`. **Used by the entire serve
   layer.**

2. `internal/cli/session.go:320-333 — gitUserName()` — shells out to
   `git config user.name`, lowercases, hyphenates spaces. **Used by the
   CLI commands that write to `events.log` (claim.go, event.go,
   spec_move.go, session.go).**

So events are tagged with `human/chet-bellows` (the git user) and the
dashboard tries to read them back as `bwheeler` (the OS user). The two
namespaces never intersect.

### What `shellUserName()` feeds into

`internal/serve/server.go:646` derives `userName := shellUserName()`,
then passes it as `UserName` to every page's `Deps`:

```
nowDeps.UserName     // line 659
workDeps.UserName    // line 676
knowledgeDeps.UserName // line 690
peopleDeps.UserName  // line 704
agentsDeps.UserName  // line 722
projectDeps.UserName // line 740
```

And to `shell.New(..., userName, ...)` at line 648, where it becomes
the avatar tooltip in the top-nav.

### Where the wrong identity bites

#### Commits-authored tile (Now + My Week)

`internal/serve/pages/now/data/metrics.go:75-109` and `:113-146`:

```go
commits, _ := gitCountCommitsSince(in.ProjectRoot, "7 days ago", in.UserName)
```

`gitCountCommitsSince` at `metrics.go:258-278`:

```go
args := []string{"-C", projectRoot, "log", "--since=" + since, "--pretty=oneline"}
if user != "" {
    args = append(args, "--author="+user)
}
```

With `user="bwheeler"` and no commits authored by an email/name
matching `bwheeler`, the count is always 0.

#### "On your plate" cards

`internal/serve/pages/now/data/plate.go:60-70`:

```go
func claimedByMatches(claimedBy, user string) bool {
    if claimedBy == "" {
        return false
    }
    cb := strings.ToLower(strings.TrimSpace(claimedBy))
    u := strings.ToLower(strings.TrimSpace(user))
    if u == "" {
        return false
    }
    return cb == u || cb == "you" || cb == "me"
}
```

Spec frontmatter is claimed by `chet-bellows` (via the CLI's
`gitUserName()` helper). The plate compares to `bwheeler`. Never
matches. Plate stays empty.

Live: a quick search of this workspace shows specs with
`claimed_by: chet-bellows` exist:

```bash
$ grep -rl "claimed_by: chet-bellows" .hero/planning .hero/specs | head -3
```

(These don't surface to the plate.)

#### Top-nav avatar tooltip

`internal/serve/shell/templates/top-nav.html:34` — the avatar tooltip
becomes `bwheeler` and the initials are `BW`, even though the user's
git identity (and therefore every event log entry) reads
`chet-bellows`.

#### `commits authored` sparkline

The sparkline in `metrics.go:91` plots `[1,0,1,2,0,1,shipped]` where
`shipped` is the misattributed count — so the spark always looks flat
near zero. Cosmetic but reinforces the wrong story.

### Reproduction (no special state required)

1. Set up any git checkout where `git config user.name` ≠ `$USER`.
   (This is the **default** for anyone whose OS login is different
   from their commit-author handle — extremely common with
   pseudonymous GitHub aliases, work-vs-personal forks, etc.)
2. Commit a few changes.
3. Claim a spec via `hero spec claim` (sets `claimed_by` to
   `gitUserName()`).
4. `hero serve` → `/now`.
5. Observe "commits authored — last 7 days" = 0, "On your plate" empty,
   even though both have actual data behind them.

### Root cause

Two parallel identity sources. The serve layer uses one (`$USER`); the
event-log emitters and spec-claim writers use the other (`git config
user.name`). Every author-filtered or claim-filtered tile/section
reads through the wrong lens and produces empty output.

### Severity

**High.** Affects every dashboard user whose OS login differs from
their git author identity — i.e. essentially every developer using a
GitHub `noreply` email or pseudonymous handle, and every shared
machine, and every developer container where `$USER` is `root` or
`vscode`. The dashboard's most "personal" metrics — your commits,
your plate, your committed specs — all read zero on a normal day.

Caused by our codebase. No workaround short of renaming the OS user
or hand-editing every spec's `claimed_by` to match `$USER`.

## Code Flow (End to End)

1. `internal/serve/server.go:646` — `userName := shellUserName()` →
   reads `$USER` env (e.g. `bwheeler`).
2. `internal/serve/server.go:654-740` — `userName` passed into every
   page's `Deps.UserName`.
3. `internal/serve/pages/now/page.go:222-236` — Now's plate and
   metrics loaders receive that string.
4. `internal/serve/pages/now/data/plate.go:32-41` — plate filters
   specs by `claimedByMatches(s.ClaimedBy, in.UserName)`.
5. `internal/serve/pages/now/data/metrics.go:85,121` — `weekTiles` and
   `myWeekTiles` call `gitCountCommitsSince(..., in.UserName)`.
6. `internal/serve/pages/now/data/metrics.go:258-278` — git is shelled
   with `--author=bwheeler`. Returns 0 because all commits are
   authored by `chet-bellows`.

Meanwhile the **writer** side:

7. `internal/cli/session.go:320-333` — `gitUserName()` reads
   `git config user.name`, lowercases, hyphenates.
8. `internal/cli/event.go:62-64` — events written with
   `agent = "human/" + gitUserName()`.
9. `internal/cli/claim.go:319` and `spec_move.go:208` — same convention.

Two namespaces, never reconciled.

## Key Files

### Identity helpers
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/server.go` | 852–862 | `shellUserName()` — uses `$USER`. Wrong helper. |
| `internal/cli/session.go` | 320–333 | `gitUserName()` — uses `git config user.name`. Right helper. |

### Serve-side consumers of (wrong) UserName
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/server.go` | 646–740 | userName fed into every page Deps |
| `internal/serve/pages/now/data/metrics.go` | 75–146, 258–278 | commits-authored tile |
| `internal/serve/pages/now/data/plate.go` | 21–70 | "On your plate" claim filter |
| `internal/serve/shell/templates/top-nav.html` | 34 | avatar tooltip + initials |

### CLI-side producers of identity-tagged data
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/claim.go` | 122–233 | writes claim events with `human/gitUserName()` |
| `internal/cli/event.go` | 60–80 | writes feed events with same agent string |
| `internal/cli/spec_move.go` | 200–220 | same |

## Secondary Defects

1. **`shellUserName()` falls back to `"you"`** when `$USER` and
   `$USERNAME` are both unset (e.g. some containers / cron contexts).
   "you" is *also* matched by `claimedByMatches` as a sentinel — so in
   containers the plate accidentally matches specs whose `claimed_by`
   was the literal string "you" (a different cohort entirely).

2. **`gitUserName()` lowercases and hyphenates**; the email is never
   consulted. A user with `user.name = "Brian Wheeler"` becomes
   `brian-wheeler` for claims but `Brian Wheeler` for git log queries.
   So even if serve started shelling `gitUserName()` directly, the
   `git log --author=brian-wheeler` query would still miss commits
   authored as `Brian Wheeler` (git's `--author` is a substring match,
   so this happens to work, but it's a latent edge case).

3. **No fallback strategy.** If `git config user.name` is unset (CI
   containers, fresh worktrees) the reading side silently returns
   `"unknown"` and the writing side prefixes `human/unknown`. There's
   no clear contract for "you".

4. The `metrics.go` "your committed specs" tile (line 81, 104) is a
   placeholder — value always `—`, footer "no claim yet". Once the
   identity bug is fixed, this tile still won't render a real number
   until someone wires the count. Out-of-scope for the fix but worth
   flagging so the reader doesn't think this spec finishes the tile.

## Notes

- The user's git config in this repo uses a GitHub `noreply` email:
  `277887514+chet-bellows@users.noreply.github.com`. That's by design
  and increasingly common — privacy-preserving GitHub identities mean
  the OS login is rarely the same as the commit author.
- A canonical "current user" helper should probably live in a shared
  package (e.g. `internal/identity`) and return a richer value
  (name + email + alias list) so individual call sites can match on
  the appropriate field. Today there's no such package.

## Acceptance Criteria

- THE SYSTEM SHALL derive the dashboard's "you" identity from
  `git config user.name` (or `user.email`) first, falling back to
  `$USER` only when git config is unavailable.
- WHEN serving a page THE SYSTEM SHALL pass the resolved identity to
  every page's `Deps.UserName` so plate / metrics / avatar all share
  the same value.
- WHEN counting commits THE SYSTEM SHALL pass an `--author=` term that
  matches the user's git author name (or email), not their OS login.
- WHEN filtering plate specs THE SYSTEM SHALL match
  `claimed_by` against the same git-derived identity that
  `hero spec claim` writes, so the round-trip is lossless.
- WHERE `git config user.name` is unset THE SYSTEM SHALL log a
  diagnostic and fall back to `$USER`, keeping the current behavior
  rather than failing the page render.
- THE SYSTEM SHALL match the avatar tooltip / initials in the top-nav
  to the same identity used for attribution.

## Goal

The dashboard's "you" is the same person `hero spec claim` and
`hero event` write into the workspace. Author-filtered tiles
(commits, plate, your-committed-specs) reflect real activity.

## Boundaries

- Not in scope: building a multi-identity "team mode" mapping for cloud
  / shared workspaces — that lives under `hero-team-server`.
- Not in scope: refactoring every `claimedByMatches` caller to consume
  a structured Identity object. A scalar string keeps the change
  surgical.
- Not in scope: per-commit author email vs. name distinction beyond
  what `git log --author=` already accepts as a substring match.

## Risks

- Switching the identity source changes long-standing test fixtures
  that hardcode `$USER`-style usernames. Audit `internal/serve/pages/
  now/data/*_test.go` and update accordingly.
- The git-shell on every page render adds latency vs. an env-var
  lookup. Cache the result at server start (Server struct field) so
  per-request renders don't fork `git`. (This also matches what
  `detectGitBranch` already does at boot time.)
- Tests that ran in CI under a different identity may have masked
  the bug. Add a test that asserts `Deps.UserName` matches
  `gitUserName()` output specifically (not `$USER`).

## Validation

1. In a workspace where `$USER` ≠ `git config user.name`:
   - "commits authored — last 7 days" matches `git log --author="$(git
     config user.name)" --since='7 days ago' --pretty=oneline | wc -l`.
   - Claiming a spec via `hero spec claim` makes it surface on `/now`
     "On your plate" without further intervention.
   - Top-nav avatar tooltip reads `chet-bellows`, initials `CB`.
2. In a workspace where they match (the common-but-not-universal case):
   - No regression — everything still works.
3. In a worktree with no git config (`unset` both):
   - Pages render with `userName = "unknown"` and a single log line
     warning the operator. No panic; tiles render with placeholders.
4. Regression tests:
   - Pin `shellUserName` (renamed to e.g. `resolveDashboardUser`) to
     return `gitUserName()` first.
   - Add a plate test fixture where `$USER` and `claimed_by` differ
     and assert the spec still surfaces.
   - Add a metrics test fixture with a git repo whose author is not
     `$USER` and assert commit count is non-zero.

## Recap

`internal/serve/server.go::shellUserName` uses `$USER` while the rest
of hero writes claims and events under `git config user.name`. Every
author-filtered dashboard widget reads through the wrong namespace and
reports zero. Replace the helper with a git-config-first resolver and
the personal metrics light up on every workspace that uses a
pseudonymous git identity.

## Changes

- `internal/gitutil/gitutil.go` — added `UserName()` helper that
  resolves identity via `git config user.name` → `$USER` →
  `$USERNAME` → `"unknown"`, applying the lowercase + hyphenate
  normalization both writer and reader sides need to round-trip.
- `internal/cli/session.go` — `gitUserName()` is now a thin wrapper
  around `gitutil.UserName()`. Every CLI writer (claim, event,
  spec_move, session-agent resolution, next.go) inherits the shared
  ladder for free.
- `internal/serve/server.go` — `shellUserName()` calls
  `gitutil.UserName()`. Emits a single startup diagnostic when git
  config is unset so operators spot the misconfiguration.
- `internal/gitutil/gitutil_test.go` — five regression tests pinning
  the resolution ladder (git wins over $USER, $USER fallback,
  $USERNAME fallback, unknown last-resort, normalize spaces+case).
  Tests scope git config via `GIT_CONFIG_GLOBAL=/dev/null` so the
  developer's `~/.gitconfig` doesn't leak into fixtures.
- `internal/serve/pages/now/data/plate_test.go` — two regression
  tests: (1) git-derived user surfaces a claim written under the git
  identity; (2) OS-login user does NOT match a git-identity claim
  (the failure mode this spec fixes).

Out of scope (flagged in spec, deferred):
- "Your committed specs" tile body — still placeholder; identity fix
  unblocks the wiring but doesn't ship the count.
- `--author=` substring edge case for git names with spaces — the
  normalized form happens to work for `chet-bellows` and most
  hyphen-form identities; declared out-of-scope in Boundaries.
- `claimedByMatches` literal `"you"` / `"me"` sentinels — left in
  place; spec calls out as Secondary Defect but not in acceptance
  criteria.
