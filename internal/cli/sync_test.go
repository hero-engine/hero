package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
	"testing"
)

// --- sync command tests ---

func TestSync_RequiresArg(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("sync", "spec")
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestSync_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("sync", "spec", "some/spec.md")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestSync_NoTrackerConfigured(t *testing.T) {
	env := newTestEnv(t)

	// Create a spec to sync
	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export

## Goal

Export data to CSV.
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	_, err := runCmd("sync", "spec", specPath)
	if err == nil {
		t.Fatal("expected error for no tracker configured")
	}
	if !strings.Contains(err.Error(), "no tracker configured") {
		t.Errorf("error = %q, want 'no tracker configured'", err.Error())
	}
}

func TestSync_InvalidSpecPath(t *testing.T) {
	env := newTestEnv(t)

	// Configure a tracker with token set
	writeTrackerConfig(env, "github", "acme/widgets")
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	_, err := runCmd("sync", "spec", "/nonexistent/spec.md")
	if err == nil {
		t.Fatal("expected error for invalid spec path")
	}
	if !strings.Contains(err.Error(), "parsing spec") {
		t.Errorf("error = %q, want 'parsing spec'", err.Error())
	}
}

func TestSync_MissingToken(t *testing.T) {
	env := newTestEnv(t)

	// Configure tracker but don't set the token env var
	writeTrackerConfig(env, "github", "acme/widgets")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	os.Unsetenv("HERO_TEST_TOKEN")
	_, err := runCmd("sync", "spec", specPath)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "HERO_TEST_TOKEN") {
		t.Errorf("error = %q, want mention of env var", err.Error())
	}
}

// --- link command tests ---

func TestLink_RequiresArgs(t *testing.T) {
	env := newTestEnv(t)
	_ = env

	_, err := runCmd("sync", "link")
	if err == nil {
		t.Fatal("expected error for missing args")
	}

	_, err = runCmd("sync", "link", "spec.md")
	if err == nil {
		t.Fatal("expected error for missing issue-id arg")
	}
}

func TestLink_NoWorkspace(t *testing.T) {
	env := newTestEnvEmpty(t)
	_ = env

	_, err := runCmd("sync", "link", "spec.md", "42")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !strings.Contains(err.Error(), "no hero workspace") {
		t.Errorf("error = %q, want 'no hero workspace'", err.Error())
	}
}

func TestLink_NoTrackerConfigured(t *testing.T) {
	env := newTestEnv(t)

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	_, err := runCmd("sync", "link", specPath, "42")
	if err == nil {
		t.Fatal("expected error for no tracker configured")
	}
	if !strings.Contains(err.Error(), "no tracker configured") {
		t.Errorf("error = %q, want 'no tracker configured'", err.Error())
	}
}

func TestLink_AlreadyLinked(t *testing.T) {
	env := newTestEnv(t)

	writeTrackerConfig(env, "github", "acme/widgets")
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
tracker_id: 99
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	_, err := runCmd("sync", "link", specPath, "42")
	if err == nil {
		t.Fatal("expected error for already-linked spec")
	}
	if !strings.Contains(err.Error(), "already linked") {
		t.Errorf("error = %q, want 'already linked'", err.Error())
	}
}

// --- injectFrontmatterField tests ---

func TestInjectFrontmatterField_NewField(t *testing.T) {
	input := "---\ntitle: My Spec\ntype: feature\n---\n# My Spec\n"
	result := spec.SetFrontmatterField(input, "tracker_id", "42")

	if !strings.Contains(result, "tracker_id: 42") {
		t.Errorf("expected tracker_id: 42, got:\n%s", result)
	}
	if !strings.Contains(result, "title: My Spec") {
		t.Errorf("original content should be preserved")
	}
}

func TestInjectFrontmatterField_UpdateExisting(t *testing.T) {
	input := "---\ntitle: My Spec\ntracker_id: old-id\n---\n# My Spec\n"
	result := spec.SetFrontmatterField(input, "tracker_id", "new-id")

	if !strings.Contains(result, "tracker_id: new-id") {
		t.Errorf("expected tracker_id: new-id, got:\n%s", result)
	}
	if strings.Contains(result, "old-id") {
		t.Errorf("old value should be replaced")
	}
}

func TestInjectFrontmatterField_NoFrontmatter(t *testing.T) {
	input := "# My Spec\n\nSome content.\n"
	result := spec.SetFrontmatterField(input, "tracker_id", "42")

	if !strings.HasPrefix(result, "---\n") {
		t.Error("expected frontmatter to be created at start")
	}
	if !strings.Contains(result, "tracker_id: 42") {
		t.Errorf("expected tracker_id: 42, got:\n%s", result)
	}
	if !strings.Contains(result, "# My Spec") {
		t.Error("original content should be preserved")
	}
}

func TestInjectFrontmatterField_EmptyFile(t *testing.T) {
	input := ""
	result := spec.SetFrontmatterField(input, "tracker_id", "42")

	if !strings.Contains(result, "tracker_id: 42") {
		t.Errorf("expected tracker_id: 42, got:\n%s", result)
	}
}

// --- writeTrackerID integration test ---

func TestWriteTrackerID(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.md")

	content := "---\ntitle: Test\ntype: feature\n---\n# Test\n"
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeTrackerID(specPath, "PROJ-42"); err != nil {
		t.Fatalf("writeTrackerID failed: %v", err)
	}

	data, _ := os.ReadFile(specPath)
	if !strings.Contains(string(data), "tracker_id: PROJ-42") {
		t.Errorf("expected tracker_id in file, got:\n%s", string(data))
	}
}

// --- splitLines / joinLines ---

func TestSplitJoinRoundTrip(t *testing.T) {
	input := "line1\nline2\nline3"
	lines := strings.Split(input, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	result := strings.Join(lines, "\n")
	if result != input {
		t.Errorf("round-trip failed: %q != %q", result, input)
	}
}

// --- Helpers ---

// writeTrackerConfig overwrites the hero.json with a tracker config.
func writeTrackerConfig(env *testEnv, trackerType, project string) {
	env.t.Helper()

	configJSON := `{
  "folder": ".hero",
  "tracker": {
    "type": "` + trackerType + `",
    "project": "` + project + `",
    "token_env": "HERO_TEST_TOKEN"
  }
}`
	configPath := filepath.Join(env.heroDir, "hero.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		env.t.Fatalf("WriteFile: %v", err)
	}
}

// writeTrackerConfigWithBaseURL overwrites hero.json with a tracker
// config that targets a custom base URL (used to point at httptest
// servers). The shape mirrors writeTrackerConfig.
func writeTrackerConfigWithBaseURL(env *testEnv, trackerType, project, baseURL string) {
	env.t.Helper()

	configJSON := `{
  "folder": ".hero",
  "tracker": {
    "type": "` + trackerType + `",
    "project": "` + project + `",
    "base_url": "` + baseURL + `",
    "token_env": "HERO_TEST_TOKEN"
  }
}`
	configPath := filepath.Join(env.heroDir, "hero.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		env.t.Fatalf("WriteFile: %v", err)
	}
}

// --- Size push: runSync → UpdateSize wiring ---

// TestRunSync_CleanPush_CallsUpdateSize verifies the SizeSyncPushToTracker
// branch in runSync invokes the adapter's UpdateSize. Drives a real
// runSync end-to-end against a stubbed GitHub server: GET returns an
// issue with no size label, PATCH replaces labels including the new
// size/large. Asserts (a) UpdateSize ran (PATCH captured), (b) the
// merged label set is correct, (c) UpdateStatus also ran (sync
// completes), and (d) success line printed on stdout.
func TestRunSync_CleanPush_CallsUpdateSize(t *testing.T) {
	var sawPatch bool
	var patchLabels []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/issues/42"):
			// GetIssue + the size-update GET both hit this; return
			// labels with no existing size label so PlanSizePush ⇒
			// SizeSyncPushToTracker.
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"number":   42,
				"title":    "test",
				"state":    "open",
				"html_url": "x",
				"labels": []map[string]string{
					{"name": "bug"},
					{"name": "hero:active"},
				},
			})
		case r.Method == "PATCH" && strings.HasSuffix(r.URL.Path, "/issues/42"):
			sawPatch = true
			var payload map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if raw, ok := payload["labels"].([]interface{}); ok {
				for _, l := range raw {
					if s, ok := l.(string); ok {
						patchLabels = append(patchLabels, s)
					}
				}
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{}`)
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/comments"):
			// UpdateStatus path posts a comment.
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{}`)
		default:
			t.Logf("unhandled: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer srv.Close()

	env := newTestEnv(t)
	writeTrackerConfigWithBaseURL(env, "github", "acme/widgets", srv.URL)
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
tracker_id: 42
size: large
---
# CSV Export

## Goal

Export data.
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	out, err := runCmd("sync", "spec", specPath)
	if err != nil {
		t.Fatalf("runCmd: %v\noutput: %s", err, out)
	}

	if !sawPatch {
		t.Fatal("expected PATCH to be captured (UpdateSize call)")
	}
	want := []string{"bug", "hero:active", "size/large"}
	if !sameLabelSetSync(patchLabels, want) {
		t.Errorf("PATCH labels = %v, want set %v", patchLabels, want)
	}
	if !strings.Contains(out, "Updated github size for issue 42 → large") {
		t.Errorf("stdout missing UpdateSize success line; got:\n%s", out)
	}
}

// TestRunSync_Conflict_DoesNotCallUpdateSize verifies the
// non-destructive contract: on SizeSyncConflict the warn-only branch
// runs and UpdateSize is NOT invoked. Drives a stub where the tracker
// has size/small but spec declares large.
func TestRunSync_Conflict_DoesNotCallUpdateSize(t *testing.T) {
	var sawPatch bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/issues/42"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"number":   42,
				"title":    "test",
				"state":    "open",
				"html_url": "x",
				"labels": []map[string]string{
					{"name": "bug"},
					{"name": "size/small"}, // conflicts with spec's "large"
				},
			})
		case r.Method == "PATCH":
			sawPatch = true
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{}`)
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{}`)
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer srv.Close()

	env := newTestEnv(t)
	writeTrackerConfigWithBaseURL(env, "github", "acme/widgets", srv.URL)
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
tracker_id: 42
size: large
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	if _, err := runCmd("sync", "spec", specPath); err != nil {
		t.Fatalf("runCmd: %v", err)
	}

	if sawPatch {
		t.Error("expected NO PATCH on conflict — non-destructive contract violated")
	}
}

// sameLabelSetSync is a local copy to avoid cross-package coupling
// with internal/tracker test helpers.
func sameLabelSetSync(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	gotMap := map[string]bool{}
	for _, l := range got {
		gotMap[l] = true
	}
	for _, w := range want {
		if !gotMap[w] {
			return false
		}
	}
	return true
}
