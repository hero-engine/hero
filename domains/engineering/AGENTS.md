# Hero — Spec-Driven AI Engineering

This project uses **Hero** for spec-driven engineering workflows. Hero manages specs, integrates with work trackers (Jira, GitHub, Linear), and provides structured workflows via slash commands.

### Session Title

On the **first interaction** of every session, set a concise, descriptive session title that reflects what the user is working on (e.g. "design: auth flow", "fix: cart total rounding", "deliver: export-csv"). This keeps the session list navigable.

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
- **Attention:** attention-lifecycle-awareness, deferred-work-suggestions — read bounded state at chat boundaries and propose meaningful out-of-scope work without bypassing user consent or current delivery obligations.
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
- `hero peer call <alias> --mode=advisory "..."` — send an asynchronous Project Mail question (no model launch or receiver-tree write)
- `hero peer call <alias> --mode=spec-out "..."` — request peer-side design over Mail; receiver promotion is explicit
- `hero handoff <spec> <alias>` — send a work-transfer Mail request without changing either spec tree
- `hero handoff receive <message-id>` — receiver explicitly promotes Mail through Intake and replies with its artifact
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

`hero install` **writes** these into your harness's own directory in that harness's native format — e.g. `.claude/commands/`, `.claude/agents/`, and `.claude/skills/` for Claude; `.codex/agents/*.toml` (TOML) plus workflow skills under `.agents/skills/` for Codex; and `.grok/agents/*.md` plus canonical and `command-*` skills under `.grok/skills/` for Grok Build. Codex and Grok have no Hero-owned commands directory, so Hero commands install there as skills. They are generated copies, **not** symlinks or views: re-running `hero install` regenerates them, so hand-edits to the installed files are overwritten on the next install.

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
