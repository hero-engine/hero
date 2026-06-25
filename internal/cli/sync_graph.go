package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/spf13/cobra"
)

// `hero sync graph` umbrella + push / pull subverbs.
//
// Push-pull is the federation protocol — clients push their team-scope
// + unit-scope graph deltas to the team server, and pull deltas other
// teammates pushed since the last cursor. ScopeLocal nodes never
// leave the machine.

var syncGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Push / pull knowledge-graph deltas with Hero Cloud",
	Long: `Federation sync for the unified knowledge graph.

  hero sync graph push   send local team-scope deltas to the cloud
  hero sync graph pull   pull deltas other teammates pushed
  hero sync graph status show the current sync cursor

ScopeLocal rows never leave the machine. Authentication is the
same as ` + "`hero sync cloud`" + ` — run ` + "`hero login`" + ` first.`,
}

var syncGraphPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push local graph deltas to Hero Cloud",
	RunE:  runSyncGraphPush,
}

var syncGraphPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull graph deltas from Hero Cloud since the last cursor",
	RunE:  runSyncGraphPull,
}

var syncGraphStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current sync cursor and pending counts",
	RunE:  runSyncGraphStatus,
}

func init() {
	syncGraphCmd.AddCommand(syncGraphPushCmd)
	syncGraphCmd.AddCommand(syncGraphPullCmd)
	syncGraphCmd.AddCommand(syncGraphStatusCmd)
	syncCmd.AddCommand(syncGraphCmd)
}

// graphSyncSetup is the common boilerplate for the three subverbs:
// load config, locate the workspace, build a SyncClient with auth.
type graphSyncContext struct {
	store    *graph.Store
	client   *graph.SyncClient
	repoKey  string
	cleanup  func()
}

func setupGraphSync(orgIDOverride string) (*graphSyncContext, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	creds, err := loadRefreshedCredentials()
	if err != nil {
		return nil, fmt.Errorf("loading cloud credentials (run 'hero login' first): %w", err)
	}
	if creds == nil || creds.AccessToken == "" {
		return nil, fmt.Errorf("not logged in — run 'hero login' first")
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return nil, fmt.Errorf("opening graph: %w", err)
	}

	orgID := orgIDOverride
	if orgID == "" && cfg.Cloud != nil {
		orgID = cfg.Cloud.OrgID
	}
	if orgID == "" {
		store.Close()
		return nil, fmt.Errorf("no org configured — set cloud.org_id in hero.json or pass --org")
	}

	cloudURL := creds.CloudURL
	if cloudURL == "" {
		cloudURL = cloudBaseURL()
	}
	repoKey := gitutil.RepoKey(projectRoot)

	// Build a SyncClient whose HTTP transport injects the bearer token
	// on every request.
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: newAuthTransport(creds),
	}
	c := graph.NewSyncClient(
		fmt.Sprintf("%s/api/v1/orgs/%s", cloudURL, orgID),
		repoKey,
		orgID,
	)
	// Override the path prefix injected into URLs — sync API expects the
	// org prefix to be already part of the ServerURL.
	c.HTTP = httpClient

	return &graphSyncContext{
		store:   store,
		client:  c,
		repoKey: repoKey,
		cleanup: func() { store.Close() },
	}, nil
}

// authTransport adds the bearer token to outbound requests and, on a
// 401, refreshes the access token once and replays the request. This is
// the CLI-side half of cloud-cli-verify AC #6 (mid-sync token refresh).
type authTransport struct {
	mu    sync.Mutex
	token string
	creds *cloudCredentials // nil disables refresh
	base  http.RoundTripper
}

func newAuthTransport(creds *cloudCredentials) *authTransport {
	return &authTransport{
		token: creds.AccessToken,
		creds: creds,
		base:  http.DefaultTransport,
	}
}

func (t *authTransport) currentToken() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.token
}

// refreshIfStale refreshes the access token unless another in-flight
// request already rotated it (detected by comparing the token that just
// failed). Returns the usable token, or "" when refresh is impossible.
func (t *authTransport) refreshIfStale(failedToken string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token != failedToken {
		return t.token // a concurrent request already refreshed
	}
	if t.creds == nil {
		return ""
	}
	refreshed := tryRefresh(t.creds)
	if refreshed == nil {
		return ""
	}
	t.creds = refreshed
	t.token = refreshed.AccessToken
	return refreshed.AccessToken
}

func (t *authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	used := t.currentToken()
	r.Header.Set("Authorization", "Bearer "+used)
	resp, err := t.base.RoundTrip(r)
	if err != nil || resp.StatusCode != http.StatusUnauthorized || t.creds == nil {
		return resp, err
	}

	// Single refresh-and-retry on 401.
	fresh := t.refreshIfStale(used)
	if fresh == "" {
		return resp, nil // surface the 401; caller renders the error
	}
	// Only retry when the body can be rebuilt — GetBody is set for the
	// bytes/strings readers the sync client uses.
	if r.Body != nil && r.GetBody == nil {
		return resp, nil
	}
	resp.Body.Close()
	retry := r.Clone(r.Context())
	if r.GetBody != nil {
		body, berr := r.GetBody()
		if berr != nil {
			return resp, nil
		}
		retry.Body = body
	}
	retry.Header.Set("Authorization", "Bearer "+fresh)
	return t.base.RoundTrip(retry)
}

// augmentAuthError appends a re-login hint when the server rejected our
// credentials (401) and an automatic refresh could not recover.
func augmentAuthError(err error) error {
	if err != nil && strings.Contains(err.Error(), "401") {
		return fmt.Errorf("%w — run 'hero login' to re-authenticate", err)
	}
	return err
}

func runSyncGraphPush(cmd *cobra.Command, args []string) error {
	ctx, err := setupGraphSync("")
	if err != nil {
		return err
	}
	defer ctx.cleanup()

	resp, err := ctx.store.Push(ctx.client)
	if err != nil {
		return augmentAuthError(fmt.Errorf("push: %w", err))
	}
	fmt.Printf("Pushed: %d rows accepted (server time %s)\n", resp.Accepted, resp.ServerTime)
	if len(resp.Conflicts) > 0 {
		fmt.Printf("Warning: %d conflict(s) — your version won, but a teammate's version was overwritten:\n", len(resp.Conflicts))
		for _, c := range resp.Conflicts {
			fmt.Printf("  %s %s — %s\n", c.NodeType, c.NodeKey, c.Reason)
		}
		fmt.Println("Run 'hero sync graph pull && hero scan' to reconcile if needed.")
		_ = savePushConflicts(resp.Conflicts)
	}
	return nil
}

// pushConflictRecord is one entry in .hero/push_conflicts.json.
type pushConflictRecord struct {
	NodeType   string `json:"node_type"`
	NodeKey    string `json:"node_key"`
	Reason     string `json:"reason"`
	DetectedAt string `json:"detected_at"`
}

// savePushConflicts appends push conflicts to .hero/push_conflicts.json
// so that `hero check conflicts` can surface them by slug.
func savePushConflicts(conflicts []graph.SyncConflict) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return err
	}
	heroDir := cfg.HeroDir(projectRoot)
	path := filepath.Join(heroDir, "push_conflicts.json")

	// Load existing records.
	var records []pushConflictRecord
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &records)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range conflicts {
		records = append(records, pushConflictRecord{
			NodeType:   c.NodeType,
			NodeKey:    c.NodeKey,
			Reason:     c.Reason,
			DetectedAt: now,
		})
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func runSyncGraphPull(cmd *cobra.Command, args []string) error {
	ctx, err := setupGraphSync("")
	if err != nil {
		return err
	}
	defer ctx.cleanup()

	_, nodesApplied, edgesApplied, edgesDeferred, err := ctx.store.Pull(ctx.client)
	if err != nil {
		return augmentAuthError(fmt.Errorf("pull: %w", err))
	}
	fmt.Printf("Pulled: %d nodes, %d edges applied", nodesApplied, edgesApplied)
	if edgesDeferred > 0 {
		fmt.Printf(" (%d edges deferred — endpoints not yet local)", edgesDeferred)
	}
	fmt.Println()
	return nil
}

func runSyncGraphStatus(cmd *cobra.Command, args []string) error {
	ctx, err := setupGraphSync("")
	if err != nil {
		return err
	}
	defer ctx.cleanup()

	cursor, _ := ctx.store.LastPullCursor(ctx.client.ServerURL)
	pending, _, _, _ := ctx.store.PendingPush(ctx.client.ServerURL)
	fmt.Printf("Server:        %s\n", ctx.client.ServerURL)
	fmt.Printf("Repo:          %s\n", ctx.repoKey)
	fmt.Printf("Pull cursor:   %s\n", cursorOrNever(cursor))
	fmt.Printf("Pending push:  %d nodes\n", len(pending))
	return nil
}

func cursorOrNever(s string) string {
	if s == "" {
		return "(never pulled)"
	}
	return s
}
