package tracker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// sizedSpec returns testSpec() with a declared size.
func sizedSpec(size string) *spec.Spec {
	s := testSpec()
	s.Size = size
	return s
}

// --- GitHub size-on-create ---

func TestGitHub_CreateIssue_AppendsSizeLabel(t *testing.T) {
	var gotLabels []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if raw, ok := payload["labels"].([]interface{}); ok {
			for _, l := range raw {
				if s, ok := l.(string); ok {
					gotLabels = append(gotLabels, s)
				}
			}
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"number": 1, "html_url": "x"})
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "tok", srv.URL)
	if _, err := g.CreateIssue(sizedSpec("medium")); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	wantSizeLabel := "size/medium"
	found := false
	for _, l := range gotLabels {
		if l == wantSizeLabel {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("labels = %v; want to contain %q", gotLabels, wantSizeLabel)
	}
	// hero:* labels still emitted (non-destructive).
	if !containsString(joinLabels(gotLabels), "hero:feature") {
		t.Errorf("labels missing original hero:feature label: %v", gotLabels)
	}
}

func TestGitHub_CreateIssue_NoSize_NoSizeLabel(t *testing.T) {
	var gotLabels []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if raw, ok := payload["labels"].([]interface{}); ok {
			for _, l := range raw {
				if s, ok := l.(string); ok {
					gotLabels = append(gotLabels, s)
				}
			}
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"number": 1, "html_url": "x"})
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "tok", srv.URL)
	if _, err := g.CreateIssue(testSpec()); err != nil { // no size
		t.Fatalf("CreateIssue: %v", err)
	}
	for _, l := range gotLabels {
		if startsWithSizeSlash(l) {
			t.Errorf("unexpected size label %q when spec has no size", l)
		}
	}
}

func TestGitHub_CreateIssue_NoMappingForTier_NoSizeLabel(t *testing.T) {
	var gotLabels []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if raw, ok := payload["labels"].([]interface{}); ok {
			for _, l := range raw {
				if s, ok := l.(string); ok {
					gotLabels = append(gotLabels, s)
				}
			}
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"number": 1, "html_url": "x"})
	}))
	defer srv.Close()

	g, _ := newGitHub("acme/widgets", "tok", srv.URL)
	// Override with an explicit mapping that omits "medium" — MapSize
	// must error and CreateIssue must skip the label cleanly.
	g.configuredSizeMapping = &config.SizeMappingConfig{
		Field:      "size/",
		Thresholds: map[string][]*float64{}, // empty → no tier maps
	}

	if _, err := g.CreateIssue(sizedSpec("medium")); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	for _, l := range gotLabels {
		if startsWithSizeSlash(l) {
			t.Errorf("unexpected size label %q when mapping has no thresholds", l)
		}
	}
}

// --- Jira size-on-create ---

func TestJira_CreateIssue_SetsStoryPoints(t *testing.T) {
	var gotFields map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if f, ok := payload["fields"].(map[string]interface{}); ok {
			gotFields = f
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"key": "PROJ-1"})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "tok", "user@example.com", srv.URL)
	if _, err := j.CreateIssue(sizedSpec("medium")); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	// Default story-points field ID is customfield_10016; medium → 3 (lower band).
	v, ok := gotFields["customfield_10016"]
	if !ok {
		t.Fatalf("fields missing customfield_10016; got keys=%v", keysOf(gotFields))
	}
	// JSON unmarshals numeric into float64.
	n, ok := v.(float64)
	if !ok {
		t.Fatalf("customfield_10016 = %v (%T); want numeric", v, v)
	}
	if n != 3 {
		t.Errorf("customfield_10016 = %v; want 3 (medium lower band)", n)
	}
}

func TestJira_CreateIssue_NoSize_NoStoryPoints(t *testing.T) {
	var gotFields map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if f, ok := payload["fields"].(map[string]interface{}); ok {
			gotFields = f
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"key": "PROJ-2"})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "tok", "user@example.com", srv.URL)
	if _, err := j.CreateIssue(testSpec()); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if _, ok := gotFields["customfield_10016"]; ok {
		t.Errorf("unexpected story-points field on no-size spec; fields=%v", keysOf(gotFields))
	}
}

func TestJira_CreateIssue_NoMappingForTier_NoStoryPoints(t *testing.T) {
	var gotFields map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if f, ok := payload["fields"].(map[string]interface{}); ok {
			gotFields = f
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"key": "PROJ-3"})
	}))
	defer srv.Close()

	j, _ := newJira("PROJ", "tok", "user@example.com", srv.URL)
	// Override mapping with empty thresholds — MapSize fails, no write.
	j.configuredSizeMapping = &config.SizeMappingConfig{
		Field:      "story_points",
		Thresholds: map[string][]*float64{},
	}
	if _, err := j.CreateIssue(sizedSpec("medium")); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if _, ok := gotFields["customfield_10016"]; ok {
		t.Errorf("unexpected story-points field when mapping has no thresholds; fields=%v", keysOf(gotFields))
	}
}

// --- Linear size-on-create ---

func TestLinear_CreateIssue_SetsEstimate(t *testing.T) {
	var gotQuery string
	var gotVars map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotQuery, _ = payload["query"].(string)
		gotVars, _ = payload["variables"].(map[string]interface{})

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"issueCreate": map[string]interface{}{
					"success": true,
					"issue": map[string]interface{}{
						"id":         "uuid-1",
						"identifier": "ENG-1",
					},
				},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "tok", srv.URL)
	if _, err := l.CreateIssue(sizedSpec("medium")); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}

	if !containsString(gotQuery, "$estimate: Float") {
		t.Errorf("mutation missing $estimate declaration; query=%q", gotQuery)
	}
	v, ok := gotVars["estimate"]
	if !ok {
		t.Fatalf("variables missing estimate; vars=%v", gotVars)
	}
	n, ok := v.(float64)
	if !ok {
		t.Fatalf("estimate = %v (%T); want numeric", v, v)
	}
	if n != 3 {
		t.Errorf("estimate = %v; want 3 (medium lower band)", n)
	}
}

func TestLinear_CreateIssue_NoSize_NoEstimate(t *testing.T) {
	var gotQuery string
	var gotVars map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotQuery, _ = payload["query"].(string)
		gotVars, _ = payload["variables"].(map[string]interface{})

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"issueCreate": map[string]interface{}{
					"success": true,
					"issue": map[string]interface{}{
						"id":         "uuid-2",
						"identifier": "ENG-2",
					},
				},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "tok", srv.URL)
	if _, err := l.CreateIssue(testSpec()); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if containsString(gotQuery, "$estimate") {
		t.Errorf("mutation should not declare $estimate when spec has no size; query=%q", gotQuery)
	}
	if _, ok := gotVars["estimate"]; ok {
		t.Errorf("variables should not contain estimate when spec has no size; vars=%v", gotVars)
	}
}

func TestLinear_CreateIssue_NoMappingForTier_NoEstimate(t *testing.T) {
	var gotQuery string
	var gotVars map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotQuery, _ = payload["query"].(string)
		gotVars, _ = payload["variables"].(map[string]interface{})

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"issueCreate": map[string]interface{}{
					"success": true,
					"issue": map[string]interface{}{
						"id":         "uuid-3",
						"identifier": "ENG-3",
					},
				},
			},
		})
	}))
	defer srv.Close()

	l, _ := newLinear("ENG", "tok", srv.URL)
	l.configuredSizeMapping = &config.SizeMappingConfig{
		Field:      "estimate",
		Thresholds: map[string][]*float64{},
	}
	if _, err := l.CreateIssue(sizedSpec("medium")); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if containsString(gotQuery, "$estimate") {
		t.Errorf("mutation should not declare $estimate when mapping has no thresholds; query=%q", gotQuery)
	}
	if _, ok := gotVars["estimate"]; ok {
		t.Errorf("variables should not contain estimate when mapping has no thresholds; vars=%v", gotVars)
	}
}

// --- helpers ---

func joinLabels(labels []string) string {
	out := ""
	for _, l := range labels {
		out += l + "|"
	}
	return out
}

func startsWithSizeSlash(label string) bool {
	return len(label) >= 5 && label[:5] == "size/"
}

func keysOf(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
