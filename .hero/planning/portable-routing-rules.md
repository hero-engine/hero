---
title: "Portable Routing Rules — One `routing.md`, Every Harness Gets It Natively"
type: feature
status: planning
priority: high
horizon: now
tags: [install, routing, harness-integration, agents-md, claude-md, multi-harness]
relations:
  - target: install-target-emits-both-claude-and-agents-md
    kind: related
  - target: monorepo-satellite-installs
    kind: related
  - target: cross-repo-peering
    kind: related
---

# Portable Routing Rules — One `routing.md`, Every Harness Gets It Natively

## Context

Hero's natural-language → slash-command routing table is currently hard-coded inside `internal/install/agents_md.go` (`generateAgentsMdBody`, lines ~237–267). That single body string is rendered into both `AGENTS.md` and `CLAUDE.md` via the shared `installManagedMarkdown` writer. The table is roughly 20 rows plus a paragraph of disambiguation prose for cross-repo peering (`/handoff` vs `hero handoff <spec> <alias>`).

Today the content reaches the model in two harnesses (Claude Code via `CLAUDE.md`, AGENTS.md-aware harnesses via `AGENTS.md`) because both files are written from the same Go string. As Hero's harness footprint grows — Cursor, opencode, Codex, Copilot, generic — the routing content needs to reach the model in **whatever idiom each harness natively understands**. Right now Cursor (`.cursor/rules/*.mdc`), Copilot (`.github/copilot-instructions.md`), Codex (concatenated `AGENTS.md`), and the generic `.ai/` layout each get a different and inconsistent treatment of routing content (or none — Copilot's `installInstructionsMd` in `target_generic.go` writes a stub that omits the routing table entirely).

The drift is small today but will get worse, not better, as:

1. The routing table grows (sprint flows, new peer modes, future skills).
2. The disambiguation prose grows (peering already added one paragraph; more is coming).
3. New harnesses get added to `hero install`.

User-validated direction from the design conversation:

- Extract the **table + disambiguation prose** into a single source file maintained in Hero's source tree.
- For each install target, emit the **native include idiom** so the routing content is loaded into the model's context as if inline — no effectiveness regression vs the current always-on inline table.
- For harnesses without a native include mechanism, inline the rendered content into the main instruction file at install time.
- This is a **generator problem**: `hero install --target X` already knows the target; it should render the right shape per target from one source.

Related work:

- `install-target-emits-both-claude-and-agents-md` (bug, planning) — separate fix gating which root instruction file is emitted per target. The portable-routing work composes cleanly with it: routing content lives in `routing.md`; whichever root file the target emits gets the right include directive or inlined content.
- `monorepo-satellite-installs` (feature, delivering) — satellites must inherit routing rules from the root install; the source `routing.md` lives at the root `.hero/` and satellites reference it through their satellite harness layout.
- `cross-repo-peering` (feature, delivering) — source of the disambiguation prose. No conflict; this spec moves the prose into a more maintainable location.

## Goal

A developer running `hero install --target <any-supported-target>` ends up with Hero's natural-language routing rules reaching the model on every session, in the idiom that target natively prefers — without Hero maintaining N parallel copies of the routing table. One canonical source (`routing.md`) lives in the Hero source tree; the install pipeline renders it into each target's expected shape. When routing rules change in Hero, a single source edit propagates to every target on the next install. For harnesses with a native include mechanism, edits also propagate on the **next session** without a reinstall.

"Done" means:

- A canonical `routing.md` exists in Hero's embedded content tree, containing the routing table and the peering disambiguation prose.
- Each supported install target (`claude`, `opencode`, `cursor`, `codex`, `copilot`, `generic`) emits routing content in the idiom that target's loader natively supports.
- The current effectiveness floor is preserved: every target's emitted artifacts cause the routing rules to be loaded into the model's context at session start, without any user action beyond `hero install`.
- `internal/install/agents_md.go`'s `generateAgentsMdBody` no longer carries an inline routing table or peering disambiguation paragraph — it carries the imperative framing and an in-file pointer line.

## Kickoff

Portable routing rules: extract the natural-language → slash-command table out of `internal/install/agents_md.go` into one `routing.md`, then render per-target include directives so every harness loads it natively.

**Status:** planning — spec just landed, no code yet.

**Pick up at:** map the per-target rendering matrix. Start at `internal/install/agents_md.go:224` (`generateAgentsMdBody`) to see the content that's moving, then walk each `target_*.go` to confirm the native include idiom (Cursor `.mdc` frontmatter, Aider `read:`, Cline `.clinerules/`, opencode `instructions:`). Bucket A = native include; Bucket B = inline at install time.

→ `.hero/planning/portable-routing-rules.md`

**Files:** `internal/install/agents_md.go:224-313`, `internal/install/target_claude.go`, `internal/install/target_opencode.go`, `internal/install/target_cursor.go`, `internal/install/target_copilot.go`, `internal/install/target_codex.go`

## Problem

Three concrete problems with the status quo:

1. **One source body, two destinations, growing.** `generateAgentsMdBody` is the only function emitting routing rules today, and it writes to both `AGENTS.md` and `CLAUDE.md` (via `installAgentsMd` and `installClaudeMd`). The opencode target shares the AGENTS.md path. Cursor (`.cursor/rules/`) gets no routing content at all in its rule files. Codex relies on the same `AGENTS.md` path that gets concatenated. Copilot's `installInstructionsMd` writes a stub at `.github/copilot-instructions.md` that **omits the routing table entirely** (see `target_generic.go:50` — three lines about `/design`, `/deliver`, `/diagnose` and that's it). Generic likewise. These omissions are silent quality bugs: Copilot users get a session where Hero's routing intent never reaches the model.

2. **Drift risk is monotonic.** Every new routing row (new slash command, new peer mode) and every disambiguation paragraph today gets added in one place — `generateAgentsMdBody`. That's currently fine. The moment Hero adds even one more harness path that wants different rendering (Cursor `.mdc` frontmatter; Aider `read:`; Cline `.clinerules/` directory), the choice becomes: maintain N copies in N target files (drift guaranteed within one release cycle) or extract a single source. We're choosing now.

3. **No native include leverage.** Several harnesses Hero already cares about have first-class "always-load this file as context" mechanisms: Claude Code's `@.hero/routing.md` import syntax in CLAUDE.md; Cursor's `alwaysApply: true` frontmatter in `.cursor/rules/*.mdc`; Aider's `read: [.hero/routing.md]` in `.aider.conf.yml`; Cline's `.clinerules/` directory; opencode's `instructions:` array in `opencode.json`. Using them means edits to routing rules propagate next session without a reinstall — strictly better operationally than baking content into the managed region of an instruction file. We're not using any of them today.

The effectiveness concern the user raised explicitly: today's inline table works because it's always-on, no decision required from the model. Whatever each harness gets after this change must keep that property — the routing content must reach the model with the same "always-on" feel, or the change has regressed user value.

## Approach

### 1. Single source of truth

Hero's source tree gains a single `routing.md` file, embedded into the binary alongside agents/commands/skills. Location: top-level `routing.md` at the repo root (mirroring `opencode.json`'s placement) so it ships via `content.go`'s embed. Contents:

- The imperative framing (`When the user describes what they want in natural language, route to the appropriate Hero slash command. Run the command — don't just suggest it.`) — included so the file is **standalone-readable** for the bucket-B inline case and for Cursor rule files where the rule file is the model's sole entry point.
- The full intent→command table.
- The peering disambiguation paragraph.
- The "When routing, pass the user's original context as arguments to the command. If the intent is ambiguous, present the top 2-3 options and ask." prose.

It does **not** include surrounding sections (Session Title, Key Workflow, CLI Commands, Project Structure, Important Rules). Those stay in `generateAgentsMdBody` because they're not part of the routing concern and don't have the same per-harness rendering shape problem.

The `routing.md` is **rendered as pure markdown** with no Hero-specific placeholder substitution. It's the same bytes for every target — only the include idiom differs.

### 2. Where it lands on the installed workspace

The installed workspace gets `.hero/routing.md` at the project root. Path resolution must agree with `monorepo-satellite-installs`:

- Root install: `<project>/.hero/routing.md`.
- Satellite installs: satellites reference the root `.hero/routing.md` via the same path-walking the harness uses to find `CLAUDE.md`/`AGENTS.md` (i.e., we do not duplicate `routing.md` into satellite subfolders; the harness includes the root file directly). For bucket-B (inline) targets in satellites, the inline content is baked at install time into the satellite's main instruction file, just like the root case.
- Global mode (`--mode=global`): the file lands under the harness's global config dir (`~/.claude/.hero/routing.md` etc.). Each target's `resolve*Paths` already distinguishes project vs global; the new install step mirrors that.

### 3. Per-target rendering matrix

Each target falls into one of two buckets:

**Bucket A — native include.** Hero writes `.hero/routing.md` AND a tiny include directive in the main instruction file pointing at it.

| Target | Routing file dest | Include directive landing site | Idiom |
|---|---|---|---|
| `claude` | `<project>/.hero/routing.md` | `CLAUDE.md` (inside managed region) | `@.hero/routing.md` on its own line |
| `opencode` | `<project>/.hero/routing.md` | `opencode.json` (instructions array) + `AGENTS.md` pointer line | `"instructions": [".hero/routing.md"]` |
| `cursor` | `<project>/.cursor/rules/hero-routing.mdc` | self-contained (rule file IS the include) | `.mdc` frontmatter `alwaysApply: true` |
| `aider` (future, not yet in Target enum) | `<project>/.hero/routing.md` | `.aider.conf.yml` | `read: [.hero/routing.md]` |
| `cline` (future, not yet in Target enum) | `<project>/.clinerules/routing.md` | self-contained | drop-file convention |

**Bucket B — inline at install time.** Hero writes the rendered routing content directly into the main instruction file (inside the Hero managed region). No separate `.hero/routing.md` is needed for these targets to function, but Hero **also writes** `<project>/.hero/routing.md` for uniformity (it's the source-of-truth file users edit; `hero install` consumes it back on the next run when source has been customized).

| Target | Main instruction file | Notes |
|---|---|---|
| `codex` | `AGENTS.md` | Codex concatenates AGENTS.md — inline content lands in scope |
| `copilot` | `.github/copilot-instructions.md` | Today omits routing entirely; this spec fixes that |
| `generic` | `AGENTS.md` | Catch-all; mirror the AGENTS.md inline content |
| Windsurf (future) | `.windsurfrules` | Single-file harness; inline at install time |

The decision rule for which bucket a target falls in: **bucket A if the harness has a documented, stable always-on include mechanism that loads the file's bytes into the model's context at session start without further configuration.** If documentation is ambiguous or the mechanism is gated by a non-default setting (e.g., Cursor's `alwaysApply` interaction with chat-mode vs agent-mode — see Risks), default to bucket B for that target.

### 4. Per-target caveats to verify during delivery

These are research items the delivery agent must resolve before declaring the target bucket-A. Each caveat that can't be resolved cleanly → fall back to bucket B for that target.

- **Cursor `alwaysApply: true`**: confirm it applies to chat-mode AND agent-mode, or only agent-mode. If only agent-mode, evaluate whether bucket A is good enough (routing matters most when the model is acting on a user intent — agent mode is the right place) or whether we should fall back to inlining into a global rule file.
- **opencode `instructions:`**: confirm it's read once at session start and concatenated alongside `AGENTS.md`. Confirm ordering — does the routing content land before or after the user-edited AGENTS.md content? If after, the imperative framing in `routing.md` itself must stand on its own (it does, by design above).
- **Aider `read:`**: confirm reload behavior on session restart and whether the file is treated as system context vs editable.
- **Cline `.clinerules/`**: confirm all files in the directory are always loaded, and whether file ordering matters. Hero's file alphabetizes near the top by naming `routing.md` (vs `zzz-routing.md`).

If a caveat can't be confirmed from public docs during delivery, **note it as a known limitation** in the install output and fall the target back to bucket B until a maintainer with that harness installed verifies behavior.

### 5. Source-of-truth split (what's in `routing.md` vs the main instruction file)

The main instruction file (CLAUDE.md / AGENTS.md / copilot-instructions.md) keeps:

- The `### Natural Language Routing` heading.
- The imperative framing (`When the user describes... Run the command — don't just suggest it.`) — kept inline so a user reading the main instruction file sees the framing without having to follow the include.
- A pointer line ("See `.hero/routing.md` for the full intent→command table and peering disambiguation.") for bucket A.
- The full rendered content of `routing.md` inlined for bucket B.

The `routing.md` itself **also** carries the imperative framing — it's the first paragraph — so when the file is loaded standalone (cursor rule file; opencode instructions file read independently of AGENTS.md) the framing reaches the model.

This means the framing lives in two places (main instruction file managed region + routing.md). That's deliberate and acceptable because:

- It's a short, stable two-sentence string. Drift cost is low.
- The framing is load-bearing for both surfaces independently — removing from either weakens the user-visible result.
- The table and disambiguation prose, which are the long and growing content, live in `routing.md` only.

### 6. Update propagation story

**Bucket A targets**: when `routing.md` source content changes in Hero and the user runs `hero install`, only the `.hero/routing.md` file changes on disk. The main instruction file's managed region is unchanged (the include directive is stable). On the next harness session, the model picks up the new file. No reinstall needed for in-flight sessions to see source edits to a checked-in `.hero/routing.md`.

**Bucket B targets**: when `routing.md` source content changes in Hero and the user runs `hero install`, both `.hero/routing.md` and the main instruction file's managed region change (the latter regenerates with new inline content). Users on a bucket-B target who don't reinstall after a Hero upgrade will see stale routing content in their main instruction file.

This asymmetry is acceptable as long as users know about it. Mitigation:

- `hero check` gains a check comparing the rendered bytes of the bucket-B target's main instruction file routing block against what the current Hero binary would render. If different, flag as "routing-stale" with a one-line fix (`hero install --target <name>`).
- `hero install` prints a final-line summary noting which routing content was emitted natively vs inlined, so users learn the model on their first install.

### 7. Verification approach

The delivery agent must produce a smoke-test fixture: a small set of natural-language phrasings that should each route to a specific command. For example:

| Phrasing | Expected route |
|---|---|
| "fix the cart total rounding bug" | `/diagnose` |
| "design an export to csv feature" | `/design` |
| "ship the auth refactor" | `/deliver` |
| "ask hero-code about the install pipeline" | `hero peer call hero-code --mode=advisory` |
| "hand off the auth-refactor spec to hero-cloud" | `hero handoff auth-refactor hero-cloud` |
| "what peers do we have" | `hero peer list` |
| "force-refresh NEXT.md before context limit" | `/handoff` |

For each target, the verification is: render Hero's install output for that target into a tmp dir, capture the bytes that would reach the model at session start (`routing.md` + main instruction file routing region), and confirm by inspection that every fixture phrasing maps to its expected route via the rendered content. This is content-presence verification, not live model evaluation — we're checking that the routing table reaches each harness's context with the right rows. Live model evaluation is out of scope.

## Changes

1. **Add embedded `routing.md` source file**
   - New file `routing.md` at repo root (alongside `opencode.json`, `AGENTS.md`, `CLAUDE.md`).
   - Contains: imperative framing, full intent→command table (verbatim from the current `generateAgentsMdBody` rows), peering disambiguation paragraph, "When routing, pass the user's original context..." prose.
   - Verify `content.go`'s embed directive captures it. If not, extend the embed pattern.

2. **Extract routing content out of `generateAgentsMdBody`** in `internal/install/agents_md.go`
   - Remove the table-row `sb.WriteString` calls (lines ~240–262).
   - Remove the peering disambiguation paragraph (line ~267).
   - Remove the "When routing, pass..." paragraph (lines ~264–265).
   - Keep the `### Natural Language Routing` heading and the imperative framing.
   - Add a routing pointer-or-inline call that switches on target bucket.

3. **Introduce a routing renderer module** at `internal/install/routing.go`
   - `renderRoutingForTarget(target Target, mode Mode) (mainFileBlock string, sidecarFile *RoutingSidecar)` returns the bytes to splice into the main instruction file's managed region AND optionally a sidecar file spec (path + content) to write.
   - For bucket A: `mainFileBlock` is `### Natural Language Routing` heading + framing + pointer line; `sidecarFile` is `.hero/routing.md` (or `.cursor/rules/hero-routing.mdc` for Cursor) with the full content.
   - For bucket B: `mainFileBlock` contains the full rendered content; `sidecarFile` is `.hero/routing.md` (written anyway for uniformity).
   - Reads the canonical routing content from the embedded `routing.md` (via `opts.sourceFS()`).

4. **Wire the routing renderer into each target installer**
   - `target_claude.go`: write `.hero/routing.md` sidecar; the `@.hero/routing.md` import directive lands inside CLAUDE.md's managed region via the updated `generateAgentsMdBody`.
   - `target_opencode.go`: write `.hero/routing.md`; ensure `opencode.json` merge adds `instructions: [".hero/routing.md"]` (preserving any user-edited entries). The `AGENTS.md` body gets the pointer-only routing block.
   - `target_cursor.go`: write `.cursor/rules/hero-routing.mdc` with `alwaysApply: true` frontmatter and the routing content body. No main-instruction-file changes — Cursor doesn't use AGENTS.md/CLAUDE.md.
   - `target_codex.go`: bucket B. `AGENTS.md` body gets the inlined full routing content. Also write `.hero/routing.md` for uniformity.
   - `target_copilot.go`: bucket B. **This is a behavior change** — `installInstructionsMd` currently writes a stub; replace its body construction (or its call) with the routing-aware renderer so `.github/copilot-instructions.md` carries the full routing content.
   - `target_generic.go`: bucket B. Same fix to `installInstructionsMd` covers generic AGENTS.md output; routing content inlined.

5. **Update `installInstructionsMd`** in `target_generic.go`
   - It currently builds a fixed 3-line "Key workflow" block omitting routing. Replace with a call to the new routing renderer so the rendered routing block appears in the file.
   - Preserve the existing "Available commands" paragraph and MCP reference.

6. **`opencode.json` merge integration**
   - In `installConfig` (`target_opencode.go:83`), the merge step today copies/merges JSON. Extend to ensure the `instructions` key contains `".hero/routing.md"` (append if missing, preserve user entries). The merge primitive lives in `files.go`'s `mergeJSONFromData` — verify it handles array-append semantics or add a small helper.

7. **`hero check` gains a routing-staleness check**
   - For each installed bucket-B target detected on disk (via `install-state.json`), compare the rendered routing block in the target's main instruction file against what the current binary would render.
   - If different, emit a warning with the target name and the fix command (`hero install --target <name>`).
   - Implementation lives next to existing check rules in `internal/cli/check.go`.

8. **Update `harness_smoke_test.go` and `harness_test.go`**
   - Extend per-target smoke tests in `internal/install/` to assert: (a) the routing sidecar file exists with correct content where expected; (b) the main instruction file's managed region contains the right shape (pointer vs inlined); (c) bucket-A targets produce a stable include directive line.
   - Add a fixture-based test: render to tmp dir for each target, then string-search the rendered output for known intent phrases ("Bug, error, broken") and expected commands ("/diagnose"). Confirms routing rows reach each target's context.

9. **Update `hero install` user-facing output**
   - Add a one-line summary at install end naming the routing rendering mode used for the target (e.g. `routing: native-include (.hero/routing.md)` or `routing: inlined (.github/copilot-instructions.md)`). Helps users learn the propagation model.

10. **Migration safety: existing installs**
    - When a user upgrades and re-runs `hero install`, the existing main instruction file's managed region (containing the old inline table) is regenerated by `installManagedMarkdown`'s in-place replacement logic. No special migration step needed — the new renderer produces the new content; the managed region swaps.
    - For Cursor users who already have `.cursor/rules/` content, `hero install` should not clobber unrelated user rule files. Verify the existing flat-install logic in `target_cursor.go` handles the new `hero-routing.mdc` filename safely (it should — `installFlat` only writes files it generates).

11. **Documentation pass**
    - Add a section to `README.md` or `GETTING-STARTED.md` ("Where the routing rules live") describing the bucket-A vs bucket-B story so users know whether they need to reinstall after a Hero upgrade.
    - No changes to `web/docs` user-facing content beyond a brief note in the install docs.

## Acceptance Criteria

- WHEN `hero install --target claude` is run in a project mode workspace, THE SYSTEM SHALL write `.hero/routing.md` containing the canonical routing content AND insert `@.hero/routing.md` inside CLAUDE.md's managed region under the `### Natural Language Routing` heading.
- WHEN `hero install --target opencode` is run in a project mode workspace, THE SYSTEM SHALL write `.hero/routing.md` AND ensure `opencode.json`'s `instructions` array contains `".hero/routing.md"` (preserving any existing user entries).
- WHEN `hero install --target cursor` is run, THE SYSTEM SHALL write `.cursor/rules/hero-routing.mdc` containing the routing content with `alwaysApply: true` frontmatter, and SHALL NOT clobber any unrelated existing files under `.cursor/rules/`.
- WHEN `hero install --target codex` is run, THE SYSTEM SHALL inline the full rendered routing content (table + disambiguation prose) into AGENTS.md's managed region AND write `.hero/routing.md` to disk for uniformity.
- WHEN `hero install --target copilot` is run, THE SYSTEM SHALL inline the full rendered routing content into `.github/copilot-instructions.md` (replacing the current stub that omits routing) AND write `.hero/routing.md` to disk.
- WHEN `hero install --target generic` is run, THE SYSTEM SHALL inline the full rendered routing content into AGENTS.md AND write `.hero/routing.md` to disk.
- WHEN the embedded `routing.md` source content changes and `hero install` is re-run, THE SYSTEM SHALL refresh all target outputs for that target (sidecar file for bucket A, inlined block + sidecar for bucket B).
- IF a bucket-B target's main instruction file routing block differs from the current binary's rendered output, THEN `hero check` SHALL flag the target as `routing-stale` with the fix command in the message.
- THE SYSTEM SHALL preserve the imperative framing (`Run the command — don't just suggest it.`) in every target's main instruction file managed region, regardless of bucket.
- THE SYSTEM SHALL include the full intent→command table (every row currently emitted by `generateAgentsMdBody`) AND the cross-repo peering disambiguation paragraph in the canonical `routing.md`.
- THE SYSTEM SHALL emit, in `hero install`'s closing output, a single line identifying the routing rendering mode used for the installed target (native-include vs inlined).
- WHEN `hero install` runs against a satellite workspace under a monorepo root install, THE SYSTEM SHALL resolve `.hero/routing.md` to the root workspace's file (no duplicated routing.md in subfolders).

## Boundaries

Out of scope:

- A config DSL for users to declare their own custom routing rows per repo. Hero owns the routing table; per-repo extension is a separate decision.
- A runtime "fetch routing.md at session start over the network" mechanism. Purely a generator/install-time concern.
- Live model evaluation that confirms a routed phrasing actually fires the expected command in the harness. We verify content presence in context; the model's interpretation of that content is the model's responsibility.
- Designing for harnesses not currently in `hero install`'s target enum (Windsurf, Aider, Cline as full first-class targets). These are noted in the per-target matrix as future work; the spec lays the foundation but does not commit to them.
- The separate fix for `install-target-emits-both-claude-and-agents-md` (which root file each target emits). This spec assumes that bug fix lands either before or alongside, and the renderer operates on whichever root file the target writes.
- Auto-deleting an orphan `.hero/routing.md` if the user uninstalls Hero. The file is harmless; cleanup is separate concern.

## Risks

1. **Cursor `alwaysApply` mode-scope uncertainty.** If `alwaysApply: true` only applies to agent-mode (not chat-mode), users in chat-mode get no routing context. Mitigation: verify during delivery; if confirmed chat-mode-only-gap, fall Cursor back to bucket B (inline the rendered content into a generic Cursor instruction surface).

2. **opencode `instructions` ordering.** If opencode concatenates `instructions:` after `AGENTS.md`, the imperative framing in `routing.md` itself must work standalone. Designed for: yes — the framing is the first paragraph of `routing.md`. But verify by inspection during delivery.

3. **opencode.json merge regression.** Extending the merge to add `instructions: [".hero/routing.md"]` while preserving user entries is a JSON-array-append operation. The existing `mergeJSONFromData` may treat conflicts as overwrite. Risk of stomping user-curated instruction lists. Mitigation: explicit array-append helper with idempotent behavior (don't duplicate if already present).

4. **Bucket-B staleness in long-running installs.** A team that installs Hero once and never reinstalls will drift from current routing rules. The `hero check` warning is the mitigation, but users who don't run `hero check` won't see it. Acceptable risk; bucket A's near-zero drift is the gravitational pull toward fewer bucket-B targets over time.

5. **Embed-FS access pattern.** The routing renderer needs to read `routing.md` from `opts.sourceFS()`. Confirm that path is available in all installer code paths (project mode, global mode, dry-run). The existing `installConfig` for `opencode.json` already does this pattern; mirror it.

6. **Satellite workspaces and `.hero/routing.md` path resolution.** If a satellite's harness expects `.hero/routing.md` relative to the satellite's CLAUDE.md but the file only exists at the root workspace, the Claude import `@.hero/routing.md` resolves up the tree to the root file — verify that Claude Code's `@`-import does walk up the directory tree (it should, mirroring CLAUDE.md auto-discovery). If not, satellites need a symlinked or rendered local copy.

7. **CLAUDE.md `@.hero/routing.md` parsing.** Confirm Claude Code parses `@.hero/routing.md` inside a managed-region markdown block (not just outside). If the managed-region wrapper interferes, place the `@` line just before or just after the wrapper.

8. **Splitting framing creates a two-source string.** The imperative framing appearing in both the main instruction file and `routing.md` is a deliberate trade-off; drift between the two is a small ongoing risk. Mitigation: render the framing string from a shared Go constant used by both `generateAgentsMdBody` and the routing renderer.

## Validation

**Unit tests** (per target):

- Render to tmp dir; assert sidecar file path + content shape.
- Render to tmp dir; assert main instruction file managed-region content matches expected (pointer for bucket A; inlined for bucket B).
- Assert idempotency: second `hero install --target X` produces zero filesystem writes when source is unchanged.

**Fixture-based smoke test**:

- For each target, render install output to tmp dir, concatenate all bytes that would reach the model at session start (sidecar + main instruction file managed region), then verify every fixture phrasing in the table from `## Approach §7` is present along with its expected command. This is the regression guard — if a row goes missing for a target, the test fails.

**`hero check` integration test**:

- Install a bucket-B target. Mutate the main instruction file's routing block. Run `hero check`. Assert `routing-stale` warning appears with the right fix command.

**Manual verification** (one per target, by a maintainer with that harness installed):

- Open a fresh session in the target harness. Ask a routed-phrase question (e.g., "fix a rounding bug in the cart") and confirm the model routes to `/diagnose`. This is the live floor-check that bucket A's "always-on" feel is preserved.

**Migration check**:

- Install Hero against an existing repo with old-format CLAUDE.md (inline table inside managed region). Run `hero install --target claude`. Assert: managed region regenerates with new pointer form; `.hero/routing.md` lands; no orphan content; user content outside the managed region is byte-identical.
