// Package fdf exposes the repo's skills and harness assets to the CLI.
package fdf

import "embed"

// Assets embeds the harness-neutral skills, the per-harness adapters that
// `fdf install` places into a harness's configuration, and the versioned
// format specs (spec/<version>.md) that `fdf init`/`fdf migrate` vendor into
// a bundle's docs/features/SPEC.md.
//
//go:embed all:skills all:harness all:spec
var Assets embed.FS
