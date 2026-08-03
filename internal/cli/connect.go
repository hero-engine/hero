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
	Long: `Configure a tracker or wiki connection interactively, or with flags for automation.

Supported types: github, jira, linear, gitlab, confluence

Examples:
  hero connect github       — guided setup for GitHub Issues
  hero connect jira         — guided setup for Jira (asks for base_url too)
  hero connect linear       — guided setup for Linear
  hero connect confluence   — guided setup for Confluence wiki sync
  printf 'TOKEN' | hero connect github --project owner/repo --role code-host --token-stdin --no-verify
  hero connect --list       — show all saved connections
  hero connect --remove github --project owner/repo  — remove a saved connection

Tokens are saved to .hero/hero.local.json (project scope, gitignored) by default,
or to ~/.config/hero/credentials.json when --global is passed.

Flag-driven connect requires --project and --token-stdin; --role selects the
connection role, --local-only keeps all integration state local, and --json
never opens the interactive form. --no-verify skips provider verification and
should be used only when that risk is understood. Non-secret configuration is
written before credentials so failed configuration writes leave no token.`,
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
	connectCmd.Flags().StringVar(&connectProject, "project", "", "project identifier (required for flag-driven connect; used with --remove)")
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

	trackerType := ""
	if len(args) > 0 {
		trackerType = strings.ToLower(args[0])
	} else {
		if connectJSON || !prompt.IsInputTTY(cmd.InOrStdin()) {
			return fmt.Errorf("usage: hero connect <type>  (github, jira, linear, gitlab, confluence)\n\nRun 'hero connect --list' to see saved connections.")
		}
		providers := connectProviderNames()
		choice, err := prompt.Choice(cmd.InOrStdin(), cmd.OutOrStdout(), "Provider", providers)
		if err != nil {
			return err
		}
		if choice == "" {
			return fmt.Errorf("usage: hero connect <type>  (github, jira, linear, gitlab, confluence)\n\nRun 'hero connect --list' to see saved connections.")
		}
		trackerType = choice
	}

	// --role belongs in this predicate for the same reason every other flag
	// here does: it carries a value only the flag-driven writer can honor.
	// While it was missing, `hero connect github --role code-host` landed on
	// the interactive path, which hardcoded `delivery` — the user asked for a
	// code-host connection explicitly, got no error, and got a delivery one.
	//
	// Changed() rather than a non-empty check: --role defaults to "delivery",
	// so the flag's value can never distinguish "unset" from "set to the
	// default".
	if connectIntegrationID != "" || connectTokenStdin || connectProject != "" || connectBaseURL != "" || connectLocalOnly || cmd.Flags().Changed("role") {
		return runConnectNonInteractive(cmd, projectRoot, creds, trackerType)
	}
	return runConnectInteractive(cmd, projectRoot, creds, trackerType, map[string]string{"provider": trackerType})
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

// runConnectNonInteractive collects a connection from flags and hands it to
// writeConnection. It obtains values differently from the interactive path;
// it does not decide differently what a valid connection is.
func runConnectNonInteractive(cmd *cobra.Command, root string, creds config.Credentials, provider string) error {
	if _, ok := connectProviders[provider]; !ok {
		return fmt.Errorf("unknown provider %q", provider)
	}
	id := connectIntegrationID
	if id == "" {
		id = canonicalIntegrationID(provider, connectProject)
	}
	if connectProject == "" {
		return fmt.Errorf("--project is required for non-interactive connect")
	}
	// Reject an impossible provider/role pairing before consuming the token
	// stream or touching the network. writeConnection re-checks it as the
	// single authority; this is only about the order in which a doomed
	// invocation gives up.
	if _, err := config.ValidateProviderRole(provider, connectResolveRole(provider, connectRole)); err != nil {
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
		values := map[string]string{
			"project": connectProject, "base_url": connectBaseURL,
			"user_email": connectUserEmail, "token": token,
		}
		if verr := connectProviders[provider].verify(values); verr != nil {
			return fmt.Errorf("could not verify %s integration %q: %w", provider, id, verr)
		}
	}
	verification := "verified"
	if connectNoVerify {
		verification = "not-checked"
	}
	return writeConnection(cmd, root, creds, connectionInput{
		provider:     provider,
		id:           id,
		project:      connectProject,
		baseURL:      connectBaseURL,
		userEmail:    connectUserEmail,
		token:        token,
		role:         connectRole,
		verification: verification,
	})
}

// ---------------------------------------------------------------------------
// the write path — one function, both entry points
// ---------------------------------------------------------------------------

// connectionInput is one fully-collected connection, ready to persist.
type connectionInput struct {
	provider  string
	id        string
	project   string // repository, project key, team key, or space key
	baseURL   string
	userEmail string
	token     string
	role      string
	// verification is what --list and the --json result report about whether
	// the provider was actually reached.
	verification string
}

// writeConnection is the only function that persists a connection.
//
// It used to have a twin. `saveConnection` + `updateHeroJSON` served the
// interactive path and disagreed with this code about all three of the things
// a connection is selected by: it hardcoded the role to delivery/docs, never
// wrote `capabilities`, and always claimed `default`. The missing
// `capabilities` was the load-bearing half — with none written,
// EffectiveCapabilities falls back to the provider's legacy `tracker`, and
// ResolveCodeHostConnection rejects the connection outright.
//
// So role, capabilities, and default resolve here and nowhere else. Adding a
// second writer reintroduces the entire class of bug.
func writeConnection(cmd *cobra.Command, root string, creds config.Credentials, in connectionInput) error {
	role := connectResolveRole(in.provider, in.role)
	// The capability derivation is exhaustive by construction:
	// ValidateProviderRole fails an unknown role and a provider that cannot
	// serve a known one, so no path reaches persistence with an empty
	// capability — which would reproduce the legacy-`tracker` fallback this
	// function exists to close.
	requiredCapability, err := config.ValidateProviderRole(in.provider, role)
	if err != nil {
		return err
	}

	settings := map[string]any{"project": in.project}
	if in.provider == "confluence" {
		delete(settings, "project")
		settings["space_key"] = in.project
	}
	if in.baseURL != "" {
		settings["base_url"] = in.baseURL
	}
	if in.userEmail != "" {
		settings["user_email"] = in.userEmail
	}
	if err := config.ValidateConnectionSettings(in.id, in.provider, settings); err != nil {
		return fmt.Errorf("cannot connect %s: %s", in.provider, settingErrorToFlag(err, in.provider))
	}

	connection := map[string]any{"provider": in.provider, "settings": settings}
	existingCfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if existingCfg.Integrations != nil {
		if existing, ok := existingCfg.Integrations.Connections[in.id]; ok {
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
		"roles":       map[string]any{role: in.id},
		"connections": map[string]any{in.id: connection},
	}
	// A code-host connection is selected by its role, never by falling back to
	// the default, so claiming the default would only steal it from whatever
	// tracker holds it.
	if role != "code-host" {
		patch["default"] = in.id
	}

	if connectLocalOnly {
		connection["auth"] = map[string]any{"token": in.token}
		p, _ := json.Marshal(patch)
		if err := config.PatchLocalIntegrations(root, config.DefaultFolder, p); err != nil {
			return err
		}
	} else {
		p, _ := json.Marshal(patch)
		if err := config.PatchCommittedIntegrations(root, config.DefaultFolder, p); err != nil {
			return err
		}
		if in.token != "" {
			if connectGlobal {
				config.SetIntegrationCredential(creds, in.id, config.CredentialEntry{Token: in.token, BaseURL: in.baseURL, UserEmail: in.userEmail})
				if err := config.SaveCredentials(creds); err != nil {
					return err
				}
			} else {
				p, _ := json.Marshal(map[string]any{"connections": map[string]any{in.id: map[string]any{"auth": map[string]any{"token": in.token}}}})
				if err := config.PatchLocalIntegrations(root, config.DefaultFolder, p); err != nil {
					return err
				}
			}
		}
	}

	result := map[string]any{
		"id": in.id, "provider": in.provider, "role": role, "capability": requiredCapability,
		"ready": in.token != "" || connectGlobal, "verification": in.verification,
	}
	if connectJSON {
		b, _ := json.Marshal(result)
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Connected integration %s (%s). Inspect with 'hero connect --list'.\n", in.id, in.provider)
	}
	return nil
}

// connectResolveRole applies connect's one role default: Confluence has no
// tracker capability, so the `delivery` default means `docs` there.
func connectResolveRole(provider, role string) string {
	if provider == "confluence" && role == "delivery" {
		return "docs"
	}
	return role
}

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
// the interactive path — collect, then write
// ---------------------------------------------------------------------------

// connectProvider is the per-provider half of interactive connect: what it
// says, and how it proves the credential works.
type connectProvider struct {
	intro  string // banner printed before the first prompt
	name   string // display name used in the verification error
	verify func(values map[string]string) error
}

var connectProviders = map[string]connectProvider{
	"github": {intro: "Connecting to GitHub Issues...", name: "GitHub", verify: func(v map[string]string) error {
		return verifyGitHubToken(v["project"], v["token"])
	}},
	"jira": {intro: "Connecting to Jira...", name: "Jira", verify: func(v map[string]string) error {
		return verifyJiraToken(v["base_url"], v["project"], v["user_email"], v["token"])
	}},
	"linear": {intro: "Connecting to Linear...", name: "Linear", verify: func(v map[string]string) error {
		return verifyLinearToken(v["project"], v["token"])
	}},
	"gitlab": {intro: "Connecting to GitLab...", name: "GitLab", verify: func(v map[string]string) error {
		return verifyGitLabToken(v["base_url"], v["project"], v["token"])
	}},
	"confluence": {intro: "Connecting to Confluence...", name: "Confluence", verify: func(v map[string]string) error {
		return verifyConfluenceToken(v["base_url"], v["project"], v["user_email"], v["token"])
	}},
}

// secretUnavailable is the message every provider's credential prompt reports
// when no terminal is available. It names --token-stdin because that is the
// non-interactive alternative; prompt.Secret deliberately will not read a
// credential from a stream.
const secretUnavailable = "secure terminal token input unavailable or empty; retry in a TTY or use --token-stdin"

// connectFields is the interactive collection order for every provider.
//
// One flat table rather than five near-identical functions. The conditionality
// is real, not decorative: base_url exists only for the three self-hosted-or-
// tenanted providers, user_email only where Cloud basic auth needs it, and the
// labels genuinely differ per provider. Ordering within a provider is the
// order these lines appear in.
//
// The values map is seeded with "provider", which is what every DependsOn
// tests against. See promptfield.go for the cap on what this may grow into.
type connectField struct {
	name       string
	provider   string
	label      string
	defaultVal string
	missingErr string
	secret     bool
}

var connectFields = []connectField{
	// github
	{name: "project", provider: "github", label: "Repository (owner/repo): ", missingErr: "repository is required"},
	{name: "token", provider: "github", secret: true, label: "Personal access token (needs 'repo' scope): ", missingErr: secretUnavailable},

	// jira
	{name: "base_url", provider: "jira", label: "Jira base URL (e.g. https://mycompany.atlassian.net): ", missingErr: "base URL is required"},
	{name: "project", provider: "jira", label: "Project key (e.g. PROJ): ", missingErr: "project key is required"},
	{name: "user_email", provider: "jira", label: "User email (for Jira Cloud basic auth): ", missingErr: "user email is required for Jira Cloud API authentication"},
	{name: "token", provider: "jira", secret: true, label: "API token (from https://id.atlassian.com/manage-profile/security/api-tokens): ", missingErr: secretUnavailable},

	// linear
	{name: "project", provider: "linear", label: "Team key (e.g. ENG): ", missingErr: "team key is required"},
	{name: "token", provider: "linear", secret: true, label: "API key (from https://linear.app/settings/api): ", missingErr: secretUnavailable},

	// gitlab
	{name: "base_url", provider: "gitlab", label: "GitLab base URL [https://gitlab.com]: ", defaultVal: "https://gitlab.com"},
	{name: "project", provider: "gitlab", label: "Project (namespace/project or numeric ID): ", missingErr: "project is required"},
	{name: "token", provider: "gitlab", secret: true, label: "Personal/Project access token (needs 'api' scope): ", missingErr: secretUnavailable},

	// confluence — user_email is optional here and required for jira, which is
	// the whole reason the message is per-field rather than per-command.
	{name: "base_url", provider: "confluence", label: "Confluence base URL (e.g. https://mycompany.atlassian.net/wiki): ", missingErr: "base URL is required"},
	{name: "project", provider: "confluence", label: "Space key (e.g. ENG): ", missingErr: "space key is required"},
	{name: "user_email", provider: "confluence", label: "User email (for Confluence Cloud basic auth): "},
	{name: "token", provider: "confluence", secret: true, label: "API token: ", missingErr: secretUnavailable},
}

// runConnectInteractive collects a connection from the user and hands it to
// writeConnection — the same writer the flag-driven path uses.
//
// known carries values that are already settled before any prompt;
// collectFields skips a field whose value is present. Production passes only
// the provider. Tests pass the credential too, because prompt.Secret reads
// /dev/tty by construction and there is deliberately no stream to feed it —
// seeding a known value is the only way to exercise the rest of this path
// without weakening that guarantee.
func runConnectInteractive(cmd *cobra.Command, root string, creds config.Credentials, provider string, known map[string]string) error {
	p, ok := connectProviders[provider]
	if !ok {
		return fmt.Errorf("unknown type %q — supported: github, jira, linear, gitlab, confluence", provider)
	}
	// --json is a machine-readable contract on stdout, and every value this
	// path needs comes from a prompt. Refuse rather than ask.
	if connectJSON {
		return fmt.Errorf("--project is required with --json: pass --project (with --integration-id and --token-stdin) instead of the interactive form")
	}

	if !prompt.IsInputTTY(cmd.InOrStdin()) {
		if err := firstMissingConnectField(provider, known); err != nil {
			return err
		}
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, p.intro)
	fmt.Fprintln(out)

	role, err := connectPromptRole(cmd, provider)
	if err != nil {
		return err
	}
	if err := collectConnectFields(cmd, provider, known); err != nil {
		return err
	}
	known["base_url"] = strings.TrimRight(known["base_url"], "/")

	verification := "not-checked"
	if !connectNoVerify {
		fmt.Fprintln(out)
		fmt.Fprint(out, "Verifying connection... ")
		if err := p.verify(known); err != nil {
			fmt.Fprintln(out, "FAILED")
			return fmt.Errorf("could not verify %s connection: %w", p.name, err)
		}
		fmt.Fprintln(out, "OK")
		verification = "verified"
	}

	return writeConnection(cmd, root, creds, connectionInput{
		provider:     provider,
		id:           canonicalIntegrationID(provider, known["project"]),
		project:      known["project"],
		baseURL:      known["base_url"],
		userEmail:    known["user_email"],
		token:        known["token"],
		role:         role,
		verification: verification,
	})
}

// connectRead reads one interactive connect answer.
//
// A read error yields "" rather than propagating: the caller's own
// "X is required" message is what an unanswered prompt has always reported,
// not a raw io.EOF. The golden fixtures under testdata/prompt_baseline record
// exactly that for a piped and a closed stdin.
func firstMissingConnectField(provider string, values map[string]string) error {
	for _, field := range connectFields {
		if field.provider == provider && values[field.name] == "" && field.missingErr != "" {
			return fmt.Errorf("%s", field.missingErr)
		}
	}
	return nil
}

func collectConnectFields(cmd *cobra.Command, provider string, values map[string]string) error {
	for _, field := range connectFields {
		if field.provider != provider || values[field.name] != "" {
			continue
		}
		var answer string
		var err error
		if field.secret {
			answer, err = prompt.Secret(field.label)
		} else {
			answer, err = prompt.Prompt(cmd.InOrStdin(), cmd.OutOrStdout(), field.label)
		}
		if err != nil {
			answer = ""
		}
		if answer == "" {
			answer = field.defaultVal
		}
		if answer == "" && field.missingErr != "" {
			return fmt.Errorf("%s", field.missingErr)
		}
		values[field.name] = answer
	}
	return nil
}

func connectPrompt(cmd *cobra.Command, label string) string {
	answer, err := prompt.Prompt(cmd.InOrStdin(), cmd.OutOrStdout(), label)
	if err != nil {
		return ""
	}
	return answer
}

func connectSecret(label string) string {
	secret, err := prompt.Secret(label)
	if err != nil {
		return ""
	}
	return secret
}

func connectProviderNames() []string {
	names := make([]string, 0, len(connectProviders))
	for name := range connectProviders {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func connectProviderOptions() []string { return connectProviderNames() }

// connectPromptRole asks what the connection is for, offering only the roles
// the provider can actually serve.
//
// Gated on a terminal for a specific reason: --role now routes to the
// flag-driven path, so every scripted invocation already carries the answer.
// Prompting into a pipe would add output to invocations that never asked a
// question before, which the prompt baseline would catch and which nobody
// would benefit from. A provider with a single possible role (Confluence) is
// not asked at all.
func connectPromptRole(cmd *cobra.Command, provider string) (string, error) {
	options := connectRoleOptions(provider)
	if len(options) < 2 || !prompt.IsInputTTY(cmd.InOrStdin()) {
		return connectRole, nil
	}
	choice, err := prompt.Choice(cmd.InOrStdin(), cmd.OutOrStdout(), "Role", options)
	if err != nil {
		return "", err
	}
	if choice == "" {
		return connectRole, nil
	}
	return choice, nil
}

// connectRoleOptions returns the roles this provider has the capability to
// serve, in a stable order.
func connectRoleOptions(provider string) []string {
	options := []string{}
	for _, role := range []string{"delivery", "docs", "roadmap", "code-host"} {
		if _, err := config.ValidateProviderRole(provider, role); err == nil {
			options = append(options, role)
		}
	}
	return options
}

var nonID = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func canonicalIntegrationID(provider, project string) string {
	v := strings.Trim(nonID.ReplaceAllString(strings.ToLower(project), "-"), "-")
	if v == "" {
		v = "default"
	}
	return provider + "-" + v
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
