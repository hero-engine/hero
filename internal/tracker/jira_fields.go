package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// jiraFieldEncode maps a canonical hero field name + Value to the Jira
// REST `fields` payload entry. Returns (jiraFieldID, jsonValue, ok).
// ok is false for fields this adapter doesn't know how to write — the
// caller skips them (forward-compat per the spec's vocabulary-mapping
// risk).
func (j *jira) jiraFieldEncode(canonical string, v Value) (string, interface{}, bool) {
	switch canonical {
	case "title":
		return "summary", v.Str, true
	case "description":
		// v1 keeps the body dumb — send plain text wrapped in a minimal
		// ADF document so the Jira Cloud API accepts it. (See spec
		// Boundaries: provider-native rendering is a follow-up.)
		return "description", textToADF(v.Str), true
	case "priority":
		return "priority", map[string]interface{}{"name": v.Str}, true
	case "labels":
		return "labels", v.Strings, true
	case "points":
		if v.Kind == ValueInt {
			return j.storyPointsField(), v.Int, true
		}
		return j.storyPointsField(), v.Str, true
	default:
		return "", nil, false
	}
}

// UpdateFields writes canonical fields to a Jira issue via PUT
// /rest/api/3/issue/<id>.
func (j *jira) UpdateFields(issueID string, fields map[string]Value) error {
	if len(fields) == 0 {
		return nil
	}
	payloadFields := map[string]interface{}{}
	for name, val := range fields {
		id, encoded, ok := j.jiraFieldEncode(name, val)
		if !ok {
			continue
		}
		payloadFields[id] = encoded
	}
	if len(payloadFields) == 0 {
		return nil
	}

	body, err := json.Marshal(map[string]interface{}{"fields": payloadFields})
	if err != nil {
		return fmt.Errorf("marshaling update: %w", err)
	}

	url := fmt.Sprintf("%s/rest/api/3/issue/%s", j.baseURL, issueID)
	resp, err := doWithRetry(j.client, func() (*http.Request, error) {
		req, rerr := http.NewRequest("PUT", url, bytes.NewReader(body))
		if rerr != nil {
			return nil, rerr
		}
		j.setHeaders(req)
		return req, nil
	}, nil)
	if err != nil {
		return fmt.Errorf("updating fields: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return classifyHTTPError("jira", resp.StatusCode, string(respBody))
	}
	return nil
}

// GetFields fetches the current canonical content-field values from a
// Jira issue (GET /rest/api/3/issue/<id>?fields=*all).
func (j *jira) GetFields(issueID string) (map[string]Value, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=*all", j.baseURL, issueID)
	resp, err := doWithRetry(j.client, func() (*http.Request, error) {
		req, rerr := http.NewRequest("GET", url, nil)
		if rerr != nil {
			return nil, rerr
		}
		j.setHeaders(req)
		return req, nil
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("getting fields: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, classifyHTTPError("jira", resp.StatusCode, string(respBody))
	}

	var raw struct {
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	out := map[string]Value{}
	if v, ok := raw.Fields["summary"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil && s != "" {
			out["title"] = StringValue(s)
		}
	}
	if v, ok := raw.Fields["description"]; ok {
		if s := adfToText(v); s != "" {
			out["description"] = StringValue(s)
		}
	}
	if v, ok := raw.Fields["priority"]; ok {
		var p struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(v, &p) == nil && p.Name != "" {
			out["priority"] = StringValue(p.Name)
		}
	}
	if v, ok := raw.Fields["labels"]; ok {
		var labels []string
		if json.Unmarshal(v, &labels) == nil && len(labels) > 0 {
			out["labels"] = StringsValue(labels)
		}
	}
	if v, ok := raw.Fields[j.storyPointsField()]; ok && string(v) != "null" {
		var n float64
		if json.Unmarshal(v, &n) == nil {
			out["points"] = IntValue(int(n))
		}
	}
	return out, nil
}

// adfToText extracts plain text from a Jira ADF (Atlassian Document
// Format) description or a plain string. Best-effort: it concatenates
// the text runs from the first paragraph layer. Used only for diff
// comparison, where we compare the local plain-text body against the
// tracker's rendered text. Returns "" if nothing extractable.
func adfToText(raw json.RawMessage) string {
	// Plain string description (Jira Server / older payloads).
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var doc struct {
		Content []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	var buf bytes.Buffer
	for i, block := range doc.Content {
		if i > 0 {
			buf.WriteString("\n")
		}
		for _, run := range block.Content {
			buf.WriteString(run.Text)
		}
	}
	return buf.String()
}
