// Package session manages per-user shell state for hero serve: which
// home a user last visited, and which sub-nav tab they last picked
// inside any sub-nav-bearing home. Backed by a small SQLite database
// at ~/.hero/shell-sessions.db so it stays isolated from per-project
// state and survives across workspaces.
//
// The store is intentionally tiny. If multi-tenant cloud needs grow
// past it, swap the backend — the public API is the contract.
package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store is the per-user shell state backing store.
type Store struct {
	db *sql.DB
	mu sync.Mutex // serialize writes
}

// Open opens (and migrates) the shell-sessions database. If path is
// empty, defaults to ~/.hero/shell-sessions.db.
func Open(path string) (*Store, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("locate home dir: %w", err)
		}
		dir := filepath.Join(home, ".hero")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create %s: %w", dir, err)
		}
		path = filepath.Join(dir, "shell-sessions.db")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the backing database.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// LastHome returns the slug of the home the user last visited. ok is
// false if no record exists.
func (s *Store) LastHome(userID string) (slug string, ok bool) {
	if s == nil || s.db == nil || userID == "" {
		return "", false
	}
	row := s.db.QueryRow(`SELECT last_home FROM shell_sessions WHERE user_id = ?`, userID)
	if err := row.Scan(&slug); err != nil {
		return "", false
	}
	return slug, true
}

// SetLastHome records the home the user just visited.
func (s *Store) SetLastHome(userID, slug string) error {
	if s == nil || s.db == nil {
		return nil
	}
	if userID == "" || slug == "" {
		return fmt.Errorf("user_id and slug required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT INTO shell_sessions (user_id, last_home, home_tab_state, updated_at)
		VALUES (?, ?, '{}', ?)
		ON CONFLICT(user_id) DO UPDATE SET
			last_home = excluded.last_home,
			updated_at = excluded.updated_at
	`, userID, slug, time.Now().UnixMilli())
	return err
}

// TabState returns the sub-nav tab slug the user last selected for the
// given home. ok is false if no record exists.
func (s *Store) TabState(userID, home string) (tab string, ok bool) {
	if s == nil || s.db == nil || userID == "" || home == "" {
		return "", false
	}
	var raw string
	row := s.db.QueryRow(`SELECT home_tab_state FROM shell_sessions WHERE user_id = ?`, userID)
	if err := row.Scan(&raw); err != nil {
		return "", false
	}
	if raw == "" {
		return "", false
	}
	state := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return "", false
	}
	t, ok := state[home]
	return t, ok && t != ""
}

// SetTabState records the sub-nav tab the user just selected for the
// given home.
func (s *Store) SetTabState(userID, home, tab string) error {
	if s == nil || s.db == nil {
		return nil
	}
	if userID == "" || home == "" {
		return fmt.Errorf("user_id and home required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read existing state (if any), update the one home's slot, write back.
	state := map[string]string{}
	var raw string
	row := s.db.QueryRow(`SELECT home_tab_state FROM shell_sessions WHERE user_id = ?`, userID)
	if err := row.Scan(&raw); err == nil && raw != "" {
		_ = json.Unmarshal([]byte(raw), &state)
	}
	if tab == "" {
		delete(state, home)
	} else {
		state[home] = tab
	}
	out, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal tab state: %w", err)
	}
	_, err = s.db.Exec(`
		INSERT INTO shell_sessions (user_id, last_home, home_tab_state, updated_at)
		VALUES (?, '', ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			home_tab_state = excluded.home_tab_state,
			updated_at = excluded.updated_at
	`, userID, string(out), time.Now().UnixMilli())
	return err
}

// UserID resolves the active user from an HTTP request. For local
// edition, this is the OS user. Team / cloud editions overlay an auth
// cookie that takes precedence when present.
//
// The shell calls this on every request. A stable non-empty value is
// what matters; the value's semantics are owned by whatever auth layer
// produced it.
func UserID(r *http.Request) string {
	if r != nil {
		if c, err := r.Cookie("hero_user"); err == nil && c.Value != "" {
			return c.Value
		}
		if h := r.Header.Get("X-Hero-User"); h != "" {
			return h
		}
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	return "local"
}

const schema = `
CREATE TABLE IF NOT EXISTS shell_sessions (
	user_id        TEXT PRIMARY KEY,
	last_home      TEXT NOT NULL DEFAULT '',
	home_tab_state TEXT NOT NULL DEFAULT '{}',
	updated_at     INTEGER NOT NULL
);
`
