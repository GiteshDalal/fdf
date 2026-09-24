package scaffold

// Index listings for the reserved directories. The spec asks each INDEX.md to
// list the documents beside it. `fdf new` and `fdf change` have always added
// theirs; these do the same for a practice, a debt and a bug, and take a
// cleared entry's listing away with its file.

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var listingLinkRe = regexp.MustCompile(`^\s*[-*+]\s+.*?\]\(([^)\s]+)\)`)

// ListEntry adds `* [<title>](/<dir>/<id>.md) - <what>.` to the INDEX.md of
// the directory the document sits in: <dir>/INDEX.md, or
// <dir>/<group>/INDEX.md for a grouped id. A group's index is created on first
// use and listed in <dir>/INDEX.md, so the group can be found. A document
// already listed is left alone.
func ListEntry(root, dir, id, title, what string, out io.Writer) int {
	idxRel := dir + "/INDEX.md"
	if group, _, grouped := strings.Cut(id, "/"); grouped {
		idxRel = dir + "/" + group + "/INDEX.md"
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(idxRel))); err != nil {
			heading := strings.ToUpper(group[:1]) + strings.ReplaceAll(group[1:], "-", " ")
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(idxRel)), []byte("# "+heading+"\n"), 0o644); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
			fmt.Fprintf(out, "wrote %s\n", idxRel)
			line := fmt.Sprintf("* [%s](/%s) - %ss in %s.\n", heading, idxRel, what, group)
			if code := addListing(root, dir+"/INDEX.md", idxRel, heading, line, out); code != 0 {
				return code
			}
		}
	}
	target := dir + "/" + id + ".md"
	return addListing(root, idxRel, target, title, fmt.Sprintf("* [%s](/%s) - %s.\n", title, target, what), out)
}

// addListing appends line to the index at idxRel unless the index already
// lists target (a bundle-relative path), and says so the way `fdf new` does:
// "updated <index> (now lists <title>)".
func addListing(root, idxRel, target, title, line string, out io.Writer) int {
	p := filepath.Join(root, filepath.FromSlash(idxRel))
	raw, err := os.ReadFile(p)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	text := string(raw)
	for _, l := range strings.Split(text, "\n") {
		if listingTarget(l, path.Dir(idxRel)) == target {
			return 0
		}
	}
	text = strings.TrimRight(text, "\n") + "\n"
	if lines := strings.Split(strings.TrimRight(text, "\n"), "\n"); !listingLinkRe.MatchString(lines[len(lines)-1]) {
		text += "\n" // a list starts after a blank line
	}
	if err := os.WriteFile(p, []byte(text+line), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "updated %s (now lists %q)\n", idxRel, title)
	return 0
}

// ListedIn returns the path of the INDEX.md beside the document at rel when
// it lists the document — the index Unlist would change — or "".
func ListedIn(root, rel string) string {
	idxRel := path.Dir(rel) + "/INDEX.md"
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(idxRel)))
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(raw), "\n") {
		if listingTarget(l, path.Dir(idxRel)) == rel {
			return idxRel
		}
	}
	return ""
}

// Unlist removes every listing of the document at rel (bundle-relative, such
// as "debts/venues/slow-hours.md") from the INDEX.md beside it, and returns
// that index's path when it changed.
func Unlist(root, rel string) (string, error) {
	idxRel := path.Dir(rel) + "/INDEX.md"
	p := filepath.Join(root, filepath.FromSlash(idxRel))
	raw, err := os.ReadFile(p)
	if err != nil {
		return "", nil
	}
	var kept []string
	lines := strings.Split(string(raw), "\n")
	for _, l := range lines {
		if listingTarget(l, path.Dir(idxRel)) != rel {
			kept = append(kept, l)
		}
	}
	if len(kept) == len(lines) {
		return "", nil
	}
	return idxRel, os.WriteFile(p, []byte(strings.Join(kept, "\n")), 0o644)
}

// listingTarget is the bundle-relative path a listing line links to, or ""
// when the line is not a listing. A link starting with / is from the bundle
// root; any other is from the index's own directory.
func listingTarget(line, idxDir string) string {
	m := listingLinkRe.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	t := m[1]
	if i := strings.IndexAny(t, "#?"); i >= 0 {
		t = t[:i]
	}
	if strings.HasPrefix(t, "/") {
		return path.Clean(strings.TrimPrefix(t, "/"))
	}
	return path.Clean(path.Join(idxDir, t))
}
