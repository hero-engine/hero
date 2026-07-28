// Package mockcodehost provides a deterministic, offline GitHub protocol fake
// for the code-host broker.
package mockcodehost

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	currentHeadSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	forcedHeadSHA  = "cccccccccccccccccccccccccccccccccccccccc"
	baseSHA        = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	releaseSHA     = "dddddddddddddddddddddddddddddddddddddddd"
)

// Scenario controls one deterministic provider behavior.
type Scenario struct {
	Name                            string
	ForcedPageSize                  int
	DeniedSections                  map[string]int
	RateLimitedSections             map[string]int
	ForkHead                        bool
	ForcePush                       bool
	GraphQLPartial                  bool
	MergeabilitySequence            []string
	OversizedDiff                   bool
	CrossOriginNext                 string
	Delay                           time.Duration
	CreatePermissionDenied          bool
	CreateWriteDenied               bool
	CreateLostResponse              bool
	CreateExternallyCompleted       bool
	CreateAmbiguousReadback         bool
	CreateStaleHead                 bool
	CreateResponseDelay             time.Duration
	CollaborationLostResponse       bool
	CollaborationAmbiguousReadback  bool
	CollaborationVisibilityDelay    int
	CollaborationPermissionSequence []bool
	CollaborationWriteDenied        bool
	CollaborationCollisionOnly      bool
	CollaborationExternalState      string
	CollaborationExternalOldHead    bool
	CollaborationExternalDismissed  bool
	CollaborationExternalOtherActor bool
	CollaborationUnmarkedComment    string
	CollaborationForcedReviewState  string
	CollaborationResponseDelay      time.Duration
	CollaborationClosed             bool
	StateEnabled                    bool
	StateOperation                  string
	StateExternallyCompleted        bool
	StateLostResponse               bool
	StateAmbiguousReadback          bool
	StateVisibilityDelay            int
	StatePermissionSequence         []bool
	StateWriteDenied                bool
	StateMissingTargetBranch        bool
	StateTargetMoves                bool
	StateTargetMovesAfterWrite      bool
	StateBaseChanges                bool
	StateMerged                     bool
	StateResponseDelay              time.Duration
}

func DefaultScenario() Scenario { return Scenario{Name: "default"} }

func PermissionsScenario() Scenario {
	return Scenario{
		Name:           "permissions",
		DeniedSections: map[string]int{"reviews": http.StatusForbidden},
	}
}

func PaginationScenario() Scenario {
	return Scenario{Name: "pagination", ForcedPageSize: 1}
}

func RateLimitScenario() Scenario {
	return Scenario{
		Name:                "rate-limit",
		RateLimitedSections: map[string]int{"checks": 17},
	}
}

func ForkScenario() Scenario {
	return Scenario{Name: "fork", ForkHead: true}
}

func ForcePushScenario() Scenario {
	return Scenario{Name: "force-push", ForcePush: true}
}

func PartialFailureScenario() Scenario {
	return Scenario{Name: "partial-graphql", GraphQLPartial: true}
}

func ChangingMergeabilityScenario() Scenario {
	return Scenario{Name: "changing-mergeability", MergeabilitySequence: []string{"UNKNOWN", "MERGEABLE"}}
}

func OversizedDiffScenario() Scenario {
	return Scenario{Name: "oversized-diff", OversizedDiff: true}
}

func CancellationScenario(delay time.Duration) Scenario {
	return Scenario{Name: "cancellation", Delay: delay}
}

func CreatePermissionDeniedScenario() Scenario {
	return Scenario{Name: "create-permission-denied", CreatePermissionDenied: true}
}

func CreateWriteDeniedScenario() Scenario {
	return Scenario{Name: "create-write-denied", CreateWriteDenied: true}
}

func CreateLostResponseScenario() Scenario {
	return Scenario{Name: "create-lost-response", CreateLostResponse: true}
}

func CreateExternallyCompletedScenario() Scenario {
	return Scenario{Name: "create-externally-completed", CreateExternallyCompleted: true}
}

func CreateAmbiguousScenario() Scenario {
	return Scenario{Name: "create-ambiguous", CreateLostResponse: true, CreateAmbiguousReadback: true}
}

func CreateStaleHeadScenario() Scenario {
	return Scenario{Name: "create-stale-head", CreateStaleHead: true}
}

func CreateCancelledAfterApplyScenario(delay time.Duration) Scenario {
	return Scenario{Name: "create-cancelled-after-apply", CreateResponseDelay: delay}
}

func CollaborationLostResponseScenario() Scenario {
	return Scenario{Name: "collaboration-lost-response", CollaborationLostResponse: true}
}

func CollaborationAmbiguousScenario() Scenario {
	return Scenario{Name: "collaboration-ambiguous", CollaborationLostResponse: true, CollaborationAmbiguousReadback: true}
}

func CollaborationDelayedVisibilityScenario(reads int) Scenario {
	return Scenario{Name: "collaboration-delayed-visibility", CollaborationLostResponse: true, CollaborationVisibilityDelay: reads}
}

func CollaborationPermissionDeniedScenario() Scenario {
	return Scenario{Name: "collaboration-permission-denied", CollaborationPermissionSequence: []bool{false}}
}

func CollaborationPermissionChangeScenario() Scenario {
	return Scenario{Name: "collaboration-permission-change", CollaborationPermissionSequence: []bool{true, false}}
}

func CollaborationWriteDeniedScenario() Scenario {
	return Scenario{Name: "collaboration-write-denied", CollaborationWriteDenied: true}
}

func CollaborationMarkerCollisionScenario() Scenario {
	return Scenario{Name: "collaboration-marker-collision", CollaborationLostResponse: true, CollaborationCollisionOnly: true}
}

func CollaborationExternallyCompletedScenario(state string) Scenario {
	return Scenario{Name: "collaboration-external-" + strings.ToLower(state), CollaborationExternalState: state}
}

func CollaborationOldHeadReviewScenario(state string) Scenario {
	return Scenario{Name: "collaboration-old-head-" + strings.ToLower(state), CollaborationExternalState: state, CollaborationExternalOldHead: true}
}

func CollaborationDismissedReviewScenario(state string) Scenario {
	return Scenario{Name: "collaboration-dismissed-" + strings.ToLower(state), CollaborationExternalState: state, CollaborationExternalDismissed: true}
}

func CollaborationOtherActorReviewScenario(state string) Scenario {
	return Scenario{Name: "collaboration-other-actor-" + strings.ToLower(state), CollaborationExternalState: state, CollaborationExternalOtherActor: true}
}

func CollaborationUnmarkedCommentScenario(body string) Scenario {
	return Scenario{Name: "collaboration-unmarked-comment", CollaborationUnmarkedComment: body}
}

func CollaborationMismatchedReviewStateScenario() Scenario {
	return Scenario{
		Name:                           "collaboration-mismatched-review-state",
		CollaborationLostResponse:      true,
		CollaborationForcedReviewState: "COMMENTED",
	}
}

func CollaborationCancelledAfterApplyScenario(delay time.Duration) Scenario {
	return Scenario{Name: "collaboration-cancelled-after-apply", CollaborationResponseDelay: delay}
}

func CollaborationClosedScenario() Scenario {
	return Scenario{Name: "collaboration-closed", CollaborationClosed: true}
}

func StateScenario(operation string) Scenario {
	return Scenario{Name: "state-" + operation, StateEnabled: true, StateOperation: operation}
}

func StateExternallyCompletedScenario(operation string) Scenario {
	scenario := StateScenario(operation)
	scenario.Name += "-external"
	scenario.StateExternallyCompleted = true
	return scenario
}

func StateLostResponseScenario(operation string) Scenario {
	scenario := StateScenario(operation)
	scenario.Name += "-lost-response"
	scenario.StateLostResponse = true
	return scenario
}

func StateAmbiguousScenario(operation string) Scenario {
	scenario := StateLostResponseScenario(operation)
	scenario.Name = "state-" + operation + "-ambiguous"
	scenario.StateAmbiguousReadback = true
	return scenario
}

func StateDelayedVisibilityScenario(operation string, reads int) Scenario {
	scenario := StateLostResponseScenario(operation)
	scenario.Name = "state-" + operation + "-delayed-visibility"
	scenario.StateVisibilityDelay = reads
	return scenario
}

func StatePermissionRaceScenario(operation string) Scenario {
	scenario := StateScenario(operation)
	scenario.Name += "-permission-race"
	scenario.StatePermissionSequence = []bool{true, false}
	return scenario
}

func StateMissingTargetBranchScenario() Scenario {
	scenario := StateScenario("retarget")
	scenario.Name = "state-retarget-missing-target"
	scenario.StateMissingTargetBranch = true
	return scenario
}

func StateTargetMovesScenario() Scenario {
	scenario := StateScenario("retarget")
	scenario.Name = "state-retarget-target-moves"
	scenario.StateTargetMoves = true
	return scenario
}

func StateTargetMovesAfterWriteScenario() Scenario {
	scenario := StateScenario("retarget")
	scenario.Name = "state-retarget-target-moves-after-write"
	scenario.StateTargetMovesAfterWrite = true
	return scenario
}

func StateBaseChangesScenario() Scenario {
	scenario := StateScenario("retarget")
	scenario.Name = "state-retarget-base-changes"
	scenario.StateBaseChanges = true
	return scenario
}

func StateForcePushScenario(operation string) Scenario {
	scenario := StateScenario(operation)
	scenario.Name += "-force-push"
	scenario.ForcePush = true
	return scenario
}

func StateMergedScenario(operation string) Scenario {
	scenario := StateScenario(operation)
	scenario.Name += "-merged"
	scenario.StateMerged = true
	return scenario
}

func StateWriteDeniedScenario(operation string) Scenario {
	scenario := StateScenario(operation)
	scenario.Name += "-write-denied"
	scenario.StateWriteDenied = true
	return scenario
}

func StateCancelledAfterApplyScenario(operation string, delay time.Duration) Scenario {
	scenario := StateScenario(operation)
	scenario.Name += "-cancelled-after-apply"
	scenario.StateResponseDelay = delay
	return scenario
}

// Request records only non-sensitive request metadata.
type Request struct {
	Method string
	Path   string
	Query  string
}

// Server implements the GitHub REST and GraphQL routes used by codehost.
type Server struct {
	scenario              Scenario
	mu                    sync.Mutex
	requests              []Request
	prReads               int
	graphQL               int
	created               []map[string]any
	createAttempts        int
	collaborationComments []map[string]any
	collaborationReviews  []map[string]any
	collaborationAttempts int
	collaborationReads    int
	permissionReads       int
	stateCurrent          lifecycleState
	statePending          *lifecycleState
	stateAttempts         int
	stateReads            int
	statePermissionReads  int
	stateTargetReads      int
}

type lifecycleState struct {
	State     string
	Draft     bool
	Merged    bool
	Base      string
	BaseSHA   string
	UpdatedAt string
}

// NewServer returns a freshly initialized deterministic fake.
func NewServer(scenario Scenario) *Server {
	if scenario.Name == "" {
		scenario.Name = "default"
	}
	server := &Server{scenario: scenario}
	server.stateCurrent = lifecycleState{
		State: "open", Base: "main", BaseSHA: baseSHA, UpdatedAt: "2026-07-27T20:30:00Z",
	}
	if scenario.StateEnabled {
		switch scenario.StateOperation {
		case "mark_ready":
			server.stateCurrent.Draft = true
		case "reopen":
			server.stateCurrent.State = "closed"
		}
		if scenario.StateExternallyCompleted {
			server.stateCurrent = desiredLifecycleState(server.stateCurrent, scenario.StateOperation)
		}
		if scenario.StateMerged {
			server.stateCurrent.State = "closed"
			server.stateCurrent.Merged = true
		}
	}
	if scenario.CreateExternallyCompleted {
		server.created = append(server.created, server.createdPullRequest("acme", "widgets", "acme", "feature/create", 45, "Create broker PR", "CREATE-BODY-CANARY create body", false))
	}
	if scenario.CollaborationExternalState != "" {
		head := currentHeadSHA
		if scenario.CollaborationExternalOldHead {
			head = baseSHA
		}
		actor := "hero-user"
		actorID := int64(99)
		if scenario.CollaborationExternalOtherActor {
			actor, actorID = "other-reviewer", 98
		}
		server.collaborationReviews = append(server.collaborationReviews, map[string]any{
			"id": 90, "node_id": "REVIEW_90", "user": user(actor, actorID), "body": "external review",
			"state": strings.ToUpper(scenario.CollaborationExternalState), "commit_id": head, "submitted_at": "2026-07-27T20:25:00Z",
		})
		if scenario.CollaborationExternalDismissed {
			server.collaborationReviews = append(server.collaborationReviews, map[string]any{
				"id": 91, "node_id": "REVIEW_91", "user": user(actor, actorID), "body": "dismissed review",
				"state": "DISMISSED", "commit_id": head, "submitted_at": "2026-07-27T20:26:00Z",
			})
		}
	}
	if scenario.CollaborationUnmarkedComment != "" {
		server.collaborationComments = append(server.collaborationComments, map[string]any{
			"id": 92, "node_id": "COMMENT_92", "user": user("hero-user", 99), "body": scenario.CollaborationUnmarkedComment,
			"html_url": "https://example.invalid/comment/92", "created_at": "2026-07-27T20:27:00Z", "updated_at": "2026-07-27T20:27:00Z",
		})
	}
	return server
}

// Requests returns a credential-free snapshot of observed provider calls.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

func (s *Server) RequestCount() int {
	return len(s.Requests())
}

func (s *Server) CreateAttempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createAttempts
}

func (s *Server) CollaborationAttempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.collaborationAttempts
}

func (s *Server) CollaborationBodies() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	bodies := make([]string, 0, len(s.collaborationComments)+len(s.collaborationReviews))
	for _, item := range s.collaborationComments {
		bodies = append(bodies, fmt.Sprint(item["body"]))
	}
	for _, item := range s.collaborationReviews {
		bodies = append(bodies, fmt.Sprint(item["body"]))
	}
	return bodies
}

func (s *Server) StateAttempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stateAttempts
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.record(request)
	if strings.TrimSpace(request.Header.Get("Authorization")) == "" {
		writeError(writer, http.StatusUnauthorized, "missing credential")
		return
	}
	if s.scenario.Delay > 0 {
		select {
		case <-request.Context().Done():
			return
		case <-time.After(s.scenario.Delay):
		}
	}
	setRateHeaders(writer, "core", 5000, 4999, 0)

	path := request.URL.Path
	if path == "/user" && request.Method == http.MethodGet {
		writeJSON(writer, http.StatusOK, user("hero-user", 99))
		return
	}
	if strings.HasSuffix(path, "/graphql") {
		s.handleGraphQL(writer, request)
		return
	}
	if index := strings.Index(path, "/repos/"); index >= 0 {
		path = path[index:]
	}
	if strings.HasPrefix(path, "/repos/") {
		s.handleRepository(writer, request, path)
		return
	}
	writeError(writer, http.StatusNotFound, "endpoint not implemented")
}

func (s *Server) record(request *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, Request{
		Method: request.Method,
		Path:   request.URL.Path,
		Query:  request.URL.RawQuery,
	})
}

func (s *Server) handleRepository(writer http.ResponseWriter, request *http.Request, path string) {
	parts := splitPath(path)
	if len(parts) < 3 || parts[0] != "repos" {
		writeError(writer, http.StatusNotFound, "repository endpoint not implemented")
		return
	}
	owner, repository := parts[1], parts[2]
	switch {
	case len(parts) == 3 && request.Method == http.MethodGet:
		s.getRepository(writer)
	case len(parts) == 4 && parts[3] == "pulls" && request.Method == http.MethodGet:
		s.listPullRequests(writer, request, owner, repository)
	case len(parts) == 4 && parts[3] == "pulls" && request.Method == http.MethodPost:
		s.createPullRequest(writer, request, owner, repository)
	case len(parts) == 5 && parts[3] == "pulls" && request.Method == http.MethodGet:
		s.getPullRequest(writer, request, owner, repository, parts[4])
	case len(parts) == 5 && parts[3] == "pulls" && request.Method == http.MethodPatch:
		s.updatePullRequest(writer, request, owner, repository, parts[4])
	case len(parts) >= 7 && parts[3] == "git" && parts[4] == "ref" && parts[5] == "heads" && request.Method == http.MethodGet:
		s.getRef(writer, owner, repository, strings.Join(parts[6:], "/"))
	case len(parts) == 6 && parts[3] == "pulls" && parts[5] == "commits" && request.Method == http.MethodGet:
		s.listCommits(writer, request)
	case len(parts) == 6 && parts[3] == "pulls" && parts[5] == "files" && request.Method == http.MethodGet:
		s.listDiffFiles(writer, request)
	case len(parts) == 6 && parts[3] == "pulls" && parts[5] == "reviews" && request.Method == http.MethodGet:
		s.listReviews(writer, request)
	case len(parts) == 6 && parts[3] == "pulls" && parts[5] == "reviews" && request.Method == http.MethodPost:
		s.createReview(writer, request)
	case len(parts) == 6 && parts[3] == "issues" && parts[5] == "comments" && request.Method == http.MethodGet:
		s.listComments(writer, request)
	case len(parts) == 6 && parts[3] == "issues" && parts[5] == "comments" && request.Method == http.MethodPost:
		s.createComment(writer, request)
	case len(parts) == 6 && parts[3] == "commits" && parts[5] == "check-runs" && request.Method == http.MethodGet:
		s.listChecks(writer, request)
	default:
		writeError(writer, http.StatusNotFound, "repository endpoint not implemented")
	}
}

func (s *Server) getRepository(writer http.ResponseWriter) {
	allowed := true
	s.mu.Lock()
	if s.scenario.StateEnabled && len(s.scenario.StatePermissionSequence) > 0 {
		index := min(s.statePermissionReads, len(s.scenario.StatePermissionSequence)-1)
		allowed = s.scenario.StatePermissionSequence[index]
		s.statePermissionReads++
	} else if len(s.scenario.CollaborationPermissionSequence) > 0 {
		index := min(s.permissionReads, len(s.scenario.CollaborationPermissionSequence)-1)
		allowed = s.scenario.CollaborationPermissionSequence[index]
		s.permissionReads++
	}
	s.mu.Unlock()
	writeJSON(writer, http.StatusOK, map[string]any{
		"permissions": map[string]bool{"pull": !s.scenario.CreatePermissionDenied && allowed, "push": allowed},
	})
}

func (s *Server) getRef(writer http.ResponseWriter, owner, repository, name string) {
	sha := currentHeadSHA
	if name == "main" {
		sha = baseSHA
	} else if name == "release" {
		if s.scenario.StateMissingTargetBranch {
			writeError(writer, http.StatusNotFound, "fixture target branch missing")
			return
		}
		s.mu.Lock()
		s.stateTargetReads++
		moved := s.scenario.StateTargetMoves && s.stateTargetReads > 1
		s.mu.Unlock()
		sha = releaseSHA
		if moved {
			sha = forcedHeadSHA
		}
	} else if s.scenario.CreateStaleHead {
		sha = forcedHeadSHA
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ref":        "refs/heads/" + name,
		"object":     map[string]any{"sha": sha},
		"repository": owner + "/" + repository,
	})
}

func (s *Server) listPullRequests(writer http.ResponseWriter, request *http.Request, owner, repository string) {
	items := []map[string]any{
		s.pullRequest(owner, repository, 42, currentHeadSHA),
		s.pullRequest(owner, repository, 43, currentHeadSHA),
		s.pullRequest(owner, repository, 44, currentHeadSHA),
	}
	s.mu.Lock()
	for _, pullRequest := range s.created {
		items = append(items, pullRequest)
	}
	s.mu.Unlock()
	if head := request.URL.Query().Get("head"); head != "" {
		if s.scenario.CreateAmbiguousReadback && s.CreateAttempts() > 0 {
			writeError(writer, http.StatusServiceUnavailable, "fixture creation read-back unavailable")
			return
		}
		base := request.URL.Query().Get("base")
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			headValue, _ := item["head"].(map[string]any)
			headRepo, _ := headValue["repo"].(map[string]any)
			headOwner, _ := headRepo["owner"].(map[string]any)
			baseValue, _ := item["base"].(map[string]any)
			if fmt.Sprint(headOwner["login"])+":"+fmt.Sprint(headValue["ref"]) == head &&
				(base == "" || fmt.Sprint(baseValue["ref"]) == base) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	start, end, hasNext, page, perPage := s.page(items, request.URL.Query())
	if hasNext {
		s.setNext(writer, request, page+1, perPage)
	}
	writeJSON(writer, http.StatusOK, items[start:end])
}

func (s *Server) createPullRequest(writer http.ResponseWriter, request *http.Request, owner, repository string) {
	if s.scenario.CreatePermissionDenied || s.scenario.CreateWriteDenied {
		s.mu.Lock()
		s.createAttempts++
		s.mu.Unlock()
		writeError(writer, http.StatusForbidden, "fixture creation denied")
		return
	}
	var payload struct {
		Title string `json:"title"`
		Head  string `json:"head"`
		Base  string `json:"base"`
		Body  string `json:"body"`
		Draft bool   `json:"draft"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.Title == "" || payload.Head == "" || payload.Base == "" {
		writeError(writer, http.StatusUnprocessableEntity, "invalid creation payload")
		return
	}
	headOwner, headRef, ok := strings.Cut(payload.Head, ":")
	if !ok || headOwner == "" || headRef == "" {
		writeError(writer, http.StatusUnprocessableEntity, "owner-qualified head required")
		return
	}
	s.mu.Lock()
	s.createAttempts++
	number := int64(100 + len(s.created))
	pullRequest := s.createdPullRequest(owner, repository, headOwner, headRef, number, payload.Title, payload.Body, payload.Draft)
	s.created = append(s.created, pullRequest)
	s.mu.Unlock()
	if s.scenario.CreateResponseDelay > 0 {
		select {
		case <-request.Context().Done():
			return
		case <-time.After(s.scenario.CreateResponseDelay):
		}
	}
	if s.scenario.CreateLostResponse {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"incomplete":`))
		return
	}
	writeJSON(writer, http.StatusCreated, pullRequest)
}

func (s *Server) getPullRequest(writer http.ResponseWriter, _ *http.Request, owner, repository, number string) {
	value, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		writeError(writer, http.StatusNotFound, "pull request not found")
		return
	}
	head := currentHeadSHA
	s.mu.Lock()
	s.prReads++
	reads := s.prReads
	unavailable := s.scenario.StateEnabled && s.scenario.StateAmbiguousReadback && s.stateAttempts > 0
	if s.scenario.ForcePush && s.prReads > 1 {
		head = forcedHeadSHA
	}
	s.mu.Unlock()
	if unavailable {
		writeError(writer, http.StatusServiceUnavailable, "fixture state read-back unavailable")
		return
	}
	writeJSON(writer, http.StatusOK, s.pullRequestAtRead(owner, repository, value, head, reads))
}

func (s *Server) pullRequest(owner, repository string, number int64, headSHA string) map[string]any {
	return s.pullRequestAtRead(owner, repository, number, headSHA, 0)
}

func (s *Server) pullRequestAtRead(owner, repository string, number int64, headSHA string, reads int) map[string]any {
	headOwner := owner
	if s.scenario.ForkHead {
		headOwner = "contributor"
	}
	state := "open"
	draft := false
	merged := false
	base := "main"
	baseValueSHA := baseSHA
	updatedAt := "2026-07-27T20:30:00Z"
	if s.scenario.CollaborationClosed {
		state = "closed"
	}
	if s.scenario.StateEnabled {
		s.mu.Lock()
		if s.statePending != nil && reads > 0 {
			s.stateReads++
			if s.stateReads > s.scenario.StateVisibilityDelay {
				s.stateCurrent = *s.statePending
				s.statePending = nil
			}
		}
		current := s.stateCurrent
		s.mu.Unlock()
		state, draft, merged = current.State, current.Draft, current.Merged
		base, baseValueSHA, updatedAt = current.Base, current.BaseSHA, current.UpdatedAt
		if s.scenario.StateBaseChanges && reads > 1 {
			base, baseValueSHA = "develop", forcedHeadSHA
		}
	}
	mergedAt := ""
	if merged {
		mergedAt = "2026-07-27T20:35:00Z"
	}
	return map[string]any{
		"id":         1000 + number,
		"node_id":    fmt.Sprintf("PR_%d", number),
		"number":     number,
		"title":      fmt.Sprintf("Pull request %d", number),
		"body":       "deterministic code-host fixture",
		"html_url":   fmt.Sprintf("https://example.invalid/%s/%s/pull/%d", owner, repository, number),
		"state":      state,
		"draft":      draft,
		"merged":     merged,
		"created_at": "2026-07-27T20:00:00Z",
		"updated_at": updatedAt,
		"merged_at":  mergedAt,
		"user":       user("contributor", 7),
		"base": map[string]any{
			"ref":  base,
			"sha":  baseValueSHA,
			"repo": repositoryValue(owner, repository, 1),
		},
		"head": map[string]any{
			"ref":  "feature/code-host",
			"sha":  headSHA,
			"repo": repositoryValue(headOwner, repository, 2),
		},
	}
}

func (s *Server) createdPullRequest(owner, repository, headOwner, headRef string, number int64, title, body string, draft bool) map[string]any {
	return map[string]any{
		"id":         1000 + number,
		"node_id":    fmt.Sprintf("PR_%d", number),
		"number":     number,
		"title":      title,
		"body":       body,
		"html_url":   fmt.Sprintf("https://example.invalid/%s/%s/pull/%d", owner, repository, number),
		"state":      "open",
		"draft":      draft,
		"merged":     false,
		"created_at": "2026-07-27T20:00:00Z",
		"updated_at": "2026-07-27T20:30:00Z",
		"merged_at":  "",
		"user":       user("contributor", 7),
		"base": map[string]any{
			"ref": "main", "sha": baseSHA, "repo": repositoryValue(owner, repository, 1),
		},
		"head": map[string]any{
			"ref": headRef, "sha": currentHeadSHA, "repo": repositoryValue(headOwner, repository, 2),
		},
	}
}

func (s *Server) updatePullRequest(writer http.ResponseWriter, request *http.Request, _, _ string, number string) {
	if number != "42" {
		writeError(writer, http.StatusNotFound, "pull request not found")
		return
	}
	var payload struct {
		State string `json:"state"`
		Base  string `json:"base"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil ||
		(payload.State == "" && payload.Base == "") ||
		(payload.State != "" && payload.Base != "") {
		writeError(writer, http.StatusUnprocessableEntity, "invalid state-transition payload")
		return
	}
	operation := "retarget"
	if payload.State == "closed" {
		operation = "close"
	} else if payload.State == "open" {
		operation = "reopen"
	} else if payload.State != "" || payload.Base != "release" {
		writeError(writer, http.StatusUnprocessableEntity, "unsupported state transition")
		return
	}
	if !s.applyStateTransition(writer, operation) {
		return
	}
	if s.finishStateWrite(writer, request, http.StatusOK, map[string]any{"node_id": "PR_42"}) {
		return
	}
}

func (s *Server) applyStateTransition(writer http.ResponseWriter, operation string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stateAttempts++
	if s.scenario.StateWriteDenied {
		writeError(writer, http.StatusForbidden, "fixture state transition denied")
		return false
	}
	if s.stateCurrent.Merged {
		writeError(writer, http.StatusConflict, "merged pull request is terminal")
		return false
	}
	desired := desiredLifecycleState(s.stateCurrent, operation)
	if operation == "retarget" && s.scenario.StateTargetMovesAfterWrite {
		desired.BaseSHA = forcedHeadSHA
	}
	desired.UpdatedAt = "2026-07-27T20:40:00Z"
	if s.scenario.StateVisibilityDelay > 0 {
		s.statePending = &desired
	} else {
		s.stateCurrent = desired
	}
	return true
}

func desiredLifecycleState(current lifecycleState, operation string) lifecycleState {
	switch operation {
	case "mark_ready":
		current.State = "open"
		current.Draft = false
	case "retarget":
		current.Base = "release"
		current.BaseSHA = releaseSHA
	case "close":
		current.State = "closed"
	case "reopen":
		current.State = "open"
	}
	return current
}

func (s *Server) finishStateWrite(writer http.ResponseWriter, request *http.Request, status int, response any) bool {
	if s.scenario.StateResponseDelay > 0 {
		select {
		case <-request.Context().Done():
			return true
		case <-time.After(s.scenario.StateResponseDelay):
		}
	}
	if s.scenario.StateLostResponse {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(`{"incomplete":`))
		return true
	}
	writeJSON(writer, status, response)
	return true
}

func (s *Server) listCommits(writer http.ResponseWriter, request *http.Request) {
	items := make([]map[string]any, 0, 120)
	for index := 0; index < 120; index++ {
		sha := fmt.Sprintf("%040x", index+1)
		items = append(items, map[string]any{
			"sha": sha, "html_url": "https://example.invalid/commit/" + sha,
			"author": user("contributor", 7),
			"commit": map[string]any{
				"message": fmt.Sprintf("Commit %d", index+1),
				"author":  map[string]any{"name": "Contributor", "date": "2026-07-27T20:00:00Z"},
			},
		})
	}
	start, end, hasNext, page, perPage := s.page(items, request.URL.Query())
	if hasNext {
		s.setNext(writer, request, page+1, perPage)
	}
	writeJSON(writer, http.StatusOK, items[start:end])
}

func (s *Server) listDiffFiles(writer http.ResponseWriter, request *http.Request) {
	count := 2
	patchSize := 64
	if s.scenario.OversizedDiff {
		count = 310
		patchSize = 8192
	}
	items := make([]map[string]any, 0, count)
	for index := 0; index < count; index++ {
		patch := "@@ -1 +1 @@\n-" + strings.Repeat("a", patchSize/2) + "\n+" + strings.Repeat("b", patchSize/2) + "\n"
		items = append(items, map[string]any{
			"filename": fmt.Sprintf("file-%03d.go", index),
			"status":   "modified", "additions": 1, "deletions": 1, "patch": patch,
		})
	}
	start, end, hasNext, page, perPage := s.page(items, request.URL.Query())
	if hasNext {
		s.setNext(writer, request, page+1, perPage)
	}
	writeJSON(writer, http.StatusOK, items[start:end])
}

func (s *Server) listChecks(writer http.ResponseWriter, request *http.Request) {
	if s.rateLimit(writer, "checks") || s.deny(writer, "checks") {
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"total_count": 2,
		"check_runs": []map[string]any{
			{"id": 1, "node_id": "CHECK_1", "name": "test", "status": "completed", "conclusion": "success", "html_url": "https://example.invalid/check/1"},
			{"id": 2, "node_id": "CHECK_2", "name": "lint", "status": "in_progress", "conclusion": "", "html_url": "https://example.invalid/check/2"},
		},
	})
}

func (s *Server) listReviews(writer http.ResponseWriter, request *http.Request) {
	if s.rateLimit(writer, "reviews") || s.deny(writer, "reviews") {
		return
	}
	if s.collaborationReadbackUnavailable(writer) {
		return
	}
	items := []map[string]any{
		{"id": 1, "node_id": "REVIEW_1", "user": user("reviewer", 8), "body": "approved", "state": "APPROVED", "commit_id": currentHeadSHA, "submitted_at": "2026-07-27T20:20:00Z"},
		{"id": 2, "node_id": "REVIEW_2", "user": user("reviewer", 8), "body": "stale", "state": "CHANGES_REQUESTED", "commit_id": baseSHA, "submitted_at": "2026-07-27T20:10:00Z"},
	}
	s.mu.Lock()
	if s.collaborationVisibleLocked() {
		items = append(items, s.collaborationReviews...)
	}
	s.mu.Unlock()
	start, end, hasNext, page, perPage := s.page(items, request.URL.Query())
	if hasNext {
		s.setNext(writer, request, page+1, perPage)
	}
	writeJSON(writer, http.StatusOK, items[start:end])
}

func (s *Server) listComments(writer http.ResponseWriter, request *http.Request) {
	if s.rateLimit(writer, "comments") || s.deny(writer, "comments") {
		return
	}
	if s.collaborationReadbackUnavailable(writer) {
		return
	}
	items := []map[string]any{
		{"id": 1, "node_id": "COMMENT_1", "user": user("commenter", 9), "body": "first", "html_url": "https://example.invalid/comment/1", "created_at": "2026-07-27T20:00:00Z", "updated_at": "2026-07-27T20:00:00Z"},
		{"id": 2, "node_id": "COMMENT_2", "user": user("commenter", 9), "body": "second", "html_url": "https://example.invalid/comment/2", "created_at": "2026-07-27T20:01:00Z", "updated_at": "2026-07-27T20:01:00Z"},
	}
	s.mu.Lock()
	if s.collaborationVisibleLocked() {
		items = append(items, s.collaborationComments...)
	}
	s.mu.Unlock()
	start, end, hasNext, page, perPage := s.page(items, request.URL.Query())
	if hasNext {
		s.setNext(writer, request, page+1, perPage)
	}
	writeJSON(writer, http.StatusOK, items[start:end])
}

func (s *Server) createComment(writer http.ResponseWriter, request *http.Request) {
	var payload struct {
		Body string `json:"body"`
	}
	if !s.decodeCollaborationWrite(writer, request, &payload) || payload.Body == "" {
		return
	}
	s.mu.Lock()
	s.collaborationAttempts++
	id := int64(100 + s.collaborationAttempts)
	actor := "hero-user"
	actorID := int64(99)
	if s.scenario.CollaborationCollisionOnly {
		actor, actorID = "other-reviewer", 98
	}
	comment := map[string]any{
		"id": id, "node_id": fmt.Sprintf("COMMENT_%d", id), "user": user(actor, actorID),
		"body": payload.Body, "html_url": fmt.Sprintf("https://example.invalid/comment/%d", id),
		"created_at": "2026-07-27T20:40:00Z", "updated_at": "2026-07-27T20:40:00Z",
	}
	s.collaborationComments = append(s.collaborationComments, comment)
	s.mu.Unlock()
	if s.finishCollaborationWrite(writer, request) {
		return
	}
	writeJSON(writer, http.StatusCreated, comment)
}

func (s *Server) createReview(writer http.ResponseWriter, request *http.Request) {
	var payload struct {
		Body     string `json:"body"`
		Event    string `json:"event"`
		CommitID string `json:"commit_id"`
	}
	if !s.decodeCollaborationWrite(writer, request, &payload) || payload.Body == "" || payload.CommitID == "" {
		return
	}
	state := "COMMENTED"
	switch payload.Event {
	case "APPROVE":
		state = "APPROVED"
	case "REQUEST_CHANGES":
		state = "CHANGES_REQUESTED"
	case "COMMENT":
	default:
		writeError(writer, http.StatusUnprocessableEntity, "invalid review event")
		return
	}
	if s.scenario.CollaborationForcedReviewState != "" {
		state = s.scenario.CollaborationForcedReviewState
	}
	s.mu.Lock()
	s.collaborationAttempts++
	id := int64(100 + s.collaborationAttempts)
	actor := "hero-user"
	actorID := int64(99)
	if s.scenario.CollaborationCollisionOnly {
		actor, actorID = "other-reviewer", 98
	}
	review := map[string]any{
		"id": id, "node_id": fmt.Sprintf("REVIEW_%d", id), "user": user(actor, actorID),
		"body": payload.Body, "state": state, "commit_id": payload.CommitID, "submitted_at": "2026-07-27T20:40:00Z",
	}
	s.collaborationReviews = append(s.collaborationReviews, review)
	s.mu.Unlock()
	if s.finishCollaborationWrite(writer, request) {
		return
	}
	writeJSON(writer, http.StatusOK, review)
}

func (s *Server) decodeCollaborationWrite(writer http.ResponseWriter, request *http.Request, target any) bool {
	if s.scenario.CollaborationWriteDenied {
		s.mu.Lock()
		s.collaborationAttempts++
		s.mu.Unlock()
		writeError(writer, http.StatusForbidden, "fixture collaboration denied")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(writer, http.StatusUnprocessableEntity, "invalid collaboration payload")
		return false
	}
	return true
}

func (s *Server) finishCollaborationWrite(writer http.ResponseWriter, request *http.Request) bool {
	if s.scenario.CollaborationResponseDelay > 0 {
		select {
		case <-request.Context().Done():
			return true
		case <-time.After(s.scenario.CollaborationResponseDelay):
		}
	}
	if s.scenario.CollaborationLostResponse {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"incomplete":`))
		return true
	}
	return false
}

func (s *Server) collaborationReadbackUnavailable(writer http.ResponseWriter) bool {
	s.mu.Lock()
	unavailable := s.scenario.CollaborationAmbiguousReadback && s.collaborationAttempts > 0
	s.mu.Unlock()
	if unavailable {
		writeError(writer, http.StatusServiceUnavailable, "fixture collaboration read-back unavailable")
		return true
	}
	return false
}

func (s *Server) collaborationVisibleLocked() bool {
	if s.collaborationAttempts == 0 {
		return true
	}
	s.collaborationReads++
	return s.collaborationReads > s.scenario.CollaborationVisibilityDelay
}

func (s *Server) handleGraphQL(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.rateLimit(writer, "graphql") || s.deny(writer, "graphql") {
		return
	}
	var envelope struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	if err := decoder.Decode(&envelope); err != nil || envelope.Query == "" {
		writeError(writer, http.StatusUnprocessableEntity, "invalid GraphQL request")
		return
	}
	if strings.Contains(envelope.Query, "markPullRequestReadyForReview") {
		if fmt.Sprint(envelope.Variables["pullRequestId"]) != "PR_42" {
			writeError(writer, http.StatusNotFound, "pull request not found")
			return
		}
		if !s.applyStateTransition(writer, "mark_ready") {
			return
		}
		setRateHeaders(writer, "graphql", 5000, 4998, 0)
		s.finishStateWrite(writer, request, http.StatusOK, map[string]any{
			"data": map[string]any{
				"markPullRequestReadyForReview": map[string]any{
					"pullRequest": map[string]any{"id": "PR_42"},
				},
			},
			"errors": []any{},
		})
		return
	}
	s.mu.Lock()
	index := s.graphQL
	s.graphQL++
	s.mu.Unlock()
	mergeability := "MERGEABLE"
	if len(s.scenario.MergeabilitySequence) > 0 {
		if index >= len(s.scenario.MergeabilitySequence) {
			index = len(s.scenario.MergeabilitySequence) - 1
		}
		mergeability = s.scenario.MergeabilitySequence[index]
	}
	viewerCanMerge := true
	pullRequest := map[string]any{
		"id": "PR_42", "headRefOid": currentHeadSHA, "baseRefOid": baseSHA,
		"isDraft": false, "mergeable": mergeability, "mergeStateStatus": "CLEAN",
		"reviewDecision": "APPROVED", "viewerCanMerge": viewerCanMerge,
		"mergeQueueEntry": nil,
		"baseRef": map[string]any{"branchProtectionRule": map[string]any{
			"requiresApprovingReviews": true, "requiredApprovingReviewCount": 1, "requiresStatusChecks": true,
		}},
		"commits": map[string]any{"nodes": []map[string]any{
			{"commit": map[string]any{"statusCheckRollup": map[string]any{"state": "SUCCESS"}}},
		}},
	}
	errorsValue := []map[string]any{}
	if s.scenario.GraphQLPartial {
		pullRequest["commits"] = map[string]any{"nodes": nil}
		errorsValue = append(errorsValue, map[string]any{
			"message": "fixture denied check data",
			"type":    "FORBIDDEN",
			"path":    []any{"repository", "pullRequest", "commits"},
		})
	}
	setRateHeaders(writer, "graphql", 5000, 4998, 0)
	writeJSON(writer, http.StatusOK, map[string]any{
		"data": map[string]any{
			"repository": map[string]any{"pullRequest": pullRequest},
			"rateLimit":  map[string]any{"limit": 5000, "remaining": 4998, "resetAt": "2026-07-27T21:00:00Z"},
		},
		"errors": errorsValue,
	})
}

func (s *Server) page(items []map[string]any, query url.Values) (start, end int, hasNext bool, page, perPage int) {
	perPage = 30
	if value, err := strconv.Atoi(query.Get("per_page")); err == nil && value > 0 {
		perPage = min(value, 100)
	}
	if s.scenario.ForcedPageSize > 0 {
		perPage = min(perPage, s.scenario.ForcedPageSize)
	}
	page = 1
	if value, err := strconv.Atoi(query.Get("page")); err == nil && value > 0 {
		page = value
	}
	start = min((page-1)*perPage, len(items))
	end = min(start+perPage, len(items))
	return start, end, end < len(items), page, perPage
}

func (s *Server) setNext(writer http.ResponseWriter, request *http.Request, page, perPage int) {
	if s.scenario.CrossOriginNext != "" {
		writer.Header().Set("Link", "<"+s.scenario.CrossOriginNext+">; rel=\"next\"")
		return
	}
	query := request.URL.Query()
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	next := scheme + "://" + request.Host + request.URL.Path + "?" + query.Encode()
	writer.Header().Set("Link", "<"+next+">; rel=\"next\"")
}

func (s *Server) deny(writer http.ResponseWriter, section string) bool {
	status := s.scenario.DeniedSections[section]
	if status == 0 {
		return false
	}
	writeError(writer, status, "fixture section denied")
	return true
}

func (s *Server) rateLimit(writer http.ResponseWriter, section string) bool {
	retryAfter := s.scenario.RateLimitedSections[section]
	if retryAfter == 0 {
		return false
	}
	setRateHeaders(writer, section, 5000, 0, retryAfter)
	writeError(writer, http.StatusTooManyRequests, "fixture rate limited")
	return true
}

func repositoryValue(owner, repository string, id int64) map[string]any {
	return map[string]any{
		"id": id, "node_id": fmt.Sprintf("R_%d", id), "name": repository,
		"full_name": owner + "/" + repository, "owner": user(owner, id+100),
	}
}

func user(login string, id int64) map[string]any {
	return map[string]any{"id": id, "node_id": fmt.Sprintf("U_%d", id), "login": login, "name": strings.ToUpper(login[:1]) + login[1:]}
}

func setRateHeaders(writer http.ResponseWriter, resource string, limit, remaining, retryAfter int) {
	writer.Header().Set("X-RateLimit-Resource", resource)
	writer.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	writer.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	writer.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Date(2026, 7, 27, 21, 0, 0, 0, time.UTC).Unix(), 10))
	if retryAfter > 0 {
		writer.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	}
}

func splitPath(value string) []string {
	parts := []string{}
	for _, part := range strings.Split(value, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"message": message})
}
