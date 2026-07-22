// Package trackerevidence defines the stable JSON contract shared by Hero's
// in-process, CLI, MCP, and Hero Code tracker-evidence consumers.
package trackerevidence

import (
	"embed"
)

// Version identifies the first tracker-evidence contract. Consumers must use
// this value, rather than inferring compatibility from individual fields.
const Version = "tracker-evidence/v1"

// State is the outcome of an explicit tracker-evidence load.
type State string

const (
	StateFetched     State = "fetched"
	StateRefreshed   State = "refreshed"
	StateCurrent     State = "current"
	StateUnsupported State = "unsupported"
	StateUnavailable State = "unavailable"
)

// ErrorCode is a stable machine-readable failure classification. Consumers
// must branch on Code and treat Message as display-only text.
type ErrorCode string

const (
	ErrorSpecNotFound        ErrorCode = "spec_not_found"
	ErrorTrackerUnlinked     ErrorCode = "tracker_unlinked"
	ErrorAmbiguousConnection ErrorCode = "ambiguous_connection"
	ErrorUnsupportedProvider ErrorCode = "unsupported_provider"
	ErrorProviderUnavailable ErrorCode = "provider_unavailable"
	ErrorInvalidManifest     ErrorCode = "invalid_manifest"
	ErrorPayloadMissing      ErrorCode = "payload_missing"
	ErrorPayloadCorrupt      ErrorCode = "payload_corrupt"
	ErrorCancelled           ErrorCode = "cancelled"
	ErrorWriteFailed         ErrorCode = "write_failed"
)

// Error is the safe error shape returned to consumers. Message must not
// contain credentials, raw evidence, provider URLs, identities, or filenames.
type Error struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Retryable bool      `json:"retryable"`
}

// Request identifies one explicit, foreground evidence load. A nil
// IncludeAttachments means true; a non-nil false value disables attachment
// downloads without changing evidence metadata collection.
type Request struct {
	SpecSlug           string `json:"spec_slug"`
	ConnectionID       string `json:"connection_id,omitempty"`
	IncludeAttachments *bool  `json:"include_attachments,omitempty"`
	ForceRefresh       bool   `json:"force_refresh,omitempty"`
}

// AttachmentsEnabled applies the v1 default for IncludeAttachments.
func (r Request) AttachmentsEnabled() bool {
	return r.IncludeAttachments == nil || *r.IncludeAttachments
}

// Status is the bounded result returned by every shared evidence-loading
// surface. It intentionally contains no evidence body, tracker URL, title,
// description, comments, changelog, identities, or attachment filenames.
type Status struct {
	Version          string `json:"version"`
	Status           State  `json:"status"`
	Provider         string `json:"provider,omitempty"`
	ConnectionID     string `json:"connection_id,omitempty"`
	SpecSlug         string `json:"spec_slug,omitempty"`
	IssueID          string `json:"issue_id,omitempty"`
	TrackerUpdatedAt string `json:"tracker_updated_at,omitempty"`
	ContentSHA256    string `json:"content_sha256,omitempty"`
	ManifestPath     string `json:"manifest_path,omitempty"`
	EvidencePath     string `json:"evidence_path,omitempty"`
	AttachmentCount  int    `json:"attachment_count"`
	OmissionCount    int    `json:"omission_count"`
	CacheHit         bool   `json:"cache_hit"`
	Error            *Error `json:"error,omitempty"`
}

// Manifest is the complete allowlist for the adjacent tracked provenance
// file. Full tracker evidence and connection identity belong only in the
// ignored private snapshot, never in this shape.
type Manifest struct {
	Version          string `json:"version"`
	Provider         string `json:"provider"`
	IssueID          string `json:"issue_id"`
	TrackerUpdatedAt string `json:"tracker_updated_at"`
	ContentSHA256    string `json:"content_sha256"`
	EvidencePath     string `json:"evidence_path"`
	AttachmentCount  int    `json:"attachment_count"`
	OmissionCount    int    `json:"omission_count"`
	RetrievedAt      string `json:"retrieved_at"`
}

//go:embed testdata/v1/*.json
var fixtures embed.FS

// ConsumerFixture returns an isolated copy of the canonical v1 Hero Code
// fixture bundle.
func ConsumerFixture() ([]byte, error) {
	b, err := fixtures.ReadFile("testdata/v1/consumer-fixtures.json")
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), b...), nil
}

// ManifestFixture returns an isolated copy of the canonical safe v1 manifest
// fixture.
func ManifestFixture() ([]byte, error) {
	b, err := fixtures.ReadFile("testdata/v1/current-manifest.json")
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), b...), nil
}
