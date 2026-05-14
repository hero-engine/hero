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

## Build from source

The Hero source repository is private at the moment. If you have
access:

```bash
git clone git@github.com:hero-engine/hero.git
cd hero
make install     # installs to ~/go/bin/
```

Requires **Go 1.21+**. Ensure `~/go/bin` is on your PATH:

```bash
export PATH="$HOME/go/bin:$PATH"
```

!!! tip
    Add the export line to your `~/.zshrc` or `~/.bashrc` to make it permanent.

## Verify

```bash
hero --version
```

You should see the installed version printed. If not, confirm the
install location is on your PATH.

## Next steps

- [Project Setup](project-setup.md) — Initialize Hero in your project
