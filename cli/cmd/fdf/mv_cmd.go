package main

import (
	"fmt"
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/refactor"
)

// runMv moves or renames a document with its whole trail — or a whole group —
// and repairs every reference to it (reference repair). It validates
// afterwards, so a move that left anything behind says so at once.
func runMv(args []string, stdout io.Writer) int {
	fs := newFlagSet("mv")
	dryRun := fs.Bool("dry-run", false, "print what would move and what would be repaired, and change nothing")
	r, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	if len(rest) != 2 {
		printUsage(stdout, "mv")
		return 2
	}
	projectRoot := ""
	if pr, standalone := fdfroot.ProjectRoot(root); !standalone {
		projectRoot = pr
	}
	announce("mv", r, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	if code := refactor.Move(root, projectRoot, rest[0], rest[1], *dryRun, stdout); code != 0 || *dryRun {
		return code
	}
	fmt.Fprintln(stdout, "\nvalidating:")
	return bundle.Validate(root, bundle.Options{RepoRoot: projectRoot, Out: stdout})
}
