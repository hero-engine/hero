---
title: Hero serve grammar pivot — why a web app, not a desktop pane chassis
type: note
status: active
created: 2026-05-17
tags: [hero-serve, surface, ui, layout, decision-history, pm-qa-boundary]
relates-to:
  - hero-surface-architecture
  - hero-surface-shell
  - hero-surface-deployment-and-rendering
  - hero-pm
  - hero-qa
---

## What this captures

The May 2026 design pivot for `hero serve`'s UI grammar — from a
desktop-pane chassis (inherited from PM/QA mockups) to a web-app
companion grammar. This note exists because the specs and mocks were
rewritten in flight and the *why* would otherwise be lost.

## The mistake we made first

When designing the Hero Surface Architecture initiative, every home
spec and the initial round of signature mockups inherited the
[hero-pm/spec.md](../../planning/features/hero-pm/spec.md) §"Dashboard
layout" grammar — the locked layout used by the PM and QA domain
packs:

```
chrome (top bar, role pills)
240px left nav │ flexible center (VS Code tab strip + scroll + bottom verb strip + chat input) │ 380px right ambient panel
```

That grammar exists because PM/QA are **desktop apps** for end users
who live inside the tool — they need persistent inventory in the left
rail, persistent verbs in the bottom strip, persistent ambient
context in the right panel.

We inherited it because the PM mocks were polished and the QA spec
explicitly said *"do not redesign this in this pack."* The
brand DNA (bolt logo, hero-blue, Inter, status chips) felt like the
visual language to share. We confused brand DNA with chrome.

## The correction

The user pushed back hard: hero serve is **not** another flavor of
the desktop tooling. It is a **web app companion to the CLI** —
something you open in a browser tab to deep-dive into your projects
beyond what one CLI command can show. You scroll, you click into
things, you close the tab when done. It is NOT a workspace you live
in all day.

PM and QA, by contrast, are end-user desktop apps. They are **a
different product**, out of scope for hero serve.

## The rules that resulted

Codified in the new
[hero-surface-deployment-and-rendering](../../planning/features/hero-surface-deployment-and-rendering/spec.md)
rendering section and the
[hero-surface-shell](../../planning/features/hero-surface-shell/spec.md)
chrome design:

1. **Only fixed chrome is a slim top nav** (~56px). Hero brand + top-
   nav text-link tabs (`Now`, `Work`, `Knowledge`, `Agents`, `People`)
   with hero-blue underline on active + ⌘K pill + avatar.
2. **Each home is a page** at `/now`, `/work`, … No VS Code-style tab
   strip. Per-item things (a spec, a session) are routes, not tabs.
3. **No fixed left/right rails. No fixed bottom strip.** Actions live
   inline with the content they affect. Context lives in page
   sections or in slide-overs invoked on demand.
4. **Scrolling content, max-width ~1200px centered**, ~32px
   horizontal padding. Sections stack vertically with ~48–56px
   breathing room.
5. **Optional sub-nav row** of text-link tabs below the top nav for
   in-home navigation (used by Knowledge / Agents / People).
6. **Brand DNA shared with PM/QA**: bolt logo, hero-blue palette,
   Inter typography, status chip aesthetics. **Chrome is not
   shared.** The Now mock
   ([01-now-default.html](../../planning/features/hero-now-home/mockups/01-now-default.html))
   is the visual source of truth for hero serve grammar.
7. **No view registry, no pack abstraction.** Earlier drafts assumed
   PM/QA would register as packs alongside engineering. Since PM/QA
   are different products, engineering is the only "pack" — which
   means it isn't a pack at all, it's just the hero serve UI. If a
   future Hero web companion ever wants to ride here, we add
   extensibility then.

## How this affected delivery

The first delivery attempt for `hero-surface-shell` was paused at the
supervised-mode pre-flight check because the spec still described the
desktop grammar. All seven hero-surface specs (initiative, decision,
shell, 5 homes) were rewritten end-to-end against the new web-app
mocks. The mocks themselves had been redone earlier; the specs caught
up in a single coordinated pass.

## What to remember

- Brand DNA is not chrome.
- "Web app companion to the CLI" is the product positioning. If a
  design choice would make hero serve feel like a workspace you live
  in, it's probably wrong for this surface.
- PM/QA are different products. Stop inheriting their layout.
- The Now mock is the source of truth. When in doubt, open it.
