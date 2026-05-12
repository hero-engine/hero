package cli

import (
	"testing"
	"time"
)

func TestShouldPull(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"empty cursor pulls", "", true},
		{"unparseable cursor pulls", "garbage", true},
		{"recent timestamp does not pull",
			time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339), false},
		{"stale timestamp pulls",
			time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldPull(tc.in); got != tc.want {
				t.Errorf("shouldPull(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
