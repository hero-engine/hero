package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

// cloudCmd is the parent for org/repo management commands that talk to
// Hero Cloud. Authentication (login/logout) stays at the top level for
// backwards compatibility; this group hosts the bootstrap helpers a
// second-person user needs (create-org, ...).
var cloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Manage Hero Cloud orgs and repos",
	Long: `Commands for bootstrapping Hero Cloud from the CLI.

Run 'hero login' first to authenticate, then use these to create an org
and link a repo without opening the dashboard.`,
}

var createOrgSlug string

var cloudCreateOrgCmd = &cobra.Command{
	Use:   "create-org <name>",
	Short: "Create a Hero Cloud organization",
	Long: `Creates a new organization in Hero Cloud and records its id in hero.json.

A URL-safe slug is derived from <name> automatically. If that slug is
taken, or you want a specific one, pass --slug.`,
	Args: cobra.ExactArgs(1),
	RunE: runCreateOrg,
}

func init() {
	cloudCreateOrgCmd.Flags().StringVar(&createOrgSlug, "slug", "", "explicit org slug (3-40 lowercase alphanumeric/hyphen); overrides the derived one")
	cloudCmd.AddCommand(cloudCreateOrgCmd)
}

func runCreateOrg(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("org name must not be empty")
	}

	slug := createOrgSlug
	if slug == "" {
		derived := orgSlugify(name)
		if !validSlug(derived) {
			return fmt.Errorf("could not derive a valid slug from %q — pass --slug (3-40 lowercase alphanumeric/hyphen)", name)
		}
		slug = derived
	}

	token := LoadCloudToken()
	if token == "" {
		return fmt.Errorf("not logged in — run 'hero login' first")
	}

	cloudURL := cloudBaseURL()
	payload, err := json.Marshal(map[string]string{"name": name, "slug": slug})
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequest("POST", cloudURL+"/api/v1/orgs", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to Hero Cloud at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusCreated:
		var org struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(body, &org); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		if org.ID == "" {
			return fmt.Errorf("cloud returned 201 but no org id")
		}
		if err := persistOrgID(org.ID); err != nil {
			// The org exists in the cloud; surface the write failure but
			// don't pretend the org wasn't created.
			fmt.Printf("Created org '%s' (%s)\n", name, org.ID)
			return fmt.Errorf("org created but failed to write hero.json: %w", err)
		}
		fmt.Printf("Created org '%s' (%s)\n", name, org.ID)
		return nil

	case http.StatusConflict:
		return fmt.Errorf("slug %q already taken — re-run with a different --slug", slug)

	case http.StatusBadRequest:
		return fmt.Errorf("invalid slug %q — must be 3-40 lowercase alphanumeric characters or hyphens", slug)

	default:
		return fmt.Errorf("create org failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

// persistOrgID writes the resolved org id into hero.json, preserving any
// existing repo id. Returns an error if the project root has no config.
func persistOrgID(orgID string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	repoID := ""
	if cfg.Cloud != nil {
		repoID = cfg.Cloud.RepoID
	}
	cfg.Cloud = &config.CloudConfig{OrgID: orgID, RepoID: repoID}
	return cfg.Save(projectRoot)
}

// orgSlugify derives a URL-safe slug from a display name:
// lowercase; spaces/underscores -> '-'; drop chars outside [a-z0-9-];
// collapse repeated '-'; trim leading/trailing '-'; clamp to 40 chars.
// The result may still be invalid (e.g. too short) — callers must check
// with validSlug.
func orgSlugify(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '-':
			b.WriteRune('-')
		default:
			// drop everything else
		}
	}

	// Collapse repeated hyphens.
	collapsed := b.String()
	for strings.Contains(collapsed, "--") {
		collapsed = strings.ReplaceAll(collapsed, "--", "-")
	}
	collapsed = strings.Trim(collapsed, "-")

	// Clamp to the 40-char ceiling, then re-trim trailing hyphens the
	// clamp may have exposed.
	if len(collapsed) > 40 {
		collapsed = strings.Trim(collapsed[:40], "-")
	}
	return collapsed
}

// validSlug reports whether s satisfies the cloud's slug contract:
// 3-40 chars, lowercase alphanumeric or hyphen, no leading/trailing hyphen.
// Mirrors `^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$` on the cloud side.
func validSlug(s string) bool {
	if len(s) < 3 || len(s) > 40 {
		return false
	}
	for i, r := range s {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if i == 0 || i == len(s)-1 {
			if !isAlnum {
				return false
			}
			continue
		}
		if !isAlnum && r != '-' {
			return false
		}
	}
	return true
}
