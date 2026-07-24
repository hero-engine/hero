---
title: "Portable Routing Rules — One Canonical Route Source, Every Native Root"
slug: portable-routing-rules
type: feature
status: completed
domain: engineering
priority: critical
size: medium
horizon: now
created: 2026-05-17
updated: 2026-07-23
parent: conversational-attention-operability
tags: [install, routing, harness-integration, agents-md, claude-md, multi-harness]
relations:
  - target: harness-native-install-target-aware-upgrade
    kind: related
  - target: generated-command-refs-validated
    kind: related
  - target: attention-conversational-routes
    kind: blocks
delivery_method: manual
completed_at: 2026-07-24T01:05:29Z
---

# Portable Routing Rules — One Canonical Route Source, Every Native Root

## Context

Hero's engineering natural-language routing table is maintained twice: once in
`domains/engineering/AGENTS.md` and once in the last-resort Go fallback inside
`internal/install/agents_md.go`. A parity test catches divergence, but every
route edit still requires synchronized hand-editing and the fallback remains a
second authoring surface.

The original version of this spec proposed per-harness sidecars and native
include idioms. That design is now obsolete. Hero's accepted harness-native
install model writes the one root file each target reads—Claude gets
`CLAUDE.md`; OpenCode, Cursor, Copilot, Codex, and Generic get `AGENTS.md`—and
renders the same managed body into those files. Cursor rules, OpenCode config,
Copilot instructions, and Claude imports are additional surfaces, not the
portable root contract.

Routing must therefore be authored once, rendered inline through the existing
managed-section pipeline, and verified in all six native root files. A
hookless harness must receive enough imperative guidance to act without
following an import or loading a target-specific sidecar.

## Goal

Give the engineering domain one canonical, standalone routing source that the
existing managed-section renderer places inline in every supported harness's
native root instruction file, while preserving user-owned file content,
target-aware installation, and target-specific workflow execution guidance.

## Kickoff

Extract engineering natural-language routing into one canonical source and
compose it inline through the established six-target root-instruction pipeline.

**Status:** in review — canonical route source, embedded fallback, native-root
rendering, and validation are complete.

**Pick up at:** cold-audit the Completion Ledger, address any HOLD findings,
then run the delivery verification gate.

→ `/deliver portable-routing-rules`

**Files:** `domains/engineering/routing.md`,
`content.go`, `internal/install/routing_guidance.go`,
`internal/install/routing_guidance_test.go`, `domains/engineering/AGENTS.md`,
`internal/install/agents_md.go`, `internal/install/agents_md_test.go`,
`internal/serve/routing_reference_test.go`
**Skip:** target-specific route copies, `@` imports, Cursor `.mdc` routing
sidecars, OpenCode instruction-array mutations, and new root-file rules.

## Design

### Canonical engineering route source

Add `domains/engineering/routing.md` as the only human-authored source for:

- the imperative natural-language routing rule;
- the intent-to-workflow table;
- slash-command versus CLI distinctions;
- mockup and cross-repo peering disambiguation; and
- the instruction to carry the user's original context into the selected
  workflow and clarify genuinely ambiguous intent.

The document must be standalone and imperative. It must not assume a Claude
hook, a Cursor rule loader, or a native slash-command implementation. The
existing Codex workflow section remains responsible for translating routed
Hero workflows into installed `command-<name>` skills.

### Managed-section composition

Add an engineering-only routing section contributor to `defaultSections`.
It reads `routing.md` from the active domain content filesystem and renders its
bytes inline immediately after the pack body and before Attention lifecycle and
shared operational guidance.

The contributor is used by both `installAgentsMd` and `installClaudeMd`, so the
same content reaches:

- Claude in `CLAUDE.md`;
- OpenCode, Cursor, Copilot, Codex, and Generic in `AGENTS.md`.

It does not write `.hero/routing.md`, mutate harness configuration, or depend
on an include directive. Target-specific secondary instruction surfaces may
continue to exist, but they are not routing authority.

### One authored source, embedded fallback

Remove the natural-language routing section from
`domains/engineering/AGENTS.md` and from the handwritten
`generateEngineeringAgentsMdBody` body. The emergency no-content-filesystem
path still needs routing, so the contributor reads the same canonical
`routing.md` bytes from Hero's embedded engineering `ContentFS`.

Tests compare the on-disk canonical markdown to the embedded fallback
byte-for-byte. This preserves the fallback floor without producing or
maintaining a second artifact.

### Existing harness-native contract remains authoritative

This feature does not change `nativeInstructionFile`, installed-target
inference, orphan handling, global-mode support, or managed-region ownership.
An install writes only the native root file for the selected target, preserves
all user content outside Hero's managed markers, and regenerates the routing
section idempotently on install or upgrade.

Codex may append its target-specific workflow execution section after the
shared managed content. That translation is allowed; the semantic routing
source itself remains identical across targets.

## Changes

1. Add the standalone canonical engineering routing document at
   `domains/engineering/routing.md`.
2. Add an engineering-only managed section contributor that loads the
   canonical routing bytes and wire it into `defaultSections`.
3. Remove the routing table and routing-specific prose from
   `domains/engineering/AGENTS.md` and the handwritten engineering Go fallback.
4. Embed the canonical routing document in the engineering content filesystem,
   use those same bytes for the no-content-filesystem fallback, and add exact
   parity validation.
5. Update pack regeneration and root-body tests so route edits require one
   source edit and cannot silently omit the embedded fallback.
6. Add a six-target install matrix proving every native root receives the
   canonical route content and no target-specific route sidecar is required.
7. Validate every advertised workflow, CLI command, MCP tool, and installed
   skill reference used by the route source against Hero's real surfaces.

## Acceptance Criteria

- **AC-1:** WHEN an engineering routing rule is authored THE SYSTEM SHALL use
  `domains/engineering/routing.md` as the only human-edited source for the
  route table and its disambiguation policy.
- **AC-2:** WHEN the canonical routing source is rendered THE SYSTEM SHALL
  preserve its standalone imperative instruction, complete intent mapping,
  slash-versus-CLI distinction, and peering disambiguation without requiring
  another file to understand it.
- **AC-3:** WHEN Hero installs Claude THE SYSTEM SHALL inline the canonical
  routing content in `CLAUDE.md`; WHEN Hero installs OpenCode, Cursor, Copilot,
  Codex, or Generic THE SYSTEM SHALL inline the same content in `AGENTS.md`.
- **AC-4:** WHEN any one target is installed THE SYSTEM SHALL write only that
  target's native root instruction file and SHALL preserve byte-for-byte all
  user-owned content outside Hero's managed region.
- **AC-5:** WHEN the active content filesystem is unavailable THE SYSTEM SHALL
  render the embedded canonical `routing.md` bytes as its fallback and SHALL NOT
  rely on a second generated or handwritten route table.
- **AC-6:** WHEN the canonical routing source changes and install or upgrade is
  rerun THE SYSTEM SHALL refresh the managed routing section idempotently in
  every installed target's native root file.
- **AC-7:** WHEN the shared semantic route selects a Hero workflow in Codex THE
  SYSTEM SHALL retain the existing target-specific translation to installed
  `command-<name>` skills without changing the canonical route definition.
- **AC-8:** WHEN route-reference validation runs THE SYSTEM SHALL fail for any
  advertised workflow, CLI subcommand, MCP tool, or installed skill that does
  not exist on the referenced surface.
- **AC-9:** WHEN routing is installed THE SYSTEM SHALL NOT require
  `.hero/routing.md`, a Claude `@` import, a Cursor routing `.mdc`, an OpenCode
  `instructions` entry, or any other harness-specific include to reach the
  model.
- **AC-10:** WHEN the six-target install matrix runs THE SYSTEM SHALL prove
  equivalent routing markers and policy text in every native root surface and
  SHALL prove a non-engineering domain does not inherit engineering routes.

## Boundaries

- No user-configurable routing DSL or per-repository custom route merge.
- No new harness target or change to native root-file selection.
- No target-specific routing authority, include directive, or sidecar.
- No live model evaluation; validation proves deterministic content and
  reference correctness.
- No changes to Mail, Focus, Attention, or other feature-specific route
  semantics; their owning specs add rows and conformance cases to this source.
- No removal of target-specific execution guidance that translates a selected
  workflow into the harness's supported mechanism.

## Risks

- **Fallback drift:** an embed declaration can omit the canonical file. Exact
  on-disk-versus-embedded parity tests must fail before merge.
- **Section-order regression:** moving routing too late or outside the managed
  block can weaken visibility. The contributor order and native-root golden
  tests pin placement.
- **Surface confusion:** slash workflow names and CLI commands look similar.
  Reference validation and explicit surface distinctions must stay in the
  canonical source.
- **Cross-domain leakage:** engineering routes are inappropriate for another
  domain pack. Contributor selection must be domain-aware.
- **Partial harness coverage:** a change tested only in Claude can appear done
  while five targets drift. The six-target matrix is a release gate.

## Validation

- `go test ./internal/install`
- `go test ./...`
- Exact canonical-on-disk-versus-embedded-fallback parity test.
- Six-target project-install matrix checking native root filename, routing
  content, managed-region ownership, and idempotent rerun.
- Non-engineering install test proving engineering routing is absent.
- Reference validator over every workflow, CLI, MCP, and skill token in the
  canonical route source.
- Repository search proving no old handwritten routing table or new
  target-specific routing sidecar remains.

## Completion Ledger

Portable engineering routing now has one human-authored source. The existing
managed-section pipeline renders it to every harness-native root, while an
embedded copy of those same bytes preserves the emergency fallback path.
Validation covered focused install behavior, command-reference drift, the
entire install package, and the full Go repository suite.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Use `domains/engineering/routing.md` as the only human-edited routing source | DONE | `domains/engineering/routing.md` owns the complete route table and disambiguation policy; the former copies were removed from `domains/engineering/AGENTS.md` and `generateEngineeringAgentsMdBody`. |
| 2 | Preserve standalone imperative routing, intent mapping, surface distinctions, and peering rules | DONE | `domains/engineering/routing.md` is self-contained and includes imperative routing, the full intent table, slash-versus-CLI guidance, mockup routing, and cross-repo peering disambiguation. |
| 3 | Inline the same canonical content in all six harness-native roots | DONE | `TestRoutingGuidanceReachesAllHarnessNativeRoots` proves Claude receives `CLAUDE.md` and OpenCode, Cursor, Copilot, Codex, and Generic receive `AGENTS.md`, all containing the same routing markers. |
| 4 | Write only the selected native root and preserve user-owned content | DONE | The routing contributor uses the unchanged native-root managed-region pipeline; the six-target matrix asserts only each selected native root exists, while the existing install suite continues to cover managed-region preservation. |
| 5 | Fall back to embedded canonical bytes without a second route table | DONE | `internal/install/routing_guidance.go` reads `domains/engineering/routing.md` from `hero.ContentFS()` when the active content filesystem is unavailable; `TestRoutingGuidanceUsesCanonicalEmbeddedSourceAsFallback` proves byte-for-byte parity. |
| 6 | Refresh routing idempotently on install or upgrade | DONE | Routing is a normal `managedSection` in `defaultSections`; the unchanged idempotent managed-region writer is exercised by the passing `internal/install` suite. |
| 7 | Retain Codex translation to installed `command-<name>` skills | DONE | The existing Codex workflow section remains target-specific and is rendered after shared content; the canonical routing source is unchanged by that translation. |
| 8 | Fail validation for invalid advertised command references | DONE | `TestCanonicalRoutingReferencesResolveAgainstRealSurfaces` checks workflows, MCP tools, and installed skills against their real source/runtime inventories; `TestRoutingReferenceValidationRejectsUnknownSurfaceNames` proves an invented reference on each surface fails. `TestMarkdownInvocationsResolveAgainstRootCmd` validates literal CLI invocations against the real root command. |
| 9 | Require no routing sidecar or harness-specific include | DONE | `TestRoutingGuidanceReachesAllHarnessNativeRoots` asserts no `.hero/routing.md` or Cursor routing sidecar is written; the implementation only contributes inline managed content. |
| 10 | Prove six-target equivalence and exclude non-engineering domains | DONE | `TestRoutingGuidanceReachesAllHarnessNativeRoots` covers all six targets; `TestEngineeringRoutingDoesNotLeakIntoOtherDomains` proves PM and sales packs do not inherit engineering routing. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add the standalone canonical engineering routing document | DONE | Added `domains/engineering/routing.md` with the authoritative routing contract. |
| 2 | Add and wire an engineering-only managed-section contributor | DONE | Added `internal/install/routing_guidance.go` and placed its section immediately after the pack body in `defaultSections`. |
| 3 | Remove routing copies from the pack body and Go fallback | DONE | Regenerated `domains/engineering/AGENTS.md` without the route table and removed routing prose from `generateEngineeringAgentsMdBody`. |
| 4 | Embed the canonical source and validate exact fallback parity | DONE | Updated `content.go`; `TestRoutingGuidanceUsesCanonicalEmbeddedSourceAsFallback` verifies the active and fallback sources match exactly. |
| 5 | Update pack regeneration and root-body tests | DONE | Updated `internal/install/agents_md_test.go` and the drift-test renderer to compose the canonical routing section. |
| 6 | Add a six-target native-root install matrix | DONE | Added `TestRoutingGuidanceReachesAllHarnessNativeRoots`, including native filename and no-sidecar assertions. |
| 7 | Validate referenced surfaces | DONE | The surface-aware routing validator passes against real workflow, MCP, and installed-skill inventories and its three negative cases reject invented names; the CLI Markdown reference test passes with 908 invocations checked and zero failures. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: the six-target install
  matrix rendered canonical routing into each target's native root, proved no
  routing sidecars were created, and the non-engineering matrix proved domain
  exclusion. Focused routing tests, `go test ./internal/install`,
  `TestCanonicalRoutingReferencesResolveAgainstRealSurfaces`, all three
  negative surface-reference cases, `TestMarkdownInvocationsResolveAgainstRootCmd`
  (908 checked, 0 failed), and `go test ./...` all passed.

### Excellence Bar self-check

- [x] Yes — one authored source now drives every native harness root and the
  emergency fallback, with deterministic parity and cross-target coverage.
