// Package environment provides read-only visibility into CI, deployment,
// and runtime environments surrounding the code.
package environment

import (
	"fmt"
	"time"
)

// PipelineStatus represents the current state of a CI pipeline.
type PipelineStatus struct {
	Provider    string    `json:"provider"`
	Branch      string    `json:"branch"`
	Status      string    `json:"status"` // "passing", "failing", "running", "unknown"
	Conclusion  string    `json:"conclusion,omitempty"` // "success", "failure", "cancelled"
	RunURL      string    `json:"run_url,omitempty"`
	CommitSHA   string    `json:"commit_sha,omitempty"`
	FailedStep  string    `json:"failed_step,omitempty"`
	FailedLog   string    `json:"failed_log,omitempty"` // truncated failure output
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	Duration    string    `json:"duration,omitempty"`
}

// CIProvider queries CI pipeline status.
type CIProvider interface {
	// PipelineStatus returns the status of the most recent pipeline run for the given branch.
	PipelineStatus(branch string) (*PipelineStatus, error)
	// Name returns the provider name.
	Name() string
}

// NewCIProvider creates a CI provider from configuration.
func NewCIProvider(providerType, project, token string) (CIProvider, error) {
	switch providerType {
	case "github-actions", "github":
		return newGitHubActions(project, token)
	default:
		return nil, fmt.Errorf("unknown CI provider %q — supported: github-actions", providerType)
	}
}
