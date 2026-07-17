package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

// TestNonInteractiveConnect_GitlabUserEmail_RejectedAndWritesNothing mirrors the
// Defect 1 repro: a gitlab connect with --user-email must fail at connect time
// with a flag-named error, write neither hero.json nor hero.local.json, and
// leave the workspace loadable (not bricked).
func TestNonInteractiveConnect_GitlabUserEmail_RejectedAndWritesNothing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Simulate an initialized workspace with a valid committed config.
	if err := config.DefaultConfig().Save(root); err != nil {
		t.Fatal(err)
	}
	committedPath := filepath.Join(root, ".hero", config.ConfigFileName)
	localPath := filepath.Join(root, ".hero", config.LocalConfigFileName)
	before, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatal(err)
	}

	connectIntegrationID = "gitlab-chronecho"
	connectProject = "noeta-studios/chronecho"
	connectBaseURL = "https://gitlab.com"
	connectUserEmail = "sean@example.com"
	connectRole = "delivery"
	connectTokenStdin = true
	connectLocalOnly = false
	connectGlobal = false
	connectJSON = false
	connectNoVerify = true

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("token-canary\n"))
	cmd.SetOut(&bytes.Buffer{})

	rerr := runConnectNonInteractive(cmd, root, config.Credentials{}, "gitlab")
	if rerr == nil {
		t.Fatal("expected connect to be rejected for gitlab + --user-email")
	}
	if !strings.Contains(rerr.Error(), "--user-email is not valid for provider gitlab") {
		t.Fatalf("error = %q, want flag-named rejection", rerr.Error())
	}

	// hero.json unchanged, hero.local.json never created.
	after, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("hero.json was modified:\nbefore=%s\nafter=%s", before, after)
	}
	if fileExists(localPath) {
		t.Fatal("hero.local.json was written despite the rejection")
	}

	// Workspace is not bricked.
	if _, err := config.Load(root); err != nil {
		t.Fatalf("workspace no longer loads after rejected connect: %v", err)
	}
}

func TestLink_AlreadyLinked_MentionsForce(t *testing.T) {
	env := newTestEnv(t)
	writeTrackerConfig(env, "github", "acme/widgets")
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
slug: csv-export
tracker_id: ECHO-176
---
# CSV Export
`)

	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")
	_, err := runCmd("sync", "link", specPath, "15")
	if err == nil {
		t.Fatal("expected error for already-linked spec without --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error = %q, want mention of --force", err.Error())
	}
}

// TestLink_Force_RepointsAndPrintsTransition drives --force end to end against a
// stubbed tracker whose GetIssue succeeds: the tracker_id is overwritten and the
// old→new transition is printed.
func TestLink_Force_RepointsAndPrintsTransition(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/issues/15") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"number": 15, "title": "Migrated issue", "state": "open", "html_url": "x",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	env := newTestEnv(t)
	writeTrackerConfigWithBaseURL(env, "github", "acme/widgets", srv.URL)
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
slug: csv-export
tracker_id: ECHO-176
---
# CSV Export
`)
	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")

	out, err := runCmd("sync", "link", specPath, "15", "--force")
	if err != nil {
		t.Fatalf("runCmd: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Re-pointed spec csv-export: ECHO-176 → 15") {
		t.Errorf("stdout missing transition line; got:\n%s", out)
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "tracker_id: 15") {
		t.Errorf("tracker_id not overwritten on disk:\n%s", data)
	}
	if strings.Contains(string(data), "ECHO-176") {
		t.Errorf("old tracker_id still present on disk:\n%s", data)
	}
}

// TestLink_Force_NonexistentIssueStillErrors verifies --force keeps the GetIssue
// verification: pointing a spec at an issue the tracker rejects still fails.
func TestLink_Force_NonexistentIssueStillErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"Not Found"}`)
	}))
	defer srv.Close()

	env := newTestEnv(t)
	writeTrackerConfigWithBaseURL(env, "github", "acme/widgets", srv.URL)
	t.Setenv("HERO_TEST_TOKEN", "fake-token")

	env.addSpec("planning/features/csv-export/spec.md", `---
title: CSV Export
type: feature
status: planning
slug: csv-export
tracker_id: ECHO-176
---
# CSV Export
`)
	specPath := filepath.Join(env.heroDir, "planning/features/csv-export/spec.md")

	_, err := runCmd("sync", "link", specPath, "9999", "--force")
	if err == nil {
		t.Fatal("expected error for a non-existent issue even under --force")
	}
	if !strings.Contains(err.Error(), "verifying issue") {
		t.Errorf("error = %q, want verification failure", err.Error())
	}

	// tracker_id must not have been overwritten.
	data, _ := os.ReadFile(specPath)
	if !strings.Contains(string(data), "tracker_id: ECHO-176") {
		t.Errorf("tracker_id was changed despite verification failure:\n%s", data)
	}
}

// TestLink_AcceptsDirAndSlug covers Defect 3: sync link resolves a spec
// directory and a bare slug, writing tracker_id to the on-disk spec.md.
func TestLink_AcceptsDirAndSlug(t *testing.T) {
	newIssueServer := func(t *testing.T, id string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/issues/"+id) {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"number": 7, "title": "Issue", "state": "open", "html_url": "x",
				})
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{}`)
		}))
	}

	specBody := func(slug string) string {
		return "---\ntitle: " + slug + "\ntype: feature\nstatus: planning\nslug: " + slug + "\n---\n# body\n"
	}

	t.Run("directory", func(t *testing.T) {
		srv := newIssueServer(t, "7")
		defer srv.Close()
		env := newTestEnv(t)
		writeTrackerConfigWithBaseURL(env, "github", "acme/widgets", srv.URL)
		t.Setenv("HERO_TEST_TOKEN", "fake-token")
		env.addSpec("planning/features/by-dir/spec.md", specBody("by-dir"))

		dir := filepath.Join(env.heroDir, "planning/features/by-dir")
		out, err := runCmd("sync", "link", dir, "7")
		if err != nil {
			t.Fatalf("runCmd: %v\noutput: %s", err, out)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "spec.md"))
		if !strings.Contains(string(data), "tracker_id: 7") {
			t.Errorf("tracker_id not written to <dir>/spec.md:\n%s", data)
		}
	})

	t.Run("slug", func(t *testing.T) {
		srv := newIssueServer(t, "7")
		defer srv.Close()
		env := newTestEnv(t)
		writeTrackerConfigWithBaseURL(env, "github", "acme/widgets", srv.URL)
		t.Setenv("HERO_TEST_TOKEN", "fake-token")
		env.addSpec("planning/features/by-slug/spec.md", specBody("by-slug"))

		out, err := runCmd("sync", "link", "by-slug", "7")
		if err != nil {
			t.Fatalf("runCmd: %v\noutput: %s", err, out)
		}
		data, _ := os.ReadFile(filepath.Join(env.heroDir, "planning/features/by-slug/spec.md"))
		if !strings.Contains(string(data), "tracker_id: 7") {
			t.Errorf("tracker_id not written to on-disk spec.md:\n%s", data)
		}
	})
}

// TestResolveSpec_DirAndSlug covers the shared resolver directly.
func TestResolveSpec_DirAndSlug(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/resolver-target/spec.md",
		"---\ntitle: Resolver Target\ntype: feature\nstatus: planning\nslug: resolver-target\n---\n# body\n")

	dir := filepath.Join(env.heroDir, "planning/features/resolver-target")

	t.Run("directory", func(t *testing.T) {
		s, err := resolveSpec(dir, env.heroDir)
		if err != nil {
			t.Fatalf("resolveSpec(dir): %v", err)
		}
		if s.Path != filepath.Join(dir, "spec.md") {
			t.Errorf("resolved path = %q, want <dir>/spec.md", s.Path)
		}
	})

	t.Run("slug", func(t *testing.T) {
		s, err := resolveSpec("resolver-target", env.heroDir)
		if err != nil {
			t.Fatalf("resolveSpec(slug): %v", err)
		}
		if s.Slug != "resolver-target" {
			t.Errorf("resolved slug = %q, want resolver-target", s.Slug)
		}
	})

	t.Run("empty-directory", func(t *testing.T) {
		empty := filepath.Join(env.heroDir, "planning/features/empty-dir")
		if err := os.MkdirAll(empty, 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := resolveSpec(empty, env.heroDir)
		if err == nil {
			t.Fatal("expected error for a directory with no spec.md or requirements.md")
		}
		if !strings.Contains(err.Error(), "no spec.md or requirements.md") {
			t.Errorf("error = %q, want clean 'no spec.md or requirements.md'", err.Error())
		}
	})
}
