package environment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const githubAPIBase = "https://api.github.com"

type githubActions struct {
	owner  string
	repo   string
	token  string
	client *http.Client
}

func newGitHubActions(project, token string) (*githubActions, error) {
	parts := strings.SplitN(project, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("github project must be in owner/repo format, got %q", project)
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("GitHub token required — set tracker.token_env or GITHUB_TOKEN")
	}
	return &githubActions{
		owner:  parts[0],
		repo:   parts[1],
		token:  token,
		client: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (g *githubActions) Name() string { return "github-actions" }

func (g *githubActions) PipelineStatus(branch string) (*PipelineStatus, error) {
	// Get the most recent workflow run for the branch
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs?branch=%s&per_page=1",
		githubAPIBase, g.owner, g.repo, branch)

	body, err := g.get(url)
	if err != nil {
		return nil, fmt.Errorf("querying GitHub Actions: %w", err)
	}

	var resp struct {
		TotalCount int `json:"total_count"`
		Runs       []struct {
			ID         int64     `json:"id"`
			Status     string    `json:"status"`      // queued, in_progress, completed
			Conclusion string    `json:"conclusion"`   // success, failure, cancelled
			HTMLURL    string    `json:"html_url"`
			HeadSHA    string    `json:"head_sha"`
			UpdatedAt  time.Time `json:"updated_at"`
			RunStarted time.Time `json:"run_started_at"`
		} `json:"workflow_runs"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing GitHub Actions response: %w", err)
	}

	if resp.TotalCount == 0 || len(resp.Runs) == 0 {
		return &PipelineStatus{
			Provider: "github-actions",
			Branch:   branch,
			Status:   "unknown",
		}, nil
	}

	run := resp.Runs[0]
	ps := &PipelineStatus{
		Provider:   "github-actions",
		Branch:     branch,
		CommitSHA:  run.HeadSHA[:8],
		RunURL:     run.HTMLURL,
		UpdatedAt:  run.UpdatedAt,
	}

	// Map GitHub statuses to our simplified model
	switch run.Status {
	case "completed":
		switch run.Conclusion {
		case "success":
			ps.Status = "passing"
			ps.Conclusion = "success"
		case "failure":
			ps.Status = "failing"
			ps.Conclusion = "failure"
			// Try to get the failed job/step
			g.enrichFailedStep(ps, run.ID)
		default:
			ps.Status = "failing"
			ps.Conclusion = run.Conclusion
		}
	case "in_progress":
		ps.Status = "running"
	case "queued":
		ps.Status = "running"
	default:
		ps.Status = "unknown"
	}

	if !run.RunStarted.IsZero() && run.Status == "completed" {
		duration := run.UpdatedAt.Sub(run.RunStarted)
		ps.Duration = formatDuration(duration)
	}

	return ps, nil
}

// enrichFailedStep queries the jobs endpoint to find which step failed.
func (g *githubActions) enrichFailedStep(ps *PipelineStatus, runID int64) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%d/jobs?per_page=10",
		githubAPIBase, g.owner, g.repo, runID)

	body, err := g.get(url)
	if err != nil {
		return
	}

	var resp struct {
		Jobs []struct {
			Name       string `json:"name"`
			Conclusion string `json:"conclusion"`
			Steps      []struct {
				Name       string `json:"name"`
				Conclusion string `json:"conclusion"`
			} `json:"steps"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return
	}

	for _, job := range resp.Jobs {
		if job.Conclusion == "failure" {
			for _, step := range job.Steps {
				if step.Conclusion == "failure" {
					ps.FailedStep = fmt.Sprintf("%s / %s", job.Name, step.Name)
					return
				}
			}
			ps.FailedStep = job.Name
			return
		}
	}
}

func (g *githubActions) get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
	}

	return body, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
