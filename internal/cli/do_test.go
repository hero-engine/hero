package cli

import (
	"strings"
	"testing"
)

func TestDoFixBug(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "fix", "the", "login", "bug")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "/diagnose") {
		t.Errorf("expected /diagnose recommendation, got: %q", output)
	}
}

func TestDoDesignFeature(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "design", "a", "new", "auth", "system")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "/design") {
		t.Errorf("expected /design recommendation, got: %q", output)
	}
}

func TestDoReviewPR(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "review", "the", "pull", "request")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "/review") {
		t.Errorf("expected /review recommendation, got: %q", output)
	}
}

func TestDoScanProject(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "scan", "this", "project")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "hero scan") {
		t.Errorf("expected hero scan recommendation, got: %q", output)
	}
}

func TestDoCapture(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "capture", "my", "thoughts")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "note") {
		t.Errorf("expected note recommendation, got: %q", output)
	}
}

func TestDoCheckHealth(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "health", "check")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "hero check") {
		t.Errorf("expected hero check recommendation, got: %q", output)
	}
}

func TestDoDashboard(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "show", "me", "the", "dashboard")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "hero dashboard") {
		t.Errorf("expected hero dashboard recommendation, got: %q", output)
	}
}

func TestDoRelease(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "prepare", "a", "release")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "/release") {
		t.Errorf("expected /release recommendation, got: %q", output)
	}
}

func TestDoDecide(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "should", "we", "use", "postgres", "or", "mysql")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "/decide") {
		t.Errorf("expected /decide recommendation, got: %q", output)
	}
}

func TestDoCompose(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "break", "down", "the", "epic")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "/compose") {
		t.Errorf("expected /compose recommendation, got: %q", output)
	}
}

func TestDoRetro(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "retrospective", "on", "the", "auth", "feature")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "/retro") {
		t.Errorf("expected /retro recommendation, got: %q", output)
	}
}

func TestDoDocs(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "write", "documentation", "for", "the", "api")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "/docs") {
		t.Errorf("expected /docs recommendation, got: %q", output)
	}
}

func TestDoConvention(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "document", "the", "coding", "convention")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "/convention") {
		t.Errorf("expected /convention recommendation, got: %q", output)
	}
}

func TestDoSearch(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "find", "the", "auth", "spec")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "hero search") {
		t.Errorf("expected hero search recommendation, got: %q", output)
	}
}

func TestDoKnowledge(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "list", "conventions")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "hero knowledge") {
		t.Errorf("expected hero knowledge recommendation, got: %q", output)
	}
}

func TestDoNoMatch(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "xyzzy", "plugh")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "No matching workflow") {
		t.Errorf("expected no-match message, got: %q", output)
	}
}

func TestDoShowsBasedOn(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("do", "fix", "the", "bug")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	if !strings.Contains(output, "Based on:") {
		t.Errorf("expected 'Based on:' in output, got: %q", output)
	}
}

func TestDoMultipleMatches(t *testing.T) {
	_ = newTestEnv(t)

	// "implement a new feature" should match both /design and /deliver
	output, err := runCmd("do", "implement", "a", "new", "feature")
	if err != nil {
		t.Fatalf("do returned error: %v", err)
	}

	// Should show multiple matching workflows
	if !strings.Contains(output, "Matching workflows") || !strings.Contains(output, "Recommended") {
		// At least one match should appear
		if !strings.Contains(output, "/design") && !strings.Contains(output, "/deliver") {
			t.Errorf("expected design or deliver in output, got: %q", output)
		}
	}
}

func TestDoRequiresArgs(t *testing.T) {
	_ = newTestEnv(t)

	_, err := runCmd("do")
	if err == nil {
		t.Fatal("do with no args should fail")
	}
}

// Unit tests for matchRoutes

func TestMatchRoutesFixBug(t *testing.T) {
	matches := matchRoutes("fix the login bug")
	if len(matches) == 0 {
		t.Fatal("expected matches for 'fix the login bug'")
	}
	if matches[0].Command != "/diagnose" {
		t.Errorf("top match = %q, want /diagnose", matches[0].Command)
	}
}

func TestMatchRoutesDesign(t *testing.T) {
	matches := matchRoutes("design a new payment flow")
	if len(matches) == 0 {
		t.Fatal("expected matches for 'design a new payment flow'")
	}
	if matches[0].Command != "/design" {
		t.Errorf("top match = %q, want /design", matches[0].Command)
	}
}

func TestMatchRoutesEmpty(t *testing.T) {
	matches := matchRoutes("xyzzy plugh")
	if len(matches) != 0 {
		t.Errorf("expected no matches for nonsense, got %d: %v", len(matches), matches)
	}
}

func TestMatchRoutesLimitedToThree(t *testing.T) {
	// Use a very broad query that might match many rules
	matches := matchRoutes("review check validate fix build design release")
	if len(matches) > 3 {
		t.Errorf("matches should be limited to 3, got %d", len(matches))
	}
}

func TestMatchRoutesRootCausePhrase(t *testing.T) {
	matches := matchRoutes("find the root cause of this crash")
	if len(matches) == 0 {
		t.Fatal("expected matches for root cause query")
	}
	if matches[0].Command != "/diagnose" {
		t.Errorf("top match = %q, want /diagnose", matches[0].Command)
	}
}

func TestMatchRoutesNotWorking(t *testing.T) {
	matches := matchRoutes("the auth flow is not working")
	if len(matches) == 0 {
		t.Fatal("expected matches for 'not working'")
	}
	if matches[0].Command != "/diagnose" {
		t.Errorf("top match = %q, want /diagnose", matches[0].Command)
	}
}

func TestMatchRoutesBreakDown(t *testing.T) {
	matches := matchRoutes("break down the user management epic")
	if len(matches) == 0 {
		t.Fatal("expected matches for 'break down'")
	}
	if matches[0].Command != "/compose" {
		t.Errorf("top match = %q, want /compose", matches[0].Command)
	}
}

func TestMatchRoutesShouldWe(t *testing.T) {
	matches := matchRoutes("should we use redis or memcached")
	if len(matches) == 0 {
		t.Fatal("expected matches for 'should we'")
	}
	if matches[0].Command != "/decide" {
		t.Errorf("top match = %q, want /decide", matches[0].Command)
	}
}
