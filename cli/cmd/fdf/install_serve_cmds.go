package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GiteshDalal/fdf/cli/internal/install"
)

func runInstall(args []string, stdout io.Writer) int {
	if len(args) != 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(stdout, "usage: fdf install <claude-code|codex|opencode>\n\nInstalls (or auto-upgrades) the FDF skills for the given AI harness.")
		return 2
	}
	install.Version = version
	return install.Run(args[0], "", stdout)
}

func runServe(args []string, stdout io.Writer) int {
	fs := newFlagSet("serve", stdout)
	root, _, ok := resolveRoot(fs, args, stdout)
	if !ok {
		return 2
	}
	if _, err := exec.LookPath("bun"); err != nil {
		fmt.Fprintf(stdout, "fdf serve wraps `bun x mdts`. bun is not installed — install it (https://bun.sh) or run your own markdown server over %s\n", root)
		return 1
	}
	cmd := exec.Command("bun", "x", "mdts", root)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(stdout, "error:", err)
		return 1
	}
	return 0
}
