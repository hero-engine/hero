package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var connectTeamCmd = &cobra.Command{
	Use:   "team <url>",
	Short: "Connect to a Hero team server",
	Long: `Register this machine with a team server so jobs, sessions, and
events flow through shared infrastructure.

Examples:
  hero connect team https://hero.internal:7437
  hero connect team http://localhost:7437 --token mytoken`,
	Args: cobra.ExactArgs(1),
	RunE: runConnectTeam,
}

var disconnectTeamCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect from the team server",
	RunE:  runDisconnectTeam,
}

var connectTeamToken string

func init() {
	connectTeamCmd.Flags().StringVar(&connectTeamToken, "token", "", "auth token (if server uses token auth)")
	connectCmd.AddCommand(connectTeamCmd)
	connectCmd.AddCommand(disconnectTeamCmd)
}

func runConnectTeam(cmd *cobra.Command, args []string) error {
	serverURL := strings.TrimRight(args[0], "/")

	// Test the connection
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", serverURL+"/api/team/status", nil)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if connectTeamToken != "" {
		req.Header.Set("Authorization", "Bearer "+connectTeamToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach team server at %s: %w", serverURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("authentication required — use --token or configure OAuth on the server")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("team server returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse team status for display
	var status struct {
		Sessions    []interface{} `json:"sessions"`
		RunningJobs []interface{} `json:"running_jobs"`
		QueuedJobs  []interface{} `json:"queued_jobs"`
	}
	json.NewDecoder(resp.Body).Decode(&status)

	// Save connection
	tc := &config.TeamConnection{
		URL:   serverURL,
		Token: connectTeamToken,
	}
	if err := config.SaveTeamConnection(tc); err != nil {
		return fmt.Errorf("saving connection: %w", err)
	}

	fmt.Printf("Connected to team server at %s\n", serverURL)
	fmt.Printf("  Active sessions: %d\n", len(status.Sessions))
	fmt.Printf("  Running jobs: %d\n", len(status.RunningJobs))
	fmt.Printf("  Queued jobs: %d\n", len(status.QueuedJobs))
	fmt.Println("\nhero run will now route jobs through the team server.")
	return nil
}

func runDisconnectTeam(cmd *cobra.Command, args []string) error {
	if err := config.RemoveTeamConnection(); err != nil {
		return fmt.Errorf("disconnecting: %w", err)
	}
	fmt.Println("Disconnected from team server. hero run will execute locally.")
	return nil
}
