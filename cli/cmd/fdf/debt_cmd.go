package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/debt"
)

// runDebt is the register's whole surface: read it, file into it, clear what
// is paid. With a positional slug it scaffolds; without one it reports.
func runDebt(args []string, stdout io.Writer) int {
	fs := newFlagSet("debt", stdout)
	open := fs.Bool("open", false, "list only open debts")
	accepted := fs.Bool("accepted", false, "list only accepted debts")
	resolved := fs.Bool("resolved", false, "list only resolved debts")
	cleanup := fs.Bool("cleanup", false, "fold resolved debts into debts/LOG.md and clear them from the register")
	dryRun := fs.Bool("dry-run", false, "with --cleanup: show what would be cleared and change nothing")
	noLog := fs.Bool("no-log", false, "with --cleanup: remove resolved debts without writing debts/LOG.md")
	root, source, rest, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return 2
	}

	var filters []string
	for name, on := range map[string]bool{"open": *open, "accepted": *accepted, "resolved": *resolved} {
		if on {
			filters = append(filters, name)
		}
	}
	if len(filters) > 1 {
		fmt.Fprintln(stdout, "usage: fdf debt takes at most one of --open, --accepted, --resolved")
		return 2
	}
	filter := ""
	if len(filters) == 1 {
		filter = filters[0]
	}

	if len(rest) > 1 {
		fmt.Fprintln(stdout, "usage: fdf debt [--open|--accepted|--resolved] [--cleanup [--dry-run] [--no-log]] [[<group>/]<slug>]")
		return 2
	}
	if len(rest) == 1 {
		if filter != "" || *cleanup {
			fmt.Fprintf(stdout, "usage: fdf debt %s scaffolds a debt; drop the flags to read the register\n", rest[0])
			return 2
		}
		announce("debt", root, source, stdout)
		return debt.New(root, rest[0], stdout)
	}
	if (*dryRun || *noLog) && !*cleanup {
		fmt.Fprintln(stdout, "usage: --dry-run and --no-log only apply to `fdf debt --cleanup`")
		return 2
	}

	announce("debt", root, source, stdout)
	if *cleanup {
		return debt.Cleanup(root, *dryRun, *noLog, stdout)
	}
	return debt.List(root, filter, stdout)
}

// debtStatusList renders the vocabulary for help text.
func debtStatusList() string { return strings.Join(debt.Statuses, " → ") }
