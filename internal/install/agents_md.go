package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		Body:        generateAgentsMdBody(),
		AllowSkip:   false,
		SkipEnabled: false,
	})
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
	// Body is the content that goes inside the managed-region markers.
	Body string
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
// Hand-edits inside an existing versioned managed region refuse to
// regenerate unless opts.ForceManagedRegion is set, so users don't lose
// in-region edits silently.
func installManagedMarkdown(opts Options, result *Result, spec installManagedSpec) error {
	if spec.AllowSkip && spec.SkipEnabled {
		return nil
	}

	region := RenderManagedRegion(opts.heroVersion(), spec.Body)

	existing := ""
	wasNew := true
	if data, err := os.ReadFile(spec.Path); err == nil {
		existing = string(data)
		wasNew = false
	}

	// Detect hand-edits inside an existing versioned managed region.
	if !opts.ForceManagedRegion {
		if mr := FindManagedRegion(existing); mr.Present && !mr.Legacy && managedRegionDrifted(mr, spec.Body) {
			return fmt.Errorf(
				"%s managed region has been edited by hand at %s — move your edits outside the markers and re-run, or use --force-managed to overwrite",
				spec.Label, spec.Path,
			)
		}
	}

	newContent := computeManagedContent(existing, region, spec.DefaultH1)

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

	// True idempotency: don't rewrite the file when content is unchanged.
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

// computeManagedContent produces the new file content given existing
// content, the rendered managed region, and an optional default H1 to use
// when creating a fresh file.
func computeManagedContent(existing, region, defaultH1 string) string {
	if existing == "" {
		var sb strings.Builder
		if defaultH1 != "" {
			sb.WriteString(defaultH1)
			sb.WriteString("\n\n")
		}
		sb.WriteString(region)
		return sb.String()
	}
	return InsertManagedRegion(existing, region)
}

// managedRegionDrifted reports whether the managed-region body in `mr`
// looks like Hero's current generated body, allowing trailing whitespace
// differences. If true, the user has hand-edited inside the markers.
func managedRegionDrifted(mr ManagedRegion, currentBody string) bool {
	if !mr.Present {
		return false
	}
	return strings.TrimSpace(mr.Body) != strings.TrimSpace(currentBody)
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
func generateAgentsMdBody() string {
	var sb strings.Builder

	sb.WriteString("## Hero — Spec-Driven AI Engineering\n\n")
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
	sb.WriteString("| Import, pull issues, fetch from tracker, sync issues | `/import` |\n\n")

	sb.WriteString("When routing, pass the user's original context as arguments to the command. ")
	sb.WriteString("If the intent is ambiguous, present the top 2-3 options and ask.\n\n")

	sb.WriteString("### Key Workflow\n\n")
	sb.WriteString("1. **Design first**: Use `/design` to create a spec before building anything\n")
	sb.WriteString("2. **Deliver from spec**: Use `/deliver` to implement from an approved spec\n")
	sb.WriteString("3. **Debug with specs**: Use `/diagnose` to investigate bugs and produce fix specs\n")
	sb.WriteString("4. **Never work on closed items**: Commands like `/diagnose` and `/deliver` check if the tracker issue is still open before starting work\n\n")

	sb.WriteString("### CLI Commands\n\n")
	sb.WriteString("These are run in the terminal, not as slash commands:\n")
	sb.WriteString("- `hero status` — workspace state and active specs\n")
	sb.WriteString("- `hero search <query>` — find specs by keyword\n")
	sb.WriteString("- `hero import` — import issues from tracker as spec scaffolds\n")
	sb.WriteString("- `hero pull <slug>` — sync spec status from tracker\n")
	sb.WriteString("- `hero note <slug>` — quick note capture\n")
	sb.WriteString("- `hero check` — health check\n\n")

	sb.WriteString("### Project Structure\n\n")
	sb.WriteString("- `commands/` — Slash command definitions (workflows like /design, /deliver, /diagnose)\n")
	sb.WriteString("- `agents/` — Specialized agent roles (feature-delivery-lead, debug-investigator, etc.)\n")
	sb.WriteString("- `skills/` — Domain-specific knowledge and patterns\n")
	sb.WriteString("- `.hero/planning/` — Active specs being worked on\n")
	sb.WriteString("- `.hero/specs/` — Completed specs (archive)\n")
	sb.WriteString("- `.hero/knowledge/` — Project knowledge base (conventions, decisions, context)\n")
	sb.WriteString("- `hero.json` — Project configuration\n\n")

	sb.WriteString("### Important Rules\n\n")
	sb.WriteString("- **Don't assume.** Surface tradeoffs and ask questions if anything is unclear. Present multiple interpretations instead of picking one silently.\n")
	sb.WriteString("- **Simplicity first.** Write the minimum code that solves the problem. No speculative features, no unnecessary abstractions, and no error handling for impossible scenarios.\n")
	sb.WriteString("- **Surgical changes.** Touch only what is strictly required. Do not \"improve\" nearby code or refactor unrelated sections. Match the existing style perfectly.\n")
	sb.WriteString("- **Verify before reporting done.** Define clear success criteria for every task. Run tests or validation scripts and iterate until the criteria are met before reporting completion.\n")
	sb.WriteString("- **Local specs first.** When asked to work on bugs, features, or any tracked items, ALWAYS check what's already imported locally before querying the tracker. Use `hero search --list --type <type>` to find local specs. Only go to the tracker if the local search comes up empty. When working on multiple items (e.g. \"diagnose 10 bugs\"), select from locally imported specs — never bulk-query the tracker to pick work items.\n")
	sb.WriteString("- Always check spec status before doing work — don't investigate closed bugs or deliver completed specs\n")
	sb.WriteString("- When a tracker is configured, sync status with `hero pull` before starting work\n")
	sb.WriteString("- **Hero handoff travels with commits.** When committing, stage any modified `.hero/NEXT.md` and `.hero/next/*.md` alongside your code changes. These are projected handoff files — if they don't travel with the commit, the next session (possibly on another machine) starts cold. `hero next install-hooks` installs a pre-commit hook that automates this; the rule is your backstop when the hook isn't installed.\n")
	sb.WriteString("- Capture novel learnings to `.hero/knowledge/` at the end of major workflows\n")
	sb.WriteString("- Specs use YAML frontmatter with fields: title, type, status, tracker_id, priority, severity\n")
	sb.WriteString("- Imported specs include tracker-prefixed fields (e.g. jira_status, jira_priority, jira_assignee) under a # Jira/GitHub/Linear comment header")

	return sb.String()
}
