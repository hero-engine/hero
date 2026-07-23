---
title: "Peer Call Multi-CLI — Pluggable Subagent Backends"
slug: peer-call-multi-cli
type: feature
status: planning
created: 2026-06-24
tags: [peering, subagent, cli, config]
superseded_by: peering-over-project-mail
---
# Peer Call Multi-CLI — Pluggable Subagent Backends

## Goal

Let `hero peer call` drive subagent backends other than `claude` (e.g. `codex`,
`gemini`, `cursor-agent`, `opencode`) without each user hand-tuning flags, env,
and error parsing. Ship a small set of built-in CLI profiles plus an escape hatch
for fully custom commands, and make "which CLI, is it installed, is it logged in"
visible *before* a call fails.

## Background

`hero peer call` shells out to an LLM CLI ([internal/peering/peercall.go](internal/peering/peercall.go)):
pipes the request envelope on stdin, parses a `<peer-call-result>` YAML fence from
stdout. Today the wiring is claude-shaped:

- `DefaultSubagentCommand = "claude"`, default args `["-p"]` (`resolveSubagentConfig`).
- The command/args/env-passthrough *are* configurable via `peering.subagent` in
  `hero.json` — so a different CLI is technically possible, but the user must know
  that CLI's headless flag, which auth env vars to pass through, and accept that
  hero's login-failure hint is claude-specific.
- Login detection (just added) recognizes generic "not logged in" signatures and
  emits claude-specific remediation only when `command == "claude"`. That's the
  seam this spec builds on.

The friction is real: nothing today tells a user "codex isn't installed" or
"gemini isn't logged in" until a call dies. And every new CLI means re-deriving
args + auth + remediation by hand.

## Design

### 1. CLI profiles (built-in registry)

Introduce a `SubagentProfile` describing how to drive one CLI:

```go
type SubagentProfile struct {
    Command        string            // binary name, e.g. "codex"
    Args           []string          // headless/print-mode invocation
    InputMode      InputMode         // Stdin | PromptArg | TempFile (how the envelope reaches the CLI)
    EnvPassthrough []string          // auth + locale env vars this CLI reads
    LoginSignature *regexp.Regexp    // overrides the generic notLoggedInRE when set
    LoginHint      string            // CLI-specific remediation, e.g. "run `codex login`"
}
```

Ship a registry keyed by command name with profiles for the CLIs we verify:
`claude` (the current default), and initial targets `codex`, `gemini`,
`cursor-agent`, `opencode`. `resolveSubagentConfig` resolves a profile by the
configured `command`, then applies any explicit `peering.subagent` overrides on
top (user config always wins).

### 2. Config schema

Extend `peering.subagent` so the common case is a one-liner and the custom case
is fully expressible:

```jsonc
"peering": {
  "subagent": {
    "command": "codex"            // resolves the built-in codex profile; args/env/auth inferred
    // optional overrides:
    // "args": ["exec", "--quiet"],
    // "env_passthrough": ["OPENAI_API_KEY", "PATH", "HOME"],
    // "input_mode": "prompt-arg"
  }
}
```

An unknown `command` with no built-in profile falls back to today's behavior
(stdin + generic login detection) and `hero check` warns that it's unprofiled.

### 3. Input mode abstraction

Not every CLI reads a prompt from stdin. `InputMode` decouples envelope delivery
from the runner:
- **Stdin** (claude today) — envelope piped to stdin.
- **PromptArg** — envelope passed as the final positional/`-p` argument.
- **TempFile** — envelope written to a temp file, path substituted into args
  (for CLIs that take `--input <file>`); file is cleaned up after.

`runSubagent` selects delivery from the resolved profile instead of hardcoding
`cmd.Stdin`.

### 4. Per-CLI login detection + preflight

- Extend the just-added error path to consult the profile's `LoginSignature` /
  `LoginHint` before the generic fallback. (`subagentRunError` already takes the
  command name — thread the profile through.)
- Add a preflight to `hero peer list` / a new `hero peer doctor`: for the resolved
  subagent, report `installed?` (PATH lookup, already done in `runSubagent`) and
  `logged-in?` (run the profile's lightweight auth probe where one exists, else
  skip with a note). Surfaces "gemini: installed, not logged in" up front instead
  of mid-call.

### 5. Result-fence robustness

The `<peer-call-result>` contract is prompt-driven and CLI-agnostic, but weaker
backends may not emit it cleanly. Keep parsing strict, but on a missing/invalid
fence return an error that quotes what the CLI *did* emit (now possible since we
surface stdout) so misbehaving backends are debuggable. Optional: one reprompt-on-
malformed retry, behind a profile flag, deferred unless testing shows it's needed.

## Changes

- `internal/config/` — extend `SubagentConfig` (input_mode); profile resolution.
- `internal/peering/peercall.go` — `SubagentProfile` + built-in registry; profile-
  aware `resolveSubagentConfig`, `runSubagent` (input modes), and `subagentRunError`
  (per-CLI login hint).
- `internal/peering/` — `hero peer doctor` (or extend `peer list`) preflight.
- `internal/cli/` — wire the doctor/list output.
- Tests: profile resolution + override precedence; each `InputMode`; per-CLI login
  detection; unprofiled-command fallback; doctor output for installed/missing/
  logged-out.
- Docs: `.hero/knowledge/` note on adding a new CLI profile; CLAUDE.md peering rows.

## Acceptance Criteria

- WHEN `peering.subagent.command` names a built-in profile (codex/gemini/cursor-agent/
  opencode) THE SYSTEM SHALL invoke it with correct args/input/env without further config.
- WHEN a user sets explicit `args`/`env_passthrough`/`input_mode` THE SYSTEM SHALL
  prefer those over the profile defaults.
- WHEN the configured CLI is not installed THE SYSTEM SHALL fail preflight with a
  clear "install/configure" message naming the command (already true via LookPath).
- WHEN the configured CLI is installed but not logged in THE SYSTEM SHALL emit that
  CLI's specific login remediation, not a claude-specific or opaque message.
- WHEN `hero peer doctor` runs THE SYSTEM SHALL report install + login status for the
  resolved subagent before any peer call is attempted.
- WHEN a CLI uses PromptArg or TempFile input THE SYSTEM SHALL deliver the full
  envelope intact and parse the result fence identically to the stdin path.
- WHEN an unknown command is configured THE SYSTEM SHALL fall back to stdin + generic
  login detection and warn that the backend is unprofiled.

## Boundaries

- Does NOT change the wire contract (`<peer-call-result>` fence, envelope shape) —
  backends comply with the existing prompt protocol.
- Does NOT add a hero-managed API key or auth broker. Each CLI owns its own auth;
  hero only detects and reports state. ("Hero never holds an API key" — Phase 2.)
- Does NOT solve the Claude-Desktop in-memory-credential case — that's an
  environment limitation of a spawned `claude`, orthogonal to backend choice.
- Initial profile set is the verified CLIs only; others ride the unprofiled
  fallback until a profile is added.

## Open Questions

- Per-peer subagent override (different CLI per peer) — useful, or is one global
  backend enough? Lean: global first, per-peer later if asked.
- Is a malformed-fence reprompt worth the budget, or should weak backends just fail
  loudly? Decide from real codex/gemini output during implementation.
