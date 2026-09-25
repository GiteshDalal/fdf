// Package layout knows where everything goes in an FDF 1.0 bundle: what the
// root holds, what each directory inside a register is, and so the position
// and ID of every Markdown file. The validator asks it, and so will every
// command that writes to a bundle, so that none of them keeps its own copy of
// these rules.
//
// The rules, from spec 1.0:
//   - The root is closed. Its Markdown files are INDEX.md, LOG.md, SPEC.md,
//     README.md and the five Context documents, and its directories that hold
//     Markdown are the six registers.
//   - releases/ is flat. Every other register files its documents flat or in
//     groups, and groups nest to any depth.
//   - A directory beside a document of the same name belongs to it: beside a
//     feature, Change or Fix it is the task directory, which holds only
//     NN-<name>.md tasks; a practice, debt or bug owns no directory. Every
//     other directory in a register is a group.
//   - A directory that holds no Markdown, such as one of images, is outside
//     FDF wherever it is, as a file that is not Markdown is.
//   - No document is named index.md or log.md, which a disk that ignores
//     case reads as the INDEX.md or LOG.md beside it.
package layout

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

// Registers are the directories a bundle root may hold, in the order the root
// INDEX.md lists them.
var Registers = []string{"features", "changes", "practices", "debts", "bugs", "releases"}

// ContextDocs are the five Context documents at the bundle root.
var ContextDocs = []string{"STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md", "DOMAIN.md"}

// IsRegister reports whether name is one of the registers.
func IsRegister(name string) bool { return in(Registers, name) }

// IsContext reports whether name is one of the Context documents.
func IsContext(name string) bool { return in(ContextDocs, name) }

// Kind is what sits at a position.
type Kind string

const (
	Index     Kind = "index"          // INDEX.md: the root's, a register's or a group's
	Log       Kind = "log"            // LOG.md: the root's, a register's or a group's
	Readme    Kind = "readme"         // README.md at the root, which FDF ignores
	Reference Kind = "reference"      // SPEC.md at the root, the vendored specification
	Context   Kind = "context"        // one of the five Context documents
	Document  Kind = "document"       // a register's document
	Trail     Kind = "trail"          // <slug>.<role>.md beside the document it belongs to
	Task      Kind = "task"           // NN-<name>.md in a task directory
	Register  Kind = "register"       // a register directory
	Group     Kind = "group"          // a directory that files a register's documents
	TaskDir   Kind = "task directory" // the directory beside a feature, Change or Fix
	Stray     Kind = "stray"          // a path with no position in a 1.0 bundle
)

// Position is where a path sits in a bundle.
type Position struct {
	Kind     Kind
	Register string // the register the path is in; "" at the root
	// ID is a Document's ID, the ID of the document a Trail, Task or TaskDir
	// belongs to, and the path of a Register or Group.
	ID      string
	Role    string // a Trail's role: spec, plan, test, surface or log
	Where   string // a Stray's path; a directory's ends in "/"
	Problem string // a Stray's problem, and where it belongs
}

var (
	nameRe  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	fileRe  = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*\.md$`)
	trailRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)\.(.+)\.md$`)
	taskRe  = regexp.MustCompile(`^\d{2}-[a-z0-9][a-z0-9-]*\.md$`)
)

// roles are the trail roles each register's documents take.
var roles = map[string][]string{
	"features":  {"spec", "plan", "test", "surface", "log"},
	"changes":   {"spec", "plan", "log"},
	"practices": {"log"},
	"debts":     {"log"},
	"bugs":      {"log"},
}

// ownsTasks: a feature, Change or Fix owns a task directory; a practice, debt
// or bug owns no directory at all.
var ownsTasks = map[string]bool{"features": true, "changes": true}

// nouns name what each register files.
var nouns = map[string]string{"features": "feature", "changes": "Change or Fix", "practices": "practice", "debts": "debt", "bugs": "bug"}

// caseTwins are the file names no document takes, each with the reserved
// file a disk that ignores case reads it as: on macOS and Windows, by
// default, index.md and INDEX.md in one directory are one file.
var caseTwins = map[string]string{"index.md": "INDEX.md", "log.md": "LOG.md"}

// CaseTwin says why no document takes the file name name, when a disk that
// ignores case reads it as the reserved file beside it — index.md as
// INDEX.md — and returns "" for any other name.
func CaseTwin(name string) string {
	twin := caseTwins[name]
	if twin == "" {
		return ""
	}
	return fmt.Sprintf("no document is named %s: a disk that ignores case reads it as the %s beside it — rename it", name, twin)
}

const (
	rootFileProblem = "the bundle root holds only INDEX.md, LOG.md, SPEC.md, README.md and the five Context documents — file it in a register, or move it out of the bundle"
	rootDirProblem  = "the bundle root holds only the registers features/, changes/, practices/, debts/, bugs/ and releases/ — a feature group belongs under features/; file anything else in its register, or move it out of the bundle"
	taskDirProblem  = "task directories may contain only NN-slug.md tasks"
)

// Bundle answers position questions about one bundle. Its file system is
// rooted at the bundle root. A Bundle reads each directory once, the first
// time a question needs it, and keeps what it read: it is a snapshot, so a
// caller that changes the bundle makes a new one to see the change.
type Bundle struct {
	fsys     fs.FS
	names    map[string]map[string]bool // directory -> the names it holds, read once
	markdown map[string]bool            // directory -> whether it holds Markdown, at any depth
}

// New returns the Bundle whose root is the root of fsys.
func New(fsys fs.FS) *Bundle {
	return &Bundle{fsys: fsys, names: map[string]map[string]bool{}, markdown: map[string]bool{}}
}

// has reports whether dir holds an entry called name, spelled exactly so: a
// case-insensitive file system must not turn a group called index/ into the
// task directory of INDEX.md.
func (b *Bundle) has(dir, name string) bool {
	names, ok := b.names[dir]
	if !ok {
		names = map[string]bool{}
		entries, _ := fs.ReadDir(b.fsys, path.Clean("./"+dir))
		for _, e := range entries {
			names[e.Name()] = true
		}
		b.names[dir] = names
	}
	return names[name]
}

// spelled returns the name dir holds that differs from name only in case,
// when dir holds nothing called name itself: on a disk that ignores case, a
// path through name opens that entry.
func (b *Bundle) spelled(dir, name string) string {
	if b.has(dir, name) {
		return ""
	}
	other := ""
	for n := range b.names[dir] {
		if strings.EqualFold(n, name) && (other == "" || n < other) {
			other = n
		}
	}
	return other
}

// HoldsMarkdown reports whether the directory at rel holds a Markdown file at
// any depth, hidden files and directories aside. One that holds none is
// outside FDF: no rule reads its position or its name.
func (b *Bundle) HoldsMarkdown(rel string) bool {
	if held, ok := b.markdown[rel]; ok {
		return held
	}
	top := path.Clean("./" + rel)
	held := false
	fs.WalkDir(b.fsys, top, func(p string, e fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case p != top && strings.HasPrefix(e.Name(), "."):
			if e.IsDir() {
				return fs.SkipDir
			}
		case !e.IsDir() && strings.HasSuffix(e.Name(), ".md"):
			held = true
			return fs.SkipAll
		}
		return nil
	})
	b.markdown[rel] = held
	return held
}

// Exists reports whether the bundle holds a file or directory at rel, a
// slash-separated path from the bundle root, its name spelled exactly so:
// on a disk that ignores case, os.Stat finds features/INDEX.md when asked
// for features/index.md, and Exists does not.
func (b *Bundle) Exists(rel string) bool {
	dir, name := path.Split(rel)
	return b.has(strings.TrimSuffix(dir, "/"), name)
}

// Dir returns the position of the directory at rel, a slash-separated path
// from the bundle root.
func (b *Bundle) Dir(rel string) Position {
	parts := strings.Split(rel, "/")
	reg := parts[0]
	switch {
	case !IsRegister(reg):
		return stray(reg+"/", rootDirProblem)
	case len(parts) == 1:
		return Position{Kind: Register, Register: reg, ID: reg}
	case reg == "releases":
		return stray("releases/"+parts[1]+"/", "releases/ is flat: a release is releases/<version>.md")
	}
	dir := reg
	for i := 1; i < len(parts); i++ {
		sub := dir + "/" + parts[i]
		if !nameRe.MatchString(parts[i]) {
			return stray(sub+"/", "directory names must be lowercase [a-z0-9-]")
		}
		if b.has(dir, parts[i]+".md") {
			switch {
			case !ownsTasks[reg]:
				return stray(sub+"/", fmt.Sprintf("shares its name with the %s %s.md, and a %s owns no directory — rename one of them", nouns[reg], sub, nouns[reg]))
			case i < len(parts)-1:
				return stray(sub+"/"+parts[i+1]+"/", taskDirProblem+", and no directory")
			}
			return Position{Kind: TaskDir, Register: reg, ID: sub}
		}
		dir = sub
	}
	return Position{Kind: Group, Register: reg, ID: dir}
}

// File returns the position of the Markdown file at rel, a slash-separated
// path from the bundle root.
func (b *Bundle) File(rel string) Position {
	dir, name := path.Split(rel)
	if dir == "" {
		return rootFile(name)
	}
	d := b.Dir(strings.TrimSuffix(dir, "/"))
	switch d.Kind {
	case Stray:
		return d
	case TaskDir:
		if taskRe.MatchString(name) {
			return Position{Kind: Task, Register: d.Register, ID: d.ID}
		}
		return stray(rel, taskDirProblem)
	}
	switch {
	case name == "INDEX.md":
		return Position{Kind: Index, Register: d.Register, ID: d.ID}
	case name == "LOG.md":
		return Position{Kind: Log, Register: d.Register, ID: d.ID}
	case CaseTwin(name) != "":
		return stray(rel, CaseTwin(name))
	case !fileRe.MatchString(name):
		return stray(rel, "filenames are lowercase; uppercase is reserved for INDEX/LOG/SPEC/STACK/ARCHITECTURE/SURFACES/INFRA/DOMAIN.md")
	case d.Register == "releases":
		return Position{Kind: Document, Register: d.Register, ID: strings.TrimSuffix(rel, ".md")}
	}
	if m := trailRe.FindStringSubmatch(name); m != nil {
		if !in(roles[d.Register], m[2]) {
			return stray(rel, unknownRole(d.Register, m[2]))
		}
		return Position{Kind: Trail, Register: d.Register, ID: d.ID + "/" + m[1], Role: m[2]}
	}
	return Position{Kind: Document, Register: d.Register, ID: strings.TrimSuffix(rel, ".md")}
}

// Place says why a new document cannot be filed at id, a full ID such as
// "features/payments/instant-refunds", in the bundle as it stands, or
// returns "". The ID starts with a register that files documents (every one
// but releases/), and its other parts are lowercase [a-z0-9-] names; the
// last, its slug, is not index or log. Nothing may stand in its place: no
// document of that name, and no directory of that name that holds Markdown,
// which the new document would own. Nor may a directory on its way belong
// to a document: a task directory holds only tasks, and a practice, debt or
// bug owns no directory. And each directory on its way that is there is
// spelled as the ID spells it: on a disk that ignores case, one spelled
// otherwise is where the document would land, under a name F3 rejects.
func (b *Bundle) Place(id string) string {
	parts := strings.Split(id, "/")
	reg := parts[0]
	if !IsRegister(reg) || reg == "releases" || len(parts) < 2 {
		return fmt.Sprintf("%s does not name a place in a register: a document's ID is <register>/[<group>/…]<slug>", id)
	}
	for _, p := range parts[1:] {
		if !nameRe.MatchString(p) {
			return fmt.Sprintf("%q in %s is not a name: names are lowercase [a-z0-9-]", p, id)
		}
	}
	dir, name := path.Dir(id), path.Base(id)
	if twin := caseTwins[name+".md"]; twin != "" {
		return fmt.Sprintf("%s is not a slug: a disk that ignores case reads %s.md as the %s beside it (F3); choose another name", name, name, twin)
	}
	for i := 0; i < len(parts)-1; i++ {
		parent := path.Join(parts[:i]...)
		if other := b.spelled(parent, parts[i]); other != "" {
			return fmt.Sprintf("%s/ is already there: a disk that ignores case would file %s in it, and directory names are lowercase (F3); rename that directory, or choose another name", path.Join(parent, other), id)
		}
	}
	if dir != reg {
		switch d := b.Dir(dir); d.Kind {
		case TaskDir:
			return fmt.Sprintf("%s/ is the task directory of %s and holds only its NN-slug.md tasks (F3); file this in a group of another name", d.ID, d.ID)
		case Stray:
			return fmt.Sprintf("%s: %s (F3)", d.Where, d.Problem)
		}
	}
	switch {
	case b.has(dir, name+".md"):
		return id + ".md already exists"
	case !b.has(dir, name) || !b.HoldsMarkdown(id):
		return ""
	case ownsTasks[reg]:
		return fmt.Sprintf("%s/ is a group, and a %s named %s would make it its task directory (F3); choose another name", id, nouns[reg], name)
	}
	return fmt.Sprintf("%s/ is a group, and a %s named %s cannot sit beside it: a %s owns no directory (F3); choose another name", id, nouns[reg], name, nouns[reg])
}

// rootFile is the position of a Markdown file at the bundle root.
func rootFile(name string) Position {
	switch {
	case name == "INDEX.md":
		return Position{Kind: Index}
	case name == "LOG.md":
		return Position{Kind: Log}
	case name == "README.md":
		return Position{Kind: Readme}
	case name == "SPEC.md":
		return Position{Kind: Reference}
	case IsContext(name):
		return Position{Kind: Context}
	}
	return stray(name, rootFileProblem)
}

// unknownRole says which roles a register's documents take instead.
func unknownRole(reg, role string) string {
	switch reg {
	case "features":
		return fmt.Sprintf("unknown trail role %q — allowed roles are spec, plan, test, surface, log", role)
	case "changes":
		return fmt.Sprintf("unknown trail role %q under changes/ — allowed roles are spec, plan, log (test and surface belong to the affected feature)", role)
	case "practices":
		return fmt.Sprintf("unknown trail role %q under practices/ — a practice owns no spec, plan, test or surface; `log` is the only role", role)
	}
	return fmt.Sprintf("unknown trail role %q under %s/ — a %s is a register entry, not a unit of work; `log` is the only role", role, reg, nouns[reg])
}

func stray(where, problem string) Position {
	return Position{Kind: Stray, Where: where, Problem: problem}
}

func in(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
