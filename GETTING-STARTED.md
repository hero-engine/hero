# Getting Started with Hero

Hero is a project memory and verified delivery system for AI-assisted
engineering. It first preserves the truth a future session needs—intent,
decisions, corrections, conventions, evidence, failures, and current state.
Its connected delivery workflow uses that memory to design, implement, audit,
and verify bounded work.

This guide uses the current CLI and harness-native workflow surfaces.

## 1. Install the binary

Prebuilt binaries do not require Go.

macOS or Linux with Homebrew:

```bash
brew install hero-engine/tap/hero
```

Linux install script:

```bash
curl -fsSL https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.sh | sh
```

Windows with Scoop:

```powershell
scoop bucket add hero-engine https://github.com/hero-engine/scoop-bucket
scoop install hero
```

Windows PowerShell install script:

```powershell
irm https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.ps1 | iex
```

Confirm the binary and available commands:

```bash
hero --version
hero --help
```

Building from source requires the version in `go.mod`, currently Go 1.26.4.
See the [installation guide](web/docs/src/getting-started/installation.md) for
direct downloads and source-build instructions.

## 2. Initialize one project corpus

From the repository root:

<!-- hero-quickstart -->
```bash
hero init
hero install project . --target codex
hero status
```

This creates the root `.hero/` corpus and installs Hero into Codex. Replace
`codex` with any supported target:

```bash
hero install project . --target opencode
hero install project . --target cursor
hero install project . --target claude
hero install project . --target copilot
hero install project . --target generic
hero install project . --target grok
```

Hero uses each harness's native surface. Claude receives command files. Codex
and Grok receive `command-*` workflow skills. Other targets receive the native
instructions and integration files they support. Natural-language requests
work across Hero-aware harnesses; slash-command syntax is not universal.

The default project uses Core plus Engineering, including lightweight PM and QA
help for engineering work. Focused PM, QA, and Sales setups are optional:

```bash
hero domain list
hero domain enable qa
hero install project . --target codex
```

## 3. Build useful project memory

Run a scan after installation:

```bash
hero scan
hero status
```

The scan detects the stack, indexes code, and seeds project knowledge. Ask your
harness to “scan this project” when you want the installed workflow to guide the
same operation.

Retrieve context from the corpus:

```bash
hero resume
hero search "OAuth session handling"
hero ask "What is our retry policy?"
hero relevant src/auth/session.go src/auth/middleware.go
hero why session-retry
hero blocked
```

Capture information that must survive the session:

```bash
hero note session-retry
hero next
hero recap --since 2d
```

Installed workflows can also codify conventions and refresh handoff state. Ask
the harness to “document our API error convention” or “prepare a session
handoff”; it will use its installed native workflow surface.

## 4. Deliver work against evidence

Ask the installed harness to “design CSV export.” Review and approve the
resulting spec before asking it to “deliver CSV export.” For a bug, ask it to
“diagnose the login timeout” before delivery.

Create an empty CLI scaffold only when you intend to author its required
sections before delivery:

```bash
hero spec new csv-export --type feature
```

For an approved, harness-authored spec, the CLI exposes its state and gates:

```bash
hero spec score csv-export
hero spec deliver csv-export --manual
hero diff .hero/planning/features/csv-export/spec.md
hero drift csv-export
hero ac list csv-export
hero coverage csv-export
```

`hero drift` and `hero coverage` return non-zero when they find drift or missing
coverage. Treat that result as evidence to resolve before verification.

The delivery workflow must finish its Completion Ledger, run the configured
build/tests, and obtain a fresh independent cold audit. Then close once:

```bash
hero spec verify csv-export
```

`hero spec verify` checks the Completion Ledger, cold audit, acceptance-criteria
coverage, and build/test result. When the hard gates pass, it marks the spec
complete and archives it. Do not add `hero spec complete` after successful
verification; that command is not the normal evidence-backed close.

## 5. Work in a monorepo

Initialize Hero once at the monorepo root. Do not run `hero init` in each
subproject. If sessions open within subfolders, materialize thin harness-native
satellite trees that point to the one root corpus:

```bash
hero install satellites                    # guided candidate review
hero install satellites --yes              # accept detected candidates
hero install satellites --repair           # repair/reconcile satellites
hero install satellites --migrate-nested   # print legacy migration plan
```

For an existing project install, select its scope when repairing:

```bash
hero install project . --target codex --repair
hero install satellites --repair
hero check
hero doctor
```

`hero doctor` diagnoses the running binary, schema, and installed targets.
`hero check` validates workspace and corpus health.

## 6. Query and coordinate work

```bash
hero list --ready --sort priority
hero queue --format kickoff
hero spec claim csv-export --agent codex
hero spec claim csv-export --release
hero feed --since 1h
hero graph csv-export
hero snapshot
```

Cross-repository peering is optional and keeps one graph per project. After
registering reachable sibling repositories, requests travel asynchronously
through Project Mail:

```bash
hero admin repos add app ../app
hero peer manifest
hero peer list
hero peer call app --mode=advisory --reason="Confirm the contract" \
  "Is the error envelope stable?"
hero handoff order-failure app --reason="Implementation belongs to app"
```

Sending does not launch a model or write the receiver tree. The receiver must
inspect and explicitly promote a work-transfer message. See
[Cross-Repo Peering](CROSS-REPO-PEERING.md).

## 7. Optional integrations

Tracker access requires a configured provider and credentials. Never put tokens
in command arguments or committed `hero.json`; use `--token-stdin`, environment
variables, or `.hero/hero.local.json`.

```bash
hero sync connect --list
hero sync import --dry-run
hero sync pull .hero/planning/features/csv-export/spec.md
```

Local project intelligence is available through `hero serve`:

```bash
hero serve --add .
hero serve --list
hero serve
```

Team mode and headless agent execution are preview capabilities. They require
deliberate auth/network configuration and, for headless work, a model provider,
credentials, and a supported execution environment:

```bash
hero agent run deliver csv-export --dry-run
hero agent jobs
```

Preview status means these paths are not a promise of unsupervised or
production-ready execution. Direct runner jobs and team queue jobs currently
use separate stores; no end-to-end approval pause/resume path between them is
documented or shipped. See [Team Server](TEAM-SERVER.md).

## 8. Configuration

This complete example is intentionally small and loads through Hero's production
configuration decoder:

<!-- hero-config -->
```json
{
  "folder": ".hero",
  "team": {
    "require_review": true,
    "stale_days": 14,
    "auto_context": true,
    "nudge_level": "assertive"
  },
  "knowledge": {
    "auto_capture": true,
    "explainer_synthesis": "review"
  },
  "next": {
    "mode": "personal",
    "projected": true
  },
  "delivery": {
    "default_mode": "supervised",
    "autopilot_halt_on": ["drift", "test", "boundary", "lint"]
  },
  "verify": {
    "run_tests": true,
    "test_command": "go test ./..."
  }
}
```

Commit `.hero/hero.json`. Keep secrets and personal overrides in the ignored
`.hero/hero.local.json` or protected environment variables.

Generated or machine-local files should remain ignored:

```gitignore
.hero/index.db
.hero/graph.db
.hero/events.log
.hero/hero.local.json
.hero/next/*.local.md
.hero/sessions/
```

The complete decoder-backed example and field reference are in
[Hero configuration](web/docs/src/configuration/hero-json.md).

## Quick reference

| Goal | Surface |
|---|---|
| Resume project context | Ask the harness to resume, or run `hero resume` |
| Design or diagnose work | Use the installed harness-native workflow |
| Inspect ready work | `hero queue --format kickoff` |
| Search project memory | `hero search "<query>"` |
| Get file-aware context | `hero relevant <paths>` |
| Verify and close delivery | `hero spec verify <slug>` |
| Diagnose binary/install drift | `hero doctor` |
| Check workspace health | `hero check` |
| Inspect exact installed inventory | `hero doctor` |
| Inspect exact MCP inventory | MCP `tools/list` after filtering |

Next: [MCP Setup](MCP-SETUP.md) or the
[hosted documentation source](web/docs/src/index.md).
