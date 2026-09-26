package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// version is set by goreleaser via -ldflags "-X main.version=...".
var version = "0.7.1"

// banner is the product header both the overview and `fdf help` print.
// The spec version is derived, never written out, so a spec bump cannot
// leave a stale literal behind in one of the two.
var banner = fmt.Sprintf("fdf — Feature Document Format tooling (SPEC v%s)", scaffold.CurrentVersion())

var commands = map[string]func(args []string, stdout io.Writer) int{
	"validate": runValidate,
	"init":     runInit,
	"new":      runNew,
	"practice": runPractice,
	"debt":     runDebt,
	"bug":      runBug,
	"adopt":    runAdopt,
	"mv":       runMv,
	"lexicon":  runLexicon,
	"log":      runLog,
	"install":  runInstall,
	"serve":    runServe,
	"migrate":  runMigrate,
	"help":     runHelp,
	"spec":     runSpec,
	"change":   runChange,
	"fix":      runFix,
	"history":  runHistory,
	"release":  runRelease,
	"version":  runVersion,
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches one invocation. What goes wrong before a command runs — no
// command, or a name fdf does not have — is reported on stderr; a command
// writes everything, its errors included, to stdout.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, overview())
		return 2
	}
	name := args[0]
	switch name {
	case "-h", "-help", "--help":
		fmt.Fprintln(stdout, overview())
		return 0
	case "-version", "--version":
		name = "version"
	}
	cmd, ok := commands[name]
	if !ok {
		unknownCommand(stderr, "fdf", name)
		return 2
	}
	return cmd(args[1:], stdout)
}

func runVersion(args []string, stdout io.Writer) int {
	fs := newFlagSet("version")
	rest, exit, ok := parseArgs(fs, args, stdout)
	if !ok {
		return exit
	}
	if len(rest) > 0 {
		printUsage(stdout, "version")
		return 2
	}
	fmt.Fprintln(stdout, "fdf", version)
	return 0
}

// announce prints the one line that makes a command's behavior explainable:
// which binary is running, and which bundle root it chose and why. A stale
// version held in place by a shim (mise, asdf) and a root resolved somewhere
// unexpected are the two failures that otherwise look like "the tool did
// nothing", and neither is visible from the command's own output.
func announce(cmd, root, source string, stdout io.Writer) {
	fmt.Fprintf(stdout, "fdf %s · %s · root: %s (%s)\n\n", version, cmd, root, source)
}

// requireBundle stops a command that works on a bundle when the root holds
// none, in the words every command uses for it.
func requireBundle(root string, stdout io.Writer) bool {
	if err := fdfroot.CheckBundle(root); err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return false
	}
	return true
}

// flagSetHook, when set, is handed every FlagSet a command builds: the tests
// use it to hold each command's flags to its help topic.
var flagSetHook func(*flag.FlagSet)

// newFlagSet builds the FlagSet a command parses with. Parse errors come back
// to parseArgs (ContinueOnError), which prints them with the command's usage
// line. Go's own flag listing is never printed: it reads back-quoted words as
// value names, and lists flags a command only defines to refuse.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	if flagSetHook != nil {
		flagSetHook(fs)
	}
	return fs
}

// strayArg names, for each command that takes no arguments, the flag an
// argument given to it most likely belongs to. These commands used to discard
// a stray argument, so `fdf migrate docs/features` ran against the DEFAULT
// root and reported an error about a path the user never typed — which reads
// as the command doing nothing at all.
var strayArg = map[string]string{
	"validate": "--root",
	"init":     "--root",
	"migrate":  "--root",
	"serve":    "--root",
	"lexicon":  "--term",
	"spec":     "-v",
}

// parseArgs parses a command's flags and returns its arguments. When the
// command must stop, ok is false and exit is the code to stop with, the
// reason already printed: 0 after the help -h or --help asked for, 2 after a
// usage error.
//
// Go's flag package stops at the first argument, so a flag written after one
// is read as an argument, and the command would run on the wrong ones. It is
// refused instead, with the command as it should have been typed. Only a token
// naming one of fs's flags counts — a log entry that starts with "-" is not
// one — and "--" ends the check: everything after it is an argument.
func parseArgs(fs *flag.FlagSet, args []string, stdout io.Writer) (rest []string, exit int, ok bool) {
	for _, a := range args {
		if a == "--" {
			break
		}
		if a == "-h" || a == "-help" || a == "--help" {
			printTopic(stdout, fs.Name())
			return nil, 0, false
		}
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printTopic(stdout, fs.Name())
			return nil, 0, false
		}
		// An argument that starts with a dash — a log entry written as a
		// list item — is read as a flag unless -- comes before it.
		for _, a := range args {
			if a == "--" {
				break
			}
			if _, isFlag := flagName(fs, a); !isFlag && strings.HasPrefix(a, "-") && strings.ContainsAny(a, " \t") {
				fmt.Fprintf(stdout, "error: %s starts with a dash, so it was read as a flag. Put -- before it:\n  %s\n", shellQuote(a), reordered(fs, args))
				return nil, 2, false
			}
		}
		fmt.Fprintln(stdout, "error:", err)
		printUsage(stdout, fs.Name())
		return nil, 2, false
	}
	rest = fs.Args()
	// A "--" the flag package consumed ended the flags before any argument.
	if n := len(args) - len(rest); n == 0 || args[n-1] != "--" {
		for i, a := range rest {
			if a == "--" {
				rest = append(rest[:i:i], rest[i+1:]...)
				break
			}
			if _, isFlag := flagName(fs, a); isFlag {
				fmt.Fprintf(stdout, "error: %s comes after an argument, so it was not read as a flag. Flags go first:\n  %s\n", a, reordered(fs, args))
				return nil, 2, false
			}
		}
	}
	if strayArg[fs.Name()] != "" && len(rest) > 0 {
		fmt.Fprintf(stdout, "error: fdf %s takes no positional arguments — did you mean `%s`?\n", fs.Name(), reordered(fs, args))
		return nil, 2, false
	}
	return rest, 0, true
}

// flagName reports the flag a token names, when it names one of fs's: -name,
// --name, -name=value or --name=value.
func flagName(fs *flag.FlagSet, a string) (string, bool) {
	if len(a) < 2 || a[0] != '-' {
		return "", false
	}
	name, _, _ := strings.Cut(strings.TrimLeft(a, "-"), "=")
	return name, name != "" && fs.Lookup(name) != nil
}

func isBoolFlag(f *flag.Flag) bool {
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && b.IsBoolFlag()
}

// reordered is the command as it should have been typed: every flag, with its
// value, before the arguments. A command that takes no arguments gets none:
// the first one goes to the flag it most likely meant (strayArg), unless that
// flag was given too.
func reordered(fs *flag.FlagSet, args []string) string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		name, isFlag := flagName(fs, a)
		if !isFlag {
			pos = append(pos, a)
			continue
		}
		flags = append(flags, a)
		if !strings.Contains(a, "=") && !isBoolFlag(fs.Lookup(name)) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	if f := strayArg[fs.Name()]; f != "" && len(pos) > 0 {
		given := false
		for _, a := range flags {
			if name, isFlag := flagName(fs, a); isFlag && name == strings.TrimLeft(f, "-") {
				given = true
			}
		}
		if !given {
			flags = append(flags, f, pos[0])
		}
		pos = nil
	}
	words := []string{"fdf", fs.Name()}
	for _, f := range flags {
		words = append(words, shellQuote(f))
	}
	for _, p := range pos {
		if strings.HasPrefix(p, "-") {
			words = append(words, "--")
			break
		}
	}
	for _, p := range pos {
		words = append(words, shellQuote(p))
	}
	return strings.Join(words, " ")
}

// shellQuote writes an argument so a shell reads it back as the same word.
func shellQuote(s string) string {
	plain := s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./,:=@%+", r))
	}) < 0
	if plain {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
