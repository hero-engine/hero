---
title: "Chat Canonical Research Pack — Extend domains/chat into the Canonical Generic Hero Chat Domain"
slug: chat-canonical-research
type: feature
status: completed
priority: P1
domain: engineering
size: medium
created: 2026-07-18
tags: [content, domains, chat, research, hero-code, client-embedded, canonical]
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  originator_slug: hero-chat-swift-app
  call_id: 18c34c1cbf179388373a062d85b5529a
  mode: spec-out
  handed_off_at: 2026-07-18T05:46:03Z
  at_commit: c5f883c9
  reason: "The standalone Swift Hero Chat product needs a canonical generic Chat domain pack owned by the Hero content repository, while hero-code will own client loading and presentation."
relations:
  - target: chat-pack-disposition
    kind: builds-on
  - target: multi-domain-context-activation
    kind: related
horizon: now
completed_at: 2026-07-18T06:11:56Z
---

## Kickoff

Extend the existing `domains/chat` pack — today six commands, one AGENTS.md, no
agents, no skills — into the **canonical generic Hero Chat domain**: the
baseline conversational pack that Hero Chat (Swift), Hero Code's Chat mode, the
CLI, or any future client can select and get a real research assistant, not just
Q&A shortcuts.

**This is content authoring plus a focused Go content-validation test.** No
`hero install` change, no `go:embed` line for chat — chat stays a client-embedded
pack (its consumer is `../hero-code`'s `crates/hero-core/build.rs`, which stages
every `domains/<name>/{agents,skills,commands}` directory from source at build
time). Adding `domains/chat/agents/` and `domains/chat/skills/` means those
directories start getting staged automatically; **hero-code owns loading them and
rendering their progress/interrupt surface** — that is explicitly out of scope
here.

**Done when:**
- The six existing commands (`ask-corpus`, `capture`, `discover`, `note`,
  `space`, `why`) are preserved unchanged in behavior.
- A new `/research` command drives a rigorous, reviewable research workflow.
- The smallest useful set of hidden skills (research doctrine) and specialist
  agents (research, document analysis, data analysis) exists — three agents, no
  zoo.
- `AGENTS.md` carries the updated routing table and exactly one conditional
  prose-writing instruction.
- A Go test validates chat pack content (frontmatter + routing/command
  consistency) even though chat is not embedded.
- Ordinary summarization / comparison / explanation / brainstorming remain
  natural-language intents (no command, no agent).

**Files:** `domains/chat/AGENTS.md`, `domains/chat/commands/research.md`,
`domains/chat/agents/*.md`, `domains/chat/skills/*/SKILL.md`,
`content_test.go`, `content.go` (comment only).

**Skip:** Swift app wiring, any UI, artifacts implementation, the custom
assistant builder, study mode, a write command / writer agent, and delivering
this spec.

## Context

`domains/chat` was formalized as a **client-embedded pack** by
`chat-pack-disposition` (completed 2026-07-08). It is deliberately excluded from
`content.go`'s `go:embed` set and from `AvailableDomains()`; `DomainFS("chat")`
errors; `clientEmbeddedDomains["chat"] = true` is the enforcement point. Its only
consumer is the sibling `../hero-code` (Rust/GPUI) client, whose
`crates/hero-core/build.rs` iterates *every* directory under `domains/` and
stages each pack's `agents/`, `skills/`, `commands/` — so any directory we add
under `domains/chat/` is picked up without a Go change.

The peer call originates from `hero-code`'s `hero-chat-swift-app` initiative: a
standalone Swift Hero Chat product. hero-code owns the client (loading,
presentation, the progress and interrupt surfaces). This repo owns the
**canonical, client-agnostic content**. In the domain type model
(`internal/domains/types.go`), chat is the **baseline** domain — `Extends` is nil
for core/chat, and every extension domain (pm, sales, code) declares
`Extends: "chat"`. Making chat a complete generic assistant is therefore not a
side pack; it is hardening the substrate every other vertical composes onto.

Today the pack is thin. The disposition audit already flagged two client-agnostic
smells to avoid repeating: `ask-corpus.md` names tool identifiers
(`semantic_search`, `read_file`) no install target guarantees, and `space.md`
references a client-private `SpaceStore` API. The convention the pack already
half-follows — describe session capabilities abstractly, mention a specific
client only as an optional aside ("in the hero-code GPUI client specifically…") —
becomes a hard rule for everything this spec adds.

## Goal

`domains/chat` becomes a self-contained, client-agnostic conversational assistant
pack: the six existing commands unchanged, one new `/research` command backed by
a rigorous research workflow (reviewable plan → controlled sources → iterative
search → source evaluation → cited synthesis, with a client-renderable progress
and interrupt protocol), three specialist agents (researcher, document-analyst,
data-analyst), a minimal set of hidden skills carrying the research doctrine, and
an `AGENTS.md` that routes correctly and carries exactly one conditional
prose-writing instruction. A Go content test validates the pack's frontmatter and
command/routing consistency despite chat being unembedded. No installable
footprint changes; ordinary summarize/compare/explain/brainstorm stay
natural-language intents.

## Approach

### Sizing — one medium feature, not an initiative

This is a single coherent content deliverable: one pack, authored, validated, and
tested in one pass. There is no independent sub-deliverable that ships on its own
schedule — the command, agents, skills, and AGENTS.md are meaningless apart from
each other, and there is no cross-cutting Go architecture (the only code is one
additive test plus a comment). Per `spec-sizing`, that is a **medium feature**.
It is deliberately *not* composed as an initiative: forcing sibling specs here
would fragment a pack that must be reviewed as a whole.

### What the pack ships (and what it deliberately does not)

| Surface | Add | Rationale |
|---|---|---|
| Commands | **1 new** (`/research`); 6 preserved | Research is the one workflow rich enough to need explicit orchestration. Everything else is conversational. |
| Agents | **3** (`researcher`, `document-analyst`, `data-analyst`) | One per specialist concern named in the request. No zoo — no summarizer, comparer, explainer, or brainstormer agent. |
| Skills | **5 hidden** (see below) | The research doctrine is heavy enough to warrant separable, reusable skills; document/data analysis get one skill each. |
| AGENTS.md | rewrite routing table + one prose rule | Identity surface for a pack with no installable footprint. |

**Ordinary summarization, comparison, explanation, and brainstorming stay
natural-language intents.** They get no command and no agent — the base model
handles them conversationally. `/discover` already covers generative
brainstorming when the user wants structured option-weighing; nothing new is
needed. The AGENTS.md routing table says so explicitly so a future editor does
not "helpfully" add a `/summarize` command.

### The research workflow (`/research` + hidden skills)

The command is thin orchestration; the rigor lives in skills the `researcher`
agent loads. The workflow has seven required properties, mapped to where each
lives:

1. **Reviewable planning** — `/research` first produces a *research plan*
   (question restated, sub-questions, candidate source classes, stopping
   criteria) and **pauses for user review/approval before searching**. The plan
   is a checkpoint the client can render and the user can edit or approve.
2. **Controlled sources** — the plan declares an explicit, bounded source set
   (corpus-only, web-allowed, or a named allowlist). The researcher does not
   silently widen scope; broadening requires an amended plan.
3. **Iterative search** — search proceeds in *rounds*; each round emits a short
   summary (queries run, what was found, what's still missing) and decides
   whether another round is warranted against the plan's stopping criteria.
4. **Source evaluation** — every retained source is triaged for
   credibility/recency/primary-vs-secondary/bias before its claims are used.
5. **Evidence synthesis** — claims are assembled from *evaluated* sources;
   contradictions between sources are surfaced, not silently resolved.
6. **Citations** — every non-obvious claim in the final report carries an inline
   citation to a specific source (corpus path or URL); a `Sources:` register
   lists them.
7. **Progress visibility + interruptibility** — because the pack is
   client-agnostic it cannot own UI. Instead the workflow defines a **checkpoint
   protocol** any client can render: `plan` → `round` (×N) → `evaluation` →
   `synthesis` → `report`. On interrupt, the researcher checkpoints partial
   findings so a client's stop control yields a usable partial report with an
   explicit "incomplete — stopped after round K" banner, never a dropped turn.

The skill defines the checkpoint *contract* (names, ordering, what each carries);
hero-code implements the *surface* (how the plan is shown for approval, how
progress renders, how the stop control signals interrupt).

### Hidden skills (smallest useful set)

| Skill | Backs | Carries |
|---|---|---|
| `research-workflow` | researcher | Orchestration doctrine: plan-first, controlled sources, round loop, stopping criteria, the checkpoint/interrupt protocol. |
| `source-evaluation` | researcher, document-analyst | How to triage a source — credibility, recency, primary vs secondary, bias, corroboration; when to discard. |
| `evidence-and-citation` | researcher, document-analyst, data-analyst | Assembling evaluated evidence into claims, inline citation format, surfacing contradictions, the `Sources:` register. |
| `document-analysis` | document-analyst | Grounded deep-read of a single provided document/corpus item — structure extraction, claim/evidence mapping, answer-with-citation, "not stated in this document" honesty. |
| `data-analysis` | data-analyst | Analysis of user-provided structured/tabular data — compute, summarize, surface patterns, and caveat limitations (sample size, missingness, correlation≠causation). |

Three of the five (`research-workflow`, `source-evaluation`,
`evidence-and-citation`) are the research doctrine; the other two are the
specialist agents' doctrine. This is the minimum that still lets each agent carry
real rigor without duplicating the citation/evaluation rules three times.

### Agents (three; no zoo)

| Agent | Role |
|---|---|
| `researcher` | Runs the `/research` workflow end to end — plan, controlled search rounds, source evaluation, cited synthesis, checkpoint emission, interrupt-safe. |
| `document-analyst` | Deep, grounded analysis of a specific document or corpus item the user points at; answers strictly from the source with citations; says plainly when the answer isn't in the document. |
| `data-analyst` | Analysis of structured data the user supplies or references; computes and summarizes, surfaces patterns, and states limitations honestly. |

All three follow the existing engineering/pm agent file shape: YAML frontmatter
(`name`, `description`, `mode: subagent`, `temperature`, `color`, `permission`)
then a body with role, when-invoked, workflow, and the "cite abstractly, name a
specific client only as an aside" client-agnostic rule.

### AGENTS.md — routing rewrite + one prose rule

Rewrite the routing table to add `/research` and keep the six existing rows;
remove the current "chat has no agents, no skills, and no delivery workflow"
line (now false); add a short "hidden agents & skills" note and an explicit
"these stay natural-language intents: summarize, compare, explain, brainstorm"
line. Then add **exactly one** conditional instruction — verbatim intent:

> **When (and only when) the user asks you to draft or revise prose meant for
> other people** (an email, a post, a doc, a message), write it naturally: match
> the audience and the user's own voice, prefer plain concrete language and
> varied sentence lengths, and avoid generic AI filler, canned transitions,
> excessive headings, repetitive phrasing, and em dashes. Follow the requested
> format and length. This applies to human-facing prose only — not to normal
> conversational replies, research reports, or analysis output.

No write command and no writer agent are created — this is a behavioral rule on
the base assistant, gated to the prose-drafting case so it never bleeds into
ordinary chat.

## Changes

1. **Add `domains/chat/commands/research.md`** — the `/research` command.
   - Frontmatter `description:` (one line), matching the existing six commands.
   - Body: restate question from `$ARGUMENTS`; produce a research plan and
     **pause for approval before any search**; declare the controlled source set;
     run iterative search rounds emitting per-round summaries; evaluate each
     retained source; synthesize a cited report; emit the `plan → round →
     evaluation → synthesis → report` checkpoints; honor interrupt by
     checkpointing partial findings.
   - Delegates the heavy lifting to the `researcher` agent where the session
     exposes subagents; otherwise runs the workflow inline.
   - References session capabilities abstractly (e.g. "the session's web-search
     and file-read capabilities"), never a named client-private symbol as the
     only path.

2. **Add `domains/chat/agents/researcher.md`, `document-analyst.md`,
   `data-analyst.md`** — the three specialist agents, in the standard agent file
   shape, each loading the relevant hidden skills and carrying the
   client-agnostic citation rule.

3. **Add hidden skills** under `domains/chat/skills/<name>/SKILL.md`:
   `research-workflow`, `source-evaluation`, `evidence-and-citation`,
   `document-analysis`, `data-analysis`. Each has `name:` + `description:`
   frontmatter and a body per the table above. `research-workflow` is the
   authoritative definition of the checkpoint/interrupt protocol.

4. **Rewrite `domains/chat/AGENTS.md`**:
   - Add the `/research` row; keep the six existing rows.
   - Replace the "no agents, no skills, no delivery workflow" paragraph with a
     "hidden agents & skills" note and the explicit "summarize / compare /
     explain / brainstorm stay natural-language intents" line.
   - Add the single conditional prose-writing instruction (above).
   - Keep the client-embedded framing (consumed from source by hero-code; not
     installed via `hero install`).

5. **Add a chat pack content test to `content_test.go`** (chat is not embedded,
   so the existing frontmatter walkers skip it):
   - Walk `domains/chat/agents/*.md` and assert `name:` + `description:`
     frontmatter (mirrors `assertAgentFrontmatter` — a missing `name:` silently
     drops an agent from the subagent registry).
   - Walk `domains/chat/skills/*/SKILL.md` and assert `description:` frontmatter
     (mirrors `assertSkillFrontmatter`).
   - Assert the shipped command set equals exactly the seven expected files, and
     that every shipped command has a routing-table row in `AGENTS.md` and vice
     versa (a chat-scoped routing-consistency check; the existing
     `markdown_drift_test` deliberately covers only the engineering pack).
   - Read from the repo checkout and **skip cleanly when `domains/` is absent**,
     matching `TestDomainsDirectory_AllEntriesAccounted`.

6. **Update the `content.go` header comment** (comment only, no code) to note
   that chat now ships `agents/` and `skills/` staged by hero-code's `build.rs`,
   so the "chat has only commands/" description no longer misleads. `chat` stays
   out of every `go:embed` line, out of `AvailableDomains()`, and in
   `clientEmbeddedDomains`.

## Boundaries

**Out of scope (explicit):**
- Swift application wiring, any UI, and the client-side loading/presentation of
  these agents/skills — **hero-code owns that** (loading resolver, plan-approval
  UI, progress rendering, the stop/interrupt control).
- Artifacts implementation, the custom assistant builder, and study mode.
- **No write command and no writer agent** — the prose guidance is one
  conditional AGENTS.md rule, nothing more.
- Engineering, delivery, QA, PM, code-editor, and tracker workflows — chat must
  not route to `/design`, `/deliver`, `/diagnose`, `/mock`, task/tracker
  commands, or any other domain's surface.
- Any `hero install` / `go:embed` / `AvailableDomains()` change — chat stays a
  client-embedded pack.
- Modifying the six existing commands' behavior (light wording touches only if
  needed for the routing-consistency test; no behavior change).
- Delivering this spec.

## Risks

- **Client-agnostic drift.** The biggest failure mode (already seen in
  `ask-corpus.md`/`space.md`) is baking a specific client's tool names or private
  APIs into canonical content. Mitigation: the "describe capabilities abstractly,
  name a client only as an optional aside" rule is stated in AGENTS.md and each
  agent, and enforced by review. A hard automated guard is hard to write well and
  is left to review rather than a brittle grep test.
- **Progress/interrupt contract split.** The pack defines the checkpoint protocol
  but cannot test it end-to-end without a client. Mitigation: the contract
  (checkpoint names, ordering, partial-report-on-interrupt) is specified
  concretely in `research-workflow` so hero-code implements against a fixed
  surface; the round-trip is validated on hero-code's side, referenced by
  `hero-chat-swift-app`.
- **Agent-zoo creep.** Three agents is the ceiling. If a reviewer wants a
  summarizer/comparer/explainer agent, that is the anti-goal — those are
  natural-language intents by design.
- **Unembedded content escaping validation.** Chat content is not in the
  `go:embed` set, so without Change 5 its frontmatter is never checked and a
  malformed agent would ship silently to hero-code. The added test closes that
  gap.

## Acceptance Criteria

Measurable EARS-form criteria derived from **Done when** and **Validation**. In
the criteria below, *the system* denotes the `domains/chat` pack — its commands,
agents, skills, `AGENTS.md`, and the Go content test that guards it. Each
criterion is independently checkable and carries an explicit *Verify* method;
`hero spec verify`'s contract gate maps tests back to these `AC-N` IDs.

- **AC-1:** the system shall continue to ship exactly the six existing commands
  (`ask-corpus`, `capture`, `discover`, `note`, `space`, `why`), each retaining
  its `description:` frontmatter and behavioral body unchanged. *Verify:*
  `git diff` shows no behavioral change to the six files (wording-only touches
  permitted solely for routing consistency).

- **AC-2:** the system shall provide `domains/chat/commands/research.md` whose
  body specifies, in order, restating the question from `$ARGUMENTS`, producing a
  research plan, pausing for approval before any search, declaring a controlled
  source set, running iterative search rounds with per-round summaries,
  evaluating each retained source, synthesizing a cited report, and emitting the
  `plan → round → evaluation → synthesis → report` checkpoints. *Verify:* the
  file exists with `description:` frontmatter and every listed element is present
  and followable read cold.

- **AC-3:** when `/research` produces its research plan, the system shall pause
  for user review and approval before issuing any search. *Verify:* `research.md`
  and the `research-workflow` SKILL.md both state the pre-search approval
  checkpoint as mandatory.

- **AC-4:** if the user interrupts a running `/research`, then the system shall checkpoint partial findings and yield a usable partial report banner-marked "incomplete — stopped after round K" rather than dropping the turn. *Verify:* the `research-workflow` SKILL.md specifies the partial-report-on-interrupt contract with those checkpoint semantics.

- **AC-5:** the system shall ship exactly three agents — `researcher`,
  `document-analyst`, `data-analyst` — each with `name:` and `description:`
  frontmatter whose `name` matches the filename stem. *Verify:*
  `domains/chat/agents/` contains exactly those three `.md` files and the content
  test asserts their frontmatter.

- **AC-6:** the system shall ship exactly five hidden skills —
  `research-workflow`, `source-evaluation`, `evidence-and-citation`,
  `document-analysis`, `data-analysis` — each a `<name>/SKILL.md` with a
  non-empty `description:` frontmatter. *Verify:* the content test asserts skill
  frontmatter and the directory contains exactly those five.

- **AC-7:** the system shall carry, in `AGENTS.md`, a routing table with seven
  command rows (the six existing plus `/research`), a hidden agents & skills
  note, an explicit "summarize / compare / explain / brainstorm stay
  natural-language intents" line, and exactly one conditional prose-writing
  instruction gated to human-facing prose. *Verify:* `AGENTS.md` contains exactly
  one prose-writing instruction and introduces no write command or writer agent.

- **AC-8:** the system shall maintain a bidirectional match between shipped
  commands and `AGENTS.md` routing rows — every shipped command has a routing row
  and every routing row points at a shipped chat command. *Verify:* the content
  test asserts the command set equals the seven expected files and cross-checks
  routing rows in both directions.

- **AC-9:** the system shall include a Go test that validates chat agent
  frontmatter (`name:` + `description:`), chat skill frontmatter (`description:`),
  the exact seven-command set, and routing-table bidirectional consistency.
  *Verify:* `go test ./... -run TestChatPack` passes.

- **AC-10:** if `domains/` is absent from the checkout, then the system shall skip its content test cleanly rather than fail. *Verify:* the test guards on `os.IsNotExist` and calls `t.Skip`, mirroring `TestDomainsDirectory_AllEntriesAccounted`.

- **AC-11:** the system shall keep `chat` out of every `go:embed` line, out of
  `AvailableDomains()`, and in `clientEmbeddedDomains`, and `DomainFS("chat")`
  shall continue to return an error. *Verify:* `TestDomainFS_ChatIsClientEmbedded`
  and `TestDomainsDirectory_AllEntriesAccounted` stay green and full
  `go test ./...` passes.

- **AC-12:** the system shall not hard-require a named client-private symbol as
  the only path in any chat file; client-specific APIs shall appear only as
  optional asides. *Verify:* manual review confirms no chat file names a
  client-private symbol as its sole path and no routing row points at another
  domain's command.

- **AC-13:** the system shall update the `content.go` header comment — and only
  that comment — to note that chat now ships `agents/` and `skills/` staged by
  hero-code's `build.rs`. *Verify:* `git diff content.go` shows only comment
  lines changed.

- **AC-14:** the system shall leave ordinary summarization, comparison,
  explanation, and brainstorming as natural-language intents, owning no command
  and no agent for them. *Verify:* no `/summarize`, `/compare`, `/explain`, or
  `/brainstorm` command or agent exists and `AGENTS.md` states these stay
  natural-language intents.

## Validation

- `go test ./... -run TestChatPack` (or the chosen test name) passes: chat agent
  and skill frontmatter present; command set is exactly the seven expected; every
  command has an AGENTS.md routing row and vice versa.
- `go test ./...` stays green — no regression in `TestDomainFS_ChatIsClientEmbedded`
  (chat still errors from `DomainFS`) or `TestDomainsDirectory_AllEntriesAccounted`
  (chat still accounted for via `clientEmbeddedDomains`, not `AvailableDomains`).
- `hero check` reports no new content warnings for the chat pack.
- Manual review checklist:
  - The six existing commands are behaviorally unchanged.
  - `AGENTS.md` contains exactly one prose-writing instruction, gated to
    human-facing prose, with no write command/agent introduced.
  - No chat file hard-requires a named client-private symbol as its only path;
    client-specific APIs appear only as optional asides.
  - No routing row points at another domain's command.
  - The `/research` workflow, read cold by an engineer agent, is followable end
    to end: plan-approval pause, controlled sources, round loop, source
    evaluation, cited synthesis, checkpoint emission, interrupt safety.
- Confirm `../hero-code`'s `crates/hero-core/build.rs` picks up the new
  `domains/chat/agents` and `domains/chat/skills` directories (staging is
  directory-iterating, so no build.rs change is expected — verify, don't assume).

## Provenance

Received via `hero peer call` **spec-out** mode (call_id
`18c34c1cbf179388373a062d85b5529a`) from peer `hero-code` (peer_id
`cd8dd06d-3df1-4878-a88f-24593dcbb4b3`), originating from that repo's
`hero-chat-swift-app` initiative. hero-code will own client loading and
presentation; this repo owns the canonical, client-agnostic Chat pack content.

## Completion Ledger

**Task as executed.** Extended `domains/chat` from six commands into the canonical
generic Hero Chat assistant: added the `/research` command, three specialist
agents (`researcher`, `document-analyst`, `data-analyst`), five hidden research /
analysis-doctrine skills, rewrote `AGENTS.md` (routing + hidden-surface note +
one gated prose rule), added the `TestChatPack` content test, and updated the
`content.go` header comment. No `hero install` / `go:embed` / `AvailableDomains()`
change — chat stays client-embedded.

**Stack & validation.** Go content pack. `go vet .` clean; `go build ./...` clean;
`go test .` green (root package, 38s); `TestChatPack`, `TestDomainFS_ChatIsClientEmbedded`,
`TestDomainsDirectory_AllEntriesAccounted`, `TestEmbeddedAgents/Skills_*` all PASS;
`hero spec lint` → 14/14 criteria classify as EARS. Verified hero-code's
`crates/hero-core/build.rs` iterates every `domains/<name>` directory and stages
`agents`/`skills`/`commands` when present (build.rs:76,88), so the new chat
directories are picked up with no build.rs change.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Six existing commands preserved unchanged | DONE | `domains/chat/commands/{ask-corpus,capture,discover,note,space,why}.md` untouched; diff shows no changes to the six |
| 2 | `/research` command complete (plan→pause→sources→rounds→eval→synth→checkpoints) | DONE | `domains/chat/commands/research.md` — every workflow element present with `description:` frontmatter |
| 3 | WHEN plan produced, pause for approval before search | DONE | `research.md` step 2 + `research-workflow/SKILL.md` Phase 1 both mark the pre-search pause mandatory |
| 4 | IF interrupted, checkpoint partial + banner, never drop turn | DONE | `research-workflow/SKILL.md` "Interrupt safety"; `research.md` interrupt clause; researcher agent |
| 5 | Exactly three agents, valid frontmatter, name==stem | DONE | `domains/chat/agents/{researcher,document-analyst,data-analyst}.md`; `TestChatPack/agents` asserts frontmatter |
| 6 | Exactly five hidden skills, non-empty description | DONE | `domains/chat/skills/{research-workflow,source-evaluation,evidence-and-citation,document-analysis,data-analysis}/SKILL.md`; `TestChatPack/skills` |
| 7 | AGENTS.md: 7 rows + hidden note + NL-intents line + exactly one prose rule | DONE | `domains/chat/AGENTS.md` — one "Writing prose for other people" section, no write command/agent |
| 8 | Routing bidirectional (command↔row) | DONE | `TestChatPack/commands_and_routing` cross-checks both directions |
| 9 | Go content test validates frontmatter + command set + routing | DONE | `content_test.go` `TestChatPack` — passes |
| 10 | IF `domains/` absent, test skips cleanly | DONE | `content_test.go:` `os.Stat`+`os.IsNotExist`→`t.Skip`, mirrors `TestDomainsDirectory_AllEntriesAccounted` |
| 11 | chat stays out of embed/AvailableDomains, in clientEmbeddedDomains, DomainFS errors | DONE | `content.go` unchanged behavior; `TestDomainFS_ChatIsClientEmbedded` + `TestDomainsDirectory_AllEntriesAccounted` green |
| 12 | No client-private symbol as only path; asides only | DONE | Each agent + `research.md` + skills state the "describe abstractly, name a client only as an aside" rule; no named private symbol as sole path |
| 13 | content.go comment-only update | DONE | `content.go:9-21` header comment only; `go build` confirms no code change |
| 14 | Summarize/compare/explain/brainstorm stay NL intents | DONE | `AGENTS.md` "These stay natural-language intents"; no such command/agent shipped |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add `commands/research.md` | DONE | Thin orchestration; delegates to `researcher` where subagents exist, inline otherwise |
| 2 | Add three agents | DONE | `researcher.md`, `document-analyst.md`, `data-analyst.md` in standard agent shape, load their skills |
| 3 | Add five hidden skills | DONE | research-workflow (authoritative checkpoint/interrupt contract) + source-evaluation, evidence-and-citation, document-analysis, data-analysis |
| 4 | Rewrite `AGENTS.md` | DONE | `/research` row added, six kept; "no agents/skills" para replaced; NL-intents line; one prose rule; client-embedded framing kept |
| 5 | Add chat pack content test | DONE | `TestChatPack` + `assertStringSetsEqual` helper; skips when `domains/` absent |
| 6 | Update `content.go` header comment | DONE | Comment-only; chat still unembedded |

### Exercise-the-feature check

- [x] Pack validated end-to-end via `go test . -run TestChatPack -v` — agents, skills, and command/routing subtests all PASS; `hero spec lint chat-canonical-research` → 14/14 EARS. The pack's user-visible surface (loadable agents/skills/commands with consistent routing) is what the content test exercises; the runtime plan-approval/progress/interrupt *rendering* is hero-code's surface and explicitly out of scope here.

### Excellence Bar self-check

Yes — a senior engineer would be proud to ship this. The research doctrine is
genuinely rigorous (plan-first with a hard approval pause, controlled sources,
round loop with honest stopping, per-source evaluation, cited synthesis with
contradiction-surfacing, interrupt-safe partial reports), the no-zoo discipline is
held (three agents, zero summarizer/comparer/explainer agents), the client-agnostic
rule is stated in every authored file, and the content test closes the exact
validation gap the spec identified for unembedded content. One honest edge: the
client-agnostic guarantee (AC-12) is enforced by review, not an automated grep —
a deliberate choice the spec's Risks section already made.

## Handoff Trail

- 2026-07-18T14:45:00Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: advisory
  originating_spec: chat-canonical-research
  at_commit: ff8c557
  result_ref: .hero/peer-calls/18c36961159146f873620cdd561c59ac.md
  reason: "Closing the loop: chat-canonical-research (spec-out from hero-chat-swift-app) is delivered on hero main; the client-agnostic checkpoint contract is fixed for hero-code to build its loading + plan-approval/progress/interrupt surface against."

