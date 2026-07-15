package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// IntegrationsConfig is shared verbatim by hero.json and hero.local.json.
type IntegrationsConfig struct {
	Default     string                       `json:"default,omitempty"`
	Roles       map[string]string            `json:"roles,omitempty"`
	Connections map[string]IntegrationConfig `json:"connections,omitempty"`
}

type IntegrationConfig struct {
	Provider string                     `json:"provider,omitempty"`
	Settings map[string]json.RawMessage `json:"settings,omitempty"`
	Auth     *IntegrationAuth           `json:"auth,omitempty"`
}

type IntegrationAuth struct {
	Token    Secret `json:"token,omitempty"`
	TokenEnv string `json:"token_env,omitempty"`
}

// Secret deliberately redacts all generic formatting and JSON serialization.
type Secret string

func (Secret) String() string   { return "[REDACTED]" }
func (Secret) GoString() string { return "[REDACTED]" }
func (s Secret) MarshalJSON() ([]byte, error) {
	if s == "" {
		return []byte(`""`), nil
	}
	return []byte(`"[REDACTED]"`), nil
}
func (s *Secret) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*s = Secret(v)
	return nil
}
func (s Secret) Reveal() string { return string(s) }

type IntegrationSource struct{ File, Path string }
type ResolvedIntegrations struct {
	Config     *IntegrationsConfig
	Provenance map[string]IntegrationSource
}

// SelectTracker resolves a delivery integration without map-order fallback.
func (c Config) SelectTracker(explicit string) (*TrackerConfig, error) {
	if c.Integrations == nil {
		if explicit != "" {
			return nil, fmt.Errorf("integration %q not found in legacy tracker configuration", explicit)
		}
		if c.Tracker == nil {
			return nil, fmt.Errorf("no delivery integration configured")
		}
		return c.Tracker, nil
	}
	r := ResolvedIntegrations{Config: c.Integrations}
	_, x, err := r.Select(explicit, "delivery")
	if err != nil {
		return nil, err
	}
	tmp := &ResolvedIntegrations{Config: &IntegrationsConfig{Default: "selected", Connections: map[string]IntegrationConfig{"selected": x}}}
	t, ok := tmp.DeliveryTracker()
	if !ok {
		return nil, fmt.Errorf("selected integration %q is not a delivery tracker", explicit)
	}
	return t, nil
}

var integrationIDRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var providers = map[string]bool{"github": true, "jira": true, "linear": true, "gitlab": true, "confluence": true}
var roles = map[string]bool{"delivery": true, "roadmap": true, "docs": true}

func ResolveIntegrationDocuments(committedPath string, committed []byte, localPath string, local []byte) (*ResolvedIntegrations, error) {
	if err := validateLegacyUnknown(committedPath, committed); err != nil {
		return nil, err
	}
	if err := validateLegacyUnknown(localPath, local); err != nil {
		return nil, err
	}
	c, cp, err := integrationNode(committedPath, committed, true)
	if err != nil {
		return nil, err
	}
	l, lp, err := integrationNode(localPath, local, false)
	if err != nil {
		return nil, err
	}
	if c == nil && l == nil {
		return nil, nil
	}
	if hasLegacyIntegration(committed) || hasLegacyIntegration(local) {
		return nil, fmt.Errorf("canonical $.integrations conflicts with legacy $.tracker/$.confluence across %s and %s; remove the legacy declaration explicitly", committedPath, localPath)
	}
	merged := c
	if l != nil {
		merged = mergePatch(c, l)
	}
	b, _ := json.Marshal(merged)
	var out IntegrationsConfig
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("integrations: %w", err)
	}
	prov := cp
	for k, v := range lp {
		prov[k] = v
	}
	r := &ResolvedIntegrations{Config: &out, Provenance: prov}
	if err := r.validate(committedPath, localPath); err != nil {
		return nil, err
	}
	return r, nil
}

func hasLegacyIntegration(data []byte) bool {
	var d map[string]json.RawMessage
	if json.Unmarshal(data, &d) != nil {
		return false
	}
	t := false
	if raw, ok := d["tracker"]; ok {
		var m map[string]any
		if json.Unmarshal(raw, &m) == nil {
			typ, _ := m["type"].(string)
			t = typ != "" && typ != "none"
		}
	}
	_, c := d["confluence"]
	return t || c
}

func integrationNode(path string, data []byte, committed bool) (any, map[string]IntegrationSource, error) {
	p := map[string]IntegrationSource{}
	if len(data) == 0 {
		return nil, p, nil
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, p, fmt.Errorf("parsing %s: %w", path, err)
	}
	raw, ok := doc["integrations"]
	if !ok {
		return nil, p, nil
	}
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, p, fmt.Errorf("%s $.integrations: %w", path, err)
	}
	walkSources(node, "$.integrations", path, p)
	if committed && containsLiteralToken(node) {
		return nil, p, fmt.Errorf("%s $.integrations: literal auth.token is forbidden in committed config; put it in .hero/hero.local.json, a global credential, or use auth.token_env", path)
	}
	return node, p, nil
}
func walkSources(v any, path, file string, p map[string]IntegrationSource) {
	p[path] = IntegrationSource{file, path}
	if m, ok := v.(map[string]any); ok {
		for k, x := range m {
			walkSources(x, path+"."+k, file, p)
		}
	}
}
func containsLiteralToken(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	for k, x := range m {
		if k == "token" {
			if s, ok := x.(string); ok && s != "" {
				return true
			}
		}
		if containsLiteralToken(x) {
			return true
		}
	}
	return false
}
func mergePatch(base, patch any) any {
	if patch == nil {
		return nil
	}
	pm, ok := patch.(map[string]any)
	if !ok {
		return patch
	}
	bm, _ := base.(map[string]any)
	out := map[string]any{}
	for k, v := range bm {
		out[k] = v
	}
	for k, v := range pm {
		if v == nil {
			delete(out, k)
		} else {
			out[k] = mergePatch(out[k], v)
		}
	}
	return out
}
func validateLegacyUnknown(path string, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var d map[string]json.RawMessage
	if json.Unmarshal(data, &d) != nil {
		return nil
	}
	if r := d["tracker"]; len(r) > 0 {
		var m map[string]json.RawMessage
		if json.Unmarshal(r, &m) == nil {
			allowed := map[string]bool{"type": true, "project": true, "token": true, "token_env": true, "base_url": true, "user_email": true, "post_on_design": true, "post_on_deliver": true, "size_mapping": true}
			for k := range m {
				if !allowed[k] {
					return fmt.Errorf("%s $.tracker.%s: unknown field; provider-keyed credentials are not supported here — define $.integrations.connections.<id> or run 'hero connect jira --integration-id <id> --project <key> --token-stdin'", path, k)
				}
			}
		}
	}
	return nil
}

func (r *ResolvedIntegrations) validate(committedPath, localPath string) error {
	c := r.Config
	for role, id := range c.Roles {
		if !roles[role] {
			return fmt.Errorf("$.integrations.roles.%s: unknown role", role)
		}
		if _, ok := c.Connections[id]; !ok {
			return fmt.Errorf("$.integrations.roles.%s references missing $.integrations.connections.%s", role, id)
		}
	}
	if c.Default != "" {
		if _, ok := c.Connections[c.Default]; !ok {
			return fmt.Errorf("$.integrations.default references missing $.integrations.connections.%s", c.Default)
		}
	}
	for id, cn := range c.Connections {
		if !integrationIDRE.MatchString(id) {
			return fmt.Errorf("$.integrations.connections.%s: invalid integration ID", id)
		}
		if !providers[cn.Provider] {
			return fmt.Errorf("$.integrations.connections.%s.provider: unknown provider %q", id, cn.Provider)
		}
		if err := validateProviderSettings(id, cn); err != nil {
			return err
		}
	}
	return nil
}
func validateProviderSettings(id string, c IntegrationConfig) error {
	stringFields := map[string]bool{"project": true, "base_url": true, "user_email": true, "space_key": true}
	commonTracker := map[string]string{"project": "string", "base_url": "string", "post_on_design": "bool", "post_on_deliver": "bool", "size_mapping": "object"}
	schema := map[string]string{}
	required := []string{}
	switch c.Provider {
	case "github", "linear":
		for k, v := range commonTracker {
			schema[k] = v
		}
		required = []string{"project"}
	case "jira":
		for k, v := range commonTracker {
			schema[k] = v
		}
		schema["user_email"] = "string"
		required = []string{"project", "base_url"}
	case "gitlab":
		for k, v := range commonTracker {
			schema[k] = v
		}
		required = []string{"project", "base_url"}
	case "confluence":
		schema = map[string]string{"space_key": "string", "base_url": "string", "user_email": "string"}
		required = []string{"space_key", "base_url"}
	default:
		return fmt.Errorf("$.integrations.connections.%s.provider: unknown provider %q", id, c.Provider)
	}
	for k := range c.Settings {
		want, ok := schema[k]
		if !ok {
			return fmt.Errorf("$.integrations.connections.%s.settings.%s: unknown %s setting", id, k, c.Provider)
		}
		path := fmt.Sprintf("$.integrations.connections.%s.settings.%s", id, k)
		switch want {
		case "string":
			if string(c.Settings[k]) == "null" {
				return fmt.Errorf("%s: expected string", path)
			}
			var v string
			if err := json.Unmarshal(c.Settings[k], &v); err != nil {
				return fmt.Errorf("%s: expected string", path)
			}
		case "bool":
			var v bool
			if err := json.Unmarshal(c.Settings[k], &v); err != nil {
				return fmt.Errorf("%s: expected boolean", path)
			}
		case "object":
			if string(c.Settings[k]) == "null" {
				return fmt.Errorf("%s: expected object", path)
			}
			var v map[string]json.RawMessage
			if err := json.Unmarshal(c.Settings[k], &v); err != nil || v == nil {
				return fmt.Errorf("%s: expected object", path)
			}
		}
	}
	for _, key := range required {
		raw, ok := c.Settings[key]
		path := fmt.Sprintf("$.integrations.connections.%s.settings.%s", id, key)
		if !ok {
			return fmt.Errorf("%s is required", path)
		}
		if stringFields[key] {
			if string(raw) == "null" {
				return fmt.Errorf("%s: expected string", path)
			}
			var v string
			if json.Unmarshal(raw, &v) != nil {
				return fmt.Errorf("%s: expected string", path)
			}
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("%s must not be empty", path)
			}
		}
	}
	if raw, ok := c.Settings["size_mapping"]; ok {
		var sm SizeMappingConfig
		if err := json.Unmarshal(raw, &sm); err != nil {
			return fmt.Errorf("$.integrations.connections.%s.settings.size_mapping: expected object", id)
		}
		if err := sm.Validate(); err != nil {
			return fmt.Errorf("$.integrations.connections.%s.settings.size_mapping: %w", id, err)
		}
	}
	if c.Provider == "jira" {
		if raw, ok := c.Settings["user_email"]; ok {
			var v string
			_ = json.Unmarshal(raw, &v)
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("$.integrations.connections.%s.settings.user_email must not be empty when provided", id)
			}
		}
	}
	return nil
}

func (r *ResolvedIntegrations) Select(explicit, role string) (string, IntegrationConfig, error) {
	c := r.Config
	id := explicit
	if id == "" && role != "" {
		id = c.Roles[role]
	}
	if id == "" {
		id = c.Default
	}
	if id == "" {
		return "", IntegrationConfig{}, fmt.Errorf("no integration selected; set integrations.default, integrations.roles.%s, or pass --integration <id>", role)
	}
	v, ok := c.Connections[id]
	if !ok {
		return "", IntegrationConfig{}, fmt.Errorf("integration %q not found", id)
	}
	return id, v, nil
}
func rawString(m map[string]json.RawMessage, k string) string {
	var s string
	_ = json.Unmarshal(m[k], &s)
	return s
}
func (r *ResolvedIntegrations) DeliveryTracker() (*TrackerConfig, bool) {
	_, c, e := r.Select("", "delivery")
	if e != nil || c.Provider == "confluence" {
		return nil, false
	}
	t := &TrackerConfig{Type: c.Provider, Project: rawString(c.Settings, "project"), BaseURL: rawString(c.Settings, "base_url"), UserEmail: rawString(c.Settings, "user_email")}
	if c.Auth != nil {
		t.Token = c.Auth.Token.Reveal()
		t.TokenEnv = c.Auth.TokenEnv
	}
	return t, true
}
func (r *ResolvedIntegrations) DocsConfluence() (*ConfluenceConfig, bool) {
	_, c, e := r.Select("", "docs")
	if e != nil || c.Provider != "confluence" {
		return nil, false
	}
	x := &ConfluenceConfig{SpaceKey: rawString(c.Settings, "space_key"), BaseURL: rawString(c.Settings, "base_url"), UserEmail: rawString(c.Settings, "user_email")}
	if c.Auth != nil {
		x.Token = c.Auth.Token.Reveal()
		x.TokenEnv = c.Auth.TokenEnv
	}
	return x, true
}
func ValidateCommittedIntegrations(c *IntegrationsConfig, path string) error {
	if c == nil {
		return nil
	}
	for id, x := range c.Connections {
		if x.Auth != nil && x.Auth.Token != "" {
			return fmt.Errorf("%s $.integrations.connections.%s.auth.token: literal credentials are forbidden in committed config", path, id)
		}
	}
	return nil
}
func SortedConnectionIDs(c *IntegrationsConfig) []string {
	ids := make([]string, 0, len(c.Connections))
	for id := range c.Connections {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func CredentialGuidance(id string) string {
	return fmt.Sprintf("integration %q has no usable credential; add auth.token to .hero/hero.local.json, configure a global credential, or set auth.token_env; inspect with 'hero connect --list'", strings.TrimSpace(id))
}

// PatchLocalIntegrations applies a merge patch to only the integrations subtree.
// Unrelated local keys are retained byte-for-byte semantically and the replacement
// is atomic and private. The patch is raw JSON so literal local tokens are never
// passed through generic Config serialization or formatting.
func PatchLocalIntegrations(projectRoot, folder string, patch json.RawMessage) error {
	if folder == "" {
		folder = DefaultFolder
	}
	dir := filepath.Join(projectRoot, folder)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, LocalConfigFileName)
	doc := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &doc); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	var p any
	if err := json.Unmarshal(patch, &p); err != nil {
		return fmt.Errorf("invalid integrations patch: %w", err)
	}
	doc["integrations"] = mergePatch(doc["integrations"], p)
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(dir, ".hero.local.json.*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// PatchCommittedIntegrations updates only the non-secret committed subtree.
func PatchCommittedIntegrations(projectRoot, folder string, patch json.RawMessage) error {
	if folder == "" {
		folder = DefaultFolder
	}
	var probe any
	if err := json.Unmarshal(patch, &probe); err != nil {
		return err
	}
	if containsLiteralToken(probe) {
		return fmt.Errorf("$.integrations: literal auth.token is forbidden in committed config")
	}
	dir := filepath.Join(projectRoot, folder)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, ConfigFileName)
	doc := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &doc); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	doc["integrations"] = mergePatch(doc["integrations"], probe)
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(dir, ".hero.json.*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
