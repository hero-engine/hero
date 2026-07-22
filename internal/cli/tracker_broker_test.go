package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	brokercontract "github.com/hero-engine/hero/contracts/trackerbroker"
)

func TestTrackerContractCommandReturnsConsumerFixture(t *testing.T) {
	var out bytes.Buffer
	trackerContractCmd.SetOut(&out)
	defer trackerContractCmd.SetOut(nil)
	if err := trackerContractCmd.RunE(trackerContractCmd, nil); err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out.Bytes(), &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != brokercontract.Version {
		t.Fatalf("version = %q", fixture.Version)
	}
}

func TestTrackerBrokerCommandReturnsStructuredInvalidInput(t *testing.T) {
	var out bytes.Buffer
	trackerBrokerCmd.SetIn(strings.NewReader(`{"unknown":true}`))
	trackerBrokerCmd.SetOut(&out)
	defer trackerBrokerCmd.SetIn(nil)
	defer trackerBrokerCmd.SetOut(nil)
	if err := trackerBrokerCmd.RunE(trackerBrokerCmd, []string{"get_issue"}); err != nil {
		t.Fatal(err)
	}
	var response brokercontract.Response
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Version != brokercontract.Version || response.Operation != brokercontract.OperationGetIssue || response.Error == nil || response.Error.Code != "invalid_input" {
		t.Fatalf("response = %+v", response)
	}
}

func TestTrackerBrokerCommandGetsACME103EvidenceThroughConfiguredConnection(t *testing.T) {
	var providerCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&providerCalls, 1)
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Errorf("tracker credential was not injected")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/3/field":
			fmt.Fprint(w, `[]`)
		case "/rest/api/3/issue/ACME-103":
			fmt.Fprint(w, `{"key":"ACME-103","fields":{"summary":"CLI evidence","description":"Full description","status":{"name":"Open"}},"names":{"summary":"Summary"},"changelog":{"histories":[]}}`)
		case "/rest/api/3/issue/ACME-103/comment":
			fmt.Fprint(w, `{"startAt":0,"maxResults":100,"total":0,"comments":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shared := fmt.Sprintf(`{"folder":".hero","integrations":{"default":"fixture-jira","connections":{"fixture-jira":{"provider":"jira","settings":{"project":"CONFIGURED","base_url":%q,"user_email":"fixture@example.com"}}}}}`, server.URL)
	local := `{"integrations":{"connections":{"fixture-jira":{"auth":{"token":"CLI-BROKER-CANARY"}}}}}`
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), []byte(shared), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.local.json"), []byte(local), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	var out bytes.Buffer
	trackerBrokerCmd.SetIn(strings.NewReader(`{"connection_id":"fixture-jira","issue_id":"ACME-103","detail":"evidence"}`))
	trackerBrokerCmd.SetOut(&out)
	trackerBrokerCmd.SetContext(context.Background())
	defer trackerBrokerCmd.SetIn(nil)
	defer trackerBrokerCmd.SetOut(nil)
	if err := trackerBrokerCmd.RunE(trackerBrokerCmd, []string{"get_issue"}); err != nil {
		t.Fatal(err)
	}

	var response brokercontract.Response
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Version != brokercontract.Version || response.Operation != brokercontract.OperationGetIssue || response.Error != nil {
		t.Fatalf("response = %+v", response)
	}
	if bytes.Contains(out.Bytes(), []byte("CLI-BROKER-CANARY")) {
		t.Fatal("broker response exposed the configured credential")
	}
	var result struct {
		IssueID string `json:"issue_id"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.IssueID != "ACME-103" {
		t.Fatalf("result = %+v", result)
	}

	callsAfterSuccess := atomic.LoadInt32(&providerCalls)
	for name, issueID := range map[string]string{
		"carriage": "ACME\r103",
		"newline":  "ACME\n103",
		"nul":      "ACME\x00103",
	} {
		t.Run("rejects_"+name+"_before_provider", func(t *testing.T) {
			payload, err := json.Marshal(map[string]string{
				"connection_id": "fixture-jira",
				"issue_id":      issueID,
				"detail":        "evidence",
			})
			if err != nil {
				t.Fatal(err)
			}
			out.Reset()
			trackerBrokerCmd.SetIn(bytes.NewReader(payload))
			if err := trackerBrokerCmd.RunE(trackerBrokerCmd, []string{"get_issue"}); err != nil {
				t.Fatal(err)
			}
			var rejected brokercontract.Response
			if err := json.Unmarshal(out.Bytes(), &rejected); err != nil {
				t.Fatal(err)
			}
			if rejected.Error == nil || rejected.Error.Code != "invalid_issue_id" {
				t.Fatalf("response = %+v", rejected)
			}
		})
	}
	if got := atomic.LoadInt32(&providerCalls); got != callsAfterSuccess {
		t.Fatalf("unsafe CLI issue IDs reached provider: calls %d -> %d", callsAfterSuccess, got)
	}
}
