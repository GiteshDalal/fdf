package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// version is set by goreleaser via -ldflags "-X main.version=...".
var version = "0.5.1"

// banner is the product header both the short usage and `fdf help` print.
// The spec version is derived, never written out, so a spec bump cannot
// leave a stale literal behind in one of the two.
var banner = fmt.Sprintf("fdf — Feature Document Format tooling (SPEC v%s)", scaffold.CurrentVersion())

var commands = map[string]func(args []string, stdout io.Writer) int{
	"validate": runValidate,
	"init":     runInit,
	"new":      runNew,
	"install":  runInstall,
	"serve":    runServe,
	"migrate":  runMigrate,
	"help":     runHelp,
	"spec":     runSpec,
	"change":   runChange,
	"fix":      runFix,
	"history":  runHistory,
	"release":  runRelease,
}

var usage = banner + `

Usage: fdf <command> [flags]

Commands:
  validate   Check the bundle against the pinned SPEC (F1-F10 + R1)
  init       Scaffold a bundle at the resolved root
  new        Scaffold a draft feature: fdf new <group>/<slug>
  change     Scaffold a post-delivery Change: fdf change --affects <id> <slug>
  fix        Scaffold a post-delivery Fix: fdf fix --affects <id> <slug>
  history    List a delivered feature's changes and fixes
  release    Derive a release document: fdf release [--ship] <version>
  install    Install/upgrade FDF skills for an AI harness: fdf install [--project] [--root <dir>] <claude-code|codex|opencode>
  serve      Serve the bundle in a browser (wraps bun x mdts)
  migrate    Upgrade a bundle to the current spec version
  spec       Print the format spec: fdf spec [-v <version>] [--list]
  version    Print the CLI version
  help       Verbose help with examples: fdf help [<command>]

Bundle root: --root flag > FDF_ROOT_DIR env > docs/features (relative paths
resolve against the project root).

Flags must precede positional arguments (e.g. fdf new --root <dir> <group>/<slug>).

Run 'fdf help' for the long form with examples for every command.`

// flagOrderHint explains a usage failure caused by argument order. Go's flag
// package stops parsing at the first non-flag argument, so a flag written
// after a positional is silently read as another positional — the command
// then fails on argument count, which explains nothing.
func flagOrderHint(rest []string, stdout io.Writer) {
	for _, a := range rest {
		if strings.HasPrefix(a, "-") {
			fmt.Fprintf(stdout, "note: %q was read as an argument, not a flag — flags must come before positional arguments\n", a)
			return
		}
	}
}

// announce prints the one line that makes a command's behavior explainable:
// which binary is running, and which bundle root it chose and why. A stale
// version held in place by a shim (mise, asdf) and a root resolved somewhere
// unexpected are the two failures that otherwise look like "the tool did
// nothing", and neither is visible from the command's own output.
func announce(cmd, root, source string, stdout io.Writer) {
	fmt.Fprintf(stdout, "fdf %s · %s · root: %s (%s)\n\n", version, cmd, root, source)
}

// rejectPositionals fails a command that accepts no positional arguments,
// naming the flag the user most likely meant. These commands used to discard
// stray args silently, so `fdf migrate docs/features` ran against the DEFAULT
// root and reported an error about a path the user never typed — which reads
// as the command doing nothing at all.
func rejectPositionals(cmd, flagName string, rest []string, stdout io.Writer) bool {
	if len(rest) == 0 {
		return true
	}
	fmt.Fprintf(stdout, "error: fdf %s takes no positional arguments — did you mean `fdf %s %s %s`?\n",
		cmd, cmd, flagName, rest[0])
	return false
}

// newFlagSet builds a FlagSet shared by every command: errors are handled by
// the caller (ContinueOnError) and usage/errors are written to stdout so
// tests can capture them alongside command output.
func newFlagSet(name string, stdout io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stdout)
	return fs
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if os.Args[1] == "version" {
		fmt.Println("fdf", version)
		return
	}
	cmd, ok := commands[os.Args[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "fdf: unknown command %q\n\n%s\n", os.Args[1], usage)
		os.Exit(2)
	}
	os.Exit(cmd(os.Args[2:], os.Stdout))
}
