# Installation

## macOS — Homebrew

```bash
brew install hero-engine/tap/hero
```

Publishes prebuilt binaries for macOS (arm64, amd64) on every release.

## Linux — Install script or Homebrew

```bash
curl -fsSL https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.sh | sh
```

Detects your OS/arch, downloads the right tarball, verifies the SHA-256
checksum, and installs to `/usr/local/bin` (falls back to `~/.local/bin`
if `/usr/local/bin` is not writable).

If you already use [Homebrew on Linux](https://docs.brew.sh/Homebrew-on-Linux),
the same formula works:

```bash
brew install hero-engine/tap/hero
```

Pin a version or override the install location with environment variables:

```bash
curl -fsSL https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.sh | \
  HERO_VERSION=v0.9.1 HERO_INSTALL="$HOME/.local/bin" sh
```

## Windows — Scoop or install script

[Scoop](https://scoop.sh):

```powershell
scoop bucket add hero-engine https://github.com/hero-engine/scoop-bucket
scoop install hero
```

PowerShell install script:

```powershell
irm https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.ps1 | iex
```

Installs to `$env:LOCALAPPDATA\Programs\hero` and adds that path to your
user PATH. Open a new terminal after install for the PATH change to take
effect.

## Direct download

Prebuilt binaries for every OS/arch are attached to each release at
[hero-engine/hero-releases](https://github.com/hero-engine/hero-releases/releases).
Download the archive for your platform, extract `hero` (or `hero.exe`),
and put it somewhere on your PATH.

## Verify

```bash
hero --version
```

You should see the installed version printed. If not, confirm the
install location is on your PATH.

## Monorepo setup

Install Hero on each platform first using the [macOS](#macos-homebrew),
[Linux](#linux-install-script-or-homebrew), or [Windows](#windows-scoop-or-install-script)
instructions above, then set up satellite workspaces as described below.

If your repository has multiple independent workspaces (e.g. a `/backend` and
`/frontend` subfolder, or an npm/pnpm/Yarn monorepo with multiple packages),
you can install Hero as a **satellite** in each subfolder. Each satellite gets
its own `.hero/` corpus scoped to that workspace.

```bash
# From each subfolder that should have its own Hero workspace:
cd backend
hero init && hero scan
hero install project . --target claude

cd ../frontend
hero init && hero scan
hero install project . --target claude
```

Each `hero install` writes harness files (e.g. `CLAUDE.md`) into that
subfolder, pointing the AI tool at the satellite corpus. Specs, conventions,
and knowledge stay scoped to the subfolder they belong to.

!!! tip "Single install at the repo root"
    If your monorepo has a single shared context, a single `hero init` at the
    root works fine. Use satellite installs only when subfolders are genuinely
    independent workspaces with different conventions, stacks, or teams.

## Next steps

- [Project Setup](project-setup.md) — Initialize Hero in your project
