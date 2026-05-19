// Package engineeringscan registers the engineering reference scanner
// with the dispatch shell (internal/scan). It is the seam every other
// pack's scanner mirrors.
//
// Scope of this v1 package: contract surface only. The Scanner
// implementation is a thin wrapper around the existing legacy entry
// points under internal/scan/{scan.go, generate.go, ...} and
// internal/codescan/. Those files have not yet been physically
// relocated under this package — the scan-pluggability spec §8 PR 2
// gates that relocation on a TestScanReferenceParity golden harness
// that compares pre- and post-move output bytewise. Until that PR
// lands, the CLI's existing direct-call path remains the primary
// engineering scan execution route; this package's Scan() exists so
// the dispatch contract is real (PM scanners can plug in alongside)
// and so `hero scan` calls Dispatch first for routing decisions.
package engineeringscan

import (
	"fmt"

	"github.com/hero-engine/hero/internal/scan"
)

const scannerID = "engineering-code-scan"

// engineeringScanner is the registered Scanner. Its Scan method
// returns a minimal Report; the actual engineering scan flow stays in
// internal/cli/scan.go (which the CLI continues to invoke after the
// dispatch decision) until the PR 2 relocation lands.
type engineeringScanner struct{}

func (engineeringScanner) ID() string { return scannerID }

// Scan is intentionally a no-op stub at this milestone. The CLI's
// existing runScan path handles the actual engineering scan; the
// dispatch indirection exists so non-engineering packs can plug in
// their own scanner without touching engineering's logic. When the
// relocation PR lands, the Analyze/Generate/Enrich/Import call tree
// moves into this package and Scan returns a populated Report; the
// CLI shrinks to a thin Dispatch caller.
func (engineeringScanner) Scan(subcommand string, opts scan.ScanOpts) (*scan.Report, error) {
	if subcommand != "scan" {
		return nil, fmt.Errorf("engineering scanner: unsupported subcommand %q", subcommand)
	}
	// Sentinel signalling "engineering pack is registered and routable"
	// without claiming to have done the scan. The CLI's runScan path
	// runs the legacy flow whether this stub is invoked or not (today's
	// behavior is preserved bit-identical — see scan-pluggability spec
	// §8 PR 2 deferred work).
	return &scan.Report{Summary: ""}, nil
}
