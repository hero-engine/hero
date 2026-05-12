package serve

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// Summary builders for two-tier MCP responses. Each function returns
// a single-line essence for the [hero envelope] header. Keep these
// aggressively short — the win is in being terse.

// buildSpecSummary returns a 1-2 sentence essence for a spec read.
// Format: "<title> (type=<type>, status=<status>) — <first prose>".
func buildSpecSummary(title, specType, status, body string) string {
	first := firstProseSentence(body)
	parts := []string{}
	if title != "" {
		parts = append(parts, title)
	}
	meta := []string{}
	if specType != "" {
		meta = append(meta, "type="+specType)
	}
	if status != "" {
		meta = append(meta, "status="+status)
	}
	if len(meta) > 0 {
		parts = append(parts, "("+strings.Join(meta, ", ")+")")
	}
	if first != "" {
		parts = append(parts, "— "+first)
	}
	return strings.Join(parts, " ")
}

// firstProseSentence returns the first non-empty sentence outside
// frontmatter, headers, and list bullets. Used to give a spec summary
// some flavour beyond the title.
func firstProseSentence(md string) string {
	lines := strings.Split(md, "\n")
	inFrontmatter := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if i == 0 && t == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if t == "---" {
				inFrontmatter = false
			}
			continue
		}
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "-") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "```") {
			continue
		}
		// First prose line — clip at first sentence boundary.
		if idx := strings.IndexAny(t, ".!?"); idx > 0 && idx < len(t)-1 {
			t = t[:idx+1]
		}
		// Cap length to keep envelopes small.
		if len(t) > 240 {
			t = t[:237] + "..."
		}
		return t
	}
	return ""
}

// fingerprintFile returns a short content+mtime fingerprint for a
// file path. Used to detect staleness on hero_expand re-fetch.
func fingerprintFile(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", path, info.Size(), info.ModTime().UnixNano())))
	return hex.EncodeToString(h[:8])
}

// fingerprintArgs hashes a stable representation of source args. Used
// for query-kind refs where the "source" is the original tool args
// rather than a file.
func fingerprintArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	// stable order
	for i := range keys {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%v|", k, args[k])
	}
	h := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(h[:8])
}

// argHash returns a stable short hash of the argument map suitable
// for use as the slug component of a session-scoped ref ID.
func argHash(args map[string]any) string {
	fp := fingerprintArgs(args)
	if fp == "" {
		return "noargs"
	}
	return fp
}
