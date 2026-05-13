package install

import (
	"os"
	"sync"
)

// linking.go — symlink-capability probe.
//
// As of the render-direct-install architecture, root install does NOT
// use symlinks for agents/commands/skills. The probe stays because:
//
//   - install-state.json records host capabilities (informational).
//   - The satellite materializer (internal/install/satellite.go) still
//     uses symlinks for monorepo subprojects (a different concern).

var (
	hostSymlinkProbeOnce sync.Once
	hostSymlinkProbeOK   bool
)

// hostSupportsSymlinks attempts a test symlink in a tempdir and reports
// whether the kernel + filesystem will allow it. Result is cached for
// the lifetime of the process.
func hostSupportsSymlinks() bool {
	hostSymlinkProbeOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "hero-symlink-probe-")
		if err != nil {
			hostSymlinkProbeOK = false
			return
		}
		defer os.RemoveAll(tmpDir)
		target := tmpDir + "/target"
		if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
			hostSymlinkProbeOK = false
			return
		}
		link := tmpDir + "/link"
		if err := os.Symlink(target, link); err != nil {
			hostSymlinkProbeOK = false
			return
		}
		hostSymlinkProbeOK = true
	})
	return hostSymlinkProbeOK
}
