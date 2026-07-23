package peering

import (
	"os"
	"path/filepath"
	"testing"
)

// AC-6, AC-7
func TestLegacyTrailAndReceivedFromRemainReadable(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "spec.md")
	content := `---
title: legacy
type: feature
status: handed_back
received_from:
  peer_id: old-peer
  originator_slug: old-work
---

# legacy

## Handoff Trail

` + "```yaml\n" + `- at: 2026-05-15T14:00:00Z
  direction: in
  peer_id: old-peer
  mode: handed-back
  originating_spec: old-work
  result_ref: abc123
` + "```\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadTrail(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ResultRef != "abc123" {
		t.Fatalf("legacy trail changed: %+v", entries)
	}
}
