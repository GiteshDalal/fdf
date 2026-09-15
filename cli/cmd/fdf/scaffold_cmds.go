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
	r, _, rest, ok := resolveRootSource(fs, args, stdout)
	return r, rest, ok
}

// resolveRootSource additionally reports which input chose the root, so a
// command can print it (see announce).
func resolveRootSource(fs *flag.FlagSet, args []string, stdout io.Writer) (string, string, []string, bool) {
	root := rootFlag(fs)
	if err := fs.Parse(args); err != nil {
		return "", "", nil, false
	}
	cwd, _ := os.Getwd()
	r, source, err := fdfroot.BundleRootWithSource(*root, cwd)
	if err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return "", "", nil, false
	}
	return r, source, fs.Args(), true
}

func runInit(args []string, stdout io.Writer) int {
	fs := newFlagSet("init", stdout)
	root, source, rest, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return 2
	}
	if !rejectPositionals("init", "--root", rest, stdout) {
		return 2
	}
	announce("init", root, source, stdout)
	return scaffold.Init(root, stdout)
}

func runNew(args []string, stdout io.Writer) int {
	fs := newFlagSet("new", stdout)
	root, source, rest, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return 2
	}
	if len(rest) != 1 {
		fmt.Fprintln(stdout, "usage: fdf new <group>/<slug>")
		return 2
	}
	announce("new", root, source, stdout)
	return scaffold.New(root, rest[0], stdout)
}
