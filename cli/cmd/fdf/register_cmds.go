package main

import (
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
	root, source, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}

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
	if len(rest) == 1 {
		if filter != "" || *cleanup {
			fmt.Fprintf(stdout, "usage: fdf %s %s scaffolds a %s; drop the flags to read the register\n", cmd, rest[0], cmd)
			return 2
		}
		announce(cmd, root, source, stdout)
		if !requireBundle(root, stdout) {
			return 1
		}
		return k.New(root, rest[0], affected, splitList(*resource), stdout)
	}
	if len(affected) > 0 || *resource != "" {
		if k.Type == "Bug" {
			fmt.Fprintln(stdout, "usage: --affects and --resource apply when filing a bug: fdf bug [--affects <ids>] [--resource <paths>] [<group>/]<slug>")
		} else {
			fmt.Fprintf(stdout, "usage: --resource applies when filing a %s: fdf %s [--resource <paths>] [<group>/]<slug>\n", cmd, cmd)
		}
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

	announce(cmd, root, source, stdout)
	if !requireBundle(root, stdout) {
		return 1
	}
	if *cleanup {
		return k.Cleanup(root, *dryRun, *noLog, stdout)
	}
	return k.List(root, filter, stdout)
}
