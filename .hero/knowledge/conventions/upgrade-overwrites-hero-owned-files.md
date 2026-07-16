---
type: convention
status: draft
scope: ["internal/cli/upgrade.go", "internal/install/files.go"]
tags: [install, upgrade, checksums, harness]
---

# Upgrade Overwrites Hero's Own Files — No Checksum Matching

## Pattern

`hero upgrade` overwrites Hero's generated content (agents, commands,
skills) **unconditionally** — `install.Options.Force = true`, always,
regardless of the `--force` flag. These files are Hero's own. Nobody
edits them, the docs say re-install regenerates them, and there is
nothing to protect.

User content lives in exactly one place: the root instruction files
(`CLAUDE.md` / `AGENTS.md`). Those go through the managed-region writer
(`internal/managed`), which merges and preserves everything outside
Hero's markers **regardless of Force** — so forcing the content copy is
safe.

## The rule

> Upgrade upgrades our files. It must never fail because it "can't tell"
> whether one of our own generated files is safe to replace. It always
> is.

## Why the checksum-trust approach is wrong here (do not re-add it)

The install path has a guard (`copyFileFromFS`, `internal/install/files.go`)
that refuses to overwrite an existing file whose bytes differ from
canonical, *unless* the bytes match a checksum recorded as Hero-installed
at a prior version (`TrustedChecksums` / `isTrustedHeroInstalledFile`).
That guard was wired into `upgrade` and it broke upgrade in v0.26.1:

    error  claude: installing agents: refusing to overwrite
           .claude/agents/engineer.md (use --force to replace)

The premise is impossible to satisfy. The trust check asks "do the
on-disk bytes match a checksum a previous Hero recorded?" But:

- **Every version embeds different content.** A file installed by
  version X has X's bytes; version Y's canonical differs. On-disk will
  not match Y's canonical.
- **The recorded checksum is also stale.** `version.json`'s
  `installed_files` records whatever some earlier binary wrote; across
  arbitrary point-release jumps (users install any x.x.x, upgrade to any
  y.y.y) it reliably does not match the current on-disk file either.
- So the on-disk file matches **neither** canonical **nor** the recorded
  checksum → the guard concludes "user edited it" → upgrade refuses to
  overwrite Hero's own file → the whole target errors out.

Verified in the field: `.claude/agents/engineer.md` recorded
`sha256:fd6e48b0…` vs on-disk `sha256:830cde99…`; nobody had touched it.
The mismatch is the *normal* cross-version state, not an edit.

Trying to infer "is this our file?" from content is solving an
impossible problem to answer a question we already know: these dirs are
Hero's by definition. Ownership is structural, not inferred.

## Guard against regression

`TestUpgradeOverwritesDriftedHeroFiles` (internal/cli/upgrade_test.go)
plants a Hero file whose bytes match neither canonical nor the recorded
checksum and asserts upgrade overwrites it with no error and no
`--force`. It reproduces the exact v0.26.1 failure against the old
`Force: upgradeForce` and passes against `Force: true`. If someone
re-introduces checksum-gating into the upgrade path, it fails.

## Anti-pattern

- **Gating overwrite of `agents/commands/skills` behind any per-file
  content check during upgrade.** They are regenerated copies; replace
  them. The only thing that must be preserved is user prose outside the
  managed markers in the root instruction files, which the managed
  writer already handles.
- **Relying on `version.json` `installed_files` matching on-disk.** It
  drifts across versions by construction. It is fine for informational
  drift *reporting*; it must never be load-bearing for whether upgrade is
  allowed to do its job.
