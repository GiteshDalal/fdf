// Package bundle validates an FDF bundle against the spec version it pins,
// which is 1.0: a bundle that pins none, or one this fdf does not validate,
// fails F1 and is checked no further. Rules F1-F14 are format conformance;
// R1 is repo integrity. See SPEC.md.
package bundle

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/links"
	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

type Options struct {
	RepoRoot string    // project root for R1; "" = standalone (skip R1, warn)
	Out      io.Writer // defaults to os.Stdout
	// FreshStubsAdvisory downgrades F9 "unfilled context stub" from error to
	// warning even when features exist. `fdf migrate` sets it: it has just
	// created the stubs and cannot fill them (only the fdf-init interview
	// can), so unfilled stubs are expected post-migration state, not a
	// migration failure. Plain `fdf validate` leaves it false and enforces F9.
	FreshStubsAdvisory bool
	// StrictDomain promotes F12's banned-word warnings to errors. Off by
	// default: the lexicon check is the one rule that matches natural
	// language, so a young bundle adopting a lexicon would otherwise be
	// unable to validate until every document had been reworded. A bundle
	// can also turn it on for itself with `strict: true` in DOMAIN.md.
	StrictDomain bool
}

// domainReportCap bounds F12's per-document lines in `fdf validate` output:
// a bundle that has just banned a word can report hundreds of documents, and
// the full list is `fdf lexicon`'s job.
const domainReportCap = 20

var (
	// structural types are rejected at the bundle root: each has a position
	// of its own in a register.
	structural  = map[string]bool{"Feature": true, "Spec": true, "Plan": true, "Task": true, "Test": true, "Release": true, "Surface": true, "Log": true, "Change": true, "Fix": true, "Practice": true, "Debt": true, "Bug": true}
	recommended = []string{"title", "description", "timestamp"}
	// fenceOpenRe/inlineCodeRe strip code from a text whose markers and
	// headings are read (linkScanText): an example quotes them.
	fenceOpenRe   = regexp.MustCompile("^(`{3,})")
	inlineCodeRe  = regexp.MustCompile("`[^`\n]*`")
	isoDateRe     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	taskFileRe    = regexp.MustCompile(`^\d{2}-[a-z0-9][a-z0-9-]*\.md$`)
	trailRoleType = map[string]string{
		"spec":    "Spec",
		"plan":    "Plan",
		"test":    "Test",
		"surface": "Surface",
		"log":     "Log",
	}
	fenceRe       = regexp.MustCompile("(?s)```gherkin[ \t]*\r?\n(.*?)```")
	gherkinStart  = regexp.MustCompile(`^\s*(Feature:|Scenario:|Scenario Outline:|Background:|Rule:|@)`)
	featureDeclRe = regexp.MustCompile(`(?m)^\s*Feature:`)
	scenarioRe    = regexp.MustCompile(`(?m)^\s*Scenario(?: Outline)?:\s*(\S[^\n]*)`)
	headingRe     = regexp.MustCompile(`^#\s+(.*\S)\s*$`)
	logDateRe     = regexp.MustCompile(`^##\s+(.*\S)\s*$`)
)

var featureStatuses = []string{"draft", "specified", "planned", "implementing", "done", "adopted", "retired"}

// changeStatuses is the Change/Fix vocabulary (v0.5); it mirrors the feature
// lifecycle minus "retired" — a change is work, not a capability.
var changeStatuses = []string{"draft", "specified", "planned", "implementing", "done"}
var taskStatuses = []string{"pending", "in-progress", "done"}
var practiceStatuses = []string{"active", "superseded"}

// debtStatuses is the vocabulary of both registers: Debt (v0.6) and Bug
// (v0.7) share it, and with it the reason for `accepted`.
var debtStatuses = []string{"open", "accepted", "resolved"}
var releaseStatuses = []string{"planned", "shipped"}

// supportedVersions are the spec versions this validator checks: those of
// spec 1. A bundle pinned to any other is not validated (pinProblem).
var supportedVersions = map[string]bool{"1.0": true}

// supportedList renders supportedVersions for error messages, so adding a
// version never leaves a stale literal behind.
func supportedList() string {
	vs := make([]string, 0, len(supportedVersions))
	for v := range supportedVersions {
		vs = append(vs, v)
	}
	specver.Sort(vs)
	return strings.Join(vs, ", ")
}

// pinProblem is F1's error for a root INDEX.md whose pin this validator
// cannot check a bundle against, naming the fix, or "" for a supported pin:
// a missing pin is written in, one that is not a MAJOR.MINOR version is
// corrected, a version newer than every one this fdf validates takes a newer
// fdf, and an older one, a 0.x bundle, `fdf migrate`.
func pinProblem(pin string) string {
	if supportedVersions[pin] {
		return ""
	}
	v, ok := specver.Parse(pin)
	switch {
	case pin == "":
		return fmt.Sprintf("INDEX.md: pins no fdf_version — the version of the spec a bundle follows is pinned in its frontmatter, as fdf_version: \"%s\"; a bundle from before 1.0 is upgraded with `fdf migrate` (F1)", newestSupported())
	case !ok:
		return fmt.Sprintf("INDEX.md: fdf_version %q is not a MAJOR.MINOR version such as %q — correct the pin (F1)", pin, newestSupported())
	}
	for s := range supportedVersions {
		if w, _ := specver.Parse(s); !w.Less(v) {
			return fmt.Sprintf("INDEX.md: fdf_version %q is not a supported version (%s) — run `fdf migrate` (F1)", pin, supportedList())
		}
	}
	return fmt.Sprintf("INDEX.md: fdf_version %q is newer than any version this fdf validates (%s) — upgrade fdf (F1)", pin, supportedList())
}

// newestSupported is the newest version this validator checks.
func newestSupported() string {
	vs := strings.Split(supportedList(), ", ")
	return vs[len(vs)-1]
}

// isStub reports whether a Context document still carries the stub marker
// outside code — a filled document may quote the marker in a code span.
func isStub(text string) bool { return strings.Contains(linkScanText(text), stubSentinel) }

// checkLogBody validates the `## <date>` headings of a LOG.md or a
// slug.log.md: each an ISO-8601 date, a real one, as a `timestamp` is, and
// newest first. A `##` line inside a fenced block is an example an entry
// quotes, not a heading.
func checkLogBody(rel, text string, errs, warns *[]string) {
	var dates []string
	for _, line := range strings.Split(linkScanText(text), "\n") {
		if m := logDateRe.FindStringSubmatch(line); m != nil {
			h := strings.TrimSpace(m[1])
			_, perr := time.Parse(time.DateOnly, h)
			switch {
			case !isoDateRe.MatchString(h):
				*errs = append(*errs, fmt.Sprintf("%s: log date heading '## %s' is not ISO-8601 (F1)", rel, h))
			case perr != nil:
				*errs = append(*errs, fmt.Sprintf("%s: log date heading '## %s' is not a real date (F1)", rel, h))
			default:
				dates = append(dates, h)
			}
		}
	}
	if !sort.SliceIsSorted(dates, func(i, j int) bool { return dates[i] > dates[j] }) {
		*warns = append(*warns, fmt.Sprintf("%s: log entries are not in newest-first order", rel))
	}
}

// linkScanText returns text with fenced code blocks and inline code spans
// removed. Links inside them are examples — the vendored SPEC.md shows a plan
// linking its tasks and a release linking its features — and checking them
// would warn about files no bundle is expected to have.
func linkScanText(text string) string {
	var b strings.Builder
	fence := ""
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimRight(strings.TrimLeft(line, " \t"), " \t")
		if fence != "" {
			// A closing fence is all backticks and at least as long as the
			// opener, so a ``` inside a ```` block does not end it.
			if strings.HasPrefix(t, fence) && strings.Trim(t, "`") == "" {
				fence = ""
			}
			continue
		}
		if m := fenceOpenRe.FindStringSubmatch(t); m != nil {
			fence = m[1]
			continue
		}
		b.WriteString(inlineCodeRe.ReplaceAllString(line, " "))
		b.WriteString("\n")
	}
	return b.String()
}

const stubSentinel = "<!-- fdf:stub -->"

// placeholderRe finds the text fdf's scaffolds write for a person to replace:
// a line, list item, declaration or frontmatter value beginning "TODO —", a
// regression case whose verification is still "TODO", and a group index entry
// still reading "- TODO". It runs on the text with code
// stripped (linkScanText), so an example that quotes a placeholder is not one.
// gherkinPlaceholderRe finds the scaffold's "As a <role>", which lives inside a
// Gherkin fence and so is matched on the raw text. An advisory: a scaffold
// passes every structural rule, and without this an untouched one reads as
// done.
var placeholderRe = regexp.MustCompile(`(?m)^\s*(?:[-*]\s+)?(?:\d+\.\s+)?(?:[a-z-]+:\s*)?TODO —|\) - TODO\.|\s(?:—|–|--)\s+TODO\b`)
var gherkinPlaceholderRe = regexp.MustCompile(`(?m)^\s*As an? <role>\s*$`)

// readPin returns the fdf_version pinned by the bundle's root INDEX.md, or ""
// if absent/unreadable. It runs before the walk: the pin decides whether the
// bundle is validated at all.
func readPin(rootAbs string) string {
	raw, err := os.ReadFile(filepath.Join(rootAbs, "INDEX.md"))
	if err != nil {
		return ""
	}
	block, delimited, _ := splitFrontmatter(strings.TrimPrefix(string(raw), "\uFEFF"))
	if !delimited {
		return ""
	}
	data, _ := parseFrontmatter(block)
	if data == nil {
		return ""
	}
	v, _ := data["fdf_version"].(string)
	return v
}

func in(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

type featureInfo struct {
	rel, status, version, body string
	resource                   []string // the code an adopted feature documents
	replacedBy                 []string // the feature that replaces a retired one
	surface                    string   // `none` when the feature has no interface
}
type pairInfo struct {
	spec, plan, test  bool
	surface, log      bool
	planRel, planBody string
	testBody          string
	testTimestamp     string
	tasks             map[string]string   // filename -> status
	deps              map[string][]string // filename -> depends-on
	depRels           map[string]string   // filename -> rel (for messages)
}
type releaseInfo struct{ rel, status, body string }

// Validate checks the bundle rooted at root and returns 0 or 1. A directory
// with no INDEX.md is no bundle. A bundle whose pin this fdf does not
// validate fails F1 and is checked no further: its rules are another
// version's, or it has none.
func Validate(root string, opts Options) int {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil || !isDir(rootAbs) || !exists(filepath.Join(rootAbs, "INDEX.md")) {
		fmt.Fprintln(out, "error:", fdfroot.NoBundle(root))
		return 1
	}
	pinnedVer := readPin(rootAbs)
	if problem := pinProblem(pinnedVer); problem != "" {
		fmt.Fprintf(out, "FAIL: %s\n\nBundle not validated: fdf checks a bundle against the version its INDEX.md pins, and validates %s.\n", problem, supportedList())
		return 1
	}

	var errs, repoErrs, warns []string
	var crossLinks []crossLink
	var resources []resourceRef
	features := map[string]*featureInfo{}
	pairs := map[string]*pairInfo{}
	releases := map[string]*releaseInfo{}
	changes := map[string]*changeInfo{}
	practices := map[string]*practiceInfo{}
	practiceTrails := map[string]string{} // practice ID -> rel of its .log.md
	debts := map[string]*debtInfo{}
	debtTrails := map[string]string{} // debt ID -> rel of its .log.md
	bugs := map[string]*bugInfo{}
	bugTrails := map[string]string{} // bug ID -> rel of its .log.md
	texts := map[string]string{}     // every .md file's text, for F12's scan
	featureDeps := map[string][]string{}
	documents := 0
	// contextDocs[name] records a seen root Context document and whether it is
	// still an unfilled stub, for F9.
	contextDocs := map[string]bool{} // name -> isStub

	c := &collection{
		errs: &errs, warns: &warns, crossLinks: &crossLinks, resources: &resources, documents: &documents,
		features: features, featureDeps: featureDeps, pairs: pairs, releases: releases, changes: changes,
		practices: practices, practiceTrails: practiceTrails, debts: debts, debtTrails: debtTrails,
		bugs: bugs, bugTrails: bugTrails, texts: texts, contextDocs: contextDocs,
	}
	c.walk(rootAbs)

	// F3: every pair entry has a sibling feature (or, under changes/, a
	// sibling Change/Fix) document.
	for fid := range pairs {
		if strings.HasPrefix(fid, "changes/") {
			if changes[fid] == nil {
				errs = append(errs, fmt.Sprintf("%s: trail file or task directory has no sibling %s.md Change or Fix (F3)", fid, fid))
			}
			continue
		}
		if features[fid] == nil {
			errs = append(errs, fmt.Sprintf("%s: trail file or task directory has no sibling feature document %s.md (F3)", fid, fid))
		}
	}

	// F4 + F8: status <-> artifact invariants.
	for fid, f := range features {
		p := pairs[fid]
		switch {
		case f.status == "draft":
			// A draft may have a log: what happened to a feature is worth
			// recording before its design is approved. Nothing else.
			if p != nil && (p.spec || p.plan || p.test || p.surface || len(p.tasks) > 0) {
				errs = append(errs, fmt.Sprintf("%s: status 'draft' but trail siblings or task directory %s/ exist (F4)", f.rel, fid))
			}
			continue
		case f.status == "adopted":
			checkAdopted(fid, f, p, &errs, &warns)
			continue
		case f.status == "retired":
			// The behavior is gone; the document stays as the record, and F8
			// live-test coverage no longer applies — that is the point of
			// retiring. A retired feature keeps the trail it had: one that was
			// adopted never had a spec, so only a built one (plan or tasks)
			// needs it.
			built := p != nil && (p.plan || len(p.tasks) > 0)
			if built && !p.spec {
				errs = append(errs, fmt.Sprintf("%s: status 'retired' with a build trail requires %s.spec.md — the record of what was built (F4)", f.rel, fid))
			}
			// A retired map entry may have no scenarios (F5 let it off); a
			// feature that was built keeps the scenarios it had.
			if built && len(scenarioRe.FindAllString(f.body, -1)) == 0 {
				errs = append(errs, fmt.Sprintf("%s: no `Scenario:` in any gherkin fence — a retired feature that was built keeps the scenarios it had (F5)", f.rel))
			}
			continue
		case !in(featureStatuses, f.status):
			continue // already reported (F2)
		case p == nil:
			// No trail at all: fall through to the per-artifact checks below
			// (against an empty pairInfo) so the error names the specific
			// missing file(s) per the F4 rule catalog.
			p = &pairInfo{tasks: map[string]string{}, deps: map[string][]string{}, depRels: map[string]string{}}
		}
		specPath, planPath, testPath := fid+".spec.md", fid+".plan.md", fid+".test.md"
		if !p.spec {
			errs = append(errs, fmt.Sprintf("%s: status '%s' requires %s (F4)", f.rel, f.status, specPath))
		}
		planned := in([]string{"planned", "implementing", "done"}, f.status)
		if planned && !p.plan {
			errs = append(errs, fmt.Sprintf("%s: status '%s' requires %s (F4)", f.rel, f.status, planPath))
		}
		if planned { // F8
			if !p.test {
				errs = append(errs, fmt.Sprintf("%s: status '%s' requires %s (F8)", f.rel, f.status, testPath))
			} else {
				checkTestCases(testPath, fid, featureScenarioNames(f.body), p.testBody, &errs, &warns)
			}
		}
		nTasks, nDone := len(p.tasks), 0
		for _, s := range p.tasks {
			if s == "done" {
				nDone++
			}
		}
		if in([]string{"implementing", "done"}, f.status) && nTasks == 0 {
			errs = append(errs, fmt.Sprintf("%s: status '%s' requires at least one task (F4)", f.rel, f.status))
		}
		if f.status == "implementing" && nTasks > 0 && nDone == nTasks {
			errs = append(errs, fmt.Sprintf("%s: status 'implementing' but every task is done — promote to 'done' (F4)", f.rel))
		}
		if f.status == "done" && nDone != nTasks {
			var open []string
			for n, s := range p.tasks {
				if s != "done" {
					open = append(open, n)
				}
			}
			sort.Strings(open)
			errs = append(errs, fmt.Sprintf("%s: status 'done' but tasks not done: %s (F4)", f.rel, strings.Join(open, ", ")))
		}
	}

	checkChangeLifecycle(changes, pairs, &errs)
	checkChangeIntegrity(changes, features, pairs, &errs)
	checkRegressionLanded(changes, pairs, &warns)
	checkSurfaces(features, pairs, &errs, &warns)
	// Feature depends-on: existing IDs, acyclic. A separate graph from the
	// task one — these are feature IDs, not siblings.
	for fid, deps := range featureDeps {
		for _, d := range deps {
			if features[d] == nil {
				errs = append(errs, fmt.Sprintf("%s: depends-on %q is not a known feature%s (F10)", features[fid].rel, d, featureHint(d, features)))
			}
		}
	}
	if cyc := findCycleIDs(featureDeps); cyc != "" {
		errs = append(errs, fmt.Sprintf("feature depends-on cycle: %s (F10)", cyc))
	}

	checkPracticeIntegrity(practices, practiceTrails, &errs, &warns)
	checkDebtIntegrity(debts, debtTrails, &errs, &warns)
	resolvedBy := checkResolves(changes, bugs, clearedBugs(rootAbs), &errs)
	checkBugIntegrity(bugs, bugTrails, features, resolvedBy, &errs, &warns)
	checkDomain(rootAbs, opts.StrictDomain, texts, &errs, &warns)

	// F9: from the first feature onward, the five Context documents must
	// exist and be past their stub state. Before any feature exists a
	// missing-or-stub context doc is only a nudge (warning).
	for _, name := range []string{"STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md", "DOMAIN.md"} {
		stub, present := contextDocs[name]
		switch {
		case !present && len(features) > 0:
			errs = append(errs, fmt.Sprintf("%s: required once the bundle has features — run the fdf-init interview to create it (F9)", name))
		case stub && len(features) > 0 && opts.FreshStubsAdvisory:
			warns = append(warns, fmt.Sprintf("%s: freshly scaffolded stub — run the fdf-init interview to populate it (F9)", name))
		case stub && len(features) > 0:
			errs = append(errs, fmt.Sprintf("%s: still an unfilled stub; the fdf-init interview must populate it before feature work (F9)", name))
		case !present:
			warns = append(warns, fmt.Sprintf("%s: recommended context document is missing — run the fdf-init interview", name))
		case stub:
			warns = append(warns, fmt.Sprintf("%s: still an unfilled stub — run the fdf-init interview to populate it", name))
		}
	}

	// F6: plan <-> tasks, depends-on resolution, acyclicity.
	for fid, p := range pairs {
		if p.plan {
			listed := map[string]bool{}
			taskDir := filepath.Join(rootAbs, filepath.FromSlash(fid))
			for _, t := range sectionTargets(p.planBody, "Tasks") {
				// A link lists a task only if it reaches this plan's task
				// directory: a same-named task elsewhere is not this one.
				if resolved := resolveLink(rootAbs, p.planRel, t); resolved != "" && filepath.Dir(resolved) == taskDir {
					listed[filepath.Base(resolved)] = true
				}
			}
			for name := range p.tasks {
				if !listed[name] {
					errs = append(errs, fmt.Sprintf("%s: `# Tasks` does not link %s (F6)", p.planRel, name))
				}
			}
			for name := range listed {
				if strings.HasSuffix(name, ".md") && p.tasks[name] == "" && !fileExistsIn(rootAbs, fid, name) {
					errs = append(errs, fmt.Sprintf("%s: `# Tasks` links %s, which does not exist (F6)", p.planRel, name))
				}
			}
		} else if len(p.tasks) > 0 {
			errs = append(errs, fmt.Sprintf("tasks exist under %s/ but %s.plan.md is missing (F6)", fid, fid))
		}
		for name, deps := range p.deps {
			for _, dep := range deps {
				if _, ok := p.tasks[dep+".md"]; !ok {
					errs = append(errs, fmt.Sprintf("%s: depends-on %q is not a sibling task (F6)", p.depRels[name], dep))
				}
			}
		}
		if cyc := findCycle(p.deps); cyc != "" {
			errs = append(errs, fmt.Sprintf("%s/: depends-on cycle: %s (F6)", fid, cyc))
		}
	}

	// F7: release <-> version bidirectional consistency.
	for version, r := range releases {
		for _, t := range sectionTargets(r.body, "Features") {
			resolved := resolveLink(rootAbs, r.rel, t)
			if resolved == "" {
				continue
			}
			fid := strings.TrimSuffix(filepath.ToSlash(relTo(rootAbs, resolved)), ".md")
			f := features[fid]
			if f == nil {
				errs = append(errs, fmt.Sprintf("%s: `# Features` links unknown feature %q (F7)", r.rel, t))
				continue
			}
			if f.version != version {
				got := f.version
				if got == "" {
					got = "unset"
				}
				errs = append(errs, fmt.Sprintf("%s: lists %s whose `version` is %s, expected %q (F7)", r.rel, fid, got, version))
			}
			// A feature retired after it shipped still shipped.
			if r.status == "shipped" && f.status != "done" && f.status != "retired" {
				errs = append(errs, fmt.Sprintf("%s: shipped release lists %s with status '%s' (F7)", r.rel, fid, f.status))
			}
		}
	}
	checkReleaseChanges(rootAbs, releases, changes, &errs)
	for fid, f := range features {
		if f.version == "" {
			continue
		}
		r := releases[f.version]
		if r == nil {
			errs = append(errs, fmt.Sprintf("%s: version %q has no releases/%s.md (F7)", f.rel, f.version, f.version))
			continue
		}
		found := false
		for _, t := range sectionTargets(r.body, "Features") {
			if resolved := resolveLink(rootAbs, r.rel, t); resolved != "" &&
				strings.TrimSuffix(filepath.ToSlash(relTo(rootAbs, resolved)), ".md") == fid {
				found = true
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("%s: version %q but not listed in %s (F7)", f.rel, f.version, r.rel))
		}
	}

	// Soft: broken cross-links.
	for _, l := range crossLinks {
		resolved := resolveLink(rootAbs, l.src, l.target)
		if resolved == "" {
			continue
		}
		if !exists(resolved) && !exists(filepath.Join(resolved, "INDEX.md")) {
			warns = append(warns, fmt.Sprintf("%s: broken cross-link -> %s", l.src, l.target))
		}
	}

	// R1: task resource paths exist in the project.
	if opts.RepoRoot == "" {
		if len(resources) > 0 {
			warns = append(warns, "standalone bundle (no project root): skipping R1 resource checks")
		}
	} else {
		for _, r := range resources {
			if !exists(filepath.Join(opts.RepoRoot, r.path)) {
				repoErrs = append(repoErrs, fmt.Sprintf("%s: `%s` path does not exist in repo -> %s (R1)", r.rel, r.field, r.path))
			}
		}
	}

	sort.Strings(errs)
	for _, w := range warns {
		fmt.Fprintf(out, "warn: %s\n", w)
	}
	for _, e := range errs {
		fmt.Fprintf(out, "FAIL: %s\n", e)
	}
	for _, e := range repoErrs {
		fmt.Fprintf(out, "FAIL: %s\n", e)
	}
	fmt.Fprintf(out, "\n%d document(s), %d feature(s), %d release(s) checked; %d error(s), %d warning(s).\n",
		documents, len(features), len(releases), len(errs)+len(repoErrs), len(warns))
	if len(errs) > 0 {
		fmt.Fprintf(out, "Bundle is NOT conformant with FDF v%s.\n", pinnedVer)
		return 1
	}
	if len(repoErrs) > 0 {
		fmt.Fprintln(out, "Bundle is FDF-conformant but has drifted from the repo (R1).")
		return 1
	}
	if opts.RepoRoot == "" {
		fmt.Fprintf(out, "Bundle is conformant with FDF v%s (R1 skipped: no project root).\n", pinnedVer)
		return 0
	}
	fmt.Fprintf(out, "Bundle is conformant with FDF v%s; repo integrity (R1) verified.\n", pinnedVer)
	return 0
}

// checkFeatureBody is F5. mapEntry allows a feature with no scenarios, which
// is legal only for an adopted feature.
func checkFeatureBody(rel, body string, mapEntry bool, errs *[]string) {
	fences := fenceRe.FindAllStringSubmatch(body, -1)
	if len(fences) == 0 {
		*errs = append(*errs, fmt.Sprintf("%s: Feature document has no ```gherkin fence (F5)", rel))
		return
	}
	decls, scenarios := 0, 0
	for _, f := range fences {
		first := ""
		for _, ln := range strings.Split(f[1], "\n") {
			if strings.TrimSpace(ln) != "" {
				first = ln
				break
			}
		}
		if !gherkinStart.MatchString(first) {
			*errs = append(*errs, fmt.Sprintf("%s: gherkin fence does not start with a Gherkin keyword (F5): %.50q", rel, strings.TrimSpace(first)))
		}
		decls += len(featureDeclRe.FindAllString(f[1], -1))
		scenarios += len(scenarioRe.FindAllString(f[1], -1))
	}
	if decls != 1 {
		*errs = append(*errs, fmt.Sprintf("%s: expected exactly one `Feature:` declaration across gherkin fences, found %d (F5)", rel, decls))
	}
	if scenarios < 1 && !mapEntry {
		*errs = append(*errs, fmt.Sprintf("%s: no `Scenario:` in any gherkin fence (F5)", rel))
	}
}

// findCycle DFS-detects a cycle in the sibling depends-on graph; returns a
// readable "a -> b -> a" chain or "".
func findCycle(deps map[string][]string) string {
	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := map[string]int{}
	var chain []string
	var visit func(n string) string
	visit = func(n string) string {
		color[n] = grey
		chain = append(chain, strings.TrimSuffix(n, ".md"))
		for _, d := range deps[n] {
			key := d + ".md"
			if _, known := deps[key]; !known {
				continue // dangling dep: reported separately
			}
			switch color[key] {
			case grey:
				return strings.Join(append(chain, d), " -> ")
			case white:
				if c := visit(key); c != "" {
					return c
				}
			}
		}
		chain = chain[:len(chain)-1]
		color[n] = black
		return ""
	}
	var names []string
	for n := range deps {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if color[n] == white {
			chain = nil
			if c := visit(n); c != "" {
				return c
			}
		}
	}
	return ""
}

// sectionTargets returns the targets of the links the links engine finds in
// each `# <heading>` section of a body, outside code. A title after a
// target, a destination in angle brackets and a reference definition read
// as they do everywhere else. It reads the body as written, so a heading
// line that starts in code, such as one a fenced sample quotes, neither
// opens a section nor closes one.
func sectionTargets(body, heading string) []string {
	code := links.Code(body)
	inCode := func(at int) bool {
		for _, s := range code {
			if s.Start <= at && at < s.End {
				return true
			}
		}
		return false
	}
	var sections [][2]int // byte ranges; an end of -1 runs to the end of body
	pos := 0
	for _, line := range strings.SplitAfter(body, "\n") {
		if m := headingRe.FindStringSubmatch(strings.TrimRight(line, "\r\n")); m != nil && !inCode(pos) {
			if n := len(sections); n > 0 && sections[n-1][1] < 0 {
				sections[n-1][1] = pos
			}
			if strings.EqualFold(strings.TrimSpace(m[1]), heading) {
				sections = append(sections, [2]int{pos + len(line), -1})
			}
		}
		pos += len(line)
	}
	var out []string
	for _, l := range links.Find(body) {
		for _, s := range sections {
			if !l.InCode && l.Start >= s[0] && (s[1] < 0 || l.Start < s[1]) {
				out = append(out, l.Target)
			}
		}
	}
	return out
}

// resolveLink returns the absolute path a link written in srcRel names, or
// "" when its target is not a path. The links engine reads the target, as it
// does the links it repairs.
func resolveLink(root, srcRel, target string) string {
	d, ok := links.Resolve(target, filepath.ToSlash(srcRel), "")
	if !ok {
		return ""
	}
	return filepath.Join(root, filepath.FromSlash(d.Path))
}

func relTo(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return r
}

func isDir(p string) bool  { fi, err := os.Stat(p); return err == nil && fi.IsDir() }
func exists(p string) bool { _, err := os.Stat(p); return err == nil }
func fileExistsIn(root, fid, name string) bool {
	return exists(filepath.Join(root, filepath.FromSlash(fid), name))
}
