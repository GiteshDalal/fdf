package main

import (
	"io"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/release"
)

// runRelease derives releases/<version>.md from the `version:` fields already
// on features, Changes and Fixes. It never chooses what ships.
func runRelease(args []string, stdout io.Writer) int {
	fs := newFlagSet("release")
	date := fs.String("date", "", "target date while planned, actual date once shipped (YYYY-MM-DD)")
	ship := fs.Bool("ship", false, "flip the release to shipped; refuses while any listed document is open")
	r, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	if len(rest) != 1 || strings.TrimSpace(rest[0]) == "" {
		printUsage(stdout, "release")
		return 2
	}
	announce("release", r, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	return release.Sync(root, rest[0], *date, *ship, stdout)
}
