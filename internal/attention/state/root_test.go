package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolvePrecedenceAndLocations(t *testing.T) {
	t.Run("injected root", func(t *testing.T) {
		want := filepath.Join(t.TempDir(), "injected")
		got, err := Resolve(Options{Root: want, XDGStateHome: "/ignored", HomeDir: "/ignored"})
		if err != nil || got != want {
			t.Fatalf("Resolve() = %q, %v; want %q", got, err, want)
		}
	})
	t.Run("xdg", func(t *testing.T) {
		base := t.TempDir()
		got, err := Resolve(Options{XDGStateHome: base, HomeDir: "/ignored"})
		if err != nil || got != filepath.Join(base, "hero") {
			t.Fatalf("Resolve() = %q, %v", got, err)
		}
	})
	t.Run("default home", func(t *testing.T) {
		home := t.TempDir()
		got, err := Resolve(Options{HomeDir: home})
		if err != nil || got != filepath.Join(home, ".local", "state", "hero") {
			t.Fatalf("Resolve() = %q, %v", got, err)
		}
	})
}

func TestResolveRejectsRepositoryAndSensitiveRoots(t *testing.T) {
	base := t.TempDir()
	for name, options := range map[string]Options{
		"project":     {Root: filepath.Join(base, "repo", ".hero", "attention"), ProjectRoot: filepath.Join(base, "repo")},
		"config":      {Root: filepath.Join(base, "config", "attention"), ConfigDir: filepath.Join(base, "config")},
		"credentials": {Root: filepath.Join(base, "credentials", "attention"), CredentialsFile: filepath.Join(base, "credentials", "auth.json")},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Resolve(options)
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("Resolve() error = %v", err)
			}
		})
	}
}

func TestEnsurePrivatePermissionsAndSeparateStores(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	root, err := Ensure(Options{Root: filepath.Join(t.TempDir(), "attention")})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{root, filepath.Join(root, MailDirectory), filepath.Join(root, FocusDirectory)} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Errorf("%s mode = %o", path, got)
		}
	}
	stateFile := filepath.Join(root, MailDirectory, "state.json")
	if err := WriteFile(stateFile, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %o", got)
	}
}
