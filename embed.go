// Package fdf exposes the repo's skills and harness assets to the CLI.
package fdf

import "embed"

// Assets embeds the harness-neutral skills and per-harness adapters that
// `fdf install` places into a harness's configuration.
//
//go:embed all:skills all:harness
var Assets embed.FS
