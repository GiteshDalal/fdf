package migrate

// Where the bundle goes, and git, which is the migration's undo: a bundle at
// …/docs/features moves to …/docs/fdf beside it, --to chooses another
// destination, and a submodule moves with git mv. In the git repository
// that tracks the bundle, migrate starts only from a clean tree, puts no file
// where git would ignore it, marks the files it writes so that `git diff -M`
// shows every move, and, should it stop partway, says which commands put
// everything back.

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
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
	paths := sortedKeys(p.outTexts)
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

// ignored lists, as the plan's refusals, what git would ignore of what
// migrate puts in a new place, in the repository where markNew marks it:
// the new path of each file git tracks that moves, and each file migrate
// writes new. Git skips a path it ignores when it marks them, so such a file
// would drop out of git, which could neither show nor undo the move. It is
// the mirror of dirty, which refuses a file git ignores at its old path; a
// hidden one there, which dirty lets move, is none git tracks, and stays
// ignored. A directory git would ignore as a whole, the destination among
// them, is named once, not each file in it. Nothing is force-added: a bundle
// in a directory git ignores would hide every file written there later.
func (p *plan) ignored() ([]string, error) {
	if p.project == "" {
		return nil, nil
	}
	// In a submodule's repository, or the bundle's own, the bundle is the
	// root, wherever it goes: its paths are its own, and --to moves none.
	repo, old, now, advice := p.project, p.old, p.new, "change the rule, or pass --to <dir> to choose another destination"
	if p.submodule || p.project == p.root {
		repo, old, now, advice = p.root, "", "", "change the rule"
	}
	spec := old
	if spec == "" {
		spec = "."
	}
	tracked, err := git(repo, "ls-files", "-z", "--", spec)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, t := range strings.Split(tracked, "\x00") {
		rel, ok := t, t != ""
		if ok && old != "" {
			rel, ok = strings.CutPrefix(t, old+"/")
		}
		if n := path.Join(now, p.to(rel)); ok && !p.goes(rel) && n != t {
			paths = append(paths, n)
		}
	}
	for rel := range p.texts {
		if p.source(rel) == "" {
			paths = append(paths, path.Join(now, rel))
		}
	}
	sort.Strings(paths)
	// Each directory the paths are in, from the bundle's root once migrated
	// down, is asked about too, as a directory: git reads a trailing slash
	// so.
	query, asked := append([]string{}, paths...), map[string]bool{}
	for _, n := range paths {
		for d := path.Dir(n); d != "." && !asked[d] && (now == "" || within(now, d)); d = path.Dir(d) {
			asked[d] = true
			query = append(query, d+"/")
		}
	}
	rules, err := checkIgnore(repo, query)
	if err != nil || len(rules) == 0 {
		return nil, err
	}
	var out []string
	named := map[string]bool{}
	for _, n := range paths {
		r, ok := rules[n]
		if !ok {
			continue
		}
		// The directory nearest the bundle's root that git ignores whole.
		top := ""
		for d := path.Dir(n); asked[d]; d = path.Dir(d) {
			if _, ok := rules[d+"/"]; ok {
				top = d
			}
		}
		switch {
		case top == "":
			out = append(out, fmt.Sprintf("%s: git would ignore it, by %s, so it could neither show nor undo what migrate puts there — %s", n, r, advice))
		case !named[top]:
			named[top] = true
			out = append(out, fmt.Sprintf("%s/: git would ignore it and every file in it, by %s, so it could neither show nor undo what migrate puts there — %s", top, rules[top+"/"], advice))
		}
	}
	sort.Strings(out)
	return out, nil
}

// rule is an ignore rule, as git check-ignore -v reports it.
type rule struct{ source, line, pattern string }

func (r rule) String() string {
	return fmt.Sprintf("the rule %s on line %s of %s", r.pattern, r.line, r.source)
}

// checkIgnore asks git, in the repository at dir, which of paths its rules
// ignore, as they read now, whether git tracks the path or not: by path, the
// rule that ignores it. A path ending in / is a directory. A path a negated
// rule matches is not ignored.
func checkIgnore(dir string, paths []string) (map[string]rule, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	cmd := exec.Command("git", "--no-optional-locks", "-C", dir, "check-ignore", "--no-index", "--stdin", "-z", "-v")
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\x00") + "\x00")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 1 {
		return nil, nil // git ignores none of them
	}
	if err != nil {
		return nil, fmt.Errorf("git check-ignore: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	rules := map[string]rule{}
	fields := strings.Split(string(out), "\x00")
	for i := 0; i+3 < len(fields); i += 4 {
		if r := (rule{fields[i], fields[i+1], fields[i+2]}); !strings.HasPrefix(r.pattern, "!") {
			rules[fields[i+3]] = r
		}
	}
	return rules, nil
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
		if len(p.outTexts) > 0 {
			lines = append(lines, fmt.Sprintf("git -C %s checkout -- %s", q(project), quoteAll(sortedKeys(p.outTexts))))
		}
		return append(lines, fmt.Sprintf("git -C %s checkout -- . && git -C %s clean -fd", old, old))
	}
	if p.relocated {
		lines = append(lines, fmt.Sprintf("mv %s %s", q(p.dest()), q(p.root)))
	}
	return append(lines, fmt.Sprintf("git -C %s checkout -- %s", q(project), quoteAll(append([]string{p.old}, sortedKeys(p.outTexts)...))),
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

// quoteAll quotes each path, and joins them with spaces.
func quoteAll(paths []string) string {
	words := make([]string, len(paths))
	for i, p := range paths {
		words[i] = quote(p)
	}
	return strings.Join(words, " ")
}
