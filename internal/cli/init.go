package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/peering"
	"github.com/hero-engine/hero/internal/scan"
	"github.com/hero-engine/hero/internal/version"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a hero workspace in the current project",
	Long: `Creates the .hero/ folder structure with a default hero.json configuration file.

Also scans the project to detect the tech stack and generates an AGENTS.md file
with project-specific build, test, and lint commands so AI coding agents can
work reliably from the first session. Use --no-agents to skip AGENTS.md generation.

Non-interactive — safe to run from CI / setup scripts. Re-running on an
existing workspace is a no-op for the directory structure (it won't clobber
custom config) but will refresh AGENTS.md unless --no-agents is set.`,
	RunE: runInit,
}

var (
	initFolder       string
	initDomain       string
	initNoAgents     bool
	initInstallHooks bool
	initNoHooks      bool
)

func init() {
	initCmd.Flags().StringVar(&initFolder, "folder", config.DefaultFolder, "folder name for the hero workspace")
	initCmd.Flags().StringVar(&initDomain, "domain", "", "domain pack to use (default: engineering); see `hero domain list`")
	initCmd.Flags().BoolVar(&initNoAgents, "no-agents", false, "skip AGENTS.md generation")
	initCmd.Flags().BoolVar(&initInstallHooks, "install-hooks", true, "install the pre-commit hook so projected NEXT files travel with commits")
	initCmd.Flags().BoolVar(&initNoHooks, "no-hooks", false, "skip installing the pre-commit hook")
}

func runInit(cmd *cobra.Command, args []string) error {
	projectRoot := findGitRoot()

	cfg := config.DefaultConfig()
	cfg.Folder = initFolder
	cfg.PeerID = peering.MintPeerID()

	if initDomain != "" {
		if _, err := hero.DomainFS(initDomain); err != nil {
			return err
		}
		cfg.Domain = initDomain
	}

	heroDir := cfg.HeroDir(projectRoot)

	// Check if already initialized
	if _, err := os.Stat(heroDir); err == nil {
		return fmt.Errorf("hero workspace already exists at %s", heroDir)
	}

	// Create directory structure
	dirs := []string{
		heroDir,
		cfg.PlanningDir(projectRoot),
		filepath.Join(cfg.PlanningDir(projectRoot), "features"),
		filepath.Join(cfg.PlanningDir(projectRoot), "bugs"),
		filepath.Join(cfg.PlanningDir(projectRoot), "initiatives"),
		cfg.SpecsDir(projectRoot),
		cfg.KnowledgeDir(projectRoot),
		cfg.ConventionsDir(projectRoot),
		cfg.DecisionsDir(projectRoot),
		cfg.RulesDir(projectRoot),
		cfg.ExternalDir(projectRoot),
		cfg.ContextDir(projectRoot),
		cfg.TemplatesDir(projectRoot),
		cfg.NotesDir(projectRoot),
		cfg.MocksDir(projectRoot),
		cfg.NextDir(projectRoot),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Write config
	if err := cfg.Save(projectRoot); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// Ensure hero.local.json is gitignored — write .hero/.gitignore
	heroGitignorePath := filepath.Join(heroDir, ".gitignore")
	if _, err := os.Stat(heroGitignorePath); os.IsNotExist(err) {
		gitignoreContent := "# Per-project local overrides (tokens, personal preferences)\nhero.local.json\n"
		if werr := os.WriteFile(heroGitignorePath, []byte(gitignoreContent), 0o644); werr != nil {
			fmt.Fprintf(os.Stderr, "  warning: could not write .hero/.gitignore: %v\n", werr)
		}
	}

	// Append (or refresh) the hero-managed block in the root
	// .gitignore. Covers hero.local.json, the gitignored graph DB,
	// per-machine NEXT state, and auto-generated code intelligence.
	// Marker-bounded so re-runs are idempotent and don't accumulate
	// stale lines. Creates the file if it doesn't exist.
	rootGitignore := filepath.Join(projectRoot, ".gitignore")
	if err := ensureManagedGitignoreBlock(rootGitignore); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: could not update .gitignore: %v\n", err)
	}

	// Stamp workspace version
	binaryVersion := rootCmd.Version
	if binaryVersion == "" {
		binaryVersion = "dev"
	}
	if err := version.StampInit(heroDir, binaryVersion); err != nil {
		fmt.Fprintf(os.Stderr, "  warning: could not write version stamp: %v\n", err)
	}

	// Record the peer_id mint in the events log so the moment of
	// identity assignment is recoverable. Best-effort.
	if cfg.PeerID != "" {
		peering.RecordPeerIDMintEvent(heroDir, cfg.PeerID, "init")
	}

	fmt.Printf("Initialized hero workspace at %s\n", heroDir)
	fmt.Printf("  planning/features/    — feature specs in progress\n")
	fmt.Printf("  planning/bugs/        — bug specs in progress\n")
	fmt.Printf("  planning/initiatives/ — multi-spec initiatives\n")
	fmt.Printf("  specs/                — completed product specs\n")
	fmt.Printf("  knowledge/            — project knowledge base\n")
	fmt.Printf("    conventions/        — how we write code\n")
	fmt.Printf("    decisions/          — why we chose X over Y (ADRs)\n")
	fmt.Printf("    rules/              — hard constraints and requirements\n")
	fmt.Printf("    external/           — external docs, runbooks, references\n")
	fmt.Printf("    context/            — team, environment, deployment info\n")
	fmt.Printf("    templates/          — custom spec templates\n")
	fmt.Printf("    notes/              — brainstorms, thinking, conversation captures\n")
	fmt.Printf("  mocks/                — design mockups and visual prototypes\n")
	fmt.Printf("  hero.json             — configuration\n")
	fmt.Printf("  hero.local.json       — local overrides, gitignored (tokens, personal prefs)\n")

	// Install pre-commit hook so projected NEXT files travel with
	// commits. Belongs at init (setup-time) rather than scan
	// (analysis-time). Best-effort: a failure doesn't block init.
	if initInstallHooks && !initNoHooks && !preCommitHookInstalled(projectRoot) {
		if _, gerr := resolveGitDir(projectRoot); gerr == nil {
			if herr := installNextHooksQuiet(projectRoot); herr != nil {
				fmt.Fprintf(os.Stderr, "  warning: pre-commit hook install failed: %v\n", herr)
			} else {
				fmt.Println()
				fmt.Println("  Installed pre-commit hook (projected NEXT files will travel with commits).")
				fmt.Println("  Pass --no-hooks next time to skip; to remove, delete the marker block in .git/hooks/pre-commit.")
			}
		}
	}

	// Generate AGENTS.md if not disabled
	if !initNoAgents {
		agentsPath := filepath.Join(projectRoot, "AGENTS.md")
		if _, err := os.Stat(agentsPath); err == nil {
			fmt.Printf("\n  AGENTS.md already exists — skipping (use --no-agents to suppress)\n")
		} else {
			if err := generateAgentsMD(projectRoot, agentsPath); err != nil {
				fmt.Fprintf(os.Stderr, "\n  Warning: could not generate AGENTS.md: %v\n", err)
			} else {
				fmt.Printf("\n  Generated AGENTS.md with detected build/test/lint commands\n")
				fmt.Printf("  Review and customize it for your project.\n")
			}
		}
	}

	return nil
}

// generateAgentsMD runs a lightweight project scan and creates an AGENTS.md
// with detected build, test, lint commands and project structure. This file
// is the cross-tool standard understood by OpenCode, Cursor, Claude Code,
// and GitHub Copilot.
func generateAgentsMD(projectRoot, outputPath string) error {
	result, err := scan.Analyze(projectRoot)
	if err != nil {
		return fmt.Errorf("scanning project: %w", err)
	}

	var sb strings.Builder

	sb.WriteString("# AGENTS.md\n\n")
	sb.WriteString("Project instructions for AI coding agents.\n")
	sb.WriteString("Generated by `hero init` — review and customize for your project.\n\n")

	// Build commands
	sb.WriteString("## Build\n\n")
	if len(result.BuildTools) > 0 {
		for _, bt := range result.BuildTools {
			cmd := inferBuildCommand(bt)
			if cmd != "" {
				sb.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", cmd))
			}
		}
	} else {
		sb.WriteString("<!-- Add your build command here -->\n\n")
	}

	// Test commands
	sb.WriteString("## Test\n\n")
	if len(result.TestFrames) > 0 || len(result.BuildTools) > 0 {
		cmd := inferTestCommand(result)
		if cmd != "" {
			sb.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", cmd))
		} else {
			sb.WriteString("<!-- Add your test command here -->\n\n")
		}
	} else {
		sb.WriteString("<!-- Add your test command here -->\n\n")
	}

	// Lint commands
	sb.WriteString("## Lint\n\n")
	if len(result.Linters) > 0 {
		for _, l := range result.Linters {
			cmd := inferLintCommand(l)
			if cmd != "" {
				sb.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", cmd))
			}
		}
	} else {
		sb.WriteString("<!-- Add your lint command here -->\n\n")
	}

	// Project structure
	sb.WriteString("## Project structure\n\n")
	if len(result.Languages) > 0 {
		var langs []string
		for _, l := range result.Languages {
			langs = append(langs, l.Name)
		}
		sb.WriteString(fmt.Sprintf("Languages: %s\n\n", strings.Join(langs, ", ")))
	}
	if len(result.Frameworks) > 0 {
		var frames []string
		for _, f := range result.Frameworks {
			frames = append(frames, f.Name)
		}
		sb.WriteString(fmt.Sprintf("Frameworks: %s\n\n", strings.Join(frames, ", ")))
	}
	if len(result.Structure.TopLevelDirs) > 0 {
		sb.WriteString("Key directories:\n")
		for _, d := range result.Structure.TopLevelDirs {
			sb.WriteString(fmt.Sprintf("- `%s/`\n", d))
		}
		sb.WriteString("\n")
	}
	if len(result.Structure.EntryPoints) > 0 {
		sb.WriteString("Entry points:\n")
		for _, ep := range result.Structure.EntryPoints {
			sb.WriteString(fmt.Sprintf("- `%s`\n", ep))
		}
		sb.WriteString("\n")
	}

	// Coding conventions (the reliability rules)
	sb.WriteString("## Coding conventions\n\n")
	sb.WriteString("- **Don't assume.** Surface tradeoffs and ask questions if anything is unclear. Present multiple interpretations instead of picking one silently.\n")
	sb.WriteString("- **Simplicity first.** Write the minimum code that solves the problem. No speculative features, no unnecessary abstractions, and no error handling for impossible scenarios.\n")
	sb.WriteString("- **Surgical changes.** Touch only what is strictly required. Do not \"improve\" nearby code or refactor unrelated sections. Match the existing style perfectly.\n")
	sb.WriteString("- **Verify before reporting done.** Define clear success criteria for every task. Run tests or validation scripts and iterate until the criteria are met before reporting completion.\n")
	sb.WriteString("- Read a file before editing it\n")
	sb.WriteString("- Run tests after making changes\n")
	sb.WriteString("- Search the codebase before creating new files\n")
	sb.WriteString("- Make one logical change at a time and verify it before moving on\n")
	sb.WriteString("- Do not suppress errors or warnings to make tests pass\n")
	sb.WriteString("- If a fix attempt fails twice, stop and reassess the approach\n")
	sb.WriteString("\n<!-- Add project-specific conventions here -->\n\n")

	// Hero workspace note
	sb.WriteString("## Hero workspace\n\n")
	sb.WriteString("This project uses [Hero](https://github.com/hero-engine/hero) for spec-driven development.\n")
	sb.WriteString("Specs live in `.hero/`. Use `/design` to create feature specs, `/diagnose` for bugs,\n")
	sb.WriteString("and `/deliver` to implement against a spec. Run `hero context --files <paths>` to see\n")
	sb.WriteString("relevant conventions and past work for the files you're touching.\n")

	return os.WriteFile(outputPath, []byte(sb.String()), 0o644)
}

// inferBuildCommand guesses the build command from a detected build tool.
func inferBuildCommand(bt scan.BuildTool) string {
	switch bt.Name {
	case "Make":
		return "make build"
	case "Go":
		return "go build ./..."
	case "npm":
		return "npm run build"
	case "yarn":
		return "yarn build"
	case "pnpm":
		return "pnpm build"
	case "Maven":
		return "mvn compile"
	case "Gradle":
		return "./gradlew build"
	case "Cargo":
		return "cargo build"
	case "pip", "Poetry":
		return "# Python — no build step required"
	case "Mix":
		return "mix compile"
	default:
		return ""
	}
}

// inferTestCommand guesses the test command from detected tools and frameworks.
func inferTestCommand(r *scan.Result) string {
	// Prefer build tool test commands
	for _, bt := range r.BuildTools {
		switch bt.Name {
		case "Make":
			return "make test"
		case "Go":
			return "go test ./..."
		case "npm":
			return "npm test"
		case "yarn":
			return "yarn test"
		case "pnpm":
			return "pnpm test"
		case "Maven":
			return "mvn test"
		case "Gradle":
			return "./gradlew test"
		case "Cargo":
			return "cargo test"
		case "Mix":
			return "mix test"
		}
	}

	// Fall back to test framework
	for _, tf := range r.TestFrames {
		switch tf.Name {
		case "pytest":
			return "pytest"
		case "unittest":
			return "python -m pytest"
		case "RSpec":
			return "bundle exec rspec"
		case "Minitest":
			return "bundle exec rake test"
		}
	}

	return ""
}

// inferLintCommand guesses the lint command from a detected linter.
func inferLintCommand(l scan.Linter) string {
	switch l.Name {
	case "ESLint":
		return "npx eslint ."
	case "Prettier":
		return "npx prettier --check ."
	case "golangci-lint":
		return "golangci-lint run"
	case "Flake8":
		return "flake8"
	case "Black":
		return "black --check ."
	case "Ruff":
		return "ruff check ."
	case "mypy":
		return "mypy ."
	case "RuboCop":
		return "bundle exec rubocop"
	case "Clippy":
		return "cargo clippy"
	case "Checkstyle":
		return "./gradlew checkstyleMain"
	case "SwiftLint":
		return "swiftlint"
	default:
		return ""
	}
}

// ensureGitignoreEntry appends entry to the .gitignore file at path if the file
// exists and does not already contain a line matching the entry. Does nothing if
// the file does not exist (we only update an existing .gitignore, not create one).
//
// Kept for backward compatibility — new entries should go in
// ensureManagedGitignoreBlock so they live inside the marker-bounded
// managed block and stay coordinated with future additions.
func ensureGitignoreEntry(gitignorePath, entry string) error {
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no root .gitignore — skip
		}
		return err
	}

	// Check whether any line already covers this entry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == entry || line == "**/.hero/hero.local.json" || line == "hero.local.json" {
			return nil // already covered
		}
	}

	// Append the entry
	suffix := "\n"
	if len(data) > 0 && data[len(data)-1] != '\n' {
		suffix = "\n\n"
	} else {
		suffix = "\n"
	}
	addition := suffix + "# Hero per-project local config (tokens, personal preferences)\n" + entry + "\n"
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(addition)
	return err
}

// gitignoreMarkerStart / gitignoreMarkerEnd delimit the hero-managed
// section of the root .gitignore. Re-running ensureManagedGitignoreBlock
// only rewrites lines between these markers; user content above and
// below is preserved verbatim.
const (
	gitignoreMarkerStart = "# >>> hero gitignore (managed) >>>"
	gitignoreMarkerEnd   = "# <<< hero gitignore (managed) <<<"
)

// managedGitignoreEntries is the canonical list of paths hero needs
// gitignored. Adding a new entry here is the only change required to
// roll it out — `hero init` (and re-runs) splice it in automatically.
var managedGitignoreEntries = []string{
	"# Per-project local overrides (tokens, personal preferences)",
	".hero/hero.local.json",
	"",
	"# Knowledge graph cache (regenerable from sources of truth)",
	".hero/graph.db",
	".hero/graph.db-wal",
	".hero/graph.db-shm",
	"",
	"# Per-machine NEXT state (rewritten every Stop hook + agent scratch)",
	".hero/next/*.local.md",
	"",
	"# Auto-generated code intelligence (re-scan to regenerate)",
	".hero/knowledge/code/",
	"",
	"# Per-machine satellite manifest (which subprojects are symlinked locally)",
	".hero/satellites.local.json",
}

// ensureManagedGitignoreBlock writes (or refreshes) the marker-bounded
// hero-managed block in the root .gitignore. Creates the file if it
// doesn't exist. Idempotent — re-runs replace only the managed block.
func ensureManagedGitignoreBlock(gitignorePath string) error {
	existing, _ := os.ReadFile(gitignorePath)

	var b strings.Builder
	b.WriteString(gitignoreMarkerStart + "\n")
	for _, line := range managedGitignoreEntries {
		b.WriteString(line + "\n")
	}
	b.WriteString(gitignoreMarkerEnd)
	managed := b.String()

	body := mergeGitignoreBlock(string(existing), managed)
	return os.WriteFile(gitignorePath, []byte(body), 0o644)
}

// mergeGitignoreBlock replaces or appends the hero-managed marker
// block. Mirrors mergeMarkerBlock in next_hooks.go but specialised to
// the gitignore markers so the two sets stay independently versioned.
func mergeGitignoreBlock(src, block string) string {
	startIdx := strings.Index(src, gitignoreMarkerStart)
	if startIdx < 0 {
		if strings.TrimSpace(src) == "" {
			return block + "\n"
		}
		return strings.TrimRight(src, "\n") + "\n\n" + block + "\n"
	}
	endIdx := strings.Index(src, gitignoreMarkerEnd)
	if endIdx < 0 {
		return strings.TrimRight(src[:startIdx], "\n") + "\n\n" + block + "\n"
	}
	endIdx += len(gitignoreMarkerEnd)
	prefix := strings.TrimRight(src[:startIdx], "\n")
	suffix := strings.TrimLeft(src[endIdx:], "\n")
	if prefix == "" && suffix == "" {
		return block + "\n"
	}
	if prefix == "" {
		return block + "\n\n" + suffix
	}
	if suffix == "" {
		return prefix + "\n\n" + block + "\n"
	}
	return prefix + "\n\n" + block + "\n\n" + suffix
}
