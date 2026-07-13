package graph

import (
	"os"
	"strings"
	"testing"
)

// TestSchemaLess guards the numeric-aware compare that replaced the
// lexical string comparison. The double-digit cases are the ones a
// lexical compare gets wrong ("10" < "9" lexically).
func TestSchemaLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"2", "4", true},
		{"4", "2", false},
		{"4", "4", false},
		{"9", "10", true},  // lexical compare returns false here — the bug
		{"10", "9", false}, // lexical compare returns true here — the bug
		{"10", "11", true},
	}
	for _, c := range cases {
		if got := schemaLess(c.a, c.b); got != c.want {
			t.Errorf("schemaLess(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// TestCheckSchemaMismatch covers the two mismatch branches plus the
// double-digit guard. Direction semantics: the graph being NEWER than
// the binary is tolerated (warn + continue, additive migrations make the
// extra columns harmless); the binary being NEWER than the graph is a
// hard error. Both messages must self-locate (os.Executable()), print
// both schemas, point at `hero doctor`, and NOT offer `hero upgrade` as
// the remediation.
func TestCheckSchemaMismatch(t *testing.T) {
	exe, _ := os.Executable()
	if exe == "" {
		exe = "unknown"
	}

	assertSelfLocating := func(t *testing.T, msg, binarySchema, graphSchema string) {
		t.Helper()
		for _, want := range []string{exe, binarySchema, graphSchema, "hero doctor"} {
			if !strings.Contains(msg, want) {
				t.Errorf("message missing %q:\n%s", want, msg)
			}
		}
		if !strings.Contains(msg, "will NOT help") {
			t.Errorf("message should note `hero upgrade` will NOT help:\n%s", msg)
		}
		if strings.Contains(msg, "run `hero upgrade`") {
			t.Errorf("message must not recommend `hero upgrade` as the fix:\n%s", msg)
		}
	}

	t.Run("equal is silent", func(t *testing.T) {
		warning, err := checkSchemaMismatch("4", "4")
		if warning != "" || err != nil {
			t.Fatalf("equal schemas: warning=%q err=%v", warning, err)
		}
	})

	t.Run("graph newer than binary warns and continues", func(t *testing.T) {
		warning, err := checkSchemaMismatch("2", "4")
		if err != nil {
			t.Fatalf("graph-newer must not error, got: %v", err)
		}
		if warning == "" {
			t.Fatal("graph-newer must produce a warning")
		}
		assertSelfLocating(t, warning, "2", "4")
	})

	t.Run("double-digit graph newer takes warn branch", func(t *testing.T) {
		// binary schema 9, graph schema 10 — a lexical compare would send
		// this to the hard-error branch. It must warn-and-continue.
		warning, err := checkSchemaMismatch("9", "10")
		if err != nil {
			t.Fatalf("9 vs 10 must not error, got: %v", err)
		}
		if warning == "" {
			t.Fatal("9 vs 10 must produce a warning (graph is newer)")
		}
		assertSelfLocating(t, warning, "9", "10")
	})

	t.Run("binary newer than graph is a hard error", func(t *testing.T) {
		warning, err := checkSchemaMismatch("4", "2")
		if err == nil {
			t.Fatal("binary-newer must error")
		}
		if warning != "" {
			t.Fatalf("hard-error branch must not also warn, got: %q", warning)
		}
		assertSelfLocating(t, err.Error(), "4", "2")
	})
}
