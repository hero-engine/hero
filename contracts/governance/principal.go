package governance

// PrincipalKind distinguishes the two principal categories: human users
// and AI agents acting on behalf of a user.
type PrincipalKind string

const (
	// PrincipalUser is a human principal authenticated as themselves.
	PrincipalUser PrincipalKind = "user"
	// PrincipalAgent is an AI agent principal acting on behalf of a user.
	PrincipalAgent PrincipalKind = "agent"
)

// Principal identifies the caller of a retrieval or write. Implementations
// in the enforcement engine wrap user and agent identities; this contract
// only requires the three accessors.
type Principal interface {
	// ID returns the stable identifier for this principal.
	ID() string
	// Kind returns whether the principal is a user or an agent.
	Kind() PrincipalKind
	// Scope returns the read+write scope the principal is authorized for.
	Scope() Scope
}

// Scope bounds what a principal may read or write. An empty slice means
// "no restriction on this axis". Classification is the maximum sensitivity
// the principal may read.
type Scope struct {
	Repos          []string       `json:"repos,omitempty"`
	Subjects       []Subject      `json:"subjects,omitempty"`
	Classification Classification `json:"classification"`
	Kinds          []string       `json:"kinds,omitempty"`
}
