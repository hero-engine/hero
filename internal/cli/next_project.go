package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/projection"
	"github.com/spf13/cobra"
)

var (
	nextProjectSession string
	nextProjectWrite   bool
)

var nextProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Render NEXT.md from the knowledge graph (preview by default)",
	Long: `Generates NEXT.md content by querying the unified knowledge graph:
recent commits, open features by priority, depends_on chains, attempts
linked to the current session.

By default prints to stdout. With --write, replaces .hero/NEXT.md
(rule: projections always win — hero writes NEXT.md, humans don't).`,
	RunE: runNextProject,
}

func init() {
	nextProjectCmd.Flags().StringVar(&nextProjectSession, "session", "",
		"session id to anchor 'Tried and failed' (default: from existing NEXT.md frontmatter)")
	nextProjectCmd.Flags().BoolVar(&nextProjectWrite, "write", false,
		"replace .hero/NEXT.md instead of printing to stdout")
	nextCmd.AddCommand(nextProjectCmd)
}

func runNextProject(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	opts := projection.NextMDOptions{
		RepoKey:                 graphRepoKey(projectRoot),
		Branch:                  gitutil.CurrentBranch(projectRoot),
		SessionID:               nextProjectSession,
		Vocab:                   activeVocab(&cfg),
		Methodology:             activeMethodology(&cfg),
		HeroDir:                 heroDir,
		ProjectRoot:             projectRoot,
		RoadmapRecencyDays:      cfg.Roadmap.AmbientRecencyDaysOrDefault(),
		RoadmapStopNaggingHours: cfg.Roadmap.StopNaggingHoursOrDefault(),
	}
	if opts.SessionID == "" {
		opts.SessionID = readSessionFromExistingNext(heroDir)
	}

	rendered, err := projection.NextMD(store, opts)
	if err != nil {
		return fmt.Errorf("rendering NEXT.md: %w", err)
	}

	if !nextProjectWrite {
		fmt.Print(rendered)
		return nil
	}

	target := filepath.Join(heroDir, "NEXT.md")
	if err := os.WriteFile(target, []byte(rendered), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	fmt.Printf("Wrote %s (%d bytes from graph)\n", target, len(rendered))
	return nil
}

// readSessionFromExistingNext returns the session id from the
// existing NEXT.md frontmatter, if any. Best-effort — returns "" on
// any failure.
func readSessionFromExistingNext(heroDir string) string {
	bytes, err := os.ReadFile(filepath.Join(heroDir, "NEXT.md"))
	if err != nil {
		return ""
	}
	s := string(bytes)
	if len(s) < 4 || s[:4] != "---\n" {
		return ""
	}
	end := indexOf(s[4:], "\n---")
	if end < 0 {
		return ""
	}
	for _, line := range splitLines(s[4 : 4+end]) {
		k, v := splitFrontmatterLine(line)
		if k == "session" {
			return v
		}
	}
	return ""
}

// Tiny string helpers — kept private to avoid exporting from cli/.
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func splitFrontmatterLine(line string) (string, string) {
	for i := 0; i < len(line); i++ {
		if line[i] == ':' {
			k := line[:i]
			v := line[i+1:]
			for len(v) > 0 && (v[0] == ' ' || v[0] == '\t') {
				v = v[1:]
			}
			return k, v
		}
	}
	return line, ""
}
