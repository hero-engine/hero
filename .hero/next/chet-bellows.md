---
user: chet-bellows
updated: 2026-07-16T07:37:09Z
repo: hero-engine/hero
---

# chet-bellows's handoff

## Last user ask

> In /Users/bwheeler/projects/hero-engine/repository/hero, `.agents/skills/` is committed to git but is stale and drifting from its source. Decide how to resolve it. Do NOT assume gitignoring is the answer — see the policy distinction below.
> 
> FACTS (verified):
> - `.agents/skills/<name>/SKILL.md` is genuinely where the Codex CLI loads repo-scoped skills (verified against openai/codex `codex-rs/core-…

## Suggested next prompt

> Today's install-integrity thread is fully closed: eraser fixed+guarded, detection shipped, MCP resolver fixed, codex-install-broken superseded. Branch fix/agents-md-erased-by-snapshot-pointer-writer has 6 commits ready to merge to main. Remaining small items: orphan-instruction-files fresh-clone false positive (chip pending), copilot un-inferable on fresh clones (targetLayouts legacy probe), .mcp.json same-principle relocation, and the cosmetic 'downgrade detected' version misread.

## Recent reflections

- Integration config should key connections by stable IDs, not provider names: provider-keyed objects cannot represent two Jira projects, while explicit default/roles separate definition from selection.
- Jira import already supports tracker.token in hero.local.json; the token_env-only error is false, and local connect can copy the resolved token into committed hero.json.
- mkdocs-material's theme feature for integrated left-nav TOC is toc.integrate, not navigation.integrate (that flag doesn't exist in 9.7.x) — check the compiled template partials before trusting a spec's exact feature-flag name

## Tried and failed (this session)

Nothing this session.

## Your recent activity

Run `git log --oneline --author=<you> -10` for recent commits.

