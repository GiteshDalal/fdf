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
	fs := newFlagSet("lexicon", stdout)
	term := fs.String("term", "", "only this term's banned words")
	all := fs.Bool("all", false, "report every occurrence, not the first few per word")
	fix := fs.Bool("fix", false, "replace each banned word with its term (a lexicon fix), logged in LOG.md")
	dryRun := fs.Bool("dry-run", false, "with --fix: print the diff and change nothing")
	root, source, rest, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return 2
	}
	if !rejectPositionals("lexicon", "--term", rest, stdout) {
		return 2
	}
	if *dryRun && !*fix {
		fmt.Fprintln(stdout, "usage: --dry-run applies to `fdf lexicon --fix`")
		return 2
	}
	announce("lexicon", root, source, stdout)
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
