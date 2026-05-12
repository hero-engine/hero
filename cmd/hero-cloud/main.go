package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hero-engine/hero/cloud/api"
	"github.com/hero-engine/hero/cloud/store"
)

var version = "dev"

func main() {
	var (
		addr    = flag.String("addr", ":8080", "listen address")
		dbURL   = flag.String("db", envOr("HERO_DB_URL", "postgresql://hero:hero@localhost:26257/hero?sslmode=disable"), "CockroachDB connection string")
		migrate = flag.Bool("migrate", false, "run database migrations and exit")
	)
	flag.Parse()

	log.Printf("hero-cloud %s starting", version)

	// Connect to CockroachDB
	db, err := store.Connect(context.Background(), *dbURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()
	log.Printf("connected to database")

	// Run migrations if requested
	if *migrate {
		if err := db.Migrate(context.Background()); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		log.Printf("migrations complete")
		return
	}

	// Ensure migrations are current
	if err := db.Migrate(context.Background()); err != nil {
		log.Fatalf("auto-migration failed: %v", err)
	}

	// Build HTTP server
	router := api.NewRouter(db, version)
	srv := &http.Server{
		Addr:         *addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("listening on %s", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-done
	log.Printf("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Printf("stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "hero-cloud %s\n\nUsage:\n", version)
		flag.PrintDefaults()
	}
}
