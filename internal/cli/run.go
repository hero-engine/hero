package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/runner"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var heroRunCmd = &cobra.Command{
	Use:   "run <command> <args>",
	Short: "Run agent work headlessly via the Claude/OpenAI API",
	Long: `Execute agent work without a chat UI. Hero drives the agent loop
directly, calling the LLM API and executing tools in-process.

Examples:
  hero run deliver csv-export              # deliver a spec
  hero run diagnose login-crash            # diagnose a bug
  hero run "fix the flaky auth test"       # natural language
  hero run deliver csv-export --provider openai --model gpt-4o`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRun,
}

var (
	runProvider       string
	runModel          string
	runAPIKey         string
	runMaxTurns       int
	runBudget         float64
	runDryRun         bool
	runBulk           bool
	runFilterType     string
	runFilterTag      string
	runFilterPri      string
	runFilterSlug     string
	runInlinePropose  bool
)

func init() {
	heroRunCmd.Flags().StringVar(&runProvider, "provider", "", "LLM provider (anthropic, openai, azure). Auto-detected from model if omitted.")
	heroRunCmd.Flags().StringVar(&runModel, "model", "", "model name (default: claude-sonnet-4-6-20250514)")
	heroRunCmd.Flags().StringVar(&runAPIKey, "api-key", "", "API key (prefer env var or hero login instead)")
	heroRunCmd.Flags().IntVar(&runMaxTurns, "max-turns", 100, "maximum agent loop iterations")
	heroRunCmd.Flags().Float64Var(&runBudget, "budget", 0, "cost cap in dollars (0 = unlimited)")
	heroRunCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "show execution plan without running")
	heroRunCmd.Flags().BoolVar(&runBulk, "all", false, "run on all matching specs (use with diagnose/deliver)")
	heroRunCmd.Flags().StringVar(&runFilterType, "type", "", "filter by spec type (bug, feature)")
	heroRunCmd.Flags().StringVar(&runFilterTag, "tag", "", "filter by tag")
	heroRunCmd.Flags().StringVar(&runFilterPri, "priority", "", "filter by priority (critical, high, medium, low)")
	heroRunCmd.Flags().StringVar(&runFilterSlug, "match", "", "filter slugs matching this substring")
	heroRunCmd.Flags().BoolVar(&runInlinePropose, "inline-propose", false, "agent emits HERO-PROPOSAL: NDJSON to stdout instead of writing to disk (see docs/contracts/inline-propose-v1.md)")
}

func runRun(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Check for team server — route jobs through it if connected
	if !runDryRun && !runBulk {
		tc := config.LoadTeamConnection()
		if tc != nil {
			if routed := tryTeamRoute(tc, args); routed {
				return nil
			}
		}
	}

	// Parse command and args
	command := args[0]
	runArgs := ""

	knownCommands := map[string]bool{
		"deliver": true, "diagnose": true, "design": true,
		"check": true, "review": true,
	}

	if knownCommands[command] {
		if len(args) > 1 {
			runArgs = strings.Join(args[1:], " ")
		}
	} else {
		command = ""
		runArgs = strings.Join(args, " ")
	}

	// Bulk mode: find matching specs and run sequentially
	if runBulk && command != "" {
		return runBulkMode(heroDir, projectRoot, command, &cfg)
	}

	// Register session with team server if connected (best-effort)
	if tc := config.LoadTeamConnection(); tc != nil && !runDryRun {
		sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
		registerTeamSession(tc, sessionID, command, runArgs)
		defer unregisterTeamSession(tc, sessionID)
	}

	job, err := runner.Run(runner.RunConfig{
		ProjectRoot:   projectRoot,
		HeroDir:       heroDir,
		Provider:      runProvider,
		Model:         runModel,
		APIKey:        runAPIKey,
		Command:       command,
		Args:          runArgs,
		MaxTurns:      runMaxTurns,
		Budget:        runBudget,
		DryRun:        runDryRun,
		InlinePropose: runInlinePropose,
	})
	if err != nil {
		return err
	}

	if job.Status == "failed" {
		os.Exit(1)
	}

	return nil
}

func runBulkMode(heroDir, projectRoot, command string, cfg *config.Config) error {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	// Determine target status based on command
	var targetStatus spec.Status
	switch command {
	case "diagnose":
		targetStatus = spec.StatusPlanning // bugs needing diagnosis
	case "deliver":
		targetStatus = spec.StatusPlanning // specs with fix plans ready
	default:
		targetStatus = spec.StatusPlanning
	}

	// Filter specs
	var candidates []*spec.Spec
	for _, s := range specs {
		if s.Status != targetStatus {
			continue
		}
		if runFilterType != "" && string(s.Type) != runFilterType {
			continue
		}
		if runFilterTag != "" && !hasTag(s.Tags, runFilterTag) {
			continue
		}
		if runFilterPri != "" && s.Priority != runFilterPri {
			continue
		}
		if runFilterSlug != "" && !strings.Contains(s.Slug, runFilterSlug) {
			continue
		}
		// For deliver, only include specs with a Changes section
		if command == "deliver" {
			if _, ok := s.Sections["changes"]; !ok {
				continue
			}
		}
		candidates = append(candidates, s)
	}

	if len(candidates) == 0 {
		fmt.Println("No matching specs found.")
		return nil
	}

	fmt.Printf("Bulk %s: %d specs matched\n\n", command, len(candidates))
	for i, s := range candidates {
		fmt.Printf("  %d. %s — %s\n", i+1, s.Slug, s.Title)
	}
	fmt.Println()

	if runDryRun {
		fmt.Println("Dry run — no jobs will be submitted.")
		return nil
	}

	succeeded := 0
	failed := 0
	for i, s := range candidates {
		fmt.Printf("\n[%d/%d] %s %s\n", i+1, len(candidates), command, s.Slug)
		fmt.Println(strings.Repeat("─", 60))

		job, err := runner.Run(runner.RunConfig{
			ProjectRoot:   projectRoot,
			HeroDir:       heroDir,
			Provider:      runProvider,
			Model:         runModel,
			APIKey:        runAPIKey,
			Command:       command,
			Args:          s.Slug,
			MaxTurns:      runMaxTurns,
			Budget:        runBudget,
			InlinePropose: runInlinePropose,
		})
		if err != nil {
			fmt.Printf("  FAILED: %v\n", err)
			failed++
			continue
		}
		if job.Status == "completed" {
			succeeded++
		} else {
			failed++
		}
	}

	fmt.Printf("\n\nBulk %s complete: %d succeeded, %d failed\n", command, succeeded, failed)
	return nil
}

func tryTeamRoute(tc *config.TeamConnection, args []string) bool {
	command := args[0]
	knownCommands := map[string]bool{
		"deliver": true, "diagnose": true, "design": true, "check": true, "review": true,
	}
	if !knownCommands[command] {
		return false
	}

	jobArgs := ""
	if len(args) > 1 {
		jobArgs = strings.Join(args[1:], " ")
	}

	body := map[string]interface{}{
		"command":   command,
		"args":      jobArgs,
		"provider":  runProvider,
		"model":     runModel,
		"budget":    runBudget,
		"max_turns": runMaxTurns,
	}
	data, _ := json.Marshal(body)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", tc.URL+"/api/jobs", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Team server: cannot create request: %v — running locally\n", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	if k, v := tc.AuthHeader(); k != "" {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Team server unreachable — running locally\n")
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		fmt.Fprintf(os.Stderr, "Team server returned %d — running locally\n", resp.StatusCode)
		return false
	}

	var result struct {
		ID      string `json:"id"`
		Command string `json:"command"`
		Args    string `json:"args"`
		Status  string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("Job submitted to team server: %s\n", result.ID)
	fmt.Printf("  Command: %s %s\n", result.Command, result.Args)
	fmt.Printf("  Status: %s\n", result.Status)
	fmt.Println("\nTrack with: hero jobs")
	return true
}

func registerTeamSession(tc *config.TeamConnection, sessionID, command, args string) {
	body := map[string]string{
		"id":        sessionID,
		"user_id":   tc.User,
		"command":   command,
		"spec_slug": args,
	}
	data, _ := json.Marshal(body)
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", tc.URL+"/api/sessions", bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if k, v := tc.AuthHeader(); k != "" {
		req.Header.Set(k, v)
	}
	client.Do(req) // best-effort, don't fail the run
}

func unregisterTeamSession(tc *config.TeamConnection, sessionID string) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("DELETE", tc.URL+"/api/sessions/"+sessionID, nil)
	if err != nil {
		return
	}
	if k, v := tc.AuthHeader(); k != "" {
		req.Header.Set(k, v)
	}
	client.Do(req) // best-effort
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}
