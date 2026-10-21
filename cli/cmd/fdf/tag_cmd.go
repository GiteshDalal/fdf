package main

import (
	"fmt"
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/tags"
)

// runTag reads and keeps the tag dictionary, TAGS.md (spec 1.2). Without an
// argument it lists the tags; with tags it adds them; --sync starts the
// dictionary from the tags documents carry, and --merge folds one tag into
// another across the bundle.
func runTag(args []string, stdout io.Writer) int {
	fs := newFlagSet("tag")
	show := fs.String("show", "", "list the documents that carry this tag")
	sync := fs.Bool("sync", false, "add every tag documents carry that TAGS.md lacks, each with a TODO definition")
	merge := fs.Bool("merge", false, "fold the tag <from> into the tag <into> in every document and in TAGS.md")
	dryRun := fs.Bool("dry-run", false, "with --sync or --merge: show what would change, write nothing")
	definition := fs.String("definition", "", "with one tag: its definition, instead of a TODO to write")
	r, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	modes := 0
	for _, on := range []bool{*show != "", *sync, *merge} {
		if on {
			modes++
		}
	}
	switch {
	case modes > 1:
		fmt.Fprintln(stdout, "usage: fdf tag takes one of --show, --sync and --merge")
		return 2
	case *dryRun && !*sync && !*merge:
		fmt.Fprintln(stdout, "usage: --dry-run applies to `fdf tag --sync` and `fdf tag --merge`")
		return 2
	case *merge && len(rest) != 2:
		fmt.Fprintln(stdout, "usage: fdf tag --merge [--dry-run] <from> <into>")
		return 2
	case (*show != "" || *sync) && len(rest) > 0:
		printUsage(stdout, "tag")
		return 2
	case *definition != "" && (modes > 0 || len(rest) != 1):
		fmt.Fprintln(stdout, "usage: --definition gives one tag its definition: fdf tag --definition '<text>' <tag>")
		return 2
	}
	announce("tag", r, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	switch {
	case *show != "":
		return tags.Show(root, *show, stdout)
	case *sync:
		return tags.Sync(root, *dryRun, stdout)
	case *merge:
		return tags.Merge(root, rest[0], rest[1], *dryRun, stdout)
	case len(rest) > 0:
		return tags.Add(root, rest, *definition, stdout)
	}
	return tags.List(root, stdout)
}
