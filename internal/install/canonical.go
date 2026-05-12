package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
)

// canonical.go — single-source-install P2 canonical content tree, with
// configurable-content-paths override.
//
// By default, `.hero/{agents,commands,skills}/` is the canonical content
// location inside an installed project. Each harness target's content
// directory becomes a symlink pointing at the canonical tree.
//
// Projects can override the canonical location via hero.json:
//
//   {
//     "content": {
//       "agents_path":   "agents",
//       "commands_path": "commands",
//       "skills_path":   "skills"
//     }
//   }
//
// When set, the configured path (project-relative) replaces the default
// .hero/<kind>/. Install does not materialize embedded content into the
// configured path — the project is asserting "this is my source
// already." This is the hero-on-hero dogfood case: hero's repo has
// agents/, commands/, skills/ at the root as embedded source, and
// pointing the canonical at those paths eliminates the rendered-copy
// duplication.
//
// Paths configured INSIDE .hero/ still get the materialization (treated
// as a moved default).

// Local alias keeps the installCanonical body readable; using os.Stat
// directly keeps the call site grep-friendly.
var osStat = os.Stat

// CanonicalDirs returns the absolute paths of the canonical content
// directories for a project at targetDir. If cfg has a non-empty Content
// override for a given kind, that path (project-relative) wins;
// otherwise the default .hero/<kind>/ applies.
func CanonicalDirs(targetDir string, cfg config.Config) (agents, commands, skills string) {
	base := filepath.Join(targetDir, ".hero")
	agents = filepath.Join(base, "agents")
	commands = filepath.Join(base, "commands")
	skills = filepath.Join(base, "skills")

	if cfg.Content == nil {
		return
	}
	if cfg.Content.AgentsPath != "" {
		agents = filepath.Join(targetDir, cfg.Content.AgentsPath)
	}
	if cfg.Content.CommandsPath != "" {
		commands = filepath.Join(targetDir, cfg.Content.CommandsPath)
	}
	if cfg.Content.SkillsPath != "" {
		skills = filepath.Join(targetDir, cfg.Content.SkillsPath)
	}
	return
}

// pathInsideHero reports whether path is a descendant of <projectRoot>/.hero/.
// Used to decide whether installCanonical should materialize embedded
// content there (yes when inside .hero/, no when configured to live
// elsewhere — the user is asserting that path already has content).
func pathInsideHero(projectRoot, path string) bool {
	heroBase := filepath.Clean(filepath.Join(projectRoot, ".hero"))
	cleanPath := filepath.Clean(path)
	if cleanPath == heroBase {
		return true
	}
	prefix := heroBase + string(filepath.Separator)
	return strings.HasPrefix(cleanPath, prefix)
}

// installCanonical materializes the embedded source content into the
// canonical content tree. Skills are written in the Anthropic SKILL.md
// directory layout. Re-running this is idempotent when content is
// unchanged.
//
// Two conditions both required for materialization to happen:
//   - ModeProject (global installs use per-harness global dirs).
//   - `.hero/` workspace already exists (initialized via `hero init`).
//
// When configurable-content-paths overrides point a kind OUTSIDE .hero/,
// that kind is NOT materialized — the project is asserting the directory
// is already populated (hero-on-hero case: hero's source `agents/`,
// `commands/`, `skills/` ARE the canonical, so the override turns the
// would-be render step into a no-op).
func installCanonical(opts Options, result *Result) error {
	if opts.Mode != ModeProject || opts.TargetDir == "" {
		return nil
	}

	heroDir := filepath.Join(opts.TargetDir, ".hero")
	if info, err := osStat(heroDir); err != nil || !info.IsDir() {
		return nil
	}

	cfg, _ := config.Load(opts.TargetDir)
	agentsDir, commandsDir, skillsDir := CanonicalDirs(opts.TargetDir, cfg)

	if pathInsideHero(opts.TargetDir, agentsDir) {
		if err := installFlat(opts, result, "agents", agentsDir); err != nil {
			return fmt.Errorf("installing canonical agents: %w", err)
		}
	}
	if pathInsideHero(opts.TargetDir, commandsDir) {
		if err := installFlat(opts, result, "commands", commandsDir); err != nil {
			return fmt.Errorf("installing canonical commands: %w", err)
		}
	}
	if pathInsideHero(opts.TargetDir, skillsDir) {
		if err := installSkillsNested(opts, result, skillsDir); err != nil {
			return fmt.Errorf("installing canonical skills: %w", err)
		}
	}
	return nil
}

// ResolveCanonicalDirs is the public helper for per-target installers:
// loads the project's hero.json (best-effort, ignoring errors), resolves
// the canonical agent/command/skill directories from it, and validates
// that any configured external paths actually exist on disk.
func ResolveCanonicalDirs(targetDir string) (agents, commands, skills string, err error) {
	cfg, _ := config.Load(targetDir)
	agents, commands, skills = CanonicalDirs(targetDir, cfg)

	for _, p := range []struct {
		path string
		kind string
	}{{agents, "agents"}, {commands, "commands"}, {skills, "skills"}} {
		if pathInsideHero(targetDir, p.path) {
			continue
		}
		// External (configured) path — must already exist or the install
		// would symlink to a missing directory.
		if _, err := os.Stat(p.path); err != nil {
			return "", "", "", fmt.Errorf("content.%s_path points at %s but the directory doesn't exist; either create it or remove the override", p.kind, p.path)
		}
	}
	return agents, commands, skills, nil
}
