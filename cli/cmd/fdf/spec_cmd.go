package main

import (
	"fmt"
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// runSpec prints an embedded spec version to stdout. The specs ship inside
// the binary (//go:embed all:spec), so this works with no bundle, no network,
// and no checkout of this repository — `fdf spec | less`, or
// `fdf spec -v 0.3` to read what a bundle pinning an older version must
// still satisfy.
func runSpec(args []string, stdout io.Writer) int {
	fs := newFlagSet("spec")
	version := fs.String("v", "", "spec version to print (default: the current version)")
	list := fs.Bool("list", false, "list the spec versions embedded in this binary")
	if _, exit, ok := parseArgs(fs, args, stdout); !ok {
		return exit
	}
	if *list {
		for _, v := range scaffold.SpecVersions() {
			mark := ""
			if v == scaffold.CurrentVersion() {
				mark = "  (current)"
			}
			fmt.Fprintf(stdout, "%s%s\n", v, mark)
		}
		return 0
	}
	v := *version
	if v == "" {
		v = scaffold.CurrentVersion()
	}
	text, err := scaffold.SpecText(v)
	if err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return 2
	}
	if _, err := stdout.Write(text); err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return 1
	}
	return 0
}
