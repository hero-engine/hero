package serve

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailread"
	"github.com/hero-engine/hero/internal/attention/mail"
	"github.com/hero-engine/hero/internal/attention/mailquery"
	"github.com/hero-engine/hero/internal/projectregistry"
)

func setupAttentionMailHTTP(t *testing.T, body string) (*API, *mail.Store, string, string) {
	t.Helper()
	state, projectA, projectB := t.TempDir(), t.TempDir(), t.TempDir()
	writeMCPMailProject(t, projectA, "peer_a", "A", map[string]string{"b": projectB})
	writeMCPMailProject(t, projectB, "peer_b", "B", map[string]string{"a": projectA})
	store, err := mail.NewStore(state)
	if err != nil {
		t.Fatal(err)
	}
	envelope := attention.MailEnvelope{
		SchemaVersion: attention.SchemaVersion, ID: "mail_http", ThreadID: "mail_thread",
		Recipient: attention.ProjectReference{PeerID: "peer_b", DisplayName: "B"},
		Sender:    attention.ProjectReference{PeerID: "peer_a", DisplayName: "A"},
		Subject:   "HTTP detail", Body: body, Kind: attention.MailKindRequest,
		IdempotencyKey: "delivery_http", CreatedAt: "2026-08-17T10:00:00Z",
	}
	delivery := attention.MailDelivery{
		SchemaVersion: attention.SchemaVersion, MessageID: envelope.ID, ThreadID: envelope.ThreadID,
		Sender: envelope.Sender, Recipient: envelope.Recipient,
		IdempotencyKey: envelope.IdempotencyKey, DeliveredAt: envelope.CreatedAt,
	}
	if _, _, err := store.Deliver(envelope, delivery); err != nil {
		t.Fatal(err)
	}
	registry := &projectregistry.Registry{Projects: map[string]*projectregistry.ProjectEntry{
		"a": {Path: projectA}, "b": {Path: projectB},
	}}
	service, err := mailquery.NewService(state, registry)
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(&Server{}, NewEventBus())
	api.SetMailQueryService(func() (*mailquery.Service, error) { return service, nil })
	return api, store, envelope.ID, envelope.ThreadID
}

func TestAttentionMailHTTPListIsMetadataOnlyAndDetailRoundTripsMaxBody(t *testing.T) {
	body := strings.Repeat("x", attention.MaxBodyBytes-1) + "!"
	api, store, messageID, _ := setupAttentionMailHTTP(t, body)

	listRequest := httptest.NewRequest(http.MethodGet, "/api/attention/v1/mail/messages?limit=1", nil)
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list = %d %s", listResponse.Code, listResponse.Body.String())
	}
	var list mailread.ListResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Error != nil || len(list.Items) != 1 || list.Items[0].MessageID != messageID || bytes.Contains(listResponse.Body.Bytes(), []byte(body[:128])) {
		t.Fatalf("metadata list = %#v", list)
	}

	detailRequest := httptest.NewRequest(http.MethodGet, "/api/attention/v1/mail/messages/"+messageID+"?project_peer_id=peer_b", nil)
	detailResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(detailResponse, detailRequest)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("detail = %d %s", detailResponse.Code, detailResponse.Body.String())
	}
	var detail mailread.DetailResponse
	if err := json.Unmarshal(detailResponse.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Envelope == nil || len(detail.Envelope.Body) != attention.MaxBodyBytes || detail.Envelope.Body != body || detail.Envelope.Body[len(detail.Envelope.Body)-1] != '!' {
		t.Fatalf("max body did not round trip: envelope=%v body_bytes=%d", detail.Envelope != nil, len(detail.Envelope.Body))
	}
	if _, err := store.Receipt("peer_b", messageID); !errors.Is(err, mail.ErrNotFound) {
		t.Fatalf("HTTP reads mutated receipt: %v", err)
	}
}

func TestAttentionMailHTTPActionReplyValidationAndReplay(t *testing.T) {
	api, store, messageID, threadID := setupAttentionMailHTTP(t, "body")
	actionBody := fmt.Sprintf(`{"schema_version":1,"project_peer_id":"peer_b","message_id":%q,"action_id":"mark_read","receipt_revision":0,"idempotency_key":"http_read"}`, messageID)
	post := func(path, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		return response
	}
	first := post("/api/attention/v1/mail/actions", actionBody)
	replay := post("/api/attention/v1/mail/actions", actionBody)
	if first.Code != http.StatusOK || replay.Code != http.StatusOK {
		t.Fatalf("action replay = %d %s / %d %s", first.Code, first.Body.String(), replay.Code, replay.Body.String())
	}
	var firstResult, replayResult mailread.ActionResponse
	_ = json.Unmarshal(first.Body.Bytes(), &firstResult)
	_ = json.Unmarshal(replay.Body.Bytes(), &replayResult)
	if firstResult.Receipt == nil || replayResult.Receipt == nil || firstResult.Receipt.Revision != replayResult.Receipt.Revision {
		t.Fatalf("action replay mutated twice = %#v / %#v", firstResult, replayResult)
	}

	conflictBody := fmt.Sprintf(`{"schema_version":1,"project_peer_id":"peer_b","message_id":%q,"action_id":"acknowledge","receipt_revision":0,"idempotency_key":"http_read","input":{"note":"different"}}`, messageID)
	conflict := post("/api/attention/v1/mail/actions", conflictBody)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), attention.ErrorIdempotencyConflict) {
		t.Fatalf("idempotency conflict = %d %s", conflict.Code, conflict.Body.String())
	}

	replyBody := fmt.Sprintf(`{"schema_version":1,"project_peer_id":"peer_b","message_id":%q,"thread_id":%q,"body":"reply","idempotency_key":"http_reply"}`, messageID, threadID)
	replied := post("/api/attention/v1/mail/replies", replyBody)
	replayed := post("/api/attention/v1/mail/replies", replyBody)
	if replied.Code != http.StatusOK || replayed.Code != http.StatusOK {
		t.Fatalf("reply replay = %d %s / %d %s", replied.Code, replied.Body.String(), replayed.Code, replayed.Body.String())
	}
	var replyResult, replayReplyResult mailread.ReplyResponse
	_ = json.Unmarshal(replied.Body.Bytes(), &replyResult)
	_ = json.Unmarshal(replayed.Body.Bytes(), &replayReplyResult)
	if replyResult.Delivery == nil || replayReplyResult.Delivery == nil || replyResult.Delivery.MessageID != replayReplyResult.Delivery.MessageID {
		t.Fatalf("reply replay duplicated = %#v / %#v", replyResult, replayReplyResult)
	}
	if _, err := store.Get("peer_a", replyResult.Delivery.MessageID); err != nil {
		t.Fatal(err)
	}

	for name, testCase := range map[string]struct {
		body string
		code int
	}{
		"invalid_json":    {`{`, http.StatusBadRequest},
		"multiple_values": {actionBody + `{}`, http.StatusBadRequest},
		"oversized":       {strings.Repeat(" ", (128<<10)+1), http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			response := post("/api/attention/v1/mail/actions", testCase.body)
			if response.Code != testCase.code || !strings.Contains(response.Body.String(), attention.ErrorValidation) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestAttentionMailHTTPUnavailableMissingAndContractHash(t *testing.T) {
	api := NewAPI(&Server{}, NewEventBus())
	api.SetMailQueryService(func() (*mailquery.Service, error) { return nil, errors.New("registry offline") })
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/attention/v1/mail/messages", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), attention.ErrorUnavailable) || strings.Contains(response.Body.String(), `"page"`) {
		t.Fatalf("unavailable list = %d %s", response.Code, response.Body.String())
	}

	live, _, _, _ := setupAttentionMailHTTP(t, "body")
	missing := httptest.NewRecorder()
	live.Handler().ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/attention/v1/mail/messages/mail_http?project_peer_id=peer_missing", nil))
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), attention.ErrorMissing) {
		t.Fatalf("missing identity = %d %s", missing.Code, missing.Body.String())
	}
	wrongProject := httptest.NewRecorder()
	live.Handler().ServeHTTP(wrongProject, httptest.NewRequest(http.MethodGet, "/api/attention/v1/mail/messages/mail_http?project_peer_id=peer_a", nil))
	if wrongProject.Code != http.StatusNotFound || !strings.Contains(wrongProject.Body.String(), attention.ErrorMissing) {
		t.Fatalf("wrong project = %d %s", wrongProject.Code, wrongProject.Body.String())
	}

	manifest, err := os.ReadFile("../../contracts/attention/mailread/conformance/v1/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(manifest)); got != mailread.ConformanceManifestSHA256 {
		t.Fatalf("compiled hash = %s, manifest = %s", mailread.ConformanceManifestSHA256, got)
	}
	contract := httptest.NewRecorder()
	live.Handler().ServeHTTP(contract, httptest.NewRequest(http.MethodGet, "/api/attention/v1/mail/contract", nil))
	if contract.Code != http.StatusOK || !strings.Contains(contract.Body.String(), mailread.ConformanceManifestSHA256) {
		t.Fatalf("contract = %d %s", contract.Code, contract.Body.String())
	}
	var descriptor mailread.ContractResponse
	if err := json.Unmarshal(contract.Body.Bytes(), &descriptor); err != nil {
		t.Fatal(err)
	}
	if contractErr := mailread.ValidateContractResponse(descriptor); contractErr != nil {
		t.Fatalf("invalid contract response: %#v", contractErr)
	}
}

func TestAttentionMailHTTPEmptyAndStaleCursorAreDistinct(t *testing.T) {
	emptyState := t.TempDir()
	emptyService, err := mailquery.NewService(emptyState, &projectregistry.Registry{Projects: map[string]*projectregistry.ProjectEntry{}})
	if err != nil {
		t.Fatal(err)
	}
	emptyAPI := NewAPI(&Server{}, NewEventBus())
	emptyAPI.SetMailQueryService(func() (*mailquery.Service, error) { return emptyService, nil })
	emptyResponse := httptest.NewRecorder()
	emptyAPI.Handler().ServeHTTP(emptyResponse, httptest.NewRequest(http.MethodGet, "/api/attention/v1/mail/messages", nil))
	if emptyResponse.Code != http.StatusOK || !strings.Contains(emptyResponse.Body.String(), `"items": []`) || !strings.Contains(emptyResponse.Body.String(), `"total_count": 0`) || !strings.Contains(emptyResponse.Body.String(), `"page"`) {
		t.Fatalf("empty response = %d %s", emptyResponse.Code, emptyResponse.Body.String())
	}

	api, store, messageID, _ := setupAttentionMailHTTP(t, "body")
	second := testMCPEnvelope("mail_second")
	second.IdempotencyKey = "delivery_second"
	if _, _, err := store.Deliver(second, testMCPDelivery(second)); err != nil {
		t.Fatal(err)
	}
	firstPage := httptest.NewRecorder()
	api.Handler().ServeHTTP(firstPage, httptest.NewRequest(http.MethodGet, "/api/attention/v1/mail/messages?limit=1", nil))
	var page mailread.ListResponse
	if err := json.Unmarshal(firstPage.Body.Bytes(), &page); err != nil || page.NextCursor == "" {
		t.Fatalf("first page = %d %s, %v", firstPage.Code, firstPage.Body.String(), err)
	}
	actionBody := fmt.Sprintf(`{"schema_version":1,"project_peer_id":"peer_b","message_id":%q,"action_id":"mark_read","receipt_revision":0,"idempotency_key":"stale_cursor_read"}`, messageID)
	actionResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(actionResponse, httptest.NewRequest(http.MethodPost, "/api/attention/v1/mail/actions", strings.NewReader(actionBody)))
	if actionResponse.Code != http.StatusOK {
		t.Fatalf("action = %d %s", actionResponse.Code, actionResponse.Body.String())
	}
	continuation := httptest.NewRecorder()
	path := "/api/attention/v1/mail/messages?limit=1&cursor=" + page.NextCursor
	api.Handler().ServeHTTP(continuation, httptest.NewRequest(http.MethodGet, path, nil))
	if continuation.Code != http.StatusConflict || !strings.Contains(continuation.Body.String(), attention.ErrorStale) {
		t.Fatalf("stale cursor = %d %s", continuation.Code, continuation.Body.String())
	}
}

func TestAttentionMailRoutesWinBeforeProjectRouter(t *testing.T) {
	api, _, _, _ := setupAttentionMailHTTP(t, "body")
	for _, path := range []string{
		"/api/attention/v1/mail/messages", "/api/attention/v1/mail/messages/mail_http?project_peer_id=peer_b",
		"/api/attention/v1/mail/contract",
	} {
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || strings.Contains(response.Body.String(), `project "attention" not found`) {
			t.Fatalf("route %s = %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestAttentionMailContractManifestPathExists(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", filepath.FromSlash(mailread.BundlePath), "manifest.json")); err != nil {
		t.Fatal(err)
	}
}
