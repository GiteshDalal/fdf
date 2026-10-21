package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/changes"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
	"github.com/GiteshDalal/fdf/cli/internal/tags"
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

// tagged runs create, a command that files a new document in register reg
// under name, and gives the document the tags names: each is checked before
// anything is written, and, from spec 1.2 on, a tag TAGS.md does not define
// yet is added to it with a TODO definition.
func tagged(root, reg, name string, names []string, create func() int, stdout io.Writer) int {
	if !tags.Check(root, names, stdout) {
		return 1
	}
	if code := create(); code != 0 || len(names) == 0 {
		return code
	}
	return tags.Stamp(root, scaffold.IDIn(reg, name), names, stdout)
}

// runPostDelivery backs both `fdf change` and `fdf fix`: the two commands
// differ only in the document type they scaffold and the body template that
// follows from it.
func runPostDelivery(cmd, docType string, args []string, stdout io.Writer) int {
	fs := newFlagSet(cmd)
	affects := fs.String("affects", "", "comma-separated feature ID(s) this touches (required unless --from names a bug that has them)")
	from := fs.String("from", "", "bugs/<id> this work repairs: copies its analysis and writes `resolves`")
	tagList := fs.String("tags", "", "comma-separated tags the new document carries (with --from, the bug's by default)")
	r, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	if len(rest) != 1 {
		printUsage(stdout, cmd)
		return 2
	}
	if len(splitList(*affects)) == 0 && *from == "" {
		fmt.Fprintf(stdout, "error: --affects is required — name the delivered feature(s) this %s touches, or start from a filed bug with --from bugs/<id>\n", docType)
		printUsage(stdout, cmd)
		return 2
	}
	announce(cmd, r, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	names := splitList(*tagList)
	if len(names) == 0 && *from != "" {
		names = tags.Inherited(root, *from)
	}
	return tagged(root, "changes", rest[0], names, func() int {
		return changes.NewFrom(root, rest[0], docType, splitList(*affects), *from, stdout)
	}, stdout)
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
	r, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	if len(rest) != 1 {
		printUsage(stdout, "history")
		return 2
	}
	announce("history", r, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	return changes.History(root, rest[0], stdout)
}
