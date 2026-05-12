package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/spec"
)

// testEnv holds the state for a CLI test run.
type testEnv struct {
	t       *testing.T
	dir     string // temp project root
	heroDir string
	origDir string
}

// newTestEnv creates a temp directory and chdir into it.
// It also initializes a .hero workspace with default config.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	dir := t.TempDir()
	heroDir := filepath.Join(dir, ".hero")

	// Create .hero directory structure
	dirs := []string{
		heroDir,
		filepath.Join(heroDir, "planning"),
		filepath.Join(heroDir, "planning", "features"),
		filepath.Join(heroDir, "planning", "bugs"),
		filepath.Join(heroDir, "planning", "initiatives"),
		filepath.Join(heroDir, "specs"),
		filepath.Join(heroDir, "knowledge"),
		filepath.Join(heroDir, "knowledge", "conventions"),
		filepath.Join(heroDir, "knowledge", "decisions"),
		filepath.Join(heroDir, "knowledge", "rules"),
		filepath.Join(heroDir, "knowledge", "external"),
		filepath.Join(heroDir, "knowledge", "context"),
		filepath.Join(heroDir, "knowledge", "templates"),
		filepath.Join(heroDir, "knowledge", "notes"),
		filepath.Join(heroDir, "mocks"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}

	// Write default config
	cfg := config.DefaultConfig()
	if err := cfg.Save(dir); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	// Chdir into the project root
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	env := &testEnv{
		t:       t,
		dir:     dir,
		heroDir: heroDir,
		origDir: origDir,
	}

	t.Cleanup(func() {
		os.Chdir(env.origDir)
	})

	return env
}

// newTestEnvEmpty creates a temp directory without a .hero workspace.
func newTestEnvEmpty(t *testing.T) *testEnv {
	t.Helper()

	dir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	env := &testEnv{
		t:       t,
		dir:     dir,
		heroDir: filepath.Join(dir, ".hero"),
		origDir: origDir,
	}

	t.Cleanup(func() {
		os.Chdir(env.origDir)
	})

	return env
}

// addSpec creates a spec file in the test environment.
func (e *testEnv) addSpec(relPath string, content string) {
	e.t.Helper()

	path := filepath.Join(e.heroDir, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		e.t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		e.t.Fatalf("WriteFile: %v", err)
	}
}

// indexAll rebuilds the index for the test workspace.
func (e *testEnv) indexAll() {
	e.t.Helper()

	_, err := index.Rebuild(e.heroDir)
	if err != nil {
		e.t.Fatalf("Rebuild index: %v", err)
	}
}

// indexSpec indexes a single spec by path.
func (e *testEnv) indexSpec(relPath string) {
	e.t.Helper()

	path := filepath.Join(e.heroDir, relPath)
	s, err := spec.ParseFile(path)
	if err != nil {
		e.t.Fatalf("ParseFile %s: %v", relPath, err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		e.t.Fatalf("ReadFile %s: %v", relPath, err)
	}

	idx, err := index.Open(e.heroDir)
	if err != nil {
		e.t.Fatalf("Open index: %v", err)
	}
	defer idx.Close()

	if err := idx.IndexSpec(s, string(content)); err != nil {
		e.t.Fatalf("IndexSpec: %v", err)
	}
}

// mu protects stdout capture — tests using captureStdout should not run in parallel.
var mu sync.Mutex

// captureStdout captures stdout during f and returns what was printed.
func captureStdout(f func()) string {
	mu.Lock()
	defer mu.Unlock()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// runCmd executes a hero CLI command, captures stdout, and returns the output and error.
func runCmd(args ...string) (string, error) {
	resetFlags()
	rootCmd.SetArgs(args)

	var cmdErr error
	output := captureStdout(func() {
		cmdErr = rootCmd.Execute()
	})

	// Reset for next call
	rootCmd.SetArgs(nil)

	return output, cmdErr
}

// resetFlags resets command-level flags to their defaults.
// Cobra persists flag values between calls in the same process.
func resetFlags() {
	// Reset search flags
	searchByFile = false
	searchType = ""
	searchStatus = ""
	searchTag = ""
	searchSince = ""
	searchListOnly = false

	// Reset install flags
	installTarget = ""
	installForce = false
	installDryRun = false

	// Reset nudge flags
	relevantFiles = nil

	// Reset check flags
	checkStaleDays = 14
	checkReconcile = false

	// Reset init flags
	initFolder = config.DefaultFolder

	// Reset new flags
	newSpecType = "feature"
	newInteractive = false

	// Reset diff flags
	diffBase = "HEAD"

	// Reset graph flags
	graphFormat = "text"

	// Reset feed flags
	feedSince = ""
	feedType = ""
	feedSlug = ""
	feedAgent = ""
	feedLimit = 20
	feedFormat = ""

	// Reset uninstall flags
	uninstallTarget = ""
	uninstallDryRun = false

	// Reset wiki-sync flags
	wikiSyncAll = false

	// Reset note flags
	noteFrom = ""

	// Reset smoke flag
	globalSmoke = false

	// Reset smoke command flags
	smokeRunAll = false
	smokeRunArea = ""
	smokeRunSince = ""

	// Reset scan flags
	scanDryRun = false
	scanForce = false

	// Reset watch flags
	watchMode = "local"
	watchInterval = 2

	// Reset serve flags
	servePort = 0
	serveNoWatch = false
	serveNoUI = false
	serveAdd = ""
	serveRemove = ""
	serveList = false

	// Reset mock flags
	mockList = false
	mockOpen = ""
	mockServe = false

	// Reset report flags
	reportOutput = ""
	reportOpen = false

	// Reset import flags
	importLabel = ""
	importLimit = 0
	syncImportDryRun = false
	importType = "feature"

	// Reset replay flags
	replayBase = "HEAD"

	// Reset cost flags
	costAll = false

	// Reset upgrade flags
	upgradeDryRun = false
	upgradeForce = false

	// Reset list flags
	listTypes = nil
	listStatuses = nil
	listHorizons = nil
	listTags = nil
	listReady = false
	listBlocked = false
	listPinned = false
	listMine = ""
	listStale = 0
	listSortKey = string(spec.SortRecency)
	listLimit = 0
	listFormat = "table"

	// Reset queue flags
	queueLimit = 0
	queueHorizons = nil
	queueFormat = "kickoff"

	// Reset next checkpoint flags
	checkpointQuiet = false

	// Reset deliver flags
	deliverManual = false
	deliverAsync = false
	deliverBatch = false

	// Reset job-run flags
	jobRunJobID = ""
	jobRunBatchID = ""

	// Reset score flags
	scoreJSON = false

	// Reset pipeline flags
	pipelineType = ""
	pipelineJSON = false
	pipelineRun = ""

	// Reset diagnose flags
	diagnoseBatch = false
	diagnoseJSON = false
	diagnoseAsync = false

	// Reset sync cloud flags
	syncCloudFull = false
	syncCloudStatus = false

	// Note: upgradeContentFS is intentionally NOT reset here — it's a test
	// injection point, not a CLI flag.
}
