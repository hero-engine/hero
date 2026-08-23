---
title: Hero Uses Hybrid Content Packs and Separate Workflow Providers
type: decision
status: accepted
created: 2026-08-22
accepted: 2026-08-22
tags: [architecture, domains, packs, plugins, providers, distribution, spec-kit]
relations:
  - target: multi-domain-context-activation
    kind: supersedes
  - target: dual-mode-pm-qa-capability-packs
    kind: related
  - target: domain-plugin-architecture
    kind: related
  - target: spec-kit-and-swappable-workflow-providers
    kind: related
  - target: chat-pack-disposition
    kind: related
  - target: hero-domains
    kind: related
---

# Hero Uses Hybrid Content Packs and Separate Workflow Providers

## Decision

Hero will use a hybrid distribution model built around three distinct concepts:

1. **Core** is the small, universal, version-coupled layer embedded in every Hero release.
2. **Domain packs** supply the language, agents, skills, spec types, and views for a kind of work.
3. **Workflow providers** run a spec or delivery lifecycle and report normalized events and evidence back to Hero.

`common` is Core, not a selectable pack. The engineering/code domain is the embedded default and recovery pack. It includes lightweight planning and quality assistance needed during ordinary coding. Full PM and QA packs can be enabled on top of engineering or selected as the primary experience in dedicated workspaces. Other first-party domains can be acquired as immutable, versioned packages and then resolved locally. External systems such as GitHub Spec Kit are integrated through separately installed provider adapters; their native content and execution remain owned by those systems.

The public pack should use an identity such as `hero/engineering`; **Hero Code** remains the name of the proprietary client and is not the owner or distribution boundary of that pack. Likewise, Hero Cloud can consume public package contracts without becoming part of the open-source package system.

“On demand” means explicit acquisition at an install, enable, or update boundary. It does not mean downloading content during an agent turn or ordinary workflow execution. Once resolved, Hero and the installed harness artifacts operate from local, hash-pinned content without requiring the network.

## Context

Hero currently embeds universal content from `core/` and the engineering, PM, and sales domains into the Go binary. Installation overlays one active domain on Core and materializes rendered copies for every supported harness. The domain manifest model already declares `builtin`, `project`, `plugin`, and `remote` sources, but plugin and remote resolution are not implemented. Install state records rendered files, not the identity and digest of the content packages that produced them.

This is a useful starting point but it conflates several decisions:

- whether content is first-party or third-party;
- whether it is bundled or acquired separately;
- whether it describes a work domain or executes a workflow;
- whether a package is present on the machine, enabled in a workspace, or active for the current operation.

The distinction matters. Marketing, sales, QA, and PM are domain or extension content. Spec Kit, OpenSpec, BMAD, or an organization-specific RFC process are workflow providers. A workspace can use the engineering domain with either Hero's native workflow or Spec Kit. Treating both as one generic plugin type would couple unrelated lifecycles and create ambiguous composition rules.

GitHub Spec Kit is also a substantial independent product. Its `specify` CLI owns project initialization and agent integrations, and its extension system has catalogs, independently installable packages, enable/disable state, presets, workflows, and its own compatibility rules. Hero should not vendor that implementation or attempt to reproduce its package manager inside a domain pack. The integration boundary is lifecycle observation and evidence exchange.

## Decision Drivers

- A new Hero workspace must remain useful offline and without an account.
- The default engineering experience must not depend on a registry being reachable.
- Teammates and CI must resolve the same content, not whichever version was latest when a command ran.
- Optional first-party packs need a release cadence independent of the Hero binary.
- External systems must retain their native artifacts, concepts, licensing, and upgrade path.
- A remote package must not gain arbitrary code execution merely by being described as a skill or pack.
- Installation must continue to produce deterministic, target-native files for all seven harnesses.
- The design must support proprietary clients without making proprietary repositories the source of truth for the public package contract.

## Package and Provider Model

### Core

Core contains only capabilities that every Hero workspace needs: shared contracts, context and evidence primitives, generic capture and continuity behavior, manifest validation, package resolution, and harness rendering. Core is embedded and versioned with the Hero binary.

Core is intentionally small. A feature does not belong in Core merely because several domains currently use it. Shared domain behavior can live in an explicit extension pack with declared capabilities and dependencies.

### Domain packs

A domain pack is data-first content: Markdown instructions, YAML manifests, templates, schemas, and static assets. It declares a stable namespaced ID, semantic version, Hero API compatibility range, dependencies, capabilities, file digests, license, publisher, and legal overlay points.

PM and QA are dual-mode capability packs. As extensions, they add their full specialist capabilities while preserving an engineering-first workspace. As the primary pack, the same content produces a dedicated PM- or QA-first workspace with different default navigation and routing. Hero does not publish separate `pm-lite`/`pm-full` or `qa-lite`/`qa-full` content forks. Lightweight planning and quality behavior belongs to engineering; the full packs add specialist artifacts, integrations, views, and agents without copying those essentials.

Sales and marketing remain candidates for primary domain packs. A pack's permitted composition roles are manifest data, not a different package format or repository layout.

Resolution order is deterministic:

1. embedded Core;
2. exactly one active primary domain;
3. explicitly enabled extension packs;
4. project-owned overrides.

`primary` and `extension` describe activation in a workspace, not separate package identities. For example, `hero/pm` may be primary in a product workspace or an extension in an engineering workspace. The package version, artifact IDs, and source content remain the same in both cases.

Shared commands are installed once as routers. Packs contribute namespaced routing handlers and artifact predicates to those routers instead of replacing files such as `design`, `deliver`, or `review`. Shared spec types likewise have one canonical identity with pack-owned lifecycle or vocabulary amendments. Exact artifact-ID conflicts fail validation unless the owning artifact declares an extension point; file order never decides behavior.

The engineering/code domain remains embedded as the default and recovery pack. This preserves the current zero-download experience and gives Hero a known-good repair path. Other first-party packs may be bundled in a release for convenience, but bundling is a package source, not a different semantic model: bundled, cached, local-development, and remote packages all pass through the same manifest validation and resolver.

First-party pack source should remain in the Hero monorepo initially so cross-pack contracts and all harness outputs can change atomically. The release pipeline can publish separate immutable artifacts from those directories. A pack does not need its own repository merely because it has its own version and distribution artifact.

### Workflow providers

A workflow provider is selected independently from the domain stack. It owns its native phases, prompts, files, terminology, and execution model. Hero normalizes only the boundary needed for continuity:

- context supplied to the provider;
- workflow and phase lifecycle events;
- artifacts created or updated;
- assumptions, decisions, and failed attempts;
- approval requests;
- verification evidence and completion outcome;
- pause, resume, cancel, and failure state.

Provider-reported completion is not automatically Hero-verified completion. Hero retains responsibility for provenance, durable memory, and the evidence standard it presents to later sessions.

The first external integration should be a narrow Spec Kit adapter that detects or invokes an independently installed, version-compatible `specify` CLI. Hero should not copy Spec Kit templates or install Spec Kit transitively. The provider contract should remain experimental until Hero-native and Spec Kit-backed workflows have both exercised it on real changes.

## Acquisition and Local Resolution

A workspace records desired packages and provider selection in committed configuration. A committed lock records the exact version, source, package digest, manifest API version, and resolved dependency graph. The precise filenames are an implementation detail, but requirements and resolution must be separate so an explicit update can produce a reviewable lock change.

Artifacts are stored in a machine-level content-addressed cache. Enabling a pack or provider performs these steps atomically:

1. resolve the requested version against trusted catalogs or an explicit source;
2. download or copy the immutable artifact;
3. verify its digest, manifest, compatibility, license metadata, and signature when available;
4. stage and validate the complete dependency graph and all harness outputs;
5. update the workspace lock only after validation succeeds;
6. retain the previous lock and cached artifacts for rollback.

Ordinary commands use only the lock and local cache. If a locked artifact is absent, Hero fails with a precise fetch or restore instruction. It does not silently select `latest`, fall back to a different pack, or access the network. Updates are explicit and reviewable.

Package state is modeled separately:

- **available**: discoverable from a bundled source or catalog;
- **installed**: present and verified in the local cache;
- **enabled**: selected by workspace configuration;
- **active**: chosen by the resolved domain/provider stack for this operation.

## Trust and Extension Boundary

Remote domain packages are non-executable in the first version. They may contain declarative content and static assets, but no shell hooks, native libraries, Go plugins, or arbitrary scripts. Conflicts fail closed unless a manifest declares an override point and the target pack explicitly allows it. Cross-pack amendments name their target and a compatible capability or version range; unrelated packages do not compose by “last one wins.”

Catalogs are discovery and policy indexes, not sources of mutable truth. Packages are immutable and digest-pinned. Hero may ship a curated first-party catalog and permit organization-owned catalogs. Direct URL or Git sources are allowed only as an explicit local-development or untrusted path with an exact digest and warning.

Provider adapters that need executable behavior run an external, independently installed process through a narrow adapter with explicit capabilities and permissions. Hero does not load untrusted code into its own process. The open-source Hero repository owns the versioned manifest and provider boundary; proprietary clients may consume that contract but do not define it.

## Options Considered

### Embed every first-party pack

This is simple, fast, and fully offline. It also couples every domain release to the Hero binary, grows the default product with immature or irrelevant material, and cannot cleanly accommodate external systems or their licenses. It is acceptable as a temporary packaging detail, not as the long-term architecture.

### Pull every pack, including engineering

This minimizes the binary and gives every pack an independent cadence. It makes first use, recovery, CI, and air-gapped work depend on registry and cache state. It also creates needless supply-chain exposure for Hero's primary experience. This option is rejected.

### Use one plugin system for domains and spec frameworks

This appears uniform but erases the most important boundary. Domain content composes declaratively with Core; a workflow system has executable lifecycle authority and native artifacts. A common plugin abstraction would either be too weak for providers or too permissive for content packs. This option is rejected.

### Hybrid embedded core/default plus locked packages and providers

This preserves the reliable first-run experience, gives optional content an independent lifecycle, and isolates external workflow systems behind a small integration contract. It adds package resolution, cache, and lockfile work, but those mechanisms are localized and can be introduced without changing current rendered output. This option is selected.

## Consequences

### Positive

- Hero remains useful offline from the first command.
- Optional domains do not bloat or delay every Hero release.
- Teams get reproducible pack and provider versions across machines and CI.
- External providers can evolve without Hero vendoring their implementation.
- The package trust model is materially smaller than a general code-plugin system.
- Existing embedded packs can migrate through an adapter with output-parity tests rather than a flag-day rewrite.
- Proprietary Hero clients can carry private packs while sharing the public contract and resolver semantics.

### Negative

- Hero must own a package manifest, resolver, cache, lock format, and rollback behavior.
- First-party pack publishing needs release automation and compatibility testing.
- Users must make one explicit acquisition step for an optional pack or external provider.
- Supporting multiple providers creates a durable integration test matrix.
- Data-only packs limit advanced integrations; executable behavior must use the stricter provider or external-tool boundary.

### Risks

- Prematurely standardizing the provider API around Hero's own workflow would produce a disguised Hero-specific interface.
- Allowing unconstrained precedence would make installed combinations nondeterministic.
- Treating a catalog listing as trust would expose users to unaudited third-party content.
- Automatically fetching during workflow execution would turn registry outages into mid-task failures.
- Keeping `latest-wins` semantics in source references would undermine the lock model.

## Migration Path

1. Formalize the terminology and versioned public manifests for Core, domain packs, workspace composition roles, extension points, and provider adapters.
2. Evolve the current Core-plus-active-domain resolver into Core plus one primary pack plus zero or more extensions. Reconcile PM and QA collisions into canonical shared routers, shared spec identities, or namespaced specialist artifacts.
3. Introduce a `PackageSource`/resolver boundary around the existing `fs.FS` install seam. Adapt embedded Core, engineering, PM, and sales through it without changing generated files.
4. Add committed requirements and lock state plus a content-addressed local cache. Prove local-directory and embedded sources before adding networking.
5. Publish one non-default first-party domain through the same artifact path and verify clean install, offline reuse, update, rollback, stale-file pruning, and all seven harness outputs.
6. Build an opt-in Spec Kit adapter that integrates with an independently installed CLI. Dogfood it on 5–10 meaningful changes alongside the Hero-native provider.
7. Stabilize the provider contract only after the two implementations expose the real common boundary.
8. Add signed remote acquisition and curated catalogs. Do not begin with a public marketplace or arbitrary executable plugins.

## Revisit Triggers

Revisit this decision if any of the following becomes true:

- the embedded engineering pack becomes large enough to materially affect Hero distribution;
- two real providers cannot share the proposed lifecycle/evidence boundary without losing essential behavior;
- a domain requires executable behavior that cannot safely live in an external integration;
- pack dependencies routinely require multiple primary domains rather than one domain plus extensions;
- content signing or catalog operation imposes more cost than the independent release cadence saves.
