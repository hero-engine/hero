package runner

import (
	"context"
	"fmt"
	"strings"
)

// Message is a single message in the conversation.
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ToolDefinition describes a tool the LLM can call.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolCall represents the LLM requesting a tool invocation.
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ChatRequest is the input to a provider's Chat method.
type ChatRequest struct {
	Model     string
	System    string
	Messages  []Message
	Tools     []ToolDefinition
	MaxTokens int

	// CacheSystem hints to the provider that the System prompt
	// should be cached across calls when the provider supports it
	// (Anthropic Messages API). Providers without prompt caching
	// (most others) ignore this — the request still works, just
	// without the cache discount.
	CacheSystem bool
}

// ChatResponse is the output from a provider's Chat method.
type ChatResponse struct {
	StopReason   string
	Text         string
	ToolCalls    []ToolCall
	InputTokens  int
	OutputTokens int

	// Cache stats — populated by providers that support prompt
	// caching (Anthropic). Other providers leave both at 0.
	CacheReadInputTokens     int
	CacheCreationInputTokens int
}

// LLMProvider is the interface for LLM API providers.
type LLMProvider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	Name() string
}

// ProviderRegistry maps provider names to constructors.
var providers = map[string]func(apiKey string, opts map[string]string) LLMProvider{
	"anthropic": func(key string, opts map[string]string) LLMProvider {
		return NewAnthropicProvider(key)
	},
	"openai": func(key string, opts map[string]string) LLMProvider {
		return NewOpenAIProvider(key, "https://api.openai.com/v1")
	},
	"azure": func(key string, opts map[string]string) LLMProvider {
		endpoint := opts["endpoint"]
		if endpoint == "" {
			endpoint = "https://your-resource.openai.azure.com"
		}
		return NewOpenAIProvider(key, endpoint)
	},
}

// GetProvider returns a provider by name.
func GetProvider(name, apiKey string, opts map[string]string) (LLMProvider, error) {
	constructor, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q (supported: anthropic, openai, azure)", name)
	}
	return constructor(apiKey, opts), nil
}

// DetectProvider guesses the provider from a model name.
func DetectProvider(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "claude"):
		return "anthropic"
	case strings.Contains(lower, "gpt") || strings.Contains(lower, "o1") || strings.Contains(lower, "o3"):
		return "openai"
	default:
		return "anthropic"
	}
}

// ResolveAPIKey finds the API key for a provider from flag, env, or stored creds.
func ResolveAPIKey(provider, flagKey string) string {
	if flagKey != "" {
		return flagKey
	}

	envVars := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
		"azure":     "AZURE_OPENAI_KEY",
	}

	if envVar, ok := envVars[provider]; ok {
		if val := getenv(envVar); val != "" {
			return val
		}
	}

	return ""
}

var getenv = func(key string) string {
	// Imported at call site to avoid os import in provider.go
	return ""
}
