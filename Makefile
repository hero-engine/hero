BINARY  := hero
CLOUD   := hero-cloud
MODULE  := github.com/hero-engine/hero
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build cloud clean install bootstrap test tidy dist release snapshot dev smoke smoke-all

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/hero/

cloud:
	go build $(LDFLAGS) -o $(CLOUD) ./cmd/hero-cloud/

clean:
	rm -f $(BINARY) $(CLOUD)
	rm -rf dist/

install: build
	rm -f $(GOPATH)/bin/$(BINARY) ~/go/bin/$(BINARY) 2>/dev/null; \
	cp $(BINARY) $(GOPATH)/bin/ 2>/dev/null || { mkdir -p ~/go/bin && cp $(BINARY) ~/go/bin/; }

# Bootstrap a fresh clone: build hero, then materialize .hero/{agents,
# commands,skills}/ and the .claude/* symlinks so working in this repo
# with Claude Code (or any other Hero-native harness) "just works."
# Idempotent — re-running is a no-op when content is unchanged.
bootstrap: build
	./$(BINARY) install project . --target claude --force --force-managed

test:
	go test ./...

# Run per-feature smokes for files changed since origin/main (mirrors PR CI).
smoke: build
	./$(BINARY) smoke --since origin/main

# Run every per-feature smoke (mirrors nightly CI).
smoke-all: build
	./$(BINARY) smoke --all

tidy:
	go mod tidy

# Local dev: start CockroachDB + hero-cloud
dev:
	docker compose up -d cockroachdb init-db
	@echo "Waiting for CockroachDB..."
	@sleep 3
	go run $(LDFLAGS) ./cmd/hero-cloud/

# Stop local dev services
dev-stop:
	docker compose down

# Cross-compile for common targets
dist:
	mkdir -p dist/
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/hero-darwin-arm64  ./cmd/hero/
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/hero-darwin-amd64  ./cmd/hero/
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/hero-linux-amd64   ./cmd/hero/
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/hero-linux-arm64   ./cmd/hero/
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/hero-windows-amd64.exe ./cmd/hero/

# Release with goreleaser (requires GITHUB_TOKEN)
release:
	goreleaser release --clean

# Build a snapshot release (no publish)
snapshot:
	goreleaser release --snapshot --clean
