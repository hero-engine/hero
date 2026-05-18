package chat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// noteHandler is the runner-free /note slash. It writes a markdown
// note under .hero/knowledge/notes/<slug>.md with a frontmatter
// header (title, type=note, status=active, created=YYYY-MM-DD) and
// the body taken from the prompt.
//
// Input shape: the slash args are interpreted as "<title>\n<body...>"
// when a newline is present, otherwise the whole string becomes both
// the title and the body. The slug is derived from the title.
func noteHandler(ctx context.Context, req DispatchRequest, out chan<- Event) error {
	args := ""
	if req.Slash != nil {
		args = req.Slash.Args
	}
	if args == "" {
		args = strings.TrimSpace(strings.TrimPrefix(req.Prompt, "/note"))
	}
	args = strings.TrimSpace(args)
	if args == "" {
		out <- ErrorEvent("slash_failed", "/note needs text — e.g. /note title\\nbody", "")
		out <- DoneEvent(0, nil)
		return nil
	}

	title, body, hasNewline := strings.Cut(args, "\n")
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if !hasNewline {
		body = title
	}

	heroDir, err := resolveHeroDir(req.Context.Workspace)
	if err != nil {
		out <- ErrorEvent("slash_failed", err.Error(), "")
		out <- DoneEvent(0, nil)
		return nil
	}

	notesDir := filepath.Join(heroDir, "knowledge", "notes")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		out <- ErrorEvent("slash_failed", fmt.Sprintf("create notes dir: %v", err), "")
		out <- DoneEvent(0, nil)
		return nil
	}

	slug := slugify(title)
	if slug == "" {
		slug = fmt.Sprintf("note-%d", time.Now().Unix())
	}
	path := filepath.Join(notesDir, slug+".md")
	// Avoid clobbering existing notes by appending a timestamp suffix.
	if _, err := os.Stat(path); err == nil {
		slug = fmt.Sprintf("%s-%d", slug, time.Now().Unix())
		path = filepath.Join(notesDir, slug+".md")
	}

	content := fmt.Sprintf(`---
title: %q
type: note
status: active
created: %s
---

%s
`, title, time.Now().UTC().Format("2006-01-02"), body)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		out <- ErrorEvent("slash_failed", fmt.Sprintf("write note: %v", err), "")
		out <- DoneEvent(0, nil)
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- TokenEvent(fmt.Sprintf("Noted. Saved to %s", path)):
	}
	outcome := map[string]interface{}{"file": path, "slug": slug}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- DoneEvent(0, outcome):
	}
	return nil
}

// slugify converts free-form text into a filesystem-friendly slug.
// Lowercase, alphanumerics + dashes only, capped at 60 chars.
func slugify(s string) string {
	var b strings.Builder
	prevDash := true
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case unicode.IsSpace(r), r == '-', r == '_', r == '/', r == '\\':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
		if b.Len() >= 60 {
			break
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}
