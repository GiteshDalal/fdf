package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
)

// rootFlag registers the uniform --root override; every command uses it.
func rootFlag(fs *flag.FlagSet) *string {
	return fs.String("root", "", "bundle root (default docs/fdf, or a docs/features from before 1.0; FDF_ROOT_DIR overrides it)")
}

func runValidate(args []string, stdout io.Writer) int {
	fs := newFlagSet("validate")
	root := rootFlag(fs)
	repoRoot := fs.String("repo-root", "", "project root for R1 resource checks (default: auto-detect)")
	strictDomain := fs.Bool("strict-domain", false, "promote F12 banned-word warnings to errors (DOMAIN.md's `strict: true` does it for every run)")
	if _, exit, ok := parseArgs(fs, args, stdout); !ok {
		return exit
	}
	cwd, _ := os.Getwd()
	r := fdfroot.Resolve(*root, cwd)
	bundleRoot := r.Root
	announce("validate", r, stdout)
	rr := *repoRoot
	if rr == "" {
		// A bundle at the top of its own repository — a docs repository cloned
		// on its own — has no project around it to check paths against.
		if pr, standalone := fdfroot.ProjectRoot(bundleRoot); !standalone && filepath.Clean(pr) != filepath.Clean(bundleRoot) {
			rr = pr
		}
	}
	return bundle.Validate(bundleRoot, bundle.Options{RepoRoot: rr, Out: stdout, StrictDomain: *strictDomain})
}
