package projectregistry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryAcceptsAndPreservesAutoRegistrationMarker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	if err := os.WriteFile(path, []byte(`{"projects":{"demo":{"path":"/tmp/demo","registered":"auto"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	entry := registry.Get("demo")
	if entry == nil || entry.Path != "/tmp/demo" || !entry.Registered.IsZero() {
		t.Fatalf("entry = %#v", entry)
	}
	if err := registry.Save(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"registered": "auto"`) {
		t.Fatalf("auto marker was not preserved: %s", b)
	}
}
