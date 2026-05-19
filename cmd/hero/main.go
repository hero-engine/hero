package main

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/cli"

	// Anchor scan-pluggability registration order: each domain pack's
	// scanner registers itself from an init(); a blank-import here
	// ensures the engineering scanner is in the dispatcher map before
	// any Dispatch call is reached. Future packs (pm, qa, ...) line
	// up the same way.
	_ "github.com/hero-engine/hero/domains/engineering/scan"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	cli.SetVersion(version)
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
