package synthesize

import "testing"

func TestNormalizeMode(t *testing.T) {
	cases := map[string]Mode{
		"auto": ModeAuto, "review": ModeReview, "off": ModeOff,
		"": ModeReview, "bogus": ModeReview, // default conservative
	}
	for in, want := range cases {
		if got := NormalizeMode(in); got != want {
			t.Errorf("NormalizeMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAction(t *testing.T) {
	cases := []struct {
		conf float64
		mode Mode
		want string
	}{
		{0.95, ModeOff, "skip"},
		{0.95, ModeReview, "review"},
		{0.95, ModeAuto, "auto"},   // high confidence + auto → auto
		{0.6, ModeAuto, "review"},  // below bar → review even in auto
		{0.9, ModeAuto, "auto"},    // exactly at bar → auto
		{0.6, ModeReview, "review"},
		{0.6, ModeOff, "skip"},
	}
	for _, c := range cases {
		if got := Action(c.conf, c.mode); got != c.want {
			t.Errorf("Action(%.2f, %q) = %q, want %q", c.conf, c.mode, got, c.want)
		}
	}
}
