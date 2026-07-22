package tracker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJiraEvidenceContextCancelsInFlightRequests(t *testing.T) {
	for _, stage := range []string{"metadata", "comments", "attachment"} {
		t.Run(stage, func(t *testing.T) {
			started := make(chan struct{})
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				block := stage == "metadata" && request.URL.Query().Get("fields") == "key,updated"
				block = block || stage == "comments" && request.URL.Path == "/rest/api/3/issue/MORPH-297/comment"
				block = block || stage == "attachment" && request.URL.Path == "/attachment/9"
				if block {
					close(started)
					<-request.Context().Done()
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch request.URL.Path {
				case "/rest/api/3/field":
					fmt.Fprint(w, `[]`)
				case "/rest/api/3/issue/MORPH-297":
					fmt.Fprint(w, `{"key":"MORPH-297","fields":{"summary":"cancel test","updated":"2026-07-21T12:00:00.000-0600","status":{"name":"Open"}},"names":{},"changelog":{"histories":[]}}`)
				default:
					http.NotFound(w, request)
				}
			}))
			defer server.Close()
			provider := &jira{baseURL: server.URL, projectKey: "MORPH", token: "canary", userEmail: "fixture@example.com", client: server.Client()}
			ctx, cancel := context.WithCancel(context.Background())
			result := make(chan error, 1)
			go func() {
				switch stage {
				case "metadata":
					_, err := provider.GetEvidenceMetadataContext(ctx, "MORPH-297")
					result <- err
				case "comments":
					_, err := provider.GetIssueEvidenceContext(ctx, "MORPH-297")
					result <- err
				case "attachment":
					_, err := provider.DownloadEvidenceAttachmentContext(ctx, server.URL+"/attachment/9")
					result <- err
				}
			}()
			<-started
			cancel()
			if err := <-result; !errors.Is(err, context.Canceled) {
				t.Fatalf("%s error = %v, want context cancellation", stage, err)
			}
		})
	}
}
