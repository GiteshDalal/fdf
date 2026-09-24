package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/install"
)

func runInstall(args []string, stdout io.Writer) int {
	fs := newFlagSet("install")
	rootFlag := fs.String("root", "", "bundle root to bake into the installed skills (default docs/features; FDF_ROOT_DIR is honored)")
	project := fs.Bool("project", false, "install into the current git project instead of the user home directory")
	rest, exit, ok := parseArgs(fs, args, stdout)
	if !ok {
		return exit
	}
	if len(rest) != 1 {
		printUsage(stdout, "install")
		return 2
	}
	if !install.IsHarness(rest[0]) {
		fmt.Fprintf(stdout, "error: unknown harness %q — fdf installs for claude-code, codex or opencode\n", rest[0])
		printUsage(stdout, "install")
		return 2
	}
	root := *rootFlag
	if root == "" {
		root = os.Getenv("FDF_ROOT_DIR")
	}
	base := ""
	if *project {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stdout, "error:", err)
			return 1
		}
		// Nearest .git wins: "the current git project" is the repo the user
		// is standing in, not the superproject R1's ProjectRoot resolves to.
		projRoot, standalone := fdfroot.NearestProjectRoot(cwd)
		if standalone {
			fmt.Fprintln(stdout, "fdf install --project requires a git project (no .git found above the current directory)")
			return 2
		}
		base = projRoot
	}
	install.Version = version
	target := "home directory"
	if *project {
		target = "project " + base
	}
	fmt.Fprintf(stdout, "fdf %s · install · harness: %s · target: %s\n\n", version, rest[0], target)
	return install.Run(rest[0], base, root, *project, stdout)
}

func runServe(args []string, stdout io.Writer) int {
	fs := newFlagSet("serve")
	root, source, _, exit, ok := resolveRootSource(fs, args, stdout)
	if !ok {
		return exit
	}
	announce("serve", root, source, stdout)
	if !requireBundle(root, stdout) {
		return 1
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
