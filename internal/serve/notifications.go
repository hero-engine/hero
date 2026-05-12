package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// NotifyConfig holds notification settings.
type NotifyConfig struct {
	WebhookURL string   `json:"webhook_url"`
	Events     []string `json:"events"`
	Format     string   `json:"format"` // slack, teams, discord, generic
}

// Notifier dispatches webhook notifications for job events.
type Notifier struct {
	cfg    *NotifyConfig
	client *http.Client
	events map[string]bool
}

// NewNotifier creates a notifier. Returns nil if no webhook is configured.
func NewNotifier(cfg *NotifyConfig) *Notifier {
	if cfg == nil || cfg.WebhookURL == "" {
		return nil
	}
	events := make(map[string]bool)
	for _, e := range cfg.Events {
		events[e] = true
	}
	format := cfg.Format
	if format == "" {
		format = "generic"
	}
	return &Notifier{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
		events: events,
	}
}

// Notify sends a notification if the event type is enabled.
func (n *Notifier) Notify(event string, data map[string]interface{}) {
	if n == nil || !n.events[event] {
		return
	}

	var payload []byte
	var err error

	switch n.cfg.Format {
	case "slack":
		payload, err = formatSlack(event, data)
	case "teams":
		payload, err = formatTeams(event, data)
	default:
		payload, err = formatGeneric(event, data)
	}

	if err != nil {
		log.Printf("hero notify: format error: %v", err)
		return
	}

	go n.send(payload, 1)
}

func (n *Notifier) send(payload []byte, retries int) {
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			time.Sleep(30 * time.Second)
		}

		resp, err := n.client.Post(n.cfg.WebhookURL, "application/json", bytes.NewReader(payload))
		if err != nil {
			log.Printf("hero notify: send error (attempt %d): %v", attempt+1, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return
		}
		log.Printf("hero notify: webhook returned %d (attempt %d)", resp.StatusCode, attempt+1)
	}
}

func formatSlack(event string, data map[string]interface{}) ([]byte, error) {
	title := eventTitle(event)
	details := eventDetails(data)

	blocks := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{"type": "plain_text", "text": title},
			},
			{
				"type": "section",
				"text": map[string]string{"type": "mrkdwn", "text": details},
			},
		},
	}

	// Add action buttons for approval events
	if event == "approval_needed" {
		if jobID, ok := data["job_id"].(string); ok {
			if serverURL, ok := data["server_url"].(string); ok {
				blocks["blocks"] = append(blocks["blocks"].([]map[string]interface{}), map[string]interface{}{
					"type": "actions",
					"elements": []map[string]interface{}{
						{"type": "button", "text": map[string]string{"type": "plain_text", "text": "Approve"}, "url": fmt.Sprintf("%s/api/jobs/%s/approve", serverURL, jobID)},
						{"type": "button", "text": map[string]string{"type": "plain_text", "text": "Reject"}, "url": fmt.Sprintf("%s/api/jobs/%s/reject", serverURL, jobID)},
					},
				})
			}
		}
	}

	return json.Marshal(blocks)
}

func formatTeams(event string, data map[string]interface{}) ([]byte, error) {
	title := eventTitle(event)
	details := eventDetails(data)
	card := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"summary":    title,
		"themeColor": eventColor(event),
		"title":      title,
		"text":       details,
	}
	return json.Marshal(card)
}

func formatGeneric(event string, data map[string]interface{}) ([]byte, error) {
	payload := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().Format(time.RFC3339),
		"data":      data,
	}
	return json.Marshal(payload)
}

func eventTitle(event string) string {
	switch event {
	case "approval_needed":
		return "⏸ Approval needed"
	case "job_completed":
		return "✓ Job completed"
	case "job_failed":
		return "✗ Job failed"
	case "budget_exceeded":
		return "💰 Budget exceeded"
	case "automation_error":
		return "⚠ Automation error"
	default:
		return event
	}
}

func eventColor(event string) string {
	switch event {
	case "job_completed":
		return "00cc00"
	case "job_failed", "automation_error":
		return "cc0000"
	case "approval_needed":
		return "ffcc00"
	case "budget_exceeded":
		return "ff6600"
	default:
		return "0078d4"
	}
}

func eventDetails(data map[string]interface{}) string {
	var parts []string
	if cmd, ok := data["command"].(string); ok {
		parts = append(parts, fmt.Sprintf("*Command:* %s", cmd))
	}
	if args, ok := data["args"].(string); ok && args != "" {
		parts = append(parts, fmt.Sprintf("*Args:* %s", args))
	}
	if user, ok := data["submitted_by"].(string); ok && user != "" {
		parts = append(parts, fmt.Sprintf("*By:* %s", user))
	}
	if cost, ok := data["cost"].(float64); ok && cost > 0 {
		parts = append(parts, fmt.Sprintf("*Cost:* $%.2f", cost))
	}
	if errMsg, ok := data["error"].(string); ok && errMsg != "" {
		parts = append(parts, fmt.Sprintf("*Error:* %s", errMsg))
	}
	if len(parts) == 0 {
		return "_No details_"
	}
	return fmt.Sprintf("%s", joinLines(parts))
}

func joinLines(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n"
		}
		result += p
	}
	return result
}
