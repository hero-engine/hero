---
type: convention
status: draft
scope: [".gitignore", "internal/install/**", "internal/cli/init.go"]
tags: [install, harness, gitignore, codex, distribution]
---

# Harness Install Artifacts: Engine Repo vs. User Projects

## Pattern

Harness directories (`.claude/`, `.agents/`, `.opencode/`, `.cursor/`,
`.codex/`, `.hero/{agents,commands,skills}`) are **committed in user
projects** and **gitignored in this engine repo**. These are opposite
rules for the same paths. Getting them backwards is easy — the
rationale below is what distinguishes them.

### User projects — artifacts TRAVEL

`hero init` splices `managedGitignoreEntries` (internal/cli/init.go)
into the project's `.gitignore`. That list contains **zero harness
dirs**. It ignores only machine-local or regenerable state:

    hero.local.json, graph.db, index.db, next/*.local.md,
    knowledge/code/, satellites.local.json, cache/, sessions/,
    install-state.json

This is deliberate. A teammate cloning a user project gets working
skills and commands **without running `hero install`**. The rendered
harness content is the only copy that exists there — nothing else in
the repo can regenerate it.

### This engine repo — artifacts are IGNORED

Hero's own repo **embeds its source for distribution**:

    //go:embed core/agents core/commands core/skills
    //go:embed domains/<x>/agents domains/<x>/commands domains/<x>/skills

The canonical source lives at `core/` and `domains/*/` and is
committed. Anything rendered into a harness dir is a **duplicate** of
that committed source, so committing it stores the same bytes twice
under two names — and the copy nothing regenerates on commit is the one
that rots. Run `make install` after cloning to materialize it.

## Rule

> **Does the repo already contain the source that regenerates this
> directory?** If yes (engine repo) → gitignore it. If no (user
> project) → commit it.

## Why `.agents/skills` is on the ignore list

Codex loads repo-scoped skills from `.agents/skills/<name>/SKILL.md`
(the `.codex/skills/` path is deprecated; `target_codex.go` only cleans
it up, never writes it). `codexSkillsDest` renders there on
`hero install --target codex`. The install path is correct — it is the
*tracking* that was wrong.

`.agents/skills` was committed and unmaintained for 14 months (5
commits, all chores). Measured against a clean render, **only 16 of 83
tracked files (19%) still matched**: 63 had drifted, 5 were never
added, and 4 were orphans from a superseded layout
(`source-command-*`, `command-prime`). The `spec-format` copy was 8KB
behind its source and still taught the pre-0d69f13 unlabeled
Acceptance Criteria form — so Codex sessions produced specs with zero
addressable `AC-N` criteria. A generated artifact that only refreshes
when someone remembers to re-run the installer will always drift; the
fix is to stop tracking it, not to remember harder.

## Anti-patterns

- **Committing a rendered harness dir in this repo** "so Codex works on
  a fresh clone." No other harness gets that treatment — `make install`
  only runs `--target claude`, so opencode, cursor, and codex are all
  already ignored-and-not-in-make-install. Codex users run
  `hero install project . --target codex`.
- **Adding harness dirs to `managedGitignoreEntries`.** That would
  break the user-project guarantee that teammates get skills on clone.
  The engine repo's ignore block is a narrow embedded-source exception
  and must not be generalized into `hero init`.
- **Hand-editing files under a harness dir.** They are generated. Edit
  the canonical source at `core/` or `domains/*/` and re-install.

## Known gap

`hero install --target codex --force` refreshes file **contents** but
does **not prune** skill dirs whose source was renamed or deleted.
`runCodex` calls `removeLegacyDir` for `.codex/agents` and
`.codex/commands`, but nothing prunes stale entries under
`.agents/skills`. Codex's loader walks that directory, so orphans keep
loading as live skills. The 4 orphans found here were removed by hand.
