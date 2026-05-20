package config

import (
	"testing"
	"time"
)

func TestHealthTTLDurationDefault(t *testing.T) {
	// Nil receiver → default. Defensive — the daemon should never
	// crash on a missing Serve block.
	var c *ServeConfig
	if got := c.HealthTTLDuration(); got != 5*time.Minute {
		t.Errorf("nil HealthTTLDuration = %v, want 5m", got)
	}

	// Empty field → default.
	cfg := &ServeConfig{}
	if got := cfg.HealthTTLDuration(); got != 5*time.Minute {
		t.Errorf("empty HealthTTLDuration = %v, want 5m", got)
	}
}

func TestHealthTTLDurationParses(t *testing.T) {
	cases := map[string]time.Duration{
		"15m":    15 * time.Minute,
		"30s":    30 * time.Second,
		"1h":     time.Hour,
		"2h30m":  2*time.Hour + 30*time.Minute,
	}
	for in, want := range cases {
		cfg := &ServeConfig{HealthTTL: in}
		if got := cfg.HealthTTLDuration(); got != want {
			t.Errorf("HealthTTL=%q → %v, want %v", in, got, want)
		}
	}
}

func TestHealthTTLDurationInvalidFallsBack(t *testing.T) {
	cases := []string{
		"not-a-duration",
		"15 minutes",
		"0",
		"-5m",
		"",
	}
	for _, in := range cases {
		cfg := &ServeConfig{HealthTTL: in}
		if got := cfg.HealthTTLDuration(); got != 5*time.Minute {
			t.Errorf("HealthTTL=%q → %v, want default 5m", in, got)
		}
	}
}
