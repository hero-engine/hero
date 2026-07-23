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
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/projection"
	"github.com/hero-engine/hero/internal/attention/suggestion"
)

type attentionHTTPMail struct{ items []mail.ListedMessage }

func (f *attentionHTTPMail) Inbox(string, bool) ([]mail.ListedMessage, error) {
	return f.items, nil
}
func (f *attentionHTTPMail) Action(mail.ActionRequest) (mail.ActionResult, error) {
	return mail.ActionResult{}, nil
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

func TestAttentionContractAdvertisesFixtureManifestChecksum(t *testing.T) {
	manifest, err := os.ReadFile("../../contracts/attention/testdata/v1/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(manifest)); got != attentionFixtureManifestSHA256 {
		t.Fatalf("compiled checksum = %s, manifest = %s", attentionFixtureManifestSHA256, got)
	}
	api := NewAPI(&Server{}, NewEventBus())
	request := httptest.NewRequest(http.MethodGet, "/api/attention/v1/contract", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), attentionFixtureManifestSHA256) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestAttentionHTTPAndMCPReturnTheSameProjectionRecords(t *testing.T) {
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
	// generated_at is intentionally volatile and excluded from revision
	// identity; semantic records and opaque revisions must remain identical.
	httpSnapshot.GeneratedAt = ""
	mcpSnapshot.GeneratedAt = ""
	httpCanonical, _ := json.Marshal(httpSnapshot)
	mcpCanonical, _ := json.Marshal(mcpSnapshot)
	var httpValue, mcpValue any
	_ = json.Unmarshal(httpCanonical, &httpValue)
	_ = json.Unmarshal(mcpCanonical, &mcpValue)
	if !reflect.DeepEqual(httpValue, mcpValue) {
		t.Fatalf("HTTP = %#v\nMCP = %#v", httpSnapshot, mcpSnapshot)
	}
}
