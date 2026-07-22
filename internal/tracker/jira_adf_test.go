package tracker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestADFToMarkdown_MORPH297Fixture(t *testing.T) {
	raw := readJiraFixture(t, "morph-297-description.json")
	want := strings.TrimSpace(string(readJiraFixture(t, "morph-297-description.md")))
	for i := 0; i < 10; i++ {
		if got := jiraADFToMarkdown(raw); got != want {
			t.Fatalf("render %d canonical markdown mismatch\n--- got ---\n%s\n--- want ---\n%s", i, got, want)
		}
	}
}

func TestADFToMarkdown_CompatibilityAndFallbacks(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{name: "plain string unchanged", raw: json.RawMessage(`"literal * markdown\nunchanged"`), want: "literal * markdown\nunchanged"},
		{name: "malformed", raw: json.RawMessage(`{"type":`), want: ""},
		{name: "unknown leaf", raw: json.RawMessage(`{"type":"doc","version":1,"content":[{"type":"future","text":"kept"}]}`), want: "kept"},
		{name: "mention fallback", raw: json.RawMessage(`{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"mention","attrs":{}}]}]}`), want: "@mention"},
		{name: "task list", raw: json.RawMessage(`{"type":"doc","version":1,"content":[{"type":"taskList","content":[{"type":"taskItem","attrs":{"state":"DONE"},"content":[{"type":"paragraph","content":[{"type":"text","text":"verified"}]}]}]}]}`), want: "- [x] verified"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jiraADFToMarkdown(tt.raw); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestADFToMarkdown_SupportedNodesMarksAndFallbacks(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{
			name: "marks use canonical order and link title",
			raw:  json.RawMessage(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"styled","marks":[{"type":"link","attrs":{"href":"https://example.test/a b","title":"Reference"}},{"type":"strike"},{"type":"em"},{"type":"strong"}]}]}]}`),
			want: `[~~_**styled**_~~](https://example.test/a%20b "Reference")`,
		},
		{
			name: "same marks in different source order",
			raw:  json.RawMessage(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"styled","marks":[{"type":"strong"},{"type":"em"},{"type":"link","attrs":{"title":"Reference","href":"https://example.test/a b"}},{"type":"strike"}]}]}]}`),
			want: `[~~_**styled**_~~](https://example.test/a%20b "Reference")`,
		},
		{
			name: "inline code and literal escaping",
			raw:  json.RawMessage(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"a` + "`" + `b","marks":[{"type":"code"}]},{"type":"text","text":" literal *stars* and [brackets]"}]}]}`),
			want: "``a`b`` literal \\*stars\\* and \\[brackets\\]",
		},
		{
			name: "panel and hard break",
			raw:  json.RawMessage(`{"type":"doc","content":[{"type":"panel","attrs":{"panelType":"info"},"content":[{"type":"paragraph","content":[{"type":"text","text":"First"},{"type":"hardBreak"},{"type":"text","text":"Second"}]}]}]}`),
			want: "> **Info:**\n> First<br>\n> Second",
		},
		{
			name: "semantic inline nodes",
			raw:  json.RawMessage(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"displayName":"Ada"}},{"type":"text","text":" "},{"type":"mention","attrs":{"id":"account-7"}},{"type":"text","text":" "},{"type":"status","attrs":{"text":"IN PROGRESS"}},{"type":"text","text":" "},{"type":"emoji","attrs":{"shortName":"wave"}}]}]}`),
			want: "@Ada @account-7 [IN PROGRESS] :wave:",
		},
		{
			name: "cards preserve labels and targets",
			raw:  json.RawMessage(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"inlineCard","attrs":{"title":"Docs","url":"https://example.test/docs"}}]},{"type":"blockCard","attrs":{"label":"MORPH-297","url":"https://example.test/issue"}}]}`),
			want: "[Docs](https://example.test/docs)\n\n[MORPH-297](https://example.test/issue)",
		},
		{
			name: "media URL and readable fallback",
			raw:  json.RawMessage(`{"type":"doc","content":[{"type":"media","attrs":{"alt":"Screenshot","url":"https://example.test/screen.png"}},{"type":"mediaGroup","content":[{"type":"media","attrs":{"filename":"trace.log"}}]}]}`),
			want: "![Screenshot](https://example.test/screen.png)\n\n[media: trace.log]",
		},
		{
			name: "missing attributes remain readable",
			raw:  json.RawMessage(`{"type":"doc","content":[{"type":"panel","attrs":{"panelType":"future-type"},"content":[]},{"type":"paragraph","content":[{"type":"inlineCard"},{"type":"text","text":" "},{"type":"media"},{"type":"text","text":" "},{"type":"date","attrs":{"timestamp":"2026-07-21"}}]}]}`),
			want: "> **Panel:**\n\n[card] [media: attachment] 2026-07-21",
		},
		{
			name: "malformed children do not discard siblings",
			raw:  json.RawMessage(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"before"}]},42,{"type":"paragraph","content":"bad"},{"type":"paragraph","content":[{"type":"text","text":"after"}]}]}`),
			want: "before\n\nafter",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jiraADFToMarkdown(tt.raw); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestADFToMarkdown_UsesFenceLongerThanContent(t *testing.T) {
	raw := json.RawMessage("{\"type\":\"doc\",\"version\":1,\"content\":[{\"type\":\"codeBlock\",\"attrs\":{\"language\":\"go lang\"},\"content\":[{\"type\":\"text\",\"text\":\"before ``` after\"}]}]}")
	if got, want := jiraADFToMarkdown(raw), "````\nbefore ``` after\n````"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestJiraADFMarkdown_IsIdenticalAcrossReadSurfaces(t *testing.T) {
	description := json.RawMessage(readJiraFixture(t, "morph-297-description.json"))
	want := strings.TrimSpace(string(readJiraFixture(t, "morph-297-description.md")))
	issueFields := map[string]any{
		"summary":     "Bulk start of discovered VMs",
		"status":      map[string]any{"name": "Open"},
		"issuetype":   map[string]any{"name": "Bug"},
		"priority":    map[string]any{"name": "Critical"},
		"description": description,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/3/issue/MORPH-297/comment":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"startAt": 0, "maxResults": 100, "total": 1,
				"comments": []any{map[string]any{"id": "1", "body": description}},
			})
		case r.URL.Path == "/rest/api/3/issue/MORPH-297":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "MORPH-297", "fields": issueFields,
				"names": map[string]string{}, "changelog": map[string]any{"histories": []any{}},
			})
		case r.URL.Path == "/rest/api/3/search/jql":
			key := "MORPH-297"
			result := map[string]any{}
			if r.URL.Query().Get("nextPageToken") == "" {
				result["nextPageToken"] = "page-2"
			} else {
				key = "MORPH-298"
			}
			result["issues"] = []any{map[string]any{"key": key, "fields": issueFields}}
			_ = json.NewEncoder(w).Encode(result)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	j, err := newJira("MORPH", "test-token", "user@example.com", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	j.fieldDiscoveryDone = true
	j.resolvedCustom = map[string]string{}

	issue, err := j.GetIssue("MORPH-297")
	if err != nil || issue.Description != want {
		t.Fatalf("GetIssue description mismatch: err=%v\n%s", err, issue.Description)
	}
	listed, err := j.ListIssues("", 2)
	if err != nil || len(listed) != 2 || listed[0].Description != want || listed[1].Description != want {
		t.Fatalf("ListIssues description mismatch: err=%v issues=%+v", err, listed)
	}
	searched, err := j.Search(SearchQuery{RawQuery: "project = MORPH", Limit: 2})
	if err != nil || len(searched) != 2 || searched[0].Description != want || searched[1].Description != want {
		t.Fatalf("Search description mismatch: err=%v issues=%+v", err, searched)
	}
	fields, err := j.GetFields("MORPH-297")
	if err != nil || fields["description"].Str != want {
		t.Fatalf("GetFields description mismatch: err=%v fields=%+v", err, fields)
	}
	evidence, err := j.GetIssueEvidence("MORPH-297")
	if err != nil || evidence.Normalized.Description != want || len(evidence.Comments) != 1 || evidence.Comments[0].Text != want {
		t.Fatalf("evidence description mismatch: err=%v evidence=%+v", err, evidence)
	}
	if jiraADFToMarkdown(evidence.RawFields["description"]) != want || jiraADFToMarkdown(evidence.Comments[0].RawBody) != want {
		t.Fatal("lossless evidence did not retain the raw ADF description and comment body")
	}

	sprint, err := newJiraSprintLoader("MORPH", "test-token", "user@example.com", server.URL, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	sprint.fieldDiscoveryDone = true
	sprint.resolvedCustom = map[string]string{}
	items, err := sprint.loadSprintItems("42", "Sprint 42")
	if err != nil || len(items) != 2 || items[0].Description != want || items[1].Description != want {
		t.Fatalf("sprint description mismatch: err=%v items=%+v", err, items)
	}
}

func readJiraFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "jira", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
