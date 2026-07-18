---
user: chet-bellows
updated: 2026-07-18T17:58:46Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Session goal

> Session opened with — <command-message>deliver</command-message>
> <command-name>/deliver</command-name>
> <command-args>chat-canonical-research
> 
> First add a dedicated measurable EARS Acceptance Criteria section from the existing “Done when” and Validation requirements. Then deliver through audit and hero spec verify before stopping.</command-args>
> Route this delivery request to the `feature-delivery-lead` agent for ex…

## Last user ask

> Route this delivery request to the `feature-delivery-lead` agent for execution.
> 
> Be the `feature-delivery-lead` agent. Load the `context-injection` skill before starting.
> 
> **Initiative guard.** If the requested target resolves to a `type: initiative`,
> do NOT deliver one child. An initiative is a parent — running the whole thing
> autonomously is `/drive`, not `/deliver`. Offer the upgrade: "That's…

## Suggested next prompt

> chat-slim-to-basic-research-seed is delivered/verified. Open follow-up ('Thing 2'): reconcile hero-code's two contradictory staging stories — its crates/hero-core/build.rs DOES stage domains/chat (6 commands), but its desktop-app extract-hero-content.sh (enumerating 'hero domain list') does NOT serve chat (runs engineering). Determines whether chat needs a go:embed to actually be served by the app. Peer hero-code to resolve before any go:embed change.

## Recent reflections

- Integration config should key connections by stable IDs, not provider names: provider-keyed objects cannot represent two Jira projects, while explicit default/roles separate definition from selection.
- Jira import already supports tracker.token in hero.local.json; the token_env-only error is false, and local connect can copy the resolved token into committed hero.json.
- mkdocs-material's theme feature for integrated left-nav TOC is toc.integrate, not navigation.integrate (that flag doesn't exist in 9.7.x) — check the compiled template partials before trusting a spec's exact feature-flag name

## Tried and failed (this session)

Nothing this session.

## Your recent activity

Run `git log --oneline --author=<you> -10` for recent commits.

