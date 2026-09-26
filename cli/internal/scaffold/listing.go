package scaffold

// Index listings. The spec asks each INDEX.md to list the documents beside
// it, and the root INDEX.md to list the groups. `fdf new` and `fdf change`
// have always added theirs; these do the same for a practice, a debt and a
// bug, list a new group in its parent's index — a feature group in the root
// INDEX.md, a reserved directory's group in that directory's — and take a
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
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(idxRel)), []byte("# "+GroupTitle(dir, group)+"\n"), 0o644); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
			fmt.Fprintf(out, "wrote %s\n", idxRel)
			if code := ListGroup(root, dir, group, out); code != 0 {
				return code
			}
		}
	}
	target := dir + "/" + id + ".md"
	return addListing(root, idxRel, target, title, fmt.Sprintf("* [%s](/%s) - %s.\n", title, target, what), out)
}

// groupNouns say what a group of each reserved directory holds, for the
// listing the directory's INDEX.md gives it: a changes/ group holds both
// kinds of post-delivery work.
var groupNouns = map[string]string{"changes": "changes and fixes", "practices": "practices", "debts": "debts", "bugs": "bugs"}

// GroupTitle is how a new group's own INDEX.md is headed: a feature group
// (dir "") as "<Group> features", a reserved directory's group by its name.
func GroupTitle(dir, group string) string {
	title, _ := groupListing(dir, group)
	if dir == "" {
		return title + " features"
	}
	return title
}

// groupListing is a group's title and the line its parent index lists it
// with: a feature group (dir "") in the bundle-root INDEX.md, a reserved
// directory's group in that directory's.
func groupListing(dir, group string) (title, line string) {
	if dir == "" {
		title = strings.ToUpper(group[:1]) + group[1:]
		return title, fmt.Sprintf("* [%s](/%s/INDEX.md) - %s features.", title, group, group)
	}
	title = strings.ToUpper(group[:1]) + strings.ReplaceAll(group[1:], "-", " ")
	return title, fmt.Sprintf("* [%s](/%s/%s/INDEX.md) - %s in %s.", title, dir, group, groupNouns[dir], group)
}

// ListGroup lists a group in its parent index — a feature group (dir "") in
// the bundle-root INDEX.md, which the spec says lists the groups; a reserved
// directory's group in that directory's INDEX.md — unless it is listed there
// already, and says so the way ListEntry does. A parent with no index is left
// without one.
func ListGroup(root, dir, group string, out io.Writer) int {
	idxRel := "INDEX.md"
	if dir != "" {
		idxRel = dir + "/INDEX.md"
	}
	p := filepath.Join(root, filepath.FromSlash(idxRel))
	raw, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	text, added := WithGroupListing(string(raw), dir, group)
	if !added {
		return 0
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	title, _ := groupListing(dir, group)
	fmt.Fprintf(out, "updated %s (now lists %q)\n", idxRel, title)
	return 0
}

var overviewRe = regexp.MustCompile(`(?i)^#\s+overview\s*$`)
var headingLineRe = regexp.MustCompile(`^#{1,6}\s`)

// WithGroupListing returns the text of a group's parent index (see ListGroup)
// with the group's listing added, and whether it added it: an index that
// already lists the group — its INDEX.md, or the directory — is returned as it
// is. The line goes after the last group the index lists, so the groups stay
// together; failing that, at the end of the `# Overview` list `fdf init`
// writes in the root index; failing that, at the end.
func WithGroupListing(text, dir, group string) (string, bool) {
	idxDir, groupRel := ".", group
	if dir != "" {
		idxDir, groupRel = dir, dir+"/"+group
	}
	_, line := groupListing(dir, group)
	lines := strings.Split(text, "\n")
	after := -1 // the line the listing goes after
	for i, l := range lines {
		t := ListingTarget(l, idxDir)
		if t == groupRel+"/INDEX.md" || t == groupRel {
			return text, false
		}
		if strings.HasSuffix(t, "/INDEX.md") && path.Dir(path.Dir(t)) == path.Clean(idxDir) {
			after = i
		}
	}
	if after < 0 {
		// The end of the `# Overview` section: its last listing, or else its
		// last line of text, or else the heading itself.
		for i := 0; i < len(lines); i++ {
			if !overviewRe.MatchString(strings.TrimSpace(lines[i])) {
				continue
			}
			after = i
			listed := false
			for j := i + 1; j < len(lines) && !headingLineRe.MatchString(lines[j]); j++ {
				if listingLinkRe.MatchString(lines[j]) {
					after, listed = j, true
				} else if strings.TrimSpace(lines[j]) != "" && !listed {
					after = j
				}
			}
			break
		}
	}
	if after < 0 {
		// Nowhere better: at the end, as addListing adds a document.
		text = strings.TrimRight(text, "\n")
		sep := "\n"
		if !listingLinkRe.MatchString(text[strings.LastIndex(text, "\n")+1:]) {
			sep = "\n\n" // a list starts after a blank line
		}
		return text + sep + line + "\n", true
	}
	add := []string{line}
	if !listingLinkRe.MatchString(lines[after]) {
		// After a heading or prose: a new list is set off by blank lines.
		add = []string{"", line}
		if after+1 < len(lines) && strings.TrimSpace(lines[after+1]) != "" {
			add = append(add, "")
		}
	}
	lines = append(lines[:after+1], append(add, lines[after+1:]...)...)
	return strings.Join(lines, "\n"), true
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
		if ListingTarget(l, path.Dir(idxRel)) == target {
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
		if ListingTarget(l, path.Dir(idxRel)) == rel {
			return idxRel
		}
	}
	return ""
}

// Unlist removes every listing of the document at rel (bundle-relative, such
// as "debts/venues/slow-hours.md") from the INDEX.md beside it, and returns
// that index's path when it changed.
func Unlist(root, rel string) (string, error) {
	return unlistFrom(root, path.Dir(rel)+"/INDEX.md", rel)
}

// UnlistGroup removes the listing of a group (bundle-relative, such as
// "debts/venues") from its parent's INDEX.md — a link to the group's index
// or to the directory — and returns that index's path when it changed.
func UnlistGroup(root, group string) (string, error) {
	return unlistFrom(root, path.Join(path.Dir(group), "INDEX.md"), group+"/INDEX.md", group)
}

// GroupListedIn returns the path of the parent index that lists a group (see
// UnlistGroup), or "".
func GroupListedIn(root, group string) string {
	idxRel := path.Join(path.Dir(group), "INDEX.md")
	for _, t := range Listed(root, idxRel) {
		if t == group+"/INDEX.md" || t == group {
			return idxRel
		}
	}
	return ""
}

// Listed returns the bundle-relative targets the index at idxRel lists.
func Listed(root, idxRel string) []string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(idxRel)))
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(string(raw), "\n") {
		if t := ListingTarget(l, path.Dir(idxRel)); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// unlistFrom removes every listing of targets from the index at idxRel, and
// returns idxRel when it changed.
func unlistFrom(root, idxRel string, targets ...string) (string, error) {
	p := filepath.Join(root, filepath.FromSlash(idxRel))
	raw, err := os.ReadFile(p)
	if err != nil {
		return "", nil
	}
	var kept []string
	lines := strings.Split(string(raw), "\n")
	for _, l := range lines {
		t := ListingTarget(l, path.Dir(idxRel))
		listed := false
		for _, target := range targets {
			listed = listed || t == target
		}
		if !listed {
			kept = append(kept, l)
		}
	}
	if len(kept) == len(lines) {
		return "", nil
	}
	return idxRel, os.WriteFile(p, []byte(strings.Join(kept, "\n")), 0o644)
}

// ListingTarget is the bundle-relative path a listing line links to, or ""
// when the line is not a listing. A link starting with / is from the bundle
// root; any other is from the index's own directory.
func ListingTarget(line, idxDir string) string {
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
