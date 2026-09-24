package main

import (
	"fmt"
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/adopt"
	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// runAdopt maps what a codebase already does (v0.7). With a feature ID it
// scaffolds an adopted feature — a map entry; without one it prints the
// adoption map: what is built, what is adopted, and the code no document
// claims yet.
func runAdopt(args []string, stdout io.Writer) int {
	fs := newFlagSet("adopt")
	resource := fs.String("resource", "", "comma-separated project-relative path(s) of the code the capability lives in (required when mapping one)")
	depth := fs.Int("depth", 2, "how many directory levels the unclaimed code is grouped by in the report")
	root, source, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	if len(rest) > 1 {
		printUsage(stdout, "adopt")
		return 2
	}
	if *depth < 1 {
		fmt.Fprintln(stdout, "usage: --depth is at least 1")
		return 2
	}
	if len(rest) == 1 && len(splitList(*resource)) == 0 {
		fmt.Fprintln(stdout, "error: --resource is required — name the code this capability lives in; with no tasks, it is the feature's only link to the code")
		printUsage(stdout, "adopt")
		return 2
	}
	if len(rest) == 0 && *resource != "" {
		fmt.Fprintln(stdout, "usage: --resource names the code of the capability being mapped: fdf adopt --resource <paths> <group>/<slug>")
		return 2
	}
	projectRoot := ""
	if pr, standalone := fdfroot.ProjectRoot(root); !standalone {
		projectRoot = pr
	}
	announce("adopt", root, source, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	if len(rest) == 1 {
		return scaffold.Adopt(root, projectRoot, rest[0], splitList(*resource), stdout)
	}
	return adopt.Map(root, projectRoot, *depth, stdout)
}
