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
	fs := newFlagSet(cmd, stdout)
	affects := fs.String("affects", "", "comma-separated feature ID(s) this touches (required)")
	root, source, rest, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return 2
	}
	if len(rest) != 1 {
		fmt.Fprintf(stdout, "usage: fdf %s [--root <dir>] --affects <group>/<slug>[,…] [<group>/]<slug>\n", cmd)
		flagOrderHint(rest, stdout)
		return 2
	}
	announce(cmd, root, source, stdout)
	return changes.New(root, rest[0], docType, splitList(*affects), stdout)
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
	fs := newFlagSet("history", stdout)
	root, source, rest, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return 2
	}
	if len(rest) != 1 {
		fmt.Fprintln(stdout, "usage: fdf history [--root <dir>] <group>/<slug>")
		flagOrderHint(rest, stdout)
		return 2
	}
	announce("history", root, source, stdout)
	return changes.History(root, rest[0], stdout)
}
