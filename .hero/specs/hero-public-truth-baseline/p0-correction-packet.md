# v0.34 public documentation correction packet

Verified: 2026-08-23 at revision `75ea3cb1`

This packet gives downstream documentation work one replacement contract for every P0 mismatch identified in `../content-truth-audit.md`. It does not edit public copy itself.

## P0-1 — satellite management commands

Current public locations:

- `README.md:105-107`
- `GETTING-STARTED.md:98-100`
- `web/docs/src/getting-started/project-setup.md:95`

Remove:

```text
hero install satellites list
hero install satellites add services/auth
```

Replace with:

```text
hero install satellites                    # guided candidate review
hero install satellites --yes              # accept detected candidates non-interactively
hero install satellites --repair           # repair/reconcile materialized satellites
hero install satellites --migrate-nested   # print a legacy nested-workspace migration plan
```

Architecture statement:

> A monorepo has one root `.hero` corpus. Satellite installs are thin harness-native trees in subproject folders that point back to the root content; they are not nested Hero workspaces.

Executable validation:

```text
go run ./cmd/hero install satellites --help
go test ./internal/install -run 'Test(MaterializeFullSatellite|FindNestedHeroDirs)' -count=1
```

Observed failure proving the stale forms are not subcommands: both positional forms enter the same candidate-walking flow and, without a terminal, return `walking subproject candidates requires an attached terminal; pass --yes or --no`.

Owner: `hero-root-docs-remediation`

## P0-2 — install repair scope

Current public locations:

- `README.md:107`
- `GETTING-STARTED.md:100`

Remove:

```text
hero install --repair
```

Replace with the repair command matching the user's intent:

```text
hero install project --repair       # repair the project install
hero install satellites --repair    # repair satellite trees
```

Executable validation:

```text
go run ./cmd/hero install --help
go run ./cmd/hero install --repair
```

Observed stale-form result: `requires at least 1 arg(s), only received 0`.

Owner: `hero-root-docs-remediation`

## P0-3 — monorepo workspace topology

Current public locations:

- `README.md`
- `GETTING-STARTED.md`
- `web/docs/src/getting-started/project-setup.md`
- `web/docs/src/project-structure.md`

Remove any instruction to run `hero init` independently inside subprojects of an existing Hero root.

Replacement:

> Initialize Hero once at the repository root. From that root, run `hero install satellites` to make supported harness content available when a session opens inside a subproject. Use `--migrate-nested` for a repository that already contains legacy nested `.hero` workspaces.

Executable validation:

```text
go run ./cmd/hero install satellites --help
go test ./internal/install -run 'Test(MaterializeFullSatellite|FindNestedHeroDirs)' -count=1
```

Owner: `hero-root-docs-remediation`

## P0-4 — delivery completion

Current public locations:

- `README.md:233,256`
- `GETTING-STARTED.md:199,431`
- `web/docs/src/cli/spec-management.md:43`
- `web/docs/src/project-structure.md:112`
- `web/docs/src/commands/index.md:101`

Remove any normal delivery sequence that runs `hero spec complete` after verification.

Replacement:

```text
hero spec verify <slug>
```

Explanation:

> `hero spec verify` checks the Completion Ledger, cold delivery audit, AC-to-test coverage, and build/test result. When the hard gates pass, it marks the spec complete and archives it. `hero spec complete` is not the normal evidence-backed delivery close.

Executable validation:

```text
go run ./cmd/hero spec verify --help
go test ./internal/cli -run 'Test.*Verify' -count=1
```

Owner: `hero-root-docs-remediation`

## P0-5 — `hero.json` full example

Current public locations:

- `web/docs/src/configuration/hero-json.md:10-130`
- smaller root examples in `README.md` and `GETTING-STARTED.md`

The hosted full example currently contains decoder-incompatible shapes, including a string-valued `import.filter`, object-valued `hooks.branch_patterns`, and nested profile objects where `serve.tool_filter.profiles` expects lists of allowed tool names.

Replacement authority:

- `internal/config/testdata/public-hero.json`
- `internal/config/public_example_test.go`

The downstream docs child should replace its full example with the fixture, or generate the block from that fixture, rather than maintaining a second hand-written shape.

Executable validation:

```text
go test ./internal/config -run TestPublicHeroConfigFixtureLoadsThroughProductionDecoder -count=1
```

Owner: `hero-hosted-docs-remediation`

## P0-6 — Go prerequisite

Current public location:

- `README.md:477`

Remove:

```text
Hero requires Go 1.21+.
```

Replacement:

> Prebuilt Hero binaries do not require a Go toolchain. Building Hero from source requires the Go version declared by `go.mod`—currently Go 1.26.4.

Executable validation:

```text
sed -n '1,8p' go.mod
```

Observed authority: `go 1.26.4`.

Owner: `hero-root-docs-remediation`

## P0-7 — installation health command

Current public location:

- `README.md:242`

Remove:

```text
hero verify-install
```

Replace by intent:

```text
hero doctor   # binary, schema, and installed-target diagnosis
hero check    # workspace health and documentation/spec hygiene
```

Executable validation:

```text
go run ./cmd/hero --help
go run ./cmd/hero doctor --help
go run ./cmd/hero check --help
```

Observed stale-form result: `unknown command "verify-install" for "hero"`.

Owner: `hero-root-docs-remediation`

## Derived public inventory authority

The counts below are evidence for generated reference pages only. Narrative product pages should describe capabilities rather than repeat mutable totals.

| Surface | Current derived value | Authority |
|---|---:|---|
| Engineering workflow commands | 29 | `hero doctor`, `internal/install/inventory.go` |
| Engineering agents | 35 | `hero doctor`, `internal/install/inventory.go` |
| Canonical engineering skills | 57 | `hero doctor`, `internal/install/inventory.go` |
| Codex/Grok installed skills | 86 | 57 canonical skills + 29 command skills, enforced by `internal/install/inventory_test.go` |
| Runtime MCP tools before filtering | 82 | 61 core tools + 21 code-host operations, enforced by `internal/serve/mcp_test.go` |
| Supported install targets | 7 | opencode, cursor, claude, copilot, codex, generic, grok in `internal/install/inventory.go` |

Validation run on 2026-08-23:

```text
go run ./cmd/hero doctor
go test ./internal/install ./internal/serve -run 'Test.*(Inventory|ToolsList)' -count=1
```
