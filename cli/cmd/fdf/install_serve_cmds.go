package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GiteshDalal/fdf/cli/internal/install"
)

func runInstall(args []string, stdout io.Writer) int {
	fs := newFlagSet("install", stdout)
	rootFlag := fs.String("root", "", "bundle root to bake into the installed skills (default docs/features; FDF_ROOT_DIR is honored)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stdout, "usage: fdf install [--root <dir>] <claude-code|codex|opencode>\n\nInstalls (or auto-upgrades) the FDF skills and instruction-file primer for\nthe given AI harness. --root (or FDF_ROOT_DIR) rewrites the bundle-root path\nreferenced by the installed skills; keep it project-relative.")
		return 2
	}
	root := *rootFlag
	if root == "" {
		root = os.Getenv("FDF_ROOT_DIR")
	}
	install.Version = version
	return install.Run(rest[0], "", root, stdout)
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
