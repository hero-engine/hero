package scan

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
)

// dispatch.go — pack-scoped scanner dispatch shell.
//
// `hero scan` is a domain-aware command: in an engineering workspace it
// runs language/framework detection and code-symbol ingest; in a PM
// workspace it should import roadmaps, parse tracker epics, ingest OKRs;
// other packs may opt out of scan entirely. The Scanner interface here
// is the seam every pack's scanner implements; Register installs a
// scanner in the dispatcher map (called from each pack's init()); and
// Dispatch picks the active pack's scanner via the loaded manifest and
// runs the requested subcommand.
//
// Scope of this file: contract surface only. The engineering scan
// implementation (language detection, generate, enrich, import, modules)
// continues to live under internal/scan/{scan.go, generate.go, ...}; the
// domains/engineering/scan package wraps those existing entry points
// behind the Scanner interface so callers route through Dispatch
// uniformly. Relocating those files under domains/engineering/scan/ and
// extracting the cross-cutting work-subgraph ingest from internal/cli/
// scan.go is follow-up work gated by TestScanReferenceParity (see
// scan-pluggability spec §8 PR 2).

// ErrScannerNotFound is returned by Dispatch when the active pack ships
// no scanner. Callers should treat this as a friendly skip ("<domain>
// pack does not ship a scanner; nothing to do") rather than an error.
var ErrScannerNotFound = errors.New("scan: no scanner registered for active domain")

// ErrSubcommandUnsupported is returned by Dispatch when the active
// pack's manifest does not declare the requested subcommand.
var ErrSubcommandUnsupported = errors.New("scan: active pack does not implement subcommand")

// Scanner is what each domain pack implements. The dispatch shell
// resolves the active pack's scanner via the manifest's scanner_id and
// invokes Scan with shared context (config, graph store, reporter,
// flag values).
type Scanner interface {
	// ID returns the scanner_id declared in the pack's manifest. Must
	// match exactly — Dispatch refuses to run a scanner whose ID does
	// not equal the manifest's scanner_id field.
	ID() string

	// Scan executes the named subcommand. Returns a Report the
	// dispatch shell prints to the user. Errors propagate.
	Scan(subcommand string, opts ScanOpts) (*Report, error)
}

// ScanOpts carries the shared context every scanner needs. Built by the
// dispatch shell from cli/scan.go's flag parsing + workspace bootstrap.
type ScanOpts struct {
	ProjectRoot string
	HeroDir     string
	Config      config.Config
	Store       *graph.Store
	Flags       map[string]any
	DryRun      bool
	Force       bool
	Reporter    Reporter
}

// Report is what a Scanner returns to the dispatch shell. Summary is a
// human-readable block printed verbatim; Warnings are non-fatal issues
// the shell surfaces with a uniform prefix.
type Report struct {
	Summary  string
	Warnings []string
}

// Reporter is the progress/log sink each scanner writes through.
// Implementations include a stdout writer (default) and a quiet sink
// (used by the test harness and --json output modes).
type Reporter interface {
	io.Writer
	Step(msg string)
	Warn(msg string)
}

// stdoutReporter routes Step / Warn / Write to the provided writer.
// Used when callers want lightweight progress output without bringing
// in a structured logger.
type stdoutReporter struct{ w io.Writer }

// StdoutReporter returns a Reporter that writes to w. nil w produces
// a no-op reporter — handy for the test harness.
func StdoutReporter(w io.Writer) Reporter {
	if w == nil {
		return discardReporter{}
	}
	return stdoutReporter{w: w}
}

func (r stdoutReporter) Write(p []byte) (int, error) { return r.w.Write(p) }
func (r stdoutReporter) Step(msg string)             { fmt.Fprintf(r.w, "%s\n", msg) }
func (r stdoutReporter) Warn(msg string)             { fmt.Fprintf(r.w, "warning: %s\n", msg) }

type discardReporter struct{}

func (discardReporter) Write(p []byte) (int, error) { return len(p), nil }
func (discardReporter) Step(string)                 {}
func (discardReporter) Warn(string)                 {}

// dispatcherState holds the registered scanners and the loaded
// per-domain manifests. Mutation is guarded by mu because Register
// runs from init() which is concurrent across packages, and tests
// can clear/re-register state. Reads after process boot are
// effectively read-only.
type dispatcherState struct {
	mu        sync.RWMutex
	scanners  map[string]Scanner          // keyed by scanner_id
	manifests map[string]*Manifest        // keyed by domain name
}

var dispatcher = &dispatcherState{
	scanners:  make(map[string]Scanner),
	manifests: make(map[string]*Manifest),
}

// Register installs a Scanner in the dispatcher map. Called from each
// pack's init(). Duplicate scanner_id is a programming error (two
// packs claiming the same id) and panics — the dispatcher is built at
// process boot and there is no recovery.
func Register(s Scanner) {
	if s == nil {
		panic("scan.Register: nil scanner")
	}
	id := s.ID()
	if id == "" {
		panic("scan.Register: scanner ID is empty")
	}
	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	if _, exists := dispatcher.scanners[id]; exists {
		panic(fmt.Sprintf("scan.Register: scanner %q already registered", id))
	}
	dispatcher.scanners[id] = s
}

// RegisterManifest installs a parsed Manifest for a domain. Called
// after embed.FS load (see domains/<name>/scan/init.go). Domains
// without a manifest are valid — Dispatch returns ErrScannerNotFound
// for them.
func RegisterManifest(domain string, m *Manifest) {
	if m == nil {
		return
	}
	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	dispatcher.manifests[domain] = m
}

// Dispatch routes the named subcommand to the active pack's scanner.
// Returns:
//
//   - (*Report, nil) on a successful scan
//   - (nil, ErrScannerNotFound) when no scanner is registered for the
//     active domain or no manifest declares it — the caller should
//     print the friendly skip message and exit 0
//   - (nil, ErrSubcommandUnsupported) when the manifest does not
//     declare the requested subcommand — the caller prints the
//     packname-specific error and exits non-zero
//   - (nil, err) for scanner-internal errors
func Dispatch(subcommand string, opts ScanOpts) (*Report, error) {
	domain := opts.Config.Domain
	if domain == "" {
		domain = "engineering"
	}

	dispatcher.mu.RLock()
	manifest := dispatcher.manifests[domain]
	dispatcher.mu.RUnlock()

	if manifest == nil {
		return nil, ErrScannerNotFound
	}

	if !manifest.HasSubcommand(subcommand) {
		return nil, fmt.Errorf("%w: %q", ErrSubcommandUnsupported, subcommand)
	}

	dispatcher.mu.RLock()
	scanner := dispatcher.scanners[manifest.ScannerID]
	dispatcher.mu.RUnlock()

	if scanner == nil {
		return nil, fmt.Errorf("scan: manifest declares scanner_id %q for domain %q but no Register call matched — check pack init() wiring",
			manifest.ScannerID, domain)
	}

	if opts.Reporter == nil {
		opts.Reporter = discardReporter{}
	}
	return scanner.Scan(subcommand, opts)
}

// LoadAndRegisterManifest reads a scan-manifest.yaml from the provided
// pack FS (rooted at domains/<name>/) and installs the parsed manifest
// for the given domain. The manifest file is optional: when absent,
// the function returns nil and the domain becomes a "no scanner
// shipped" pack (Dispatch will return ErrScannerNotFound for it).
//
// Called from each pack's scan/init.go after a successful Register so
// the dispatcher table is consistent: a scanner Register'd without a
// matching manifest is unreachable; a manifest registered without a
// matching scanner fails Dispatch with a clear error pointing at the
// init wiring.
func LoadAndRegisterManifest(domain string, packFS fs.FS) error {
	data, err := fs.ReadFile(packFS, "scan-manifest.yaml")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read scan-manifest.yaml for domain %q: %w", domain, err)
	}
	m, err := ParseManifest(data)
	if err != nil {
		return fmt.Errorf("parse scan-manifest.yaml for domain %q: %w", domain, err)
	}
	RegisterManifest(domain, m)
	return nil
}

// resetDispatcherForTest clears the registry. Test-only — exported via
// the _test.go file in this package; external packages cannot reach it.
func resetDispatcherForTest() {
	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	dispatcher.scanners = make(map[string]Scanner)
	dispatcher.manifests = make(map[string]*Manifest)
}
