package chat

import (
	"context"
	"sync"
	"testing"
)

type fakeAdapter struct {
	name    string
	version string
	kinds   []Kind
}

func (f *fakeAdapter) Name() string    { return f.name }
func (f *fakeAdapter) Version() string { return f.version }
func (f *fakeAdapter) Kinds() []Kind   { return f.kinds }
func (f *fakeAdapter) Close() error    { return nil }
func (f *fakeAdapter) Stream(ctx context.Context, req DispatchRequest) (<-chan Event, error) {
	ch := make(chan Event, 1)
	go func() {
		defer close(ch)
		ch <- DoneEvent(0, nil)
	}()
	return ch, nil
}

func TestRegistryRoundTrip(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("a", &fakeAdapter{name: "hero-code", kinds: []Kind{KindInteractive, KindHeadless}}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(r.All()) != 1 {
		t.Fatalf("expected 1 adapter, got %d", len(r.All()))
	}
	if r.Get("a") == nil {
		t.Fatal("get returned nil for known id")
	}
	r.Deregister("a")
	if len(r.All()) != 0 {
		t.Fatal("expected 0 adapters after deregister")
	}
}

func TestRegistryEmptyIDRejected(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("", &fakeAdapter{name: "x", kinds: []Kind{KindInteractive}}); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestRegistryDuplicateRejected(t *testing.T) {
	r := NewRegistry()
	a := &fakeAdapter{name: "hero-code", kinds: []Kind{KindInteractive}}
	if err := r.Register("a", a); err != nil {
		t.Fatal(err)
	}
	if err := r.Register("a", a); err == nil {
		t.Fatal("expected error for duplicate id")
	}
}

func TestRegistryByKindAndPreference(t *testing.T) {
	r := NewRegistry()
	bridge := &fakeAdapter{name: "claude-code-bridge", kinds: []Kind{KindInteractive}}
	hc := &fakeAdapter{name: "hero-code", kinds: []Kind{KindInteractive, KindHeadless}}
	if err := r.Register("bridge", bridge); err != nil {
		t.Fatal(err)
	}
	if err := r.Register("hc", hc); err != nil {
		t.Fatal(err)
	}
	if got := r.PreferHeroCode(KindInteractive); got == nil || got.Name() != "hero-code" {
		t.Fatalf("PreferHeroCode interactive = %+v, want hero-code", got)
	}
	if got := r.ByKind(KindHeadless); got == nil || got.Name() != "hero-code" {
		t.Fatalf("ByKind headless = %+v, want hero-code", got)
	}
}

func TestRegistryConcurrent(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "a" + string(rune('A'+i%26)) + string(rune('0'+i/26))
			_ = r.Register(id, &fakeAdapter{name: "hero-code", kinds: []Kind{KindInteractive}})
			r.All()
			r.Touch(id)
			r.Deregister(id)
		}(i)
	}
	wg.Wait()
}
