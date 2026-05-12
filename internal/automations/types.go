package automations

import (
	"time"
)

// AutomationConfig is a single automation definition loaded from YAML.
type AutomationConfig struct {
	Name     string         `yaml:"name" json:"name"`
	Enabled  bool           `yaml:"enabled" json:"enabled"`
	Trigger  TriggerConfig  `yaml:"trigger" json:"trigger"`
	Action   ActionConfig   `yaml:"action" json:"action"`
	Approval ApprovalConfig `yaml:"approval" json:"approval"`
	FilePath string         `yaml:"-" json:"file_path"`
}

// TriggerConfig defines what event fires the automation.
type TriggerConfig struct {
	Type   string            `yaml:"type" json:"type"`     // tracker, webhook, schedule, file, feed
	Event  string            `yaml:"event" json:"event"`   // e.g. issue_created, push, cron expr
	Filter map[string]string `yaml:"filter" json:"filter"` // key-value filters on the event payload
}

// ActionConfig defines what to do when triggered.
type ActionConfig struct {
	Command string  `yaml:"command" json:"command"` // diagnose, deliver, check, context
	Args    string  `yaml:"args" json:"args"`       // template string with {{variables}}
	Mode    string  `yaml:"mode" json:"mode"`       // autopilot, supervised
	Model   string  `yaml:"model" json:"model"`
	Budget  float64 `yaml:"budget" json:"budget"`
}

// ApprovalConfig defines whether human approval is needed.
type ApprovalConfig struct {
	Required  bool     `yaml:"required" json:"required"`
	Gate      string   `yaml:"gate" json:"gate"`       // status to park at (e.g. "in-review")
	Notify    string   `yaml:"notify" json:"notify"`   // notification channel
	Reviewers []string `yaml:"reviewers" json:"reviewers"`
}

// AutomationStatus tracks runtime state of an automation.
type AutomationStatus struct {
	Config    AutomationConfig `json:"config"`
	LastFired time.Time        `json:"last_fired,omitempty"`
	FireCount int              `json:"fire_count"`
	LastError string           `json:"last_error,omitempty"`
}

// LogEntry is one execution record.
type LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Automation string    `json:"automation"`
	Trigger    string    `json:"trigger"`
	Action     string    `json:"action"`
	Status     string    `json:"status"` // fired, completed, failed, approval_pending
	JobID      string    `json:"job_id,omitempty"`
	Error      string    `json:"error,omitempty"`
}
