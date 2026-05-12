package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchScope(t *testing.T) {
	root := t.TempDir()

	cases := []struct {
		name     string
		cwd      string
		declared []string
		want     string
	}{
		{
			name: "at root",
			cwd:  root,
			want: RootScope,
		},
		{
			name:     "exact match",
			cwd:      filepath.Join(root, "engines", "mlx"),
			declared: []string{"engines/mlx"},
			want:     "engines/mlx",
		},
		{
			name:     "deeper than declared returns declared",
			cwd:      filepath.Join(root, "engines", "mlx", "src", "x"),
			declared: []string{"engines/mlx"},
			want:     "engines/mlx",
		},
		{
			name:     "longest-prefix wins",
			cwd:      filepath.Join(root, "engines", "mlx", "src"),
			declared: []string{"engines", "engines/mlx"},
			want:     "engines/mlx",
		},
		{
			name:     "no match returns root scope",
			cwd:      filepath.Join(root, "other"),
			declared: []string{"engines/mlx"},
			want:     RootScope,
		},
		{
			name:     "trailing slash in declared is normalised",
			cwd:      filepath.Join(root, "app"),
			declared: []string{"app/"},
			want:     "app",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.MkdirAll(tc.cwd, 0o755); err != nil {
				t.Fatal(err)
			}
			got := MatchScope(root, tc.cwd, tc.declared)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestScopeUsesMarker(t *testing.T) {
	ws := &Workspace{
		IsSatellite: true,
		MarkerScope: "marked-scope",
		Root:        "/r",
		CWD:         "/r/x",
	}
	got := ws.Scope([]string{"x"})
	if got != "marked-scope" {
		t.Errorf("expected marker scope to win, got %q", got)
	}
}

func TestScopeFallsBackToDeclared(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "app", "src")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	ws := &Workspace{
		Root: root,
		CWD:  cwd,
	}
	got := ws.Scope([]string{"app"})
	if got != "app" {
		t.Errorf("got %q, want app", got)
	}
}
