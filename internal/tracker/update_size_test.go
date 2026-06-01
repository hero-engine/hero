package tracker

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

// --- Jira UpdateSize ---

func TestJiraUpdateSize_CleanPush_EmitsExpectedPayload(t *testing.T) {
	var gotPath, gotMethod string
	var gotFields map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if f, ok := payload["fields"].(map[string]interface{}); ok {
			gotFields = f
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "tok", "user@example.com", srv.URL)
	if err := j.UpdateSize("PROJ-123", "large"); err != nil {
		t.Fatalf("UpdateSize: %v", err)
	}

	if gotMethod != "PUT" {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/rest/api/3/issue/PROJ-123") {
		t.Errorf("path = %q, want suffix /rest/api/3/issue/PROJ-123", gotPath)
	}
	// Default story-points field is customfield_10016; large → 8 (lower band).
	v, ok := gotFields["customfield_10016"]
	if !ok {
		t.Fatalf("fields missing customfield_10016; got keys=%v", keysOf(gotFields))
	}
	n, ok := v.(float64)
	if !ok {
		t.Fatalf("customfield_10016 = %v (%T); want numeric", v, v)
	}
	if n != 8 {
		t.Errorf("customfield_10016 = %v; want 8 (large lower band)", n)
	}
}

func TestJiraUpdateSize_PropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"errorMessages":["boom"]}`)
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "tok", "user@example.com", srv.URL)
	err := j.UpdateSize("PROJ-123", "medium")
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want to mention status code", err.Error())
	}
}

func TestJiraUpdateSize_NoMappingForTier_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be called when mapping fails")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "tok", "user@example.com", srv.URL)
	j.configuredSizeMapping = &config.SizeMappingConfig{
		Field:      "story_points",
		Thresholds: map[string][]*float64{},
	}
	if err := j.UpdateSize("PROJ-123", "medium"); err == nil {
		t.Fatal("expected error when mapping has no thresholds, got nil")
	}
}

// --- Linear UpdateSize ---

func TestLinearUpdateSize_CleanPush_EmitsExpectedMutation(t *testing.T) {
	var gotQuery string
	var gotVars map[string]interface{}
	var callCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		query, _ := payload["query"].(string)

		// First call is resolveIssueID (a query, not a mutation).
		if strings.Contains(query, "query GetIssue") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"issue": map[string]interface{}{"id": "uuid-issue-1"},
				},
			})
			return
		}

		gotQuery = query
		gotVars, _ = payload["variables"].(map[string]interface{})
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"issueUpdate": map[string]interface{}{"success": true},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "tok", srv.URL)
	if err := l.UpdateSize("ENG-7", "large"); err != nil {
		t.Fatalf("UpdateSize: %v", err)
	}

	if !containsString(gotQuery, "issueUpdate") {
		t.Errorf("mutation missing issueUpdate; query=%q", gotQuery)
	}
	if !containsString(gotQuery, "$estimate: Float!") {
		t.Errorf("mutation missing $estimate Float! declaration; query=%q", gotQuery)
	}
	if !containsString(gotQuery, "estimate: $estimate") {
		t.Errorf("mutation input missing estimate binding; query=%q", gotQuery)
	}
	id, _ := gotVars["id"].(string)
	if id != "uuid-issue-1" {
		t.Errorf("variables id = %q, want resolved uuid-issue-1", id)
	}
	v, ok := gotVars["estimate"]
	if !ok {
		t.Fatalf("variables missing estimate; vars=%v", gotVars)
	}
	n, ok := v.(float64)
	if !ok {
		t.Fatalf("estimate = %v (%T); want numeric", v, v)
	}
	if n != 8 {
		t.Errorf("estimate = %v; want 8 (large lower band)", n)
	}
}

func TestLinearUpdateSize_EstimationDisabled_LogsAndReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		query, _ := payload["query"].(string)
		if strings.Contains(query, "query GetIssue") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"issue": map[string]interface{}{"id": "uuid-2"},
				},
			})
			return
		}
		// Simulate Linear's estimation-disabled error shape.
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{
				{"message": "The estimate field is not enabled for this team."},
			},
		})
	}))
	defer srv.Close()

	// Capture stderr.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	l, _ := newLinear("ENG", "tok", srv.URL)
	err := l.UpdateSize("ENG-8", "medium")

	w.Close()
	os.Stderr = oldStderr
	captured, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("expected nil error on estimation-disabled, got: %v", err)
	}
	if !strings.Contains(string(captured), "estimation disabled") {
		t.Errorf("stderr = %q, want to contain 'estimation disabled'", string(captured))
	}
	if !strings.Contains(string(captured), "ENG-8") {
		t.Errorf("stderr = %q, want to mention issue ENG-8", string(captured))
	}
}

func TestLinearUpdateSize_OtherErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		query, _ := payload["query"].(string)
		if strings.Contains(query, "query GetIssue") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"issue": map[string]interface{}{"id": "uuid-3"},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{
				{"message": "Permission denied"},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "tok", srv.URL)
	err := l.UpdateSize("ENG-9", "medium")
	if err == nil {
		t.Fatal("expected error to propagate, got nil")
	}
	if !strings.Contains(err.Error(), "Permission denied") {
		t.Errorf("error = %q, want to contain 'Permission denied'", err.Error())
	}
}

// --- GitHub UpdateSize ---

func TestGitHubUpdateSize_StripsOldSizeLabelKeepsOthers(t *testing.T) {
	var patchLabels []string
	var sawGET, sawPATCH bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			sawGET = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"number": 42,
				"labels": []map[string]string{
					{"name": "bug"},
					{"name": "hero:active"},
					{"name": "size/small"},
				},
			})
		case "PATCH":
			sawPATCH = true
			var payload map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if raw, ok := payload["labels"].([]interface{}); ok {
				for _, l := range raw {
					if s, ok := l.(string); ok {
						patchLabels = append(patchLabels, s)
					}
				}
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{}`)
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "tok", srv.URL)
	if err := g.UpdateSize("42", "large"); err != nil {
		t.Fatalf("UpdateSize: %v", err)
	}
	if !sawGET || !sawPATCH {
		t.Fatalf("expected GET and PATCH; sawGET=%v sawPATCH=%v", sawGET, sawPATCH)
	}

	want := []string{"bug", "hero:active", "size/large"}
	if !sameLabelSet(patchLabels, want) {
		t.Errorf("PATCH labels = %v; want set %v", patchLabels, want)
	}
}

func TestGitHubUpdateSize_PreservesNonSizeLabelsWhenNoOldSizeLabel(t *testing.T) {
	var patchLabels []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"number": 99,
				"labels": []map[string]string{
					{"name": "bug"},
					{"name": "hero:active"},
				},
			})
		case "PATCH":
			var payload map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if raw, ok := payload["labels"].([]interface{}); ok {
				for _, l := range raw {
					if s, ok := l.(string); ok {
						patchLabels = append(patchLabels, s)
					}
				}
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "tok", srv.URL)
	if err := g.UpdateSize("99", "large"); err != nil {
		t.Fatalf("UpdateSize: %v", err)
	}

	want := []string{"bug", "hero:active", "size/large"}
	if !sameLabelSet(patchLabels, want) {
		t.Errorf("PATCH labels = %v; want set %v", patchLabels, want)
	}
}

func TestGitHubUpdateSize_PropagatesPatchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"number": 1,
				"labels": []map[string]string{{"name": "bug"}},
			})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"message":"Forbidden"}`)
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "tok", srv.URL)
	err := g.UpdateSize("1", "medium")
	if err == nil {
		t.Fatal("expected error from 403 PATCH, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %q, want to mention 403", err.Error())
	}
}

// --- helpers ---

func sameLabelSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	a := append([]string(nil), got...)
	b := append([]string(nil), want...)
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
