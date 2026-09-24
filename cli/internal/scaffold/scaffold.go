// Package scaffold implements fdf init and fdf new.
package scaffold

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	fdf "github.com/GiteshDalal/fdf"
)

const currentVersion = "0.7"
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
	"     Once no document uses a banned word, add `strict: true` to this file's\n" +
	"     frontmatter: every validation then treats one as an error.\n\n" +
	"## Venue\n" +
	"A physical location where a merchant sells.\n" +
	"- instead-of: store, business, tenant\n" +
	"- except: data store\n" +
	"- code: `Venue` (model), `venues` (table)\n" +
	"-->\n"

// specDoc renders the embedded spec for the current version as a bundle-root
// Reference document. Agents and readers of the bundle need no external
// context to learn the format: the pinned version's spec travels with the
// bundle at /SPEC.md.
func specDoc() ([]byte, error) {
	raw, err := fs.ReadFile(fdf.Assets, "spec/"+currentVersion+".md")
	if err != nil {
		return nil, err
	}
	fm := fmt.Sprintf("---\ntype: Reference\ntitle: Feature Document Format spec\ndescription: The FDF v%s specification this bundle conforms to.\ntimestamp: %s\n---\n\n", currentVersion, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	return append([]byte(fm), raw...), nil
}

// stubSentinel marks an unfilled Context document; it MUST match
// bundle.stubSentinel so validation can distinguish filled from unfilled.
const stubSentinel = "<!-- fdf:stub -->"

// EnsureSpec, RefreshSpec, and EnsureContextStubs let other commands (e.g.
// migrate) place the bundle-root spec copy and Context stubs without
// re-implementing them. EnsureSpec writes only if absent (init); RefreshSpec
// overwrites to the current version's spec (migrate), so a bumped pin never
// leaves a stale vendored spec behind.
func EnsureSpec(root string, out io.Writer) int         { return writeSpec(root, false, out) }
func RefreshSpec(root string, out io.Writer) int        { return writeSpec(root, true, out) }
func EnsureContextStubs(root string, out io.Writer) int { return writeContextStubs(root, out) }

// EnsurePracticesIndex creates practices/INDEX.md if absent. Practices (v0.6)
// are the project's binding answers to recurring mechanisms; scaffolding the
// index makes the directory discoverable rather than something the first
// practice has to invent.
func EnsurePracticesIndex(root string, out io.Writer) int {
	dir := filepath.Join(root, "practices")
	idx := filepath.Join(dir, "INDEX.md")
	if _, err := os.Stat(idx); err == nil {
		return 0
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	body := "# Practices\n\nHow this project does the things it does the same way every time —\nauthorization, permission checks, payment capture, database access. Each is\nbinding on all code it applies to, and changes only with human approval.\n\n* [Format reference](/SPEC.md) - how practices are structured.\n"
	if err := os.WriteFile(idx, []byte(body), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintln(out, "wrote practices/INDEX.md (project practices)")
	return 0
}

// EnsureDebtsIndex creates debts/INDEX.md if absent. The debt register (v0.6)
// records known gaps between what the project says and what the code does.
func EnsureDebtsIndex(root string, out io.Writer) int {
	dir := filepath.Join(root, "debts")
	idx := filepath.Join(dir, "INDEX.md")
	if _, err := os.Stat(idx); err == nil {
		return 0
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	body := "# Debt\n\nKnown gaps between what this project says and what the code does — work\nleft undone, and rules the codebase does not follow everywhere yet. Run\n`fdf debt` to read the register.\n\n* [Format reference](/SPEC.md) - how debts are structured.\n"
	if err := os.WriteFile(idx, []byte(body), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintln(out, "wrote debts/INDEX.md (the debt register)")
	return 0
}

// EnsureBugsIndex creates bugs/INDEX.md if absent. The bug register (v0.7)
// records known defects that have not been repaired yet.
func EnsureBugsIndex(root string, out io.Writer) int {
	dir := filepath.Join(root, "bugs")
	idx := filepath.Join(dir, "INDEX.md")
	if _, err := os.Stat(idx); err == nil {
		return 0
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	body := "# Bugs\n\nKnown defects — the software doing something wrong that someone could\nobserve — that have not been repaired yet. Each is repaired by a Fix or a\nChange that names it in `resolves`. Run `fdf bug` to read the register.\n\n* [Format reference](/SPEC.md) - how bugs are structured.\n"
	if err := os.WriteFile(idx, []byte(body), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintln(out, "wrote bugs/INDEX.md (the bug register)")
	return 0
}

// ensureIndexes creates whichever of the reserved directories' indexes
// `fdf init` scaffolds are absent.
func ensureIndexes(root string, out io.Writer) int {
	for _, ensure := range []func(string, io.Writer) int{EnsureChangesIndex, EnsurePracticesIndex, EnsureDebtsIndex, EnsureBugsIndex} {
		if code := ensure(root, out); code != 0 {
			return code
		}
	}
	return 0
}

// EnsureChangesIndex creates changes/INDEX.md if absent. Post-delivery work
// (v0.5) lives under changes/; scaffolding the index makes the directory
// discoverable in a fresh bundle rather than something the first change has
// to invent.
func EnsureChangesIndex(root string, out io.Writer) int {
	dir := filepath.Join(root, "changes")
	idx := filepath.Join(dir, "INDEX.md")
	if _, err := os.Stat(idx); err == nil {
		return 0
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	body := "# Changes\n\nPost-delivery changes and fixes for delivered features.\nA `Change` alters documented behavior; a `Fix` restores behavior the feature\ndocument already describes. Both may be filed flat here or in groups.\n\n* [Format reference](/SPEC.md) - how changes and fixes are structured.\n"
	if err := os.WriteFile(idx, []byte(body), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintln(out, "wrote changes/INDEX.md (post-delivery changes and fixes)")
	return 0
}

// specVersionRe matches an embedded spec filename stem (spec/<MAJOR.MINOR>.md),
// so spec/README.md is skipped when listing versions.
var specVersionRe = regexp.MustCompile(`^\d+\.\d+$`)

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
		if v != e.Name() && specVersionRe.MatchString(v) {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// SpecText returns the embedded normative text of a spec version, exactly as
// published under spec/ — no bundle frontmatter (that is specDoc's job).
func SpecText(version string) ([]byte, error) {
	if !specVersionRe.MatchString(version) {
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
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	for _, c := range contextDocs {
		path := filepath.Join(root, c.file)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		body := fmt.Sprintf(`---
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

%s`, c.title, "Project "+strings.ToLower(c.title)+" — current snapshot.", now, stubSentinel, c.title, "⚠️", c.purpose, c.headings)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		fmt.Fprintf(out, "wrote %s (context stub — fill via the fdf-init skill)\n", c.file)
	}
	return 0
}

// writeSpec places /SPEC.md. With force=false it is a no-op when a copy
// already exists (init); with force=true it overwrites to the current
// version's spec (migrate), reporting whether it wrote or refreshed.
func writeSpec(root string, force bool, out io.Writer) int {
	specPath := filepath.Join(root, "SPEC.md")
	_, statErr := os.Stat(specPath)
	existed := statErr == nil
	if existed && !force {
		return 0
	}
	doc, err := specDoc()
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := os.WriteFile(specPath, doc, 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	verb := "wrote"
	if existed {
		verb = "refreshed"
	}
	fmt.Fprintf(out, "%s %s (FDF v%s spec copy)\n", verb, filepath.Base(specPath), currentVersion)
	return 0
}

// contextDocNames lists the Context documents for user-facing messages, so a
// new one never leaves a stale literal behind.
// ContextDocNames is contextDocNames for other packages (migrate's next-step
// message), so the list of Context documents has one source.
func ContextDocNames() string { return contextDocNames() }

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

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9-]*$`)
var practiceSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
var practiceGroupedRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)/([a-z0-9][a-z0-9-]*)$`)

// reservedGroups are the bundle-root directories that each hold one kind of
// document, with the command that files one there. A feature's group is any
// other name — and any of these the bundle's pin does not reserve yet
// (ReservedDirs): on a v0.6 bundle, bugs/ is a feature group like any other.
var reservedGroups = map[string]string{
	"changes":   "holds Changes and Fixes (start one with fdf change or fdf fix)",
	"practices": "holds the project's practices (write one with fdf practice <slug>)",
	"debts":     "holds the debt register (file one with fdf debt <slug>)",
	"bugs":      "holds the bug register (file one with fdf bug <slug>)",
	"releases":  "holds the releases (write one with fdf release <version>)",
}

// reservedGroup refuses a feature ID whose group the bundle's pin reserves,
// naming the command that files what belongs there.
func reservedGroup(root, id string, out io.Writer) bool {
	group, _, _ := strings.Cut(id, "/")
	if ReservedDirs(root)[group] {
		fmt.Fprintf(out, "error: %s/ %s; a feature's group is any other name\n", group, reservedGroups[group])
		return true
	}
	return false
}

// Practice scaffolds practices/<id>.md, where id is "<slug>" or
// "<group>/<slug>" — or the full ID, practices/<slug>, which files it in the
// same place. A practice has no trail and no tasks: the document is the whole
// thing, so there is nothing else to create. Practices are v0.6: an older
// pin reads practices/ as a feature group, so the command refuses there.
func Practice(root, id string, out io.Writer) int {
	if !RequirePin(root, 6, "practices", "practices/ is a feature group, and a Practice written there fails validation (F3)", out) {
		return 1
	}
	id = strings.TrimPrefix(id, "practices/")
	if !practiceSlugRe.MatchString(id) && !practiceGroupedRe.MatchString(id) {
		fmt.Fprintf(out, "error: id must be <slug> or <group>/<slug>, lowercase [a-z0-9-]; got %q\n", id)
		return 1
	}
	path := filepath.Join(root, "practices", filepath.FromSlash(id)+".md")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(out, "error: practices/%s.md already exists\n", id)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	slug := id
	if m := practiceGroupedRe.FindStringSubmatch(id); m != nil {
		slug = m[2]
	}
	title := strings.ToUpper(slug[:1]) + strings.ReplaceAll(slug[1:], "-", " ")
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

TODO — why it is this way, so a later change knows what it is trading away.
Optional.
`, title, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if code := EnsurePracticesIndex(root, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "wrote practices/%s.md (type: Practice, status: active)\n", id)
	if code := ListEntry(root, "practices", id, title, "practice", out); code != 0 {
		return code
	}
	fmt.Fprintln(out, "next: fill `# Rules` and set `applies-to` to the paths this governs.")
	fmt.Fprintln(out, "      a practice binds all future code — get human approval before it lands.")
	return 0
}

// pinRe tolerates unquoted pins (`fdf_version: 0.4`), matching how the
// validator and migrate read them.
var pinRe = regexp.MustCompile(`fdf_version:\s*"?([^"\s]+)"?`)

func Init(root string, out io.Writer) int {
	idx := filepath.Join(root, "INDEX.md")
	if raw, err := os.ReadFile(idx); err == nil {
		if m := pinRe.FindSubmatch(raw); m != nil && string(m[1]) == currentVersion {
			// Backfill what a bundle initialized before it existed, or since
			// lost, is missing: the spec copy, Context stubs and indexes.
			if code := writeSpec(root, false, out); code != 0 {
				return code
			}
			if code := writeContextStubs(root, out); code != 0 {
				return code
			}
			if code := ensureIndexes(root, out); code != 0 {
				return code
			}
			fmt.Fprintf(out, "\ndone: bundle at %s was already initialized and up to date (fdf_version %s)\n", root, currentVersion)
			fmt.Fprintln(out, "  nothing was overwritten; only missing files above were added")
			return 0
		} else if m != nil {
			fmt.Fprintf(out, "bundle at %s pins fdf_version %q; run `fdf migrate` to upgrade to %s\n", root, m[1], currentVersion)
			return 1
		}
		fmt.Fprintf(out, "bundle at %s has an INDEX.md without an fdf_version pin; run `fdf migrate`\n", root)
		return 1
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	today := time.Now().UTC().Format("2006-01-02")
	index := fmt.Sprintf("---\nfdf_version: %q\n---\n\n# Feature Bundle\n\nThis bundle conforms to [FDF v%s](/SPEC.md): features in Markdown + Gherkin,\neach with its spec/plan/test trail as stem-named siblings and tasks in a paired directory.\n\n# Overview\n\n* [FDF spec](/SPEC.md) - the format this bundle pins ([upstream](%s)).\n\n# Conventions\n\n* Feature IDs are file paths minus `.md`.\n* Validate with `fdf validate`; see [`LOG.md`](/LOG.md) for change history.\n",
		currentVersion, currentVersion, specURL)
	log := fmt.Sprintf("# Bundle Update Log\n\n## %s\n* **Initialization**: scaffolded by `fdf init` (FDF v%s).\n", today, currentVersion)
	if err := os.WriteFile(idx, []byte(index), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(root, "LOG.md"), []byte(log), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if code := writeSpec(root, false, out); code != 0 {
		return code
	}
	if code := writeContextStubs(root, out); code != 0 {
		return code
	}
	if code := ensureIndexes(root, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "\ndone: initialized FDF bundle at %s\n", root)
	fmt.Fprintf(out, "  pinned fdf_version %s; wrote INDEX.md, LOG.md, SPEC.md and %d Context stub(s)\n", currentVersion, len(contextDocs))
	fmt.Fprintln(out, "next: run the fdf-init skill to fill "+contextDocNames()+" before adding features.")
	fmt.Fprintln(out, "      `fdf validate` warns about them now, and fails F9 once a feature exists while any is unfilled.")
	fmt.Fprintln(out, "      On an existing codebase, map what it already does with `fdf adopt` (the fdf-adopt skill).")
	return 0
}

func New(root, id string, out io.Writer) int {
	if !idRe.MatchString(id) {
		fmt.Fprintf(out, "error: feature id must be <group>/<slug>, lowercase [a-z0-9-]; got %q\n", id)
		return 1
	}
	if reservedGroup(root, id, out) {
		return 1
	}
	group, slug, _ := strings.Cut(id, "/")
	featurePath := filepath.Join(root, group, slug+".md")
	if _, err := os.Stat(featurePath); err == nil {
		fmt.Fprintf(out, "error: %s already exists\n", featurePath)
		return 1
	}
	if err := os.MkdirAll(filepath.Join(root, group), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	title := strings.ToUpper(slug[:1]) + strings.ReplaceAll(slug[1:], "-", " ")
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
	if err := os.WriteFile(featurePath, []byte(feature), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	newGroup, code := appendGroupIndex(root, group, fmt.Sprintf("* [%s](/%s/%s.md) - TODO.\n", title, group, slug), out)
	if code != 0 {
		return code
	}
	fmt.Fprintf(out, "created %s (status: draft)\n", filepath.Join(group, slug+".md"))
	fmt.Fprintf(out, "updated %s (now lists %q)\n", filepath.Join(group, "INDEX.md"), title)
	if newGroup {
		if code := ListGroup(root, "", group, out); code != 0 {
			return code
		}
	}
	fmt.Fprintf(out, "\ndone: feature %s is a draft — one Feature: fence, one Scenario: fence, no trail siblings yet\n", id)
	fmt.Fprintln(out, "next: write the Gherkin, then add "+slug+".spec.md to reach `specified` (the fdf-brainstorm skill drives this).")
	return 0
}

// appendGroupIndex adds one listing line to a group's INDEX.md, creating the
// index when the group is new — which it reports, so the caller can list the
// new group in the root INDEX.md (ListGroup).
func appendGroupIndex(root, group, entry string, out io.Writer) (created bool, code int) {
	gidx := filepath.Join(root, group, "INDEX.md")
	if raw, err := os.ReadFile(gidx); err == nil {
		if err := os.WriteFile(gidx, append(raw, []byte(entry)...), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return false, 1
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(gidx, fmt.Appendf(nil, "# %s\n\n%s", GroupTitle("", group), entry), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return false, 1
		}
		return true, 0
	} else {
		fmt.Fprintln(out, "error:", err)
		return false, 1
	}
	return false, 0
}

// Adopt scaffolds an adopted feature (v0.7) at <group>/<slug>.md: a
// capability the software already has, documented from the code rather than
// built through the lifecycle. It starts as a map entry — a Feature: block
// and the code it lives in, no scenarios — and gets no spec, plan or tasks,
// ever. resources are the project-relative paths of that code; projectRoot,
// when set, is where they must exist. An older pin has no `adopted` status,
// so the command refuses there.
func Adopt(root, projectRoot, id string, resources []string, out io.Writer) int {
	if !RequirePin(root, 7, "adopted features", "`adopted` is not a feature status, and an adopted feature fails validation (F2)", out) {
		return 1
	}
	if !idRe.MatchString(id) {
		fmt.Fprintf(out, "error: feature id must be <group>/<slug>, lowercase [a-z0-9-]; got %q\n", id)
		return 1
	}
	if reservedGroup(root, id, out) {
		return 1
	}
	if len(resources) == 0 {
		fmt.Fprintln(out, "error: --resource is required — name the code this capability lives in; with no tasks, it is the feature's only link to the code")
		return 2
	}
	if projectRoot != "" {
		for _, r := range resources {
			if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(r))); err != nil {
				fmt.Fprintf(out, "error: --resource %s does not exist under %s — adoption documents code that exists\n", r, projectRoot)
				return 1
			}
		}
	}
	group, slug, _ := strings.Cut(id, "/")
	featurePath := filepath.Join(root, group, slug+".md")
	if _, err := os.Stat(featurePath); err == nil {
		fmt.Fprintf(out, "error: %s already exists\n", filepath.Join(group, slug+".md"))
		return 1
	}
	if err := os.MkdirAll(filepath.Join(root, group), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	title := strings.ToUpper(slug[:1]) + strings.ReplaceAll(slug[1:], "-", " ")
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
	if err := os.WriteFile(featurePath, []byte(feature), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	newGroup, code := appendGroupIndex(root, group, fmt.Sprintf("* [%s](/%s/%s.md) - TODO.\n", title, group, slug), out)
	if code != 0 {
		return code
	}
	fmt.Fprintf(out, "created %s (status: adopted — a map entry: a Feature: block and its code, no scenarios yet)\n", filepath.Join(group, slug+".md"))
	fmt.Fprintf(out, "updated %s (now lists %q)\n", filepath.Join(group, "INDEX.md"), title)
	if newGroup {
		if code := ListGroup(root, "", group, out); code != 0 {
			return code
		}
	}
	fmt.Fprintln(out, "\nnext: fill the Feature: block — who uses this, and for what. Scenarios come later, one at a time,")
	fmt.Fprintln(out, "      each with its case in "+slug+".test.md, passing against the code as it stands (the fdf-adopt skill).")
	return 0
}
