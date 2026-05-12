package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAIProvider implements LLMProvider for OpenAI and Azure OpenAI.
type OpenAIProvider struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

// NewOpenAIProvider creates an OpenAI/Azure provider.
func NewOpenAIProvider(apiKey, endpoint string) *OpenAIProvider {
	return &OpenAIProvider{apiKey: apiKey, endpoint: endpoint, client: &http.Client{}}
}

func (o *OpenAIProvider) Name() string {
	if o.endpoint != "https://api.openai.com/v1" {
		return "azure"
	}
	return "openai"
}

func (o *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = "gpt-4o"
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	messages := convertMessagesOpenAI(req.System, req.Messages)

	body := map[string]interface{}{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   messages,
	}
	if len(req.Tools) > 0 {
		body["tools"] = convertToolsOpenAI(req.Tools)
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := o.endpoint + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(httpReq)
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

	return parseOpenAIResponse(respBody)
}

func convertMessagesOpenAI(system string, msgs []Message) []map[string]interface{} {
	var out []map[string]interface{}
	if system != "" {
		out = append(out, map[string]interface{}{
			"role":    "system",
			"content": system,
		})
	}
	for _, m := range msgs {
		out = append(out, map[string]interface{}{
			"role":    m.Role,
			"content": m.Content,
		})
	}
	return out
}

func convertToolsOpenAI(tools []ToolDefinition) []map[string]interface{} {
	var out []map[string]interface{}
	for _, t := range tools {
		out = append(out, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			},
		})
	}
	return out
}

func parseOpenAIResponse(data []byte) (*ChatResponse, error) {
	var raw struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	resp := &ChatResponse{
		InputTokens:  raw.Usage.PromptTokens,
		OutputTokens: raw.Usage.CompletionTokens,
	}

	if len(raw.Choices) > 0 {
		choice := raw.Choices[0]
		resp.StopReason = choice.FinishReason
		resp.Text = choice.Message.Content

		for _, tc := range choice.Message.ToolCalls {
			var input map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &input)
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			})
		}
	}

	return resp, nil
}
