package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

// orgRepoServer stands in for the cloud's orgs/repos endpoints. It counts
// how many times the orgs list endpoint is hit so tests can assert the
// config short-circuit avoids re-listing.
type orgRepoServer struct {
	orgsListCount int
}

func newOrgRepoServer(t *testing.T) (*httptest.Server, *orgRepoServer) {
	t.Helper()
	state := &orgRepoServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/orgs":
			state.orgsListCount++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"orgs": []map[string]string{{"id": "org_1", "name": "Acme", "slug": "acme"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/orgs/org_1/repos":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"repos": []map[string]string{{"id": "repo_1", "name": "hero-cloud"}},
			})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv, state
}

func TestResolveCloudTargetWriteBack(t *testing.T) {
	env := newTestEnv(t)
	srv, state := newOrgRepoServer(t)
	defer srv.Close()

	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	orgID, repoID, err := resolveCloudTarget(cfg, "tok", srv.URL, env.dir, "")
	if err != nil {
		t.Fatalf("resolveCloudTarget: %v", err)
	}
	// hero-cloud repo matches by name (project dir name is the temp dir,
	// so it won't match — single repo fallback applies). Either way ids
	// must resolve and persist.
	if orgID != "org_1" || repoID != "repo_1" {
		t.Fatalf("resolved (%q,%q), want (org_1,repo_1)", orgID, repoID)
	}

	reloaded, _ := config.Load(env.dir)
	if reloaded.Cloud == nil || reloaded.Cloud.OrgID != "org_1" || reloaded.Cloud.RepoID != "repo_1" {
		t.Fatalf("config not persisted: %+v", reloaded.Cloud)
	}
	if state.orgsListCount != 1 {
		t.Fatalf("orgs listed %d times on first resolve, want 1", state.orgsListCount)
	}
}

func TestResolveCloudTargetNoRelistWhenConfigPresent(t *testing.T) {
	env := newTestEnv(t)
	srv, state := newOrgRepoServer(t)
	defer srv.Close()

	cfg, _ := config.Load(env.dir)
	cfg.Cloud = &config.CloudConfig{OrgID: "org_pre", RepoID: "repo_pre"}

	orgID, repoID, err := resolveCloudTarget(cfg, "tok", srv.URL, env.dir, "")
	if err != nil {
		t.Fatalf("resolveCloudTarget: %v", err)
	}
	if orgID != "org_pre" || repoID != "repo_pre" {
		t.Fatalf("resolved (%q,%q), want config values", orgID, repoID)
	}
	if state.orgsListCount != 0 {
		t.Fatalf("orgs listed %d times despite config present, want 0", state.orgsListCount)
	}
}

func TestPersistCloudTargetIdempotent(t *testing.T) {
	env := newTestEnv(t)

	cfg, _ := config.Load(env.dir)
	cfg.Cloud = &config.CloudConfig{OrgID: "org_1", RepoID: "repo_1"}
	// Persist once to establish the file.
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	heroJSON := env.heroDir + "/hero.json"
	info1, err := os.Stat(heroJSON)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	// Unchanged values must not rewrite the file.
	persistCloudTarget(cfg, env.dir, "org_1", "repo_1")
	info2, err := os.Stat(heroJSON)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info1.ModTime() != info2.ModTime() {
		t.Errorf("hero.json was rewritten despite unchanged ids")
	}
}
