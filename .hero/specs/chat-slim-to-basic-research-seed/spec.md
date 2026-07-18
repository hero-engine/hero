---
title: "Slim domains/chat to Basic Chat and Preserve the Research Apparatus as a Dormant Hero Research Seed"
slug: chat-slim-to-basic-research-seed
type: feature
status: completed
priority: P1
domain: engineering
size: medium
created: 2026-07-18
tags: [content, domains, chat, research, hero-code, client-embedded, seed, canonical]
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  originator_slug: hero-chat-swift-app
  call_id: 18c370942a72d9a8e2b54779ba784fd7
  mode: spec-out
  handed_off_at: 2026-07-18T16:54:19Z
  at_commit: b44e9e87
  reason: "Give the canonical content owner one deliverable spec for slimming Chat while preserving the extracted Research apparatus as future product seed material, governed by the chat-sheds-research-to-seed decision."
relations:
  - target: chat-sheds-research-to-seed
    kind: depends-on
  - target: chat-canonical-research
    kind: related
horizon: now
completed_at: 2026-07-18T17:53:42Z
---

## Kickoff

`domains/chat` must be **basic Chat** — six commands, one light `AGENTS.md`, no
research — and the guided-research apparatus that `chat-canonical-research`
added (and that commit `04a0b5d` then *deleted*) must be **recovered and
preserved verbatim, dormant, outside `domains/`** as seed material for a future
Hero Research product.

This is governed by the accepted decision **`chat-sheds-research-to-seed`**. It
is **content + a focused Go content-test change** — no `hero install`, no
`go:embed`, no `AvailableDomains()` change. Chat stays a client-embedded pack.

**Two things are true on disk today and shape the work:**
1. `04a0b5d` already slimmed `domains/chat` back to the six commands and a light
   `AGENTS.md`. The slim is *mostly* done — this spec **finalizes and guards** it,
   it does not re-delete anything.
2. `04a0b5d` **deleted** the research content instead of preserving it. The
   apparatus now exists **only in git history** at commit `3a09d27`
   (`04a0b5d^`). The genuinely-undelivered work is **recovering it into a dormant
   seed** and keeping it from bit-rotting or being re-staged.

**Done when:**
- `domains/chat` ships exactly the six original commands
  (`ask-corpus`, `capture`, `discover`, `note`, `space`, `why`), a light
  `AGENTS.md`, and **no `agents/` or `skills/` directory**.
- The six commands are **behaviorally unchanged**.
- The extracted research apparatus (`research.md`, three agents, five skills) is
  **recovered byte-for-byte from `3a09d27`** into `seeds/hero-research/` with a
  provenance/dormancy `README.md`.
- Nothing in chat — command, routing row, comment, or test — **advertises or
  validates** the extracted research workflow.
- The content test enforces the basic-chat shape and validates the seed;
  chat's client-embedded invariant stays green.

**Files:** `seeds/hero-research/**` (new), `domains/chat/AGENTS.md` (confirm
light), `content_test.go`, `content.go` (comment only, confirm/repair).

**Skip:** Swift/client changes; plan/progress/interrupt UI; a Hero Research app
or initiative; a `research` or lightweight `code` domain; any hero-code change;
reopening `chat-canonical-research`; delivering this spec.

## Context

`domains/chat` is a **client-embedded pack** (`chat-pack-disposition`,
completed 2026-07-08): excluded from `content.go`'s `go:embed` set and from
`AvailableDomains()`, `DomainFS("chat")` errors, and
`clientEmbeddedDomains["chat"] = true` is the enforcement point. Its only
consumer is the sibling `../hero-code` (Rust/GPUI) client, whose
`crates/hero-core/build.rs` **iterates every directory under `domains/`** and
stages each pack's `agents/`, `skills/`, and `commands/` from source at build
time.

`chat-canonical-research` (completed 2026-07-18) extended chat with a `/research`
workflow, three specialist agents, five research/analysis skills, and a
machine-readable checkpoint/interrupt contract. The `chat-sheds-research-to-seed`
decision (accepted 2026-07-18) reversed that boundary: **basic chat does not own
research; the apparatus is extracted to a dormant seed and preserved for a
possible future Hero Research product.** That decision also superseded
`chat-app-stays-single-surface`'s "research stays baked into baseline chat"
conclusion while re-adopting its no-domain-switcher and no-lightweight-`code`
conclusions.

The ad-hoc trim (`04a0b5d`) did the removal but not the preservation, and left
the corpus briefly inconsistent. This spec is the disciplined finish: preserve
the good work, guard the boundary, and make the tree match the decision.

### The staging contract — why the seed location is what it is

`crates/hero-core/build.rs` resolves `domains_root = <hero>/domains` and does
`for entry in fs::read_dir(&domains_root)`: **every** child directory is treated
as a domain, and its `agents/` / `skills/` / `commands/` subdirs are copied into
the client's embedded content. Consequences:

- Anything under `domains/chat/…` (even a `domains/chat/_seed/agents/` subdir)
  is staged **into chat** — re-baking research into basic chat.
- A new `domains/research/…` pack is staged as a **live research domain** —
  re-exposing the apparatus to any client and contradicting the decision.
- Relying on "the seed dir has no `agents/skills/commands` child so it stages
  nothing" is fragile — it's exactly the accidental-exposure trap the decision
  names.

The build script reads **only `domains/`**; Go's `go:embed` set enumerates
specific `domains/*` and nothing else. Therefore a tree at repo root **outside
`domains/`** is invisible to both stagers. This spec uses **`seeds/hero-research/`**.

## Goal

`domains/chat` is basic Chat (six commands + light `AGENTS.md`, no research
surface); the extracted research apparatus is preserved verbatim and dormant at
`seeds/hero-research/` with documented provenance; the content test enforces the
basic-chat shape, guards against re-introducing research into chat, and validates
the seed; and chat's client-embedded invariant and the six commands' behavior are
untouched. No installable-footprint change.

## Approach

### Sizing — one medium feature

This is a single coherent content deliverable governed by one decision: recover
content into a seed, confirm/guard the slim, and update one Go test plus a
comment. There is no independent sub-deliverable that ships on its own schedule,
and no cross-cutting Go architecture. Per `spec-sizing`, a **medium feature** —
deliberately not an initiative.

### Seed location — `seeds/hero-research/` (repo root, outside `domains/`)

Chosen by inspecting the staging contract above. Rejected alternatives:
`domains/chat/<subdir>` (staged into chat), `domains/research/` (staged as a
live domain). `.hero/seeds/` is viable but `.hero/` is workspace/knowledge
metadata; the seed is **source content**, so a top-level `seeds/` tree reads more
honestly and self-documents "future product seed, not built."

Layout mirrors a pack so a future `research` domain or Hero Research app can lift
it directly:

```
seeds/hero-research/
  README.md
  commands/research.md
  agents/researcher.md
  agents/document-analyst.md
  agents/data-analyst.md
  skills/research-workflow/SKILL.md
  skills/source-evaluation/SKILL.md
  skills/evidence-and-citation/SKILL.md
  skills/document-analysis/SKILL.md
  skills/data-analysis/SKILL.md
```

The nine content files are recovered **byte-for-byte** from commit `3a09d27`
(e.g. `git show 3a09d27:domains/chat/commands/research.md`), not re-authored, so
the preserved doctrine is exactly the reviewed-and-shipped version. `README.md`
is new (see Change 2).

### What the tree already has vs. what this delivers

| Area | On disk after `04a0b5d` | This spec |
|---|---|---|
| `domains/chat` commands | Six original only | Confirm + guard; unchanged |
| `domains/chat` agents/skills | Absent | Confirm absent + assert-absent test |
| `domains/chat/AGENTS.md` | Light habits + one prose rule | Confirm no research-workflow advertising |
| Research apparatus | **Deleted** (git-only) | **Recover verbatim into `seeds/hero-research/`** |
| `seeds/hero-research/README.md` | — | **New** (provenance + dormancy + revival) |
| `content_test.go` (`TestChatPack`) | Commands-only shape | Add assert-no-agents/skills + seed validation |
| `content.go` header comment | Reverted by `04a0b5d` | Confirm accurate; repair only if stale |

## Changes

1. **Recover the research apparatus into `seeds/hero-research/`.** Copy the nine
   files from `3a09d27` verbatim into the layout above. No edits to their bodies —
   they are a frozen seed. (The client-agnostic checkpoint contract and doctrine
   travel with them, intact, for future use.)

2. **Add `seeds/hero-research/README.md`** documenting:
   - **Provenance** — authored by `chat-canonical-research`, removed from baseline
     chat by `chat-sheds-research-to-seed`, recovered from commit `3a09d27`.
   - **Dormancy** — not a live domain, **not** staged by hero-code's `build.rs`
     (which reads only `domains/`), not in any `go:embed` set; nothing loads it.
   - **Revival guidance** — reviving it (a `research` domain that `Extends: chat`,
     or a standalone Hero Research app) is a **new decision**, not a silent
     restore; do not move this back under `domains/`.

3. **Confirm `domains/chat` is basic Chat** — six commands, no `agents/`/`skills/`
   dir, light `AGENTS.md` whose only research references are the soft "habits"
   (cite sources, don't fabricate, look things up, read what's shared) and the
   single conditional natural-writing rule. Remove any residual line, routing row,
   comment, or example that advertises `/research`, the researcher lifecycle,
   plan approval, controlled-source rounds, a source ledger, source evaluation,
   evidence-synthesis progress, checkpoint state, interrupt-safe partial reports,
   or report/paper authoring. (Expected to already hold post-`04a0b5d`; this is a
   verify-and-repair step, not a rewrite. The six commands are not modified.)

4. **Update `content_test.go` (`TestChatPack`)** to enforce the boundary:
   - Assert `domains/chat` ships **exactly** the six commands and cross-checks
     each against an `AGENTS.md` routing row (bidirectional), as today.
   - **Assert `domains/chat/agents/` and `domains/chat/skills/` do not exist** —
     a regression guard so research can never be silently re-baked into chat.
   - **Validate the seed**: `seeds/hero-research/` exists with the nine expected
     files; each agent has `name:` + `description:` frontmatter (name == stem),
     each `SKILL.md` has a non-empty `description:`, and `research.md` has a
     `description:` — so the dormant seed cannot rot into malformed frontmatter.
   - **Skip cleanly** when `domains/` **or** `seeds/` is absent from the checkout,
     mirroring `TestDomainsDirectory_AllEntriesAccounted`'s `os.IsNotExist` guard.

5. **Confirm `content.go`'s header comment** describes chat as a commands-only
   client-embedded pack (no agents/skills). Repair the comment **only if** it
   still describes the research build; comment-only, no code change. Chat stays
   out of every `go:embed` line, out of `AvailableDomains()`, and in
   `clientEmbeddedDomains`.

## Boundaries

**In scope:** recovering the research apparatus into `seeds/hero-research/` with a
README; confirming and guarding the basic-chat shape of `domains/chat`; the
`TestChatPack` changes; a comment-only confirm/repair of `content.go`.

**Out of scope (explicit):**
- Swift/client changes and any plan/progress/interrupt UI — **hero-code owns**
  loading and presentation; it should no longer build chat research UI.
- A Hero Research **application or initiative**, a `research` domain, or a
  lightweight `code` domain. The seed is dormant *content*, not a product.
- Any **hero-code** change (no `build.rs` edit is needed — it reads only
  `domains/`, which the seed is outside of).
- Any `hero install` / `go:embed` / `AvailableDomains()` change — chat stays
  client-embedded.
- **Modifying the six existing commands' behavior** (wording touches only if a
  routing-consistency test demands it; no behavior change).
- **Reopening, failing, or rewriting `chat-canonical-research`** — it stays
  completed and historically correct.
- Delivering this spec.

## Risks

- **Re-staging the seed by accident.** If the seed is ever placed under
  `domains/`, `build.rs` stages it and re-exposes research. Mitigation: the seed
  is at repo-root `seeds/`, the README states the invariant, and the content test
  asserts `domains/chat` has no `agents/`/`skills/` dir. A future `research`
  domain is a deliberate decision, not an accident.
- **Seed bit-rot.** Dormant, unloaded content drifts into malformed frontmatter
  unnoticed. Mitigation: `TestChatPack` validates the seed's frontmatter even
  though nothing loads it.
- **Residual advertising of the removed workflow.** A stray comment or example
  referencing `/research` as live would mislead the client. Mitigation: Change 3
  is an explicit grep-and-repair; the routing bidirectional check fails on any
  `/research` row.
- **Corpus inconsistency during the window.** Until this lands, the tree (slim)
  and history (deleted content) disagree with the intent (preserve). Mitigation:
  the governing decision is already accepted and the old decision superseded;
  this spec is the reconciling delivery.

## Acceptance Criteria

Measurable EARS-form criteria. *The system* denotes the `domains/chat` pack, the
`seeds/hero-research/` seed, and the Go content test that guards them. Each is
independently checkable with the stated *Verify*; `hero spec verify`'s contract
gate maps tests back to these `AC-N` IDs.

- **AC-1:** the system shall ship `domains/chat` with exactly the six commands
  `ask-corpus`, `capture`, `discover`, `note`, `space`, and `why`, and with no
  `agents/` directory and no `skills/` directory. *Verify:* `TestChatPack`
  asserts the command set equals those six and that `domains/chat/agents` and
  `domains/chat/skills` do not exist.

- **AC-2:** the system shall preserve the six commands' behavior unchanged.
  *Verify:* `git diff` shows no behavioral change to the six command files
  (wording-only touches permitted solely for routing consistency).

- **AC-3:** the system shall contain, in `domains/chat`, no command, routing row,
  comment, agent, skill, or test that advertises or validates `/research`, the
  researcher lifecycle, research-plan approval, controlled-source rounds, a source
  ledger, source evaluation, evidence-synthesis progress, checkpoint state,
  interrupt-safe partial reports, or report/paper authoring. *Verify:* `grep -ri`
  over `domains/chat` finds no live-workflow reference beyond the soft "habits";
  no `AGENTS.md` routing row names `/research`.

- **AC-4:** the system shall preserve the extracted research apparatus at
  `seeds/hero-research/` as nine files — `commands/research.md`, three agents, and
  five `SKILL.md` files — byte-identical to their content at commit `3a09d27`.
  *Verify:* each seed file `diff`s clean against
  `git show 3a09d27:domains/chat/<same relative path>`.

- **AC-5:** the system shall place the research seed entirely outside `domains/`,
  such that hero-code's `build.rs` (iterating only `domains/<name>/{agents,skills,
  commands}`) stages none of it. *Verify:* every seed path begins with `seeds/`
  and no path begins with `domains/`; a build/staging dry run copies no seed file.

- **AC-6:** the system shall include `seeds/hero-research/README.md` documenting
  the seed's provenance (`chat-canonical-research`), its removal by
  `chat-sheds-research-to-seed`, its dormancy (not staged, not embedded, not a
  live domain), and revival guidance. *Verify:* the README exists and contains
  those four elements.

- **AC-7:** the system shall keep chat's client-embedded invariant intact — chat
  stays out of every `go:embed` line and out of `AvailableDomains()`, remains in
  `clientEmbeddedDomains`, and `DomainFS("chat")` returns an error. *Verify:*
  `TestDomainFS_ChatIsClientEmbedded` and `TestDomainsDirectory_AllEntriesAccounted`
  stay green and full `go test ./...` passes.

- **AC-8:** the system shall provide a Go content test that asserts the six-command
  chat shape, asserts the absence of `domains/chat/agents` and
  `domains/chat/skills`, validates seed frontmatter (agent `name:`+`description:`
  with name == stem, skill `description:`, `research.md` `description:`), and skips
  cleanly when `domains/` or `seeds/` is absent. *Verify:*
  `go test ./... -run TestChatPack` passes and the skip path uses `os.IsNotExist`.

- **AC-9:** the system shall fail its content test whenever an `agents/` or
  `skills/` directory exists under `domains/chat`, guarding against research being
  silently re-baked into chat. *Verify:* temporarily creating `domains/chat/agents/`
  makes `TestChatPack` fail (documented, not committed).

- **AC-10:** the system shall keep `AGENTS.md`'s research references limited to the
  soft, non-machine "habits" plus the single conditional natural-writing rule
  present exactly once, with no `/research` routing row. *Verify:* `AGENTS.md`
  review confirms one prose rule, soft habits only, and the routing table has
  exactly the six command rows.

- **AC-11:** the system shall leave `chat-canonical-research` unmodified as a
  completed spec — status `completed`, delivery ledger intact. *Verify:*
  `git diff` shows no change to that spec's status or ledger.

- **AC-12:** the system shall keep `content.go`'s header comment accurate to the
  basic-chat reality (commands-only, no agents/skills), changing only the comment
  if a repair is needed. *Verify:* the comment matches the tree; if touched,
  `git diff content.go` shows only comment lines.

## Validation

- `go test ./... -run TestChatPack` passes: six-command shape, no chat
  agents/skills dirs, seed present with valid frontmatter, routing bidirectional.
- `go test ./...` stays green — no regression in
  `TestDomainFS_ChatIsClientEmbedded` or `TestDomainsDirectory_AllEntriesAccounted`.
- Byte-identity check: for each of the nine seed files,
  `diff <(git show 3a09d27:domains/chat/<path>) seeds/hero-research/<path>` is
  empty.
- Un-stageable check: `git ls-files seeds/hero-research | grep -c '^domains/'` is
  `0`; a `cargo build` of hero-core (or a build.rs dry run) stages no `seeds/`
  file.
- `grep -ri 'research\|/research\|checkpoint\|plan approval' domains/chat` returns
  only the soft-habit lines in `AGENTS.md` — no live-workflow surface.
- `hero check` reports no new content warnings for the chat pack, and no
  contradictory accepted decisions (`chat-app-stays-single-surface` is
  `superseded`; `chat-sheds-research-to-seed` is the single accepted governor).
- Manual review: the six commands are behaviorally unchanged; `AGENTS.md` carries
  exactly one prose rule; the seed README states the dormancy invariant; no chat
  file names a client-private symbol as its only path.

## Kickoff

> **Deliver `chat-slim-to-basic-research-seed`** (feature, `.hero/planning/features/chat-slim-to-basic-research-seed/spec.md`).
>
> Governed by the accepted decision `chat-sheds-research-to-seed`. This is content
> + one Go content-test change — **no** `hero install`, `go:embed`, or
> `AvailableDomains()` change; chat stays client-embedded.
>
> Two facts about the tree: (1) commit `04a0b5d` already slimmed `domains/chat`
> back to the six commands and a light `AGENTS.md` — do **not** re-delete; (2)
> `04a0b5d` **deleted** the research apparatus, which now lives only at commit
> `3a09d27` — you must **recover it**.
>
> Do:
> 1. Recover the nine research files byte-for-byte from `3a09d27` into
>    `seeds/hero-research/` (`commands/research.md`; `agents/{researcher,
>    document-analyst,data-analyst}.md`; `skills/{research-workflow,
>    source-evaluation,evidence-and-citation,document-analysis,data-analysis}/SKILL.md`).
>    Use `git show 3a09d27:domains/chat/<path>`; do not re-author.
> 2. Write `seeds/hero-research/README.md`: provenance (`chat-canonical-research`),
>    removal (`chat-sheds-research-to-seed`), dormancy (outside `domains/`, not
>    staged by hero-code `build.rs`, not embedded), revival = new decision.
> 3. Confirm `domains/chat` is basic Chat: six commands, no `agents/`/`skills/`
>    dir, light `AGENTS.md` (soft research habits + one conditional prose rule).
>    Remove any residual `/research`-workflow advertising. Do not touch the six
>    commands' behavior.
> 4. Update `TestChatPack`: exact six-command set + bidirectional routing; assert
>    no `domains/chat/agents` and no `domains/chat/skills`; validate seed
>    frontmatter; skip when `domains/` or `seeds/` absent (`os.IsNotExist`).
> 5. Confirm `content.go`'s header comment matches basic-chat reality; repair
>    comment-only if stale.
>
> Green bar: `go test ./... -run TestChatPack` and full `go test ./...` pass;
> `TestDomainFS_ChatIsClientEmbedded` + `TestDomainsDirectory_AllEntriesAccounted`
> stay green; nine seed files `diff` clean vs `3a09d27`;
> `git ls-files seeds/hero-research | grep -c '^domains/'` is `0`; `hero check`
> clean. Do **not** reopen `chat-canonical-research`. Run the delivery audit and
> `hero spec verify chat-slim-to-basic-research-seed` in the same turn.

## Completion Ledger

**Task as executed.** Recovered the nine research files byte-for-byte from commit
`3a09d27` into `seeds/hero-research/` (outside `domains/`), wrote the seed
`README.md` (provenance / dormancy / revival), confirmed `domains/chat` is basic
Chat (six commands, light `AGENTS.md`, no `agents/`/`skills/`), extended
`TestChatPack` with the no-agents/skills guard + seed-frontmatter validation +
`domains/`-or-`seeds/`-absent skip, and confirmed the `content.go` comment. No
`hero install` / `go:embed` / `AvailableDomains()` change.

**Stack & validation.** Go content pack. `go vet .` / `go build ./...` / `go test .`
all clean. `TestChatPack` (basic_chat_shape + research_seed subtests),
`TestDomainFS_ChatIsClientEmbedded`, `TestDomainsDirectory_AllEntriesAccounted`
green. Nine seed files verified byte-identical to `3a09d27`. `find seeds -type f |
grep -c '^domains/'` = 0. hero-core rebuild stages only chat's six commands and
no `seeds/` content. `hero spec lint` → 12/12 EARS.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | chat = six commands, no agents/ or skills/ dir | DONE | `TestChatPack/basic_chat_shape` asserts the six-set + absent dirs |
| 2 | six commands behaviorally unchanged | DONE | untouched since original; `git diff` shows no command-body change |
| 3 | no `/research`-workflow advertising in chat | DONE | `grep -rin` over `domains/chat` returns only soft-habit lines in AGENTS.md; no `/research` routing row |
| 4 | seed 9 files byte-identical to `3a09d27` | DONE | per-file `diff` vs `git show 3a09d27:…` all empty |
| 5 | seed outside `domains/`, un-stageable | DONE | all paths under `seeds/`; hero-core rebuild stages no seed file |
| 6 | seed `README.md` (provenance/removal/dormancy/revival) | DONE | `seeds/hero-research/README.md` carries all four |
| 7 | chat client-embedded invariant intact | DONE | `TestDomainFS_ChatIsClientEmbedded` + accounting test green; chat still in `clientEmbeddedDomains` |
| 8 | content test: shape + no agents/skills + seed frontmatter + skip | DONE | `TestChatPack` two subtests; skips on `domains/` or `seeds/` absent via `os.IsNotExist` |
| 9 | test fails if `domains/chat/agents` or `skills` exists | DONE | `basic_chat_shape` stat-guards both dirs and errors if present |
| 10 | AGENTS.md soft habits + one prose rule, no `/research` row | DONE | one "Writing prose" rule; habits soft; routing table has exactly the six rows |
| 11 | `chat-canonical-research` unmodified (stays completed) | DONE | not touched this delivery |
| 12 | `content.go` comment accurate (commands-only) | DONE | already commands-only from `04a0b5d`; no repair needed |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Recover 9 research files into `seeds/hero-research/` | DONE | byte-for-byte from `3a09d27` |
| 2 | Add `seeds/hero-research/README.md` | DONE | provenance + dormancy + revival guidance |
| 3 | Confirm `domains/chat` is basic Chat | DONE | verify-and-repair; no residual research surface found |
| 4 | Update `TestChatPack` (guard + seed validation) | DONE | + `assertDescriptionFrontmatter` helper |
| 5 | Confirm `content.go` header comment | DONE | already accurate; comment-only, no change needed |

### Exercise-the-feature check

- [x] Exercised end-to-end: `go test . -run TestChatPack -v` (both subtests pass);
  byte-identity `diff` loop vs `3a09d27` (all nine clean); hero-core `cargo build`
  stages chat's six commands and zero `seeds/` files — confirming the basic-chat
  shape and the seed's dormancy/un-stageability against the real staging path.

### Excellence Bar self-check

Yes. The good research doctrine is preserved verbatim (not deleted, not re-authored)
with a clear provenance/revival README; the basic-chat boundary is enforced by a
regression guard that fails if research is ever re-baked in; and the dormant seed's
frontmatter is validated so it can't rot. One honest note: the spec's `build.rs`
staging premise differs from hero-code's later "extract-hero-content.sh enumerates
`hero domain list`" account — but the seed-outside-`domains/` conclusion is safe
under both mechanisms (verified: the current hero-core build stages no seed file),
so the deliverable is correct regardless.

## Provenance

Received via `hero peer call` **spec-out** mode (call_id
`18c370942a72d9a8e2b54779ba784fd7`) from peer `hero-code`
(peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`), originating from that repo's
`hero-chat-swift-app` initiative. Governed by the `chat-sheds-research-to-seed`
decision. hero-code owns client loading and presentation; this repo owns the
canonical, client-agnostic Chat pack content and the dormant research seed.
