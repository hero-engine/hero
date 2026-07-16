---
title: "Hero MCP binary path resolved from the installer's ambient PATH, not the running binary — wrong-hero and non-portable configs"
slug: codex-mcp-binary-path-resolution
type: bug
status: completed
severity: medium
priority: P2
domain: engineering
root_cause_class: code
created: 2026-07-15
tags: [codex, mcp, install, path-resolution, portability, harness]
relations:
  - target: codex-install-broken
    kind: related
  - target: agents-md-erased-by-snapshot-pointer-writer
    kind: related
  - target: install-integrity-self-check
    kind: related
completed_at: 2026-07-16T07:36:28Z
---

# Hero MCP binary path resolved from the installer's ambient PATH

## Correction (2026-07-16, post-delivery)

**The approach this spec delivered was reversed the same day. The body below
describes the superseded design — read it as history, not as what shipped.**

This spec delivered two things: (1) `findHeroBinary` → `os.Executable()`, and
(2) moving the Codex MCP block to the machine-local User layer
(`~/.codex/config.toml`) with a migration out of the project file. On review
the user rejected the User-layer move as backwards: `~/.codex/config.toml` is
the *user's own* file (their model, plugins, projects, other MCP servers — see
the real example that prompted this), and Hero writing its block there put our
content in the user's personal global config. An MCP server serves a project,
so its wiring belongs in that project's config layer.

**What actually shipped (commit follows this amendment):** all four MCP-writing
targets (cursor, claude, opencode, codex) write the **portable** `command =
"hero"` into their **project-level** config file — the value the code's own doc
comments always described. That travels with the repo (a teammate who clones
gets working wiring), needs no machine-specific path, and makes the whole
resolver moot: `findHeroBinary`, the `os.Executable`/`LookPath` machinery, the
User-layer move, and the migration were all deleted. The residual PATH-roulette
risk (wrong `hero` wins) is what `hero doctor` diagnoses.

This correction also fixed the same latent bug in claude's `.mcp.json`, which
carried an absolute `/Users/.../hero` path — the identical non-portable defect,
now portable too.

Rationale for the reversal is recorded in
[[mcp-config-is-portable-in-project-layer]].

---

## Kickoff

`hero install` writes the MCP config by asking `exec.LookPath("hero")` — whichever hero
happens to be first on the installer's PATH — instead of the hero that's actually running.
The result is a machine-specific absolute path baked into a git-tracked config.

**Status:** delivering — **implemented, awaiting audit+verify.** All four Fix Direction
items landed 2026-07-16: `findHeroBinary` is `os.Executable()`-first with errors surfaced,
the Codex MCP block writes to the User layer (`~/.codex/config.toml`) in both modes, and
project-mode installs migrate Hero's managed block out of the project `.codex/config.toml`
(span-exact, file left in place). Dogfooded on this repo: migration fired, User layer points
at the installing binary, re-run idempotent. See `## Completion Ledger`.

**Pick up at:** `## Completion Ledger` — run the cold delivery audit, then
`hero spec verify codex-mcp-binary-path-resolution`.

→ `.hero/planning/bugs/codex-mcp-binary-path-resolution/spec.md`

**Files:** `internal/install/mcp.go` (resolver + codex writer + migration),
`internal/install/mcp_test.go`, `internal/install/install_test.go`,
`internal/install/main_test.go` + `internal/cli/main_test.go` (package-level HOME isolation
backstop — tests must never write the real `~/.codex/config.toml`), `.codex/config.toml`
(migrated: now whitespace-empty)

**Skip:** don't add `setup_steps` to config.toml — **the key does not exist in Codex** and
unknown keys are silently ignored. Don't build sandbox detection for local Codex — local MCP
servers run *outside* the sandbox and inherit PATH.

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | medium — real defect, but latent on the dogfooding machine. Not the P1 the original spec claimed; its stated mechanism does not exist. |
| **Ease of Fix** | easy — the resolution fix is a few lines. The portability/git-tracking question is a design call, not a code fix. |
| **Caused by our codebase?** | Yes — `findHeroBinary()` in `internal/install/mcp.go`. Nothing to do with Codex's sandbox. |
| **Needs more research?** | **Yes** — no confirmed user-visible failure has been observed. See `## What I could not establish`. |

### Background

Inherited from the superseded `codex-install-broken` (its Fix 5, skipped for two months with
"needs research into Codex sandbox setup_steps support"). That research is now done. **The
original diagnosis was wrong on all three of its load-bearing claims.** A real but different
and less severe defect sits underneath it.

### Analysis

The original spec asserted Hero writes `command = "hero"` and that this fails because Codex
runs in a sandbox with no local PATH. Neither half survives contact with evidence:

- Hero writes an **absolute path**, not `"hero"` (verified on disk and in the writer).
- Local Codex MCP servers run **outside** the sandbox with **PATH inherited** from the
  launching shell (verified in openai/codex source).

What *is* wrong: Hero resolves that absolute path with `exec.LookPath("hero")` — asking the
installer's ambient PATH "where is some hero?" — rather than `os.Executable()`, which would
name **the hero that is doing the installing**. Those are different questions with different
answers, and the code's own doc comment claims it asks the second one.

### Root Cause

**Confirmed.** `internal/install/mcp.go:48-58`:

```go
// findHeroBinary locates the hero binary. Checks:
// 1. The running binary itself      <-- never happens
// 2. PATH lookup
func findHeroBinary() (string, error) {
	// Try to find in PATH
	path, err := exec.LookPath("hero")
	if err == nil {
		return path, nil
	}
	// If we can't find it, return a reasonable default
	return "hero", nil
}
```

Step 1 of the documented contract **is not implemented**. There is no `os.Executable()` call
— confirmed by grep: the codebase uses `os.Executable()` in six other places including
`internal/cli/doctor.go:63`, which exists specifically to report *which binary actually
resolves*. The MCP writer is the one place that needed it and doesn't use it.

Three consequences follow:

1. **Wrong-hero.** `LookPath` returns whichever `hero` is first on the installer's PATH,
   which need not be the running one. **Confirmed on this machine — two different heroes:**
   `/Users/developer/go/bin/hero` (`v0.25.1-4-gfa3b339-dirty`) and `/opt/homebrew/bin/hero`
   (`0.25.1`). Whichever wins is an accident of PATH order at install time.
2. **Non-portable config.** The resolved absolute path is machine-specific and is written
   into a **git-tracked** file. This is the only mechanism by which "MCP unreachable from
   Codex" is actually real — but it bites *teammates and containers*, not sandboxes.
3. **Silent fallback.** When `LookPath` fails, the error is swallowed and the literal string
   `"hero"` is returned with `nil` error. This is the *only* path that produces the config
   the original spec described — and it's an unreported edge case, not the normal case.

### Source

`internal/install/mcp.go` — `findHeroBinary()`, shared by all four MCP-writing targets
(cursor, claude, opencode, codex) via `RegisterMCP`.

### Fix Direction

Resolve the binary from `os.Executable()` rather than the ambient PATH, stop swallowing the
failure, and write the Codex MCP block to the **machine-local User layer**
(`~/.codex/config.toml`) instead of the repo's `.codex/config.toml`. Decided with the user
2026-07-16 — see `## Fix Direction (decided)`.

---

## Problem Statement

Carried from the superseded `codex-install-broken` spec, Evidence #4 / Fix 5, priority P1:

> "This assumes `hero` is in PATH. Codex runs in a sandboxed VM/container — the user's local
> `hero` binary is not present. There is no `setup_steps`, `Dockerfile`, or install script to
> ensure `hero` exists in the Codex environment. The MCP server silently fails to start,
> leaving all `mcp__hero__*` tools unavailable."

This spec was asked to verify that claim rather than trust it. **It does not hold.** Each
sub-claim is addressed under `## Falsified Claims` with citations.

No user-reported reproduction exists. The item was triaged from code reading, not from an
observed failure. That is a material gap — see `## What I could not establish`.

## Environment Details

- Repo: `hero-engine/hero` @ `153ab29`, branch `main`.
- `.codex/config.toml` and `.mcp.json` both present and **git-tracked**.
- Two `hero` binaries on PATH with differing versions (see Root Cause).
- Codex evidence gathered against `openai/codex` @ `1d94125` (2026-07-16) plus hosted docs.
  Note: `developers.openai.com/codex/*` now 308-redirects to `learn.chatgpt.com/docs/*`, and
  in-repo `docs/config.md` is a 9-line stub pointing at the hosted docs — the authoritative
  in-repo config reference is the Rust structs.

---

## Falsified Claims

The house standard for this file (`internal/install/target_codex.go:9-31`) is source-verified
citations. Applying that bar to the original spec:

### Claim 1 — "Hero writes `command = \"hero\"`, assuming PATH" → **FALSE**

The writer interpolates the *resolved* path (`internal/install/mcp.go:238-239`):

```go
heroBlock := fmt.Sprintf("%s\n[mcp_servers.hero]\ncommand = %q\nargs = %s\n%s",
	codexMCPMarker, heroPath, args, codexMCPEndMarker)
```

The actual artifact on disk, and in git (`git show HEAD:.codex/config.toml`):

```toml
# hero:managed
[mcp_servers.hero]
command = "/Users/developer/go/bin/hero"
args = ["mcp"]
# end:hero:managed
```

The literal `"hero"` appears **only** via the swallowed-error fallback at `mcp.go:57`.

### Claim 2 — "Codex runs in a sandbox; the local binary isn't present" → **FALSE for local CLI**

Local Codex MCP servers are launched **outside** the sandbox. `LocalStdioServerLauncher::launch_server`
(`codex-rs/rmcp-client/src/stdio_server_launcher.rs:246-278`) uses a plain `tokio::process::Command`
— no `sandbox-exec`, no landlock wrapper. The executor path states it outright: `sandbox: None`
(`stdio_server_launcher.rs:517`).

Even for sandboxed processes, the seatbelt base policy explicitly permits execution
(`codex-rs/sandboxing/src/seatbelt_base_policy.sbpl:8-13`):

```
(allow process-exec)
(allow process-fork)
```

The sandbox restricts **filesystem writes and network** — not execution, not PATH.

**PATH is inherited.** `create_env_for_mcp_server` (`codex-rs/rmcp-client/src/utils.rs:12-24`)
filters Codex's own environment through an allowlist that includes `PATH`
(`utils.rs:130-142`). So a local Codex CLI session resolves `hero` against the launching
shell's PATH exactly as any other process would.

**Codex Cloud is the different surface** — it runs in a `codex-universal` container
provisioned by a setup script configured in the environment settings UI, not from a repo
file. Hero writes the same `.codex/config.toml` for both, but the local case — which is what
this repo's dogfooding actually uses — is not broken by sandboxing.

### Claim 3 — "`setup_steps`" → **DOES NOT EXIST**

`grep -rn "setup_steps"` over the full openai/codex clone returns **zero hits**. The
authoritative top-level key list is `ConfigToml` (`codex-rs/config/src/config_toml.rs:153-516`);
`setup_steps` is not among them, and the hosted [Configuration Reference](https://learn.chatgpt.com/docs/config-file/config-reference)
does not list it. It was invented by the original spec.

Worse for anyone tempted to try it: **unknown keys are silently ignored.** `ConfigToml` uses
`#[schemars(deny_unknown_fields)]`, which is a JSON-Schema-generation attribute, not serde's
— serde drops unknown keys. Rejection only happens under the opt-in `--strict-config` flag
(`codex-rs/config/src/strict_config.rs:125`). A `setup_steps` block would be a silent no-op.

**No local pre-MCP bootstrap hook exists.** The nearest candidate, `hooks.SessionStart`
(`codex-rs/config/src/hook_config.rs:20-45`), fires too late: hooks are queued into session
state (`codex-rs/core/src/session/session.rs:1278`) and drained at **turn** start
(`codex-rs/core/src/session/turn.rs:188` → `codex-rs/core/src/hook_runtime.rs:103-107`),
whereas the MCP connection manager is constructed earlier during session init
(`session.rs:1221`). A SessionStart hook cannot install a binary in time for MCP startup.

### Claim 4 — "silently fails to start" → **FALSE in the TUI**

Startup failures are classified (`codex-rs/codex-mcp/src/connection_manager.rs:313-330`),
formatted by `mcp_init_error_display` (`connection_manager.rs:1058-1091`), emitted as
`McpStartupUpdate`/`McpStartupComplete`, and rendered. Committed snapshot
`codex-rs/tui/src/chatwidget/snapshots/codex_tui__chatwidget__tests__app_server_mcp_startup_failure_renders_warning_history.snap`:

```
⚠ MCP client for `alpha` failed to start: handshake failed
⚠ MCP startup incomplete (failed: alpha)
```

Default startup timeout is **30s** (`codex-rs/codex-mcp/src/rmcp_client.rs:93`) — note the
hosted MCP docs claim 10; the docs are stale. `required = true`
(`codex-rs/config/src/mcp_types.rs:171`) escalates to a hard error via
`validate_required_servers` (`connection_manager.rs:378-427`).

**Partial carve-out:** `McpStartupUpdate`/`McpStartupComplete` are consumed only by
`codex-rs/tui/` and `codex-rs/rollout-trace/`; `codex-rs/exec/src/` has no handler. So under
`codex exec`, a non-required server failing to start plausibly *is* quiet — server stderr is
logged only at `info!` (`stdio_server_launcher.rs:288`). **UNVERIFIED** — traced by absence
of a handler, not by running the binary. "Silent" may be true for `codex exec` only.

---

## Root Cause Analysis

**Confirmed defect.** `findHeroBinary()` (`internal/install/mcp.go:48-58`) asks the wrong
question. `exec.LookPath("hero")` means *"find some hero on the ambient PATH"*; the correct
question at install time is *"where am I?"* — `os.Executable()`.

The doc comment asserts the correct behavior and the code doesn't implement it. That gap is
the whole bug, and it's why nobody caught it by reading: the comment reads as if it were
already right.

**Evidence for wrong-hero (confirmed):**
```
$ which -a hero
/Users/developer/go/bin/hero        → v0.25.1-4-gfa3b339-dirty
/opt/homebrew/bin/hero             → 0.25.1  (symlink to Cellar/hero/0.25.1)
```
Two heroes, two versions. `LookPath` takes the first. If the user runs
`/opt/homebrew/bin/hero install` while `~/go/bin` precedes it on PATH, the config is wired to
the *other* binary. Nothing warns. This is precisely the failure mode already known to bite
in this project — a stale `~/go/bin/hero` misclassifying spec criteria — generalized: **"hero
is on PATH" was never the requirement; "the right hero is on PATH" is.** `os.Executable()`
makes that requirement unrepresentable-in-the-negative rather than merely hoped for.

**Evidence for non-portability (confirmed):**
`.codex/config.toml` is git-tracked (`git ls-files` matches) and `.gitignore:79-84` excludes
`.codex/agents`, `.codex/commands`, `.codex/skills` — but **not** `config.toml`. The tracked
content hardcodes `/Users/developer/go/bin/hero`. For any teammate, CI runner, or container,
that path does not exist → the MCP server genuinely fails to start → `mcp__hero__*` tools
genuinely unavailable. **The original spec's symptom is real; its mechanism is not.** It
blamed a sandbox that isn't there and missed the absolute path sitting in the file it quoted.

`.mcp.json` (claude) carries the identical `/Users/developer/go/bin/hero` and the identical
fragility, since it shares `findHeroBinary`. **Noted, not fixed here**, per scope — but see
the tripwire note below, because the shared locus makes the distinction mostly academic.

**Hypothesis, not confirmed:** that this has actually bitten a real user. See below.

---

## What I could not establish

Recorded explicitly rather than papered over — the superseded spec's failure was laundering
exactly this kind of gap into asserted fact.

1. **No observed failure.** No user report, log, or reproduction of Hero's MCP being
   unreachable from Codex exists. The item was triaged from code reading. On this machine the
   committed path points at a real, current binary, so the bug is **latent here**. It would
   manifest for a second developer or a container — this repo has had one dogfooder.
2. ~~Whether `.codex/config.toml` / `.mcp.json` are *intended* to be tracked.~~ **RESOLVED
   2026-07-16 by the user:** `.codex/config.toml` is *Codex's* file, not Hero's — it can hold
   the user's own model/approval/MCP settings, so gitignoring it to hide Hero's four lines is
   wrong, and tracking-vs-not is a per-team call Hero shouldn't force. The machine-specific
   wiring moves to Codex's machine-local **User layer** instead. See `## Fix Direction
   (decided)`.
3. **`codex exec` silence** — traced by absence of an event handler, not executed (see
   Claim 4).
4. **The exact OS error text** for a not-found binary interpolated into `{err:#}` (likely
   `No such file or directory (os error 2)`; not executed).

**Needs more research? → No longer blocking.** Item 1 (no observed failure) stands but only
affects priority, not direction. Item 2 — the gate — is resolved above.

---

## Code Flow (End to End)

1. `internal/cli/install.go:396` — `install.RegisterMCP(target, wsOpts)` (also
   `internal/install/install.go:181`).
2. `internal/install/mcp.go:29` — `RegisterMCP` calls `findHeroBinary()` **first**, before any
   target dispatch. Shared by all four MCP targets.
3. `internal/install/mcp.go:52` — `exec.LookPath("hero")` searches the **installer's ambient
   PATH**. ← **defect: should be `os.Executable()`; the doc comment at :48-50 claims it is.**
4. `internal/install/mcp.go:57` — on failure, error swallowed; literal `"hero"` returned with
   `nil`. ← **secondary defect: silent degradation.**
5. `internal/install/mcp.go:41` → `registerMCPCodex(heroPath, opts)` (:220).
6. `internal/install/mcp.go:232` — resolves `<project>/.codex/config.toml` (project) or
   `~/.codex/config.toml` (global).
7. `internal/install/mcp.go:238` — `upsertCodexConfig` interpolates `heroPath` via `%q` into
   the `hero:managed` block.
8. `internal/install/mcp.go:263` — `os.WriteFile`. Machine-specific path now on disk in a
   git-tracked file.
9. *(Codex side)* `codex-rs/rmcp-client/src/stdio_server_launcher.rs:246-278` — spawns
   `command` unsandboxed, PATH inherited. Absolute path either exists (works) or doesn't
   (fails).
10. *(Codex side)* `codex-rs/codex-mcp/src/connection_manager.rs:1089` — on failure, renders
    ``MCP client for `hero` failed to start: {err:#}`` in the TUI.

---

## Key Files

### Defect site (shared across targets)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/install/mcp.go` | 48–58 | `findHeroBinary` — the root cause. Doc comment claims `os.Executable()`; code does `LookPath`. |
| `internal/install/mcp.go` | 26–45 | `RegisterMCP` — resolves the path once, before target dispatch. Why this is not Codex-specific. |

### Codex writer
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/install/mcp.go` | 220–263 | `registerMCPCodex` / `upsertCodexConfig` — interpolates the path into the managed block. |
| `internal/install/mcp.go` | 267–330 | `stripUnmanagedCodexHeroTable` — dedup logic; correct, not implicated. |
| `.codex/config.toml` | 1–5 | The real artifact. Absolute path, git-tracked. |
| `internal/install/target_codex.go` | 9–46 | House-standard citation block. `runCodex` does **not** wire MCP — `RegisterMCP` is a separate path. |

### The correct pattern, already in-repo
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/doctor.go` | 43, 63 | `os.Executable()` — `hero doctor` exists to report which binary actually resolves. The pattern the MCP writer should have used. |
| `internal/graph/graph.go` | 334–343 | Same self-locating pattern for mismatch errors. |

### Comparison target (noted, out of scope)
| File | Lines | Relevance |
|------|-------|-----------|
| `.mcp.json` | 1–9 | Claude config — identical hardcoded `/Users/developer/go/bin/hero`, identical fragility. |

---

## Fix Options

Direction is decided — see `## Fix Direction (decided)` below. The original spec's three
sketched options are recorded here as rejected so they aren't proposed a third time.

### Rejected outright (falsified by evidence)
- **(A) `setup_steps` in config.toml installing hero from a release URL** — **impossible.**
  The key does not exist; it would be silently ignored (Claim 3).
- **(B) Detect the sandbox and emit `codex-setup.sh`** — **solves a non-problem** for local
  Codex. MCP servers run outside the sandbox with PATH inherited (Claim 2). Would add
  detection machinery for a failure mode that isn't there.
- **(C) Ship a static Go binary at a known download URL** — **out of scope and unjustified.**
  A distribution change proposed to fix a sandbox that doesn't apply. If Codex *Cloud* support
  is wanted, that's its own spec with its own justification.

## Fix Direction (decided)

Decided with the user 2026-07-16, after verifying Codex's config layering against
openai/codex @ `1d94125`:

**Codex has a machine-local config layer, and that is where machine-specific wiring belongs.**
The effective config is deep-merged from layered sources — `ConfigLayerSource`
(`codex-rs/config/src/config_layer_source.rs:6-27`) with precedence MDM(0) < System(10) <
EnterpriseManaged(15) < **User = `~/.codex/config.toml`(20)** < **Project =
`<repo>/.codex/`(25)** < SessionFlags(30). The merge is a recursive per-key table merge
(`merge_toml_values`, `codex-rs/config/src/merge.rs:7-35`), so an `[mcp_servers.hero]` block
in the User layer is live in every project on the machine and coexists with any project-layer
MCP servers; only an identically-named `mcp_servers.hero` in a project would override it.

Therefore:

1. **Write the Codex MCP block to `~/.codex/config.toml` (User layer), not the project's
   `.codex/config.toml`** — on both project and global installs. The writer already targets
   this file in global mode (`registerMCPCodex`, `mcp.go:232`); project mode changes to do the
   same for MCP only. A machine-specific absolute path in a machine-local file is correct, and
   one block serves every Hero project on the box (`hero mcp` resolves the workspace from the
   session's cwd).
2. **Migration: remove Hero's managed MCP block from the project `.codex/config.toml`.** The
   file is the user's — it can carry their model/approval/other-MCP settings — and Hero only
   ever owned its marked block. Remove only the `# hero:managed` … `# end:hero:managed` span;
   never touch anything outside it, and never gitignore the file (rejected: it would ignore
   the *user's* config to hide Hero's four lines). Whether the file is tracked stays a
   per-team decision Hero doesn't make.
3. **`findHeroBinary`: `os.Executable()` first, `LookPath` as fallback.** Matches the doc
   comment already in the file and the `doctor.go:63` precedent; guarantees the User-layer
   block points at *the hero that did the install*, killing wrong-hero.
4. **Stop swallowing the `LookPath` failure.** Return the error instead of `"hero", nil`;
   degrading silently to a string that may not resolve is what made this invisible.

Documented variant for teams that want clone-and-go shared wiring: a portable
`command = "hero"` block in the project file — works for local Codex per Claim 2 (PATH is
inherited), at the cost of PATH roulette, which `hero doctor` diagnoses. Not the default.

Out of scope but same question: `.mcp.json` (claude) carries the identical absolute path.
When it's addressed, apply the same principle — machine-specific values belong in
machine-local scope (Claude Code's user-scope MCP config), shared files carry only portable
values.

### Tripwire note — `harness-changes-cover-all-targets` [high]

Called `hero_anchor` before proposing direction, as required. The tripwire **applies and is
satisfied, not scoped around.** The task framing anticipated a Codex-specific fix needing an
explicit exemption; the evidence says otherwise. `findHeroBinary` is called by `RegisterMCP`
*before* target dispatch (`mcp.go:29`), so the defect is shared by all four MCP-writing
targets (cursor/claude/opencode/codex) and any fix at that locus covers them inherently. There
is no Codex-specific version of this fix to scope. Copilot and generic write no MCP config, so
six-target coverage is complete by construction.

This is worth stating plainly because it cuts against the instruction to keep the fix
Codex-only: **the diagnosis is scoped to Codex reachability, but the defect is not
Codex-specific, and fixing it where it lives is not scope expansion — it's the correct
locus.** Deliberately *not* folded in: the `.mcp.json` portability decision, which is the same
design question as above and should be answered once for both.

---

## Secondary Defects

1. **Doc comment asserts unimplemented behavior** (`mcp.go:48-50`). "Checks: 1. The running
   binary itself" — never happens. A comment that describes the correct design while the code
   does something else is worse than no comment: it stops the next reader from looking.
2. **Swallowed error** (`mcp.go:57`) — `LookPath` failure returns `"hero", nil`. The `error`
   return of `findHeroBinary` is dead: it can never be non-nil.
3. **`upsertMCPConfig` treats malformed JSON as absent** (`mcp.go:127-131`) — a corrupt
   `.mcp.json` is silently reset, discarding the user's other MCP servers. Same shape of
   defect as the `[mcp_servers.hero]` duplicate-table bug that `stripUnmanagedCodexHeroTable`
   was written to fix. **Not this bug; not fixed here.** Plausibly belongs with
   `install-integrity-self-check`.

---

## Test Plan

### Existing test review
- `internal/install/mcp_test.go` — solid coverage of `upsertCodexConfig` marker/dedup
  semantics (duplicate tables, preserving `[mcp_servers.other]`, re-upsert idempotency).
  **All of it passes `heroPath` in as a parameter, so none of it exercises `findHeroBinary`.**
  The defect is upstream of every existing test. That's why it survived.
- `internal/install/install_test.go:1521-1549` — asserts `[mcp_servers.hero]` *exists* and is
  not duplicated. Never asserts the `command` value is correct or resolvable.

The missing test class: *"the config points at a binary that exists and is the right one,"* as
distinct from *"the config is well-formed."* Same shape as the gap in
`agents-md-erased-by-snapshot-pointer-writer` — the suite modeled structure, not truth.

### Tests needed
1. `findHeroBinary` returns `os.Executable()` when it differs from the `LookPath` hit —
   the wrong-hero guard. Needs the resolver injectable, or an integration test with a fake
   hero earlier on `PATH`.
2. `findHeroBinary` surfaces an error rather than the literal `"hero"` when resolution fails
   (pairs with option 2).
3. Table-driven across cursor/claude/opencode/codex asserting the written `command` is an
   existing executable — the cross-target guard the tripwire wants.

### Regression scope
- All four MCP-writing targets — shared resolver.
- `--project-root` arg threading (`mcp.go:140-143`, `:234-236`) is untouched by the resolver
  change but travels with it.
- If the portability question resolves toward gitignoring, `.codex/config.toml` and
  `.mcp.json` leave the tree — check nothing in CI or the smoke suite reads them.

---

## Notes

The instructive part is *why the original diagnosis felt right*. It observed a true symptom
class (Hero MCP not reachable from Codex), reached for a plausible mechanism (sandbox, no
PATH), and never checked the file it was quoting — which contained an absolute path that
contradicts the claim in its own first line. It then invented `setup_steps` to fix the
imagined mechanism and hedged with "if Codex supports it." The hedge was the tell, and it sat
unexamined for two months.

This mirrors `agents-md-erased-by-snapshot-pointer-writer` exactly: the same superseded spec
blamed the writer that should FILL a file rather than the code that EMPTIES it. Both errors
share a signature — **a plausible mechanism accepted without checking the artifact on disk.**

## Recap

`hero install` resolves the MCP binary path with `exec.LookPath("hero")` instead of
`os.Executable()`, despite a doc comment claiming otherwise, so the config can point at a
different hero than the one installing and always bakes in a machine-specific absolute path —
which lands in a git-tracked file and is therefore broken for anyone but the author. The
original P1 framing (Codex sandbox, no PATH, `setup_steps`, silent failure) is falsified on
every point: `setup_steps` does not exist, local Codex MCP runs outside the sandbox with PATH
inherited, and failures are surfaced in the TUI. Real defect, medium severity, latent on the
only machine that has run it.

**Direction decided 2026-07-16:** MCP block moves to Codex's machine-local User layer
(`~/.codex/config.toml`, deep-merged with project config), Hero's managed block is removed
from the project `.codex/config.toml` (the file is the user's), and the resolver becomes
`os.Executable()` with errors surfaced. Ready to deliver.

---

## Completion Ledger

Delivered 2026-07-16. Stack: Go. Validation: `go build ./...`,
`go test -race -count=1 ./internal/install/ ./internal/cli/` (ok, 3.6s / 55.0s),
full `go test -count=1 ./...` exit 0 (the known `internal/serve/pages/now/data`
time-of-day flake did not fire). Dogfooded on this repo with a fresh
`go build -o hero ./cmd/hero`.

**Delivery note — one scoped refinement:** workspace-mode installs
(`opts.ProjectRoot != ""`) keep writing the codex block to the *workspace's*
`.codex/config.toml`. Their `--project-root` arg is project-specific, not
machine-generic — putting it in the User layer would pin every codex session on the
machine to one repo. Codex's Project layer overriding the User layer per-key is exactly
the right home for it. Normal project and global installs both write the User layer as
decided.

**Delivery note — leak found and sealed:** with project-mode installs now writing
`~/.codex/config.toml`, several tests that run codex installs outside the shared harness
(e.g. `prune_test.go`) rewrote the REAL user config to point at a transient go-test
binary — observed live during delivery. Fixed with package-level `TestMain` HOME
isolation in both `internal/install` and `internal/cli` (backstop), plus explicit
`t.Setenv("HOME", t.TempDir())` in the directly-affected tests. The user's real config
was restored byte-identical (verified against a pre-run backup).

### Acceptance Criteria

| # | Fix Direction item | Status | Note |
|---|---|---|---|
| 1 | Codex MCP block written to User layer (`~/.codex/config.toml`) on both project and global installs | DONE | `internal/install/mcp.go` `registerMCPCodex` — both modes target the User layer (workspace mode excepted, see delivery note). Tests: `TestRegisterMCP_Codex_Project` (block in `$HOME`, NOT in project), `TestRegisterMCP_Codex_Idempotent`. Dogfood: block landed in real `~/.codex/config.toml`. |
| 2 | Migration: remove Hero's managed span from project `.codex/config.toml`; bytes outside untouched; file left in place; one-line notice | DONE | `removeCodexManagedBlock` in `mcp.go` — span-exact removal (+ its trailing newline), never deletes the file, prints `moved hero MCP block: ... -> ...`. Tests: `TestRegisterMCPCodex_MigratesProjectBlockToUserLayer` (user bytes byte-identical), `TestRegisterMCPCodex_MigrationBlockOnlyFileLeftInPlace`. Dogfood: repo `.codex/config.toml` now present + empty, notice printed once, second run silent. |
| 3 | `findHeroBinary`: `os.Executable()` first (symlinks resolved, matching `doctor.go`), `LookPath` fallback; doc comment fixed | DONE | `mcp.go` `findHeroBinary` — `os.Executable` + `filepath.EvalSymlinks`, PATH only as fallback; doc comment now describes the actual behavior. Test: `TestFindHeroBinary_PrefersRunningBinaryOverPATH` (decoy hero earlier on PATH loses), `TestFindHeroBinary_FallsBackToPATH`. Benefits all four MCP targets: `TestRegisterMCP_CommandPointsAtRunningBinary_AllTargets` (cursor/claude/opencode/codex). |
| 4 | Stop swallowing `LookPath` failure — return a real error, callers handle it | DONE | `findHeroBinary` returns `fmt.Errorf(...)` when both resolutions fail; the `"hero", nil` fallback is gone. `RegisterMCP`'s existing error handling (previously dead) is now live. Test: `TestFindHeroBinary_ErrorWhenUnresolvable` (injectable seams `osExecutable`/`execLookPath`). |

### Changes

| # | Change | Status | Note |
|---|---|---|---|
| 1 | `internal/install/mcp.go` — resolver rewrite, User-layer codex writer, `removeCodexManagedBlock` migration helper, test seams | DONE | ~90 lines changed/added. `upsertCodexConfig`/`stripUnmanagedCodexHeroTable` semantics untouched (all 6 pre-existing marker/dedup tests pass unmodified). |
| 2 | `internal/install/mcp_test.go` — resolver, migration, cross-target tests | DONE | +7 tests: 3 × `TestFindHeroBinary_*`, 2 × `TestRegisterMCPCodex_Migrat*`, `TestRegisterMCP_CommandPointsAtRunningBinary_AllTargets` (4 subtests), helper `testExecutable`. |
| 3 | `internal/install/install_test.go` — codex MCP assertions updated for User-layer write | DONE | `TestRegisterMCP_Codex_Project` now asserts $HOME placement + running-binary command + no project block; `TestRegisterMCP_Codex_Idempotent` reads $HOME; `TestRunCodexProject` HOME-isolated. |
| 4 | `internal/install/main_test.go`, `internal/cli/main_test.go` (new) + `harness_test.go`, `install_hooks_test.go`, `trust_test.go` — HOME isolation | DONE | Package-level `TestMain` backstops + targeted `t.Setenv("HOME", ...)`. Verified empirically: cleaned real config, ran both packages, file untouched. |
| 5 | Dogfood migration of this repo's `.codex/config.toml` + real `~/.codex/config.toml` | DONE | Tracked file lost the block (whitespace-empty, still present); real User layer gained the block pointing at the freshly built binary. `./hero check` unchanged from pre-change baseline (verified via stash round-trip); `./hero doctor` reports sensibly and its wrong-hero WARNING now demonstrates exactly the case the fix guards. |

### Exercise-the-feature check

- [x] Exercised end-to-end: `go build -o hero ./cmd/hero && ./hero install project . --target codex --force` on this repo — migration notice printed, project `.codex/config.toml` emptied (file kept), real `~/.codex/config.toml` gained the managed block with `command = "/Users/developer/projects/hero-engine/repository/hero/hero"`; second run printed no migration notice and left exactly one block; `./hero check` clean relative to baseline; `./hero doctor` verdict OK.

### Excellence Bar self-check

Yes — the fix lands at the shared locus (all four MCP targets), the migration is
byte-surgical with the user's file treated as theirs, the silent-fallback failure mode is
gone, and the delivery caught + permanently sealed a live "tests rewrite the developer's
real config" hazard that the change itself would otherwise have introduced.
