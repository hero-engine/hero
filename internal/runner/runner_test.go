package runner

import (
	"testing"
)

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"claude-sonnet-4-6", "anthropic"},
		{"claude-opus-4-6", "anthropic"},
		{"gpt-4o", "openai"},
		{"gpt-4o-mini", "openai"},
		{"o3", "openai"},
		{"o1-preview", "openai"},
		{"some-custom-model", "anthropic"},
	}

	for _, tt := range tests {
		got := DetectProvider(tt.model)
		if got != tt.want {
			t.Errorf("DetectProvider(%q) = %q, want %q", tt.model, got, tt.want)
		}
	}
}

func TestEstimateCost(t *testing.T) {
	cost := estimateCost("anthropic", 1000, 500)
	if cost <= 0 {
		t.Error("expected positive cost")
	}

	costOAI := estimateCost("openai", 1000, 500)
	if costOAI <= 0 {
		t.Error("expected positive cost for openai")
	}
}

func TestResolveAPIKey_Flag(t *testing.T) {
	key := ResolveAPIKey("anthropic", "sk-test-123")
	if key != "sk-test-123" {
		t.Errorf("expected flag key, got %q", key)
	}
}

func TestResolveAPIKey_Empty(t *testing.T) {
	key := ResolveAPIKey("anthropic", "")
	// With no env var set in test, should return empty
	if key != "" {
		t.Errorf("expected empty key without env, got %q", key)
	}
}

func TestTruncateLog(t *testing.T) {
	short := "hello"
	if truncateLog(short, 10) != short {
		t.Error("short string should not be truncated")
	}

	long := "this is a long string that should be truncated"
	result := truncateLog(long, 10)
	if len(result) > 13 { // 10 + "..."
		t.Errorf("expected truncated string, got %q", result)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	cfg := RunConfig{
		ProjectRoot: "/tmp/nonexistent",
		HeroDir:     "/tmp/nonexistent/.hero",
		Command:     "deliver",
		Args:        "test-slug",
	}
	prompt := buildSystemPrompt(cfg)
	if prompt == "" {
		t.Error("expected non-empty system prompt")
	}
	if !contains(prompt, "headless") {
		t.Error("expected 'headless' in prompt")
	}
}

func TestBuildUserMessage(t *testing.T) {
	tests := []struct {
		command string
		args    string
		want    string
	}{
		{"deliver", "csv-export", "Deliver"},
		{"diagnose", "login-crash", "Diagnose"},
		{"design", "new-feature", "Design"},
		{"", "fix the auth bug", "fix the auth bug"},
	}

	for _, tt := range tests {
		cfg := RunConfig{Command: tt.command, Args: tt.args}
		msg := buildUserMessage(cfg)
		if !contains(msg, tt.want) {
			t.Errorf("buildUserMessage(%q, %q) = %q, want to contain %q", tt.command, tt.args, msg, tt.want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
