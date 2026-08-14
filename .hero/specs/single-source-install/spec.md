---
title: Single-Source Install — One Canonical Tree, Every Harness Reads It
slug: single-source-install
type: initiative
status: planning
priority: P0
tags: [install, upgrade, harness, agents-md, cleanup, multi-harness, hygiene]
created: 2026-05-11
relations:
  - target: hero-platform
    kind: parent
  - target: hero-team-experience
    kind: related
  - target: harness-instruction-file-survey
    kind: motivated-by
horizon: now
mission_alignment: |
  Hero's job is to make every harness session start smarter than the last.
  When a project has three trees of duplicated agent files drifting against
  each other, three instruction files lying about what's current, and no
  single place to edit a convention — that mission fails before the model
  ever loads. This initiative makes the corpus structurally honest: one
  canonical content tree, one user-edited instruction file, every harness
  reads the same bytes. The floor rises for everyone because no agent ever
  sees stale context drawn from a copy that drifted six weeks ago.
principles_check: |
  Serves "it just works" (#1) by collapsing N install targets to one source
  of truth — edits propagate without thinking. Serves "the floor rises for
  everyone" (#4) by ending the per-harness expertise tax (knowing where
  Claude reads from vs where opencode reads from). Risks "it just works"
  if migration of existing messy installs is unreliable; mitigated by
  Phase 3's idempotent `--migrate` with verification + dry-run. Risks
  cross-platform portability via symlinks; mitigated by Phase 4's
  rendered-copy fallback with `hero verify-install` drift detection.
size: giant
---

> # ✏️ AMENDED (2026-07-12) — root-instruction-file model reversed
> **What still stands:** the *canonical content tree* goal — one copy of
> `.hero/{agents,commands,skills}` on disk, harnesses reach it via
> symlink/config-redirect/rendered-copy, no drifted duplicate agent/skill trees
> (Phases 2 & 4). That is the durable core of this initiative and is unchanged.
>
> **What changed:** the *root instruction file* thesis — "**one** root file at
> `AGENTS.md`, every harness reads it, `CLAUDE.md → AGENTS.md` symlink" — is
> **reversed**. Hero's install model is now **harness-native, target-aware**:
> `--target claude` → CLAUDE.md only; other targets → AGENTS.md; multi-target →
> both; and `hero upgrade` only touches the native files of previously-installed
> targets. **Phase 1 is superseded** by
> [`harness-native-install-target-aware-upgrade`](../../features/harness-native-install-target-aware-upgrade/spec.md).
> Read statements below about "the only root instruction file / AGENTS.md as the
> single canonical instruction file / CLAUDE.md symlink" through that lens — they
> are historical and describe the retired convergence model.

## Goal

A Hero-installed project has **one** canonical content tree in `.hero/`,
**one** root instruction file at `AGENTS.md`, and zero duplicated content
across harness directories. Every supported harness reads from the
canonical via config-redirect, symlink, or rendered-copy fallback,
chosen automatically based on what the harness supports and what the
host filesystem allows. `hero install` and `hero upgrade` are the only
operations that touch harness directories, and both are idempotent,
non-destructive of user content, and capable of cleaning up legacy
messy installs.

## The mess this exists to end

Today, a multi-harness Hero project accumulates:

- `.claude/agents/`, `.opencode/agent/`, possibly `.cursor/rules/`,
  `.openhands/skills/`, etc. — each with a physical copy of the same
  agent/command/skill files. Files drift over time because installs
  run at different moments.
- `CLAUDE.md`, `AGENTS.md`, maybe `.cursorrules`, `.windsurfrules`,
  `.github/copilot-instructions.md` — multiple instruction files
  saying overlapping things. The model sees inconsistent guidance
  depending on which harness is in the loop.
- Zero source of truth. The user doesn't know which file to edit.
  Hero doesn't know which copy is canonical. Drift is undetectable
  without an external diff.

The example codebase project today exhibits the textbook version of this:
34 agent files in `.claude/agents/` and `.opencode/agents/`, **14 of
which have drifted** between the two trees. 44 skills present in
`.claude/skills/` and **79 in `.opencode/skills/`** — one tree was
upgraded more recently. A 28KB hand-written `AGENTS.md` sits beside
a 31-line `CLAUDE.md` stub, neither aware of the other.

This isn't example codebase's bug. It's the inevitable outcome of Hero
writing the same content into N harness-specific destinations and
expecting them to stay in sync.

## The architecture

```
project-root/
├── .hero/                                # canonical content (single source of truth)
│   ├── agents/                           # one copy on disk
│   ├── commands/                         # one copy on disk
│   ├── skills/                           # one copy on disk
│   ├── instructions/
│   │   └── hero-managed.md               # block injected into AGENTS.md
│   ├── specs/, knowledge/, planning/     # existing
│   └── ...
│
├── AGENTS.md                             # the only root instruction file
│   ├── <user content>                    # user owns this; Hero never touches
│   └── <!-- hero:managed-start v=X -->   # Hero owns this region
│       <generated content>
│       <!-- hero:managed-end -->
│
├── CLAUDE.md → AGENTS.md                 # symlink (or "@AGENTS.md" one-line shim on Windows)
│
├── .claude/                              # harness-specific glue, no duplicate content
│   ├── agents → ../.hero/agents          # directory symlink
│   ├── commands → ../.hero/commands
│   ├── skills → ../.hero/skills
│   ├── rules → ../.hero/skills           # if rules-as-skills is preferred
│   └── settings.json                     # genuine harness config (hero-managed top, user bottom)
│
├── .opencode/                            # opencode supports config-redirect — no symlinks needed
│   └── (empty or minimal — content via opencode.json instructions[])
│
├── opencode.json                         # hero-managed top, user-extensible
│   {
│     "instructions": [".hero/AGENTS.md", ".hero/rules/*.md"]
│   }
│
├── .codex/config.toml                    # hero-managed top, user-extensible
├── .cursor/rules → ../.hero/skills       # symlink if .md acceptable; else rendered
├── .github/copilot-instructions.md → AGENTS.md   # symlink if installed
└── .windows-no-symlinks-fallback/        # rendered copies only when symlinks unavailable
```

**Properties this delivers:**

1. **Drift becomes structurally impossible.** There's nothing to drift
   from — every harness reads the same bytes.
2. **One place to edit.** A user editing an agent does it in
   `.hero/agents/engineer.md`. A user editing instructions does it
   in `AGENTS.md`. No "which file owns this?" question.
3. **Install/upgrade is trivial.** Touch `.hero/` only. Symlinks and
   harness configs need no updates unless a target is added or removed.
4. **Add a harness with one command.** `hero install --target cursor`
   creates the symlink. `hero uninstall --target cursor` removes it.
   Canonical is untouched either way.
5. **User instruction content survives forever.** Hero's contribution
   to `AGENTS.md` is a marked region. User content above and below is
   sacred.

## Phases

This initiative is delivered as four feature specs that ship
independently but together produce the full value. Each phase
materially improves the situation for new and existing projects;
none is optional for the architecture to be coherent.

### Phase 1 — ⛔ SUPERSEDED — Harness-native root instruction files
*(retired spec: `single-source-install-p1-agents-md`; replaced by
`.hero/planning/features/harness-native-install-target-aware-upgrade/`)*

**Original (retired) intent:** replace the per-harness root instruction file
with a single `AGENTS.md`, with a `CLAUDE.md → AGENTS.md` shim/symlink for
Claude Code. **This convergence model was reversed.**

**Current model (harness-native, target-aware):** `hero install` writes each
installed target's *native* root file — `--target claude` → CLAUDE.md only,
every other target → AGENTS.md, multi-target-with-claude → both — each with the
same Hero-managed block. `hero upgrade` regenerates only the native files of
previously-installed targets (persisted in `install-state.json`) and never
creates a CLAUDE.md if claude was never a target. The managed-region mechanics
(versioned markers, preserve-user-content, idempotence) carry over unchanged.

**User-visible win:** every harness gets exactly the file it natively reads —
no phantom AGENTS.md in a Claude-only repo, and upgrade stays faithful to what
was installed.

### Phase 2 — Canonical `.hero/{agents,commands,skills}/` + harness-aware install modes
*(spec: `.hero/planning/features/single-source-install-p2-canonical-tree/`)*

Move agent/command/skill content into `.hero/` as the canonical tree.
Per harness, choose install mode by capability:

- **config-redirect** where the harness allows pointing at custom
  paths (opencode `instructions[]`, codex `model_instructions_file`,
  aider `read:`).
- **directory symlinks** where the harness has hardcoded paths but
  follows filesystem links (Claude Code, most others).
- **rendered copies + drift detection** where symlinks don't work
  (Cline, Windows without Developer Mode).

`hero install --target X` materializes whichever mode applies for X.

**User-visible win:** no more duplicated agent trees. Edit once, every
harness sees it.

### Phase 3 — `hero install --migrate` for legacy messy installs
*(spec: TBD — write after P1 + P2 ship)*

Idempotent migration of existing pre-architecture installs. Detects
physical copies in harness dirs, reconciles them against `.hero/`,
prompts on real conflicts, and replaces with symlinks or
config-redirect. Specifically handles:

- Paperboy-shape state: drifted copies in `.claude/` and `.opencode/`.
- CLAUDE.md + AGENTS.md side-by-side with overlapping content.
- Orphaned harness directories whose target was removed.
- Custom user content inside harness dirs (preserved or surfaced for
  decision).

Includes `--dry-run` and verbose reporting.

**User-visible win:** running `hero install --migrate` on example codebase
produces a clean repo without losing anything.

### Phase 4 — `hero verify-install` + rendered-copy drift detection
*(spec: TBD — pairs with Phase 2's rendered-copy fallback)*

A standalone command that audits the install:

- Identifies harness dirs that should be symlinks but are copies (drift
  risk).
- Detects content drift between rendered copies and canonical.
- Validates harness configs point at the right paths.
- Detects broken/dangling symlinks.
- Detects symlinks that escape the workspace (security).

Runs on `hero check`, on CI, and on demand. Required for rendered-copy
mode to be safe.

**User-visible win:** Windows / Cline users get the same correctness
guarantees as symlink users, via active drift detection.

## The Hero-Native Harness Contract

The single-source-install architecture is also the **public contract**
for any Hero-native harness — an editor, IDE plugin, native client, or
future tool that wants to leverage Hero content without setup
ceremony. The contract is deliberately asymmetric: **Hero produces;
harnesses consume. Hero never gains harness-specific knowledge.**

### Consumer-side (any Hero-native harness, e.g. a Hero-native client)

A Hero-native harness must:

1. **Discover Hero by the presence of `.hero/`** at the workspace
   root. No registration, no install flag, no marker file. If
   `.hero/` exists, Hero is in use.

2. **Read canonical content directly:**
   - `.hero/agents/` for agent definitions
   - `.hero/commands/` for command definitions
   - `.hero/skills/` for skill definitions (SKILL.md format)
   - `AGENTS.md` at workspace root for primary instructions
   - `.hero/specs/`, `.hero/knowledge/`, `.hero/planning/` for
     workflow context (read access; never write)

3. **Prefer root `AGENTS.md`** as the canonical instruction file.
   Fall back to legacy locations only when AGENTS.md is absent:
   ```
   1. AGENTS.md            (root — canonical under P1)
   2. .hero/memory.md      (user override, still respected)
   3. CLAUDE.md            (user-authored supplementary; read-only)
   4. .hero/AGENTS.md      (legacy)
   5. .hero/CLAUDE.md      (legacy)
   ```

4. **Treat user-authored harness files as supplementary, never
   primary.** A user-authored `CLAUDE.md` exists because the user
   wanted Claude-specific guidance — a Hero-native harness reads it
   for context but does not replace its AGENTS.md primacy with it,
   and never writes to it.

5. **Provide a self-contained embedded baseline** so a fresh project
   without `.hero/` still works. a Hero-native client does this today via
   `build.rs` compiling Hero content into the binary. The presence
   of `.hero/` overlays project-local customization on top of the
   baseline.

6. **Store its own overrides in its own location**, never in
   `.hero/`. Settings to disable a skill, override an agent, or
   customize a command live in the harness's own settings store
   (e.g., `~/.config/a Hero-native client/`, a settings UI, etc.). Hero's
   canonical tree stays canonical; per-harness preferences are the
   consumer's concern.

7. **Never write to canonical Hero content.** Reading is fine;
   modification is not. If the user wants to change an agent
   permanently, they edit `.hero/agents/<name>.md` directly — the
   harness exposes this via its own UI but writes through to the
   canonical file, not to a private mirror.

### Producer-side (Hero)

Hero must:

1. **Maintain the canonical tree** as the single source of truth
   for agents, commands, and skills.

2. **Never gain harness-specific awareness in core logic.** Hero
   does not need a `hero install --target a Hero-native client` (or any other
   Hero-native harness target). Hero produces; the harness reads.
   No filesystem operations are needed for a Hero-native harness to
   work — it Just Works when `.hero/` is present.

3. **Respect user content absolutely.** AGENTS.md outside the
   managed region; any user-authored harness file (CLAUDE.md,
   .cursorrules, etc.); any user-customized canonical content
   (`.hero/agents/<custom>.md`) — all inviolate.

4. **Be agnostic about consumers.** The canonical tree is described
   by its own conventions, not by what each consumer needs. Hero's
   internal contracts (filenames, frontmatter shapes, directory
   structure) form the public surface; consumers depend on those,
   not the other way around.

### Why this asymmetry matters

If Hero gained per-consumer awareness (a `a Hero-native client` target, a
`zed` target, etc.), each new consumer would require Hero changes
to enable. The floor would never rise consistently — new harnesses
would lag Hero's awareness of them.

The asymmetric contract means new Hero-native harnesses can be
built and shipped by *anyone* without touching Hero's codebase.
A future "Hero plugin for VS Code" or "hero-cli-tui" or "hero-cody"
just reads canonical paths and reports `.hero/`-presence to its
users. Hero stays small; the ecosystem grows independently.

### What this means for a Hero-native client specifically

- **No `hero install --target a Hero-native client` exists or is needed.**
  a Hero-native client reads `.hero/` directly when it's present.
- **a Hero-native client's own alignment work** (priority-order update,
  switching user-content discovery from `.ai/`/`.agents/` to
  `.hero/{agents,commands,skills}/`) is a client-repo task,
  tracked in `../a Hero-native client/.hero/planning/` rather than here. The
  contract above is Hero's side; a Hero-native client's adherence is its own
  responsibility.

## What changes in `hero install` and `hero upgrade`

The install/upgrade flow is the load-bearing component of this
initiative. Both commands gain a layout-mode resolver that:

1. **Detects** what's already on disk (existing canonical, existing
   harness copies, existing instruction files).
2. **Resolves** the right install mode per harness target based on
   harness capability + filesystem capability (symlinks available?
   Windows? Developer Mode? Cline target?).
3. **Plans** the operations: write canonical, create shims, update
   configs, leave alone what's already correct.
4. **Executes** idempotently — re-running produces no changes if state
   is already correct.
5. **Reports** what was done in human-readable form.

`hero upgrade` reuses the same resolver — on a version bump, it
regenerates the managed block in `AGENTS.md` and refreshes canonical
content. It does **not** touch user content under any circumstance.

## Acceptance Criteria (initiative-level)

These are the must-be-true-at-completion conditions for the whole
initiative. Phase specs decompose into testable EARS criteria.

- THE SYSTEM SHALL store agent/command/skill content in exactly one
  filesystem location per project (`.hero/agents/`, `.hero/commands/`,
  `.hero/skills/`) regardless of how many harnesses are installed
- THE SYSTEM SHALL write, per installed target, that target's native
  root instruction file (claude → `CLAUDE.md`; all other targets →
  `AGENTS.md`), each with a Hero-managed region and user-owned regions
  clearly delimited — and SHALL NOT create a root instruction file for
  a harness that was never an install target *(amended — see banner;
  detailed in `harness-native-install-target-aware-upgrade`)*
- WHEN multiple harnesses are installed in one project THE SYSTEM
  SHALL NOT create more than one physical copy of any agent, command,
  or skill file
- WHEN a user edits content outside the Hero-managed region of
  `AGENTS.md` THE SYSTEM SHALL preserve those edits verbatim through
  every install and upgrade
- WHEN `hero install --migrate` runs against a legacy multi-harness
  install with drifted copies THE SYSTEM SHALL consolidate into the
  canonical layout without losing any content, prompting on real
  conflicts
- WHEN a harness target is added with `hero install --target X` THE
  SYSTEM SHALL create the appropriate harness-side shim
  (config-redirect, symlink, or rendered copy) without modifying any
  other harness's setup
- WHEN a harness target is removed THE SYSTEM SHALL remove only that
  target's shims, leaving canonical content and other harness shims
  untouched
- IF the host filesystem does not support symlinks (Windows without
  Developer Mode) OR the harness is known not to follow symlinks
  (Cline) THEN THE SYSTEM SHALL fall back to rendered copies and
  enable drift detection via `hero verify-install`
- THE SYSTEM SHALL be idempotent on re-run: `hero install` against
  an already-correct project produces zero filesystem changes

## Boundaries

- **Not in scope:** changing the *content* of any agent, command, or
  skill — this is a packaging/install change, not a redesign of
  what's installed.
- **Not in scope:** authoring or distributing new harnesses. Hero
  supports what the harness vendors ship; if a harness changes its
  config schema, Hero adapts.
- **Not in scope:** the `.hero/specs/` and `.hero/knowledge/` content
  — those are workspace-internal, not subject to harness reads.
- **Not in scope:** Windows native installs without WSL or Developer
  Mode get rendered-copy mode automatically; this initiative does not
  introduce Windows-specific path handling beyond the symlink-vs-copy
  decision.
- **Not in scope:** existing CLAUDE.md content that isn't recognized
  as Hero-managed (no marker) — Phase 3 migration prompts the user
  on what to do with it; doesn't merge automatically.
- Phases 1 and 2 ship before Phase 3 migration is fully built; until
  Phase 3 lands, manual migration is documented but not automated.

## Open questions

- **Claude Code AGENTS.md support.** Anthropic issue #6235 is open
  since Aug 2025. If native support lands during this initiative's
  delivery, the CLAUDE.md shim becomes unnecessary. Phase 1 design
  must accommodate both states.
- **Cursor `.cursor/rules/*.mdc` extension.** If `.md` symlinks
  aren't accepted by Cursor (extension mismatch), we may need to
  render `.mdc` copies for Cursor specifically even when symlinks
  are otherwise available. Verify empirically during Phase 2.
- **Plugin systems.** Claude Code 2.x has `.claude-plugin/`. If a
  project uses both a plugin install and a direct install, Hero's
  symlinks need to coexist. Verify Phase 2.

## References

- Survey: `.hero/knowledge/harness-instruction-file-survey.md`
- Existing satellite implementation: `internal/install/satellite.go`
  (already implements the symlink pattern for sub-roots; this
  initiative extends it to project roots)
- Existing target layouts: `internal/install/satellite.go:57`
  (`targetLayouts` registry)
- Existing managed-marker pattern: `internal/install/mcp.go:247`
  (`# hero:managed` / `# end:hero:managed` markers in TOML)
