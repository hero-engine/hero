// Package nextdoc parses .hero/NEXT.md and writes its session-scoped
// signals into the knowledge graph.
//
// NEXT.md is a per-session projection. The "Tried and failed" section
// is the most graph-worthy: each bullet is an Attempt — something the
// agent tried and learned from — that should be visible to future
// sessions and teammates so the same dead end isn't walked twice.
package nextdoc

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// WriteGraph reads .hero/NEXT.md and emits Attempt + Session nodes.
//
// The NEXT.md frontmatter `session:` field becomes a Session node
// (or links to an existing one with the same id). Each bullet under
// "## Tried and failed" becomes an Attempt node with edge
// `attempted_in` → Session.
//
// repoKey stamps the partition column. Sessions and Attempts always
// belong to the repo whose .hero/NEXT.md they came from.
//
// If NEXT.md is missing, this is a no-op. If "Tried and failed" is
// empty or absent, no Attempt nodes are written.
func WriteGraph(heroDir, repoKey string, store *graph.Store) (*Summary, error) {
	path := filepath.Join(heroDir, "NEXT.md")
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Summary{}, nil
		}
		return nil, fmt.Errorf("read NEXT.md: %w", err)
	}

	parsed, err := parseNext(string(bytes))
	if err != nil {
		return nil, err
	}
	if parsed.session == "" && len(parsed.attempts) == 0 {
		return &Summary{}, nil
	}

	source := map[string]any{"kind": "next-md", "path": "NEXT.md"}
	summary := &Summary{}

	var sessionID int64
	if parsed.session != "" {
		props := map[string]any{}
		if parsed.updated != "" {
			props["updated"] = parsed.updated
		}
		if parsed.branch != "" {
			props["branch"] = parsed.branch
		}
		props["from_next_md"] = true
		id, err := store.UpsertNode(&graph.Node{
			Type:        "Session",
			Domain:      "engineering",
			Key:         parsed.session,
			Props:       props,
			Repo:        repoKey,
			ContentHash: shortHash("session", parsed.session, parsed.updated, parsed.branch),
			Source:      source,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert Session: %w", err)
		}
		sessionID = id
		summary.Sessions++
	}

	for _, body := range parsed.attempts {
		key := parsed.session + ":" + shortHash("attempt", body)
		id, err := store.UpsertNode(&graph.Node{
			Type:        "Attempt",
			Domain:      "engineering",
			Key:         key,
			Props:       map[string]any{"body": body, "outcome": "failed"},
			Repo:        repoKey,
			ContentHash: shortHash("attempt-body", body),
			Source:      source,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert Attempt: %w", err)
		}
		summary.Attempts++

		if sessionID != 0 {
			if _, err := store.UpsertEdge(&graph.Edge{
				FromID: id, ToID: sessionID, Type: "attempted_in",
				Repo:   repoKey,
				Source: source,
			}); err != nil {
				return nil, fmt.Errorf("upsert attempted_in edge: %w", err)
			}
			summary.Edges++
		}
	}

	return summary, nil
}

// Summary reports counts written by WriteGraph.
type Summary struct {
	Sessions int
	Attempts int
	Edges    int
}

type parsed struct {
	session  string
	updated  string
	branch   string
	attempts []string
}

func parseNext(s string) (parsed, error) {
	out := parsed{}
	rest := s

	// Frontmatter.
	if strings.HasPrefix(rest, "---\n") {
		end := strings.Index(rest[4:], "\n---")
		if end < 0 {
			return out, errors.New("NEXT.md frontmatter never closes")
		}
		fm := rest[4 : 4+end]
		for _, line := range strings.Split(fm, "\n") {
			k, v, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			switch k {
			case "session":
				out.session = v
			case "updated":
				out.updated = v
			case "branch":
				out.branch = v
			}
		}
		rest = rest[4+end+len("\n---"):]
	}

	// Find the "Tried and failed" section.
	const heading = "## Tried and failed"
	idx := strings.Index(rest, heading)
	if idx < 0 {
		return out, nil
	}
	body := rest[idx+len(heading):]
	// Trim until next heading or end.
	if next := nextHeadingIndex(body); next >= 0 {
		body = body[:next]
	}

	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line == "Nothing this session." || line == "Nothing." {
			continue
		}
		if !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") {
			continue
		}
		text := strings.TrimSpace(line[2:])
		if text == "" {
			continue
		}
		out.attempts = append(out.attempts, text)
	}
	return out, nil
}

// nextHeadingIndex returns the byte offset of the next markdown
// heading (line starting with "## ") in s, or -1 if none.
func nextHeadingIndex(s string) int {
	for i := 0; i < len(s); {
		// Match start-of-line "## "
		if (i == 0 || s[i-1] == '\n') && i+3 <= len(s) && s[i] == '#' && s[i+1] == '#' && s[i+2] == ' ' {
			return i
		}
		// Advance to next newline
		nl := strings.IndexByte(s[i:], '\n')
		if nl < 0 {
			return -1
		}
		i += nl + 1
	}
	return -1
}

func shortHash(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
