---
title: "Native Mockup Rendering — Real Platform UI from /mock"
type: feature
status: delivering
priority: p2
---

## Goal

When `/mock` targets a native app (SwiftUI, Compose, Flutter, etc.), generate actual platform UI code, compile and run it headlessly, capture a screenshot of the real rendered output, and save both the source and the PNG. HTML mockups remain the default for web projects and the fallback when native toolchains aren't available.

## Kickoff

Verify native-mockup-rendering delivery. All files are implemented: `swiftui-mockup-renderer` skill (domain + .claude overlay), mock command updated with renderer dispatch, ui-designer agent updated with SwiftUI pipeline, stack-detection has Swift markers, `mock.go` shows `[native]`/`[html]` tags with passing tests. Pick up at: end-to-end test in a real Swift project to confirm the full `swiftc` → `ImageRenderer` → PNG pipeline works. Spec at `.hero/planning/features/native-mockup-rendering/spec.md`.

## Problem

The current `/mock` command always generates self-contained HTML. For native app projects (iOS, Android, desktop), this creates a fidelity gap:

- **System controls look wrong.** HTML can't render real `NSButton`, SwiftUI `Toggle`, Material `Switch`, or platform navigation bars. Designers and PMs see a web approximation, not the real thing.
- **Typography and spacing diverge.** SF Pro, system font weights, platform-specific metrics, safe-area insets, and Dynamic Type can't be replicated in CSS.
- **Platform behaviors are missing.** Blur effects, vibrancy, haptic-implied affordances, platform animation curves — none of these exist in an HTML mockup.
- **Review friction.** Stakeholders reviewing an HTML mock of a native app must mentally translate "what it will actually look like," which delays approval and causes surprise in real builds.

The user has proven this flow works in another project: generate SwiftUI → compile → capture PNG. Hero should formalize this as a renderer pipeline so any native stack can plug in.

## Design

### Architecture: Renderer Dispatch

The core idea is a **renderer abstraction** that the mock command selects based on the project stack. Each renderer is a skill that knows how to generate source, compile, run, and capture.

```
/mock "settings screen for our iOS app"
  │
  ├─ stack detection → Swift project detected
  ├─ renderer dispatch → selects swiftui-mockup-renderer
  ├─ ui-designer agent generates SwiftUI source
  ├─ renderer compiles + launches + captures PNG
  └─ saves: .hero/mocks/{slug}/
       ├─ MockView.swift  (source)
       ├─ screenshot.png  (rendered output)
       └─ index.html      (optional — lightweight viewer wrapping the PNG)
```

### Renderer Selection

Detection uses the existing `stack-detection` skill signals plus new root markers:

| Signal | Renderer |
|---|---|
| `.swift` files, `Package.swift`, `*.xcodeproj`, `*.xcworkspace` | `swiftui-mockup-renderer` |
| `.kt` files, `build.gradle.kts` with `compose` dependency | `compose-mockup-renderer` (future) |
| `.dart` files, `pubspec.yaml` | `flutter-mockup-renderer` (future) |
| Everything else / toolchain not available | `html-mockup-generation` (existing) |

The user can override with an explicit flag: `/mock --renderer=swiftui "..."` or `/mock --renderer=html "..."`.

### Renderer Skill Interface

Each native renderer skill must define:

1. **Source generation guidelines** — equivalent to `html-mockup-generation`'s CSS design system, but for the native framework. Component patterns, layout idioms, realistic data, what not to do.
2. **Build command** — how to compile the generated source into a runnable artifact.
3. **Run + capture command** — how to launch headlessly and capture a screenshot.
4. **Output manifest** — what files the renderer produces and where they go.

### SwiftUI Renderer (First Renderer)

**Skill:** `swiftui-mockup-renderer/SKILL.md`

**Source generation:**
- Single-file SwiftUI view (`MockView.swift`) with an `@main` App wrapper
- Uses system components: `NavigationStack`, `List`, `Form`, `Toggle`, `Button`, etc.
- SF Symbols for icons (no custom assets)
- System colors and dynamic type
- Both light and dark mode variants captured
- Preview-sized window (390×844 for iPhone 15 Pro, or explicit frame for macOS)

**Build + capture pipeline (macOS host):**
```bash
# Compile
swiftc MockView.swift \
  -framework SwiftUI \
  -framework AppKit \
  -o MockApp

# Run headlessly and capture (macOS approach)
# Use a small Swift helper that renders the view to an NSImage and writes PNG
swift render_to_png.swift MockView.swift --output screenshot.png --size 390x844
```

The practical approach: generate a self-contained Swift script that uses `ImageRenderer` (iOS 16+ / macOS 13+) or `NSHostingView` snapshot to render the SwiftUI view to a PNG without needing to launch a full app or simulator.

**Render helper pattern:**
```swift
import SwiftUI
import AppKit

// The mock view (generated per-request)
struct MockView: View { ... }

// Capture logic (bundled as part of the renderer)
let renderer = ImageRenderer(content: MockView().frame(width: 390, height: 844))
renderer.scale = 2.0 // @2x
if let image = renderer.nsImage {
    let data = image.tiffRepresentation!
    let bitmap = NSBitmapImageRep(data: data)!
    let png = bitmap.representation(using: .png, properties: [:])!
    try! png.write(to: URL(fileURLWithPath: "screenshot.png"))
}
```

This avoids needing Xcode, simulators, or a full app target — just `swiftc` from the Xcode command-line tools.

**Toolchain detection:** Before attempting native rendering, verify the toolchain exists:
- `which swiftc` must succeed
- macOS only (for SwiftUI rendering — cross-platform Swift doesn't include SwiftUI)
- If toolchain is missing, fall back to HTML with a note: "SwiftUI renderer unavailable (swiftc not found), falling back to HTML"

**Output structure:**
```
.hero/mocks/{slug}/
├── MockView.swift       # Generated SwiftUI source
├── render.swift         # Capture harness (can be reused)
├── screenshot.png       # Rendered output (@2x)
├── screenshot-dark.png  # Dark mode variant
└── index.html           # Lightweight viewer (embeds PNGs, shows both modes)
```

The `index.html` viewer is a simple page that displays the screenshots side-by-side (light/dark) with the source in a collapsible section. This ensures `hero mock --open` and `hero mock --serve` keep working.

### Mock Command Changes

Update `domains/engineering/commands/mock.md` and `.claude/commands/mock.md`:

1. Before routing to `ui-designer`, detect the project stack
2. If a native renderer is available and the toolchain is present, instruct the ui-designer to use the native renderer skill instead of `html-mockup-generation`
3. Pass `--renderer=<name>` through to the agent
4. The "Surface outputs" block now also lists the PNG screenshot with a link

### UI Designer Agent Changes

Update `domains/engineering/agents/ui-designer.md`:

1. Add a "Renderer selection" section that checks which renderer skill to load
2. When using a native renderer: generate platform source, invoke the build/capture pipeline via shell, verify the screenshot was produced, generate the viewer `index.html`
3. When iterating (`--iterate`): read the existing source file, modify it, re-run the capture pipeline
4. Error handling: if compilation fails, show the error, attempt to fix the source, retry once. If still failing, fall back to HTML with an explanation.

### CLI Changes (mock.go)

Minimal — the existing `hero mock --list` / `--open` / `--serve` commands work as-is because:
- The slug directory structure is unchanged
- `index.html` is still present (as a viewer)
- `--open` and `--serve` continue to serve `index.html`

One enhancement: `hero mock --list` should show a `[native]` or `[html]` tag per mock based on whether `screenshot.png` exists alongside `index.html`.

### hero.json Configuration

Add a `mockups` config section:

```json
{
  "mockups": {
    "renderer": "auto",
    "swiftui": {
      "device_frame": "iPhone 15 Pro",
      "size": "390x844",
      "scale": 2,
      "capture_dark_mode": true
    }
  }
}
```

- `renderer`: `"auto"` (detect from stack), `"html"` (force HTML), or a specific renderer name
- Per-renderer config for device sizing and capture options

## Scope

### In scope
- Renderer dispatch layer in mock command and ui-designer agent
- `swiftui-mockup-renderer` skill with source generation guidelines
- SwiftUI → PNG capture pipeline using `ImageRenderer` / `NSHostingView`
- Light and dark mode capture
- Lightweight HTML viewer that wraps the PNG screenshots
- Toolchain detection and graceful HTML fallback
- `--renderer` flag override
- `hero.json` mockups config section
- `hero mock --list` native/html tag

### Out of scope (future renderers)
- Jetpack Compose renderer (requires Android SDK / `composePreviews`)
- Flutter renderer (requires `flutter test --update-goldens` or similar)
- React Native renderer
- Simulator-based capture (full Xcode Simulator launch)
- Video/animation capture
- Interactive native previews

## Acceptance Criteria

WHEN the user runs `/mock` in a project containing `.swift` files or `Package.swift` THE SYSTEM SHALL select the SwiftUI renderer automatically and generate a `MockView.swift` + `screenshot.png` in `.hero/mocks/{slug}/`.

WHEN `swiftc` is not available on the host THE SYSTEM SHALL fall back to HTML rendering and inform the user why native rendering was skipped.

WHEN the user passes `--renderer=html` THE SYSTEM SHALL use the HTML renderer regardless of the detected project stack.

WHEN the user passes `--renderer=swiftui` in a non-Swift project THE SYSTEM SHALL attempt SwiftUI rendering if the toolchain is available, regardless of project stack.

WHEN SwiftUI source compilation fails THE SYSTEM SHALL attempt one auto-fix cycle (read error, patch source, retry). IF the retry also fails THEN THE SYSTEM SHALL fall back to HTML with the compilation error shown to the user.

WHEN `capture_dark_mode` is enabled (default: true) THE SYSTEM SHALL produce both `screenshot.png` and `screenshot-dark.png` with the appropriate `colorScheme` environment override.

THE SYSTEM SHALL generate an `index.html` viewer in every mock directory that displays the PNG screenshots and provides a collapsible source view, so `hero mock --open` and `hero mock --serve` continue working.

WHEN listing mocks via `hero mock --list` THE SYSTEM SHALL display a `[native]` or `[html]` tag for each mock based on whether a `screenshot.png` exists.

THE SYSTEM SHALL list all generated files (source, screenshots, viewer) with clickable links at the end of the `/mock` response.

WHEN a new renderer is added in the future, it SHALL only need to provide a skill file implementing the renderer interface (source guidelines, build command, capture command, output manifest) — no changes to the mock command or ui-designer agent dispatch logic.

## Changes

| File | Change |
|---|---|
| `domains/engineering/skills/swiftui-mockup-renderer/SKILL.md` | New skill — SwiftUI source generation guidelines, build/capture pipeline, output manifest |
| `domains/engineering/skills/swiftui-mockup-renderer/render.swift` | Reusable capture harness — takes a Swift source file and produces PNG |
| `domains/engineering/commands/mock.md` | Add renderer dispatch: stack detection → renderer selection → skill routing |
| `.claude/commands/mock.md` | Mirror renderer dispatch from domain command |
| `domains/engineering/agents/ui-designer.md` | Add renderer selection section, native generation flow, error recovery |
| `domains/engineering/skills/stack-detection/SKILL.md` | Add Swift/Xcode root markers → renderer hint |
| `internal/cli/mock.go` | Add `[native]`/`[html]` tag to `--list` output |
| `internal/cli/mock_test.go` | Test native tag detection in list output |

## Risks

- **macOS-only for SwiftUI rendering.** `swiftc` with SwiftUI frameworks is only available on macOS. Linux and Windows users always get HTML fallback. This is acceptable — SwiftUI development itself requires macOS.
- **`ImageRenderer` availability.** Requires macOS 13+ / iOS 16+. Older systems fall back to `NSHostingView` snapshot approach, which is slightly more complex but works on macOS 12+.
- **Compilation errors from generated code.** The AI-generated SwiftUI may not compile on the first attempt. The one-retry auto-fix mitigates this, but complex views may still fail. HTML fallback ensures the user always gets *something*.
- **Xcode CLI tools required.** Users need `xcode-select --install` at minimum. This is standard for any Swift development, but worth documenting.

## Future Direction

The renderer dispatch architecture is designed so adding a new platform is skill-only work:

1. Create `{framework}-mockup-renderer/SKILL.md` with generation guidelines + pipeline
2. Add root markers to `stack-detection`
3. The mock command and ui-designer agent pick it up automatically

Likely next renderers: Jetpack Compose (Android Studio preview capture), Flutter (golden test capture), Electron/Tauri (hybrid desktop).
