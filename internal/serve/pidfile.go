package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PIDInfo is the JSON shape written to the PID file. Keeping these
// fields plus a `version` lets `hero serve status` print something
// useful even when the daemon is wedged and not answering HTTP.
type PIDInfo struct {
	PID       int       `json:"pid"`
	Port      int       `json:"port"`
	StartedAt time.Time `json:"started_at"`
	Version   string    `json:"version"`
}

// DefaultPort is the daemon's default listening port. Sharing the
// constant keeps the PID-file naming rule (bare filename for default,
// suffix for non-default) consistent across server, CLI, and tests.
const DefaultPort = 7437

// PIDFilePath returns the canonical PID file path for a port. The
// default port uses the bare ~/.hero/serve.pid filename; non-default
// ports get ~/.hero/serve-<port>.pid so multiple daemons on different
// ports can coexist.
func PIDFilePath(port int) (string, error) {
	dir, err := registryDir()
	if err != nil {
		return "", err
	}
	if port == 0 || port == DefaultPort {
		return filepath.Join(dir, "serve.pid"), nil
	}
	return filepath.Join(dir, fmt.Sprintf("serve-%d.pid", port)), nil
}

// WritePIDFile writes a JSON PID file at the canonical path for port.
// Creates the parent directory if missing. Caller is responsible for
// removing the file at shutdown.
func WritePIDFile(port int, version string) (string, error) {
	path, err := PIDFilePath(port)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("creating pid file directory: %w", err)
	}
	info := PIDInfo{
		PID:       os.Getpid(),
		Port:      port,
		StartedAt: time.Now().UTC(),
		Version:   version,
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling pid file: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("writing pid file %s: %w", path, err)
	}
	return path, nil
}

// ReadPIDFile reads and parses the PID file for the given port. Returns
// (nil, nil) when the file does not exist — callers treat "no file" as
// "no daemon running" rather than an error.
func ReadPIDFile(port int) (*PIDInfo, string, error) {
	path, err := PIDFilePath(port)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, path, nil
		}
		return nil, path, fmt.Errorf("reading pid file %s: %w", path, err)
	}
	var info PIDInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, path, fmt.Errorf("parsing pid file %s: %w", path, err)
	}
	return &info, path, nil
}

// RemovePIDFile deletes the PID file for the given port. Tolerates
// already-removed files.
func RemovePIDFile(port int) error {
	path, err := PIDFilePath(port)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing pid file %s: %w", path, err)
	}
	return nil
}
