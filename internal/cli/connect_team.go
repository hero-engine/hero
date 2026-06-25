package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
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

	// If --token is provided, use direct token auth.
	if connectTeamToken != "" {
		return connectWithToken(serverURL, connectTeamToken)
	}

	// No token — check if server supports OAuth.
	oauthCfg, err := checkOAuthConfig(serverURL)
	if err != nil || oauthCfg == nil || !oauthCfg.Enabled {
		// Fall back to token-less connection attempt.
		return connectWithToken(serverURL, "")
	}

	// Start OAuth flow.
	return connectWithOAuth(serverURL, oauthCfg.Provider)
}

type oauthConfigResponse struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
}

func checkOAuthConfig(serverURL string) (*oauthConfigResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(serverURL + "/auth/oauth/config")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OAuth not available (status %d)", resp.StatusCode)
	}

	var cfg oauthConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func connectWithToken(serverURL, token string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", serverURL+"/api/team/status", nil)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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

	var status struct {
		Sessions    []interface{} `json:"sessions"`
		RunningJobs []interface{} `json:"running_jobs"`
		QueuedJobs  []interface{} `json:"queued_jobs"`
	}
	json.NewDecoder(resp.Body).Decode(&status)

	tc := &config.TeamConnection{
		URL:   serverURL,
		Token: token,
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

func connectWithOAuth(serverURL, provider string) error {
	// Start a temporary localhost HTTP server to receive the OAuth callback.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start callback listener: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

	type callbackResult struct {
		Token string
		Email string
		Name  string
		Err   error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		email := r.URL.Query().Get("email")
		name := r.URL.Query().Get("name")

		if token == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no token received"
			}
			fmt.Fprintf(w, "<html><body><h2>Authentication failed</h2><p>%s</p><p>You can close this window.</p></body></html>", errMsg)
			resultCh <- callbackResult{Err: fmt.Errorf("OAuth failed: %s", errMsg)}
			return
		}

		fmt.Fprintf(w, "<html><body><h2>Authenticated</h2><p>Connected as %s. You can close this window.</p></body></html>", email)
		resultCh <- callbackResult{Token: token, Email: email, Name: name}
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	defer srv.Shutdown(context.Background())

	// Open browser to the OAuth login URL.
	loginURL := fmt.Sprintf("%s/auth/oauth/login?provider=%s&redirect_uri=%s",
		serverURL, provider, callbackURL)

	fmt.Printf("Opening browser for %s authentication...\n", provider)
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", loginURL)
	_ = openBrowser(loginURL)

	// Wait for the callback (with timeout).
	select {
	case result := <-resultCh:
		if result.Err != nil {
			return result.Err
		}

		tc := &config.TeamConnection{
			URL:   serverURL,
			Token: result.Token,
			User:  result.Email,
		}
		if err := config.SaveTeamConnection(tc); err != nil {
			return fmt.Errorf("saving connection: %w", err)
		}

		displayName := result.Name
		if displayName == "" {
			displayName = result.Email
		}
		fmt.Printf("Connected as %s (%s)\n", displayName, provider)
		fmt.Println("hero run will now route jobs through the team server.")
		return nil

	case <-time.After(2 * time.Minute):
		return fmt.Errorf("OAuth login timed out — no callback received within 2 minutes")
	}
}


func runDisconnectTeam(cmd *cobra.Command, args []string) error {
	if err := config.RemoveTeamConnection(); err != nil {
		return fmt.Errorf("disconnecting: %w", err)
	}
	fmt.Println("Disconnected from team server. hero run will execute locally.")
	return nil
}
