package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// AC-8
func TestLoadWarnsOnceForIgnoredPeeringSubagent(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heroDir, "hero.json"), []byte(`{
  "folder": ".hero",
  "peering": {"subagent": {"command": "must-not-run"}}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	legacySubagentWarning = sync.Once{}
	old := os.Stderr
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = write
	defer func() { os.Stderr = old }()
	if _, err := Load(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatal(err)
	}
	_ = write.Close()
	data, _ := io.ReadAll(read)
	if strings.Count(string(data), "peering.subagent is deprecated and ignored") != 1 {
		t.Fatalf("warning output = %q", data)
	}
}
