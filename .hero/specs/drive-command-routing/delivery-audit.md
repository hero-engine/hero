# Delivery audit — drive-command-routing

**Audited:** working tree — `git diff -- internal/cli/deliver.go internal/cli/deliver_test.go CLAUDE.md domains/engineering/commands/deliver.md` + untracked `core/commands/drive.md`, `domains/engineering/commands/drive.md`, `domains/engineering/skills/drive/SKILL.md`
**Verdict:** SHIP
**Surface:** noteworthy

This is a mostly-harness-config spec: five of six ACs are realized as command/skill/routing markdown (the product surface), and one (the deliver guard) is code-enforceable and unit-tested. The audit treats markdown-surface ACs as delivered when the canonical definition exists and says what the AC requires; it does not pretend they carry test evidence.

## Acceptance criteria

- [✓] AC1 — `/drive <init>` arms after first-arm confirmation (condition/mode/guardrails) — `domains/engineering/skills/drive/SKILL.md` "Arming a run" steps 1–4 (confirm on first arm shows run condition, `autonomy:` mode, guardrails; offers `--dry-run 3`) + `core/commands/drive.md:13-18`. Markdown-surface AC; no code path, no test — correctly so.
- [✓] AC2 — "autopilot this initiative" + synonyms route to `/drive` — `CLAUDE.md` routing row added (`Autopilot/run a whole initiative … | /drive <initiative>`); synonyms also named in `core/commands/drive.md:8-11`. Markdown-surface.
- [✓] AC3 — `/drive` on a non-initiative declines → `/deliver` — `domains/engineering/skills/drive/SKILL.md:19-21` step 1 declines with `/deliver` pointer; `core/commands/drive.md:20-22` restates it. Markdown-surface (the skill-side decline). The CLI's complementary guard is AC4.
- [✓] AC4 — `/deliver` on an initiative offers `/drive`, no silent child delivery — `internal/cli/deliver.go:101-107` guard returns an error pointing to `/drive` / `hero goal` when `target.Type == spec.TypeInitiative`, placed before the kickoff/flip logic so the initiative is never flipped to `delivering`. Test: `TestDeliverRejectsInitiativeWithDrivePointer` (`internal/cli/deliver_test.go:454-474`) asserts the command errors and the error contains `/drive`. PASS. `domains/engineering/commands/deliver.md:8-15` adds the conversational offer. This is the one AC with real code + test evidence.
- [✓] AC5 — while armed, surface pause questions / accept answers, delegating to `hero goal --check` — `domains/engineering/skills/drive/SKILL.md:32-41` "Per turn (relay only)": continue/pause/done handled by relaying the `hero goal <init> --check` verdict; verdict authority stays in `hero goal`. Markdown-surface; the pause/resume mechanics belong to sibling specs (drive-pause-resume, hero-goal-command).
- [✓] AC6 — implemented as a skill, not a dedicated agent — `domains/engineering/skills/drive/SKILL.md` exists as a skill; verified no `agents/drive*.md` in either `core/agents/` or `domains/engineering/agents/`. Skill frontmatter declares `audience: main-loop`; body (lines 12-15, 44-48) explicitly forbids reimplementing the loop or delegating to a sub-agent, keeping the boundary in `needs_me`.

## Changes

- [✓] `core/commands/drive.md` + `domains/engineering/commands/drive.md` — both present; byte-identical (`diff` reports no differences). The `/drive <initiative>` command with arm-the-run instructions and not-`/deliver` guidance.
- [✓] `domains/engineering/skills/drive/SKILL.md` — the `drive` skill (resolve, ensure Goal, confirm-on-first-arm, emit, wire Stop hook, relay pauses). Skill, not agent — confirmed.
- [✓] `internal/cli/deliver.go` — initiative guard added (lines 101-107).
- [✓] `domains/engineering/commands/deliver.md` — `/deliver`-on-initiative fallback offer added (lines 8-15).
- [✓] `CLAUDE.md` — NL routing row added; `/drive` inserted into the slash-only list (between `/discover` and `/mock`).
- [✓] `internal/cli/deliver_test.go` — `TestDeliverRejectsInitiativeWithDrivePointer` added.

## Open items

None. No PARTIAL / SKIPPED / BLOCKED rows in the ledger.

## Audit notes

- **Build / vet / test all clean.** `go build ./...` exit 0; `go vet ./internal/cli/` exit 0; `go test ./internal/cli/` PASS (full package, 14.3s). `TestDeliverRejectsInitiativeWithDrivePointer` passes and its observed error string matches the guard: `"big-thing" is an initiative, not a single spec — run /drive big-thing …`.
- **Guard placement verified by reading the code, not just the test.** The guard sits immediately after the not-found check and before the kickoff precondition, so an initiative cannot be flipped to `delivering` or have a child stranded — it errors first.
- **Ledger honesty is good.** The Excellence Bar self-check explicitly distinguishes the one code-enforceable AC (unit-tested + exercised live) from the markdown-surface ACs (realized as canonical command/skill/routing definitions). It does not overclaim test coverage on the markdown ACs. AC4's "exercised live" claim (ran `hero spec deliver --manual` on the real initiative and got the guard error) is consistent with the verified guard behavior.
- **Skill contains no loop-driving / judgment logic** — confirmed by reading SKILL.md end to end. It relays the `hero goal --check` verdict (continue/pause/done) and repeatedly delegates judgment to `needs_me` / `hero goal`; guardrails are stated as relay rules ("when unsure, pause"), not as decision logic implemented in the skill.
- **Surface = noteworthy** is set not because of any defect but because the user should know this delivery is overwhelmingly harness-config: five of six ACs ship as markdown surface with no executable test, and their real behavior depends on sibling specs (hero-goal-command, drive-pause-resume, initiative-goal-section) that this spec references but does not implement. That dependency is by design (this spec is the entry-point layer), but it's worth the user's attention that "armed run works end-to-end" is not provable from this spec's artifacts alone — only the deliver guard is.
- **No scope drift.** The diff touches exactly the files named in the Changes section; nothing extra.
