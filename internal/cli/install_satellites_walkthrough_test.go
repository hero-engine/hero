package cli

import (
	"testing"
)

func TestIsUnder(t *testing.T) {
	cases := []struct {
		child, parent string
		want          bool
	}{
		{"vendor/a", "vendor", true},
		{"vendor", "vendor", true},
		{"vendor/a/b", "vendor", true},
		{"engines/mlx", "vendor", false},
		{"vendoring", "vendor", false}, // prefix without slash boundary
		{"a/b", "a/b/c", false},
	}
	for _, tc := range cases {
		got := isUnder(tc.child, tc.parent)
		if got != tc.want {
			t.Errorf("isUnder(%q, %q) = %v, want %v", tc.child, tc.parent, got, tc.want)
		}
	}
}

func TestIsUnderAny(t *testing.T) {
	parents := []string{"vendor", "engines/legacy"}
	cases := map[string]bool{
		"vendor/lib":         true,
		"engines/legacy/old": true,
		"engines/mlx":        false,
		"app":                false,
		"vendor":             true,
	}
	for child, want := range cases {
		if got := isUnderAny(child, parents); got != want {
			t.Errorf("isUnderAny(%q) = %v, want %v", child, got, want)
		}
	}
}
