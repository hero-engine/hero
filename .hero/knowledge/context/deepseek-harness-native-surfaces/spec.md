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

## Desktop app and plugin model (added 2026-09-30)

These are source findings at the same commit, plus the user's installed DeepSeek Harness desktop app 0.2.0-rc.2.

**Desktop app**
- It boots profile `desktop` (bundles `dsh-base` + `dsh-web-app`) through `apps/desktop-host/src/index.ts` with `patchFiles: []`. `--patch` overlays therefore never load there.
- Every profile composes bundle layers → the app-managed profile patch (`~/.dsh/profiles/<p>/cordis.patch.yml`) → the home patch `$DSH_HOME/cordis.patch.yml` → overlays (`packages/boot/app-boot/src/profile-context.ts` `readProfilePatches`).
- The desktop app never writes the home patch; crash recovery reports `homePatch: 'unchanged'`.
- GUI launches do not inherit the shell PATH.

**MCP client**
- One profile-level plugin per server, spawned once with a fixed `cwd` (default: the host process cwd), not per session.
- `serverName` must be unique, `[A-Za-z0-9_-]{1,32}`, and namespaces tools as `mcp__<serverName>__*`.

**Plugin model**
- Every product feature is a Cordis plugin on a documented extension point (`docs/cookbook/extension-cookbook.md`, "feature → mechanism map"; `docs/capability-seams.md`):
  - skills: `ctx.skills` provider, e.g. `skill-office`
  - presets: `ctx.agentPresets`, declared in YAML
  - human slash commands: `ctx.commands`
  - instruction sections: `ctx.systemPrompt.section()`
  - native tools: `ctx.tools.register()`
  - lifecycle hooks: `agent/created`, `agent/turn-stopping`, `tools/pre|post-execute`
  - compaction: `ctx.compaction`
- Native tools see the calling session's workspace via `exec.agent.session.header.cwd` (e.g. `packages/lsp/tool-lsp/src/session-cwd.ts`), so a plugin can route per session where an MCP entry cannot.
- `dsh-hooks-claude-code` / `dsh-hooks-codex` already run existing Claude Code / Codex `hooks.json` hooks.

**Distribution**
- Third-party code ships as a **bundle**: an npm package or GitHub repo whose `package.json` declares `dsh.bundle.patch` → `cordis.patch.yml`.
- Bundles install per profile through the plugin manager: the desktop **Plugins** page, `dsh plugin --profile <p> add <pkg>`, or an agent tool.
