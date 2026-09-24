package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// resolveRootSource adds --root to fs, parses args (see parseArgs), and
// resolves the bundle root, reporting which input chose it so a command can
// print it (see announce). When ok is false the command stops with exit.
func resolveRootSource(fs *flag.FlagSet, args []string, stdout io.Writer) (root, source string, rest []string, exit int, ok bool) {
	rootF := rootFlag(fs)
	if rest, exit, ok = parseArgs(fs, args, stdout); !ok {
		return "", "", nil, exit, false
	}
	cwd, _ := os.Getwd()
	root, source, err := fdfroot.BundleRootWithSource(*rootF, cwd)
	if err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return "", "", nil, 2, false
	}
	return root, source, rest, 0, true
}

func runInit(args []string, stdout io.Writer) int {
	fs := newFlagSet("init")
	root, source, _, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	announce("init", root, source, stdout)
	return scaffold.Init(root, stdout)
}

func runNew(args []string, stdout io.Writer) int {
	fs := newFlagSet("new")
	root, source, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	if len(rest) != 1 {
		printUsage(stdout, "new")
		return 2
	}
	announce("new", root, source, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	return scaffold.New(root, rest[0], stdout)
}

func runPractice(args []string, stdout io.Writer) int {
	fs := newFlagSet("practice")
	root, source, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	if len(rest) != 1 {
		printUsage(stdout, "practice")
		return 2
	}
	announce("practice", root, source, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	return scaffold.Practice(root, rest[0], stdout)
}
