package automations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Engine manages automation loading, matching, and execution.
type Engine struct {
	heroDir     string
	projectRoot string
	automations []AutomationStatus
	logPath     string
}

// NewEngine creates an automation engine for the workspace.
func NewEngine(heroDir, projectRoot string) *Engine {
	return &Engine{
		heroDir:     heroDir,
		projectRoot: projectRoot,
		logPath:     filepath.Join(heroDir, "automations", "log.jsonl"),
	}
}

// Load reads all automation YAML files from .hero/automations/.
func (e *Engine) Load() error {
	dir := filepath.Join(e.heroDir, "automations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading automations dir: %w", err)
	}

	e.automations = nil
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg AutomationConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}
		cfg.FilePath = path
		if cfg.Name == "" {
			cfg.Name = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		}
		// Default enabled
		if !cfg.Enabled {
			cfg.Enabled = true
		}
		e.automations = append(e.automations, AutomationStatus{Config: cfg})
	}

	return nil
}

// List returns all loaded automations with their status.
func (e *Engine) List() []AutomationStatus {
	return e.automations
}

// Match finds automations that match an event.
func (e *Engine) Match(eventType, event string, payload map[string]string) []AutomationConfig {
	var matched []AutomationConfig
	for _, as := range e.automations {
		cfg := as.Config
		if !cfg.Enabled {
			continue
		}
		if cfg.Trigger.Type != eventType {
			continue
		}
		if cfg.Trigger.Event != "" && cfg.Trigger.Event != event {
			continue
		}
		if !matchesFilter(cfg.Trigger.Filter, payload) {
			continue
		}
		matched = append(matched, cfg)
	}
	return matched
}

// ResolveArgs templates action args with payload values.
func ResolveArgs(template string, payload map[string]string) string {
	result := template
	for k, v := range payload {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
		result = strings.ReplaceAll(result, "{{issue."+k+"}}", v)
		result = strings.ReplaceAll(result, "{{spec."+k+"}}", v)
		result = strings.ReplaceAll(result, "{{pr."+k+"}}", v)
	}
	return result
}

// Log appends an entry to the automation log.
func (e *Engine) Log(entry LogEntry) error {
	if err := os.MkdirAll(filepath.Dir(e.logPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(e.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// ReadLog returns recent log entries.
func (e *Engine) ReadLog(limit int) ([]LogEntry, error) {
	data, err := os.ReadFile(e.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var entries []LogEntry

	// Read from end
	start := 0
	if len(lines) > limit {
		start = len(lines) - limit
	}
	for _, line := range lines[start:] {
		if line == "" {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Test dry-runs an automation against sample data.
func (e *Engine) Test(name string, payload map[string]string) (string, error) {
	for _, as := range e.automations {
		if as.Config.Name == name {
			cfg := as.Config
			matched := matchesFilter(cfg.Trigger.Filter, payload)
			args := ResolveArgs(cfg.Action.Args, payload)

			var sb strings.Builder
			fmt.Fprintf(&sb, "Automation: %s\n", cfg.Name)
			fmt.Fprintf(&sb, "Trigger: %s / %s\n", cfg.Trigger.Type, cfg.Trigger.Event)
			fmt.Fprintf(&sb, "Filter match: %v\n", matched)
			fmt.Fprintf(&sb, "Action: hero run %s %s\n", cfg.Action.Command, args)
			if cfg.Action.Model != "" {
				fmt.Fprintf(&sb, "Model: %s\n", cfg.Action.Model)
			}
			if cfg.Action.Budget > 0 {
				fmt.Fprintf(&sb, "Budget: $%.2f\n", cfg.Action.Budget)
			}
			if cfg.Approval.Required {
				fmt.Fprintf(&sb, "Approval: required (gate: %s)\n", cfg.Approval.Gate)
			}
			return sb.String(), nil
		}
	}
	return "", fmt.Errorf("automation %q not found", name)
}

// MarkFired updates the last-fired timestamp for an automation.
func (e *Engine) MarkFired(name string) {
	for i := range e.automations {
		if e.automations[i].Config.Name == name {
			e.automations[i].LastFired = time.Now()
			e.automations[i].FireCount++
			return
		}
	}
}

func matchesFilter(filter map[string]string, payload map[string]string) bool {
	for k, v := range filter {
		if pv, ok := payload[k]; !ok || !strings.EqualFold(pv, v) {
			return false
		}
	}
	return true
}
