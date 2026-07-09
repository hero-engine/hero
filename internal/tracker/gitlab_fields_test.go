package tracker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- GetFields round-trip ---

func TestGitLab_GetFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"iid":42,"title":"T","description":"D","labels":["a","b"],"weight":5}`))
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "tok", srv.URL)
	fields, err := g.GetFields("42")
	if err != nil {
		t.Fatalf("GetFields failed: %v", err)
	}
	if fields["title"].Str != "T" {
		t.Errorf("title = %q, want T", fields["title"].Str)
	}
	if fields["description"].Str != "D" {
		t.Errorf("description = %q, want D", fields["description"].Str)
	}
	if !fields["labels"].Equal(StringsValue([]string{"a", "b"})) {
		t.Errorf("labels = %v, want [a b]", fields["labels"])
	}
	if fields["points"].Int != 5 {
		t.Errorf("points = %d, want 5", fields["points"].Int)
	}
}

// --- UpdateFields: title + description via PUT ---

func TestGitLab_UpdateFields_TitleDescription(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&got)
		json.NewEncoder(w).Encode(map[string]interface{}{"iid": 42})
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "tok", srv.URL)
	err := g.UpdateFields("42", map[string]Value{
		"title":       StringValue("New Title"),
		"description": StringValue("New Body"),
	})
	if err != nil {
		t.Fatalf("UpdateFields failed: %v", err)
	}
	if got["title"] != "New Title" {
		t.Errorf("title = %v, want New Title", got["title"])
	}
	if got["description"] != "New Body" {
		t.Errorf("description = %v, want New Body", got["description"])
	}
}

// --- UpdateFields: label merge preserves existing non-hero labels ---

func TestGitLab_UpdateFields_LabelMergePreservation(t *testing.T) {
	var putLabels string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Write([]byte(`{"iid":42,"labels":["keep-me","team::payments"]}`))
		case "PUT":
			var p map[string]interface{}
			json.NewDecoder(r.Body).Decode(&p)
			putLabels, _ = p["labels"].(string)
			json.NewEncoder(w).Encode(map[string]interface{}{"iid": 42})
		}
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "tok", srv.URL)
	err := g.UpdateFields("42", map[string]Value{
		"labels": StringsValue([]string{"new-label"}),
	})
	if err != nil {
		t.Fatalf("UpdateFields failed: %v", err)
	}
	for _, want := range []string{"keep-me", "team::payments", "new-label"} {
		if !strings.Contains(putLabels, want) {
			t.Errorf("merged labels %q missing %q", putLabels, want)
		}
	}
}

// --- UpdateFields: priority rotation replaces priority::* labels ---

func TestGitLab_UpdateFields_PriorityRotation(t *testing.T) {
	var putLabels string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Write([]byte(`{"iid":42,"labels":["priority::low","keep"]}`))
		case "PUT":
			var p map[string]interface{}
			json.NewDecoder(r.Body).Decode(&p)
			putLabels, _ = p["labels"].(string)
			json.NewEncoder(w).Encode(map[string]interface{}{"iid": 42})
		}
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "tok", srv.URL)
	err := g.UpdateFields("42", map[string]Value{
		"priority": StringValue("high"),
	})
	if err != nil {
		t.Fatalf("UpdateFields failed: %v", err)
	}
	if strings.Contains(putLabels, "priority::low") {
		t.Errorf("expected priority::low rotated out, got %q", putLabels)
	}
	if !strings.Contains(putLabels, "priority::high") {
		t.Errorf("expected priority::high rotated in, got %q", putLabels)
	}
	if !strings.Contains(putLabels, "keep") {
		t.Errorf("expected non-priority label preserved, got %q", putLabels)
	}
}

// --- UpdateFields: weight 403 degrades (Premium) rather than failing ---

func TestGitLab_UpdateFields_WeightPremiumDegradation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"weight is premium"}`))
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "tok", srv.URL)
	err := g.UpdateFields("42", map[string]Value{"points": IntValue(8)})
	if err != nil {
		t.Fatalf("expected weight-only 403 to degrade gracefully, got %v", err)
	}
}

// --- UpdateFields: 429 honored with a single retry (AC-10) ---

func TestGitLab_UpdateFields_429SingleRetry(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"iid": 42})
	}))
	defer srv.Close()

	g, _ := newGitLab("group/proj", "tok", srv.URL)
	err := g.UpdateFields("42", map[string]Value{"title": StringValue("X")})
	if err != nil {
		t.Fatalf("expected success after one retry, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (429 then retry), got %d", calls)
	}
}

func TestGitLab_UpdateFields_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("empty patch should make no network call")
	}))
	defer srv.Close()
	g, _ := newGitLab("group/proj", "tok", srv.URL)
	if err := g.UpdateFields("42", map[string]Value{}); err != nil {
		t.Fatalf("empty patch should be a no-op, got %v", err)
	}
}
