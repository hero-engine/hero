package data

import (
	"testing"
	"time"
)

func TestLoadRegistry_HappyPath(t *testing.T) {
	at := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	out := LoadRegistry(RegistryInputs{
		Slug:  "myproj",
		Entry: &RegistryEntryView{Path: "/abs/myproj", RegisteredAt: at},
	})
	if !out.Registered {
		t.Fatal("expected Registered=true")
	}
	if out.Path != "/abs/myproj" {
		t.Errorf("Path = %q, want /abs/myproj", out.Path)
	}
	if out.RegisteredAtPretty == "" {
		t.Error("expected RegisteredAtPretty non-empty")
	}
}

func TestLoadRegistry_NotRegistered(t *testing.T) {
	out := LoadRegistry(RegistryInputs{Slug: "ghost", Entry: nil})
	if out.Registered {
		t.Fatal("expected Registered=false on nil Entry")
	}
	if out.Slug != "ghost" {
		t.Errorf("Slug = %q, want ghost", out.Slug)
	}
	if out.Path != "" {
		t.Errorf("Path should be empty, got %q", out.Path)
	}
}
