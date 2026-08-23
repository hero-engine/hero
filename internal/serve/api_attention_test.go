package serve

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailthread"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/projection"
	"github.com/hero-engine/hero/internal/attention/suggestion"
)

type attentionHTTPMail struct {
	items       []mail.ListedMessage
	inboxCalls  int
	actionCalls int
}

func (f *attentionHTTPMail) Inbox(string, bool) ([]mail.ListedMessage, error) {
	f.inboxCalls++
	return f.items, nil
}
func (f *attentionHTTPMail) Action(mail.ActionRequest) (mail.ActionResult, error) {
	f.actionCalls++
	return mail.ActionResult{}, nil
}

type attentionHTTPThreadMail struct {
	attentionHTTPMail
	items []mailthread.ThreadSummary
}

func (f *attentionHTTPThreadMail) Threads(mailthread.ThreadListRequest) mailthread.ThreadListResponse {
	counts := mailthread.ThreadCounts{Total: len(f.items)}
	for _, item := range f.items {
		if item.Actionable {
			counts.Actionable++
			if item.Unread {
				counts.ActionableUnread++
			}
		}
	}
	return mailthread.ThreadListResponse{SchemaVersion: mailthread.SchemaVersion, Revision: "thread-snapshot", Counts: counts, Items: f.items}
}

func (*attentionHTTPThreadMail) ThreadAction(mailthread.ActionRequest) (mailthread.ThreadView, error) {
	return mailthread.ThreadView{}, nil
}

type attentionHTTPFocus struct{ items []focus.ListedItem }

func (f *attentionHTTPFocus) List(string) ([]focus.ListedItem, error) { return f.items, nil }
func (f *attentionHTTPFocus) Get(string) (focus.ListedItem, error) {
	return focus.ListedItem{}, focus.ErrNotFound
}
func (f *attentionHTTPFocus) Move(string, string, int64) (focus.Item, error) {
	return focus.Item{}, nil
}
func (f *attentionHTTPFocus) LaunchIntent(string) (focus.LaunchIntent, error) {
	return focus.LaunchIntent{}, nil
}

type attentionHTTPSuggestions struct{}

func (*attentionHTTPSuggestions) List(bool) ([]suggestion.Presented, error) { return nil, nil }
func (*attentionHTTPSuggestions) Act(string, string, int64, string) (suggestion.ActionResult, error) {
	return suggestion.ActionResult{}, nil
}

func TestAttentionRouteWinsWithoutSelectedProject(t *testing.T) {
	project := attention.ProjectReference{PeerID: "peer", DisplayName: "Project"}
	service := projection.NewService(
		&attentionHTTPMail{items: []mail.ListedMessage{{MailEnvelope: attention.MailEnvelope{
			ID: "mail_1", Recipient: project, Subject: "Hello", CreatedAt: "2026-07-22T18:00:00Z",
		}}}},
		&attentionHTTPFocus{},
		&attentionHTTPSuggestions{},
	)
	api := NewAPI(&Server{}, NewEventBus())
	api.SetAttentionService(func() (*projection.Service, error) { return service, nil })
	request := httptest.NewRequest(http.MethodGet, "/api/attention/v1/snapshot", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var snapshot attention.AttentionSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Rows) != 1 || snapshot.Rows[0].ID != "mail:mail_1" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestAttentionUnavailableIsStructuredAndNotEmptySnapshot(t *testing.T) {
	api := NewAPI(&Server{}, NewEventBus())
	api.SetAttentionService(func() (*projection.Service, error) {
		return nil, errors.New("private state cannot be loaded")
	})
	request := httptest.NewRequest(http.MethodGet, "/api/attention/v1/snapshot", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code": "unavailable"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), `"rows"`) {
		t.Fatalf("unavailable response masqueraded as a snapshot: %s", response.Body.String())
	}
}

func TestAttentionContractAdvertisesFixtureAndBundleChecksumsAcrossHTTPAndMCP(t *testing.T) {
	manifest, err := os.ReadFile("../../contracts/attention/testdata/v1/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(manifest)); got != attentionFixtureManifestSHA256 {
		t.Fatalf("compiled checksum = %s, manifest = %s", attentionFixtureManifestSHA256, got)
	}
	bundleManifest, err := os.ReadFile("../../contracts/attention/conformance/v1/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(bundleManifest)); got != attentionBundleManifestSHA256 {
		t.Fatalf("compiled bundle checksum = %s, manifest = %s", attentionBundleManifestSHA256, got)
	}
	api := NewAPI(&Server{}, NewEventBus())
	request := httptest.NewRequest(http.MethodGet, "/api/attention/v1/contract", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), attentionFixtureManifestSHA256) ||
		!strings.Contains(response.Body.String(), attentionBundleManifestSHA256) ||
		!strings.Contains(response.Body.String(), attentionBundlePath) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}

	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	mcpResult, err := server.toolAttentionContract(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mcpResult, attentionFixtureManifestSHA256) ||
		!strings.Contains(mcpResult, attentionBundleManifestSHA256) ||
		!strings.Contains(mcpResult, attentionBundlePath) {
		t.Fatalf("MCP contract = %s", mcpResult)
	}
}

func TestAttentionMCPReturnsCompactWindowFromHTTPProjectionAuthority(t *testing.T) {
	project := attention.ProjectReference{PeerID: "peer", DisplayName: "Project"}
	service := projection.NewService(
		&attentionHTTPMail{items: []mail.ListedMessage{{MailEnvelope: attention.MailEnvelope{
			ID: "mail_parity", Recipient: project, Subject: "Parity", Body: "Same service record",
			CreatedAt: "2026-07-22T18:00:00Z",
		}}}},
		&attentionHTTPFocus{},
		&attentionHTTPSuggestions{},
	)
	serviceFactory := func() (*projection.Service, error) { return service, nil }

	api := NewAPI(&Server{}, NewEventBus())
	api.SetAttentionService(serviceFactory)
	request := httptest.NewRequest(http.MethodGet, "/api/attention/v1/snapshot", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, body = %s", response.Code, response.Body.String())
	}
	var httpSnapshot attention.AttentionSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &httpSnapshot); err != nil {
		t.Fatal(err)
	}

	mcp := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	mcp.attentionService = serviceFactory
	mcpJSON, err := mcp.toolAttentionSnapshot(nil)
	if err != nil {
		t.Fatal(err)
	}
	var mcpSnapshot attention.AttentionSnapshot
	if err := json.Unmarshal([]byte(mcpJSON), &mcpSnapshot); err != nil {
		t.Fatal(err)
	}
	if mcpSnapshot.Revision != httpSnapshot.Revision ||
		!reflect.DeepEqual(mcpSnapshot.Counts, httpSnapshot.Counts) {
		t.Fatalf("MCP lost projection authority: HTTP = %#v MCP = %#v", httpSnapshot, mcpSnapshot)
	}
	if mcpSnapshot.Window == nil ||
		mcpSnapshot.Window.Limit != projection.DefaultAwarenessLimit ||
		mcpSnapshot.Window.Returned != 1 ||
		mcpSnapshot.Window.Truncated {
		t.Fatalf("MCP window = %#v", mcpSnapshot.Window)
	}
	if mcpSnapshot.Rows[0].Body != "" || mcpSnapshot.Rows[0].Summary != "" {
		t.Fatalf("MCP exposed Mail body content: %#v", mcpSnapshot.Rows[0])
	}
	if httpSnapshot.Rows[0].Body != "Same service record" {
		t.Fatalf("HTTP full projection was unexpectedly compacted: %#v", httpSnapshot.Rows[0])
	}
}

func TestAttentionMCPUsesThreadRowsAndActionableUnreadBadge(t *testing.T) {
	project := attention.ProjectReference{PeerID: "peer", DisplayName: "Project"}
	thread := func(id string, unread bool, revision int64) mailthread.ThreadSummary {
		return mailthread.ThreadSummary{
			Identity: mailthread.Identity{ProjectPeerID: project.PeerID, ThreadID: id}, Project: project,
			Subject: id, Kind: attention.MailKindRequest, ActivityAt: "2026-08-22T10:00:00Z",
			Unread: unread, Actionable: true, Lifecycle: mailthread.LifecycleOpen, Bucket: mailthread.BucketNeedsAttention,
			MessageCount: 2, UnreadCount: map[bool]int{true: 2, false: 0}[unread], Revision: revision,
		}
	}
	service := projection.NewService(&attentionHTTPThreadMail{items: []mailthread.ThreadSummary{thread("unread", true, 1), thread("read", false, 2)}}, &attentionHTTPFocus{}, &attentionHTTPSuggestions{})
	mcp := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	mcp.attentionService = func() (*projection.Service, error) { return service, nil }
	raw, err := mcp.toolAttentionSnapshot(nil)
	var snapshot attention.AttentionSnapshot
	if err != nil || json.Unmarshal([]byte(raw), &snapshot) != nil || len(snapshot.Rows) != 2 || snapshot.Counts.Mail != 1 || snapshot.Rows[0].Body != "" || snapshot.Rows[1].Body != "" {
		t.Fatalf("thread MCP snapshot = %#v, %v", snapshot, err)
	}
}

func TestAttentionMCPLimitValidationAndUnavailableAreStructured(t *testing.T) {
	mcp := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	for _, value := range []any{0, -1, 21, 1.5} {
		raw, err := mcp.toolAttentionSnapshot(map[string]any{"limit": value})
		if err != nil {
			t.Fatal(err)
		}
		var result attention.ActionResult
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			t.Fatal(err)
		}
		if result.Error == nil || result.Error.Code != attention.ErrorValidation || result.Error.Field != "limit" {
			t.Fatalf("limit %#v result = %#v", value, result)
		}
	}

	project := attention.ProjectReference{PeerID: "peer", DisplayName: "Project"}
	items := make([]mail.ListedMessage, 22)
	for i := range items {
		items[i] = mail.ListedMessage{MailEnvelope: attention.MailEnvelope{
			ID:        fmt.Sprintf("mail_%02d", i),
			Recipient: project,
			Subject:   fmt.Sprintf("Subject %02d", i),
			Body:      "private body",
			CreatedAt: fmt.Sprintf("2026-07-22T18:%02d:00Z", i),
		}}
	}
	mailSource := &attentionHTTPMail{items: items}
	service := projection.NewService(
		mailSource,
		&attentionHTTPFocus{},
		&attentionHTTPSuggestions{},
	)
	mcp.attentionService = func() (*projection.Service, error) { return service, nil }
	for _, testCase := range []struct {
		args  map[string]any
		limit int
	}{
		{nil, projection.DefaultAwarenessLimit},
		{map[string]any{"limit": 1}, 1},
		{map[string]any{"limit": 20}, 20},
	} {
		raw, err := mcp.toolAttentionSnapshot(testCase.args)
		if err != nil {
			t.Fatal(err)
		}
		var snapshot attention.AttentionSnapshot
		if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
			t.Fatal(err)
		}
		if snapshot.Window == nil || snapshot.Window.Limit != testCase.limit ||
			snapshot.Window.Returned != testCase.limit || !snapshot.Window.Truncated ||
			len(snapshot.Rows) != testCase.limit || snapshot.Counts.Total != len(items) {
			t.Fatalf("limit %d snapshot = %#v", testCase.limit, snapshot)
		}
		for _, row := range snapshot.Rows {
			if row.Body != "" || row.Summary != "" {
				t.Fatalf("Mail content escaped compact snapshot: %#v", row)
			}
		}
	}
	if mailSource.inboxCalls != 3 || mailSource.actionCalls != 0 {
		t.Fatalf("awareness source calls = inbox %d action %d", mailSource.inboxCalls, mailSource.actionCalls)
	}

	emptyService := projection.NewService(
		&attentionHTTPMail{},
		&attentionHTTPFocus{},
		&attentionHTTPSuggestions{},
	)
	mcp.attentionService = func() (*projection.Service, error) { return emptyService, nil }
	raw, err := mcp.toolAttentionSnapshot(nil)
	if err != nil {
		t.Fatal(err)
	}
	var empty attention.AttentionSnapshot
	if err := json.Unmarshal([]byte(raw), &empty); err != nil {
		t.Fatal(err)
	}
	if empty.Window == nil || empty.Window.State != attention.AttentionStateEmpty ||
		empty.Counts.Total != 0 {
		t.Fatalf("empty snapshot = %#v", empty)
	}

	mcp.attentionService = func() (*projection.Service, error) {
		return nil, errors.New("authority offline")
	}
	raw, err = mcp.toolAttentionSnapshot(map[string]any{"limit": 1})
	if err != nil {
		t.Fatal(err)
	}
	var unavailable attention.ActionResult
	if err := json.Unmarshal([]byte(raw), &unavailable); err != nil {
		t.Fatal(err)
	}
	if unavailable.Error == nil || unavailable.Error.Code != attention.ErrorUnavailable {
		t.Fatalf("unavailable result = %#v", unavailable)
	}
}
