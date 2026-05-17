// Package propose implements the inline-propose contract — the
// envelope schema, in-memory proposal store, and lifecycle bookkeeping
// the daemon needs to mediate between agents and the dashboard.
//
// The full wire contract is at docs/contracts/inline-propose-v1.md.
package propose

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SchemaVersion is the current envelope contract version.
const SchemaVersion = "1.0"

// HeroProposalPrefix is the stdout line prefix the shim looks for.
const HeroProposalPrefix = "HERO-PROPOSAL: "

// AnchorKind identifies how the anchor field points into the artifact.
type AnchorKind string

const (
	AnchorFrontmatter AnchorKind = "frontmatter"
	AnchorSection     AnchorKind = "section"
	AnchorHeading     AnchorKind = "heading"
	AnchorListItem    AnchorKind = "list_item"
	AnchorFree        AnchorKind = "free"
)

// AnchorPosition is how proposed content relates to the existing anchor.
type AnchorPosition string

const (
	PositionReplace AnchorPosition = "replace"
	PositionAppend  AnchorPosition = "append"
	PositionPrepend AnchorPosition = "prepend"
	PositionBefore  AnchorPosition = "before"
	PositionAfter   AnchorPosition = "after"
)

// ContentFormat is the format of the proposed content body.
type ContentFormat string

const (
	FormatMarkdown ContentFormat = "markdown"
	FormatText     ContentFormat = "text"
	FormatYAML     ContentFormat = "yaml"
)

// Anchor identifies where in the target spec the proposal applies.
type Anchor struct {
	Kind     AnchorKind     `json:"kind"`
	Value    string         `json:"value"`
	Position AnchorPosition `json:"position,omitempty"`
}

// Target identifies which spec the proposal applies to.
type Target struct {
	SpecSlug string `json:"spec_slug"`
	SpecPath string `json:"spec_path,omitempty"`
	Anchor   Anchor `json:"anchor"`
}

// Content carries the proposed body and its format.
type Content struct {
	Format ContentFormat `json:"format"`
	Body   string        `json:"body"`
}

// Envelope is the wire shape an agent emits on stdout under the
// HERO-PROPOSAL: prefix and which the daemon ingests verbatim.
type Envelope struct {
	SchemaVersion string    `json:"schema_version"`
	ProposalID    string    `json:"proposal_id"`
	BatchID       string    `json:"batch_id"`
	SessionID     string    `json:"session_id"`
	Agent         string    `json:"agent"`
	SkillChain    []string  `json:"skill_chain,omitempty"`
	Target        Target    `json:"target"`
	Content       Content   `json:"content"`
	Rationale     string    `json:"rationale,omitempty"`
	EmittedAt     time.Time `json:"emitted_at,omitempty"`
}

// Validate returns an error if the envelope is missing required
// fields or carries values outside the documented enums.
func (e *Envelope) Validate() error {
	if e == nil {
		return errors.New("nil envelope")
	}
	if e.SchemaVersion == "" {
		return errors.New("schema_version is required")
	}
	if e.ProposalID == "" {
		return errors.New("proposal_id is required")
	}
	if e.BatchID == "" {
		return errors.New("batch_id is required")
	}
	if e.SessionID == "" {
		return errors.New("session_id is required")
	}
	if e.Agent == "" {
		return errors.New("agent is required")
	}
	if e.Target.SpecSlug == "" {
		return errors.New("target.spec_slug is required")
	}
	switch e.Target.Anchor.Kind {
	case AnchorFrontmatter, AnchorSection, AnchorHeading, AnchorListItem, AnchorFree:
	case "":
		return errors.New("target.anchor.kind is required")
	default:
		return fmt.Errorf("target.anchor.kind %q is not a known kind", e.Target.Anchor.Kind)
	}
	if e.Target.Anchor.Value == "" {
		return errors.New("target.anchor.value is required")
	}
	switch e.Target.Anchor.Position {
	case "", PositionReplace, PositionAppend, PositionPrepend, PositionBefore, PositionAfter:
	default:
		return fmt.Errorf("target.anchor.position %q is not a known position", e.Target.Anchor.Position)
	}
	switch e.Content.Format {
	case FormatMarkdown, FormatText, FormatYAML:
	case "":
		return errors.New("content.format is required")
	default:
		return fmt.Errorf("content.format %q is not a known format", e.Content.Format)
	}
	return nil
}

// ParseEnvelope decodes a JSON envelope and validates it. The bytes
// must already have the HERO-PROPOSAL: prefix stripped.
func ParseEnvelope(data []byte) (*Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return &e, nil
}

// anchorKey returns the composite key used to detect per-anchor
// replacement under Decision 2 of the contract — scoped to the same
// agent so two agents can propose on the same anchor without
// clobbering each other.
func (e *Envelope) anchorKey() anchorKey {
	return anchorKey{
		sessionID: e.SessionID,
		specSlug:  e.Target.SpecSlug,
		anchorK:   string(e.Target.Anchor.Kind),
		anchorV:   e.Target.Anchor.Value,
		agent:     e.Agent,
	}
}

type anchorKey struct {
	sessionID string
	specSlug  string
	anchorK   string
	anchorV   string
	agent     string
}
