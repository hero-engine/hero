package engineeringscan

import (
	"fmt"

	hero "github.com/hero-engine/hero"
	"github.com/hero-engine/hero/internal/scan"
)

// init() registers the engineering scanner and loads its manifest
// from the embedded pack FS. Per scan-pluggability spec §8 each pack's
// scanner registers itself from its own package's init(); a
// blank-import anchored in cmd/hero/main.go ensures registration runs
// before any Dispatch call.
func init() {
	scan.Register(engineeringScanner{})
	packFS, err := hero.DomainFS("engineering")
	if err != nil {
		// Engineering pack is always embedded — Sub() failing here means
		// the binary is malformed. Panic so the boot failure is loud.
		panic(fmt.Sprintf("engineeringscan: domain FS unavailable: %v", err))
	}
	if err := scan.LoadAndRegisterManifest("engineering", packFS); err != nil {
		panic(fmt.Sprintf("engineeringscan: load manifest: %v", err))
	}
}
