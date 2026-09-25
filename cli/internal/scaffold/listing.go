package scaffold

// Index listings. The spec asks each register's and each group's INDEX.md to
// list the documents and groups beside it. Every command that files a
// document lists it there, lists each new group in its parent's index, and
// takes a cleared entry's listing away with its file.

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/links"
)

var (
	listingLinkRe = regexp.MustCompile(`^\s*[-*+]\s+.*?\]\(([^)\s]+)\)`)
	listItemRe    = regexp.MustCompile(`^\s*[-*+]\s`)
)

// ListEntry adds `* [<title>](/<dir>/<id>.md) - <what>.` to the INDEX.md of
// the directory the document sits in: <dir>/INDEX.md for an id with no group,
// or the index of its innermost group, at any depth. A group's index is
// created on first use and listed in its parent's, so every group can be
// found from its register's index. A document already listed is left alone.
func ListEntry(root, dir, id, title, what string, out io.Writer) int {
	parent := dir
	parts := strings.Split(id, "/")
	for _, group := range parts[:len(parts)-1] {
		idxRel := parent + "/" + group + "/INDEX.md"
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(idxRel))); err != nil {
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(idxRel)), []byte("# "+GroupTitle(parent, group)+"\n"), 0o644); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
			fmt.Fprintf(out, "wrote %s\n", idxRel)
			if code := ListGroup(root, parent, group, out); code != 0 {
				return code
			}
		}
		parent += "/" + group
	}
	target := dir + "/" + id + ".md"
	return addListing(root, parent+"/INDEX.md", target, title, fmt.Sprintf("* [%s](/%s) - %s.\n", title, target, what), out)
}

// groupNouns say what a group in each register holds, for the listing its
// parent's INDEX.md gives it: a changes/ group holds both kinds of
// post-delivery work.
var groupNouns = map[string]string{"features": "features", "changes": "changes and fixes", "practices": "practices", "debts": "debts", "bugs": "bugs"}

// GroupTitle is how a new group's own INDEX.md is headed: by its name.
func GroupTitle(dir, group string) string {
	title, _ := groupListing(dir, group)
	return title
}

// groupListing is a group's title and the line its parent's INDEX.md lists
// it with. dir is the parent: a register, or a group in one, at any depth.
func groupListing(dir, group string) (title, line string) {
	reg, _, _ := strings.Cut(dir, "/")
	title = strings.ToUpper(group[:1]) + strings.ReplaceAll(group[1:], "-", " ")
	return title, fmt.Sprintf("* [%s](/%s/%s/INDEX.md) - %s in %s.", title, dir, group, groupNouns[reg], group)
}

// registerListings are the lines the root INDEX.md lists each register with,
// in the order it lists them.
var registerListings = []struct{ reg, title, line string }{
	{"features", "Features", "* [Features](/features/INDEX.md) - what the software does."},
	{"changes", "Changes", "* [Changes](/changes/INDEX.md) - work on delivered features."},
	{"practices", "Practices", "* [Practices](/practices/INDEX.md) - how recurring mechanisms are done."},
	{"debts", "Debts", "* [Debts](/debts/INDEX.md) - known gaps between the documents and the code."},
	{"bugs", "Bugs", "* [Bugs](/bugs/INDEX.md) - known defects not repaired yet."},
	{"releases", "Releases", "* [Releases](/releases/INDEX.md) - what shipped in each version."},
}

// RegisterLine is the line the root INDEX.md lists the register reg with, and
// the register's title.
func RegisterLine(reg string) (line, title string) {
	for _, r := range registerListings {
		if r.reg == reg {
			return r.line, r.title
		}
	}
	return "", ""
}

// ListRegister lists a register in the root INDEX.md, after the registers it
// lists already, unless it lists this one, and says so the way ListEntry
// does. A root index that lists no register gets the line at its end.
func ListRegister(root, reg string, out io.Writer) int {
	p := filepath.Join(root, "INDEX.md")
	raw, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	text, added := WithRegisterListing(string(raw), reg)
	if !added {
		return 0
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	_, title := RegisterLine(reg)
	fmt.Fprintf(out, "updated INDEX.md (now lists %q)\n", title)
	return 0
}

// WithRegisterListing returns the root INDEX.md's text with the register
// reg's listing added after the registers it lists already, or at its end
// when it lists none, and whether it added it: a text that already lists the
// register — its INDEX.md, or the directory — is returned as it is.
func WithRegisterListing(text, reg string) (string, bool) {
	line, _ := RegisterLine(reg)
	if line == "" {
		return text, false
	}
	lines := strings.Split(text, "\n")
	after := -1 // the line the listing goes after
	for i, l := range lines {
		t := ListingTarget(l, ".")
		if t == reg+"/INDEX.md" || t == reg {
			return text, false
		}
		if r, ok := strings.CutSuffix(t, "/INDEX.md"); ok && layout.IsRegister(r) {
			after = i
		}
	}
	if after < 0 {
		return strings.TrimRight(text, "\n") + "\n\n" + line + "\n", true
	}
	return strings.Join(append(lines[:after+1], append([]string{line}, lines[after+1:]...)...), "\n"), true
}

// ListGroup lists a group in its parent's INDEX.md, dir, unless it is listed
// there already, and says so the way ListEntry does. A parent with no index
// is left without one.
func ListGroup(root, dir, group string, out io.Writer) int {
	idxRel := dir + "/INDEX.md"
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

// WithGroupListing returns the text of a group's parent index, dir's, with
// the group's listing added, and whether it added it: an index that already
// lists the group — its INDEX.md, or the directory — is returned as it is.
// The line goes after the last group the index lists, so the groups stay
// together; failing that, at the end.
func WithGroupListing(text, dir, group string) (string, bool) {
	groupRel := dir + "/" + group
	_, line := groupListing(dir, group)
	lines := strings.Split(text, "\n")
	after := -1 // the line the listing goes after
	for i, l := range lines {
		t := ListingTarget(l, dir)
		if t == groupRel+"/INDEX.md" || t == groupRel {
			return text, false
		}
		if strings.HasSuffix(t, "/INDEX.md") && path.Dir(path.Dir(t)) == dir {
			after = i
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
// when the line is not a listing: a list item whose first link outside code
// names a path. The link is read as links.Find reads it, so a title after
// the target or a <…> destination is not part of the path. A link starting
// with / is from the bundle root; any other is from the index's own
// directory.
func ListingTarget(line, idxDir string) string {
	if !listItemRe.MatchString(line) {
		return ""
	}
	for _, l := range links.Find(line) {
		if l.InCode {
			continue
		}
		if d, ok := links.Resolve(l.Target, path.Join(idxDir, "INDEX.md"), ""); ok {
			return d.Path
		}
		return ""
	}
	return ""
}
