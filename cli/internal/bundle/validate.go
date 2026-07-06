// Package bundle validates an FDF v0.2 bundle. Rules F1-F8 are format
// conformance; R1 is repo integrity. See SPEC.md.
package bundle

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Options struct {
	RepoRoot string    // project root for R1; "" = standalone (skip R1, warn)
	Out      io.Writer // defaults to os.Stdout
}

var (
	reserved      = map[string]bool{"INDEX.md": true, "LOG.md": true}
	trailNames    = map[string]string{"SPEC.md": "Spec", "PLAN.md": "Plan", "TEST.md": "Test"}
	structural    = map[string]bool{"Feature": true, "Spec": true, "Plan": true, "Task": true, "Test": true, "Release": true}
	recommended   = []string{"title", "description", "timestamp"}
	linkRe        = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	isoDateRe     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	taskFileRe    = regexp.MustCompile(`^\d{2}-[a-z0-9][a-z0-9-]*\.md$`)
	lowerFileRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*\.md$`)
	lowerDirRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	fenceRe       = regexp.MustCompile("(?s)```gherkin[ \t]*\r?\n(.*?)```")
	gherkinStart  = regexp.MustCompile(`^\s*(Feature:|Scenario:|Scenario Outline:|Background:|Rule:|@)`)
	featureDeclRe = regexp.MustCompile(`(?m)^\s*Feature:`)
	scenarioRe    = regexp.MustCompile(`(?m)^\s*Scenario(?: Outline)?:\s*(\S[^\n]*)`)
	headingRe     = regexp.MustCompile(`^#\s+(.*\S)\s*$`)
	logDateRe     = regexp.MustCompile(`^##\s+(.*\S)\s*$`)
)

var featureStatuses = []string{"draft", "specified", "planned", "implementing", "done"}
var taskStatuses = []string{"pending", "in-progress", "done"}
var releaseStatuses = []string{"planned", "shipped"}

func in(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

type featureInfo struct{ rel, status, version, body string }
type pairInfo struct {
	spec, plan, test  bool
	planRel, planBody string
	testBody          string
	tasks             map[string]string   // filename -> status
	deps              map[string][]string // filename -> depends-on
	depRels           map[string]string   // filename -> rel (for messages)
}
type releaseInfo struct{ rel, status, body string }

// Validate checks the bundle rooted at root and returns 0 or 1.
func Validate(root string, opts Options) int {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil || !isDir(rootAbs) {
		fmt.Fprintf(out, "error: %s is not a directory\n", root)
		return 1
	}

	var errs, repoErrs, warns []string
	type link struct{ src, target string }
	var links []link
	var resources []struct{ rel, path string }
	features := map[string]*featureInfo{}
	pairs := map[string]*pairInfo{}
	releases := map[string]*releaseInfo{}
	documents := 0

	pair := func(fid string) *pairInfo {
		if pairs[fid] == nil {
			pairs[fid] = &pairInfo{tasks: map[string]string{}, deps: map[string][]string{}, depRels: map[string]string{}}
		}
		return pairs[fid]
	}

	filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && path != rootAbs {
				base := filepath.Base(path)
				if base != "releases" && !lowerDirRe.MatchString(base) {
					errs = append(errs, fmt.Sprintf("%s/: directory names must be lowercase [a-z0-9-] (F3)", relTo(rootAbs, path)))
				}
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".md") {
			return nil // non-markdown files are outside FDF's rules
		}
		rel := relTo(rootAbs, path)
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			warns = append(warns, fmt.Sprintf("%s: could not read file (%v)", rel, rerr))
			return nil
		}
		text := strings.TrimPrefix(string(raw), "\uFEFF")
		for _, m := range linkRe.FindAllStringSubmatch(text, -1) {
			links = append(links, link{rel, m[1]})
		}

		// README.md at bundle root: legal, ignored.
		if rel == "README.md" {
			return nil
		}

		if reserved[name] {
			block, delimited, _ := splitFrontmatter(text)
			if name == "INDEX.md" {
				if delimited {
					data, _ := parseFrontmatter(block)
					if rel != "INDEX.md" || data == nil || data["fdf_version"] == nil {
						warns = append(warns, fmt.Sprintf("%s: index file should not carry frontmatter", rel))
					} else if v, _ := data["fdf_version"].(string); v != "0.2" {
						errs = append(errs, fmt.Sprintf("INDEX.md: fdf_version %q is not \"0.2\" — run `fdf migrate` (F1)", v))
					}
				} else if rel == "INDEX.md" {
					warns = append(warns, "INDEX.md: root index should pin fdf_version")
				}
				if !regexp.MustCompile(`(?m)^\s*[-*]\s+\[.*\]\(.*\)`).MatchString(text) {
					warns = append(warns, fmt.Sprintf("%s: index file has no bulleted listing", rel))
				}
			} else { // LOG.md
				var dates []string
				for _, line := range strings.Split(text, "\n") {
					if m := logDateRe.FindStringSubmatch(line); m != nil {
						h := strings.TrimSpace(m[1])
						if isoDateRe.MatchString(h) {
							dates = append(dates, h)
						} else {
							errs = append(errs, fmt.Sprintf("%s: log date heading '## %s' is not ISO-8601 (F1)", rel, h))
						}
					}
				}
				if !sort.SliceIsSorted(dates, func(i, j int) bool { return dates[i] > dates[j] }) {
					warns = append(warns, fmt.Sprintf("%s: log entries are not in newest-first order", rel))
				}
			}
			return nil
		}

		// Casing gate for everything non-reserved.
		if _, isTrail := trailNames[name]; !isTrail {
			if !lowerFileRe.MatchString(name) && !taskFileRe.MatchString(name) {
				errs = append(errs, fmt.Sprintf("%s: filenames are lowercase; uppercase is reserved for INDEX/LOG/SPEC/PLAN/TEST.md (F3)", rel))
				return nil
			}
		}

		documents++
		block, delimited, body := splitFrontmatter(text)
		if !delimited {
			errs = append(errs, fmt.Sprintf("%s: missing or unterminated frontmatter block (F1)", rel))
			return nil
		}
		data, perr := parseFrontmatter(block)
		if perr != nil {
			errs = append(errs, fmt.Sprintf("%s: frontmatter is not parseable (F1): %v", rel, perr))
			return nil
		}
		docType, _ := data["type"].(string)
		if docType == "" {
			errs = append(errs, fmt.Sprintf("%s: missing required non-empty `type` (F1)", rel))
		}
		for _, f := range recommended {
			if data[f] == nil {
				warns = append(warns, fmt.Sprintf("%s: missing recommended `%s`", rel, f))
			}
		}
		status, _ := data["status"].(string)

		parts := strings.Split(filepath.ToSlash(rel), "/")
		switch {
		case len(parts) == 1: // bundle root
			if structural[docType] {
				errs = append(errs, fmt.Sprintf("%s: `type: %s` documents cannot live at the bundle root — move it to its FDF position (F3)", rel, docType))
			}
		case parts[0] == "releases" && len(parts) == 2:
			if docType != "Release" {
				errs = append(errs, fmt.Sprintf("%s: expected `type: Release`, got %q (F3)", rel, docType))
			}
			if !in(releaseStatuses, status) {
				errs = append(errs, fmt.Sprintf("%s: Release `status` must be one of %s, got %q (F2)", rel, strings.Join(releaseStatuses, "|"), status))
			}
			if data["date"] == nil {
				warns = append(warns, fmt.Sprintf("%s: missing recommended `date`", rel))
			}
			releases[strings.TrimSuffix(parts[1], ".md")] = &releaseInfo{rel, status, body}
		case len(parts) == 2: // feature
			if docType != "Feature" {
				errs = append(errs, fmt.Sprintf("%s: expected `type: Feature`, got %q (F3)", rel, docType))
			}
			if !in(featureStatuses, status) {
				errs = append(errs, fmt.Sprintf("%s: Feature `status` must be one of %s, got %q (F2)", rel, strings.Join(featureStatuses, "|"), status))
			}
			version, _ := data["version"].(string)
			features[strings.TrimSuffix(filepath.ToSlash(rel), ".md")] = &featureInfo{rel, status, version, body}
			checkFeatureBody(rel, body, &errs)
		case len(parts) == 3:
			fid := parts[0] + "/" + parts[1]
			if want, isTrail := trailNames[parts[2]]; isTrail {
				if docType != want {
					errs = append(errs, fmt.Sprintf("%s: expected `type: %s`, got %q (F3)", rel, want, docType))
				}
				p := pair(fid)
				switch parts[2] {
				case "SPEC.md":
					p.spec = true
				case "PLAN.md":
					p.plan, p.planRel, p.planBody = true, rel, body
				case "TEST.md":
					p.test, p.testBody = true, body
				}
			} else if taskFileRe.MatchString(parts[2]) {
				if docType != "Task" {
					errs = append(errs, fmt.Sprintf("%s: expected `type: Task`, got %q (F3)", rel, docType))
				}
				if !in(taskStatuses, status) {
					errs = append(errs, fmt.Sprintf("%s: Task `status` must be one of %s, got %q (F2)", rel, strings.Join(taskStatuses, "|"), status))
				}
				p := pair(fid)
				p.tasks[parts[2]] = status
				p.deps[parts[2]] = asList(data["depends-on"])
				p.depRels[parts[2]] = rel
				for _, res := range asList(data["resource"]) {
					resources = append(resources, struct{ rel, path string }{rel, res})
				}
			} else {
				errs = append(errs, fmt.Sprintf("%s: paired directories may contain only SPEC.md, PLAN.md, TEST.md, and NN-slug.md tasks (F3)", rel))
			}
		default:
			errs = append(errs, fmt.Sprintf("%s: nested deeper than FDF structure allows (F3)", rel))
		}
		return nil
	})

	// F3: every paired directory has a sibling feature document.
	for fid := range pairs {
		if features[fid] == nil {
			errs = append(errs, fmt.Sprintf("%s/: paired directory has no sibling feature document %s.md (F3)", fid, fid))
		}
	}

	// F4 + F8: status <-> artifact invariants.
	for fid, f := range features {
		p := pairs[fid]
		switch {
		case f.status == "draft":
			if p != nil {
				errs = append(errs, fmt.Sprintf("%s: status 'draft' but paired directory %s/ exists (F4)", f.rel, fid))
			}
			continue
		case !in(featureStatuses, f.status):
			continue // already reported (F2)
		case p == nil:
			// No paired directory at all: fall through to the per-artifact
			// checks below (against an empty pairInfo) so the error names
			// the specific missing file(s) (SPEC.md/PLAN.md/TEST.md/tasks)
			// per the F4 rule catalog, instead of a generic directory-only
			// message that names nothing.
			p = &pairInfo{tasks: map[string]string{}, deps: map[string][]string{}, depRels: map[string]string{}}
		}
		if !p.spec {
			errs = append(errs, fmt.Sprintf("%s: status '%s' requires %s/SPEC.md (F4)", f.rel, f.status, fid))
		}
		planned := in([]string{"planned", "implementing", "done"}, f.status)
		if planned && !p.plan {
			errs = append(errs, fmt.Sprintf("%s: status '%s' requires %s/PLAN.md (F4)", f.rel, f.status, fid))
		}
		if planned { // F8
			if !p.test {
				errs = append(errs, fmt.Sprintf("%s: status '%s' requires %s/TEST.md (F8)", f.rel, f.status, fid))
			} else {
				for _, m := range scenarioRe.FindAllStringSubmatch(f.body, -1) {
					name := strings.TrimSpace(m[1])
					if !strings.Contains(p.testBody, name) {
						errs = append(errs, fmt.Sprintf("%s/TEST.md: scenario %q has no test case (F8)", fid, name))
					}
				}
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

	// F6: plan <-> tasks, depends-on resolution, acyclicity.
	for fid, p := range pairs {
		if p.plan {
			listed := map[string]bool{}
			for _, t := range sectionLinks(p.planBody, "Tasks") {
				listed[filepath.Base(strings.SplitN(strings.SplitN(t, "#", 2)[0], "?", 2)[0])] = true
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
			errs = append(errs, fmt.Sprintf("%s/: tasks exist but PLAN.md is missing (F6)", fid))
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
		for _, t := range sectionLinks(r.body, "Features") {
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
			if r.status == "shipped" && f.status != "done" {
				errs = append(errs, fmt.Sprintf("%s: shipped release lists %s with status '%s' (F7)", r.rel, fid, f.status))
			}
		}
	}
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
		for _, t := range sectionLinks(r.body, "Features") {
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
	for _, l := range links {
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
				repoErrs = append(repoErrs, fmt.Sprintf("%s: `resource` path does not exist in repo -> %s (R1)", r.rel, r.path))
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
		fmt.Fprintln(out, "Bundle is NOT conformant with FDF v0.2.")
		return 1
	}
	if len(repoErrs) > 0 {
		fmt.Fprintln(out, "Bundle is FDF-conformant but has drifted from the repo (R1).")
		return 1
	}
	if opts.RepoRoot == "" {
		fmt.Fprintln(out, "Bundle is conformant with FDF v0.2 (R1 skipped: no project root).")
		return 0
	}
	fmt.Fprintln(out, "Bundle is conformant with FDF v0.2; repo integrity (R1) verified.")
	return 0
}

func checkFeatureBody(rel, body string, errs *[]string) {
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
	if scenarios < 1 {
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

func sectionLinks(body, heading string) []string {
	var out []string
	inSection := false
	for _, line := range strings.Split(body, "\n") {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			inSection = strings.EqualFold(strings.TrimSpace(m[1]), heading)
			continue
		}
		if inSection {
			for _, l := range linkRe.FindAllStringSubmatch(line, -1) {
				out = append(out, l[1])
			}
		}
	}
	return out
}

func resolveLink(root, srcRel, target string) string {
	t := strings.TrimSpace(target)
	if t == "" || strings.HasPrefix(t, "#") || strings.Contains(t, "://") ||
		strings.HasPrefix(t, "mailto:") || strings.HasPrefix(t, "tel:") {
		return ""
	}
	t = strings.SplitN(strings.SplitN(t, "#", 2)[0], "?", 2)[0]
	if t == "" {
		return ""
	}
	if strings.HasPrefix(t, "/") {
		return filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(t, "/")))
	}
	return filepath.Clean(filepath.Join(root, filepath.Dir(filepath.FromSlash(srcRel)), filepath.FromSlash(t)))
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
