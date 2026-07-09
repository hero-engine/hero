package tracker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// gitlabSprintLoader loads GitLab Iterations (Premium) as Hero sprints.
// Iterations are group-scoped, so the loader derives the group from the
// project's namespace (everything before the final path segment) and
// lists the project's issues filtered by iteration_id. On a tier that
// does not expose iterations (403/404) it returns a clear error rather
// than a confusing empty result (AC-9 / AC-8 degradation).
type gitlabSprintLoader struct {
	project string
	group   string
	token   string
	baseURL string
	client  *http.Client
}

func newGitLabSprintLoader(project, token, baseURL string) (*gitlabSprintLoader, error) {
	if project == "" {
		return nil, fmt.Errorf("gitlab project is required")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("gitlab requires base_url in tracker config")
	}
	return &gitlabSprintLoader{
		project: project,
		group:   gitlabGroupOf(project),
		token:   token,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// gitlabGroupOf returns the group path for a "namespace/.../project"
// identifier (everything before the final segment). Returns "" for a
// numeric project ID or a bare project with no namespace.
func gitlabGroupOf(project string) string {
	if idx := strings.LastIndex(project, "/"); idx > 0 {
		return project[:idx]
	}
	return ""
}

func (l *gitlabSprintLoader) apiURL(format string, a ...interface{}) string {
	return l.baseURL + "/api/v4" + fmt.Sprintf(format, a...)
}

func (l *gitlabSprintLoader) setHeaders(req *http.Request) {
	req.Header.Set("PRIVATE-TOKEN", l.token)
	req.Header.Set("Accept", "application/json")
}

func (l *gitlabSprintLoader) LoadSprint(sprintRef string) ([]SprintItem, *SprintInfo, error) {
	return l.LoadIteration(sprintRef)
}

func (l *gitlabSprintLoader) LoadActiveSprint(boardRef string) ([]SprintItem, *SprintInfo, error) {
	return l.LoadIteration("")
}

type gitLabIteration struct {
	ID        int    `json:"id"`
	IID       int    `json:"iid"`
	Title     string `json:"title"`
	State     int    `json:"state"` // 1=upcoming, 2=current, 3=closed
	StartDate string `json:"start_date"`
	DueDate   string `json:"due_date"`
}

// LoadIteration resolves an iteration by ID, title, or (when ref is
// empty) the current one, then returns its issues as SprintItems.
func (l *gitlabSprintLoader) LoadIteration(iterationRef string) ([]SprintItem, *SprintInfo, error) {
	if l.group == "" {
		return nil, nil, fmt.Errorf("gitlab iterations are group-scoped; project %q has no resolvable group (use namespace/project)", l.project)
	}

	iters, err := l.fetchIterations()
	if err != nil {
		return nil, nil, err
	}
	if len(iters) == 0 {
		return nil, nil, fmt.Errorf("no iterations found for group %q", l.group)
	}

	chosen := pickIteration(iters, iterationRef)
	info := &SprintInfo{
		ID:    strconv.Itoa(chosen.ID),
		Name:  chosen.Title,
		Start: chosen.StartDate,
		End:   chosen.DueDate,
		State: iterationState(chosen.State),
	}

	items, err := l.fetchIterationIssues(chosen.ID, chosen.Title)
	if err != nil {
		return nil, nil, err
	}
	return items, info, nil
}

// fetchIterations lists the group's iterations, degrading 403/404 (the
// tier does not expose iterations) into a clear error.
func (l *gitlabSprintLoader) fetchIterations() ([]gitLabIteration, error) {
	apiURL := l.apiURL("/groups/%s/iterations?per_page=100", url.PathEscape(l.group))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	l.setHeaders(req)
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing iterations: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("gitlab tier does not expose iterations for group %q (Premium feature)", l.group)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitlab API returned %d: %s", resp.StatusCode, string(body))
	}
	var iters []gitLabIteration
	if err := json.NewDecoder(resp.Body).Decode(&iters); err != nil {
		return nil, fmt.Errorf("decoding iterations: %w", err)
	}
	return iters, nil
}

// fetchIterationIssues lists the project's issues for one iteration.
func (l *gitlabSprintLoader) fetchIterationIssues(iterationID int, iterationTitle string) ([]SprintItem, error) {
	q := url.Values{}
	q.Set("iteration_id", strconv.Itoa(iterationID))
	q.Set("per_page", "100")
	apiURL := l.apiURL("/projects/%s/issues?%s", url.PathEscape(l.project), q.Encode())
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	l.setHeaders(req)
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing iteration issues: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitlab API returned %d: %s", resp.StatusCode, string(body))
	}
	var issues []gitLabIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("decoding issues: %w", err)
	}

	items := make([]SprintItem, 0, len(issues))
	for _, gi := range issues {
		item := SprintItem{
			ID:          strconv.Itoa(gi.IID),
			Title:       gi.Title,
			Description: gi.Description,
			Type:        heroIssueType(gi.IssueType, gi.Labels),
			Status:      gi.State,
			Priority:    priorityFromLabels(gi.Labels),
			Labels:      gi.Labels,
			SprintName:  iterationTitle,
			URL:         gi.WebURL,
		}
		if gi.Assignee != nil {
			item.Assignee = gi.Assignee.Username
		}
		if gi.Weight != nil {
			item.StoryPoints = float64(*gi.Weight)
		}
		if gi.Epic != nil {
			item.EpicID = strconv.Itoa(gi.Epic.IID)
			item.EpicTitle = gi.Epic.Title
		}
		items = append(items, item)
	}
	return items, nil
}

// pickIteration selects the iteration matching ref (by numeric ID or a
// case-insensitive title substring); an empty ref picks the current
// iteration (state 2), falling back to the first.
func pickIteration(iters []gitLabIteration, ref string) gitLabIteration {
	if ref == "" {
		for _, it := range iters {
			if it.State == 2 { // current
				return it
			}
		}
		return iters[0]
	}
	for _, it := range iters {
		if strconv.Itoa(it.ID) == ref {
			return it
		}
		if strings.Contains(strings.ToLower(it.Title), strings.ToLower(ref)) {
			return it
		}
	}
	return iters[0]
}

func iterationState(state int) string {
	switch state {
	case 1:
		return "future"
	case 2:
		return "active"
	case 3:
		return "closed"
	default:
		return ""
	}
}

var _ SprintLoader = (*gitlabSprintLoader)(nil)
