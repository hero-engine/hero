---
title: "Agents confabulate a schema/version narrative when a stray hero binary meets a schema-4 graph"
slug: agent-hero-version-schema-confusion
type: bug
status: completed
priority: high
severity: high
domain: engineering
created: 2026-07-13
origin: session
root_cause_class: design
severity_detail: "high — silently makes hero feel unreliable inside Codex; agents invent a false 'schema 2' migration story and burn turns; no data loss but erodes trust in the tool"
size: large
tags: [mcp, watchdog, schema, version, path, codex, terminal-fallback, self-location, doctor, cross-codebase, harness-agnostic]
relates-to:
  - hero-mcp-orphan-no-parent-liveness
  - desktop-sidebar-mcp-not-running
  - codex-install-broken
root_cause_class_detail: "multi-cause — code (Defect 1 watchdog can't reap live redundant children; Defect 3 wrong remediation string + no self-location), env/process (Defect 2 GUI PATH skew binds a stale binary), design (no `hero doctor`, no version/schema stamping on the MCP surface, terminal-mode fallback undetected)"
completed_at: 2026-07-13T16:20:50Z
---

# Agents confabulate a schema/version narrative when a stray hero binary meets a schema-4 graph

## Summary

### Categorization
| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — no data loss, but hero feels unreliable inside Codex: agents invent a false "your graph is schema 2, run `hero upgrade`" narrative, follow a remediation that cannot work, and burn turns. Directly violates the mission ("start smarter than the last session ended"). |
| **Ease of Fix** | moderate — the engine-side self-location + `hero doctor` + version/schema stamping are small, harness-agnostic Go changes. Defect 1 (MCP dedup/supersede) is a moderate design change against the existing watchdog. Defect 2 is fundamentally environmental (PATH) — the fix is *detection + loud self-locating failure*, not owning the caller's env. The harness-routing part (Defect 4) must cover all six install targets. |
| **Caused by our codebase?** | Partially. Defects 1 and 3 are code/design defects in this repo. Defect 2 is a macOS GUI-PATH-inheritance fact hero cannot own — but hero *can* detect it and fail loudly instead of silently binding the wrong binary. |
| **Needs more research?** | Yes — for the confirming repro only. The literal `db=X binary=Y` error text and `command -v hero; hero --version` output must be captured from *inside a Codex terminal session in hero-code*. That evidence is not reproducible from this repo or the developer's login shell. All three code-level defects are already confirmed by reading source. |

### Background
The developer reported hero behaving as if "the installed CLI understands schema 2 while the workspace graph is schema 4" — a report they **cannot reproduce from their own terminal**, and which rebuilding `~/go/bin/hero` does not fix. Both Claude and Codex have been observed confabulating a version-migration narrative around this. Live investigation on `hero-engine/repository/hero` and its sibling `hero-code` found **three separate, interacting defects** that together produce the symptom and make it un-diagnosable in-agent.

This is not one bug. It is a code defect (redundant live MCP daemons never reaped), an environment defect (a GUI subprocess resolving a stale `hero` off the macOS default PATH), and a design defect (a schema-mismatch error whose remediation is actively wrong, with no `hero doctor` and no version/schema stamped anywhere an agent can see it). The design defect is what turns a recoverable environment mismatch into an agent hallucination.

### Analysis
- **Defect 1 (code):** The parent-death watchdog reaps *orphans* only. It cannot reap redundant *live* children of a *live* client, so a long-running client that reconnects accumulates N `hero mcp` daemons forever.
- **Defect 2 (env/process):** Codex is a GUI app (`/Applications/ChatGPT.app/.../codex`). `launchctl getenv PATH` is empty, so GUI subprocesses inherit the macOS default `/usr/bin:/bin:/usr/sbin:/sbin` — which contains neither `~/go/bin` nor Homebrew. A bare `hero` invoked from a Codex terminal-mode fallback can resolve to a *different, older* binary than the developer's login shell resolves. Schema-2-era binaries exist on disk.
- **Defect 3 (design):** When an old binary opens a schema-4 graph, `graph.go:282` errors with a remediation (`run hero upgrade`) that upgrades workspace overlay files, not the binary — so following it does nothing. The error prints neither *which* binary is complaining (`os.Executable()`) nor its schema. There is no `hero doctor` to reconcile which-binary/which-graph. So the agent has no ground truth and confabulates one.

### Root Cause
**A design gap: hero is neither self-locating nor self-consistent about which binary is running against which graph, and it offers a false remediation when they disagree.** Layered on top are one code defect (live-parent MCP daemon accumulation) and one environmental precondition (GUI PATH skew binding a stale binary). The environmental precondition is the *trigger*; the design gap is why the trigger becomes an un-diagnosable agent hallucination instead of a one-line "you're running the wrong binary, here's its path."

### Source
- `internal/graph/graph.go:269-284` — schema-mismatch error (wrong remediation, no self-location) and the inconsistent tolerate-newer warn branch.
- `internal/serve/mcp_watchdog.go:34-54` — orphan-only watchdog; no live-parent dedup.
- `internal/serve/mcp.go:54-73`, `internal/cli/mcp.go:35-63` — MCP server construction; no per-workspace singleton/supersede.
- `internal/serve/mcp_lifecycle.go:125-136` — `initialize` result stamps `ServerInfo.Version` but no schema version.
- `internal/cli/upgrade.go:27-60` — proves `hero upgrade` upgrades overlay files, not the binary.
- No `hero doctor` command exists (confirmed — no registration in `internal/cli/`).

### Fix Direction
Make hero **self-locating and self-consistent** (print `os.Executable()`, binary version, binary schema, graph schema, and a *true* remediation in both the error and a new `hero doctor`); **stamp version + schema on the MCP `initialize` surface** so a harness can *see* skew instead of inventing it; **dedup/supersede redundant MCP daemons** per client+workspace; and **steer agents to the MCP surface** (harness-agnostic, all six targets) with detection of terminal-mode fallback. Split by concern — see Suggested Fix Approach and Recommended Sequencing.

---

## Problem Statement

The reported symptom: hero acts as though "the installed CLI is schema 2 while the workspace graph is schema 4," producing a schema-version-mismatch error that suggests running `hero upgrade`. Running `hero upgrade` doesn't help. Rebuilding `~/go/bin/hero` doesn't help. The developer cannot reproduce it from their own terminal. Agents (both Claude and Codex) respond by narrating a version-migration story that is not grounded in any observable fact.

Live investigation found three defects:

### Defect 1 — Stale MCP daemon accumulation (live-parent reaping gap)
On `hero-code`, Codex (pid 94751, ALIVE) is the parent of **five** simultaneous live `hero mcp` daemons (16214, 17381, 22448, 22879, 23623). Each has live stdio pipes — `lsof` shows fds 0/1/2 all connected pipes, not broken. They are not orphans.

The parent-death watchdog (`internal/serve/mcp_watchdog.go:34-54`) exits **only** when `getppid()` changes from its startup value — i.e. only when the *original parent died* and the process was reparented to launchd/init. It structurally cannot reap a redundant, still-connected child of a *still-alive* client. A long-running client that opens multiple sessions or reconnects therefore accumulates one daemon per connection, forever. The prior reaping work (`hero-mcp-orphan-no-parent-liveness`) only ever scoped orphan cleanup — this live-duplication case was out of scope and remains uncovered.

Note: all five daemons are `~/go/bin/hero`, schema 4, and read the graph fine. **The daemons are not the source of the "schema 2" report** — they are a separate resource-leak defect that compounds the "hero feels flaky in Codex" impression.

### Defect 2 — Terminal-fallback PATH skew binds the wrong hero binary
Codex is a GUI app launched from `/Applications/ChatGPT.app/Contents/Resources/codex`. For `hero-code` it is configured `= "terminal"` in `~/.codex/config.toml`, so besides the MCP connection it also shells out to a bare `hero` command.

`launchctl getenv PATH` is **empty** → the GUI-inherited PATH is the macOS default `/usr/bin:/bin:/usr/sbin:/sbin`, which contains **neither** `~/go/bin` **nor** Homebrew. The developer's interactive login shell resolves `hero` → `~/go/bin/hero` (schema 4). A GUI subprocess resolves `hero` differently — potentially to a **stale/old** binary. Schema-2-era binaries exist on disk (e.g. `/Users/developer/projects/personal/repository/hero/dist/hero_darwin_arm64_v8.0/hero`; a Homebrew 0.8.1 existed until it was uninstalled this session).

This is the most likely source of the "installed CLI understands schema 2, graph is schema 4" report: a stale binary, resolved only inside the GUI subprocess, reading the current schema-4 graph. It explains every anomaly: the developer can't reproduce it (their shell resolves the right binary), and rebuilding `~/go/bin/hero` doesn't fix it (Codex resolves a *different file*).

**Constraint:** hero cannot own the caller's PATH. The fix here is **detection and loud, self-locating failure**, not silently owning env.

### Defect 3 — Schema-mismatch error is a non-actionable dead-end
`internal/graph/graph.go:282-283`:
```go
return fmt.Errorf("graph schema version mismatch: db=%s binary=%s — "+
    "run `hero upgrade` or rebuild the binary", currentVersion, schemaVersion)
```
But `hero upgrade` (`internal/cli/upgrade.go:27-60`) is documented "Upgrade the workspace to match the current hero binary" — it regenerates workspace **overlay files** (agents/commands/skills, CLAUDE.md/AGENTS.md) and stamps `version.json`. It does **not** rebuild or replace the binary, and it cannot reach a stray binary on a different PATH. So the primary suggested remediation is actively wrong for the actual failure.

The error also does **not** print:
- `os.Executable()` — *which* hero binary is complaining,
- the offending binary's version,
- any hint that the problem might be *which binary is on PATH* rather than *which schema the graph is at*.

And there is **no `hero doctor`** to reconcile which-binary / which-graph / do-they-agree. With no ground truth surfaced, the agent invents one.

**Inconsistent sibling path:** `graph.go:270-280` handles the *opposite* direction (db newer than binary) by warning to stderr and continuing. So db-newer-than-binary is tolerated-and-continues while binary-newer-than-db is a hard error — the two directions give inconsistent guidance for what is fundamentally the same "binary and graph disagree" situation.

## Environment Details
- macOS (Darwin 25.5.0). Codex runs as a GUI app under ChatGPT.app.
- `launchctl getenv PATH` empty → GUI default PATH excludes `~/go/bin` and Homebrew.
- `~/.codex/config.toml` sets hero `= "terminal"` for hero-code (terminal-mode fallback active alongside MCP).
- hero-code graph is schema 4, continuously upgraded; it did **not** sit at schema 2.
- Multiple hero binaries on disk at different schema eras (schema-2 dist build; a since-removed Homebrew 0.8.1).

---

## Root Cause Analysis

### Confirmed by reading source (this session)
1. **Watchdog is orphan-only.** `mcp_watchdog.go:47-49` fires `watchdogExit(0)` iff `watchdogGetppid() != startPpid`. There is no branch that considers "another live daemon already serves this client+workspace." `internal/cli/mcp.go:35-63` constructs a fresh `MCPServer` on every launch with no pidfile, no singleton, no supersede; `mcp.go:62` derives `sessionID` from `os.Getpid()`, so every launch is unique and uncoordinated. → **Defect 1 confirmed: nothing dedups live redundant daemons.**
2. **`hero upgrade` does not touch the binary.** `upgrade.go:27-60` (Short/Long) + body: it resolves install targets, regenerates overlay content via `install.Run`, and stamps `version.json`. No binary rebuild/replace anywhere. → **Defect 3 remediation confirmed wrong.**
3. **Error omits self-location.** `graph.go:282-283` formats only `currentVersion` (db) and `schemaVersion` (the constant, `graph.go:116 = "4"`). It never calls `os.Executable()` nor prints the binary's semantic version. → **Defect 3 self-location gap confirmed.**
4. **Inconsistent directions.** `graph.go:270-280` warns+continues when db > binary; `graph.go:282` hard-errors when binary > db. → **confirmed.**
5. **MCP `initialize` doesn't stamp schema.** `mcp_lifecycle.go:125-136` sets `ServerInfo.Version = s.version` but no schema version; `s.version` is `rootCmd.Version` (`internal/cli/mcp.go:51`). A harness sees the binary version but never the schema, and never the graph schema. → **Fix direction #2 is feasible and confirmed absent.**
6. **No `hero doctor`.** No command registration for `doctor` in `internal/cli/`. → **confirmed absent.**

### Environmental (confirmed by the investigation's live evidence, not re-derivable from this repo)
7. **GUI PATH skew.** Empty `launchctl` PATH + terminal-mode fallback + multiple on-disk binaries → a Codex subprocess can bind a stale `hero`. This is the trigger for the "schema 2" report. → **Defect 2: environmentally confirmed; the exact stale binary Codex binds is the one open repro item (see below).**

### Secondary defect (found while reading — not the reported bug)
8. **Schema comparison is a string comparison.** `graph.go:270`: `if currentVersion > schemaVersion` compares the version *strings* lexically. Correct for single digits ("4" > "2"), but at schema version 10 this breaks: `"10" > "9"` is `false` and `"10" > "2"` is also `false` lexically, so a schema-10 db read by a schema-9 binary would fall through to the hard-error branch instead of the tolerate-newer warn branch — inverting the intended behavior. Latent today (schema is single-digit) but a live trap the moment schema reaches double digits. Fix alongside the self-location work since both live in the same block.

### Hypothesis vs. confirmed
- **Confirmed:** Defects 1, 3, secondary #8 (source-level). Defect 2's *mechanism* (empty launchctl PATH, terminal fallback, stale binaries on disk).
- **Hypothesis (strongly supported, one repro step open):** that the specific "schema 2 vs schema 4" report was produced by Codex binding a specific stale schema-2 binary. Confirming requires the literal error text and PATH resolution *from inside Codex* (see Open Repro Step). Do **not** fabricate the `db=`/`binary=` values.

---

## Code Flow (End to End)

**How the "schema 2" report is produced (hypothesized trigger + confirmed handling):**
1. Codex GUI process starts with PATH = macOS default (empty `launchctl` PATH). `~/go/bin` and Homebrew are absent from PATH.
2. Codex, in terminal-mode fallback (`~/.codex/config.toml` hero `= "terminal"`), shells out to bare `hero <cmd>` in `hero-code`.
3. Shell resolves `hero` off the GUI PATH → a **stale** binary (schema-2 era), not `~/go/bin/hero`.
4. The stale binary opens the schema-4 graph → `internal/graph/graph.go` migration check.
5. `graph.go:269` `currentVersion (4) != schemaVersion (2)`; `graph.go:270` `currentVersion > schemaVersion` — but this stale binary's `schemaVersion` is "2", so `"4" > "2"` is true → it takes the **warn-and-continue** branch (`graph.go:276-280`) OR, depending on the exact stale build, the hard-error branch. Either way the binary/graph disagreement surfaces as text with no self-location.
6. Agent (Claude or Codex) reads the mismatch text, has no `os.Executable()` path, no `hero doctor`, no schema on the MCP surface → **confabulates** a "graph is schema 2, run `hero upgrade`" narrative.
7. Developer runs `hero upgrade` from their *own* shell (correct binary) → overlay files regenerate, binary is untouched, graph already schema 4 → nothing changes. Rebuilding `~/go/bin/hero` also changes nothing because Codex binds a different file. Loop.

**Parallel resource leak (Defect 1):**
1. Codex opens an MCP session → launches `hero mcp` (`internal/cli/mcp.go:35`).
2. `mcp.go:61` `NewMCPServerWithFilter(...)`; `mcp_lifecycle.go:40-44` starts the parent watchdog only in real-stdio mode.
3. Codex reconnects / opens another session → a *second* `hero mcp` launches. Neither is an orphan; both parents (Codex) are alive.
4. `mcp_watchdog.go:47` never fires for either (ppid unchanged). Repeat → five live daemons.

---

## Key Files

### Schema / version reconciliation (engine, harness-agnostic)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/graph/graph.go` | 116 | `schemaVersion = "4"` — the binary's compiled schema. |
| `internal/graph/graph.go` | 269-284 | Mismatch handling: wrong remediation, no self-location, string comparison (secondary defect), inconsistent two-direction guidance. |
| `internal/cli/upgrade.go` | 27-60 | Proves `hero upgrade` upgrades overlay files/version.json, not the binary — the misleading remediation target. |
| `internal/version/` | — | `version.Read`/`CompareVersions` — reuse for a correct semantic compare and for `hero doctor` version display. |

### MCP surface (engine, harness-agnostic)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_lifecycle.go` | 125-136 | `initialize` result — where to stamp binary version + schema + graph schema on `ServerInfo`/`_meta`. |
| `internal/serve/mcp.go` | 29-73 | `MCPServer` struct + constructors; `version` field carried from `rootCmd.Version`. Home for a workspace-singleton/supersede handle. |
| `internal/cli/mcp.go` | 35-63 | `hero mcp` entrypoint; where a per-client+workspace dedup/pidfile check would gate a new daemon. |

### MCP daemon lifecycle (Defect 1)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/mcp_watchdog.go` | 34-54 | Orphan-only watchdog; cannot reap live redundant children. |
| `internal/serve/mcp_watchdog_other.go` | 1-9 | darwin path relies solely on the ppid poll — relevant because macOS is where the accumulation was observed. |
| `internal/serve/mcp_lifecycle.go` | 40-44 | Where the watchdog is started; a supersede handshake would live near here or in `mcp.go` startup. |
| `internal/serve/mcp_watchdog_test.go` | — | Existing watchdog test coverage to extend without regressing orphan reaping. |

### Harness routing (Defect 4 — all six targets)
| File | Lines | Relevance |
|------|-------|-----------|
| `.hero/` knowledge/convention source | — | Author-once location for "prefer the MCP surface; if you see a schema/version mismatch, run `hero doctor` — do NOT invent a migration story" guidance, propagated by `hero install` to all six targets. |
| `internal/cli/install.go` / `internal/install/` | — | Propagation to `opencode | cursor | claude | copilot | codex | generic`. |

---

## Secondary Defects
1. **String comparison of schema versions** (`graph.go:270`) — latent double-digit-schema inversion bug (detail #8 above). Fix with the self-location work.
2. **Inconsistent two-direction guidance** (`graph.go:270-280` warn+continue vs `graph.go:282` hard-error) — same disagreement, opposite handling. `hero doctor` should reconcile both into one coherent report.
3. **MCP `initialize` exposes version but not schema** (`mcp_lifecycle.go:125-136`) — the harness has a version number it can't act on and no schema at all. Not the reported bug, but the reason skew is invisible to agents.
4. **Terminal-mode fallback is undetected** — when a client uses both MCP and terminal-mode against the same workspace, hero has no way to notice the terminal path might bind a different binary than the MCP path. Candidate signal for `hero doctor` and for the harness-routing guidance.

---

## Suggested Fix Approach

Scope note: fixes 1–3 are **engine-side, harness-agnostic**. Fix 4 is **harness-facing** and MUST cover all six install targets (`opencode | cursor | claude | copilot | codex | generic`) per tripwire `harness-changes-cover-all-targets`. Consider splitting into separate specs — see Recommended Sequencing.

### 1. Self-locating, self-consistent schema-mismatch error + `hero doctor` (engine)
**File:** `internal/graph/graph.go`, function containing the migration check (around lines 269-284).

**Before:**
```go
if currentVersion != schemaVersion {
    if currentVersion > schemaVersion {
        fmt.Fprintf(os.Stderr,
            "Warning: graph schema is newer than this binary (db=%s, binary=%s). "+
                "Update your hero binary to silence this warning.\n",
            currentVersion, schemaVersion)
        return nil
    }
    return fmt.Errorf("graph schema version mismatch: db=%s binary=%s — "+
        "run `hero upgrade` or rebuild the binary", currentVersion, schemaVersion)
}
```

**After (shape — engineer to finalize wording/helpers):**
```go
if currentVersion != schemaVersion {
    exe, _ := os.Executable() // which binary is actually running
    // Use a numeric/semver-aware compare, not lexical string compare.
    if schemaVersionLess(schemaVersion, currentVersion) { // db newer than binary
        fmt.Fprintf(os.Stderr,
            "Warning: graph schema is newer than this binary.\n"+
                "  running binary: %s (schema %s)\n"+
                "  graph schema:   %s\n"+
                "This binary is older than the workspace. Point your harness at a "+
                "newer hero, or run `hero doctor` to see which binary is on PATH.\n",
            exe, schemaVersion, currentVersion)
        return nil
    }
    return fmt.Errorf(
        "graph schema version mismatch — the hero binary is older than the workspace graph.\n"+
            "  running binary: %s (schema %s)\n"+
            "  graph schema:   %s\n"+
            "This is almost always the WRONG hero binary on PATH, not a workspace problem.\n"+
            "Run `hero doctor` to see which binary your shell/harness resolves and how to fix it.\n"+
            "(`hero upgrade` will NOT help — it updates workspace files, not this binary.)",
        exe, schemaVersion, currentVersion)
}
```
**Why:** Prints the offending binary path + both schemas, replaces the false `hero upgrade` remediation with `hero doctor`, and fixes the lexical comparison (secondary defect). Explicitly names the likely real cause (wrong binary on PATH).

**New:** `internal/cli/doctor.go` — `hero doctor` that prints, in one report:
- `os.Executable()` and its resolved real path,
- what `command -v hero` / PATH resolution yields vs. `os.Executable()` (flag when they differ — the Defect 2 signal),
- binary semantic version (`rootCmd.Version`) and compiled `schemaVersion`,
- the workspace graph schema,
- a verdict line: agree / binary-older / binary-newer, with the true remediation for each,
- (stretch) whether a terminal-mode fallback is configured for a client that also uses MCP.

This is harness-agnostic (a CLI command), so the all-targets tripwire does not gate it.

### 2. Stamp binary version + schema on the MCP surface (engine)
**File:** `internal/serve/mcp_lifecycle.go`, `handleInitialize` (lines 125-136).

**Before:** `ServerInfo: MCPServerInfo{Name: "hero", Version: s.version}` with no schema.

**After:** add the compiled `schemaVersion` and the graph's schema (read at init) to `ServerInfo` (or a `_meta` block on the initialize result — whichever the protocol types allow without breaking clients). Optionally surface a compact `hero_env` line on tool responses so an agent sees skew mid-session.
**Why:** Gives the harness/agent a *fact* to read ("binary schema 2, graph schema 4") instead of a vacuum it fills by confabulation. Requires threading the graph schema into `MCPServer` at construction (`mcp.go:54-73`).

### 3. MCP connection dedup / supersede per client+workspace (Defect 1, engine)
**File:** `internal/cli/mcp.go` (startup) + `internal/serve/` (a small workspace-scoped singleton/pidfile with liveness check).

**Approach to evaluate (do not regress orphan cleanup):** on `hero mcp` startup, take a workspace-scoped lock/pidfile keyed by client+workspace. If a *live* daemon already holds it, either (a) refuse the redundant connection cleanly, or (b) supersede — signal the incumbent to exit and take over. Reuse `internal/serve/lifecycle.go` liveness helpers (`IsProcessAlive`, etc.). Keep the existing orphan watchdog intact — this adds a *live-duplicate* guard the watchdog structurally can't provide.
**Why:** Stops a long-running client from accumulating N daemons. Extend `mcp_watchdog_test.go` and add a supersede test; verify orphan reaping still passes.

### 4. Steer agents to the MCP surface + detect terminal-mode fallback (harness-facing — ALL SIX TARGETS)
**Author-once** in `.hero/` (knowledge/convention/skill), propagated by `hero install` to `opencode | cursor | claude | copilot | codex | generic`. Content: "Prefer Hero's MCP tools over shelling out to a bare `hero`. If you see a schema/version mismatch, run `hero doctor` and report its output — do NOT invent a migration narrative or run `hero upgrade` to 'fix schema'." Make the AGENTS.md guidance self-contained and imperative (no reliance on a Claude-only hook), per the tripwire.
**Why:** Even with perfect engine errors, an agent that shells out to a stale binary and hallucinates is the felt failure. This closes the loop for every harness, not just Claude/Codex. Assess whether terminal-mode fallback should be actively discouraged/warned when an MCP connection to the same workspace already exists.

### Recommended Sequencing / Splitting
- **Spec A (engine, high priority, small):** Fixes 1 + 2 + secondary string-compare. Self-location + `hero doctor` + MCP stamping. Highest leverage: kills the confabulation at the source.
- **Spec B (engine, medium):** Fix 3 — MCP daemon dedup/supersede. Independent; standalone value (resource leak).
- **Spec C (harness-facing, medium, all six targets):** Fix 4 — routing guidance + terminal-fallback detection. Gated by the all-targets tripwire; keep separate so its target-coverage review doesn't block the engine fixes.

---

## Test Plan

### Existing test review
- `internal/serve/mcp_watchdog_test.go`, `internal/serve/mcp_watchdog_linux_test.go` — cover orphan reaping via seam vars (`watchdogExit`, `watchdogGetppid`). Any Defect-1 change must keep these green.
- `internal/graph/` migration tests (grep for schema/migration tests around `graph.go`) — cover the up-migration path; extend for the mismatch branches.
- `internal/serve/mcp_test.go` — MCP protocol tests; extend for the `initialize` schema stamping.
- `internal/cli/upgrade.go` tests — confirm upgrade behavior unchanged (we are not touching upgrade, only the error that mis-points at it).

### Test changes needed
1. **graph mismatch (Spec A):** table test over (binarySchema, graphSchema) → assert (a) binary-older = hard error containing `os.Executable()` path, both schemas, `hero doctor`, and NOT `hero upgrade` as the fix; (b) binary-newer = warn+continue; (c) **double-digit case** binarySchema="9", graphSchema="10" takes the warn-and-continue branch (guards the string-compare fix).
2. **`hero doctor` (Spec A):** test that output includes exe path, binary version, binary schema, graph schema, and a correct verdict for each of agree / older / newer. Simulate PATH-vs-`os.Executable()` divergence and assert the "wrong binary on PATH" flag fires.
3. **MCP initialize stamping (Spec A):** drive `handleInitialize` with a `bytes.Buffer`, assert the response carries binary version + binary schema + graph schema.
4. **MCP dedup/supersede (Spec B):** simulate two `hero mcp` startups for the same client+workspace; assert the second refuses or supersedes, and that a *single* daemon survives. Add an explicit test that orphan reaping is unaffected (existing watchdog tests must still pass).
5. **Harness propagation (Spec C):** per-target install test asserting the new routing guidance lands in each of the six targets' instruction surfaces (extend the existing install/upgrade target-coverage tests).

### Regression scope
- Do not alter the up-migration success path or `hero upgrade` behavior (only the *error text* pointing at it).
- Defect-1 dedup must not race with the orphan watchdog or kill the wrong daemon on legitimate reconnect — cover reconnect explicitly.
- MCP `initialize` schema additions must be backward-compatible (extra fields, not changed/removed ones) so existing clients don't break.

---

## Open Repro Step (the one piece only reproducible inside Codex)
Capture, from **inside a Codex terminal session in hero-code**:
1. the **literal** error text including the real `db=X binary=Y` values,
2. the output of `command -v hero; hero --version`,
3. (ideally) `ls -l "$(command -v hero)"` to identify the exact stale binary Codex binds.

This confirms Defect 2's specific trigger and closes the "schema 2" attribution. Until captured, that attribution is a strongly-supported hypothesis, not a confirmed fact. **Do not fabricate the values.** `Needs more research? → Yes` applies to this attribution only; all three code-level defects are confirmed.

---

## Notes
- `hero-code`'s five live daemons and the "schema 2" report are **different problems** that happen to co-occur in Codex. Don't conflate them: the daemons are `~/go/bin/hero`/schema-4 and read the graph fine (Defect 1, a leak); the "schema 2" report comes from a stale binary bound only in a Codex subprocess (Defect 2, an env trigger surfaced as a Defect 3 dead-end).
- Mission fit: the felt failure is "hero is less reliable in Codex than in Claude." Fixes 1–4 collectively make the next session start with a *fact* (which binary, which schema, run `hero doctor`) instead of a hallucination — raising the floor for anyone, not just the dev who already knows to check PATH.
- Anchor check (this session) surfaced tripwire `harness-changes-cover-all-targets` [high]: only Fix 4 is harness-facing; it must cover all six targets or be explicitly scoped. Engine fixes 1–3 are harness-agnostic.

---

## Completion Ledger

Delivered on branch `fix/agent-hero-version-schema-confusion` (commit 7dba572). Full suite: 85 packages OK; the only failure is the pre-existing, unrelated `TestMarkdownInvocationsResolveAgainstRootCmd` (release-notes doc drift — tracked separately). Cold audit: **SHIP / clean / high confidence** (`delivery-audit.md`).

### Spec A — engine: self-location + `hero doctor` + MCP schema stamping

| Item | Status | Evidence |
|------|--------|----------|
| Mismatch branches print `os.Executable()` + binary schema + graph schema | DONE | `internal/graph/graph.go` `checkSchemaMismatch`; `TestCheckSchemaMismatch` asserts exe+both schemas each branch. |
| Replace false `hero upgrade` remedy → `hero doctor`; name wrong-binary-on-PATH cause | DONE | Both messages point at `hero doctor`, carry "`hero upgrade` will NOT help"; `hero upgrade` (`upgrade.go`) unchanged. |
| Fix lexical schema compare (double-digit inversion) | DONE | `schemaLess()` (numeric); `TestSchemaLess` covers `"9"<"10"`. |
| Up-migration success path untouched | DONE | Only post-loop mismatch block changed; existing graph migration tests green. |
| New `hero doctor` (exe/PATH divergence flag, versions, schemas, verdict, graceful outside workspace) | DONE | `internal/cli/doctor.go` + `root.go`; `TestBuildDoctorReport` (7 subtests); exercised live — real PATH-divergence warning fired. |
| Stamp binary+graph schema on MCP `initialize` (additive) | DONE | `mcp_protocol.go`/`mcp.go`/`mcp_lifecycle.go`; `TestMCP_Initialize_StampsSchema`; live `serverInfo` emits `schema`/`graphSchema`. |

### Spec B — engine: MCP daemon dedup/supersede

| Item | Status | Evidence |
|------|--------|----------|
| Per-(workspace heroDir + parent pid) singleton; supersede live incumbent on reconnect | DONE | `internal/serve/mcp_singleton.go`; wired in `Run()` (`mcp_lifecycle.go`), gated `s.input == os.Stdin`. `TestMCPSingleton_SupersedesLiveIncumbent`. |
| Stale pidfile (dead holder) treated as free; reuse `IsProcessAlive` | DONE | `TestMCPSingleton_StalePidfileTreatedAsFree`; distinct clients isolated (`TestMCPSingleton_DistinctClientsDoNotCollide`). |
| Ownership-checked release on clean shutdown | DONE | Deferred `release()`; exercised live (pidfile removed on EOF). |
| Orphan watchdog not regressed | DONE | Watchdog code untouched; `TestMCPSingleton_OrphanWatchdogStillFires` + existing `mcp_watchdog_test.go` all pass (incl. `-race`). |
| Exercised: two daemons → one survivor | DONE | Two real `hero mcp` under one parent: incumbent superseded, exactly one survivor. |

### Spec C — harness: route agents to MCP surface + `hero doctor` (all six targets)

| Item | Status | Evidence |
|------|--------|----------|
| Guidance in author-once source | DONE | `internal/install/agents_md.go` `generateEngineeringAgentsMdBody`; mirror `domains/engineering/AGENTS.md` (regen test green). |
| Propagates to all six targets (opencode/cursor/claude/copilot/codex/generic) | DONE | `TestHarnessNative_DoctorRoutingGuidanceAllTargets` (table over all six); live `hero install` proof for a CLAUDE.md and an AGENTS.md target. |
| Test fails if any target drops guidance (tripwire teeth) | DONE | Per-target `t.Fatalf` on missing substring. |

### Disclosed scope

- **Domain packs:** guidance added to the **engineering** pack only (the active domain). The `pm`/`sales`/`chat` packs have independent bodies and were not touched — orthogonal to the six-target tripwire, which is fully satisfied. Optional small follow-up if wanted.
- **Open repro item (non-code):** literal Codex-side error text (`db=X binary=Y`) + `command -v hero; hero --version` from inside a Codex terminal session in hero-code — confirms Defect-2's specific stale-binary attribution. `Needs more research → Yes` for that attribution only.

## Kickoff

Agents invent a "your graph is schema 2, run `hero upgrade`" story when a stale `hero` binary (bound via Codex's GUI PATH) reads a schema-4 graph. Make hero self-locating instead.

**Status:** completed (verified, archived) — all three concerns delivered on commit 7dba572; cold audit SHIP/clean. One non-code repro step remains open (see below).

**Pick up at:** the only remaining thread is the **open repro item** — from inside a Codex terminal session in hero-code, capture the literal `db=X binary=Y` error text + `command -v hero; hero --version` to confirm exactly which stale binary Codex bound (Defect-2 attribution). Optional follow-up: extend the `hero doctor` routing guidance to the `pm`/`sales`/`chat` domain packs (engineering pack is done). Now that `hero doctor` exists, the fastest way to close the repro is to run it inside Codex and read its PATH-divergence verdict.

→ `.hero/planning/bugs/agent-hero-version-schema-confusion/spec.md`

**Files:** `internal/graph/graph.go:269-284`, `internal/cli/upgrade.go:27-60`, `internal/serve/mcp_lifecycle.go:125-136`, `internal/serve/mcp_watchdog.go:34-54`, `internal/cli/mcp.go:35-63`
**Skip:** `hero upgrade` as the remediation (upgrades overlay files, not the binary); blaming the five live MCP daemons for the schema report (they're `~/go/bin/hero`/schema-4 — that's the separate Defect 1 leak).
