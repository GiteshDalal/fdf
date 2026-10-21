// Package tags implements `fdf tag`, and the --tags that the commands which
// create a document take: the bundle's tag dictionary, TAGS.md (spec 1.2).
// A tag files a document with others that share a concern its register and
// group do not capture; it finds them only if everyone spells it one way and
// means one thing by it. So the commands here keep the dictionary and the
// documents in step: they list what is defined and how much it is used, add
// a tag with a TODO definition for a person to write, start a dictionary from
// the tags a bundle already carries, and merge two spellings of one tag
// across every document, as a tag fix.
package tags

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/logs"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// todoDefinition is the definition fdf writes for a person to replace, which
// F15 reports until it is.
const todoDefinition = "TODO — what this tag gathers, and what it does not when a neighboring tag is close."

// now is the UTC time every fdf command writes.
func now() string { return time.Now().UTC().Format("2006-01-02T15:04:05Z") }

// dictionaryTaken reports whether the bundle's pin takes TAGS.md: spec 1.2
// added it.
func dictionaryTaken(root string) bool { return specver.Since(scaffold.Pin(root), layout.TagsSince) }

// writable says whether fdf may write the bundle's TAGS.md, and why not when
// it may not: the commands work on the bundle's pin, which must be 1.2 or
// later.
func writable(root string, out io.Writer) bool {
	if !scaffold.RequireSupported(root, out) {
		return false
	}
	if !dictionaryTaken(root) {
		fmt.Fprintf(out, "error: TAGS.md is a file of the bundle from spec %s, and this bundle pins %s — `fdf migrate` moves the pin first, the user's decision\n", layout.TagsSince, scaffold.Pin(root))
		return false
	}
	return true
}

// carriers maps each tag the bundle's documents carry to the documents that
// carry it, by path from the bundle root, sorted.
type carriers map[string][]string

// sorted returns the tags, sorted.
func (c carriers) sorted() []string {
	out := make([]string, 0, len(c))
	for t := range c {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// documents counts the documents that carry any tag.
func (c carriers) documents() int {
	docs := map[string]bool{}
	for _, rels := range c {
		for _, r := range rels {
			docs[r] = true
		}
	}
	return len(docs)
}

// scan reads the tags every document of the bundle carries: every Markdown
// file layout places as a document, and no index, log or README, and nothing
// with no place in the bundle.
func scan(root string) carriers {
	c := carriers{}
	b := layout.New(os.DirFS(root))
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case d.IsDir() && p != root && strings.HasPrefix(d.Name(), "."):
			return filepath.SkipDir
		case d.IsDir() || !strings.HasSuffix(d.Name(), ".md") || strings.HasPrefix(d.Name(), "."):
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		switch b.File(rel).Kind {
		case layout.Index, layout.Log, layout.Readme, layout.Stray:
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		data, _, ok := bundle.Frontmatter(string(raw))
		if !ok {
			return nil
		}
		seen := map[string]bool{}
		for _, t := range bundle.DocumentTags(data) {
			if !seen[t] {
				seen[t] = true
				c[t] = append(c[t], rel)
			}
		}
		return nil
	})
	for _, rels := range c {
		sort.Strings(rels)
	}
	return c
}

// dictionary reads the bundle's TAGS.md: its text and what it defines. exists
// is false when the bundle has none; d is nil when TAGS.md cannot be read as
// a dictionary, which the validator reports.
func dictionary(root string) (text string, d *bundle.TagDictionary, exists bool) {
	// Its name spelled exactly: on a disk that ignores case, a tags.md would
	// open as TAGS.md, and F3 rejects it.
	if !layout.New(os.DirFS(root)).Exists(layout.Tags) {
		return "", nil, false
	}
	raw, err := os.ReadFile(filepath.Join(root, layout.Tags))
	if err != nil {
		return "", nil, false
	}
	d, _ = bundle.LoadTags(root)
	// Edited with LF line ends; write gives the file back the ones it had.
	return strings.ReplaceAll(string(raw), "\r\n", "\n"), d, true
}

// caseTwin names the file at the root a disk that ignores case would read
// as TAGS.md, such as tags.md, or "".
func caseTwin(root string) string {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.Name() != layout.Tags && strings.EqualFold(e.Name(), layout.Tags) {
			return e.Name()
		}
	}
	return ""
}

// List prints the dictionary: each tag TAGS.md defines, with how many
// documents carry it and its definition, then each tag documents carry that
// it does not define. A bundle with no TAGS.md gets the tags its documents
// carry, and how to start one from them.
func List(root string, out io.Writer) int {
	if !scaffold.RequireSupported(root, out) {
		return 1
	}
	c := scan(root)
	_, d, exists := dictionary(root)
	if !exists || d == nil {
		if len(c) == 0 {
			fmt.Fprintln(out, "no document carries a tag yet.")
		} else {
			w := width(c.sorted(), "TAG")
			fmt.Fprintf(out, "%-*s  %4s\n", w, "TAG", "DOCS")
			for _, t := range c.sorted() {
				fmt.Fprintf(out, "%-*s  %4d\n", w, t, len(c[t]))
			}
			fmt.Fprintf(out, "\n%d tag(s) on %d document(s).\n", len(c), c.documents())
		}
		switch {
		case exists:
			fmt.Fprintln(out, "TAGS.md has no `# Tags` section to read — `fdf validate` says what to repair.")
		case !dictionaryTaken(root):
			fmt.Fprintf(out, "the tag dictionary, TAGS.md, is a file of the bundle from spec %s; this bundle pins %s.\n", layout.TagsSince, scaffold.Pin(root))
		case len(c) > 0:
			fmt.Fprintln(out, "no TAGS.md yet: `fdf tag --sync` starts one from these, each with a TODO definition to write.")
		default:
			fmt.Fprintln(out, "no TAGS.md yet: `fdf tag <tag>` starts one with that tag.")
		}
		return 0
	}

	var names []string
	for _, t := range d.Tags {
		if d.Lookup(t.Name) == t {
			names = append(names, t.Name)
		}
	}
	todo := 0
	if len(names) == 0 {
		fmt.Fprintln(out, "TAGS.md defines no tag yet.")
	} else {
		w := width(names, "TAG")
		fmt.Fprintf(out, "%-*s  %4s  %s\n", w, "TAG", "DOCS", "DEFINITION")
		for _, n := range names {
			t := d.Lookup(n)
			def := clip(strings.TrimSpace(t.Definition), 72)
			switch {
			case def == "":
				def, todo = "(none)", todo+1
			case !t.Defined():
				def, todo = "TODO", todo+1
			}
			fmt.Fprintf(out, "%-*s  %4d  %s\n", w, n, len(c[n]), def)
		}
	}

	var undefined []string
	for _, t := range c.sorted() {
		if d.Lookup(t) == nil {
			undefined = append(undefined, t)
		}
	}
	if len(undefined) > 0 {
		fmt.Fprintln(out, "\ncarried, but not defined in TAGS.md:")
		for _, t := range undefined {
			fmt.Fprintf(out, "  %s — %s\n", t, repair(d, t, len(c[t])))
		}
	}
	fmt.Fprintf(out, "\n%d tag(s) defined", len(names))
	if todo > 0 {
		fmt.Fprintf(out, ", %d still to define", todo)
	}
	fmt.Fprintf(out, "; %d document(s) carry tags.\n", c.documents())
	if todo > 0 || len(undefined) > 0 {
		fmt.Fprintln(out, "`fdf validate` reports each of these until it is repaired (F15).")
	}
	return 0
}

// repair says what to do about a tag documents carry that the dictionary
// does not define.
func repair(d *bundle.TagDictionary, tag string, n int) string {
	docs := fmt.Sprintf("%d document(s)", n)
	switch {
	case !bundle.TagName(tag):
		if s := bundle.SuggestTag(tag); s != "" {
			return fmt.Sprintf("%s; not a tag name: `fdf tag --merge '%s' %s`", docs, tag, s)
		}
		return docs + "; not a tag name: rename it by hand"
	case d.Replacement(tag) != "":
		into := d.Replacement(tag)
		return fmt.Sprintf("%s; listed under `instead-of` of %s: `fdf tag --merge %s %s`", docs, into, tag, into)
	}
	return fmt.Sprintf("%s; `fdf tag %s` adds it, or `fdf tag --merge %s <tag>` folds it into a defined tag", docs, tag, tag)
}

// Show lists the documents that carry tag, by ID, under its definition.
func Show(root, tag string, out io.Writer) int {
	if !scaffold.RequireSupported(root, out) {
		return 1
	}
	c := scan(root)
	_, d, _ := dictionary(root)
	switch t := d.Lookup(tag); {
	case t != nil && t.Defined():
		fmt.Fprintf(out, "%s: %s\n\n", tag, strings.TrimSpace(t.Definition))
	case t != nil:
		fmt.Fprintf(out, "%s: its definition is still to write (TAGS.md)\n\n", tag)
	case d.Replacement(tag) != "":
		fmt.Fprintf(out, "%s is listed under `instead-of` of %s in TAGS.md: documents carry %s instead.\n\n", tag, d.Replacement(tag), d.Replacement(tag))
	}
	rels := c[tag]
	if len(rels) == 0 {
		fmt.Fprintf(out, "no document carries %s.\n", tag)
		return 0
	}
	for _, r := range rels {
		fmt.Fprintln(out, strings.TrimSuffix(r, ".md"))
	}
	fmt.Fprintf(out, "\n%d document(s) carry %s.\n", len(rels), tag)
	return 0
}

// refused says why a tag a person names cannot be given to a document, or
// "": it is not written as a tag is, or the dictionary lists it as a
// spelling of another tag.
func refused(d *bundle.TagDictionary, tag string) string {
	switch {
	case !bundle.TagName(tag):
		hint := ""
		if s := bundle.SuggestTag(tag); s != "" && s != tag {
			hint = fmt.Sprintf(", such as %s", s)
		}
		return fmt.Sprintf("%q is not a tag: a tag is lowercase letters and digits, in words joined by single hyphens%s", tag, hint)
	case d.Replacement(tag) != "":
		return fmt.Sprintf("%s is listed under `instead-of` of %s in TAGS.md — use %s", tag, d.Replacement(tag), d.Replacement(tag))
	}
	return ""
}

// Add adds each tag to TAGS.md that it does not define yet, with definition,
// or a TODO definition for a person to write when it is "". A bundle with no
// TAGS.md gets one.
func Add(root string, names []string, definition string, out io.Writer) int {
	if !writable(root, out) {
		return 1
	}
	text, d, exists := dictionary(root)
	if exists && d == nil {
		fmt.Fprintln(out, "error: TAGS.md cannot be read as a dictionary — `fdf validate` says what to repair first")
		return 1
	}
	// Starting the dictionary holds every tag the documents carry to it at
	// once (F15), so it starts only from all of them, by --sync — never as
	// a side effect of adding one.
	if c := scan(root); !exists && len(c) > 0 {
		fmt.Fprintf(out, "error: the bundle has no TAGS.md, and %d document(s) carry %d tag(s): starting the dictionary holds every one of them to it at once (F15) — `fdf tag --sync` starts it from them all, which is the user's decision; until then tags are free-form\n", c.documents(), len(c))
		return 1
	}
	for _, n := range names {
		if why := refused(d, n); why != "" {
			fmt.Fprintln(out, "error: "+why)
			return 1
		}
	}
	var add []string
	for _, n := range uniq(names) {
		t := d.Lookup(n)
		switch {
		case t != nil && definition != "":
			// One tag, defined anew: its TODO filled in, or its meaning
			// changed, which reaches every document that carries it.
			redefined := t.Defined()
			if code := write(root, withDefinition(text, n, definition), false, out); code != 0 {
				return code
			}
			if !redefined {
				fmt.Fprintf(out, "defined %s in TAGS.md\n", n)
				return 0
			}
			fmt.Fprintf(out, "redefined %s in TAGS.md\n", n)
			return logRoot(root, fmt.Sprintf("* **Tags**: redefined `%s`: %s", n, definition), out)
		case t != nil:
			def := strings.TrimSpace(t.Definition)
			if !t.Defined() {
				def = "its definition is still to write — `fdf tag --definition '<text>' " + n + "` writes it"
			}
			fmt.Fprintf(out, "%s is defined already: %s\n", n, def)
			continue
		}
		add = append(add, n)
	}
	if len(add) == 0 {
		return 0
	}
	noteResemblance(add, d, out)
	if definition == "" {
		definition = todoDefinition
	}
	if !exists {
		text = scaffold.TagsDoc(now())
	}
	text = withEntries(text, add, func(string) string { return definition })
	if code := write(root, text, !exists, out); code != 0 {
		return code
	}
	for _, n := range add {
		fmt.Fprintf(out, "added %s to TAGS.md\n", n)
	}
	if definition == todoDefinition {
		fmt.Fprintln(out, "next: replace each TODO definition in TAGS.md with what the tag gathers, and what it does not")
		fmt.Fprintln(out, "      when a neighboring tag is close — `fdf validate` reports it until then (F15).")
	}
	return 0
}

// write writes TAGS.md, its timestamp set to now: a tag added or merged
// changes the dictionary's substance. created says the bundle had none, and
// then the root INDEX.md lists it and LOG.md records it.
func write(root, text string, created bool, out io.Writer) int {
	return writeStamped(root, text, created, true, out)
}

// writeStamped is write, its timestamp set only when stamp says: a tag fix
// is a maintenance edit, which leaves it as it is. A text that would not
// change the file is not written.
func writeStamped(root, text string, created, stamp bool, out io.Writer) int {
	p := filepath.Join(root, layout.Tags)
	if twin := caseTwin(root); twin != "" {
		fmt.Fprintf(out, "error: %s is at the bundle root, and a disk that ignores case reads it as TAGS.md; F3 rejects it — rename it to TAGS.md first\n", twin)
		return 1
	}
	old, err := os.ReadFile(p)
	crlf := err == nil && strings.Contains(string(old), "\r\n")
	if err == nil && strings.ReplaceAll(string(old), "\r\n", "\n") == text {
		return 0
	}
	if stamp {
		text = withTimestamp(text, now())
	}
	if crlf {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if !created {
		fmt.Fprintln(out, "updated TAGS.md")
		return 0
	}
	fmt.Fprintln(out, "wrote TAGS.md (type: Tags) — the bundle's tag dictionary; F15 now holds every tag to it")
	if code := scaffold.ListTags(root, out); code != 0 {
		return code
	}
	return logRoot(root, "* **Tags**: started the tag dictionary, TAGS.md; every tag a document carries is defined there from now on (F15).", out)
}

// logRoot records an entry in the bundle-root LOG.md, under today's date.
func logRoot(root, line string, out io.Writer) int {
	p := filepath.Join(root, "LOG.md")
	raw, err := os.ReadFile(p)
	body := string(raw)
	if err != nil {
		body = "# Bundle Update Log\n"
	}
	if err := os.WriteFile(p, []byte(logs.Insert(body, line+"\n")), 0o644); err != nil {
		fmt.Fprintln(out, "error: writing LOG.md:", err)
		return 1
	}
	return 0
}

// Sync adds every tag the bundle's documents carry that TAGS.md does not
// define, each with a TODO definition, and starts TAGS.md when the bundle
// has none: how a bundle whose documents carried tags before 1.2 takes up the
// dictionary. A tag it cannot add — one not written as a tag is, or a
// spelling the dictionary lists under another tag — is named, with its
// repair. dryRun prints the plan and writes nothing.
func Sync(root string, dryRun bool, out io.Writer) int {
	if !writable(root, out) {
		return 1
	}
	if twin := caseTwin(root); twin != "" {
		fmt.Fprintf(out, "error: %s is at the bundle root, and a disk that ignores case reads it as TAGS.md; F3 rejects it — rename it to TAGS.md first\n", twin)
		return 1
	}
	c := scan(root)
	text, d, exists := dictionary(root)
	if exists && d == nil {
		fmt.Fprintln(out, "error: TAGS.md cannot be read as a dictionary — `fdf validate` says what to repair first")
		return 1
	}
	var add, held []string
	for _, t := range c.sorted() {
		switch {
		case d.Lookup(t) != nil:
		case refused(d, t) != "":
			held = append(held, t)
		default:
			add = append(add, t)
		}
	}
	verb := "added"
	if dryRun {
		verb = "would add"
	}
	switch {
	case len(add) == 0 && exists:
		fmt.Fprintln(out, "TAGS.md defines every tag written as one that the documents carry.")
	case len(add) == 0 && len(held) == 0:
		fmt.Fprintln(out, "no document carries a tag yet; nothing to start TAGS.md from (`fdf tag <tag>` starts it with one).")
		return 0
	case len(add) == 0:
		fmt.Fprintln(out, "no tag the documents carry can start TAGS.md as it is written — repair these first:")
	default:
		fmt.Fprintf(out, "%s %d tag(s) to TAGS.md, each with a TODO definition: %s\n", verb, len(add), strings.Join(add, ", "))
	}
	for _, t := range held {
		fmt.Fprintf(out, "  left out %s — %s\n", t, repair(d, t, len(c[t])))
	}
	count := map[string]int{}
	var all []string
	for _, t := range append(defined(d), add...) {
		if count[t] == 0 {
			all = append(all, t)
		}
		count[t] = len(c[t])
	}
	sort.Strings(all)
	for i, a := range all {
		for _, b := range all[i+1:] {
			if keep, drop, alike := bundle.LookAlike(a, b, count); alike {
				fmt.Fprintf(out, "  %s and %s look like one tag: `fdf tag --merge %s %s` if they are\n", a, b, drop, keep)
			} else if resembles(a, b) {
				// Weaker than a plural, and often two tags: said once, here,
				// with no command to run — whether they are one is the user's
				// to say.
				fmt.Fprintf(out, "  %s and %s may be one tag — ask the user before merging them\n", a, b)
			}
		}
	}
	if dryRun {
		fmt.Fprintln(out, "\ndry run: nothing was written. Re-run without --dry-run to write it.")
		return 0
	}
	if len(add) == 0 {
		return 0
	}
	if !exists {
		text = scaffold.TagsDoc(now())
	}
	text = withEntries(text, add, func(string) string { return todoDefinition })
	if code := write(root, text, !exists, out); code != 0 {
		return code
	}
	if code := logRoot(root, fmt.Sprintf("* **Tags**: added %d tag(s) the documents carry to TAGS.md, each with a TODO definition to write (fdf tag --sync).", len(add)), out); code != 0 {
		return code
	}
	fmt.Fprintln(out, "\nnext: first fold the spellings of one tag into one — `fdf tag --merge <from> <into>` for each pair")
	fmt.Fprintln(out, "      above that is one tag, keeping the one more documents carry; then write each TODO definition")
	fmt.Fprintln(out, "      left (`fdf tag --definition '<what it gathers>' <tag>`). `fdf validate` reports each until then (F15).")
	return 0
}

// defined lists the tags d defines, as written.
func defined(d *bundle.TagDictionary) []string {
	if d == nil {
		return nil
	}
	var out []string
	for _, t := range d.Tags {
		if d.Lookup(t.Name) == t {
			out = append(out, t.Name)
		}
	}
	return out
}

// Merge folds the tag from into the tag into, as a tag fix: every document
// that carries from carries into instead, listed once, and in TAGS.md from's
// entry goes, its definition becoming into's when into has none, and from,
// with the spellings it listed, goes under into's `instead-of` so that it
// does not come back. Only `tags` changes in a document, and no timestamp
// does: a tag fix is a maintenance edit. dryRun prints the plan and writes
// nothing.
func Merge(root, from, into string, dryRun bool, out io.Writer) int {
	if !writable(root, out) {
		return 1
	}
	if from == into {
		fmt.Fprintf(out, "error: --merge takes two different tags, and got %s twice\n", from)
		return 2
	}
	text, d, exists := dictionary(root)
	if exists && d == nil {
		fmt.Fprintln(out, "error: TAGS.md cannot be read as a dictionary — `fdf validate` says what to repair first")
		return 1
	}
	if !bundle.TagName(into) {
		fmt.Fprintln(out, "error: "+refused(d, into))
		return 1
	}
	switch r := d.Replacement(into); {
	case r == from:
		fmt.Fprintf(out, "error: %s is listed under `instead-of` of %s in TAGS.md: it is a spelling of %s, not a tag to merge into — to make %s the tag, take it out of that `instead-of` first\n", into, from, from, into)
		return 1
	case r != "":
		fmt.Fprintf(out, "error: %s is listed under `instead-of` of %s in TAGS.md — merge into %s\n", into, r, r)
		return 1
	}
	if r := d.Replacement(from); r != "" && r != into {
		fmt.Fprintf(out, "error: %s is listed under `instead-of` of %s in TAGS.md — merge it into %s\n", from, r, r)
		return 1
	}
	c := scan(root)
	rels := c[from]
	fromTag, intoTag := d.Lookup(from), d.Lookup(into)
	if len(rels) == 0 && fromTag == nil {
		fmt.Fprintf(out, "error: no document carries %s, and TAGS.md does not define it — nothing to merge (`fdf tag` lists the tags)\n", from)
		return 1
	}

	verb := tense(dryRun, "would retag", "retagged")
	changed := 0
	for _, rel := range rels {
		p := filepath.Join(root, filepath.FromSlash(rel))
		raw, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		updated, ok := retag(string(raw), func(ts []string) []string { return replaced(ts, from, into) })
		if ok {
			// Read back as the validator reads it: a rewrite that left from
			// in place changed nothing, whatever it wrote.
			data, _, parsed := bundle.Frontmatter(updated)
			ok = parsed && !carriesTag(bundle.DocumentTags(data), from)
		}
		if !ok {
			fmt.Fprintf(out, "  left %s as it is: its `tags` are not written as fdf can rewrite them — retag it by hand\n", rel)
			continue
		}
		if !dryRun {
			if err := os.WriteFile(p, []byte(updated), 0o644); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
		}
		changed++
		fmt.Fprintf(out, "%s %s: %s → %s\n", verb, strings.TrimSuffix(rel, ".md"), from, into)
	}

	// from goes under into's instead-of only where there is a dictionary to
	// hold it, and only written as a tag is: F15 rejects any other spelling
	// wherever it stands, so it needs no listing to stay out.
	listed := exists && bundle.TagName(from)
	switch {
	case !exists:
		fmt.Fprintf(out, "note: the bundle has no TAGS.md, so nothing records %s as a spelling of %s\n", from, into)
	case !bundle.TagName(from):
		fmt.Fprintf(out, "note: %s is not written as a tag is, so it is not listed under %s's `instead-of`: F15 rejects it wherever it stands\n", from, into)
	}
	if exists {
		var also []string
		if listed {
			also = append(also, from)
		}
		switch {
		case fromTag != nil:
			also = append(also, fromTag.InsteadOf...)
			if listed {
				fmt.Fprintf(out, "%s TAGS.md's entry for %s, and %s it under %s's `instead-of`\n", tense(dryRun, "would drop", "dropped"), from, tense(dryRun, "would list", "listed"), into)
			} else {
				fmt.Fprintf(out, "%s TAGS.md's entry for %s\n", tense(dryRun, "would drop", "dropped"), from)
			}
			text = withoutEntry(text, from)
		case listed:
			fmt.Fprintf(out, "%s %s under %s's `instead-of` in TAGS.md\n", tense(dryRun, "would list", "listed"), from, into)
		}
		switch {
		case intoTag == nil:
			def := todoDefinition
			if fromTag != nil && fromTag.Defined() {
				def = strings.TrimSpace(fromTag.Definition)
			}
			fmt.Fprintf(out, "%s %s to TAGS.md\n", tense(dryRun, "would add", "added"), into)
			text = withEntries(text, []string{into}, func(string) string { return def })
		case !intoTag.Defined() && fromTag != nil && fromTag.Defined():
			// The one meaning anyone wrote down is from's: it is not lost.
			fmt.Fprintf(out, "%s %s's definition to %s, which had none\n", tense(dryRun, "would give", "gave"), from, into)
			text = withDefinition(text, into, strings.TrimSpace(fromTag.Definition))
		}
		text = withInsteadOf(text, into, also)
	}
	if dryRun {
		fmt.Fprintln(out, "\ndry run: nothing was written. Re-run without --dry-run to merge them.")
		return 0
	}
	if exists {
		if code := writeStamped(root, text, false, false, out); code != 0 {
			return code
		}
	}
	entry := fmt.Sprintf("* **Tag fix**: `%s` → `%s` in %d document(s)", from, into, changed)
	if listed {
		entry += fmt.Sprintf("; `%s` is listed under `%s`'s `instead-of`", from, into)
	}
	if code := logRoot(root, entry+" (fdf tag --merge).", out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "\ndone: %d document(s) carry %s instead of %s; logged in LOG.md.\n", changed, into, from)
	return 0
}

// carriesTag reports whether ts holds tag.
func carriesTag(ts []string, tag string) bool {
	for _, t := range ts {
		if t == tag {
			return true
		}
	}
	return false
}

// tense is what a dry run would do, or what a run did.
func tense(dryRun bool, would, did string) string {
	if dryRun {
		return would
	}
	return did
}

// replaced is ts with from replaced by into, into listed once.
func replaced(ts []string, from, into string) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range ts {
		if t == from {
			t = into
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// Check says whether the tags a person gives a new document can be given:
// from spec 1.2 on, each is written as a tag is and is no spelling TAGS.md
// lists under another tag. A tag the dictionary lacks is fine: Stamp adds it.
func Check(root string, names []string, out io.Writer) bool {
	if len(names) == 0 || !dictionaryTaken(root) {
		return true
	}
	_, d, _ := dictionary(root)
	for _, n := range names {
		if why := refused(d, n); why != "" {
			fmt.Fprintln(out, "error: --tags: "+why)
			return false
		}
	}
	return true
}

// Stamp gives the new document id the tags names, and adds each the bundle's
// TAGS.md does not define to it, with a TODO definition for a person to
// write. A bundle with no TAGS.md gets none: its tags may predate the
// dictionary, which it takes up with fdf tag --sync.
func Stamp(root, id string, names []string, out io.Writer) int {
	names = uniq(names)
	if len(names) == 0 {
		return 0
	}
	p := filepath.Join(root, filepath.FromSlash(id)+".md")
	raw, err := os.ReadFile(p)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	updated, ok := retag(string(raw), func([]string) []string { return names })
	if !ok {
		fmt.Fprintf(out, "error: could not write `tags` into %s.md\n", id)
		return 1
	}
	if err := os.WriteFile(p, []byte(updated), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "tagged it %s\n", strings.Join(names, ", "))
	if !dictionaryTaken(root) {
		return 0
	}
	text, d, exists := dictionary(root)
	if !exists {
		fmt.Fprintln(out, "note: the bundle has no TAGS.md; `fdf tag --sync` starts one from the tags its documents carry.")
		return 0
	}
	if d == nil {
		return 0 // the validator says what to repair
	}
	var add []string
	for _, n := range names {
		if d.Lookup(n) == nil {
			add = append(add, n)
		}
	}
	if len(add) == 0 {
		return 0
	}
	noteResemblance(add, d, out)
	if code := write(root, withEntries(text, add, func(string) string { return todoDefinition }), false, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "added %s to TAGS.md with a TODO definition — replace it with what the tag gathers; `fdf validate` reports it until then (F15)\n", strings.Join(add, ", "))
	return 0
}

// noteResemblance says, for each tag about to be added, which tags the
// dictionary defines already that it may be another spelling of, so the
// person adding it can reuse one instead: the moment a tag is added is when
// a second spelling is cheapest to stop. Each other tag added with it counts
// too.
func noteResemblance(add []string, d *bundle.TagDictionary, out io.Writer) {
	known := defined(d)
	for i, n := range add {
		var like []string
		for _, k := range append(append([]string{}, known...), add[:i]...) {
			if resembles(n, k) {
				like = append(like, k)
			}
		}
		if len(like) > 0 {
			fmt.Fprintf(out, "note: %s looks like %s — if it means the same, carry that tag instead (`fdf tag --merge %s %s` folds it in later)\n", n, strings.Join(like, ", "), n, like[0])
		}
	}
}

// resembles reports whether two tags may be spellings of one: they differ
// only by hyphens or a plural ending, one starts with the other, or both
// start with the same five letters.
func resembles(a, b string) bool {
	if _, _, alike := bundle.LookAlike(a, b, nil); alike {
		return true
	}
	x, y := strings.ReplaceAll(a, "-", ""), strings.ReplaceAll(b, "-", "")
	if len(y) < len(x) {
		x, y = y, x
	}
	if len(x) >= 3 && strings.HasPrefix(y, x) {
		return true
	}
	return len(x) >= 5 && x[:5] == y[:5]
}

// Inherited returns the tags the document id carries that a new document may
// carry too: each written as a tag is, and none the dictionary lists as a
// spelling of another. A Fix or Change started from a bug takes the bug's.
func Inherited(root, id string) []string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(id)+".md"))
	if err != nil {
		return nil
	}
	data, _, ok := bundle.Frontmatter(string(raw))
	if !ok {
		return nil
	}
	_, d, _ := dictionary(root)
	var out []string
	for _, t := range bundle.DocumentTags(data) {
		if refused(d, t) == "" {
			out = append(out, t)
		}
	}
	return uniq(out)
}

// uniq is names without repeats, in order.
func uniq(names []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, n := range names {
		if n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// width is the widest of names and head, for a table's first column.
func width(names []string, head string) int {
	w := len(head)
	for _, n := range names {
		if len(n) > w {
			w = len(n)
		}
	}
	return w
}

// clip shortens s to n characters, marking the cut.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

var (
	keyLineRe   = regexp.MustCompile(`^([A-Za-z0-9_-]+):\s?(.*)$`)
	itemLineRe  = regexp.MustCompile(`^(\s*-\s+)(.*)$`)
	stampLineRe = regexp.MustCompile(`^timestamp:`)
)

// retag rewrites a document's `tags` with the list fn makes of the ones it
// carries, keeping the form it was written in — `[a, b]`, one value, or a
// block of `- a` lines — and writes `tags: [a, b]` before its timestamp when
// it has none. ok is false when the text has no frontmatter to rewrite.
func retag(text string, fn func([]string) []string) (string, bool) {
	lines := strings.SplitAfter(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return text, false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return text, false
	}
	eol := func(l string) string {
		if strings.HasSuffix(l, "\r\n") {
			return "\r\n"
		}
		return "\n"
	}
	inline := func(ts []string) string { return "tags: [" + strings.Join(ts, ", ") + "]" }
	for i := 1; i < end; i++ {
		m := keyLineRe.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
		if m == nil || m[1] != "tags" {
			continue
		}
		val := strings.TrimSpace(m[2])
		var out []string
		switch {
		case strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]"):
			var ts []string
			for _, p := range strings.Split(val[1:len(val)-1], ",") {
				if q := unquote(p); q != "" {
					ts = append(ts, q)
				}
			}
			out = append(out, lines[:i]...)
			out = append(out, inline(fn(ts))+eol(lines[i]))
			out = append(out, lines[i+1:]...)
		case val != "":
			out = append(out, lines[:i]...)
			out = append(out, inline(fn([]string{unquote(val)}))+eol(lines[i]))
			out = append(out, lines[i+1:]...)
		default:
			// A block of `- tag` lines, which comment lines may sit among, as
			// the frontmatter parser reads them: the comments stay where they
			// are, and the new tags take the old ones' places in order.
			j, indent := i+1, "  - "
			var ts []string
			var slots []int // the lines that held a tag
			for ; j < end; j++ {
				l := strings.TrimRight(lines[j], "\r\n")
				if t := strings.TrimSpace(l); t == "" || strings.HasPrefix(t, "#") {
					continue // a blank or comment line the parser skips too
				}
				im := itemLineRe.FindStringSubmatch(l)
				if im == nil {
					break
				}
				if len(slots) == 0 {
					indent = im[1]
				}
				slots = append(slots, j)
				ts = append(ts, unquote(im[2]))
			}
			next := fn(ts)
			out = append(out, lines[:i+1]...)
			k := 0
			for n := i + 1; n < j; n++ {
				if k < len(slots) && slots[k] == n {
					if k < len(next) {
						out = append(out, indent+next[k]+eol(lines[i]))
					}
					k++
					continue
				}
				out = append(out, lines[n])
			}
			for ; k < len(next); k++ {
				out = append(out, indent+next[k]+eol(lines[i]))
			}
			out = append(out, lines[j:]...)
		}
		return strings.Join(out, ""), true
	}
	at := end
	for i := 1; i < end; i++ {
		if stampLineRe.MatchString(lines[i]) {
			at = i
			break
		}
	}
	ts := fn(nil)
	if len(ts) == 0 {
		return text, true
	}
	out := append([]string{}, lines[:at]...)
	out = append(out, inline(ts)+eol(lines[0]))
	out = append(out, lines[at:]...)
	return strings.Join(out, ""), true
}

// unquote drops the quotes a frontmatter value may be written in.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	return strings.Trim(s, `'`)
}

// read returns TAGS.md's lines as a reader reads them: comments and code
// blanked, so that a heading or a bullet inside one is not taken for an
// entry. It has a line for each of text's, at the same index.
func read(text string) []string { return strings.Split(bundle.TagsBody(text), "\n") }

var (
	topHeadingRe = regexp.MustCompile(`^#\s+(.*\S)\s*$`)
	subHeadRe    = regexp.MustCompile(`^##\s+(.*\S)\s*$`)
	insteadRe    = regexp.MustCompile(`^(\s*[-*]\s+instead-of\s*:\s*)(.*?)\s*$`)
)

// tagsSection returns the line range of TAGS.md's `# Tags` section: from the
// line after its heading to the line before the next `#` heading, or the end.
// start is -1 when there is no such section.
func tagsSection(read []string) (start, end int) {
	start, end = -1, len(read)
	for i, l := range read {
		m := topHeadingRe.FindStringSubmatch(strings.TrimRight(l, "\r"))
		if m == nil {
			continue
		}
		if start >= 0 {
			return start, i
		}
		if strings.EqualFold(strings.TrimSpace(m[1]), "Tags") {
			start = i + 1
		}
	}
	return start, end
}

// withEntries adds a `## <tag>` entry for each of names at the end of
// TAGS.md's `# Tags` section, with the definition def gives it. A text with
// no such section gets one at its end.
func withEntries(text string, names []string, def func(string) string) string {
	lines := strings.Split(text, "\n")
	start, end := tagsSection(read(text))
	var entry []string
	for _, n := range names {
		entry = append(entry, "## "+n, def(n), "")
	}
	if start < 0 {
		body := strings.TrimRight(text, "\n") + "\n\n# Tags\n\n" + strings.Join(entry, "\n")
		return strings.TrimRight(body, "\n") + "\n"
	}
	// After the section's last line that is not blank.
	at := end
	for at > start && strings.TrimSpace(lines[at-1]) == "" {
		at--
	}
	head := append([]string{}, lines[:at]...)
	if len(head) > 0 && strings.TrimSpace(head[len(head)-1]) != "" {
		head = append(head, "")
	}
	tail := lines[end:]
	out := append(head, entry...)
	if len(tail) > 0 {
		out = append(out, tail...)
	} else {
		out = append(out[:len(out)-1], "")
	}
	return strings.Join(out, "\n")
}

// withDefinition makes def tag's definition in TAGS.md, in place of the
// prose it has; its `instead-of` stays.
func withDefinition(text, tag, def string) string {
	lines, seen := strings.Split(text, "\n"), read(text)
	from, to := entryRange(seen, tag)
	if from < 0 {
		return text
	}
	kept := []string{lines[from], def}
	for i := from + 1; i < to; i++ {
		// Its bullets stay, and so does a comment or code in it, which a
		// reader does not read as the definition.
		if tagBulletRe.MatchString(seen[i]) || seen[i] != lines[i] {
			kept = append(kept, lines[i])
		}
	}
	// A blank line ends the entry: before the next heading, or, for the last
	// entry, the file's final newline.
	kept = append(kept, "")
	return strings.Join(append(append(append([]string{}, lines[:from]...), kept...), lines[to:]...), "\n")
}

// tagBulletRe is a bullet of a tag's entry, such as its `instead-of`.
var tagBulletRe = regexp.MustCompile(`^\s*[-*]\s+[a-z-]+\s*:`)

// entryRange returns the lines of tag's `## <tag>` entry in TAGS.md's
// `# Tags`: its heading through the line before the next heading. from is -1
// when there is none.
func entryRange(read []string, tag string) (from, to int) {
	start, end := tagsSection(read)
	from, to = -1, end
	if start < 0 {
		return -1, -1
	}
	for i := start; i < end; i++ {
		m := subHeadRe.FindStringSubmatch(strings.TrimRight(read[i], "\r"))
		if m == nil {
			continue
		}
		if from >= 0 {
			return from, i
		}
		if strings.TrimSpace(m[1]) == tag {
			from = i
		}
	}
	return from, to
}

// withoutEntry removes tag's entry from TAGS.md.
func withoutEntry(text, tag string) string {
	lines := strings.Split(text, "\n")
	from, to := entryRange(read(text), tag)
	if from < 0 {
		return text
	}
	return strings.Join(append(append([]string{}, lines[:from]...), lines[to:]...), "\n")
}

// withInsteadOf lists words under tag's `instead-of` in TAGS.md, after those
// it lists, each once and never the tag itself.
func withInsteadOf(text, tag string, words []string) string {
	lines, seen := strings.Split(text, "\n"), read(text)
	from, to := entryRange(seen, tag)
	if from < 0 {
		return text
	}
	at := -1
	var have []string
	var extra []int // instead-of bullets after the first, folded into it
	for i := from + 1; i < to; i++ {
		if m := insteadRe.FindStringSubmatch(strings.TrimRight(seen[i], "\r")); m != nil {
			if at < 0 {
				at = i
			} else {
				extra = append(extra, i)
			}
			for _, w := range strings.Split(m[2], ",") {
				if w = strings.TrimSpace(w); w != "" {
					have = append(have, w)
				}
			}
		}
	}
	listed := map[string]bool{tag: true}
	for _, w := range have {
		listed[w] = true
	}
	all := have
	for _, w := range words {
		if !listed[w] {
			listed[w] = true
			all = append(all, w)
		}
	}
	if len(all) == len(have) && len(extra) == 0 {
		return text
	}
	line := "- instead-of: " + strings.Join(all, ", ")
	if at >= 0 {
		lines[at] = line
		for k := len(extra) - 1; k >= 0; k-- {
			lines = append(lines[:extra[k]], lines[extra[k]+1:]...)
		}
		return strings.Join(lines, "\n")
	}
	last := to
	for last > from+1 && strings.TrimSpace(lines[last-1]) == "" {
		last--
	}
	out := append(append(append([]string{}, lines[:last]...), line), lines[last:]...)
	return strings.Join(out, "\n")
}

// withTimestamp sets the frontmatter's timestamp to ts, or writes one before
// the block's closing line when it has none.
func withTimestamp(text, ts string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return text
	}
	for i := 1; i < len(lines); i++ {
		switch {
		case strings.TrimSpace(lines[i]) == "---":
			out := append(append(append([]string{}, lines[:i]...), "timestamp: "+ts), lines[i:]...)
			return strings.Join(out, "\n")
		case stampLineRe.MatchString(lines[i]):
			lines[i] = "timestamp: " + ts
			return strings.Join(lines, "\n")
		}
	}
	return text
}
