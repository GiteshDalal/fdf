package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/register"
)

// runDebt and runBug are the two registers' whole surface: read one, file into
// it, clear what is closed. With a positional slug the command scaffolds;
// without one it reports.
func runDebt(args []string, stdout io.Writer) int { return runRegister(register.Debt, args, stdout) }
func runBug(args []string, stdout io.Writer) int  { return runRegister(register.Bug, args, stdout) }

func runRegister(k register.Kind, args []string, stdout io.Writer) int {
	cmd := strings.ToLower(k.Type)
	fs := newFlagSet(cmd)
	open := fs.Bool("open", false, "list only open "+cmd+"s")
	accepted := fs.Bool("accepted", false, "list only accepted "+cmd+"s")
	resolved := fs.Bool("resolved", false, "list only resolved "+cmd+"s")
	cleanup := fs.Bool("cleanup", false, "fold resolved "+cmd+"s into "+k.Dir+"/LOG.md and clear them from the register")
	dryRun := fs.Bool("dry-run", false, "with --cleanup: show what would be cleared and change nothing")
	// Defined for bugs too, only to refuse it with the reason below.
	noLog := fs.Bool("no-log", false, "with --cleanup: remove resolved "+cmd+"s without writing "+k.Dir+"/LOG.md")
	var affects *string
	if k.Type == "Bug" {
		affects = fs.String("affects", "", "comma-separated feature ID(s) the defect shows up in (when filing)")
	}
	resource := fs.String("resource", "", "comma-separated project-relative path(s) carrying it (when filing)")
	cost := fs.Int("cost", register.Unrated, "0-9: what it costs while it stays (when filing; spec 1.2)")
	effort := fs.Int("effort", register.Unrated, "0-9: the work to close it (when filing; spec 1.2)")
	tagList := fs.String("tags", "", "comma-separated tags it carries (when filing)")
	tag := fs.String("tag", "", "list only the "+cmd+"s that carry this tag")
	minCost := fs.Int("min-cost", 0, "list only the "+cmd+"s whose cost is at least this, 1 to 9 (spec 1.2)")
	untriaged := fs.Bool("untriaged", false, "list only the open and accepted "+cmd+"s with no cost yet (spec 1.2)")
	sortBy := fs.String("sort", "cost", "order the listing by cost, or by value: cost − effort (spec 1.2)")
	r, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	root := r.Root

	var filters []string
	for name, on := range map[string]bool{"open": *open, "accepted": *accepted, "resolved": *resolved} {
		if on {
			filters = append(filters, name)
		}
	}
	if len(filters) > 1 {
		fmt.Fprintf(stdout, "usage: fdf %s takes at most one of --open, --accepted, --resolved\n", cmd)
		return 2
	}
	filter := ""
	if len(filters) == 1 {
		filter = filters[0]
	}

	if len(rest) > 1 {
		printUsage(stdout, cmd)
		return 2
	}
	var affected []string
	if affects != nil {
		affected = splitList(*affects)
	}
	// -1 is how a cost or effort not given reads, so one typed is checked
	// as typed: --cost -1 is no way to leave it out.
	var bad []string
	fs.Visit(func(f *flag.Flag) {
		if v := f.Value.String(); (f.Name == "cost" || f.Name == "effort") && (v < "0" || v > "9" || len(v) != 1) {
			bad = append(bad, "--"+f.Name+" "+v)
		}
	})
	if len(bad) > 0 {
		fmt.Fprintf(stdout, "usage: --cost and --effort take a whole number from 0 to 9, not %s — SPEC.md's *Cost and effort* says what each level means\n", strings.Join(bad, ", "))
		return 2
	}
	listing := *tag != "" || *minCost != 0 || *untriaged || *sortBy != "cost"
	if *sortBy != "cost" && *sortBy != "value" {
		fmt.Fprintf(stdout, "usage: --sort takes cost or value, not %q\n", *sortBy)
		return 2
	}
	if len(rest) == 1 {
		if filter != "" || *cleanup || listing {
			fmt.Fprintf(stdout, "usage: fdf %s %s scaffolds a %s; drop the listing flags to read the register\n", cmd, rest[0], cmd)
			return 2
		}
		announce(cmd, r, stdout)
		if !requireBundle(root, stdout) {
			return 1
		}
		f := register.Filing{Affects: affected, Resources: splitList(*resource), Cost: *cost, Effort: *effort}
		return tagged(root, k.Dir, rest[0], splitList(*tagList), func() int { return k.File(root, rest[0], f, stdout) }, stdout)
	}
	if len(affected) > 0 || *resource != "" || *cost != register.Unrated || *effort != register.Unrated || *tagList != "" {
		if k.Type == "Bug" {
			fmt.Fprintln(stdout, "usage: --affects, --resource, --cost, --effort and --tags apply when filing a bug: fdf bug [--affects <ids>] [--resource <paths>] [--cost <0-9>] [--effort <0-9>] [--tags <tags>] [<group>/…]<slug>")
		} else {
			fmt.Fprintf(stdout, "usage: --resource, --cost, --effort and --tags apply when filing a %s: fdf %s [--resource <paths>] [--cost <0-9>] [--effort <0-9>] [--tags <tags>] [<group>/…]<slug>\n", cmd, cmd)
		}
		return 2
	}
	if *minCost < 0 || *minCost > 9 {
		fmt.Fprintf(stdout, "usage: --min-cost takes a whole number from 1 to 9, not %d\n", *minCost)
		return 2
	}
	if *cleanup && listing {
		fmt.Fprintf(stdout, "usage: --cleanup clears every resolved %s; it takes no listing flag\n", cmd)
		return 2
	}
	// A cleared bug stays findable in bugs/LOG.md: a done Fix or Change that
	// `resolves` it is checked against that log (F10), so it is never dropped.
	if *noLog && k.Type == "Bug" {
		fmt.Fprintln(stdout, "usage: fdf bug --cleanup always logs — a done Fix or Change that `resolves` a cleared bug is checked against bugs/LOG.md (F10)")
		return 2
	}
	if (*dryRun || *noLog) && !*cleanup {
		what := "--dry-run and --no-log only apply"
		if k.Type == "Bug" {
			what = "--dry-run only applies"
		}
		fmt.Fprintf(stdout, "usage: %s to `fdf %s --cleanup`\n", what, cmd)
		return 2
	}

	announce(cmd, r, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	if *cleanup {
		return k.Cleanup(root, *dryRun, *noLog, stdout)
	}
	return k.List(root, register.ListOptions{Status: filter, Tag: *tag, MinCost: *minCost, Untriaged: *untriaged, ByValue: *sortBy == "value"}, stdout)
}
