package main

import (
	"flag"
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/migrate"
)

func runMigrate(args []string, stdout io.Writer) int {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	root, _, ok := resolveRoot(fs, args, stdout)
	if !ok {
		return 2
	}
	return migrate.Run(root, stdout)
}
