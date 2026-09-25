package migrate

// Where the bundle goes, and git, which is the migration's undo: a bundle at
// …/docs/features moves to …/docs/fdf beside it, --to chooses another
// destination, and a submodule moves with git mv. In the git repository
// that tracks the bundle, migrate starts only from a clean tree, marks the
// files it writes so that `git diff -M` shows every move, and, should it
// stop partway, says which commands put everything back.

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
)

// destination is where the bundle at root goes: to when it is given,
// docs/fdf beside a bundle at …/docs/features however it was found, and
// where it is otherwise. It refuses a destination that is there and not
// empty, one inside the bundle, one inside git's own directory, and one a
// file stands in the way of; and, in the git repository at project, one
// outside the project, or the move of a bundle that is its own repository.
// problem says why.
func destination(to, root, project string) (dest, problem string) {
	dest = root
	switch {
	case to != "":
		dest = onDisk(filepath.Clean(to))
	case filepath.Base(root) == "features" && filepath.Base(filepath.Dir(root)) == "docs":
		dest = filepath.Join(filepath.Dir(root), "fdf")
	}
	if dest == root {
		return dest, ""
	}
	switch {
	case within(root, dest):
		return "", fmt.Sprintf("%s is inside the bundle, which cannot move into itself", dest)
	case project == root:
		return "", fmt.Sprintf("the bundle at %s is the root of its own git repository, which migrate does not move — move it yourself, or pass --to %s to keep it where it is", root, root)
	case project != "" && !within(project, dest):
		return "", fmt.Sprintf("%s is outside the project at %s, where git could neither show the move nor undo it — pass --to a directory inside the project", dest, project)
	case strings.Contains(filepath.ToSlash(dest)+"/", "/.git/"):
		return "", fmt.Sprintf("%s is inside git's own directory — pass --to <dir> to choose another destination", dest)
	}
	for d := filepath.Dir(dest); d != filepath.Dir(d); d = filepath.Dir(d) {
		if fi, err := os.Stat(d); err == nil {
			if !fi.IsDir() {
				return "", fmt.Sprintf("%s is a file, where %s needs a directory — pass --to <dir> to choose another destination", d, dest)
			}
			break
		}
	}
	if fi, err := os.Stat(dest); err == nil {
		entries, _ := os.ReadDir(dest)
		if !fi.IsDir() || len(entries) > 0 {
			return "", fmt.Sprintf("%s is there and not empty — pass --to <dir> to choose another destination, or --to %s to keep the bundle where it is", dest, root)
		}
	}
	return dest, ""
}

// within reports whether path is dir or under it.
func within(dir, path string) bool {
	r, err := filepath.Rel(dir, path)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

// onDisk is p as the disk spells it: each name that is there, as its
// directory lists it. On a disk that ignores case, --root Docs/Features
// finds docs/features, the path git knows, since git reads paths exactly; a
// name that is not there stays as given.
func onDisk(p string) string {
	parent, name := filepath.Dir(p), filepath.Base(p)
	if parent == p {
		return p
	}
	parent = onDisk(parent)
	entries, err := os.ReadDir(parent)
	if err != nil {
		return filepath.Join(parent, name)
	}
	for _, e := range entries {
		if e.Name() == name {
			return filepath.Join(parent, name)
		}
	}
	for _, e := range entries {
		if strings.EqualFold(e.Name(), name) {
			return filepath.Join(parent, e.Name())
		}
	}
	return filepath.Join(parent, name)
}

// repository is the root of the git repository that tracks the bundle at
// root, and so undoes its migration: the nearest directory at or above root
// that holds a .git — not the topmost, since a repository checked out inside
// another tracks its own files. A bundle that is a submodule's root is moved
// by the repository around it, with git mv, so for one it is that
// repository's root. It is "" outside a git repository.
func repository(root string) string {
	for d := root; ; d = filepath.Dir(d) {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			if d == root && fdfroot.Submodule(d) {
				if super := repository(filepath.Dir(d)); super != "" {
					return super
				}
			}
			return d
		}
		if filepath.Dir(d) == d {
			return ""
		}
	}
}

// git runs git in dir and returns what it printed. It takes no optional
// lock, so that git status, which would refresh the index, leaves a dry run
// changing nothing, git's own files included.
func git(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"--no-optional-locks", "-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// dirty lists what git reports under the paths migrate writes, in the
// project at project: a change not committed, staged or not, and a file git
// does not track, one it ignores included, since git could not put it back.
// A hidden file, such as a Finder .DS_Store, is none of migrate's. Git is
// the migration's undo, and a diff that mixes someone's edits with
// migrate's can be neither reviewed nor reverted cleanly.
func (p *plan) dirty(project string) ([]string, error) {
	status := func(dir string, paths ...string) ([]string, error) {
		out, err := git(dir, append([]string{"status", "--porcelain", "--untracked-files=all", "--ignored", "--"}, paths...)...)
		if err != nil {
			return nil, err
		}
		var lines []string
		for _, l := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			if l != "" && !(strings.HasPrefix(l, "!! ") && hidden(l[3:])) {
				lines = append(lines, l)
			}
		}
		return lines, nil
	}
	var paths []string
	var lines []string
	switch {
	case p.submodule:
		// The bundle's files are in the submodule's own repository.
		inside, err := status(p.root, ".")
		if err != nil {
			return nil, err
		}
		lines = append(lines, inside...)
		if p.relocates() {
			paths = append(paths, ".gitmodules")
		}
	case project == p.root:
		paths = append(paths, ".")
	default:
		paths = append(paths, p.old)
		if p.relocates() {
			paths = append(paths, p.new)
		}
	}
	if len(paths) > 0 {
		out, err := status(project, paths...)
		if err != nil {
			return nil, err
		}
		lines = append(lines, out...)
	}
	return lines, nil
}

// hidden reports whether a path git names is a hidden file, or one in a
// hidden directory.
func hidden(p string) bool {
	for _, name := range strings.Split(strings.Trim(p, `"`), "/") {
		if strings.HasPrefix(name, ".") {
			return true
		}
	}
	return false
}

// relocates reports whether the bundle itself moves.
func (p *plan) relocates() bool { return p.new != p.old }

// dest is the bundle's directory once migrated.
func (p *plan) dest() string { return filepath.Join(p.base, filepath.FromSlash(p.new)) }

// relocate moves the bundle to its destination: a plain directory is
// renamed, and a submodule is moved with git mv, which also updates
// .gitmodules. An empty directory there gives way to it.
func (p *plan) relocate() error {
	if !p.relocates() {
		return nil
	}
	dest := p.dest()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	os.Remove(dest)
	if p.submodule {
		if _, err := git(p.base, "mv", p.old, p.new); err != nil {
			return err
		}
	} else if err := os.Rename(p.root, dest); err != nil {
		return err
	}
	p.relocated = true
	return nil
}

// markNew marks every file of the migrated bundle that git does not track
// with git add --intent-to-add, which stages no content: `git diff -M` then
// shows each move as a rename, and the edits in it. --ignore-removal leaves
// the files that moved away unstaged, so that the diff can pair them. A
// rename that changes only case, in a bundle that stays where it is, is one
// git cannot see on a disk that ignores case, where the old name still finds
// the file: git forgets the old name first, so that the new one is marked,
// and committed, as it is spelled.
func (p *plan) markNew(project string) error {
	dir, at := project, p.new
	if p.submodule || project == p.root {
		dir, at = p.dest(), "."
	}
	for _, o := range sortedKeys(p.moves) {
		if n := p.to(o); p.relocates() || o == n || !strings.EqualFold(o, n) {
			continue
		}
		if _, err := os.Lstat(filepath.Join(p.root, filepath.FromSlash(o))); err != nil {
			continue // a disk that reads case: git sees the rename
		}
		if _, err := git(dir, "rm", "--cached", "--quiet", "--", path.Join(at, o)); err != nil {
			return err
		}
	}
	_, err := git(dir, "add", "--intent-to-add", "--ignore-removal", "--", at)
	return err
}

// undo is what puts everything back after a migration that stopped partway:
// in a git repository, the clean tree migrate started from is there to
// restore. A bundle that moved goes back first: a plain directory with mv,
// which takes back the files git ignores too, and a submodule with git mv,
// then .gitmodules as it was. Then git restores what migrate changed, and
// removes what it wrote.
func (p *plan) undo(project string) []string {
	q := quote
	switch {
	case project == "":
		return []string{"the bundle is not in a git repository: restore it from a copy"}
	case project == p.root:
		return []string{fmt.Sprintf("git -C %s checkout -- . && git -C %s clean -fd", q(project), q(project))}
	}
	var lines []string
	if p.submodule {
		old := q(filepath.Join(project, filepath.FromSlash(p.old)))
		if p.relocated {
			lines = append(lines, fmt.Sprintf("git -C %s mv %s %s", q(project), q(p.new), q(p.old)),
				fmt.Sprintf("git -C %s checkout HEAD -- .gitmodules", q(project)))
		}
		return append(lines, fmt.Sprintf("git -C %s checkout -- . && git -C %s clean -fd", old, old))
	}
	if p.relocated {
		lines = append(lines, fmt.Sprintf("mv %s %s", q(p.dest()), q(p.root)))
	}
	return append(lines, fmt.Sprintf("git -C %s checkout -- %s", q(project), q(p.old)),
		fmt.Sprintf("git -C %s clean -fd -- %s", q(project), q(p.old)))
}

// quote writes s as one word of a POSIX shell command: as it is when no
// character in it is special to a shell, and in single quotes otherwise.
func quote(s string) string {
	plain := s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./,:=@%+", r))
	}) < 0
	if plain {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
