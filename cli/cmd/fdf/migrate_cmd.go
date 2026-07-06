package main

import (
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/migrate"
)

func runMigrate(args []string, stdout io.Writer) int {
	fs := newFlagSet("migrate", stdout)
	root, _, ok := resolveRoot(fs, args, stdout)
	if !ok {
		return 2
	}
	return migrate.Run(root, stdout)
}
