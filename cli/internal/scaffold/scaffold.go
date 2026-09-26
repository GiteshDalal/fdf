// Package scaffold implements fdf init and fdf new.
package scaffold

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	fdf "github.com/GiteshDalal/fdf"
	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

const currentVersion = "1.0"
const specURL = "https://github.com/GiteshDalal/fdf/blob/main/spec/" + currentVersion + ".md"

// contextDocs are the bundle-root Context documents fdf init scaffolds as
// stubs; the fdf-init interview fills them. Each stub carries the
// bundle.stubSentinel so `fdf validate` can tell filled from unfilled.
var contextDocs = []struct{ file, title, purpose, headings string }{
	{"STACK.md", "Technology Stack",
		"the languages, frameworks, runtimes, libraries, data stores, and versions this project uses",
		"## Languages & runtimes\n\n## Frameworks & libraries\n\n## Data stores\n\n## Notable dependencies\n"},
	{"ARCHITECTURE.md", "Architecture & Principles",
		"the architecture style, how code is organized, and the design principles contributors follow",
		"## Style\n\n## Code organization\n\n## Design principles\n\n## Key decisions\n"},
	{"SURFACES.md", "Surfaces & Interaction Design",
		"how this project’s surfaces present themselves and are engaged with — APIs, UIs, CLIs, events, and inputs — plus links to assets and exemplars",
		"## Purpose\n\n## Surfaces\n\n## Principles\n\n## Conventions by surface\n\n### API / machine interfaces\n\n### Human UI\n\n### CLI / operator surfaces\n\n### Inputs and processing\n\n## Assets and exemplars\n\n## Out of scope\n"},
	{"INFRA.md", "Build & Deployment Infrastructure",
		"how the project is built, tested, packaged, and deployed, and the environments and targets it runs on",
		"## Build & test\n\n## Packaging\n\n## Environments & targets\n\n## Deployment\n"},
	{"DOMAIN.md", "Domain Language",
		"the canonical name for each thing this project is about, and the words that must not be used for it instead",
		domainTermsStub},
}

// domainTermsStub models DOMAIN.md's parsed `# Terms` grammar (F12): one
// `## <Term>` per canonical term, definition first, optional bullets after.
const domainTermsStub = "# Terms\n\n" +
	"<!-- One `## <Term>` heading per canonical term. The first line under it is\n" +
	"     the definition; then, optionally:\n" +
	"       - instead-of: the words this term replaces (banned in every document\n" +
	"         and code identifier, not in the copy a surface shows people)\n" +
	"       - except: phrases in which a banned word means something else\n" +
	"         (\"data store\"), never the banned word alone\n" +
	"       - code: how the term appears in the code (type, table, field)\n" +
	"       - see: a link to the practice that explains it in depth\n" +
	"     Once no document uses a banned word, propose adding `strict: true` to\n" +
	"     this file's frontmatter: every validation then treats one as an error.\n\n" +
	"## Venue\n" +
	"A physical location where a merchant sells.\n" +
	"- instead-of: store, business, tenant\n" +
	"- except: data store\n" +
	"- code: `Venue` (model), `venues` (table)\n" +
	"-->\n"

// SpecDoc renders the embedded spec of a version as a bundle-root Reference
// document. Agents and readers of the bundle need no external context to
// learn the format: the pinned version's spec travels with the bundle at
// /SPEC.md.
func SpecDoc(version string) ([]byte, error) {
	raw, err := fs.ReadFile(fdf.Assets, "spec/"+version+".md")
	if err != nil {
		return nil, err
	}
	fm := fmt.Sprintf("---\ntype: Reference\ntitle: Feature Document Format spec\ndescription: The FDF v%s specification this bundle conforms to.\ntimestamp: %s\n---\n\n", version, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	return append([]byte(fm), raw...), nil
}

// stubSentinel marks an unfilled Context document; it MUST match
// bundle.stubSentinel so validation can distinguish filled from unfilled.
const stubSentinel = "<!-- fdf:stub -->"

// EnsureSpec writes the spec of version, the one the bundle pins, to
// /SPEC.md when the bundle has none (init): a bundle restores the spec its
// pin names, not the current one, which a later minor would make newer.
// fdf migrate vendors the spec it pins with SpecDoc.
func EnsureSpec(root, version string, out io.Writer) int {
	specPath := filepath.Join(root, "SPEC.md")
	if _, err := os.Stat(specPath); err == nil {
		return 0
	}
	doc, err := SpecDoc(version)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := os.WriteFile(specPath, doc, 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "wrote SPEC.md (FDF v%s spec copy)\n", version)
	return 0
}

// registerIndexes are the INDEX.md fdf writes for each register but
// releases/ (whose index `fdf release` keeps): what the register holds, and
// how the line that reports writing it names the register.
var registerIndexes = map[string]struct{ body, what string }{
	"features": {"# Features\n\nWhat the software does: one document per feature, in Markdown and Gherkin,\nfiled flat here or in groups.\n\n* [Format reference](/SPEC.md) - how features are structured.\n",
		"what the software does"},
	"changes": {"# Changes\n\nPost-delivery changes and fixes for delivered features.\nA `Change` alters documented behavior; a `Fix` restores behavior the feature\ndocument already describes. Both may be filed flat here or in groups.\n\n* [Format reference](/SPEC.md) - how changes and fixes are structured.\n",
		"post-delivery changes and fixes"},
	"practices": {"# Practices\n\nHow this project does the things it does the same way every time —\nauthorization, permission checks, payment capture, database access. Each is\nbinding on all code it applies to, and changes only with human approval.\n\n* [Format reference](/SPEC.md) - how practices are structured.\n",
		"project practices"},
	"debts": {"# Debt\n\nKnown gaps between what this project says and what the code does — work\nleft undone, and rules the codebase does not follow everywhere yet. Run\n`fdf debt` to read the register.\n\n* [Format reference](/SPEC.md) - how debts are structured.\n",
		"the debt register"},
	"bugs": {"# Bugs\n\nKnown defects — the software doing something wrong that someone could\nobserve — that have not been repaired yet. Each is repaired by a Fix or a\nChange that names it in `resolves`. Run `fdf bug` to read the register.\n\n* [Format reference](/SPEC.md) - how bugs are structured.\n",
		"the bug register"},
}

// IndexText is the INDEX.md fdf writes for the register reg (features,
// changes, practices, debts or bugs), and whether it writes one.
func IndexText(reg string) (string, bool) {
	ix, ok := registerIndexes[reg]
	return ix.body, ok
}

// EnsureIndex creates the INDEX.md of the register reg (features, changes,
// practices, debts or bugs) if absent. A register with an index is found from
// the start, rather than being something its first document has to invent.
func EnsureIndex(root, reg string, out io.Writer) int {
	ix, ok := registerIndexes[reg]
	if !ok {
		fmt.Fprintf(out, "error: %s is not a register fdf writes an index for\n", reg)
		return 1
	}
	dir := filepath.Join(root, reg)
	idx := filepath.Join(dir, "INDEX.md")
	if _, err := os.Stat(idx); err == nil {
		return 0
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := os.WriteFile(idx, []byte(ix.body), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "wrote %s/INDEX.md (%s)\n", reg, ix.what)
	return 0
}

// ensureIndexes creates whichever of the registers' indexes `fdf init`
// scaffolds are absent: that of every register layout knows but releases/,
// which `fdf release` writes with the first release.
func ensureIndexes(root string, out io.Writer) int {
	for _, reg := range layout.Registers {
		if reg == "releases" {
			continue
		}
		if code := EnsureIndex(root, reg, out); code != 0 {
			return code
		}
	}
	return 0
}

// CurrentVersion is the spec version `fdf init` pins into new bundles and the
// version `fdf spec` prints when none is requested.
func CurrentVersion() string { return currentVersion }

// SpecVersions lists the spec versions embedded in the binary, ascending.
func SpecVersions() []string {
	entries, err := fs.ReadDir(fdf.Assets, "spec")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		v := strings.TrimSuffix(e.Name(), ".md")
		// spec/README.md is not a version; spec/<MAJOR.MINOR>.md is.
		if _, ok := specver.Parse(v); ok && v != e.Name() {
			out = append(out, v)
		}
	}
	specver.Sort(out)
	return out
}

// SpecText returns the embedded normative text of a spec version, exactly as
// published under spec/ — no bundle frontmatter (that is specDoc's job).
func SpecText(version string) ([]byte, error) {
	if _, ok := specver.Parse(version); !ok {
		return nil, fmt.Errorf("%q is not a spec version (expected MAJOR.MINOR, e.g. %s)", version, currentVersion)
	}
	raw, err := fs.ReadFile(fdf.Assets, "spec/"+version+".md")
	if err != nil {
		return nil, fmt.Errorf("no embedded spec for version %s (available: %s)", version, strings.Join(SpecVersions(), ", "))
	}
	return raw, nil
}

// writeContextStubs places the Context stubs at the bundle root, each
// only if absent. The fdf-init interview replaces the stub bodies later.
func writeContextStubs(root string, out io.Writer) int {
	for _, c := range contextDocs {
		path := filepath.Join(root, c.file)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		body, _ := ContextStub(c.file)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		fmt.Fprintf(out, "wrote %s (context stub — fill via the fdf-init skill)\n", c.file)
	}
	return 0
}

// ContextStub is the stub fdf writes for the Context document named file,
// such as DOMAIN.md, and whether it is one. The fdf-init interview fills it.
func ContextStub(file string) (string, bool) {
	for _, c := range contextDocs {
		if c.file != file {
			continue
		}
		now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		return fmt.Sprintf(`---
type: Context
title: %s
description: %s
timestamp: %s
---

%s

# %s

> %s **This is a critical context document.** It is the current snapshot of
> %s. Run the `+"`fdf-init`"+` skill to fill it through a guided
> interview, and change it afterward only with explicit human approval,
> logging each change. Accurate context here is what lets an agent do real
> engineering instead of guessing. Delete this banner once filled.

%s`, c.title, "Project "+strings.ToLower(c.title)+" — current snapshot.", now, stubSentinel, c.title, "⚠️", c.purpose, c.headings), true
	}
	return "", false
}

// contextDocNames lists the Context documents for user-facing messages, so a
// new one never leaves a stale literal behind.
func contextDocNames() string {
	names := make([]string, 0, len(contextDocs))
	for _, c := range contextDocs {
		names = append(names, c.file)
	}
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
}

// Practice scaffolds a practice at practices/[<group>/…]<slug>.md. A practice
// has no trail and no tasks: the document is the whole thing, so there is
// nothing else to create.
func Practice(root, name string, out io.Writer) int {
	if !RequireSupported(root, out) {
		return 1
	}
	id := NewID(root, "practices", name, out)
	if id == "" {
		return 1
	}
	file := filepath.Join(root, filepath.FromSlash(id)+".md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	title := Title(path.Base(id))
	body := fmt.Sprintf(`---
type: Practice
status: active
title: %s
description: TODO — one sentence on what this practice governs.
# applies-to: [internal/authz, internal/http]   # the repo paths this governs (R1: they must exist)
timestamp: %s
---

# Rules

- TODO — the binding statements, imperative and short: what code MUST do.

# How

TODO — the canonical mechanism, with links to the code that implements it.

# Boundaries

TODO — where this does not apply, and what to do instead there. Optional.

# Rationale

TODO — why it is this way, so a later change knows what it is trading away. Optional.
`, title, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	if err := WriteNew(root, id, body); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if code := EnsureIndex(root, "practices", out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "wrote %s.md (type: Practice, status: active)\n", id)
	if code := ListEntry(root, "practices", strings.TrimPrefix(id, "practices/"), title, "TODO", out); code != 0 {
		return code
	}
	fmt.Fprintln(out, "next: fill `# Rules` and set `applies-to` to the paths this governs.")
	fmt.Fprintln(out, "      a practice binds all future code — get human approval before it lands.")
	return 0
}

// rootIndex is the root INDEX.md `fdf init` writes: the pin, the registers
// it creates (every one but releases/, which `fdf release` lists with the
// first release), the Context documents and the vendored spec.
func rootIndex() string {
	var registers strings.Builder
	for _, r := range registerListings {
		if r.reg != "releases" {
			registers.WriteString(r.line + "\n")
		}
	}
	return fmt.Sprintf(`---
fdf_version: %q
---

# Feature Bundle

This bundle conforms to [FDF v%s](/SPEC.md). A document's ID is its path from
here without `+"`.md`"+`, such as `+"`features/payments/instant-refunds`"+`.

%s* [Stack](/STACK.md), [Architecture](/ARCHITECTURE.md), [Surfaces](/SURFACES.md),
  [Infrastructure](/INFRA.md), [Domain](/DOMAIN.md) - the project's context.
* [Format reference](/SPEC.md) - the spec this bundle pins ([upstream](%s)).

Validate with `+"`fdf validate`"+`; [LOG.md](/LOG.md) records what happened to the bundle.
`, currentVersion, currentVersion, registers.String(), specURL)
}

func Init(root string, out io.Writer) int {
	idx := filepath.Join(root, "INDEX.md")
	if _, err := os.Stat(idx); err == nil {
		// A bundle is here already. Its pin is read as every command reads
		// it, and one the commands cannot work on is refused as they refuse
		// it: a 0.x pin, or none, with `fdf migrate`, a newer one with a
		// newer fdf.
		if !RequireSupported(root, out) {
			return 1
		}
		// Backfill what a bundle initialized before it existed, or since
		// lost, is missing: the spec copy of the version it pins, Context
		// stubs and indexes.
		if code := EnsureSpec(root, Pin(root), out); code != 0 {
			return code
		}
		if code := writeContextStubs(root, out); code != 0 {
			return code
		}
		if code := ensureIndexes(root, out); code != 0 {
			return code
		}
		fmt.Fprintf(out, "\ndone: bundle at %s was already initialized and up to date (fdf_version %s)\n", root, Pin(root))
		fmt.Fprintln(out, "  nothing was overwritten; only missing files above were added")
		return 0
	}
	// No bundle here yet. A directory inside a pinned bundle is one of its
	// registers or groups, not the place for a second bundle.
	if bundle := fdfroot.BundleAbove(root); bundle != "" {
		fmt.Fprintln(out, "error:", fdfroot.InsideBundle(root, bundle))
		return 1
	}
	// Nor is a directory that holds Markdown already, such as a bundle from
	// before 1.0 that never had an INDEX.md: init would write its LOG.md over
	// the one there, and pin 1.0 over the old layout. Such a bundle is fdf
	// migrate's.
	if file, route := fdfroot.Unindexed(root); file != "" {
		fmt.Fprintf(out, "error: %s holds %s but no INDEX.md, and fdf init starts a bundle only where there is none — %s; move anything else out first\n", root, file, route)
		return 1
	}
	// Nor is docs/fdf beside a docs/features whose INDEX.md pins nothing, as
	// a bundle's from before 1.0 may: fdf migrate moves such a bundle here
	// (fdfroot.Beside).
	if old := fdfroot.Beside(root); old != "" {
		fmt.Fprintf(out, "error: %s beside %s holds an INDEX.md that pins no version, as a bundle's from before 1.0 may, and fdf init starts no bundle where fdf migrate would move that one — %s; if that is no bundle, rename its INDEX.md, and run fdf init again\n", old, root, fdfroot.Upgrading("it", old))
		return 1
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	today := time.Now().UTC().Format("2006-01-02")
	log := fmt.Sprintf("# Bundle Update Log\n\n## %s\n* **Initialization**: scaffolded by `fdf init` (FDF v%s).\n", today, currentVersion)
	if err := os.WriteFile(idx, []byte(rootIndex()), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(root, "LOG.md"), []byte(log), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if code := EnsureSpec(root, currentVersion, out); code != 0 {
		return code
	}
	if code := writeContextStubs(root, out); code != 0 {
		return code
	}
	if code := ensureIndexes(root, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "\ndone: initialized FDF bundle at %s\n", root)
	fmt.Fprintf(out, "  pinned fdf_version %s; wrote INDEX.md, LOG.md, SPEC.md, %d Context stub(s) and the registers' indexes\n", currentVersion, len(contextDocs))
	fmt.Fprintln(out, "next: run the fdf-init skill to fill "+contextDocNames()+" before adding features.")
	fmt.Fprintln(out, "      `fdf validate` warns about them now, and fails F9 once a feature exists while any is unfilled.")
	fmt.Fprintln(out, "      On an existing codebase, map what it already does with `fdf adopt` (the fdf-adopt skill).")
	return 0
}

// New scaffolds a draft feature at features/[<group>/…]<slug>.md, listed in
// the index beside it; a new group is listed in its parent's index.
func New(root, name string, out io.Writer) int {
	if !RequireSupported(root, out) {
		return 1
	}
	id := NewID(root, "features", name, out)
	if id == "" {
		return 1
	}
	slug := path.Base(id)
	featurePath := filepath.Join(root, filepath.FromSlash(id)+".md")
	if err := os.MkdirAll(filepath.Dir(featurePath), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	title := Title(slug)
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	feature := fmt.Sprintf(`---
type: Feature
title: %s
description: TODO — one sentence.
status: draft
timestamp: %s
---

# Feature

`+"```gherkin"+`
Feature: %s
  As a <role>
  I want <capability>
  So that <value>
`+"```"+`

# Scenarios

`+"```gherkin"+`
Scenario: Replace me
  Given a precondition
  When something happens
  Then an observable outcome
`+"```"+`
`, title, now, title)
	if err := WriteNew(root, id, feature); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "created %s.md (status: draft)\n", id)
	if code := listFeature(root, id, title, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "\ndone: feature %s is a draft — one Feature: fence, one Scenario: fence, no trail siblings yet\n", id)
	fmt.Fprintln(out, "next: ask the user about the feature, one question at a time (fdf-brainstorm step 3); then replace the placeholders here and in its INDEX.md listing, write its Gherkin, and present the design to the user; only once they approve it, write "+slug+".spec.md and set status: specified (the fdf-brainstorm skill drives this).")
	return 0
}

// listFeature lists a new feature in the index beside it, with a TODO where
// its description goes, creating features/INDEX.md and any group's index on
// the way.
func listFeature(root, id, title string, out io.Writer) int {
	if code := EnsureIndex(root, "features", out); code != 0 {
		return code
	}
	return ListEntry(root, "features", strings.TrimPrefix(id, "features/"), title, "TODO", out)
}

// Adopt scaffolds an adopted feature (v0.7) at features/[<group>/…]<slug>.md:
// a capability the software already has, documented from the code rather
// than built through the lifecycle. It starts as a map entry — a Feature:
// block and the code it lives in, no scenarios — and gets no spec, plan or
// tasks, ever. resources are the project-relative paths of that code;
// projectRoot, when set, is where they must exist.
func Adopt(root, projectRoot, name string, resources []string, out io.Writer) int {
	if !RequireSupported(root, out) {
		return 1
	}
	if len(resources) == 0 {
		fmt.Fprintln(out, "error: --resource is required — name the code this capability lives in; with no tasks, it is the feature's only link to the code")
		return 2
	}
	id := NewID(root, "features", name, out)
	if id == "" {
		return 1
	}
	if projectRoot != "" {
		for _, r := range resources {
			if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(r))); err != nil {
				fmt.Fprintf(out, "error: --resource %s does not exist under %s — adoption documents code that exists\n", r, projectRoot)
				return 1
			}
		}
	}
	slug := path.Base(id)
	featurePath := filepath.Join(root, filepath.FromSlash(id)+".md")
	if err := os.MkdirAll(filepath.Dir(featurePath), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	title := Title(slug)
	feature := fmt.Sprintf(`---
type: Feature
title: %s
description: TODO — one sentence on what this capability does today.
status: adopted
resource: [%s]
timestamp: %s
---

# Feature

`+"```gherkin"+`
Feature: %s
  As a <role>
  I want <capability>
  So that <value>
`+"```"+`

Built before this bundle existed and documented from the code as it stands.
Scenarios are backfilled as work reaches it: each describes what the code
already does, and its case in %s passes today.
`, title, strings.Join(resources, ", "), time.Now().UTC().Format("2006-01-02T15:04:05Z"), title, "`"+slug+".test.md`")
	if err := WriteNew(root, id, feature); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "created %s.md (status: adopted — a map entry: a Feature: block and its code, no scenarios yet)\n", id)
	if code := listFeature(root, id, title, out); code != 0 {
		return code
	}
	fmt.Fprintln(out, "\nnext: fill the Feature: block — who uses this, and for what. Scenarios come later, one at a time,")
	fmt.Fprintln(out, "      each with its case in "+slug+".test.md, passing against the code as it stands (the fdf-adopt skill).")
	return 0
}
