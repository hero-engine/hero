# Hero positioning authority

Status: canonical messaging source for the v0.34 public refresh; publication remains gated
Evidence baseline: `.hero/specs/hero-public-truth-baseline/public-claim-registry.yaml`
Last reviewed: 2026-08-23

This document governs public product language. Downstream pages may shorten it, but they must preserve its hierarchy, evidence labels, and product boundaries.

## Position in one sentence

Hero gives AI coding tools durable project memory and a verified delivery system, so decisions survive sessions and agents finish work against evidence.

## Category and outcome

**Category:** project memory and delivery system for AI-assisted engineering.

**Internal category shorthand:** an operating layer for AI-assisted engineering. Do not use “spec layer” as the category.

**Outcome:** each AI session can inherit the project's intent, decisions, corrections, conventions, evidence, and current state, then use a structured delivery system to complete work and leave trustworthy context for the next session.

Hero is two connected systems in one:

1. **Durable project memory is the headline.** Hero keeps the parts of a project that prompts, chat history, and tool-local rule files routinely lose: why decisions were made, which corrections matter, what conventions apply, what failed, what evidence exists, and what should happen next.
2. **Verified delivery is the execution layer.** Specs, specialized agents, Completion Ledgers, cold audits, and verification use that project memory to move work from intent to tested completion.

The reinforcing loop is the product model: memory informs delivery; delivery produces decisions, evidence, corrections, and current state; those artifacts improve later memory (`product-two-system-loop`, Class A, shipped).

## Audience

### Lead audience

**AI-native engineers and hands-on technical leads working in long-lived codebases.** They use coding agents for real implementation work, switch between sessions or tools, and remain accountable for correctness. They feel the cost of repeated explanation, forgotten decisions, context reconstruction, partial delivery, and work declared complete without evidence.

Every general public page leads with this audience. “Teams,” executives, and broad knowledge workers are not co-equal lead audiences.

### Secondary audiences

- **Engineering leads adopting AI across a team:** care about shared conventions, inspectable decisions, reliable handoffs, and a consistent completion bar. Team/cloud capabilities must not be implied; the shipped proof is repository-local and harness-native.
- **Developer productivity and platform engineers:** care about repeatable installation, project context, integration boundaries, and workflows that survive tool changes.
- **Maintainers of multi-repository systems:** may use optional one-graph-per-project peering, asynchronous Project Mail, advisory/spec-out calls, and explicit handoffs after setup (`cross-repo-peering`, Class A, optional).

### Expansion audiences

PM and QA are expansion paths after the engineering wedge. The default Engineering setup includes lightweight PM and QA help used inside coding work (`engineering-default-pack`, Class A, shipped). Focused PM and QA setups can also be installed alone or composed where supported (`optional-domain-packs`, Class B, optional). Public copy must label those focused packs as optional and maturity-bounded, not as proof that Hero is already a broad departmental platform. Sales follows the same expansion rule and is not part of the v0.34 lead story.

## Jobs to be done

When I start or resume work with an AI coding tool, help it recover the project's current truth without making me reconstruct the whole project in a prompt.

When a decision, correction, convention, failure, or piece of evidence matters later, preserve it in a project-owned form that another session can retrieve.

When I ask an agent to build or fix something, turn the request into bounded work, equip the right specialist, and require evidence before calling it complete.

When I change coding tools, keep project knowledge and delivery discipline in the repository instead of trapping them in one vendor's chat history.

When work crosses repositories or external trackers, keep boundaries explicit and require setup and consent for optional operations.

## Messaging house

### Roof: durable continuity with a completion bar

**Your project remembers. Your agents deliver.** Hero carries project truth between AI sessions and connects it to a delivery system that closes work against evidence.

### Pillar 1: project memory that survives the session

- Preserve intent, decisions, corrections, conventions, evidence, failures, and current state.
- Retrieve relevant project context instead of relying on one giant prompt or one tool's chat history.
- Keep the project—not an individual model session—as the durable unit.

Proof: `product-memory-system` — Class A, shipped, requires a Hero workspace and a supported harness or CLI entry point.

### Pillar 2: verified delivery, not ceremonial specs

- Use specs to bound intent and acceptance, not to add paperwork for its own sake.
- Route work through appropriate agents and project conventions.
- Require a Completion Ledger, a fresh cold audit, and build/test gates before verified completion.

Proof: `product-delivery-system` — Class A, shipped, requires the Engineering pack and active-harness agent execution.
Proof: `cold-audit-and-verify` — Class A, shipped, requires a delivery workflow and testable project.

### Pillar 3: harness-native and project-owned

- Install workflows into the surfaces each supported harness actually uses.
- Keep specifications, knowledge, and handoff state in the project.
- Change AI tools without discarding the project's operating context.

Proof: `harness-native-workflows` — Class A, shipped. Say “harness-native workflows”; never promise universal slash commands.
Proof: `supported-install-targets` — Class A, shipped for the target-specific host surfaces recorded by the installer.

### Supporting capabilities, not the headline

- Bounded Attention, Project Mail, and private Focus (`attention-mail-focus`, Class A, shipped with stated semantics).
- Local project intelligence through `hero serve` (`hero-serve-intelligence`, Class A, shipped; team/external access is separate).
- Configured tracker and code-host operations (`tracker-evidence-and-mutations` and `code-host-operations`, Class A, optional with credentials and consent).
- Headless agent supervision (`headless-runtime`, Class A, preview with provider and execution prerequisites).

These capabilities support the memory-and-delivery story. Do not turn a mutable feature count into the value proposition.

## Proof register

| Public proof | Baseline claim | Evidence | Availability | Required qualifier |
|---|---|---|---|---|
| Project intent and decisions persist beyond a session | `product-memory-system` | A | shipped | Hero workspace plus supported harness or CLI |
| Specs and agents structure delivery | `product-delivery-system` | A | shipped | Engineering pack; agent execution depends on harness |
| Completion requires ledger, cold audit, and build/test gates | `cold-audit-and-verify` | A | shipped | Testable project and delivery workflow |
| Workflows adapt to supported tool surfaces | `harness-native-workflows` | A | shipped | Never call them universal slash commands |
| Default setup is Core plus Engineering with lightweight PM/QA assistance | `engineering-default-pack` | A | shipped | Scope to default Engineering setup |
| Focused PM/QA/Sales setups can be selected or composed | `optional-domain-packs` | B | optional | State setup and maturity boundaries |
| Cross-repository work uses explicit project boundaries | `cross-repo-peering` | A | optional | One graph per project; setup required |
| Sprout remains a separate MIT dependency outside Hero's Apache-2.0 grant | `sprout-license-boundary` | A | shipped | Keep the repository and license boundary explicit |
| Memory and delivery improve one another across sessions and supported tools | `product-two-system-loop` | A | shipped | Hero workspace plus supported harness or CLI |

## What Hero is

- A durable, repository-centered memory for AI-assisted engineering.
- A connected system for designing, implementing, auditing, and verifying work.
- A harness-native way to carry project context and delivery discipline across supported AI coding tools.
- A local-first CLI and project corpus, with optional integrations that state their prerequisites and consent boundaries.

## What Hero is not

- Not merely a spec template, spec kit, or ceremony layer.
- Not a replacement for the coding agent or harness that writes code.
- Not a general-purpose wiki, issue tracker, or hosted vector-memory service.
- Not one shared graph spanning every repository; peering keeps one graph per project.
- Not a claim that Hero Code or Hero Cloud is open source.
- Not a shipped adapter layer for GitHub Spec Kit, OpenSpec, or other external spec systems; that interoperability is planned direction only (`pluggable-spec-systems`, Class D, planned/prohibited for shipped copy).

## Preferred vocabulary

| Prefer | Use it to mean | Avoid |
|---|---|---|
| project memory | Durable intent, decisions, corrections, conventions, evidence, and state | “chat memory,” “infinite memory,” “perfect memory” |
| memory and delivery system | The two-system product category | “spec layer,” “spec kit with agents” |
| verified delivery | Work closed through ledger, cold audit, and build/test evidence | “autonomous completion” without prerequisites |
| harness-native workflows | Workflows rendered into each supported tool's actual surfaces | “slash commands everywhere” |
| specialized agents | Roles selected for bounded work | Mutable roster counts as differentiation |
| project-owned context | Repository/workspace artifacts that survive model sessions | “model remembers everything” |
| one graph per project | Peering's explicit repository boundary | “one cross-repo graph” |
| optional / preview / planned | Honest capability maturity | “production-ready” without release evidence |
| Engineering setup / focused PM or QA setup | Installed domain composition | “all-in-one company operating system” |

## Prohibited claims

- This `hero` repository is Apache-2.0 licensed. Do not call its source publicly available until the separate visibility gate succeeds (`apache-license-status`, Class A, shipped).
- Do not imply Hero Code or Hero Cloud is included in this repository or its grant. Both remain proprietary (`proprietary-repository-boundary`, Class B, shipped boundary).
- Treat Sprout as a separate MIT project, never as part of Hero's Apache grant (`sprout-license-boundary`, Class A, shipped).
- Do not claim cloud, team, outpost, or remote-server readiness without a public access path and release evidence.
- Do not claim external spec providers are pluggable today.
- Do not present fictional output, unsupported status names, mutable counts, or stale version numbers as product proof.
- Do not promise that Hero eliminates supervision, remembers everything, works with every tool, or guarantees correctness.

## Objections and responses

### “My coding tool already has rules or memory.”

Tool-local rules are useful and Hero should complement them. Hero's role is to keep project knowledge, decisions, evidence, and delivery state in a durable project corpus that can be rendered or retrieved across supported harnesses. It adds a completion workflow rather than replacing a tool's native instructions.

### “This sounds like a spec kit.”

Specs are one execution mechanism. Hero leads with durable project memory: the project truth that later sessions need. Its delivery system uses that memory to implement, audit, and verify work, then produces new evidence and decisions for later sessions. A spec-only reading misses half the product and the reinforcing loop.

### “This sounds like too much ceremony for a small change.”

Hero should scale process to the work. Notes, direct queries, and lightweight context remain available; specs and full closing gates are for work where intent, risk, and proof justify them. The positioning must never celebrate ceremony as the product.

### “Why not use a wiki and issue tracker?”

Keep them. Wikis communicate durable human knowledge and trackers coordinate work. Hero connects project context to agent execution and verification. Optional integrations can read evidence or perform bounded operations after setup; Hero does not require replacing the systems a team already trusts.

### “Will this lock us into Hero's spec format?”

Today, Hero's shipped delivery workflows use Hero's own spec system. External provider adapters are planned, not shipped. The public comparison should be candid about that limitation and revisit it only after a provider contract and at least two proven adapters exist.

## Fair comparison guidance

Comparisons explain fit; they do not manufacture a winner.

- **AI coding harnesses** write and reason about code. Hero supplies durable project context and delivery discipline through their native surfaces. Use both.
- **Rule files and instruction files** are excellent for stable guidance. Hero adds evolving decisions, evidence, current work state, retrieval, and verified closing workflows. Keep concise rules where the harness expects them.
- **Wikis and documentation systems** explain the project to humans. Hero can preserve and retrieve agent-relevant project knowledge but should link to, not displace, authoritative human documentation.
- **Issue trackers** coordinate commitments and external status. Hero can import evidence and perform bounded operations when configured; the tracker remains the coordination system of record.
- **Spec frameworks such as GitHub Spec Kit or OpenSpec** focus on structuring agent work. Hero's intended distinction is the connected project-memory system and evidence-gated delivery loop. Do not publish competitor-specific claims until `hero-transparent-comparisons` completes current, first-party research. Praise what each tool does well, state where Hero is heavier or lighter, and include a clear “choose them when” section.

## Tagline candidates

**Recommended lead:** Your project remembers. Your agents deliver.

Other candidates:

- Project memory for AI-assisted engineering.
- Give every coding agent the project, not just a prompt.
- Durable context. Verified delivery.

## Reusable boilerplate

The memory, delivery, and reinforcing-loop statements in these blocks are grounded in shipped Class A claims `product-memory-system`, `product-delivery-system`, and `product-two-system-loop`.

### 25 words

Hero gives AI coding tools durable project memory and a verified delivery system, so decisions survive sessions and agents finish work against evidence, not guesswork.

### 50 words

Hero gives AI coding tools durable project memory: intent, decisions, corrections, conventions, evidence, and state that survive across sessions. Its connected delivery system uses specs, specialized agents, cold audits, and verification to turn that context into finished engineering work and leave the project better prepared for the next cold session.

### 150 words

Hero is a project memory and delivery system for AI-assisted engineering. It preserves intent, decisions, corrections, conventions, evidence, failures, and current state that coding agents need after a chat ends. That durable context helps engineers resume work without rebuilding the project in every prompt or depending on one tool's history. Hero's connected delivery system turns remembered context into bounded execution: specs define intent and acceptance, specialized agents do focused work, and Completion Ledgers, cold audits, builds, and tests establish whether the result is done. Each verified delivery can leave decisions and evidence for later sessions, creating a reinforcing loop between memory and execution. Hero installs harness-native workflows for supported coding tools and keeps its project corpus local and inspectable. Engineering is the lead setup, with lightweight PM and QA help included; focused PM, QA, and Sales packs are optional expansion paths. Hero Code and Hero Cloud remain separate proprietary products.

## Surface rules

- **Landing:** lead with the one-sentence position, show memory first and delivery second, then one concrete proof for each. Do not lead with agent/spec counts.
- **README:** explain the two-system model before installation. Give the lead audience a short “why,” then a factual quickstart.
- **Hosted docs:** make the memory model, delivery model, and reinforcing loop separately navigable. Keep exact commands and availability qualifiers near each claim.
- **Repository metadata:** Apache-2.0 is accurate for this repository; add public-source language only after the visibility gate.
- **Comparisons:** remain generic until the transparent-comparisons research spec is delivered.

## Downstream compatibility contract

| Child surface | Required inheritance |
|---|---|
| Root documentation | Explain memory first, delivery second, before installation; use factual quickstarts and no mutable roster counts as proof. |
| Hosted documentation | Give memory, delivery, and their loop distinct navigation; keep evidence qualifiers beside claims and remove stale or unlicensed generated assets. |
| Landing page | Lead with “Your project remembers. Your agents deliver.” and show one concrete, revision-tied proof for each system. |
| Drift guard | Enforce prohibited claims, valid commands/configuration, derived inventories, release versions, deployment revision, and production links. |
| Repository readiness | Describe only this repository's current Apache-2.0 grant; keep Hero Code, Hero Cloud, and Sprout outside the boundary. |
| Release and launch gates | Keep open-source, public visibility, release publication, and live-site claims behind their named approvals and evidence. |
