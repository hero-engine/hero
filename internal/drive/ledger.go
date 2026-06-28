package drive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PendingPause records the open question a Drive run is waiting on.
type PendingPause struct {
	Spec     string `json:"spec"`
	Category string `json:"category"`
	Reason   string `json:"reason"`
}

// RunLedger is the small on-disk state for one Drive run, at
// `.hero/drive/<initiative>.json`. It exists so `hero goal --check` stays
// stateless about *answers*: the verdict is recomputed from spec status each
// turn, but whether the human has cleared a given pause lives here, and
// travels with commits like other handoff state.
type RunLedger struct {
	Initiative string            `json:"initiative"`
	Answered   map[string]string `json:"answered,omitempty"` // spec slug → answer text
	Pause      *PendingPause     `json:"pause,omitempty"`

	path string
}

// LedgerPath returns the on-disk path for an initiative's run ledger.
func LedgerPath(heroDir, initiative string) string {
	return filepath.Join(heroDir, "drive", initiative+".json")
}

// LoadLedger reads the run ledger, returning an empty (but usable) ledger
// when none exists yet.
func LoadLedger(heroDir, initiative string) (*RunLedger, error) {
	p := LedgerPath(heroDir, initiative)
	l := &RunLedger{Initiative: initiative, Answered: map[string]string{}, path: p}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, l); err != nil {
		return nil, fmt.Errorf("parsing run ledger %s: %w", p, err)
	}
	if l.Answered == nil {
		l.Answered = map[string]string{}
	}
	l.path = p
	return l, nil
}

// Save writes the ledger to disk, creating `.hero/drive/` as needed.
func (l *RunLedger) Save() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, append(data, '\n'), 0o644)
}

// IsAnswered reports whether the human has cleared a pause for this spec.
func (l *RunLedger) IsAnswered(spec string) bool {
	_, ok := l.Answered[spec]
	return ok
}

// RecordAnswer clears the pending pause by recording the human's answer for
// the spec it stopped at (or for an explicitly named spec).
func (l *RunLedger) RecordAnswer(answer string) (string, bool) {
	if l.Pause == nil {
		return "", false
	}
	spec := l.Pause.Spec
	if l.Answered == nil {
		l.Answered = map[string]string{}
	}
	l.Answered[spec] = strings.TrimSpace(answer)
	l.Pause = nil
	return spec, true
}

// SetPause records the open question. No-op if an identical pause is already
// recorded (keeps --check idempotent while a pause is unanswered).
func (l *RunLedger) SetPause(p *PendingPause) {
	l.Pause = p
}

// ClearPause drops any pending pause (used when the run resumes or completes).
func (l *RunLedger) ClearPause() { l.Pause = nil }
