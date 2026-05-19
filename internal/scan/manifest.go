package scan

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Manifest mirrors the shape of domains/<name>/scan-manifest.yaml.
// The parser is intentionally narrow — only fields the dispatcher
// needs at v1. Unknown fields are preserved by yaml.v3's default
// strict mode but ignored by the runtime; future fields land as
// additive parser updates without breaking older binaries.
type Manifest struct {
	ManifestVersion string             `yaml:"manifest_version"`
	ScannerID       string             `yaml:"scanner_id"`
	DisplayName     string             `yaml:"display_name"`
	Subcommands     []ManifestSubcmd   `yaml:"subcommands"`
	Emits           ManifestEmits      `yaml:"emits"`
	ConfigKeys      []string           `yaml:"config_keys"`
}

// ManifestSubcmd declares one sub-command the scanner accepts.
type ManifestSubcmd struct {
	ID          string         `yaml:"id"`
	Description string         `yaml:"description"`
	Flags       []ManifestFlag `yaml:"flags"`
}

// ManifestFlag declares a typed flag for a sub-command. Type is one of
// "bool", "string", "int"; loader rejects others.
type ManifestFlag struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Default     any    `yaml:"default"`
	Description string `yaml:"description"`
}

// ManifestEmits declares the node types and edge kinds the scanner
// writes. Today consumed by `hero domain show <name>`; future tooling
// (graph-aware lint, dashboard view registry) reads it too.
type ManifestEmits struct {
	NodeTypes []string `yaml:"node_types"`
	EdgeKinds []string `yaml:"edge_kinds"`
}

// ParseManifest parses a scan-manifest.yaml document. Returns a clear
// error pointing at the offending field for malformed YAML; this is
// the user-facing failure mode when a pack ships a typo'd manifest.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid scan-manifest.yaml: %w", err)
	}
	if m.ScannerID == "" {
		return nil, fmt.Errorf("scan-manifest.yaml: scanner_id is required")
	}
	if m.ManifestVersion != "" && m.ManifestVersion != "1" {
		return nil, fmt.Errorf("scan-manifest.yaml: unsupported manifest_version %q (expected \"1\")", m.ManifestVersion)
	}
	for _, sub := range m.Subcommands {
		if sub.ID == "" {
			return nil, fmt.Errorf("scan-manifest.yaml: subcommand id is required")
		}
		for _, f := range sub.Flags {
			if f.Name == "" {
				return nil, fmt.Errorf("scan-manifest.yaml: flag name required under subcommand %q", sub.ID)
			}
			switch f.Type {
			case "bool", "string", "int":
				// ok
			case "":
				return nil, fmt.Errorf("scan-manifest.yaml: flag %q under subcommand %q missing type", f.Name, sub.ID)
			default:
				return nil, fmt.Errorf("scan-manifest.yaml: flag %q under subcommand %q has unsupported type %q (want bool|string|int)", f.Name, sub.ID, f.Type)
			}
		}
	}
	return &m, nil
}

// HasSubcommand reports whether the manifest declares a sub-command
// with the given id.
func (m *Manifest) HasSubcommand(id string) bool {
	if m == nil {
		return false
	}
	for _, s := range m.Subcommands {
		if s.ID == id {
			return true
		}
	}
	return false
}
