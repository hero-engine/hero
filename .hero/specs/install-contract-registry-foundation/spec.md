---
title: Install Contract Registry Foundation — Per-Target Validators for Installed Output
type: feature
status: completed
priority: P0
completed: 2026-05-12
severity: high
created: 2026-05-12
tags: [install, testing, contracts, foundation, claude, harness]
relations:
  - target: install-upgrade-contract-coverage
    kind: child
  - target: claude-subagent-frontmatter-registration
    kind: builds-on
---

# Install Contract Registry Foundation — Per-Target Validators for Installed Output

## Goal

Introduce a per-target contract registry so each install target
declares the shape of installed file each content kind must
produce, and the install harness can prove the declared contract
holds against the rendered output. Land Claude × commands and
Claude × skills as the first two consumers, and prove the
primitive works on the highest-stakes target.

After this spec lands, every (target, kind) cell that has a
declared contract is provably enforced on every install. The other
five children of the
[install-upgrade-contract-coverage](../../initiatives/install-upgrade-contract-coverage/spec.md)
initiative all build on the primitive added here.

## Kickoff

Foundation child of `install-upgrade-contract-coverage` — landed.
Adds `HarnessContract` + `mustSatisfyContract(target, kind)` and
lands Claude × {agents, commands, skills} as the first three
consumers.

**Status:** completed — full suite green, vet clean, manual install
smoke confirms 34 agents / 27 commands / 45 skills all satisfy
their contracts in a fresh install.

**Pick up at:** confirm commits look right and push. Then design
the next children of the parent initiative —
`install-smoke-coverage-codex-copilot-generic` (#2) and
`install-contract-coverage-opencode-cursor` (#3) are unblocked
and parallelizable.

**What landed:**
- [internal/install/contracts.go](internal/install/contracts.go) — `HarnessContract`, `ContentKind`, `targetContracts` registry, `ContractsFor` lookup.
- [internal/install/harness_test.go](internal/install/harness_test.go) — `mustSatisfyContract`, `assertContract`, `harnessDirFor`, `harnessExtractFrontmatter`, `harnessFrontmatterValue`. Removed legacy `mustBeRegisterableSubagent`.
- [internal/install/contracts_test.go](internal/install/contracts_test.go) — `TestEveryInstalledKindHasContract` meta-test with named-skip messages pointing to children #2/#3.
- [internal/install/harness_smoke_test.go](internal/install/harness_smoke_test.go) — `TestHarness_SmokeClaude` now exercises all 3 contract cells.
- 4 skills got `description:` added (html-mockup-generation × 2, root-cause-classification × 2 in root + engineering domain).
- [content_test.go](content_test.go) — added `TestEmbeddedSkills_HaveRequiredFrontmatter` to enforce description at canonical-source layer.

**Validation done:**
- `go test ./...` — full suite passes.
- `go vet ./...` — clean.
- `go test ./internal/install/... -run TestEveryInstalledKindHasContract -v` — 3 PASS for claude, 15 SKIP for the cells children #2/#3 will close.
- Manual: fresh `hero init && hero install project . --target claude`. 34 agents (all with name+description), 27 commands (all with description), 45 skills (all with description). `.claude/agents` symlinked to `../.hero/agents` — single-source architecture preserved.

→ `hero spec complete .hero/planning/features/install-contract-registry-foundation/spec.md`

**Why this matters:** the primitive that closes 17 latent install
bug cells. The next contributor adding a target/kind cell can
either declare a contract or get a loud test failure naming the
gap — no more silent degradation.

**Skip:** opencode/cursor/codex/copilot/generic contracts (children
#2/#3); upgrade-path coverage (#4); CI gate policy (#5);
real-harness validation (#6).

## Problem

The `claude-subagent-frontmatter-registration` fix proved a class
of latent install bug: files land at the destination path, the
file-existence test passes, but the consuming harness silently
rejects them because the YAML frontmatter doesn't satisfy the
harness's contract. That class of bug is structurally invisible to
the existing test suite because the assertion vocabulary
(`mustBeRegularFile`, `mustBeSymlink`, `mustContain`,
`mustHaveSameContent`) is file-existence-shaped, not
content-contract-shaped.

The fix today closed one cell of an 18-cell coverage matrix
(6 targets × 3 content kinds). Closing the remaining 17 cells
one-spec-at-a-time is the wrong shape — it's whack-a-mole and the
next contributor adding a new target or content kind has no
scaffolding to plug into. The only sustainable fix is a primitive
that makes contract enforcement easy to declare and impossible to
forget.

### What investigation found

**Claude Code subagents** (`.claude/agents/<name>.md`) require
`name:` and `description:` to register. Without them the file is
silently dropped from the Task tool's `subagent_type` list. (Source:
https://code.claude.com/docs/en/subagents.md.) This contract is
already enforced canonical-source-side by `content_test.go` and
install-side by today's `mustBeRegisterableSubagent` helper.

**Claude Code slash commands** (`.claude/commands/<name>.md`) and
**skills** (`.claude/skills/<name>/SKILL.md`) have NO required
frontmatter per Claude Code itself — both fall back to defaults
(filename → name, first paragraph → description). However, this
spec adopts a **stricter Hero contract**: `description:` is
required for both. Reasons:
- `description:` is load-bearing for model-driven discovery — without
  it, Claude doesn't know when to invoke the slash command or skill.
- Hero authoring standard already populates `description:` on every
  command and almost every skill (audit below).
- Enforcing the field at install time prevents a future Hero
  contributor from adding a discovery-broken command/skill that
  technically loads but won't actually be invoked.

Audit of current canonical content:
- `commands/*.md` (56 files across all roots): **100% have
  `description:`**.
- `skills/<name>/SKILL.md`: **4 files missing `description:`**:
  - `skills/html-mockup-generation/SKILL.md`
  - `skills/root-cause-classification/SKILL.md`
  - `domains/engineering/skills/html-mockup-generation/SKILL.md`
  - `domains/engineering/skills/root-cause-classification/SKILL.md`

These four skills get a one-line `description:` added as part of
this spec — they're the first thing the new contract test would
flag, and fixing them is the proof point that the contract works.

**Hooks/permissions/MCP config files** (claude_hooks.json mutations,
opencode.json MCP block, codex_hooks.go output) stay covered by
their existing per-feature unit tests for now. They have a
fundamentally different contract shape (JSON with merge semantics,
not standalone files) and pulling them into the registry creates
more complexity than it pays back in #1. Revisit in
`install-upgrade-path-coverage` (child #4).

## Goal (refined)

A `HarnessContract` type and per-target registry exist in
production code (not test-only). Each target declares contracts
for the content kinds it installs. A new
`(*installHarness).mustSatisfyContract(target, kind)` test
primitive walks the installed destination dir and validates every
file against the declared contract. `TestHarness_SmokeClaude`
exercises the primitive for agents, commands, and skills.
Targets that install a kind without declaring a contract fail a
meta-test loudly.

## Design

### Type definitions (production code)

New file [internal/install/contracts.go](internal/install/contracts.go):

```go
package install

// ContentKind names a category of canonical content the installer
// renders into a target. Each target installs zero or more kinds
// and must declare a HarnessContract for every kind it installs.
type ContentKind string

const (
    KindAgents   ContentKind = "agents"
    KindCommands ContentKind = "commands"
    KindSkills   ContentKind = "skills"
)

// HarnessContract describes the shape an installed file must take
// for the consuming harness to register and use it. Each
// (Target, ContentKind) cell declares its contract; the install
// harness enforces it post-render via mustSatisfyContract.
//
// Start minimal — fields here exist because at least one declared
// contract uses them. Expand only when a consumer needs a new
// validator shape.
type HarnessContract struct {
    // RequiredFrontmatter lists YAML keys that must be present and
    // non-empty in every file's leading `---\n...\n---\n` block.
    // Empty slice means no frontmatter requirements.
    RequiredFrontmatter []string

    // FilenameRequired, when set, must match the destination
    // file's basename. For nested kinds (skills land at
    // <name>/SKILL.md), match against the trailing path segment
    // ("SKILL.md"). Empty string disables the check.
    FilenameRequired string

    // ContentValidator, when set, runs against the file's full
    // bytes after frontmatter checks pass. Used for whole-file
    // validators (e.g. JSON schema for opencode.json in a future
    // child spec). Nil disables.
    ContentValidator func([]byte) error
}

// targetContracts is the per-target registry. Lookups go through
// ContractsFor — never index directly so the meta-test can detect
// missing cells.
var targetContracts = map[Target]map[ContentKind]HarnessContract{
    TargetClaude: {
        KindAgents: {
            RequiredFrontmatter: []string{"name", "description"},
        },
        KindCommands: {
            RequiredFrontmatter: []string{"description"},
        },
        KindSkills: {
            RequiredFrontmatter: []string{"description"},
            FilenameRequired:    "SKILL.md",
        },
    },
    // Other targets land in children #2 (codex/copilot/generic)
    // and #3 (opencode/cursor) of install-upgrade-contract-coverage.
    // Empty maps here so ContractsFor returns "no contract
    // declared" rather than "target unknown".
    TargetOpenCode: {},
    TargetCursor:   {},
    TargetCodex:    {},
    TargetCopilot:  {},
    TargetGeneric:  {},
}

// ContractsFor returns the contract for the (target, kind) cell
// or (zero, false) if none is declared. Test harnesses use the
// boolean to distinguish "no contract declared yet" from "contract
// declared with zero requirements".
func ContractsFor(target Target, kind ContentKind) (HarnessContract, bool) {
    kinds, ok := targetContracts[target]
    if !ok {
        return HarnessContract{}, false
    }
    c, ok := kinds[kind]
    return c, ok
}
```

Rationale for production code (vs test-only): the contracts ARE
the per-target install spec. Keeping them adjacent to
`target_*.go` (same package) makes them discoverable to anyone
adding a new target — they see "I need a `Contracts` declaration
too." Also lets future runtime install code consume them
(e.g. `hero install --verify` could run them at install time).

### Harness primitive (test code)

Extend [internal/install/harness_test.go](internal/install/harness_test.go):

```go
// mustSatisfyContract walks the installed destination directory
// for the given (target, kind) and asserts every file matches the
// declared HarnessContract. Reads bytes off the destination so it
// works for both symlinked and rendered installs.
//
// Fails the test if no contract is declared for the cell — every
// (target, kind) the target installs must have a contract.
func (h *installHarness) mustSatisfyContract(target Target, kind ContentKind) {
    h.t.Helper()
    contract, ok := ContractsFor(target, kind)
    if !ok {
        h.t.Fatalf("no HarnessContract declared for (%s, %s) — add one to internal/install/contracts.go", target, kind)
    }
    dir := h.harnessDirFor(target, kind) // see below
    entries, err := os.ReadDir(dir)
    if err != nil {
        h.t.Fatalf("read installed %s/%s dir %s: %v", target, kind, dir, err)
    }
    found := 0
    for _, e := range entries {
        relPath := filepath.Join(dir, e.Name())
        if kind == KindSkills {
            // Nested layout: <name>/SKILL.md
            if !e.IsDir() { continue }
            relPath = filepath.Join(relPath, "SKILL.md")
            if _, err := os.Stat(relPath); err != nil { continue }
        } else {
            if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") { continue }
        }
        h.assertContract(relPath, contract)
        found++
    }
    if found == 0 {
        h.t.Fatalf("(%s, %s): expected at least one file at %s, found none", target, kind, dir)
    }
}

func (h *installHarness) assertContract(absPath string, c HarnessContract) {
    h.t.Helper()
    data, err := os.ReadFile(absPath)
    if err != nil {
        h.t.Fatalf("read %s: %v", absPath, err)
    }
    if c.FilenameRequired != "" && filepath.Base(absPath) != c.FilenameRequired {
        h.t.Fatalf("%s: filename %q does not match required %q", absPath, filepath.Base(absPath), c.FilenameRequired)
    }
    if len(c.RequiredFrontmatter) > 0 {
        fm, ok := extractFrontmatter(data)
        if !ok {
            h.t.Fatalf("%s: missing or malformed YAML frontmatter (required keys: %v)", absPath, c.RequiredFrontmatter)
        }
        for _, key := range c.RequiredFrontmatter {
            val, present := frontmatterValue(fm, key)
            if !present {
                h.t.Fatalf("%s: missing required frontmatter key `%s:` (required by %s contract)", absPath, key, "/* contract owner */")
            }
            if strings.TrimSpace(val) == "" {
                h.t.Fatalf("%s: frontmatter key `%s:` is present but empty", absPath, key)
            }
        }
    }
    if c.ContentValidator != nil {
        if err := c.ContentValidator(data); err != nil {
            h.t.Fatalf("%s: content validator failed: %v", absPath, err)
        }
    }
}
```

`harnessDirFor(target, kind)` is a small switch from
`(target, kind)` → installed destination subdir. For Claude:
`(TargetClaude, KindAgents)` → `.claude/agents`,
`(TargetClaude, KindCommands)` → `.claude/commands`,
`(TargetClaude, KindSkills)` → `.claude/skills`. For other
targets, use the same destBase logic the per-target installer
uses (mirror the `runX` switch in `target_*.go`).

`extractFrontmatter` and `frontmatterValue` reuse the logic
already in [content_test.go](content_test.go) `splitFrontmatter`
plus a tiny YAML key probe. Promote that helper from
`content_test.go` to a small shared file in the root package
(`frontmatter.go`) so both layers can use it without duplication.
Alternative: keep two copies and accept the small duplication —
pick during delivery based on whether sharing creates an awkward
import. Default: shared helper.

### Meta-test: every installed kind has a contract

```go
// TestEveryInstalledKindHasContract asserts that for every
// (target, kind) cell where the target actually renders files,
// a contract is declared. Catches the "added a target, forgot
// the contract" regression.
func TestEveryInstalledKindHasContract(t *testing.T) {
    for _, target := range []Target{TargetClaude, TargetOpenCode, TargetCursor, TargetCodex, TargetCopilot, TargetGeneric} {
        h := newInstallHarness(t)
        h.Run(target, nil)
        for _, kind := range []ContentKind{KindAgents, KindCommands, KindSkills} {
            dir := h.harnessDirFor(target, kind)
            if !dirHasFiles(dir) {
                continue // target doesn't install this kind
            }
            if _, ok := ContractsFor(target, kind); !ok {
                t.Errorf("target %s installs %s but has no HarnessContract — add one to internal/install/contracts.go", target, kind)
            }
        }
    }
}
```

This will currently fail for cells the installer fills but no
contract is declared (e.g. `(TargetCursor, KindAgents)` after
this spec lands). That's intentional — the test fails are the
work list for children #2 and #3. Add a `t.Skip` shim with a
TODO referencing the child specs so the failure is loud-but-not-blocking
until those children land. Or accept the failures and document
them as expected. **Decision for delivery:** use `t.Skip` with
explicit message naming the responsible child spec, so CI stays
green and the gap is documented, not silently tolerated.

### Wire first consumers into TestHarness_SmokeClaude

Extend [internal/install/harness_smoke_test.go](internal/install/harness_smoke_test.go)
`TestHarness_SmokeClaude`:

```go
h.mustSatisfyContract(TargetClaude, KindAgents)
h.mustSatisfyContract(TargetClaude, KindCommands)
h.mustSatisfyContract(TargetClaude, KindSkills)
```

Replace the existing `mustBeRegisterableSubagent(...)` calls — the
new primitive subsumes them and is broader. Remove the now-redundant
helper (or keep as a thin wrapper if other tests reference it —
grep first).

### Seed source must satisfy contracts

The current `seedSource` in `harness_test.go` writes minimal
markdown for commands and skills with no frontmatter. The new
contracts will reject those. Update seed to include realistic
frontmatter:

```go
"commands/design.md": "---\ndescription: Produces a spec.\n---\n# /design command\n",
"commands/deliver.md": "---\ndescription: Implements a spec.\n---\n# /deliver command\n",
"skills/spec-format/SKILL.md": "---\ndescription: Defines spec structure.\n---\n# spec-format skill\n",
"skills/test-strategy/SKILL.md": "---\ndescription: Test pyramid guidance.\n---\n# test-strategy skill\n",
```

(Skill seed currently lands as flat `skills/spec-format.md` —
update to nested `<name>/SKILL.md` to match canonical layout.
Existing assertion `h.mustBeRegularFile(".claude/skills/spec-format/SKILL.md")`
already expects the nested shape, so seed needs to match.)

### Fix four canonical skills missing `description:`

Append `description: <one line>` to:
- `skills/html-mockup-generation/SKILL.md`
- `skills/root-cause-classification/SKILL.md`
- `domains/engineering/skills/html-mockup-generation/SKILL.md`
- `domains/engineering/skills/root-cause-classification/SKILL.md`

Read each file first to write a description that matches what the
skill actually does. Write same body for the duplicated pair
(domains/engineering/* mirrors root skills/*).

### Strengthen content_test.go to match

The new install-side contract is stricter than today's
`content_test.go` for skills (description required). Update
`content_test.go` to enforce `description:` on every embedded
skill file the same way it enforces `name:`+`description:` on
agents. Catches the gap at canonical-source layer too.

## Boundaries

- Do NOT extend the registry to opencode/cursor/codex/copilot/generic
  contracts beyond empty placeholder maps. Children #2 and #3
  fill those.
- Do NOT pull hooks/permissions/MCP wiring into the contract
  registry. Different shape (JSON merge semantics); covered by
  per-feature tests today; revisit in child #4.
- Do NOT add a `ContentValidator` for any cell yet. Field exists
  for opencode.json schema in child #3; no consumer in this spec.
- Do NOT introduce per-harness frontmatter translation. Contracts
  validate; they never transform.
- Do NOT touch `linkOrRenderDir`. The bug class is at the
  validate layer, not the link/render layer.
- Do NOT remove `content_test.go` — it stays as the
  canonical-source layer; the new install-side primitive is the
  rendered-output layer.
- Do NOT introduce a generic "validate every YAML field" framework.
  Start with `RequiredFrontmatter []string`; only add fields when
  a real consumer needs more.
- Do NOT make slash command frontmatter strictly required by
  Claude Code's standards (it isn't — see Investigation). Hero's
  contract is intentionally stricter than the harness minimum
  for discovery reasons.

## Risks

- **Hidden bug surface in commands/skills.** All 56 commands
  already have `description:`, but only 4 skills are missing it.
  After this lands, those 4 fail the test until described. Already
  in the change set.
- **Seed-source frontmatter changes break other harness tests.**
  Mitigation: run full `internal/install/...` test suite before
  landing. Likely only `TestHarness_Smoke<Target>` cares about
  seed shape.
- **Skip shims for empty contract maps create silent gaps.**
  Mitigation: the skip message names the responsible child spec
  so it's a loud TODO, not a buried one. When child #2 / #3 land,
  the skip removes itself by way of contracts populating.
- **Frontmatter parser disagreement.** YAML-quoted values, trailing
  whitespace, Windows line endings. Mitigation: keep the parser
  as a tiny line-based key probe (no full YAML parse), match
  prefix `<key>:` and trim — same approach `content_test.go`
  uses today, no regressions. Only escalate to `yaml.Unmarshal`
  if a real test case requires it.
- **`harnessDirFor` divergence from per-target installer paths.**
  If the installer changes destination paths, `harnessDirFor` must
  follow. Mitigation: write `harnessDirFor` as a small switch that
  mirrors the per-target installer, with a comment pointing to
  each `target_*.go` file. Future audits can grep for the comment.
- **Test runtime growth.** Three additional `mustSatisfyContract`
  calls per target × six targets = ~18 extra ReadDir + per-file
  parses. Negligible — order of microseconds at current content
  size.

## Validation

- `go test ./internal/install/...` passes, including the new
  `TestEveryInstalledKindHasContract` and the extended
  `TestHarness_SmokeClaude`.
- `go test .` (root pkg) passes — `content_test.go` strengthened
  for skills.
- `go test ./...` full suite passes.
- `go vet ./...` clean.
- Manual smoke: in a throwaway dir, `hero init && hero install
  project . --target claude`. Inspect `.claude/agents/`,
  `.claude/commands/`, `.claude/skills/` — every file has the
  contract-required frontmatter.
- Multi-harness regression guard: install Claude then OpenCode
  in same dir, both succeed without `--force`. (Inherited from
  `multi-harness-install-collision`; verify still green.)

## Acceptance Criteria

- WHEN `mustSatisfyContract(target, kind)` is called for a cell
  with a declared HarnessContract THE SYSTEM SHALL walk the
  installed destination dir for that cell, parse each file's
  frontmatter, and assert every required key is present and
  non-empty.
- WHEN `mustSatisfyContract` is called for a cell with no
  declared HarnessContract THE SYSTEM SHALL fail the test with
  an actionable message naming the (target, kind) and pointing to
  `internal/install/contracts.go`.
- WHEN `TestEveryInstalledKindHasContract` runs THE SYSTEM SHALL
  identify every (target, kind) cell where the installer produced
  files and either find a declared contract OR record a `t.Skip`
  with the responsible child spec name.
- WHEN `TestHarness_SmokeClaude` runs THE SYSTEM SHALL exercise
  the primitive for agents, commands, and skills, and SHALL pass
  given current canonical content (after the 4-skill description
  fix).
- WHEN a Hero contributor adds a new install target THE SYSTEM
  SHALL fail tests until that target declares a HarnessContract
  for every content kind it installs.
- WHEN a Hero contributor adds a slash command without
  `description:` THE SYSTEM SHALL fail
  `TestHarness_SmokeClaude`'s commands contract assertion.
- WHEN a Hero contributor adds a skill without `description:` or
  not located at `<name>/SKILL.md` THE SYSTEM SHALL fail
  `TestHarness_SmokeClaude`'s skills contract assertion.
- IF a target installs a content kind without declaring a
  contract THEN `TestEveryInstalledKindHasContract` SHALL surface
  the gap (failure or skip-with-TODO) before release.
- THE SYSTEM SHALL preserve the single-source canonical-symlink
  install architecture — `mustSatisfyContract` reads the
  destination file's bytes regardless of whether they arrived via
  symlink or rendered copy.
- THE SYSTEM SHALL keep `content_test.go` as the canonical-source
  contract layer; the new install-side primitive is additive, not
  replacement.
- THE SYSTEM SHALL NOT introduce per-harness frontmatter
  translation. Contracts are validators only.
