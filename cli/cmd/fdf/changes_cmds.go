package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/changes"
)

// splitList parses a comma-separated flag value into trimmed, non-empty parts.
func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// runPostDelivery backs both `fdf change` and `fdf fix`: the two commands
// differ only in the document type they scaffold and the body template that
// follows from it.
func runPostDelivery(cmd, docType string, args []string, stdout io.Writer) int {
	fs := newFlagSet(cmd)
	affects := fs.String("affects", "", "comma-separated feature ID(s) this touches (required unless --from names a bug that has them)")
	from := fs.String("from", "", "bugs/<id> this work repairs: copies its analysis and writes `resolves`")
	root, source, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	if len(rest) != 1 {
		printUsage(stdout, cmd)
		return 2
	}
	if len(splitList(*affects)) == 0 && *from == "" {
		fmt.Fprintf(stdout, "error: --affects is required — name the delivered feature(s) this %s touches, or start from a filed bug with --from bugs/<id>\n", docType)
		printUsage(stdout, cmd)
		return 2
	}
	announce(cmd, root, source, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	return changes.NewFrom(root, rest[0], docType, splitList(*affects), *from, stdout)
}

func runChange(args []string, stdout io.Writer) int {
	return runPostDelivery("change", "Change", args, stdout)
}

func runFix(args []string, stdout io.Writer) int {
	return runPostDelivery("fix", "Fix", args, stdout)
}

// runHistory lists a feature's post-delivery trail, computed from `affects`
// rather than from back-links the feature would have to maintain by hand.
func runHistory(args []string, stdout io.Writer) int {
	fs := newFlagSet("history")
	root, source, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	if len(rest) != 1 {
		printUsage(stdout, "history")
		return 2
	}
	announce("history", root, source, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	return changes.History(root, rest[0], stdout)
}
