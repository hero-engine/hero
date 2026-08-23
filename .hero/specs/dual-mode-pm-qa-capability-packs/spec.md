---
title: Dual-Mode PM and QA Capability Packs
slug: dual-mode-pm-qa-capability-packs
type: feature
status: completed
domain: engineering
size: large
priority: P0
created: 2026-08-22
tags: [domains, engineering, pm, qa, capability-packs, composition, routing]
relations:
  - target: hero-domains
    kind: parent
  - target: hybrid-content-packs-and-workflow-providers
    kind: related
  - target: multi-domain-context-activation
    kind: related
  - target: hero-pm
    kind: related
  - target: hero-qa
    kind: related
depends-on:
  - domain-plugin-architecture
  - domain-routing-and-agents
  - spec-type-registry
  - pm-public-pack
  - qa-public-pack
horizon: next
smoke: deferred
delivery_method: manual
completed_at: 2026-08-23T00:25:33Z
---

# Dual-Mode PM and QA Capability Packs

## Goal

Let engineering workspaces retain lightweight planning and quality
assistance by default while enabling the full PM and QA packs as optional,
bounded extensions. The same PM or QA package must also work as the primary
experience in a dedicated workspace. Composition must remain deterministic,
keep model-facing rosters lean, preserve artifact ownership, and never rely on
duplicate `lite`/`full` content trees or file-order collision resolution.

## Kickoff

Add one-primary-plus-extensions composition so engineering can enable full PM
and QA capabilities without switching away from engineering or loading every
domain roster at once.

**Status:** planning — activation, routing, ownership, collision, and one-command setup contracts are designed; delivery is sequenced after the full PM and QA packs exist.

**Pick up at:** implement the workspace configuration migration and a resolver that returns Core, one primary pack, and ordered bounded extensions.

→ `/deliver dual-mode-pm-qa-capability-packs`

**Files:** `internal/domains/types.go`, `content.go`, `internal/cli/install.go`, `internal/install/manifest.go`, `internal/graph/scope.go`
**Skip:** separate PM/QA lite packages, arbitrary all-domain roster unions, and last-file-wins collision handling.

## Problem

Hero currently resolves one active domain over Core:

```text
Core <- one active domain
```

That model is reliable and keeps rosters small, but it forces a false choice.
An engineer benefits from product framing, acceptance-criteria help, test
strategy, functional validation, and release confidence without wanting to
leave an engineering workspace. The full PM and QA packs add valuable
specialist capabilities, yet selecting either as the active domain replaces
the engineering experience rather than augmenting it.

Copying each pack into `lite` and `full` variants would solve the immediate
toggle problem but create two versions of every prompt, command, spec type, and
methodology. Unioning all enabled domain files is also unsafe: PM and
engineering already share names such as `roadmap-reviewer`, `discover`,
`handoff`, and `scrub`; QA intentionally reuses `design`, `deliver`,
`diagnose`, `review`, and `why`. Current `OverlayFS` precedence would make the
winner an ordering accident.

The current single `active domain` also carries too many meanings. In a
composed workspace, engineering can be the primary experience while a QA-routed
operation creates a QA-owned test plan. Workspace default, UI focus, routing
context, and artifact ownership cannot remain one field.

## Evidence From the Current Packs

- Engineering already contains lightweight planning and quality roles:
  `product-ideator`, `feature-delivery-lead`, `test-architect`,
  `functional-qa-engineer`, `testing-and-validation`, and `test-strategy`.
- Planned QA has no exact agent-name collision with engineering, but it shares
  five command names and several common skills, and it overlaps semantically
  with engineering review, investigation, release, and testing roles.
- PM has one exact agent collision (`roadmap-reviewer`), three exact command
  collisions (`discover`, `handoff`, `scrub`), and pack-local declarations for
  both of its currently authored spec types even though `intake` and `prd`
  already exist in Core.
- Hero Code's QA UI design already distinguishes developer-facing lightweight
  QA surfaces from a full QA professional workspace. This feature makes that
  two-tier product idea a public Hero content and routing contract rather than
  a proprietary-client exception.

## Design

### 1. Workspace composition

Workspace configuration evolves from one `domain` value to a composition:

```json
{
  "domains": {
    "primary": "engineering",
    "extensions": ["pm", "qa"]
  }
}
```

Exactly one primary pack is required. Extensions are an ordered, deduplicated
set validated against each pack's permitted composition roles. Missing domain
configuration continues to mean `engineering` with no extensions. A legacy
`"domain": "pm"` value migrates in memory to primary `pm` with no extensions;
Hero writes the new shape only during an explicit configuration mutation or
upgrade.

The resolved workspace stack is:

```text
Core -> primary pack -> declared extension contributions -> project overrides
```

This is not a generic N-layer filesystem union. Each extension exposes a
bounded contribution projection that is validated before installation.

### 2. One package, two activation roles

PM and QA each remain one package with one version and one source tree. Their
manifests declare what is exposed when the pack is:

- **primary** — its default navigation, complete domain roster, primary natural-
  language routing, specialist artifacts, and views are active;
- **extension** — its specialist capabilities remain available, but the primary
  pack retains default navigation and fallback routing. Only declared extension
  entry points are advertised; deeper agents and skills load on demand.

Activation projections are manifest metadata, not copied prompts. The same
artifact ID always resolves to the same content regardless of activation role.

### 3. Engineering essentials

Engineering remains useful with no PM or QA extension enabled. Its existing
lightweight planning and quality capabilities stay available under engineering
ownership:

- planning essentials: problem framing, feature design, acceptance criteria,
  issue intake, and implementation handoff;
- quality essentials: test strategy, test architecture, functional validation,
  regression awareness, and release confidence.

The full PM pack adds discovery, PRDs, roadmaps, prioritization, portfolio and
capacity work, experiments, metrics, PM artifacts, and PM views. The full QA
pack adds first-class test plans/cases, coverage operations, regression and
flake curation, QA integrations, release gates, QA artifacts, and QA views.
The specialist packs complement the engineering essentials; they do not copy or
silently replace them.

### 4. Stable artifact identities and collision rules

Every contributed agent, skill, command handler, spec-type amendment, and view
has a stable namespaced ID and a declared owner. Composition validates IDs and
output paths before changing install state.

- If two entries are the same capability, one package owns the canonical
  artifact and the other references it.
- If two entries have similar labels but different behavior, their stable IDs
  and rendered filenames are namespaced.
- If a pack amends a canonical artifact, the owner must expose an extension
  point and the amendment must name it.
- Otherwise, duplicate IDs or output paths fail with an actionable conflict
  report. Pack order never decides the winner.

The first PM/QA reconciliation must resolve the known collisions named in
`## Evidence From the Current Packs` and produce an inventory test that prevents
them from returning.

### 5. Shared commands are composed routers

User-facing verbs such as `design`, `deliver`, `diagnose`, `review`, `discover`,
`handoff`, `scrub`, and `why` are installed once. A pack contributes a
namespaced handler containing:

- the command/router ID it extends;
- supported artifact types and lifecycle states;
- optional intent/capability predicates;
- target agent or workflow;
- priority only within the declared extension point;
- owning domain for resulting artifacts.

At install time Hero renders one deterministic router per shared command. At
runtime the most specific matching handler wins; ambiguous equal-specificity
matches fail with the competing handler IDs instead of choosing by pack order.
Codex and Grok receive the equivalent composed `command-*` skills; other
harnesses receive their native command representation.

### 6. Shared spec types use amendments, not shadows

Canonical type identity is singular. A PM or QA pack may contribute declared
lifecycle, vocabulary, optional-field, validation, or view amendments to a
canonical type. It does not install a second shadow definition at the same
path. Amendments are namespaced, compatible with the owner's declared extension
points, and reported in the resolved manifest.

This contract applies immediately to PM's `intake` and `prd` definitions and to
QA lifecycle additions such as `qa-ready` and `qa-rejected` on engineering/PM
work items.

### 7. Primary, focus, routing, and ownership are separate

- **Primary domain** is committed workspace configuration and determines the
  default experience and fallback router.
- **Focused domain** is ephemeral client/session UI state and determines which
  enabled pack's views and entry points are foregrounded.
- **Routing domain** is selected for one command or agent invocation from the
  matched contribution.
- **Artifact domain** is durable provenance stamped from the owning handler or
  artifact type, not blindly copied from the workspace primary.

Therefore a QA handler in an engineering-primary workspace creates
`domain: qa` artifacts, while ordinary feature delivery remains
`domain: engineering`. Generic capture that has no pack owner remains Core or
uses the primary domain according to its existing contract.

### 8. Retrieval over the enabled stack

Default workspace reads include artifacts owned by Core, the primary pack, and
enabled extensions. Ranking prefers the focused domain, then the primary, then
other enabled extensions. Explicit `--domain`/MCP scope remains available, and
workspace-wide reads can still request all domains. Disabling an extension
removes its handlers and views but never hides or rewrites its historical
artifacts; those remain discoverable through explicit or workspace-wide reads.

### 9. Install and removal behavior

Resolution produces one canonical install manifest before any files change.
The existing safe stale-file pruning then materializes the stack for every
installed harness. Enabling or disabling an extension is atomic: validation
must pass for the complete next stack, and a failure leaves both workspace
configuration and rendered files unchanged.

### 10. One-command setup and lifecycle commands

The non-interactive `hero init` contract remains safe for CI and setup
scripts. It gains explicit composition and install flags rather than an
interactive-only wizard:

```text
hero init --domain pm --target codex
hero init --domain qa --target claude
hero init --domain engineering --with pm,qa --target cursor
```

`--domain` selects the primary pack. Repeatable or comma-separated `--with`
values select extensions. When one or more `--target` values are supplied,
initialization materializes the resolved Hero content for those harnesses in
the same transaction after the workspace is created. With no target, existing
non-interactive behavior remains compatible and the output prints the exact
`hero install project` command needed to finish materialization.

Existing workspaces use explicit lifecycle commands:

```text
hero domain enable qa
hero domain enable pm
hero domain disable qa
hero domain switch pm
```

`enable` and `disable` mutate the extension set and reinstall all previously
installed harness targets atomically. `switch` changes the primary pack,
removes that pack from extensions if present, and retains every other
compatible extension. `hero domain` and `hero domain list` show the primary,
enabled extensions, permitted roles, and whether each bundled pack is ready.

PM and QA are official bundled packs for this feature. Initialization,
switching, enabling, disabling, and ordinary use require no registry or
network. The resolver architecture may support independently published
updates later, but an unavailable remote service cannot prevent creation or
repair of an official PM or QA workspace.

## User Experience

The default remains zero-ceremony engineering. Enabling full PM or QA is an
explicit workspace action. The CLI must support these motions without editing
JSON by hand:

```text
engineering only
engineering + PM
engineering + QA
engineering + PM + QA
PM primary
QA primary
```

The status/list surface shows the primary pack, enabled extensions, their
activation roles, and any validation conflict. A user can disable an extension
without deleting its data.

## Acceptance Criteria

- WHEN Hero resolves a workspace with the new domain configuration THE SYSTEM SHALL require exactly one primary pack, deduplicate and validate ordered extensions, and expose the resolved stack through one canonical API. **AC-1 — Composition config**
- IF domain configuration is absent or uses the legacy scalar field THEN THE SYSTEM SHALL resolve to the equivalent primary-only workspace without requiring a network call or immediate config rewrite. **AC-2 — Legacy compatibility**
- WHERE PM or QA is enabled THE SYSTEM SHALL use the same package identity, version, and source content for primary and extension activation while applying the role's declared projection. **AC-3 — Dual-mode packages**
- WHILE no PM or QA extension is enabled THE SYSTEM SHALL retain engineering's lightweight planning and quality agents, skills, commands, and test/release behavior. **AC-4 — Engineering essentials**
- WHILE PM or QA is enabled as an extension THE SYSTEM SHALL keep engineering as the fallback roster and advertise only the extension's declared entry points, loading deeper specialist content on demand rather than unioning complete domain rosters. **AC-5 — Bounded roster**
- IF two enabled contributions claim the same stable ID or rendered output without a valid declared extension relationship THEN THE SYSTEM SHALL reject the next composition with both owners and the conflicting identity, leaving the installed state unchanged. **AC-6 — Collision safety**
- WHEN enabled packs contribute handlers to a shared command THE SYSTEM SHALL render one deterministic command router, select handlers by declared artifact/intent predicates, and fail ambiguous equal-specificity matches rather than using pack order. **AC-7 — Shared command routers**
- WHEN a pack extends a canonical spec type THE SYSTEM SHALL register a namespaced amendment against an owner-declared extension point and SHALL NOT install a shadow type definition at the canonical path. **AC-8 — Spec-type amendments**
- WHEN a PM- or QA-owned handler creates or mutates an artifact in an engineering-primary workspace THE SYSTEM SHALL stamp and preserve the handler's owning domain independently of primary or focused domain state. **AC-9 — Artifact ownership**
- WHILE extensions are enabled THE SYSTEM SHALL include their artifacts in default workspace retrieval with focused/primary ranking, while preserving explicit single-domain and all-domain query controls. **AC-10 — Enabled-stack retrieval**
- WHEN an extension is disabled THE SYSTEM SHALL atomically remove its handlers, views, and Hero-owned rendered files while preserving project-owned files, historical artifacts, and the ability to retrieve those artifacts explicitly. **AC-11 — Safe disable**
- THE SYSTEM SHALL reconcile the existing PM/engineering and QA/engineering overlaps into canonical shared artifacts, namespaced specialist artifacts, or declared amendments, with an inventory test proving there are no unexplained collisions. **AC-12 — Known collision reconciliation**
- WHERE any supported harness is installed THE SYSTEM SHALL produce semantically equivalent composed routing and roster behavior for `opencode`, `cursor`, `claude`, `copilot`, `codex`, `generic`, and `grok`, including native `command-*` skills where required. **AC-13 — Harness parity**
- WHILE a resolved stack is installed THE SYSTEM SHALL perform ordinary routing, retrieval, and artifact creation from local content without lazy package acquisition or network dependence. **AC-14 — Local deterministic operation**
- WHEN a user initializes with `--domain pm` or `--domain qa` THE SYSTEM SHALL create a workspace with that official bundled pack as primary without requiring a registry or network. **AC-15 — Standalone initialization**
- WHEN a user initializes with one or more `--with` values THE SYSTEM SHALL validate and persist those packs as extensions of the selected or default engineering primary without requiring manual JSON edits. **AC-16 — Extension initialization**
- WHERE one or more `--target` values are supplied to `hero init` THE SYSTEM SHALL materialize the resolved stack for every requested harness during initialization, while preserving the existing non-interactive no-target behavior. **AC-17 — Init-and-install**
- WHEN `hero domain enable`, `disable`, or `switch` succeeds THE SYSTEM SHALL atomically persist the new composition and reinstall every previously installed harness target, while a failed validation leaves configuration and files unchanged. **AC-18 — Lifecycle commands**
- WHEN Hero shows or lists domains THE SYSTEM SHALL identify the primary pack, enabled extensions, each pack's permitted roles, bundled readiness, and any composition conflict. **AC-19 — Inspectable setup**

## Validation

- Resolver unit tests cover missing/legacy config, each primary-only mode,
  engineering plus PM, engineering plus QA, engineering plus both, duplicate
  extensions, unsupported roles, and deterministic ordering.
- Collision tests cover duplicate IDs, duplicate output paths, legal command
  handlers, legal spec-type amendments, and atomic rollback on failure.
- Routing tests cover general engineering work, PM/QA specialist intents,
  artifact-state predicates, ambiguous handlers, and domain stamping.
- Retrieval tests cover primary/focused ranking, enabled extension visibility,
  disabled-extension history, explicit domain scope, and all-domain scope.
- Golden install tests run each composition across all seven harness targets and
  verify one shared router per verb, correct native placement, stable output,
  and safe stale-file pruning after disable.
- PM/QA inventory tests prove there are no unexplained shared IDs or rendered
  paths and that no `lite` content tree exists.
- CLI integration tests exercise standalone PM and QA initialization,
  engineering with each extension combination, init-time target installation,
  enable/disable/switch reinstalls, rollback after a conflict, and offline use.

## Approach

1. Deliver the official full PM and QA packs so both are truthful bundled
   primary or extension experiences rather than placeholders.
2. Add the composition/config types and backward-compatible resolver, keeping
   the current primary-only output byte-for-byte stable.
3. Add activation projections, stable contribution identities, collision
   validation, and an inspectable resolved manifest.
4. Compose shared command routers and spec-type amendments, then reconcile the
   current PM/QA collisions.
5. Separate primary, focused, routing, and artifact-domain state through the
   install, command, graph-stamping, and retrieval boundaries.
6. Add init-time materialization plus extension enable/disable/switch UX and
   atomic install/prune behavior.
7. Prove all six workspace compositions and all seven harness targets before
   changing defaults or documenting the capability as shipped.

## Out of Scope

- Remote pack catalogs, signing, download caches, and lockfiles.
- External workflow providers such as GitHub Spec Kit.
- Proprietary QA UI surfaces; the public full QA content pack is a dependency.
- Moving Hero Code or Hero Cloud into the open-source package boundary.
- Arbitrary simultaneous unions of unrelated primary domains such as sales and
  marketing.
- A public plugin marketplace or executable third-party pack hooks.

## Risks and Mitigations

- **Roster growth:** extension projections advertise only bounded entry points;
  specialist agents and skills load on demand.
- **Routing surprise:** predicates and specificity are inspectable; ambiguity
  fails instead of falling through to priority order.
- **Domain provenance drift:** artifact ownership is derived from the routed
  contribution and tested independently from UI focus.
- **Legacy breakage:** primary-only resolution remains the compatibility
  baseline and golden output.
- **Cross-repo contract drift:** public Hero owns the manifest and resolver
  contract; proprietary clients consume it and must preserve all-seven-harness
  semantics where they render Hero content.

## Completion Ledger

Delivered one-primary-plus-extensions composition for the bundled Engineering,
PM, and QA packs. The implementation uses local embedded content, preserves the
legacy primary-only contract, composes bounded extension surfaces and shared
routers, exposes composed spec types and retrieval scope, and adds lifecycle and
init/install controls across every supported harness.

Validation performed: `go test ./... -count=1`, a fresh `go build`, focused
composition/config/spec-type/graph/CLI/install suites, and fresh-binary workspace
exercises for PM primary on Codex, QA primary on Claude, Engineering + PM + QA on
Cursor and Codex, composed routing, deferred PM specialist loading, a QA-owned
graph artifact write, focused/default/explicit QA search behavior, QA
disable/re-enable, and switching PM to primary while retaining QA.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Require one primary and expose a validated, ordered, deduplicated stack | DONE | `internal/domains/composition.go`, `internal/domains/types.go`, and `internal/config/config.go` implement the canonical resolver; resolver/config tests cover valid and invalid shapes. |
| 2 | Preserve absent and legacy scalar configuration without rewriting or network access | DONE | `Config.ResolveDomains` maps absent configuration to Engineering and legacy `domain` to the equivalent primary-only composition; compatibility tests verify no mutation. |
| 3 | Use one PM/QA package source for primary and bounded extension activation | DONE | `composition_content.go` resolves both activation roles from the same embedded `DomainFS`; composition tests verify full standalone packs and bounded projections. |
| 4 | Retain lightweight Engineering planning and quality capabilities by default | DONE | The primary-only byte-compatibility test compares composed Engineering against the legacy Core + Engineering overlay, including its existing planning/testing agents, skills, and commands. |
| 5 | Keep Engineering fallback and advertise only declared extension entry points | DONE | PM/QA install bounded agent/command entry points; deeper agents, commands, and skills are deferred in the manifest and load read-only from embedded pack bytes through `hero domain content <stable-id>` without joining the installed roster. |
| 6 | Reject undeclared stable-ID/output collisions and leave installed state unchanged | DONE | Composition claims stable IDs and output paths independently; duplicate identities plus undeclared command/non-command paths fail with both owners and paths, while lifecycle failure tests prove config/render rollback. |
| 7 | Render one deterministic shared-command router with ambiguity failure | DONE | Manifests and rendered routers carry typed artifact, lifecycle, intent, target, priority, and owner predicates; `SelectCommandHandler` executes specificity/priority selection and returns a typed error for equal matches. |
| 8 | Register namespaced spec-type amendments without shadow definitions | DONE | Canonical feature types declare the lifecycle extension point; `applyAmendment` validates owner/target/extension-point compatibility and applies QA lifecycle states to the canonical registry and exported status enum without a shadow type. |
| 9 | Stamp routed-handler ownership independently of primary/focus | DONE | Executable selection validates owners, and `hero graph node add --handler-owner qa` carries that owner through the real graph mutation boundary; integration coverage creates a QA TestPlan, changes primary context, and verifies durable `qa` provenance. |
| 10 | Retrieve Core, primary, and enabled-extension artifacts with focus-aware ranking and explicit scopes | DONE | `hero search`, `hero list`, and `hero blocked` filter to Core + primary + enabled extensions and rank focus/primary stably; behavioral tests prove QA default/focused visibility, disabled exclusion, explicit QA history, and all-domain access. |
| 11 | Disable extensions safely while preserving project files and history | DONE | Install tests prove QA-owned rendered files are pruned while project-owned files survive; CLI lifecycle tests preserve historical QA artifacts and explicit/all-domain retrieval remains available. |
| 12 | Reconcile known PM/Engineering and QA/Engineering overlaps | DONE | Known shared commands require per-owner typed descriptors, canonical feature lifecycle additions use a declared amendment, and command-to-agent closure plus collision inventory tests reject unexplained overlap or any `lite` tree. |
| 13 | Preserve composed semantics across all seven harnesses | DONE | The all-harness install test iterates every advertised extension agent, every extension command target, and every resolved shared router/handler across OpenCode, Cursor, Claude, Copilot, Codex, Generic, and Grok. |
| 14 | Operate from local content without package acquisition | DONE | Composition uses embedded Core/domain filesystems only; fresh-binary standalone and combined workspace exercises completed without a registry or network dependency. |
| 15 | Initialize bundled PM or QA as standalone primary packs | DONE | CLI integration tests and fresh-binary exercises verify `hero init --domain pm --target codex` and `hero init --domain qa --target claude`. |
| 16 | Accept, validate, and persist repeatable/comma-separated `--with` extensions | DONE | `internal/cli/init.go` uses canonical composition validation and StringSlice parsing; integration tests verify Engineering with ordered PM + QA persistence without manual JSON edits. |
| 17 | Materialize every supplied init target and preserve no-target behavior | DONE | CLI integration tests verify comma-separated multi-target init and the exact no-target follow-up command; fresh binary installed the combined stack to Cursor and Codex in one init. |
| 18 | Atomically enable, disable, or switch domains across installed targets | DONE | `internal/cli/domain.go` reinstalls the union of recorded/detected targets, saves config last, and rolls every target back on failure; tests cover failure rollback, enable/disable, stale pruning, and switching PM primary while retaining QA. |
| 19 | Show primary, extensions, roles, and bundled readiness | DONE | `hero domain` and `hero domain list` render the resolved composition, state, permitted roles, and local bundled readiness; CLI tests assert the complete surface. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add canonical composition and pack-role types | DONE | `internal/domains/types.go` and `internal/domains/composition.go` define bundled packs, activation roles, validation, deduplication, and stack helpers. |
| 2 | Migrate configuration through a compatibility resolver | DONE | `internal/config/config.go` reads legacy scalar configuration, writes canonical `domains`, validates on load/save, and exposes the resolved primary to existing consumers. |
| 3 | Compose local pack content and an inspectable manifest | DONE | `composition_content.go` builds installed and deferred projections, validates stable IDs and paths, exposes local deferred resolution, and provides typed executable routing with owner-validated selections. |
| 4 | Compose spec-type registries and amendments | DONE | `internal/spectypes/loader.go`, `registry.go`, and `export.go` validate declared owner extension points, apply lifecycle amendments to canonical types, and export the composed result without shadows. |
| 5 | Separate enabled retrieval, focus ranking, and routed ownership | DONE | Graph scope/stamp contracts now drive search, list, blocked, command selection, and handler-owned graph writes; focus affects result order while durable ownership survives context changes. |
| 6 | Add composed init, install, and lifecycle commands | DONE | `internal/cli/init.go`, `install.go`, and `domain.go` support `--with`, repeated targets, standalone packs, enable/disable/switch, inspection, reinstall, and rollback. |
| 7 | Route remaining primary-domain consumers through canonical config | DONE | CLI check/docs/scan/upgrade/cache paths and scan dispatch now use `ResolveDomains`/`PrimaryDomain`, retaining legacy compatibility throughout the application. |
| 8 | Add composition, compatibility, collision, graph, CLI, and harness validation | DONE | Tests cover six workspace modes, deferred local loading, ID/path collisions, routing ambiguity, applied amendments, handler-owned writes, default/focused/historical retrieval, rollback/pruning, and complete surfaces on all seven harnesses. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end with a freshly built Hero binary: initialized PM→Codex, QA→Claude, and Engineering + PM + QA→Cursor+Codex; loaded a deferred PM metrics specialist without installing it; created and searched a QA-owned TestPlan; verified disabled QA was hidden by default but explicitly retrievable; re-enabled QA; switched PM primary and verified QA remained enabled.

### Excellence Bar self-check

- [x] Yes — the implementation preserves primary-only compatibility, fails closed at composition boundaries, proves transactional lifecycle rollback, and exercises every bundled mode and supported harness from a fresh build.
