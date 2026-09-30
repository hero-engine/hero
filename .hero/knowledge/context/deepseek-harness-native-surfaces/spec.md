---
title: DeepSeek harness native discovery and activation surfaces
slug: deepseek-harness-native-surfaces
type: context
status: active
domain: engineering
created: 2026-09-28
tags: [deepseek, harness, install]
relations:
  - target: deepseek-harness-install-target
    kind: related
---

# DeepSeek harness native discovery and activation surfaces

Delivery validated these contracts against the real pinned loaders and MCP client with `scripts/deepseek-compatibility.mjs`: engineering, PM, and QA all load their generated skills, compose the headless profile with Hero's overlay, and execute `hero_status` without a model call. An explicit `--workspace` overlay also resolves its declared root when launched outside that project. Canonical PM agent descriptions contain unquoted colons, so the DeepSeek renderer must safely normalize source scalars and emit valid YAML rather than copy permissive source frontmatter directly.

Inspected `deepseek-ai/deepseek-harness` at commit `477b4f420553e8a52c2fbccc464d7561b239c443`; these are source findings, not runtime validation. The `dsh` CLI reads `AGENTS.md` and `CLAUDE.md`, and its skill filesystem discovers `.dsh/skills` and `.agents/skills` at the first ancestor `.git` root (cwd if none), plus user roots. `$DSH_HOME` defaults to `~/.dsh`; blank values default, `~/` expands, and relative values resolve against cwd. The agent-preset registry does not scan Markdown agent directories: roles require native plugin declarations or an explicitly described instruction-skill fallback. MCP is a Cordis `@deepseek-ai/dsh-mcp-client` plugin entry, and a project overlay only activates via `--patch`; `.mcp.json` and a bare `.dsh` config file do not provide that activation. Its stdio child binds its launch workspace, so a globally reusable overlay does not create per-session multi-project routing. The skill tool is named `skill` with a `name` argument; skill metadata uses `user-invocable` and `disable-model-invocation`, not camel-case aliases. Recheck the pinned loader paths before implementation: `packages/context/agent-instructions/src/config.ts`, `packages/skill/skill-filesystem/src/index.ts`, `packages/preset/agent-preset-registry/README.md`, `packages/mcp/mcp-client/src/index.ts`, and `apps/cli/config/examples/mcp-memory/mcp-reference-memory.cordis.yml`.
