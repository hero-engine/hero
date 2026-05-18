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

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/serve/api"
	"github.com/hero-engine/hero/internal/serve/chat"
	"github.com/hero-engine/hero/internal/serve/edition"
	agentspage "github.com/hero-engine/hero/internal/serve/pages/agentspage"
	agentsdata "github.com/hero-engine/hero/internal/serve/pages/agentspage/data"
	knowledgepage "github.com/hero-engine/hero/internal/serve/pages/knowledge"
	nowpage "github.com/hero-engine/hero/internal/serve/pages/now"
	nowdata "github.com/hero-engine/hero/internal/serve/pages/now/data"
	peoplepage "github.com/hero-engine/hero/internal/serve/pages/people"
	workpage "github.com/hero-engine/hero/internal/serve/pages/work"
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

	// Chat dispatcher subsystem. Initialized in Run; nil before then.
	chatRegistry *chat.Registry
	chatStore    *chat.Store
	chatAPI      *chat.API

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

	// Initialize the chat dispatcher. The registry, store, and API
	// live alongside the rest of the daemon; the MCP server receives
	// the registry handle so adapter clients can register on
	// initialize. Store open failures are best-effort: chat falls
	// back to in-memory conversation ids when persistence is absent.
	s.initChat()

	// Compose the shell on top of the API handler. The shell owns
	// /, the five home roots, and /_kitchen-sink; the API owns
	// /api/*, /health, /auth/* (team mode), and team-mode job mounts.
	// Static shell assets are served from /static/shell/.
	if s.uiEnabled {
		shellRouter := s.buildShellRouter()
		topMux := http.NewServeMux()
		// Chat endpoints under /api/chat/* must mount BEFORE the
		// generic /api/ catch-all so the chat-specific handlers win.
		if s.chatAPI != nil {
			s.chatAPI.Mount(topMux)
		}
		// Now SSE channel + per-section fragment endpoints. Mounted
		// before the generic /api/ catch-all for the same reason as
		// chat above.
		nowHandler := api.NewNowHandler(nowpage.Deps{
			ProjectRoot:  s.projectRoot,
			HeroDir:      s.heroDir,
			Workspace:    s.shellWorkspaceName(),
			Branch:       detectGitBranch(s.projectRoot),
			UserName:     shellUserName(),
			Proposals:    s.snapshotProposals,
			ChatRegistry: s.chatRegistry,
			LiveSessions: s.snapshotLiveSessions,
		}, busSubscriber{bus: s.bus})
		nowHandler.Mount(topMux)

		// Knowledge SSE channel + per-section fragment endpoints. Same
		// mount-before-catch-all reason as the handlers above.
		knowledgeHandler := api.NewKnowledgeHandler(knowledgepage.Deps{
			ProjectRoot: s.projectRoot,
			HeroDir:     s.heroDir,
			Workspace:   s.shellWorkspaceName(),
			Branch:      detectGitBranch(s.projectRoot),
			UserName:    shellUserName(),
		}, busSubscriber{bus: s.bus})
		knowledgeHandler.Mount(topMux)

		// Work SSE channel + per-section fragment endpoints. Same
		// shape as the Now handler block above: registered before the
		// generic /api/ catch-all so the work-specific routes win.
		workHandler := api.NewWorkHandler(workpage.Deps{
			ProjectRoot: s.projectRoot,
			HeroDir:     s.heroDir,
			Workspace:   s.shellWorkspaceName(),
			Branch:      detectGitBranch(s.projectRoot),
			UserName:    shellUserName(),
		}, busSubscriber{bus: s.bus})
		workHandler.Mount(topMux)

		// People & ROI SSE channel + per-section fragment endpoints.
		// Same mount-before-catch-all reason as the handlers above.
		peopleHandler := api.NewPeopleHandler(peoplepage.Deps{
			ProjectRoot: s.projectRoot,
			HeroDir:     s.heroDir,
			Workspace:   s.shellWorkspaceName(),
			Branch:      detectGitBranch(s.projectRoot),
			UserName:    shellUserName(),
		}, busSubscriber{bus: s.bus})
		peopleHandler.Mount(topMux)

		// Agents SSE channel + per-section fragment endpoints. Same
		// mount-before-catch-all reason as the handlers above.
		agentsHandler := api.NewAgentsHandler(agentspage.Deps{
			ProjectRoot:  s.projectRoot,
			HeroDir:      s.heroDir,
			Workspace:    s.shellWorkspaceName(),
			Branch:       detectGitBranch(s.projectRoot),
			UserName:     shellUserName(),
			LiveSessions: s.snapshotLiveSessions,
			Proposals:    s.snapshotAgentsProposals,
		}, busSubscriber{bus: s.bus})
		agentsHandler.Mount(topMux)

		topMux.Handle("/api/", handler)
		topMux.Handle("/health", handler)
		topMux.Handle("/auth/", handler)
		topMux.Handle("/static/shell/", http.StripPrefix("/static/shell/", http.FileServer(http.FS(shell.StaticFS()))))
		topMux.Handle("/", shellRouter.Handler())
		handler = topMux
	} else if s.chatAPI != nil {
		// UI disabled but chat still needs a mount point — graft
		// chat routes onto the API handler via a wrapping mux.
		wrap := http.NewServeMux()
		s.chatAPI.Mount(wrap)
		wrap.Handle("/", handler)
		handler = wrap
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

// initChat constructs the chat registry, store, and API. Called from
// Run before the HTTP mux is composed. Best-effort: store-open
// failures log and continue (the API still serves capability + turn,
// they just don't persist).
func (s *Server) initChat() {
	if s.chatRegistry != nil {
		return
	}
	s.chatRegistry = chat.NewRegistry()

	store, err := chat.Open("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: chat store unavailable: %v\n", err)
		store = nil
	}
	s.chatStore = store

	streamer := chat.NewStreamer(&busAdapter{bus: s.bus})
	s.chatAPI = chat.NewAPI(s.chatRegistry, store, streamer, s.projectRoot)

	// If hero.json configures a hero-code endpoint, probe it now.
	// Failures are non-fatal — we log once and proceed in
	// no-adapter mode until hero-code reconnects (or registers via
	// MCP later).
	if s.projectRoot != "" {
		cfg, cerr := config.Load(s.projectRoot)
		if cerr == nil && cfg.Chat != nil && cfg.Chat.Headless != nil {
			ep := cfg.Chat.Headless.Endpoint
			if ep != "" {
				if adapter, perr := chat.TryConnectHeroCode(ep); perr == nil {
					if rerr := s.chatRegistry.Register("hero-code-"+ep, adapter); rerr != nil {
						fmt.Fprintf(os.Stderr, "hero serve: register hero-code adapter: %v\n", rerr)
					} else {
						fmt.Fprintf(os.Stderr, "hero serve: hero-code adapter registered (%s)\n", ep)
					}
				} else {
					fmt.Fprintf(os.Stderr, "hero serve: hero-code endpoint configured at %s but unreachable — falling back to no-adapter mode (%v)\n", ep, perr)
				}
			}
		}
	}
}

// busAdapter satisfies chat.EventBus by translating BusEvents into
// serve.Events. The serve event bus is project-scoped via the
// Project field; chat events leave Project blank and rely on the
// Type prefix ("chat.") + Topic for routing.
type busAdapter struct {
	bus *EventBus
}

func (a *busAdapter) Publish(ev chat.BusEvent) {
	if a == nil || a.bus == nil {
		return
	}
	a.bus.Publish(Event{
		Type:      EventType(ev.Type),
		Slug:      ev.Topic,
		Payload:   ev.Payload,
		Timestamp: ev.Timestamp,
	})
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

	// Register the real Now home in place of its (no-longer-present)
	// stub. Wired with a per-project proposal-store snapshotter that
	// surfaces pending proposals in the Needs-your-input section.
	nowDeps := nowpage.Deps{
		ProjectRoot:  s.projectRoot,
		HeroDir:      s.heroDir,
		Workspace:    workspace,
		Branch:       branch,
		UserName:     userName,
		Proposals:    s.snapshotProposals,
		ChatRegistry: s.chatRegistry,
		LiveSessions: s.snapshotLiveSessions,
	}
	if err := nowpage.Register(r, nowDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Now home: %v\n", err)
	}

	// Register the real Work home in place of its (no-longer-present)
	// stub. Per-spec: this owns /work; sibling /work/* routes land in
	// follow-on work.
	workDeps := workpage.Deps{
		ProjectRoot:              s.projectRoot,
		HeroDir:                  s.heroDir,
		Workspace:                workspace,
		Branch:                   branch,
		UserName:                 userName,
		ChatInteractiveConnected: s.chatInteractiveConnected,
	}
	if err := workpage.Register(r, workDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Work home: %v\n", err)
	}

	// Register the real Knowledge home in place of its (no-longer-present)
	// stub. Deps mirror the Now wiring — no proposal-store hook yet.
	knowledgeDeps := knowledgepage.Deps{
		ProjectRoot:              s.projectRoot,
		HeroDir:                  s.heroDir,
		Workspace:                workspace,
		Branch:                   branch,
		UserName:                 userName,
		ChatInteractiveConnected: s.chatInteractiveConnected,
	}
	if err := knowledgepage.Register(r, knowledgeDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Knowledge home: %v\n", err)
	}

	// Register the real People & ROI home in place of its (no-longer-present)
	// stub. Deps mirror the other home wiring.
	peopleDeps := peoplepage.Deps{
		ProjectRoot:              s.projectRoot,
		HeroDir:                  s.heroDir,
		Workspace:                workspace,
		Branch:                   branch,
		UserName:                 userName,
		ChatInteractiveConnected: s.chatInteractiveConnected,
	}
	if err := peoplepage.Register(r, peopleDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register People & ROI home: %v\n", err)
	}

	// Register the real Agents home in place of its (no-longer-present)
	// stub. Wired with the live-session snapshot reader (which is the
	// canonical "live ledger" the Now home will consume in a follow-up)
	// plus a proposal snapshotter. Scheduled / automation hooks are left
	// nil until those engines land per the spec's build order; the page
	// renders empty-state notices when they are.
	agentsDeps := agentspage.Deps{
		ProjectRoot:              s.projectRoot,
		HeroDir:                  s.heroDir,
		Workspace:                workspace,
		Branch:                   branch,
		UserName:                 userName,
		LiveSessions:             s.snapshotLiveSessions,
		Proposals:                s.snapshotAgentsProposals,
		ChatInteractiveConnected: s.chatInteractiveConnected,
	}
	if err := agentspage.Register(r, agentsDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Agents home: %v\n", err)
	}
	return r
}

// snapshotProposals returns the pending proposals across every session
// for the primary project, formatted for the Now inbox renderer. Solo
// mode collapses everything into one workspace; team / cloud editions
// are handled by their own home spec.
func (s *Server) snapshotProposals() []*nowdata.ProposalRow {
	if s == nil || s.api == nil || s.api.proposals == nil {
		return nil
	}
	slug := filepath.Base(s.projectRoot)
	if slug == "" || slug == "." {
		return nil
	}
	store := s.api.proposals.get(slug)
	if store == nil {
		return nil
	}
	// The store is keyed by session; we don't track active sessions
	// from here, so we have no list of sessions to iterate. The propose
	// store does not expose a global "all sessions" enumerator yet —
	// when the agents home wires the live session ledger this gains a
	// real source. Until then return an empty slice so the inbox
	// renders cleanly.
	_ = store
	return nil
}

// chatInteractiveConnected probes the chat-adapter registry and
// returns true when at least one adapter advertises the Interactive
// kind. Used by the four non-Now homes to flip their inline chat-
// input into its disabled state. Passing a probe (rather than the
// registry itself) keeps those packages free of chat / runner
// dependencies — see internal/serve/pages/{work,knowledge,
// agentspage,people}/import_test.go.
func (s *Server) chatInteractiveConnected() bool {
	if s == nil || s.chatRegistry == nil {
		return false
	}
	return chat.Resolve(s.chatRegistry, "").Interactive != ""
}

// snapshotLiveSessions is the canonical "live session ledger" reader
// for the Agents home. v1 reads from the team-mode job queue's
// sessions table (which the runner & MCP server register against);
// when team mode is not configured the snapshot is empty. The
// SessionRow shape is the stable contract the Now home will consume
// in a follow-up to replace its mocked agents card.
func (s *Server) snapshotLiveSessions() []agentsdata.SessionRow {
	if s == nil || s.jobQueue == nil {
		return nil
	}
	raw, err := s.jobQueue.ActiveSessions()
	if err != nil {
		return nil
	}
	out := make([]agentsdata.SessionRow, 0, len(raw))
	for _, m := range raw {
		started, _ := time.Parse(time.RFC3339, m["started_at"])
		lastSeen, _ := time.Parse(time.RFC3339, m["last_seen"])
		out = append(out, agentsdata.SessionRow{
			ID:           m["id"],
			Agent:        m["agent"],
			Spec:         m["spec_slug"],
			Command:      m["command"],
			UserID:       m["user_id"],
			Status:       "live",
			StartedAt:    started,
			LastActiveAt: lastSeen,
		})
	}
	return out
}

// snapshotAgentsProposals returns the pending proposals for the
// primary project shaped for the Agents home's approval-row builder.
// Same data origin and caveat as snapshotProposals: the propose store
// does not yet expose a global enumerator, so this returns nil until
// the store gains a list method. Returning nil keeps the page's
// empty-state rendering correct.
func (s *Server) snapshotAgentsProposals() []agentsdata.ProposalRow {
	if s == nil || s.api == nil || s.api.proposals == nil {
		return nil
	}
	slug := filepath.Base(s.projectRoot)
	if slug == "" || slug == "." {
		return nil
	}
	store := s.api.proposals.get(slug)
	if store == nil {
		return nil
	}
	_ = store
	return nil
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

// busSubscriber adapts *EventBus to api.Subscriber by stripping the
// internal Event shape down to just the Type field that the Now SSE
// channel inspects. The channel is wrapped so the bus's strongly
// typed Event flows are translated lazily without holding the bus
// goroutine.
type busSubscriber struct{ bus *EventBus }

func (b busSubscriber) Subscribe(bufSize int) (uint64, <-chan api.Event) {
	id, src := b.bus.Subscribe(bufSize)
	dst := make(chan api.Event, bufSize)
	go func() {
		defer close(dst)
		for ev := range src {
			select {
			case dst <- api.Event{Type: string(ev.Type)}:
			default:
				// drop on slow consumer — matches bus semantics
			}
		}
	}()
	return id, dst
}

func (b busSubscriber) Unsubscribe(id uint64) {
	b.bus.Unsubscribe(id)
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
