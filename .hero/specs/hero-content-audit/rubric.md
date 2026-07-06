# Audit rubric — hero-content-audit

Applied to every file in [inventory.md](inventory.md). Score each dimension
`ok` / `flag` / `n/a`; every `flag` must carry a finding with file path,
evidence, and (for verbosity) what to cut. Findings-only — no edits to
`core/` or `domains/`.

## Dimensions

1. **Earns its place.** Does this agent/command/skill do a distinct job, or
   does it overlap another roster entry? Name the overlapping entry. For
   agents: could a session pick between the two from their descriptions alone?
2. **Token efficiency.** Signal density. Flag a file when the same guidance
   could land in materially fewer words, or when boilerplate repeats across
   files and belongs in one shared skill. Every verbosity flag names the
   section(s) to cut or merge — "too long" alone is not a finding.
3. **Actionability.** Can a cold agent act on this? Concrete file paths,
   commands, decision rules, and examples beat aspirational prose ("be
   thorough", "ensure quality"). Flag sections that state values without
   telling the agent what to *do*.
4. **Freshness.** References to commands, CLI subcommands, flags, file paths,
   skills, or agents that don't exist (or were renamed). Verify the target
   exists before flagging; cite both the referencing line and the missing
   target.
5. **Triggering.** Frontmatter `description` quality: would the right session
   load this at the right time? Descriptions that are circular ("skill for X"
   where X is the name), overbroad, or missing trigger cues get flagged.
6. **Harness-agnosticism.** Per the `harness-changes-cover-all-targets`
   tripwire: Hero installs to six targets (`opencode | cursor | claude |
   copilot | codex | generic`). Flag Claude-only assumptions (CLAUDE.md
   references, Claude-specific hooks/tools presented as the mechanism rather
   than an enhancement), and any harness-specific guidance not explicitly
   scoped.
7. **Format consistency.** Frontmatter schema and section structure consistent
   within the surface. Known variants to check against (from inventory):
   - skills: `name,description,compatibility,metadata` (80/94) — outliers lack
     `compatibility` or `metadata`; also judge whether `compatibility:
     opencode` is correct at all given six install targets.
   - agents: four schema variants (`role` and `domains` present in some,
     absent in others) — determine which is canonical.
   - commands: `description` only (70/72).

## Word-count reference bands (from the current distribution)

| Surface | n | p50 | p90 | max | Verbosity review threshold |
|---------|---|-----|-----|-----|---------------------------|
| agent | 58 | 388 | 1,307 | 2,685 | > ~1,300 words (p90) |
| command | 72 | 239 | 529 | 2,673 | > ~530 words (p90) |
| skill | 94 | 994 | 1,721 | 2,482 | > ~1,700 words (p90) |
| routing | 3 | 1,954 | — | 2,247 | judge individually |

Above-threshold files get a mandatory token-efficiency review; length alone
is not a defect — a p95 file that is all signal passes. Below-threshold files
can still be flagged if padded.

## Finding format

```
### [SEV] <one-line defect> — <file path>
- Surface/dimension: <surface> / <dimension #>
- Evidence: <quote or line refs; for freshness, the missing target>
- Fix shape: <one line — what remediation looks like>
```

Severity: `S1` (actively misleading agents — wrong/stale instructions),
`S2` (material waste or drift — duplication, heavy padding, roster overlap),
`S3` (polish — inconsistency, weak descriptions). Rank by severity ×
blast-radius (how many sessions/harnesses load the file).

## Classification notes

- Per-domain `README.md` files are documentation, not installed content —
  audit only for freshness (dead references).
- `AGENTS.md` routing files: audit claims against pack contents (every
  listed agent/command/skill must exist; nothing shipped goes unlisted).
