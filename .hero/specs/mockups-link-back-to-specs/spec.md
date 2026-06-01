---
title: Mockups link back to specs
slug: mockups-link-back-to-specs
type: feature
status: completed
tags: [mockups, deliver, spec-format, workflow]
created: 2026-05-20
domain: engineering
completed_at: 2026-05-21T05:17:42Z
---
# Mockups link back to specs

## Kickoff

Delivered. `/mock` now writes a `## Mockups` entry into the originating
spec (or `_adhoc/` for free-text), `/deliver` surfaces
`.hero/mocks/{slug}/` paths to the engineer pre-flight, and the
`spec-format` skill documents the section as part of feature and bug
templates. Five files edited: mock.md, html-mockup-generation/SKILL.md,
deliver.md, spec-format/SKILL.md, ui-designer.md (step 6 added for
reliability). Pick up at: smoke-test by running `/mock` against an
in-flight spec and confirming the `## Mockups` section appears in the
spec body with today's date.

## Context

`/mock` generates self-contained HTML mockups and saves them to
`.hero/mocks/{slug}/index.html` (verified in
`domains/engineering/commands/mock.md` line 10 and
`domains/engineering/agents/ui-designer.md` line 26). The path is already
slug-keyed, so the mockup-to-spec association exists implicitly on disk.

What doesn't exist: any reference back from the spec to its mockup, and
any awareness in `/deliver` that a mockup might be sitting in
`.hero/mocks/{slug}/`. The spec body has no `## Mockups` section,
`spec-format/SKILL.md` doesn't list one, and `deliver.md` never mentions
mockups. Result: a delivery agent reading a spec has no signal that a
visual reference was produced during design — the user has to remember
and point at it manually.

A grep across `.hero/` confirms only one existing match for `## Mockups`,
in `.hero/planning/initiatives/hero-surface-architecture/spec.md`, and
that usage is `### Mockups (per child spec)` describing a directory
layout convention for a different surface. It's a heading-3 inside an
initiative; the new top-level `## Mockups` section in feature/bug specs
does not collide.

## Goal

A spec produced by `/design` or `/diagnose` and then mocked with `/mock`
ends up with a top-level `## Mockups` section in the spec body that lists
each mockup with its path, date, and one-line description. When the same
spec is later delivered with `/deliver`, the delivery agent is told
upfront that `.hero/mocks/{slug}/` exists and given the file paths inside
it, even for mockups generated before this feature landed. No
auto-rendering, no gallery, no validation — just a written link in both
directions.

## Approach

Two coordinated edits, neither of which requires code.

**Write-back from `/mock`.** Extend the `/mock` routing instruction so
the `ui-designer` agent, after saving the mockup, opens the originating
spec at `.hero/planning/{features|bugs}/{slug}/spec.md` (planning first;
fall back to `.hero/specs/{slug}/spec.md` if the spec has already been
archived) and ensures a `## Mockups` section exists with an entry for
this mockup. Format:

```markdown
## Mockups

- [{Mockup name}](.hero/mocks/{slug}/index.html) — YYYY-MM-DD — one-line description of what the mockup shows
```

Multiple mockups per spec are supported as additional list items. The
"Mockup name" defaults to a humanized slug (e.g. `Hero landing page`) but
the agent can use a more specific name when iterating produces a variant
(e.g. `Hero landing page — dense variant`). The relative path is written
verbatim so it works both as a clickable link in markdown renderers and
as a copy-paste-able file path.

**`--iterate` semantics.** When `/mock --iterate` modifies an existing
mockup at the same `index.html` path, the agent updates the matching
entry's date and (only if the iteration meaningfully changed what's
depicted) its description, rather than appending a duplicate row. Match
by path. If the iteration produces a new file under
`.hero/mocks/{slug}/` (e.g. `dense.html`), append a new entry.

**Body-only, no frontmatter field.** The Mockups section lives in the
spec body, not in frontmatter. Tradeoff considered: a frontmatter array
(`mockups: [.hero/mocks/foo/index.html]`) would be easier to parse
mechanically and surface in `hero list` output. We're choosing the body
section because (a) the delivery agent reads the spec body anyway, so a
visible section is more discoverable than a frontmatter field; (b) the
description and date carry signal a path alone doesn't; (c) we have no
near-term consumer that needs structured mockup metadata. If a future
feature wants programmatic access (e.g. a `hero mocks list` command), we
can add a frontmatter field then without losing the body section.

**Free-text `/mock` calls.** When `/mock` is invoked with free-text
rather than a spec slug, there is no spec to write back to. Save the
mockup under `.hero/mocks/_adhoc/{kebab-case-summary}/index.html` and
skip the write-back step entirely. This keeps the `.hero/mocks/` root
clean (slug-keyed dirs match real specs; `_adhoc/` clearly doesn't) and
avoids the ambiguity of inventing a fake slug.

**Auto-load from `/deliver`.** Add a pre-flight step to the delivery
flow: before kicking off the engineer, check whether
`.hero/mocks/{slug}/` exists. If it does, list the files inside it (paths
only, no rendering) in the kickoff context handed to the engineer
agent — something like "Mockups available for this spec:
`.hero/mocks/{slug}/index.html` — read if visual reference helps." The
agent decides whether to open the HTML. This is intentionally passive:
it picks up orphan mockups generated before the write-back behavior
existed, and it's belt-and-suspenders for cases where someone drops
files directly into `.hero/mocks/{slug}/` without going through `/mock`.

**Document in `spec-format`.** The `## Mockups` section becomes a
recognized (optional) section in the feature spec template and bug spec
template, with a one-line description and the canonical entry format.
This makes it real shape, not ad-hoc convention.

## Changes

1. Update `domains/engineering/commands/mock.md`
   - After the existing step 4 ("Save it to `.hero/mocks/{slug}/index.html`"),
     add step 5: "If a spec slug was provided, append (or update on
     `--iterate`) a `## Mockups` entry in the originating spec's body
     at `.hero/planning/{features|bugs}/{slug}/spec.md` (or
     `.hero/specs/{slug}/spec.md` if already archived)."
   - Add a short note describing the entry format (link + date + description)
   - Add a sentence covering the free-text case: save under
     `.hero/mocks/_adhoc/{summary-slug}/index.html` and skip write-back.

2. Update `domains/engineering/skills/html-mockup-generation/SKILL.md`
   - Add a new top-level section, `## Spec write-back`, after the existing
     "What NOT to Do" section. Document:
     - The exact entry format: `- [{Name}](.hero/mocks/{slug}/index.html) — YYYY-MM-DD — description`
     - That the section is `## Mockups` (heading-2), appended near the
       end of the spec body if not already present
     - The `--iterate` rule: match by path, update date in place, only
       rewrite description if the iteration meaningfully changed what's
       shown
     - The free-text exception: no write-back, save under
       `.hero/mocks/_adhoc/`

3. Update `domains/engineering/commands/deliver.md`
   - In the "Single spec mode" section (around line 86), add a pre-flight
     bullet before the delivery loop: "Check for `.hero/mocks/{slug}/` —
     if it exists, surface the file paths inside it in the kickoff
     context so the engineer knows a visual reference is available.
     Don't read or render the HTML; let the engineer open it if
     useful." Apply the same check in batch and queue modes (one
     sentence near the per-spec loop step is sufficient — don't
     duplicate the full instruction).

4. Update `domains/engineering/skills/spec-format/SKILL.md`
   - In the feature spec template (around line 38), add an optional
     `## Mockups` section between `## Changes` and `## Boundaries`,
     with a one-line description and a placeholder entry.
   - Mirror the same optional section in the bug spec template (around
     line 82) between `## Changes` and `## Boundaries`.
   - Add a short paragraph after the templates documenting the section:
     - Format: `- [Name](.hero/mocks/{slug}/index.html) — YYYY-MM-DD — description`
     - One entry per mockup; multiple mockups supported
     - Auto-populated by `/mock` against a spec slug
     - Optional — omit if no mockup was produced

5. Verify `domains/engineering/agents/ui-designer.md` does not need
   parallel edits
   - It currently delegates the "what to do" instructions to the loaded
     `html-mockup-generation` skill (line 49). With change #2 covering
     the skill, the agent will pick up the new write-back step
     automatically. If review shows the agent's "Your approach" steps
     (lines 8-26) need an explicit numbered step for write-back to be
     reliably triggered, add it as step 6 ("Update the originating
     spec's `## Mockups` section"); otherwise leave the file alone.

## Acceptance Criteria

- WHEN `/mock <spec-slug>` runs against an existing spec THE SYSTEM SHALL
  append a `## Mockups` entry to that spec's body listing the mockup
  path, today's date, and a one-line description.
- WHEN `/mock <spec-slug> --iterate` runs against a spec that already has
  an entry for the same mockup path THE SYSTEM SHALL update the existing
  entry's date in place rather than appending a duplicate row.
- IF `/mock` is invoked with free-text and no spec slug THEN THE SYSTEM
  SHALL save the mockup under `.hero/mocks/_adhoc/{summary}/index.html`
  and skip the spec write-back step.
- WHEN `/deliver <spec-slug>` runs and `.hero/mocks/{slug}/` exists THE
  SYSTEM SHALL include the mockup file paths in the kickoff context
  handed to the implementing agent.
- THE SYSTEM SHALL document the `## Mockups` section as part of the
  feature and bug spec templates in the `spec-format` skill.

## Boundaries

Out of scope:

- Building a mockup gallery, index page, or browse UI.
- Validating that a mockup actually matches the spec it's linked from.
- Versioning mockups separately in git (e.g. `index-v1.html`,
  `index-v2.html`). Iteration overwrites in place; git history is the
  version log.
- Auto-screenshotting mockups or embedding images in the spec.
- A `hero mocks` CLI subcommand or any frontmatter field for mockups.
- Migrating existing orphan mockups under `.hero/mocks/` to add their
  spec links retroactively — change #2 (`/deliver` auto-load) covers
  them on the read side, and that's sufficient.

## Risks

- **Collision with the existing `### Mockups` heading in
  `.hero/planning/initiatives/hero-surface-architecture/spec.md`.**
  Verified: that occurrence is heading-3 inside a different section, and
  this feature uses heading-2 in feature/bug specs. No collision in
  practice, but worth a re-check during implementation.
- **Spec archive path resolution.** If a spec has been archived from
  `.hero/planning/` to `.hero/specs/` between mockup generation and
  iteration, the write-back logic must check both locations. Documented
  in the mock.md change; verify the ui-designer agent actually follows
  the fallback.
- **Behavior depends on agents following written instructions.** This
  feature is documentation-only; there's no enforcement. If the
  `ui-designer` agent skips the write-back step, no error fires. Manual
  smoke test after rollout (run `/mock` against an active spec,
  inspect the spec) is the verification.
- **Free-text `_adhoc` directory could grow unboundedly.** Acceptable —
  same lifecycle as any other `.hero/mocks/{slug}/` directory; pruning
  is a separate concern.

## Validation

- Manual smoke test: run `/mock hero-landing-page` (an active in-flight
  feature spec) and confirm the spec at
  `.hero/planning/features/hero-landing-page/spec.md` gets a `## Mockups`
  section with one entry.
- Iterate test: run `/mock hero-landing-page --iterate "make it denser"`
  and confirm the existing entry's date updates rather than a duplicate
  being appended.
- Free-text test: run `/mock "a settings page with two columns"` and
  confirm the mockup lands under `.hero/mocks/_adhoc/` and no spec is
  touched.
- Deliver auto-load test: run `/deliver hero-landing-page` (or any spec
  with a pre-existing `.hero/mocks/{slug}/` directory) and confirm the
  kickoff context surfaced to the engineer mentions the mockup path.
- Read the four edited markdown files end-to-end to confirm the new
  instructions are coherent and don't contradict surrounding guidance.
