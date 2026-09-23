package bundle

// v0.5 post-delivery documents. A Change alters documented behavior; a Fix
// restores behavior the feature document already describes. Both live under
// changes/ (flat or in groups) and declare, in their body, the effects they
// will have on the features they affect — which is what lets F10 confirm the
// effects actually landed before the document may be `done`.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// toSlash normalizes a relative path for ID comparison.
func toSlash(p string) string { return filepath.ToSlash(p) }

type changeInfo struct {
	rel, id, docType, status, version, body string
	affects, retires                        []string
	resolves                                []string // v0.7: bug IDs this work repairs
	timestamp                               string
}

// changeDecl holds one affected feature's declared effects.
type changeDecl struct {
	adds, modifies, removes []string // Change
	regressions             []string // Fix: scenario name -> verification present
	missingVerification     []string // Fix entries with no verification half
	malformed               []string // Change list items that are none of the three forms
}

const (
	scenarioChangesHeading = "Scenario changes"
	regressionCasesHeading = "Regression cases"
)

var (
	subHeadingRe  = regexp.MustCompile(`^##\s+(.*\S)\s*$`)
	declVerbRe    = regexp.MustCompile(`^\s*[-*]\s+(add|modify|remove)\s*:\s*(\S.*?)\s*$`)
	listItemRe    = regexp.MustCompile(`^\s*[-*]\s+(\S.*?)\s*$`)
	verbatimSepRe = regexp.MustCompile(`\s(?:—|–|--)\s`)
)

// hasHeading reports whether a body carries a top-level `# <heading>`.
func hasHeading(body, heading string) bool {
	for _, line := range strings.Split(body, "\n") {
		if m := headingRe.FindStringSubmatch(line); m != nil &&
			strings.EqualFold(strings.TrimSpace(m[1]), heading) {
			return true
		}
	}
	return false
}

// parseDecls slices `# <heading>` out of a body and returns one entry per
// `## <feature-id>` subheading under it. verbs selects the Change grammar
// (`- add:/modify:/remove: <scenario>`) over the Fix grammar
// (`- <scenario> — <verification>`).
func parseDecls(body, heading string, verbs bool) map[string]*changeDecl {
	out := map[string]*changeDecl{}
	inSection, current := false, ""
	for _, line := range strings.Split(body, "\n") {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			inSection = strings.EqualFold(strings.TrimSpace(m[1]), heading)
			current = ""
			continue
		}
		if !inSection {
			continue
		}
		if m := subHeadingRe.FindStringSubmatch(line); m != nil {
			current = strings.TrimSpace(m[1])
			if out[current] == nil {
				out[current] = &changeDecl{}
			}
			continue
		}
		if current == "" {
			continue
		}
		d := out[current]
		if verbs {
			if m := declVerbRe.FindStringSubmatch(line); m != nil {
				switch m[1] {
				case "add":
					d.adds = append(d.adds, m[2])
				case "modify":
					d.modifies = append(d.modifies, m[2])
				case "remove":
					d.removes = append(d.removes, m[2])
				}
			} else if m := listItemRe.FindStringSubmatch(line); m != nil {
				d.malformed = append(d.malformed, m[1])
			}
			continue
		}
		if m := listItemRe.FindStringSubmatch(line); m != nil {
			entry := m[1]
			if loc := verbatimSepRe.FindStringIndex(entry); loc != nil {
				name := strings.TrimSpace(entry[:loc[0]])
				if strings.TrimSpace(entry[loc[1]:]) == "" {
					d.missingVerification = append(d.missingVerification, name)
				} else {
					d.regressions = append(d.regressions, name)
				}
			} else {
				d.missingVerification = append(d.missingVerification, entry)
			}
		}
	}
	return out
}

// strayDeclEntries returns the list items under `# <heading>` that come before
// its first `## <feature-id>` heading: entries that belong to no feature.
func strayDeclEntries(body, heading string) []string {
	var out []string
	inSection, seenSub := false, false
	for _, line := range strings.Split(body, "\n") {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			inSection = strings.EqualFold(strings.TrimSpace(m[1]), heading)
			seenSub = false
			continue
		}
		if !inSection || seenSub {
			continue
		}
		if subHeadingRe.MatchString(line) {
			seenSub = true
			continue
		}
		if m := listItemRe.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

// scenarioNames returns every Scenario:/Scenario Outline: name in a feature body.
func scenarioNames(body string) map[string]bool {
	out := map[string]bool{}
	for _, m := range scenarioRe.FindAllStringSubmatch(body, -1) {
		out[strings.TrimSpace(m[1])] = true
	}
	return out
}

// removals returns the names a declaration takes out of its feature. With
// replaces set (v0.6), a name the same declaration also adds is replaced, not
// removed: a lexicon fix turns a finished rename's `remove: Store owner X` and
// `add: Venue owner X` into one name, and the record must stay valid.
func removals(d *changeDecl, replaces bool) []string {
	if !replaces {
		return d.removes
	}
	added := map[string]bool{}
	for _, n := range d.adds {
		added[n] = true
	}
	var out []string
	for _, n := range d.removes {
		if !added[n] {
			out = append(out, n)
		}
	}
	return out
}

// checkChangeIntegrity is F10. It runs for bundles pinning v0.5 and later;
// replaces is v0.6's reading of a name both removed and added (see removals).
// v7 adds the v0.7 rules: a declaration holds only well-formed entries,
// `retires` declares its features from the start and names only features in
// `affects`, `replaced-by` names a feature that exists, and a name another
// done Change adds back is no longer held against the Change that removed it.
func checkChangeIntegrity(changes map[string]*changeInfo, features map[string]*featureInfo, pairs map[string]*pairInfo, replaces, v7 bool, errs *[]string) {
	ids := make([]string, 0, len(changes))
	for id := range changes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	retiredBy := map[string][]string{}

	// Scenario names retired from a feature by some landed Change. An episodic
	// document is a frozen record: a Fix that proved scenario X, or an older
	// Change that added it, was telling the truth when it was written, and a
	// later Change removing X must not force those documents to be rewritten.
	// Without this, F10 would contradict the very rule it enforces.
	superseded := map[string]bool{}
	// addedBy records which done Changes add each name to a feature (v0.7): a
	// name a later Change brings back does not make the Change that removed it
	// wrong, and that Change is a frozen record.
	addedBy := map[string][]string{}
	for _, id := range ids {
		c := changes[id]
		if c.status != "done" || c.docType != "Change" {
			continue
		}
		for fid, d := range parseDecls(c.body, scenarioChangesHeading, true) {
			for _, nm := range removals(d, replaces) {
				superseded[fid+"\x00"+nm] = true
			}
			for _, nm := range d.adds {
				addedBy[fid+"\x00"+nm] = append(addedBy[fid+"\x00"+nm], id)
			}
		}
	}
	readdedByOther := func(fid, name, self string) bool {
		for _, other := range addedBy[fid+"\x00"+name] {
			if other != self {
				return true
			}
		}
		return false
	}

	for _, id := range ids {
		c := changes[id]
		isFix := c.docType == "Fix"
		wantHeading, otherHeading := scenarioChangesHeading, regressionCasesHeading
		if isFix {
			wantHeading, otherHeading = regressionCasesHeading, scenarioChangesHeading
		}

		if len(c.affects) == 0 {
			*errs = append(*errs, fmt.Sprintf("%s: `%s` requires a non-empty `affects` naming the feature(s) it touches (F10)", c.rel, c.docType))
		}
		affected := map[string]bool{}
		for _, fid := range c.affects {
			affected[fid] = true
			f := features[fid]
			if f == nil {
				*errs = append(*errs, fmt.Sprintf("%s: `affects` names unknown feature %q (F10)", c.rel, fid))
				continue
			}
			// Delivered: done, retired, or (v0.7) adopted — a capability that
			// existed before its document. `adopted` fails F2 on older pins.
			if f.status != "done" && f.status != "retired" && f.status != "adopted" {
				*errs = append(*errs, fmt.Sprintf("%s: `affects` names %s with status '%s' — a feature that is not delivered is edited directly, not change-requested (F10)", c.rel, fid, f.status))
			}
		}
		if isFix && len(c.retires) > 0 {
			*errs = append(*errs, fmt.Sprintf("%s: `retires` is not valid on a Fix — retiring behavior is a deliberate Change (F10)", c.rel))
		}
		if hasHeading(c.body, otherHeading) {
			*errs = append(*errs, fmt.Sprintf("%s: a %s carries `# %s`, not `# %s` (F10)", c.rel, c.docType, wantHeading, otherHeading))
		}
		if !hasHeading(c.body, wantHeading) {
			*errs = append(*errs, fmt.Sprintf("%s: `%s` requires a `# %s` section (F10)", c.rel, c.docType, wantHeading))
			continue
		}

		decls := parseDecls(c.body, wantHeading, !isFix)
		if v7 {
			for _, e := range strayDeclEntries(c.body, wantHeading) {
				*errs = append(*errs, fmt.Sprintf("%s: `# %s` entry %q sits under no `## <feature-id>` heading (F10)", c.rel, wantHeading, e))
			}
			for _, fid := range sortedDeclKeys(decls) {
				for _, e := range decls[fid].malformed {
					*errs = append(*errs, fmt.Sprintf("%s: `## %s` entry %q is not `- add: <scenario>`, `- modify: <scenario>` or `- remove: <scenario>` (F10)", c.rel, fid, e))
				}
			}
		}
		for fid := range affected {
			if decls[fid] == nil {
				*errs = append(*errs, fmt.Sprintf("%s: `# %s` has no `## %s` heading for an affected feature (F10)", c.rel, wantHeading, fid))
			}
		}
		declFids := make([]string, 0, len(decls))
		for fid := range decls {
			declFids = append(declFids, fid)
		}
		sort.Strings(declFids)
		for _, fid := range declFids {
			if !affected[fid] {
				*errs = append(*errs, fmt.Sprintf("%s: `# %s` declares `## %s`, which is not listed in `affects` (F10)", c.rel, wantHeading, fid))
			}
		}

		for _, fid := range c.retires {
			f := features[fid]
			if f == nil {
				*errs = append(*errs, fmt.Sprintf("%s: `retires` names unknown feature %q (F10)", c.rel, fid))
				continue
			}
			switch {
			case !v7 && f.status != "retired":
				*errs = append(*errs, fmt.Sprintf("%s: `retires` names %s whose status is '%s', expected 'retired' (F10)", c.rel, fid, f.status))
			case v7 && c.status == "done" && f.status != "retired":
				*errs = append(*errs, fmt.Sprintf("%s: done, and `retires` names %s whose status is '%s' — retire it in the same edit (F10)", c.rel, fid, f.status))
			case v7 && c.status != "done" && f.status != "done" && f.status != "adopted":
				*errs = append(*errs, fmt.Sprintf("%s: `retires` names %s whose status is '%s' — until this Change is done, the feature it retires is still delivered ('done' or 'adopted') (F10)", c.rel, fid, f.status))
			}
			if v7 && !affected[fid] {
				*errs = append(*errs, fmt.Sprintf("%s: `retires` names %s, which is not in `affects` (F10)", c.rel, fid))
			}
			if !hasHeading(c.body, "Rationale") {
				*errs = append(*errs, fmt.Sprintf("%s: retires %s but has no `# Rationale` — a retirement records why (F10)", c.rel, fid))
			} else if body, _ := sectionText(c.body, "Rationale"); v7 && strings.TrimSpace(body) == "" {
				*errs = append(*errs, fmt.Sprintf("%s: retires %s but its `# Rationale` is empty — a retirement records why (F10)", c.rel, fid))
			}
			if c.status == "done" {
				retiredBy[fid] = append(retiredBy[fid], c.rel)
			}
		}

		if c.status != "done" {
			continue
		}
		// Done: the declared effects must be reality.
		for _, fid := range declFids {
			f := features[fid]
			if f == nil {
				continue
			}
			names := scenarioNames(f.body)
			d := decls[fid]
			for _, n := range append(append([]string{}, d.adds...), d.modifies...) {
				if !names[n] && !superseded[fid+"\x00"+n] {
					*errs = append(*errs, fmt.Sprintf("%s: done, but %s has no scenario %q (F10)", c.rel, fid, n))
				}
			}
			for _, n := range removals(d, replaces) {
				if names[n] && !(v7 && readdedByOther(fid, n, id)) {
					*errs = append(*errs, fmt.Sprintf("%s: done, but %s still has scenario %q (F10)", c.rel, fid, n))
				}
			}
			for _, n := range d.regressions {
				if !names[n] && !superseded[fid+"\x00"+n] {
					*errs = append(*errs, fmt.Sprintf("%s: done, but %s has no scenario %q — a fix proves scenarios that already exist (F10)", c.rel, fid, n))
				}
			}
			for _, n := range d.missingVerification {
				*errs = append(*errs, fmt.Sprintf("%s: regression case %q gives no verification — name the command, test path, or manual procedure after an em dash (F10)", c.rel, n))
			}
			// The regression must be recorded where it lasts: the feature's test doc.
			if p := pairs[fid]; p != nil && p.test {
				for _, n := range d.regressions {
					if superseded[fid+"\x00"+n] {
						continue
					}
					if !strings.Contains(p.testBody, n) {
						*errs = append(*errs, fmt.Sprintf("%s: done, but %s.test.md has no case for %q (F10)", c.rel, fid, n))
					}
				}
			}
		}
	}

	// `replaced-by` names the feature that replaces a retired one (v0.7 checks
	// it; earlier pins only described it).
	if v7 {
		for _, fid := range sortedFeatureIDs(features) {
			if r := features[fid].replacedBy; r != "" && features[r] == nil {
				*errs = append(*errs, fmt.Sprintf("%s: `replaced-by` names unknown feature %q (F10)", features[fid].rel, r))
			}
		}
	}

	// Every retired feature is accounted for by exactly one done Change.
	fids := make([]string, 0, len(features))
	for fid := range features {
		fids = append(fids, fid)
	}
	sort.Strings(fids)
	for _, fid := range fids {
		if features[fid].status != "retired" {
			continue
		}
		switch n := len(retiredBy[fid]); {
		case n == 0:
			*errs = append(*errs, fmt.Sprintf("%s: status 'retired' but no done Change under changes/ `retires` it — the reason must be written down (F10)", features[fid].rel))
		case n > 1:
			*errs = append(*errs, fmt.Sprintf("%s: status 'retired' and retired by %d done Changes (%s) — exactly one records the reason (F10)", features[fid].rel, n, strings.Join(retiredBy[fid], ", ")))
		}
	}
}

func sortedDeclKeys(m map[string]*changeDecl) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedFeatureIDs(m map[string]*featureInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// stamp reads a `timestamp` field, a date or an RFC 3339 time, as text.
func stamp(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.Trim(strings.TrimSpace(t), `"'`)
	default:
		return fmt.Sprint(t)
	}
}

// day is the YYYY-MM-DD a timestamp falls on, or "" when it has none.
func day(ts string) string {
	if len(ts) >= 10 && isoDateRe.MatchString(ts[:10]) {
		return ts[:10]
	}
	return ""
}

// checkRegressionLanded is the soft check the spec has promised since v0.5: a
// feature's slug.test.md last changed before a done Fix or Change that
// altered its scenarios, so the case that work needed may never have landed.
// It compares days, not times, so a same-day edit never warns.
func checkRegressionLanded(changes map[string]*changeInfo, pairs map[string]*pairInfo, warns *[]string) {
	ids := make([]string, 0, len(changes))
	for id := range changes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := changes[id]
		changed := day(c.timestamp)
		if c.status != "done" || changed == "" {
			continue
		}
		heading := scenarioChangesHeading
		if c.docType == "Fix" {
			heading = regressionCasesHeading
		}
		decls := parseDecls(c.body, heading, c.docType == "Change")
		for _, fid := range sortedDeclKeys(decls) {
			d := decls[fid]
			if len(d.adds)+len(d.modifies)+len(d.removes)+len(d.regressions) == 0 {
				continue
			}
			p := pairs[fid]
			if p == nil || !p.test {
				continue
			}
			if tested := day(p.testTimestamp); tested != "" && tested < changed {
				*warns = append(*warns, fmt.Sprintf("%s.test.md: `timestamp` %s predates %s (done %s), which changed %s's scenarios — the case it needed may not have landed; update the test document and its timestamp", fid, tested, c.rel, changed, fid))
			}
		}
	}
}

// findCycleIDs is findCycle over bare IDs (feature `depends-on`), where the
// task variant's ".md" suffix does not apply.
func findCycleIDs(deps map[string][]string) string {
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
		chain = append(chain, n)
		for _, d := range deps[n] {
			if _, known := deps[d]; !known {
				continue // dangling: reported separately
			}
			switch color[d] {
			case grey:
				return strings.Join(append(chain, d), " -> ")
			case white:
				if c := visit(d); c != "" {
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

// checkChangeLifecycle is F4 for Change and Fix documents. A Change passes
// through `specified` because altering documented behavior is a decision
// someone approves; a Fix restores already-approved behavior and needs no
// such gate. Tasks are optional for both — the floor for a Fix is one file.
func checkChangeLifecycle(changes map[string]*changeInfo, pairs map[string]*pairInfo, errs *[]string) {
	ids := make([]string, 0, len(changes))
	for id := range changes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := changes[id]
		p := pairs[id]
		if p == nil {
			p = &pairInfo{tasks: map[string]string{}}
		}
		if c.status == "draft" && (p.spec || p.plan || len(p.tasks) > 0) {
			*errs = append(*errs, fmt.Sprintf("%s: status 'draft' but trail siblings or a task directory exist (F4)", c.rel))
			continue
		}
		advanced := c.status == "specified" || c.status == "planned" || c.status == "implementing" || c.status == "done"
		if c.docType == "Change" && advanced && !p.spec {
			*errs = append(*errs, fmt.Sprintf("%s: a Change at '%s' requires %s.spec.md — the approved design (F4)", c.rel, c.status, id))
		}
		if (c.status == "planned" || c.status == "implementing") && !p.plan {
			*errs = append(*errs, fmt.Sprintf("%s: status '%s' requires %s.plan.md (F4)", c.rel, c.status, id))
		}
		nDone := 0
		for _, st := range p.tasks {
			if st == "done" {
				nDone++
			}
		}
		if c.status == "implementing" && len(p.tasks) == 0 {
			*errs = append(*errs, fmt.Sprintf("%s: status 'implementing' requires at least one task (F4)", c.rel))
		}
		if c.status == "implementing" && len(p.tasks) > 0 && nDone == len(p.tasks) {
			*errs = append(*errs, fmt.Sprintf("%s: status 'implementing' but every task is done — promote to 'done' (F4)", c.rel))
		}
		if c.status == "done" && nDone != len(p.tasks) {
			var open []string
			for name, st := range p.tasks {
				if st != "done" {
					open = append(open, name)
				}
			}
			sort.Strings(open)
			*errs = append(*errs, fmt.Sprintf("%s: status 'done' but tasks not done: %s (F4)", c.rel, strings.Join(open, ", ")))
		}
	}
}

// checkReleaseChanges is the F7 half covering a release's `# Changes` list:
// the same bidirectional check features get, for Change and Fix documents.
func checkReleaseChanges(rootAbs string, releases map[string]*releaseInfo, changes map[string]*changeInfo, errs *[]string) {
	versions := make([]string, 0, len(releases))
	for v := range releases {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	for _, version := range versions {
		r := releases[version]
		for _, t := range sectionLinks(r.body, "Changes") {
			resolved := resolveLink(rootAbs, r.rel, t)
			if resolved == "" {
				continue
			}
			cid := strings.TrimSuffix(toSlash(relTo(rootAbs, resolved)), ".md")
			c := changes[cid]
			if c == nil {
				*errs = append(*errs, fmt.Sprintf("%s: `# Changes` links unknown change %q (F7)", r.rel, t))
				continue
			}
			if c.version != version {
				got := c.version
				if got == "" {
					got = "unset"
				}
				*errs = append(*errs, fmt.Sprintf("%s: lists %s whose `version` is %s, expected %q (F7)", r.rel, cid, got, version))
			}
			if r.status == "shipped" && c.status != "done" {
				*errs = append(*errs, fmt.Sprintf("%s: shipped release lists %s with status '%s' (F7)", r.rel, cid, c.status))
			}
		}
	}
	ids := make([]string, 0, len(changes))
	for id := range changes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := changes[id]
		if c.version == "" {
			continue
		}
		r := releases[c.version]
		if r == nil {
			*errs = append(*errs, fmt.Sprintf("%s: version %q has no releases/%s.md (F7)", c.rel, c.version, c.version))
			continue
		}
		found := false
		for _, t := range sectionLinks(r.body, "Changes") {
			if resolved := resolveLink(rootAbs, r.rel, t); resolved != "" &&
				strings.TrimSuffix(toSlash(relTo(rootAbs, resolved)), ".md") == id {
				found = true
			}
		}
		if !found {
			*errs = append(*errs, fmt.Sprintf("%s: version %q but not listed under `# Changes` in %s (F7)", c.rel, c.version, r.rel))
		}
	}
}
