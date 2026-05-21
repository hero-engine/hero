---
description: Generate a visual HTML mockup from a spec or free-text description.
---
Route this mockup request to the `ui-designer` agent.

The ui-designer will:
1. Read the spec (if a slug is provided) or parse the free-text description
2. Identify the key screens, components, and interactions needed
3. Generate a self-contained HTML mockup with embedded CSS and inline JS
4. Save it to `.hero/mocks/{slug}/index.html` (or `.hero/mocks/_adhoc/{summary-slug}/index.html` when invoked with free-text and no spec slug)
5. If a spec slug was provided, append (or update on `--iterate`) a `## Mockups` entry in the originating spec at `.hero/planning/features/{slug}/spec.md` or `.hero/planning/bugs/{slug}/spec.md` (or `.hero/specs/{slug}/spec.md` if already archived). Entry format: `- [{Name}](.hero/mocks/{slug}/index.html) — YYYY-MM-DD — one-line description`. Free-text requests skip this step.

If the user provides `--iterate` feedback, read the existing mockup first and modify it based on the feedback rather than regenerating from scratch. On iterate, update the matching `## Mockups` entry's date in place rather than appending a duplicate.

**Output requirements:**
- Single self-contained `index.html` — no external dependencies
- Professional, clean design with modern CSS (flexbox/grid)
- Basic interactivity where appropriate (tabs, modals, dropdowns)
- Responsive layout (works on mobile and desktop)
- HTML comment header with spec slug, date, and description

Load the `html-mockup-generation` skill for detailed design guidance.

Request: $ARGUMENTS
