package install

import (
	"fmt"
	"io/fs"
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
		// Global-mode AGENTS.md only has a defined home for the two
		// harnesses that natively read a global AGENTS.md (Codex,
		// OpenCode). Cursor/Copilot/Generic have no global root
		// instruction file, so return "" rather than mis-writing one
		// into an unrelated harness dir.
		switch opts.Target {
		case TargetCodex:
			return filepath.Join(home, ".codex", "AGENTS.md")
		case TargetOpenCode:
			return filepath.Join(home, ".config", "opencode", "AGENTS.md")
		default:
			return ""
		}
	default:
		return ""
	}
}

// nativeInstructionFile returns the base name of the root instruction file
// a target natively reads. Claude Code reads CLAUDE.md; every other harness
// (codex, opencode, cursor, copilot, generic) reads AGENTS.md. This is the
// single source of truth for the harness-native install mapping — no target
// can silently miss coverage because each run<Target> routes through
// installNativeInstructionFile, which consults this function.
func nativeInstructionFile(t Target) string {
	if t == TargetClaude {
		return "CLAUDE.md"
	}
	return "AGENTS.md"
}

// installNativeInstructionFile writes Hero's managed block into the one root
// instruction file the current target natively reads (per
// nativeInstructionFile). Claude → CLAUDE.md (installClaudeMd semantics,
// including --no-touch-claude-md and legacy-symlink cleanup); every other
// target → AGENTS.md. Both files share the same managed body via
// defaultSections, so a multi-target install with Claude produces byte-
// identical managed regions in CLAUDE.md and AGENTS.md.
func installNativeInstructionFile(opts Options, result *Result) error {
	if nativeInstructionFile(opts.Target) == "CLAUDE.md" {
		_, claudeMdPath, err := resolveClaudePaths(opts)
		if err != nil {
			return err
		}
		return installClaudeMd(opts, result, claudeMdPath)
	}
	agentsMdPath := resolveAgentsMdPath(opts)
	if agentsMdPath == "" {
		return nil
	}
	return installAgentsMd(opts, result, agentsMdPath)
}

// instructionFileIsHeroManagedOnly reports whether a root instruction file's
// content is entirely Hero-owned — nothing outside the managed markers
// except (optionally) the default H1 that Hero itself writes for a fresh
// file. This is the "safe to delete" predicate for
// --prune-orphaned-instruction-files: any user-authored content outside the
// markers makes it return false, so a file a user has edited is never
// pruned. Extends managed.IsLegacyHeroStub, which does not tolerate the
// Hero-written default H1.
func instructionFileIsHeroManagedOnly(content, defaultH1 string) bool {
	region := managed.FindManagedRegion(content)
	if !region.Present {
		return false
	}
	prefix := strings.TrimSpace(content[:region.StartIdx])
	suffix := strings.TrimSpace(content[region.EndIdx:])
	if suffix != "" {
		return false
	}
	return prefix == "" || prefix == strings.TrimSpace(defaultH1)
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

// InstructionFileOrphanAction is the outcome of applying the upgrade
// orphan-instruction-file policy to a single root instruction file.
type InstructionFileOrphanAction string

const (
	// OrphanAbsent — no such file on disk; nothing done (never created).
	OrphanAbsent InstructionFileOrphanAction = "absent"
	// OrphanPreserved — file has user content and no Hero managed region;
	// left byte-for-byte untouched.
	OrphanPreserved InstructionFileOrphanAction = "preserved"
	// OrphanMaintained — file carries a Hero managed region for a target
	// not in the resolved set; the region was regenerated in place so it
	// doesn't rot, all content outside the markers preserved.
	OrphanMaintained InstructionFileOrphanAction = "maintained"
	// OrphanPruned — file was entirely Hero-managed, its target is not in
	// the resolved set, and --prune-orphaned-instruction-files was set;
	// the file was deleted.
	OrphanPruned InstructionFileOrphanAction = "pruned"
)

// ApplyOrphanInstructionFilePolicy handles a single root instruction file
// (AGENTS.md or CLAUDE.md at fileName) whose owning target is NOT in the
// resolved upgrade/install set. It enforces the migration-safety invariant:
//
//   - Absent file → OrphanAbsent, no write. Orphan handling never creates
//     an instruction file for an un-inferred target.
//   - No Hero managed region (pure user file) → OrphanPreserved, untouched.
//   - prune==true AND entirely Hero-managed → the file is deleted
//     (OrphanPruned). A file with any user content outside the markers is
//     never deleted, even with prune set.
//   - Otherwise (Hero-managed region present) → the region is regenerated
//     in place (OrphanMaintained), preserving user content outside it.
//
// opts supplies the content FS, version, project dir, and DryRun. opts.Target
// only affects managed-body rendering (the Codex-specific section); callers
// pass a representative target for the file.
func ApplyOrphanInstructionFilePolicy(opts Options, fileName string, prune bool) (InstructionFileOrphanAction, error) {
	root := opts.projectRoot()
	if root == "" {
		return OrphanAbsent, nil
	}
	path := filepath.Join(root, fileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return OrphanAbsent, nil
	}
	content := string(data)
	defaultH1 := "# " + fileName

	if !managed.FindManagedRegion(content).Present {
		// Pure user-authored file — never touch.
		return OrphanPreserved, nil
	}

	if prune && instructionFileIsHeroManagedOnly(content, defaultH1) {
		if opts.DryRun {
			return OrphanPruned, nil
		}
		if err := os.Remove(path); err != nil {
			return "", err
		}
		return OrphanPruned, nil
	}

	// Maintain the managed region in place so an orphaned-but-managed file
	// doesn't rot. installManagedMarkdown short-circuits to zero writes when
	// the regenerated content already matches (idempotent).
	res := &Result{}
	if err := installManagedMarkdown(opts, res, installManagedSpec{
		Path:      path,
		Label:     fileName,
		DefaultH1: defaultH1,
		Sections:  defaultSections(opts, path),
	}); err != nil {
		return "", err
	}
	return OrphanMaintained, nil
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

// loadPackAgentsMdBody resolves the active pack's AGENTS.md body through
// the fixed chain documented in domain-routing-and-agents:
//
//  1. Explicit override (opts.AgentsMdBodyOverride) — test seam + future
//     third-party-pack seam. Skips all FS lookups.
//  2. Active pack on disk: AGENTS.md at the root of opts.sourceFS()
//     (the install pipeline points this at OverlayFS(domainFS, coreFS),
//     where domainFS is rooted at domains/<domain>/).
//  3. Engineering Go fallback (generateEngineeringAgentsMdBody) — last-
//     resort floor when the pack's AGENTS.md is missing or unreadable.
//
// Returns the body (everything after the leading H1 line) and the title
// (the H1 text, minus the leading "# "). When the chain falls through
// to the Go fallback, title is empty and the caller renders the legacy
// "Hero — Spec-Driven AI Engineering" H2.
//
// Pack authors: the heading depth, section order, and path-placeholder
// rules a pack body must follow are documented in
// .hero/knowledge/conventions/domain-agents-md-skeleton.md.
func loadPackAgentsMdBody(opts Options) (body, title string, fellBack bool) {
	if len(opts.AgentsMdBodyOverride) > 0 {
		raw := string(opts.AgentsMdBodyOverride)
		b, t := splitPackAgentsMd(raw)
		return b, t, false
	}

	srcFS := opts.sourceFS()
	if srcFS != nil {
		if data, err := fs.ReadFile(srcFS, "AGENTS.md"); err == nil && len(data) > 0 {
			b, t := splitPackAgentsMd(string(data))
			return b, t, false
		}
	}

	if opts.Domain != "" && opts.Domain != "engineering" {
		fmt.Fprintf(os.Stderr, "warning: domain %q has no AGENTS.md — falling back to engineering routing table\n", opts.Domain)
	}
	return generateEngineeringAgentsMdBody(resolveContentPathsForBody(opts)), "", true
}

// splitPackAgentsMd separates the leading H1 line from the rest of the
// pack's AGENTS.md content. The first non-blank line beginning with "# "
// becomes the title; everything after that line (with one leading blank
// line stripped) is the body. Files without an H1 yield an empty title
// and the whole content as the body — the loader will then emit the
// legacy H2 from the section contributor.
func splitPackAgentsMd(content string) (body, title string) {
	content = strings.TrimLeft(content, "\n")
	if !strings.HasPrefix(content, "# ") {
		return strings.TrimRight(content, "\n"), ""
	}
	nl := strings.IndexByte(content, '\n')
	if nl < 0 {
		return "", strings.TrimSpace(strings.TrimPrefix(content, "# "))
	}
	title = strings.TrimSpace(strings.TrimPrefix(content[:nl], "# "))
	rest := content[nl+1:]
	rest = strings.TrimLeft(rest, "\n")
	return strings.TrimRight(rest, "\n"), title
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
// after `hero scan`. See
// .hero/knowledge/conventions/domain-agents-md-skeleton.md for the
// full skeleton (heading depth, section order, path placeholders) every
// pack body — including this one — must follow.
// RenderAgentsMdBodyForDriftTest exposes the rendered managed-region
// body (orchestrator output, contributors stitched together) for the
// markdown invocation drift test in internal/cli. The shape is what
// install would write inside the managed markers for AGENTS.md /
// CLAUDE.md given default content paths and the renderable section
// contributors. Kept narrow on purpose: the function takes no
// arguments and produces deterministic output suitable for grep-style
// invocation extraction.
//
// Forces the Go fallback path (override = empty, no sourceFS) so the
// drift test sees the legacy body shape regardless of how the active
// pack's AGENTS.md happens to phrase its CLI references.
func RenderAgentsMdBodyForDriftTest() []byte {
	body := generateEngineeringAgentsMdBody(resolveContentPathsForBody(Options{}))
	return []byte(body)
}

// agentsMdBodySection adapts the active pack's AGENTS.md body to
// managed.SectionContributor. The section's H2 ("Hero — Spec-Driven AI
// Engineering" for engineering, "Hero PM — ..." for PM) is rendered by
// the orchestrator from SectionTitle; the body returned by Render is
// everything after the pack file's H1.
//
// The body and title are loaded eagerly at construction time so
// SectionTitle() can return the pack-sourced H2 without re-reading the
// FS on every call.
type agentsMdBodySection struct {
	opts  Options
	body  string
	title string
}

func newAgentsMdBodySection(opts Options) agentsMdBodySection {
	body, title, _ := loadPackAgentsMdBody(opts)
	return agentsMdBodySection{opts: opts, body: body, title: title}
}

func (s agentsMdBodySection) SectionID() string { return "install:agents-md-body" }
func (s agentsMdBodySection) SectionTitle() string {
	if s.title != "" {
		return s.title
	}
	return "Hero — Spec-Driven AI Engineering"
}

func (s agentsMdBodySection) Render(_ managed.Context) (string, error) {
	body := s.body + renderActiveDialectBlock(s.opts)
	if s.opts.Target == TargetCodex {
		body += renderCodexWorkflowSection()
	}
	return body, nil
}

// renderCodexWorkflowSection appends a Codex-specific section that teaches
// the agent to execute Hero workflows via skill files. Codex's SlashCommand
// is a built-in enum — it cannot load external command definitions. Hero
// emits each command as a skill (command-<name>/SKILL.md); this section
// tells the agent to read and follow those files step-by-step rather than
// treating them as documentation.
func renderCodexWorkflowSection() string {
	var sb strings.Builder
	sb.WriteString("\n\n### Running Hero Workflows in Codex\n\n")
	sb.WriteString("Hero's workflow commands are **not slash commands in Codex** — they are skill files you read and follow step-by-step.\n\n")
	sb.WriteString("**When the user asks you to deliver, diagnose, design, or run any Hero workflow:**\n\n")
	sb.WriteString("1. Read the workflow skill file at `.agents/skills/command-<name>/SKILL.md`\n")
	sb.WriteString("   (e.g. `.agents/skills/command-deliver/SKILL.md` when the user says \"deliver\")\n")
	sb.WriteString("2. Follow each step in the file as your workflow. These are **instructions to execute**, not documentation.\n")
	sb.WriteString("3. **Do NOT** skip steps, flip spec frontmatter as a shortcut, or treat the workflow as informational.\n\n")
	sb.WriteString("**Workflow routing table for Codex:**\n\n")
	sb.WriteString("| User intent | Skill file to read and follow |\n|---|---|\n")
	sb.WriteString("| Deliver, implement, ship, execute | `.agents/skills/command-deliver/SKILL.md` |\n")
	sb.WriteString("| Diagnose, investigate, debug, fix | `.agents/skills/command-diagnose/SKILL.md` |\n")
	sb.WriteString("| Design, plan, spec, add feature | `.agents/skills/command-design/SKILL.md` |\n")
	sb.WriteString("| Review, PR, pull request | `.agents/skills/command-review/SKILL.md` |\n")
	sb.WriteString("| Check, health, validate workspace | `.agents/skills/command-check/SKILL.md` |\n")
	sb.WriteString("| Note, capture, remember | `.agents/skills/command-note/SKILL.md` |\n")
	sb.WriteString("| Compose, break down, epic | `.agents/skills/command-compose/SKILL.md` |\n")
	sb.WriteString("| Discover, brainstorm, explore | `.agents/skills/command-discover/SKILL.md` |\n\n")
	sb.WriteString("If the skill file doesn't exist, fall back to reading `.claude/commands/<name>.md` directly.\n\n")
	sb.WriteString("**A Hero workflow is not finished until its closing gate runs.** For `/deliver`, that gate is `hero spec verify <slug>` passing — and verify requires the cold delivery audit to have run first. Do NOT yield back to the user with a spec still in `planning` or `delivering` and the audit unrun. The audit and verify run in the **same turn** as the implementation — they are not a follow-up step the user triggers later. If you find yourself about to say \"the audit still needs to run\" or \"I did not mark the spec complete because the gate still needs to run\" — **run it now instead.** Stopping one step short of the closing gate is an unfinished delivery, not a handoff. This holds in every delivery mode, including the default supervised mode: \"pause at handoffs\" does not include the closing gates.\n")
	return sb.String()
}

// generateEngineeringAgentsMdBody renders the engineering pack body
// inline as a last-resort fallback when the loader can't find a pack
// AGENTS.md on disk. Kept content-identical to the canonical
// domains/engineering/AGENTS.md (enforced by
// TestEngineeringPackBodyMatchesGoFallback) until a release window
// passes with no fallback-path bug reports and the function can be
// removed.
func generateEngineeringAgentsMdBody(paths contentPathsForBody) string {
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
	sb.WriteString("| Autopilot/run a whole initiative, \"put X on autopilot\", \"drive the initiative\", keep working autonomously | `/drive <initiative>` |\n")
	sb.WriteString("| Review, PR, pull request, code review | `/review` |\n")
	sb.WriteString("| Break down, decompose, epic, sequence | `/compose` |\n")
	sb.WriteString("| Convention, pattern, standard, style | `/convention` |\n")
	sb.WriteString("| Decision, tradeoff, compare, choose, ADR | `/decide` |\n")
	sb.WriteString("| Explore, brainstorm, roadmap, ideate | `/discover` |\n")
	sb.WriteString("| Mockup, mock, wireframe, prototype, visualize a screen, \"what would X look like\", \"is that a swift mock?\" | `/mock` |\n")
	sb.WriteString("| Document, docs, explain, write docs | `/docs` |\n")
	sb.WriteString("| Release, deploy, version, ship | `/release` |\n")
	sb.WriteString("| Retro, postmortem, lessons learned | `/retro` |\n")
	sb.WriteString("| Note, capture, remember, save thought | `/note` |\n")
	sb.WriteString("| Scan, detect, onboard, stack analysis | `/scan` |\n")
	sb.WriteString("| Check, health, validate workspace | `/check` |\n")
	sb.WriteString("| Sprint, iteration, load sprint | `/sprint` |\n")
	sb.WriteString("| Import, pull issues, fetch from tracker, sync issues | `/import` |\n")
	sb.WriteString("| What's stuck, blocked items, dependencies, can't move forward | `/blocked` |\n")
	sb.WriteString("| Capture, extract learnings, persist session knowledge to the knowledge base | `/capture` |\n")
	sb.WriteString("| Challenge or revise a diagnosis, push back on root cause with new context | `/challenge` |\n")
	sb.WriteString("| Start of session, load ranked context, what's in flight | `/resume` |\n")
	sb.WriteString("| Roadmap drift triage, \"review the roadmap for staleness\" | `/roadmap-review` |\n")
	sb.WriteString("| Scrub the codebase — dead code, weak types, duplication, bad comments, legacy cruft | `/scrub` |\n")
	sb.WriteString("| Break a large spec into smaller, independently deliverable child specs | `/split` |\n")
	sb.WriteString("| Trace where something came from, chain of decisions/specs/commits | `/why` |\n")
	sb.WriteString("| Not sure which command to use, route my request | `/hero` |\n")
	sb.WriteString("| Ask sibling/peer repo a question, check with peer | `hero peer call <alias> --mode=advisory \"...\"` |\n")
	sb.WriteString("| Have peer design something, let peer handle design | `hero peer call <alias> --mode=spec-out \"...\"` |\n")
	sb.WriteString("| Hand off a spec to a peer repo, drop on peer's queue, transfer to sibling | `hero handoff <spec> <alias>` |\n")
	sb.WriteString("| Pick up handed-back spec, accept the handoff, peer finished | `hero handoff accept <spec>` |\n")
	sb.WriteString("| What peers do we have, list siblings, which repos are linked | `hero peer list` |\n")
	sb.WriteString("| What does peer expose, peer surface, peer conventions, inspect peer | `hero peer show <alias>` |\n")
	sb.WriteString("| Cross-repo peering front door (session-level; picks advisory/spec-out/handoff/list/show for you) | `/peer` |\n")
	sb.WriteString("| Force-refresh NEXT.md/QUEUE.md before switching tools (session-level; distinct from the cross-repo rows above) | `/handoff` |\n\n")

	sb.WriteString("When routing, pass the user's original context as arguments to the command. ")
	sb.WriteString("If the intent is ambiguous, present the top 2-3 options and ask.\n\n")

	sb.WriteString("**Slash commands ≠ CLI subcommands.** Slash commands (e.g. `/discover`, `/convention`) run inside the AI tool's session only — they are **not** `hero discover` or `hero convention` terminal commands. Some commands exist on both surfaces, but many are slash-only. Do not hallucinate CLI subcommands from slash command names. <!-- drift-test:ignore (illustrative: `hero discover`/`hero convention` above are explicitly non-existent subcommands) -->\n\n")
	sb.WriteString("| Surface | Commands |\n|---|---|\n")
	sb.WriteString("| **Slash-only** (no `hero <name>` equivalent) | `/capture`, `/challenge`, `/compose`, `/convention`, `/decide`, `/discover`, `/drive`, `/mock`, `/release`, `/retro`, `/review`, `/roadmap-review`, `/scrub`, `/split` |\n")
	sb.WriteString("| **Both slash and CLI** | `/blocked`, `/check`, `/deliver`, `/design`, `/diagnose`, `/docs`, `/handoff` (slash = NEXT.md refresh; CLI `hero handoff <spec> <alias>` = cross-repo drop to a peer), `/hero` (\"which command do I use\" meta-help; CLI equivalent `hero do <request>`), `/import` (slash = tracker import via `hero sync import`; root `hero import` is unrelated knowledge-base ingestion), `/note`, `/peer`, `/resume`, `/scan`, `/sprint`, `/why` |\n")
	sb.WriteString("| **CLI-only** (see CLI Commands below) | `hero status`, `hero search`, `hero ask`, `hero list`, `hero queue`, `hero spec verify`, `hero spec score`, `hero diff`, `hero drift`, etc. |\n\n")

	sb.WriteString("**Mockup routing.** Any request to mock, wireframe, prototype, or visualize a screen — including casual questions like \"what would this look like?\" or \"is that a swift mock?\" — routes to `/mock`. **Never hand-generate a mockup outside that command, and never pick the format yourself.** `/mock` runs `hero spec mock detect`, which chooses the renderer (HTML vs. native SwiftUI) deterministically from the repo's stack and announces it before generating. There is **no \"HTML-first, then port to SwiftUI\" workflow** — that is a confabulation, not a real Hero pattern. In a native app you produce a native SwiftUI mockup directly (compiled, with real screenshots); in a web app you produce HTML. Do **not** generate an HTML approximation \"to iterate faster\" on a native project. Always end your response with the clickable file inventory `/mock` surfaces — never make the user ask for the links.\n\n")

	sb.WriteString("**Cross-repo peering disambiguation.** The session-level `/handoff` slash command (force-refresh NEXT.md) and the cross-repo `hero handoff <spec> <alias>` command share a verb but do different things. Disambiguate by whether the user names a peer alias: if they do, it's cross-repo; if not, it's session handoff. When a user says \"ask hero-code about X\" or \"hand off to hero-cloud,\" route to the cross-repo command and **compose the prompt yourself** — don't paraphrase the user's words verbatim. A good peer-call prompt names the specific question, references the active spec via `--related-spec <slug>` when one exists, and includes `--reason` explaining why the call is happening. Pick the mode: **advisory** (need a fact, peer writes nothing), **spec-out** (peer designs the fix on its side), or **handoff** (you already did the investigation, dropping it on peer's queue).\n\n")

	sb.WriteString("### Key Workflow\n\n")
	sb.WriteString("1. **Design first**: Use `/design` to create a spec before building anything\n")
	sb.WriteString("2. **Deliver from spec**: Use `/deliver` to implement from an approved spec\n")
	sb.WriteString("3. **Debug with specs**: Use `/diagnose` to investigate bugs and produce fix specs\n")
	sb.WriteString("4. **Never work on closed items**: Commands like `/diagnose` and `/deliver` check if the tracker issue is still open before starting work\n")
	sb.WriteString("5. **Finish the closing gate before yielding**: `/deliver` is not done until `hero spec verify <slug>` passes — and verify requires the cold delivery audit to run first. The audit and verify run in the **same turn** as the implementation, not as a follow-up the user triggers. Never stop with a spec left in `planning`/`delivering` and the audit unrun, and never say \"the audit still needs to run\" — run it now instead. This holds in every delivery mode, including the default supervised mode.\n\n")

	sb.WriteString("### Agents Reference\n\n")
	sb.WriteString("Grouped by role (every installed agent, no links):\n\n")
	sb.WriteString("- **Delivery leads:** feature-delivery-lead, platform-delivery-lead — product features vs. platform/migration work.\n")
	sb.WriteString("- **Architects & reviewers:** greenfield-architect, brownfield-architect, architecture-reviewer, design-reviewer, pr-reviewer, security-reviewer, roadmap-reviewer — design-time and review gates.\n")
	sb.WriteString("- **Specialist engineers:** engineer, api-engineer, database-engineer, devops-engineer, integration-engineer, migration-engineer, performance-engineer, release-engineer — build and ship by concern.\n")
	sb.WriteString("- **QA & investigation:** functional-qa-engineer, test-architect, debug-investigator, dependency-analyst, issue-tracker, product-ideator, ui-designer — testing, root-cause work, dependency mapping, issue triage, ideation, UI review.\n")
	sb.WriteString("- **Scrubbers:** comment-scrubber, deadcode-scrubber, dedup-scrubber, defensive-scrubber, dependency-scrubber, legacy-scrubber, type-scrubber — one code-quality concern each.\n")
	sb.WriteString("- **Core (installed with every pack):** convention-author, documentation-engineer, project-context-builder, session-primer.\n\n")

	sb.WriteString("### Skills Reference\n\n")
	sb.WriteString("Grouped by concern (every installed skill, no links):\n\n")
	sb.WriteString("- **Stacks & detection:** database-stack, go-stack, groovy-stack, java-stack, javascript-stack, python-stack, react-stack, rust-stack, stack-detection — conventions per detected stack.\n")
	sb.WriteString("- **Architecture & design:** api-design-and-contracts, architecture-principles, greenfield-scaffolding, implementation-principles, integration-boundaries — design-time reasoning for new and evolving systems.\n")
	sb.WriteString("- **Delivery & spec process:** batch-discipline, delivery-audit, drive, spec-composition, spec-sizing — sizing, composing, delivering, and cold-auditing specs.\n")
	sb.WriteString("- **Investigation & quality:** challenge-diagnosis, debugging-investigation, dependency-analysis, pr-review, root-cause-classification, security-review, test-strategy, testing-and-validation — diagnosing, reviewing, testing.\n")
	sb.WriteString("- **Scrub:** code-scrub — shared methodology behind the scrubber agents.\n")
	sb.WriteString("- **Ops, incident & release:** devops-and-operations, incident-response, release-and-deployment — production operations lifecycle.\n")
	sb.WriteString("- **Mockups:** html-mockup-generation, swiftui-mockup-renderer — the two `/mock` renderer paths.\n")
	sb.WriteString("- **Cross-repo & reporting:** cross-repo-peering, deep-code-enrichment, issue-list-report — peer calls, enrichment passes, report formatting.\n")
	sb.WriteString("- **Roadmap & performance:** performance-optimization, roadmap-review — perf tuning and roadmap-shape triage.\n")
	sb.WriteString("- **Migration:** migration-safety — safe migration/refactor patterns.\n")
	sb.WriteString("- **Core (installed with every pack):** agent-reliability, auto-knowledge-capture, completion-ledger, context-injection, convention-writing, documentation-practices, executive-report, explainer-format, kickoff-prompt, knowledge-flywheel, next-handoff-emit, next-md, note-capture, nudge-awareness, project-context-generation, spec-format.\n\n")

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
	sb.WriteString("`hero install` **writes** these into your harness's own directory in that harness's native format — e.g. `.claude/commands/`, `.claude/agents/`, and `.claude/skills/` for Claude; `.codex/agents/*.toml` (TOML) plus workflow skills under `.agents/skills/` for Codex (Codex has no commands directory — its slash commands are a built-in enum, so Hero commands install there as skills). They are generated copies, **not** symlinks or views: re-running `hero install` regenerates them, so hand-edits to the installed files are overwritten on the next install.\n\n")

	sb.WriteString("### Declaring Spec Relationships\n\n")
	sb.WriteString("Relationships (parent/child, depends-on, blocks) become knowledge-graph edges **only** through frontmatter. Body `[[wikilinks]]` are searchable text and form **no** edges. Two syntaxes work:\n\n")
	sb.WriteString("Top-level shorthand (simplest):\n\n")
	sb.WriteString("```yaml\nparent: i1-config-plane          # also accepted: initiative: i1-config-plane\ndepends-on: [f2-store, f3-watcher]   # also accepted: depends_on:\nchild:\n  - sub-a\n  - sub-b\n```\n\n")
	sb.WriteString("`relations:` block (for mixed kinds):\n\n")
	sb.WriteString("```yaml\nrelations:\n  - target: i1-config-plane\n    kind: parent\n  - target: other-spec\n    kind: related\n```\n\n")
	sb.WriteString("Pitfalls: inline flow style (`- { kind: parent, target: x }`) does **not** parse — use the block form with `target:`/`kind:` on separate lines. Recognized kinds: `parent`, `child`, `depends-on`, `blocks`, `supersedes`, `related`. `hero check` warns when a spec uses edge-intent `[[wikilinks]]`.\n\n")

	sb.WriteString("### Internal Lookups — Tool Routing\n\n")
	sb.WriteString("When **you** need to look something up mid-task (as opposed to running a slash command for the user), pick the tool that matches the *shape* of the question, not the one that feels exhaustive:\n\n")
	sb.WriteString("| Shape of question | Tool |\n|---|---|\n")
	sb.WriteString("| \"Does spec/knowledge entry X exist? Has this been discussed?\" | `hero_search` with `compact: true` — single-line count, no excerpt noise |\n")
	sb.WriteString("| \"What's the status / frontmatter of spec X?\" | `hero_read_spec` |\n")
	sb.WriteString("| \"What's in flight / ready / blocked / mine?\" | `hero_list`, `hero_queue`, `hero_blocked` |\n")
	sb.WriteString("| \"Where did this come from? What chain of decisions led here?\" | `hero_why` — graph traversal beats grep on relations |\n")
	sb.WriteString("| Literal string `foo_bar_baz` across code | `rg` / `grep` |\n")
	sb.WriteString("| Known file at a known path | `Read` |\n")
	sb.WriteString("| Recent commits / git history | `git log` |\n")
	sb.WriteString("| Broad exploration across many files | a context-protective read-only search subagent, where your harness provides one (e.g. Claude Code's `Explore` agent); otherwise `rg` + targeted reads |\n\n")
	sb.WriteString("**Rule of thumb:** graph- or spec-shaped questions → Hero MCP tools (`hero_*` — on Claude Code these surface as `mcp__hero__<name>`). String-shaped → grep. File-shaped → Read. Don't reach for `grep` on `.hero/` to answer \"does spec X exist?\" — substring search only finds *literal matches*, not *semantically related* specs (e.g. a spec slugged `domain-routing-and-agents` is the same concept as \"domain swap\" but won't match either word as a phrase).\n\n")
	sb.WriteString("Some harnesses defer MCP tool schemas behind a one-time lookup before the tool is callable — e.g. Claude Code's `ToolSearch`. The load is one round-trip and worth it; it's not a reason to fall back to a weaker tool.\n\n")
	sb.WriteString("**Prefer Hero's MCP tools over shelling out to a bare `hero` in a terminal.** A GUI-launched harness can resolve a *different or stale* `hero` binary on its PATH than your login shell does; the MCP surface is the in-process Hero you're already connected to, so it can't drift out from under you. When you must use the CLI and hit a schema/version mismatch or a confusing `hero` version error, **run `hero doctor` and act on its output** — it reports which binary is actually on PATH, its schema, the graph's schema, and the real remediation. Do NOT invent a schema-migration narrative, and do NOT run `hero upgrade` to \"fix schema\": `hero upgrade` updates workspace files, not the binary, so it cannot fix a wrong-binary-on-PATH situation.\n\n")

	sb.WriteString("### Important Rules\n\n")
	sb.WriteString("- **Don't assume.** Surface tradeoffs and ask questions if anything is unclear. Present multiple interpretations instead of picking one silently.\n")
	sb.WriteString("- **Honest over agreeable.** Push back when you disagree — say what's wrong, propose the better path, then proceed. Don't reverse your position because the user pushed; reverse it when new evidence warrants it.\n")
	sb.WriteString("- **Label what you know vs. think.** State facts as facts and opinions as opinions. \"I'm not sure\" beats a confident guess.\n")
	sb.WriteString("- **Say the hard thing.** If the user's approach has a flaw, point it out before implementing. If a request conflicts with these rules, name the conflict rather than silently following.\n")
	sb.WriteString("- **Simplicity first.** Write the minimum code that solves the problem. No speculative features, no unnecessary abstractions, and no error handling for impossible scenarios.\n")
	sb.WriteString("- **Surgical changes.** Touch only what is strictly required. Do not \"improve\" nearby code or refactor unrelated sections. Match the existing style perfectly.\n")
	sb.WriteString("- **Verify before reporting done.** Define clear success criteria for every task. Run tests or validation scripts and iterate until the criteria are met before reporting completion.\n")
	sb.WriteString("- **Local specs first.** When asked to work on bugs, features, or any tracked items, ALWAYS check what's already imported locally before querying the tracker. Use `hero search --list --type <type>` to find local specs. Only go to the tracker if the local search comes up empty. When working on multiple items (e.g. \"diagnose 10 bugs\"), select from locally imported specs — never bulk-query the tracker to pick work items.\n")
	sb.WriteString("- Always check spec status before doing work — don't investigate closed bugs or deliver completed specs\n")
	sb.WriteString("- When a tracker is configured, sync status with `hero sync pull` before starting work\n")
	sb.WriteString("- **Hero handoff travels with commits.** Projected handoff files (`.hero/NEXT.md`, `.hero/next/*.md`, `.hero/SNAPSHOT.md`, `.hero/QUEUE.md`) must travel with the commit or the next session (possibly on another machine) starts cold. Every Hero hook install path now wires a pre-commit hook that stages these automatically — you don't normally need to think about it. `hero check` flags a repo where the staging block is missing. As a backstop only, if `hero check` warns that staging isn't wired and you can't install hooks, stage the projected handoff files by hand alongside your code changes.\n")
	sb.WriteString("- Capture novel learnings to `.hero/knowledge/` at the end of major workflows\n")
	sb.WriteString("- Specs use YAML frontmatter with fields: title, type, status, tracker_id, priority, severity\n")
	sb.WriteString("- Imported specs include tracker-prefixed fields (e.g. jira_status, jira_priority, jira_assignee) under a # Jira/GitHub/Linear comment header")

	return sb.String()
}
