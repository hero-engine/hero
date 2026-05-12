package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DefaultBranchPatterns are the default patterns for matching branches to spec slugs.
var DefaultBranchPatterns = []string{
	"feat/{{slug}}",
	"feature/{{slug}}",
	"fix/{{slug}}",
	"{{slug}}",
}

// HookEvent represents a git hook event.
type HookEvent struct {
	Name string // "post-checkout", "post-merge", "post-commit", "prepare-commit-msg"
	Args []string
}

// MatchBranch tries to extract a spec slug from a branch name using patterns.
// Returns the slug and true if matched, empty string and false otherwise.
// Patterns use {{slug}} as the placeholder for the spec slug.
func MatchBranch(branch string, patterns []string) (string, bool) {
	for _, pattern := range patterns {
		parts := strings.SplitN(pattern, "{{slug}}", 2)
		if len(parts) != 2 {
			continue
		}
		prefix := parts[0]
		suffix := parts[1]

		if !strings.HasPrefix(branch, prefix) {
			continue
		}
		remainder := strings.TrimPrefix(branch, prefix)

		if suffix == "" {
			if remainder != "" {
				return remainder, true
			}
			continue
		}

		if strings.HasSuffix(remainder, suffix) {
			slug := strings.TrimSuffix(remainder, suffix)
			if slug != "" {
				return slug, true
			}
		}
	}
	return "", false
}

// FindMatchingSpec finds a spec in the corpus that matches the given slug.
// Uses longest-prefix matching if exact match fails.
func FindMatchingSpec(slug string, specs []string) (string, bool) {
	// Exact match first
	for _, s := range specs {
		if s == slug {
			return s, true
		}
	}

	// Longest-prefix match
	best := ""
	for _, s := range specs {
		if strings.HasPrefix(slug, s) && len(s) > len(best) {
			best = s
		}
	}
	if best != "" {
		return best, true
	}

	return "", false
}

// currentBranch returns the current git branch name by running git symbolic-ref HEAD.
func currentBranch() (string, error) {
	out, err := exec.Command("git", "symbolic-ref", "--short", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HandlePostCheckout handles the post-checkout hook.
// args: prevHead, newHead, isBranchCheckout (1 if branch, 0 if file)
// Returns the spec slug that was transitioned, or "" if no match.
func HandlePostCheckout(args []string, specs []string, patterns []string) (slug string, oldStatus string, newStatus string) {
	// Need at least 3 args: prevHead, newHead, isBranchCheckout
	if len(args) < 3 {
		return "", "", ""
	}

	// Only handle branch checkouts (not file checkouts)
	if args[2] != "1" {
		return "", "", ""
	}

	branch, err := currentBranch()
	if err != nil || branch == "" {
		return "", "", ""
	}

	matchedSlug, ok := MatchBranch(branch, patterns)
	if !ok {
		return "", "", ""
	}

	found, ok := FindMatchingSpec(matchedSlug, specs)
	if !ok {
		return "", "", ""
	}

	return found, "planning", "delivering"
}

// HandlePostMerge handles the post-merge hook.
// Returns the spec slug transitioned to done, or "".
func HandlePostMerge(args []string, specs []string, patterns []string) (slug string) {
	branch, err := currentBranch()
	if err != nil || branch == "" {
		return ""
	}

	matchedSlug, ok := MatchBranch(branch, patterns)
	if !ok {
		return ""
	}

	found, ok := FindMatchingSpec(matchedSlug, specs)
	if !ok {
		return ""
	}

	return found
}

// LogEvent appends an event to .hero/events.log as JSONL.
// The event map should contain an "event" key and any other relevant fields.
// A "t" field with RFC3339 timestamp is added automatically.
func LogEvent(eventsLogPath string, event map[string]string) error {
	event["t"] = time.Now().UTC().Format(time.RFC3339)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	f, err := os.OpenFile(eventsLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening events log: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}
