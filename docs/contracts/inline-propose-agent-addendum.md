# Inline-Propose Agent Prompt Addendum

This is the canonical system-prompt fragment appended to any agent
running under `--inline-propose`. The runner injects it automatically
(see `internal/runner/runner.go` — constant `InlineProposeAddendum`).

When authoring agents that need bespoke wording, copy this verbatim
and adapt the surrounding instructions. The wire shape itself is
non-negotiable; it is pinned by `docs/contracts/inline-propose-v1.md`.

---

## Output mode: inline-propose

You are running under `--inline-propose`. Do NOT write directly to
spec files for the artifact content you would normally author.
Instead, emit each unit of proposed content as a single line on
stdout in this exact shape:

```
HERO-PROPOSAL: {"schema_version":"1.0","proposal_id":"p-<6-hex>","batch_id":"b-<6-hex>","session_id":"<session>","agent":"<your-slug>","target":{"spec_slug":"<slug>","anchor":{"kind":"section","value":"<section-id>","position":"append"}},"content":{"format":"markdown","body":"..."},"rationale":"..."}
```

Rules:
- One proposal per line, ASCII prefix `HERO-PROPOSAL: ` (with a
  trailing space).
- The line MUST be valid JSON after the prefix. No multi-line JSON.
- `proposal_id` is unique within the session. `batch_id` groups
  proposals that should bulk-accept together (one batch_id per
  "draft N AC" invocation).
- `target.anchor.kind` is one of: `frontmatter`, `section`,
  `heading`, `list_item`, `free`. `target.anchor.value` identifies
  the anchor within the spec (e.g. the section heading slug or the
  frontmatter field name).
- `content.format` is `markdown` unless the target anchor is
  `frontmatter` (then `yaml`) or a free-text label (then `text`).
- `session_id` comes from the `HERO_SESSION_ID` environment
  variable. If unset, the wrapping shim will fill it in.
- After emitting your proposals, finish your turn. The dashboard
  surfaces accept / edit / reject controls; you do NOT apply the
  changes yourself. Disk writes for proposed content are reserved
  for the user's accept action.

Non-proposal stdout (status messages, progress, etc.) is passed
through to the user and is fine to emit.
