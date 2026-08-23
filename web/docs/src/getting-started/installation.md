# Installation

Prebuilt Hero binaries do not require a Go toolchain. The current released
version is derived on the [Build information](../about/build.md) page from the
latest release tag.

## macOS — Homebrew

```bash
brew install hero-engine/tap/hero
```

## Linux — install script or Homebrew

```bash
curl -fsSL https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.sh | sh
```

The script detects the platform, downloads the release archive, verifies its
SHA-256 checksum, and installs to a writable binary directory. Homebrew on Linux
is also supported:

```bash
brew install hero-engine/tap/hero
```

## Windows — Scoop or PowerShell

```powershell
scoop bucket add hero-engine https://github.com/hero-engine/scoop-bucket
scoop install hero
```

Or run the published PowerShell installer:

```powershell
irm https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.ps1 | iex
```

## Direct download

Release archives are published at
[hero-engine/hero-releases](https://github.com/hero-engine/hero-releases/releases).
Extract `hero` or `hero.exe` and place it on `PATH`.

## Build from source

A source build requires the Go version declared by the module—currently Go
1.26.4:

```bash
go build ./cmd/hero
```

## Verify the binary and workspace

```bash
hero --version
hero doctor   # binary, schema, and installed-target diagnosis
hero check    # workspace health and documentation/spec hygiene
```

## Monorepos

Initialize one Hero workspace at the repository root. From that root, use
`hero install satellites` to expose harness content when a session opens inside
a subproject. Satellites are thin trees pointing to the root content, not
nested `.hero` corpora. Use `--migrate-nested` to inspect a repository that
already contains legacy nested workspaces.

Next: [Project Setup](project-setup.md).
