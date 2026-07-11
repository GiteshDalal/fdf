package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/install"
)

const installUsage = `usage: fdf install [--project] [--root <dir>] <claude-code|codex|opencode>

Installs (or auto-upgrades) the FDF skills and instruction-file primer for
the given AI harness. Default is user-level (home directory). --project
installs into the current git project instead (skills under the harness's
project config dir; primer in the repo-root instruction file). --root (or
FDF_ROOT_DIR) rewrites the bundle-root path referenced by the installed
skills; keep it project-relative.`

func runInstall(args []string, stdout io.Writer) int {
	fs := newFlagSet("install", stdout)
	rootFlag := fs.String("root", "", "bundle root to bake into the installed skills (default docs/features; FDF_ROOT_DIR is honored)")
	project := fs.Bool("project", false, "install into the current git project instead of the user home directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stdout, installUsage)
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
	return install.Run(rest[0], base, root, *project, stdout)
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
