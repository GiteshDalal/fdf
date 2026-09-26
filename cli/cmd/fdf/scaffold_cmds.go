package main

import (
	"flag"
	"io"
	"os"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// resolveRootSource adds --root to fs, parses args (see parseArgs), and
// resolves the bundle root, reporting which input chose it so a command can
// print it (see announce). When ok is false the command stops with exit.
func resolveRootSource(fs *flag.FlagSet, args []string, stdout io.Writer) (r fdfroot.Resolution, rest []string, exit int, ok bool) {
	rootF := rootFlag(fs)
	if rest, exit, ok = parseArgs(fs, args, stdout); !ok {
		return fdfroot.Resolution{}, nil, exit, false
	}
	cwd, _ := os.Getwd()
	return fdfroot.Resolve(*rootF, cwd), rest, 0, true
}

func runInit(args []string, stdout io.Writer) int {
	fs := newFlagSet("init")
	r, _, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	announce("init", r, stdout)
	return scaffold.Init(root, stdout)
}

func runNew(args []string, stdout io.Writer) int {
	fs := newFlagSet("new")
	r, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	if len(rest) != 1 {
		printUsage(stdout, "new")
		return 2
	}
	announce("new", r, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	return scaffold.New(root, rest[0], stdout)
}

func runPractice(args []string, stdout io.Writer) int {
	fs := newFlagSet("practice")
	r, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	if len(rest) != 1 {
		printUsage(stdout, "practice")
		return 2
	}
	announce("practice", r, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	return scaffold.Practice(root, rest[0], stdout)
}
