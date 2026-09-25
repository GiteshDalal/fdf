package main

import (
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/migrate"
)

func runMigrate(args []string, stdout io.Writer) int {
	fs := newFlagSet("migrate")
	dryRun := fs.Bool("dry-run", false, "print the plan, and change nothing")
	r, _, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	announce("migrate", r, stdout)
	repoRoot := ""
	if pr, standalone := fdfroot.ProjectRoot(root); !standalone {
		repoRoot = pr
	}
	migrate.Version = version
	return migrate.Run(migrate.Options{Root: root, Project: repoRoot, DryRun: *dryRun}, stdout)
}
