# Delivery audit — portable-routing-rules

**Audited:** `HEAD 9a3795d`; `git diff HEAD -- content.go domains/engineering/AGENTS.md internal/install/agents_md.go internal/install/agents_md_test.go .hero/planning/portable-routing-rules.md`, plus direct reads of the four new untracked delivery files
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — `domains/engineering/routing.md:1-56` is the standalone route source; the scoped diff removes the route table from `domains/engineering/AGENTS.md` and `generateEngineeringAgentsMdBody`.
- [✓] AC-2 — `domains/engineering/routing.md:3-56` contains the imperative routing rule, complete intent table, ambiguity instruction, slash/CLI distinction, mockup policy, and peering disambiguation without an external include.
- [✓] AC-3 — `internal/install/agents_md.go:215-228` adds one routing contributor to the shared native-root section pipeline; `TestRoutingGuidanceReachesAllHarnessNativeRoots` exercises Claude, OpenCode, Cursor, Copilot, Codex, and Generic and asserts routing markers in each selected native root.
- [✓] AC-4 — `internal/install/agents_md.go:80-92` retains one-target/one-native-root dispatch and the existing managed writer at `internal/install/agents_md.go:343-413` retains outside content and no-op reruns. `TestHarnessNative_PerTargetFileSet`, `TestRunClaude_PreservesUserAuthoredClaudeMd`, and the passing install suite cover exact root selection, user content preservation, and idempotency.
- [✓] AC-5 — `content.go:33` embeds the canonical file; `internal/install/routing_guidance.go:19-36` falls back to `hero.ContentFS()` rather than a handwritten route table; `TestRoutingGuidanceUsesCanonicalEmbeddedSourceAsFallback` compares the embedded title/body with the on-disk canonical source.
- [✓] AC-6 — routing is a normal contributor in `defaultSections` (`internal/install/agents_md.go:215-228`) and therefore participates in managed-region regeneration. `TestHarnessNative_Idempotent`, `TestHarnessNative_TwoNonClaudeTargetsShareAgentsMd`, and the passing install suite exercise byte-stable reruns.
- [✓] AC-7 — `internal/install/agents_md.go:502-536` retains the Codex-only workflow-to-`command-<name>` translation; `TestHarness_SmokeCodex` asserts the installed command skill and its root guidance while the canonical routing file remains target-neutral.
- [✓] AC-8 — `internal/serve/routing_reference_test.go:27-43` validates canonical workflow references against real command inventories; `internal/serve/routing_reference_test.go:45-79` proves invented workflow, MCP-tool, and installed-skill references fail; `internal/cli/markdown_drift_test.go:82-128` validates CLI invocations in the rendered root body against the real root command. Provided runs passed, including 908 CLI invocations with zero failures.
- [✓] AC-9 — `internal/install/routing_guidance.go` contributes inline content only; `TestRoutingGuidanceReachesAllHarnessNativeRoots` asserts that neither `.hero/routing.md` nor a Cursor routing sidecar is created.
- [✓] AC-10 — `TestRoutingGuidanceReachesAllHarnessNativeRoots` covers all six targets with the same routing policy markers; `TestEngineeringRoutingDoesNotLeakIntoOtherDomains` proves PM and sales roots omit engineering routing.

## Changes

- [✓] Add the standalone canonical engineering routing document — `domains/engineering/routing.md:1-56`.
- [✓] Add and wire the engineering-only managed-section contributor — `internal/install/routing_guidance.go:11-48`; `internal/install/agents_md.go:215-228`.
- [✓] Remove routing copies from the pack body and Go fallback — scoped deletions in `domains/engineering/AGENTS.md` and `internal/install/agents_md.go`; both now proceed from session-title guidance directly to key-workflow guidance.
- [✓] Embed the canonical source and validate fallback parity — `content.go:33`; `internal/install/routing_guidance_test.go:12-34`.
- [✓] Update pack regeneration and root-body tests — `internal/install/agents_md_test.go:57-95` composes the canonical routing source into roster validation; `RenderAgentsMdBodyForDriftTest` renders the contributor pipeline at `internal/install/agents_md.go:452-470`.
- [✓] Add the six-target native-root matrix — `internal/install/routing_guidance_test.go:36-76`, supported by exact native-file-set coverage in `TestHarnessNative_PerTargetFileSet`.
- [✓] Validate referenced surfaces — `internal/serve/routing_reference_test.go:13-168` covers workflow/MCP/skill inventories and negative cases; `TestMarkdownInvocationsResolveAgainstRootCmd` covers CLI references in rendered output.

## Open items

None. The Completion Ledger contains no PARTIAL, SKIPPED, or BLOCKED rows.

## Audit notes

None.
