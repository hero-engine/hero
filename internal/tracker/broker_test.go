package tracker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	brokercontract "github.com/hero-engine/hero/contracts/trackerbroker"
	"github.com/hero-engine/hero/internal/config"
)

const brokerCanary = "BROKER-CREDENTIAL-CANARY-91e742"

func brokerTestConfig(provider, project, baseURL string) config.Config {
	str := func(v string) json.RawMessage { b, _ := json.Marshal(v); return b }
	return config.Config{Integrations: &config.IntegrationsConfig{Connections: map[string]config.IntegrationConfig{
		"test": {
			Provider: provider,
			Settings: map[string]json.RawMessage{"project": str(project), "base_url": str(baseURL), "user_email": str("agent@example.com")},
			Auth:     &config.IntegrationAuth{Token: config.Secret(brokerCanary)},
		},
	}}}
}

func testBroker(cfg config.Config) *Broker {
	b := NewBroker("/unused")
	b.loadConfig = func(string) (config.Config, error) { return cfg, nil }
	return b
}

func TestBrokerGetIssueAcceptsFullJiraKeyWithoutProjectConstraint(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			fmt.Fprint(w, `[]`)
			return
		}
		path = r.URL.Path
		if r.URL.Query().Get("fields") == "" {
			t.Error("missing requested fields")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"key":"ACME-103","fields":{"summary":"Direct key","status":{"name":"Open"},"created":"2026-01-01T00:00:00Z","updated":"2026-02-01T00:00:00Z"}}`)
	}))
	defer server.Close()

	resp := testBroker(brokerTestConfig("jira", "CONFIGURED", server.URL)).GetIssue(context.Background(), brokercontract.GetIssueRequest{IssueID: "ACME-103"})
	if resp.Error != nil {
		t.Fatalf("response error: %+v", resp.Error)
	}
	if path != "/rest/api/3/issue/ACME-103" {
		t.Fatalf("path = %q", path)
	}
	var issue brokercontract.Issue
	if err := json.Unmarshal(resp.Result, &issue); err != nil {
		t.Fatal(err)
	}
	if issue.ID != "ACME-103" || issue.CreatedAt != "2026-01-01T00:00:00Z" || issue.UpdatedAt != "2026-02-01T00:00:00Z" {
		t.Fatalf("issue = %+v", issue)
	}
}

func TestBrokerGetIssueEvidenceAcceptsACME103(t *testing.T) {
	var issueCalls, commentCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/3/field":
			fmt.Fprint(w, `[]`)
		case "/rest/api/3/issue/ACME-103":
			atomic.AddInt32(&issueCalls, 1)
			if r.URL.Query().Get("fields") != "*all" || r.URL.Query().Get("expand") != "names,changelog" {
				t.Errorf("evidence query = %q", r.URL.RawQuery)
			}
			fmt.Fprint(w, `{"key":"ACME-103","fields":{"summary":"Evidence key","description":"Full description","status":{"name":"Open"}},"names":{"summary":"Summary"},"changelog":{"histories":[]}}`)
		case "/rest/api/3/issue/ACME-103/comment":
			atomic.AddInt32(&commentCalls, 1)
			fmt.Fprint(w, `{"startAt":0,"maxResults":100,"total":0,"comments":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	resp := testBroker(brokerTestConfig("jira", "CONFIGURED", server.URL)).GetIssue(
		context.Background(),
		brokercontract.GetIssueRequest{IssueID: "ACME-103", Detail: brokercontract.DetailEvidence},
	)
	if resp.Error != nil {
		t.Fatalf("response = %+v", resp)
	}
	if atomic.LoadInt32(&issueCalls) != 1 || atomic.LoadInt32(&commentCalls) != 1 {
		t.Fatalf("issue calls=%d comment calls=%d", issueCalls, commentCalls)
	}
	var evidence IssueEvidence
	if err := json.Unmarshal(resp.Result, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.IssueID != "ACME-103" || evidence.Normalized == nil || evidence.Normalized.Title != "Evidence key" {
		t.Fatalf("evidence = %+v", evidence)
	}
}

func TestValidateBrokerIssueIDAcceptsOrdinaryProviderIDsAndRejectsPathControls(t *testing.T) {
	for _, tc := range []struct {
		provider string
		issueID  string
	}{
		{provider: "jira", issueID: "MORPH-297"},
		{provider: "jira", issueID: "ACME-103"},
		{provider: "jira", issueID: "ZERO-10"},
		{provider: "linear", issueID: "runtime-nx-10"},
		{provider: "github", issueID: "10"},
		{provider: "gitlab", issueID: "10"},
	} {
		t.Run("valid/"+tc.provider+"/"+tc.issueID, func(t *testing.T) {
			if err := validateBrokerIssueID(tc.provider, tc.issueID); err != nil {
				t.Fatalf("validateBrokerIssueID(%q, %q) = %v", tc.provider, tc.issueID, err)
			}
		})
	}

	for name, issueID := range map[string]string{
		"slash":     "ACME/103",
		"backslash": `ACME\103`,
		"query":     "ACME?103",
		"fragment":  "ACME#103",
		"traversal": "ACME..103",
		"carriage":  "ACME\r103",
		"newline":   "ACME\n103",
		"nul":       "ACME\x00103",
	} {
		t.Run("invalid/"+name, func(t *testing.T) {
			if err := validateBrokerIssueID("jira", issueID); err == nil {
				t.Fatalf("validateBrokerIssueID accepted unsafe ID %q", issueID)
			}
		})
	}

	for _, tc := range []struct {
		provider string
		issueID  string
	}{
		{provider: "github", issueID: "not-10"},
		{provider: "gitlab", issueID: "not-10"},
	} {
		if err := validateBrokerIssueID(tc.provider, tc.issueID); err == nil {
			t.Fatalf("validateBrokerIssueID(%q, %q) accepted a non-numeric issue number", tc.provider, tc.issueID)
		}
	}
}

func TestBrokerJiraSearchPreservesCrossProjectQueryAndCursor(t *testing.T) {
	query := `project in (PCS, OTHER) AND summary ~ "quoted value" ORDER BY updated DESC`
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			fmt.Fprint(w, `[]`)
			return
		}
		atomic.AddInt32(&calls, 1)
		if got := r.URL.Query().Get("jql"); got != query {
			t.Errorf("jql changed: %q", got)
		}
		token := r.URL.Query().Get("nextPageToken")
		w.Header().Set("Content-Type", "application/json")
		if token == "" {
			fmt.Fprint(w, `{"issues":[{"key":"OTHER-7","fields":{"summary":"One","status":{"name":"Open"}}}],"nextPageToken":"native-next"}`)
			return
		}
		if token != "native-next" {
			t.Errorf("native cursor = %q", token)
		}
		fmt.Fprint(w, `{"issues":[{"key":"PCS-8","fields":{"summary":"Two","status":{"name":"Open"}}}]}`)
	}))
	defer server.Close()
	b := testBroker(brokerTestConfig("jira", "CONFIGURED", server.URL))
	first := b.Search(context.Background(), brokercontract.SearchRequest{Query: query, Limit: 1})
	if first.Error != nil || first.NextCursor == "" {
		t.Fatalf("first = %+v", first)
	}
	second := b.Search(context.Background(), brokercontract.SearchRequest{Query: query, Limit: 1, Cursor: first.NextCursor})
	if second.Error != nil || second.NextCursor != "" {
		t.Fatalf("second = %+v", second)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("calls = %d", calls)
	}
	mismatch := b.Search(context.Background(), brokercontract.SearchRequest{Query: query + " ", Limit: 1, Cursor: first.NextCursor})
	if mismatch.Error == nil || mismatch.Error.Code != "cursor_mismatch" {
		t.Fatalf("mismatch = %+v", mismatch)
	}
}

func TestBrokerGitHubSearchDoesNotInjectConfiguredRepo(t *testing.T) {
	query := `is:issue org:other label:"needs review"`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/issues" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != query {
			t.Errorf("query = %q", got)
		}
		fmt.Fprint(w, `{"total_count":1,"items":[{"number":7,"title":"Other repo","state":"open","html_url":"https://github.example/other/repo/issues/7"}]}`)
	}))
	defer server.Close()
	resp := testBroker(brokerTestConfig("github", "configured/repo", server.URL)).Search(context.Background(), brokercontract.SearchRequest{Query: query, Limit: 10})
	if resp.Error != nil || strings.Contains(string(resp.Result), "configured/repo") {
		t.Fatalf("response = %+v", resp)
	}
}

func TestBrokerGitLabSearchUsesInstanceWideEndpoint(t *testing.T) {
	query := "native search value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/issues" || r.URL.Query().Get("search") != query {
			t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		fmt.Fprint(w, `[{"id":77,"iid":3,"title":"Global","state":"opened"}]`)
	}))
	defer server.Close()
	resp := testBroker(brokerTestConfig("gitlab", "configured/repo", server.URL)).Search(context.Background(), brokercontract.SearchRequest{Query: query, Limit: 10})
	if resp.Error != nil {
		t.Fatalf("response = %+v", resp)
	}
}

func TestBrokerLinearSearchPreservesNativeQueryAndCursor(t *testing.T) {
	query := "cross-team native text"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Variables["query"] != query {
			t.Errorf("query = %#v", payload.Variables["query"])
		}
		fmt.Fprint(w, `{"data":{"searchIssues":{"nodes":[{"id":"uuid","identifier":"ENG-7","title":"Result","state":{"name":"Open"}}],"pageInfo":{"hasNextPage":true,"endCursor":"linear-next"}}}}`)
	}))
	defer server.Close()
	resp := testBroker(brokerTestConfig("linear", "CONFIGURED", server.URL)).Search(context.Background(), brokercontract.SearchRequest{Query: query, Limit: 10})
	if resp.Error != nil || resp.NextCursor == "" {
		t.Fatalf("response = %+v", resp)
	}
}

func TestBrokerFullProjectIssueIDsForGitHubAndGitLab(t *testing.T) {
	for _, tc := range []struct {
		provider string
		issueID  string
		path     string
		body     string
	}{
		{provider: "github", issueID: "other/repo#7", path: "/repos/other/repo/issues/7", body: `{"number":7,"title":"Other","state":"open"}`},
		{provider: "gitlab", issueID: "other/repo#7", path: "/api/v4/projects/other%2Frepo/issues/7", body: `{"id":77,"iid":7,"title":"Other","state":"opened"}`},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.EscapedPath() != tc.path {
					t.Errorf("path = %q", r.URL.EscapedPath())
				}
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			resp := testBroker(brokerTestConfig(tc.provider, "configured/repo", server.URL)).GetIssue(context.Background(), brokercontract.GetIssueRequest{IssueID: tc.issueID})
			if resp.Error != nil {
				t.Fatalf("response = %+v", resp)
			}
		})
	}
}

func TestBrokerRejectsIssueIDPathConfusionBeforeProviderRequest(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
	}))
	defer server.Close()
	for _, tc := range []struct {
		provider string
		project  string
		issueID  string
	}{
		{"jira", "P", "../../rest/api/3/myself"},
		{"github", "owner/repo", "https://evil.example/repo#7"},
		{"github", "owner/repo", "owner/repo#not-a-number"},
		{"gitlab", "owner/repo", "/owner/repo#7"},
		{"jira", "P", "ACME\r103"},
		{"jira", "P", "ACME\n103"},
		{"jira", "P", "ACME\x00103"},
	} {
		resp := testBroker(brokerTestConfig(tc.provider, tc.project, server.URL)).GetIssue(context.Background(), brokercontract.GetIssueRequest{IssueID: tc.issueID})
		if resp.Error == nil || resp.Error.Code != "invalid_issue_id" {
			t.Fatalf("%s %q response = %+v", tc.provider, tc.issueID, resp)
		}
	}
	if calls != 0 {
		t.Fatalf("unsafe issue IDs reached provider %d times", calls)
	}
}

func TestBrokerRequestSameOriginBoundsAndRedacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Errorf("authentication was not injected")
		}
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		fmt.Fprint(w, brokerCanary+"-response")
	}))
	defer server.Close()
	resp := testBroker(brokerTestConfig("jira", "P", server.URL)).Request(context.Background(), brokercontract.RequestRequest{
		Method: "GET", RelativePath: "/start", OutputLimit: 64,
	})
	if resp.Error != nil {
		t.Fatalf("response = %+v", resp)
	}
	if strings.Contains(resp.Body, brokerCanary) || !strings.Contains(resp.Body, "[REDACTED]") {
		t.Fatalf("body was not redacted: %q", resp.Body)
	}
	if resp.Effect != brokercontract.EffectRead || resp.StatusCode == nil || *resp.StatusCode != 200 {
		t.Fatalf("metadata = %+v", resp)
	}

	truncated := testBroker(brokerTestConfig("jira", "P", server.URL)).Request(context.Background(), brokercontract.RequestRequest{Method: "GET", RelativePath: "/final", OutputLimit: 4})
	if !truncated.Truncated || len(truncated.Body) > len("[REDACTED]") {
		t.Fatalf("truncated = %+v", truncated)
	}
}

func TestBrokerRequestRejectsOriginAndHeaderConfusionBeforeSend(t *testing.T) {
	var originCalls, targetCalls int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&targetCalls, 1)
	}))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&originCalls, 1)
		http.Redirect(w, r, target.URL+"/steal", http.StatusFound)
	}))
	defer origin.Close()
	b := testBroker(brokerTestConfig("jira", "P", origin.URL))

	for _, path := range []string{target.URL + "/x", "//evil.example/x", "https://user@evil.example/x", "/x#fragment", `\evil.example\x`} {
		resp := b.Request(context.Background(), brokercontract.RequestRequest{Method: "GET", RelativePath: path})
		if resp.Error == nil || resp.Error.Code != "unsafe_request" {
			t.Fatalf("path %q response = %+v", path, resp)
		}
	}
	for _, headers := range []map[string]string{{"Authorization": "caller-secret"}, {"Host": "evil.example"}, {"X-Forwarded-Host": "evil.example"}} {
		header := b.Request(context.Background(), brokercontract.RequestRequest{Method: "GET", RelativePath: "/x", Headers: headers})
		if header.Error == nil || header.Error.Code != "unsafe_headers" {
			t.Fatalf("header response = %+v", header)
		}
	}
	redirect := b.Request(context.Background(), brokercontract.RequestRequest{Method: "GET", RelativePath: "/redirect"})
	if redirect.Error == nil || redirect.Error.Code != "unsafe_redirect" {
		t.Fatalf("redirect response = %+v", redirect)
	}
	if atomic.LoadInt32(&originCalls) != 1 || atomic.LoadInt32(&targetCalls) != 0 {
		t.Fatalf("calls origin=%d target=%d", originCalls, targetCalls)
	}
}

func TestBrokerRequestCancellationAndNonIdempotentSingleAttempt(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Method == http.MethodPost {
			http.Error(w, "fail", http.StatusInternalServerError)
			return
		}
		<-r.Context().Done()
	}))
	defer server.Close()
	b := testBroker(brokerTestConfig("jira", "P", server.URL))
	post := b.Request(context.Background(), brokercontract.RequestRequest{Method: "POST", RelativePath: "/write"})
	if post.Effect != brokercontract.EffectWriteNonIdempotent || post.Error == nil || atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("post = %+v calls=%d", post, calls)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	cancelled := b.Request(ctx, brokercontract.RequestRequest{Method: "GET", RelativePath: "/wait"})
	if cancelled.Error == nil || cancelled.Error.Code != "cancelled" {
		t.Fatalf("cancelled = %+v", cancelled)
	}
}

func TestBrokerRequestEnforcesRedirectBound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		http.Redirect(w, r, fmt.Sprintf("/loop?n=%d", n+1), http.StatusFound)
	}))
	defer server.Close()
	resp := testBroker(brokerTestConfig("jira", "P", server.URL)).Request(context.Background(), brokercontract.RequestRequest{Method: "GET", RelativePath: "/loop"})
	if resp.Error == nil || resp.Error.Code != "unsafe_redirect" {
		t.Fatalf("response = %+v", resp)
	}
}

func TestBrokerRequestRejectsNonIdempotentRedirectWithoutSecondAttempt(t *testing.T) {
	var first, second int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/write" {
			atomic.AddInt32(&first, 1)
			http.Redirect(w, r, "/second", http.StatusTemporaryRedirect)
			return
		}
		atomic.AddInt32(&second, 1)
	}))
	defer server.Close()
	resp := testBroker(brokerTestConfig("jira", "P", server.URL)).Request(context.Background(), brokercontract.RequestRequest{Method: "POST", RelativePath: "/write", Body: `{}`})
	if resp.Error == nil || resp.Error.Code != "unsafe_redirect" || first != 1 || second != 0 {
		t.Fatalf("response=%+v first=%d second=%d", resp, first, second)
	}
}

func TestBrokerRedactsCredentialAcrossOutputBoundary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("x", 13)+brokerCanary+"tail")
	}))
	defer server.Close()
	resp := testBroker(brokerTestConfig("jira", "P", server.URL)).Request(context.Background(), brokercontract.RequestRequest{Method: "GET", RelativePath: "/x", OutputLimit: 16})
	if !resp.Truncated || strings.Contains(resp.Body, brokerCanary) || strings.Contains(resp.Body, brokerCanary[:3]) {
		t.Fatalf("response = %+v", resp)
	}
}

func TestBrokerCLIExactArgvChildOnlyEnvironmentAndRedaction(t *testing.T) {
	if os.Getenv("GO_WANT_BROKER_HELPER") == "1" {
		if os.Getenv("BROKER_HELPER_SLEEP") == "1" {
			time.Sleep(5 * time.Second)
		}
		fmt.Printf("argv=%q token=%s host=%s", os.Args[1:], os.Getenv("GH_TOKEN"), os.Getenv("GH_HOST"))
		fmt.Fprintf(os.Stderr, "stderr-token=%s", os.Getenv("GH_TOKEN"))
		os.Exit(0)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GO_WANT_BROKER_HELPER", "1")
	t.Setenv("GH_TOKEN", "ambient-parent-token")
	b := testBroker(brokerTestConfig("github", "owner/repo", ""))
	b.lookPath = func(name string) (string, error) {
		if name != "gh" {
			t.Fatalf("lookup = %q", name)
		}
		return exe, nil
	}
	resp := b.CLI(context.Background(), brokercontract.CLIRequest{Executable: "gh", Arguments: []string{"-test.run=TestBrokerCLIExactArgvChildOnlyEnvironmentAndRedaction"}, OutputLimit: 4096})
	if resp.Error != nil {
		t.Fatalf("response = %+v", resp)
	}
	if !strings.Contains(resp.Stdout, `-test.run=TestBrokerCLIExactArgvChildOnlyEnvironmentAndRedaction`) || strings.Contains(resp.Stdout, brokerCanary) || !strings.Contains(resp.Stdout, "token=[REDACTED]") || !strings.Contains(resp.Stdout, "host=github.com") {
		t.Fatalf("stdout = %q", resp.Stdout)
	}
	if strings.Contains(resp.Stderr, brokerCanary) || !strings.Contains(resp.Stderr, "stderr-token=[REDACTED]") {
		t.Fatalf("stderr = %q", resp.Stderr)
	}
	if got := os.Getenv("GH_TOKEN"); got != "ambient-parent-token" {
		t.Fatalf("parent GH_TOKEN mutated: %q", got)
	}
}

func TestBrokerCLICancellation(t *testing.T) {
	if os.Getenv("GO_WANT_BROKER_HELPER") == "1" {
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GO_WANT_BROKER_HELPER", "1")
	b := testBroker(brokerTestConfig("github", "owner/repo", ""))
	b.lookPath = func(string) (string, error) { return exe, nil }
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	resp := b.CLI(ctx, brokercontract.CLIRequest{Executable: "gh", Arguments: []string{"-test.run=TestBrokerCLICancellation"}})
	if resp.Error == nil || resp.Error.Code != "cancelled" || resp.ExitCode == nil || *resp.ExitCode != -1 {
		t.Fatalf("response = %+v", resp)
	}
}

func TestBrokerCLIRejectsUnsafeInputsBeforeLookup(t *testing.T) {
	b := testBroker(brokerTestConfig("github", "owner/repo", ""))
	var lookups int32
	b.lookPath = func(string) (string, error) { atomic.AddInt32(&lookups, 1); return "", nil }
	for _, request := range []brokercontract.CLIRequest{
		{Executable: "/usr/bin/gh", Arguments: []string{"api", "repos"}},
		{Executable: "gh", Arguments: []string{"auth", "token"}},
		{Executable: "gh", Arguments: []string{"api", "--hostname=evil.example"}},
		{Executable: "gh", Arguments: []string{"api", "x;cat"}},
		{Executable: "gh", Arguments: []string{"api", brokerCanary}},
		{Executable: "gh", Arguments: []string{"api", base64.StdEncoding.EncodeToString([]byte(brokerCanary))}},
		{Executable: "gh", Arguments: []string{"api", "https://evil.example/x"}},
	} {
		resp := b.CLI(context.Background(), request)
		if resp.Error == nil || resp.Error.Code != "unsafe_cli" {
			t.Fatalf("request %+v response = %+v", request, resp)
		}
	}
	if lookups != 0 {
		t.Fatalf("unsafe commands reached executable lookup %d times", lookups)
	}
}

func TestBrokerEffectClassificationIsConservative(t *testing.T) {
	if got := httpEffect("POST"); got != brokercontract.EffectWriteNonIdempotent {
		t.Fatalf("POST effect = %q", got)
	}
	if got := httpEffect("PUT"); got != brokercontract.EffectWriteIdempotent {
		t.Fatalf("PUT effect = %q", got)
	}
	for _, tc := range []struct {
		args []string
		want brokercontract.Effect
	}{
		{[]string{"api", "repos/o/r/issues"}, brokercontract.EffectRead},
		{[]string{"api", "repos/o/r/issues", "-f", "title=x"}, brokercontract.EffectWriteNonIdempotent},
		{[]string{"issue", "list"}, brokercontract.EffectRead},
		{[]string{"workflow", "run", "build"}, brokercontract.EffectWriteNonIdempotent},
		{[]string{"api", "x", "--method=DELETE"}, brokercontract.EffectWriteIdempotent},
	} {
		if got := cliEffect(tc.args); got != tc.want {
			t.Errorf("cliEffect(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestBrokerInputBoundsRejectBeforeProviderAttempt(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
	}))
	defer server.Close()
	b := testBroker(brokerTestConfig("jira", "P", server.URL))
	response := b.Request(context.Background(), brokercontract.RequestRequest{Method: "POST", RelativePath: "/x", Body: strings.Repeat("x", maxBrokerOutputLimit+1)})
	if response.Error == nil || response.Error.Code != "input_too_large" || calls != 0 {
		t.Fatalf("response=%+v calls=%d", response, calls)
	}
	cli := b.CLI(context.Background(), brokercontract.CLIRequest{Executable: "gh", Arguments: []string{"api", "x"}, Stdin: strings.Repeat("x", maxBrokerOutputLimit+1)})
	if cli.Error == nil || cli.Error.Code != "input_too_large" {
		t.Fatalf("cli = %+v", cli)
	}
}
