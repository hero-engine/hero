---
title: Mockup renderer selection biases to HTML on Swift projects
slug: mockup-renderer-selection-swiftui-bias
type: bug
status: completed
priority: high
severity: major
domain: engineering
tags: [mockups, /mock, ui-designer, swiftui, agent-discipline]
created: 2026-05-31
completed_at: 2026-06-01T03:09:16Z
---

# Mockup renderer selection biases to HTML on Swift projects

## Goal

When `/mock` is invoked in a project containing Swift sources, the agent
selects the **SwiftUI renderer** and compiles real platform UI — never an
HTML approximation — unless the user explicitly passes `--renderer=html`.
The renderer choice and the reason for it are visible to the user **before**
generation starts, so a wrong pick is caught in the same turn it happens.
Renderer selection is grounded in a deterministic CLI signal, not LLM
judgment over loose prompt instructions.

## Kickoff

Delivered. `hero spec mock detect` ships as the deterministic
renderer-selection signal; agent prompts now call it and use the
`renderer` field verbatim. Cold audit: SHIP / clean / high confidence.

**Status:** completed. Selection lives in Go (`internal/cli/mock_detect.go`,
13 tests). `mock.md` and `ui-designer.md` rewritten to call the CLI and
halt on conflict. `Mockups.Renderer` config field added with 3 round-trip
tests. Risk #1 (Cobra subcommand vs bare flags) verified — `hero spec
mock --list` still dispatches correctly.

**Pick up at:** nothing on this spec. The parent feature
`native-mockup-rendering` is now ready to flip to completed too —
its 5 remaining ACs all verify on disk against the post-fix code.

**Files shipped:** `internal/cli/mock_detect.go` (new),
`internal/cli/mock_detect_test.go` (new), `internal/config/config.go`
(+ MockupsConfig), `domains/engineering/commands/mock.md` (rewrite +
announce step), `domains/engineering/agents/ui-designer.md` (rewrite +
announce step), both renderer skills (When-to-use updated).

**Note for the next person:** the spec was written calling out
`hero mock detect`; the real CLI path is `hero spec mock detect` because
`mockCmd` is registered under `specCmd`. All prompts and help text use
the full path correctly.

## Problem

On a SwiftUI Mac app, the agent invoking `/mock` selected
`--renderer=html` (or auto-chose HTML despite Swift signals being present)
and produced an HTML approximation of the UI. The user had to intervene:
*"why would we have html renders — we render in swift and capture
images."* This has happened multiple times across sessions.

The four-step algorithm to prevent this already exists in two places:

- `domains/engineering/commands/mock.md:6-20` — the `/mock` Renderer
  Selection block (explicit flag → auto-detect → toolchain gate →
  hero.json override)
- `domains/engineering/agents/ui-designer.md:31-43` — a parallel block
  for when `ui-designer` is called directly

So the algorithm is documented. The bug is that the agent doesn't *run*
it reliably — or runs it and silently picks HTML anyway. This is a
recurring failure mode, not a one-off.

## Investigation

### What we know

- The /mock command body explicitly lists Swift triggers: `.swift`,
  `Package.swift`, `*.xcodeproj`, `*.xcworkspace` → SwiftUI renderer
  (`domains/engineering/commands/mock.md:12`).
- The `ui-designer` agent re-states the same auto-detect heuristics
  (`domains/engineering/agents/ui-designer.md:36-39`).
- The `swiftui-mockup-renderer` skill already documents the SwiftUI
  capture pipeline end-to-end and is the right artifact when Swift is
  detected (`domains/engineering/skills/swiftui-mockup-renderer/SKILL.md`).
- Hero already has Swift-aware code in `internal/scan/scan.go:223`
  (`.swift → "Swift"` in the language map) and stack-shape inference in
  `internal/snapshot/detect.go:34-86` (`ScanRepo` + `Detect`), but
  **neither is wired to the mockup renderer decision**. The /mock
  workflow does not call any Go helper to decide renderer — the LLM
  reads the prompt and decides.
- `internal/cli/mock.go` exposes `--list`, `--open`, `--serve` but no
  `detect` subcommand. There is no CLI surface that emits the "use
  renderer X" recommendation.
- The `hero.json` → `mockups.renderer` override is documented in
  `domains/engineering/commands/mock.md:15` but does not exist in
  `internal/config/`. It's a documented-but-unimplemented escape hatch.

### Root causes (three, compounding)

1. **No deterministic signal.** Renderer selection lives entirely in
   prompt text the LLM reads and interprets. The four-step algorithm is
   accurate but every invocation re-evaluates it from first principles
   against whatever the LLM happens to look at. Free-form glob-checking
   like "is there a `*.xcodeproj` anywhere in the repo?" is exactly the
   kind of probe LLMs do badly — they skip dirs, mis-glob, or assume
   "it's a web project" from one `package.json` in a subfolder.

2. **No announce-before-generate gate.** The current flow lets the
   agent go straight from "interpret the request" to "generate the
   mockup." There is no step that forces "I am using renderer X
   because Y" in a single visible line *before* file generation. The
   wrong choice is therefore invisible until the user opens the output
   and sees HTML. By then the agent has already committed several
   minutes of generation work and the user has to correct after the
   fact.

3. **No contradiction warning.** Step 1 ("explicit `--renderer=html`
   wins") fires unconditionally, even when the user passes
   `--renderer=html` on a project with `Package.swift` at the root —
   which is almost certainly a mistake (or a stale habit). The
   algorithm silently honors the conflicting flag instead of pausing
   to confirm. This is the exact failure mode the user reported.

The toolchain gate is fine: it's late in the algorithm by design, and
its job is to handle environments where `swiftc` is missing. The bug is
not "fallback fires too eagerly." The bug is "selection picks HTML
*before* the toolchain gate ever runs."

### Why the user sees this repeatedly

The agent's failure is not random — it's biased. HTML is the lower-risk
choice from an LLM's perspective: it always succeeds, requires no
external toolchain, and produces an `index.html` that opens. SwiftUI
requires `swiftc`, requires compile-success, and can fail at capture
time. Given any ambiguity, an LLM will systematically pick the path
with fewer failure modes. The prompt tries to counter this with explicit
trigger rules, but the bias re-emerges because the *cost of being
wrong* is asymmetric: silently picking HTML on a Swift project produces
a finished artifact the user has to reject; silently picking SwiftUI on
an HTML project produces a compile error the agent must recover from.

The fix has to make the wrong choice visible *before* generation
commits, and ideally remove the choice from the LLM entirely.

## Approach

Three layers, in priority order. **Layer 1 is the load-bearing fix.**
Layers 2 and 3 harden it.

### Layer 1 — `hero mock detect`: CLI emits the recommendation

Add a `hero mock detect` subcommand in `internal/cli/mock.go` that
performs the renderer-selection algorithm in Go and emits a single line
of JSON. The agent calls it as the first step of every `/mock`
invocation and uses the output verbatim — no judgment, no re-derivation.

Output shape:

```json
{
  "renderer": "swiftui",
  "reason": "detected .xcodeproj at repo root",
  "signals": ["MyApp.xcodeproj", "Package.swift", "12 .swift files"],
  "toolchain_ok": true,
  "toolchain_path": "/usr/bin/swiftc",
  "config_override": null,
  "explicit_flag": null,
  "conflict": null
}
```

Conflict shape (user passed `--renderer=html` on a Swift project):

```json
{
  "renderer": "html",
  "reason": "explicit --renderer=html",
  "signals": ["Package.swift", "8 .swift files"],
  "toolchain_ok": true,
  "config_override": null,
  "explicit_flag": "html",
  "conflict": "explicit flag --renderer=html overrides detected SwiftUI stack — confirm before generating"
}
```

The command reuses `internal/snapshot/detect.ScanRepo` for the walk
(already walks monorepo containers at depth 1) and adds Swift signal
detection. The Go side is the single source of truth for renderer
choice; the LLM stops deciding.

This addresses design question #6 directly: the decision belongs in
code, not in prompts.

### Layer 2 — Mandatory announce step

Update `/mock` command and `ui-designer` agent to require, as the
first user-visible output:

```
Renderer: SwiftUI
Reason: detected .xcodeproj at repo root
swiftc: /usr/bin/swiftc (available)
```

For conflicts, the announce step is a hard halt:

```
Renderer choice conflict.
You passed --renderer=html, but I detect Package.swift + 8 .swift files.
SwiftUI is almost certainly the right renderer for this project.
Confirm one:
  [keep HTML]   I really want an HTML approximation
  [use SwiftUI] override my flag and render natively
```

The agent does not proceed to generation until either (a) no conflict,
or (b) the user confirms. This addresses design questions #2 and #4.

### Layer 3 — Prompt cleanup so Layer 1 is the only path

Rip the inline renderer-selection algorithm out of
`domains/engineering/commands/mock.md` and
`domains/engineering/agents/ui-designer.md`. Replace both with a single
instruction: "Run `hero mock detect [--renderer=<flag-if-passed>]`;
use the `renderer` field from its JSON output; halt on `conflict`."

Two copies of the algorithm drifting against the LLM's interpretation
is part of what got us here. Reduce to one canonical implementation
(Go) plus one prompt rule (call it).

### What about `hero.json` mockups.renderer?

The documented config is currently fiction — no field exists in
`internal/config/`. Layer 1 should add it (struct field + load + a
`Mockups.Renderer` accessor) so the documented behavior matches
reality. `hero mock detect` reads it and includes it in the JSON
output. If unset, ignore. If set, it overrides auto-detect (but not an
explicit `--renderer` flag).

### What we are explicitly NOT doing

- We are not changing how SwiftUI mockups *render* — sizing,
  iOS-vs-macOS semantic colors, silent HTML fallback on compile error.
  Those are output-quality bugs carved out in Follow-ups.
- We are not adding a "hero check" rule for this. The detect command is
  enough; a check rule would just re-derive the same answer.

## Acceptance Criteria

- WHEN `/mock` is invoked in a project containing `Package.swift`, `*.xcodeproj`, `*.xcworkspace`, or `.swift` files THE SYSTEM SHALL select the SwiftUI renderer.
- WHEN `hero mock detect` runs THE SYSTEM SHALL emit a single line of JSON to stdout with fields `renderer`, `reason`, `signals`, `toolchain_ok`, `toolchain_path`, `config_override`, `explicit_flag`, and `conflict`.
- WHEN the agent invokes `/mock` THE SYSTEM SHALL print a one-line `Renderer: X — reason: Y — swiftc: <path|unavailable>` announcement to the user before any mockup file is generated.
- IF the user passes `--renderer=html` on a project with Swift signals THEN THE SYSTEM SHALL populate the `conflict` field in the detect JSON output and the agent SHALL halt and ask for confirmation before generating.
- IF `swiftc` is unavailable AND the SwiftUI renderer was auto-selected THEN THE SYSTEM SHALL fall back to HTML and announce the fallback explicitly (`Renderer: HTML (SwiftUI unavailable — swiftc not found)`).
- WHEN `hero.json` contains `mockups.renderer = "html"` or `"swiftui"` THE SYSTEM SHALL honor it as an override of auto-detection (but not of an explicit `--renderer` flag).
- THE SYSTEM SHALL ground renderer selection in the `hero mock detect` output rather than LLM interpretation of prompt rules.
- `domains/engineering/commands/mock.md` and `domains/engineering/agents/ui-designer.md` SHALL each instruct the agent to call `hero mock detect` and use its output verbatim, with no inline reproduction of the four-step algorithm.

## Changes

**Delivered files** (this section was updated during delivery to reflect actual files touched):

- `internal/cli/mock_detect.go` — new file. Contains `mockDetectCmd`, `runMockDetect`, `computeMockDetect`, `scanSwiftSignals`, plus the `lookSwiftc` test seam.
- `internal/cli/mock.go` — added cross-reference to `detect` in `mockCmd.Long`.
- `internal/cli/mock_detect_test.go` — new file. 13 tests covering pure-Go, Package.swift, *.xcodeproj, weak `.swift` signal, hero.json override (both directions), explicit-flag conflicts on Swift and missing-swiftc, monorepo `apps/ios/Package.swift`, single-line JSON shape, and real-env sanity.
- `internal/cli/helpers_test.go` — added `mockDetectRenderer` reset.
- `internal/config/config.go` — added `MockupsConfig` struct and `Mockups *MockupsConfig` field on `Config`.
- `internal/config/config_test.go` — three round-trip tests for `mockups.renderer = "html" | "swiftui" | unset`.
- `domains/engineering/commands/mock.md` — replaced the 4-step renderer-selection algorithm with "call `hero spec mock detect`, halt on conflict" plus a mandatory announce-step block.
- `domains/engineering/agents/ui-designer.md` — replaced the parallel 4-step block with the same CLI-driven instruction plus announce-step.
- `domains/engineering/skills/swiftui-mockup-renderer/SKILL.md` — added a one-line "When to use" note pointing at `hero spec mock detect`.
- `domains/engineering/skills/html-mockup-generation/SKILL.md` — added a one-line "When to use" note pointing at `hero spec mock detect`.

**Notes on planned Changes that the spec authored before delivery:**

- Change #7 (hero.json schema docs): no canonical schema doc exists in the repo. Per the spec's own fallback guidance, behavior is documented by the config tests, the spec body, the `mockDetectCmd.Long` help text, and the prompt updates in mock.md / ui-designer.md.
- The original commands invoked `hero mock detect`; on this codebase the actual command path is `hero spec mock detect` (because `mockCmd` is registered under `specCmd`). All prompt rewrites and tests use the full path.

---

**Original Changes plan** (kept for reference; delivered state above):

1. Add `hero mock detect` subcommand in `internal/cli/mock.go`
   - New Cobra subcommand under `mockCmd` (sibling of `--list`, `--open`, `--serve`)
   - Accepts `--renderer=<html|swiftui>` flag (passes through user's explicit choice)
   - Reuses `internal/snapshot/detect.ScanRepo` for the repo walk so monorepo subdirs are covered consistently
   - Adds a new helper (`detectSwiftStack`) that scans for `.swift` extensions in scan output, plus root-level checks for `Package.swift`, `*.xcodeproj`, `*.xcworkspace`
   - Resolves `swiftc` via `exec.LookPath("swiftc")`, populates `toolchain_ok` and `toolchain_path`
   - Reads `hero.json` `mockups.renderer` via the new config field (item 2)
   - Applies precedence: explicit flag → config override → auto-detect → toolchain gate
   - Emits flag conflict (`--renderer=html` + Swift signals, or `--renderer=swiftui` + no Swift signals + no swiftc) in the `conflict` field
   - Emits one line of JSON to stdout; non-zero exit only on internal error, never on a "no Swift detected" result

2. Add `Mockups.Renderer` field to `internal/config/` config struct
   - New `Mockups struct { Renderer string }` on the loaded config
   - Loader honors `mockups.renderer` from `hero.json` (`"html"` | `"swiftui"` | empty)
   - Empty/unset is treated as "no override"
   - Unit test: load a `hero.json` with each value and assert it round-trips

3. Test coverage for `hero mock detect` in `internal/cli/mock_test.go`
   - Table-driven test with synthetic repo layouts: pure-Go, Swift-only, mixed, monorepo with Swift in `apps/ios`, `hero.json` override variants
   - Assert JSON output shape and renderer choice for each case
   - Assert conflict field populated when explicit flag opposes detection
   - Mock `exec.LookPath("swiftc")` outcome via a tiny seam (env var or function variable) so tests run on Linux CI

4. Rewrite `domains/engineering/commands/mock.md` Renderer Selection section
   - Replace the four-step algorithm (lines 6-20) with: "Before routing, run `hero mock detect [--renderer=<flag>]`. Use the `renderer` field of the JSON output verbatim. If `conflict` is non-null, halt and surface the message to the user — do not proceed until they confirm."
   - Add an "Announce step" block: agent must emit a one-line `Renderer: X — reason: Y — swiftc: Z` message before delegating to `ui-designer`
   - Keep the renderer→skill mapping and output paths block (lines 17-19) — those are still accurate post-detect

5. Rewrite `domains/engineering/agents/ui-designer.md` Renderer selection section
   - Replace lines 31-43 with: "Renderer is chosen by `hero mock detect` (run by `/mock` or by you directly if called without a hint). Use its `renderer` field verbatim. Do not re-derive."
   - Add a parallel announce-step requirement for when `ui-designer` is called directly outside `/mock`
   - Keep the SwiftUI capture pipeline block (lines 92-119) unchanged — that's separate concern

6. Add a one-line note to `domains/engineering/skills/swiftui-mockup-renderer/SKILL.md`
   - Under "When to use" (line 27), add: "Selection is now driven by `hero mock detect` (see `cmd/hero mock detect --help`). The agent does not decide — the CLI does."
   - Mirror the same note in `domains/engineering/skills/html-mockup-generation/SKILL.md`

7. Update `hero.json` schema docs (wherever Mockups config is referenced)
   - Add `mockups.renderer` to the config reference with valid values and precedence rules
   - If there is no schema doc, the new field's behavior is documented by the test added in item 3 plus the spec body — flag this in delivery, don't invent a doc location

8. Update `/mock` slash command help text and `cmd/hero mock detect --help` to mention each other
   - The CLI subcommand should print one example and reference the slash command
   - The slash command body should mention that `hero mock detect` is the source of truth for renderer choice

## Boundaries

In scope:
- Renderer selection logic (which renderer is chosen, by what signal)
- Agent discipline around announcing the choice before generating
- Conflict warnings when flag opposes detection
- Implementing the documented-but-missing `hero.json` `mockups.renderer`
  field

Explicitly **out of scope** (carved out by the user):
- SwiftUI output quality (iPhone-sized output on a Mac app, iOS-only
  semantic colors that break macOS compiles, silent HTML fallback on
  SwiftUI compile errors, capture pipeline robustness)
- Any change to the `html-mockup-generation` or
  `swiftui-mockup-renderer` skills' generation rules
- Adding new renderers (web/React, iOS UIKit, Android, etc.)
- Replacing the `ui-designer` agent or restructuring `/mock` flow
- Making `hero check` lint mockup-renderer config

## Follow-ups (out of scope)

Track these as separate specs if/when prioritized. They were explicitly
carved out of this fix:

1. **SwiftUI mockups default to iPhone (390×844) on Mac app projects.**
   The capture entry point in `MockView.swift` hardcodes iOS phone
   dimensions (`internal/snapshot/.../SKILL.md` line ~66). For a Mac
   app, the right defaults are macOS window dimensions and AppKit
   chrome.

2. **iOS-only semantic colors break macOS compiles.** The SwiftUI
   skill's example palette uses tokens that exist on iOS but not on
   macOS, producing compile failures the agent has to recover from on
   Mac-target projects.

3. **Silent HTML fallback on SwiftUI compile error.** When `swiftc`
   fails twice (per `ui-designer.md:102`), the agent falls back to
   HTML with only a brief note. This is the *output-side* version of
   the bug this spec fixes on the *selection side*. The user wants a
   harder failure that surfaces "SwiftUI compile failed, here's the
   error, want me to retry or fall back?" rather than silent
   degradation.

4. **Detect the SwiftUI target platform (iOS vs macOS vs visionOS).**
   Once #1 and #2 are fixed, the renderer needs to know which platform
   it's targeting so it picks the right dimensions and palette.
   `Package.swift` and `*.xcodeproj` carry this info — extending
   `hero mock detect` to emit a `target_platform` field would feed it.

## Risks

- **Adding a Cobra subcommand under `mockCmd`.** The current `mockCmd`
  uses bare flags (`--list`, `--open`, `--serve`) rather than
  subcommands. Adding `hero mock detect` introduces a subcommand
  pattern. Make sure `runMock` (the bare-flag dispatcher) doesn't
  swallow the `detect` argument. Standard Cobra layering handles this,
  but worth a deliberate test.

- **Monorepo Swift detection.** A repo with one small Swift folder in
  `apps/ios/` but a primary Go or web stack would currently auto-select
  SwiftUI under the new rules. This is technically correct for a
  spec-less `/mock` invocation in that subdir, but if `/mock` is run
  from the repo root the user probably means the dominant stack.
  Tests must cover this. Possible mitigations: weight signals by
  proximity to CWD, or require a `Package.swift`/xcodeproj root for
  auto-select while .swift-only counts as weak signal. Decide during
  delivery — call out in PR.

- **`swiftc` lookup latency.** `exec.LookPath` is fast but runs on
  every detect call. Acceptable. Don't cache across runs; renderer
  selection has to reflect current shell environment.

- **Backward compat for the prompt.** Anyone who has skills/commands
  fork-edited locally and modified the renderer-selection block will
  end up with a stale algorithm. Add a one-line note in the
  changelog/release notes that the prompt rule was simplified and now
  defers to CLI.

- **LLM bypass.** Nothing prevents a sufficiently confused agent from
  ignoring `hero mock detect` and re-deriving the choice anyway. The
  announce-step mitigates this by making the wrong choice visible
  immediately. If the regression recurs, the next-tier fix is to gate
  the `ui-designer` agent's tool use behind a "did you call
  detect?" check — but that's heavier and not in this spec.

## Validation

How to verify the fix works:

1. **Repro the original failure**: in a Swift macOS app project (or a
   synthetic dir with `Package.swift` + a few `.swift` files), run
   `/mock` with no flags. Confirm:
   - Agent prints `Renderer: SwiftUI — reason: ... — swiftc: ...`
     **before** generating
   - Generated output is `MockView.swift` + `screenshot.png`, not an
     `index.html` approximation

2. **Conflict path**: same repo, run `/mock --renderer=html`. Confirm:
   - Agent halts at the announce step
   - User is asked to confirm with the conflict message
   - Generation does not start until confirmation

3. **HTML project unchanged**: in a Node/Go/Python project with no
   Swift signals, run `/mock`. Confirm:
   - Agent prints `Renderer: HTML — reason: no Swift signals detected`
   - HTML mockup generates as before

4. **swiftc-missing fallback**: on a Linux machine (or with `PATH`
   stripped of `swiftc`), run `/mock` on a Swift project. Confirm:
   - Detect output has `toolchain_ok: false`
   - Renderer falls back to `html` with explicit announce message

5. **Config override**: set `hero.json` `mockups.renderer = "html"` in a
   Swift project. Confirm:
   - Detect picks HTML
   - Announce step shows reason: "hero.json mockups.renderer = html"
   - Explicit `--renderer=swiftui` still wins over the config

6. **Regression**: run the test suite from item 3 (`internal/cli/mock_test.go`).
   All renderer-selection scenarios pass.

7. **Manual smoke**: pick a third-party Swift repo (something obviously
   not Hero), point `/mock` at it from a fresh session, and verify the
   announce-then-generate flow ends in screenshots, not HTML.

The success criterion the user cares about: never again has to type
*"why would we have html renders — we render in swift and capture
images."*
