# Delivery audit — harness-agnosticism-sweep

**Audited:** `git diff HEAD` (working tree against 414344c, uncommitted)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC1 — Internal Lookups names harness-exclusive tools only inside scoped examples — `domains/engineering/AGENTS.md:121-138` + `internal/install/agents_md.go:471-486` rewritten in lockstep (byte-identical body, `TestEngineeringPackBodyMatchesGoFallback` passes); rendered `opencode` install-smoke output confirms bare `hero_*` names with `Explore`/`ToolSearch`/`mcp__hero__` only inside "e.g. Claude Code's ..." asides
- [✓] AC2 — Parity table generated from pack source + Go fallback — table added to both files at the same location; regenerated via `hero install project <tmp> --target opencode` and present in the rendered `AGENTS.md`
- [✓] AC3 — `/resume`, `/blocked`, `/peer` → Both; `/roadmap-review` → slash-only — confirmed in both the diff and the rendered install output
- [✓] AC4 — `/import`/`/handoff` annotated with differing semantics — inline parenthetical present in the Both-surface cell in both files
- [✓] AC5 — Hookless-harness instruction for `next-md`/`next-handoff-emit` — `core/skills/next-md/SKILL.md:33-38`, `core/skills/next-handoff-emit/SKILL.md:42-49`; verified live: codex install produces a real `.codex/hooks.json` Stop hook running `hero next checkpoint --quiet`, cursor install's rendered `next-md.md` correctly instructs manual `hero next checkpoint`
- [✓] AC6 — `/release` skips `hero docs check` pre-flight when no hero-repo docs layout — `domains/engineering/commands/release.md:8-16`, conditional gate added, no behavior change for this repo
- [~] AC7 — "No installed content references hero-repo-only artifacts" — **not fully true as shipped.** See Audit notes below; downgraded from the ledger's DONE.
- [✓] AC8 — Harness-neutral subagent phrasing in diagnose/deliver/mock — all three lines rewritten exactly per spec wording (`diagnose.md:39`, `deliver.md:188-190`, `mock.md:85`)
- [✓] AC9 — Zero `compatibility:`/`role:` keys in pack frontmatter — `grep -rcE '^compatibility:|^role:' core domains` → `0` (re-ran myself)
- [✓] AC10 — Zero `domains:` keys in domain-pack agents; `readAgentDomainsFrontmatter` intact — `grep -rlE '^domains:' domains/*/agents` → empty; `internal/install/content.go:83,95-100` untouched and still called
- [✓] AC11 — `TestEngineeringPackBodyMatchesGoFallback` + `content_parity_test.go` (`TestDomainPacks_NoUnannotatedCoreShadows`) pass — both re-run directly, both green; full `go test ./...` also green (86 packages, 0 FAIL)

## Changes

- [✓] 1. Rewrite Internal Lookups harness-neutrally (dual-edit) — verified identical bodies via diff of both files, test passes
- [✓] 2. Parity table into pack source (dual-edit) — verified in both files; `core/commands/handoff.md:30-31` confirmed already clean (untouched in this diff — the stale `skills/next-md.md`-style links are gone, replaced by skill-name references), consistent with the ledger's claim that a prior sibling delivery already fixed it
- [✓] 3. Harness-neutral subagent phrasing (F23) — `diagnose.md:39`, `deliver.md:188-190`, `mock.md:85` all match spec wording
- [✓] 4. Scope Stop-hook machinery in next-md/next-handoff-emit — scoping paragraphs present, broken relative links (`next-md.md`, `next-handoff-emit.md`) replaced with skill-name prose
- [✓] 5. De-dogfood drive skill — `scripts/drive/stop-hook.sh` reference removed from `domains/engineering/skills/drive/SKILL.md`; text scoped to "supervisor runs the check manually between turns" rather than describing a hook that doesn't exist. Independently verified: `grep -rn "hero goal\|stop-hook\|HERO_DRIVE_INITIATIVE" internal/install/*.go` → no matches — confirms no shipped hook actually wires `hero goal --check` for claude or codex, so the ledger's claim is accurate, not a convenient excuse
- [✓] 6. De-dogfood cross-repo-peering — lines 14, 151, 161-163 rephrased to conditional (`.hero/knowledge/...` gated on "if your workspace carries...")
- [✓] 7. De-dogfood roadmap-review + spec-composition — both fixed at cited lines (`internal/sizing/ambient.go`, `roadmap-review-ambient-surfacing`, `internal/snapshot/rollup.go`, `multi-spec-design-routing` all removed); bonus fix in `domains/engineering/agents/roadmap-reviewer.md:161-164` (sibling-spec-slug pattern) confirmed present
- [✓] 8. De-dogfood pm skills — `pm-preset-detection/SKILL.md:26,36`, `handoff-protocol/SKILL.md:123`, `story-writing-invest/SKILL.md:117` all fixed as specified; bonus fix in `core/skills/completion-ledger/SKILL.md:64` (`internal/spec/ledger.go`/`internal/cli/verify.go`) confirmed present
- [✓] 9. Fix project-context-builder + roadmap-reviewer agents — `core/agents/project-context-builder.md:13,25` scoped ("future agent sessions", harness-mechanism example); `domains/engineering/agents/roadmap-reviewer.md:61-63` uses bare `hero_*`
- [✓] 10. Gate `/release` docs pre-flight (F14) — conditional wording added, matches spec
- [✓] 11. Mechanical frontmatter strip (~96 files) — re-verified independently (not just trusting the ledger's numbers): 68 files had `compatibility:` at HEAD (15 core + 34 engineering + 19 pm — exactly matches the ledger's re-verified count, which itself differs from the spec's original 67), 16 files had `role:`, 13 had `domains:`, union = 96 unique files, matching the ledger exactly. Sampled every non-dual-purpose stripped file's diff programmatically: every removed line is exactly a `compatibility:`/`role:`/`domains:` key (or, for `code-scrub/SKILL.md`, the key plus its 3 YAML-list item lines) — zero stray body deletions found across all 96 files. All 104 touched `.md` files still have a well-formed frontmatter block (opening/closing `---`).

## Open items (if any)

- **AC7 / Change 7 "No installed content references hero-repo-only artifacts"** — PARTIAL, understated as DONE. See Audit notes.

## Audit notes

**The one real problem: AC7 is marked DONE but the shipped content still violates it, and the ledger's own disclosure undersells the scale.**

AC7's exact wording is: "THE SYSTEM SHALL ship no installed skill, agent, or command that references hero-engine-repo-only artifacts (`internal/…`, `scripts/…`, **`core/spec-types/`**, `CROSS-REPO-PEERING.md`, sibling spec slugs)" — `core/spec-types/` is named explicitly, by the spec itself, as a banned pattern.

`core/spec-types/` and `domains/pm/spec-types/` are not installed to any of the six targets — confirmed by grepping `internal/install/*.go` for "spec-types" (zero hits: nothing copies that directory anywhere). So any live instructional text in installed pack content that tells the agent to "see `core/spec-types/epic.md`" is asserting a path that will not exist in the user's actual workspace — precisely the failure mode this spec exists to close.

I found 8 unfixed instances across 5 files that **are** installed pack content (confirmed via `internal/install/content.go`'s domain-agent merge logic, and confirmed none of these files appear anywhere in this delivery's diff):
- `domains/pm/AGENTS.md:29,67` (2) — `core/spec-types/epic.md`
- `domains/pm/agents/pm-reviewer.md:24-26` (3) — `core/spec-types/feature.md`, `epic.md`, `initiative.md`
- `domains/pm/agents/roadmap-curator.md:19` (1) — `core/spec-types/initiative.md`
- `domains/pm/agents/handoff-coordinator.md:31` (1) — `core/spec-types/feature.md`
- `domains/pm/agents/story-writer.md:87` (1) — `core/spec-types/`

None of these are inside fenced illustrative examples (unlike the `kickoff-prompt` exception, which I checked and is genuinely a historical-format code block, correctly left alone) — they are live "go read this file" instructions.

The ledger does disclose that it found "a broader...pattern...across the pm domain" and filed `task_50f097a2` as a follow-on rather than expanding scope. That's a defensible instinct in isolation (Change 8 only enumerated one file; discovering a systemic pattern mid-delivery is exactly the kind of thing that should be scoped out per the spec's own Boundaries precedent for other out-of-scope content-accuracy findings). But two things don't hold up:
1. The row is marked **DONE**, not PARTIAL — the criterion as literally worded is not met, and the note undersells it ("2 noted exceptions" — actually there are 8 more, unmentioned, in 5 more files) even though `kickoff-prompt` is a legitimately-scoped exception.
2. Given Change 8 already established the exact fix pattern (`` `core/spec-types/` `` → "the registered spec types" / "the X spec type") on an immediately adjacent file, fixing the other 5 files would have been a same-pattern extension of already-in-scope work — the same standard the ledger itself applied when it fixed the `roadmap-reviewer.md` and `completion-ledger` bonus instances "because they're a trivial same-pattern extension." Leaving this one out while fixing those two is an inconsistent scope call.

This does not change the verdict to HOLD — everything else in the delivery is real, well-scoped, and independently verified (dual-edit lockstep, full test suite, grep-zero checks, mechanical-strip integrity, and a real three-target install smoke that caught and honestly disclosed a genuine pre-existing gap in `target_cursor.go`). But it is a real gap in a named acceptance criterion, and it should go back to the engineer (or the filed follow-on task) rather than be waved through as complete.

**Positive finding worth calling out:** the ledger's "Exercise-the-feature check" claim that `target_cursor.go` never calls `installAgentsMd`/`installManagedMarkdown` (so cursor gets no managed AGENTS.md/CLAUDE.md at all) is real and verified — I confirmed by grep that `target_claude.go`, `target_opencode.go`, and `target_codex.go` all call `installAgentsMd` while `target_cursor.go` does not, and a live `--target cursor` install produced no `AGENTS.md`/`CLAUDE.md` in the output tree. This is a genuine pre-existing gap, honestly disclosed and correctly substituted around in the validation rather than silently ignored or falsely claimed as covered.

**Positive finding:** the ledger's claim that the spec's original file-count audit (67/16/13) had drifted and the real numbers were 68/16/13 was independently re-verified byte-for-byte (15 core + 34 engineering + 19 pm = 68 `compatibility:` files; 16 `role:`; 13 `domains:`; union = 96). This is exactly right, including the core/engineering/pm split — a sign the delivery re-checked its own assumptions rather than trusting stale audit numbers.

## Round 2 — verification of AC7 fix

The delivering agent claims a second pass fixed all 8 previously-found `core/spec-types/` references plus one more (`domains/pm/agents/intake-triager.md`), and updated the Completion Ledger to describe this honestly rather than re-asserting bare DONE. Verified fresh:

- **Ledger text**: AC7's row and Change 7's row now explicitly name the round-1 finding ("The cold audit (round 1) correctly flagged that AC7 was overstated...") list the exact files, and state the fix count (6 files, 9 references including `intake-triager.md`). This is a disclosure, not a re-assertion — it reads honestly, including admitting the original ledger undersold the scale.
- **Grep-zero confirmed**: `grep -rn 'core/spec-types\|domains/pm/spec-types' domains/pm/AGENTS.md domains/pm/agents/*.md` from repo root returns nothing (exit 1 / no matches). `intake-triager.md` specifically has no `spec-types` string left; it now reads "The registered `intake` spec type is the artifact you author."
- **Spot-checked replacement text for coherence** in `pm-reviewer.md` ("What you review" section — each bullet now reads "the registered `X` spec type" in place of the old `core/spec-types/x.md` path reference, grammatically clean and consistent across all five artifact bullets), `handoff-coordinator.md` (line 30: "The registered `feature` spec type is your input" — reads naturally as a lead-in to the next sentence about the owner flip), `roadmap-curator.md`, `story-writer.md`, and `AGENTS.md` — all six files use the identical "the registered `X` spec type" substitution pattern, matching what Change 8's original fix already established on an adjacent file. No dangling paths, no broken markdown, no leftover fragments.
- **`domains/pm/mission.md` correctly left alone**: still contains a live `core/spec-types/` reference (line 40, in a table cell describing shared vs. PM-led spec types). Confirmed via `grep -rn "mission.md" internal/install/` → zero hits — no install-target code references `mission.md` at all, so it is never installed to any of the six harness targets. It is repo-authoring/reference content, genuinely out of AC7's scope (which only binds *installed* content), and the ledger's self-check note about this is accurate.
- **Build and tests**: `go build ./...` clean (no output, exit 0). `go test ./...` fully green across all packages, zero `FAIL` lines.
- **Diff size sanity check**: `git diff --stat` on the six touched pm files shows small, surgical diffs (2-13 lines each) — consistent with a targeted same-pattern fix, not a rewrite or scope creep.

**Round 2 verdict: SHIP.** The round-1 gap is closed — the fix is complete (grep-zero, all 6 files including the previously-unenumerated `intake-triager.md`), the replacement text is grammatically sound and consistent, `mission.md` was correctly left out of scope with a real (not convenient) justification, and the Completion Ledger now accurately narrates the round-1 finding and its remediation rather than silently re-marking DONE. No new issues found.
