package attention

const (
	FocusInbox = "inbox"
	FocusToday = "today"
	FocusLater = "later"
	FocusDone  = "done"
)

type FocusItem struct {
	SchemaVersion int              `json:"schema_version"`
	ID            string           `json:"id"`
	Project       ProjectReference `json:"project"`
	Title         string           `json:"title"`
	Prompt        string           `json:"prompt"`
	Lifecycle     string           `json:"lifecycle"`
	Revision      int64            `json:"revision"`
	CreatedAt     string           `json:"created_at"`
	UpdatedAt     string           `json:"updated_at"`
}

type CreateFocusRequest struct {
	SchemaVersion int              `json:"schema_version"`
	ID            string           `json:"id"`
	Project       ProjectReference `json:"project"`
	Title         string           `json:"title"`
	Prompt        string           `json:"prompt"`
	Lifecycle     string           `json:"lifecycle"`
	CreatedAt     string           `json:"created_at"`
}

type UpdateFocusRequest struct {
	SchemaVersion int    `json:"schema_version"`
	ID            string `json:"id"`
	Title         string `json:"title,omitempty"`
	Prompt        string `json:"prompt,omitempty"`
	Lifecycle     string `json:"lifecycle,omitempty"`
	Revision      int64  `json:"revision"`
	UpdatedAt     string `json:"updated_at"`
}
