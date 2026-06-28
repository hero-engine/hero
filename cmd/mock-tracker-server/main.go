// Command mock-tracker-server is a single-binary, offline HTTP fake that
// speaks the GitHub / Jira / Linear / GitLab API subset hero's tracker
// adapters call, backed by an in-memory SQLite DB seeded by sprout-Go.
//
// It lives in cmd/ so `go build ./...` for hero proper never links it
// into the main hero binary (AC-12). CI builds it as a separate artifact.
//
// Usage:
//
//	mock-tracker-server --port 0 [--seed <dir>] [--single-mode <mode>]
//	                    [--require-token <tok>] [--log-requests]
//
// With --port 0 the OS picks a free port; the chosen port is printed on
// stdout as "listening on :PORT" so a harness can capture it.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/hero-engine/hero/internal/mocktracker"
)

func main() {
	var (
		port         = flag.Int("port", 0, "TCP port to listen on (0 = OS-assigned)")
		seedDir      = flag.String("seed", "", "directory of Sprout seed YAML (default: embedded Acme fixture)")
		singleMode   = flag.String("single-mode", "", "serve a single tracker without its URL prefix: github|jira|linear|gitlab")
		requireToken = flag.String("require-token", "", "require this exact token (default: any non-empty token)")
		logRequests  = flag.Bool("log-requests", false, "log each request as one JSON line on stderr")
	)
	flag.Parse()

	switch *singleMode {
	case "", "github", "jira", "linear", "gitlab":
	default:
		fmt.Fprintf(os.Stderr, "invalid --single-mode %q (want github|jira|linear|gitlab)\n", *singleMode)
		os.Exit(2)
	}

	ctx := context.Background()
	srv, err := mocktracker.NewServer(ctx, mocktracker.Options{
		SeedDir:      *seedDir,
		SingleMode:   *singleMode,
		RequireToken: *requireToken,
		LogRequests:  *logRequests,
	})
	if err != nil {
		log.Fatalf("mock-tracker-server: %v", err)
	}
	defer srv.Close()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("mock-tracker-server: listen: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	// AC-1: report the chosen port on stdout.
	fmt.Printf("listening on :%d\n", addr.Port)

	if err := http.Serve(ln, srv); err != nil {
		log.Fatalf("mock-tracker-server: serve: %v", err)
	}
}
