---
title: Demo Recording — Playwright Video Capture for Delivered Specs
slug: demo-recording
type: feature
status: completed
milestone: v0.5
tags: [demos, recording, playwright, video, delivery]
created: 2026-04-14
relations:
  - target: playwright-test-generation
    kind: related
  - target: git-hook-integration
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Let teams automatically or manually record video demos of delivered features by running Playwright tests with video capture enabled. Demos are saved to `.hero/demos/<slug>/` and serve as living proof that a feature works as specified.

## Problem

After delivering a feature, there's no artifact that proves it works beyond "the tests pass." Product managers, stakeholders, and future engineers benefit from seeing a recorded demo of each feature. Today, recording requires manual screen capture tools (Loom, OBS) which are tedious and quickly go stale.

Playwright already supports video recording natively. If Hero has generated tests for a spec (via the test generation feature), it can re-run those tests with `video: 'on'` to produce demo recordings automatically.

## Design

### Configuration

`hero.json` gains a `demos` block:

```json
{
  "demos": {
    "mode": "manual",
    "framework": "playwright",
    "output_dir": ".hero/demos",
    "video_size": { "width": 1280, "height": 720 },
    "on_deliver": false
  }
}
```

| Field | Type | Default | Description |
|---|---|---|---|
| `mode` | string | `"manual"` | Recording trigger: `auto` (on deliver) or `manual` (on-demand) |
| `framework` | string | `"playwright"` | Recording framework adapter |
| `output_dir` | string | `".hero/demos"` | Base directory for demo recordings |
| `video_size` | object | `{width:1280, height:720}` | Video resolution |
| `on_deliver` | bool | `false` | Auto-record when spec status changes to completed |

### CLI Commands

```
hero demo record <slug>         # Record a demo for a spec by running its tests with video
hero demo record --all          # Record demos for all specs that have test files
hero demo list                  # List specs with demo recording status
hero demo show <slug>           # Show demo file paths for a spec
hero demo clean [<slug>]        # Remove demo recordings (all or for a specific spec)
```

### Recording Flow

`hero demo record <slug>`:

1. Verify a test file exists for the spec (from `hero test generate`)
2. Create output directory: `<output_dir>/<slug>/`
3. Run the test with Playwright video recording enabled:
   ```
   npx playwright test <test_dir>/<slug>.spec.ts --config <config_path> \
     --project=chromium \
     --reporter=list
   ```
   The command sets `PWVIDEO=1` environment variable. The generated test files check this env var and enable video recording via `test.use({ video: 'on' })` when set.
4. After test completion, move WebM files from Playwright's `test-results/` to `.hero/demos/<slug>/`
5. Generate a `manifest.json` in the demo directory:
   ```json
   {
     "slug": "my-feature",
     "title": "My Feature",
     "recorded_at": "2026-04-14T10:30:00Z",
     "videos": [
       {
         "name": "criterion-1-description",
         "path": "criterion-1-description.webm",
         "size_bytes": 245760,
         "duration_hint": "test passed"
       }
     ],
     "test_file": "e2e/my-feature.spec.ts",
     "status": "pass"
   }
   ```

### Pluggable Framework Interface

```go
type DemoFramework interface {
    Name() string
    Record(slug string, testFile string, cfg DemosConfig, testCfg TestingConfig) (*DemoResult, error)
    VideoDir(slug string, cfg DemosConfig) string
    Clean(slug string, cfg DemosConfig) error
}
```

`PlaywrightDemoFramework` implements this. Additional adapters (Cypress with `video: true`, manual screen capture wrappers) can be added.

### Auto Mode

When `mode: "auto"` and `on_deliver: true`, the post-checkout hook (or a future post-deliver hook) triggers demo recording after a spec transitions to `completed`. This is advisory — if tests don't exist or fail, the demo is simply skipped with a warning.

### Demo Output Structure

```
.hero/demos/
  my-feature/
    manifest.json
    criterion-1-description.webm
    criterion-2-description.webm
  another-feature/
    manifest.json
    ...
```

### MCP Tool

```json
{
  "name": "hero_demo_record",
  "description": "Record a video demo for a delivered spec by running its tests with video capture",
  "inputSchema": {
    "type": "object",
    "properties": {
      "slug": { "type": "string", "description": "Spec slug to record demo for" }
    },
    "required": ["slug"]
  }
}
```

## Changes

- `internal/config/config.go` — add `DemosConfig` struct and `Demos` field to `Config`
- `internal/demos/framework.go` — `DemoFramework` interface, result types, registry
- `internal/demos/playwright.go` — Playwright video adapter implementing `DemoFramework`
- `internal/demos/record.go` — orchestration: find test file, run with video, collect artifacts
- `internal/cli/demo.go` — `hero demo record|list|show|clean` commands
- `internal/cli/root.go` — register `demoCmd`
- `internal/serve/mcp.go` — add `hero_demo_record` tool

## Acceptance Criteria

- `hero demo record <slug>` runs the spec's test file with video recording enabled
- Video files are saved to `.hero/demos/<slug>/` directory
- A `manifest.json` is generated with recording metadata
- `hero demo list` shows specs with demo status (recorded / not recorded / no tests)
- `hero demo show <slug>` prints the manifest and file paths
- `hero demo clean <slug>` removes the demo directory for a spec
- `hero demo clean` (no slug) removes all demo recordings
- `--all` flag on record operates on all specs with test files
- Configuration is read from `hero.json` `demos` section with sensible defaults
- Framework is pluggable — adding a new adapter requires implementing `DemoFramework` interface only
- Works with no `demos` config (defaults to Playwright, manual mode, `.hero/demos/` directory)
- `hero_demo_record` MCP tool records a demo and returns the manifest

## Boundaries

- Does **not** make LLM API calls
- Does **not** install Playwright or npm packages — assumes the project has them
- Does **not** transcode video — output is WebM (Playwright's native format)
- Does **not** host or serve demo videos — output is local files only
- Does **not** start a dev server — assumes the application is running at `base_url`
- Does **not** create test files — requires `hero test generate` to have been run first
- Auto mode is advisory — failures are logged as warnings, not errors
