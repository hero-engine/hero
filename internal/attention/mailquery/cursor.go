package mailquery

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/hero-engine/hero/contracts/attention/mailthread"
)

const cursorVersion = 1

type cursorFilters struct {
	ProjectPeerID string `json:"project_peer_id,omitempty"`
	ThreadID      string `json:"thread_id,omitempty"`
	UnreadOnly    *bool  `json:"unread_only,omitempty"`
}

type threadCursorFilters struct {
	ProjectPeerID string               `json:"project_peer_id,omitempty"`
	Bucket        mailthread.Bucket    `json:"bucket,omitempty"`
	Lifecycle     mailthread.Lifecycle `json:"lifecycle,omitempty"`
}

type threadCursor struct {
	Version    int                 `json:"version"`
	Filters    threadCursorFilters `json:"filters"`
	Revision   string              `json:"revision"`
	ActivityAt string              `json:"activity_at"`
	PeerID     string              `json:"peer_id"`
	ThreadID   string              `json:"thread_id"`
}

func encodeThreadCursor(cursor threadCursor) (string, error) {
	cursor.Version = cursorVersion
	b, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeThreadCursor(value string) (threadCursor, error) {
	var cursor threadCursor
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || json.Unmarshal(b, &cursor) != nil || cursor.Version != cursorVersion || cursor.Revision == "" || cursor.ActivityAt == "" || cursor.PeerID == "" || cursor.ThreadID == "" {
		return cursor, errors.New("cursor is invalid")
	}
	return cursor, nil
}

type messageCursor struct {
	Version    int           `json:"version"`
	Filters    cursorFilters `json:"filters"`
	Revision   string        `json:"revision"`
	ActivityAt string        `json:"activity_at"`
	PeerID     string        `json:"peer_id"`
	MessageID  string        `json:"message_id"`
}

func encodeCursor(cursor messageCursor) (string, error) {
	cursor.Version = cursorVersion
	b, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeCursor(value string) (messageCursor, error) {
	var cursor messageCursor
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cursor, errors.New("cursor is invalid")
	}
	if err := json.Unmarshal(b, &cursor); err != nil {
		return cursor, errors.New("cursor is invalid")
	}
	if cursor.Version != cursorVersion || cursor.Revision == "" || cursor.ActivityAt == "" || cursor.PeerID == "" || cursor.MessageID == "" {
		return cursor, errors.New("cursor is invalid")
	}
	return cursor, nil
}

func sameFilters(a, b cursorFilters) bool {
	if a.ProjectPeerID != b.ProjectPeerID || a.ThreadID != b.ThreadID {
		return false
	}
	if a.UnreadOnly == nil || b.UnreadOnly == nil {
		return a.UnreadOnly == nil && b.UnreadOnly == nil
	}
	return *a.UnreadOnly == *b.UnreadOnly
}
