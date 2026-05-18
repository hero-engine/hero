package chat

import "testing"

func TestResolveCapability(t *testing.T) {
	hc := &fakeAdapter{name: "hero-code", kinds: []Kind{KindInteractive, KindHeadless}}
	bridge := &fakeAdapter{name: "claude-code-bridge", kinds: []Kind{KindInteractive}}

	tests := []struct {
		name           string
		setup          func(r *Registry)
		pref           string
		wantInteractor string // adapter Name expected at the chosen Interactive id, "" for empty
		wantHeadless   string // same for Headless
	}{
		{
			name:           "no adapters",
			setup:          func(r *Registry) {},
			wantInteractor: "",
			wantHeadless:   "",
		},
		{
			name: "hero-code only",
			setup: func(r *Registry) {
				_ = r.Register("hc", hc)
			},
			wantInteractor: "hero-code",
			wantHeadless:   "hero-code",
		},
		{
			name: "ide bridge only",
			setup: func(r *Registry) {
				_ = r.Register("bridge", bridge)
			},
			wantInteractor: "claude-code-bridge",
			wantHeadless:   "", // bridges never receive headless
		},
		{
			name: "both, no pref defaults to hero-code",
			setup: func(r *Registry) {
				_ = r.Register("bridge", bridge)
				_ = r.Register("hc", hc)
			},
			wantInteractor: "hero-code",
			wantHeadless:   "hero-code",
		},
		{
			name: "both, user prefers bridge",
			setup: func(r *Registry) {
				_ = r.Register("bridge", bridge)
				_ = r.Register("hc", hc)
			},
			pref:           "claude-code-bridge",
			wantInteractor: "claude-code-bridge",
			wantHeadless:   "hero-code",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			tc.setup(r)
			cap := Resolve(r, tc.pref)
			if got := nameOf(r, cap.Interactive); got != tc.wantInteractor {
				t.Errorf("interactive = %q, want %q", got, tc.wantInteractor)
			}
			if got := nameOf(r, cap.Headless); got != tc.wantHeadless {
				t.Errorf("headless = %q, want %q", got, tc.wantHeadless)
			}
		})
	}
}

func nameOf(r *Registry, id string) string {
	if id == "" {
		return ""
	}
	a := r.Get(id)
	if a == nil {
		return ""
	}
	return a.Name()
}
