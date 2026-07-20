package cli

import (
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestValidateJiraBulkPush_RequiresBoundedTransitionCohort(t *testing.T) {
	cfg := config.Config{Jira: &config.JiraConfig{PushStatusTransitions: map[string]string{
		"delivering": "31",
	}}}

	if err := validateJiraBulkPush(cfg, false, ""); err != nil {
		t.Fatalf("dry run should remain unrestricted: %v", err)
	}
	if err := validateJiraBulkPush(cfg, true, ""); err == nil {
		t.Fatal("unbounded bulk push was not rejected")
	}
	if err := validateJiraBulkPush(cfg, true, "completed"); err == nil {
		t.Fatal("comment-only fallback cohort was not rejected")
	}
	if err := validateJiraBulkPush(cfg, true, "delivering"); err != nil {
		t.Fatalf("configured transition cohort should be accepted: %v", err)
	}
}
