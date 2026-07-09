package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Team server status and management",
	Long: `View team server connection status, active sessions, and usage.

Examples:
  hero team status     # show connection + team activity
  hero team usage      # detailed usage breakdown`,
	RunE: runTeamStatus,
}

var teamStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show team server connection and activity",
	RunE:  runTeamStatus,
}

var teamUsageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show per-user usage breakdown",
	RunE:  runTeamUsage,
}

var teamSessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Show active sessions on the team server",
	RunE:  runTeamSessions,
}

func init() {
	teamCmd.AddCommand(teamStatusCmd)
	teamCmd.AddCommand(teamUsageCmd)
	teamCmd.AddCommand(teamSessionsCmd)
}

func runTeamStatus(cmd *cobra.Command, args []string) error {
	tc := config.LoadTeamConnection()
	if tc == nil {
		fmt.Println("Not connected to a team server.")
		fmt.Println("Run: hero connect team <url>")
		return nil
	}

	fmt.Printf("Connected to: %s\n", tc.URL)
	if tc.User != "" {
		fmt.Printf("User: %s\n", tc.User)
	}
	fmt.Printf("Connected since: %s\n", tc.ConnectedAt)
	fmt.Println()

	// Fetch live status
	data, err := teamGet(tc, "/api/team/status")
	if err != nil {
		fmt.Printf("Server unreachable: %v\n", err)
		return nil
	}

	var status struct {
		Sessions        []map[string]string `json:"sessions"`
		RunningJobs     []map[string]interface{} `json:"running_jobs"`
		QueuedJobs      []map[string]interface{} `json:"queued_jobs"`
		AwaitingApproval []map[string]interface{} `json:"awaiting_approval"`
	}
	json.Unmarshal(data, &status)

	fmt.Printf("Active sessions: %d\n", len(status.Sessions))
	for _, s := range status.Sessions {
		fmt.Printf("  %s — %s (%s %s)\n", s["user_id"], s["agent"], s["command"], s["spec_slug"])
	}

	fmt.Printf("\nRunning jobs: %d\n", len(status.RunningJobs))
	for _, j := range status.RunningJobs {
		fmt.Printf("  %s %s (by %s)\n", j["command"], j["args"], j["submitted_by"])
	}

	fmt.Printf("Queued jobs: %d\n", len(status.QueuedJobs))
	if len(status.AwaitingApproval) > 0 {
		fmt.Printf("Awaiting approval: %d\n", len(status.AwaitingApproval))
		for _, j := range status.AwaitingApproval {
			fmt.Printf("  %s %s — hero approve %s\n", j["command"], j["args"], j["id"])
		}
	}

	return nil
}

func runTeamUsage(cmd *cobra.Command, args []string) error {
	tc := config.LoadTeamConnection()
	if tc == nil {
		fmt.Println("Not connected to a team server.")
		return nil
	}

	data, err := teamGet(tc, "/api/team/usage")
	if err != nil {
		return fmt.Errorf("fetching usage: %w", err)
	}

	var usage []map[string]interface{}
	json.Unmarshal(data, &usage)

	if len(usage) == 0 {
		fmt.Println("No usage data yet.")
		return nil
	}

	fmt.Println("Usage (last 7 days):")
	fmt.Println()
	fmt.Printf("  %-20s  %-6s  %-12s  %-12s  %s\n", "User", "Jobs", "Input tokens", "Output tokens", "Cost")
	fmt.Printf("  %-20s  %-6s  %-12s  %-12s  %s\n", "────", "────", "────────────", "────────────", "────")
	for _, u := range usage {
		fmt.Printf("  %-20s  %-6.0f  %-12.0f  %-12.0f  $%.2f\n",
			u["user_id"], u["jobs"], u["input_tokens"], u["output_tokens"], u["total_cost"])
	}

	return nil
}

func runTeamSessions(cmd *cobra.Command, args []string) error {
	tc := config.LoadTeamConnection()
	if tc == nil {
		fmt.Println("Not connected to a team server.")
		fmt.Println("Run: hero connect team <url>")
		return nil
	}

	data, err := teamGet(tc, "/api/sessions")
	if err != nil {
		return fmt.Errorf("fetching sessions: %w", err)
	}

	var sessions []map[string]string
	json.Unmarshal(data, &sessions)

	if len(sessions) == 0 {
		fmt.Println("No active sessions.")
		return nil
	}

	fmt.Printf("%-20s  %-15s  %-12s  %-20s  %s\n", "SESSION", "USER", "COMMAND", "SPEC", "LAST SEEN")
	fmt.Printf("%-20s  %-15s  %-12s  %-20s  %s\n", "───────", "────", "───────", "────", "─────────")
	for _, s := range sessions {
		fmt.Printf("%-20s  %-15s  %-12s  %-20s  %s\n",
			truncate(s["id"], 20), truncate(s["user_id"], 15),
			truncate(s["command"], 12), truncate(s["spec_slug"], 20), s["last_seen"])
	}
	return nil
}

func teamGet(tc *config.TeamConnection, path string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", tc.URL+path, nil)
	if err != nil {
		return nil, err
	}
	if k, v := tc.AuthHeader(); k != "" {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
