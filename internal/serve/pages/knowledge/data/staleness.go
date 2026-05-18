package data

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// StalenessInputs is the per-request input bundle for the worth-re-
// checking section.
type StalenessInputs struct {
	HeroDir string
}

// contradictionsFile mirrors the on-disk shape of
// `.hero/knowledge/contradictions.json`. Field set intentionally
// minimal — anything we don't parse falls through as a count-only.
type contradictionsFile struct {
	Contradictions []struct {
		Slug    string `json:"slug"`
		Summary string `json:"summary"`
	} `json:"contradictions"`
}

// LoadStaleness reads `.hero/knowledge/contradictions.json` and returns
// the worth-re-checking payload. Available=false when the file is
// absent — the home spec routes a follow-up to
// knowledge-contradiction-detection in that case.
func LoadStaleness(in StalenessInputs) Staleness {
	out := Staleness{}
	if in.HeroDir == "" {
		return out
	}
	path := filepath.Join(in.HeroDir, "knowledge", "contradictions.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var doc contradictionsFile
	if err := json.Unmarshal(b, &doc); err != nil {
		return out
	}
	out.Available = true
	out.Total = len(doc.Contradictions)
	return out
}
