// Package extract is hero's Tier-2 ingest layer — the LLM-driven step
// that reads dense prose (notes, specs, ingested docs) and surfaces
// structured graph nodes (Decisions, Concepts, Alternatives) the
// digester can rank and link.
//
// Provider-agnostic by design. The extractor works with any
// runner.LLMProvider — Anthropic, OpenAI, Azure, or future providers
// (Gemini, local models, etc.) — so teams aren't locked to one model.
//
// Design principles:
//
//   1. Pluggable. Provider is injected, not hardcoded. Hero's
//      extraction works with whatever model the team has access to.
//
//   2. Idempotent. Each source has a content_hash; we skip re-
//      extracting unchanged content. Re-extracting changed content
//      replaces (via the bitemporal model — prior version preserved
//      in history).
//
//   3. Cheap when the provider supports it. CacheSystem hint enables
//      Anthropic prompt caching (~90% cost reduction on the cached
//      portion); other providers ignore the hint.
//
//   4. Best-effort. LLM calls fail open: extraction errors get
//      logged and skipped, never break ingest.
//
//   5. Verifiable. Every extracted node carries a source pointer
//      back to the originating Note/Spec/Document so a human can
//      audit the lineage.
package extract

import (
	"context"
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/runner"
)

// LLM is the narrow interface extraction needs. Any implementation
// of runner.LLMProvider satisfies this — the package doesn't need
// the full Tools/StopReason surface.
type LLM interface {
	Chat(ctx context.Context, req runner.ChatRequest) (*runner.ChatResponse, error)
	Name() string
}

// Client is the extraction-flavored facade over an LLM provider.
// It picks an appropriate model for extraction work (small + fast),
// enables prompt caching when the provider supports it, and
// surfaces token + cache stats for cost diagnostics.
type Client struct {
	provider LLM
	model    string
}

// NewClient wraps any LLMProvider for extraction use. modelOverride
// is optional — pass "" to use the provider-appropriate default
// extraction model (Haiku for Anthropic, gpt-4o-mini for OpenAI, etc.).
func NewClient(provider LLM, modelOverride string) *Client {
	c := &Client{provider: provider}
	c.model = modelOverride
	if c.model == "" {
		c.model = defaultModelFor(provider.Name())
	}
	return c
}

// NewDefaultClient builds an Anthropic-backed extractor from
// ANTHROPIC_API_KEY (the most common case). Returns a client with
// HasKey()==false if no key is set; callers should check before
// running extraction.
func NewDefaultClient() *Client {
	key := os.Getenv("ANTHROPIC_API_KEY")
	provider := runner.NewAnthropicProvider(key)
	c := &Client{provider: provider, model: defaultModelFor("anthropic")}
	if key == "" {
		c.provider = nil // signal HasKey=false; Run will return ErrNoAPIKey
	}
	return c
}

// HasKey reports whether the client has a usable backend.
func (c *Client) HasKey() bool { return c.provider != nil }

// ErrNoAPIKey is returned when extraction is attempted without a
// configured provider (no key set, no provider injected).
var ErrNoAPIKey = fmt.Errorf("no LLM provider configured — set ANTHROPIC_API_KEY (or pass a provider) — extraction skipped")

// Request is the input to Run.
type Request struct {
	System string // cached across calls when provider supports it
	User   string // per-extraction prompt body
	MaxOut int    // max output tokens; default 1024
}

// Response is what Run returns. Cache stats are 0 for providers
// that don't support prompt caching.
type Response struct {
	Text         string
	InputTokens  int
	OutputTokens int
	CacheReads   int
	CacheCreates int
	ProviderName string
}

// Run sends a single extraction request through the provider.
func (c *Client) Run(ctx context.Context, req Request) (*Response, error) {
	if !c.HasKey() {
		return nil, ErrNoAPIKey
	}
	if req.MaxOut == 0 {
		req.MaxOut = 1024
	}

	chatReq := runner.ChatRequest{
		Model:       c.model,
		System:      req.System,
		Messages:    []runner.Message{{Role: "user", Content: req.User}},
		MaxTokens:   req.MaxOut,
		CacheSystem: true, // ignored by providers without prompt caching
	}
	resp, err := c.provider.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}
	return &Response{
		Text:         resp.Text,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		CacheReads:   resp.CacheReadInputTokens,
		CacheCreates: resp.CacheCreationInputTokens,
		ProviderName: c.provider.Name(),
	}, nil
}

// defaultModelFor returns a sensible small/fast model name for the
// given provider. Callers can override via NewClient's modelOverride.
func defaultModelFor(provider string) string {
	switch provider {
	case "anthropic":
		return "claude-haiku-4-5-20251001"
	case "openai", "azure":
		return "gpt-4o-mini"
	default:
		return ""
	}
}
