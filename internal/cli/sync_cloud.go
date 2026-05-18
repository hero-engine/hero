package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var syncCloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Push specs to Hero Cloud",
	Long: `Pushes spec metadata from the local workspace to Hero Cloud.

By default only metadata is synced (slug, title, type, status, tags, etc.).
Use --full to include full spec body content for cross-repo search.

Requires authentication — run 'hero login' first.

Examples:
  hero sync cloud                # push metadata to cloud
  hero sync cloud --full         # push full spec bodies
  hero sync cloud --status       # show sync status without pushing`,
	RunE: runSyncCloud,
}

var (
	syncCloudFull   bool
	syncCloudStatus bool
)

func init() {
	syncCloudCmd.Flags().BoolVar(&syncCloudFull, "full", false, "include full spec body content")
	syncCloudCmd.Flags().BoolVar(&syncCloudStatus, "status", false, "show sync status without pushing")
}

// cloudSpec is the payload sent to the cloud sync endpoint.
type cloudSpec struct {
	Slug         string            `json:"slug"`
	Title        string            `json:"title"`
	Type         string            `json:"type"`
	Status       string            `json:"status"`
	Priority     string            `json:"priority,omitempty"`
	ClaimedBy    string            `json:"claimed_by,omitempty"`
	TrackerID    string            `json:"tracker_id,omitempty"`
	Subproject   string            `json:"subproject,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	FilesTouched []string          `json:"files_touched,omitempty"`
	Sections     map[string]string `json:"sections,omitempty"`
	RawContent   string            `json:"raw_content,omitempty"`
	Checksum     string            `json:"checksum"`
}

func runSyncCloud(cmd *cobra.Command, args []string) error {
	token := LoadCloudToken()
	if token == "" {
		return fmt.Errorf("not logged in — run 'hero login' first")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	// Discover all specs
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	if len(specs) == 0 {
		fmt.Println("No specs found to sync.")
		return nil
	}

	if syncCloudStatus {
		fmt.Printf("Found %d specs to sync:\n", len(specs))
		for _, s := range specs {
			fmt.Printf("  %s — %s (%s)\n", s.Slug, s.Title, s.Status)
		}
		fmt.Println("\nRun 'hero sync cloud' (without --status) to push.")
		return nil
	}

	// Build payload
	cloudSpecs := make([]cloudSpec, 0, len(specs))
	for _, s := range specs {
		cs := cloudSpec{
			Slug:         s.Slug,
			Title:        s.Title,
			Type:         string(s.Type),
			Status:       string(s.Status),
			Priority:     s.Priority,
			ClaimedBy:    s.ClaimedBy,
			TrackerID:    s.TrackerID,
			Subproject:   s.Subproject,
			Tags:         s.Tags,
			FilesTouched: s.FilesTouched,
			Checksum:     contentChecksum(s.RawContent),
		}

		if syncCloudFull {
			cs.RawContent = s.RawContent
			// Include full sections
			cs.Sections = s.Sections
		} else {
			// Metadata only — just section headings
			if len(s.Sections) > 0 {
				headings := make(map[string]string, len(s.Sections))
				for k := range s.Sections {
					headings[k] = "" // heading only, no content
				}
				cs.Sections = headings
			}
		}

		cloudSpecs = append(cloudSpecs, cs)
	}

	// Determine org and repo from config or credentials
	cloudURL := cloudBaseURL()
	orgID, repoID, err := resolveCloudTarget(cfg, token, cloudURL)
	if err != nil {
		return fmt.Errorf("resolving cloud target: %w", err)
	}

	// POST to sync endpoint
	payload, err := json.Marshal(map[string]interface{}{
		"specs": cloudSpecs,
	})
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/orgs/%s/repos/%s/sync", cloudURL, orgID, repoID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("syncing to cloud: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloud sync failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Synced int `json:"synced"`
		Total  int `json:"total"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Printf("Synced %d/%d specs to Hero Cloud.\n", result.Synced, result.Total)
	return nil
}

// resolveCloudTarget determines the org_id and repo_id for sync.
// It checks cloud config in hero.json first, then falls back to listing orgs.
func resolveCloudTarget(cfg config.Config, token, cloudURL string) (string, string, error) {
	if cfg.Cloud != nil && cfg.Cloud.OrgID != "" && cfg.Cloud.RepoID != "" {
		return cfg.Cloud.OrgID, cfg.Cloud.RepoID, nil
	}

	// Fall back to listing user's orgs and picking the first one
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", cloudURL+"/api/v1/orgs", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("listing orgs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to list orgs (status %d)", resp.StatusCode)
	}

	var orgsResp struct {
		Orgs []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"orgs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&orgsResp); err != nil {
		return "", "", err
	}
	if len(orgsResp.Orgs) == 0 {
		return "", "", fmt.Errorf("no orgs found — create one at Hero Cloud first")
	}

	orgID := orgsResp.Orgs[0].ID

	// List repos in the org, try to find one matching the project name
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/api/v1/orgs/%s/repos", cloudURL, orgID), nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp2, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("listing repos: %w", err)
	}
	defer resp2.Body.Close()

	var reposResp struct {
		Repos []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"repos"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&reposResp); err != nil {
		return "", "", err
	}

	// Try to match by project directory name
	projectName := projectDirName()
	for _, r := range reposResp.Repos {
		if strings.EqualFold(r.Name, projectName) {
			return orgID, r.ID, nil
		}
	}

	// If only one repo, use it
	if len(reposResp.Repos) == 1 {
		return orgID, reposResp.Repos[0].ID, nil
	}

	// No matching repo — create one
	createPayload, _ := json.Marshal(map[string]string{"name": projectName})
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/api/v1/orgs/%s/repos", cloudURL, orgID), bytes.NewReader(createPayload))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp3, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("creating repo: %w", err)
	}
	defer resp3.Body.Close()

	var newRepo struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&newRepo); err != nil {
		return "", "", fmt.Errorf("parsing new repo: %w", err)
	}

	fmt.Printf("Created repo '%s' in org '%s'\n", projectName, orgsResp.Orgs[0].Name)
	return orgID, newRepo.ID, nil
}

func projectDirName() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	parts := strings.Split(dir, string(os.PathSeparator))
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

func contentChecksum(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:8]) // short 16-char hash
}
