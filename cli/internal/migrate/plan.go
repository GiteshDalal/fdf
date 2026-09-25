package migrate

// A migration is worked out in full before anything is written: every file
// that moves, every file that goes, and the text of every file migrate
// writes. Nothing touches the bundle until the plan is complete, so a
// refusal leaves it as it was.

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/links"
	"github.com/GiteshDalal/fdf/cli/internal/logs"
)

// plan is a migration worked out in full.
type plan struct {
	root string // the bundle root, absolute
	from string // the pin migrate found, "" for none

	// files are the bundle's files as they stand, by bundle-relative path,
	// hidden files and directories aside; texts0 holds each Markdown file's
	// text as it stands.
	files  []string
	texts0 map[string]string

	// moves are the files the older layouts' steps rename, by path before ->
	// after: 0.1's case renames and 0.3's trail lift, one entry per file.
	moves map[string]string
	// gone are the files migrate removes: v0.1's vendored fdf-spec.md.
	gone []string
	// aliases name, for the link engine only, where a link to a file that
	// does not move on must point now: fdf-spec.md is the vendored SPEC.md,
	// and a missing nested TEST.md is the stub written beside the feature.
	aliases map[string]string
	// texts are the files migrate writes, by path after the moves: a file's
	// new text, or a new file.
	texts map[string]string
	// stubs are the test documents migrate writes for features that need
	// one, with how many scenario cases each holds.
	stubs map[string]int

	tags int // status tags removed from index listings
}

// rootWrites are the files at the bundle root that migrate writes whatever
// they hold — INDEX.md, LOG.md and SPEC.md — and the names v0.1 gave them.
var rootWrites = map[string]bool{"INDEX.md": true, "LOG.md": true, "SPEC.md": true, "index.md": true, "log.md": true, "spec.md": true}

// newPlan reads the bundle at root, pinned to pin, and works out its
// migration to target. problems are the reasons it cannot be migrated, found
// before anything is written.
func newPlan(root, pin string) (p *plan, problems []string, err error) {
	// A bundle that is a symbolic link is not where the link is, and the
	// plan reads the files where they are: migrate works on the directory
	// the link names.
	if to, err := os.Readlink(root); err == nil {
		return nil, []string{fmt.Sprintf("%s: a symbolic link to %s — migrate the directory it names, with --root", root, to)}, nil
	}
	p = &plan{root: root, from: pin, texts0: map[string]string{}, moves: map[string]string{},
		aliases: map[string]string{}, texts: map[string]string{}, stubs: map[string]int{}}
	var linked []string
	err = filepath.WalkDir(root, func(q string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if q != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel := relSlash(root, q)
		p.files = append(p.files, rel)
		// A symlink moves as the link it is: migrate never writes through
		// it into the file it names. It writes the root's INDEX.md, LOG.md
		// and SPEC.md whatever they hold, so one of them that is a link is
		// refused.
		if d.Type()&os.ModeSymlink != 0 && rootWrites[rel] {
			to, _ := os.Readlink(q)
			linked = append(linked, fmt.Sprintf("%s: a symbolic link to %s, which migrate would write through — replace it with the file it names", rel, to))
		}
		if strings.HasSuffix(rel, ".md") && d.Type().IsRegular() {
			raw, err := os.ReadFile(q)
			if err != nil {
				return err
			}
			p.texts0[rel] = string(raw)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(linked) > 0 {
		return nil, linked, nil
	}
	// A bundle already in the stem-qualified layout (v0.4 onward) needs no
	// structural work: 0.4 → 0.5 only adds changes/, 0.5 → 0.6 only adds
	// practices/, debts/ and DOMAIN.md, and 0.6 → 0.7 only adds bugs/.
	// Running the pre-0.4 steps over one would be actively wrong — pre-flight
	// reads every `slug.spec.md` as an illegal dotted basename.
	if pin != "0.4" && pin != "0.5" && pin != "0.6" {
		// Refuse content the v0.4 layout cannot hold.
		if problems := preflightV4(root); len(problems) > 0 {
			return nil, problems, nil
		}
		p.caseRenames()
		if err := p.liftTrails(); err != nil {
			return nil, nil, err
		}
		p.stubTests()
	}
	p.repair()
	return p, nil, nil
}

// after is where the file at rel is once the moves are made.
func (p *plan) after(rel string) string {
	if to, ok := p.moves[rel]; ok {
		return to
	}
	return rel
}

// goes reports whether the file at rel goes, as v0.1's vendored fdf-spec.md
// does: migrate writes nothing in its place.
func (p *plan) goes(rel string) bool { return slices.Contains(p.gone, rel) }

// present is the set of the bundle's files once the moves made so far are
// made.
func (p *plan) present() map[string]bool {
	out := map[string]bool{}
	for _, f := range p.files {
		out[p.after(f)] = true
	}
	return out
}

// text is the text of the file at rel once the plan is applied: its planned
// text, or the text of the file that is there now, or "" for none.
func (p *plan) text(rel string) string {
	if t, ok := p.texts[rel]; ok {
		return t
	}
	for _, f := range p.files {
		if p.after(f) == rel {
			return p.texts0[f]
		}
	}
	return ""
}

// source is the file of the bundle, as it stands, that is at rel once
// migrated, or "" when migrate writes rel new.
func (p *plan) source(rel string) string {
	for _, f := range p.files {
		if p.after(f) == rel {
			return f
		}
	}
	return ""
}

// caseRenames plans v0.1's renames: index.md, log.md, spec.md and plan.md are
// INDEX.md, LOG.md, SPEC.md and PLAN.md wherever they are. Its vendored spec,
// fdf-spec.md, goes, and a link to it names the vendored SPEC.md.
func (p *plan) caseRenames() {
	for _, f := range p.files {
		dir, base := path.Split(f)
		if to, ok := renames[base]; ok {
			p.moves[f] = dir + to
		}
	}
	for _, f := range p.files {
		if f == "fdf-spec.md" {
			p.gone = append(p.gone, f)
			p.aliases[f] = "SPEC.md"
		}
	}
}

// liftTrails plans v0.4's trail lift: group/slug/{SPEC,PLAN,TEST,LOG}.md is
// group/slug.<role>.md. It refuses to lift a file onto one that is there.
func (p *plan) liftTrails() error {
	present := p.present()
	for _, f := range p.files {
		cur := p.after(f)
		parts := strings.Split(cur, "/")
		if len(parts) != 3 {
			continue
		}
		role, ok := trailBasenames[parts[2]]
		if !ok {
			continue
		}
		dest := parts[0] + "/" + parts[1] + "." + role + ".md"
		if present[dest] {
			return fmt.Errorf("cannot move %s: %s already exists", cur, dest)
		}
		p.moves[f] = dest
	}
	return nil
}

// stubTests plans a test document for each planned, implementing or done
// feature that has none: one `## <scenario name>` case per scenario, the
// form F8 matches. A link to the nested TEST.md it would have had names the
// stub.
func (p *plan) stubTests() {
	present := p.present()
	for _, f := range p.files {
		parts := strings.Split(f, "/")
		base := path.Base(f)
		if len(parts) != 2 || parts[0] == "releases" || base == "INDEX.md" || base == "LOG.md" ||
			!strings.HasSuffix(base, ".md") || strings.Contains(strings.TrimSuffix(base, ".md"), ".") {
			continue
		}
		raw := p.texts0[f]
		m := statusRe.FindStringSubmatch(raw)
		if m == nil || m[1] != "planned" && m[1] != "implementing" && m[1] != "done" {
			continue
		}
		stem := strings.TrimSuffix(f, ".md")
		test := stem + ".test.md"
		if present[test] {
			continue
		}
		ts := "2026-01-01T00:00:00Z"
		if tm := timestampRe.FindStringSubmatch(raw); tm != nil {
			ts = tm[1]
		}
		var cases []string
		for _, sc := range scenarioRe.FindAllStringSubmatch(raw, -1) {
			cases = append(cases, fmt.Sprintf("## %s\n\nTODO: specify the concrete verification.\n", strings.TrimSpace(sc[1])))
		}
		p.texts[test] = fmt.Sprintf("---\ntype: Test\ntitle: %s acceptance\ndescription: How this feature is proven.\ntimestamp: %s\n---\n\n# Test Cases\n\n%s",
			strings.TrimSuffix(parts[1], ".md"), ts, strings.Join(cases, "\n"))
		p.stubs[test] = len(cases)
		p.aliases[stem+"/TEST.md"] = test
	}
}

// move is the plan as the link engine reads it: every file that moves, and
// every alias.
func (p *plan) move() links.Move {
	files := map[string]string{}
	for o, n := range p.moves {
		files[o] = n
	}
	for o, n := range p.aliases {
		files[o] = n
	}
	return links.Move{Files: files}
}

// repair plans the new text of every Markdown file: a lifted log gains the
// frontmatter a stem sibling needs, an index loses the status tags older
// tools wrote after its listings, and every link is repaired by the engine,
// from where its file was to where it is now. The vendored SPEC.md, which
// migrate replaces, is left to it, and a file that goes is left alone.
func (p *plan) repair() {
	mv := p.move()
	for _, f := range p.files {
		text, ok := p.texts0[f]
		if !ok || f == "SPEC.md" || p.goes(f) {
			continue
		}
		to := p.after(f)
		if strings.HasSuffix(to, ".log.md") && path.Base(f) != path.Base(to) {
			text = withLogFrontmatter(text, to)
		}
		if path.Base(to) == "INDEX.md" {
			var n int
			text, n = stripStatusTags(text)
			p.tags += n
		}
		text = repairLinks(text, links.Site{OldPath: f, NewPath: to}, mv)
		if text != p.texts0[f] || to != f {
			p.texts[to] = text
		}
	}
}

// repairLinks rewrites every link in text that the move changes, as written
// in the file at s.OldPath and now read from s.NewPath. A link in code is a
// sample and stays as it is.
func repairLinks(text string, s links.Site, m links.Move) string {
	var b strings.Builder
	last := 0
	for _, l := range links.Find(text) {
		if l.InCode {
			continue
		}
		if nt, ok := links.Retarget(l.Target, s, m); ok {
			b.WriteString(text[last:l.Start])
			b.WriteString(nt)
			last = l.End
		}
	}
	b.WriteString(text[last:])
	return b.String()
}

// withLogFrontmatter gives a lifted feature log the type: Log frontmatter a
// stem sibling needs: v0.2 and v0.3 read a feature directory's LOG.md as a
// reserved file, which needs none.
func withLogFrontmatter(text, to string) string {
	body := strings.TrimPrefix(text, "\uFEFF")
	if strings.HasPrefix(strings.TrimSpace(body), "---") {
		return text
	}
	title := strings.TrimSuffix(path.Base(to), ".log.md") + " feature log"
	out := fmt.Sprintf("---\ntype: Log\ntitle: %s\ndescription: Per-feature history.\ntimestamp: 2026-01-01T00:00:00Z\n---\n\n%s", title, body)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}

// stripStatusTags removes the status tag older tools put after an index
// listing (` (**draft**)`), outside code, and says how many it removed.
// Nothing kept it current, so it went stale as soon as the document moved on;
// a status lives only in its document.
func stripStatusTags(text string) (string, int) {
	lines := strings.Split(text, "\n")
	fence, removed := "", 0
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			fence = t[:3]
			continue
		}
		if m := statusTagRe.FindStringSubmatch(line); m != nil {
			lines[i] = m[1] + m[2]
			removed++
		}
	}
	return strings.Join(lines, "\n"), removed
}

// withPin returns the root INDEX.md's text pinned to version: its
// fdf_version line rewritten, or the pin added to the frontmatter the text
// has, or to a frontmatter block of its own. A line keeps its ending, and a
// line added ends as the text's lines do.
func withPin(text, version string) string {
	line := fmt.Sprintf(`fdf_version: "%s"`, version)
	if pinLineRe.MatchString(text) {
		return pinLineRe.ReplaceAllString(text, line)
	}
	eol := "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	}
	bom := ""
	if strings.HasPrefix(text, "\uFEFF") {
		bom, text = text[:3], text[3:]
	}
	lines := strings.SplitAfter(text, "\n")
	if strings.TrimRight(lines[0], "\r\n") == "---" {
		for _, l := range lines[1:] {
			if strings.TrimRight(l, "\r\n") == "---" {
				return bom + lines[0] + line + eol + strings.Join(lines[1:], "")
			}
		}
	}
	return bom + "---" + eol + line + eol + "---" + eol + eol + text
}

// crlf reports whether text ends every line with CRLF.
func crlf(text string) bool {
	return strings.Contains(text, "\r\n") && strings.Count(text, "\n") == strings.Count(text, "\r\n")
}

// withCRLF ends every line of text with CRLF.
func withCRLF(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
}

// withLogEntry returns the root LOG.md's text with entry added, newest first;
// a bundle without a LOG.md gets one.
func withLogEntry(text, entry string) string {
	if text == "" {
		text = "# Bundle Update Log\n"
	}
	return logs.Insert(text, logs.Entry(entry))
}

// apply makes the plan's changes to the bundle: each move, through a
// temporary name so that no move lands on a file another has yet to leave
// (and a rename that changes only case takes on a disk that ignores case);
// then each removal; then each text, written where its file now is. A
// directory a move leaves empty goes too.
func (p *plan) apply(out io.Writer) error {
	abs := func(rel string) string { return filepath.Join(p.root, filepath.FromSlash(rel)) }
	olds := sortedKeys(p.moves)
	for _, o := range olds {
		if err := os.Rename(abs(o), abs(o)+".migrating"); err != nil {
			return fmt.Errorf("moving %s: %w", o, err)
		}
	}
	for _, o := range olds {
		n := p.moves[o]
		if err := os.MkdirAll(filepath.Dir(abs(n)), 0o755); err != nil {
			return err
		}
		if err := os.Rename(abs(o)+".migrating", abs(n)); err != nil {
			return fmt.Errorf("moving %s -> %s: %w", o, n, err)
		}
		if path.Dir(o) == path.Dir(n) {
			fmt.Fprintf(out, "renamed %s -> %s\n", o, path.Base(n))
		} else {
			fmt.Fprintf(out, "moved %s -> %s\n", o, n)
		}
	}
	for _, g := range p.gone {
		if err := os.Remove(abs(g)); err != nil {
			return err
		}
		if g == "fdf-spec.md" {
			fmt.Fprintln(out, "removed vendored fdf-spec.md (the spec is vendored as SPEC.md now)")
		}
	}
	for _, rel := range sortedKeys(p.texts) {
		if err := os.MkdirAll(filepath.Dir(abs(rel)), 0o755); err != nil {
			return err
		}
		// A file written with CRLF line endings keeps them, on the lines
		// migrate adds too.
		text, f := p.texts[rel], p.source(rel)
		if f != "" && crlf(p.texts0[f]) {
			text = withCRLF(text)
		}
		if err := writeFile(abs(rel), text, f == ""); err != nil {
			return err
		}
		if n, ok := p.stubs[rel]; ok {
			fmt.Fprintf(out, "stubbed %s (%d scenario case(s))\n", rel, n)
		}
	}
	for _, o := range olds {
		for d := filepath.Dir(abs(o)); d != p.root && strings.HasPrefix(d, p.root); d = filepath.Dir(d) {
			if os.Remove(d) != nil {
				break // not empty
			}
		}
	}
	return nil
}

// writeFile writes text to the file at name, over the file there, or, when
// isNew, to a file that is not there: migrate never writes over a file its
// plan did not read, nor through a symbolic link, and stops instead.
func writeFile(name, text string, isNew bool) error {
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if isNew {
		flag = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	} else if fi, err := os.Lstat(name); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symbolic link, and migrate writes through none", name)
	}
	f, err := os.OpenFile(name, flag, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// lifted counts the plan's trail lifts: the moves that change a file's
// directory.
func (p *plan) lifted() int {
	n := 0
	for o, to := range p.moves {
		if path.Dir(o) != path.Dir(to) {
			n++
		}
	}
	return n
}

func relSlash(root, p string) string {
	r, _ := filepath.Rel(root, p)
	return filepath.ToSlash(r)
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
