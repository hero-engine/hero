---
title: "`hero recap unregister` references linger after removal; `hero recap` errors hard on a fresh repo with no commits"
type: bug
status: completed
severity: medium
root_cause_class: process  # Defect A — cleanup process missed surfaces; Defect B is `code`
tags: [recap, slash-commands, install, dx, fresh-repo]
relates-to: [recap-register-missing, context-files-flag-drift]
created: 2026-05-18
---

# `hero recap unregister` references linger after removal; `hero recap` errors hard on a fresh repo with no commits

## Kickoff

Two `hero recap`-adjacent papercuts, both delivered: the stale `/diagnose` and `/deliver` slash commands in `.claude/commands/` no longer instruct agents to run the removed `hero recap register/unregister` subcommands, and `hero recap` on a freshly-initialized repo with no commits now exits 0 with "No activity in this window." instead of error-128'ing.

**Status:** completed — both defects fixed, tests pass, manual sanity confirmed.

**Pick up at:** nothing — this spec is done. If you're here looking for the follow-up `hero check` rule for known-removed CLI surfaces in markdown / installed copies, that's listed under "Related concerns" in this spec but was deliberately deferred. Start a new spec for it.

→ `.hero/specs/bugs/recap-unregister-stale-and-empty-repo/spec.md` (post-archive)

**Files changed:**
- `.claude/commands/diagnose.md` — replaced `recap register/unregister` instruction with `hero next ask` wording (mirrors source `commands/diagnose.md`).
- `.claude/commands/deliver.md` — same fix.
- `internal/recap/recap.go` — `gitCommits` now distinguishes "no commits yet" (returns nil slice, nil error) from genuine git failures; captures stderr via `*exec.ExitError`.
- `internal/recap/recap_test.go` — added `TestBuild_EmptyRepo` and `TestBuild_NotAGitRepo` covering both branches.

**Skip:** implementing `recap register/unregister` for real — superseded by `hero next ask/suggest/reflection` per `.hero/specs/recap-register-missing/spec.md`.

---

## Issue

Two defects filed together because they both touch `hero recap` user-facing behavior.

**Defect A (process):** The `recap register` / `recap unregister` subcommands were removed in [`.hero/specs/recap-register-missing/spec.md`](../../../specs/recap-register-missing/spec.md). That spec's validation step ran `grep -rn "recap register\|recap unregister" commands/ domains/` and confirmed zero matches. But the grep didn't cover `.claude/commands/` (the locally-installed copies) — and that's the directory Claude Code actually reads when the user invokes `/diagnose` or `/deliver` in this repo. Two stale files survived. The `/diagnose` skill body that fired in this very session printed `hero recap register <session-id> <slug> /diagnose` as step one.

**Defect B (code):** `internal/recap/recap.go:54` returns `fmt.Errorf("reading git log: %w", err)` when the underlying `git log` call in `gitCommits` (line 145–174) fails. On a brand-new repo with no commits, `git log` exits 128 with `fatal: your current branch 'main' does not have any commits yet`. Expected behavior: empty recap, exit 0 — "no activity yet" is a fact about a fresh repo, not an error.

**Severity:** medium. Defect A: agent friction every time `/diagnose` or `/deliver` fires in a repo with stale installed slash commands — the first step of the workflow shells out a non-existent subcommand and either errors or (since [recap-register-missing] removed the symbol) returns `unknown command "register" for "hero recap"`. Recoverable, but each user hits it. Defect B: blocks `hero recap` for first-time users immediately after `git init` and before the first commit — the "fresh repo" walkthrough is broken.

---

## Investigation

### Defect A — stale slash-command copies

Live offenders (confirmed by `grep -rn "recap register\|recap unregister"`):

| File | Lines | Note |
|------|-------|------|
| `.claude/commands/diagnose.md` | 10–14 | "**Before starting work**, register the active spec so context survives compaction: `hero recap register <session-id> <slug> /diagnose` … unregister with `hero recap unregister <session-id>`." |
| `.claude/commands/deliver.md` | 8–13 | Same pattern, `/deliver` variant. |

Confirmed-clean (do not edit):

| File | Status |
|------|--------|
| `commands/diagnose.md` | Already rewritten to "emit `hero next ask`" — diff against `.claude/commands/diagnose.md` shows the divergence. |
| `commands/deliver.md` | Same — clean. |
| `domains/**` | `grep` returns no matches. |
| `agents/**` | No matches. |
| `skills/**` | No matches. |
| `internal/**` | No matches (no skill content embedded in Go strings still references the removed commands). |

**Documentation surfaces that mention the removed commands intentionally** (do not edit — these document the removal itself):

- `.hero/specs/recap-register-missing/spec.md` — the archived diagnosis spec for the original removal. It cites the strings inside fenced code blocks as examples of what used to be there. Touching this rewrites history.

**Why the previous validation missed `.claude/commands/`:** the validation grep in `recap-register-missing/spec.md` line 22 read:

```
grep -rn "recap register\|recap unregister" commands/ domains/
```

It only covered the *source* directories (`commands/`, `domains/`). `.claude/commands/` is the *installed copy* (this repo's `.gitignore` excludes `.claude/agents`, `.claude/commands`, `.claude/skills` — confirmed via `git check-ignore`). The fix was authored against the source tree, but the in-repo install of the source tree (used by Claude Code when running in this very codebase) was never refreshed.

This is a recurring pattern — see the related `context-files-flag-drift` spec being filed in this same batch: a removed/renamed CLI surface persists in some installed copy or template after the source is cleaned.

### Defect B — empty repo handling

Reproduction (1 line):

```
mkdir /tmp/empty && cd /tmp/empty && git init -q && git log --since=2024-01-01 ; echo $?
```

Output:

```
fatal: your current branch 'main' does not have any commits yet
128
```

That non-zero exit code propagates through `exec.Command.Output()` at `internal/recap/recap.go:151`:

```go
out, err := cmd.Output()
if err != nil {
    return nil, err
}
```

`Build` at line 52–54 wraps that into `"reading git log: ..."` and returns it. The CLI at `internal/cli/recap.go:50–53` re-wraps it as `"building recap: ..."`. End result: `hero recap` in a fresh repo prints:

```
Error: building recap: reading git log: exit status 128
```

(The actual stderr from git is lost because `cmd.Output()` only captures stdout; `cmd.Stderr` is never wired, so the user never even sees the "does not have any commits yet" hint.)

`git log` exit conditions worth distinguishing:

| Condition | Exit | Stderr |
|-----------|------|--------|
| Empty repo, no commits | 128 | `fatal: your current branch '<name>' does not have any commits yet` |
| Not a git repo | 128 | `fatal: not a git repository` |
| Bad `--since` arg | 128 | `fatal: invalid date format` |
| Empty result, valid query | 0 | (empty stdout) |

The fix must distinguish "no commits" (legitimate empty result) from genuine git errors (corrupted repo, bad ref, etc.) — otherwise we'd silently swallow real problems.

### Root cause

**Defect A — process.** The cleanup spec defined "validation passed" as `grep` returning empty against `commands/` and `domains/`. That gate did not include the installed-copy surfaces (`.claude/commands/`, `.claude/agents/`, `.claude/skills/`). Because `.claude/` is gitignored, the stale copies in *this* repo are invisible to source review — but they're exactly what Claude Code reads when a workflow fires in this repo. A clean source tree gave a false sense of completion.

**Defect B — code.** `gitCommits` treats all non-zero exits from `git log` as fatal, with no branch for the "newly-initialized repo" case. The error message also drops stderr entirely, hiding the actual git diagnostic from the user.

### Code Flow (End to End)

**Defect A flow (per-session, agent-facing):**

1. User runs `/diagnose <bug>` in this repo.
2. Claude Code reads `.claude/commands/diagnose.md` (the *installed* slash-command body, not `commands/diagnose.md`).
3. The injected workflow tells the agent: "Before starting work, register the active spec: `hero recap register …`".
4. Agent shells out `hero recap register <session-id> <slug> /diagnose`.
5. cobra at `internal/cli/recap.go:12` rejects with `unknown command "register" for "hero recap"`.
6. Agent either proceeds anyway (best case) or treats step one as required and stalls (worst case).

**Defect B flow:**

1. User: `git init && hero recap` in a fresh project.
2. `internal/cli/recap.go:33` `runRecap` runs.
3. `internal/cli/recap.go:50` calls `recap.Build(heroDir, projectRoot, since)`.
4. `internal/recap/recap.go:52` calls `gitCommits(projectRoot, since)`.
5. `internal/recap/recap.go:151` `cmd.Output()` returns `*exec.ExitError` (exit 128, stderr "does not have any commits yet").
6. `internal/recap/recap.go:153` returns `nil, err` — error preserved, stderr discarded.
7. `internal/recap/recap.go:54` wraps as `"reading git log: %w"`.
8. `internal/cli/recap.go:52` wraps again as `"building recap: %w"`.
9. cobra prints `Error: building recap: reading git log: exit status 128` and exits 1.

User sees an error message that doesn't even mention "no commits."

### Key Files

#### Slash-command surfaces (Defect A)

| File | Lines | Relevance |
|------|-------|-----------|
| `.claude/commands/diagnose.md` | 10–14 | Live stale instruction injected when `/diagnose` fires in this repo. **Must rewrite.** |
| `.claude/commands/deliver.md` | 8–13 | Same for `/deliver`. **Must rewrite.** |
| `commands/diagnose.md` | 8–10 | Source already correct — use as the template for the rewrite. |
| `commands/deliver.md` | 8–10 | Source already correct — use as the template. |
| `.hero/specs/recap-register-missing/spec.md` | (whole) | Archived diagnosis. **Do not touch** — references are inside fenced examples documenting the prior bug. |

#### Recap code (Defect B)

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/recap/recap.go` | 49–55 | `Build` — wraps the `gitCommits` error. |
| `internal/recap/recap.go` | 145–174 | `gitCommits` — root of the unhandled exit-128 case. |
| `internal/recap/recap.go` | 251–299 | `RenderText` — already gracefully handles `len(r.Specs) == 0`, so once `Build` returns `&Recap{}` instead of error, the output flow works. |
| `internal/cli/recap.go` | 50–53 | Caller — does the second wrap into "building recap". |
| `internal/recap/recap_test.go` | (whole) | Only covers `ParseSince`. **No coverage for `gitCommits` or `Build` against a fresh-repo fixture.** |

#### Install / process surfaces (root cause of Defect A)

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/install/target_claude.go` | 12 | Documents `.claude/commands/<name>.md` as the install target. |
| `internal/install/install_test.go` | 705–708 | Confirms commands install to `.claude/commands/`. |
| `.gitignore` | (claude block) | `.claude/agents`, `.claude/commands`, `.claude/skills` — explains why source-tree grep missed the installed copies. |

---

## Goal

1. **Defect A** — every live agent-facing surface in this repo stops telling agents to run the removed `hero recap register/unregister` commands. Fresh `/diagnose` and `/deliver` invocations follow the same `hero next ask` instruction as the source `commands/` files.
2. **Defect B** — `hero recap` against a freshly-initialized git repo (no commits yet) exits 0 with the "No activity in this window." rendering, not a non-zero error.
3. **Process** — extend the validation pattern from `recap-register-missing` so future removals of CLI subcommands also cover installed-copy surfaces and embedded Go strings. Either widen the grep gate in `hero check` or document the broader sweep in the relevant skill.

---

## Suggested Fix Approach

### Defect A — Update stale `.claude/commands/` files

The source files (`commands/diagnose.md`, `commands/deliver.md`) already have the corrected text. The fix is to make `.claude/commands/diagnose.md` and `.claude/commands/deliver.md` match.

**File:** `.claude/commands/diagnose.md`

Before (lines 10–14):
```markdown
**Before starting work**, register the active spec so context survives compaction:
```
hero recap register <session-id> <slug> /diagnose
```
When diagnosis completes, unregister with `hero recap unregister <session-id>`.
```

After (mirror `commands/diagnose.md` lines 8–10):
```markdown
**Before starting work**, emit `hero next ask` to capture the bug report
the user pasted in. This preserves session intent across compaction — see
the `next-handoff-emit` skill for the full pattern.
```

**Why:** removes the broken subcommand reference and points agents at the canonical compaction-survival mechanism (`hero next ask`), matching the source tree that ships to users.

**File:** `.claude/commands/deliver.md`

Before (lines 8–13):
```markdown
**Before starting work**, register the active spec so context survives compaction:
```
hero recap register <session-id> <slug> /deliver
```
Use any unique session identifier (timestamp, hostname, etc.). When delivery
completes, unregister with `hero recap unregister <session-id>`.
```

After (mirror `commands/deliver.md` lines 8–10):
```markdown
**Before starting work**, emit a `hero next ask` capturing what the user
asked for. This preserves session intent across compaction — see the
`next-handoff-emit` skill for the full pattern (ask / suggest / reflection).
```

**Why:** same reason as `diagnose.md`. Brings the installed copy back in sync with the source.

**Note on `.claude/` being gitignored:** the fix changes a gitignored file — no commit will include the edit. That's fine: the file is regenerated on every `hero install`. To make the fix durable, ensure a clean `hero install` (or `hero install --force`) on this repo runs after the cleanup to re-sync from the source. The actual durable surface is the source `commands/` (already correct).

### Defect B — `gitCommits` handles empty repo

**File:** `internal/recap/recap.go`, function `gitCommits` (lines 145–174)

Before:
```go
func gitCommits(projectRoot string, since time.Time) ([]CommitSummary, error) {
	sinceStr := since.Format("2006-01-02T15:04:05")
	cmd := exec.Command("git", "-C", projectRoot, "log",
		"--since="+sinceStr,
		"--pretty=format:%H\t%s\t%an\t%aI",
		"--no-merges")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	...
}
```

After:
```go
func gitCommits(projectRoot string, since time.Time) ([]CommitSummary, error) {
	sinceStr := since.Format("2006-01-02T15:04:05")
	cmd := exec.Command("git", "-C", projectRoot, "log",
		"--since="+sinceStr,
		"--pretty=format:%H\t%s\t%an\t%aI",
		"--no-merges")
	out, err := cmd.Output()
	if err != nil {
		// Fresh repo with no commits yet is a legitimate empty result, not an error.
		if ee, ok := err.(*exec.ExitError); ok {
			stderr := string(ee.Stderr)
			if strings.Contains(stderr, "does not have any commits yet") ||
				strings.Contains(stderr, "bad default revision 'HEAD'") {
				return nil, nil
			}
		}
		return nil, err
	}
	...
}
```

**Why:** the "no commits yet" case is a legitimate empty result for a brand-new repo. Other failures (corrupt repo, bad date format, not a git repo) still surface as errors. `strings` is already imported. Matching against the stderr substring is the same approach `git` itself uses for porcelain compatibility — git's exit codes don't distinguish this case any other way.

Note: `cmd.Output()` only captures stderr in the `ExitError` when `cmd.Stderr` is `nil`. That's already the case here, so `ee.Stderr` will be populated.

The downstream caller `Build` at line 52–55 already handles `commits == nil` correctly — the `for _, c := range commits` loop at line 69 simply iterates zero times, and the returned `*Recap` renders as "No activity in this window." via `RenderText`'s existing branch at line 258.

### Related concerns — broaden the cleanup gate

This is the second time in one batch a removed/renamed CLI surface has shown up where the source-tree grep already returned clean. Two complementary fixes:

1. **Widen the validation pattern** in any future "remove a CLI subcommand" spec to include:
   - `commands/` and `domains/` (source)
   - `.claude/commands/`, `.claude/agents/`, `.claude/skills/` (installed copies — even though gitignored, they live in the repo and feed into the running agent)
   - `internal/` Go strings (`grep -rn 'hero <removed>' internal/`)
   - `agents/` and `skills/` (source — these were already covered for `recap-register-missing` by accident but should be explicit)

2. **Add a `hero check` rule** for a curated list of known-removed commands. Each removed subcommand goes into a registry (e.g. `internal/check/removed_commands.go`) keyed by `parent + " " + subcommand` (e.g. `recap register`, `recap unregister`). `hero check` greps every Markdown surface in the workspace (`commands/`, `domains/`, `agents/`, `skills/`, `.claude/`, `.hero/`) for those tokens and warns with a pointer to the spec that removed them.

The second is more durable. The registry approach can be cross-linked with the `context-files-flag-drift` spec (filed in this same batch) — the symptom is the same (stale references to removed/renamed CLI surfaces), and one `hero check` rule could catch both classes.

---

## Test Plan

### Existing test review

- `internal/recap/recap_test.go` — covers only `ParseSince` (5 cases). **No coverage for `gitCommits`, `Build`, or `RenderText`.**
- No tests touch the `.claude/commands/` slash-command bodies. (The install harness at `internal/install/harness_smoke_test.go` confirms files exist but doesn't grep content.)

### Test changes needed

**Defect B:**

1. Add `TestBuild_EmptyRepo` to `internal/recap/recap_test.go`:
   - Create a temp dir, run `git init`, do not commit.
   - Call `recap.Build(heroDir, tempDir, time.Now().Add(-24*time.Hour))`.
   - Assert `err == nil`.
   - Assert returned `*Recap` is non-nil with empty `Specs` / `Knowledge` / `Unmatched`.
2. Add `TestBuild_NotAGitRepo`:
   - Create a temp dir, do **not** run `git init`.
   - Call `recap.Build`.
   - Assert `err != nil` and error contains "reading git log" — confirms we didn't over-swallow.

**Defect A:**

3. (Optional, lower-leverage) Add an install-smoke assertion: after `hero install --target claude` in a scratch workspace, grep the installed `.claude/commands/*.md` for `recap register|recap unregister` and assert zero matches. Belongs in `internal/install/harness_smoke_test.go`.

**Process gate:**

4. If pursuing the `hero check` rule from "Related concerns," add a test that runs the rule against a fixture workspace containing a stale reference and asserts the warning fires.

### Regression scope

- `hero recap` against a populated repo — must still surface commits as before. Covered by adding `TestBuild_PopulatedRepo` if not already present (it isn't).
- `hero recap --cross-repo` — the cross-repo loop at `internal/cli/recap.go:56–74` swallows per-repo build errors silently. With Defect B's fix, an empty peer repo will now contribute nothing instead of being silently skipped — same observable behavior, slightly better internal hygiene.
- The `.claude/commands/` edits do not affect the source `commands/` tree, so `hero install` regenerates them on next install. No risk to the install path.

---

## Boundaries

- **Out of scope:** implementing `recap register` / `recap unregister` for real. That was already deliberately rejected in `recap-register-missing/spec.md` — `hero next ask/suggest/reflection` is the canonical compaction-survival surface.
- **Out of scope:** editing `.hero/specs/recap-register-missing/spec.md`. That archived spec is documentation about the prior removal; its in-line code-fence references are intentional historical record.
- **Out of scope:** changing the `.gitignore` rules for `.claude/`. Re-tracking the installed copies would force two-copy maintenance.
- **Out of scope:** generalizing the `hero check` "removed-command registry" idea beyond a recommendation — that's its own design spec if pursued.

---

## Risks

- **Defect B fix is stderr-substring-based.** Future git versions could rephrase the message. Mitigation: the secondary check for `"bad default revision 'HEAD'"` covers older git phrasings; the test fixture pins behavior; if git's wording shifts, the test will catch it.
- **Defect A fix only patches a gitignored file in this repo.** Other developers' repos will still have stale `.claude/commands/` until they re-run `hero install`. Acceptable — the source tree (which they pulled) is already correct, and the next install refreshes them. Calling this out in `next-handoff-emit` release notes would be friendly.
- **`exec.ExitError.Stderr` is empty if `cmd.Stderr` is wired to something else.** Confirmed not wired in `gitCommits`. If future changes route stderr (e.g. for logging), the substring check would silently fail. Document this with a comment near the fix.

---

## Validation

1. **Defect A — manual:** after the edit, run `grep -rn "recap register\|recap unregister" .claude/ commands/ domains/ agents/ skills/ internal/`. Must return zero matches outside `.hero/specs/recap-register-missing/`.
2. **Defect A — runtime:** start a fresh session, invoke `/diagnose` against a fixture bug spec, confirm the agent does **not** try to run `hero recap register`.
3. **Defect B — manual:** `mkdir /tmp/empty && cd /tmp/empty && git init -q && hero init && hero recap`. Must exit 0 with `Activity since <ts>` and `No activity in this window.`
4. **Defect B — automated:** new tests in `internal/recap/recap_test.go` pass; `go test ./internal/recap/...` clean.
5. **Regression:** `go build ./...` clean; full `go test ./...` clean; `hero recap` in this populated repo still produces a non-empty digest.

---

## Recap

Two `hero recap`–adjacent papercuts share a spec because they surface the same DX failure pattern. Defect A is a process miss: the `recap-register-missing` cleanup validated `commands/` and `domains/` but never grepped `.claude/commands/`, leaving the locally-installed slash-command copies still telling agents to run a subcommand that no longer exists. Defect B is a code defect: `gitCommits` treats `git log`'s "no commits yet" exit (legitimate empty result on a fresh repo) as a fatal error. Both fixes are mechanical; the more interesting recommendation is a `hero check` rule (or a widened grep gate) for known-removed CLI commands so this class of drift gets caught at PR time rather than at agent-runtime.
