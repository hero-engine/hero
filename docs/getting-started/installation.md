# Installation

## Homebrew (macOS / Linux)

```bash
brew install hero-engine/tap/hero
```

The tap publishes prebuilt binaries for macOS (arm64, amd64) and Linux
(arm64, amd64).

## Direct download

Prebuilt binaries for every release are attached to
[hero-engine/hero-releases](https://github.com/hero-engine/hero-releases/releases).
Download the archive for your OS/arch, extract `hero`, and put it
somewhere on your PATH.

## Build from Source

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

## Next Steps

- [Project Setup](project-setup.md) — Initialize Hero in your project
