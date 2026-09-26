// Package fdfroot resolves the project root (superproject-aware) and the
// bundle root: --root, then FDF_ROOT_DIR, then docs/fdf, or docs/features
// where a bundle from before 1.0 still is.
package fdfroot

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// NoBundle is what every command says when the bundle root holds no bundle,
// so a wrong --root reads the same whichever command met it first. A root
// that holds Markdown nonetheless, such as a bundle from before 1.0 that
// never had an INDEX.md, is sent to fdf migrate, not to fdf init, which
// refuses it (Unindexed).
func NoBundle(root string) error {
	if file, route := Unindexed(root); file != "" {
		return fmt.Errorf("no bundle at %s (no INDEX.md), though it holds %s — %s; or point --root at the bundle", root, file, route)
	}
	return fmt.Errorf("no bundle at %s (no INDEX.md) — run `fdf init` first, or point --root at the bundle", root)
}

// Unindexed returns a Markdown file in the directory at root, which holds no
// INDEX.md, by its path from root, and how fdf migrate takes the bundle from
// before 1.0 that the file makes it: a v0.1 bundle's index.md, which migrate
// renames, as it is, and any other once it has an INDEX.md, which need not
// pin a version. Both are "" when root holds no Markdown but its README.md;
// what is hidden is passed over.
func Unindexed(root string) (file, route string) {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.Name() == "index.md" {
			return "index.md", fmt.Sprintf("run `fdf migrate --root %s`, which renames a v0.1 bundle's index.md", root)
		}
	}
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case p != root && strings.HasPrefix(d.Name(), "."):
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		case d.IsDir() || !strings.HasSuffix(d.Name(), ".md"):
			return nil
		}
		if rel, _ := filepath.Rel(root, p); rel != "README.md" {
			file = filepath.ToSlash(rel)
			return filepath.SkipAll
		}
		return nil
	})
	if file == "" {
		return "", ""
	}
	return file, fmt.Sprintf("a bundle from before 1.0 needs an INDEX.md, which need not pin a version, committed where git tracks it, and then `fdf migrate --root %s`", root)
}

// InsideBundle is what every command, fdf migrate included, says when root is
// a directory inside the bundle at bundle — one of its registers or groups,
// whose INDEX.md pins nothing — so the slip reads the same whichever command
// met it first.
func InsideBundle(root, bundle string) error {
	return fmt.Errorf("%s is inside the bundle at %s, not a bundle of its own — pass --root %s, or leave --root out", root, bundle, bundle)
}

// Upgrading is how a message that meets a bundle from before 1.0 names the
// upgrade of what: the user's decision, which fdf migrate's dry run informs,
// with --root when root is not "". The upgrade moves the bundle's documents
// and rewrites references to them across the project, and an agent reads
// these messages: fdf 0.7's skills told it to run fdf migrate whenever a pin
// was not supported.
func Upgrading(what, root string) string {
	dryRun := "`fdf migrate --dry-run`"
	if root != "" {
		dryRun = "`fdf migrate --root " + root + " --dry-run`"
	}
	return "upgrading " + what + " is the user's decision, since `fdf migrate` moves its documents and rewrites references to them across the project: " + dryRun + " shows the plan"
}

// CheckBundle returns NoBundle unless root holds a bundle: an INDEX.md at its
// top.
func CheckBundle(root string) error {
	if _, err := os.Stat(filepath.Join(root, "INDEX.md")); err != nil {
		return NoBundle(root)
	}
	return nil
}

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

// Submodule reports whether dir is the root of a git submodule: it holds a
// .git file, and not one that belongs to a linked worktree.
func Submodule(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && !fi.IsDir() && !linkedWorktree(dir)
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

// Resolution is the bundle root a command works on, and which input chose it.
// Commands print both: when a tool silently looks somewhere other than where
// the user expects, the resolved path alone rarely explains why.
type Resolution struct {
	Root   string
	Source string // "--root", "FDF_ROOT_DIR", "default docs/fdf" or "pre-1.0 default docs/features"
	// Shadowed is a second bundle the default passed over: docs/features,
	// when docs/fdf holds a bundle too.
	Shadowed string
}

// Resolve applies the uniform root-resolution precedence: --root, then
// FDF_ROOT_DIR, then the default. A relative value resolves against the
// project root, and an absolute one is used as it is.
func Resolve(flagRoot, cwd string) Resolution {
	val, source := flagRoot, "--root"
	if val == "" {
		val, source = os.Getenv("FDF_ROOT_DIR"), "FDF_ROOT_DIR"
	}
	pr, _ := ProjectRoot(cwd)
	switch {
	case val == "":
		return Default(pr)
	case filepath.IsAbs(val):
		return Resolution{Root: filepath.Clean(val), Source: source}
	}
	return Resolution{Root: filepath.Join(pr, val), Source: source}
}

// Default is the bundle root when neither --root nor FDF_ROOT_DIR names one:
// the first of docs/fdf and docs/features under projectRoot that holds a
// bundle, so an upgraded fdf still finds a bundle from before 1.0, and
// docs/fdf, where `fdf init` creates one, when neither does. docs/features
// holds a bundle only when it is pinned: the rest of docs/ belongs to the
// project, and a documentation site may have a docs/features section.
func Default(projectRoot string) Resolution {
	root, old := filepath.Join(projectRoot, "docs", "fdf"), filepath.Join(projectRoot, "docs", "features")
	switch {
	case !pinned(old):
		return Resolution{Root: root, Source: "default docs/fdf"}
	case CheckBundle(root) == nil:
		return Resolution{Root: root, Source: "default docs/fdf", Shadowed: old}
	}
	return Resolution{Root: old, Source: "pre-1.0 default docs/features"}
}

// pinKeyRe is a frontmatter line that holds the fdf_version key.
var pinKeyRe = regexp.MustCompile(`^fdf_version:\s?(.*)$`)

// Pin returns the spec version that index, the text of a bundle's root
// INDEX.md, pins: the value of the fdf_version key in its frontmatter,
// unquoted, or "" when it has none. It is the one reader of a pin: the
// validator, every command, root resolution and fdf migrate read it here.
// The key is read line by line, so a line of the frontmatter that does not
// parse hides no pin, and the validator, which reports that line as F1, reads
// the pin the commands read. A pin written anywhere but the frontmatter, such
// as in a sample in the body, is none, and so is one in a block that no `---`
// line closes (Unclosed).
func Pin(index string) string {
	block, closed := frontmatter(index)
	if !closed {
		return ""
	}
	// A delimited block: the pin is its fdf_version key, unquoted.
	for _, l := range block {
		if m := pinKeyRe.FindStringSubmatch(l); m != nil {
			return strings.Trim(strings.Trim(strings.TrimSpace(m[1]), `"`), `'`)
		}
	}
	return ""
}

// frontmatter returns the lines of the frontmatter block that index opens, a
// byte order mark and CRLF line ends tolerated, and whether a `---` line
// closes it; nil when index opens none.
func frontmatter(index string) (block []string, closed bool) {
	lines := strings.Split(strings.ReplaceAll(strings.TrimPrefix(index, "\uFEFF"), "\r\n", "\n"), "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return nil, false
	}
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return lines[1 : i+1], true
		}
	}
	return lines[1:], false
}

// PinOf returns the pin of the bundle at root: Pin of its INDEX.md, or ""
// when it has none, or no INDEX.md.
func PinOf(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if err != nil {
		return ""
	}
	return Pin(string(raw))
}

// Unclosed reports whether the INDEX.md at root opens a frontmatter block
// that no `---` line closes. Pin reads no pin there, whatever the block
// says, and the readers of a pin say so rather than that it is missing.
func Unclosed(root string) bool {
	raw, err := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if err != nil {
		return false
	}
	block, closed := frontmatter(string(raw))
	return block != nil && !closed
}

// pinned reports whether dir holds an INDEX.md, its name spelled exactly so,
// that pins an fdf_version, as every bundle fdf has written since 0.2 does.
// A site's docs/features/index.md is no bundle, though a disk that ignores
// case opens it as INDEX.md.
func pinned(dir string) bool {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() == "INDEX.md" {
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			return err == nil && Pin(string(raw)) != ""
		}
	}
	return false
}

// BundleAbove returns the nearest directory above root whose INDEX.md pins an
// fdf_version, as pinned reads it, or "" when there is none. A root whose own
// INDEX.md pins nothing, below such a directory, is no bundle but a register
// or a group of that one: --root docs/fdf/features, meant for docs/fdf.
func BundleAbove(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	for dir := filepath.Dir(abs); ; dir = filepath.Dir(dir) {
		if pinned(dir) {
			return dir
		}
		if dir == filepath.Dir(dir) {
			return ""
		}
	}
}
