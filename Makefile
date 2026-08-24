BINARY  := hero
MODULE  := github.com/hero-engine/hero
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build clean install bootstrap test tidy dist release snapshot smoke smoke-all sync-install-scripts

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/hero/

clean:
	rm -f $(BINARY)
	rm -rf dist/

install: build
	rm -f $(GOPATH)/bin/$(BINARY) ~/go/bin/$(BINARY) 2>/dev/null; \
	cp $(BINARY) $(GOPATH)/bin/ 2>/dev/null || { mkdir -p ~/go/bin && cp $(BINARY) ~/go/bin/; }

# Bootstrap a fresh clone: build hero, then materialize .hero/{agents,
# commands,skills}/ and the .claude/* symlinks so working in this repo
# with Claude Code (or any other Hero-native harness) "just works."
# Idempotent — re-running is a no-op when content is unchanged.
bootstrap: build
	./$(BINARY) install project . --target claude --force

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

# Sync install.sh/install.ps1 to hero-engine/hero-releases (the public URL
# users curl/iex from). The canonical source lives in scripts/; the
# release repo is a mirror so the raw URLs remain stable independently of
# source-repository layout. Re-run after editing either script.
sync-install-scripts:
	@tmp=$$(mktemp -d) && \
	git clone -q git@github.com-hero-engine:hero-engine/hero-releases.git $$tmp && \
	cp scripts/install.sh scripts/install.ps1 $$tmp/ && \
	cd $$tmp && \
	if git diff --quiet; then echo "install scripts already in sync"; else \
	  git add install.sh install.ps1 && \
	  git -c user.email=277887514+chet-bellows@users.noreply.github.com -c user.name=hero-engine-bot commit -q -m "Sync install scripts from hero@$$(cd - >/dev/null && git rev-parse --short HEAD)" && \
	  git push -q origin main && \
	  echo "synced install scripts to hero-releases"; \
	fi; \
	rm -rf $$tmp

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
