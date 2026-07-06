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
