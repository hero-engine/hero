---
name: domain-glossary-maintenance
description: Keeping a shared PM/eng domain vocabulary alive as a knowledge entry in .hero/knowledge — term entry shape, add/retire discipline, and how a missing entry shows up as spec ambiguity.
metadata:
  audience: pm-reviewer, convention-author, product-strategist
  purpose: framework-guidance
---

## What I do

Give agents the practice of maintaining a **domain glossary** — one shared, living definition of every load-bearing term the product and engineering teams use — as a knowledge entry under `.hero/knowledge/`. Most cross-team confusion is vocabulary confusion: PM says "account," eng hears "user," and a whole spec is built on a word two people read differently. The glossary is the single place a term means one thing, owned and dated, so specs can reference it instead of silently re-defining it.

## When to use me

- a spec review keeps stalling on "wait, what do we mean by X?" (`pm-reviewer`)
- codifying team-specific terms that no vocabulary preset covers (`convention-author`)
- onboarding — a new PM or engineer needs the shared language, not tribal knowledge (`product-strategist`)
- two documents use the same word for different things, or different words for the same thing

## Where it lives

The glossary is a knowledge entry in `.hero/knowledge/` (e.g. `.hero/knowledge/domain-glossary.md`), so it's version-controlled, travels with the repo, and is searchable by every agent via `hero search`. It is **not** a wiki page in a tool half the team can't see, and not a section buried in one PRD. One canonical file, referenced by slug from specs that use its terms.

## Why the glossary and not just "ask someone"

Tribal vocabulary — the meaning that lives only in a senior teammate's head — is a tax on every new hire and a landmine in every spec written by someone who wasn't in the room. The glossary converts that tacit knowledge into an asset the whole team (and every Hero agent) can reference. The moment a term's meaning is contested, the *cost of not writing it down* is a spec built on a misunderstanding; the glossary is cheap insurance against a whole class of expensive rework.

## Term entry shape

Each term is a small structured block. The five fields each prevent a specific class of confusion:

```
### Account
- **Definition:** a billing entity that owns one or more seats; the unit we invoice.
- **Owner:** billing squad (PM: dana)
- **Aliases:** "org," "workspace" (UI-facing), "tenant" (infra-facing)
- **Not to be confused with:** *User* (a single human login; an Account has many)
  or *Team* (a permissions grouping *within* an Account).
- **Added:** 2026-05 · last reviewed 2026-07
```

- **Definition** — one sentence, unambiguous, in the domain's terms. No "see below."
- **Owner** — who arbitrates when the definition is contested. A term with no owner drifts.
- **Aliases** — the other words teams actually use for this, especially UI vs. infra vs. sales naming, so search and reading resolve to one entry.
- **Not to be confused with** — the sibling terms people *do* confuse it with, named explicitly. This field does the most work; ambiguity lives in the gaps between near-synonyms.
- **Dates** — added and last-reviewed, so staleness is visible.

## When to add a term

Add a term when it's **load-bearing and contested** — it appears in specs, and two people would define it differently. Signals: a review comment asking "do you mean X or Y?"; the same word used for two concepts in two specs; a new concept a feature introduces (a new object, state, or role). Don't add every noun — a glossary of 200 obvious words is a glossary nobody reads. Add the terms where a wrong reading changes what gets built.

## When to retire a term

Retire a term when the concept is gone (a feature was removed) or when it merged into another term (two words reconciled to one). **Retire explicitly** — mark it deprecated with a pointer to the replacement rather than deleting it, so old specs that reference it still resolve. A silently deleted term leaves dangling references and re-opens the confusion the entry closed.

## How missing entries show up as spec ambiguity

A missing or drifted glossary entry rarely announces itself — it surfaces downstream as a spec that reads fine to its author and wrong to its reviewer. When a review finds "this acceptance criterion is ambiguous," trace it back: often the ambiguity is one undefined term doing double duty ("notify the *user*" — which user, in an Account with many?). The fix isn't just clarifying that one spec; it's adding the glossary entry so the *next* spec doesn't reintroduce the ambiguity. Treat recurring "what do you mean by…" review comments as glossary bug reports.

## Reconcile PM and eng, don't fork

The failure mode that kills glossaries is two of them — a PM glossary and an eng glossary that define the same terms differently and never meet. The value is precisely in the *shared* definition. When PM and eng disagree on a term, that disagreement is the artifact worth capturing: record both readings, name the owner who arbitrates, and converge to one entry. A glossary that only one side reads is just that side talking to itself.

## Keeping it alive — the review cadence

A glossary that isn't maintained is worse than none, because people trust a stale definition. Two lightweight habits keep it honest:

- **Review-triggered updates.** Every time a spec review surfaces a "what do you mean by X?" comment, that's a glossary event — add or sharpen the entry as part of closing the review, not as separate work nobody schedules.
- **Periodic staleness sweep.** On a cadence (quarterly is plenty), scan for entries whose *last reviewed* date is old and whose owner has moved on. Confirm they still hold or retire them. The date fields exist precisely so this sweep is mechanical.

The glossary should be **referenced, not recited** — specs link to the term by slug rather than restating its definition, so there's exactly one place the meaning lives and one place to change it. A term that's never referenced from any spec is a candidate for retirement: if no artifact needs it, it isn't load-bearing.

## Anti-patterns

- **Glossary nobody reads.** 200 obvious terms, no owners, never referenced from a spec. Dead weight. Keep it to load-bearing, contested terms and reference it from the specs that use them.
- **Duplicate terms with drifted definitions.** "Account" defined one way in the glossary and another in a PRD. The glossary must be the single source; specs reference, they don't re-define.
- **Eng-only or PM-only glossaries.** Two forks that never reconcile defeat the entire purpose. One shared entry, one owner per term, disagreements recorded and arbitrated.
- **Ownerless terms.** No owner means no one arbitrates drift, and the definition rots. Every term names who decides.
- **Silent deletion.** Removing a retired term instead of deprecating it with a pointer, leaving old specs referencing a definition that vanished.
- **Definitions that defer.** "Account: see the billing docs." A glossary entry that points elsewhere for its own definition isn't an entry.
- **Encyclopedia entries.** A paragraph of history and caveats where a sentence would do. The value is a crisp, shared definition; length buries it and guarantees nobody reads it.
- **Aspirational definitions.** Defining a term as the team *wishes* it were used rather than how it's actually used. The glossary records the real, shared meaning; if you want to change usage, that's a rename, tracked explicitly.

## Cross-references

- `convention-writing` — a contested term often signals a convention worth codifying; the glossary names the vocabulary, the convention names the pattern.
- `pm-preset-detection` — the active vocabulary preset renames *artifacts* (Story/Scope/Card); the glossary records the *domain* terms the preset doesn't cover.
- `story-writing-invest` — ambiguous acceptance criteria frequently trace to one undefined term; the glossary is the fix, not just clearer AC.
- `evidence-synthesis` — synthesis surfaces the words users actually use, which become aliases in the glossary.
- Prior art: Domain-Driven Design's "Ubiquitous Language" (Eric Evans) — one shared vocabulary between domain experts and engineers.
