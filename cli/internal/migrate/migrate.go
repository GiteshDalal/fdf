// Package migrate mechanically upgrades a bundle between adjacent FDF spec
// versions. Chains forward to 0.7, the last 0.x version:
//
//	v0.1 → case renames, vendored-spec removal, TEST stubs
//	v0.2/v0.3 → lift nested trail to stem-qualified siblings
//	v0.4/v0.5/v0.6 → nothing structural: 0.4→0.5, 0.5→0.6 and 0.6→0.7 add
//	      documents, not moves
//	any → links repaired by the one link engine, index status tags dropped,
//	      pin 0.7, RefreshSpec, EnsureContextStubs, changes/, practices/,
//	      debts/ and bugs/ INDEX.md, log, validate
//
// The whole migration is worked out first (plan.go), and only then applied,
// so a bundle it refuses is left as it was.
//
// Only a bundle pinned to 0.x, or to nothing, is migrated. A bundle pinned to
// 1.0 or later is refused: its layout is not one these steps know, and
// running them over it would pin it back to 0.7. So is a pin that is not a
// version, such as 1.0.0, and a root whose INDEX.md pins nothing inside a
// pinned bundle: a register or a group of it, where the steps would build a
// second bundle.
//
// Ends by validating the result with FreshStubsAdvisory so unfilled Context
// stubs do not fail the migration (plain `fdf validate` will still enforce F9).
package migrate

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/refactor"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// target is the pin migrate writes: 0.7, the last 0.x version. Spec 1.0 files
// every feature under features/, which no step here does.
const target = "0.7"

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
var timestampRe = regexp.MustCompile(`(?m)^timestamp:\s*(\S+)`)

func Run(root, repoRoot string, out io.Writer) int {
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

	// Only a bundle pinned to 0.x, or to nothing, goes ahead.
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
	case v.Major != 0:
		fmt.Fprintf(out, "cannot migrate: the bundle pins fdf_version %s, and %s upgrades a 0.x bundle to %s — the bundle was left as it is.\n", pin, binaryName(), target)
		return 1
	}
	if pin == target {
		// Idempotent repair path: a re-run (or a hand-pinned bundle) still
		// gets the spec copy and any missing Context stubs, and validates
		// with the same stub leniency as a fresh migration — so running
		// migrate twice in a row cannot flip from success to failure.
		fmt.Fprintf(out, "nothing to migrate: the bundle already pins fdf_version %s, the version %s upgrades a bundle to.\n", target, binaryName())
		fmt.Fprintln(out, "if a newer spec version exists, upgrade fdf and re-run — a version-pinned shim (mise, asdf) can hold an older fdf in this directory.")
		fmt.Fprintln(out, "ensuring spec copy and context stubs:")
		if code := scaffold.RefreshSpec(root, target, out); code != 0 {
			return code
		}
		if code := scaffold.EnsureContextStubs(root, out); code != 0 {
			return code
		}
		if code := scaffold.EnsureChangesIndex(root, out); code != 0 {
			return code
		}
		if code := scaffold.EnsurePracticesIndex(root, out); code != 0 {
			return code
		}
		if code := scaffold.EnsureDebtsIndex(root, out); code != 0 {
			return code
		}
		if code := scaffold.EnsureBugsIndex(root, out); code != 0 {
			return code
		}
		fmt.Fprintln(out, "\nvalidating bundle:")
		return bundle.Validate(root, bundle.Options{RepoRoot: repoRoot, Out: out, FreshStubsAdvisory: true})
	}

	// v0.7 reserves bugs/ for the bug register. A bundle that already uses it
	// as a feature group has to move that group first; nothing else here can
	// decide its new name. Refused before anything is touched.
	if problem := bugsGroupConflict(root); problem != "" {
		fmt.Fprintln(out, "cannot migrate — fix this first (bundle left unchanged):")
		fmt.Fprintln(out, "  "+problem)
		return 1
	}

	p, problems, err := newPlan(rootAbs, pin)
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

	// The pin, and the migration's entry in the bundle-root log, where every
	// bundle-wide event goes.
	from := pin
	if from == "" {
		from = "unpinned"
	}
	p.texts["INDEX.md"] = withPin(p.text("INDEX.md"), target)
	entry := fmt.Sprintf("**Migrated**: fdf_version %s → %s with `fdf migrate`.", from, target)
	if p.tags > 0 {
		entry += fmt.Sprintf(" Removed the status tag from %d index listing(s); a document's status lives only in its frontmatter.", p.tags)
	}
	p.texts["LOG.md"] = withLogEntry(p.text("LOG.md"), entry)
	if err := p.apply(out); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	// Refresh the bundle-root spec copy and scaffold missing Context stubs
	// (including SURFACES.md on v0.4) and the registers' indexes.
	if code := scaffold.RefreshSpec(root, target, out); code != 0 {
		return code
	}
	if code := scaffold.EnsureContextStubs(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsureChangesIndex(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsurePracticesIndex(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsureDebtsIndex(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsureBugsIndex(root, out); code != 0 {
		return code
	}

	// Report what actually changed, then validate. Without this the only
	// evidence of a migration is a scroll of per-file lines, and a migration
	// that moved nothing is indistinguishable from one that did.
	fmt.Fprintf(out, "\ndone: migrated bundle at %s\n", rootAbs)
	fmt.Fprintf(out, "  fdf_version %s -> %s\n", from, target)
	lifted := p.lifted()
	fmt.Fprintf(out, "  %d trail file(s) lifted to stem-qualified siblings\n", lifted)
	if lifted == 0 {
		fmt.Fprintln(out, "  (no nested trail files were present — layout already matched)")
	}
	if p.tags > 0 {
		fmt.Fprintf(out, "  %d status tag(s) removed from index listings\n", p.tags)
	}
	fmt.Fprintln(out, "  logged in LOG.md")

	// Freshly scaffolded Context stubs are advisory here — migration
	// succeeded; filling them is the human's next step via fdf-init.
	fmt.Fprintln(out, "\nvalidating migrated bundle:")
	var report bytes.Buffer
	code := bundle.Validate(root, bundle.Options{RepoRoot: repoRoot, Out: io.MultiWriter(out, &report), FreshStubsAdvisory: true})
	// Say so only when validation found a stub, and name the ones it found:
	// F9 fails a plain validate only while one is unfilled and the bundle has
	// a feature.
	if stubs := stubsIn(report.String()); code == 0 && len(stubs) > 0 {
		fmt.Fprintln(out, "\nnext: run the fdf-init skill to fill "+joinNames(stubs)+".")
		if freshStubRe.MatchString(report.String()) {
			fmt.Fprintln(out, "warning: the next plain `fdf validate` will fail F9 until those stubs are filled (migrate reports an unfilled stub as a warning, not an error).")
		}
	}
	reportV07(root, report.String(), out)
	return code
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

// bugsGroupConflict reports a bundle that uses bugs/ as a feature group, which
// v0.7 reserves for the bug register. Bug documents already there are fine.
func bugsGroupConflict(root string) string {
	var offender string
	filepath.WalkDir(filepath.Join(root, "bugs"), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || offender != "" || !strings.HasSuffix(p, ".md") {
			return nil
		}
		base := filepath.Base(p)
		if base == "INDEX.md" || base == "LOG.md" || strings.HasSuffix(base, ".log.md") {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		if m := typeLineRe.FindSubmatch(raw); m == nil || strings.Trim(string(m[1]), `"'`) != "Bug" {
			offender = rel(root, p)
		}
		return nil
	})
	if offender == "" {
		return ""
	}
	return fmt.Sprintf("bugs/ is a feature group (%s), but v0.7 reserves bugs/ for the bug register — rename the group first (its directory, its listing in INDEX.md and the links to it), then re-run fdf migrate", filepath.ToSlash(offender))
}

var typeLineRe = regexp.MustCompile(`(?m)^type:\s*(\S+)`)

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
