package mockcodehost

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestScenarioBuildersCoverDeclaredBehaviors(t *testing.T) {
	scenarios := []Scenario{
		PermissionsScenario(),
		PaginationScenario(),
		RateLimitScenario(),
		ForkScenario(),
		ForcePushScenario(),
		PartialFailureScenario(),
		ChangingMergeabilityScenario(),
		OversizedDiffScenario(),
		CreatePermissionDeniedScenario(),
		CreateWriteDeniedScenario(),
		CreateLostResponseScenario(),
		CreateExternallyCompletedScenario(),
		CreateAmbiguousScenario(),
		CreateStaleHeadScenario(),
		CreateCancelledAfterApplyScenario(time.Millisecond),
		CollaborationLostResponseScenario(),
		CollaborationAmbiguousScenario(),
		CollaborationDelayedVisibilityScenario(1),
		CollaborationPermissionDeniedScenario(),
		CollaborationPermissionChangeScenario(),
		CollaborationWriteDeniedScenario(),
		CollaborationMarkerCollisionScenario(),
		CollaborationExternallyCompletedScenario("APPROVED"),
		CollaborationOldHeadReviewScenario("APPROVED"),
		CollaborationDismissedReviewScenario("APPROVED"),
		CollaborationOtherActorReviewScenario("APPROVED"),
		CollaborationUnmarkedCommentScenario("same body"),
		CollaborationMismatchedReviewStateScenario(),
		CollaborationCancelledAfterApplyScenario(time.Millisecond),
		CollaborationClosedScenario(),
	}
	seen := map[string]bool{}
	for _, scenario := range scenarios {
		if scenario.Name == "" || seen[scenario.Name] {
			t.Fatalf("invalid scenario identity %q", scenario.Name)
		}
		seen[scenario.Name] = true
	}
}

func TestPaginationForkAndForcePushAreDeterministic(t *testing.T) {
	pagination := NewServer(PaginationScenario())
	server := httptest.NewServer(pagination)
	t.Cleanup(server.Close)
	response := get(t, server.URL+"/repos/acme/widgets/pulls?per_page=100&page=1")
	if response.StatusCode != http.StatusOK || !strings.Contains(response.Header.Get("Link"), `rel="next"`) {
		t.Fatalf("pagination status=%d link=%q", response.StatusCode, response.Header.Get("Link"))
	}
	var pulls []map[string]any
	decode(t, response, &pulls)
	if len(pulls) != 1 {
		t.Fatalf("pulls=%d", len(pulls))
	}

	fork := NewServer(ForkScenario())
	forkServer := httptest.NewServer(fork)
	t.Cleanup(forkServer.Close)
	response = get(t, forkServer.URL+"/repos/acme/widgets/pulls/42")
	var pull map[string]any
	decode(t, response, &pull)
	head := pull["head"].(map[string]any)
	repository := head["repo"].(map[string]any)
	if repository["full_name"] != "contributor/widgets" {
		t.Fatalf("fork head=%v", repository)
	}

	force := NewServer(ForcePushScenario())
	forceServer := httptest.NewServer(force)
	t.Cleanup(forceServer.Close)
	first := pullHead(t, forceServer.URL)
	second := pullHead(t, forceServer.URL)
	if first != currentHeadSHA || second != forcedHeadSHA {
		t.Fatalf("force-push sequence=%q/%q", first, second)
	}
}

func TestRateLimitPermissionsAndGraphQLPartialFailures(t *testing.T) {
	rate := NewServer(RateLimitScenario())
	rateServer := httptest.NewServer(rate)
	t.Cleanup(rateServer.Close)
	response := get(t, rateServer.URL+"/repos/acme/widgets/commits/"+currentHeadSHA+"/check-runs")
	if response.StatusCode != http.StatusTooManyRequests ||
		response.Header.Get("Retry-After") != "17" ||
		response.Header.Get("X-RateLimit-Remaining") != "0" {
		t.Fatalf("rate status=%d headers=%v", response.StatusCode, response.Header)
	}
	response.Body.Close()

	permissions := NewServer(PermissionsScenario())
	permissionServer := httptest.NewServer(permissions)
	t.Cleanup(permissionServer.Close)
	response = get(t, permissionServer.URL+"/repos/acme/widgets/pulls/42/reviews")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("permission status=%d", response.StatusCode)
	}
	response.Body.Close()

	partial := NewServer(PartialFailureScenario())
	partialServer := httptest.NewServer(partial)
	t.Cleanup(partialServer.Close)
	request, err := http.NewRequest(http.MethodPost, partialServer.URL+"/graphql", bytes.NewBufferString(`{"query":"fixture"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer fixture")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Data   map[string]any   `json:"data"`
		Errors []map[string]any `json:"errors"`
	}
	decode(t, response, &body)
	if len(body.Data) == 0 || len(body.Errors) != 1 {
		t.Fatalf("partial GraphQL response=%+v", body)
	}
}

func TestOversizedDiffAndChangingMergeability(t *testing.T) {
	diff := NewServer(OversizedDiffScenario())
	diffServer := httptest.NewServer(diff)
	t.Cleanup(diffServer.Close)
	response := get(t, diffServer.URL+"/repos/acme/widgets/pulls/42/files?per_page=100&page=1")
	var files []map[string]any
	decode(t, response, &files)
	if len(files) != 100 || response.Header.Get("Link") == "" {
		t.Fatalf("oversized diff page files=%d link=%q", len(files), response.Header.Get("Link"))
	}

	changing := NewServer(ChangingMergeabilityScenario())
	changingServer := httptest.NewServer(changing)
	t.Cleanup(changingServer.Close)
	if first, second := graphMergeability(t, changingServer.URL), graphMergeability(t, changingServer.URL); first != "UNKNOWN" || second != "MERGEABLE" {
		t.Fatalf("mergeability sequence=%q/%q", first, second)
	}
}

func TestRequestInventoryNeverStoresAuthorization(t *testing.T) {
	fake := NewServer(DefaultScenario())
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	response := get(t, server.URL+"/repos/acme/widgets/pulls/42")
	response.Body.Close()
	encoded, err := json.Marshal(fake.Requests())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "Bearer") || strings.Contains(string(encoded), "fixture-token") {
		t.Fatalf("request inventory stored credentials: %s", encoded)
	}
}

func get(t *testing.T, endpoint string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer fixture-token")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func decode(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		data, _ := io.ReadAll(response.Body)
		t.Fatalf("decode: %v body=%q", err, data)
	}
}

func pullHead(t *testing.T, baseURL string) string {
	t.Helper()
	response := get(t, baseURL+"/repos/acme/widgets/pulls/42")
	var pull struct {
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	decode(t, response, &pull)
	return pull.Head.SHA
}

func graphMergeability(t *testing.T, baseURL string) string {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, baseURL+"/graphql", bytes.NewBufferString(`{"query":"fixture"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer fixture-token")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					Mergeable string `json:"mergeable"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	decode(t, response, &body)
	return body.Data.Repository.PullRequest.Mergeable
}
