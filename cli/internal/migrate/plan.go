package migrate

// A migration is worked out in full before anything is written: every file
// that moves, every file that goes, and the text of every file migrate
// writes. Nothing touches the bundle until the plan is complete, so a
// refusal leaves it as it was, and a dry run prints the plan and stops.

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/links"
	"github.com/GiteshDalal/fdf/cli/internal/logs"
	"github.com/GiteshDalal/fdf/cli/internal/refactor"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// reservedSince is the spec version from which each bundle-root directory is
// a register: releases/ under every pin, changes/ from 0.5, practices/ and
// debts/ from 0.6, and bugs/ from 0.7. Under an older pin the validator reads
// the same directory as a feature group, so migrate moves it into features/
// with the others.
var reservedSince = map[string]specver.Version{
	"releases": {}, "changes": {Minor: 5}, "practices": {Minor: 6}, "debts": {Minor: 6}, "bugs": {Minor: 7},
}

// reserved reports whether the bundle-root directory name is a register under
// pin, a 0.x version or "" for none.
func reserved(name, pin string) bool {
	since, ok := reservedSince[name]
	v, _ := specver.Parse(pin)
	return ok && v.AtLeast(since)
}

// plan is a migration worked out in full.
type plan struct {
	root string // the bundle root, absolute
	from string // the pin migrate found, "" for none

	// The link engine reads every path from base: the project root, or,
	// outside a git repository, the directory that holds the bundle. old and
	// new are the bundle's path from there, before and after it moves.
	base, old, new string
	project        string // the project root; "" outside a git repository
	submodule      bool   // the bundle is a git submodule, which git mv moves
	relocated      bool   // the bundle has moved: apply got that far

	// files are the bundle's files as they stand, by bundle-relative path,
	// hidden files and directories aside; texts0 holds each Markdown file's
	// text as it stands.
	files  []string
	texts0 map[string]string

	// symlinks are the bundle's symbolic links that name their target by a
	// relative path, by path -> that target; relinks are those a move would
	// break, by path once migrated -> the target they name then.
	symlinks, relinks map[string]string

	// The older layouts' steps, which make a 0.x bundle 0.7-shaped.
	moves   map[string]string // 0.1's case renames and 0.3's trail lift, by path before -> after
	gone    []string          // v0.1's vendored fdf-spec.md, which goes
	aliases map[string]string // for the link engine only: where a link to a file that does not move on points now
	stubs   map[string]int    // test documents written for features that need one, with their cases
	tags    int               // status tags removed from index listings

	// The 1.0 steps.
	groups  []string          // the root directories that move into features/
	isGroup map[string]bool   // the same, by name
	ids     map[string]string // each feature's ID before -> after

	// texts are the files migrate writes, by path once migrated: a file's new
	// text, or a new file. why says, for each, what changes in it.
	texts map[string]string
	why   map[string][]string

	links     int // links repaired, inside the bundle
	linkFiles int // files they are in
	mentions  int // feature ID mentions rewritten
	idFiles   int // documents they are in
	logIDs    int // feature ID mentions logs keep
	paths     int // mentions of the bundle's path rewritten, inside the bundle
	pathFiles int // documents they are in
	logPaths  int // mentions of the bundle's path logs keep
	listings  int // group listings moved from the root INDEX.md
	generated int // listings written for groups the root never listed

	// Outside the bundle (outside.go): each file rewritten, by its path from
	// the project root, and what changes in it; what is counted; and every
	// mention left as it is.
	outTexts                                             map[string]string
	outWhy                                               map[string][]string
	outLinks, outLinkFiles, outMentions, outMentionFiles int
	managed                                              int // mentions and links in what fdf install manages
	left                                                 []left

	// skip holds --skip's globs, and skipped the files outside the bundle
	// they name, by path from the project root: the outside pass leaves
	// them as they are, and lists what they say of the bundle.
	skip    []string
	skipped map[string]bool
}

// rootWrites are the files at the bundle root that migrate writes whatever
// they hold — INDEX.md, LOG.md and SPEC.md — and the names v0.1 gave them.
var rootWrites = map[string]bool{"INDEX.md": true, "LOG.md": true, "SPEC.md": true, "index.md": true, "log.md": true, "spec.md": true}

// newPlan reads the bundle at root, pinned to pin, and works out its
// migration to target, the bundle ending at dest, in the project at project
// ("" outside a git repository), leaving the files outside the bundle that
// skip's globs name as they are. problems are the reasons it cannot be
// migrated, found before anything is written.
func newPlan(root, pin, project, dest string, skip []string) (p *plan, problems []string, err error) {
	// A bundle that is a symbolic link is not where the link is, and the
	// plan reads the files where they are: migrate works on the directory
	// the link names.
	if to, err := os.Readlink(root); err == nil {
		return nil, []string{fmt.Sprintf("%s: a symbolic link to %s — migrate the directory it names, with --root", root, to)}, nil
	}
	p = &plan{root: root, from: pin, project: project, texts0: map[string]string{}, moves: map[string]string{},
		aliases: map[string]string{}, stubs: map[string]int{}, isGroup: map[string]bool{},
		ids: map[string]string{}, texts: map[string]string{}, why: map[string][]string{},
		symlinks: map[string]string{}, relinks: map[string]string{},
		outTexts: map[string]string{}, outWhy: map[string][]string{},
		skip: skip, skipped: map[string]bool{}}
	p.base = project
	if project == "" || project == root {
		p.base = filepath.Dir(root)
	}
	p.old, p.new = relSlash(p.base, root), relSlash(p.base, dest)
	p.submodule = project != "" && project != root && fdfroot.Submodule(root)
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
		// it into the file it names, and names again a relative target
		// that the move would change (relink). It writes the root's
		// INDEX.md, LOG.md and SPEC.md whatever they hold, so one of them
		// that is a link is refused.
		if d.Type()&os.ModeSymlink != 0 {
			to, _ := os.Readlink(q)
			if rootWrites[rel] {
				linked = append(linked, linkedFile(rel, to))
			}
			if to != "" && !filepath.IsAbs(to) {
				p.symlinks[rel] = filepath.ToSlash(to)
			}
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
	// structural work to be 0.7-shaped: 0.4 → 0.5 only adds changes/,
	// 0.5 → 0.6 only adds practices/, debts/ and DOMAIN.md, and 0.6 → 0.7
	// only adds bugs/. Running the pre-0.4 steps over one would be actively
	// wrong — pre-flight reads every `slug.spec.md` as an illegal dotted
	// basename.
	v, _ := specver.Parse(pin)
	stem := v.AtLeast(specver.Version{Minor: 4})
	if !stem {
		// Refuse content the v0.4 layout cannot hold.
		if problems := preflightV4(root); len(problems) > 0 {
			return nil, problems, nil
		}
		if err := p.caseRenames(); err != nil {
			return nil, nil, err
		}
		if err := p.liftTrails(); err != nil {
			return nil, nil, err
		}
	}
	p.findGroups()
	if problems := p.refusals(); len(problems) > 0 {
		return nil, problems, nil
	}
	if !stem {
		p.stubTests()
	}
	p.relink()
	p.repair()
	if err := p.indexes(); err != nil {
		return nil, nil, err
	}
	// A bundle that is its own repository has no outside, nor one outside a
	// git repository, which is not searched: --skip names nothing there.
	switch {
	case project != "" && project != root:
		if problems, err := p.skips(); err != nil || len(problems) > 0 {
			return nil, problems, err
		}
		if err := p.outside(); err != nil {
			return nil, nil, err
		}
	case len(skip) > 0 && project == "":
		return nil, []string{"--skip: the bundle is not in a git repository, so migrate reads no file outside it — run it without --skip"}, nil
	case len(skip) > 0:
		return nil, []string{"--skip: the bundle is its own git repository, so it has no outside — run it without --skip"}, nil
	}
	// Git must see every file where migrate puts it (relocate.go).
	if problems, err := p.ignored(); err != nil || len(problems) > 0 {
		return nil, problems, err
	}
	return p, nil, nil
}

// refusals are what 1.0 has no place for, and migrate cannot move for the
// bundle: a Markdown file at the root that is none of 1.0's own; a document
// named index.md or log.md, which a disk that ignores case reads as the
// INDEX.md or LOG.md beside it; a practice, debt or bug that shares its name
// with a directory beside it that holds Markdown; and a feature group whose
// place in features/ is taken, or a features at the root that is no
// directory. 0.7 read each of these, and 1.0 rejects it (F3), so each is
// fixed by hand before migrating. They are read once the older layouts'
// moves are made, and named where they are now. So is a directory of
// Markdown that is a symbolic link, wherever it is, which the plan does not
// read through: its documents would be left as they are. And so is a
// register migrate writes into that is a symbolic link, whatever it holds:
// its index would be written, and the feature groups moved, through it.
func (p *plan) refusals() []string {
	b := layout.New(os.DirFS(p.root))
	var out []string
	for _, f := range p.files {
		full := filepath.Join(p.root, filepath.FromSlash(f))
		if fi, err := os.Lstat(full); err != nil || fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		to, _ := os.Readlink(full)
		if layout.IsRegister(f) && f != "releases" {
			out = append(out, linkedRegister(f, to))
		} else if fi, err := os.Stat(full); err == nil && fi.IsDir() && holdsMarkdown(full) {
			out = append(out, fmt.Sprintf("%s: a symbolic link to %s, a directory of Markdown that migrate does not read through — replace it with the directory it names", f, to))
		}
	}
	gone := map[string]bool{}
	for _, g := range p.gone {
		gone[g] = true
	}
	for _, f := range p.files {
		shaped := p.after(f)
		switch {
		case !strings.HasSuffix(f, ".md") || gone[f]:
		case !strings.Contains(shaped, "/"):
			if pos := b.File(shaped); pos.Kind == layout.Stray {
				out = append(out, f+": "+pos.Problem)
			}
		case layout.CaseTwin(path.Base(shaped)) != "":
			out = append(out, f+": "+layout.CaseTwin(path.Base(shaped)))
		}
	}
	for _, reg := range []string{"practices", "debts", "bugs"} {
		if !reserved(reg, p.from) {
			continue
		}
		filepath.WalkDir(filepath.Join(p.root, reg), func(q string, d os.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			rel := relSlash(p.root, q)
			if _, err := os.Stat(q + ".md"); err == nil && b.HoldsMarkdown(rel) {
				if pos := b.Dir(rel); pos.Kind == layout.Stray {
					out = append(out, pos.Where+": "+pos.Problem)
					return filepath.SkipDir
				}
			}
			return nil
		})
	}
	if fi, err := os.Lstat(filepath.Join(p.root, "features")); err == nil && !fi.IsDir() && fi.Mode()&os.ModeSymlink == 0 {
		out = append(out, "features: not a directory, where the features/ register goes — move it out of the bundle root")
	}
	if !p.isGroup["features"] {
		for _, g := range p.groups {
			if _, err := os.Stat(filepath.Join(p.root, "features", g)); err == nil {
				out = append(out, fmt.Sprintf("features/%s: already there, where the feature group %s/ moves — move it out of features/", g, g))
			}
		}
	}
	sort.Strings(out)
	return out
}

// linkedFile and linkedRegister say why migrate refuses a symbolic link it
// would write through: a file it writes whatever it holds, and a register
// it writes into.
func linkedFile(rel, to string) string {
	return fmt.Sprintf("%s: a symbolic link to %s, which migrate would write through — replace it with the file it names", rel, to)
}

func linkedRegister(reg, to string) string {
	return fmt.Sprintf("%s: a symbolic link to %s, a register that migrate would write through — replace it with the directory it names", reg, to)
}

// holdsMarkdown reports whether the directory at dir, which may be a
// symbolic link, holds a Markdown file at any depth.
func holdsMarkdown(dir string) bool {
	found := false
	fs.WalkDir(os.DirFS(dir), ".", func(q string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(q, ".md") {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

// after is where the file at rel is once the older layouts' moves are made:
// its 0.7-shaped path.
func (p *plan) after(rel string) string {
	if to, ok := p.moves[rel]; ok {
		return to
	}
	return rel
}

// goes reports whether the file at rel goes, as v0.1's vendored fdf-spec.md
// does: migrate writes nothing in its place.
func (p *plan) goes(rel string) bool { return slices.Contains(p.gone, rel) }

// grouped is where a 0.7-shaped path is in 1.0: under features/ when its
// root directory is a feature group.
func (p *plan) grouped(rel string) string {
	if dir, _, ok := strings.Cut(rel, "/"); ok && p.isGroup[dir] {
		return "features/" + rel
	}
	return rel
}

// to is where the file at rel is once migrated.
func (p *plan) to(rel string) string { return p.grouped(p.after(rel)) }

// present is the set of the bundle's files once the moves made so far are
// made, by 0.7-shaped path.
func (p *plan) present() map[string]bool {
	out := map[string]bool{}
	for _, f := range p.files {
		out[p.after(f)] = true
	}
	return out
}

// text is the text of the file at rel once migrated: its planned text, or
// the text of the file that is there now, or "" for none.
func (p *plan) text(rel string) string {
	if t, ok := p.texts[rel]; ok {
		return t
	}
	for _, f := range p.files {
		if p.to(f) == rel {
			return p.texts0[f]
		}
	}
	return ""
}

// exists reports whether the bundle holds a file at rel once migrated.
func (p *plan) exists(rel string) bool {
	if _, ok := p.texts[rel]; ok {
		return true
	}
	for _, f := range p.files {
		if p.to(f) == rel {
			return true
		}
	}
	return false
}

// write plans text for the file at rel, once migrated, and says why.
func (p *plan) write(rel, text, why string) {
	p.texts[rel] = text
	if why != "" {
		p.why[rel] = append(p.why[rel], why)
	}
}

// source is the file of the bundle, as it stands, that is at rel once
// migrated, or "" when migrate writes rel new.
func (p *plan) source(rel string) string {
	for _, f := range p.files {
		if p.to(f) == rel {
			return f
		}
	}
	return ""
}

// caseRenames plans v0.1's renames: index.md, log.md, spec.md and plan.md are
// INDEX.md, LOG.md, SPEC.md and PLAN.md wherever they are. It refuses to
// rename a file onto another that is there, as a disk that reads case can
// hold both, and the rename would write over the other. Its vendored spec,
// fdf-spec.md, goes, and a link to it names the vendored SPEC.md.
func (p *plan) caseRenames() error {
	present := p.present()
	for _, f := range p.files {
		dir, base := path.Split(f)
		if to, ok := renames[base]; ok {
			if present[dir+to] {
				return fmt.Errorf("cannot move %s: %s already exists", f, dir+to)
			}
			p.moves[f] = dir + to
		}
	}
	for _, f := range p.files {
		if f == "fdf-spec.md" {
			p.gone = append(p.gone, f)
			p.aliases[f] = "SPEC.md"
		}
	}
	return nil
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
		to := p.grouped(test)
		p.write(to, fmt.Sprintf("---\ntype: Test\ntitle: %s acceptance\ndescription: How this feature is proven.\ntimestamp: %s\n---\n\n# Test Cases\n\n%s",
			strings.TrimSuffix(parts[1], ".md"), ts, strings.Join(cases, "\n")), "")
		p.stubs[to] = len(cases)
		p.aliases[stem+"/TEST.md"] = test
	}
}

// findGroups finds the feature groups, which move into features/: every root
// directory that is not hidden, holds Markdown, and is not a register under
// the bundle's pin. A directory that holds no Markdown stays where it is,
// outside FDF. A feature is a document directly in a group, and its ID gains
// features/.
func (p *plan) findGroups() {
	for _, f := range p.files {
		dir, _, ok := strings.Cut(f, "/")
		if ok && strings.HasSuffix(f, ".md") && !reserved(dir, p.from) && !p.isGroup[dir] {
			p.isGroup[dir] = true
			p.groups = append(p.groups, dir)
		}
	}
	sort.Strings(p.groups)
	for _, f := range p.files {
		s := p.after(f)
		parts := strings.Split(s, "/")
		name := strings.TrimSuffix(parts[len(parts)-1], ".md")
		if len(parts) == 2 && p.isGroup[parts[0]] && strings.HasSuffix(s, ".md") &&
			name != "INDEX" && name != "LOG" && !strings.Contains(name, ".") {
			id := strings.TrimSuffix(s, ".md")
			p.ids[id] = "features/" + id
		}
	}
}

// move is the plan as the link engine reads it, in paths from base: every
// file the older layouts move, every alias, every group, and the bundle
// itself, each to where it is once migrated.
func (p *plan) move() links.Move {
	m := links.Move{Files: map[string]string{}, Dirs: map[string]string{}}
	for o := range p.moves {
		m.Files[path.Join(p.old, o)] = path.Join(p.new, p.to(o))
	}
	for o, a := range p.aliases {
		m.Files[path.Join(p.old, o)] = path.Join(p.new, p.grouped(a))
	}
	for _, g := range p.groups {
		m.Dirs[path.Join(p.old, g)] = path.Join(p.new, "features", g)
	}
	if p.relocates() {
		m.Dirs[p.old] = p.new
	}
	return m
}

// relink plans the symbolic links a move would break: a link that names its
// target by a relative path names it again, from where the link is once
// migrated, at the place that target is once migrated.
func (p *plan) relink() {
	mv := p.move()
	for _, f := range sortedKeys(p.symlinks) {
		to := p.symlinks[f]
		now, _ := mv.New(path.Join(p.old, path.Dir(f), to))
		r, err := filepath.Rel(filepath.FromSlash(path.Join(p.new, path.Dir(p.to(f)))), filepath.FromSlash(now))
		if nt := filepath.ToSlash(r); err == nil && nt != path.Clean(to) {
			p.relinks[p.to(f)] = nt
		}
	}
}

// isLog reports whether rel is a log, whose words record what things were
// called then: LOG.md, or a <slug>.log.md.
func isLog(rel string) bool {
	return path.Base(rel) == "LOG.md" || strings.HasSuffix(rel, ".log.md")
}

// repair plans the new text of every Markdown file. A lifted log gains the
// frontmatter a stem sibling needs, and an index loses the status tags older
// tools wrote after its listings. Every link is repaired by the engine, from
// where its file was to where it is now. Outside logs, which keep their
// words, every mention of a feature's ID gains features/ — in fields,
// headings, prose, code spans and a Gherkin Scenario line, while the rest of
// a code block, such as a Gherkin step or a shell sample, keeps its words —
// and, in a git repository the bundle is part of, every mention of the
// bundle's path follows the move (as outside the bundle): a reference repair
// after a move, which reaches frozen documents too. The vendored SPEC.md,
// which migrate replaces, is left to it, and a file that goes is left alone.
func (p *plan) repair() {
	mv := p.move()
	for _, f := range p.files {
		text, ok := p.texts0[f]
		if !ok || f == "SPEC.md" || p.goes(f) {
			continue
		}
		shaped, to := p.after(f), p.to(f)
		if strings.HasSuffix(shaped, ".log.md") && path.Base(f) != path.Base(shaped) {
			text = withLogFrontmatter(text, shaped)
			p.why[to] = append(p.why[to], "log frontmatter")
		}
		if path.Base(shaped) == "INDEX.md" {
			var n int
			if text, n = stripStatusTags(text); n > 0 {
				p.tags += n
				p.why[to] = append(p.why[to], count(n, "status tag"))
			}
		}
		var n int
		site := links.Site{OldPath: path.Join(p.old, f), NewPath: path.Join(p.new, to), OldBase: p.old, NewBase: p.new}
		paths := p.project != "" && p.project != p.root
		if paths {
			// Read before the engine repairs them: a link it repairs leads
			// where it should. A log's is counted with its words.
			for _, l := range p.leadsElsewhere(text, site, mv) {
				if isLog(shaped) {
					p.logPaths++
				} else {
					p.left = append(p.left, left{path.Join(p.new, to), lineOf(text, l.Start), l.Target, elsewhere})
				}
			}
		}
		if text, n = repairLinks(text, site, mv); n > 0 {
			p.links += n
			p.linkFiles++
			p.why[to] = append(p.why[to], count(n, "link"))
		}
		skip := linkTargets(text)
		var reps []refactor.Replacement
		if paths {
			for _, m := range p.pathMentions(text, pathTargets(text), mv) {
				switch {
				case isLog(shaped):
					p.logPaths++
				case m.why != "":
					p.left = append(p.left, left{path.Join(p.new, to), lineOf(text, m.start), m.path, m.why})
				default:
					reps = append(reps, refactor.Replacement{Start: m.start, End: m.end, Text: m.to})
				}
				for k := m.start; k < m.end; k++ {
					skip[k] = true
				}
			}
			if n := len(reps); n > 0 {
				p.paths += n
				p.pathFiles++
				p.why[to] = append(p.why[to], count(n, "path mention"))
			}
		}
		// A scenario's name is matched by its test case and by the
		// declarations that name it, which are prose: its Scenario line is
		// rewritten with them, and the rest of a code block keeps its words.
		named := map[int]bool{}
		for _, loc := range scenarioLineRe.FindAllStringIndex(text, -1) {
			for k := loc[0]; k < loc[1]; k++ {
				named[k] = true
			}
		}
		for _, b := range links.Blocks(text) {
			for k := b.Start; k < b.End; k++ {
				if !named[k] {
					skip[k] = true
				}
			}
		}
		if isLog(shaped) {
			p.logIDs += len(refactor.IDMentions(text, p.ids, "", skip, true))
		} else {
			for k := range fieldPaths(text) {
				skip[k] = true
			}
			if ids := refactor.IDMentions(text, p.ids, "", skip, true); len(ids) > 0 {
				reps = append(reps, ids...)
				p.mentions += len(ids)
				p.idFiles++
				p.why[to] = append(p.why[to], count(len(ids), "ID"))
			}
		}
		text = replace(text, reps)
		if text != p.texts0[f] {
			p.texts[to] = text
		}
	}
	// A test stub names each scenario as its feature's Gherkin does, whose
	// Scenario lines gain features/ with every other copy of the name.
	for to := range p.stubs {
		text := p.texts[to]
		p.texts[to] = replace(text, refactor.IDMentions(text, p.ids, "", map[int]bool{}, true))
	}
}

// indexes plans the indexes 1.0 asks for. Each feature group's listing moves
// from the root INDEX.md to a new features/INDEX.md, description and all,
// as a listing moves with what it lists in fdf mv, and the root lists the
// Features register where the first of them stood. A group the root never
// listed gets the listing fdf new gives a new group, which links the
// group's directory when it has no index. The other registers get the
// indexes fdf init writes, and the root lists every register the bundle
// has: releases/ once it has an index, which fdf release writes. Then the
// pin, the vendored spec of the version it pins, and any Context stub
// missing.
func (p *plan) indexes() error {
	root := p.text("INDEX.md")
	featuresLine, _ := scaffold.RegisterLine("features")
	var kept, moved []string
	for _, line := range strings.Split(root, "\n") {
		if !p.listsGroup(scaffold.ListingTarget(line, ".")) {
			kept = append(kept, line)
			continue
		}
		if len(moved) == 0 {
			kept = append(kept, featuresLine)
		}
		// A listing moves without the CR of a CRLF line: features/INDEX.md
		// is new, and ends each line as scaffold's text does.
		moved = append(moved, reexpress(strings.TrimSuffix(line, "\r"), links.Site{OldPath: "INDEX.md", NewPath: "features/INDEX.md"}))
	}
	p.listings = len(moved)
	root = strings.Join(kept, "\n")
	features, _ := scaffold.IndexText("features")
	if len(moved) > 0 {
		features = strings.TrimRight(features, "\n") + "\n" + strings.Join(moved, "\n") + "\n"
	}
	for _, g := range p.groups {
		var added bool
		if features, added = scaffold.WithGroupListing(features, "features", g); added {
			p.generated++
			if !p.exists("features/" + g + "/INDEX.md") {
				features = strings.Replace(features, "(/features/"+g+"/INDEX.md)", "(/features/"+g+"/)", 1)
			}
		}
	}
	p.write("features/INDEX.md", features, "")
	for _, reg := range layout.Registers {
		if reg == "features" || reg == "releases" {
			continue
		}
		if idx := reg + "/INDEX.md"; !p.exists(idx) {
			body, _ := scaffold.IndexText(reg)
			p.write(idx, body, "")
		}
	}
	for _, reg := range layout.Registers {
		if reg == "releases" && !p.exists("releases/INDEX.md") {
			continue
		}
		root, _ = scaffold.WithRegisterListing(root, reg)
	}
	p.write("INDEX.md", withPin(root, target), "")
	if p.listings > 0 {
		p.why["INDEX.md"] = append(p.why["INDEX.md"], count(p.listings, "group listing")+" moved to features/INDEX.md")
	}
	p.why["INDEX.md"] = append(p.why["INDEX.md"], "pinned to "+target)
	for _, name := range layout.ContextDocs {
		if !p.exists(name) {
			stub, _ := scaffold.ContextStub(name)
			p.write(name, stub, "")
		}
	}
	doc, err := scaffold.SpecDoc(target)
	if err != nil {
		return err
	}
	p.write("SPEC.md", string(doc), "")
	return nil
}

// listsGroup reports whether a root listing's target, t, is a feature group
// once migrated: its INDEX.md, or its directory.
func (p *plan) listsGroup(t string) bool {
	rest, ok := strings.CutPrefix(t, "features/")
	g, tail, nested := strings.Cut(rest, "/")
	return ok && p.isGroup[g] && (!nested || tail == "INDEX.md")
}

// logEntry plans the migration's entry in the bundle-root log, where every
// bundle-wide event goes, with what it moved and repaired.
func (p *plan) logEntry(from string) {
	entry := fmt.Sprintf("**Migrated**: fdf_version %s → %s with `fdf migrate`", from, target)
	var did []string
	if p.relocates() {
		// From the project root, or the directory that holds the bundle: a
		// log records no machine's path.
		did = append(did, fmt.Sprintf("moved the bundle from `%s/` to `%s/`", p.old, p.new))
	}
	if len(p.groups) > 0 {
		did = append(did, fmt.Sprintf("moved %s into `features/` (%s)", p.groupList("`", "/`"), count(len(p.ids), "feature")))
	}
	if p.mentions > 0 || p.links > 0 {
		did = append(did, fmt.Sprintf("repaired %s in %s and %s in %s", count(p.mentions, "feature ID mention"), count(p.idFiles, "document"), count(p.links, "link"), count(p.linkFiles, "file")))
	}
	if n := len(p.outTexts); n > 0 {
		did = append(did, fmt.Sprintf("rewrote %s and %s in %s outside the bundle", count(p.outMentions, "mention"), count(p.outLinks, "link"), count(n, "file")))
	}
	if len(did) > 0 {
		entry += ": " + strings.Join(did, "; ")
	}
	entry += "."
	if p.tags > 0 {
		entry += fmt.Sprintf(" Removed the status tag from %d index listing(s); a document's status lives only in its frontmatter.", p.tags)
	}
	text := p.text("LOG.md")
	if text == "" {
		text = "# Bundle Update Log\n"
	}
	p.write("LOG.md", logs.Insert(text, logs.Entry(entry)), "the migration's entry")
}

// groupList names the groups, each between before and after.
func (p *plan) groupList(before, after string) string {
	names := make([]string, len(p.groups))
	for i, g := range p.groups {
		names[i] = before + g + after
	}
	return strings.Join(names, ", ")
}

// reexpress rewrites a line's links, as written in the file at s.OldPath, so
// that they name the same files from s.NewPath: a listing that moves from one
// index to another, while nothing it names moves.
func reexpress(line string, s links.Site) string {
	text, _ := repairLinks(line, s, links.Move{})
	return text
}

// repairLinks rewrites every link in text that the move changes, as written
// in the file at s.OldPath and now read from s.NewPath, and says how many it
// rewrote. A link in code is a sample and stays as it is.
func repairLinks(text string, s links.Site, m links.Move) (string, int) {
	var reps []refactor.Replacement
	for _, l := range links.Find(text) {
		if l.InCode {
			continue
		}
		if nt, ok := links.Retarget(l.Target, s, m); ok {
			reps = append(reps, refactor.Replacement{Start: l.Start, End: l.End, Text: nt})
		}
	}
	return replace(text, reps), len(reps)
}

// linkTargets marks the bytes of every link target in text, which an ID
// mention never overlaps: the engine repairs them, and a URL keeps its words.
func linkTargets(text string) map[int]bool {
	out := map[int]bool{}
	for _, l := range links.Find(text) {
		for i := l.Start; i < l.End; i++ {
			out[i] = true
		}
	}
	return out
}

// fieldPaths marks the bytes of the resource and applies-to values in text's
// frontmatter, inline or as a list: paths in the project, which name code,
// never a feature.
func fieldPaths(text string) map[int]bool {
	out := map[int]bool{}
	if !strings.HasPrefix(text, "---") {
		return out
	}
	in, pos := false, 0
	for i, line := range strings.SplitAfter(text, "\n") {
		start := pos
		pos += len(line)
		t := strings.TrimSpace(line)
		switch {
		case i > 0 && t == "---":
			return out
		case strings.HasPrefix(line, "resource:") || strings.HasPrefix(line, "applies-to:"):
			in = true
		case in && strings.HasPrefix(t, "-"):
			// an item of the field's list
		default:
			in = false
		}
		if in {
			for k := start; k < pos; k++ {
				out[k] = true
			}
		}
	}
	return out
}

// replace applies non-overlapping replacements to text, in any order; of two
// that overlap, the earlier one wins.
func replace(text string, reps []refactor.Replacement) string {
	sort.Slice(reps, func(i, j int) bool { return reps[i].Start < reps[j].Start })
	var b strings.Builder
	last := 0
	for _, r := range reps {
		if r.Start < last {
			continue
		}
		b.WriteString(text[last:r.Start])
		b.WriteString(r.Text)
		last = r.End
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

// apply makes the plan's changes to the bundle. First each move the older
// layouts make, through a temporary name so that no move lands on a file
// another has yet to leave (and a rename that changes only case takes on a
// disk that ignores case), and the directories those moves leave empty go.
// Then each removal. Then each feature group moves into features/, through a
// hidden directory, since a group may be called features. Then each text is
// written where its file now is, and the bundle moves to its destination.
// Last, the files outside the bundle that name it are rewritten.
func (p *plan) apply() error {
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
	}
	for _, o := range olds {
		for d := filepath.Dir(abs(o)); d != p.root && strings.HasPrefix(d, p.root); d = filepath.Dir(d) {
			if os.Remove(d) != nil {
				break // not empty
			}
		}
	}
	for _, g := range p.gone {
		if err := os.Remove(abs(g)); err != nil {
			return err
		}
	}
	if len(p.groups) > 0 {
		tmp := abs(".fdf-migrate")
		if err := os.Mkdir(tmp, 0o755); err != nil {
			return err
		}
		for _, g := range p.groups {
			if err := os.Rename(abs(g), filepath.Join(tmp, g)); err != nil {
				return fmt.Errorf("moving %s/: %w", g, err)
			}
		}
		if _, err := os.Stat(abs("features")); err != nil {
			if err := os.Rename(tmp, abs("features")); err != nil {
				return fmt.Errorf("moving the feature groups into features/: %w", err)
			}
		} else {
			// features/ is there, holding no Markdown: the groups go into it.
			for _, g := range p.groups {
				if err := os.Rename(filepath.Join(tmp, g), abs("features/"+g)); err != nil {
					return fmt.Errorf("moving %s/ into features/: %w", g, err)
				}
			}
			if err := os.Remove(tmp); err != nil {
				return err
			}
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
	}
	for _, rel := range sortedKeys(p.relinks) {
		if err := os.Remove(abs(rel)); err != nil {
			return err
		}
		if err := os.Symlink(filepath.FromSlash(p.relinks[rel]), abs(rel)); err != nil {
			return err
		}
	}
	if err := p.relocate(); err != nil {
		return err
	}
	return p.writeOutside()
}

// shown is a path from base as a person reads it: from the project root, or,
// outside a git repository, from the file system's root.
func (p *plan) shown(rel string) string {
	if p.project != "" {
		return rel
	}
	return filepath.ToSlash(filepath.Join(p.base, filepath.FromSlash(rel)))
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

// count says n of what, as "1 link" or "3 links".
func count(n int, what string) string {
	if n == 1 {
		return "1 " + what
	}
	return fmt.Sprintf("%d %ss", n, what)
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
