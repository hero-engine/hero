package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Folder != ".hero" {
		t.Errorf("Folder = %q, want %q", cfg.Folder, ".hero")
	}
	if cfg.Team == nil {
		t.Fatal("Team config is nil")
	}
	if cfg.Team.StaleDays != 14 {
		t.Errorf("StaleDays = %d, want 14", cfg.Team.StaleDays)
	}
	if cfg.Team.NudgeLevel != "gentle" {
		t.Errorf("NudgeLevel = %q, want %q", cfg.Team.NudgeLevel, "gentle")
	}
	if cfg.Team.AutoContext != true {
		t.Error("AutoContext should be true by default")
	}
	if cfg.Conventions == nil {
		t.Fatal("Conventions config is nil")
	}
	if cfg.Conventions.Enforce != true {
		t.Error("Conventions.Enforce should be true by default")
	}
	if cfg.Conventions.ScopeDefault != "*" {
		t.Errorf("ScopeDefault = %q, want %q", cfg.Conventions.ScopeDefault, "*")
	}
	if cfg.Tracker == nil {
		t.Fatal("Tracker config is nil")
	}
	if cfg.Tracker.Type != "none" {
		t.Errorf("Tracker.Type = %q, want %q", cfg.Tracker.Type, "none")
	}
}

func TestLoadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should return defaults
	if cfg.Folder != ".hero" {
		t.Errorf("Folder = %q, want default %q", cfg.Folder, ".hero")
	}
	if cfg.Team.NudgeLevel != "gentle" {
		t.Errorf("NudgeLevel = %q, want default %q", cfg.Team.NudgeLevel, "gentle")
	}
}

func TestLoadCustomConfig(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	configJSON := `{
  "folder": ".hero",
  "team": {
    "require_review": true,
    "stale_days": 7,
    "auto_context": false,
    "nudge_level": "assertive"
  },
  "tracker": {
    "type": "github",
    "project": "acme/widgets",
    "token_env": "GITHUB_TOKEN",
    "post_on_design": true,
    "post_on_deliver": true
  },
  "conventions": {
    "enforce": false,
    "scope_default": "src/**/*"
  }
}`

	configPath := filepath.Join(heroDir, "hero.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Team.RequireReview != true {
		t.Error("RequireReview should be true")
	}
	if cfg.Team.StaleDays != 7 {
		t.Errorf("StaleDays = %d, want 7", cfg.Team.StaleDays)
	}
	if cfg.Team.AutoContext != false {
		t.Error("AutoContext should be false")
	}
	if cfg.Team.NudgeLevel != "assertive" {
		t.Errorf("NudgeLevel = %q, want %q", cfg.Team.NudgeLevel, "assertive")
	}
	if cfg.Tracker.Type != "github" {
		t.Errorf("Tracker.Type = %q, want %q", cfg.Tracker.Type, "github")
	}
	if cfg.Tracker.Project != "acme/widgets" {
		t.Errorf("Tracker.Project = %q, want %q", cfg.Tracker.Project, "acme/widgets")
	}
	if cfg.Tracker.TokenEnv != "GITHUB_TOKEN" {
		t.Errorf("Tracker.TokenEnv = %q, want %q", cfg.Tracker.TokenEnv, "GITHUB_TOKEN")
	}
	if cfg.Tracker.PostOnDesign != true {
		t.Error("PostOnDesign should be true")
	}
	if cfg.Conventions.Enforce != false {
		t.Error("Enforce should be false")
	}
	if cfg.Conventions.ScopeDefault != "src/**/*" {
		t.Errorf("ScopeDefault = %q, want %q", cfg.Conventions.ScopeDefault, "src/**/*")
	}
}

func TestLoadNudgeLevelDefault(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Config without nudge_level should default to "gentle"
	configJSON := `{
  "team": {
    "stale_days": 10
  }
}`
	configPath := filepath.Join(heroDir, "hero.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Team.NudgeLevel != "gentle" {
		t.Errorf("NudgeLevel = %q, want default %q", cfg.Team.NudgeLevel, "gentle")
	}
}

func TestSaveAndReload(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Team.NudgeLevel = "off"
	cfg.Team.StaleDays = 30

	if err := cfg.Save(tmpDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Team.NudgeLevel != "off" {
		t.Errorf("NudgeLevel = %q, want %q", loaded.Team.NudgeLevel, "off")
	}
	if loaded.Team.StaleDays != 30 {
		t.Errorf("StaleDays = %d, want 30", loaded.Team.StaleDays)
	}
}

func TestHeroDir(t *testing.T) {
	cfg := Config{Folder: ".hero"}
	got := cfg.HeroDir("/project")
	want := filepath.Join("/project", ".hero")
	if got != want {
		t.Errorf("HeroDir = %q, want %q", got, want)
	}
}

func TestConventionsDir(t *testing.T) {
	cfg := Config{Folder: ".hero"}
	got := cfg.ConventionsDir("/project")
	want := filepath.Join("/project", ".hero", "knowledge", "conventions")
	if got != want {
		t.Errorf("ConventionsDir = %q, want %q", got, want)
	}
}

func TestDecisionsDir(t *testing.T) {
	cfg := Config{Folder: ".hero"}
	got := cfg.DecisionsDir("/project")
	want := filepath.Join("/project", ".hero", "knowledge", "decisions")
	if got != want {
		t.Errorf("DecisionsDir = %q, want %q", got, want)
	}
}

func TestKnowledgeDir(t *testing.T) {
	cfg := Config{Folder: ".hero"}
	got := cfg.KnowledgeDir("/project")
	want := filepath.Join("/project", ".hero", "knowledge")
	if got != want {
		t.Errorf("KnowledgeDir = %q, want %q", got, want)
	}
}

func TestRulesDir(t *testing.T) {
	cfg := Config{Folder: ".hero"}
	got := cfg.RulesDir("/project")
	want := filepath.Join("/project", ".hero", "knowledge", "rules")
	if got != want {
		t.Errorf("RulesDir = %q, want %q", got, want)
	}
}

func TestExternalDir(t *testing.T) {
	cfg := Config{Folder: ".hero"}
	got := cfg.ExternalDir("/project")
	want := filepath.Join("/project", ".hero", "knowledge", "external")
	if got != want {
		t.Errorf("ExternalDir = %q, want %q", got, want)
	}
}

func TestContextDir(t *testing.T) {
	cfg := Config{Folder: ".hero"}
	got := cfg.ContextDir("/project")
	want := filepath.Join("/project", ".hero", "knowledge", "context")
	if got != want {
		t.Errorf("ContextDir = %q, want %q", got, want)
	}
}

func TestTemplatesDir(t *testing.T) {
	cfg := Config{Folder: ".hero"}
	got := cfg.TemplatesDir("/project")
	want := filepath.Join("/project", ".hero", "knowledge", "templates")
	if got != want {
		t.Errorf("TemplatesDir = %q, want %q", got, want)
	}
}

func TestNotesDir(t *testing.T) {
	cfg := Config{Folder: ".hero"}
	got := cfg.NotesDir("/project")
	want := filepath.Join("/project", ".hero", "knowledge", "notes")
	if got != want {
		t.Errorf("NotesDir = %q, want %q", got, want)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := Load(tmpDir)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

// --- EmbeddingsConfig ---

func TestIsEmbeddingsEnabled_Default(t *testing.T) {
	cfg := Config{}
	if !cfg.IsEmbeddingsEnabled() {
		t.Error("expected embeddings enabled by default (nil config)")
	}
}

func TestIsEmbeddingsEnabled_NilEnabled(t *testing.T) {
	cfg := Config{Embeddings: &EmbeddingsConfig{}}
	if !cfg.IsEmbeddingsEnabled() {
		t.Error("expected embeddings enabled when Enabled is nil")
	}
}

func TestIsEmbeddingsEnabled_True(t *testing.T) {
	enabled := true
	cfg := Config{Embeddings: &EmbeddingsConfig{Enabled: &enabled}}
	if !cfg.IsEmbeddingsEnabled() {
		t.Error("expected embeddings enabled when set to true")
	}
}

func TestIsEmbeddingsEnabled_False(t *testing.T) {
	disabled := false
	cfg := Config{Embeddings: &EmbeddingsConfig{Enabled: &disabled}}
	if cfg.IsEmbeddingsEnabled() {
		t.Error("expected embeddings disabled when set to false")
	}
}

func TestEmbeddingsScope_Default(t *testing.T) {
	cfg := Config{}
	scope := cfg.EmbeddingsScope()
	want := []string{"spec", "knowledge", "convention", "event", "code"}
	if len(scope) != len(want) {
		t.Fatalf("scope length = %d, want %d", len(scope), len(want))
	}
	for i, s := range scope {
		if s != want[i] {
			t.Errorf("scope[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestEmbeddingsScope_Custom(t *testing.T) {
	cfg := Config{Embeddings: &EmbeddingsConfig{Scope: []string{"spec", "knowledge"}}}
	scope := cfg.EmbeddingsScope()
	if len(scope) != 2 || scope[0] != "spec" || scope[1] != "knowledge" {
		t.Errorf("expected custom scope [spec knowledge], got %v", scope)
	}
}

func TestEmbeddingsModel_Default(t *testing.T) {
	cfg := Config{}
	if cfg.EmbeddingsModel() != "hero-embed-v1" {
		t.Errorf("EmbeddingsModel() = %q, want %q", cfg.EmbeddingsModel(), "hero-embed-v1")
	}
}

func TestEmbeddingsModel_Custom(t *testing.T) {
	cfg := Config{Embeddings: &EmbeddingsConfig{Model: "custom-model"}}
	if cfg.EmbeddingsModel() != "custom-model" {
		t.Errorf("EmbeddingsModel() = %q, want %q", cfg.EmbeddingsModel(), "custom-model")
	}
}

// --- ImportConfig.InventoryEnabled ---

func TestInventoryEnabled_NilConfig(t *testing.T) {
	var c *ImportConfig
	if !c.InventoryEnabled() {
		t.Error("nil ImportConfig should default to true")
	}
}

func TestInventoryEnabled_NilBool(t *testing.T) {
	c := &ImportConfig{}
	if !c.InventoryEnabled() {
		t.Error("nil Inventory bool should default to true")
	}
}

func TestInventoryEnabled_True(t *testing.T) {
	v := true
	c := &ImportConfig{Inventory: &v}
	if !c.InventoryEnabled() {
		t.Error("InventoryEnabled() should return true when set to true")
	}
}

func TestInventoryEnabled_False(t *testing.T) {
	v := false
	c := &ImportConfig{Inventory: &v}
	if c.InventoryEnabled() {
		t.Error("InventoryEnabled() should return false when set to false")
	}
}

// --- ImportConfig.EffectiveInventoryPath ---

func TestEffectiveInventoryPath_EmptyBug(t *testing.T) {
	c := &ImportConfig{}
	got := c.EffectiveInventoryPath("bug")
	want := "bugs/inventory.md"
	if got != want {
		t.Errorf("EffectiveInventoryPath(bug) = %q, want %q", got, want)
	}
}

func TestEffectiveInventoryPath_EmptyFeature(t *testing.T) {
	c := &ImportConfig{}
	got := c.EffectiveInventoryPath("feature")
	want := "features/inventory.md"
	if got != want {
		t.Errorf("EffectiveInventoryPath(feature) = %q, want %q", got, want)
	}
}

func TestEffectiveInventoryPath_Custom(t *testing.T) {
	c := &ImportConfig{InventoryPath: "custom/path.md"}
	got := c.EffectiveInventoryPath("bug")
	want := "custom/path.md"
	if got != want {
		t.Errorf("EffectiveInventoryPath = %q, want %q", got, want)
	}
}

// --- RefreshBehavior ---

func TestRefreshBehavior_NilDefaults(t *testing.T) {
	var r *RefreshBehavior

	if !r.ShouldUpdateStatus() {
		t.Error("nil RefreshBehavior should default ShouldUpdateStatus to true")
	}
	if !r.ShouldMarkReassigned() {
		t.Error("nil RefreshBehavior should default ShouldMarkReassigned to true")
	}
	if r.ResolvedAction() != "mark" {
		t.Errorf("nil RefreshBehavior ResolvedAction = %q, want %q", r.ResolvedAction(), "mark")
	}
}

func TestRefreshBehavior_ExplicitValues(t *testing.T) {
	f := false
	r := &RefreshBehavior{
		UpdateStatus:   &f,
		MarkReassigned: &f,
		RemoveResolved: "archive",
	}

	if r.ShouldUpdateStatus() {
		t.Error("ShouldUpdateStatus should return false when set to false")
	}
	if r.ShouldMarkReassigned() {
		t.Error("ShouldMarkReassigned should return false when set to false")
	}
	if r.ResolvedAction() != "archive" {
		t.Errorf("ResolvedAction = %q, want %q", r.ResolvedAction(), "archive")
	}
}

func TestRefreshBehavior_EmptyRemoveResolved(t *testing.T) {
	r := &RefreshBehavior{RemoveResolved: ""}
	if r.ResolvedAction() != "mark" {
		t.Errorf("ResolvedAction with empty string = %q, want %q", r.ResolvedAction(), "mark")
	}
}

// --- PrimeConfig ---

func TestPrimeConfig_Defaults(t *testing.T) {
	var c *PrimeConfig
	if !c.AutoEnabled() {
		t.Error("nil PrimeConfig should default AutoEnabled to true")
	}
	if !c.KnowledgeEnabled() {
		t.Error("nil PrimeConfig should default KnowledgeEnabled to true")
	}
}

func TestPrimeConfig_Disabled(t *testing.T) {
	f := false
	c := &PrimeConfig{Auto: &f}
	if c.AutoEnabled() {
		t.Error("AutoEnabled should return false when Auto is set to false")
	}
}

// --- ServeConfig.UIEnabled ---

func boolPtr(v bool) *bool { return &v }

func TestServeConfig_UIEnabled_Default(t *testing.T) {
	var c *ServeConfig
	if !c.UIEnabled() {
		t.Error("nil ServeConfig should default UIEnabled to true")
	}
}

func TestServeConfig_UIEnabled_False(t *testing.T) {
	c := &ServeConfig{UI: boolPtr(false)}
	if c.UIEnabled() {
		t.Error("UIEnabled should return false when UI is set to false")
	}
}

// --- TrackerConfig.ResolveToken ---

func TestTrackerResolveToken_LiteralToken(t *testing.T) {
	cfg := &TrackerConfig{Token: "mytoken"}
	got, err := cfg.ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "mytoken" {
		t.Errorf("ResolveToken = %q, want %q", got, "mytoken")
	}
}

func TestTrackerResolveToken_EnvVar(t *testing.T) {
	t.Setenv("TEST_HERO_TOKEN", "envtoken")
	cfg := &TrackerConfig{TokenEnv: "TEST_HERO_TOKEN"}
	got, err := cfg.ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "envtoken" {
		t.Errorf("ResolveToken = %q, want %q", got, "envtoken")
	}
}

func TestTrackerResolveToken_LiteralTakesPriority(t *testing.T) {
	t.Setenv("TEST_HERO_TOKEN2", "envtoken")
	cfg := &TrackerConfig{Token: "literal", TokenEnv: "TEST_HERO_TOKEN2"}
	got, err := cfg.ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "literal" {
		t.Errorf("expected literal token to take priority, got %q", got)
	}
}

func TestTrackerResolveToken_MissingEnvVar(t *testing.T) {
	os.Unsetenv("TEST_HERO_MISSING")
	cfg := &TrackerConfig{TokenEnv: "TEST_HERO_MISSING"}
	_, err := cfg.ResolveToken()
	if err == nil {
		t.Fatal("expected error for missing env var, got nil")
	}
}

func TestTrackerResolveToken_NoConfig(t *testing.T) {
	cfg := &TrackerConfig{}
	_, err := cfg.ResolveToken()
	if err == nil {
		t.Fatal("expected error when neither Token nor TokenEnv is set, got nil")
	}
}

// --- ConfluenceConfig.ResolveToken ---

func TestConfluenceResolveToken_LiteralToken(t *testing.T) {
	cfg := &ConfluenceConfig{Token: "conftoken"}
	got, err := cfg.ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "conftoken" {
		t.Errorf("ResolveToken = %q, want %q", got, "conftoken")
	}
}

func TestConfluenceResolveToken_EnvVar(t *testing.T) {
	t.Setenv("TEST_CONF_TOKEN", "fromenv")
	cfg := &ConfluenceConfig{TokenEnv: "TEST_CONF_TOKEN"}
	got, err := cfg.ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "fromenv" {
		t.Errorf("ResolveToken = %q, want %q", got, "fromenv")
	}
}

// --- LoadLocal / SaveLocal / MergeLocal ---

func TestLoadLocalMissing(t *testing.T) {
	tmpDir := t.TempDir()
	local, err := LoadLocal(tmpDir, DefaultFolder)
	if err != nil {
		t.Fatalf("LoadLocal should not error on missing file, got: %v", err)
	}
	// Should return zero-value Config
	if local.Tracker != nil {
		t.Error("expected nil Tracker in zero-value local config")
	}
}

func TestSaveAndLoadLocal(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	local := Config{
		Tracker: &TrackerConfig{Token: "secret-token"},
	}
	if err := SaveLocal(tmpDir, DefaultFolder, local); err != nil {
		t.Fatalf("SaveLocal: %v", err)
	}

	loaded, err := LoadLocal(tmpDir, DefaultFolder)
	if err != nil {
		t.Fatalf("LoadLocal: %v", err)
	}
	if loaded.Tracker == nil {
		t.Fatal("Tracker is nil after reload")
	}
	if loaded.Tracker.Token != "secret-token" {
		t.Errorf("Token = %q, want %q", loaded.Tracker.Token, "secret-token")
	}
}

func TestSaveLocalFileMode(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	local := Config{Tracker: &TrackerConfig{Token: "tok"}}
	if err := SaveLocal(tmpDir, DefaultFolder, local); err != nil {
		t.Fatalf("SaveLocal: %v", err)
	}

	info, err := os.Stat(filepath.Join(heroDir, LocalConfigFileName))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestMergeLocal_TrackerToken(t *testing.T) {
	base := DefaultConfig()
	base.Tracker = &TrackerConfig{Type: "github", Project: "acme/widgets"}

	local := Config{
		Tracker: &TrackerConfig{Token: "my-token"},
	}

	merged := MergeLocal(base, local)
	if merged.Tracker.Token != "my-token" {
		t.Errorf("Token = %q, want %q", merged.Tracker.Token, "my-token")
	}
	// Base fields preserved
	if merged.Tracker.Type != "github" {
		t.Errorf("Type = %q, want %q", merged.Tracker.Type, "github")
	}
	if merged.Tracker.Project != "acme/widgets" {
		t.Errorf("Project = %q, want %q", merged.Tracker.Project, "acme/widgets")
	}
}

func TestMergeLocal_NudgeLevel(t *testing.T) {
	base := DefaultConfig()
	local := Config{
		Team: &TeamConfig{NudgeLevel: "off"},
	}
	merged := MergeLocal(base, local)
	if merged.Team.NudgeLevel != "off" {
		t.Errorf("NudgeLevel = %q, want %q", merged.Team.NudgeLevel, "off")
	}
	// StaleDays not overridden
	if merged.Team.StaleDays != 14 {
		t.Errorf("StaleDays = %d, want 14", merged.Team.StaleDays)
	}
}

func TestMergeLocal_EmptyLocalNoChange(t *testing.T) {
	base := DefaultConfig()
	base.Tracker = &TrackerConfig{Type: "linear", Project: "ENG", Token: "existing"}
	merged := MergeLocal(base, Config{})
	if merged.Tracker.Token != "existing" {
		t.Errorf("Token should not be overwritten by empty local, got %q", merged.Tracker.Token)
	}
}

func TestLoad_AppliesCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write hero.json with tracker but no token
	baseJSON := `{"tracker":{"type":"github","project":"acme/widgets"}}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(baseJSON), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Write credentials to XDG path
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	creds := make(Credentials)
	SetCredential(creds, "github", "acme/widgets", CredentialEntry{Token: "cred-token"})
	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Tracker == nil {
		t.Fatal("Tracker is nil")
	}
	if cfg.Tracker.Token != "cred-token" {
		t.Errorf("Token = %q, want %q from credentials store", cfg.Tracker.Token, "cred-token")
	}
}

func TestLoad_LocalTokenTakesPriorityOverCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	baseJSON := `{"tracker":{"type":"github","project":"acme/widgets"}}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(baseJSON), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Local config has a token
	localJSON := `{"tracker":{"token":"local-wins"}}`
	if err := os.WriteFile(filepath.Join(heroDir, LocalConfigFileName), []byte(localJSON), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Credentials also have a token for the same project
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	creds := make(Credentials)
	SetCredential(creds, "github", "acme/widgets", CredentialEntry{Token: "cred-token"})
	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Tracker.Token != "local-wins" {
		t.Errorf("Token = %q, want %q (local should win over creds)", cfg.Tracker.Token, "local-wins")
	}
}

func TestLoad_AppliesLocalOverride(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write base hero.json
	baseJSON := `{"tracker":{"type":"github","project":"acme/widgets","token_env":"GITHUB_TOKEN"}}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(baseJSON), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Write local hero.local.json with token override
	localJSON := `{"tracker":{"token":"local-secret"}}`
	if err := os.WriteFile(filepath.Join(heroDir, LocalConfigFileName), []byte(localJSON), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Tracker == nil {
		t.Fatal("Tracker is nil")
	}
	if cfg.Tracker.Token != "local-secret" {
		t.Errorf("Token = %q, want %q", cfg.Tracker.Token, "local-secret")
	}
	// Base fields still intact
	if cfg.Tracker.Type != "github" {
		t.Errorf("Type = %q, want %q", cfg.Tracker.Type, "github")
	}
}

// --- MergeLocal: dialect fields (vocabulary + methodology) ---

func TestMergeLocal_MethodologyOverridesScalar(t *testing.T) {
	base := DefaultConfig()
	base.Methodology = "scrum"
	base.Vocabulary = "agile-scrum"

	local := Config{Methodology: "shape-up"}
	merged := MergeLocal(base, local)

	if merged.Methodology != "shape-up" {
		t.Errorf("Methodology = %q, want %q", merged.Methodology, "shape-up")
	}
	// Vocabulary not touched
	if merged.Vocabulary != "agile-scrum" {
		t.Errorf("Vocabulary = %q, want %q (untouched)", merged.Vocabulary, "agile-scrum")
	}
}

func TestMergeLocal_VocabularyOverridesScalar(t *testing.T) {
	base := DefaultConfig()
	base.Vocabulary = "agile-scrum"

	local := Config{Vocabulary: "shape-up"}
	merged := MergeLocal(base, local)

	if merged.Vocabulary != "shape-up" {
		t.Errorf("Vocabulary = %q, want %q", merged.Vocabulary, "shape-up")
	}
}

func TestMergeLocal_BothDialectScalars(t *testing.T) {
	base := DefaultConfig()
	base.Methodology = "scrum"
	base.Vocabulary = "agile-scrum"

	local := Config{Methodology: "shape-up", Vocabulary: "shape-up"}
	merged := MergeLocal(base, local)

	if merged.Methodology != "shape-up" {
		t.Errorf("Methodology = %q, want %q", merged.Methodology, "shape-up")
	}
	if merged.Vocabulary != "shape-up" {
		t.Errorf("Vocabulary = %q, want %q", merged.Vocabulary, "shape-up")
	}
}

func TestMergeLocal_VocabularyOverridesMapMerge(t *testing.T) {
	base := DefaultConfig()
	base.VocabularyOverrides = map[string]string{
		"types.spec":        "BaseStory",
		"sections.criteria": "BaseCriteria",
	}

	local := Config{
		VocabularyOverrides: map[string]string{
			"types.spec": "LocalStory", // collision: local wins
			"types.epic": "LocalEpic",  // new key
		},
	}

	merged := MergeLocal(base, local)

	if got := merged.VocabularyOverrides["types.spec"]; got != "LocalStory" {
		t.Errorf("types.spec = %q, want %q (local should win on collision)", got, "LocalStory")
	}
	if got := merged.VocabularyOverrides["types.epic"]; got != "LocalEpic" {
		t.Errorf("types.epic = %q, want %q (new local key)", got, "LocalEpic")
	}
	if got := merged.VocabularyOverrides["sections.criteria"]; got != "BaseCriteria" {
		t.Errorf("sections.criteria = %q, want %q (non-colliding base preserved)", got, "BaseCriteria")
	}
}

func TestMergeLocal_MethodologyOverridesMapMerge(t *testing.T) {
	base := DefaultConfig()
	base.MethodologyOverrides = map[string]string{
		"time_boxes.iteration.duration_default": "2w",
		"in_flight_tracking":                    "wip_aging",
	}

	local := Config{
		MethodologyOverrides: map[string]string{
			"time_boxes.iteration.duration_default": "3w",       // collision: local wins
			"estimation.feature.required_field":     "appetite", // new key
		},
	}

	merged := MergeLocal(base, local)

	if got := merged.MethodologyOverrides["time_boxes.iteration.duration_default"]; got != "3w" {
		t.Errorf("duration_default = %q, want %q (local should win)", got, "3w")
	}
	if got := merged.MethodologyOverrides["estimation.feature.required_field"]; got != "appetite" {
		t.Errorf("required_field = %q, want %q (new local key)", got, "appetite")
	}
	if got := merged.MethodologyOverrides["in_flight_tracking"]; got != "wip_aging" {
		t.Errorf("in_flight_tracking = %q, want %q (non-colliding base preserved)", got, "wip_aging")
	}
}

func TestMergeLocal_VocabularyOverridesIntoNilBase(t *testing.T) {
	base := DefaultConfig()
	// base.VocabularyOverrides intentionally nil

	local := Config{
		VocabularyOverrides: map[string]string{"types.spec": "LocalStory"},
	}

	merged := MergeLocal(base, local)
	if got := merged.VocabularyOverrides["types.spec"]; got != "LocalStory" {
		t.Errorf("types.spec = %q, want %q (should populate from nil base)", got, "LocalStory")
	}
}

func TestMergeLocal_EmptyLocalLeavesDialectUntouched(t *testing.T) {
	base := DefaultConfig()
	base.Methodology = "scrum"
	base.Vocabulary = "agile-scrum"
	base.VocabularyOverrides = map[string]string{"types.spec": "Story"}
	base.MethodologyOverrides = map[string]string{"in_flight_tracking": "wip_aging"}

	merged := MergeLocal(base, Config{})

	if merged.Methodology != "scrum" {
		t.Errorf("Methodology = %q, want %q (no change expected)", merged.Methodology, "scrum")
	}
	if merged.Vocabulary != "agile-scrum" {
		t.Errorf("Vocabulary = %q, want %q (no change expected)", merged.Vocabulary, "agile-scrum")
	}
	if got := merged.VocabularyOverrides["types.spec"]; got != "Story" {
		t.Errorf("VocabularyOverrides[types.spec] = %q, want %q (no change)", got, "Story")
	}
	if got := merged.MethodologyOverrides["in_flight_tracking"]; got != "wip_aging" {
		t.Errorf("MethodologyOverrides[in_flight_tracking] = %q, want %q (no change)", got, "wip_aging")
	}
}

// --- MockupsConfig ---

func TestMockupsConfig_RoundTripSwiftUI(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	configJSON := `{"folder":".hero","mockups":{"renderer":"swiftui"}}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Mockups == nil {
		t.Fatal("Mockups should be populated, got nil")
	}
	if cfg.Mockups.Renderer != "swiftui" {
		t.Errorf("Mockups.Renderer = %q, want %q", cfg.Mockups.Renderer, "swiftui")
	}
}

func TestMockupsConfig_RoundTripHTML(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	configJSON := `{"folder":".hero","mockups":{"renderer":"html"}}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Mockups == nil {
		t.Fatal("Mockups should be populated, got nil")
	}
	if cfg.Mockups.Renderer != "html" {
		t.Errorf("Mockups.Renderer = %q, want %q", cfg.Mockups.Renderer, "html")
	}
}

func TestMockupsConfig_UnsetMeansNil(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	configJSON := `{"folder":".hero"}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(configJSON), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Mockups != nil {
		t.Errorf("Mockups should be nil when unset, got %+v", cfg.Mockups)
	}
}

func TestLoad_AppliesLocalDialectOverride(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Team workspace: scrum + agile-scrum.
	baseJSON := `{"methodology":"scrum","vocabulary":"agile-scrum"}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(baseJSON), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Developer override: shape-up.
	localJSON := `{"methodology":"shape-up","vocabulary":"shape-up"}`
	if err := os.WriteFile(filepath.Join(heroDir, LocalConfigFileName), []byte(localJSON), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Methodology != "shape-up" {
		t.Errorf("Methodology = %q, want %q (local override should win)", cfg.Methodology, "shape-up")
	}
	if cfg.Vocabulary != "shape-up" {
		t.Errorf("Vocabulary = %q, want %q (local override should win)", cfg.Vocabulary, "shape-up")
	}
}

// --- SizeMappingConfig validation ---
// Slice 5 of spec-size-and-promotion-nudge: hero.json may carry a
// `tracker.size_mapping` block describing how Hero's local size tier
// maps to a tracker-side field. Absent block is fine; present block
// must validate at load time so a typo doesn't silently disable size
// sync.

func TestSizeMappingConfig_AbsentIsNoop(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Tracker with no size_mapping should load fine.
	json := `{
  "tracker": {"type": "jira", "project": "PROJ", "base_url": "https://example.atlassian.net", "token_env": "EX"}
}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(json), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Tracker == nil || cfg.Tracker.SizeMapping != nil {
		t.Errorf("expected nil SizeMapping, got %+v", cfg.Tracker.SizeMapping)
	}
}

func TestSizeMappingConfig_ValidLoads(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	json := `{
  "tracker": {
    "type": "jira", "project": "PROJ", "base_url": "https://example.atlassian.net", "token_env": "EX",
    "size_mapping": {
      "field": "story_points",
      "thresholds": {
        "trivial": [0, 1],
        "small": [2, 2],
        "medium": [3, 5],
        "large": [8, 8],
        "x-large": [13, 13],
        "giant": [20, null]
      },
      "container_field": "epic_label"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(json), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Tracker.SizeMapping == nil {
		t.Fatal("SizeMapping is nil")
	}
	if cfg.Tracker.SizeMapping.Field != "story_points" {
		t.Errorf("Field = %q, want story_points", cfg.Tracker.SizeMapping.Field)
	}
	if cfg.Tracker.SizeMapping.ContainerField != "epic_label" {
		t.Errorf("ContainerField = %q, want epic_label", cfg.Tracker.SizeMapping.ContainerField)
	}
	if len(cfg.Tracker.SizeMapping.Thresholds) != 6 {
		t.Errorf("got %d thresholds, want 6", len(cfg.Tracker.SizeMapping.Thresholds))
	}
	if band := cfg.Tracker.SizeMapping.Thresholds["giant"]; len(band) != 2 || band[0] == nil || *band[0] != 20 || band[1] != nil {
		t.Errorf("giant band = %+v, want [20, nil]", band)
	}
}

func TestSizeMappingConfig_MissingFieldRejected(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	json := `{
  "tracker": {
    "type": "jira", "project": "PROJ", "base_url": "https://example.atlassian.net", "token_env": "EX",
    "size_mapping": {
      "thresholds": {"small": [1, 2]}
    }
  }
}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(json), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := Load(tmpDir)
	if err == nil {
		t.Fatal("expected Load error for missing field")
	}
}

func TestSizeMappingConfig_UnknownTierRejected(t *testing.T) {
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	json := `{
  "tracker": {
    "type": "jira", "project": "PROJ", "base_url": "https://example.atlassian.net", "token_env": "EX",
    "size_mapping": {
      "field": "story_points",
      "thresholds": {"enormous": [1, 2]}
    }
  }
}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(json), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := Load(tmpDir)
	if err == nil {
		t.Fatal("expected Load error for unknown tier")
	}
}

func TestSizeMappingConfig_NoneTrackerSkipsValidation(t *testing.T) {
	// `tracker.type: "none"` (the slice-5 workspace default) should
	// never fail validation even if a stale size_mapping block exists.
	tmpDir := t.TempDir()
	heroDir := filepath.Join(tmpDir, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	json := `{
  "tracker": {
    "type": "none",
    "size_mapping": {"field": ""}
  }
}`
	if err := os.WriteFile(filepath.Join(heroDir, ConfigFileName), []byte(json), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := Load(tmpDir); err != nil {
		t.Errorf("Load should not error on tracker.type=none: %v", err)
	}
}

// TestNextGoalCapture_DefaultsToFloor pins the default and the typo-safe
// fallback for the SessionGoal capture knob.
func TestNextGoalCapture_DefaultsToFloor(t *testing.T) {
	// No Next config → "floor".
	if got := DefaultConfig().NextGoalCapture(); got != "floor" {
		t.Errorf("default NextGoalCapture = %q, want %q", got, "floor")
	}
	// Empty Next block → "floor".
	cfg := DefaultConfig()
	cfg.Next = &NextConfig{}
	if got := cfg.NextGoalCapture(); got != "floor" {
		t.Errorf("empty Next NextGoalCapture = %q, want %q", got, "floor")
	}
	// Explicit "embed" → "embed".
	cfg.Next = &NextConfig{GoalCapture: "embed"}
	if got := cfg.NextGoalCapture(); got != "embed" {
		t.Errorf("embed NextGoalCapture = %q, want %q", got, "embed")
	}
	// Unrecognized → "floor" (never silently disable capture).
	cfg.Next = &NextConfig{GoalCapture: "bogus"}
	if got := cfg.NextGoalCapture(); got != "floor" {
		t.Errorf("unrecognized NextGoalCapture = %q, want fallback %q", got, "floor")
	}
}
