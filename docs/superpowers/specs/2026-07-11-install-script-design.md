# install.sh — curl-installable fdf binaries

**Date:** 2026-07-11
**Status:** approved

## Goal

Let anyone install the `fdf` CLI with a single command, without Go installed:

```bash
curl -fsSL https://raw.githubusercontent.com/GiteshDalal/fdf/main/install.sh | bash
```

The script downloads the prebuilt goreleaser binary for the caller's platform
from GitHub Releases, verifies its checksum, and installs it.

## Approach

A static `install.sh` at the repo root, served from raw.githubusercontent.com.
Alternatives rejected: GitHub Pages / custom domain (infrastructure for no
functional gain); attaching the script to each release (puts a version in the
one-liner URL).

## Script behavior

- `#!/usr/bin/env bash`, `set -euo pipefail`; all logic inside `main()` invoked
  on the last line, so a truncated download executes nothing.
- **Platform:** `uname -s`/`uname -m` → `darwin|linux` × `amd64|arm64`.
  Anything else (including Windows/MINGW) exits with a pointer to the releases
  page and `go install`. macOS/Linux only by design.
- **Version:** `FDF_VERSION` env var (or `--version <v>` arg) if set, with or
  without the `v` prefix; otherwise resolve latest by following the
  `https://github.com/GiteshDalal/fdf/releases/latest` redirect and parsing the
  tag from the `Location` header — no GitHub API call, so no rate limits.
- **Download:** `fdf_<ver>_<os>_<arch>.tar.gz` + `checksums.txt` into a
  `mktemp -d` dir removed by an EXIT trap. `curl` preferred, `wget` fallback.
- **Verify:** sha256 of the tarball against its line in `checksums.txt`, using
  `sha256sum` or `shasum -a 256` (whichever exists). Mismatch or missing entry
  is a hard failure.
- **Install:** extract, `install -m 755` the `fdf` binary into
  `FDF_INSTALL_DIR` (default `~/.local/bin`, created if missing). No sudo.
- **Finish:** run `<dir>/fdf version` to confirm, and if the install dir is not
  on `PATH`, print the exact `export PATH=...` line for the user's shell.
- **Dependencies:** curl or wget, tar, sha256sum or shasum — stock on macOS and
  virtually every Linux.

## README

Add the curl one-liner as the first item under `## Install`.

## CI

Add a `shellcheck install.sh` step to the existing lint job in
`.github/workflows/ci.yml`.

## Testing

- `shellcheck install.sh` clean.
- End-to-end: run the script with `FDF_INSTALL_DIR` pointing at a temp dir
  (both latest and pinned `FDF_VERSION=v0.2.2`), assert `fdf version` output.
- Failure paths exercised by hand: unsupported platform, bad version, checksum
  mismatch.
