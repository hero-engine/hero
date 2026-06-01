---
title: "Design Mockups — Visual Prototyping from Specs"
slug: design-mockups
type: feature
status: completed
tags: [mock, design, ux, html, prototype]
created: 2026-04-12
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Progress

- [x] Spec written
- [x] Agent command (`commands/mock.md`)
- [x] Skill for mockup generation (`skills/html-mockup-generation.md`)
- [x] CLI command (`hero mock`) — list, open, serve existing mocks (6 tests)
- [x] Agent for UI/UX design work (`agents/ui-designer.md`)
- [x] Directory convention established (`.hero/mocks/{slug}/`)
- [x] Tests for CLI command (6 tests, all passing)
- [x] Integration with `/do` router (mock/prototype/wireframe keywords)
- [x] `hero init` creates `mocks/` directory
- [x] `config.MocksDir()` helper added

Started: 2026-04-12
Completed: 2026-04-12

## Problem

Specs describe *what* to build, but the team can't visualize the UX until code is written. By then, it's expensive to change. We need a way to generate shareable visual mockups from a spec or free-text description *before* implementation starts, so stakeholders can review and iterate on the design cheaply.

## Goal

`/mock` generates a self-contained HTML mockup from a spec or description. The mockup is a clickable prototype viewable in any browser — no accounts, no tooling, no dependencies. Mocks are saved into the project's `.hero/mocks/` directory and can be listed, opened, and iterated on.

## Approach

### Agent Command (`/mock`)

The `/mock` command is an agent-driven workflow. The AI reads the spec (or free-text description), reasons about the UI/UX, and generates a self-contained HTML file.

**Usage patterns:**
- `/mock auth-login` — generate mockup from the auth-login spec
- `/mock "admin dashboard with usage charts, user table, and export button"` — generate from description
- `/mock auth-login --iterate "make the sidebar collapsible"` — refine an existing mock

**Output format:** Self-contained HTML with embedded CSS and inline SVG/icons. No external dependencies. The file should:
- Be a single `index.html` that opens in any browser
- Use modern CSS (flexbox/grid) for layout
- Include basic interactivity (tab switching, modals, dropdowns) via inline JS
- Look professional — use a clean design system (neutral colors, good typography)
- Include a header comment with the spec slug, generation date, and description
- Be responsive (mobile + desktop)

**Output location:** `.hero/mocks/{slug}/index.html`

For free-text descriptions without a spec slug, derive a slug from the description (e.g., "admin-dashboard").

### CLI Command (`hero mock`)

Minimal Go CLI for managing existing mocks:

```
hero mock --list              # list all mocks with slug, date, path
hero mock --open {slug}       # open mock in default browser
hero mock --serve             # serve mocks directory on localhost for easy sharing
```

The `--serve` option starts a simple file server on a random port, serving `.hero/mocks/` so the user can share the URL with teammates on the same network.

### UI Designer Agent

A new `agents/ui-designer.md` agent specializes in:
- Translating feature descriptions into visual layouts
- Choosing appropriate UI patterns (tables, forms, cards, dashboards, wizards)
- Generating clean, professional HTML/CSS
- Iterating on designs based on feedback

The `/mock` command delegates to this agent.

### Skill

A `skills/html-mockup-generation.md` skill provides detailed guidance on:
- HTML structure conventions for mockups
- CSS design system (colors, spacing, typography)
- Common UI patterns and when to use them
- Interactivity patterns (what JS to include)
- Accessibility basics
- File naming and organization

### Integration with `/design`

The `/design` command's output (the spec) can reference mockups. After writing a spec, the delivery lead can suggest: "Run `/mock {slug}` to generate a visual prototype before implementation."

## Directory Convention

```
.hero/
  mocks/
    auth-login/
      index.html          # the mockup
    admin-dashboard/
      index.html
```

Mocks directory should be added to the default `.hero/` structure created by `hero init`. The directory should be git-tracked (mockups are project artifacts worth versioning).

## Changes

- `commands/mock.md` — new agent command
- `skills/html-mockup-generation.md` — new skill
- `agents/ui-designer.md` — new agent
- `internal/cli/mock.go` — new CLI command (list, open, serve)
- `internal/cli/mock_test.go` — tests
- `internal/cli/root.go` — register mockCmd
- `internal/cli/init.go` — add `mocks/` to init scaffold

## Boundaries

- The AI generates the mockup — the Go binary does NOT render anything
- No Figma integration (Figma API is read/edit, not generate-from-scratch)
- No image generation — HTML/CSS only
- No component library downloads — everything is inline/self-contained
- Cross-project mock sharing is a cloud feature (future)

## Validation

- `/mock "login page with email and password"` generates a valid HTML file in `.hero/mocks/login-page/index.html`
- `/mock auth-login` reads the spec and generates a contextually appropriate mockup
- `hero mock --list` shows all mocks
- `hero mock --open login-page` opens in browser
- Generated HTML is valid, responsive, and looks professional
- Iterating with `/mock auth-login --iterate "add forgot password link"` updates the existing mock
