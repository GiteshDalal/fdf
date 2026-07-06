package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
)

// rootFlag registers the uniform --root override; every command uses it.
func rootFlag(fs *flag.FlagSet) *string {
	return fs.String("root", "", "bundle root (overrides FDF_ROOT_DIR; default docs/features)")
}

func runValidate(args []string, stdout io.Writer) int {
	fs := newFlagSet("validate", stdout)
	root := rootFlag(fs)
	repoRoot := fs.String("repo-root", "", "project root for R1 resource checks (default: auto-detect)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cwd, _ := os.Getwd()
	bundleRoot, err := fdfroot.BundleRoot(*root, cwd)
	if err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return 2
	}
	rr := *repoRoot
	if rr == "" {
		if pr, standalone := fdfroot.ProjectRoot(bundleRoot); !standalone {
			rr = pr
		}
	}
	return bundle.Validate(bundleRoot, bundle.Options{RepoRoot: rr, Out: stdout})
}
