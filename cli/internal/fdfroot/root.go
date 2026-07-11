// Package fdfroot resolves the project root (superproject-aware) and the
// bundle root (--root flag > FDF_ROOT_DIR > docs/features), per SPEC v0.2.
package fdfroot

import (
	"os"
	"path/filepath"
)

// ProjectRoot walks up from start to the topmost enclosing git working tree.
// A .git directory marks a working tree; a .git FILE marks a submodule
// boundary — the walk records it and continues to the superproject.
// standalone is true only when no .git (file or dir) exists anywhere above.
func ProjectRoot(start string) (string, bool) {
	cur, _ := filepath.Abs(start)
	var lastGit string
	for {
		if fi, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			lastGit = cur
			if fi.IsDir() {
				// a real repo; keep walking only if it is itself nested —
				// the topmost .git DIRECTORY wins, so remember and continue.
			}
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
	val := flagRoot
	if val == "" {
		val = os.Getenv("FDF_ROOT_DIR")
	}
	if val == "" {
		val = filepath.Join("docs", "features")
	}
	if filepath.IsAbs(val) {
		return filepath.Clean(val), nil
	}
	root, _ := ProjectRoot(cwd)
	return filepath.Join(root, val), nil
}
