package serve

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/mailquery"
	"github.com/hero-engine/hero/internal/attention/projection"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
	"github.com/hero-engine/hero/internal/attention/suggestion"
	"github.com/hero-engine/hero/internal/cloud"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/projectregistry"
	"github.com/hero-engine/hero/internal/serve/api"
	"github.com/hero-engine/hero/internal/serve/chat"
	"github.com/hero-engine/hero/internal/serve/edition"
	"github.com/hero-engine/hero/internal/serve/healthcache"
	"github.com/hero-engine/hero/internal/serve/opsrunner"
	agentspage "github.com/hero-engine/hero/internal/serve/pages/agentspage"
	agentsdata "github.com/hero-engine/hero/internal/serve/pages/agentspage/data"
	knowledgepage "github.com/hero-engine/hero/internal/serve/pages/knowledge"
	nowpage "github.com/hero-engine/hero/internal/serve/pages/now"
	nowdata "github.com/hero-engine/hero/internal/serve/pages/now/data"
	peoplepage "github.com/hero-engine/hero/internal/serve/pages/people"
	rolluppage "github.com/hero-engine/hero/internal/serve/pages/rollup"
	workpage "github.com/hero-engine/hero/internal/serve/pages/work"
	projectpage "github.com/hero-engine/hero/internal/serve/projectpage"
	projectpagedata "github.com/hero-engine/hero/internal/serve/projectpage/data"
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
	startedAt  time.Time
	bus        *EventBus
	api        *API
	httpServer *http.Server
	registry   *Registry

	// Legacy single-project fields (used when no registry is configured)
	heroDir     string
	projectRoot string
	autoWatch   bool
	uiEnabled   bool

	// projectRouters caches one shell.Router per project slug for the
	// /p/<slug>/<page> routes. Lazily initialized on first /p/ request.
	projectRouters *projectRouterCache

	// Chat dispatcher subsystem. Initialized in Run; nil before then.
	chatRegistry *chat.Registry
	chatStore    *chat.Store
	chatAPI      *chat.API

	// opsRunner backs the /api/{slug}/ops/{verb} endpoints and the
	// Operations section on /p/<slug>/project. Constructed once in
	// NewServer and threaded into both the API and the per-project
	// projectpage Deps so the same runner serves the section render
	// (via Lookup) and the API dispatch.
	opsRunner *opsrunner.Runner

	// healthCache is the per-project TTL cache of `hero check` results
	// + peer reachability probes that backs the /p/<slug>/project
	// Health and Peers sections. Phase 5 of hero-serve-project-section.
	// One instance per daemon, shared across all per-project handlers.
	healthCache *healthcache.Cache

	// pendingRemove tracks registry-remove operations inside their
	// 5-second grace window. Phase 4 of hero-serve-project-section —
	// entries do NOT survive a daemon restart (intentional safety).
	pendingRemove *pendingRemoveQueue

	// Team mode
	teamMode       bool
	jobQueue       *JobQueue
	workerPool     *WorkerPool
	scheduledTasks *ScheduledTasks
	authToken      string

	// Cloud auto-sync and presence
	cloudDaemon   *cloud.Daemon
	cloudPresence *cloud.PresenceReporter
	cloudBusSub   uint64
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
	s.api.SetAttentionService(s.newAttentionProjectionService)
	s.api.SetMailQueryService(s.newMailQueryService)

	// OpsRunner backs the lifecycle-ops buttons on the Project page.
	// Constructed before per-project Deps are built so both the API
	// handler and the projectpage renderer share one registry.
	s.opsRunner = opsrunner.New(context.Background())
	s.api.SetOpsRunner(s.opsRunner)

	// Health cache: TTL sourced from hero.json's serve.health_ttl (with
	// 5-minute default). Read here rather than per-request because
	// re-parsing the config string per page render is wasteful and the
	// TTL is daemon-lifetime config — the operator restarts to change it.
	healthTTL := 5 * time.Minute
	if cfg.ProjectRoot != "" {
		if loaded, err := config.Load(cfg.ProjectRoot); err == nil {
			healthTTL = loaded.Serve.HealthTTLDuration()
		}
	}
	s.healthCache = healthcache.New(healthTTL, healthcache.Options{Ops: s.opsRunner})
	s.api.SetHealthCache(s.healthCache)

	// Pending-remove queue backs the 5-second grace window for the
	// registry Remove button. Phase 4 of hero-serve-project-section.
	s.pendingRemove = newPendingRemoveQueue()

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

func (s *Server) newMailQueryService() (*mailquery.Service, error) {
	root, err := attentionstate.Ensure(attentionstate.Options{ProjectRoot: s.projectRoot})
	if err != nil {
		return nil, err
	}
	registry := s.registry
	if registry == nil {
		registry, err = projectregistry.Load()
		if err != nil {
			return nil, err
		}
	}
	return mailquery.NewService(root, registry)
}

func (s *Server) newAttentionProjectionService() (*projection.Service, error) {
	root, err := attentionstate.Ensure(attentionstate.Options{ProjectRoot: s.projectRoot})
	if err != nil {
		return nil, err
	}
	registry := s.registry
	if registry == nil {
		registry, err = projectregistry.Load()
		if err != nil {
			return nil, err
		}
	}
	mailSource, err := projection.NewRegistryMailSource(root, registry)
	if err != nil {
		return nil, err
	}
	resolver := focus.NewRegistryResolver(registry)
	focusStore, err := focus.NewStore(root)
	if err != nil {
		return nil, err
	}
	focusService := focus.NewService(focusStore, resolver)
	suggestionStore, err := suggestion.NewStore(root)
	if err != nil {
		return nil, err
	}
	return projection.NewService(
		mailSource,
		focusService,
		suggestion.NewService(suggestionStore, focusService, resolver),
	), nil
}

// startCloudSync starts the background sync daemon if a team connection
// with cloud config exists and auto-sync is enabled.
func (s *Server) startCloudSync() {
	tc := config.LoadTeamConnection()
	if tc == nil || !tc.AutoSyncEnabled() {
		return
	}

	cfg, err := config.Load(s.projectRoot)
	if err != nil || cfg.Cloud == nil || cfg.Cloud.OrgID == "" || cfg.Cloud.RepoID == "" {
		return
	}

	cloudURL := tc.URL
	if cloudURL == "" {
		return
	}

	cloudCfg := cloud.Config{
		CloudURL:    cloudURL,
		Token:       tc.Token,
		OrgID:       cfg.Cloud.OrgID,
		RepoID:      cfg.Cloud.RepoID,
		ProjectRoot: s.projectRoot,
		HeroDir:     s.heroDir,
	}

	daemon := cloud.NewDaemon(cloudCfg)
	daemon.Start()

	id, events := s.bus.Subscribe(64)
	go func() {
		for ev := range events {
			switch ev.Type {
			case EventSpecCreated, EventSpecModified, EventSpecDeleted, EventIndexRebuilt:
				daemon.Notify()
			}
		}
	}()

	s.cloudDaemon = daemon
	s.cloudBusSub = id

	// Start presence reporter (shares the same bearer-auth client)
	authClient := cloud.NewAuthenticatedClient(tc.Token)
	reporter := cloud.NewPresenceReporter(cloudCfg, authClient)
	reporter.Start("", "serve")
	s.cloudPresence = reporter

	fmt.Fprintf(os.Stderr, "hero serve: cloud auto-sync and presence enabled\n")
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
//
// When the daemon was started with a registry, this also unregisters
// the slug from ~/.hero/projects.json and persists the change. The
// in-memory removal happens first; registry persistence is best-effort
// and surfaces an error on failure, but the in-memory removal is not
// rolled back.
func (s *Server) RemoveProject(slug string) error {
	s.mu.Lock()
	pc, ok := s.projects[slug]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("project %q not found", slug)
	}

	if pc.watcher != nil {
		pc.watcher.Stop()
	}
	if pc.refresher != nil {
		pc.refresher.Stop()
	}

	delete(s.projects, slug)
	reg := s.registry
	s.mu.Unlock()

	// Persist the registry change to disk. Skip when no registry is
	// wired (single-project mode launched without a registry path).
	if reg != nil {
		if reg.HasProject(slug) {
			if err := reg.Remove(slug); err != nil {
				return fmt.Errorf("remove %s from registry: %w", slug, err)
			}
			if err := reg.Save(); err != nil {
				return fmt.Errorf("persist registry after removing %s: %w", slug, err)
			}
		}
	}
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
			pc.refresher = StartImportRefresher(pc.Path, pc.Slug, s.bus)
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

		// Wire OAuth if configured via env.
		oauthClientID := os.Getenv("HERO_OAUTH_CLIENT_ID")
		oauthClientSecret := os.Getenv("HERO_OAUTH_CLIENT_SECRET")
		oauthProvider := os.Getenv("HERO_OAUTH_PROVIDER")
		if oauthClientID != "" && oauthClientSecret != "" && oauthProvider != "" {
			oauthCfg := &OAuthConfig{
				Provider:     oauthProvider,
				ClientID:     oauthClientID,
				ClientSecret: oauthClientSecret,
				Org:          os.Getenv("HERO_OAUTH_ORG"),
				HostedDomain: os.Getenv("HERO_OAUTH_HOSTED_DOMAIN"),
			}
			if redirectURI := os.Getenv("HERO_OAUTH_REDIRECT_URI"); redirectURI != "" {
				oauthRedirectURI = redirectURI
			}
			RegisterOAuthAPI(mux, s.jobQueue, jwtSecret, oauthCfg)
			fmt.Fprintf(os.Stderr, "hero serve: OAuth enabled (provider: %s)\n", oauthProvider)
		}

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

	// Start cloud auto-sync daemon if team connection and cloud config exist.
	s.startCloudSync()

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

		// Multi-project routing: /p/<slug>/<page> resolves to a per-
		// project shell router. /p/all/<page> renders cross-project
		// stubs (items 7 and 8 fill in the per-page aggregate views).
		topMux.Handle("/p/", s.projectHandler())

		// Legacy redirects: bookmarks to /now, /work, etc. land on the
		// default project's namespaced URL. Registered explicitly so
		// they take precedence over the catch-all "/" below.
		for _, legacy := range legacyPagePaths {
			lp := legacy
			topMux.HandleFunc(lp, func(w http.ResponseWriter, r *http.Request) {
				// Only rewrite the exact path — sub-paths under /work
				// (e.g. /work/spec/<slug>) still resolve via the
				// catch-all to the primary-project shell router
				// behavior for backwards compat.
				if r.URL.Path != lp {
					shellRouter.Handler().ServeHTTP(w, r)
					return
				}
				slug := s.resolveDefaultProjectSlug(r)
				if slug == "" {
					shellRouter.Handler().ServeHTTP(w, r)
					return
				}
				http.Redirect(w, r, "/p/"+slug+lp, http.StatusFound)
			})
		}

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
		return s.diagnoseBindError(err)
	}

	// Record start time and write PID file. Failure to write the PID
	// file is non-fatal — log and continue (lifecycle commands will
	// still work via the fallback HTTP probe).
	s.startedAt = time.Now().UTC()
	if pidPath, perr := WritePIDFile(s.port, s.version); perr != nil {
		fmt.Fprintf(os.Stderr, "hero serve: could not write pid file: %v\n", perr)
	} else {
		fmt.Fprintf(os.Stderr, "hero serve: pid file %s\n", pidPath)
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

	// Stop cloud presence and auto-sync
	if s.cloudPresence != nil {
		s.cloudPresence.Stop()
	}
	if s.cloudDaemon != nil {
		s.cloudDaemon.Stop()
		s.bus.Unsubscribe(s.cloudBusSub)
	}

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

	// Remove the PID file last. Tolerates missing file (already cleaned
	// by a concurrent stop, etc.).
	defer func() {
		if err := RemovePIDFile(s.port); err != nil {
			fmt.Fprintf(os.Stderr, "hero serve: could not remove pid file: %v\n", err)
		}
	}()

	var httpErr error
	if s.httpServer != nil {
		httpErr = s.httpServer.Shutdown(ctx)
	}

	// Reap in-flight ops subprocesses and wait for their pump goroutines
	// AFTER the HTTP server has drained, so no handler can spawn a new op
	// concurrently with the wait. The runner was scoped to
	// context.Background (nothing else cancels it), and its subprocesses
	// run in their own process groups — without this Stop they'd orphan
	// when the daemon exits.
	if s.opsRunner != nil {
		s.opsRunner.Stop()
	}

	return httpErr
}

// StartedAt returns the wall-clock time this server began listening.
// Zero value before Run is called.
func (s *Server) StartedAt() time.Time {
	return s.startedAt
}

// Version returns the daemon version string.
func (s *Server) Version() string {
	return s.version
}

// Port returns the port the daemon is listening on.
func (s *Server) Port() int {
	return s.port
}

// diagnoseBindError turns a net.Listen failure into actionable output.
// When the port is held by a hero daemon (probe of /api/status
// succeeds), the message names the running PID and points the user at
// hero serve stop / --force. Foreign holders get a different message.
func (s *Server) diagnoseBindError(listenErr error) error {
	if !isAddrInUse(listenErr) {
		return fmt.Errorf("listen 127.0.0.1:%d: %w", s.port, listenErr)
	}

	info := probeHeroDaemon(s.port)
	if info != nil {
		return fmt.Errorf(
			"a hero daemon is already running on 127.0.0.1:%d (PID %d) and serves all your projects — "+
				"you don't need a second one. Use `hero serve status` to inspect, or "+
				"`hero serve stop` (or `hero serve --force`) to terminate.",
			s.port, info.PID,
		)
	}
	return fmt.Errorf(
		"port %d is in use by another process (not a hero daemon). "+
			"Try `hero serve --port <other>` or free the port.",
		s.port,
	)
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

// buildShellRouter assembles the shell router for the daemon's primary
// project — the project the daemon was launched from. Per-project
// routers for the /p/<slug>/... routes are built by
// buildShellRouterFor below.
func (s *Server) buildShellRouter() *shell.Router {
	pc := &ProjectContext{
		Slug:    filepath.Base(s.projectRoot),
		Path:    s.projectRoot,
		HeroDir: s.heroDir,
	}
	return s.buildShellRouterForOpts(pc, true)
}

// buildShellRouterFor assembles a shell router scoped to a single
// project. Each page's Deps capture the project's ProjectRoot / HeroDir
// via closures, so routers must be one-per-project and cached by slug
// (see projectRouterCache).
//
// Hooks that are inherently daemon-global (chat registry, live-session
// snapshots, chrome workspace label) pull from the Server. Hooks that
// are inherently per-project (proposal snapshots, project root paths)
// pull from the ProjectContext.
func (s *Server) buildShellRouterFor(pc *ProjectContext) *shell.Router {
	return s.buildShellRouterForOpts(pc, false)
}

// buildShellRouterForOpts is the variant that lets callers mark a
// router as the daemon's single-project fallback. Only Phase 4 of
// hero-serve-project-section uses the flag — see Deps.IsFallbackProject.
func (s *Server) buildShellRouterForOpts(pc *ProjectContext, isFallback bool) *shell.Router {
	ed := edition.Resolve()

	// Session store is best-effort. A nil store still serves pages
	// (the shell silently skips writes) — the / redirect just falls
	// back to /now.
	store, err := session.Open("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: shell session store unavailable: %v\n", err)
		store = nil
	}

	workspace := filepath.Base(pc.Path)
	if workspace == "" {
		workspace = "hero"
	}
	branch := detectGitBranch(pc.Path)
	userName := shellUserName()

	r := shell.New(ed, store, workspace, branch, userName, s.version)
	r.SetAdapterProbe(s.shellAdapterState)
	r.SetProjectSelectorProbe(s.projectSelectorFor(pc.Slug))
	shell.RegisterStubHomes(r)

	// Register the real Now home in place of its (no-longer-present)
	// stub. Wired with a per-project proposal-store snapshotter that
	// surfaces pending proposals in the Needs-your-input section.
	nowDeps := nowpage.Deps{
		ProjectRoot:  pc.Path,
		HeroDir:      pc.HeroDir,
		Workspace:    workspace,
		Branch:       branch,
		UserName:     userName,
		Proposals:    s.proposalsForProject(pc.Slug),
		ChatRegistry: s.chatRegistry,
		LiveSessions: s.snapshotLiveSessions,
	}
	if err := nowpage.Register(r, nowDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Now home for %s: %v\n", pc.Slug, err)
	}

	// Register the real Work home in place of its (no-longer-present)
	// stub. Per-spec: this owns /work; sibling /work/* routes land in
	// follow-on work.
	workDeps := workpage.Deps{
		ProjectRoot:              pc.Path,
		HeroDir:                  pc.HeroDir,
		Workspace:                workspace,
		Branch:                   branch,
		UserName:                 userName,
		ChatInteractiveConnected: s.chatInteractiveConnected,
		HasSprintConfig:          hasSprintConfig(pc.Path),
	}
	if err := workpage.Register(r, workDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Work home for %s: %v\n", pc.Slug, err)
	}

	// Register the real Knowledge home in place of its (no-longer-present)
	// stub. Deps mirror the Now wiring — no proposal-store hook yet.
	knowledgeDeps := knowledgepage.Deps{
		ProjectRoot:              pc.Path,
		HeroDir:                  pc.HeroDir,
		Workspace:                workspace,
		Branch:                   branch,
		UserName:                 userName,
		ChatInteractiveConnected: s.chatInteractiveConnected,
	}
	if err := knowledgepage.Register(r, knowledgeDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Knowledge home for %s: %v\n", pc.Slug, err)
	}

	// Register the real People & ROI home in place of its (no-longer-present)
	// stub. Deps mirror the other home wiring.
	peopleDeps := peoplepage.Deps{
		ProjectRoot:              pc.Path,
		HeroDir:                  pc.HeroDir,
		Workspace:                workspace,
		Branch:                   branch,
		UserName:                 userName,
		ChatInteractiveConnected: s.chatInteractiveConnected,
	}
	if err := peoplepage.Register(r, peopleDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register People & ROI home for %s: %v\n", pc.Slug, err)
	}

	// Register the real Agents home in place of its (no-longer-present)
	// stub. Wired with the live-session snapshot reader (which is the
	// canonical "live ledger" the Now home will consume in a follow-up)
	// plus a proposal snapshotter. Scheduled / automation hooks are left
	// nil until those engines land per the spec's build order; the page
	// renders empty-state notices when they are.
	agentsDeps := agentspage.Deps{
		ProjectRoot:              pc.Path,
		HeroDir:                  pc.HeroDir,
		Workspace:                workspace,
		Branch:                   branch,
		UserName:                 userName,
		LiveSessions:             s.snapshotLiveSessions,
		Proposals:                s.agentsProposalsForProject(pc.Slug),
		ChatInteractiveConnected: s.chatInteractiveConnected,
	}
	if err := agentspage.Register(r, agentsDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Agents home for %s: %v\n", pc.Slug, err)
	}

	// Register the Project section page at /project — read-only
	// per-project home with eight stacked sections (identity, health,
	// stack, registry, peers, trackers, knowledge, config). Phase 1 of
	// hero-serve-project-section. Registered BEFORE Rollup so it
	// appears ahead of Rollup in the top-nav tab order — Project is the
	// primary per-project surface; Rollup is the legacy project-shape
	// rollup retained for discoverability.
	projectDeps := projectpage.Deps{
		ProjectRoot:       pc.Path,
		HeroDir:           pc.HeroDir,
		Slug:              pc.Slug,
		RegistryEntry:     s.projectpageRegistryEntry(pc.Slug),
		OpsRunner:         s.opsRunner,
		IsFallbackProject: isFallback,
		HealthCache:       healthCacheAdapter{cache: s.healthCache},
		PeerCache:         peerCacheAdapter{cache: s.healthCache},
	}
	if err := projectpage.Register(r, projectDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Project section page for %s: %v\n", pc.Slug, err)
	}

	// Register the Rollup home — surfaces table, initiatives, archives
	// timeline. Owned by the project-snapshot spec, mounted at /rollup
	// (previously /project; renamed when the per-project section page
	// took over that slot — see hero-serve-project-section). Archive
	// bodies render ONLY at /rollup/snapshots/<date>; the home itself
	// shows metadata only, per the isolation invariants.
	rollupDeps := rolluppage.Deps{
		ProjectRoot: pc.Path,
		HeroDir:     pc.HeroDir,
		Workspace:   workspace,
		Branch:      branch,
		UserName:    userName,
	}
	if err := rolluppage.Register(r, rollupDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register Rollup home for %s: %v\n", pc.Slug, err)
	}
	return r
}

// healthCacheAdapter bridges the projectpage data loader's interface
// shape (which uses data.CachedHealth, declared in the data package to
// avoid a circular import) to the concrete healthcache.Cache.
type healthCacheAdapter struct{ cache *healthcache.Cache }

func (a healthCacheAdapter) Health(slug string) (projectpagedata.CachedHealth, bool) {
	if a.cache == nil {
		return projectpagedata.CachedHealth{}, false
	}
	r, ok := a.cache.Health(slug)
	if !ok {
		return projectpagedata.CachedHealth{}, false
	}
	rows := make([]projectpagedata.HealthRow, len(r.Rows))
	for i, row := range r.Rows {
		rows[i] = projectpagedata.HealthRow(row)
	}
	return projectpagedata.CachedHealth{
		Captured:  r.Captured,
		Rows:      rows,
		FromDisk:  r.FromDisk,
		Timestamp: r.Timestamp,
		TTL:       r.TTL,
	}, true
}

// peerCacheAdapter mirrors healthCacheAdapter for peer probes.
type peerCacheAdapter struct{ cache *healthcache.Cache }

func (a peerCacheAdapter) Peer(slug, alias string) (projectpagedata.CachedPeer, bool) {
	if a.cache == nil {
		return projectpagedata.CachedPeer{}, false
	}
	r, ok := a.cache.Peer(slug, alias)
	if !ok {
		return projectpagedata.CachedPeer{}, false
	}
	return projectpagedata.CachedPeer{
		Reachable: r.Reachable,
		LastOK:    r.LastOK,
		LastError: r.LastError,
		Timestamp: r.Timestamp,
		TTL:       r.TTL,
	}, true
}

// projectpageRegistryEntry adapts the daemon-side ProjectEntry into the
// projectpage's read-only view. Returns nil when the project isn't in
// the global registry — projectpage renders the "not registered"
// empty state in that case.
func (s *Server) projectpageRegistryEntry(slug string) *projectpage.RegistryEntry {
	if s == nil || s.registry == nil || slug == "" {
		return nil
	}
	entry := s.registry.Get(slug)
	if entry == nil {
		return nil
	}
	return &projectpage.RegistryEntry{
		Path:         entry.Path,
		RegisteredAt: entry.Registered,
	}
}

// proposalsForProject returns a per-project proposals-snapshotter that
// the Now home reads via Deps. Factored out of snapshotProposals so
// the per-project shell router can scope to the right slug.
func (s *Server) proposalsForProject(slug string) func() []*nowdata.ProposalRow {
	return func() []*nowdata.ProposalRow {
		if s == nil || s.api == nil || s.api.proposals == nil || slug == "" {
			return nil
		}
		envs := s.api.proposals.snapshotProject(slug)
		if len(envs) == 0 {
			return nil
		}
		rows := make([]*nowdata.ProposalRow, 0, len(envs))
		for _, e := range envs {
			if e == nil {
				continue
			}
			rows = append(rows, &nowdata.ProposalRow{
				ProposalID:  e.ProposalID,
				SessionID:   e.SessionID,
				SpecSlug:    e.Target.SpecSlug,
				Agent:       e.Agent,
				AnchorValue: e.Target.Anchor.Value,
				EmittedAt:   e.EmittedAt,
				BatchID:     e.BatchID,
			})
		}
		return rows
	}
}

// agentsProposalsForProject is the Agents-home equivalent of
// proposalsForProject. The propose store doesn't yet expose a global
// enumerator, so this returns nil today; the seam is in place for the
// follow-up to fill it.
func (s *Server) agentsProposalsForProject(slug string) func() []agentsdata.ProposalRow {
	return func() []agentsdata.ProposalRow {
		if s == nil || s.api == nil || s.api.proposals == nil || slug == "" {
			return nil
		}
		store := s.api.proposals.get(slug)
		if store == nil {
			return nil
		}
		_ = store
		return nil
	}
}

// projectRouterCacheRouter returns (and lazily builds) the per-project
// shell router for pc. Wired by Server.Run after the primary shell
// router is constructed.
func (s *Server) projectRouterCacheRouter(pc *ProjectContext) *shell.Router {
	if s.projectRouters == nil {
		s.projectRouters = newProjectRouterCache(s.buildShellRouterFor)
	}
	return s.projectRouters.get(pc)
}

// aggregateRouter returns (and lazily builds) the cross-project
// aggregate shell router used by /p/all/<page>. Each render rebuilds
// the underlying MultiProject slice from the current project registry,
// so a project added or removed at runtime is reflected on the next
// request. Returns nil when no projects are registered.
func (s *Server) aggregateRouter() *shell.Router {
	pc := &ProjectContext{
		Slug:    AllProjectsSlug,
		Path:    s.projectRoot,
		HeroDir: s.heroDir,
	}
	r := s.buildAggregateShellRouter(pc)
	return r
}

// buildAggregateShellRouter assembles a shell router for the
// cross-project /p/all/ aggregate view. The Now and Work homes are
// registered with MultiProject populated from the live project
// registry, so the page-data loaders fan out across every project.
func (s *Server) buildAggregateShellRouter(pc *ProjectContext) *shell.Router {
	ed := edition.Resolve()
	store, err := session.Open("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: shell session store unavailable (aggregate): %v\n", err)
		store = nil
	}
	workspace := "All projects"
	branch := ""
	userName := shellUserName()

	r := shell.New(ed, store, workspace, branch, userName, s.version)
	r.SetAdapterProbe(s.shellAdapterState)
	r.SetProjectSelectorProbe(s.projectSelectorFor(AllProjectsSlug))
	shell.RegisterStubHomes(r)

	mp := s.aggregateProjects()

	nowDeps := nowpage.Deps{
		ProjectRoot:  pc.Path,
		HeroDir:      pc.HeroDir,
		Workspace:    workspace,
		Branch:       branch,
		UserName:     userName,
		ChatRegistry: s.chatRegistry,
		LiveSessions: s.snapshotLiveSessions,
		MultiProject: mp,
	}
	if err := nowpage.Register(r, nowDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register aggregate Now home: %v\n", err)
	}

	workDeps := workpage.Deps{
		ProjectRoot:              pc.Path,
		HeroDir:                  pc.HeroDir,
		Workspace:                workspace,
		Branch:                   branch,
		UserName:                 userName,
		ChatInteractiveConnected: s.chatInteractiveConnected,
		MultiProject:             toWorkProjects(mp),
		// Aggregate view: sprint UI never makes sense across multiple
		// projects (each has its own sprint config), so the gate stays
		// off regardless of any single project's setting.
		HasSprintConfig: false,
	}
	if err := workpage.Register(r, workDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register aggregate Work home: %v\n", err)
	}

	// Register the aggregate Project section page at /project (served
	// under /p/all/project after the routing rewrite). Phase 2 of
	// hero-serve-project-section — cross-project directory + daemon ops
	// + health rollup + peers map.
	projectDeps := projectpage.AggregateDeps{
		Projects:       s.aggregateProjectpageProjects(),
		DaemonSnapshot: s.daemonOpsSnapshot,
	}
	if err := projectpage.RegisterAggregate(r, projectDeps); err != nil {
		fmt.Fprintf(os.Stderr, "hero serve: register aggregate Project home: %v\n", err)
	}
	return r
}

// aggregateProjectpageProjects snapshots the current project registry
// into the shape the projectpage aggregate loaders consume. Built per
// request so a project added/removed at runtime reflects on the next
// page load.
func (s *Server) aggregateProjectpageProjects() []projectpagedata.DirectoryProject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]projectpagedata.DirectoryProject, 0, len(s.projects))
	for slug, pc := range s.projects {
		if pc == nil {
			continue
		}
		dp := projectpagedata.DirectoryProject{
			Slug:        slug,
			ProjectRoot: pc.Path,
			HeroDir:     pc.HeroDir,
		}
		if s.registry != nil {
			if entry := s.registry.Get(slug); entry != nil {
				dp.RegisteredAt = entry.Registered
			}
		}
		out = append(out, dp)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

// daemonOpsSnapshot is the in-process snapshot the projectpage Daemon
// Ops section consumes. Same source of truth as /api/status — no HTTP
// round-trip, no network.
func (s *Server) daemonOpsSnapshot() *projectpagedata.DaemonOpsSnapshot {
	if s == nil {
		return nil
	}
	started := s.StartedAt()
	var uptime int64
	if !started.IsZero() {
		uptime = int64(time.Since(started).Seconds())
	}
	return &projectpagedata.DaemonOpsSnapshot{
		PID:           os.Getpid(),
		Port:          s.Port(),
		Version:       s.Version(),
		StartedAt:     started,
		UptimeSeconds: uptime,
		ProjectCount:  s.ProjectCount(),
	}
}

// aggregateProjects snapshots the current project registry into the
// shape the page data loaders consume.
func (s *Server) aggregateProjects() []nowdata.ActivityProject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]nowdata.ActivityProject, 0, len(s.projects))
	for slug, pc := range s.projects {
		if pc == nil {
			continue
		}
		out = append(out, nowdata.ActivityProject{
			Slug:    slug,
			Path:    pc.Path,
			HeroDir: pc.HeroDir,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

// toWorkProjects shifts the now-data shape into the work-data shape so
// the two packages stay leaf-free of each other.
func toWorkProjects(in []nowdata.ActivityProject) []workpage.AggregateProject {
	out := make([]workpage.AggregateProject, 0, len(in))
	for _, p := range in {
		out = append(out, workpage.AggregateProject{
			Slug:    p.Slug,
			Path:    p.Path,
			HeroDir: p.HeroDir,
		})
	}
	return out
}

// projectSelectorFor returns the probe a project-scoped shell router
// uses to populate the top-nav dropdown on every page render. The
// activeSlug is baked into the closure so each per-project router
// reports the right "active" entry without re-reading the URL.
func (s *Server) projectSelectorFor(activeSlug string) func(*http.Request) shell.ProjectSelector {
	return func(req *http.Request) shell.ProjectSelector {
		slugs := s.Projects()
		sort.Strings(slugs)

		options := make([]shell.ProjectSelectorOption, 0, len(slugs)+1)
		// "All projects" first.
		options = append(options, shell.ProjectSelectorOption{
			Slug:  AllProjectsSlug,
			Label: "All projects",
		})
		for _, sl := range slugs {
			options = append(options, shell.ProjectSelectorOption{
				Slug:  sl,
				Label: sl,
			})
		}

		// Derive the current page (the part after /p/<slug>/) from the
		// request URL so the dropdown's navigation preserves which
		// page the user is on. Legacy URLs (/now etc.) supply the
		// page directly.
		currentPage := currentPageFromRequest(req, activeSlug)

		label := activeSlug
		if activeSlug == AllProjectsSlug {
			label = "All projects"
		}

		return shell.ProjectSelector{
			Active:      activeSlug,
			ActiveLabel: label,
			CurrentPage: currentPage,
			Options:     options,
		}
	}
}

// currentPageFromRequest extracts the inner page slug from a request,
// handling both /p/<slug>/<page> and legacy /<page> URLs. Returns "now"
// when the path can't be parsed — the safest landing page.
func currentPageFromRequest(req *http.Request, activeSlug string) string {
	if req == nil || req.URL == nil {
		return "now"
	}
	p := strings.TrimPrefix(req.URL.Path, "/")
	if p == "" {
		return "now"
	}
	// /p/<slug>/<page>/... — explicit project-namespaced path. After
	// path rewriting in projectHandler we usually see only the inner
	// path, so this branch only matches the legacy / pre-rewrite URLs.
	if strings.HasPrefix(p, "p/") || p == "p" {
		rest := strings.TrimPrefix(p, "p/")
		// /p/<slug> with no trailing path → default page "now".
		if !strings.Contains(rest, "/") {
			return "now"
		}
		_, after := splitFirst(rest, "/")
		page, _ := splitFirst(after, "/")
		if page == "" {
			return "now"
		}
		return page
	}
	// /<page>/... — first segment is the page.
	page, _ := splitFirst(p, "/")
	if page == "" {
		return "now"
	}
	return page
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
	envs := s.api.proposals.snapshotProject(slug)
	if len(envs) == 0 {
		return nil
	}
	rows := make([]*nowdata.ProposalRow, 0, len(envs))
	for _, e := range envs {
		if e == nil {
			continue
		}
		rows = append(rows, &nowdata.ProposalRow{
			ProposalID:  e.ProposalID,
			SessionID:   e.SessionID,
			SpecSlug:    e.Target.SpecSlug,
			Agent:       e.Agent,
			AnchorValue: e.Target.Anchor.Value,
			EmittedAt:   e.EmittedAt,
			BatchID:     e.BatchID,
		})
	}
	return rows
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

// shellAdapterState is the probe the top-nav adapter chip reads on
// every request. Returns Connected=true with the adapter type name
// (e.g. "hero-code") when chat.Resolve picks an interactive adapter,
// otherwise Connected=false with empty DisplayName (chip renders muted
// "no adapter"). Same source of truth as chatInteractiveConnected and
// the Now install-panel — chrome and body widgets agree by construction.
func (s *Server) shellAdapterState() shell.AdapterState {
	if s == nil || s.chatRegistry == nil {
		return shell.AdapterState{}
	}
	cap := chat.Resolve(s.chatRegistry, "")
	if cap.Interactive == "" {
		return shell.AdapterState{}
	}
	display := lookupAdapterDisplayName(cap.Adapters, cap.Interactive)
	return shell.AdapterState{Connected: true, DisplayName: display}
}

// lookupAdapterDisplayName returns the adapter TYPE (e.g. "hero-code")
// for the given connection id. Falls back to the connection id when no
// match is found — keeps the chip text non-empty even on unexpected
// registry shapes.
func lookupAdapterDisplayName(adapters []chat.AdapterInfo, id string) string {
	for _, a := range adapters {
		if a.ID == id {
			return a.Adapter
		}
	}
	return id
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

// shellUserName returns the canonical workspace identity for the
// dashboard "you" surfaces (avatar, plate, author-filtered metrics).
//
// Resolution lives in gitutil.UserName and prefers
// `git config user.name` — the same source every event/claim writer
// uses — so the reader and writer namespaces stay reconciled. Falls
// back to `$USER` then `"unknown"` when git config is unavailable
// (fresh checkouts, CI containers).
//
// Logs a single diagnostic on first call when git config is unset so
// operators can spot misconfigured workspaces without per-request
// noise.
func shellUserName() string {
	name := gitutil.UserName()
	logIdentityFallbackOnce(name)
	return name
}

var identityFallbackLogged sync.Once

// logIdentityFallbackOnce emits one diagnostic line if the resolved
// identity fell back past `git config user.name`. Quiet otherwise.
// hasSprintConfig reads the project's hero.json and reports whether
// the workspace has opted into sprint UI. A load error or absent
// sprint block both fall back to false — the rolling-window-only
// default the dashboard redesign assumes.
func hasSprintConfig(projectRoot string) bool {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return false
	}
	return cfg.HasSprintConfig()
}

func logIdentityFallbackOnce(resolved string) {
	if resolved == "" || resolved == "unknown" {
		identityFallbackLogged.Do(func() {
			fmt.Fprintf(os.Stderr,
				"hero serve: git config user.name unset; dashboard \"you\" identity falling back to %q\n",
				resolved)
		})
	}
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
