package main

import (
	"fmt"
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/release"
)

// runRelease derives releases/<version>.md from the `version:` fields already
// on features, changes and fixes. It never chooses what ships.
func runRelease(args []string, stdout io.Writer) int {
	fs := newFlagSet("release", stdout)
	date := fs.String("date", "", "target date while planned, actual date once shipped (YYYY-MM-DD)")
	ship := fs.Bool("ship", false, "flip the release to shipped; refuses while any listed document is open")
	root, source, rest, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return 2
	}
	if len(rest) != 1 {
		fmt.Fprintln(stdout, "usage: fdf release [--root <dir>] [--date <YYYY-MM-DD>] [--ship] <version>")
		flagOrderHint(rest, stdout)
		return 2
	}
	announce("release", root, source, stdout)
	return release.Sync(root, rest[0], *date, *ship, stdout)
}
