---
type: convention
status: draft
scope: ["internal/install/mcp.go"]
tags: [install, mcp, portability, config, harness]
relations:
  - target: codex-mcp-binary-path-resolution
    kind: introduced-by
---

# MCP Config: Portable Command, Project Layer, User's File Is Theirs

## Pattern

When Hero wires its MCP server into a harness config, three rules hold for
every target (cursor, claude, opencode, codex):

1. **Portable command, not an absolute path.** Write `command = "hero"`,
   never the resolved path of the installing binary. MCP config files live
   in project-root files that travel with the repo — a teammate who clones
   gets working wiring without re-running install. An absolute path is
   machine-specific and breaks the instant the file reaches another
   machine. Anyone who can use Hero's MCP server already has `hero`
   installed, so the harness resolves it from PATH at launch.

2. **Project layer, not the machine-global layer.** An MCP server serves a
   *project* (Hero resolves the workspace from the session's cwd), so its
   wiring belongs in that project's config file — `.codex/config.toml`,
   `.mcp.json`, `.cursor/mcp.json`, `opencode.json` — not in the user's
   machine-wide config. A global entry fires in *every* project on the
   machine, including non-Hero ones, where it launches a spurious server or
   binds to an unrelated ancestor workspace.

3. **The user's own config file is theirs.** `.codex/config.toml` (and
   `.mcp.json`) hold the user's model, approval, plugin, and other-MCP
   settings alongside Hero's block. Hero owns only its marked span
   (`# hero:managed` … `# end:hero:managed`). Never gitignore the whole
   file to hide Hero's few lines, and never write Hero's block into the
   user's *personal global* config (`~/.codex/config.toml`) — that inverts
   ownership, putting our content in their space.

## Why this exists

`codex-mcp-binary-path-resolution` first shipped the opposite of rules 1 and
2: it resolved the installing binary via `os.Executable()` and wrote that
absolute path into the machine-local User layer (`~/.codex/config.toml`),
reasoning that a machine-specific path shouldn't sit in a tracked project
file. That solved the wrong half. The real fix is to not write a
machine-specific path at all.

The tell was in the code the whole time: every writer's doc comment already
described `"command": "hero"` while the code interpolated an absolute path.
The portable form was the original intent; the code had drifted.

The User-layer version also produced a vivid demonstration of rule 3 — it
appended Hero's block to a real `~/.codex/config.toml` that held the user's
model, ~8 plugins, a dozen trusted projects, and other MCP servers. Our four
lines did not belong in that file.

## Rule

> MCP wiring is portable (`command = "hero"`) and lives in the **project's**
> config layer. The user's own config files — project or global — are theirs;
> Hero writes only its marked span and picks the *portable* value so the span
> travels.

## Residual risk

Portable `command = "hero"` depends on `hero` being resolvable on the PATH
the harness launches with. Two cases:

- **Wrong-hero:** more than one `hero` on PATH, and the stale one wins. This
  is what `hero doctor` diagnoses (it reports which binary actually resolves).
- **GUI-launched harness with no PATH:** a desktop-launched harness may not
  inherit the login shell's PATH, so `hero` may not resolve at all. For Codex
  specifically this is not a problem — its MCP launcher inherits PATH
  (verified in openai/codex `create_env_for_mcp_server`). For other harnesses
  it is a known tradeoff accepted in favor of portability.

An absolute path would trade these away for non-portability, which is the
worse failure in a file meant to travel. See [[reference_hero_binary_build_target]]
for the related "which hero is on PATH" hazard that bit this repo's own dev loop.
