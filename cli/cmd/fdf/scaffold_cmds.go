package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

func resolveRoot(fs *flag.FlagSet, args []string, stdout io.Writer) (string, []string, bool) {
	root := rootFlag(fs)
	fs.SetOutput(stdout)
	if err := fs.Parse(args); err != nil {
		return "", nil, false
	}
	cwd, _ := os.Getwd()
	r, err := fdfroot.BundleRoot(*root, cwd)
	if err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return "", nil, false
	}
	return r, fs.Args(), true
}

func runInit(args []string, stdout io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	root, _, ok := resolveRoot(fs, args, stdout)
	if !ok {
		return 2
	}
	return scaffold.Init(root, stdout)
}

func runNew(args []string, stdout io.Writer) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	root, rest, ok := resolveRoot(fs, args, stdout)
	if !ok {
		return 2
	}
	if len(rest) != 1 {
		fmt.Fprintln(stdout, "usage: fdf new <group>/<slug>")
		return 2
	}
	return scaffold.New(root, rest[0], stdout)
}
