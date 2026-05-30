---
description: Generate a visual mockup from a spec or free-text description. Supports HTML (web) and native renderers (SwiftUI, more coming).
---
Route this mockup request to the `ui-designer` agent.

## Renderer Selection

Before routing, determine which renderer to use:

1. **Explicit override wins:** If the user passed `--renderer=html` or `--renderer=swiftui`, use that renderer.
2. **Auto-detect from stack:** Check the project root for stack signals:
   - `.swift` files, `Package.swift`, `*.xcodeproj`, `*.xcworkspace` → **SwiftUI renderer**
   - Everything else → **HTML renderer**
3. **Toolchain gate:** If SwiftUI is selected, verify `swiftc` is available. If not, fall back to HTML and note: "SwiftUI renderer unavailable (swiftc not found), using HTML."
4. **Config override:** Check `hero.json` → `mockups.renderer` — if set to `"html"` or `"swiftui"`, that overrides auto-detection (but not explicit `--renderer` flags).

Tell the ui-designer which renderer to use and which skill to load:
- **HTML:** Load `html-mockup-generation` skill. Output: `.hero/mocks/{slug}/index.html`
- **SwiftUI:** Load `swiftui-mockup-renderer` skill. Output: `.hero/mocks/{slug}/MockView.swift` + `screenshot.png` + `screenshot-dark.png` + `index.html` (viewer)

## Workflow

The ui-designer will:
1. Read the spec (if a slug is provided) or parse the free-text description
2. Identify the key screens, components, and interactions needed
3. Generate the mockup using the selected renderer:
   - **HTML:** Self-contained `index.html` with embedded CSS and JS
   - **SwiftUI:** `MockView.swift` source → compile → capture PNG screenshots (light + dark)
4. Save to `.hero/mocks/{slug}/` (or `.hero/mocks/_adhoc/{summary-slug}/` for free-text)
5. If a spec slug was provided, append (or update on `--iterate`) a `## Mockups` entry in the originating spec at `.hero/planning/features/{slug}/spec.md` or `.hero/planning/bugs/{slug}/spec.md` (or `.hero/specs/{slug}/spec.md` if already archived). Entry format:
   - HTML: `- [{Name}](.hero/mocks/{slug}/index.html) — YYYY-MM-DD — one-line description`
   - SwiftUI: `- [{Name}](.hero/mocks/{slug}/screenshot.png) — YYYY-MM-DD — SwiftUI native render`

If the user provides `--iterate` feedback, read the existing mockup first and modify it based on the feedback rather than regenerating from scratch. On iterate, update the matching `## Mockups` entry's date in place rather than appending a duplicate.

**Output requirements (HTML renderer):**
- Single self-contained `index.html` — no external dependencies
- Professional, clean design with modern CSS (flexbox/grid)
- Basic interactivity where appropriate (tabs, modals, dropdowns)
- Responsive layout (works on mobile and desktop)
- HTML comment header with spec slug, date, and description

**Output requirements (SwiftUI renderer):**
- Single self-contained `MockView.swift` — no SPM dependencies
- Real SwiftUI components, SF Symbols, system colors
- Both light and dark mode screenshots captured
- Viewer `index.html` wrapping the PNGs with light/dark toggle

**Surface outputs:** At the end of your response, always list every file created or updated with clickable links so the user can find and preview them immediately. Format:

```
**Files created:**
- [Mockup name](.hero/mocks/{slug}/index.html)
- [Screenshot — light](.hero/mocks/{slug}/screenshot.png)
- [Screenshot — dark](.hero/mocks/{slug}/screenshot-dark.png)
- [SwiftUI source](.hero/mocks/{slug}/MockView.swift)
```

Request: $ARGUMENTS
