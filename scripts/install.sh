#!/usr/bin/env sh
# Hero CLI install script.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/hero-engine/hero-releases/main/install.sh | sh
#
# Environment:
#   HERO_VERSION   pin a specific version (e.g. v0.9.1); default: latest
#   HERO_INSTALL   install directory; default: /usr/local/bin (falls back to $HOME/.local/bin if not writable)

set -eu

REPO="hero-engine/hero-releases"
RELEASES="https://github.com/${REPO}/releases"

bold=""
reset=""
red=""
green=""
if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
  bold="$(tput bold 2>/dev/null || true)"
  reset="$(tput sgr0 2>/dev/null || true)"
  red="$(tput setaf 1 2>/dev/null || true)"
  green="$(tput setaf 2 2>/dev/null || true)"
fi

info()  { printf '%s\n' "$*"; }
ok()    { printf '%s%s%s\n' "$green" "$*" "$reset"; }
err()   { printf '%s%s%s\n' "$red" "$*" "$reset" >&2; }
fatal() { err "error: $*"; exit 1; }

command -v curl >/dev/null 2>&1 || fatal "curl is required"
command -v tar  >/dev/null 2>&1 || fatal "tar is required"

# Detect OS.
uname_s="$(uname -s)"
case "$uname_s" in
  Darwin) os="darwin" ;;
  Linux)  os="linux"  ;;
  *)      fatal "unsupported OS: $uname_s (install.sh covers macOS and Linux; on Windows, use install.ps1)" ;;
esac

# Detect arch.
uname_m="$(uname -m)"
case "$uname_m" in
  x86_64|amd64)  arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)             fatal "unsupported architecture: $uname_m" ;;
esac

# Resolve version.
version="${HERO_VERSION:-}"
if [ -z "$version" ]; then
  info "Resolving latest release..."
  version="$(curl -fsSL -o /dev/null -w '%{url_effective}' "${RELEASES}/latest" | sed 's|.*/tag/||')"
  [ -n "$version" ] || fatal "could not determine latest version"
fi
case "$version" in
  v*) ;;
  *)  version="v${version}" ;;
esac
# Goreleaser strips the leading 'v' in archive names.
version_bare="${version#v}"

archive="hero_${version_bare}_${os}_${arch}.tar.gz"
url="${RELEASES}/download/${version}/${archive}"
sums_url="${RELEASES}/download/${version}/checksums.txt"

info "Downloading ${bold}hero ${version}${reset} (${os}/${arch})..."

tmp="$(mktemp -d 2>/dev/null || mktemp -d -t hero)"
trap 'rm -rf "$tmp"' EXIT INT TERM

curl -fsSL "$url" -o "$tmp/$archive" || fatal "failed to download $url"

# Verify checksum (best-effort: skip with a warning if checksums.txt is absent).
if curl -fsSL "$sums_url" -o "$tmp/checksums.txt" 2>/dev/null; then
  expected="$(awk -v f="$archive" '$2 == f {print $1}' "$tmp/checksums.txt")"
  if [ -n "$expected" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      actual="$(sha256sum "$tmp/$archive" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
      actual="$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')"
    else
      actual=""
    fi
    if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
      fatal "checksum mismatch (expected $expected, got $actual)"
    fi
  fi
fi

tar -xzf "$tmp/$archive" -C "$tmp" hero || fatal "failed to extract $archive"

# Choose install directory.
install_dir="${HERO_INSTALL:-}"
if [ -z "$install_dir" ]; then
  if [ -w /usr/local/bin ] 2>/dev/null; then
    install_dir="/usr/local/bin"
  elif [ -d /usr/local/bin ] && command -v sudo >/dev/null 2>&1; then
    install_dir="/usr/local/bin"
    use_sudo=1
  else
    install_dir="$HOME/.local/bin"
    mkdir -p "$install_dir"
  fi
fi

dest="$install_dir/hero"
if [ "${use_sudo:-0}" = "1" ]; then
  sudo install -m 0755 "$tmp/hero" "$dest"
else
  install -m 0755 "$tmp/hero" "$dest" 2>/dev/null || {
    mkdir -p "$install_dir"
    install -m 0755 "$tmp/hero" "$dest"
  }
fi

ok "Installed hero ${version} to ${dest}"

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    info ""
    info "${bold}Note:${reset} ${install_dir} is not on your PATH."
    info "Add it to your shell profile, e.g.:"
    info "  echo 'export PATH=\"${install_dir}:\$PATH\"' >> ~/.zshrc"
    ;;
esac

info ""
info "Run ${bold}hero --help${reset} to get started."
