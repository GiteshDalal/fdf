package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/logs"
)

// runLog writes one entry to the log it belongs in: the <slug>.log.md beside
// the feature, Change, Fix, practice, debt or bug it is about, a group's
// LOG.md, or the bundle-root LOG.md when no ID is given. It creates that log
// on first use and keeps it newest first, so logging is one command rather
// than a file an agent has to remember how to start.
func runLog(args []string, stdout io.Writer) int {
	fs := newFlagSet("log")
	root, source, rest, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	var id, entry string
	switch len(rest) {
	case 1:
		entry = rest[0]
		if logs.IsID(root, entry) {
			fmt.Fprintf(stdout, "usage: %q names a document — the entry comes after it: fdf log %s \"<entry>\"\n", entry, entry)
			return 2
		}
	case 2:
		id, entry = rest[0], rest[1]
	default:
		printUsage(stdout, "log")
		return 2
	}
	if strings.TrimSpace(entry) == "" {
		fmt.Fprintln(stdout, "error: the entry is empty")
		printUsage(stdout, "log")
		return 2
	}
	announce("log", root, source, stdout)
	return logs.Append(root, id, entry, stdout)
}
