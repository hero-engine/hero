package serve

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/session"
	"github.com/hero-engine/hero/internal/serve/shell"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/watch"
)

// ProjectContext holds all state for a single project within the daemon.
type ProjectContext struct {
	Slug      string
	Path      string // absolute path to project root
	HeroDir   string // absolute path to .hero directory
	watcher   *watch.Watcher
	refresher *ImportRefresher
	autoWatch bool
}

// Server ties together the HTTP API, file watchers, and event bus.
// It supports multiple projects simultaneously.
type Server struct {
	mu         sync.RWMutex
	projects   map[string]*ProjectContext
	version    string
	port       int
	bus        *EventBus
	api        *API
	httpServer *http.Server
	registry   *Registry

	// Legacy single-project fields (used when no registry is configured)
	heroDir     string
	projectRoot string
	autoWatch   bool
	uiEnabled   bool

	// Team mode
	teamMode       bool
	jobQueue       *JobQueue
	workerPool     *WorkerPool
	scheduledTasks *ScheduledTasks
	authToken      string
}

// ServerConfig configures the daemon.
type ServerConfig struct {
	HeroDir      string
	ProjectRoot  string
	Version      string
	Port         int
	AutoWatch    bool
	UIEnabled    bool   // serve embedded dashboard UI (default: true)
	RegistryPath string // optional: path to projects.json (empty = use default)
	TeamMode     bool   // enable team server (job queue, workers, API)
	Workers      int    // number of job execution workers (team mode)
	AuthToken    string // optional auth token for team API
}

// NewServer creates a daemon with all subsystems.
func NewServer(cfg ServerConfig) *Server {
	bus := NewEventBus()

	s := &Server{
		projects:    make(map[string]*ProjectContext),
		heroDir:     cfg.HeroDir,
		projectRoot: cfg.ProjectRoot,
		version:     cfg.Version,
		port:        cfg.Port,
		bus:         bus,
		autoWatch:   cfg.AutoWatch,
		uiEnabled:   cfg.UIEnabled,
	}

	if s.port == 0 {
		s.port = 7437
	}

	// Load registry if available
	if cfg.RegistryPath != "" {
		reg, err := LoadRegistryFrom(cfg.RegistryPath)
		if err == nil {
			s.registry = reg
		}
	}

	// If a primary project is specified (single-project mode or cwd project),
	// add it as a project context
	if cfg.HeroDir != "" && cfg.ProjectRoot != "" {
		slug := filepath.Base(cfg.ProjectRoot)
		s.projects[slug] = &ProjectContext{
			Slug:      slug,
			Path:      cfg.ProjectRoot,
			HeroDir:   cfg.HeroDir,
			autoWatch: cfg.AutoWatch,
		}
	}

	// Create the API with multi-project support. The shell (top-nav
	// home routing, /-redirect) is composed in Run.
	s.api = NewAPI(s, bus)

	// Team mode setup
	if cfg.TeamMode {
		s.teamMode = true
		s.authToken = cfg.AuthToken

		jq, err := NewJobQueue(cfg.HeroDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hero serve: failed to open job queue: %v\n", err)
		} else {
			s.jobQueue = jq
			s.workerPool = NewWorkerPool(jq, cfg.ProjectRoot, cfg.HeroDir, cfg.Workers)
		}
	}

	return s
}

// Bus returns the event bus (for external publishing or testing).
func (s *Server) Bus() *EventBus {
	return s.bus
}

// Projects returns a snapshot of all project slugs.
func (s *Server) Projects() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	slugs := make([]string, 0, len(s.projects))
	for k := range s.projects {
		slugs = append(slugs, k)
	}
	return slugs
}

// GetProject returns the ProjectContext for a slug, or nil if not found.
func (s *Server) GetProject(slug string) *ProjectContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projects[slug]
}

// AddProject registers and starts a project in the running daemon.
func (s *Server) AddProject(slug, projectRoot, heroDir string, autoWatch bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.projects[slug]; ok {
		return nil // idempotent
	}

	pc := &ProjectContext{
		Slug:      slug,
		Path:      projectRoot,
		HeroDir:   heroDir,
		autoWatch: autoWatch,
	}
	if _, err := index.Rebuild(heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: index rebuild for %s failed: %v\n", slug, err)
	}

	// Start watcher if enabled
	if autoWatch {
		s.startProjectWatcher(pc)
	}

	s.projects[slug] = pc
	return nil
}

// RemoveProject stops and removes a project from the running daemon.
func (s *Server) RemoveProject(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pc, ok := s.projects[slug]
	if !ok {
		return fmt.Errorf("project %q not found", slug)
	}

	if pc.watcher != nil {
		pc.watcher.Stop()
	}
	if pc.refresher != nil {
		pc.refresher.Stop()
	}

	delete(s.projects, slug)
	return nil
}

// ProjectCount returns the number of registered projects.
func (s *Server) ProjectCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.projects)
}

// Run starts the HTTP server and optional file watchers for all projects.
// It blocks until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	s.loadRegistryProjects()

	s.mu.RLock()
	for _, pc := range s.projects {
		if _, err := index.Rebuild(pc.HeroDir); err != nil {
			fmt.Fprintf(os.Stderr, "hero serve: initial index rebuild for %s failed: %v\n", pc.Slug, err)
		}
		if pc.autoWatch && pc.watcher == nil {
			s.startProjectWatcher(pc)
		}
		// Start auto-refresh if configured
		if pc.refresher == nil {
			pc.refresher = StartImportRefresher(pc.Path, pc.HeroDir, pc.Slug, s.bus)
		}
	}
	s.mu.RUnlock()
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	handler := s.api.Handler()

	// Register team API endpoints if team mode is enabled
	if s.teamMode && s.jobQueue != nil {
		mux, ok := handler.(*http.ServeMux)
		if !ok {
			mux = http.NewServeMux()
			mux.Handle("/", handler)
		}
		var authMiddleware func(http.Handler) http.Handler
		if s.authToken != "" {
			authMiddleware = TokenAuthMiddleware(s.authToken)
		}
		jwtSecret := os.Getenv("HERO_JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = s.authToken // fall back to auth token as JWT secret
		}

		// Use TeamAuthMiddleware if users exist, otherwise fall back to token
		if s.jobQueue.UserCount() > 0 || jwtSecret != "" {
			authMiddleware = TeamAuthMiddleware(s.jobQueue, s.authToken, jwtSecret)
		}

		RegisterAuthAPI(mux, s.jobQueue, jwtSecret)
		RegisterJobsAPI(mux, s.jobQueue, authMiddleware)
		RegisterTeamCoordinationAPI(mux, s.jobQueue, authMiddleware)
		handler = mux

		// Start workers
		s.workerPool.Start()

		// Start scheduled tasks
		s.scheduledTasks = NewScheduledTasks(s.projectRoot, s.heroDir)
		s.scheduledTasks.Start()

		fmt.Fprintf(os.Stderr, "hero serve: team mode enabled (%d workers)\n", s.workerPool.count)
	}

	// Compose the shell on top of the API handler. The shell owns
	// /, the five home roots, and /_kitchen-sink; the API owns
	// /api/*, /health, /auth/* (team mode), and team-mode job mounts.
	// Static shell assets are served from /static/shell/.
	if s.uiEnabled {
		shellRouter := s.buildShellRouter()
		topMux := http.NewServeMux()
		topMux.Handle("/api/", handler)
		topMux.Handle("/health", handler)
		topMux.Handle("/auth/", handler)
		topMux.Handle("/static/shell/", http.StripPrefix("/static/shell/", http.FileServer(http.FS(shell.StaticFS()))))
		topMux.Handle("/", shellRouter.Handler())
		handler = topMux
	}

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	// Server started
	projectCount := s.ProjectCount()
	fmt.Fprintf(os.Stderr, "hero serve v%s: listening on http://%s (%d project(s))\n", s.version, addr, projectCount)
	fmt.Fprintf(os.Stderr, "hero serve: press Ctrl+C to stop\n")

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for cancellation
	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	// Graceful shutdown
	return s.shutdown()
}

func (s *Server) loadRegistryProjects() {
	if s.registry == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for slug, entry := range s.registry.List() {
		if _, ok := s.projects[slug]; ok {
			continue // already loaded
		}

		heroDir := filepath.Join(entry.Path, ".hero")
		if _, err := os.Stat(heroDir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "hero serve: skipping %s — .hero not found at %s\n", slug, entry.Path)
			continue
		}

		s.projects[slug] = &ProjectContext{
			Slug:      slug,
			Path:      entry.Path,
			HeroDir:   heroDir,
			autoWatch: s.autoWatch,
		}
	}
}

func (s *Server) shutdown() error {
	fmt.Fprintf(os.Stderr, "hero serve: shutting down...\n")

	// Stop all watchers and refreshers
	s.mu.RLock()
	for _, pc := range s.projects {
		if pc.watcher != nil {
			pc.watcher.Stop()
		}
		if pc.refresher != nil {
			pc.refresher.Stop()
		}
	}
	s.mu.RUnlock()

	// Stop team mode components
	if s.scheduledTasks != nil {
		s.scheduledTasks.Stop()
	}
	if s.workerPool != nil {
		s.workerPool.Stop()
	}
	if s.jobQueue != nil {
		s.jobQueue.Close()
	}

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) startProjectWatcher(pc *ProjectContext) {
	slug := pc.Slug
	heroDir := pc.HeroDir

	handler := func(events []watch.Event) {
		specEvents := watch.SpecEvents(events)
		if len(specEvents) == 0 {
			return
		}

		// Reindex changed specs
		s.reindexSpecs(heroDir, slug, specEvents)

		// Publish events to the bus
		for _, e := range specEvents {
			var evType EventType
			switch e.Kind {
			case watch.EventCreated:
				evType = EventSpecCreated
			case watch.EventModified:
				evType = EventSpecModified
			case watch.EventDeleted:
				evType = EventSpecDeleted
			}

			evSlug := slugFromPath(e.Path)
			s.bus.Publish(Event{
				Type:    evType,
				Project: slug,
				Slug:    evSlug,
				Path:    e.Path,
			})
		}
	}

	pc.watcher = watch.New(heroDir, 2*time.Second, handler)
	go func() {
		if err := pc.watcher.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "hero serve: watcher error for %s: %v\n", slug, err)
		}
	}()
}

func (s *Server) reindexSpecs(heroDir, project string, events []watch.Event) {
	idx, err := index.Open(heroDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: could not open index for %s: %v\n", project, err)
		return
	}
	defer idx.Close()

	for _, e := range events {
		switch e.Kind {
		case watch.EventCreated, watch.EventModified:
			sp, err := spec.ParseFile(e.Path)
			if err != nil {
				continue
			}
			content, err := os.ReadFile(e.Path)
			if err != nil {
				continue
			}
			if err := idx.IndexSpec(sp, string(content)); err != nil {
				continue
			}
		case watch.EventDeleted:
			slug := slugFromPath(e.Path)
			if slug != "" {
				idx.RemoveSpec(slug)
			}
		}
	}

	s.bus.Publish(Event{
		Type:    EventIndexRebuilt,
		Project: project,
		Message: fmt.Sprintf("reindexed %d spec(s)", len(events)),
	})
}

// slugFromPath extracts a slug from a spec path using the directory name.
func slugFromPath(path string) string {
	dir := filepath.Dir(path)
	return filepath.Base(dir)
}

// buildShellRouter assembles the shell router with the active edition,
// session store, identifying chrome strings, and the five stub home
// registrations. Each home spec replaces its stub when it lands.
func (s *Server) buildShellRouter() *shell.Router {
	ed := edition.Resolve()

	// Session store is best-effort. A nil store still serves pages
	// (the shell silently skips writes) — the / redirect just falls
	// back to /now.
	store, err := session.Open("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: shell session store unavailable: %v\n", err)
		store = nil
	}

	workspace := s.shellWorkspaceName()
	branch := detectGitBranch(s.projectRoot)
	userName := shellUserName()

	r := shell.New(ed, store, workspace, branch, userName, s.version)
	shell.RegisterStubHomes(r)
	return r
}

// shellWorkspaceName picks a workspace label for the top-nav. Prefers
// the project root's basename; falls back to "hero" when running
// without a primary project.
func (s *Server) shellWorkspaceName() string {
	if s.projectRoot != "" {
		return filepath.Base(s.projectRoot)
	}
	return "hero"
}

// shellUserName returns a display name for the avatar. Reads the
// standard OS env vars; "you" when nothing is set.
func shellUserName() string {
	if v := os.Getenv("USER"); v != "" {
		return v
	}
	if v := os.Getenv("USERNAME"); v != "" {
		return v
	}
	return "you"
}

// detectGitBranch returns the current git branch for the given project
// root, or "" when not in a git checkout. Best-effort — used only for
// display in the top-nav workspace-state chip.
func detectGitBranch(projectRoot string) string {
	if projectRoot == "" {
		return ""
	}
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
