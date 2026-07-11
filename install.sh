#!/usr/bin/env bash
# Installs a prebuilt fdf binary from GitHub Releases.
#   curl -fsSL https://raw.githubusercontent.com/GiteshDalal/fdf/main/install.sh | bash
# Env: FDF_VERSION (tag to install, default latest), FDF_INSTALL_DIR (default ~/.local/bin)
set -euo pipefail

REPO="GiteshDalal/fdf"

info() { printf 'fdf-install: %s\n' "$*"; }
fail() { printf 'fdf-install: error: %s\n' "$*" >&2; exit 1; }

have() { command -v "$1" >/dev/null 2>&1; }

download() { # $1 url, $2 dest
  if have curl; then curl -fsSL -o "$2" "$1"
  elif have wget; then wget -q -O "$2" "$1"
  else fail "need curl or wget"
  fi
}

latest_tag() {
  # Follow the releases/latest redirect; the final URL ends in /tag/<tag>.
  local url=""
  if have curl; then
    url=$(curl -fsSI -o /dev/null -w '%{url_effective}' -L "https://github.com/$REPO/releases/latest")
  elif have wget; then
    url=$(wget -q --max-redirect=0 -S -O /dev/null "https://github.com/$REPO/releases/latest" 2>&1 \
      | tr -d '\r' | sed -n 's/.*[Ll]ocation: *//p' | tail -1) || true
  else
    fail "need curl or wget"
  fi
  case "$url" in
    */tag/*) printf '%s\n' "${url##*/}" ;;
    *) fail "could not resolve latest release; pin one with FDF_VERSION=vX.Y.Z" ;;
  esac
}

detect_platform() { # sets OS and ARCH
  case "$(uname -s)" in
    Darwin) OS=darwin ;;
    Linux) OS=linux ;;
    *) fail "unsupported OS $(uname -s) — grab a binary from https://github.com/$REPO/releases or use: go install github.com/$REPO/cli/cmd/fdf@latest" ;;
  esac
  case "$(uname -m)" in
    x86_64 | amd64) ARCH=amd64 ;;
    arm64 | aarch64) ARCH=arm64 ;;
    *) fail "unsupported architecture $(uname -m) — see https://github.com/$REPO/releases" ;;
  esac
}

verify_checksum() { # $1 tarball, $2 checksums.txt
  local name want got
  name=$(basename "$1")
  want=$(awk -v n="$name" '$2 == n { print $1 }' "$2")
  [ -n "$want" ] || fail "no entry for $name in checksums.txt"
  if have sha256sum; then got=$(sha256sum "$1" | awk '{ print $1 }')
  elif have shasum; then got=$(shasum -a 256 "$1" | awk '{ print $1 }')
  else fail "need sha256sum or shasum to verify the download"
  fi
  [ "$got" = "$want" ] || fail "checksum mismatch for $name (expected $want, got $got)"
}

main() {
  local version="${FDF_VERSION:-}"
  while [ $# -gt 0 ]; do
    case "$1" in
      --version) version="${2:-}"; [ -n "$version" ] || fail "--version needs an argument"; shift 2 ;;
      *) fail "unknown argument: $1" ;;
    esac
  done

  detect_platform

  if [ -z "$version" ]; then
    info "resolving latest release..."
    version=$(latest_tag)
  fi
  version="v${version#v}"
  local plain="${version#v}"

  local install_dir="${FDF_INSTALL_DIR:-$HOME/.local/bin}"
  local asset="fdf_${plain}_${OS}_${ARCH}.tar.gz"
  local base="https://github.com/$REPO/releases/download/$version"

  # not local: the EXIT trap runs after main() returns, outside its scope
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  info "downloading $asset ($version)..."
  download "$base/$asset" "$tmp/$asset"
  download "$base/checksums.txt" "$tmp/checksums.txt"
  verify_checksum "$tmp/$asset" "$tmp/checksums.txt"

  tar -xzf "$tmp/$asset" -C "$tmp" fdf
  mkdir -p "$install_dir"
  install -m 755 "$tmp/fdf" "$install_dir/fdf"

  info "installed $("$install_dir/fdf" version) to $install_dir/fdf"
  case ":$PATH:" in
    *":$install_dir:"*) ;;
    *)
      info "note: $install_dir is not on your PATH; add this to your shell profile:"
      info "  export PATH=\"$install_dir:\$PATH\""
      ;;
  esac
}

main "$@"
