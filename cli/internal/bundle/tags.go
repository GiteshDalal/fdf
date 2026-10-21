package bundle

// The tag dictionary (spec 1.2). TAGS.md defines every tag the bundle's
// documents carry, once, with what it gathers: `test`, `testing` and `tests`
// are three tags to a search, and a tag invented for one document finds
// nothing but that document. F15 holds a bundle that has a TAGS.md to it; a
// bundle without one keeps free-form tags, as before 1.2, with a warning.
// TAGS.md turns F15 on by being there, not by the pin, because tags are older
// than the dictionary: moving a 1.1 bundle's pin must break nothing.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/links"
)

// tagNameRe is how a tag is written: lowercase letters and digits, in words
// joined by single hyphens.
var tagNameRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// TagName reports whether s is written as a tag is: lowercase letters and
// digits, in words joined by single hyphens (`gtm`, `oauth2`, `internal-api`).
func TagName(s string) bool { return tagNameRe.MatchString(s) }

// SuggestTag is the tag s would be if written as one: lowercased, with each
// run of anything but a letter or digit made one hyphen, or "" when nothing
// is left.
func SuggestTag(s string) string {
	var b strings.Builder
	hyphen := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if hyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			hyphen = false
			b.WriteRune(r)
			continue
		}
		hyphen = true
	}
	return b.String()
}

// Tag is one `## <tag>` entry of TAGS.md's `# Tags`.
type Tag struct {
	Name string
	// Definition is the tag's prose, its lines joined: every line under the
	// heading that is not an `instead-of` bullet. "" when there is none.
	Definition string
	// InsteadOf are the spellings documents must not carry for this tag.
	InsteadOf []string
}

// Defined reports whether the tag has a definition that is not still a
// placeholder.
func (t *Tag) Defined() bool {
	d := strings.TrimSpace(t.Definition)
	return d != "" && !placeholderDefinition(d)
}

// todoDefRe is a definition that is still a placeholder: TODO alone, or
// followed by a dash or a colon, as fdf tag writes it. "TODO comments left in
// the code" is a definition.
var todoDefRe = regexp.MustCompile(`^TODO\s*(?:$|[—–:-])`)

// placeholderDefinition reports whether d is still a placeholder.
func placeholderDefinition(d string) bool { return todoDefRe.MatchString(d) }

// TagDictionary is TAGS.md's `# Tags`, parsed.
type TagDictionary struct {
	// Tags are the entries in the order TAGS.md lists them, a tag defined
	// twice included.
	Tags      []*Tag
	byName    map[string]*Tag
	insteadOf map[string]string // a spelling not to use -> the tag that lists it
}

// Lookup returns the tag TAGS.md defines by that name, or nil.
func (d *TagDictionary) Lookup(name string) *Tag {
	if d == nil {
		return nil
	}
	return d.byName[name]
}

// Replacement returns the tag whose `instead-of` lists name, or "".
func (d *TagDictionary) Replacement(name string) string {
	if d == nil {
		return ""
	}
	return d.insteadOf[name]
}

// tagInsteadOfRe is a tag's `- instead-of:` bullet.
var tagInsteadOfRe = regexp.MustCompile(`^\s*[-*]\s+instead-of\s*:\s*(.*?)\s*$`)

// TagsBody is TAGS.md's body with what a reader does not read as an entry
// blanked out, byte for byte, line breaks kept: comments, and code, whose
// headings and bullets are samples.
func TagsBody(body string) string {
	b := []byte(body)
	blank := func(start, end int) {
		for i := start; i < end && i < len(b); i++ {
			if b[i] != '\n' {
				b[i] = ' '
			}
		}
	}
	for _, s := range links.Blocks(body) {
		blank(s.Start, s.End)
	}
	for _, m := range htmlCommentRe.FindAllIndex(b, -1) {
		blank(m[0], m[1])
	}
	return string(b)
}

// ParseTags reads the `# Tags` section of TAGS.md's body. ok is false when
// there is no such section.
func ParseTags(body string) (d *TagDictionary, ok bool) {
	section, ok := sectionText(TagsBody(body), "Tags")
	if !ok {
		return &TagDictionary{byName: map[string]*Tag{}, insteadOf: map[string]string{}}, false
	}
	d = &TagDictionary{byName: map[string]*Tag{}, insteadOf: map[string]string{}}
	var cur *Tag
	var prose []string
	flush := func() {
		if cur != nil {
			cur.Definition = strings.Join(prose, " ")
		}
		prose = nil
	}
	for _, line := range strings.Split(section, "\n") {
		if m := subHeadingRe.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Tag{Name: strings.TrimSpace(m[1])}
			d.Tags = append(d.Tags, cur)
			if d.byName[cur.Name] == nil {
				d.byName[cur.Name] = cur
			}
			continue
		}
		if cur == nil {
			continue
		}
		if m := tagInsteadOfRe.FindStringSubmatch(line); m != nil {
			for _, w := range commaList(m[1]) {
				cur.InsteadOf = append(cur.InsteadOf, w)
				if d.insteadOf[w] == "" {
					d.insteadOf[w] = cur.Name
				}
			}
			continue
		}
		if t := strings.TrimSpace(line); t != "" {
			prose = append(prose, t)
		}
	}
	flush()
	return d, true
}

// LoadTags reads the bundle's TAGS.md. exists is false when the bundle has
// none, and d is nil when it has one with no frontmatter to read past or no
// `# Tags` section: what the validator reports (F1, F15).
func LoadTags(root string) (d *TagDictionary, exists bool) {
	// Its name spelled exactly: on a disk that ignores case, a tags.md opens
	// as TAGS.md, and F3 reports it as a stray.
	if !layout.New(os.DirFS(root)).Exists(layout.Tags) {
		return nil, false
	}
	raw, err := os.ReadFile(filepath.Join(root, layout.Tags))
	if err != nil {
		return nil, false
	}
	_, delimited, body := splitFrontmatter(strings.TrimPrefix(string(raw), "\uFEFF"))
	if !delimited {
		return nil, true
	}
	d, ok := ParseTags(body)
	if !ok {
		return nil, true
	}
	return d, true
}

// DocumentTags returns the tags a document's frontmatter carries: `tags`, as
// a string or a list.
func DocumentTags(data map[string]any) []string { return asList(data["tags"]) }

// tagCarriers is which documents carry each tag: tag -> the rels of the
// documents, sorted.
type tagCarriers map[string][]string

func (c tagCarriers) add(tag, rel string) {
	for _, r := range c[tag] {
		if r == rel {
			return
		}
	}
	c[tag] = append(c[tag], rel)
}

// sortedKeys returns the tags in c, sorted.
func (c tagCarriers) sortedKeys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// carriedBy names the documents that carry a tag, the first few of them,
// for a message.
func carriedBy(rels []string) string {
	const shown = 3
	sort.Strings(rels)
	n := len(rels)
	what := "1 document carries it"
	if n != 1 {
		what = fmt.Sprintf("%d documents carry it", n)
	}
	if n > shown {
		return fmt.Sprintf("%s (%s, …)", what, strings.Join(rels[:shown], ", "))
	}
	return fmt.Sprintf("%s (%s)", what, strings.Join(rels, ", "))
}

// checkTags is F15, and its soft checks, for a bundle pinned to 1.2 or later:
// carried maps each tag a document carries to the documents that carry it.
// With no TAGS.md, documents that carry tags draw one warning. With one,
// every tag a document carries is a tag it defines.
func checkTags(rootAbs string, carried tagCarriers, errs, warns *[]string) {
	d, exists := LoadTags(rootAbs)
	if !exists {
		if len(carried) > 0 {
			docs := map[string]bool{}
			for _, rels := range carried {
				for _, r := range rels {
					docs[r] = true
				}
			}
			*warns = append(*warns, fmt.Sprintf("%d document(s) carry %d tag(s), and the bundle has no TAGS.md to define them — `fdf tag --sync` starts one from them, and F15 then holds every tag to it", len(docs), len(carried)))
		}
		return
	}
	if d == nil {
		if raw, err := os.ReadFile(filepath.Join(rootAbs, layout.Tags)); err == nil {
			if _, delimited, _ := splitFrontmatter(strings.TrimPrefix(string(raw), "\uFEFF")); delimited {
				*errs = append(*errs, "TAGS.md: missing required `# Tags` section — one `## <tag>` heading per tag, its definition under it (F15)")
			}
		}
		return // no frontmatter: F1 has reported it
	}

	// (a)-(b): the entries themselves.
	seen := map[string]bool{}
	for _, t := range d.Tags {
		switch {
		case !TagName(t.Name):
			hint := ""
			if s := SuggestTag(t.Name); s != "" {
				hint = fmt.Sprintf(", such as `%s`", s)
			}
			*errs = append(*errs, fmt.Sprintf("TAGS.md: `## %s` is not a tag — a tag is lowercase letters and digits, in words joined by single hyphens%s (F15)", t.Name, hint))
			continue
		case seen[t.Name]:
			*errs = append(*errs, fmt.Sprintf("TAGS.md: tag `%s` is defined twice — keep one entry (F15)", t.Name))
			continue
		}
		seen[t.Name] = true
		switch def := strings.TrimSpace(t.Definition); {
		case def == "":
			*errs = append(*errs, fmt.Sprintf("TAGS.md: tag `%s` has no definition — write a line on what it gathers, and what it does not when a neighboring tag is close (F15)", t.Name))
		case placeholderDefinition(def):
			*errs = append(*errs, fmt.Sprintf("TAGS.md: tag `%s` still has a TODO definition — replace it with a line on what the tag gathers, and what it does not when a neighboring tag is close (F15)", t.Name))
		}
	}
	// (c): instead-of.
	claimed := map[string]string{}
	for _, t := range d.Tags {
		for _, w := range t.InsteadOf {
			switch {
			case !TagName(w):
				*errs = append(*errs, fmt.Sprintf("TAGS.md: `%s` under `instead-of` of `%s` is not written as a tag is — lowercase letters and digits, in words joined by single hyphens (F15)", w, t.Name))
			case d.Lookup(w) != nil:
				*errs = append(*errs, fmt.Sprintf("TAGS.md: `%s` is listed under `instead-of` of `%s` but is a tag itself — merge it with `fdf tag --merge %s %s`, or take it out of `instead-of` (F15)", w, t.Name, w, t.Name))
			case claimed[w] != "" && claimed[w] != t.Name:
				*errs = append(*errs, fmt.Sprintf("TAGS.md: `%s` is listed under `instead-of` of both `%s` and `%s` — one spelling, one tag (F15)", w, claimed[w], t.Name))
			default:
				claimed[w] = t.Name
			}
		}
	}

	// (d): every tag a document carries.
	for _, tag := range carried.sortedKeys() {
		rels := carried[tag]
		switch {
		case d.Lookup(tag) != nil:
		case !TagName(tag):
			fix := "write it as a tag is: lowercase letters and digits, in words joined by single hyphens"
			if strings.Contains(tag, ",") {
				fix = "a list of tags is written `tags: [a, b]`"
			} else if s := SuggestTag(tag); s != "" {
				fix = fmt.Sprintf("`fdf tag --merge '%s' %s` renames it in every document", tag, s)
			}
			*errs = append(*errs, fmt.Sprintf("tag %q is not a tag name, and %s — %s (F15)", tag, carriedBy(rels), fix))
		case d.Replacement(tag) != "":
			into := d.Replacement(tag)
			*errs = append(*errs, fmt.Sprintf("tag `%s` is listed under `instead-of` of `%s` in TAGS.md, and %s — carry `%s` instead: `fdf tag --merge %s %s` (F15)", tag, into, carriedBy(rels), into, tag, into))
		default:
			*errs = append(*errs, fmt.Sprintf("tag `%s` is not defined in TAGS.md, and %s — carry a tag it defines (`fdf tag` lists them), or add it with `fdf tag %s` and write its definition (F15)", tag, carriedBy(rels), tag))
		}
	}

	// Soft: a defined tag no document carries, and two that look like one.
	var names []string
	for _, t := range d.Tags {
		if TagName(t.Name) && d.Lookup(t.Name) == t {
			names = append(names, t.Name)
		}
	}
	for _, n := range names {
		if len(carried[n]) == 0 {
			*warns = append(*warns, fmt.Sprintf("TAGS.md: no document carries tag `%s` — remove it, or tag the documents it gathers", n))
		}
	}
	count := map[string]int{}
	for t, rels := range carried {
		count[t] = len(rels)
	}
	for i, a := range names {
		for _, b := range names[i+1:] {
			keep, drop, alike := LookAlike(a, b, count)
			ta, tb := d.Lookup(a), d.Lookup(b)
			switch {
			case !alike:
				continue
			case ta.Defined() && tb.Defined():
				continue // both defined: their definitions say how they differ
			case ta.Defined() != tb.Defined():
				// Keep the one whose meaning is written down.
				keep, drop = a, b
				if tb.Defined() {
					keep, drop = b, a
				}
			}
			*warns = append(*warns, fmt.Sprintf("TAGS.md: tags `%s` and `%s` look like one tag — merge them with `fdf tag --merge %s %s`, or define both, saying in each how it differs from the other", a, b, drop, keep))
		}
	}
}

// LookAlike reports whether two tags differ only by hyphens or by a plural
// ending (`-s`, `-es`), and which to keep: the one more documents carry, by
// count, or the shorter.
func LookAlike(a, b string, count map[string]int) (keep, drop string, alike bool) {
	x, y := strings.ReplaceAll(a, "-", ""), strings.ReplaceAll(b, "-", "")
	if !(x == y || y == x+"s" || y == x+"es" || x == y+"s" || x == y+"es") {
		return "", "", false
	}
	keep, drop = a, b
	switch na, nb := count[a], count[b]; {
	case nb > na:
		keep, drop = b, a
	case na == nb && len(b) < len(a):
		keep, drop = b, a
	}
	return keep, drop, true
}
