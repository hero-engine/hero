package tracker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	projectcontract "github.com/hero-engine/hero/contracts/trackerproject"
)

func TestJiraProjectSnapshotLoadsBoardIterationsAndMembership(t *testing.T) {
	searchCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/agile/1.0/board":
			json.NewEncoder(w).Encode(map[string]any{"values": []any{
				map[string]any{"id": 8, "name": "Other team"},
				map[string]any{"id": 7, "name": "MORPH board"},
				map[string]any{"id": 9, "name": "MORPH"},
			}})
		case "/rest/agile/1.0/board/7/sprint":
			if r.URL.Query().Get("state") != "active,future" {
				t.Fatalf("state = %q", r.URL.Query().Get("state"))
			}
			json.NewEncoder(w).Encode(map[string]any{"isLast": true, "values": []any{
				map[string]any{"id": 71, "name": "Next", "state": "future", "startDate": "2026-08-01", "endDate": "2026-08-31"},
				map[string]any{"id": 70, "name": "Current", "state": "active", "startDate": "2026-07-01", "endDate": "2026-07-31"},
			}})
		case "/rest/api/3/search/jql":
			searchCalls++
			if fields := r.URL.Query().Get("fields"); fields != "summary,issuetype,status,assignee" {
				t.Fatalf("project snapshot fields = %q", fields)
			}
			jql := r.URL.Query().Get("jql")
			sprintID := "70"
			status := map[string]any{"name": "In Progress", "statusCategory": map[string]any{"key": "indeterminate"}}
			if strings.Contains(jql, "sprint = 71") {
				sprintID = "71"
				status = map[string]any{"name": "Assigned", "statusCategory": map[string]any{"key": "new"}}
			}
			wantJQL := `project = "MORPH" AND sprint = ` + sprintID + ` ORDER BY rank ASC`
			if jql != wantJQL {
				t.Fatalf("jql = %q, want %q", jql, wantJQL)
			}
			json.NewEncoder(w).Encode(map[string]any{"issues": []any{map[string]any{
				"key":    "MORPH-" + sprintID,
				"fields": map[string]any{"summary": "Item " + sprintID, "issuetype": map[string]any{"name": "Bug"}, "status": status, "priority": map[string]any{"name": "High"}, "assignee": map[string]any{"displayName": "Ada"}},
			}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	loader, err := newJiraSprintLoader("MORPH", "token", "ada@example.com", server.URL, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	loader.fieldDiscoveryDone = true
	loader.resolvedCustom = map[string]string{}
	snapshot, err := loader.LoadProjectSnapshot("")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Version != projectcontract.Version || snapshot.Board == nil || snapshot.Board.ID != "7" || len(snapshot.Iterations) != 2 || len(snapshot.Items) != 2 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if snapshot.Iterations[0].ID != "70" || snapshot.Iterations[1].ID != "71" || snapshot.Items[0].TrackerID != "MORPH-70" || snapshot.Items[1].TrackerID != "MORPH-71" {
		t.Fatalf("snapshot ordering = iterations %+v, items %+v", snapshot.Iterations, snapshot.Items)
	}
	if snapshot.Items[0].StatusCategory != "in-progress" || snapshot.Items[1].StatusCategory != "todo" || searchCalls != 2 || !snapshot.Complete || snapshot.Truncated {
		t.Fatalf("items=%+v calls=%d", snapshot.Items, searchCalls)
	}
}

func TestJiraProjectSnapshotBoardSelectionDoesNotGuessAmongAmbiguousBoards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"values": []any{
			map[string]any{"id": 2, "name": "Zulu"},
			map[string]any{"id": 1, "name": "Alpha"},
		}})
	}))
	defer server.Close()

	loader, err := newJiraSprintLoader("MORPH", "token", "ada@example.com", server.URL, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = loader.resolveBoard("")
	if err == nil || !strings.Contains(err.Error(), "multiple boards found") || !strings.Contains(err.Error(), "1 (Alpha), 2 (Zulu)") || !strings.Contains(err.Error(), "--board") {
		t.Fatalf("ambiguity error = %v", err)
	}
}

func TestJiraProjectSnapshotBoardSelectionAllowsSingleBoard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"values": []any{map[string]any{"id": 4, "name": "Delivery"}}})
	}))
	defer server.Close()

	loader, err := newJiraSprintLoader("MORPH", "token", "ada@example.com", server.URL, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	id, name, err := loader.resolveBoard("")
	if err != nil || id != "4" || name != "Delivery" {
		t.Fatalf("board = %q %q, err = %v", id, name, err)
	}
}

func TestJiraProjectSnapshotBoardSelectionUsesExactProjectKeyName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"values": []any{
			map[string]any{"id": 3, "name": "Other"},
			map[string]any{"id": 4, "name": "MORPH"},
		}})
	}))
	defer server.Close()

	loader, err := newJiraSprintLoader("MORPH", "token", "ada@example.com", server.URL, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	id, name, err := loader.resolveBoard("")
	if err != nil || id != "4" || name != "MORPH" {
		t.Fatalf("board = %q %q, err = %v", id, name, err)
	}
}

func TestJiraProjectSnapshotBoardSelectionPaginatesBeforeSelection(t *testing.T) {
	boardCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		boardCalls++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("startAt") {
		case "0":
			json.NewEncoder(w).Encode(map[string]any{"startAt": 0, "maxResults": 1, "total": 2, "isLast": false, "values": []any{map[string]any{"id": 3, "name": "Other"}}})
		case "1":
			json.NewEncoder(w).Encode(map[string]any{"startAt": 1, "maxResults": 1, "total": 2, "isLast": true, "values": []any{map[string]any{"id": 4, "name": "MORPH board"}}})
		default:
			t.Fatalf("unexpected startAt = %q", r.URL.Query().Get("startAt"))
		}
	}))
	defer server.Close()

	loader, err := newJiraSprintLoader("MORPH", "token", "ada@example.com", server.URL, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, boardRef := range []string{"", "MORPH board"} {
		id, name, resolveErr := loader.resolveBoard(boardRef)
		if resolveErr != nil || id != "4" || name != "MORPH board" {
			t.Fatalf("resolveBoard(%q) = %q %q, err = %v", boardRef, id, name, resolveErr)
		}
	}
	if boardCalls != 4 {
		t.Fatalf("board calls = %d, want 4", boardCalls)
	}
}

func TestNormalizeStatusCategoryFallsBackToNativeStatus(t *testing.T) {
	for _, tc := range []struct{ category, status, want string }{
		{"done", "", "done"}, {"", "In Test", "in-progress"}, {"", "Closed", "done"}, {"", "New", "todo"}, {"", "", "unknown"},
	} {
		if got := normalizeStatusCategory(tc.category, tc.status); got != tc.want {
			t.Errorf("normalizeStatusCategory(%q, %q) = %q, want %q", tc.category, tc.status, got, tc.want)
		}
	}
}
