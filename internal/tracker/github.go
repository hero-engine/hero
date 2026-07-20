package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

const defaultGitHubAPI = "https://api.github.com"

// gitHub implements the Tracker interface for GitHub Issues.
type gitHub struct {
	owner   string
	repo    string
	token   string
	baseURL string
	client  *http.Client
	// configuredSizeMapping is the workspace-configured size mapping
	// (hero.json: tracker.size_mapping). Nil → use the shipped default.
	// See internal/tracker/size_mapping.go.
	configuredSizeMapping *config.SizeMappingConfig
}

func newGitHub(project, token, baseURL string) (*gitHub, error) {
	parts := strings.SplitN(project, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("github project must be in owner/repo format, got %q", project)
	}
	if baseURL == "" {
		baseURL = defaultGitHubAPI
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &gitHub{
		owner:   parts[0],
		repo:    parts[1],
		token:   token,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (g *gitHub) Name() string { return "github" }

// CreateIssue creates a GitHub issue from a spec. Returns the issue number as a string.
func (g *gitHub) CreateIssue(s *spec.Spec) (string, error) {
	labels := []string{
		fmt.Sprintf("hero:%s", s.Type),
		fmt.Sprintf("hero:%s", StatusLabel(s.Status)),
	}
	// Non-destructive size write on create: if the spec carries a
	// declared size and the mapping (configured or shipped default)
	// resolves it cleanly, append the mapped label. CreateIssue has
	// nothing to overwrite, so the planner isn't invoked — the
	// overwrite-safety check only matters on update.
	if s.Size != "" {
		if v, err := g.MapSize(s.Size); err == nil && v != "" {
			labels = append(labels, v)
		}
	}
	payload := map[string]interface{}{
		"title":  fmt.Sprintf("[%s] %s", s.Type, s.Title),
		"body":   IssueBody(s),
		"labels": labels,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling issue: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues", g.baseURL, g.owner, g.repo)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	g.setHeaders(req)

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("creating issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Number int    `json:"number"`
		URL    string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return fmt.Sprintf("%d", result.Number), nil
}

// UpdateSize rotates the size/<tier> label on a GitHub issue. Because
// GitHub PATCH on /issues replaces the whole label set, we do a
// read-then-write: fetch current labels, strip any label whose name
// starts with the configured size prefix, append the mapped
// <prefix><tier> label, and PATCH the merged set. All non-size labels
// — including hero:* labels — are preserved verbatim.
//
// One extra GET round-trip per push is the cost of safely preserving
// labels we don't own.
func (g *gitHub) UpdateSize(issueID, localTier string) error {
	newLabel, err := g.MapSize(localTier)
	if err != nil {
		return fmt.Errorf("mapping size %q: %w", localTier, err)
	}
	if newLabel == "" {
		return fmt.Errorf("size %q maps to empty label", localTier)
	}
	prefix := g.sizeMapping().Field // e.g. "size/"

	// Fetch current labels.
	getURL := fmt.Sprintf("%s/repos/%s/%s/issues/%s", g.baseURL, g.owner, g.repo, issueID)
	getReq, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return fmt.Errorf("creating GET request: %w", err)
	}
	g.setHeaders(getReq)
	getResp, err := g.client.Do(getReq)
	if err != nil {
		return fmt.Errorf("fetching issue labels: %w", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(getResp.Body)
		return fmt.Errorf("github API returned %d on GET: %s", getResp.StatusCode, string(respBody))
	}
	var current struct {
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&current); err != nil {
		return fmt.Errorf("decoding labels: %w", err)
	}

	// Build merged set: keep non-size labels, append the new size label.
	merged := make([]string, 0, len(current.Labels)+1)
	for _, l := range current.Labels {
		if !strings.HasPrefix(l.Name, prefix) {
			merged = append(merged, l.Name)
		}
	}
	merged = append(merged, newLabel)

	payload := map[string]interface{}{"labels": merged}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling labels: %w", err)
	}
	patchReq, err := http.NewRequest("PATCH", getURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating PATCH request: %w", err)
	}
	g.setHeaders(patchReq)
	patchResp, err := g.client.Do(patchReq)
	if err != nil {
		return fmt.Errorf("updating labels: %w", err)
	}
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(patchResp.Body)
		return fmt.Errorf("github API returned %d on PATCH: %s", patchResp.StatusCode, string(respBody))
	}
	return nil
}

// UpdateStatus adds a comment to the issue and optionally closes/reopens it.
func (g *gitHub) UpdateStatus(issueID string, status spec.Status) error {
	// Add a status comment
	comment := map[string]string{
		"body": fmt.Sprintf("**Hero status update:** %s", StatusLabel(status)),
	}
	commentBody, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("marshaling comment: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", g.baseURL, g.owner, g.repo, issueID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(commentBody))
	if err != nil {
		return fmt.Errorf("creating comment request: %w", err)
	}
	g.setHeaders(req)

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}
	resp.Body.Close()

	// Close the issue if the spec is completed or superseded
	if status == spec.StatusCompleted || status == spec.StatusSuperseded {
		return g.setIssueState(issueID, "closed")
	}

	return nil
}

// GetIssue retrieves issue info from GitHub.
func (g *gitHub) GetIssue(issueID string) (*Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", g.baseURL, g.owner, g.repo, issueID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	g.setHeaders(req)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		URL       string `json:"html_url"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	labels := make([]string, 0, len(result.Labels))
	for _, l := range result.Labels {
		labels = append(labels, l.Name)
	}

	return &Issue{
		ID:        fmt.Sprintf("%d", result.Number),
		Title:     result.Title,
		Status:    result.State,
		URL:       result.URL,
		CreatedAt: result.CreatedAt,
		UpdatedAt: result.UpdatedAt,
		Labels:    labels,
	}, nil
}

func (g *gitHub) setIssueState(issueID, state string) error {
	payload := map[string]string{"state": state}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", g.baseURL, g.owner, g.repo, issueID)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating patch request: %w", err)
	}
	g.setHeaders(req)

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("updating issue state: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github API returned %d when updating state", resp.StatusCode)
	}
	return nil
}

func (g *gitHub) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

// AddComment posts a comment to a GitHub issue.
func (g *gitHub) AddComment(issueID, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", g.baseURL, g.owner, g.repo, issueID)
	payload, _ := json.Marshal(map[string]string{"body": body})
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	g.setHeaders(req)

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github comment API returned %d", resp.StatusCode)
	}
	return nil
}

// AttachFile posts file contents as a comment (GitHub Issues doesn't support file attachments via API).
func (g *gitHub) AttachFile(issueID, filePath, fileName string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	body := fmt.Sprintf("**Attached: %s**\n\n<details>\n<summary>Click to expand</summary>\n\n```\n%s\n```\n\n</details>", fileName, string(content))
	return g.AddComment(issueID, body)
}

// ListIssues fetches open issues from GitHub. Optionally filters by label.
func (g *gitHub) ListIssues(label string, limit int) ([]Issue, error) {
	if limit <= 0 {
		limit = 30
	}

	pageSize := min(limit, 100)
	listURL := fmt.Sprintf("%s/repos/%s/%s/issues?state=open&per_page=%d", g.baseURL, g.owner, g.repo, pageSize)
	if label != "" {
		listURL += "&labels=" + url.QueryEscape(label)
	}

	return g.fetchIssueList(listURL, limit)
}

// Search fetches issues from GitHub using a structured query.
//
// Field coverage (parity baseline): Status, Priority (priority::<level>
// scoped label), Assignee, Labels, IssueType (mapped onto the type-label
// convention, since GitHub Issues have no native type), OrderBy, and
// Limit all map to native list params. RawQuery routes to the GitHub
// search API; FilterID is Jira-specific and ignored (documented no-op).
func (g *gitHub) Search(query SearchQuery) ([]Issue, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 30
	}
	if query.FilterID != "" {
		fmt.Fprintln(os.Stderr, "Note: GitHub ignores --filter (Jira saved-filter ID is Jira-only).")
	}

	// GitHub doesn't support raw JQL or filter IDs, but we can build a search query
	if query.RawQuery != "" {
		// Use GitHub search API with the raw query
		return g.searchIssues(query.RawQuery, limit)
	}

	// Build URL from field-level filters using the list endpoint
	state := "open"
	if query.Status != "" {
		switch strings.ToLower(query.Status) {
		case "closed", "done", "completed":
			state = "closed"
		case "all":
			state = "all"
		}
	}

	pageSize := min(limit, 100)
	listURL := fmt.Sprintf("%s/repos/%s/%s/issues?state=%s&per_page=%d", g.baseURL, g.owner, g.repo, state, pageSize)

	// Labels: user labels plus, when IssueType is set, the type-label
	// convention (Bug/epic/initiative), and when Priority is set the
	// priority::<level> scoped label.
	labels := append([]string{}, query.Labels...)
	if lbl := typeLabelFor(query.IssueType); lbl != "" {
		labels = append(labels, lbl)
	}
	if query.Priority != "" {
		labels = append(labels, "priority::"+strings.ToLower(query.Priority))
	}
	if len(labels) > 0 {
		listURL += "&labels=" + url.QueryEscape(strings.Join(labels, ","))
	}

	// Assignee
	if query.Assignee != "" {
		switch strings.ToLower(query.Assignee) {
		case "unassigned", "none", "empty":
			listURL += "&assignee=none"
		default:
			listURL += "&assignee=" + url.QueryEscape(query.Assignee)
		}
	}

	// Sort
	if query.OrderBy != "" {
		switch strings.ToLower(query.OrderBy) {
		case "created", "created desc", "created asc":
			listURL += "&sort=created"
		case "updated", "updated desc", "updated asc":
			listURL += "&sort=updated"
		case "comments", "comments desc", "comments asc":
			listURL += "&sort=comments"
		}
		if strings.HasSuffix(strings.ToLower(query.OrderBy), " asc") {
			listURL += "&direction=asc"
		}
	}

	return g.fetchIssueList(listURL, limit)
}

// searchIssues uses the GitHub search API for raw queries.
func (g *gitHub) searchIssues(rawQuery string, limit int) ([]Issue, error) {
	// Scope to this repo
	q := fmt.Sprintf("repo:%s/%s %s", g.owner, g.repo, rawQuery)
	searchURL := fmt.Sprintf("%s/search/issues", g.baseURL)

	type searchResult struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Number    int    `json:"number"`
			Title     string `json:"title"`
			State     string `json:"state"`
			URL       string `json:"html_url"`
			Body      string `json:"body"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
			User      struct {
				Login string `json:"login"`
			} `json:"user"`
			Assignee *struct {
				Login string `json:"login"`
			} `json:"assignee"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		} `json:"items"`
	}

	var issues []Issue
	page := 1
	totalCount := 0
	for len(issues) < limit {
		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		params := req.URL.Query()
		params.Set("q", q)
		params.Set("per_page", fmt.Sprintf("%d", min(limit-len(issues), 100)))
		params.Set("page", strconv.Itoa(page))
		req.URL.RawQuery = params.Encode()
		g.setHeaders(req)

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("searching issues: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(respBody))
		}
		var result searchResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		resp.Body.Close()
		totalCount = result.TotalCount
		for _, r := range result.Items {
			issue := Issue{
				ID: fmt.Sprintf("%d", r.Number), Title: r.Title, Status: r.State,
				URL: r.URL, Reporter: r.User.Login, CreatedAt: r.CreatedAt,
				UpdatedAt: r.UpdatedAt, Description: r.Body,
			}
			if r.Assignee != nil {
				issue.Assignee = r.Assignee.Login
			}
			for _, l := range r.Labels {
				issue.Labels = append(issue.Labels, l.Name)
			}
			issue.IssueType = githubIssueType(issue.Labels)
			issues = append(issues, issue)
			if len(issues) == limit {
				break
			}
		}
		if len(result.Items) == 0 || len(issues) >= totalCount {
			break
		}
		page++
	}
	if len(issues) == limit && totalCount > len(issues) {
		fmt.Fprintf(os.Stderr, "Warning: GitHub search reached its %d-issue limit and %d results remain; results are incomplete.\n", limit, totalCount-len(issues))
	}
	return issues, nil
}

// githubIssueType infers the hero spec type for a GitHub issue from its
// labels. GitHub Issues have no native type, so bug/epic/initiative ride
// labels (the same conventions the other adapters recognize via
// typeFromLabels). Falls through to "story" when no type label is
// present. Shared recognition keeps classification consistent
// tracker-wide.
func githubIssueType(labels []string) string {
	if t := typeFromLabels(labels); t != "" {
		return t
	}
	return "story"
}

// fetchIssueList fetches issues from a GitHub list endpoint URL.
func (g *gitHub) fetchIssueList(firstURL string, limit int) ([]Issue, error) {
	type issueResult struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		URL       string `json:"html_url"`
		Body      string `json:"body"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		User      *struct {
			Login string `json:"login"`
		} `json:"user"`
		Assignee *struct {
			Login string `json:"login"`
		} `json:"assignee"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	var issues []Issue
	next := firstURL
	for next != "" && len(issues) < limit {
		req, err := http.NewRequest("GET", next, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		g.setHeaders(req)
		resp, err := g.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("listing issues: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(respBody))
		}
		var results []issueResult
		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		nextLink := nextLink(resp.Header.Get("Link"))
		resp.Body.Close()
		remaining := limit - len(issues)
		truncatedCurrentPage := len(results) > remaining
		for _, r := range results {
			issue := Issue{
				ID: fmt.Sprintf("%d", r.Number), Title: r.Title, Status: r.State,
				URL: r.URL, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
				Description: r.Body,
			}
			if r.User != nil {
				issue.Reporter = r.User.Login
			}
			if r.Assignee != nil {
				issue.Assignee = r.Assignee.Login
			}
			for _, l := range r.Labels {
				issue.Labels = append(issue.Labels, l.Name)
			}
			issue.IssueType = githubIssueType(issue.Labels)
			issues = append(issues, issue)
			if len(issues) == limit {
				break
			}
		}
		if len(issues) == limit && (truncatedCurrentPage || nextLink != "") {
			fmt.Fprintf(os.Stderr, "Warning: GitHub list reached its %d-issue limit and more results remain; results are incomplete.\n", limit)
		}
		next = nextLink
	}
	return issues, nil
}
