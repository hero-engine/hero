package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/managed"
	"github.com/hero-engine/hero/internal/snapshot"
)

// agents_md.go — AGENTS.md as the single canonical root instruction file
// (single-source-install P1).
//
// AGENTS.md is the user-visible file users edit. Hero owns a single managed
// region inside it (versioned `<!-- hero:managed-start v=X -->` ...
// `<!-- hero:managed-end -->`); the user owns everything outside. Re-running
// install regenerates the region in place — user content above and below
// is preserved byte-for-byte.
//
// Migration is automatic: a pre-P1 legacy `<!-- hero:managed -->` /
// `<!-- hero:managed -->` pair (or single marker) is detected and replaced
// with the new versioned form on the next install.

// resolveAgentsMdPath returns the path for AGENTS.md based on install mode
// and target. Project mode: AGENTS.md at project root. Global mode varies by
// target.
func resolveAgentsMdPath(opts Options) string {
	switch opts.Mode {
	case ModeProject:
		if opts.TargetDir == "" {
			return ""
		}
		return filepath.Join(opts.TargetDir, "AGENTS.md")
	case ModeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		if opts.Target == TargetCodex {
			return filepath.Join(home, ".codex", "AGENTS.md")
		}
		return filepath.Join(home, ".config", "opencode", "AGENTS.md")
	default:
		return ""
	}
}

// installAgentsMd writes Hero's managed block into AGENTS.md. See
// installManagedMarkdown for the shared three-case logic.
func installAgentsMd(opts Options, result *Result, agentsMdPath string) error {
	return installManagedMarkdown(opts, result, installManagedSpec{
		Path:        agentsMdPath,
		Label:       "AGENTS.md",
		DefaultH1:   "# AGENTS.md",
		Sections:    defaultSections(opts, agentsMdPath),
		AllowSkip:   false,
		SkipEnabled: false,
	})
}

// defaultSections returns the canonical section contributor order for
// the consolidated managed region: install body first, snapshot pointer
// last. All callers (AGENTS.md, CLAUDE.md) use this same ordering so
// the managed block is identical across files.
func defaultSections(opts Options, filePath string) []managed.SectionContributor {
	return []managed.SectionContributor{
		newAgentsMdBodySection(opts),
		snapshot.NewPointerSection(filePath, snapshotPointerRelativePath(opts, filePath)),
	}
}

// snapshotPointerRelativePath computes the SNAPSHOT.md path relative to
// the file being rendered. AGENTS.md and CLAUDE.md live at project root,
// so the canonical relative path is .hero/SNAPSHOT.md. NEXT.md lives in
// .hero/, so the relative path is just SNAPSHOT.md — but NEXT.md is
// written from the snapshot projector, not from here, so this helper
// only needs the root-file case.
func snapshotPointerRelativePath(opts Options, filePath string) string {
	return ".hero/SNAPSHOT.md"
}

// contentPathsForBody holds project-relative content paths to embed in
// the managed-region body. Defaults match the canonical .hero/ layout
// when no override is configured; hero-on-hero uses content.<kind>_path
// in hero.json to point at top-level agents/commands/skills/.
type contentPathsForBody struct {
	Agents   string
	Commands string
	Skills   string
}

// resolveContentPathsForBody returns project-relative paths describing
// where agents/commands/skills live for the installed harness. Under
// render-direct-install, each target writes to its own harness dir, so
// the AGENTS.md body just lists generic per-harness destinations.
func resolveContentPathsForBody(opts Options) contentPathsForBody {
	return contentPathsForBody{
		Agents:   "<harness>/agents/",
		Commands: "<harness>/commands/",
		Skills:   "<harness>/skills/",
	}
}

// installManagedSpec describes how installManagedMarkdown should handle a
// single file.
type installManagedSpec struct {
	// Path is the absolute path to the file (e.g. AGENTS.md, CLAUDE.md).
	Path string
	// Label is the short name used in log output ("AGENTS.md", "CLAUDE.md").
	Label string
	// DefaultH1 is the top line written when creating a fresh file (e.g.
	// "# AGENTS.md"). Empty to omit.
	DefaultH1 string
	// Sections are the contributors composing the managed region body,
	// in canonical order.
	Sections []managed.SectionContributor
	// AllowSkip indicates a per-file skip flag exists (e.g. NoTouchClaudeMd).
	AllowSkip bool
	// SkipEnabled is the value of that skip flag for this run.
	SkipEnabled bool
}

// installManagedMarkdown applies the unified managed-region logic to any
// Markdown file (AGENTS.md, CLAUDE.md, future harness instruction files).
// Three behaviors:
//
//  1. File doesn't exist → create with DefaultH1 (if any) + managed region.
//  2. File exists, no managed region → insert managed region at the top
//     (after the H1 if any), preserve all existing content below.
//  3. File exists with a managed region (versioned or legacy) → replace the
//     region in place with the current version, preserve all content
//     outside the region byte-for-byte.
//
// All paths idempotent — re-running with no content change produces zero
// filesystem writes.
//
// The managed region is always regenerated in place. The markers themselves
// signal that the content between them is owned by Hero; users who edit
// inside the markers have ignored that signal and will lose their edits on
// the next install.
func installManagedMarkdown(opts Options, result *Result, spec installManagedSpec) error {
	if spec.AllowSkip && spec.SkipEnabled {
		return nil
	}

	writer := managed.Writer{
		File:      spec.Path,
		Sections:  spec.Sections,
		DefaultH1: spec.DefaultH1,
	}
	ctx := managed.Context{
		File:        spec.Path,
		HeroVersion: opts.heroVersion(),
		ProjectDir:  opts.projectRoot(),
	}

	existing, newContent, exists, err := writer.PlanContent(ctx)
	if err != nil {
		return err
	}
	wasNew := !exists

	if opts.DryRun {
		if wasNew {
			progressf(opts, "  %s -> %s (create)\n", spec.Label, spec.Path)
			result.Copied = append(result.Copied, CopyAction{Source: "generated", Dest: spec.Path})
		} else if newContent != existing {
			progressf(opts, "  %s -> %s (update managed region)\n", spec.Label, spec.Path)
			result.Merged = append(result.Merged, spec.Path)
		}
		return nil
	}

	if !wasNew && newContent == existing {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(spec.Path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(spec.Path, []byte(newContent), 0o644); err != nil {
		return err
	}

	if wasNew {
		progressf(opts, "  %s -> %s (create)\n", spec.Label, spec.Path)
		result.Copied = append(result.Copied, CopyAction{Source: "generated", Dest: spec.Path})
	} else {
		progressf(opts, "  %s -> %s (update managed region)\n", spec.Label, spec.Path)
		result.Merged = append(result.Merged, spec.Path)
	}

	return nil
}

// projectRoot returns opts.ProjectRoot or opts.TargetDir, in that order.
// Used to populate managed.Context.ProjectDir for section contributors.
func (o Options) projectRoot() string {
	if o.ProjectRoot != "" {
		return o.ProjectRoot
	}
	return o.TargetDir
}

// heroVersion returns opts.Version or "dev" if empty — used to stamp the
// managed region.
func (o Options) heroVersion() string {
	if o.Version == "" {
		return "dev"
	}
	return o.Version
}

// generateAgentsMdBody produces the Hero-managed content that goes inside
// the AGENTS.md managed region. No markers — InsertManagedRegion wraps it.
//
// The body does NOT include a top-level H1: that's the user's project
// title at the top of the file. Hero's content starts with an H2 section
// header so it composes cleanly under whatever the user's H1 is. (Or, for
// fresh files, under the default `# AGENTS.md` title in
// computeAgentsMdContent.)
//
// paths describe the project-relative canonical content locations the
// "Project Structure" section should reference. They MUST match the
// actual on-disk layout — pointing the model at directories that don't
// exist sends it hunting through the workspace from first principles
// and is the most common reason a non-Claude-Code harness wanders
// after `hero scan`.
// RenderAgentsMdBodyForDriftTest exposes the rendered managed-region
// body (orchestrator output, contributors stitched together) for the
// markdown invocation drift test in internal/cli. The shape is what
// install would write inside the managed markers for AGENTS.md /
// CLAUDE.md given default content paths and the renderable section
// contributors. Kept narrow on purpose: the function takes no
// arguments and produces deterministic output suitable for grep-style
// invocation extraction.
func RenderAgentsMdBodyForDriftTest() []byte {
	writer := managed.Writer{
		File:     "AGENTS.md",
		Sections: []managed.SectionContributor{newAgentsMdBodySection(Options{})},
	}
	body, err := writer.RenderBody(managed.Context{File: "AGENTS.md"})
	if err != nil {
		return nil
	}
	return []byte(body)
}

// agentsMdBodySection adapts the install-side AGENTS.md body content
// (the Hero CLI/skill guide) to managed.SectionContributor. The
// section's H2 ("Hero — Spec-Driven AI Engineering") is rendered by
// the orchestrator from SectionTitle; the body returned by Render
// starts at H3 ("Session Title").
type agentsMdBodySection struct {
	opts Options
}

func newAgentsMdBodySection(opts Options) agentsMdBodySection {
	return agentsMdBodySection{opts: opts}
}

func (s agentsMdBodySection) SectionID() string    { return "install:agents-md-body" }
func (s agentsMdBodySection) SectionTitle() string { return "Hero — Spec-Driven AI Engineering" }

func (s agentsMdBodySection) Render(_ managed.Context) (string, error) {
	body := generateAgentsMdBody(resolveContentPathsForBody(s.opts))
	body += renderActiveDialectBlock(s.opts)
	return body, nil
}

func generateAgentsMdBody(paths contentPathsForBody) string {
	var sb strings.Builder

	// The H2 heading is owned by the orchestrator (managed.Writer
	// emits it from SectionTitle). Body starts at the H3 subsection.
	sb.WriteString("This project uses **Hero** for spec-driven engineering workflows. ")
	sb.WriteString("Hero manages specs, integrates with work trackers (Jira, GitHub, Linear), ")
	sb.WriteString("and provides structured workflows via slash commands.\n\n")

	sb.WriteString("### Session Title\n\n")
	sb.WriteString("On the **first interaction** of every session, set a concise, descriptive session title that reflects what the user is working on ")
	sb.WriteString("(e.g. \"design: auth flow\", \"fix: cart total rounding\", \"deliver: export-csv\"). ")
	sb.WriteString("This keeps the session list navigable.\n\n")

	sb.WriteString("### Natural Language Routing\n\n")
	sb.WriteString("When the user describes what they want in natural language, route to the appropriate Hero slash command. ")
	sb.WriteString("**Run the command — don't just suggest it.**\n\n")
	sb.WriteString("| User intent | Command |\n|---|---|\n")
	sb.WriteString("| Bug, error, broken, fix, investigate, diagnose | `/diagnose` |\n")
	sb.WriteString("| New feature, build, design, add, plan | `/design` |\n")
	sb.WriteString("| Implement, deliver, ship, code, execute | `/deliver` |\n")
	sb.WriteString("| Review, PR, pull request, code review | `/review` |\n")
	sb.WriteString("| Break down, decompose, epic, sequence | `/compose` |\n")
	sb.WriteString("| Convention, pattern, standard, style | `/convention` |\n")
	sb.WriteString("| Decision, tradeoff, compare, choose, ADR | `/decide` |\n")
	sb.WriteString("| Explore, brainstorm, roadmap, ideate | `/discover` |\n")
	sb.WriteString("| Document, docs, explain, write docs | `/docs` |\n")
	sb.WriteString("| Release, deploy, version, ship | `/release` |\n")
	sb.WriteString("| Retro, postmortem, lessons learned | `/retro` |\n")
	sb.WriteString("| Note, capture, remember, save thought | `/note` |\n")
	sb.WriteString("| Scan, detect, onboard, stack analysis | `/scan` |\n")
	sb.WriteString("| Check, health, validate workspace | `/check` |\n")
	sb.WriteString("| Sprint, iteration, load sprint | `/sprint` |\n")
	sb.WriteString("| Import, pull issues, fetch from tracker, sync issues | `/import` |\n")
	sb.WriteString("| Ask sibling/peer repo a question, check with peer | `hero peer call <alias> --mode=advisory \"...\"` |\n")
	sb.WriteString("| Have peer design something, let peer handle design | `hero peer call <alias> --mode=spec-out \"...\"` |\n")
	sb.WriteString("| Hand off a spec to a peer repo, drop on peer's queue, transfer to sibling | `hero handoff <spec> <alias>` |\n")
	sb.WriteString("| Pick up handed-back spec, accept the handoff, peer finished | `hero handoff accept <spec>` |\n")
	sb.WriteString("| What peers do we have, list siblings, which repos are linked | `hero peer list` |\n")
	sb.WriteString("| What does peer expose, peer surface, peer conventions, inspect peer | `hero peer show <alias>` |\n\n")

	sb.WriteString("When routing, pass the user's original context as arguments to the command. ")
	sb.WriteString("If the intent is ambiguous, present the top 2-3 options and ask.\n\n")

	sb.WriteString("**Cross-repo peering disambiguation.** The session-level `/handoff` slash command (force-refresh NEXT.md) and the cross-repo `hero handoff <spec> <alias>` command share a verb but do different things. Disambiguate by whether the user names a peer alias: if they do, it's cross-repo; if not, it's session handoff. When a user says \"ask hero-code about X\" or \"hand off to hero-cloud,\" route to the cross-repo command and **compose the prompt yourself** — don't paraphrase the user's words verbatim. A good peer-call prompt names the specific question, references the active spec via `--related-spec <slug>` when one exists, and includes `--reason` explaining why the call is happening. Pick the mode: **advisory** (need a fact, peer writes nothing), **spec-out** (peer designs the fix on its side), or **handoff** (you already did the investigation, dropping it on peer's queue).\n\n")

	sb.WriteString("### Key Workflow\n\n")
	sb.WriteString("1. **Design first**: Use `/design` to create a spec before building anything\n")
	sb.WriteString("2. **Deliver from spec**: Use `/deliver` to implement from an approved spec\n")
	sb.WriteString("3. **Debug with specs**: Use `/diagnose` to investigate bugs and produce fix specs\n")
	sb.WriteString("4. **Never work on closed items**: Commands like `/diagnose` and `/deliver` check if the tracker issue is still open before starting work\n\n")

	sb.WriteString("### CLI Commands\n\n")
	sb.WriteString("These are run in the terminal, not as slash commands:\n")
	sb.WriteString("- `hero status` — workspace state and active specs\n")
	sb.WriteString("- `hero search <query>` — find specs by keyword\n")
	sb.WriteString("- `hero snapshot` — render the project-shape rollup (surfaces, stages, recent activity, risks)\n")
	sb.WriteString("- `hero sync import` — import issues from tracker as spec scaffolds\n")
	sb.WriteString("- `hero sync pull <slug>` — sync spec status from tracker\n")
	sb.WriteString("- `hero note <slug>` — quick note capture\n")
	sb.WriteString("- `hero check` — health check\n")
	sb.WriteString("- `hero peer list` — list registered sibling repos with reachability + manifest status\n")
	sb.WriteString("- `hero peer show <alias>` — inspect one peer (manifest contents, in-flight handoffs)\n")
	sb.WriteString("- `hero peer call <alias> --mode=advisory \"...\"` — ask peer's Hero a question (no writes on peer)\n")
	sb.WriteString("- `hero peer call <alias> --mode=spec-out \"...\"` — have peer's Hero design a spec natively on its side\n")
	sb.WriteString("- `hero handoff <spec> <alias>` — async-drop a local spec on peer's queue\n")
	sb.WriteString("- `hero handoff status` / `hero handoff accept <spec>` — track handoffs across the boundary\n")
	sb.WriteString("- `hero admin repos add <alias> <path>` — register a sibling repo as a peer (one-time setup)\n\n")

	sb.WriteString("### Project Structure\n\n")
	sb.WriteString(fmt.Sprintf("- `%s` — Slash command definitions (workflows like /design, /deliver, /diagnose)\n", paths.Commands))
	sb.WriteString(fmt.Sprintf("- `%s` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)\n", paths.Agents))
	sb.WriteString(fmt.Sprintf("- `%s` — Domain-specific knowledge and patterns (each skill is a subdir with SKILL.md)\n", paths.Skills))
	sb.WriteString("- `.hero/planning/` — Active specs being worked on\n")
	sb.WriteString("- `.hero/specs/` — Completed specs (archive)\n")
	sb.WriteString("- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)\n")
	sb.WriteString("- `.hero/hero.json` — Project configuration\n\n")
	sb.WriteString("Your harness may expose the agent/command/skill directories under its own prefix (`.claude/`, `.opencode/`, `.cursor/`, etc.) as symlinks back to the canonical paths above. Edit only the canonical files — harness directories are views.\n\n")

	sb.WriteString("### Important Rules\n\n")
	sb.WriteString("- **Don't assume.** Surface tradeoffs and ask questions if anything is unclear. Present multiple interpretations instead of picking one silently.\n")
	sb.WriteString("- **Simplicity first.** Write the minimum code that solves the problem. No speculative features, no unnecessary abstractions, and no error handling for impossible scenarios.\n")
	sb.WriteString("- **Surgical changes.** Touch only what is strictly required. Do not \"improve\" nearby code or refactor unrelated sections. Match the existing style perfectly.\n")
	sb.WriteString("- **Verify before reporting done.** Define clear success criteria for every task. Run tests or validation scripts and iterate until the criteria are met before reporting completion.\n")
	sb.WriteString("- **Local specs first.** When asked to work on bugs, features, or any tracked items, ALWAYS check what's already imported locally before querying the tracker. Use `hero search --list --type <type>` to find local specs. Only go to the tracker if the local search comes up empty. When working on multiple items (e.g. \"diagnose 10 bugs\"), select from locally imported specs — never bulk-query the tracker to pick work items.\n")
	sb.WriteString("- Always check spec status before doing work — don't investigate closed bugs or deliver completed specs\n")
	sb.WriteString("- When a tracker is configured, sync status with `hero sync pull` before starting work\n")
	sb.WriteString("- Capture novel learnings to `.hero/knowledge/` at the end of major workflows\n")
	sb.WriteString("- Specs use YAML frontmatter with fields: title, type, status, tracker_id, priority, severity\n")
	sb.WriteString("- Imported specs include tracker-prefixed fields (e.g. jira_status, jira_priority, jira_assignee) under a # Jira/GitHub/Linear comment header")

	return sb.String()
}
