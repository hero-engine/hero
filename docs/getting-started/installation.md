# Installation

## Homebrew (recommended)

```bash
brew install hero-engine/tap/hero
```

## Build from Source

Requires **Go 1.21+**.

```bash
git clone https://github.com/hero-engine/hero.git
cd hero
make install
```

This installs the `hero` binary to `~/go/bin/`. Ensure it's on your PATH:

```bash
export PATH="$HOME/go/bin:$PATH"
```

!!! tip
    Add the export line to your `~/.zshrc` or `~/.bashrc` to make it permanent.

## Verify

```bash
hero --version
```

You should see the installed version printed. If not, confirm `~/go/bin` is on your PATH.

## Next Steps

- [Project Setup](project-setup.md) — Initialize Hero in your project
