package data

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ConfigInputs is the per-request input bundle for the Config section.
//
// Both ProjectRoot and HeroDir are read: hero.json lives under
// `.hero/hero.json` by convention (the hero folder), and the loader
// also surfaces whether hero.local.json (same folder) is present.
type ConfigInputs struct {
	ProjectRoot string
	HeroDir     string
}

// Config is what the partial renders.
type Config struct {
	// HeroJSONPath is the absolute path to the project's hero.json on
	// disk. Empty when the file is missing.
	HeroJSONPath string
	// HeroJSONExists is true when hero.json was found and read OK.
	HeroJSONExists bool
	// HeroLocalJSONExists is true when hero.local.json is also present
	// (read-only — we don't dump its body in case it contains secrets).
	HeroLocalJSONExists bool
	// PrettyJSON is the indented JSON body of hero.json, suitable for
	// display in a <pre> block. Empty when no file or unreadable.
	PrettyJSON string
	// OpenInEditorURL is the file:// link the partial renders.
	OpenInEditorURL string
}

// LoadConfig reads hero.json and hero.local.json into a read-only view.
// hero.local.json is reported as "exists" but its contents are NOT
// rendered — that file is the documented home for secrets.
func LoadConfig(in ConfigInputs) Config {
	heroDir := in.HeroDir
	if heroDir == "" {
		if in.ProjectRoot == "" {
			return Config{}
		}
		heroDir = filepath.Join(in.ProjectRoot, ".hero")
	}
	heroJSON := filepath.Join(heroDir, "hero.json")
	heroLocal := filepath.Join(heroDir, "hero.local.json")
	out := Config{
		HeroJSONPath:    heroJSON,
		OpenInEditorURL: "file://" + heroJSON,
	}
	data, err := os.ReadFile(heroJSON)
	if err != nil {
		return out
	}
	out.HeroJSONExists = true
	if _, lerr := os.Stat(heroLocal); lerr == nil {
		out.HeroLocalJSONExists = true
	}
	// Round-trip through map for stable pretty-printing. Failure
	// keeps PrettyJSON empty rather than 500ing.
	var raw any
	if err := json.Unmarshal(data, &raw); err == nil {
		if buf, err := json.MarshalIndent(raw, "", "  "); err == nil {
			out.PrettyJSON = string(buf)
		}
	}
	if out.PrettyJSON == "" {
		out.PrettyJSON = string(data)
	}
	return out
}
