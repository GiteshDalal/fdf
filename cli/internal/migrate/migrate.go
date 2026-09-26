// Package migrate upgrades a 0.x bundle, at any pin from 0.1 to 0.7 or none,
// to spec 1.0 in one run:
//
//	older layouts → 0.7-shaped: v0.1's case renames and the removal of its
//	      vendored spec, v0.3's trail lift and the test stubs, and the status
//	      tags older tools wrote after index listings
//	1.0 → every feature group moves into features/, and every mention of a
//	      feature's ID gains features/ (logs keep their words); the one link
//	      engine repairs every link; features/INDEX.md takes the groups'
//	      listings, the other registers get their indexes; the pin, the spec
//	      copy and any missing Context stub; the log
//	then → a bundle at …/docs/features moves to …/docs/fdf beside it, or
//	      where --to says, a submodule with git mv (relocate.go); the rest of
//	      the project's git-tracked text files follow it, links into it and
//	      mentions of its path (outside.go); then validation
//
// The whole migration is worked out first (plan.go) and printed; a dry run
// stops there, and nothing is written until the plan is complete, so a
// bundle migrate refuses is left as it was. In a git repository, which is
// its undo, migrate starts only from a clean tree, puts no file where git
// would ignore it, marks what it wrote with git add -N so that git diff -M
// shows every move, and prints the git commands that put everything back,
// should it stop partway, or, once done, that back it out, after a git
// reset of those marks. A bundle already pinned to 1.0 moves nothing:
// migrate restores its spec copy, indexes and Context stubs. A root whose
// INDEX.md pins nothing inside a pinned bundle — a register or a group of
// it, where the steps would build a second bundle — is refused, and so is a
// pin that is not a version, or one newer than this fdf knows.
//
// Ends by validating the result with FreshStubsAdvisory so unfilled Context
// stubs do not fail the migration (plain `fdf validate` will still enforce F9).
package migrate

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/refactor"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// target is the pin migrate writes: 1.0.
const target = "1.0"

// known0x are the pins migrate upgrades from: every 0.x version FDF had.
var known0x = map[string]bool{"0.1": true, "0.2": true, "0.3": true, "0.4": true, "0.5": true, "0.6": true, "0.7": true}

// Version is the CLI version, set by the command wrapper. A migrate that
// finds the pin already current is indistinguishable from a migrate that has
// nothing to do — unless the message names the binary doing the looking. An
// old fdf held in place by a version shim reports "already current" about a
// spec several versions behind, which reads as the command being broken.
var Version string

func binaryName() string {
	if Version == "" {
		return "this binary"
	}
	return "fdf " + Version
}

// renames: v0.1 lowercase reserved basenames → uppercase.
var renames = map[string]string{"index.md": "INDEX.md", "log.md": "LOG.md", "spec.md": "SPEC.md", "plan.md": "PLAN.md"}

// trailBasenames: nested trail files under group/slug/ → stem role suffix.
var trailBasenames = map[string]string{
	"SPEC.md": "spec",
	"PLAN.md": "plan",
	"TEST.md": "test",
	"LOG.md":  "log",
}

var pinLineRe = regexp.MustCompile(`(?m)^fdf_version:[^\r\n]*`)

// pinValueRe tolerates unquoted pins (`fdf_version: 0.4`) and single-quoted
// ones (`fdf_version: '1.0'`): the validator's YAML-based readPin accepts
// them, and migrate must agree with the validator about what version a
// bundle pins.
var pinValueRe = regexp.MustCompile(`fdf_version:\s*["']?([^"'\s]+)["']?`)
var taskFileRe = regexp.MustCompile(`^\d{2}-[a-z0-9][a-z0-9-]*\.md$`)
var statusRe = regexp.MustCompile(`(?m)^status:\s*(\S+)`)
var scenarioRe = regexp.MustCompile(`(?m)^\s*Scenario(?: Outline)?:\s*(\S[^\n]*)`)
var scenarioLineRe = regexp.MustCompile(`(?m)^[ \t]*Scenario(?: Outline)?:[^\n]*`)
var timestampRe = regexp.MustCompile(`(?m)^timestamp:\s*(\S+)`)

// Options are how fdf migrate was asked to run.
type Options struct {
	Root    string // the bundle root
	Project string // the project root: git's, which R1 checks paths against; "" outside a git repository
	DryRun  bool   // print the plan and change nothing
	To      string // where the bundle goes, absolute; "" for the default
	EnvRoot string // the bundle root FDF_ROOT_DIR names, absolute; "" when it is not set
	// Skip holds globs, read from the project root as git reads a pathspec
	// with :(glob) magic, naming files outside the bundle that the outside
	// pass leaves as they are, listing what they say of the bundle.
	Skip []string
}

// Run upgrades the bundle at o.Root to target: it works out the whole
// migration, prints it, and, unless o.DryRun, applies it and validates the
// result.
func Run(o Options, out io.Writer) int {
	root := o.Root
	// Nothing to migrate without a bundle: an INDEX.md, or the lowercase
	// index.md of a v0.1 bundle, which the migration renames.
	if !exists(filepath.Join(root, "INDEX.md")) && !exists(filepath.Join(root, "index.md")) {
		fmt.Fprintln(out, "error:", fdfroot.NoBundle(root))
		return 1
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		rootAbs = root
	}
	rootAbs = onDisk(rootAbs)

	// A bundle pinned to 0.x, or to nothing, is migrated; one at target takes
	// the repair path.
	pin := readPin(root)
	switch v, ok := specver.Parse(pin); {
	case pin == "":
		// A register's or a group's INDEX.md pins nothing: the steps would
		// build a second bundle inside the pinned one.
		if bundle := fdfroot.BundleAbove(root); bundle != "" {
			fmt.Fprintln(out, "error:", fdfroot.InsideBundle(root, bundle))
			return 1
		}
	case !ok:
		fmt.Fprintf(out, "cannot migrate: the bundle pins fdf_version %s, which is not a MAJOR.MINOR version such as %s — correct the pin in INDEX.md; the bundle was left as it is.\n", pin, scaffold.CurrentVersion())
		return 1
	case pin == target && o.To != "" && onDisk(filepath.Clean(o.To)) != rootAbs:
		fmt.Fprintf(out, "cannot migrate: the bundle already pins fdf_version %s, and migrate moves nothing in a bundle at %s — move it with git mv, then point --root or FDF_ROOT_DIR at it; the bundle was left as it is.\n", target, target)
		return 1
	case pin == target && len(o.Skip) > 0:
		fmt.Fprintf(out, "cannot migrate: the bundle already pins fdf_version %s, and migrate reads no file outside a bundle at %s — run it without --skip; the bundle was left as it is.\n", target, target)
		return 1
	case pin == target:
		return repair(o, rootAbs, out)
	case v.Major == 0 && !known0x[pin]:
		fmt.Fprintf(out, "cannot migrate: the bundle pins fdf_version %s, which is no 0.x version %s knows (0.1 to 0.7) — correct the pin in INDEX.md; the bundle was left as it is.\n", pin, binaryName())
		return 1
	case v.Major != 0:
		fmt.Fprintf(out, "cannot migrate: the bundle pins fdf_version %s, newer than any spec %s knows (%s) — upgrade fdf; the bundle was left as it is.\n", pin, binaryName(), target)
		return 1
	}

	// Git is the migration's undo, in the repository that tracks the bundle.
	project := ""
	if o.Project != "" {
		project = repository(rootAbs)
	}
	dest, problem := destination(o.To, rootAbs, project)
	if problem != "" {
		fmt.Fprintf(out, "cannot migrate: %s; the bundle was left as it is.\n", problem)
		return 1
	}
	p, problems, err := newPlan(rootAbs, pin, project, dest, o.Skip)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}
	if len(problems) > 0 {
		fmt.Fprintln(out, "cannot migrate — fix these first (bundle left unchanged):")
		for _, problem := range problems {
			fmt.Fprintln(out, "  "+problem)
		}
		return 1
	}
	// Git is the migration's undo: it starts from a clean tree.
	if project != "" {
		lines, err := p.dirty(project)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}
		if len(lines) > 0 {
			fmt.Fprintln(out, "cannot migrate: files migrate would change have changes not committed, or are files git does not track — commit them, stash them or move them out of the bundle first, so that git can show the migration and undo it (bundle left unchanged):")
			for i, l := range lines {
				if i == 10 {
					fmt.Fprintf(out, "  …and %d more\n", len(lines)-10)
					break
				}
				fmt.Fprintln(out, "  "+l)
			}
			return 1
		}
	}
	from := pin
	if from == "" {
		from = "unpinned"
	}
	p.logEntry(from)
	p.print(out, from, o.DryRun)
	if o.DryRun {
		return 0
	}
	if err := p.apply(); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		fmt.Fprintln(out, "the migration stopped partway. To put everything back as it was:")
		for _, l := range p.undo(project) {
			fmt.Fprintln(out, "  "+l)
		}
		return 1
	}
	if project != "" {
		if err := p.markNew(project); err != nil {
			fmt.Fprintf(out, "warning: %v — mark the new files with `git add -N` yourself, so that `git diff -M` shows each move\n", err)
		}
	}
	root = p.dest()
	if p.relocates() {
		fmt.Fprintf(out, "\ndone: migrated the bundle at %s to fdf_version %s, and moved it to %s; logged in LOG.md.\n", rootAbs, target, root)
	} else {
		fmt.Fprintf(out, "\ndone: migrated the bundle at %s to fdf_version %s; logged in LOG.md.\n", rootAbs, target)
	}

	// Freshly scaffolded Context stubs are advisory here — migration
	// succeeded; filling them is the human's next step via fdf-init.
	fmt.Fprintln(out, "\nvalidating migrated bundle:")
	var report bytes.Buffer
	code := bundle.Validate(root, bundle.Options{RepoRoot: o.Project, Out: io.MultiWriter(out, &report), FreshStubsAdvisory: true})
	// Say so only when validation found a stub, and name the ones it found:
	// F9 fails a plain validate only while one is unfilled and the bundle has
	// a feature.
	if stubs := stubsIn(report.String()); code == 0 && len(stubs) > 0 {
		fmt.Fprintln(out, "\nnext: run the fdf-init skill to fill "+joinNames(stubs)+".")
		if freshStubRe.MatchString(report.String()) {
			fmt.Fprintln(out, "warning: the next plain `fdf validate` will fail F9 until those stubs are filled (migrate reports an unfilled stub as a warning, not an error).")
		}
	}
	// 0.7's checks reach further than older versions did: a bundle from
	// before 0.7 hears what they found.
	if v, _ := specver.Parse(pin); v.Less(specver.Version{Minor: 7}) {
		reportV07(root, report.String(), out)
	}
	fmt.Fprintln(out, "\nnext: re-run `fdf install` as you installed fdf: the installed skills and primer still describe 0.7.")
	switch {
	case project == "":
		fmt.Fprintln(out, "      then review the migration: the bundle is not in a git repository, so nothing can undo it.")
	case p.submodule:
		fmt.Fprintf(out, "      then review it inside the submodule, with `git -C %s diff -M`, and commit it there first;\n", root)
		fmt.Fprintln(out, "      then commit the submodule's new commit here, with .gitmodules when it moved.")
	default:
		fmt.Fprintln(out, "      then review it with `git diff -M` — migrate marked the files it wrote with `git add -N`,")
		fmt.Fprintln(out, "      so each move shows as a rename — and commit it.")
	}
	if o.EnvRoot != "" && onDisk(o.EnvRoot) == rootAbs && p.relocates() {
		fmt.Fprintf(out, "      FDF_ROOT_DIR still names %s: point it at %s.\n", rootAbs, root)
	}
	if project != "" {
		fmt.Fprintln(out, "      to back the migration out instead, run these commands; the first takes back the")
		fmt.Fprintln(out, "      marks of `git add -N`, on which git stash and git clean would trip:")
		for _, l := range p.backOut(project) {
			fmt.Fprintln(out, "        "+l)
		}
	}
	return code
}

// repair takes a bundle already at target, in which nothing moves: it
// restores the vendored spec when it is missing or not target's, each
// register's index but releases/', and each missing Context stub, never
// through a symbolic link, then validates the bundle with the same stub
// leniency as a migration — so running migrate twice in a row cannot flip
// from success to failure.
func repair(o Options, root string, out io.Writer) int {
	fmt.Fprintf(out, "nothing to migrate: the bundle already pins fdf_version %s, the version %s upgrades a bundle to.\n", target, binaryName())
	fmt.Fprintln(out, "if a newer spec version exists, upgrade fdf and re-run — a version-pinned shim (mise, asdf) can hold an older fdf in this directory.")
	restore := map[string]string{}
	if !specCurrent(root) {
		doc, err := scaffold.SpecDoc(target)
		if err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		restore["SPEC.md"] = string(doc)
	}
	for _, reg := range layout.Registers {
		if idx := reg + "/INDEX.md"; reg != "releases" && !exists(filepath.Join(root, idx)) {
			restore[idx], _ = scaffold.IndexText(reg)
		}
	}
	for _, name := range layout.ContextDocs {
		if !exists(filepath.Join(root, name)) {
			restore[name], _ = scaffold.ContextStub(name)
		}
	}
	// Nothing is written through a symbolic link: a file it would restore
	// that is one, such as a SPEC.md or a dangling Context document, or a
	// register it would restore an index into, is refused first, as a
	// migration refuses it; and so is a register whose place a file takes.
	var linked []string
	for _, rel := range sortedKeys(restore) {
		if reg := path.Dir(rel); reg != "." {
			if to, err := os.Readlink(filepath.Join(root, reg)); err == nil {
				linked = append(linked, linkedRegister(reg, to))
				continue
			}
			if problem := registerFile(root, reg); problem != "" {
				linked = append(linked, problem)
				continue
			}
		}
		if to, err := os.Readlink(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			linked = append(linked, linkedFile(rel, to))
		}
	}
	if len(linked) > 0 {
		fmt.Fprintln(out, "cannot restore — fix these first (bundle left unchanged):")
		for _, l := range linked {
			fmt.Fprintln(out, "  "+l)
		}
		return 1
	}
	switch {
	case len(restore) == 0:
		fmt.Fprintln(out, "nothing to restore: the spec copy, the registers' indexes and the Context documents are all there.")
	case o.DryRun:
		fmt.Fprintln(out, "would restore: "+strings.Join(sortedKeys(restore), ", "))
	default:
		fmt.Fprintln(out, "restored: "+strings.Join(sortedKeys(restore), ", "))
	}
	if o.DryRun {
		return 0
	}
	for _, rel := range sortedKeys(restore) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		if err := os.WriteFile(p, []byte(restore[rel]), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
	}
	fmt.Fprintln(out, "\nvalidating bundle:")
	return bundle.Validate(root, bundle.Options{RepoRoot: o.Project, Out: out, FreshStubsAdvisory: true})
}

// specCurrent reports whether the bundle's SPEC.md is the vendored spec of
// target: the embedded text under its frontmatter.
func specCurrent(root string) bool {
	raw, err := os.ReadFile(filepath.Join(root, "SPEC.md"))
	want, werr := scaffold.SpecText(target)
	if err != nil || werr != nil {
		return false
	}
	text := string(raw)
	if rest, ok := strings.CutPrefix(text, "---\n"); ok {
		if i := strings.Index(rest, "\n---\n"); i >= 0 {
			text = strings.TrimPrefix(rest[i+len("\n---\n"):], "\n")
		}
	}
	return text == string(want)
}

// stubRe matches the validator's messages for a Context document that is
// still an unfilled stub (bundle's F9 check): "freshly scaffolded stub" while
// the bundle has features — migrate's advisory form of F9 — and "still an
// unfilled stub" while it has none. Only those lines mean the fdf-init
// interview has work to do; a document that merely has "stub" in its name
// (debts/stub-gateway.md) is not one of them.
var stubRe = regexp.MustCompile(`(?m)^warn: ([A-Z]+\.md): (?:freshly scaffolded stub|still an unfilled stub) — run the fdf-init interview to populate it`)

// freshStubRe is stubRe's advisory F9 form: a stub the next plain validate
// fails, because the bundle has a feature.
var freshStubRe = regexp.MustCompile(`(?m)^warn: [A-Z]+\.md: freshly scaffolded stub — .*\(F9\)$`)

// stubsIn lists the Context documents a validation report calls unfilled
// stubs, in the order it reports them.
func stubsIn(report string) []string {
	var names []string
	for _, m := range stubRe.FindAllStringSubmatch(report, -1) {
		names = append(names, m[1])
	}
	return names
}

// joinNames writes names as prose: "A", "A and B", "A, B, and C".
func joinNames(names []string) string {
	switch len(names) {
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
}

// reportV07 says what v0.7 changes about a bundle that has just reached it:
// F12 now reads every document and name, and the debt register may hold
// defects that belong in the new bug register. Neither is a migration step —
// both are judgments — so the command names the tools and stops there.
func reportV07(root, validation string, out io.Writer) {
	if lex, _ := bundle.LoadLexicon(root); lex != nil {
		if occ := bundle.ScanBundle(root, lex); len(occ) > 0 {
			fmt.Fprintf(out, "\nv0.7: the domain language now reaches every document and name — %s.\n", refactor.BannedSummary(occ))
			fmt.Fprintln(out, "      `fdf lexicon` lists them; triage the other senses into `except:`, then sweep one term at a time")
			fmt.Fprintln(out, "      with `fdf lexicon --term <Term> --fix --dry-run` and `--fix`.")
		}
	}
	if n := strings.Count(validation, "has no test case — a case is a `## "); n > 0 {
		fmt.Fprintf(out, "\nv0.7: a test case is a `## <scenario name>` heading under `# Test Cases`, matched exactly —\n")
		fmt.Fprintf(out, "      %d scenario(s) have none (F8). Rewrite those test documents' cases as headings, by hand:\n", n)
		fmt.Fprintln(out, "      bullets and tables naming a scenario no longer count.")
	}
	if n := strings.Count(validation, "is neither a date (2026-02-14) nor an RFC 3339 time"); n > 0 {
		fmt.Fprintf(out, "\nv0.7: %d timestamp(s) are neither a date nor an RFC 3339 time with Z or an offset (F1).\n", n)
		fmt.Fprintln(out, "      Give each the Z or offset it was written in, or keep only its date.")
	}
	if n := strings.Count(validation, ".surface.md — write one"); n > 0 {
		fmt.Fprintf(out, "\nv0.7: %d feature(s) have no slug.surface.md. Write one where the feature adds or changes an interface,\n", n)
		fmt.Fprintln(out, "      or say `surface: none` in its frontmatter.")
	}
	if n := countRegisterEntries(filepath.Join(root, "debts")); n > 0 {
		fmt.Fprintf(out, "\nv0.7: of the %d debt(s) on the register, any that describes the software doing something wrong\n", n)
		fmt.Fprintln(out, "      is a bug — re-file it with `fdf mv debts/<id> bugs/<id>`, then give it the `# Expected` a bug states.")
	}
}

// statusTagRe matches an index listing that ends in the status tag `fdf new`,
// `fdf change`, `fdf fix` and `fdf adopt` wrote before v0.7, or one a person
// kept up by hand: ` (**draft**)`. Only a status word counts, so other bold
// text in parentheses stays.
var statusTagRe = regexp.MustCompile(`^([ \t]*[-*+][ \t].*\]\(.*\).*?)[ \t]*\(\*\*(?:draft|specified|planned|implementing|done|retired|adopted|pending|in-progress|active|superseded|open|accepted|resolved|shipped)\*\*\)[ \t]*(\r?)$`)

// countRegisterEntries counts the documents in a register directory, groups
// included, leaving out its index, its log and each entry's log sibling.
func countRegisterEntries(dir string) int {
	n := 0
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		if base := filepath.Base(p); base != "INDEX.md" && base != "LOG.md" && !strings.HasSuffix(base, ".log.md") {
			n++
		}
		return nil
	})
	return n
}

// preflightV4 scans for content the v0.4 layout cannot represent and that
// this migration cannot mechanically fix: dotted group-level filenames
// (v0.4 reserves the dot for trail roles), non-trail non-task files inside
// feature directories (v0.4 task dirs hold only NN-slug.md tasks), and
// draft features with a feature-dir LOG.md (lifting it would create a trail
// sibling, which v0.4 forbids on drafts). Basenames are normalized through
// the v0.1 rename map so pre-rename bundles are screened too.
func preflightV4(root string) []string {
	var problems []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		relPath := filepath.ToSlash(rel(root, p))
		parts := strings.Split(relPath, "/")
		base := filepath.Base(p)
		if to, ok := renames[base]; ok {
			base = to
		}
		switch len(parts) {
		case 2:
			if parts[0] == "releases" || base == "INDEX.md" || base == "LOG.md" {
				return nil
			}
			if strings.Contains(strings.TrimSuffix(base, ".md"), ".") {
				problems = append(problems, fmt.Sprintf("%s: filename contains a dot, which v0.4 reserves for trail roles (slug.spec.md) — rename it before migrating", relPath))
			}
		case 3:
			if _, liftable := trailBasenames[base]; liftable {
				if base == "LOG.md" && featureIsDraft(root, parts[0], parts[1]) {
					problems = append(problems, fmt.Sprintf("%s: draft features may not have trail files under v0.4 — fold this log into the root LOG.md (or advance the feature) before migrating", relPath))
				}
				return nil
			}
			if taskFileRe.MatchString(base) {
				return nil
			}
			problems = append(problems, fmt.Sprintf("%s: v0.4 task directories may contain only NN-slug.md tasks — move or remove this file before migrating", relPath))
		}
		return nil
	})
	sort.Strings(problems)
	return problems
}

// featureIsDraft reports whether the sibling feature document of a paired
// directory carries status: draft.
func featureIsDraft(root, group, slug string) bool {
	raw, err := os.ReadFile(filepath.Join(root, group, slug+".md"))
	if err != nil {
		return false
	}
	m := statusRe.FindSubmatch(raw)
	return m != nil && string(m[1]) == "draft"
}

func readPin(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if err != nil {
		// Also try lowercase pre-rename form.
		raw, err = os.ReadFile(filepath.Join(root, "index.md"))
		if err != nil {
			return ""
		}
	}
	if m := pinValueRe.FindSubmatch(raw); m != nil {
		return string(m[1])
	}
	return ""
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }
