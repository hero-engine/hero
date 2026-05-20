package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// HealthInputs is the per-request input bundle for the Health section.
type HealthInputs struct {
	HeroDir string
}

// Health is what the partial renders. Phase 1 reads a cached artifact
// only — no live `hero check` invocation. CapturedAt zero renders
// "as of: never".
type Health struct {
	CapturedAt       time.Time
	CapturedAtPretty string
	HasArtifact      bool
	AllClear         bool
	Rows             []HealthRow
}

// HealthRow is one row in the read-out. Status: "pass" | "warn" |
// "fail" | "info".
type HealthRow struct {
	Name    string
	Status  string
	Message string
}

// cachedHealthArtifact is the on-disk JSON shape we look for under
// .hero/cache/. Schema is deliberately minimal: a captured-at and a
// list of rows. Phase 5 owns producing this file; Phase 1 only reads
// whatever happens to be there.
type cachedHealthArtifact struct {
	CapturedAt time.Time         `json:"captured_at"`
	Rows       []cachedHealthRow `json:"rows"`
}

type cachedHealthRow struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// healthArtifactPath is the conventional location for the cached
// artifact (under .hero/cache/health.json). Exposed as a var so a
// later phase can override without touching callers.
var healthArtifactPath = func(heroDir string) string {
	return filepath.Join(heroDir, "cache", "health.json")
}

// LoadHealth reads the cached artifact (if present). Missing file is
// not an error — the section degrades to "as of: never".
func LoadHealth(in HealthInputs) Health {
	if in.HeroDir == "" {
		return Health{}
	}
	data, err := os.ReadFile(healthArtifactPath(in.HeroDir))
	if err != nil {
		return Health{}
	}
	var art cachedHealthArtifact
	if err := json.Unmarshal(data, &art); err != nil {
		return Health{}
	}
	rows := make([]HealthRow, 0, len(art.Rows))
	allClear := true
	for _, r := range art.Rows {
		if r.Status != "pass" && r.Status != "" {
			allClear = false
		}
		rows = append(rows, HealthRow{
			Name: r.Name, Status: r.Status, Message: r.Message,
		})
	}
	out := Health{
		HasArtifact: true,
		CapturedAt:  art.CapturedAt,
		Rows:        rows,
		AllClear:    allClear && len(rows) > 0,
	}
	if !art.CapturedAt.IsZero() {
		out.CapturedAtPretty = art.CapturedAt.Format("2006-01-02 15:04")
	}
	return out
}
