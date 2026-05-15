package governance

import "time"

// PolicyNode is itself a graph node. Policies are versioned, supersede
// prior versions, and are themselves classified (default Restricted).
// "What policy was in effect at time T" is answered by selecting the
// PolicyNode whose effective range covers T.
type PolicyNode struct {
	ID             string         `json:"id"`
	OrgID          string         `json:"org_id"`
	Version        int            `json:"version"`
	Effective      time.Time      `json:"effective"`
	Supersedes     string         `json:"supersedes,omitempty"`
	Rules          []Rule         `json:"rules"`
	Authors        []string       `json:"authors"`
	ReviewedBy     []string       `json:"reviewed_by,omitempty"`
	Classification Classification `json:"classification"`
}

// Rule is one entry inside a PolicyNode. The concrete shape of Match
// and Action depends on Kind; this struct carries the union and the
// human-readable Reason that audit events reference.
type Rule struct {
	ID     string   `json:"id"`
	Kind   RuleKind `json:"kind"`
	Match  Matcher  `json:"match"`
	Action Action   `json:"action"`
	Reason string   `json:"reason"`
}

// RuleKind names the category of a Rule. The five built-in kinds
// correspond to the rule types named in the governance spec.
type RuleKind string

const (
	// RulePathExclude rejects ingress for matching paths.
	RulePathExclude RuleKind = "path_exclude"
	// RuleRedact masks matching substrings on ingest.
	RuleRedact RuleKind = "redact"
	// RuleMinClassification raises the classification of matching nodes
	// to at least the named level.
	RuleMinClassification RuleKind = "min_classification"
	// RuleAutoSubject attaches a subject to matching nodes on ingest.
	RuleAutoSubject RuleKind = "auto_subject"
	// RuleBlock refuses to capture matching content entirely.
	RuleBlock RuleKind = "block"
)

// Matcher is the predicate side of a Rule. Implementations live in the
// enforcement engine; the contract carries the matcher as opaque JSON so
// new matcher shapes don't break the wire.
type Matcher struct {
	// Glob is a file-path pattern for path-oriented rules.
	Glob string `json:"glob,omitempty"`
	// Pattern is a regex for content-oriented rules.
	Pattern string `json:"pattern,omitempty"`
	// SubjectType narrows the match to nodes tagged with this subject type.
	SubjectType SubjectType `json:"subject_type,omitempty"`
	// MinClassification narrows the match to nodes at or above this level.
	MinClassification Classification `json:"min_classification,omitempty"`
}

// Action is the consequence side of a Rule. Fields are populated only
// for the kinds that use them.
type Action struct {
	// Decision is the allow/deny/redact verdict.
	Decision Decision `json:"decision"`
	// SetClassification is set when the action raises classification.
	SetClassification Classification `json:"set_classification,omitempty"`
	// AddSubject is set when the action attaches a subject.
	AddSubject *Subject `json:"add_subject,omitempty"`
	// RequireEgress names a purpose required for the action (e.g.
	// "deny unless purpose is llm_self_hosted").
	RequireEgress Purpose `json:"require_egress,omitempty"`
}

// Decision is the allow/deny/redact verdict carried in an Action or
// returned by Retriever.Filter for a single node.
type Decision string

const (
	// DecisionAllow permits the operation.
	DecisionAllow Decision = "allow"
	// DecisionDeny rejects the operation.
	DecisionDeny Decision = "deny"
	// DecisionRedact permits the operation with content masked.
	DecisionRedact Decision = "redact"
)
