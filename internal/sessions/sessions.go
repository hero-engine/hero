package sessions

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Session represents a reasoning log session.
type Session struct {
	ID        string     `json:"id"`
	Name      string     `json:"name,omitempty"`
	Agent     string     `json:"agent,omitempty"`
	Start     time.Time  `json:"start"`
	End       *time.Time `json:"end,omitempty"`
	SpecSlug  string     `json:"spec_slug,omitempty"`
	HeroCalls int        `json:"hero_calls"`
	SpecsDone int        `json:"specs_completed"`
}

// NewID generates a new session ID: 8 random bytes hex-encoded (16 chars).
func NewID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp-based ID
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// SessionsDir returns the path to .hero/sessions/.
func SessionsDir(heroDir string) string {
	return filepath.Join(heroDir, "sessions")
}

// IndexPath returns the path to .hero/sessions/index.jsonl.
func IndexPath(heroDir string) string {
	return filepath.Join(SessionsDir(heroDir), "index.jsonl")
}

// LogPath returns the path to .hero/sessions/<id>.jsonl.
func LogPath(heroDir, id string) string {
	return filepath.Join(SessionsDir(heroDir), id+".jsonl")
}

// Start creates a new session, writes it to the index, and returns it.
func Start(heroDir, name, agent string) (*Session, error) {
	if err := os.MkdirAll(SessionsDir(heroDir), 0o755); err != nil {
		return nil, fmt.Errorf("creating sessions dir: %w", err)
	}

	s := &Session{
		ID:    NewID(),
		Name:  name,
		Agent: agent,
		Start: time.Now(),
	}

	if err := appendIndex(heroDir, s); err != nil {
		return nil, fmt.Errorf("writing session index: %w", err)
	}

	return s, nil
}

// End marks a session as ended and updates the index.
func End(heroDir, id string) error {
	sessions, err := readIndex(heroDir)
	if err != nil {
		return err
	}

	found := false
	now := time.Now()
	for _, s := range sessions {
		if s.ID == id {
			s.End = &now
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("session %q not found", id)
	}

	return writeIndex(heroDir, sessions)
}

// Load loads a session from the index by ID.
// If id is empty, returns the most recent session.
func Load(heroDir, id string) (*Session, error) {
	sessions, err := readIndex(heroDir)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}

	if id == "" {
		// Return most recent (last written = index 0 after sort)
		return sessions[0], nil
	}

	for _, s := range sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, fmt.Errorf("session %q not found", id)
}

// List returns all sessions from the index, newest first.
func List(heroDir string) ([]*Session, error) {
	return readIndex(heroDir)
}

// Prune removes sessions older than retentionDays and their log files.
// Returns the number of sessions pruned.
func Prune(heroDir string, retentionDays int) (int, error) {
	sessions, err := readIndex(heroDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	var keep []*Session
	pruned := 0

	for _, s := range sessions {
		if s.Start.Before(cutoff) {
			// Remove the log file
			logFile := LogPath(heroDir, s.ID)
			_ = os.Remove(logFile)
			pruned++
		} else {
			keep = append(keep, s)
		}
	}

	if pruned == 0 {
		return 0, nil
	}

	if err := writeIndex(heroDir, keep); err != nil {
		return 0, err
	}

	return pruned, nil
}

// appendIndex appends a session to the index JSONL file.
func appendIndex(heroDir string, s *Session) error {
	f, err := os.OpenFile(IndexPath(heroDir), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

// readIndex reads all sessions from the index JSONL, newest first.
func readIndex(heroDir string) ([]*Session, error) {
	f, err := os.Open(IndexPath(heroDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var sessions []*Session
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var s Session
		if err := json.Unmarshal(line, &s); err != nil {
			continue // skip malformed lines
		}
		sessions = append(sessions, &s)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Sort newest first by Start
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Start.After(sessions[j].Start)
	})

	return sessions, nil
}

// writeIndex rewrites the entire index JSONL file.
func writeIndex(heroDir string, sessions []*Session) error {
	if err := os.MkdirAll(SessionsDir(heroDir), 0o755); err != nil {
		return err
	}

	f, err := os.Create(IndexPath(heroDir))
	if err != nil {
		return err
	}
	defer f.Close()

	// Write in chronological order (oldest first) so appends keep working naturally
	ordered := make([]*Session, len(sessions))
	copy(ordered, sessions)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Start.Before(ordered[j].Start)
	})

	for _, s := range ordered {
		data, err := json.Marshal(s)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	return nil
}
