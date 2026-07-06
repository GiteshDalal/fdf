package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// version is set by goreleaser via -ldflags "-X main.version=...".
var version = "0.2.0-dev"

var commands = map[string]func(args []string, stdout io.Writer) int{
	"validate": runValidate,
	"init":     runInit,
	"new":      runNew,
	"install":  runInstall,
	"serve":    runServe,
	"migrate":  runMigrate,
}

const usage = `fdf — Feature Document Format tooling (SPEC v0.2)

Usage: fdf <command> [flags]

Commands:
  validate   Check the bundle against SPEC v0.2 (F1-F8 + R1)
  init       Scaffold a bundle at the resolved root
  new        Scaffold a draft feature: fdf new <group>/<slug>
  install    Install/upgrade the FDF skills for an AI harness (claude-code|codex|opencode)
  serve      Serve the bundle in a browser (wraps bun x mdts)
  migrate    Upgrade a bundle to the current spec version
  version    Print the CLI version

Bundle root: --root flag > FDF_ROOT_DIR env > docs/features (relative paths
resolve against the project root).

Flags must precede positional arguments (e.g. fdf new --root <dir> <group>/<slug>).`

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
