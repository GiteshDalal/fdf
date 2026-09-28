package main

import (
	"bytes"
	"fmt"
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// runSpec prints an embedded spec version to stdout. The specs ship inside
// the binary (//go:embed all:spec), so this works with no bundle, no network,
// and no checkout of this repository — `fdf spec | less`, or
// `fdf spec -v 0.3` to read what a bundle pinning an older version must
// still satisfy. It prints what a bundle pinning the version vendors as
// SPEC.md; from 1.1 on, a version's examples are a file of their own, which
// --examples prints, as a bundle vendors it in SPEC.examples.md, and --full
// prints after the spec.
func runSpec(args []string, stdout io.Writer) int {
	fs := newFlagSet("spec")
	version := fs.String("v", "", "spec version to print (default: the current version)")
	list := fs.Bool("list", false, "list the spec versions embedded in this binary")
	examples := fs.Bool("examples", false, "print only the version's examples")
	full := fs.Bool("full", false, "print the spec, then its examples")
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
	if *examples && *full {
		fmt.Fprintln(stdout, "error: --examples prints the examples alone and --full the spec with them — pass one")
		return 2
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
	ex, hasExamples := scaffold.ExamplesText(v)
	switch {
	case *examples && !hasExamples && bytes.Contains(text, []byte("\n# Document examples\n")):
		fmt.Fprintf(stdout, "error: spec %s has no examples file: its examples are inside the spec, under `# Document examples` — `fdf spec -v %s` prints it\n", v, v)
		return 2
	case *examples && !hasExamples:
		fmt.Fprintf(stdout, "error: spec %s has no examples\n", v)
		return 2
	case *examples:
		text = ex
	case *full && hasExamples:
		text = append(append(append([]byte{}, text...), '\n'), ex...)
	}
	if _, err := stdout.Write(text); err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return 1
	}
	return 0
}
