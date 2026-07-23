// Package state resolves and creates the private user-state directories used
// by durable attention stores. It does not implement either store.
package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	MailDirectory  = "mail"
	FocusDirectory = "focus"
)

type Options struct {
	Root            string
	XDGStateHome    string
	HomeDir         string
	ProjectRoot     string
	ConfigDir       string
	CredentialsFile string
}

func Resolve(options Options) (string, error) {
	root := options.Root
	if root == "" {
		xdg := options.XDGStateHome
		if xdg == "" {
			xdg = os.Getenv("XDG_STATE_HOME")
		}
		if xdg != "" {
			root = filepath.Join(xdg, "hero")
		} else {
			home := options.HomeDir
			if home == "" {
				var err error
				home, err = os.UserHomeDir()
				if err != nil {
					return "", fmt.Errorf("resolve home directory: %w", err)
				}
			}
			root = filepath.Join(home, ".local", "state", "hero")
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve attention state root: %w", err)
	}
	for label, forbidden := range map[string]string{
		"project repository":      options.ProjectRoot,
		"configuration directory": options.ConfigDir,
		"credentials directory":   filepath.Dir(options.CredentialsFile),
	} {
		if forbidden == "" || (label == "credentials directory" && options.CredentialsFile == "") {
			continue
		}
		blocked, err := filepath.Abs(forbidden)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", label, err)
		}
		if within(abs, blocked) {
			return "", fmt.Errorf("attention state root must not be inside %s", label)
		}
	}
	return filepath.Clean(abs), nil
}

func Ensure(options Options) (string, error) {
	root, err := Resolve(options)
	if err != nil {
		return "", err
	}
	for _, dir := range []string{root, filepath.Join(root, MailDirectory), filepath.Join(root, FocusDirectory)} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", fmt.Errorf("create attention state directory %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return "", fmt.Errorf("secure attention state directory %s: %w", dir, err)
		}
	}
	return root, nil
}

func WriteFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func within(path, parent string) bool {
	rel, err := filepath.Rel(parent, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
