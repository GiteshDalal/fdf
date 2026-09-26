// Package fdf exposes the repo's skills and specs to the CLI.
package fdf

import "embed"

// Assets embeds the harness-neutral skills that `fdf install` places into an
// AI harness's configuration, and the versioned format specs (spec/<version>.md)
// that `fdf init`/`fdf migrate` vendor into a bundle's docs/fdf/SPEC.md.
//
//go:embed all:skills all:spec
var Assets embed.FS
