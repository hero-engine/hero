package workmodel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"

	"github.com/hero-engine/hero/internal/spec"
)

// Revision fingerprints everything an item's rendering depends on: the spec
// file's bytes, the slug:status of every related or declared-child spec, the
// audit report's verdict and mtime, and the derived lane/verify state. It
// changes exactly when one of those does, so clients can cache per item.
func Revision(s *spec.Spec, c *Corpus, derived string) string {
	h := sha256.New()
	if data, err := os.ReadFile(s.Path); err == nil {
		h.Write(data)
	} else {
		h.Write([]byte(s.RawContent))
	}

	related := map[string]bool{}
	for _, r := range s.Relations {
		related[r.Target] = true
	}
	if s.Type == spec.TypeInitiative {
		for _, slug := range spec.DeclaredChildren(s) {
			related[slug] = true
		}
	}
	slugs := make([]string, 0, len(related))
	for slug := range related {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	for _, slug := range slugs {
		status := "missing"
		if t := c.Lookup(slug); t != nil {
			status = string(t.Status)
		}
		fmt.Fprintf(h, "\x00rel:%s:%s", slug, status)
	}

	if audit := spec.FindAuditReport(s); audit.Path != "" {
		fmt.Fprintf(h, "\x00audit:%s", audit.Verdict)
		if info, err := os.Stat(audit.Path); err == nil {
			fmt.Fprintf(h, ":%d", info.ModTime().UnixNano())
		}
	}
	fmt.Fprintf(h, "\x00derived:%s", derived)
	return hex.EncodeToString(h.Sum(nil))[:16]
}
