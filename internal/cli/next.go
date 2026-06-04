package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

const (
	nextFileName = "NEXT.md" // legacy/solo shared file
	nextDirName  = "next"    // per-user directory
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show your handoff briefing — personal in team mode, shared in solo mode",
	Long: `NEXT.md is a three-section briefing (Just finished / Next /
Context to carry forward) the agent overwrites at the end of meaningful
turns so a fresh session can pick up where the last one left off.

Mode is controlled by "next.mode" in hero.json:

  solo (default): single shared .hero/NEXT.md
  team:           per-user files in .hero/next/<user>.md

Subcommands:

  hero next           — show your briefing (personal or shared, per mode)
  hero next team      — show all team members' recent handoffs
  hero next shared    — show the project-level NEXT.md
  hero next path      — print the file path the agent should write to
  hero next migrate   — move .hero/NEXT.md into .hero/next/<user>.md`,
	RunE: runNextShow,
}

var nextShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print your handoff briefing",
	RunE:  runNextShow,
}

var nextTeamCmd = &cobra.Command{
	Use:   "team",
	Short: "Show all team members' recent handoff briefings",
	RunE:  runNextTeam,
}

var nextSharedCmd = &cobra.Command{
	Use:   "shared",
	Short: "Show the project-level NEXT.md",
	RunE:  runNextShared,
}

var nextPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the file path the agent should write to",
	RunE:  runNextPath,
}

var nextMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Move .hero/NEXT.md into .hero/next/<user>.md for team mode",
	RunE:  runNextMigrate,
}

func init() {
	nextCmd.AddCommand(nextShowCmd)
	nextCmd.AddCommand(nextTeamCmd)
	nextCmd.AddCommand(nextSharedCmd)
	nextCmd.AddCommand(nextPathCmd)
	nextCmd.AddCommand(nextMigrateCmd)
	nextCmd.AddCommand(nextCheckpointCmd)
	nextCmd.AddCommand(nextCompactHandoffCmd)
	nextCmd.AddCommand(nextSuggestCmd)
	nextCmd.AddCommand(nextAskCmd)
	nextCmd.AddCommand(nextGoalCmd)
	nextCmd.AddCommand(nextReflectionCmd)
	nextCmd.AddCommand(nextIngestCmd)
	nextCmd.AddCommand(nextInstallHooksCmd)
	nextCmd.AddCommand(nextMigrateProjectionCmd)
}

// nextUserSlug returns the current user's slug for personal next files.
// Priority: hero.local.json tracking.defaultAgent > hero.json tracking.defaultAgent > git user.name.
func nextUserSlug(cfg config.Config) string {
	if cfg.Tracking != nil && cfg.Tracking.DefaultAgent != "" {
		agent := cfg.Tracking.DefaultAgent
		// Strip "human/" prefix if present
		if strings.HasPrefix(agent, "human/") {
			agent = strings.TrimPrefix(agent, "human/")
		}
		return agent
	}
	return gitUserName()
}

// resolveNextPath returns the path the agent should read from and write to,
// based on the configured mode.
func resolveNextPath(heroDir string, cfg config.Config) string {
	if cfg.NextMode() == "team" {
		user := nextUserSlug(cfg)
		return filepath.Join(heroDir, nextDirName, user+".md")
	}
	return filepath.Join(heroDir, nextFileName)
}

func runNextShow(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	path := resolveNextPath(heroDir, cfg)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// In team mode, also check the legacy shared file as a hint
			if cfg.NextMode() == "team" {
				shared := filepath.Join(heroDir, nextFileName)
				if _, serr := os.Stat(shared); serr == nil {
					fmt.Printf("No personal handoff yet, but a shared .hero/NEXT.md exists.\n")
					fmt.Printf("Run 'hero next migrate' to move it to your personal file, or 'hero next shared' to view it.\n")
					return nil
				}
			}
			fmt.Println("No handoff briefing yet. The agent writes it at the end of meaningful turns — see skills/next-md.md.")
			return nil
		}
		return err
	}
	defer f.Close()

	if cfg.NextMode() == "team" {
		user := nextUserSlug(cfg)
		fmt.Printf("# %s's handoff  (%s)\n\n", user, filepath.Base(path))
	}
	if _, err = io.Copy(os.Stdout, f); err != nil {
		return err
	}

	// Append per-machine state (machine block + hand-written scratch)
	// from the gitignored local file so a single `hero next` shows
	// both project context and your live machine snapshot.
	localPath := resolveLocalStatePath(heroDir, cfg)
	if data, lerr := os.ReadFile(localPath); lerr == nil && len(data) > 0 {
		fmt.Print("\n")
		os.Stdout.Write(data)
	}
	return nil
}

func runNextShared(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	path := filepath.Join(cfg.HeroDir(projectRoot), nextFileName)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No shared NEXT.md yet.")
			return nil
		}
		return err
	}
	defer f.Close()
	_, err = io.Copy(os.Stdout, f)
	return err
}

func runNextPath(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	path := resolveNextPath(heroDir, cfg)
	fmt.Println(path)
	return nil
}

func runNextTeam(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	nextDir := filepath.Join(heroDir, nextDirName)

	entries, err := os.ReadDir(nextDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No personal handoff files yet. Set next.mode to \"team\" in hero.json and run 'hero next migrate'.")
			return nil
		}
		return err
	}

	type userNext struct {
		user    string
		path    string
		modTime time.Time
	}

	var users []userNext
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		user := strings.TrimSuffix(e.Name(), ".md")
		users = append(users, userNext{
			user:    user,
			path:    filepath.Join(nextDir, e.Name()),
			modTime: info.ModTime(),
		})
	}

	if len(users) == 0 {
		fmt.Println("No personal handoff files found in .hero/next/.")
		return nil
	}

	// Sort by most recently modified
	sort.Slice(users, func(i, j int) bool {
		return users[i].modTime.After(users[j].modTime)
	})

	currentUser := nextUserSlug(cfg)
	for i, u := range users {
		if i > 0 {
			fmt.Print("\n---\n\n")
		}
		marker := ""
		if u.user == currentUser {
			marker = "  (you)"
		}
		fmt.Printf("# %s%s  — updated %s\n\n", u.user, marker, u.modTime.Format("Jan 2 15:04"))

		content, err := os.ReadFile(u.path)
		if err != nil {
			fmt.Printf("  (could not read: %v)\n", err)
			continue
		}
		// Strip frontmatter for the team view — just show the sections
		body := stripFrontmatter(string(content))
		fmt.Print(body)
		if !strings.HasSuffix(body, "\n") {
			fmt.Println()
		}
	}

	return nil
}

func runNextMigrate(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	sharedPath := filepath.Join(heroDir, nextFileName)
	user := nextUserSlug(cfg)
	personalDir := filepath.Join(heroDir, nextDirName)
	personalPath := filepath.Join(personalDir, user+".md")

	// Check shared file exists
	if _, err := os.Stat(sharedPath); os.IsNotExist(err) {
		fmt.Println("No .hero/NEXT.md to migrate.")
		return nil
	}

	// Check personal file doesn't already exist
	if _, err := os.Stat(personalPath); err == nil {
		return fmt.Errorf("personal file already exists at %s — won't overwrite", personalPath)
	}

	// Create next directory
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", personalDir, err)
	}

	// Read shared file
	content, err := os.ReadFile(sharedPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", sharedPath, err)
	}

	// Write personal file
	if err := os.WriteFile(personalPath, content, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", personalPath, err)
	}

	fmt.Printf("Migrated .hero/NEXT.md → .hero/next/%s.md\n", user)
	fmt.Printf("You can delete .hero/NEXT.md when ready (or keep it as shared project state).\n")
	return nil
}

// stripFrontmatter removes YAML frontmatter (--- ... ---) from markdown content.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	// Find closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return content
	}
	// Skip past the closing --- and any trailing newline
	after := rest[idx+4:]
	if strings.HasPrefix(after, "\n") {
		after = after[1:]
	}
	return after
}
