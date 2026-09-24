// Package refactor implements `fdf mv` and `fdf lexicon --fix` (v0.7): the
// two maintenance edits a tool can do whole. A move renames a document with
// its trail and repairs every reference to it; a lexicon fix replaces banned
// words with their terms, keeping every scenario name's joins intact. Both
// reach frozen documents, because neither changes a fact about the work, and
// both log what they did.
package refactor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/links"
	"github.com/GiteshDalal/fdf/cli/internal/logs"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

var (
	segRe      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	taskBaseRe = regexp.MustCompile(`^\d{2}-[a-z0-9][a-z0-9-]*$`)
	trailRe    = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)\.([a-z]+)\.md$`)
	typeLineRe = regexp.MustCompile(`(?m)^type:\s*(\S+)\s*$`)
	fenceRe    = regexp.MustCompile("^(`{3,}|~{3,})")
)

// registers are the reserved bundle-root directories holding documents of
// one kind each (releases/ holds releases and nothing moves in it). Which of
// them a bundle has follows its pin (scaffold.ReservedDirs): on a v0.6
// bundle, bugs/ is a feature group and moves like one.
var registers = map[string]string{"changes": "change", "practices": "practice", "debts": "debt", "bugs": "bug"}

// kind is what a move moves.
type kind int

const (
	kFeature kind = iota
	kGroup
	kRegisterDoc
	kRegisterGroup
	kTask
)

// plan is a move worked out in full before anything is touched.
type plan struct {
	from, to string
	kind     kind
	files    map[string]string // old rel -> new rel, every file that moves
	dirs     map[string]string // old dir rel -> new dir rel
	ids      map[string]string // old document ID -> new document ID
	task     [2]string         // a task rename: old and new stem, within one task directory
	flip     map[string]string // new rel -> the type it takes (a debt re-filed as a bug, or back)
	prefix   string            // the bundle's path in the project, e.g. "docs/features/"
	reserved map[string]bool   // the bundle-root directories the pin reserves, releases/ included
	reg      map[string]string // the registers among them: directory -> the noun of what it holds
}

// Move moves or renames a document — with its trail, task directory and log —
// or a whole group, and repairs every reference to it across the bundle.
// projectRoot, when set, is searched for references outside the bundle, which
// are reported and never edited. dryRun prints the plan and changes nothing.
func Move(root, projectRoot, from, to string, dryRun bool, out io.Writer) int {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	from, to = cleanID(from), cleanID(to)
	p, err := makePlan(rootAbs, from, to)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if projectRoot != "" {
		if r, rerr := filepath.Rel(projectRoot, rootAbs); rerr == nil && r != "." && !strings.HasPrefix(r, "..") {
			p.prefix = filepath.ToSlash(r) + "/"
		}
	}

	edits, err := rewriteAll(rootAbs, p)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	listingMoves := moveListings(rootAbs, p, edits)

	verb := map[bool]string{true: "would move", false: "moved"}[dryRun]
	olds := sortedKeys(p.files)
	for _, o := range olds {
		fmt.Fprintf(out, "%s %s -> %s\n", verb, o, p.files[o])
	}
	for _, rel := range sortedKeys(p.flip) {
		fmt.Fprintf(out, "  %s re-filed: type is now %s\n", rel, p.flip[rel])
	}
	changed := 0
	refs := 0
	for _, rel := range sortedEditKeys(edits) {
		e := edits[rel]
		if e.count > 0 {
			changed++
			refs += e.count
			// Named where the file is once the move is done: a moved or
			// re-filed document's repairs are reported at its new path.
			fmt.Fprintf(out, "  %s: %d reference(s) repaired\n", e.newRel, e.count)
		}
	}
	for _, m := range listingMoves {
		fmt.Fprintln(out, "  "+m)
	}
	external := externalRefs(rootAbs, projectRoot, p)

	// changed counts files, not documents: an INDEX.md or LOG.md whose links
	// were repaired is one of them.
	if dryRun {
		fmt.Fprintf(out, "\ndry run: %d file(s) would move and %d reference(s) in %d file(s) would be repaired. Nothing was changed.\n", len(p.files), refs, changed)
		reportExternal(external, out)
		return 0
	}

	if err := apply(rootAbs, p, edits); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := logMove(rootAbs, from, to, refs, changed); err != nil {
		fmt.Fprintln(out, "error: writing LOG.md:", err)
		return 1
	}
	fmt.Fprintf(out, "\ndone: %d file(s) moved; %d reference(s) repaired in %d file(s); logged in LOG.md.\n", len(p.files), refs, changed)
	if left := emptiedGroup(rootAbs, p); left != "" {
		fmt.Fprintf(out, "note: %s now holds no documents — remove it, and its listing, if the group is gone.\n", left)
	}
	reportExternal(external, out)
	return 0
}

func cleanID(id string) string {
	id = strings.Trim(filepath.ToSlash(id), "/")
	return strings.TrimSuffix(id, ".md")
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }
func isDir(p string) bool  { fi, err := os.Stat(p); return err == nil && fi.IsDir() }

// makePlan classifies the source, checks the target, and lists everything
// that moves.
func makePlan(rootAbs, from, to string) (*plan, error) {
	p := &plan{from: from, to: to, files: map[string]string{}, dirs: map[string]string{}, ids: map[string]string{}, flip: map[string]string{},
		reserved: scaffold.ReservedDirs(rootAbs), reg: map[string]string{}}
	for dir, noun := range registers {
		if p.reserved[dir] {
			p.reg[dir] = noun
		}
	}
	fp, tp := strings.Split(from, "/"), strings.Split(to, "/")
	for _, s := range append(append([]string{}, fp...), tp...) {
		if !segRe.MatchString(s) && !taskBaseRe.MatchString(s) {
			return nil, fmt.Errorf("IDs are lowercase [a-z0-9-] segments; %q is not one", s)
		}
	}
	if from == to {
		return nil, fmt.Errorf("%s is already where it is", from)
	}
	src := func(rel string) string { return filepath.Join(rootAbs, filepath.FromSlash(rel)) }
	docExists := func(id string) bool { return exists(src(id + ".md")) }

	switch {
	case len(fp) == 1:
		if p.reserved[fp[0]] {
			return nil, fmt.Errorf("%s/ is a reserved directory; move the documents or groups inside it instead", fp[0])
		}
		if !isDir(src(from)) {
			return nil, fmt.Errorf("%s is not a group in this bundle", from)
		}
		if len(tp) != 1 || p.reserved[tp[0]] {
			return nil, fmt.Errorf("a feature group moves to another group name, not to %s", to)
		}
		p.kind = kGroup
	case p.reg[fp[0]] != "":
		switch {
		case (len(fp) == 2 || len(fp) == 3) && docExists(from):
			p.kind = kRegisterDoc
		case len(fp) == 2 && isDir(src(from)):
			p.kind = kRegisterGroup
		case taskBaseRe.MatchString(fp[len(fp)-1]) && docExists(from):
			p.kind = kTask
		default:
			return nil, fmt.Errorf("%s is not a document or group in this bundle", from)
		}
	case len(fp) == 2 && docExists(from):
		p.kind = kFeature
	case len(fp) == 3 && taskBaseRe.MatchString(fp[2]) && docExists(from):
		p.kind = kTask
	default:
		return nil, fmt.Errorf("%s is not a feature, group, change, fix, practice, debt, bug or task in this bundle", from)
	}

	if exists(src(to+".md")) || (p.kind == kGroup || p.kind == kRegisterGroup) && exists(src(to)) {
		return nil, fmt.Errorf("%s already exists — a move never overwrites; pick another name, or move documents into an existing group one at a time", to)
	}

	switch p.kind {
	case kFeature:
		if len(tp) != 2 || p.reserved[tp[0]] {
			return nil, fmt.Errorf("a feature moves to <group>/<slug>, not %s", to)
		}
		p.addDoc(rootAbs, from, to, true)
	case kGroup:
		p.dirs[from] = to
		filepath.WalkDir(src(from), func(q string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel := relSlash(rootAbs, q)
			p.files[rel] = to + strings.TrimPrefix(rel, from)
			if base := path.Base(rel); strings.HasSuffix(base, ".md") && !strings.Contains(strings.TrimSuffix(base, ".md"), ".") &&
				base != "INDEX.md" && base != "LOG.md" && strings.Count(rel, "/") == 1 {
				id := strings.TrimSuffix(rel, ".md")
				p.ids[id] = to + strings.TrimPrefix(id, from)
			}
			return nil
		})
	case kRegisterDoc:
		fromReg, toReg := fp[0], tp[0]
		// Re-filing needs both registers, and bugs/ is one from v0.7 only.
		both := p.reg["debts"] != "" && p.reg["bugs"] != ""
		refile := both && ((fromReg == "debts" && toReg == "bugs") || (fromReg == "bugs" && toReg == "debts"))
		if fromReg != toReg && !refile {
			why := "nothing changes register"
			if both {
				why = "a debt and a bug can be re-filed as each other; nothing else changes register"
			}
			return nil, fmt.Errorf("a %s stays under %s/ (%s)", p.reg[fromReg], fromReg, why)
		}
		if len(tp) < 2 || len(tp) > 3 {
			return nil, fmt.Errorf("%s moves to %s/<slug> or %s/<group>/<slug>, not %s", from, toReg, toReg, to)
		}
		if isDir(src(from)) && fromReg != "changes" {
			return nil, fmt.Errorf("%s has a directory beside it, which a %s never owns", from, p.reg[fromReg])
		}
		if toReg == "changes" {
			if why := scaffold.ChangePlaceTaken(rootAbs, strings.TrimPrefix(to, "changes/")); why != "" {
				return nil, fmt.Errorf("%s", why)
			}
		}
		p.addDoc(rootAbs, from, to, fromReg == "changes")
		if refile {
			p.flip[to+".md"] = map[string]string{"bugs": "Bug", "debts": "Debt"}[toReg]
		}
	case kRegisterGroup:
		if len(tp) != 2 || tp[0] != fp[0] {
			return nil, fmt.Errorf("a %s/ group moves to another %s/<group>, not %s", fp[0], fp[0], to)
		}
		p.dirs[from] = to
		filepath.WalkDir(src(from), func(q string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel := relSlash(rootAbs, q)
			p.files[rel] = to + strings.TrimPrefix(rel, from)
			if base := path.Base(rel); strings.Count(rel, "/") == 2 && strings.HasSuffix(base, ".md") &&
				!strings.Contains(strings.TrimSuffix(base, ".md"), ".") && base != "INDEX.md" && base != "LOG.md" {
				id := strings.TrimSuffix(rel, ".md")
				p.ids[id] = to + strings.TrimPrefix(id, from)
			}
			return nil
		})
	case kTask:
		if path.Dir(from) != path.Dir(to) || !taskBaseRe.MatchString(tp[len(tp)-1]) {
			return nil, fmt.Errorf("a task is renamed within its own task directory, to NN-slug: %s/NN-<slug>", path.Dir(from))
		}
		p.files[from+".md"] = to + ".md"
		p.ids[from] = to // a task's ID is its path, like any document's
		p.task = [2]string{path.Base(from), path.Base(to)}
	}
	// A move never overwrites: not the target document, and not a stray file
	// that happens to sit where one of its siblings would land.
	for _, o := range sortedKeys(p.files) {
		if exists(src(p.files[o])) {
			return nil, fmt.Errorf("moving %s would overwrite %s, which already exists", o, p.files[o])
		}
	}
	return p, nil
}

// addDoc adds a document with its stem-qualified siblings and, when it may
// own one, its task directory.
func (p *plan) addDoc(rootAbs, from, to string, tasks bool) {
	p.ids[from] = to
	dir := filepath.Join(rootAbs, filepath.FromSlash(path.Dir(from)))
	stem := path.Base(from)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == stem+".md" {
			p.files[from+".md"] = to + ".md"
		} else if m := trailRe.FindStringSubmatch(e.Name()); m != nil && m[1] == stem {
			p.files[path.Dir(from)+"/"+e.Name()] = to + "." + m[2] + ".md"
		}
	}
	if tasks && isDir(filepath.Join(rootAbs, filepath.FromSlash(from))) {
		p.dirs[from] = to
		filepath.WalkDir(filepath.Join(rootAbs, filepath.FromSlash(from)), func(q string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				rel := relSlash(rootAbs, q)
				p.files[rel] = to + strings.TrimPrefix(rel, from)
			}
			return nil
		})
	}
}

func relSlash(rootAbs, p string) string {
	r, _ := filepath.Rel(rootAbs, p)
	return filepath.ToSlash(r)
}

// move is the plan as the link engine reads it: bundle-relative paths.
func (p *plan) move() links.Move { return links.Move{Files: p.files, Dirs: p.dirs} }

// newPathOf maps a bundle-relative path through the move, or returns it as is.
func (p *plan) newPathOf(rel string) (string, bool) { return p.move().New(rel) }

// edit is one file's new text and how many references changed in it.
type edit struct {
	newRel, text string
	count        int
}

type span struct {
	start, end int
	text       string
}

// rewriteAll computes every file's new text: links and link-style references
// to anything that moves, and — outside logs, which keep their words — every
// mention of a moved document's ID.
func rewriteAll(rootAbs string, p *plan) (map[string]*edit, error) {
	edits := map[string]*edit{}
	err := filepath.WalkDir(rootAbs, func(q string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if q != rootAbs && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel := relSlash(rootAbs, q)
		if !strings.HasSuffix(rel, ".md") || rel == "SPEC.md" {
			return nil
		}
		raw, rerr := os.ReadFile(q)
		if rerr != nil {
			return rerr
		}
		text := string(raw)
		newRel, _ := p.newPathOf(rel)
		base := path.Base(rel)
		isLog := base == "LOG.md" || strings.HasSuffix(base, ".log.md")

		var reps []span
		targets := map[int]bool{} // link targets, which mentions leave to the engine
		site := links.Site{OldPath: rel, NewPath: newRel}
		for _, l := range links.Find(text) {
			for i := l.Start; i < l.End; i++ {
				targets[i] = true
			}
			if l.InCode {
				continue // a sample, not a reference
			}
			if nt, ok := links.Retarget(l.Target, site, p.move()); ok {
				reps = append(reps, span{l.Start, l.End, nt})
			}
		}
		if !isLog {
			reps = append(reps, p.mentions(text, rel, targets)...)
		}
		if t, ok := p.flip[newRel]; ok {
			reps = append(reps, flipType(text, t)...)
		}
		if len(reps) == 0 && newRel == rel {
			return nil
		}
		edits[rel] = &edit{newRel: newRel, text: applySpans(text, reps), count: len(reps)}
		return nil
	})
	return edits, err
}

// isPathByte reports the characters a path or an ID is made of.
func isPathByte(c byte) bool {
	return c == '-' || c == '_' || c == '.' || c == '/' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// mentions finds every mention of a moved document's ID outside a link
// target: in frontmatter edges (affects, depends-on, replaced-by, retires,
// superseded-by, resolves), declaration headings, code spans and prose. An ID
// is a whole token: not inside a longer path such as src/<id>/handler.go,
// though "/<id>" at a boundary is a bundle-relative path and counts. The
// suffixes it may carry — ".spec.md", "/01-task.md" — move with it.
func (p *plan) mentions(text, rel string, targets map[int]bool) []span {
	var out []span
	ids := sortedKeys(p.ids)
	sort.Slice(ids, func(i, j int) bool { return len(ids[i]) > len(ids[j]) }) // longest first
	taken := map[int]bool{}
	for _, old := range ids {
		for i := 0; ; {
			j := strings.Index(text[i:], old)
			if j < 0 {
				break
			}
			s, e := i+j, i+j+len(old)
			i = e
			if targets[s] || taken[s] {
				continue
			}
			before := byte(' ')
			if s > 0 {
				before = text[s-1]
			}
			switch {
			case p.prefix != "" && s >= len(p.prefix) && text[s-len(p.prefix):s] == p.prefix &&
				(s == len(p.prefix) || !isPathByte(text[s-len(p.prefix)-1])):
				// the document by its path in the project: docs/features/<id>
			case before == '/':
				if s > 1 && isPathByte(text[s-2]) {
					continue
				}
			case isPathByte(before):
				continue
			}
			if e < len(text) {
				after := text[e]
				if after == '-' || after == '_' || after >= '0' && after <= '9' || after >= 'a' && after <= 'z' || after >= 'A' && after <= 'Z' {
					continue
				}
			}
			for k := s; k < e; k++ {
				taken[k] = true
			}
			out = append(out, span{s, e, p.ids[old]})
		}
	}
	// A renamed task is named by its siblings' `depends-on`.
	if p.task[0] != "" && path.Dir(rel) == path.Dir(p.from) && path.Base(rel) != p.task[0]+".md" {
		out = append(out, dependsOnRenames(text, p.task[0], p.task[1])...)
	}
	return out
}

var dependsOnRe = regexp.MustCompile(`(?m)^depends-on:.*$`)

func dependsOnRenames(text, old, new string) []span {
	var out []span
	for _, loc := range dependsOnRe.FindAllStringIndex(text, -1) {
		line := text[loc[0]:loc[1]]
		for i := 0; ; {
			j := strings.Index(line[i:], old)
			if j < 0 {
				break
			}
			s, e := i+j, i+j+len(old)
			i = e
			if (s == 0 || !isPathByte(line[s-1])) && (e == len(line) || !isPathByte(line[e])) {
				out = append(out, span{loc[0] + s, loc[0] + e, new})
			}
		}
	}
	return out
}

// flipType re-types a re-filed register entry: a debt's `# Gap` is a bug's
// `# Symptom`, and back.
func flipType(text, to string) []span {
	var out []span
	if loc := typeLineRe.FindStringSubmatchIndex(text); loc != nil {
		out = append(out, span{loc[2], loc[3], to})
	}
	from, heading := "# Gap", "# Symptom"
	if to == "Debt" {
		from, heading = "# Symptom", "# Gap"
	}
	for i := 0; ; {
		j := strings.Index(text[i:], from)
		if j < 0 {
			break
		}
		s := i + j
		i = s + len(from)
		if (s == 0 || text[s-1] == '\n') && (i == len(text) || text[i] == '\n' || text[i] == '\r' || text[i] == ' ') {
			out = append(out, span{s, i, heading})
			break
		}
	}
	return out
}

// applySpans replaces non-overlapping spans, in any order.
func applySpans(text string, reps []span) string {
	sort.Slice(reps, func(i, j int) bool { return reps[i].start < reps[j].start })
	var b strings.Builder
	last := 0
	for _, r := range reps {
		if r.start < last {
			continue // overlapping: the earlier one wins
		}
		b.WriteString(text[last:r.start])
		b.WriteString(r.text)
		last = r.end
	}
	b.WriteString(text[last:])
	return b.String()
}

var listLinkRe = regexp.MustCompile(`^\s*[-*]\s+.*\]\(([^)\s]+)`)

// moveListings moves a moved document's line from its old index to its new
// one, when the move changes the directory it is listed in. It edits the
// already-rewritten texts and reports what it did.
func moveListings(rootAbs string, p *plan, edits map[string]*edit) []string {
	if p.kind != kFeature && p.kind != kRegisterDoc {
		return nil
	}
	oldDir, newDir := path.Dir(p.from), path.Dir(p.to)
	if oldDir == newDir {
		return nil
	}
	oldIdx, newIdx := oldDir+"/INDEX.md", newDir+"/INDEX.md"
	textOf := func(rel string) (string, bool) {
		if e, ok := edits[rel]; ok {
			return e.text, true
		}
		raw, err := os.ReadFile(filepath.Join(rootAbs, filepath.FromSlash(rel)))
		return string(raw), err == nil
	}
	src, ok := textOf(oldIdx)
	if !ok {
		return nil
	}
	var kept, movedLines []string
	for _, line := range strings.Split(src, "\n") {
		if m := listLinkRe.FindStringSubmatch(line); m != nil {
			target := m[1]
			if i := strings.IndexAny(target, "#?"); i >= 0 {
				target = target[:i]
			}
			var resolved string
			if strings.HasPrefix(target, "/") {
				resolved = path.Clean(strings.TrimPrefix(target, "/"))
			} else {
				resolved = path.Clean(path.Join(oldDir, target))
			}
			if resolved == p.to+".md" { // the old index's line, already retargeted
				movedLines = append(movedLines, line)
				continue
			}
		}
		kept = append(kept, line)
	}
	if len(movedLines) == 0 {
		return nil
	}
	setText := func(rel, text string, delta int) {
		if e, ok := edits[rel]; ok {
			e.text, e.count = text, e.count+delta
			return
		}
		edits[rel] = &edit{newRel: rel, text: text, count: delta}
	}
	setText(oldIdx, strings.Join(kept, "\n"), 0)
	// Re-express each line's links from the new index's position: the text
	// moves from one index to the other, and nothing it names moves.
	site := links.Site{OldPath: oldIdx, NewPath: newIdx}
	for i, line := range movedLines {
		var reps []span
		for _, l := range links.Find(line) {
			if nt, ok := links.Retarget(l.Target, site, links.Move{}); ok && !l.InCode {
				reps = append(reps, span{l.Start, l.End, nt})
			}
		}
		movedLines[i] = applySpans(line, reps)
	}
	notes := []string{fmt.Sprintf("listing moved from %s to %s", oldIdx, newIdx)}
	dst, ok := textOf(newIdx)
	if !ok {
		// A new group is headed, and listed beside its siblings, the way
		// `fdf new` and the registers' commands head and list one: a feature
		// group in the root INDEX.md, a register's group in the register's.
		parent, group := path.Split(newDir)
		parent = strings.TrimSuffix(parent, "/")
		heading := strings.ToUpper(group[:1]) + group[1:] // a register itself, named plainly
		if isGroup := parent == "" && p.reg[group] == "" || p.reg[parent] != ""; isGroup {
			heading = scaffold.GroupTitle(parent, group)
			parentIdx := path.Join(parent, "INDEX.md")
			if text, ok := textOf(parentIdx); ok {
				if listed, added := scaffold.WithGroupListing(text, parent, group); added {
					setText(parentIdx, listed, 0)
					notes = append(notes, fmt.Sprintf("group %s/ listed in %s", newDir, parentIdx))
				}
			}
		}
		dst = "# " + heading + "\n\n"
	}
	dst = strings.TrimRight(dst, "\n") + "\n" + strings.Join(movedLines, "\n") + "\n"
	setText(newIdx, dst, len(movedLines))
	return notes
}

// apply performs the move: renames first, then every rewritten text written
// where its file now lives.
func apply(rootAbs string, p *plan, edits map[string]*edit) error {
	abs := func(rel string) string { return filepath.Join(rootAbs, filepath.FromSlash(rel)) }
	for _, o := range sortedKeys(p.files) {
		n := p.files[o]
		if err := os.MkdirAll(filepath.Dir(abs(n)), 0o755); err != nil {
			return err
		}
		if err := os.Rename(abs(o), abs(n)); err != nil {
			return fmt.Errorf("moving %s: %w", o, err)
		}
	}
	for _, rel := range sortedEditKeys(edits) {
		e := edits[rel]
		if err := os.MkdirAll(filepath.Dir(abs(e.newRel)), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(abs(e.newRel), []byte(e.text), 0o644); err != nil {
			return err
		}
	}
	// Directories left empty by the move go with it.
	for _, o := range sortedKeys(p.files) {
		for d := filepath.Dir(abs(o)); d != rootAbs && strings.HasPrefix(d, rootAbs); d = filepath.Dir(d) {
			if err := os.Remove(d); err != nil {
				break // not empty
			}
		}
	}
	return nil
}

// emptiedGroup names a feature group the move left holding only its index.
func emptiedGroup(rootAbs string, p *plan) string {
	if p.kind != kFeature {
		return ""
	}
	g := path.Dir(p.from)
	if g == path.Dir(p.to) {
		return ""
	}
	entries, err := os.ReadDir(filepath.Join(rootAbs, g))
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.Name() != "INDEX.md" {
			return ""
		}
	}
	return g + "/"
}

// logMove records the move in the bundle-root LOG.md, newest first. The IDs
// sit in code spans: the entry mentions them, it does not choose them.
func logMove(rootAbs, from, to string, refs, files int) error {
	p := filepath.Join(rootAbs, "LOG.md")
	raw, err := os.ReadFile(p)
	body := string(raw)
	if err != nil {
		body = "# Bundle Update Log\n"
	}
	line := fmt.Sprintf("* **Moved**: `%s` → `%s` (fdf mv; %d reference(s) repaired in %d file(s)).\n", from, to, refs, files)
	return os.WriteFile(p, []byte(logs.Insert(body, line)), 0o644)
}

// externalRefs finds files outside the bundle that still name a moved path:
// instruction files, code comments, READMEs. They are reported, never edited.
func externalRefs(rootAbs, projectRoot string, p *plan) []string {
	if projectRoot == "" {
		return nil
	}
	bundleRel, err := filepath.Rel(projectRoot, rootAbs)
	if err != nil || strings.HasPrefix(bundleRel, "..") {
		return nil
	}
	bundleRel = filepath.ToSlash(bundleRel)
	var patterns []string
	for _, old := range sortedKeys(p.ids) {
		patterns = append(patterns, bundleRel+"/"+old)
	}
	for _, old := range sortedKeys(p.dirs) {
		patterns = append(patterns, bundleRel+"/"+old+"/")
	}
	if len(patterns) == 0 {
		return nil
	}
	args := []string{"-C", projectRoot, "grep", "-n", "-F"}
	for _, pat := range patterns {
		args = append(args, "-e", pat)
	}
	args = append(args, "--", ".", ":(exclude)"+bundleRel)
	outb, _ := exec.Command("git", args...).Output()
	var hits []string
	for _, l := range strings.Split(strings.TrimSpace(string(outb)), "\n") {
		if l != "" {
			hits = append(hits, l)
		}
	}
	return hits
}

func reportExternal(hits []string, out io.Writer) {
	if len(hits) == 0 {
		return
	}
	fmt.Fprintf(out, "\n%d reference(s) outside the bundle still name the old path — fdf mv does not edit them:\n", len(hits))
	for i, h := range hits {
		if i == 10 {
			fmt.Fprintf(out, "  …and %d more\n", len(hits)-10)
			break
		}
		fmt.Fprintln(out, "  "+h)
	}
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedEditKeys(m map[string]*edit) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
