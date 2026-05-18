package chat

import (
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// MaxHistoryTurns caps how many turns we return from HistoryByScope.
// Conversations themselves are unbounded; this is a read-side limit.
const MaxHistoryTurns = 50

// Store backs chat conversation + message persistence with SQLite.
//
// Schema is embedded; Open creates the database file if missing,
// migrates idempotently, and is safe for concurrent reads. Writes
// serialize on an internal mutex so SQLite's single-writer model
// doesn't surface as "database is locked" under the chat layer.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// Open opens (and migrates) the chat database. If path is empty,
// defaults to ~/.hero/chat.db so chat state survives across
// workspaces (matching the shell-sessions store layout).
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
		path = filepath.Join(dir, "chat.db")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate chat schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// NewConversation creates a conversation row and returns its id.
func (s *Store) NewConversation(userID, scope string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user_id required")
	}
	id, err := newID()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	_, err = s.db.Exec(`
		INSERT INTO chat_conversations (id, user_id, scope, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, userID, scope, now, now)
	if err != nil {
		return "", fmt.Errorf("insert conversation: %w", err)
	}
	return id, nil
}

// AppendMessage records one message in a conversation and bumps the
// conversation's updated_at.
func (s *Store) AppendMessage(conversationID, role, content string) error {
	if conversationID == "" {
		return fmt.Errorf("conversation_id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO chat_messages (conversation_id, role, content, created_at)
		VALUES (?, ?, ?, ?)
	`, conversationID, role, content, now); err != nil {
		tx.Rollback()
		return fmt.Errorf("insert message: %w", err)
	}
	if _, err := tx.Exec(`UPDATE chat_conversations SET updated_at = ? WHERE id = ?`, now, conversationID); err != nil {
		tx.Rollback()
		return fmt.Errorf("update conversation: %w", err)
	}
	return tx.Commit()
}

// History returns the most recent N turns for one conversation,
// oldest-first. limit <= 0 uses MaxHistoryTurns.
func (s *Store) History(conversationID string, limit int) ([]HistoryTurn, error) {
	if limit <= 0 {
		limit = MaxHistoryTurns
	}
	rows, err := s.db.Query(`
		SELECT role, content
		  FROM chat_messages
		 WHERE conversation_id = ?
		 ORDER BY created_at DESC, id DESC
		 LIMIT ?
	`, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()
	var out []HistoryTurn
	for rows.Next() {
		var t HistoryTurn
		if err := rows.Scan(&t.Role, &t.Content); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	// Reverse to oldest-first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// HistoryByScope returns the most recent N turns across the most
// recently updated conversation for (user, scope). If no conversation
// exists, returns an empty slice with no error.
func (s *Store) HistoryByScope(userID, scope string, limit int) ([]HistoryTurn, error) {
	if userID == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = MaxHistoryTurns
	}
	var convID string
	err := s.db.QueryRow(`
		SELECT id FROM chat_conversations
		 WHERE user_id = ? AND scope = ?
		 ORDER BY updated_at DESC
		 LIMIT 1
	`, userID, scope).Scan(&convID)
	if err == sql.ErrNoRows {
		return []HistoryTurn{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lookup conversation: %w", err)
	}
	return s.History(convID, limit)
}

// LatestConversation returns the id of the most recently updated
// conversation for (user, scope), or "" if none exists.
func (s *Store) LatestConversation(userID, scope string) (string, error) {
	var id string
	err := s.db.QueryRow(`
		SELECT id FROM chat_conversations
		 WHERE user_id = ? AND scope = ?
		 ORDER BY updated_at DESC
		 LIMIT 1
	`, userID, scope).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

// Clear deletes all messages for a conversation. The conversation
// row itself is left in place so future appends keep their FK target.
func (s *Store) Clear(conversationID string) error {
	if conversationID == "" {
		return fmt.Errorf("conversation_id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM chat_messages WHERE conversation_id = ?`, conversationID)
	if err != nil {
		return fmt.Errorf("clear messages: %w", err)
	}
	return nil
}

// SetPreference upserts the user's preferred interactive adapter
// (e.g. "hero-code"). An empty value clears the preference.
func (s *Store) SetPreference(userID, adapter string) error {
	if userID == "" {
		return fmt.Errorf("user_id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`
		INSERT INTO chat_preferences (user_id, interactive_adapter, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			interactive_adapter = excluded.interactive_adapter,
			updated_at = excluded.updated_at
	`, userID, adapter, now)
	return err
}

// Preference returns the user's preferred interactive adapter, or ""
// if unset.
func (s *Store) Preference(userID string) (string, error) {
	var pref string
	err := s.db.QueryRow(`SELECT interactive_adapter FROM chat_preferences WHERE user_id = ?`, userID).Scan(&pref)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return pref, nil
}

// newID mints an opaque conversation id. 16 random bytes hex-encoded
// (32 chars) — short enough for URLs, long enough to dodge collisions.
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
