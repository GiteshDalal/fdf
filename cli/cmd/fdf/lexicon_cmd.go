package main

import (
	"fmt"
	"io"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/refactor"
)

// runLexicon reports every banned word F12 sees, with its place, or — with
// --fix — replaces them with their terms (v0.7's lexicon fix). A real fix is
// validated at once: F8 and F10 are what prove the scenario-name joins held.
func runLexicon(args []string, stdout io.Writer) int {
	fs := newFlagSet("lexicon")
	term := fs.String("term", "", "only this term's banned words")
	all := fs.Bool("all", false, "report every occurrence, not the first few per word")
	fix := fs.Bool("fix", false, "replace each banned word with its term (a lexicon fix), logged in LOG.md")
	dryRun := fs.Bool("dry-run", false, "with --fix: print the diff and change nothing")
	r, _, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root
	if *dryRun && !*fix {
		t := "<Term>"
		if *term != "" {
			t = shellQuote(*term)
		}
		fmt.Fprintf(stdout, "usage: --dry-run goes with --fix: fdf lexicon --term %s --fix --dry-run\n", t)
		return 2
	}
	announce("lexicon", r, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	code := refactor.Lexicon(root, refactor.LexiconOptions{Term: *term, All: *all, Fix: *fix, DryRun: *dryRun}, stdout)
	if code != 0 || !*fix || *dryRun {
		return code
	}
	projectRoot := ""
	if pr, standalone := fdfroot.ProjectRoot(root); !standalone {
		projectRoot = pr
	}
	fmt.Fprintln(stdout, "\nvalidating:")
	return bundle.Validate(root, bundle.Options{RepoRoot: projectRoot, Out: stdout})
}
