package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// UpdateFields writes canonical fields to a GitLab issue via
// PUT /api/v4/projects/:id/issues/:iid. title → title, description →
// description (Markdown round-trips). labels are merged with the
// existing set so non-hero labels survive (GitLab PUT replaces the whole
// labels string), exactly like github.go's mergeLabels. points → weight
// (Premium): warn-and-skip on 403 rather than failing the whole push.
// priority rotates the priority::<level> scoped label, read-then-write,
// the same shape as the size labels.
//
// Errors are classified via classifyHTTPError("gitlab", …) so the CLI
// maps 401/403 → auth and applies the 429 retry policy. An empty patch
// is a no-op.
func (g *gitLab) UpdateFields(issueID string, fields map[string]Value) error {
	if len(fields) == 0 {
		return nil
	}

	payload := map[string]interface{}{}
	var labelPatch *Value
	var priorityPatch *Value
	for name, val := range fields {
		switch name {
		case "title":
			payload["title"] = val.Str
		case "description":
			payload["description"] = val.Str
		case "labels":
			v := val
			labelPatch = &v
		case "points":
			payload["weight"] = val.Int
		case "priority":
			v := val
			priorityPatch = &v
		default:
			fmt.Fprintf(os.Stderr, "Warning: gitlab adapter does not support field %q; skipping\n", name)
		}
	}

	// labels and priority both rotate the label set; resolve them
	// together against one GET so we don't clobber each other.
	if labelPatch != nil || priorityPatch != nil {
		merged, err := g.mergeLabels(issueID, labelPatch, priorityPatch)
		if err != nil {
			return err
		}
		payload["labels"] = strings.Join(merged, ",")
	}

	if len(payload) == 0 {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling update: %w", err)
	}

	apiURL := g.apiURL("/projects/%s/issues/%s", g.projectPath(), issueID)
	resp, err := doWithRetry(g.client, func() (*http.Request, error) {
		req, rerr := http.NewRequest("PUT", apiURL, bytes.NewReader(body))
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

	if resp.StatusCode == http.StatusForbidden && payloadOnlyHasWeight(payload) {
		// Weight is Premium-only; degrade rather than failing the push.
		fmt.Fprintf(os.Stderr, "Warning: gitlab weight (points) requires a Premium tier; skipping\n")
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return classifyHTTPError("gitlab", resp.StatusCode, string(respBody))
	}
	return nil
}

// payloadOnlyHasWeight reports whether weight is the only field in the
// payload, so a 403 can be attributed to the Premium-weight degradation
// path rather than a credential problem.
func payloadOnlyHasWeight(payload map[string]interface{}) bool {
	if _, ok := payload["weight"]; !ok {
		return false
	}
	return len(payload) == 1
}

// mergeLabels fetches the issue's current labels and produces the
// intended label set: every existing label is preserved (push is
// additive, never destructive — matching github.go), the hero-supplied
// labels are unioned in, and when priorityPatch is set the existing
// priority::* labels are rotated out and the new one rotated in.
func (g *gitLab) mergeLabels(issueID string, labelPatch, priorityPatch *Value) ([]string, error) {
	current, err := g.getRawIssue(issueID)
	if err != nil {
		return nil, err
	}

	rotatingPriority := priorityPatch != nil
	seen := map[string]bool{}
	merged := make([]string, 0, len(current.Labels)+2)
	for _, l := range current.Labels {
		if rotatingPriority && strings.HasPrefix(strings.ToLower(l), "priority::") {
			continue // rotated out, replaced below
		}
		if !seen[l] {
			seen[l] = true
			merged = append(merged, l)
		}
	}
	if labelPatch != nil {
		for _, l := range labelPatch.Strings {
			if !seen[l] {
				seen[l] = true
				merged = append(merged, l)
			}
		}
	}
	if rotatingPriority && priorityPatch.Str != "" {
		newLabel := "priority::" + priorityPatch.Str
		if !seen[newLabel] {
			merged = append(merged, newLabel)
		}
	}
	return merged, nil
}

// GetFields fetches the current canonical content-field values from a
// GitLab issue. v1 reads title, description, and labels — the fields
// UpdateFields can write — so the diff path round-trips.
func (g *gitLab) GetFields(issueID string) (map[string]Value, error) {
	apiURL := g.apiURL("/projects/%s/issues/%s", g.projectPath(), issueID)
	resp, err := doWithRetry(g.client, func() (*http.Request, error) {
		req, rerr := http.NewRequest("GET", apiURL, nil)
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
		return nil, classifyHTTPError("gitlab", resp.StatusCode, string(respBody))
	}

	var result gitLabIssue
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	out := map[string]Value{}
	if result.Title != "" {
		out["title"] = StringValue(result.Title)
	}
	if result.Description != "" {
		out["description"] = StringValue(result.Description)
	}
	if len(result.Labels) > 0 {
		out["labels"] = StringsValue(result.Labels)
	}
	if result.Weight != nil {
		out["points"] = IntValue(*result.Weight)
	}
	return out, nil
}
