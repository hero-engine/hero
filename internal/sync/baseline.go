// Package sync implements the shared-field 3-way merge and the persisted
// last-synced baseline that the merge reconciles against.
//
// The baseline is the common ancestor for a 3-way merge: the last value of
// each synced shared field on which local and tracker were known-equal. It is
// stored per spec at .hero/sync-state/<slug>.json so both the Go CLI (this
// package) and the Swift app (SyncStateStore) can read and write it.
//
// The file format is a cross-repo contract — keep it simple and stable:
//
//	{
//	  "tracker_id": "PROJ-123",
//	  "base": { "title": "...", "body": "...", "tags": ["a", "b"] },
//	  "synced_at": "2026-06-30T12:00:00Z"
//	}
//
// Keys inside "base" are the Swift-side shared-field names (title, body, tags),
// NOT the Go canonical field names (title, description, labels). The mapping
// lives in this package (see baseFieldForCanonical) so the on-disk contract
// stays aligned with FieldOwnership.swift.
package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// StateDirName is the sibling state directory under .hero that holds the
// per-spec last-synced baselines. It travels with the repo like other .hero
// state.
const StateDirName = "sync-state"

// Baseline is the persisted last-synced value baseline for one spec — the
// common ancestor a 3-way merge reconciles against. Serialized to
// .hero/sync-state/<slug>.json.
type Baseline struct {
	TrackerID string          `json:"tracker_id"`
	Base      map[string]Base `json:"base"`
	SyncedAt  string          `json:"synced_at"`
}

// Base is one shared field's last-synced value. Exactly one of Text / Tags is
// meaningful, keyed by the field's shape. title/body are Text; tags are Tags.
// A JSON string decodes into Text; a JSON array decodes into Tags — so the
// on-disk shape mirrors the field's natural type without a discriminator.
type Base struct {
	Text string
	Tags []string
	// isTags records which arm is populated, so a legitimately empty value
	// (a title cleared to "" vs. tags emptied to []) round-trips faithfully.
	isTags bool
}

// TextBase builds a scalar (title/body) baseline value.
func TextBase(s string) Base { return Base{Text: s} }

// TagsBase builds a tag-set baseline value.
func TagsBase(ss []string) Base { return Base{Tags: append([]string(nil), ss...), isTags: true} }

// IsTags reports whether this baseline value is a tag set (vs. scalar text).
func (b Base) IsTags() bool { return b.isTags }

// MarshalJSON encodes a Base as its bare JSON value: a string for text, an
// array for tags. There is no wrapper object — the shape is the discriminator.
func (b Base) MarshalJSON() ([]byte, error) {
	if b.isTags {
		tags := b.Tags
		if tags == nil {
			tags = []string{}
		}
		return json.Marshal(tags)
	}
	return json.Marshal(b.Text)
}

// UnmarshalJSON decodes a bare string into Text or a bare array into Tags.
func (b *Base) UnmarshalJSON(data []byte) error {
	var ss []string
	if err := json.Unmarshal(data, &ss); err == nil {
		b.Tags = ss
		b.isTags = true
		b.Text = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	b.Text = s
	b.isTags = false
	b.Tags = nil
	return nil
}

// StatePath returns the absolute path to a spec's baseline file:
// <heroDir>/sync-state/<slug>.json.
func StatePath(heroDir, slug string) string {
	return filepath.Join(heroDir, StateDirName, slug+".json")
}

// ReadBaseline loads a spec's baseline. A missing file returns (nil, nil) — the
// first-run / no-baseline case the caller adopts remote as base for. Any other
// read/parse error is returned so the caller can degrade to a no-op rather than
// merge against a corrupt base.
func ReadBaseline(heroDir, slug string) (*Baseline, error) {
	path := StatePath(heroDir, slug)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	if b.Base == nil {
		b.Base = map[string]Base{}
	}
	return &b, nil
}

// WriteBaseline persists a spec's baseline atomically (write-temp-then-rename)
// so a crash mid-write never leaves a half-written contract file. SyncedAt is
// stamped to now (RFC3339) when empty. The sync-state dir is created on demand.
func WriteBaseline(heroDir, slug string, b *Baseline) error {
	if b.SyncedAt == "" {
		b.SyncedAt = time.Now().UTC().Format(time.RFC3339)
	}
	dir := filepath.Join(heroDir, StateDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := StatePath(heroDir, slug)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
