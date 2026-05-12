package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// WebhookEvent is the parsed result of a GitHub webhook delivery.
type WebhookEvent struct {
	Action       string
	EventType    string // "pull_request", "installation", etc.
	Installation *InstallationPayload
	PullRequest  *PullRequestPayload
	Repository   *RepositoryPayload
	Sender       *SenderPayload
}

// InstallationPayload is the installation object from webhook events.
type InstallationPayload struct {
	ID      int64  `json:"id"`
	Account struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
		Type  string `json:"type"` // "Organization" or "User"
	} `json:"account"`
	AppID int64 `json:"app_id"`
}

// PullRequestPayload is the pull_request object from webhook events.
type PullRequestPayload struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	Head    struct {
		SHA string `json:"sha"`
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		SHA string `json:"sha"`
		Ref string `json:"ref"`
	} `json:"base"`
	User SenderPayload `json:"user"`
}

// RepositoryPayload is the repository object from webhook events.
type RepositoryPayload struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Private bool `json:"private"`
}

// SenderPayload is the user who triggered the event.
type SenderPayload struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}

// ParseWebhook validates the webhook signature and parses the event.
func ParseWebhook(r *http.Request, secret string) (*WebhookEvent, error) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	// Verify signature
	if secret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if sig == "" {
			return nil, fmt.Errorf("missing X-Hub-Signature-256 header")
		}
		if !verifySignature(body, sig, secret) {
			return nil, fmt.Errorf("invalid webhook signature")
		}
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		return nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	// Parse into a generic map to extract common fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing webhook body: %w", err)
	}

	event := &WebhookEvent{
		EventType: eventType,
	}

	// Extract action
	if actionRaw, ok := raw["action"]; ok {
		json.Unmarshal(actionRaw, &event.Action)
	}

	// Extract installation
	if instRaw, ok := raw["installation"]; ok {
		var inst InstallationPayload
		json.Unmarshal(instRaw, &inst)
		event.Installation = &inst
	}

	// Extract pull_request
	if prRaw, ok := raw["pull_request"]; ok {
		var pr PullRequestPayload
		json.Unmarshal(prRaw, &pr)
		event.PullRequest = &pr
	}

	// Extract repository
	if repoRaw, ok := raw["repository"]; ok {
		var repo RepositoryPayload
		json.Unmarshal(repoRaw, &repo)
		event.Repository = &repo
	}

	// Extract sender
	if senderRaw, ok := raw["sender"]; ok {
		var sender SenderPayload
		json.Unmarshal(senderRaw, &sender)
		event.Sender = &sender
	}

	return event, nil
}

// verifySignature checks the HMAC-SHA256 webhook signature.
func verifySignature(body []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sigHex := signature[7:]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sigHex), []byte(expected))
}
