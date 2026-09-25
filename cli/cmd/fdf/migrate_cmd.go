package main

import (
	"io"
	"os"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/migrate"
)

func runMigrate(args []string, stdout io.Writer) int {
	fs := newFlagSet("migrate")
	dryRun := fs.Bool("dry-run", false, "print the plan, and change nothing")
	to := fs.String("to", "", "where the bundle goes")
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
	// --to resolves as --root does, and FDF_ROOT_DIR is read as every
	// command reads it, so that migrate can say when it names the old path.
	cwd, _ := os.Getwd()
	dest, envRoot := "", ""
	if *to != "" {
		dest = fdfroot.Resolve(*to, cwd).Root
	}
	if os.Getenv("FDF_ROOT_DIR") != "" {
		envRoot = fdfroot.Resolve("", cwd).Root
	}
	migrate.Version = version
	return migrate.Run(migrate.Options{Root: root, Project: repoRoot, DryRun: *dryRun, To: dest, EnvRoot: envRoot}, stdout)
}
