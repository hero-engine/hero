package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve"
	"github.com/spf13/cobra"
)

var (
	servePort    int
	serveNoWatch bool
	serveNoUI    bool
	serveAdd     string
	serveRemove  string
	serveList    bool
	serveTeam    bool
	serveWorkers int
	serveAuthToken string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Hero daemon (HTTP API + MCP server + file watcher)",
	Long: `Starts a local daemon that bundles the HTTP API, dashboard UI, and file watcher.

The daemon manages all registered Hero projects on this machine from a single
process and port. Projects are registered with --add and listed with --list.

Modes:

  hero serve              Start HTTP API + dashboard + file watcher on localhost:7437
  hero serve --port 8080  Start on a custom port
  hero serve --no-ui      Disable the embedded dashboard (API-only mode)

Project management:

  hero serve --add .           Register current directory as a project
  hero serve --add /path/to   Register a specific directory
  hero serve --remove myapp   Unregister a project by slug
  hero serve --list            List all registered projects

Endpoints (HTTP mode):
  GET /                          Dashboard UI (disable with --no-ui)
  GET /health                    Daemon health check
  GET /api/projects              List all registered projects
  GET /api/{project}/status      Workspace summary
  GET /api/{project}/specs       List specs (with ?type, ?status, ?tag filters)
  GET /api/{project}/specs/:slug Get single spec detail
  GET /api/{project}/search?q=   Full-text search
  GET /api/{project}/context?f=  Context block for files
  GET /api/{project}/check       Health check results
  GET /api/{project}/knowledge   Knowledge entries
  GET /api/{project}/inventory   Bug inventory
  GET /api/events                SSE event stream (all projects, ?project= to filter)`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 0, "HTTP port (default 7437)")
	serveCmd.Flags().BoolVar(&serveNoWatch, "no-watch", false, "disable automatic file watching")
	serveCmd.Flags().BoolVar(&serveNoUI, "no-ui", false, "disable the embedded dashboard UI")
	serveCmd.Flags().StringVar(&serveAdd, "add", "", "register a project directory")
	serveCmd.Flags().StringVar(&serveRemove, "remove", "", "unregister a project by slug")
	serveCmd.Flags().BoolVar(&serveList, "list", false, "list all registered projects")
	serveCmd.Flags().BoolVar(&serveTeam, "team", false, "enable team mode (job queue, workers, auth)")
	serveCmd.Flags().IntVar(&serveWorkers, "workers", 1, "number of job execution workers (team mode)")
	serveCmd.Flags().StringVar(&serveAuthToken, "auth-token", "", "require this token for API access (team mode)")
	serveCmd.Flags().BoolVar(&serveForce, "force", false, "stop any existing daemon on the target port before starting")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Handle project management subcommands first
	if serveList {
		return runServeList()
	}
	if serveAdd != "" {
		return runServeAdd(serveAdd)
	}
	if serveRemove != "" {
		return runServeRemove(serveRemove)
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := filepath.Join(projectRoot, cfg.Folder)

	// Check that .hero directory exists
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("hero workspace not found at %s (run 'hero init' first)", heroDir)
	}

	// HTTP + watcher + dashboard mode
	port := servePort
	if port == 0 {
		if cfg.Serve != nil && cfg.Serve.Port > 0 {
			port = cfg.Serve.Port
		} else {
			port = 7437
		}
	}

	// --force: stop any existing daemon on this port before starting.
	// Errors stopping the running daemon are surfaced but non-fatal —
	// the bind step that follows will report the real outcome.
	if serveForce {
		if err := stopDaemon(port, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "hero serve --force: stop failed: %v (proceeding to start anyway)\n", err)
		}
	}

	autoWatch := !serveNoWatch
	if cfg.Serve != nil && !cfg.Serve.AutoWatch {
		autoWatch = false
	}
	// CLI flag --no-watch overrides config
	if serveNoWatch {
		autoWatch = false
	}

	uiEnabled := cfg.Serve.UIEnabled()
	if serveNoUI {
		uiEnabled = false
	}

	version := rootCmd.Version
	if version == "" {
		version = "dev"
	}

	srv := serve.NewServer(serve.ServerConfig{
		HeroDir:     heroDir,
		ProjectRoot: projectRoot,
		Version:     version,
		Port:        port,
		AutoWatch:   autoWatch,
		UIEnabled:   uiEnabled,
		TeamMode:    serveTeam,
		Workers:     serveWorkers,
		AuthToken:   serve.ResolveAuthToken(serveAuthToken),
	})

	// Auto-register this project in the global registry
	reg, err := serve.LoadRegistry()
	if err == nil {
		if _, err := reg.Add(projectRoot); err == nil {
			reg.Save()
		}
	}

	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return srv.Run(ctx)
}

func runServeList() error {
	reg, err := serve.LoadRegistry()
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	projects := reg.List()
	if len(projects) == 0 {
		fmt.Println("No projects registered. Use 'hero serve --add .' to register one.")
		return nil
	}

	fmt.Printf("Registered projects (%d):\n\n", len(projects))
	for slug, entry := range projects {
		fmt.Printf("  %-20s  %s\n", slug, entry.Path)
	}
	return nil
}

func runServeAdd(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	reg, err := serve.LoadRegistry()
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	slug, err := reg.Add(absPath)
	if err != nil {
		return err
	}

	if err := reg.Save(); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	fmt.Printf("Registered project %q (%s)\n", slug, absPath)
	return nil
}

func runServeRemove(slug string) error {
	reg, err := serve.LoadRegistry()
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	if err := reg.Remove(slug); err != nil {
		return err
	}

	if err := reg.Save(); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	fmt.Printf("Unregistered project %q\n", slug)
	return nil
}
