package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CachedHealth is the minimum shape the Health loader needs from the
// shared health cache. Mirrors healthcache.HealthResult so the cache's
// concrete type satisfies this interface without a circular import.
// Phase 5 of hero-serve-project-section.
type CachedHealth struct {
	Captured  time.Time
	Rows      []HealthRow
	FromDisk  bool
	Timestamp time.Time
	TTL       time.Duration
}

// HealthLookup is the read interface for the in-process health cache.
// Nil-tolerant on the call site — when Deps.HealthCache is nil the
// loader degrades to reading the on-disk artifact directly.
type HealthLookup interface {
	Health(slug string) (CachedHealth, bool)
}

// HealthInputs is the per-request input bundle for the Health section.
type HealthInputs struct {
	HeroDir string
	Slug    string
	// Cache is the optional cache to consult before the on-disk
	// artifact. When nil the loader falls back to reading
	// .hero/cache/health.json directly.
	Cache HealthLookup
}

// Health is what the partial renders. Phase 5 adds in-memory cache
// awareness: HasArtifact is true if either the cache OR the on-disk
// artifact exists; Stale becomes true when the cache entry's age
// exceeds its TTL (and so the renderer should mark the section "stale
// — refresh now"). CapturedAt zero renders "as of: never".
type Health struct {
	CapturedAt       time.Time
	CapturedAtPretty string
	CapturedRelative string
	HasArtifact      bool
	AllClear         bool
	Rows             []HealthRow

	// Stale reports that the cache result is older than its TTL. The
	// page still renders the rows; the head gets a "stale" chip and
	// the "Refresh now" button is offered. Always false when the
	// section was loaded from the on-disk artifact (no in-memory TTL
	// applies).
	Stale bool

	// RefreshAvailable is true when the cache is wired and the
	// "Refresh now" button can POST /api/{slug}/health/refresh.
	RefreshAvailable bool

	// Slug is plumbed through so the template can render the
	// refresh URL without a separate input.
	Slug string
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

// LoadHealth resolves the health snapshot. Resolution order:
//
//  1. If Inputs.Cache is wired AND has a hit for Slug, use that.
//     A cached entry whose age (now − Timestamp) exceeds TTL is
//     marked Stale=true but still rendered (Phase 5 spec: never
//     block the page on a live run).
//  2. Otherwise fall through to reading .hero/cache/health.json from
//     disk — the Phase 1 contract.
//
// Missing file + cache miss → "as of: never".
func LoadHealth(in HealthInputs) Health {
	out := Health{Slug: in.Slug, RefreshAvailable: in.Cache != nil && in.Slug != ""}

	if in.Cache != nil && in.Slug != "" {
		if cached, ok := in.Cache.Health(in.Slug); ok {
			allClear := true
			for _, r := range cached.Rows {
				if r.Status != "pass" && r.Status != "" {
					allClear = false
				}
			}
			out.HasArtifact = true
			out.Rows = cached.Rows
			out.AllClear = allClear && len(cached.Rows) > 0
			out.CapturedAt = cached.Captured
			if !cached.Captured.IsZero() {
				out.CapturedAtPretty = cached.Captured.Format("2006-01-02 15:04")
				out.CapturedRelative = relativeAgo(cached.Captured, time.Now())
			}
			if cached.TTL > 0 && !cached.Timestamp.IsZero() {
				if time.Since(cached.Timestamp) > cached.TTL {
					out.Stale = true
				}
			}
			return out
		}
	}

	if in.HeroDir == "" {
		return out
	}
	data, err := os.ReadFile(healthArtifactPath(in.HeroDir))
	if err != nil {
		return out
	}
	var art cachedHealthArtifact
	if err := json.Unmarshal(data, &art); err != nil {
		return out
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
	out.HasArtifact = true
	out.CapturedAt = art.CapturedAt
	out.Rows = rows
	out.AllClear = allClear && len(rows) > 0
	if !art.CapturedAt.IsZero() {
		out.CapturedAtPretty = art.CapturedAt.Format("2006-01-02 15:04")
		out.CapturedRelative = relativeAgo(art.CapturedAt, time.Now())
	}
	return out
}

// relativeAgo formats t as a relative duration ("3 minutes ago",
// "2 hours ago", "just now"). Used for the "as of" chip in the
// Health/Peers sections.
func relativeAgo(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	if d < 0 {
		return "just now"
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return formatPluralAgo(mins, "minute")
	case d < 24*time.Hour:
		hrs := int(d.Hours())
		if hrs == 1 {
			return "1 hour ago"
		}
		return formatPluralAgo(hrs, "hour")
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return formatPluralAgo(days, "day")
	}
}

func formatPluralAgo(n int, unit string) string {
	// Tiny helper to avoid pulling fmt into the hot path.
	// Used by relativeAgo only.
	return itoa(n) + " " + unit + "s ago"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
