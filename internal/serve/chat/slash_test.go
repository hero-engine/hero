package chat

import (
	"context"
	"testing"
)

func TestLookupDefaults(t *testing.T) {
	resetSlashesForTest()
	t.Cleanup(resetSlashesForTest)
	for _, name := range []string{"ask", "note", "scheduled", "design", "deliver", "diagnose"} {
		if _, ok := Lookup(name); !ok {
			t.Errorf("expected default slash %q", name)
		}
	}
}

func TestRegisterCollision(t *testing.T) {
	resetSlashesForTest()
	t.Cleanup(resetSlashesForTest)
	if err := Register(Slash{Name: "ask", RunnerFree: true, Handler: askHandler}); err == nil {
		t.Fatal("expected collision error registering existing slash")
	}
}

func TestRegisterRunnerFreeRequiresHandler(t *testing.T) {
	resetSlashesForTest()
	t.Cleanup(resetSlashesForTest)
	if err := Register(Slash{Name: "pm:roadmap", RunnerFree: true}); err == nil {
		t.Fatal("expected error for runner-free slash without handler")
	}
}

func TestRegisterFreshSlash(t *testing.T) {
	resetSlashesForTest()
	t.Cleanup(resetSlashesForTest)
	handler := func(_ context.Context, _ DispatchRequest, out chan<- Event) error {
		out <- DoneEvent(0, nil)
		return nil
	}
	if err := Register(Slash{Name: "pm:roadmap", RunnerFree: true, Handler: handler}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, ok := Lookup("pm:roadmap"); !ok {
		t.Fatal("registered slash not found")
	}
}

func TestAllSorted(t *testing.T) {
	resetSlashesForTest()
	t.Cleanup(resetSlashesForTest)
	all := All()
	for i := 1; i < len(all); i++ {
		if all[i-1].Name > all[i].Name {
			t.Fatalf("All not sorted: %s > %s", all[i-1].Name, all[i].Name)
		}
	}
}

func TestParseSlash(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
		wantArgs string
		wantHit  bool
	}{
		{"/ask what specs do we have", "ask", "what specs do we have", true},
		{"/note", "note", "", true},
		{"  /design auth flow ", "design", "auth flow", true},
		{"hello world", "", "", false},
		{"/pm:roadmap show", "pm:roadmap", "show", true},
		{"   ", "", "", false},
	}
	for _, c := range cases {
		got, _ := ParseSlash(c.in)
		if c.wantHit && got == nil {
			t.Errorf("ParseSlash(%q) = nil, want hit", c.in)
			continue
		}
		if !c.wantHit && got != nil {
			t.Errorf("ParseSlash(%q) = %+v, want miss", c.in, got)
			continue
		}
		if c.wantHit {
			if got.Name != c.wantName {
				t.Errorf("ParseSlash(%q).Name = %q, want %q", c.in, got.Name, c.wantName)
			}
			if got.Args != c.wantArgs {
				t.Errorf("ParseSlash(%q).Args = %q, want %q", c.in, got.Args, c.wantArgs)
			}
		}
	}
}
