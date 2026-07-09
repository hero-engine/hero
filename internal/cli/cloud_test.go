package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
)

func TestOrgSlugify(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Acme Inc", "acme-inc"},
		{"underscores", "acme_widgets_co", "acme-widgets-co"},
		{"strip invalid", "Açmé! Inc.™", "am-inc"},
		{"collapse repeats", "a   b___c", "a-b-c"},
		{"trim edges", "  -Acme-  ", "acme"},
		{"already slug", "acme-inc", "acme-inc"},
		{"clamp 40", strings.Repeat("a", 50), strings.Repeat("a", 40)},
		{"clamp then trim trailing hyphen", strings.Repeat("a", 39) + " bbb", strings.Repeat("a", 39)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := orgSlugify(c.in); got != c.want {
				t.Errorf("orgSlugify(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestValidSlug(t *testing.T) {
	valid := []string{"abc", "acme-inc", "a1b", strings.Repeat("a", 40), "org-123"}
	invalid := []string{"ab", "", "-abc", "abc-", "a", "AB-CD", "a b c", strings.Repeat("a", 41), "a--", "--a"}
	for _, s := range valid {
		if !validSlug(s) {
			t.Errorf("validSlug(%q) = false, want true", s)
		}
	}
	for _, s := range invalid {
		if validSlug(s) {
			t.Errorf("validSlug(%q) = true, want false", s)
		}
	}
}

func TestSelectOrg(t *testing.T) {
	orgs := []cloudOrg{
		{ID: "org_1", Name: "Acme Inc", Slug: "acme"},
		{ID: "org_2", Name: "Globex", Slug: "globex"},
	}

	t.Run("match by id", func(t *testing.T) {
		o, err := selectOrg(orgs, "org_2")
		if err != nil || o.ID != "org_2" {
			t.Fatalf("got (%v, %v), want org_2", o, err)
		}
	})

	t.Run("match by name case-insensitive", func(t *testing.T) {
		o, err := selectOrg(orgs, "acme inc")
		if err != nil || o.ID != "org_1" {
			t.Fatalf("got (%v, %v), want org_1", o, err)
		}
	})

	t.Run("match by slug", func(t *testing.T) {
		o, err := selectOrg(orgs, "GLOBEX")
		if err != nil || o.ID != "org_2" {
			t.Fatalf("got (%v, %v), want org_2", o, err)
		}
	})

	t.Run("no match errors", func(t *testing.T) {
		_, err := selectOrg(orgs, "nope")
		if err == nil || !strings.Contains(err.Error(), "no org matching") {
			t.Fatalf("expected no-match error, got %v", err)
		}
	})

	t.Run("multi-org no selector errors with --org hint", func(t *testing.T) {
		_, err := selectOrg(orgs, "")
		if err == nil || !strings.Contains(err.Error(), "--org") {
			t.Fatalf("expected --org hint error, got %v", err)
		}
	})

	t.Run("single org no selector takes it", func(t *testing.T) {
		o, err := selectOrg(orgs[:1], "")
		if err != nil || o.ID != "org_1" {
			t.Fatalf("got (%v, %v), want org_1", o, err)
		}
	})

	t.Run("empty list errors with create-org hint", func(t *testing.T) {
		_, err := selectOrg(nil, "")
		if err == nil || !strings.Contains(err.Error(), "create-org") {
			t.Fatalf("expected create-org hint, got %v", err)
		}
	})
}

// writeCreds writes a credentials file under a temp HOME so LoadCloudToken
// returns a non-empty token without an expiry. Returns the cloud URL set.
func writeCreds(t *testing.T, cloudURL string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	credDir := filepath.Join(home, ".hero")
	if err := os.MkdirAll(credDir, 0o700); err != nil {
		t.Fatalf("mkdir creds: %v", err)
	}
	creds := cloudCredentials{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
		CloudURL:     cloudURL,
		// no ExpiresAt -> treated as non-expiring by LoadCloudToken
	}
	data, _ := json.MarshalIndent(creds, "", "  ")
	if err := os.WriteFile(filepath.Join(credDir, "credentials.json"), data, 0o600); err != nil {
		t.Fatalf("write creds: %v", err)
	}
}

func TestRunCreateOrgHappyPathPersists(t *testing.T) {
	env := newTestEnv(t)

	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/orgs" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "org_abc", "name": gotBody["name"], "slug": gotBody["slug"]})
	}))
	defer srv.Close()

	t.Setenv("HERO_CLOUD_URL", srv.URL)
	writeCreds(t, srv.URL)
	createOrgSlug = ""

	if err := runCreateOrg(nil, []string{"Acme Inc"}); err != nil {
		t.Fatalf("runCreateOrg: %v", err)
	}

	if gotBody["name"] != "Acme Inc" || gotBody["slug"] != "acme-inc" {
		t.Errorf("posted body = %v, want name=Acme Inc slug=acme-inc", gotBody)
	}

	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if cfg.Cloud == nil || cfg.Cloud.OrgID != "org_abc" {
		t.Errorf("cloud config = %+v, want OrgID=org_abc", cfg.Cloud)
	}
}

func TestRunCreateOrgSlugOverride(t *testing.T) {
	newTestEnv(t)

	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "org_x"})
	}))
	defer srv.Close()

	t.Setenv("HERO_CLOUD_URL", srv.URL)
	writeCreds(t, srv.URL)
	createOrgSlug = "custom-slug"
	t.Cleanup(func() { createOrgSlug = "" })

	if err := runCreateOrg(nil, []string{"Acme Inc"}); err != nil {
		t.Fatalf("runCreateOrg: %v", err)
	}
	if gotBody["slug"] != "custom-slug" {
		t.Errorf("slug = %q, want custom-slug", gotBody["slug"])
	}
}

func TestRunCreateOrgStatusMapping(t *testing.T) {
	cases := []struct {
		status   int
		wantSub  string
	}{
		{http.StatusConflict, "already taken"},
		{http.StatusBadRequest, "3-40 lowercase"},
	}
	for _, c := range cases {
		t.Run(http.StatusText(c.status), func(t *testing.T) {
			newTestEnv(t)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
			}))
			defer srv.Close()

			t.Setenv("HERO_CLOUD_URL", srv.URL)
			writeCreds(t, srv.URL)
			createOrgSlug = ""

			err := runCreateOrg(nil, []string{"Acme Inc"})
			if err == nil || !strings.Contains(err.Error(), c.wantSub) {
				t.Fatalf("err = %v, want substring %q", err, c.wantSub)
			}
		})
	}
}
