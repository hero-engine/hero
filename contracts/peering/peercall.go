package peering

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ApproxInt is a non-negative integer field that tolerates a few
// approximation forms historical peers and external responders may emit
// when reporting estimates:
//
//   - plain int:        22000        -> 22000
//   - tilde-prefix:     ~22000       -> 22000   (the "~" is dropped)
//   - float form:       22000.0      -> 22000   (truncated, not rounded)
//   - string forms of any of the above
//   - empty / missing -> 0
//
// Round-trips back to a plain integer; the tilde is an input
// tolerance, not an output format. Wire shape is `int` for both YAML
// and JSON. Negative values are rejected at unmarshal time to keep
// "budget consumed" semantics honest.
type ApproxInt int

// Int returns the underlying integer value.
func (a ApproxInt) Int() int { return int(a) }

func parseApproxIntString(raw string) (ApproxInt, error) {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, `"'`)
	s = strings.TrimPrefix(s, "~")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n < 0 {
			return 0, fmt.Errorf("approxint: negative value %d", n)
		}
		return ApproxInt(n), nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if f < 0 {
			return 0, fmt.Errorf("approxint: negative value %g", f)
		}
		return ApproxInt(int(f)), nil
	}
	return 0, fmt.Errorf("approxint: cannot parse %q as integer", raw)
}

// UnmarshalYAML accepts !!int, !!float, and !!str scalars whose
// string form parses as a (possibly tilde-prefixed) non-negative
// integer.
func (a *ApproxInt) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		*a = 0
		return nil
	}
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("approxint: expected scalar, got kind=%d", node.Kind)
	}
	v, err := parseApproxIntString(node.Value)
	if err != nil {
		return err
	}
	*a = v
	return nil
}

// MarshalYAML emits a plain integer.
func (a ApproxInt) MarshalYAML() (any, error) { return int(a), nil }

// UnmarshalJSON applies the same tolerance as YAML so symmetric
// transports stay honest. Accepts JSON numbers, JSON strings of the
// same shape, and null -> 0.
func (a *ApproxInt) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "" || s == "null" {
		*a = 0
		return nil
	}
	v, err := parseApproxIntString(s)
	if err != nil {
		return err
	}
	*a = v
	return nil
}

// MarshalJSON emits a canonical integer.
func (a ApproxInt) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Itoa(int(a))), nil
}

// PeerCallMode classifies an asynchronous Project Mail request.
type PeerCallMode string

const (
	// PeerCallAdvisory — investigate and return findings. No writes
	// on the peer side beyond the call record. v1.
	PeerCallAdvisory PeerCallMode = "advisory"

	// PeerCallSpecOut asks the receiver to design work. The request itself
	// creates no work; receiver promotion is explicit.
	PeerCallSpecOut PeerCallMode = "spec-out"

	// PeerCallFull — design AND deliver on the peer side. Gated by
	// approval + budget. v2.
	PeerCallFull PeerCallMode = "full"
)

// BudgetSpec is non-enforced advisory metadata for an external responder.
type BudgetSpec struct {
	Turns  ApproxInt `yaml:"turns,omitempty" json:"turns,omitempty"`
	Tokens ApproxInt `yaml:"tokens,omitempty" json:"tokens,omitempty"`
}

// BudgetConsumed records actual consumption after the call returns.
type BudgetConsumed struct {
	Turns  ApproxInt `yaml:"turns" json:"turns"`
	Tokens ApproxInt `yaml:"tokens" json:"tokens"`
}

// PeerCallRequest is the legacy-compatible peering request shape. New transport
// uses contracts/attention MailEnvelope and retains this shape for history.
type PeerCallRequest struct {
	// ContractsVersion is the PeeringContractsVersion at call time.
	ContractsVersion int `json:"contracts_version" yaml:"contracts_version"`

	// CallID is a unique ULID for this call. Recorded on both sides.
	CallID string `json:"call_id" yaml:"call_id"`

	// OriginPeerID is the caller workspace's UUID.
	OriginPeerID string `json:"origin_peer_id" yaml:"origin_peer_id"`

	// TargetPeerID is the target workspace's UUID.
	TargetPeerID string `json:"target_peer_id" yaml:"target_peer_id"`

	// Mode is the interaction mode.
	Mode PeerCallMode `json:"mode" yaml:"mode"`

	// Prompt is the user-supplied prompt for the receiving project.
	Prompt string `json:"prompt" yaml:"prompt"`

	// Budget caps consumption. Zero values mean "use default".
	Budget BudgetSpec `json:"budget,omitempty" yaml:"budget,omitempty"`

	// RelatedSpec is the originator-side spec slug, if any — used
	// to anchor trail entries.
	RelatedSpec string `json:"related_spec,omitempty" yaml:"related_spec,omitempty"`

	// Reason is a free-form rationale captured at call time.
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`

	// At is the wall-clock time of the call.
	At time.Time `json:"at" yaml:"at"`

	// AtCommit is the originator's git commit SHA at call time.
	AtCommit string `json:"at_commit,omitempty" yaml:"at_commit,omitempty"`

	Transport      string `json:"transport,omitempty" yaml:"transport,omitempty"`
	MessageID      string `json:"message_id,omitempty" yaml:"message_id,omitempty"`
	ThreadID       string `json:"thread_id,omitempty" yaml:"thread_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty" yaml:"idempotency_key,omitempty"`
}

// PeerCallResultKind enumerates the result shapes a peer call returns.
type PeerCallResultKind string

const (
	// ResultFindings — advisory mode result: free-form findings text.
	ResultFindings PeerCallResultKind = "findings"
	// ResultSpecRef — spec-out mode result: a peer-side spec slug.
	ResultSpecRef PeerCallResultKind = "spec-ref"
	// ResultCommitRef — full-delivery mode result: a commit/PR ref.
	ResultCommitRef PeerCallResultKind = "commit-ref"
)

// PeerCallResult is retained for historical artifacts and external responses.
type PeerCallResult struct {
	// ContractsVersion is the PeeringContractsVersion at return time.
	ContractsVersion int `json:"contracts_version" yaml:"contracts_version,omitempty"`

	// CallID echoes the request's CallID.
	CallID string `json:"call_id" yaml:"call_id,omitempty"`

	// Mode echoes the request's mode.
	Mode PeerCallMode `json:"mode" yaml:"mode,omitempty"`

	// Kind describes which result fields are populated.
	Kind PeerCallResultKind `json:"kind" yaml:"kind"`

	// Findings is set when Kind == ResultFindings (advisory).
	Findings string `json:"findings,omitempty" yaml:"findings,omitempty"`

	// SpecSlug is set when Kind == ResultSpecRef (spec-out).
	SpecSlug string `json:"spec_slug,omitempty" yaml:"spec_slug,omitempty"`

	// PeerStatus snapshots the produced spec's status.
	PeerStatus string `json:"peer_status,omitempty" yaml:"peer_status,omitempty"`

	// CommitRef is set when Kind == ResultCommitRef (full delivery).
	CommitRef string `json:"commit_ref,omitempty" yaml:"commit_ref,omitempty"`

	// PRURL is set when Kind == ResultCommitRef and a PR was opened.
	PRURL string `json:"pr_url,omitempty" yaml:"pr_url,omitempty"`

	// BudgetConsumed records what the peer actually used.
	BudgetConsumed BudgetConsumed `json:"budget_consumed" yaml:"budget_consumed,omitempty"`

	// At is the wall-clock time of the result.
	At time.Time `json:"at" yaml:"at,omitempty"`

	// Error is non-empty when the call failed to complete.
	Error string `json:"error,omitempty" yaml:"error,omitempty"`

	Transport string `json:"transport,omitempty" yaml:"transport,omitempty"`
	MessageID string `json:"message_id,omitempty" yaml:"message_id,omitempty"`
	ThreadID  string `json:"thread_id,omitempty" yaml:"thread_id,omitempty"`
	Status    string `json:"status,omitempty" yaml:"status,omitempty"`
	ResultRef string `json:"result_ref,omitempty" yaml:"result_ref,omitempty"`
}
