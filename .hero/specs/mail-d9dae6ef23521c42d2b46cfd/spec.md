---
title: "Canonical agent-purpose metadata for model routing"
slug: mail-d9dae6ef23521c42d2b46cfd
type: feature
status: completed
domain: engineering
priority: high
size: medium
created: 2026-08-26
tags: [agents, metadata, model-routing, embedded-content, hero-code-peer]
relations:
  - target: mail-d9dae6ef23521c42d2b46cfd
    kind: derived_from
delivery_method: manual
completed_at: 2026-08-26T07:54:30Z
---

# Canonical agent-purpose metadata for model routing

## Context

Hero Code's `declared-agent-purpose-model-routing` delivery found that every
shipped built-in agent originates in this repository under `core/agents/` or
`domains/{engineering,pm,sales}/agents/`. Hero Code's bundled copies are
generated and ignored, so assigning purpose in the client or in an amendment
table would create a second slug-to-policy source that drifts from the agent
definition.

This peer-owned feature adds purpose at the canonical descriptor. Hero Code
will consume the field through its existing `ModelCategory` and model resolver;
it remains responsible for custom-agent fallback and runtime routing.

## Goal

Give every shipped canonical agent an explicit, validated model-routing purpose
that survives raw pack extraction, without introducing name, slug, or natural-
language inference and without changing any harness's agent behavior.

## Kickoff

Adds explicit model-routing purpose to Hero's canonical agent descriptors so
Hero Code can route agents without guessing from their names.

**Status:** delivering — canonical descriptors, validation, and install-contract
coverage, peer extraction, and the required threaded Mail reply are complete.

**Pick up at:** complete the descriptor inventory and source validator, then run
the focused install tests and version-matched extraction gate.

→ `/deliver mail-d9dae6ef23521c42d2b46cfd`

**Files:** `core/agents/`, `domains/engineering/agents/`, `domains/pm/agents/`, `domains/sales/agents/`, `internal/install/content.go`
**Skip:** central slug/name lookup tables and inferred purpose from descriptions.

## Problem

Agent descriptors declare identity, description, execution mode, and
permissions, but not the kind of model work they perform. A consumer therefore
has to guess from the agent name or use one generic category. The former is
brittle and the latter ignores the model assignments users configured for
design, diagnosis, review, drafting, and lightweight assistance.

The embedded-content manifest hashes raw descriptor bytes but does not own
semantic agent metadata. The purpose must remain in the descriptor itself and
be checked before any target-specific renderer or downstream extraction uses
it.

## Approach

- Add one `purpose:` scalar to every installable canonical agent descriptor.
- Use the closed portable vocabulary `design | diagnose | agent | draft |
  review | assist`, matching the routed sub-agent categories Hero Code already
  exposes. The runtime-only `chat`, `goalEvaluator`, and `embed` categories are
  not valid agent purposes.
- Classify from the role's actual contract. Delivery leads and architects that
  coordinate or author designs use `design`; investigators use `diagnose`;
  implementers and operational executors use `agent`; document/spec prose
  authors use `draft`; critics and validation roles use `review`; bounded
  context/report helpers use `assist`.
- Validate canonical source while enumerating/installing content, before
  harness-specific rendering. Raw Markdown remains the single semantic source;
  list manifests continue to identify and hash files rather than duplicating
  routing policy.
- Preserve current target output contracts. Purpose metadata must not change
  prompts, permissions, model declarations, or target eligibility.

## Changes

1. Add `purpose:` to every shipped descriptor in `core/agents/`,
   `domains/engineering/agents/`, `domains/pm/agents/`, and
   `domains/sales/agents/`.
   - Classify each role from its stated responsibilities, not its filename.
   - Exclude directory `README.md` files and domains with no shipped agents.
2. Extend the shared canonical-agent frontmatter reader in
   `internal/install/content.go` (or a narrowly named adjacent file) with a
   typed closed purpose vocabulary.
   - Reject a missing, empty, or unknown purpose for canonical pack agents.
   - Keep the parser bounded to leading YAML frontmatter and reuse the same
     selection path used by install and `ContentManifest` enumeration.
3. Add focused tests in `internal/install/content_test.go`,
   `internal/install/manifest_test.go`, and target install contract tests.
   - Inventory all current canonical descriptors dynamically and prove each
     has exactly one allowed purpose.
   - Prove missing and unknown values fail closed before installation.
   - Prove every supported install target still materializes its native agent
     format successfully; do not require target runtimes to interpret purpose.
4. Publish the raw-source contract to the Hero Code peer.
   - Build the current Hero binary and regenerate Hero Code's `hero-content`
     from the same source revision so descriptor bytes and manifest hashes
     remain version-matched.
   - Reply on Project Mail with the source revision, purpose vocabulary, and
     validation evidence after delivery verifies.

## Acceptance Criteria

- **AC-1:** WHEN canonical agent content is enumerated THE SYSTEM SHALL find exactly one explicit `purpose` value on every installable agent descriptor.
- **AC-2:** THE SYSTEM SHALL accept only `design`, `diagnose`, `agent`, `draft`, `review`, or `assist` as canonical agent purposes.
- **AC-3:** IF a canonical descriptor omits purpose or declares an unknown value THEN THE SYSTEM SHALL reject the content before target installation or downstream extraction.
- **AC-4:** WHEN any supported harness target installs canonical agents THE SYSTEM SHALL preserve its existing native agent contract and behavior.
- **AC-5:** THE SYSTEM SHALL keep purpose policy in each authoritative descriptor and SHALL NOT introduce a central filename, slug, description, or display-name mapping.
- **AC-6:** WHEN Hero Code extracts version-matched raw content THE SYSTEM SHALL preserve each descriptor's purpose inside the hashed source bytes.

## Boundaries

- No Hero Code Swift routing changes in this peer feature.
- No model IDs, provider assignments, temperatures, permissions, prompts, or
  agent availability changes.
- No new purpose categories and no purpose inference from prose.
- No purpose duplication in the list-only `ContentManifest`; the descriptor
  and its manifest hash remain authoritative.
- Custom project agents are not canonical pack content and may continue to omit
  purpose; Hero Code owns their documented generic fallback.

## Risks

- A role can legitimately design and deliver. Classification must follow its
  primary contract, and future role changes must update the descriptor field
  explicitly rather than adding inference.
- A validator wired after target rendering would miss or misread transformed
  formats; validate the canonical Markdown source first.
- Hero and Hero Code source revisions must match during extraction or the
  existing pack integrity gate will correctly refuse regeneration.

## Validation

- Run focused installer/content/manifest tests for canonical purpose parsing,
  missing/unknown rejection, dynamic inventory, and target compatibility.
- Run `go test ./internal/install/...` and the existing install-contract smoke
  for all supported targets.
- Build the Hero binary with the repository's version stamp and use it for one
  Hero Code content extraction; verify manifest hashes and parsed purposes.
- Run `hero spec lint`, `hero spec score`, `hero drift`, and
  `hero spec verify mail-d9dae6ef23521c42d2b46cfd` after the cold audit.

## Completion Ledger

Implementation covered the Go installer and the canonical Core, Engineering,
PM, and Sales descriptor packs. Validation completed with focused purpose
tests, the full installer package, and the repository-wide Go suite. The local
binary was rebuilt with the current dirty revision stamp, and Hero Code
successfully extracted that exact binary/source revision. The versioned raw-
source contract and validation evidence were published back to Hero Code on
the originating Project Mail thread as `mail_522b3d0fcc03d8062bfb9120`.
Spec lint reports 6/6 EARS criteria, score is 95/100 (A), and drift reports no
warnings.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Every installable canonical descriptor declares exactly one purpose | DONE | `TestCanonicalAgentPurposesCoverDynamicInventory` enumerates every non-README descriptor from Core, Engineering, PM, and Sales and validates each source file. |
| 2 | Only the six portable purpose values are accepted | DONE | `AgentPurpose` and `TestCanonicalAgentPurposeAcceptsClosedVocabulary` cover `design`, `diagnose`, `agent`, `draft`, `review`, and `assist`. |
| 3 | Missing, empty, duplicate, or unknown purpose fails before canonical install | DONE | Every production CLI caller supplies canonical `ContentFS`, which `Run` validates before target dispatch without content sniffing. `TestCanonicalAgentPurposeRunRejectsEntirelyMissingPackBeforeWrites` exercises the real boundary; deprecated custom/test `SourceDir` remains explicitly lenient. |
| 4 | All seven harnesses preserve their native agent contracts | DONE | `TestCanonicalAgentPurposeInstallContractsAllTargets` installs and contract-validates Claude, OpenCode, Cursor, Copilot, Codex, Generic, and Grok outputs; the full installer suite also passed. |
| 5 | Purpose stays descriptor-owned without a name/slug/description map | DONE | Each descriptor owns its literal `purpose:`; `internal/install/agent_purpose.go` contains only the typed vocabulary and frontmatter validation. `ContentManifest` was not changed. |
| 6 | Version-matched raw extraction preserves purpose in hashed bytes | DONE | Hero Code extracted `v0.33.0-27-g4cb6c40f-dirty` from the same stamped local binary and source revision; a post-extraction check matched all 70 manifest hashes to their descriptor bytes and found exactly one purpose declaration in each. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add purpose to Core, Engineering, PM, and Sales descriptors | DONE | Updated every dynamically enumerated non-README descriptor under the four specified agent roots; delivery leads and PM design coordinators use `design`. |
| 2 | Add typed canonical frontmatter validation | DONE | Added `internal/install/agent_purpose.go` and the pre-dispatch validation call in `internal/install/install.go`; canonical `ContentFS` fails closed even when every required purpose is absent, while QA-only descriptors and custom `SourceDir` agents remain outside the contract. |
| 3 | Add inventory, rejection, and seven-target tests | DONE | Added `internal/install/agent_purpose_test.go` and updated two synthetic canonical fixtures with valid purpose metadata. `go test ./internal/install/...` and `go test ./...` pass. |
| 4 | Publish raw-source contract and regenerate Hero Code content | DONE | Built and successfully extracted the version-matched source as `v0.33.0-27-g4cb6c40f-dirty`, then replied on the originating Project Mail thread with the vocabulary, 70-file hash/purpose evidence, seven-target compatibility, production `/deliver` routing evidence, and SHIP verdict (`mail_522b3d0fcc03d8062bfb9120`; idempotency key `agent-purpose-routing-v0.33.0-27-g4cb6c40f`). |

### Exercise-the-feature check

- [x] Canonical source validation and every native harness agent contract were exercised with `go test ./internal/install -run '^TestCanonicalAgentPurpose'`; the full installer and repository suites also passed.

### Excellence Bar self-check

- [x] Yes for the Hero-owned slice: the policy is descriptor-owned, closed, dynamically checked, and covered across all seven targets without adding a central role-name mapping or changing `ContentManifest`.
