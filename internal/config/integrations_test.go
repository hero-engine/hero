package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationOverlayAndProvenance(t *testing.T) {
	c := []byte(`{"integrations":{"default":"jira-delivery","roles":{"delivery":"jira-delivery"},"connections":{"jira-delivery":{"provider":"github","settings":{"project":"owner/repo","base_url":"https://github.example"}}}}}`)
	l := []byte(`{"integrations":{"connections":{"jira-delivery":{"auth":{"token":"canary-do-not-print"},"settings":{"base_url":""}}}}}`)
	r, err := ResolveIntegrationDocuments("hero.json", c, "hero.local.json", l)
	if err != nil {
		t.Fatal(err)
	}
	_, x, err := r.Select("", "delivery")
	if err != nil {
		t.Fatal(err)
	}
	if got := rawString(x.Settings, "base_url"); got != "" {
		t.Fatalf("empty override lost: %q", got)
	}
	if x.Auth.Token.Reveal() != "canary-do-not-print" {
		t.Fatal("local auth not overlaid")
	}
	if s := r.Provenance["$.integrations.connections.jira-delivery.auth.token"]; s.File != "hero.local.json" {
		t.Fatalf("provenance=%+v", s)
	}
	if got := fmt.Sprintf("%v/%#v", x.Auth.Token, x.Auth.Token); strings.Contains(got, "canary") {
		t.Fatal("generic formatting leaked secret")
	}
}

func TestMergePatchPreservesFalseZeroAndEmpty(t *testing.T) {
	base := map[string]any{"enabled": true, "limit": 5.0, "label": "x"}
	got := mergePatch(base, map[string]any{"enabled": false, "limit": 0.0, "label": ""}).(map[string]any)
	if got["enabled"] != false || got["limit"] != 0.0 || got["label"] != "" {
		t.Fatalf("scalar presence lost: %#v", got)
	}
}

func TestIntegrationNullDeletesAndDanglingSelector(t *testing.T) {
	c := []byte(`{"integrations":{"default":"a","connections":{"a":{"provider":"github","settings":{"project":"o/r"}}}}}`)
	l := []byte(`{"integrations":{"connections":{"a":null}}}`)
	_, err := ResolveIntegrationDocuments("hero.json", c, "hero.local.json", l)
	if err == nil || !strings.Contains(err.Error(), "$.integrations.default") || !strings.Contains(err.Error(), "$.integrations.connections.a") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommittedSecretRejectedWithoutEcho(t *testing.T) {
	b := []byte(`{"integrations":{"connections":{"x":{"provider":"github","settings":{"project":"o/r"},"auth":{"token":"CANARY-SECRET"}}}}}`)
	_, err := ResolveIntegrationDocuments("/p/.hero/hero.json", b, "local", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "CANARY") {
		t.Fatal("error leaked token")
	}
}

func TestUnknownLegacyProviderObjectHasMigrationGuidance(t *testing.T) {
	_, err := ResolveIntegrationDocuments("hero.local.json", []byte(`{"tracker":{"jira":{"token":"x"}}}`), "local", nil)
	if err == nil || !strings.Contains(err.Error(), "$.tracker.jira") || !strings.Contains(err.Error(), "hero connect jira") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestCanonicalLegacyConflictNamesBothPaths(t *testing.T) {
	b := []byte(`{"tracker":{"type":"jira","project":"P"},"integrations":{"default":"x","connections":{"x":{"provider":"jira","settings":{"project":"P"}}}}}`)
	_, err := ResolveIntegrationDocuments("hero.json", b, "hero.local.json", nil)
	if err == nil || !strings.Contains(err.Error(), "$.integrations") || !strings.Contains(err.Error(), "$.tracker") {
		t.Fatalf("unexpected %v", err)
	}
}

func TestStableIDsAndExplicitSelection(t *testing.T) {
	b := []byte(`{"integrations":{"default":"one","connections":{"one":{"provider":"jira","settings":{"project":"A","base_url":"https://a"}},"two":{"provider":"jira","settings":{"project":"B","base_url":"https://b"}}}}}`)
	r, err := ResolveIntegrationDocuments("hero.json", b, "local", nil)
	if err != nil {
		t.Fatal(err)
	}
	id, x, err := r.Select("two", "delivery")
	if err != nil || id != "two" || rawString(x.Settings, "project") != "B" {
		t.Fatalf("%s %+v %v", id, x, err)
	}
}

func TestJiraBoardSettingFlowsThroughTrackerResolution(t *testing.T) {
	b := []byte(`{"integrations":{"default":"jira","roles":{"delivery":"jira"},"connections":{"jira":{"provider":"jira","settings":{"project":"MORPH","board":"MORPH board","base_url":"https://jira"}}}}}`)
	r, err := ResolveIntegrationDocuments("hero.json", b, "local", nil)
	if err != nil {
		t.Fatal(err)
	}
	tracker, ok := r.DeliveryTracker()
	if !ok || tracker.Board != "MORPH board" {
		t.Fatalf("delivery tracker = %+v, ok = %v", tracker, ok)
	}
	connection, err := (Config{Integrations: r.Config}).ResolveTrackerConnection("jira")
	if err != nil {
		t.Fatal(err)
	}
	if connection.Board != "MORPH board" || connection.TrackerConfig().Board != "MORPH board" {
		t.Fatalf("connection = %+v, tracker = %+v", connection, connection.TrackerConfig())
	}
}

func TestPatchLocalIntegrationsPreservesKeysAndMode(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, LocalConfigFileName), []byte(`{"personal":"keep","integrations":{"connections":{"old":{"provider":"github","settings":{"project":"o/r"}}}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	patch := json.RawMessage(`{"connections":{"new":{"provider":"jira","settings":{"project":"P","base_url":"https://jira"},"auth":{"token":"LOCAL-CANARY"}}}}`)
	if err := PatchLocalIntegrations(root, ".hero", patch); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, LocalConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"personal": "keep"`) || !strings.Contains(string(b), `"old"`) || !strings.Contains(string(b), `"LOCAL-CANARY"`) {
		t.Fatalf("content not preserved: %s", b)
	}
	st, _ := os.Stat(filepath.Join(dir, LocalConfigFileName))
	if st.Mode().Perm() != 0600 {
		t.Fatalf("mode=%o", st.Mode().Perm())
	}
}

func TestLoadMutateSaveNeverCommitsEffectiveTrackerOrConfluenceSecrets(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	dir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	shared := `{"folder":".hero","integrations":{"default":"jira-delivery","roles":{"delivery":"jira-delivery","docs":"conf-docs"},"connections":{"jira-delivery":{"provider":"jira","settings":{"project":"P","base_url":"https://jira"}},"conf-docs":{"provider":"confluence","settings":{"space_key":"DOCS","base_url":"https://conf"}}}}}`
	local := `{"integrations":{"connections":{"jira-delivery":{"auth":{"token":"TRACKER-SAVE-CANARY"}},"conf-docs":{"auth":{"token":"CONFLUENCE-SAVE-CANARY"}}}}}`
	os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(shared), 0644)
	os.WriteFile(filepath.Join(dir, LocalConfigFileName), []byte(local), 0600)
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tracker.Token == "" || cfg.Confluence.Token == "" {
		t.Fatal("runtime secrets were not materialized")
	}
	cfg.Domain = "changed"
	if err := cfg.Save(root); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(filepath.Join(dir, ConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"TRACKER-SAVE-CANARY", "CONFLUENCE-SAVE-CANARY", "[REDACTED]", `"token"`} {
		if strings.Contains(string(saved), bad) {
			t.Fatalf("committed save leaked %q: %s", bad, saved)
		}
	}
	if !strings.Contains(string(saved), `"domain": "changed"`) {
		t.Fatal("mutation was not saved")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(saved, &top); err != nil {
		t.Fatal(err)
	}
	if _, ok := top["tracker"]; ok {
		t.Fatal("derived legacy tracker was committed")
	}
	if _, ok := top["confluence"]; ok {
		t.Fatal("derived legacy confluence was committed")
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("saved canonical config no longer reloads: %v", err)
	}
}

func TestProviderSettingsValidationIsSpecificAndTypeStrict(t *testing.T) {
	cases := []struct{ name, provider, settings, want string }{
		{"github-inapplicable", "github", `{"project":"o/r","user_email":"x@y"}`, "settings.user_email"},
		{"github-inapplicable-board", "github", `{"project":"o/r","board":"Roadmap"}`, "settings.board"},
		{"github-wrong-project", "github", `{"project":7}`, "settings.project: expected string"},
		{"linear-null-project", "linear", `{"project":null}`, "settings.project: expected string"},
		{"jira-empty-project", "jira", `{"project":"","base_url":"https://jira"}`, "settings.project must not be empty"},
		{"jira-empty-board", "jira", `{"project":"P","board":"","base_url":"https://jira"}`, "settings.board must not be empty"},
		{"jira-wrong-bool", "jira", `{"project":"P","base_url":"https://jira","post_on_design":"yes"}`, "settings.post_on_design: expected boolean"},
		{"gitlab-missing-base", "gitlab", `{"project":"g/p"}`, "settings.base_url is required"},
		{"gitlab-inapplicable-email", "gitlab", `{"project":"g/p","base_url":"https://gitlab","user_email":"x@y"}`, "settings.user_email"},
		{"confluence-null-space", "confluence", `{"space_key":null,"base_url":"https://conf"}`, "settings.space_key: expected string"},
		{"confluence-wrong-base", "confluence", `{"space_key":"D","base_url":false}`, "settings.base_url: expected string"},
		{"confluence-inapplicable-project", "confluence", `{"space_key":"D","base_url":"https://conf","project":"P"}`, "settings.project"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := fmt.Sprintf(`{"integrations":{"default":"x","connections":{"x":{"provider":%q,"settings":%s}}}}`, tc.provider, tc.settings)
			_, err := ResolveIntegrationDocuments("hero.json", []byte(doc), "hero.local.json", nil)
			if err == nil || !strings.Contains(err.Error(), "$.integrations.connections.x."+tc.want) {
				t.Fatalf("error=%v, want path fragment %q", err, tc.want)
			}
		})
	}
}

// TestValidateConnectionSettings exercises the write-time adapter that connect
// runs before persisting an assembled settings map. It must agree with the
// read-time validator: a flag/provider mismatch is rejected, a valid map is ok.
func TestValidateConnectionSettings(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		settings map[string]any
		wantErr  string // "" means expect nil
	}{
		{"gitlab-user-email", "gitlab", map[string]any{"project": "g/p", "base_url": "https://gitlab", "user_email": "x@y"}, "settings.user_email"},
		{"gitlab-ok", "gitlab", map[string]any{"project": "g/p", "base_url": "https://gitlab"}, ""},
		{"jira-user-email-ok", "jira", map[string]any{"project": "P", "base_url": "https://jira", "user_email": "x@y"}, ""},
		{"jira-board-ok", "jira", map[string]any{"project": "P", "board": "P board", "base_url": "https://jira"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConnectionSettings("x", tc.provider, tc.settings)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "$.integrations.connections.x."+tc.wantErr) {
				t.Fatalf("error=%v, want path fragment %q", err, tc.wantErr)
			}
		})
	}
}

// TestPatchBackstopRejectsInvalidProviderSettings verifies the 1d backstop: the
// persistence functions refuse a provider-bearing connection whose settings
// fail the schema (gitlab + user_email), so no connect path can write a config
// the loader rejects — for both the committed and local files.
func TestPatchBackstopRejectsInvalidProviderSettings(t *testing.T) {
	invalid := json.RawMessage(`{"connections":{"gl":{"provider":"gitlab","settings":{"project":"g/p","base_url":"https://gitlab","user_email":"x@y"}}}}`)

	t.Run("committed", func(t *testing.T) {
		root := t.TempDir()
		err := PatchCommittedIntegrations(root, ".hero", invalid)
		if err == nil || !strings.Contains(err.Error(), "settings.user_email") {
			t.Fatalf("error=%v, want settings.user_email rejection", err)
		}
		if fileExistsTest(filepath.Join(root, ".hero", ConfigFileName)) {
			t.Fatal("committed config was written despite invalid settings")
		}
	})

	t.Run("local", func(t *testing.T) {
		root := t.TempDir()
		err := PatchLocalIntegrations(root, ".hero", invalid)
		if err == nil || !strings.Contains(err.Error(), "settings.user_email") {
			t.Fatalf("error=%v, want settings.user_email rejection", err)
		}
	})
}

// TestPatchBackstopAcceptsTokenOnlyOverlay is the false-positive regression
// guard: a credential overlay that carries only auth.token (no provider) must
// still be accepted — the backstop only validates provider-bearing connections.
func TestPatchBackstopAcceptsTokenOnlyOverlay(t *testing.T) {
	root := t.TempDir()
	overlay := json.RawMessage(`{"connections":{"gl":{"auth":{"token":"LOCAL-CANARY"}}}}`)
	if err := PatchLocalIntegrations(root, ".hero", overlay); err != nil {
		t.Fatalf("token-only overlay rejected: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".hero", LocalConfigFileName))
	if err != nil {
		t.Fatalf("local file not written: %v", err)
	}
	if !strings.Contains(string(b), "LOCAL-CANARY") {
		t.Fatalf("token not persisted: %s", b)
	}
}

func fileExistsTest(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func TestResolveTrackerConnectionRequiresUnambiguousSelection(t *testing.T) {
	setting := func(value string) json.RawMessage {
		b, _ := json.Marshal(value)
		return b
	}
	cfg := Config{Integrations: &IntegrationsConfig{
		Default: "jira-a",
		Roles:   map[string]string{"delivery": "jira-a"},
		Connections: map[string]IntegrationConfig{
			"jira-a":   {Provider: "jira", Settings: map[string]json.RawMessage{"project": setting("A"), "base_url": setting("https://a.example")}},
			"github-b": {Provider: "github", Settings: map[string]json.RawMessage{"project": setting("o/r")}},
			"docs":     {Provider: "confluence", Settings: map[string]json.RawMessage{"space_key": setting("D"), "base_url": setting("https://docs.example")}},
		},
	}}
	if _, err := cfg.ResolveTrackerConnection(""); err == nil || !strings.Contains(err.Error(), "connection_id is required") {
		t.Fatalf("expected ambiguity despite default/role, got %v", err)
	}
	got, err := cfg.ResolveTrackerConnection("github-b")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "github-b" || got.Provider != "github" || got.Project != "o/r" {
		t.Fatalf("connection = %+v", got)
	}
}

func TestResolveTrackerConnectionInfersOnlyTracker(t *testing.T) {
	project, _ := json.Marshal("A")
	base, _ := json.Marshal("https://a.example")
	cfg := Config{Integrations: &IntegrationsConfig{Connections: map[string]IntegrationConfig{
		"jira-a": {Provider: "jira", Settings: map[string]json.RawMessage{"project": project, "base_url": base}, Auth: &IntegrationAuth{Token: Secret("canary")}},
	}}}
	got, err := cfg.ResolveTrackerConnection("")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "jira-a" || got.Token.Reveal() != "canary" {
		t.Fatalf("connection = %+v", got)
	}
}

func TestResolveTrackerConnectionPreservesLegacySingleTracker(t *testing.T) {
	cfg := Config{Tracker: &TrackerConfig{Type: "jira", Project: "P", Board: "P board", BaseURL: "https://jira.example", Token: "canary"}}
	got, err := cfg.ResolveTrackerConnection("")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "legacy" || got.Provider != "jira" || got.Board != "P board" || got.TrackerConfig().Board != "P board" || got.Token.Reveal() != "canary" {
		t.Fatalf("connection = %+v", got)
	}
	if _, err := cfg.ResolveTrackerConnection("other"); err == nil {
		t.Fatal("explicit non-legacy ID unexpectedly selected legacy tracker")
	}
}
