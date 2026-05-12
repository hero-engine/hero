package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/hero-engine/hero/internal/config"
)

// impactReply mirrors the JSON shape returned by
// GET /api/v1/orgs/{org_id}/graph/impact.
type impactReply struct {
	TargetType string                  `json:"target_type"`
	TargetKey  string                  `json:"target_key"`
	Total      int                     `json:"total"`
	Repos      []string                `json:"repos"`
	ByRepo     map[string][]impactRow  `json:"by_repo"`
}

type impactRow struct {
	FromRepo  string          `json:"from_repo"`
	FromType  string          `json:"from_type"`
	FromKey   string          `json:"from_key"`
	EdgeType  string          `json:"edge_type"`
	EdgeProps json.RawMessage `json:"edge_props,omitempty"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func runImpactCrossRepo(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("cross-repo impact requires a target key (symbol / file / package)")
	}
	target := args[0]

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	creds, err := loadCredentials()
	if err != nil {
		return fmt.Errorf("loading cloud credentials (run 'hero login' first): %w", err)
	}
	if creds.AccessToken == "" {
		return fmt.Errorf("not logged in — run 'hero login' first")
	}

	orgID := ""
	if cfg.Cloud != nil {
		orgID = cfg.Cloud.OrgID
	}
	if orgID == "" {
		return fmt.Errorf("no org configured — set cloud.org_id in hero.json")
	}

	q := url.Values{}
	q.Set("type", impactType)
	q.Set("key", target)

	endpoint := fmt.Sprintf("%s/api/v1/orgs/%s/graph/impact?%s", cloudBaseURL(), orgID, q.Encode())
	req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("impact request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("server %d", resp.StatusCode)
	}

	var reply impactReply
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if impactFormatJSON {
		out, _ := json.MarshalIndent(reply, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if reply.Total == 0 {
		fmt.Printf("No cross-repo callers found for %s `%s`.\n", reply.TargetType, reply.TargetKey)
		return nil
	}

	fmt.Printf("Cross-repo blast radius for %s `%s` — %d caller(s) across %d repo(s):\n\n",
		reply.TargetType, reply.TargetKey, reply.Total, len(reply.Repos))

	repos := make([]string, 0, len(reply.Repos))
	repos = append(repos, reply.Repos...)
	sort.Strings(repos)

	for _, repo := range repos {
		callers := reply.ByRepo[repo]
		fmt.Printf("**%s** (%d):\n", repo, len(callers))
		for _, c := range callers {
			fmt.Printf("  - %s `%s` _(%s, last %s)_\n",
				c.FromType, c.FromKey, c.EdgeType,
				c.UpdatedAt.Format("2006-01-02"),
			)
		}
		fmt.Println()
	}
	return nil
}
