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

const defaultGitLabBaseURL = "https://gitlab.com"

// gitLab implements the Tracker interface for GitLab Issues (REST API v4).
//
// It mirrors github.go structurally: both speak REST, both have flat
// issue + label shapes, and both address work through a namespace +
// project (GitHub owner/repo, GitLab namespace/project or a numeric
// project ID). Where GitLab's model is richer (Epics, Iterations) the
// adapter leans on the existing convention — tracker-specific values
// ride the gitlab_* frontmatter namespace, parent linkage rides the
// embedded epic/milestone/iteration references on each issue.
//
// REST-only by decision (see the spec's open question): GitLab's GraphQL
// surface is more uniform, but mixing REST + GraphQL in one adapter
// complicates retries and pagination for no v1 benefit.
//
// Stable external ID. The agreed cross-repo idempotency key (matching
// the Swift importer's GitLabImportShape) is gitlab:<project>:<iid>. The
// Go adapter addresses issues by their per-project IID over REST, so
// Issue.ID carries the IID (the github-parallel native id); the global
// gitlab_id and the web gitlab_url ride CustomFields for the import
// writer. GitLab's own #/& global-reference notation is display-only.
type gitLab struct {
	project string // "namespace/project" or numeric project ID
	token   string
	baseURL string // instance root, e.g. https://gitlab.com (api/v4 is appended)
	client  *http.Client
	// configuredSizeMapping is the workspace-configured size mapping
	// (hero.json: tracker.size_mapping). Nil → use the shipped default
	// (defaultGitLabSizeMapping). See internal/tracker/size_mapping.go.
	configuredSizeMapping *config.SizeMappingConfig
}

// newGitLab builds a GitLab adapter. baseURL is required and rejected
// when empty (AC-1) — GitLab self-hosted instances each have their own
// host, and `hero connect gitlab` always persists one (defaulting to
// https://gitlab.com), so a missing base_url is a misconfiguration, not
// a gitlab.com shortcut.
func newGitLab(project, token, baseURL string) (*gitLab, error) {
	if project == "" {
		return nil, fmt.Errorf("gitlab project is required (namespace/project or numeric ID)")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("gitlab requires base_url in tracker config (e.g. %s) — run `hero connect gitlab`", defaultGitLabBaseURL)
	}
	return &gitLab{
		project: project,
		token:   token,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (g *gitLab) Name() string { return "gitlab" }

// SupportsHierarchy: GitLab models hierarchy through group Epics and
// Iterations (Premium), so the spec-sizing nudge treats it like jira /
// linear rather than flat github.
func (g *gitLab) SupportsHierarchy() bool { return true }

// apiURL builds an api/v4 URL with the project path already escaped.
// project may be "group/sub/project" (slash-encoded) or a numeric ID.
func (g *gitLab) apiURL(format string, a ...interface{}) string {
	return g.baseURL + "/api/v4" + fmt.Sprintf(format, a...)
}

// projectPath returns the URL-escaped project identifier for a path
// segment. GitLab accepts "group%2Fproject" or a numeric ID verbatim.
func (g *gitLab) projectPath() string {
	return url.PathEscape(g.project)
}

func (g *gitLab) setHeaders(req *http.Request) {
	req.Header.Set("PRIVATE-TOKEN", g.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}

// gitLabIssue is the wire shape of a GitLab issue (the subset hero reads).
type gitLabIssue struct {
	ID          int      `json:"id"`  // global ID
	IID         int      `json:"iid"` // per-project ID (addresses REST routes)
	Title       string   `json:"title"`
	Description string   `json:"description"`
	State       string   `json:"state"` // "opened" | "closed"
	WebURL      string   `json:"web_url"`
	Labels      []string `json:"labels"`
	IssueType   string   `json:"issue_type"` // issue | incident | test_case | task
	Weight      *int     `json:"weight"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Assignee    *struct {
		Username string `json:"username"`
	} `json:"assignee"`
	Author *struct {
		Username string `json:"username"`
	} `json:"author"`
	Epic *struct {
		ID    int    `json:"id"`
		IID   int    `json:"iid"`
		Title string `json:"title"`
	} `json:"epic"`
	Milestone *struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	} `json:"milestone"`
	Iteration *struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	} `json:"iteration"`
}

// toIssue projects a GitLab issue into the tracker-neutral Issue. The
// IID is the native id (github-parallel); the global id and web url ride
// CustomFields for the import frontmatter writer.
func (g *gitLab) toIssue(gi gitLabIssue) Issue {
	issue := Issue{
		ID:           strconv.Itoa(gi.IID),
		Title:        gi.Title,
		Status:       gi.State,
		URL:          gi.WebURL,
		Labels:       gi.Labels,
		IssueType:    heroIssueType(gi.IssueType, gi.Labels),
		CreatedAt:    gi.CreatedAt,
		UpdatedAt:    gi.UpdatedAt,
		Description:  gi.Description,
		Priority:     priorityFromLabels(gi.Labels),
		CustomFields: map[string]string{},
	}
	issue.CustomFields["gitlab_id"] = strconv.Itoa(gi.ID)
	issue.CustomFields["gitlab_iid"] = strconv.Itoa(gi.IID)
	if gi.WebURL != "" {
		issue.CustomFields["gitlab_url"] = gi.WebURL
	}
	if gi.Weight != nil {
		issue.CustomFields["weight"] = strconv.Itoa(*gi.Weight)
	}
	if gi.Assignee != nil {
		issue.Assignee = gi.Assignee.Username
	}
	if gi.Author != nil {
		issue.Reporter = gi.Author.Username
	}
	if gi.Epic != nil {
		issue.EpicKey = strconv.Itoa(gi.Epic.IID)
		issue.CustomFields["gitlab_epic_id"] = strconv.Itoa(gi.Epic.ID)
	}
	if gi.Milestone != nil {
		issue.CustomFields["gitlab_milestone"] = gi.Milestone.Title
	}
	if gi.Iteration != nil {
		issue.SprintName = gi.Iteration.Title
		issue.CustomFields["gitlab_iteration"] = gi.Iteration.Title
	}
	return issue
}

// heroIssueType maps GitLab's issue_type onto a string inferSpecType
// recognizes. incident → bug; test_case → task; otherwise the label
// conventions decide (bug/epic/initiative via typeFromLabels, shared
// with the other adapters for consistent classification). Falls through
// to story when no type signal is present.
func heroIssueType(gitlabType string, labels []string) string {
	switch strings.ToLower(gitlabType) {
	case "incident":
		return "bug"
	case "test_case":
		return "task"
	}
	if t := typeFromLabels(labels); t != "" {
		return t
	}
	return "story"
}

// priorityFromLabels extracts a priority::<level> scoped label, GitLab's
// conventional way to carry priority. Returns "" when none is present.
func priorityFromLabels(labels []string) string {
	for _, l := range labels {
		if strings.HasPrefix(strings.ToLower(l), "priority::") {
			return strings.TrimPrefix(l, "priority::")
		}
	}
	return ""
}

// CreateIssue creates a GitLab issue from a spec. Returns the IID.
func (g *gitLab) CreateIssue(s *spec.Spec) (string, error) {
	labels := []string{
		fmt.Sprintf("hero::%s", s.Type),
		fmt.Sprintf("hero::%s", StatusLabel(s.Status)),
	}
	if s.Size != "" {
		if v, err := g.MapSize(s.Size); err == nil && v != "" {
			labels = append(labels, v)
		}
	}
	payload := map[string]interface{}{
		"title":       fmt.Sprintf("[%s] %s", s.Type, s.Title),
		"description": IssueBody(s),
		"labels":      strings.Join(labels, ","),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling issue: %w", err)
	}

	apiURL := g.apiURL("/projects/%s/issues", g.projectPath())
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
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
		return "", fmt.Errorf("gitlab API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result gitLabIssue
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	return strconv.Itoa(result.IID), nil
}

// UpdateStatus posts a status note and closes/reopens the issue to track
// spec lifecycle. Completed/superseded → close; anything else reopens
// (state_event=reopen) so a re-opened spec re-opens the issue (AC-5).
func (g *gitLab) UpdateStatus(issueID string, status spec.Status) error {
	note := fmt.Sprintf("**Hero status update:** %s", StatusLabel(status))
	if err := g.AddComment(issueID, note); err != nil {
		return err
	}
	if status == spec.StatusCompleted || status == spec.StatusSuperseded {
		return g.setIssueState(issueID, "close")
	}
	return g.setIssueState(issueID, "reopen")
}

// setIssueState issues PUT state_event=close|reopen.
func (g *gitLab) setIssueState(issueID, event string) error {
	payload := map[string]string{"state_event": event}
	body, _ := json.Marshal(payload)
	apiURL := g.apiURL("/projects/%s/issues/%s", g.projectPath(), issueID)
	req, err := http.NewRequest("PUT", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	g.setHeaders(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("updating issue state: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gitlab API returned %d when updating state", resp.StatusCode)
	}
	return nil
}

// UpdateSize rotates the scoped size label (workflow::size/<tier>) on a
// GitLab issue, read-then-write so non-size labels are preserved.
// Mirrors github.go's UpdateSize. When the configured mapping targets a
// numeric field (weight), the label rotation is skipped and the weight
// is PUT directly.
func (g *gitLab) UpdateSize(issueID, localTier string) error {
	newValue, err := g.MapSize(localTier)
	if err != nil {
		return fmt.Errorf("mapping size %q: %w", localTier, err)
	}
	if newValue == "" {
		return fmt.Errorf("size %q maps to empty value", localTier)
	}
	mapping := g.sizeMapping()

	// Numeric weight field: PUT weight directly.
	if !isLabelPrefixField(mapping.Field) {
		w, convErr := strconv.Atoi(newValue)
		if convErr != nil {
			return fmt.Errorf("size %q maps to non-integer weight %q", localTier, newValue)
		}
		return g.putFields(issueID, map[string]interface{}{"weight": w})
	}

	// Label-prefix field: read current labels, strip the size prefix,
	// append the new label, PUT the merged comma-separated set.
	current, err := g.getRawIssue(issueID)
	if err != nil {
		return err
	}
	prefix := mapping.Field
	merged := make([]string, 0, len(current.Labels)+1)
	for _, l := range current.Labels {
		if !strings.HasPrefix(l, prefix) {
			merged = append(merged, l)
		}
	}
	merged = append(merged, newValue)
	return g.putFields(issueID, map[string]interface{}{"labels": strings.Join(merged, ",")})
}

// putFields PUTs a raw field map to an issue (shared by UpdateSize and
// the field-level write-back).
func (g *gitLab) putFields(issueID string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling update: %w", err)
	}
	apiURL := g.apiURL("/projects/%s/issues/%s", g.projectPath(), issueID)
	req, err := http.NewRequest("PUT", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	g.setHeaders(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("updating fields: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return classifyHTTPError("gitlab", resp.StatusCode, string(respBody))
	}
	return nil
}

// getRawIssue GETs a single issue in its native wire shape.
func (g *gitLab) getRawIssue(issueID string) (*gitLabIssue, error) {
	apiURL := g.apiURL("/projects/%s/issues/%s", g.projectPath(), issueID)
	req, err := http.NewRequest("GET", apiURL, nil)
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
		return nil, classifyHTTPError("gitlab", resp.StatusCode, string(respBody))
	}
	var gi gitLabIssue
	if err := json.NewDecoder(resp.Body).Decode(&gi); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &gi, nil
}

// GetIssue retrieves issue info from GitLab.
func (g *gitLab) GetIssue(issueID string) (*Issue, error) {
	gi, err := g.getRawIssue(issueID)
	if err != nil {
		return nil, err
	}
	issue := g.toIssue(*gi)
	return &issue, nil
}

// AddComment posts a note to a GitLab issue.
func (g *gitLab) AddComment(issueID, body string) error {
	apiURL := g.apiURL("/projects/%s/issues/%s/notes", g.projectPath(), issueID)
	payload, _ := json.Marshal(map[string]string{"body": body})
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	g.setHeaders(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("posting note: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("gitlab note API returned %d", resp.StatusCode)
	}
	return nil
}

// AttachFile posts file contents as a note (the issues API has no direct
// attachment endpoint we depend on), matching github.go's behavior.
func (g *gitLab) AttachFile(issueID, filePath, fileName string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	body := fmt.Sprintf("**Attached: %s**\n\n<details>\n<summary>Click to expand</summary>\n\n```\n%s\n```\n\n</details>", fileName, string(content))
	return g.AddComment(issueID, body)
}

// ListIssues fetches open issues, following Link rel="next" pagination
// to the cap. Optionally filters by a single label.
func (g *gitLab) ListIssues(label string, limit int) ([]Issue, error) {
	if limit <= 0 {
		limit = 30
	}
	q := url.Values{}
	q.Set("state", "opened")
	q.Set("per_page", "100")
	if label != "" {
		q.Set("labels", label)
	}
	base := g.apiURL("/projects/%s/issues", g.projectPath())
	return g.fetchIssuePages(base+"?"+q.Encode(), limit)
}

// Search fetches issues with a structured query. GitLab has no JQL; we
// translate the field-level filters onto issue list parameters.
//
// Field coverage (parity baseline): Status, Priority (priority::<level>
// scoped label), Assignee, Labels, IssueType (mapped onto the type-label
// convention, since GitLab's native issue_type only covers
// issue/incident/task), OrderBy, and Limit all map to native list
// params. RawQuery maps to the free-text `search` param; FilterID is
// Jira-specific and ignored (documented no-op).
func (g *gitLab) Search(query SearchQuery) ([]Issue, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 30
	}
	if query.FilterID != "" {
		fmt.Fprintln(os.Stderr, "Note: GitLab ignores --filter (Jira saved-filter ID is Jira-only).")
	}
	q := url.Values{}
	q.Set("per_page", "100")

	state := "opened"
	if query.Status != "" {
		switch strings.ToLower(query.Status) {
		case "closed", "done", "completed":
			state = "closed"
		case "all":
			state = "all"
		}
	}
	if state != "all" {
		q.Set("state", state)
	}
	// Labels: user labels plus, when IssueType is set, the type-label
	// convention (Bug/epic/initiative) and, when Priority is set, the
	// priority::<level> scoped label GitLab conventionally carries.
	labels := append([]string{}, query.Labels...)
	if lbl := typeLabelFor(query.IssueType); lbl != "" {
		labels = append(labels, lbl)
	}
	if query.Priority != "" {
		labels = append(labels, "priority::"+strings.ToLower(query.Priority))
	}
	if len(labels) > 0 {
		q.Set("labels", strings.Join(labels, ","))
	}
	if query.Assignee != "" {
		switch strings.ToLower(query.Assignee) {
		case "unassigned", "none", "empty":
			q.Set("assignee_id", "None")
		default:
			q.Set("assignee_username", query.Assignee)
		}
	}
	if query.RawQuery != "" {
		q.Set("search", query.RawQuery)
	}
	if query.OrderBy != "" {
		switch strings.ToLower(query.OrderBy) {
		case "created", "created desc", "created asc":
			q.Set("order_by", "created_at")
		case "updated", "updated desc", "updated asc":
			q.Set("order_by", "updated_at")
		}
		if strings.HasSuffix(strings.ToLower(query.OrderBy), " asc") {
			q.Set("sort", "asc")
		}
	}

	base := g.apiURL("/projects/%s/issues", g.projectPath())
	return g.fetchIssuePages(base+"?"+q.Encode(), limit)
}

// fetchIssuePages walks GitLab Link rel="next" pagination starting at
// firstURL, stopping at limit items or when no next link remains. Some
// self-managed instances omit the Link header — when page 1 returns no
// Link, we treat it as a single page (documented fallback).
func (g *gitLab) fetchIssuePages(firstURL string, limit int) ([]Issue, error) {
	var out []Issue
	next := firstURL
	for next != "" {
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
			return nil, fmt.Errorf("gitlab API returned %d: %s", resp.StatusCode, string(respBody))
		}
		var page []gitLabIssue
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		linkNext := nextLink(resp.Header.Get("Link"))
		resp.Body.Close()

		for i, gi := range page {
			out = append(out, g.toIssue(gi))
			if limit > 0 && len(out) >= limit {
				if linkNext != "" || i+1 < len(page) {
					fmt.Fprintf(os.Stderr, "Warning: GitLab query reached its %d-issue limit and more results remain; results are incomplete.\n", limit)
				}
				return out, nil
			}
		}
		next = linkNext
	}
	return out, nil
}

// nextLink parses an RFC 5988 Link header and returns the URL whose
// rel="next", or "" when absent.
func nextLink(header string) string {
	if header == "" {
		return ""
	}
	for _, part := range strings.Split(header, ",") {
		segs := strings.Split(strings.TrimSpace(part), ";")
		if len(segs) < 2 {
			continue
		}
		rawURL := strings.TrimSpace(segs[0])
		rawURL = strings.TrimPrefix(rawURL, "<")
		rawURL = strings.TrimSuffix(rawURL, ">")
		for _, attr := range segs[1:] {
			attr = strings.TrimSpace(attr)
			if attr == `rel="next"` || attr == "rel=next" {
				return rawURL
			}
		}
	}
	return ""
}

// gitLabEpic is the wire shape of a group Epic (Premium).
type gitLabEpic struct {
	ID       int    `json:"id"`
	IID      int    `json:"iid"`
	Title    string `json:"title"`
	State    string `json:"state"`
	WebURL   string `json:"web_url"`
	ParentID *int   `json:"parent_id"`
}

// ListEpics fetches a group's epics, projecting them into Issues typed
// "epic"/"initiative" for the import path. Epics are a GitLab Premium
// feature: on 403/404 (open-source or unlicensed instance) it returns
// (nil, false, nil) so the caller can emit ONE notice line and continue
// without them (AC-8). available is false only on the tier-degradation
// path; real errors are returned as errors.
func (g *gitLab) ListEpics(group string) (epics []Issue, available bool, err error) {
	if group == "" {
		return nil, false, nil
	}
	apiURL := g.apiURL("/groups/%s/epics?per_page=100", url.PathEscape(group))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("creating request: %w", err)
	}
	g.setHeaders(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("listing epics: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return nil, false, nil // premium tier does not expose epics
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("gitlab API returned %d: %s", resp.StatusCode, string(respBody))
	}
	var page []gitLabEpic
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, false, fmt.Errorf("decoding response: %w", err)
	}
	for _, e := range page {
		specType := "initiative"
		if e.ParentID != nil {
			specType = "epic"
		}
		epics = append(epics, Issue{
			ID:        strconv.Itoa(e.IID),
			Title:     e.Title,
			Status:    e.State,
			URL:       e.WebURL,
			IssueType: specType,
			CustomFields: map[string]string{
				"gitlab_epic_id": strconv.Itoa(e.ID),
			},
		})
	}
	return epics, true, nil
}

var _ Tracker = (*gitLab)(nil)
