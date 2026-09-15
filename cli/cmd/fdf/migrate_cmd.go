package main

import (
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/migrate"
)

func runMigrate(args []string, stdout io.Writer) int {
	fs := newFlagSet("migrate", stdout)
	root, source, rest, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return 2
	}
	if !rejectPositionals("migrate", "--root", rest, stdout) {
		return 2
	}
	announce("migrate", root, source, stdout)
	repoRoot := ""
	if pr, standalone := fdfroot.ProjectRoot(root); !standalone {
		repoRoot = pr
	}
	migrate.Version = version
	return migrate.Run(root, repoRoot, stdout)
}
