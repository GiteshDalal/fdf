// Package fdfroot resolves the project root (superproject-aware) and the
// bundle root (--root flag > FDF_ROOT_DIR > docs/features), per SPEC v0.2.
package fdfroot

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectRoot walks up from start to the topmost enclosing git working tree.
// A .git directory marks a working tree; a .git FILE marks a submodule
// boundary — the walk records it and continues to the superproject. A .git
// file that belongs to a linked worktree (`git worktree add`) ends the walk:
// the worktree is a checkout of its own, even when it sits inside the main
// repository's directory, and its paths are the ones to check.
// standalone is true only when no .git (file or dir) exists anywhere above.
func ProjectRoot(start string) (string, bool) {
	cur, _ := filepath.Abs(start)
	var lastGit string
	for {
		if fi, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			lastGit = cur
			if !fi.IsDir() && linkedWorktree(cur) {
				return cur, false
			}
			// Otherwise the topmost .git wins: remember it and keep walking.
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	if lastGit != "" {
		return lastGit, false
	}
	abs, _ := filepath.Abs(start)
	return abs, true
}

// linkedWorktree reports whether dir's .git file points at a linked
// worktree's administrative directory, which holds a `commondir` file. A
// submodule's .git file points at a directory under the superproject's
// .git/modules, which does not.
func linkedWorktree(dir string) bool {
	raw, err := os.ReadFile(filepath.Join(dir, ".git"))
	if err != nil {
		return false
	}
	line := strings.TrimSpace(strings.SplitN(string(raw), "\n", 2)[0])
	if !strings.HasPrefix(line, "gitdir:") {
		return false
	}
	gitdir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(dir, gitdir)
	}
	_, err = os.Stat(filepath.Join(gitdir, "commondir"))
	return err == nil
}

// NearestProjectRoot walks up from start to the NEAREST enclosing git working
// tree — the first .git (directory or file) wins. This is what "the current
// git project" means to a user standing in a nested repo, a git worktree
// (whose root holds a .git file), or a submodule: that tree, not its
// superproject. Contrast with ProjectRoot, whose topmost-wins contract exists
// for R1 resource verification. standalone is true when no .git exists
// anywhere above.
func NearestProjectRoot(start string) (string, bool) {
	cur, _ := filepath.Abs(start)
	for {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur, false
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	abs, _ := filepath.Abs(start)
	return abs, true
}

// BundleRoot applies the uniform root-resolution precedence.
func BundleRoot(flagRoot, cwd string) (string, error) {
	root, _, err := BundleRootWithSource(flagRoot, cwd)
	return root, err
}

// BundleRootWithSource resolves the bundle root and reports WHICH input chose
// it ("--root", "FDF_ROOT_DIR", or "default docs/features"). Commands print
// that alongside the path: when a tool silently looks somewhere other than
// where the user expects, the resolved path alone rarely explains why.
func BundleRootWithSource(flagRoot, cwd string) (root, source string, err error) {
	val, source := flagRoot, "--root"
	if val == "" {
		val, source = os.Getenv("FDF_ROOT_DIR"), "FDF_ROOT_DIR"
	}
	if val == "" {
		val, source = filepath.Join("docs", "features"), "default docs/features"
	}
	if filepath.IsAbs(val) {
		return filepath.Clean(val), source, nil
	}
	pr, _ := ProjectRoot(cwd)
	return filepath.Join(pr, val), source, nil
}
