# install.sh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `curl | bash` installer that puts a prebuilt `fdf` binary from GitHub Releases onto macOS/Linux machines without Go.

**Architecture:** One static POSIX-y bash script (`install.sh`) at the repo root, served via raw.githubusercontent.com. It detects platform, resolves the release tag (env/arg or the `releases/latest` redirect), downloads the goreleaser tarball + `checksums.txt`, verifies sha256, and installs to `FDF_INSTALL_DIR` (default `~/.local/bin`). README gains the one-liner; CI gains a shellcheck step.

**Tech Stack:** bash, curl/wget, tar, sha256sum/shasum, shellcheck, GitHub Actions.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-11-install-script-design.md`
- macOS/Linux only; `darwin|linux` × `amd64|arm64`. Anything else exits with releases-page + `go install` pointer.
- No sudo, no GitHub API calls, no dependencies beyond curl-or-wget, tar, sha256sum-or-shasum.
- Release assets are named `fdf_<version-without-v>_<os>_<arch>.tar.gz`; tags carry a `v` prefix (e.g. tag `v0.2.2` → asset `fdf_0.2.2_darwin_arm64.tar.gz`).
- `checksums.txt` lines are `<sha256>  <filename>`.
- All logic inside `main()` called on the last line (truncated-download safety).
- `shellcheck install.sh` must be clean; CI must keep passing.

---

### Task 1: install.sh

**Files:**
- Create: `install.sh` (repo root, mode 755)

**Interfaces:**
- Consumes: GitHub release assets of `GiteshDalal/fdf` (goreleaser naming above).
- Produces: `install.sh` honoring `FDF_VERSION`, `FDF_INSTALL_DIR`, and `--version <tag>`; Task 2's README one-liner and Task 3's CI step reference this exact path.

- [ ] **Step 1: Write the script**

```bash
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

  local tmp
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
```

- [ ] **Step 2: Make it executable and run shellcheck (the failing/passing gate)**

Run: `chmod +x install.sh && shellcheck install.sh`
Expected: no output, exit 0. (If shellcheck is missing locally: `brew install shellcheck`.)

- [ ] **Step 3: End-to-end test, pinned version**

Run:
```bash
d=$(mktemp -d) && FDF_INSTALL_DIR="$d" FDF_VERSION=v0.2.2 bash install.sh && "$d/fdf" version && rm -rf "$d"
```
Expected: `downloading fdf_0.2.2_<os>_<arch>.tar.gz`, `installed ... to .../fdf`, and `fdf version` prints `fdf 0.2.2` (or similar).

- [ ] **Step 4: End-to-end test, latest + pipe mode (as users will run it)**

Run:
```bash
d=$(mktemp -d) && FDF_INSTALL_DIR="$d" bash -c 'cat install.sh | bash' && "$d/fdf" version && rm -rf "$d"
```
Expected: resolves latest (v0.2.2 today), installs, prints version. Also confirm failure paths: `FDF_VERSION=v9.9.9 bash install.sh` fails with a download error, and corrupting the tarball path is covered by the checksum test only if quick to simulate — otherwise skip.

- [ ] **Step 5: Commit**

```bash
git add install.sh
git commit -m "feat: add curl-installable install.sh for prebuilt binaries"
```

---

### Task 2: README one-liner

**Files:**
- Modify: `README.md:34-44` (the `## Install` section)

**Interfaces:**
- Consumes: `install.sh` at repo root on `main` (Task 1).

- [ ] **Step 1: Add the one-liner as the first install option**

Change the `## Install` list to:

```markdown
## Install

- One-liner (macOS/Linux, no Go needed):
```bash
curl -fsSL https://raw.githubusercontent.com/GiteshDalal/fdf/main/install.sh | bash
```
- Binaries on the [releases page](https://github.com/GiteshDalal/fdf/releases)
- With [mise](https://mise.jdx.dev)
```
mise use ubi:GiteshDalal/fdf
```
- With [Go](https://go.dev/)
```
go install github.com/GiteshDalal/fdf/cli/cmd/fdf@latest
```
```

(Keep the existing fenced-block style of the section; `FDF_VERSION`/`FDF_INSTALL_DIR` knobs are documented in the script header, not the README — YAGNI.)

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add curl one-liner to install section"
```

---

### Task 3: CI shellcheck step

**Files:**
- Modify: `.github/workflows/ci.yml:17-23`

**Interfaces:**
- Consumes: `install.sh` (Task 1). shellcheck is preinstalled on ubuntu-latest and macos-latest runners.

- [ ] **Step 1: Add the step after the gofmt check**

Insert after the `test -z "$(gofmt -l .)"` line:

```yaml
      - run: shellcheck install.sh
```

- [ ] **Step 2: Verify locally**

Run: `shellcheck install.sh && go vet ./... && go test ./... && test -z "$(gofmt -l .)"`
Expected: all pass (Go checks unaffected, but run them since CI will).

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: shellcheck install.sh"
```
