package cli

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect [type]",
	Short: "Connect hero to an external tracker or wiki service",
	Long: `Interactively configure a tracker or wiki connection and save credentials.

Supported types: github, jira, linear, gitlab, confluence

Examples:
  hero connect github       — guided setup for GitHub Issues
  hero connect jira         — guided setup for Jira (asks for base_url too)
  hero connect linear       — guided setup for Linear
  hero connect confluence   — guided setup for Confluence wiki sync
  hero connect --list       — show all saved connections
  hero connect --remove github --project owner/repo  — remove a saved connection

Tokens are saved to .hero/hero.local.json (project scope, gitignored) by default,
or to ~/.config/hero/credentials.json when --global is passed.

Non-secret config (type, project, base_url) is written to hero.json if not already set.`,
	RunE:    runConnect,
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"conn"},
}

var (
	connectList    bool
	connectRemove  string
	connectGlobal  bool
	connectProject string
)

func init() {
	connectCmd.Flags().BoolVar(&connectList, "list", false, "list saved connections")
	connectCmd.Flags().StringVar(&connectRemove, "remove", "", "remove connection for the given tracker type")
	connectCmd.Flags().BoolVar(&connectGlobal, "global", false, "save token to global credentials (~/.config/hero/credentials.json) instead of hero.local.json")
	connectCmd.Flags().StringVar(&connectProject, "project", "", "project identifier (used with --remove)")
}

func runConnect(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()

	creds, err := config.LoadCredentials()
	if err != nil {
		return fmt.Errorf("loading credentials: %w", err)
	}

	if connectList {
		return runConnectList(creds)
	}

	if connectRemove != "" {
		return runConnectRemove(creds, connectRemove, connectProject)
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: hero connect <type>  (github, jira, linear, gitlab, confluence)\n\nRun 'hero connect --list' to see saved connections.")
	}

	trackerType := strings.ToLower(args[0])
	switch trackerType {
	case "github":
		return runConnectGitHub(projectRoot, creds)
	case "jira":
		return runConnectJira(projectRoot, creds)
	case "linear":
		return runConnectLinear(projectRoot, creds)
	case "gitlab":
		return runConnectGitLab(projectRoot, creds)
	case "confluence":
		return runConnectConfluence(projectRoot, creds)
	default:
		return fmt.Errorf("unknown type %q — supported: github, jira, linear, gitlab, confluence", trackerType)
	}
}

// ---------------------------------------------------------------------------
// --list
// ---------------------------------------------------------------------------

func runConnectList(creds config.Credentials) error {
	if len(creds) == 0 {
		fmt.Println("No connections saved in", config.CredentialsPath())
		fmt.Println("Run 'hero connect <type>' to set one up.")
		return nil
	}
	fmt.Printf("Saved connections (%s):\n\n", config.CredentialsPath())
	for key, entry := range creds {
		parts := strings.SplitN(key, ":", 2)
		trackerType := key
		project := ""
		if len(parts) == 2 {
			trackerType = parts[0]
			project = parts[1]
		}
		fmt.Printf("  %-12s  %-30s  token: %s\n", trackerType, project, maskToken(entry.Token))
		if entry.BaseURL != "" {
			fmt.Printf("  %12s  base_url: %s\n", "", entry.BaseURL)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// --remove
// ---------------------------------------------------------------------------

func runConnectRemove(creds config.Credentials, trackerType, project string) error {
	if project == "" {
		// Try to find a single match for this type
		var matches []string
		prefix := trackerType + ":"
		for k := range creds {
			if strings.HasPrefix(k, prefix) {
				matches = append(matches, k)
			}
		}
		switch len(matches) {
		case 0:
			return fmt.Errorf("no saved connection for %q", trackerType)
		case 1:
			project = strings.TrimPrefix(matches[0], prefix)
		default:
			return fmt.Errorf("multiple %q connections saved — specify --project to disambiguate: %s",
				trackerType, strings.Join(matches, ", "))
		}
	}

	key := config.CredentialKey(trackerType, project)
	if _, ok := creds[key]; !ok {
		return fmt.Errorf("no saved connection for %s:%s", trackerType, project)
	}

	config.RemoveCredential(creds, trackerType, project)
	if err := config.SaveCredentials(creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}
	fmt.Printf("Removed connection: %s:%s\n", trackerType, project)
	return nil
}

// ---------------------------------------------------------------------------
// github
// ---------------------------------------------------------------------------

func runConnectGitHub(projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to GitHub Issues...")
	fmt.Println()

	project := prompt("Repository (owner/repo): ")
	if project == "" {
		return fmt.Errorf("repository is required")
	}

	token := promptSecret("Personal access token (needs 'repo' scope): ")
	if token == "" {
		return fmt.Errorf("token is required")
	}

	fmt.Println()
	fmt.Print("Verifying connection... ")

	if err := verifyGitHubToken(project, token); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("could not verify GitHub connection: %w", err)
	}
	fmt.Println("OK")

	entry := config.CredentialEntry{Token: token}
	return saveConnection(projectRoot, creds, "github", project, entry, "", "")
}

// ---------------------------------------------------------------------------
// jira
// ---------------------------------------------------------------------------

func runConnectJira(projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to Jira...")
	fmt.Println()

	baseURL := prompt("Jira base URL (e.g. https://mycompany.atlassian.net): ")
	if baseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	project := prompt("Project key (e.g. PROJ): ")
	if project == "" {
		return fmt.Errorf("project key is required")
	}

	userEmail := prompt("User email (for Jira Cloud basic auth): ")
	if userEmail == "" {
		return fmt.Errorf("user email is required for Jira Cloud API authentication")
	}

	token := promptSecret("API token (from https://id.atlassian.com/manage-profile/security/api-tokens): ")
	if token == "" {
		return fmt.Errorf("token is required")
	}

	fmt.Println()
	fmt.Print("Verifying connection... ")

	if err := verifyJiraToken(baseURL, project, userEmail, token); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("could not verify Jira connection: %w", err)
	}
	fmt.Println("OK")

	entry := config.CredentialEntry{Token: token, BaseURL: baseURL, UserEmail: userEmail}
	return saveConnection(projectRoot, creds, "jira", project, entry, baseURL, userEmail)
}

// ---------------------------------------------------------------------------
// linear
// ---------------------------------------------------------------------------

func runConnectLinear(projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to Linear...")
	fmt.Println()

	project := prompt("Team key (e.g. ENG): ")
	if project == "" {
		return fmt.Errorf("team key is required")
	}

	token := promptSecret("API key (from https://linear.app/settings/api): ")
	if token == "" {
		return fmt.Errorf("token is required")
	}

	fmt.Println()
	fmt.Print("Verifying connection... ")

	if err := verifyLinearToken(project, token); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("could not verify Linear connection: %w", err)
	}
	fmt.Println("OK")

	entry := config.CredentialEntry{Token: token}
	return saveConnection(projectRoot, creds, "linear", project, entry, "", "")
}

// ---------------------------------------------------------------------------
// gitlab
// ---------------------------------------------------------------------------

func runConnectGitLab(projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to GitLab...")
	fmt.Println()

	baseURL := prompt("GitLab base URL [https://gitlab.com]: ")
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	project := prompt("Project (namespace/project or numeric ID): ")
	if project == "" {
		return fmt.Errorf("project is required")
	}

	token := promptSecret("Personal/Project access token (needs 'api' scope): ")
	if token == "" {
		return fmt.Errorf("token is required")
	}

	fmt.Println()
	fmt.Print("Verifying connection... ")

	if err := verifyGitLabToken(baseURL, project, token); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("could not verify GitLab connection: %w", err)
	}
	fmt.Println("OK")

	entry := config.CredentialEntry{Token: token, BaseURL: baseURL}
	return saveConnection(projectRoot, creds, "gitlab", project, entry, baseURL, "")
}

// ---------------------------------------------------------------------------
// confluence
// ---------------------------------------------------------------------------

func runConnectConfluence(projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to Confluence...")
	fmt.Println()

	baseURL := prompt("Confluence base URL (e.g. https://mycompany.atlassian.net/wiki): ")
	if baseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	spaceKey := prompt("Space key (e.g. ENG): ")
	if spaceKey == "" {
		return fmt.Errorf("space key is required")
	}

	userEmail := prompt("User email (for Confluence Cloud basic auth): ")

	token := promptSecret("API token: ")
	if token == "" {
		return fmt.Errorf("token is required")
	}

	fmt.Println()
	fmt.Print("Verifying connection... ")

	if err := verifyConfluenceToken(baseURL, spaceKey, userEmail, token); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("could not verify Confluence connection: %w", err)
	}
	fmt.Println("OK")

	entry := config.CredentialEntry{Token: token, BaseURL: baseURL, UserEmail: userEmail}
	return saveConnection(projectRoot, creds, "confluence", spaceKey, entry, baseURL, userEmail)
}

// ---------------------------------------------------------------------------
// save helpers
// ---------------------------------------------------------------------------

// saveConnection persists the credential and updates hero.json / hero.local.json.
func saveConnection(projectRoot string, creds config.Credentials, trackerType, project string, entry config.CredentialEntry, baseURL, userEmail string) error {
	if connectGlobal {
		// Save to global credentials file
		config.SetCredential(creds, trackerType, project, entry)
		if err := config.SaveCredentials(creds); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}
		fmt.Printf("Token saved to %s\n", config.CredentialsPath())
	} else {
		// Save to project-local hero.local.json
		cfg, err := config.Load(projectRoot)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		folder := cfg.Folder
		if folder == "" {
			folder = config.DefaultFolder
		}

		var local config.Config
		if trackerType == "confluence" {
			local.Confluence = &config.ConfluenceConfig{
				Token: entry.Token,
			}
			if baseURL != "" {
				local.Confluence.BaseURL = baseURL
			}
			if userEmail != "" {
				local.Confluence.UserEmail = userEmail
			}
		} else {
			local.Tracker = &config.TrackerConfig{
				Token: entry.Token,
			}
			if userEmail != "" {
				local.Tracker.UserEmail = userEmail
			}
		}

		if err := config.SaveLocal(projectRoot, folder, local); err != nil {
			return fmt.Errorf("saving hero.local.json: %w", err)
		}
		fmt.Printf("Token saved to .hero/hero.local.json\n")
	}

	// Update hero.json with non-secret config if not already set
	if err := updateHeroJSON(projectRoot, trackerType, project, baseURL, userEmail); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not update hero.json: %v\n", err)
	}

	fmt.Printf("Connected. Run 'hero connect --list' to see all connections.\n")
	return nil
}

// updateHeroJSON writes non-secret tracker/confluence config to hero.json if not already set.
func updateHeroJSON(projectRoot, trackerType, project, baseURL, userEmail string) error {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		// No hero.json yet — not an error, nothing to update
		return nil
	}

	changed := false

	if trackerType == "confluence" {
		if cfg.Confluence == nil {
			cfg.Confluence = &config.ConfluenceConfig{}
		}
		if cfg.Confluence.SpaceKey == "" {
			cfg.Confluence.SpaceKey = project
			changed = true
		}
		if baseURL != "" && cfg.Confluence.BaseURL == "" {
			cfg.Confluence.BaseURL = baseURL
			changed = true
		}
		if userEmail != "" && cfg.Confluence.UserEmail == "" {
			cfg.Confluence.UserEmail = userEmail
			changed = true
		}
	} else {
		if cfg.Tracker == nil {
			cfg.Tracker = &config.TrackerConfig{}
		}
		if cfg.Tracker.Type == "" || cfg.Tracker.Type == "none" {
			cfg.Tracker.Type = trackerType
			changed = true
		}
		if cfg.Tracker.Project == "" {
			cfg.Tracker.Project = project
			changed = true
		}
		if baseURL != "" && cfg.Tracker.BaseURL == "" {
			cfg.Tracker.BaseURL = baseURL
			changed = true
		}
		if userEmail != "" && cfg.Tracker.UserEmail == "" {
			cfg.Tracker.UserEmail = userEmail
			changed = true
		}
	}

	if !changed {
		return nil
	}
	return cfg.Save(projectRoot)
}

// ---------------------------------------------------------------------------
// verification — real HTTP calls to validate the token
// ---------------------------------------------------------------------------

// verifyGitHubToken checks that the token can read the given repo.
func verifyGitHubToken(project, token string) error {
	parts := strings.SplitN(project, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("project must be in owner/repo format")
	}
	_, err := httpGET("https://api.github.com/repos/"+project, map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	})
	return err
}

// verifyJiraToken checks that the token can reach the Jira project.
// Jira Cloud requires basic auth (email:token). Jira Server/DC uses Bearer (PAT).
// We try basic auth first (since connect always collects email for Cloud).
func verifyJiraToken(baseURL, project, userEmail, token string) error {
	_, err := httpGETBasicAuthEmail(baseURL+"/rest/api/3/project/"+project, userEmail, token)
	return err
}

// verifyLinearToken checks that the token can query the Linear API.
func verifyLinearToken(_, token string) error {
	_, err := httpPOST("https://api.linear.app/graphql", `{"query":"{ viewer { id name } }"}`, map[string]string{
		"Authorization": token,
		"Content-Type":  "application/json",
	})
	return err
}

// verifyGitLabToken checks that the token can read the given project via
// the REST v4 API. Uses the PRIVATE-TOKEN header (PAT/project token).
func verifyGitLabToken(baseURL, project, token string) error {
	_, err := httpGET(baseURL+"/api/v4/projects/"+url.PathEscape(project), map[string]string{
		"PRIVATE-TOKEN": token,
		"Accept":        "application/json",
	})
	return err
}

// verifyConfluenceToken checks that the token can read the Confluence space.
func verifyConfluenceToken(baseURL, spaceKey, userEmail, token string) error {
	_, err := httpGETBasicAuthEmail(baseURL+"/rest/api/space/"+spaceKey, userEmail, token)
	return err
}

// ---------------------------------------------------------------------------
// minimal HTTP helpers (stdlib only, no external deps)
// ---------------------------------------------------------------------------

func httpGET(url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func httpGETBasicAuthEmail(url, userEmail, token string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(userEmail, token)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func httpPOST(url, jsonBody string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// ---------------------------------------------------------------------------
// I/O helpers
// ---------------------------------------------------------------------------

func prompt(label string) string {
	fmt.Print(label)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// promptSecret reads a token from the terminal. On Unix systems it reads from
// /dev/tty so the input comes from the terminal even when stdin is piped.
// We avoid CGo by not suppressing echo — the user sees their token as they type,
// but the token is never written to any log file.
func promptSecret(label string) string {
	fmt.Print(label)
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		defer tty.Close()
		scanner := bufio.NewScanner(tty)
		if scanner.Scan() {
			fmt.Fprintln(tty)
			return strings.TrimSpace(scanner.Text())
		}
		return ""
	}
	// Fallback: plain stdin
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// maskToken returns a redacted version of a token for display.
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
