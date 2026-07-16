# AGENTS.md

<!-- hero:managed-start v=v0.25.1-3-g153ab29-dirty -->
## Hero — Spec-Driven AI Engineering

This project uses **Hero** for spec-driven engineering workflows. Hero manages specs, integrates with work trackers (Jira, GitHub, Linear), and provides structured workflows via slash commands.

### Session Title

On the **first interaction** of every session, set a concise, descriptive session title that reflects what the user is working on (e.g. "design: auth flow", "fix: cart total rounding", "deliver: export-csv"). This keeps the session list navigable.

### Natural Language Routing

When the user describes what they want in natural language, route to the appropriate Hero slash command. **Run the command — don't just suggest it.**

| User intent | Command |
|---|---|
| Bug, error, broken, fix, investigate, diagnose | `/diagnose` |
| New feature, build, design, add, plan | `/design` |
| Implement, deliver, ship, code, execute | `/deliver` |
| Autopilot/run a whole initiative, "put X on autopilot", "drive the initiative", keep working autonomously | `/drive <initiative>` |
| Review, PR, pull request, code review | `/review` |
| Break down, decompose, epic, sequence | `/compose` |
| Convention, pattern, standard, style | `/convention` |
| Decision, tradeoff, compare, choose, ADR | `/decide` |
| Explore, brainstorm, roadmap, ideate | `/discover` |
| Mockup, mock, wireframe, prototype, visualize a screen, "what would X look like", "is that a swift mock?" | `/mock` |
| Document, docs, explain, write docs | `/docs` |
| Release, deploy, version, ship | `/release` |
| Retro, postmortem, lessons learned | `/retro` |
| Note, capture, remember, save thought | `/note` |
| Scan, detect, onboard, stack analysis | `/scan` |
| Check, health, validate workspace | `/check` |
| Sprint, iteration, load sprint | `/sprint` |
| Import, pull issues, fetch from tracker, sync issues | `/import` |
| What's stuck, blocked items, dependencies, can't move forward | `/blocked` |
| Capture, extract learnings, persist session knowledge to the knowledge base | `/capture` |
| Challenge or revise a diagnosis, push back on root cause with new context | `/challenge` |
| Start of session, load ranked context, what's in flight | `/resume` |
| Roadmap drift triage, "review the roadmap for staleness" | `/roadmap-review` |
| Scrub the codebase — dead code, weak types, duplication, bad comments, legacy cruft | `/scrub` |
| Break a large spec into smaller, independently deliverable child specs | `/split` |
| Trace where something came from, chain of decisions/specs/commits | `/why` |
| Not sure which command to use, route my request | `/hero` |
| Ask sibling/peer repo a question, check with peer | `hero peer call <alias> --mode=advisory "..."` |
| Have peer design something, let peer handle design | `hero peer call <alias> --mode=spec-out "..."` |
| Hand off a spec to a peer repo, drop on peer's queue, transfer to sibling | `hero handoff <spec> <alias>` |
| Pick up handed-back spec, accept the handoff, peer finished | `hero handoff accept <spec>` |
| What peers do we have, list siblings, which repos are linked | `hero peer list` |
| What does peer expose, peer surface, peer conventions, inspect peer | `hero peer show <alias>` |
| Cross-repo peering front door (session-level; picks advisory/spec-out/handoff/list/show for you) | `/peer` |
| Force-refresh NEXT.md/QUEUE.md before switching tools (session-level; distinct from the cross-repo rows above) | `/handoff` |

When routing, pass the user's original context as arguments to the command. If the intent is ambiguous, present the top 2-3 options and ask.

**Slash commands ≠ CLI subcommands.** Slash commands (e.g. `/discover`, `/convention`) run inside the AI tool's session only — they are **not** `hero discover` or `hero convention` terminal commands. Some commands exist on both surfaces, but many are slash-only. Do not hallucinate CLI subcommands from slash command names. <!-- drift-test:ignore (illustrative: `hero discover`/`hero convention` above are explicitly non-existent subcommands) -->

| Surface | Commands |
|---|---|
| **Slash-only** (no `hero <name>` equivalent) | `/capture`, `/challenge`, `/compose`, `/convention`, `/decide`, `/discover`, `/drive`, `/mock`, `/release`, `/retro`, `/review`, `/roadmap-review`, `/scrub`, `/split` |
| **Both slash and CLI** | `/blocked`, `/check`, `/deliver`, `/design`, `/diagnose`, `/docs`, `/handoff` (slash = NEXT.md refresh; CLI `hero handoff <spec> <alias>` = cross-repo drop to a peer), `/hero` ("which command do I use" meta-help; CLI equivalent `hero do <request>`), `/import` (slash = tracker import via `hero sync import`; root `hero import` is unrelated knowledge-base ingestion), `/note`, `/peer`, `/resume`, `/scan`, `/sprint`, `/why` |
| **CLI-only** (see CLI Commands below) | `hero status`, `hero search`, `hero ask`, `hero list`, `hero queue`, `hero spec verify`, `hero spec score`, `hero diff`, `hero drift`, etc. |

**Mockup routing.** Any request to mock, wireframe, prototype, or visualize a screen — including casual questions like "what would this look like?" or "is that a swift mock?" — routes to `/mock`. **Never hand-generate a mockup outside that command, and never pick the format yourself.** `/mock` runs `hero spec mock detect`, which chooses the renderer (HTML vs. native SwiftUI) deterministically from the repo's stack and announces it before generating. There is **no "HTML-first, then port to SwiftUI" workflow** — that is a confabulation, not a real Hero pattern. In a native app you produce a native SwiftUI mockup directly (compiled, with real screenshots); in a web app you produce HTML. Do **not** generate an HTML approximation "to iterate faster" on a native project. Always end your response with the clickable file inventory `/mock` surfaces — never make the user ask for the links.

**Cross-repo peering disambiguation.** The session-level `/handoff` slash command (force-refresh NEXT.md) and the cross-repo `hero handoff <spec> <alias>` command share a verb but do different things. Disambiguate by whether the user names a peer alias: if they do, it's cross-repo; if not, it's session handoff. When a user says "ask hero-code about X" or "hand off to hero-cloud," route to the cross-repo command and **compose the prompt yourself** — don't paraphrase the user's words verbatim. A good peer-call prompt names the specific question, references the active spec via `--related-spec <slug>` when one exists, and includes `--reason` explaining why the call is happening. Pick the mode: **advisory** (need a fact, peer writes nothing), **spec-out** (peer designs the fix on its side), or **handoff** (you already did the investigation, dropping it on peer's queue).

### Key Workflow

1. **Design first**: Use `/design` to create a spec before building anything
2. **Deliver from spec**: Use `/deliver` to implement from an approved spec
3. **Debug with specs**: Use `/diagnose` to investigate bugs and produce fix specs
4. **Never work on closed items**: Commands like `/diagnose` and `/deliver` check if the tracker issue is still open before starting work
5. **Finish the closing gate before yielding**: `/deliver` is not done until `hero spec verify <slug>` passes — and verify requires the cold delivery audit to run first. The audit and verify run in the **same turn** as the implementation, not as a follow-up the user triggers. Never stop with a spec left in `planning`/`delivering` and the audit unrun, and never say "the audit still needs to run" — run it now instead. This holds in every delivery mode, including the default supervised mode.

### Agents Reference

Grouped by role (every installed agent, no links):

- **Delivery leads:** feature-delivery-lead, platform-delivery-lead — product features vs. platform/migration work.
- **Architects & reviewers:** greenfield-architect, brownfield-architect, architecture-reviewer, design-reviewer, pr-reviewer, security-reviewer, roadmap-reviewer — design-time and review gates.
- **Specialist engineers:** engineer, api-engineer, database-engineer, devops-engineer, integration-engineer, migration-engineer, performance-engineer, release-engineer — build and ship by concern.
- **QA & investigation:** functional-qa-engineer, test-architect, debug-investigator, dependency-analyst, issue-tracker, product-ideator, ui-designer — testing, root-cause work, dependency mapping, issue triage, ideation, UI review.
- **Scrubbers:** comment-scrubber, deadcode-scrubber, dedup-scrubber, defensive-scrubber, dependency-scrubber, legacy-scrubber, type-scrubber — one code-quality concern each.
- **Core (installed with every pack):** convention-author, documentation-engineer, project-context-builder, session-primer.

### Skills Reference

Grouped by concern (every installed skill, no links):

- **Stacks & detection:** database-stack, go-stack, groovy-stack, java-stack, javascript-stack, python-stack, react-stack, rust-stack, stack-detection — conventions per detected stack.
- **Architecture & design:** api-design-and-contracts, architecture-principles, greenfield-scaffolding, implementation-principles, integration-boundaries — design-time reasoning for new and evolving systems.
- **Delivery & spec process:** batch-discipline, delivery-audit, drive, spec-composition, spec-sizing — sizing, composing, delivering, and cold-auditing specs.
- **Investigation & quality:** challenge-diagnosis, debugging-investigation, dependency-analysis, pr-review, root-cause-classification, security-review, test-strategy, testing-and-validation — diagnosing, reviewing, testing.
- **Scrub:** code-scrub — shared methodology behind the scrubber agents.
- **Ops, incident & release:** devops-and-operations, incident-response, release-and-deployment — production operations lifecycle.
- **Mockups:** html-mockup-generation, swiftui-mockup-renderer — the two `/mock` renderer paths.
- **Cross-repo & reporting:** cross-repo-peering, deep-code-enrichment, issue-list-report — peer calls, enrichment passes, report formatting.
- **Roadmap & performance:** performance-optimization, roadmap-review — perf tuning and roadmap-shape triage.
- **Migration:** migration-safety — safe migration/refactor patterns.
- **Core (installed with every pack):** agent-reliability, auto-knowledge-capture, completion-ledger, context-injection, convention-writing, documentation-practices, executive-report, explainer-format, kickoff-prompt, knowledge-flywheel, next-handoff-emit, next-md, note-capture, nudge-awareness, project-context-generation, spec-format.

### CLI Commands

These are run in the terminal, not as slash commands:
- `hero status` — workspace state and active specs
- `hero search <query>` — find specs by keyword
- `hero snapshot` — render the project-shape rollup (surfaces, stages, recent activity, risks)
- `hero sync import` — import issues from tracker as spec scaffolds
- `hero sync pull <slug>` — sync spec status from tracker
- `hero note <slug>` — quick note capture
- `hero check` — health check
- `hero peer list` — list registered sibling repos with reachability + manifest status
- `hero peer show <alias>` — inspect one peer (manifest contents, in-flight handoffs)
- `hero peer call <alias> --mode=advisory "..."` — ask peer's Hero a question (no writes on peer)
- `hero peer call <alias> --mode=spec-out "..."` — have peer's Hero design a spec natively on its side
- `hero handoff <spec> <alias>` — async-drop a local spec on peer's queue
- `hero handoff status` / `hero handoff accept <spec>` — track handoffs across the boundary
- `hero admin repos add <alias> <path>` — register a sibling repo as a peer (one-time setup)

### Project Structure

- `<harness>/commands/` — Slash command definitions (workflows like /design, /deliver, /diagnose)
- `<harness>/agents/` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)
- `<harness>/skills/` — Domain-specific knowledge and patterns (each skill is a subdir with SKILL.md)
- `.hero/planning/` — Active specs being worked on
- `.hero/specs/` — Completed specs (archive)
- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)
- `.hero/hero.json` — Project configuration

`hero install` **writes** these into your harness's own directory in that harness's native format — e.g. `.claude/commands/`, `.claude/agents/`, and `.claude/skills/` for Claude; `.codex/agents/*.toml` (TOML) plus workflow skills under `.agents/skills/` for Codex (Codex has no commands directory — its slash commands are a built-in enum, so Hero commands install there as skills). They are generated copies, **not** symlinks or views: re-running `hero install` regenerates them, so hand-edits to the installed files are overwritten on the next install.

### Declaring Spec Relationships

Relationships (parent/child, depends-on, blocks) become knowledge-graph edges **only** through frontmatter. Body `[[wikilinks]]` are searchable text and form **no** edges. Two syntaxes work:

Top-level shorthand (simplest):

```yaml
parent: i1-config-plane          # also accepted: initiative: i1-config-plane
depends-on: [f2-store, f3-watcher]   # also accepted: depends_on:
child:
  - sub-a
  - sub-b
```

`relations:` block (for mixed kinds):

```yaml
relations:
  - target: i1-config-plane
    kind: parent
  - target: other-spec
    kind: related
```

Pitfalls: inline flow style (`- { kind: parent, target: x }`) does **not** parse — use the block form with `target:`/`kind:` on separate lines. Recognized kinds: `parent`, `child`, `depends-on`, `blocks`, `supersedes`, `related`. `hero check` warns when a spec uses edge-intent `[[wikilinks]]`.

### Internal Lookups — Tool Routing

When **you** need to look something up mid-task (as opposed to running a slash command for the user), pick the tool that matches the *shape* of the question, not the one that feels exhaustive:

| Shape of question | Tool |
|---|---|
| "Does spec/knowledge entry X exist? Has this been discussed?" | `hero_search` with `compact: true` — single-line count, no excerpt noise |
| "What's the status / frontmatter of spec X?" | `hero_read_spec` |
| "What's in flight / ready / blocked / mine?" | `hero_list`, `hero_queue`, `hero_blocked` |
| "Where did this come from? What chain of decisions led here?" | `hero_why` — graph traversal beats grep on relations |
| Literal string `foo_bar_baz` across code | `rg` / `grep` |
| Known file at a known path | `Read` |
| Recent commits / git history | `git log` |
| Broad exploration across many files | a context-protective read-only search subagent, where your harness provides one (e.g. Claude Code's `Explore` agent); otherwise `rg` + targeted reads |

**Rule of thumb:** graph- or spec-shaped questions → Hero MCP tools (`hero_*` — on Claude Code these surface as `mcp__hero__<name>`). String-shaped → grep. File-shaped → Read. Don't reach for `grep` on `.hero/` to answer "does spec X exist?" — substring search only finds *literal matches*, not *semantically related* specs (e.g. a spec slugged `domain-routing-and-agents` is the same concept as "domain swap" but won't match either word as a phrase).

Some harnesses defer MCP tool schemas behind a one-time lookup before the tool is callable — e.g. Claude Code's `ToolSearch`. The load is one round-trip and worth it; it's not a reason to fall back to a weaker tool.

### Important Rules

- **Don't assume.** Surface tradeoffs and ask questions if anything is unclear. Present multiple interpretations instead of picking one silently.
- **Honest over agreeable.** Push back when you disagree — say what's wrong, propose the better path, then proceed. Don't reverse your position because the user pushed; reverse it when new evidence warrants it.
- **Label what you know vs. think.** State facts as facts and opinions as opinions. "I'm not sure" beats a confident guess.
- **Say the hard thing.** If the user's approach has a flaw, point it out before implementing. If a request conflicts with these rules, name the conflict rather than silently following.
- **Simplicity first.** Write the minimum code that solves the problem. No speculative features, no unnecessary abstractions, and no error handling for impossible scenarios.
- **Surgical changes.** Touch only what is strictly required. Do not "improve" nearby code or refactor unrelated sections. Match the existing style perfectly.
- **Verify before reporting done.** Define clear success criteria for every task. Run tests or validation scripts and iterate until the criteria are met before reporting completion.
- **Local specs first.** When asked to work on bugs, features, or any tracked items, ALWAYS check what's already imported locally before querying the tracker. Use `hero search --list --type <type>` to find local specs. Only go to the tracker if the local search comes up empty. When working on multiple items (e.g. "diagnose 10 bugs"), select from locally imported specs — never bulk-query the tracker to pick work items.
- Always check spec status before doing work — don't investigate closed bugs or deliver completed specs
- When a tracker is configured, sync status with `hero sync pull` before starting work
- **Hero handoff travels with commits.** Projected handoff files (`.hero/NEXT.md`, `.hero/next/*.md`, `.hero/SNAPSHOT.md`, `.hero/QUEUE.md`) must travel with the commit or the next session (possibly on another machine) starts cold. Every Hero hook install path now wires a pre-commit hook that stages these automatically — you don't normally need to think about it. `hero check` flags a repo where the staging block is missing. As a backstop only, if `hero check` warns that staging isn't wired and you can't install hooks, stage the projected handoff files by hand alongside your code changes.
- Capture novel learnings to `.hero/knowledge/` at the end of major workflows
- Specs use YAML frontmatter with fields: title, type, status, tracker_id, priority, severity
- Imported specs include tracker-prefixed fields (e.g. jira_status, jira_priority, jira_assignee) under a # Jira/GitHub/Linear comment header

### Running Hero Workflows in Codex

Hero's workflow commands are **not slash commands in Codex** — they are skill files you read and follow step-by-step.

**When the user asks you to deliver, diagnose, design, or run any Hero workflow:**

1. Read the workflow skill file at `.agents/skills/command-<name>/SKILL.md`
   (e.g. `.agents/skills/command-deliver/SKILL.md` when the user says "deliver")
2. Follow each step in the file as your workflow. These are **instructions to execute**, not documentation.
3. **Do NOT** skip steps, flip spec frontmatter as a shortcut, or treat the workflow as informational.

**Workflow routing table for Codex:**

| User intent | Skill file to read and follow |
|---|---|
| Deliver, implement, ship, execute | `.agents/skills/command-deliver/SKILL.md` |
| Diagnose, investigate, debug, fix | `.agents/skills/command-diagnose/SKILL.md` |
| Design, plan, spec, add feature | `.agents/skills/command-design/SKILL.md` |
| Review, PR, pull request | `.agents/skills/command-review/SKILL.md` |
| Check, health, validate workspace | `.agents/skills/command-check/SKILL.md` |
| Note, capture, remember | `.agents/skills/command-note/SKILL.md` |
| Compose, break down, epic | `.agents/skills/command-compose/SKILL.md` |
| Discover, brainstorm, explore | `.agents/skills/command-discover/SKILL.md` |

If the skill file doesn't exist, fall back to reading `.claude/commands/<name>.md` directly.

**A Hero workflow is not finished until its closing gate runs.** For `/deliver`, that gate is `hero spec verify <slug>` passing — and verify requires the cold delivery audit to have run first. Do NOT yield back to the user with a spec still in `planning` or `delivering` and the audit unrun. The audit and verify run in the **same turn** as the implementation — they are not a follow-up step the user triggers later. If you find yourself about to say "the audit still needs to run" or "I did not mark the spec complete because the gate still needs to run" — **run it now instead.** Stopping one step short of the closing gate is an unfinished delivery, not a handoff. This holds in every delivery mode, including the default supervised mode: "pause at handoffs" does not include the closing gates.

## Hero Binary & MCP Surface

**Prefer Hero's MCP tools over shelling out to a bare `hero` in a terminal.** A GUI-launched harness can resolve a *different or stale* `hero` binary on its PATH than your login shell does; the MCP surface is the in-process Hero you're already connected to, so it can't drift out from under you. When you must use the CLI and hit a schema/version mismatch or a confusing `hero` version error, **run `hero doctor` and act on its output** — it reports which binary is actually on PATH, its schema, the graph's schema, and the real remediation. Do NOT invent a schema-migration narrative, and do NOT run `hero upgrade` to "fix schema": `hero upgrade` updates workspace files, not the binary, so it cannot fix a wrong-binary-on-PATH situation.

Tracker connections use stable IDs under `integrations.connections`. Shared non-secret settings belong in `.hero/hero.json`; personal `auth.token` belongs at the same path in `.hero/hero.local.json`. Use `hero connect --list` to inspect readiness and `hero sync import` to import tracker issues. Never put credentials in argv or committed config; automation uses `--token-stdin`.

## Project snapshot

Project shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).
<!-- hero:managed-end -->
