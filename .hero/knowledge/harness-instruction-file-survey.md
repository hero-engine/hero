# Harness Instruction-File Survey (late 2025 / early 2026)

Research date: 2026-05-11. Compiled to inform Hero's install architecture: where each AI
coding harness reads project-level instructions, agents, commands, and skills, and which
harnesses allow Hero to consolidate everything inside `.hero/` instead of scattering
copies into harness-specific directories.

## TL;DR — the four findings that drive the install architecture

1. **AGENTS.md is the de facto cross-harness instruction file.** Stewarded by the
   Agentic AI Foundation (Linux Foundation) since Dec 2025, with native support in
   Codex, Cursor (Agent mode), GitHub Copilot, Windsurf, Amp, Junie, Roo Code,
   opencode, OpenHands, goose, Gemini CLI, Zed, Warp, Factory, and ~24 others.
   Claude Code is the **conspicuous holdout** — it still reads only `CLAUDE.md`,
   though Anthropic's official guidance is "use `@AGENTS.md` import or `ln -s
   AGENTS.md CLAUDE.md`."
2. **Claude Code has almost zero path configurability for agents/commands/skills.**
   `.claude/settings.json` exposes only `autoMemoryDirectory` and `plansDirectory`.
   There is no `agentsPath`, `skillsPath`, or `commandsPath`. Everything lives at
   the hardcoded `.claude/{agents,commands,skills,rules}/`. There is no
   config-redirect option — only symlink or rendered copies.
3. **Symlink reliability is mixed and recently improved for Claude Code.** Per
   official docs, `.claude/rules/` *explicitly supports symlinks* (both file and
   directory symlinks, with cycle detection). `CLAUDE.md → AGENTS.md` symlinks
   are officially recommended (caveat: Windows needs Admin / Developer Mode).
   For `.claude/agents/`, `.claude/commands/`, `.claude/skills/` the docs are
   silent — community reports say they generally work but historical bugs exist
   (issue #764). **Cline is known broken** for symlinks in `.clinerules/`
   (issue #3092 still open).
4. **opencode is the friendliest harness for an external `.hero/` layout** —
   `opencode.json` has an `instructions: []` field that accepts arbitrary
   paths AND glob patterns (e.g. `.hero/rules/*.md`). Codex is second-friendliest
   via `project_doc_fallback_filenames` and `model_instructions_file`. Most other
   harnesses are "hardcoded path, hope it follows symlinks."

Net effect on Hero install design:

- **One thing to write at the project root: `AGENTS.md`** — buys you ~9 harnesses
  native, no per-harness work.
- **One symlink: `CLAUDE.md → AGENTS.md`** — covers Claude Code on macOS/Linux.
  On Windows, write `CLAUDE.md` whose entire content is `@AGENTS.md`.
- **For agents/commands/skills:** symlink `.claude/{agents,commands,skills}` →
  `.hero/{agents,commands,skills}`. Document as "best-effort, fall back to copy
  with drift check on Windows or older Claude Code." For opencode, point
  `instructions` at `.hero/AGENTS.md` and the matching `.opencode/*` dirs at
  `.hero/*` (symlinks; opencode appears to follow them since it explicitly
  documents reading rule files via globs).
- **Codex:** put `AGENTS.md` at root (covered already), no further action needed.
  Optionally add `project_doc_fallback_filenames = [".hero/AGENTS.md"]` for
  monorepo subroot installs.

The "rendered copy with drift detection" mode is only needed for: Cline
(symlinks broken), Windows-without-Developer-Mode, and any harness where the
user hasn't enabled `git config core.symlinks=true`.

---

## 1. Per-harness reference table

Legend for "Custom path":
- **YES** = harness has a documented config knob to point at an arbitrary path.
- **SYMLINK** = path is hardcoded, but harness follows filesystem symlinks.
- **NO** = path is hardcoded and symlinks are unreliable/undocumented.
- **N/A** = harness has no such concept.

### Claude Code (Anthropic) — v2.1.126 as of May 2026

| Aspect | Behavior |
| --- | --- |
| Instruction file | `CLAUDE.md` (+ `CLAUDE.local.md`, `.claude/CLAUDE.md`). **Does NOT natively read `AGENTS.md`** — open issue #6235 since Aug 2025. Official workaround: `@AGENTS.md` import inside `CLAUDE.md`, or `ln -s AGENTS.md CLAUDE.md`. |
| Instruction path | `claudeMdExcludes` lets you exclude; **no include-redirect setting**. Imports via `@path` (depth 5) and `--add-dir` + `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` for sibling dirs. |
| Agents path | `.claude/agents/` hardcoded. No `agentsPath` setting. |
| Commands path | `.claude/commands/` hardcoded. No `commandsPath` setting. |
| Skills path | `.claude/skills/` hardcoded. Officially the *recommended* extension surface as of v2.1.101. No `skillsPath` setting. |
| Rules path | `.claude/rules/` — added in 2.x. **Explicitly supports symlinks** (file and directory, with cycle detection) per official docs. |
| Settings file | `.claude/settings.json`, `.claude/settings.local.json`, `~/.claude/settings.json`, managed-policy locations. Configurable path knobs: `autoMemoryDirectory`, `plansDirectory`, `claudeMdExcludes`. **Nothing for agents/commands/skills.** |
| Symlinks | `.claude/rules/` officially supported. `CLAUDE.md → AGENTS.md` officially recommended. `.claude/agents/`, `.claude/commands/`, `.claude/skills/` undocumented but appear to work (historical fix in 2.1.101 explicitly resolves symlink targets for Read tool permissions). Pre-2.1.64 there was a CVE about following symlinks out of workspace. |
| Plugin system | Yes — `.claude-plugin/` with `plugin.json` + bundled `agents/commands/skills/`. Marketplace ecosystem exists. |
| AGENTS.md support | **Unsupported natively.** Workaround via `@AGENTS.md` import in `CLAUDE.md` or symlink. |
| Sources | https://code.claude.com/docs/en/memory · https://code.claude.com/docs/en/settings · https://github.com/anthropics/claude-code/issues/6235 · https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md · https://github.com/anthropics/claude-code/issues/764 |

### opencode (sst/opencode)

| Aspect | Behavior |
| --- | --- |
| Instruction file | Walks up directory tree for `AGENTS.md`, `CLAUDE.md`. Reads `~/.config/opencode/AGENTS.md` globally. Reads `~/.claude/CLAUDE.md` as fallback unless disabled. |
| Instruction path | **YES — `opencode.json` has `instructions: []` field accepting arbitrary paths AND glob patterns** (e.g. `".cursor/rules/*.md"`, `"docs/guidelines.md"`, remote URLs). All concatenated with discovered `AGENTS.md`. |
| Agents path | `.opencode/agents/` or `~/.config/opencode/agents/`. Plural canonical; singular `agent/` accepted for backcompat. Can also inline-define agents in `opencode.json`. |
| Commands path | `.opencode/commands/`. Same plural convention. |
| Skills path | `.opencode/skills/` (Anthropic SKILL.md format). Feature request open (issue #14370) for `skills.paths` config; not yet shipped. |
| Settings file | `opencode.json` / `opencode.jsonc`. Env: `OPENCODE_CONFIG`, `OPENCODE_CONFIG_DIR`. |
| Symlinks | Not explicitly documented either way. Glob discovery suggests they follow filesystem semantics; no known issue reports. |
| AGENTS.md support | **Native, first-class** — searched alongside `CLAUDE.md`. |
| Sources | https://opencode.ai/docs/config/ · https://opencode.ai/docs/rules/ · https://opencode.ai/docs/agents/ · https://opencode.ai/docs/skills/ · https://github.com/anomalyco/opencode/issues/14370 |

### Codex CLI (OpenAI)

| Aspect | Behavior |
| --- | --- |
| Instruction file | **`AGENTS.md` native and primary.** Discovery walks from git root down. Precedence: `AGENTS.override.md` > `AGENTS.md` > fallbacks. Global at `~/.codex/AGENTS.md`. |
| Instruction path | **YES** — `model_instructions_file` (user/profile-level) replaces built-in. `project_doc_fallback_filenames = [...]` adds alternates when `AGENTS.md` missing. `project_doc_max_bytes` caps size. `project_root_markers` controls root detection. |
| Agents path | `agents.<name>.config_file` in `.codex/config.toml` — TOML config files, paths relative to config file. |
| Commands path | Not standardized as a directory convention; commands are typically prompts or inline. |
| Skills path | `skills.config[].path` in config.toml — points at a folder containing `SKILL.md`. **Path is fully configurable.** |
| Settings file | `~/.codex/config.toml` (user), `.codex/config.toml` (project). Env: `CODEX_HOME`. |
| Symlinks | Standard filesystem; no special handling claimed but no broken-symlink reports either. |
| AGENTS.md support | **Native, primary mechanism — Codex helped define the spec.** |
| Sources | https://developers.openai.com/codex/guides/agents-md · https://developers.openai.com/codex/config-reference · https://developers.openai.com/codex/config-sample |

### Cursor

| Aspect | Behavior |
| --- | --- |
| Instruction file | Three concurrent surfaces: legacy `.cursorrules` (still read); `.cursor/rules/*.mdc` (current canonical); `AGENTS.md` (read by Agent mode only; Chat/Composer ignore it). Nested `AGENTS.md` in subdirs supported. |
| Instruction path | Not configurable to arbitrary paths. Rules must live under `.cursor/rules/`. |
| Agents path | N/A as a separate concept (uses rules + modes). |
| Commands path | N/A. |
| Skills path | Adopts Anthropic SKILL.md format (per Agent Skills standard); discovery path not clearly documented in primary sources. |
| Symlinks | No specific guarantees; community reports of symlinking `.cursor/rules/` → shared dir generally work. |
| AGENTS.md support | **Native for Agent mode**, not for Chat/Composer. Caveat important. |
| Sources | https://cursor.com/docs/rules · https://docs.cursor.com/ |

### GitHub Copilot

| Aspect | Behavior |
| --- | --- |
| Instruction file | `.github/copilot-instructions.md` (legacy primary). Also reads `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` natively — "nearest in directory tree wins." Nested AGENTS.md supported. |
| Instruction path | Multiple files supported in `.github/instructions/*.instructions.md`. Path within `.github/` not configurable. |
| Agents path | Custom agents via `.github/copilot/agents/` and VS Code config for VS / IntelliJ. |
| Commands path | N/A standardized. |
| Skills path | Adopts Anthropic Agent Skills standard (16+ tools as of Mar 2026). |
| Symlinks | Standard git/fs behavior; no specific issues reported. |
| AGENTS.md support | **Native** since Aug 2025 changelog. JetBrains IDE Copilot added support Mar 2026. |
| Sources | https://github.blog/changelog/2025-08-28-copilot-coding-agent-now-supports-agents-md-custom-instructions/ · https://docs.github.com/copilot/customizing-copilot/adding-custom-instructions-for-github-copilot |

### Aider

| Aspect | Behavior |
| --- | --- |
| Instruction file | `CONVENTIONS.md` is canonical. **AGENTS.md supported** when listed in `.aider.conf.yml` under `read:`. Aider also auto-reads `AGENTS.md` in recent versions. |
| Instruction path | **YES — fully configurable.** `.aider.conf.yml` `read:` key takes file or list of files. |
| Agents path | N/A — aider is a pair-programmer not multi-agent. |
| Commands path | N/A. |
| Skills path | N/A natively. |
| Symlinks | Standard. |
| AGENTS.md support | **Configurable, effectively native** — declare it in conf or pass `--read AGENTS.md`. |
| Sources | https://aider.chat/docs/usage/conventions.html · https://aider.chat/docs/config/aider_conf.html |

### Cline

| Aspect | Behavior |
| --- | --- |
| Instruction file | `.clinerules/` directory of `.md`/`.txt` files (newer; replaces old single-file). VS Code setting `cline.customInstructions` can point at a single file path (e.g. AGENTS.md anywhere on disk). |
| Instruction path | VS Code setting allows arbitrary file path for `customInstructions`. The `.clinerules/` directory itself is hardcoded. |
| Agents path | N/A. |
| Commands path | N/A. |
| Skills path | N/A natively (third-party plugins exist). |
| Symlinks | **Known broken** — issue #3092 (open) reports Cline ignores symlinks in `.clinerules/`. Hardlinks (Windows `mklink /h`) reported as workaround. |
| AGENTS.md support | Via the `customInstructions` VS Code setting, or via discussion #6162 (open feature request for native AGENTS.md in `.clinerules/`). |
| Sources | https://docs.cline.bot/customization/cline-rules · https://github.com/cline/cline/issues/3092 · https://github.com/cline/cline/discussions/6162 |

### Continue.dev

| Aspect | Behavior |
| --- | --- |
| Instruction file | `.continue/rules/*.md`. Rules concatenated into system message. |
| Instruction path | `config.yaml` `rules:` block can reference inline rules; path-config of rules directory not clearly documented as redirectable. |
| Agents path | Agents = models + rules + tools, defined in `config.yaml`. |
| Commands path | Inline in `config.yaml`. |
| Skills path | N/A as of search. |
| Symlinks | Not specifically documented. |
| AGENTS.md support | **Not native** — issue #6716 (open) requests AGENTS.md support. |
| Sources | https://docs.continue.dev/reference · https://docs.continue.dev/customize/rules · https://github.com/continuedev/continue/issues/6716 |

### Windsurf (Codeium)

| Aspect | Behavior |
| --- | --- |
| Instruction file | `.windsurfrules` (root) and `.windsurf/rules/*.md`. **AGENTS.md natively supported** by Cascade with directory-scoped activation, walking up to git root. Case-insensitive (`AGENTS.md`/`agents.md`). |
| Instruction path | Not configurable; fixed discovery locations. |
| Agents path | N/A (uses Cascade). |
| Commands path | `.windsurf/workflows/`. |
| Skills path | Anthropic Agent Skills supported. |
| Symlinks | Not specifically documented. |
| AGENTS.md support | **Native** with subdirectory scoping. |
| Sources | https://docs.windsurf.com/windsurf/cascade/agents-md |

### Sourcegraph Amp

| Aspect | Behavior |
| --- | --- |
| Instruction file | **AGENTS.md native, primary.** Walks from CWD and editor workspace roots + parents. |
| Instruction path | Not configurable. |
| Agents path | N/A standardized. |
| Commands path | N/A. |
| Skills path | Anthropic Agent Skills supported. |
| Symlinks | Standard. |
| AGENTS.md support | **Native, primary**; Amp helped originate the spec. |
| Sources | https://sourcegraph.com/amp · https://github.com/sourcegraph/amp-examples-and-guides |

### JetBrains Junie

| Aspect | Behavior |
| --- | --- |
| Instruction file | Discovery order: `.junie/AGENTS.md` → `AGENTS.md` → `.junie/guidelines.md` (legacy) → `.junie/guidelines/` folder. |
| Instruction path | Not configurable. |
| Agents path | N/A as separate concept. |
| Commands path | N/A. |
| Skills path | Anthropic Agent Skills supported (Junie was an early adopter). |
| Symlinks | Standard. |
| AGENTS.md support | **Native, with two priority locations.** Junie tracks AGENTS.md status at YouTrack JUNIE-618. |
| Sources | https://junie.jetbrains.com/docs/guidelines-and-memory.html · https://junie.jetbrains.com/docs/agent-skills.html |

### OpenHands (formerly OpenDevin)

| Aspect | Behavior |
| --- | --- |
| Instruction file | V1: `.openhands/skills/` (e.g. `repo.md`). V0: `.openhands/microagents/repo.md`. AGENTS.md is read as repo summary per docs (referenced in their own repo). |
| Instruction path | Not configurable; hardcoded discovery. |
| Agents path | N/A (uses skills). |
| Commands path | N/A. |
| Skills path | `.openhands/skills/` (V1, preferred) or `.openhands/microagents/` (V0 backcompat). |
| Symlinks | Standard. |
| AGENTS.md support | Used as repo overview file; primary mechanism is `.openhands/skills/repo.md`. |
| Sources | https://docs.all-hands.dev/usage/prompting/microagents-repo · https://github.com/OpenHands/OpenHands/blob/main/skills/README.md |

### Roo Code

| Aspect | Behavior |
| --- | --- |
| Instruction file | `.roo/rules/`, `.roorules-{mode}`, `~/.roo/rules/`. **AGENTS.md (or AGENT.md fallback) supported natively** — loaded after mode-specific rules. |
| Instruction path | Toggle via `roo-cline.useAgentRules`. Not redirectable to arbitrary paths. |
| Agents path | Custom modes / agents defined in `.roomodes`. |
| Commands path | N/A. |
| Skills path | Anthropic Agent Skills supported. |
| Symlinks | **Officially: symbolic links to files or directories are resolved before reading.** Explicit doc statement. |
| AGENTS.md support | **Native, version-controlled standard.** |
| Sources | https://docs.roocode.com/features/custom-instructions · https://docs.roocode.com/features/skills · https://github.com/RooCodeInc/Roo-Code/blob/main/AGENTS.md |

### goose (Block → AAIF)

| Aspect | Behavior |
| --- | --- |
| Instruction file | `.goosehints` (legacy) AND `AGENTS.md` (walks git root down, override > main > fallbacks — same convention as Codex). |
| Instruction path | Similar Codex-style overrides documented. |
| Agents path | Extensions system (not directory-based agents). |
| Commands path | N/A. |
| Skills path | Anthropic Agent Skills supported. |
| Symlinks | Standard. |
| AGENTS.md support | **Native.** Goose was donated to AAIF same time AGENTS.md was; tightly aligned. |
| Sources | https://github.com/block/goose/blob/main/AGENTS.md · https://goose-docs.ai/ |

### Gemini CLI / Google Jules

| Aspect | Behavior |
| --- | --- |
| Instruction file | AGENTS.md supported (per agents.md adopter list). Google-specific files: `GEMINI.md`. |
| Skills path | `~/.gemini/antigravity/skills/` per Agent Skills adopters report. |
| AGENTS.md support | **Native.** |
| Sources | agents.md adopter list (https://agents.md/); paperclipped agent-skills survey |

---

## 2. Cross-harness convergence summary

### AGENTS.md adoption (the only convergence that actually exists)

**Stewardship:** Agentic AI Foundation (AAIF) under Linux Foundation since Dec 2025.
Anthropic donated MCP, Block donated goose, OpenAI donated AGENTS.md at the same
formation event.

**Adoption (May 2026):**

- Native, primary: Codex CLI, Amp (Sourcegraph), goose, Windsurf, opencode,
  Junie, Roo Code, Gemini CLI, Jules, Factory, Devin, Cursor (Agent mode),
  GitHub Copilot, Zed, Warp.
- Native, secondary: VS Code (via Copilot integration).
- Configurable / opt-in: Aider (`.aider.conf.yml` `read:`).
- Workaround only: Claude Code (use `@AGENTS.md` import or symlink to CLAUDE.md).
- Open feature requests: Cline (#6162), Continue (#6716).

**60k+ open-source repos report adopting AGENTS.md** per the agents.md homepage.

### `.ai/` and `.agents/` directories

**No meaningful adoption.** Searches for `.ai/` and `.agents/` as cross-harness
conventions return blog posts proposing them but **no harness reads them by
default.** The community pattern of "centralize in `.agents/` and symlink
`.claude/`, `.cursor/` etc. to it" exists, but every harness still wants its
own hardcoded directory.

### Anthropic Agent Skills (SKILL.md)

Released Dec 18, 2025. 32 adopters by Mar 2026 per agentskills.io. **Format is
standardized; install paths are NOT.** Each harness chose its own dir:

- Claude Code: `.claude/skills/`
- Codex: `.agents/skills/` *(close to a neutral location but Codex-specific)*
- Gemini: `~/.gemini/antigravity/skills/`
- OpenHands: `.openhands/skills/`
- opencode: `.opencode/skills/`
- Junie: `.junie/skills/` (via JetBrains skills system)
- Roo Code: in `.roo/`

Skills' SKILL.md format is the only true cross-harness payload — the directory
is per-harness.

---

## 3. Recommendation matrix — cleanest install mode per harness

Goal: Hero installs once into `.hero/` with `agents/`, `commands/`, `skills/`,
`rules/`, and `AGENTS.md` (or a generator that produces it). Pick the cheapest
viable mode per harness in this priority: **shared file > config redirect >
symlink > rendered copy**.

| Harness | Instructions | Agents | Commands | Skills | Notes |
| --- | --- | --- | --- | --- | --- |
| **Codex CLI** | Shared `AGENTS.md` at root (native). Optional `project_doc_fallback_filenames` for nested. | Config-redirect via `agents.<n>.config_file` | N/A | Config-redirect via `skills.config[].path` | Cleanest. Pure config. |
| **opencode** | Config-redirect `instructions: [".hero/AGENTS.md", ".hero/rules/*.md"]` in `opencode.json` | Symlink `.opencode/agents → .hero/agents` (or accept default and write there) | Symlink `.opencode/commands → .hero/commands` | Symlink `.opencode/skills → .hero/skills` | Very clean. Glob support in `instructions` is the killer feature. |
| **Claude Code** | Symlink `CLAUDE.md → AGENTS.md` (mac/Linux) OR write `CLAUDE.md` containing `@AGENTS.md` (Windows-safe). Symlink `.claude/rules → .hero/rules` (officially supported). | Symlink `.claude/agents → .hero/agents` (works but undocumented; verify per Hero release) | Symlink `.claude/commands → .hero/commands` (same caveat) | Symlink `.claude/skills → .hero/skills` (same caveat) | No config-redirect option exists. Symlink is the only non-copy path. **Fallback to rendered copies + drift detection on Windows-without-Developer-Mode or where `git config core.symlinks=true` is not set.** |
| **Cursor** | Shared `AGENTS.md` (Agent mode only). For Chat/Composer: write `.cursor/rules/hero.mdc` that imports/references `.hero/rules/*.md`. | N/A | N/A | Anthropic Skills convention; path unclear | Mixed: AGENTS.md covers Agent mode; rules need a stub. |
| **GitHub Copilot** | Shared `AGENTS.md` (native, all surfaces incl JetBrains as of Mar 2026). Also covers nested `AGENTS.md`. | `.github/copilot/agents/` — symlink if Hero ships agents, otherwise leave alone. | N/A | Agent Skills format | Cleanest. AGENTS.md does the work. |
| **Aider** | Config-redirect: add `read: [AGENTS.md, .hero/rules/*.md]` to `.aider.conf.yml` (paths arbitrary). | N/A | N/A | N/A | Cleanest. Pure config redirect. |
| **Cline** | VS Code setting `cline.customInstructions` → `${workspaceFolder}/AGENTS.md`. For `.clinerules/`: **must use rendered copies** (symlinks broken per #3092). | N/A | N/A | N/A | Only harness that genuinely forces copies. |
| **Continue.dev** | Inline rules in `config.yaml` referencing `.hero/rules/*.md`. AGENTS.md not native (issue #6716 open). | N/A | N/A | N/A | Config-redirect via YAML inline. |
| **Windsurf** | Shared `AGENTS.md` (native, walks git root). | N/A | `.windsurf/workflows/` — symlink if needed. | Agent Skills supported. | Cleanest. |
| **Amp** | Shared `AGENTS.md` (native, primary). | N/A | N/A | Agent Skills. | Cleanest. |
| **Junie** | Symlink `.junie/AGENTS.md → ../AGENTS.md` (or just let root `AGENTS.md` be picked up — it's also discovered). | N/A | N/A | Agent Skills. | Cleanest. |
| **OpenHands** | Write `.openhands/skills/repo.md` whose first line is `@AGENTS.md` equivalent — OR symlink. | N/A | N/A | `.openhands/skills/` (V1) — symlink to `.hero/skills` if Hero ships skills. | Mid: needs a stub or symlink. |
| **Roo Code** | Shared `AGENTS.md` (native). Optionally symlink `.roo/rules → .hero/rules` (Roo officially resolves symlinks). | `.roomodes` for modes. | N/A | Agent Skills. | Cleanest. Roo is the only harness with an *explicit symlink-resolution guarantee* in its docs. |
| **goose** | Shared `AGENTS.md` (native, Codex-style discovery). | Extensions system. | N/A | Agent Skills. | Cleanest. |
| **Gemini CLI / Jules** | Shared `AGENTS.md` (native per AAIF adopter list). | N/A | N/A | `~/.gemini/antigravity/skills/` — Hero would need symlink for shared skills. | Cleanest for instructions; skills path is user-global so harder to project-scope. |

### Hero install architecture implications

1. **Always** write/maintain a single root `AGENTS.md` (or symlink to
   `.hero/AGENTS.md`). This file alone wins for 11 of the 15 harnesses.
2. **Always** ensure `CLAUDE.md` exists — either as a symlink to `AGENTS.md`
   (POSIX, or Windows w/ Developer Mode + core.symlinks=true) or as a one-line
   `@AGENTS.md` import file. Detect symlink capability at install time.
3. **Always** ensure `.claude/rules/` symlinks to `.hero/rules/` if Hero ships
   rules — officially supported, zero risk.
4. **Mostly** symlink `.claude/{agents,commands,skills}` → `.hero/{...}` —
   community-validated but not officially blessed. Add an `install verify`
   command that detects broken symlinks and re-renders.
5. **opencode users:** drop a templated `opencode.json` adding
   `instructions: [".hero/AGENTS.md", ".hero/rules/**/*.md"]`. This is the
   strongest config-redirect path available anywhere.
6. **Codex users:** AGENTS.md at root covers it. For monorepo satellite
   installs, add `project_doc_fallback_filenames = [".hero/AGENTS.md"]` to
   `.codex/config.toml`.
7. **Cline users:** render copies into `.clinerules/` and run a drift check.
   This is the only forced-copy path in the matrix.
8. **Windows-without-symlinks:** treat as Cline-tier — render copies into
   every harness dir, run drift detection. This will be a noticeable minority
   of users but must be supported.

### Suggested Hero install modes (in declining preference)

- **`shared`** mode: just `AGENTS.md` + `CLAUDE.md` (symlink or `@import`).
  Works for 11 harnesses out of the box. ~zero clutter.
- **`linked`** mode: `shared` plus symlinks `.claude/{agents,commands,skills,rules}`
  → `.hero/...`, `.opencode/{agents,commands,skills}` → `.hero/...`,
  `.openhands/skills/` → `.hero/skills`, etc. Adds `opencode.json` and
  `.aider.conf.yml` config-redirect stubs.
- **`rendered`** mode: `linked` falls back to file copies on each install for
  harnesses/platforms where symlinks fail or are unavailable. Includes
  `hero verify-install` to detect drift between `.hero/` source and rendered
  copies.

---

## 4. Open questions needing empirical verification

Items the docs do not pin down but matter for the install architecture:

1. **Claude Code symlink behavior for `.claude/agents/`, `.claude/commands/`,
   `.claude/skills/`** — only `.claude/rules/` is *officially* documented to
   support symlinks. The other three appear to work based on community reports
   and the fact that the loader walks the directories with standard fs calls,
   but Anthropic has not stated this explicitly. **Test in current Claude Code
   2.1.x:** create a symlink, restart, see if items load.
2. **Claude Code behavior with `CLAUDE.md` as a symlink that points OUTSIDE the
   workspace** — post-CVE-2026-39861 fix, symlink-following was tightened.
   Verify that `CLAUDE.md → ../shared/AGENTS.md` still resolves rather than
   being treated as a sandbox escape attempt. Within-workspace symlinks should
   be fine.
3. **opencode symlink behavior for `.opencode/agents/`, etc.** — docs don't say.
   No bug reports either. Test empirically.
4. **Windows + `core.symlinks=true` + Git for Windows** — symlinks should work
   if Developer Mode is on. But the practical fraction of Hero users who have
   this set up is unknown. Need a Windows install path that auto-detects and
   falls back to rendered copies if `ln -s` fails.
5. **Cursor: does `.cursor/rules/` follow symlinks?** Not documented. The
   `.cursor/rules/foo.mdc → .hero/rules/foo.md` pattern is community-suggested
   but unverified.
6. **Continue.dev: can `config.yaml` `rules` reference external file globs**
   the way opencode's `instructions:` does? Docs imply inline-only; needs
   verification.
7. **OpenHands V1: does `.openhands/skills/repo.md` support `@import`-style
   references**, or must content be inline? If inline-only, a generator step
   is needed instead of a symlink.
8. **Anthropic Agent Skills frontmatter** — confirmed required fields are
   `name` and `description`. Anything else (e.g. `model`, `tools`,
   `auto_load`) is harness-specific and ornamental from the spec's POV. But
   *Claude Code* extends SKILL.md frontmatter with its own fields (e.g. the
   skills auto-load flag added for subagents). A skill written for Claude Code
   may not load identically in opencode. **Test SKILL.md portability across
   harnesses.**
9. **Plugins as an install vector for Claude Code** — instead of `.claude/`
   symlinks, Hero could ship a `.claude-plugin/` and use the plugin marketplace
   path. This sidesteps the path-config gap entirely. Worth a separate spike.
10. **Whether `claude /init` will read `AGENTS.md`** if `CLAUDE.md` already
    exists as a symlink to it — per docs, `/init` reads existing AGENTS.md and
    folds it into the generated CLAUDE.md. If the user runs `/init` on a
    Hero-installed repo, does it overwrite the symlink? Needs verification or
    a doc note.

---

## Sources

- AGENTS.md spec & adopters: https://agents.md/
- AAIF formation: https://www.linuxfoundation.org/press/linux-foundation-announces-the-formation-of-the-agentic-ai-foundation
- Claude Code memory: https://code.claude.com/docs/en/memory
- Claude Code settings: https://code.claude.com/docs/en/settings
- Claude Code AGENTS.md feature request: https://github.com/anthropics/claude-code/issues/6235
- Claude Code symlink bug (resolved): https://github.com/anthropics/claude-code/issues/764
- Claude Code symlink discussion: https://news.ycombinator.com/item?id=45788866
- Claude Code CHANGELOG: https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md
- Codex AGENTS.md guide: https://developers.openai.com/codex/guides/agents-md
- Codex config reference: https://developers.openai.com/codex/config-reference
- opencode config: https://opencode.ai/docs/config/
- opencode rules: https://opencode.ai/docs/rules/
- opencode agents: https://opencode.ai/docs/agents/
- opencode skill paths feature request: https://github.com/anomalyco/opencode/issues/14370
- Cursor rules: https://cursor.com/docs/rules
- GitHub Copilot AGENTS.md changelog: https://github.blog/changelog/2025-08-28-copilot-coding-agent-now-supports-agents-md-custom-instructions/
- Copilot JetBrains AGENTS.md: https://github.blog/changelog/2026-03-11-major-agentic-capabilities-improvements-in-github-copilot-for-jetbrains-ides/
- Aider conventions: https://aider.chat/docs/usage/conventions.html
- Aider conf: https://aider.chat/docs/config/aider_conf.html
- Windsurf AGENTS.md: https://docs.windsurf.com/windsurf/cascade/agents-md
- Junie guidelines: https://junie.jetbrains.com/docs/guidelines-and-memory.html
- Junie skills: https://junie.jetbrains.com/docs/agent-skills.html
- Sourcegraph Amp: https://sourcegraph.com/amp
- Roo Code custom instructions: https://docs.roocode.com/features/custom-instructions
- Roo Code AGENTS.md: https://github.com/RooCodeInc/Roo-Code/blob/main/AGENTS.md
- Cline rules: https://docs.cline.bot/customization/cline-rules
- Cline symlink bug: https://github.com/cline/cline/issues/3092
- Cline AGENTS.md feature request: https://github.com/cline/cline/discussions/6162
- Continue.dev rules: https://docs.continue.dev/customize/rules
- Continue.dev AGENTS.md feature request: https://github.com/continuedev/continue/issues/6716
- OpenHands skills: https://github.com/OpenHands/OpenHands/blob/main/skills/README.md
- OpenHands microagents: https://docs.all-hands.dev/usage/prompting/microagents-repo
- goose AGENTS.md: https://github.com/block/goose/blob/main/AGENTS.md
- Agent Skills interoperability: https://www.paperclipped.de/en/blog/agent-skills-open-standard-interoperability/
- Symlink-sharing community guide: https://www.rushis.com/sharing-ai-agent-configs-between-cursor-and-claude-with-symlinks/
- SSW symlink rule: https://www.ssw.com.au/rules/symlink-agents-to-claude
