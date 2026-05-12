package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"

// AnthropicProvider implements LLMProvider for the Claude API.
type AnthropicProvider struct {
	apiKey string
	client *http.Client
}

// NewAnthropicProvider creates an Anthropic provider.
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{apiKey: apiKey, client: &http.Client{}}
}

func (a *AnthropicProvider) Name() string { return "anthropic" }

func (a *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-6-20250514"
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	// Build request body
	body := map[string]interface{}{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   convertMessages(req.Messages),
	}
	if req.System != "" {
		if req.CacheSystem {
			// Send system as a content-block array with cache_control
			// so the prompt is cached (5-min ephemeral) for high-hit-
			// rate workloads like Tier-2 extraction.
			body["system"] = []map[string]any{
				{
					"type":          "text",
					"text":          req.System,
					"cache_control": map[string]any{"type": "ephemeral"},
				},
			}
		} else {
			body["system"] = req.System
		}
	}
	if len(req.Tools) > 0 {
		body["tools"] = convertToolsAnthropic(req.Tools)
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPI, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return parseAnthropicResponse(respBody)
}

func convertMessages(msgs []Message) []map[string]interface{} {
	var out []map[string]interface{}
	for _, m := range msgs {
		out = append(out, map[string]interface{}{
			"role":    m.Role,
			"content": m.Content,
		})
	}
	return out
}

func convertToolsAnthropic(tools []ToolDefinition) []map[string]interface{} {
	var out []map[string]interface{}
	for _, t := range tools {
		out = append(out, map[string]interface{}{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": t.InputSchema,
		})
	}
	return out
}

func parseAnthropicResponse(data []byte) (*ChatResponse, error) {
	var raw struct {
		Content []struct {
			Type  string                 `json:"type"`
			Text  string                 `json:"text,omitempty"`
			ID    string                 `json:"id,omitempty"`
			Name  string                 `json:"name,omitempty"`
			Input map[string]interface{} `json:"input,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	resp := &ChatResponse{
		StopReason:               raw.StopReason,
		InputTokens:              raw.Usage.InputTokens,
		OutputTokens:             raw.Usage.OutputTokens,
		CacheReadInputTokens:     raw.Usage.CacheReadInputTokens,
		CacheCreationInputTokens: raw.Usage.CacheCreationInputTokens,
	}

	for _, block := range raw.Content {
		switch block.Type {
		case "text":
			resp.Text += block.Text
		case "tool_use":
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	return resp, nil
}
