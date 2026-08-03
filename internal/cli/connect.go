package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/cli/prompt"
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
	connectList          bool
	connectRemove        string
	connectGlobal        bool
	connectProject       string
	connectIntegrationID string
	connectRole          string
	connectBaseURL       string
	connectUserEmail     string
	connectTokenStdin    bool
	connectLocalOnly     bool
	connectJSON          bool
	connectNoVerify      bool
)

func init() {
	connectCmd.Flags().BoolVar(&connectList, "list", false, "list saved connections")
	connectCmd.Flags().StringVar(&connectRemove, "remove", "", "remove connection for the given tracker type")
	connectCmd.Flags().BoolVar(&connectGlobal, "global", false, "save token to global credentials (~/.config/hero/credentials.json) instead of hero.local.json")
	connectCmd.Flags().StringVar(&connectProject, "project", "", "project identifier (used with --remove)")
	addIntegrationConnectFlags(connectCmd)
}

func addIntegrationConnectFlags(c *cobra.Command) {
	c.Flags().StringVar(&connectIntegrationID, "integration-id", "", "stable integration ID")
	c.Flags().StringVar(&connectRole, "role", "delivery", "selection role (delivery, docs, roadmap, code-host)")
	c.Flags().StringVar(&connectBaseURL, "base-url", "", "provider base URL")
	c.Flags().StringVar(&connectUserEmail, "user-email", "", "user email (Jira/Confluence Cloud only)")
	c.Flags().BoolVar(&connectTokenStdin, "token-stdin", false, "read token from protected standard input (never argv)")
	c.Flags().BoolVar(&connectLocalOnly, "local-only", false, "write the complete integration and selectors only to hero.local.json")
	c.Flags().BoolVar(&connectJSON, "json", false, "emit machine-readable redacted status")
	c.Flags().BoolVar(&connectNoVerify, "no-verify", false, "save without a provider network verification")
}

func runConnect(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()

	creds, err := config.LoadCredentials()
	if err != nil {
		return fmt.Errorf("loading credentials: %w", err)
	}

	if connectList {
		return runConnectList(cmd.OutOrStdout(), projectRoot, creds)
	}

	if connectRemove != "" {
		return runConnectRemove(creds, connectRemove, connectProject)
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: hero connect <type>  (github, jira, linear, gitlab, confluence)\n\nRun 'hero connect --list' to see saved connections.")
	}

	trackerType := strings.ToLower(args[0])
	if connectJSON || connectIntegrationID != "" || connectTokenStdin || connectProject != "" || connectBaseURL != "" || connectLocalOnly {
		return runConnectNonInteractive(cmd, projectRoot, creds, trackerType)
	}
	if !prompt.IsInputTTY(cmd.InOrStdin()) {
		return fmt.Errorf("interactive connect requires an attached terminal; supply --integration-id, --project, and --token-stdin for automation")
	}
	switch trackerType {
	case "github":
		return runConnectGitHub(cmd, projectRoot, creds)
	case "jira":
		return runConnectJira(cmd, projectRoot, creds)
	case "linear":
		return runConnectLinear(cmd, projectRoot, creds)
	case "gitlab":
		return runConnectGitLab(cmd, projectRoot, creds)
	case "confluence":
		return runConnectConfluence(cmd, projectRoot, creds)
	default:
		return fmt.Errorf("unknown type %q — supported: github, jira, linear, gitlab, confluence", trackerType)
	}
}

// ---------------------------------------------------------------------------
// --list
// ---------------------------------------------------------------------------

func runConnectList(w io.Writer, root string, creds config.Credentials) error {
	type row struct {
		ID               string            `json:"id"`
		Provider         string            `json:"provider"`
		Capabilities     []string          `json:"capabilities"`
		Default          bool              `json:"default"`
		Roles            []string          `json:"roles"`
		CredentialSource string            `json:"credential_source"`
		Ready            bool              `json:"ready"`
		Verified         string            `json:"verification"`
		Sources          map[string]string `json:"sources"`
	}
	rows := []row{}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if cfg.Integrations != nil {
		for _, id := range config.SortedConnectionIDs(cfg.Integrations) {
			x := cfg.Integrations.Connections[id]
			rr := row{ID: id, Provider: x.Provider, Default: cfg.Integrations.Default == id, Verified: "not-checked", Sources: map[string]string{}}
			for _, capability := range x.EffectiveCapabilities() {
				rr.Capabilities = append(rr.Capabilities, string(capability))
			}
			sort.Strings(rr.Capabilities)
			prefix := "$.integrations.connections." + id
			for path, src := range cfg.IntegrationProvenance {
				if strings.HasPrefix(path, prefix) || path == "$.integrations.default" {
					layer := "project"
					if strings.HasSuffix(src.File, config.LocalConfigFileName) {
						layer = "local"
					}
					rr.Sources[path] = layer
				}
			}
			for role, v := range cfg.Integrations.Roles {
				if v == id {
					rr.Roles = append(rr.Roles, role)
				}
			}
			sort.Strings(rr.Roles)
			tokenPath := prefix + ".auth.token"
			_, tokenFromWorkspace := cfg.IntegrationProvenance[tokenPath]
			if x.Auth != nil && x.Auth.Token != "" && tokenFromWorkspace {
				rr.Ready = true
				rr.CredentialSource = "local"
			} else if _, ok := creds[config.IntegrationCredentialKey(id)]; ok {
				rr.Ready = true
				rr.CredentialSource = "global"
			} else if x.Auth != nil && x.Auth.TokenEnv != "" {
				rr.Ready = os.Getenv(x.Auth.TokenEnv) != ""
				rr.CredentialSource = "environment:" + x.Auth.TokenEnv
			} else if x.Auth != nil && x.Auth.Token != "" {
				rr.Ready = true
				rr.CredentialSource = "global-legacy"
			} else {
				rr.CredentialSource = "missing"
			}
			rows = append(rows, rr)
		}
	}
	if connectJSON {
		b, _ := json.MarshalIndent(map[string]any{"integrations": rows}, "", "  ")
		fmt.Fprintln(w, string(b))
		return nil
	}
	if len(rows) == 0 {
		fmt.Fprintln(w, "No project integrations configured. Run 'hero connect <provider> --integration-id <id> --project <project> --token-stdin'.")
		return nil
	}
	for _, r := range rows {
		fmt.Fprintf(w, "%-24s provider=%-10s capabilities=%s ready=%t source=%s", r.ID, r.Provider, strings.Join(r.Capabilities, ","), r.Ready, r.CredentialSource)
		if r.Default {
			fmt.Fprint(w, " default")
		}
		if len(r.Roles) > 0 {
			fmt.Fprintf(w, " roles=%s", strings.Join(r.Roles, ","))
		}
		fmt.Fprintln(w)
	}
	return nil
}

func runConnectNonInteractive(cmd *cobra.Command, root string, creds config.Credentials, provider string) error {
	if !providersCLI[provider] {
		return fmt.Errorf("unknown provider %q", provider)
	}
	id := connectIntegrationID
	if id == "" {
		id = canonicalIntegrationID(provider, connectProject)
	}
	if connectProject == "" {
		return fmt.Errorf("--project is required for non-interactive connect")
	}
	role := connectRole
	if provider == "confluence" && role == "delivery" {
		role = "docs"
	}
	requiredCapability, err := config.ValidateProviderRole(provider, role)
	if err != nil {
		return err
	}
	token := ""
	if connectTokenStdin {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("reading protected token stdin: %w", err)
		}
		token = strings.TrimSpace(string(b))
		if token == "" {
			return fmt.Errorf("protected token input was empty")
		}
	}
	if token == "" && !connectGlobal {
		return fmt.Errorf("no credential supplied; use --token-stdin, --global with an existing credential, or configure auth.token_env")
	}
	if token == "" && connectGlobal {
		if _, ok := creds[config.IntegrationCredentialKey(id)]; !ok {
			return fmt.Errorf("no global credential exists for integration %q; pipe one with --token-stdin", id)
		}
	}
	if token != "" && !connectNoVerify {
		var verr error
		switch provider {
		case "github":
			verr = verifyGitHubToken(connectProject, token)
		case "jira":
			verr = verifyJiraToken(connectBaseURL, connectProject, connectUserEmail, token)
		case "linear":
			verr = verifyLinearToken(connectProject, token)
		case "gitlab":
			verr = verifyGitLabToken(connectBaseURL, connectProject, token)
		case "confluence":
			verr = verifyConfluenceToken(connectBaseURL, connectProject, connectUserEmail, token)
		}
		if verr != nil {
			return fmt.Errorf("could not verify %s integration %q: %w", provider, id, verr)
		}
	}
	settings := map[string]any{"project": connectProject}
	if provider == "confluence" {
		delete(settings, "project")
		settings["space_key"] = connectProject
	}
	if connectBaseURL != "" {
		settings["base_url"] = connectBaseURL
	}
	if connectUserEmail != "" {
		settings["user_email"] = connectUserEmail
	}
	if err := config.ValidateConnectionSettings(id, provider, settings); err != nil {
		return fmt.Errorf("cannot connect %s: %s", provider, settingErrorToFlag(err, provider))
	}
	connection := map[string]any{"provider": provider, "settings": settings}
	if existingCfg, loadErr := config.Load(root); loadErr != nil {
		return loadErr
	} else if existingCfg.Integrations != nil {
		if existing, ok := existingCfg.Integrations.Connections[id]; ok {
			capabilities := existing.EffectiveCapabilities()
			present := false
			for _, capability := range capabilities {
				present = present || capability == requiredCapability
			}
			if !present {
				capabilities = append(capabilities, requiredCapability)
			}
			connection["capabilities"] = capabilities
		}
	}
	if _, explicit := connection["capabilities"]; !explicit && requiredCapability == config.CapabilityCodeHost {
		connection["capabilities"] = []config.IntegrationCapability{config.CapabilityCodeHost}
	}
	patch := map[string]any{
		"roles":       map[string]any{role: id},
		"connections": map[string]any{id: connection},
	}
	if role != "code-host" {
		patch["default"] = id
	}
	if connectLocalOnly {
		connection["auth"] = map[string]any{"token": token}
		p, _ := json.Marshal(patch)
		if err := config.PatchLocalIntegrations(root, config.DefaultFolder, p); err != nil {
			return err
		}
	} else {
		p, _ := json.Marshal(patch)
		if err := config.PatchCommittedIntegrations(root, config.DefaultFolder, p); err != nil {
			return err
		}
		if token != "" {
			if connectGlobal {
				config.SetIntegrationCredential(creds, id, config.CredentialEntry{Token: token, BaseURL: connectBaseURL, UserEmail: connectUserEmail})
				if err := config.SaveCredentials(creds); err != nil {
					return err
				}
			} else {
				p, _ := json.Marshal(map[string]any{"connections": map[string]any{id: map[string]any{"auth": map[string]any{"token": token}}}})
				if err := config.PatchLocalIntegrations(root, config.DefaultFolder, p); err != nil {
					return err
				}
			}
		}
	}
	verification := "verified"
	if connectNoVerify {
		verification = "not-checked"
	}
	result := map[string]any{
		"id": id, "provider": provider, "role": role, "capability": requiredCapability,
		"ready": token != "" || connectGlobal, "verification": verification,
	}
	if connectJSON {
		b, _ := json.Marshal(result)
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Connected integration %s (%s). Inspect with 'hero connect --list'.\n", id, provider)
	}
	return nil
}

var providersCLI = map[string]bool{"github": true, "jira": true, "linear": true, "gitlab": true, "confluence": true}

// settingFlagNames maps a provider-schema settings key to the connect flag the
// user actually chose, so a schema rejection can be reported in flag vocabulary.
var settingFlagNames = map[string]string{
	"user_email": "--user-email",
	"base_url":   "--base-url",
	"project":    "--project",
}

// settingErrorToFlag translates a settings-schema error into flag vocabulary,
// since connect is where the offending flag was selected. Falls back to the raw
// error when no flag maps to the failing key.
func settingErrorToFlag(err error, provider string) string {
	for key, flag := range settingFlagNames {
		if strings.Contains(err.Error(), ".settings."+key+":") {
			return fmt.Sprintf("%s is not valid for provider %s", flag, provider)
		}
	}
	return err.Error()
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

func runConnectGitHub(cmd *cobra.Command, projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to GitHub Issues...")
	fmt.Println()

	project := connectPrompt(cmd, "Repository (owner/repo): ")
	if project == "" {
		return fmt.Errorf("repository is required")
	}

	token := connectSecret("Personal access token (needs 'repo' scope): ")
	if token == "" {
		return fmt.Errorf("secure terminal token input unavailable or empty; retry in a TTY or use --token-stdin")
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

func runConnectJira(cmd *cobra.Command, projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to Jira...")
	fmt.Println()

	baseURL := connectPrompt(cmd, "Jira base URL (e.g. https://mycompany.atlassian.net): ")
	if baseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	project := connectPrompt(cmd, "Project key (e.g. PROJ): ")
	if project == "" {
		return fmt.Errorf("project key is required")
	}

	userEmail := connectPrompt(cmd, "User email (for Jira Cloud basic auth): ")
	if userEmail == "" {
		return fmt.Errorf("user email is required for Jira Cloud API authentication")
	}

	token := connectSecret("API token (from https://id.atlassian.com/manage-profile/security/api-tokens): ")
	if token == "" {
		return fmt.Errorf("secure terminal token input unavailable or empty; retry in a TTY or use --token-stdin")
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

func runConnectLinear(cmd *cobra.Command, projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to Linear...")
	fmt.Println()

	project := connectPrompt(cmd, "Team key (e.g. ENG): ")
	if project == "" {
		return fmt.Errorf("team key is required")
	}

	token := connectSecret("API key (from https://linear.app/settings/api): ")
	if token == "" {
		return fmt.Errorf("secure terminal token input unavailable or empty; retry in a TTY or use --token-stdin")
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

func runConnectGitLab(cmd *cobra.Command, projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to GitLab...")
	fmt.Println()

	baseURL := connectPrompt(cmd, "GitLab base URL [https://gitlab.com]: ")
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	project := connectPrompt(cmd, "Project (namespace/project or numeric ID): ")
	if project == "" {
		return fmt.Errorf("project is required")
	}

	token := connectSecret("Personal/Project access token (needs 'api' scope): ")
	if token == "" {
		return fmt.Errorf("secure terminal token input unavailable or empty; retry in a TTY or use --token-stdin")
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

func runConnectConfluence(cmd *cobra.Command, projectRoot string, creds config.Credentials) error {
	fmt.Println("Connecting to Confluence...")
	fmt.Println()

	baseURL := connectPrompt(cmd, "Confluence base URL (e.g. https://mycompany.atlassian.net/wiki): ")
	if baseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	spaceKey := connectPrompt(cmd, "Space key (e.g. ENG): ")
	if spaceKey == "" {
		return fmt.Errorf("space key is required")
	}

	userEmail := connectPrompt(cmd, "User email (for Confluence Cloud basic auth): ")

	token := connectSecret("API token: ")
	if token == "" {
		return fmt.Errorf("secure terminal token input unavailable or empty; retry in a TTY or use --token-stdin")
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
	integrationID := canonicalIntegrationID(trackerType, project)
	if connectGlobal {
		// Save to global credentials file
		config.SetIntegrationCredential(creds, integrationID, entry)
		if err := config.SaveCredentials(creds); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}
		fmt.Printf("Token saved to %s\n", config.CredentialsPath())
	} else {
		patch, _ := json.Marshal(map[string]any{"connections": map[string]any{integrationID: map[string]any{"auth": map[string]any{"token": entry.Token}}}})
		if err := config.PatchLocalIntegrations(projectRoot, config.DefaultFolder, patch); err != nil {
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

var nonID = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func canonicalIntegrationID(provider, project string) string {
	v := strings.Trim(nonID.ReplaceAllString(strings.ToLower(project), "-"), "-")
	if v == "" {
		v = "default"
	}
	return provider + "-" + v
}

// updateHeroJSON writes non-secret tracker/confluence config to hero.json if not already set.
func updateHeroJSON(projectRoot, trackerType, project, baseURL, userEmail string) error {
	id := canonicalIntegrationID(trackerType, project)
	settings := map[string]any{}
	if baseURL != "" {
		settings["base_url"] = baseURL
	}
	if userEmail != "" {
		settings["user_email"] = userEmail
	}
	role := "delivery"
	if trackerType == "confluence" {
		settings["space_key"] = project
		role = "docs"
	} else {
		settings["project"] = project
	}
	if err := config.ValidateConnectionSettings(id, trackerType, settings); err != nil {
		return fmt.Errorf("cannot connect %s: %s", trackerType, settingErrorToFlag(err, trackerType))
	}
	patch, _ := json.Marshal(map[string]any{"default": id, "roles": map[string]any{role: id}, "connections": map[string]any{id: map[string]any{"provider": trackerType, "settings": settings}}})
	return config.PatchCommittedIntegrations(projectRoot, config.DefaultFolder, patch)
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

// connectPrompt reads one answer from the command's input stream.
//
// The stream comes from cmd.InOrStdin() rather than a package-level
// bufio.Reader. The old mutable `connectInput` var existed only because a
// bufio.Reader buffers past the newline, so a fresh one per prompt would
// swallow the next answer — prompt.Prompt reads unbuffered, so the shared
// mutable state is no longer needed and is gone.
//
// A read error yields "", preserving the previous behaviour: the caller's own
// "X is required" error is the message the user sees, not a raw io.EOF.
func connectPrompt(cmd *cobra.Command, label string) string {
	answer, err := prompt.Prompt(cmd.InOrStdin(), cmd.OutOrStdout(), label)
	if err != nil {
		return ""
	}
	return answer
}

// connectSecret reads a credential from the terminal, never from stdin.
//
// Automation must use --token-stdin; silently falling back to echoed input
// would expose credentials. Returning "" on refusal preserves the existing
// caller behaviour, which reports "secure terminal token input unavailable or
// empty; retry in a TTY or use --token-stdin".
func connectSecret(label string) string {
	secret, err := prompt.Secret(label)
	if err != nil {
		return ""
	}
	return secret
}
