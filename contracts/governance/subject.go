package governance

// SubjectType names the category of a Subject (e.g. "customer", "repo",
// "incident"). Open-ended; this package ships representative built-ins
// and orgs add their own.
type SubjectType string

const (
	// SubjectCustomer tags nodes that are about a named customer.
	SubjectCustomer SubjectType = "customer"
	// SubjectRepo tags nodes that belong to a named source repository.
	SubjectRepo SubjectType = "repo"
	// SubjectSystem tags nodes that describe an internal system.
	SubjectSystem SubjectType = "system"
)

// Subject describes what a node is *about* — not who can see it.
// Policies match on subjects to restrict reads (e.g. "anything about
// customer acme is visible only to principals scoped to customer:acme").
type Subject struct {
	Type  SubjectType `json:"type"`
	ID    string      `json:"id"`
	Label string      `json:"label,omitempty"`
}
