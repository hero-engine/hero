package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// brokerSearchPage is deliberately separate from GitHub's configured-repo
// Search path. It sends the caller's native query byte-for-byte without adding
// repo:<configured>.
func (g *gitHub) brokerSearchPage(ctx context.Context, query string, limit int, cursor string) ([]Issue, string, error) {
	page := 1
	if cursor != "" {
		var err error
		page, err = strconv.Atoi(cursor)
		if err != nil || page < 1 {
			return nil, "", fmt.Errorf("invalid GitHub page cursor")
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"/search/issues", nil)
	if err != nil {
		return nil, "", err
	}
	values := req.URL.Query()
	values.Set("q", query)
	values.Set("per_page", strconv.Itoa(limit))
	values.Set("page", strconv.Itoa(page))
	req.URL.RawQuery = values.Encode()
	g.setHeaders(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, defaultBrokerOutputLimit))
		return nil, "", fmt.Errorf("github API returned %d: %s", resp.StatusCode, body)
	}
	var result struct {
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
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}
	issues := make([]Issue, 0, len(result.Items))
	for _, raw := range result.Items {
		issue := Issue{ID: strconv.Itoa(raw.Number), Title: raw.Title, Status: raw.State, URL: raw.URL, Reporter: raw.User.Login, CreatedAt: raw.CreatedAt, UpdatedAt: raw.UpdatedAt, Description: raw.Body}
		if raw.Assignee != nil {
			issue.Assignee = raw.Assignee.Login
		}
		for _, label := range raw.Labels {
			issue.Labels = append(issue.Labels, label.Name)
		}
		issue.IssueType = githubIssueType(issue.Labels)
		issues = append(issues, issue)
	}
	next := ""
	if page*limit < result.TotalCount && len(result.Items) > 0 {
		next = strconv.Itoa(page + 1)
	}
	return issues, next, nil
}

// brokerSearchPage uses GitLab's instance-wide issues endpoint rather than the
// configured project endpoint. query is passed unchanged as the native search
// value; the opaque outer cursor carries GitLab's numeric page.
func (g *gitLab) brokerSearchPage(ctx context.Context, query string, limit int, cursor string) ([]Issue, string, error) {
	page := "1"
	if cursor != "" {
		if n, err := strconv.Atoi(cursor); err != nil || n < 1 {
			return nil, "", fmt.Errorf("invalid GitLab page cursor")
		}
		page = cursor
	}
	values := url.Values{}
	values.Set("scope", "all")
	values.Set("search", query)
	values.Set("per_page", strconv.Itoa(limit))
	values.Set("page", page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.apiURL("/issues")+"?"+values.Encode(), nil)
	if err != nil {
		return nil, "", err
	}
	g.setHeaders(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, defaultBrokerOutputLimit))
		return nil, "", fmt.Errorf("gitlab API returned %d: %s", resp.StatusCode, body)
	}
	var raw []gitLabIssue
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, "", err
	}
	issues := make([]Issue, 0, len(raw))
	for _, issue := range raw {
		issues = append(issues, g.toIssue(issue))
	}
	return issues, resp.Header.Get("X-Next-Page"), nil
}

// brokerSearchPage uses Linear's native issueSearch connection. Unlike the
// older structured Search adapter it does not discard RawQuery.
func (l *linear) brokerSearchPage(ctx context.Context, query string, limit int, cursor string) ([]Issue, string, error) {
	graphql := `query BrokerIssueSearch($query: String!, $first: Int!, $after: String) {
		searchIssues(term: $query, first: $first, after: $after) {
			nodes { id identifier title description priority createdAt updatedAt url
				state { name } assignee { displayName } creator { displayName } labels { nodes { name } }
			}
			pageInfo { hasNextPage endCursor }
		}
	}`
	variables := map[string]interface{}{"query": query, "first": limit}
	if cursor != "" {
		variables["after"] = cursor
	}
	payload, _ := json.Marshal(map[string]interface{}{"query": graphql, "variables": variables})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", l.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, defaultBrokerOutputLimit))
		return nil, "", fmt.Errorf("linear API returned %d: %s", resp.StatusCode, body)
	}
	var result struct {
		Data struct {
			SearchIssues struct {
				Nodes    []interface{} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"searchIssues"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}
	if len(result.Errors) > 0 {
		return nil, "", fmt.Errorf("linear API: %s", result.Errors[0].Message)
	}
	issues := l.parseIssueNodes(result.Data.SearchIssues.Nodes)
	next := ""
	if result.Data.SearchIssues.PageInfo.HasNextPage {
		next = result.Data.SearchIssues.PageInfo.EndCursor
	}
	return issues, next, nil
}
