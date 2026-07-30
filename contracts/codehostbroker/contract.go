// Package codehostbroker defines the stable provider-neutral JSON contract
// shared by Hero's in-process code-host broker, CLI, MCP, and Swift consumers.
package codehostbroker

import (
	"embed"
	"encoding/json"
	"strings"
)

const Version = "code-host-broker/v1"

type Operation string

const (
	OperationCapabilities          Operation = "capabilities"
	OperationGetAuthenticatedActor Operation = "get_authenticated_actor"
	OperationListPullRequests      Operation = "list_pull_requests"
	OperationSearchPullRequests    Operation = "search_pull_requests"
	OperationGetPullRequest        Operation = "get_pull_request"
	OperationGetCommits            Operation = "get_commits"
	OperationGetDiff               Operation = "get_diff"
	OperationGetChecks             Operation = "get_checks"
	OperationGetReviews            Operation = "get_reviews"
	OperationGetComments           Operation = "get_comments"
	OperationGetMergeReadiness     Operation = "get_merge_readiness"
	OperationCreatePullRequest     Operation = "create_pull_request"
	OperationComment               Operation = "comment"
	OperationSubmitReview          Operation = "submit_review"
	OperationApprove               Operation = "approve"
	OperationRequestChanges        Operation = "request_changes"
	OperationMarkReady             Operation = "mark_ready"
	OperationRetarget              Operation = "retarget"
	OperationClose                 Operation = "close"
	OperationReopen                Operation = "reopen"
	OperationMerge                 Operation = "merge"
)

type Effect string

const (
	EffectRead          Effect = "read"
	EffectExternalWrite Effect = "external_write"
	EffectCommitment    Effect = "commitment"
)

type Consent string

const (
	ConsentNone               Consent = "none"
	ConsentExplicitUser       Consent = "explicit_user"
	ConsentExplicitAcceptance Consent = "explicit_acceptance"
)

type RetryGuidance string

const (
	RetryNone             RetryGuidance = "none"
	RetrySameKey          RetryGuidance = "same_key"
	RetryRefreshThenRetry RetryGuidance = "refresh_then_retry"
	RetryAfter            RetryGuidance = "retry_after"
	RetryReconcile        RetryGuidance = "reconcile"
)

type Availability string

const (
	AvailabilityAvailable   Availability = "available"
	AvailabilityPartial     Availability = "partial"
	AvailabilityUnavailable Availability = "unavailable"
	AvailabilityUnknown     Availability = "unknown"
)

type Completeness string

const (
	CompletenessComplete    Completeness = "complete"
	CompletenessPartial     Completeness = "partial"
	CompletenessTruncated   Completeness = "truncated"
	CompletenessUnavailable Completeness = "unavailable"
)

type Freshness string

const (
	FreshnessCurrent     Freshness = "current"
	FreshnessStale       Freshness = "stale"
	FreshnessUnknown     Freshness = "unknown"
	FreshnessUnavailable Freshness = "unavailable"
)

type ReconciliationStatus string

const (
	ReconciliationApplied             ReconciliationStatus = "applied"
	ReconciliationReplayed            ReconciliationStatus = "replayed"
	ReconciliationReconciledApplied   ReconciliationStatus = "reconciled_applied"
	ReconciliationExternallyCompleted ReconciliationStatus = "externally_completed"
	ReconciliationNotApplied          ReconciliationStatus = "not_applied"
	ReconciliationInProgress          ReconciliationStatus = "in_progress"
	ReconciliationAmbiguous           ReconciliationStatus = "ambiguous"
)

type RepositoryIdentity struct {
	Host       string `json:"host"`
	ProviderID string `json:"provider_id,omitempty"`
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	FullName   string `json:"full_name"`
}

type RefIdentity struct {
	Repository RepositoryIdentity `json:"repository"`
	Name       string             `json:"name"`
	SHA        string             `json:"sha"`
}

type PullRequestIdentity struct {
	ConnectionID string             `json:"connection_id"`
	Repository   RepositoryIdentity `json:"repository"`
	ProviderID   string             `json:"provider_id"`
	Number       int64              `json:"number"`
}

type Actor struct {
	ProviderID string `json:"provider_id,omitempty"`
	Login      string `json:"login"`
	Display    string `json:"display,omitempty"`
}

type PullRequest struct {
	Identity  PullRequestIdentity `json:"identity"`
	Title     string              `json:"title"`
	Body      string              `json:"body,omitempty"`
	URL       string              `json:"url"`
	State     string              `json:"state"`
	Draft     bool                `json:"draft"`
	Author    Actor               `json:"author"`
	Base      RefIdentity         `json:"base"`
	Head      RefIdentity         `json:"head"`
	CreatedAt string              `json:"created_at,omitempty"`
	UpdatedAt string              `json:"updated_at,omitempty"`
	MergedAt  string              `json:"merged_at,omitempty"`
}

type Commit struct {
	SHA        string `json:"sha"`
	Message    string `json:"message"`
	Author     Actor  `json:"author"`
	AuthoredAt string `json:"authored_at,omitempty"`
	URL        string `json:"url,omitempty"`
}

type DiffHunk struct {
	Header string `json:"header"`
	Patch  string `json:"patch"`
}

type DiffFile struct {
	Path      string     `json:"path"`
	Status    string     `json:"status"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	Hunks     []DiffHunk `json:"hunks"`
	Truncated bool       `json:"truncated"`
}

type Check struct {
	ProviderID   string       `json:"provider_id,omitempty"`
	Name         string       `json:"name"`
	Status       string       `json:"status"`
	Conclusion   string       `json:"conclusion,omitempty"`
	URL          string       `json:"url,omitempty"`
	Availability Availability `json:"availability"`
}

type Review struct {
	ProviderID  string `json:"provider_id"`
	Author      Actor  `json:"author"`
	State       string `json:"state"`
	Body        string `json:"body,omitempty"`
	HeadSHA     string `json:"head_sha"`
	SubmittedAt string `json:"submitted_at,omitempty"`
}

type Comment struct {
	ProviderID string `json:"provider_id"`
	Author     Actor  `json:"author"`
	Body       string `json:"body"`
	URL        string `json:"url,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

type MergeReadiness struct {
	State            string       `json:"state"`
	Checks           Availability `json:"checks"`
	Reviews          Availability `json:"reviews"`
	BranchProtection Availability `json:"branch_protection"`
	Permissions      Availability `json:"permissions"`
	Mergeability     Availability `json:"mergeability"`
	Queue            Availability `json:"queue"`
	Reasons          []string     `json:"reasons"`
}

type RateLimit struct {
	Resource   string `json:"resource,omitempty"`
	Limit      int64  `json:"limit,omitempty"`
	Remaining  int64  `json:"remaining,omitempty"`
	ResetAt    string `json:"reset_at,omitempty"`
	RetryAfter int64  `json:"retry_after_seconds,omitempty"`
	ObservedAt string `json:"observed_at"`
}

type Page struct {
	Limit      int    `json:"limit"`
	Count      int    `json:"count"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type PartialFailure struct {
	Section string `json:"section"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Receipt struct {
	ProviderReceiptID string `json:"provider_receipt_id,omitempty"`
	OperationID       string `json:"operation_id"`
	TargetRevision    string `json:"target_revision,omitempty"`
}

type Reconciliation struct {
	Status ReconciliationStatus `json:"status"`
	Key    string               `json:"key"`
}

type ContractError struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Field   string        `json:"field,omitempty"`
	Retry   RetryGuidance `json:"retry"`
	RetryAt string        `json:"retry_at,omitempty"`
}

func (e *ContractError) Error() string { return e.Message }

type Bounds struct {
	RepositoryScopes int `json:"repository_scopes"`
	PageSize         int `json:"page_size"`
	Items            int `json:"items"`
	TextBytes        int `json:"text_bytes"`
	BodyBytes        int `json:"body_bytes"`
	DiffBytes        int `json:"diff_bytes"`
	DiffFiles        int `json:"diff_files"`
	DiffHunks        int `json:"diff_hunks"`
	PartialFailures  int `json:"partial_failures"`
	ErrorDetailBytes int `json:"error_detail_bytes"`
	DurationMS       int `json:"duration_ms"`
	Redirects        int `json:"redirects"`
	JournalEntries   int `json:"journal_entries"`
	IdempotencyBytes int `json:"idempotency_bytes"`
}

type OperationPolicy struct {
	Operation                Operation `json:"operation"`
	Effect                   Effect    `json:"effect"`
	Consent                  Consent   `json:"consent"`
	RequiresUniqueTarget     bool      `json:"requires_unique_target"`
	RequiresIdempotency      bool      `json:"requires_idempotency"`
	RequiresFreshObservation bool      `json:"requires_fresh_observation"`
	RequiresReconciliation   bool      `json:"requires_reconciliation"`
	ReplaySafe               bool      `json:"replay_safe"`
	Bounds                   Bounds    `json:"bounds"`
}

type Capability struct {
	Policy    OperationPolicy  `json:"policy"`
	Available bool             `json:"available"`
	Reason    string           `json:"reason,omitempty"`
	Merge     *MergeCapability `json:"merge,omitempty"`
}

type MergeCapability struct {
	Methods        []string `json:"methods"`
	QueueSupported bool     `json:"queue_supported"`
	QueueRequired  bool     `json:"queue_required"`
	Revision       string   `json:"revision"`
}

type CapabilitiesResult struct {
	Capabilities []Capability `json:"capabilities"`
}

type AuthenticatedActorResult struct {
	Actor Actor `json:"actor"`
}

type PullRequestsResult struct {
	PullRequests []PullRequest `json:"pull_requests"`
}

type CommitsResult struct {
	Commits []Commit `json:"commits"`
}

type DiffResult struct {
	Files []DiffFile `json:"files"`
}

type ChecksResult struct {
	Checks []Check `json:"checks"`
}

type ReviewsResult struct {
	Reviews []Review `json:"reviews"`
}

type CommentsResult struct {
	Comments []Comment `json:"comments"`
}

type MutationResult struct {
	PullRequest           PullRequest          `json:"pull_request"`
	Outcome               string               `json:"outcome"`
	Actor                 *Actor               `json:"actor,omitempty"`
	InvalidatedOperations []Operation          `json:"invalidated_operations,omitempty"`
	Merge                 *MergeMutationResult `json:"merge,omitempty"`
}

type MergeMutationResult struct {
	State         string `json:"state"`
	MergeCommitID string `json:"merge_commit_id,omitempty"`
	QueueID       string `json:"queue_id,omitempty"`
}

type Response struct {
	Version             string             `json:"version"`
	Operation           Operation          `json:"operation"`
	Provider            string             `json:"provider"`
	ConnectionID        string             `json:"connection_id"`
	Repository          RepositoryIdentity `json:"repository"`
	Policy              OperationPolicy    `json:"policy"`
	CapabilityRevision  string             `json:"capability_revision"`
	ObservationRevision string             `json:"observation_revision"`
	ObservedAt          string             `json:"observed_at"`
	Freshness           Freshness          `json:"freshness"`
	RateLimit           RateLimit          `json:"rate_limit"`
	Bounds              Bounds             `json:"bounds"`
	Completeness        Completeness       `json:"completeness"`
	Page                *Page              `json:"page,omitempty"`
	PartialFailures     []PartialFailure   `json:"partial_failures"`
	Result              json.RawMessage    `json:"result"`
	Receipt             *Receipt           `json:"receipt,omitempty"`
	Reconciliation      *Reconciliation    `json:"reconciliation,omitempty"`
	Truncated           bool               `json:"truncated"`
	DurationMS          int64              `json:"duration_ms"`
	Redirects           int                `json:"redirects"`
	JournalEntries      int                `json:"journal_entries"`
	Error               *ContractError     `json:"error"`
}

// PreparationResponse is the bounded, non-mutating preflight result used by
// process and MCP transports before a separately authorized mutation call.
// The caller applies the returned revisions to its original typed request; the
// payload is intentionally not echoed into transport output.
type PreparationResponse struct {
	Version             string         `json:"version"`
	Operation           Operation      `json:"operation"`
	CapabilityRevision  string         `json:"capability_revision,omitempty"`
	ObservationRevision string         `json:"observation_revision,omitempty"`
	Error               *ContractError `json:"error"`
}

type Request struct {
	Version             string               `json:"version"`
	Operation           Operation            `json:"operation"`
	Provider            string               `json:"provider"`
	ConnectionID        string               `json:"connection_id"`
	Repository          RepositoryIdentity   `json:"repository"`
	Repositories        []RepositoryIdentity `json:"repositories,omitempty"`
	PullRequest         *PullRequestIdentity `json:"pull_request,omitempty"`
	IntentSource        string               `json:"intent_source,omitempty"`
	Consent             Consent              `json:"consent,omitempty"`
	IdempotencyKey      string               `json:"idempotency_key,omitempty"`
	CapabilityRevision  string               `json:"capability_revision,omitempty"`
	ObservationRevision string               `json:"observation_revision,omitempty"`
	ReconciliationKey   string               `json:"reconciliation_key,omitempty"`
	Query               string               `json:"query,omitempty"`
	Order               string               `json:"order,omitempty"`
	Limit               int                  `json:"limit,omitempty"`
	Cursor              string               `json:"cursor,omitempty"`
	Payload             json.RawMessage      `json:"payload,omitempty"`
}

type CreatePullRequestPayload struct {
	Base  RefIdentity `json:"base"`
	Head  RefIdentity `json:"head"`
	Title string      `json:"title"`
	Body  string      `json:"body,omitempty"`
	Draft bool        `json:"draft"`
}

type CommentPayload struct {
	ExpectedHeadSHA string `json:"expected_head_sha"`
	Body            string `json:"body"`
}

type ReviewPayload struct {
	ExpectedHeadSHA string `json:"expected_head_sha"`
	Body            string `json:"body,omitempty"`
}

type RetargetPayload struct {
	ExpectedHeadSHA string      `json:"expected_head_sha"`
	CurrentBase     RefIdentity `json:"current_base"`
	NewBase         RefIdentity `json:"new_base"`
}

type LifecyclePayload struct {
	ExpectedHeadSHA string `json:"expected_head_sha"`
}

type MergePayload struct {
	ExpectedHeadSHA string      `json:"expected_head_sha"`
	ObservedBase    RefIdentity `json:"observed_base"`
	Method          string      `json:"method"`
	CommitTitle     string      `json:"commit_title,omitempty"`
	CommitMessage   string      `json:"commit_message,omitempty"`
}

type CursorMaterial struct {
	Version      string               `json:"version"`
	Provider     string               `json:"provider"`
	ConnectionID string               `json:"connection_id"`
	Repositories []RepositoryIdentity `json:"repositories"`
	Operation    Operation            `json:"operation"`
	Query        string               `json:"query"`
	Order        string               `json:"order"`
	Position     string               `json:"position"`
}

type CursorEnvelope struct {
	Material    CursorMaterial `json:"material"`
	Fingerprint string         `json:"fingerprint"`
}

type RevisionMaterial struct {
	ConnectionID string               `json:"connection_id"`
	Repository   RepositoryIdentity   `json:"repository"`
	PullRequest  *PullRequestIdentity `json:"pull_request,omitempty"`
	Base         *RefIdentity         `json:"base,omitempty"`
	Head         *RefIdentity         `json:"head,omitempty"`
	State        string               `json:"state,omitempty"`
	UpdatedAt    string               `json:"updated_at,omitempty"`
	Permissions  []string             `json:"permissions,omitempty"`
}

type FixtureCase struct {
	Name     string   `json:"name"`
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

type ConsumerFixtureBundle struct {
	Version       string                     `json:"version"`
	Operations    []Capability               `json:"operations"`
	Cases         []FixtureCase              `json:"cases"`
	Preparations  []PreparationResponse      `json:"preparations"`
	Errors        []ContractError            `json:"errors"`
	UnknownFields map[string]json.RawMessage `json:"future_additive"`
}

//go:embed testdata/v1/consumer-fixture.json testdata/v1/consumer-fixture.sha256
var fixtures embed.FS

func ConsumerFixture() ([]byte, error) {
	b, err := fixtures.ReadFile("testdata/v1/consumer-fixture.json")
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), b...), nil
}

func ConsumerFixtureDigest() (string, error) {
	b, err := fixtures.ReadFile("testdata/v1/consumer-fixture.sha256")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
