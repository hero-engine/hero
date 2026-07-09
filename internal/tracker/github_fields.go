package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// UpdateFields writes canonical fields to a GitHub issue via PATCH
// /repos/{owner}/{repo}/issues/{number}. title → title, description →
// body. labels are merged with the existing set so non-hero labels are
// preserved (GitHub PATCH replaces the whole label array). points and
// priority live on GitHub Projects v2 (a separate GraphQL surface) and
// are skipped with a warning in v1 (forward-compat, per the spec's
// vocabulary-mapping risk note).
func (g *gitHub) UpdateFields(issueID string, fields map[string]Value) error {
	if len(fields) == 0 {
		return nil
	}

	payload := map[string]interface{}{}
	var labelPatch *Value
	for name, val := range fields {
		switch name {
		case "title":
			payload["title"] = val.Str
		case "description":
			payload["body"] = val.Str
		case "labels":
			v := val
			labelPatch = &v
		default:
			fmt.Fprintf(os.Stderr, "Warning: github adapter does not support field %q; skipping\n", name)
		}
	}

	if labelPatch != nil {
		merged, err := g.mergeLabels(issueID, labelPatch.Strings)
		if err != nil {
			return err
		}
		payload["labels"] = merged
	}

	if len(payload) == 0 {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling update: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", g.baseURL, g.owner, g.repo, issueID)
	resp, err := doWithRetry(g.client, func() (*http.Request, error) {
		req, rerr := http.NewRequest("PATCH", url, bytes.NewReader(body))
		if rerr != nil {
			return nil, rerr
		}
		g.setHeaders(req)
		return req, nil
	}, nil)
	if err != nil {
		return fmt.Errorf("updating fields: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return classifyHTTPError("github", resp.StatusCode, string(respBody))
	}
	return nil
}

// mergeLabels fetches the issue's current labels and merges the
// hero-supplied set on top, preserving any label not in the supplied
// set's namespace. Because hero pushes the full intended label set, we
// keep labels GitHub/other tooling owns (anything starting with a
// non-hero prefix is preserved) while replacing the hero-managed ones.
// For simplicity in v1 we preserve every existing label and union in
// the new ones — push is additive, never destructive (the spec's
// last-write-wins applies to scalar fields, not label removal).
func (g *gitHub) mergeLabels(issueID string, want []string) ([]string, error) {
	getURL := fmt.Sprintf("%s/repos/%s/%s/issues/%s", g.baseURL, g.owner, g.repo, issueID)
	req, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating GET request: %w", err)
	}
	g.setHeaders(req)
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching issue labels: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, classifyHTTPError("github", resp.StatusCode, string(respBody))
	}
	var current struct {
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&current); err != nil {
		return nil, fmt.Errorf("decoding labels: %w", err)
	}

	seen := map[string]bool{}
	merged := make([]string, 0, len(current.Labels)+len(want))
	for _, l := range current.Labels {
		if !seen[l.Name] {
			seen[l.Name] = true
			merged = append(merged, l.Name)
		}
	}
	for _, l := range want {
		if !seen[l] {
			seen[l] = true
			merged = append(merged, l)
		}
	}
	return merged, nil
}

// GetFields fetches the current canonical content-field values from a
// GitHub issue. v1 reads title, body (→ description), and labels — the
// fields UpdateFields can write — so the diff path round-trips.
func (g *gitHub) GetFields(issueID string) (map[string]Value, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", g.baseURL, g.owner, g.repo, issueID)
	resp, err := doWithRetry(g.client, func() (*http.Request, error) {
		req, rerr := http.NewRequest("GET", url, nil)
		if rerr != nil {
			return nil, rerr
		}
		g.setHeaders(req)
		return req, nil
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("getting fields: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, classifyHTTPError("github", resp.StatusCode, string(respBody))
	}

	var result struct {
		Title  string `json:"title"`
		Body   string `json:"body"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	out := map[string]Value{}
	if result.Title != "" {
		out["title"] = StringValue(result.Title)
	}
	if result.Body != "" {
		out["description"] = StringValue(result.Body)
	}
	if len(result.Labels) > 0 {
		labels := make([]string, 0, len(result.Labels))
		for _, l := range result.Labels {
			labels = append(labels, l.Name)
		}
		out["labels"] = StringsValue(labels)
	}
	return out, nil
}
