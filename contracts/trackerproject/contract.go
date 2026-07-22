package trackerproject

import "time"

const Version = "tracker-project-snapshot/v1"

type Snapshot struct {
	Version      string      `json:"version"`
	Provider     string      `json:"provider"`
	ConnectionID string      `json:"connection_id,omitempty"`
	Project      Project     `json:"project"`
	Board        *Board      `json:"board,omitempty"`
	Iterations   []Iteration `json:"iterations"`
	Items        []Item      `json:"items"`
	GeneratedAt  time.Time   `json:"generated_at"`
	Cursor       string      `json:"cursor,omitempty"`
	Complete     bool        `json:"complete"`
	Truncated    bool        `json:"truncated"`
	Stale        bool        `json:"stale"`
	Error        *Error      `json:"error,omitempty"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type Board struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Iteration struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Goal  string `json:"goal,omitempty"`
	State string `json:"state"`
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type Item struct {
	TrackerID      string   `json:"tracker_id"`
	LocalSlug      string   `json:"local_slug,omitempty"`
	Title          string   `json:"title"`
	Type           string   `json:"type"`
	NativeStatus   string   `json:"native_status"`
	StatusCategory string   `json:"status_category"`
	Assignee       string   `json:"assignee,omitempty"`
	Rank           int      `json:"rank"`
	IterationIDs   []string `json:"iteration_ids"`
	URL            string   `json:"url,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
