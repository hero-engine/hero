package drive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// PromoteThreshold (K) is how many consecutive approved-unchanged outcomes
// promote a pause-category to auto-proceed. Conservative by default.
const PromoteThreshold = 3

// Outcome classifies how a human resolved a pause.
const (
	OutcomeApproved   = "approved"   // proceeded as recommended, no change
	OutcomeEdited     = "edited"     // changed the spec/plan before continuing
	OutcomeRedirected = "redirected" // chose a different option / stopped
)

// CategoryTrust is the learned state for one pause-category for one user.
type CategoryTrust struct {
	Streak   int  `json:"streak"`
	Promoted bool `json:"promoted"`
}

// Promotions is the per-user learned autonomy state: which pause-categories
// the user has rubber-stamped enough to auto-proceed. It is the durable,
// inspectable record of the rubber-stamp learning, at
// `.hero/drive/trust/<user>.json`.
type Promotions struct {
	User       string                    `json:"user"`
	Categories map[string]*CategoryTrust `json:"categories,omitempty"`

	path string
}

// TrustPath returns the on-disk path for a user's promotions.
func TrustPath(heroDir, user string) string {
	return filepath.Join(heroDir, "drive", "trust", user+".json")
}

// LoadPromotions reads a user's promotions, returning an empty (usable) set
// when none exists.
func LoadPromotions(heroDir, user string) (*Promotions, error) {
	p := &Promotions{User: user, Categories: map[string]*CategoryTrust{}, path: TrustPath(heroDir, user)}
	data, err := os.ReadFile(p.path)
	if os.IsNotExist(err) {
		return p, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("parsing promotions %s: %w", p.path, err)
	}
	if p.Categories == nil {
		p.Categories = map[string]*CategoryTrust{}
	}
	p.path = TrustPath(heroDir, user)
	return p, nil
}

// Save writes the promotions to disk.
func (p *Promotions) Save() error {
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.path, append(data, '\n'), 0o644)
}

// RecordOutcome advances or resets the streak for a category. Only Promotable
// categories are ever tracked — the hard-pause guardrails (Irreversible,
// HardCap, Unknown, VerifyStuck, Blocked) are never promoted.
func (p *Promotions) RecordOutcome(cat PauseCategory, outcome string) {
	if !cat.Promotable() {
		return
	}
	key := string(cat)
	t := p.Categories[key]
	if t == nil {
		t = &CategoryTrust{}
		p.Categories[key] = t
	}
	switch outcome {
	case OutcomeApproved:
		t.Streak++
		if t.Streak >= PromoteThreshold {
			t.Promoted = true
		}
	case OutcomeEdited, OutcomeRedirected:
		t.Streak = 0
		t.Promoted = false
	}
}

// IsPromoted reports whether a category should auto-proceed for this user.
// Always false for non-Promotable categories, regardless of stored state.
func (p *Promotions) IsPromoted(cat PauseCategory) bool {
	if !cat.Promotable() {
		return false
	}
	t := p.Categories[string(cat)]
	return t != nil && t.Promoted
}

// Reset clears the learned state for a category (re-enables pausing).
func (p *Promotions) Reset(cat PauseCategory) {
	delete(p.Categories, string(cat))
}

// PromotedList returns the categories currently promoted, sorted.
func (p *Promotions) PromotedList() []string {
	var out []string
	for k, t := range p.Categories {
		if t.Promoted {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
