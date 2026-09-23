// Package bundle validates an FDF bundle (spec v0.2 through v0.6). Rules
// F1-F13 are format conformance; R1 is repo integrity. See SPEC.md.
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
	// FreshStubsAdvisory downgrades F9 "unfilled context stub" from error to
	// warning even when features exist. `fdf migrate` sets it: it has just
	// created the stubs and cannot fill them (only the fdf-init interview
	// can), so unfilled stubs are expected post-migration state, not a
	// migration failure. Plain `fdf validate` leaves it false and enforces F9.
	FreshStubsAdvisory bool
	// StrictDomain promotes F12's banned-word warnings to errors. Off by
	// default: the lexicon check is the one rule that matches natural
	// language, so a young bundle adopting a lexicon would otherwise be
	// unable to validate until every document had been reworded.
	StrictDomain bool
}

var (
	reserved     = map[string]bool{"INDEX.md": true, "LOG.md": true}
	trailNames   = map[string]string{"SPEC.md": "Spec", "PLAN.md": "Plan", "TEST.md": "Test"}
	contextNames = map[string]bool{"STACK.md": true, "ARCHITECTURE.md": true, "INFRA.md": true, "SURFACES.md": true, "DOMAIN.md": true}
	// structural types are rejected at the bundle root (position is fixed).
	// Surface and Log exist only under v0.4 and are gated at the check site,
	// so v0.2/v0.3 bundles keep accepting them as ordinary root doc types.
	structural   = map[string]bool{"Feature": true, "Spec": true, "Plan": true, "Task": true, "Test": true, "Release": true}
	structuralV4 = map[string]bool{"Surface": true, "Log": true}
	structuralV5 = map[string]bool{"Change": true, "Fix": true}
	structuralV6 = map[string]bool{"Practice": true, "Debt": true}
	// changeTrailRoleRe: under changes/, only spec/plan/log are legal roles —
	// test and surface are living documents owned by the affected feature.
	changeTrailRoleRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)\.(spec|plan|log)\.md$`)
	// logTrailRoleRe: under practices/ and debts/, `log` is the only legal
	// role — both are living references with no episodic trail of their own.
	logTrailRoleRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)\.(log)\.md$`)
	recommended    = []string{"title", "description", "timestamp"}
	linkRe         = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	// fenceOpenRe/inlineCodeRe strip code before links are harvested: a link
	// written inside an example is sample text, not a cross-link.
	fenceOpenRe  = regexp.MustCompile("^(`{3,})")
	inlineCodeRe = regexp.MustCompile("`[^`\n]*`")
	isoDateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	taskFileRe   = regexp.MustCompile(`^\d{2}-[a-z0-9][a-z0-9-]*\.md$`)
	lowerFileRe  = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*\.md$`)
	lowerDirRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	// trailRoleRe matches known v0.4 stem-qualified trail siblings.
	trailRoleRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)\.(spec|plan|test|surface|log)\.md$`)
	// anyTrailAttemptRe matches any group-level dotted basename (slug.something.md).
	// Feature slugs are [a-z0-9-], so an extra dot implies a trail-role attempt.
	anyTrailAttemptRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)\.(.+)\.md$`)
	trailRoleType     = map[string]string{
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

// featureStatuses: "retired" is v0.5 only and gated at the check site.
var featureStatuses = []string{"draft", "specified", "planned", "implementing", "done"}
var featureStatusesV5 = []string{"draft", "specified", "planned", "implementing", "done", "retired"}

// changeStatuses is the Change/Fix vocabulary (v0.5); it mirrors the feature
// lifecycle minus "retired" — a change is work, not a capability.
var changeStatuses = []string{"draft", "specified", "planned", "implementing", "done"}
var taskStatuses = []string{"pending", "in-progress", "done"}
var practiceStatuses = []string{"active", "superseded"}
var debtStatuses = []string{"open", "accepted", "resolved"}
var releaseStatuses = []string{"planned", "shipped"}

// supportedVersions are the spec versions this validator understands; a pin
// outside this set is an F1 error directing the user to `fdf migrate`.
var supportedVersions = map[string]bool{"0.2": true, "0.3": true, "0.4": true, "0.5": true, "0.6": true}

// supportedList renders supportedVersions for error messages, so adding a
// version never leaves a stale literal behind.
func supportedList() string {
	vs := make([]string, 0, len(supportedVersions))
	for v := range supportedVersions {
		vs = append(vs, v)
	}
	sort.Strings(vs)
	return strings.Join(vs, ", ")
}

// checkLogBody validates ISO-8601 ## date headings and newest-first order
// for reserved LOG.md and v0.4 slug.log.md files.
func checkLogBody(rel, text string, errs, warns *[]string) {
	var dates []string
	for _, line := range strings.Split(text, "\n") {
		if m := logDateRe.FindStringSubmatch(line); m != nil {
			h := strings.TrimSpace(m[1])
			if isoDateRe.MatchString(h) {
				dates = append(dates, h)
			} else {
				*errs = append(*errs, fmt.Sprintf("%s: log date heading '## %s' is not ISO-8601 (F1)", rel, h))
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

// readPin returns the fdf_version pinned by the bundle's root INDEX.md, or ""
// if absent/unreadable. It runs before the walk because version-gated rules
// (Context documents, feature-directory LOG.md) must be known for every file,
// and WalkDir visits files lexically — ARCHITECTURE.md sorts before INDEX.md.
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
	var resources []struct{ rel, field, path string }
	features := map[string]*featureInfo{}
	pairs := map[string]*pairInfo{}
	releases := map[string]*releaseInfo{}
	changes := map[string]*changeInfo{}
	practices := map[string]*practiceInfo{}
	practiceTrails := map[string]string{} // practice ID -> rel of its .log.md
	debts := map[string]*debtInfo{}
	debtTrails := map[string]string{} // debt ID -> rel of its .log.md
	featureDeps := map[string][]string{}
	documents := 0

	// Version gate: Context docs / F9 apply on v0.3+; stem-trail layout and
	// SURFACES.md on v0.4 only. Anything else validates under v0.2 rules.
	pinnedVer := readPin(rootAbs)
	// specStem: stem-qualified trail layout (v0.4 onward). specV5 additionally
	// enables changes/, the Change/Fix types, `retired`, feature depends-on,
	// and F10.
	specStem := pinnedVer == "0.4" || pinnedVer == "0.5" || pinnedVer == "0.6"
	// specV5: v0.5 and later — changes/, Change/Fix, `retired`, feature
	// depends-on, F10. specV6 adds practices/, DOMAIN.md, F11 and F12.
	specV5 := pinnedVer == "0.5" || pinnedVer == "0.6"
	specV6 := pinnedVer == "0.6"
	specHasContext := pinnedVer == "0.3" || specStem
	// contextDocs[name] records a seen root Context document and whether it is
	// still an unfilled stub, for F9.
	contextDocs := map[string]bool{} // name -> isStub

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
		parts := strings.Split(filepath.ToSlash(rel), "/")
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			warns = append(warns, fmt.Sprintf("%s: could not read file (%v)", rel, rerr))
			return nil
		}
		text := strings.TrimPrefix(string(raw), "\uFEFF")
		for _, m := range linkRe.FindAllStringSubmatch(linkScanText(text), -1) {
			links = append(links, link{rel, m[1]})
		}

		// README.md at bundle root: legal, ignored.
		if rel == "README.md" {
			return nil
		}

		if reserved[name] {
			// A changes/ group carries INDEX.md like any group; only a task
			// directory (one with a sibling <name>.md) forbids it.
			inChangeGroup := specV5 && len(parts) == 3 && parts[0] == "changes" &&
				!exists(filepath.Join(rootAbs, "changes", parts[1]+".md"))
			// Every directory under practices/ is a group: a practice owns no
			// task directory, so there is no ambiguity to resolve.
			inPracticeGroup := specV6 && len(parts) == 3 &&
				(parts[0] == "practices" || parts[0] == "debts")
			inChangeGroup = inChangeGroup || inPracticeGroup
			// Trap 12: v0.4 task directories may contain only NN-slug.md tasks —
			// reject INDEX.md/LOG.md here (reserved returns before len==3 case).
			if specStem && len(parts) == 3 && !inChangeGroup {
				errs = append(errs, fmt.Sprintf("%s: task directories may contain only NN-slug.md tasks (F3)", rel))
				return nil
			}
			block, delimited, _ := splitFrontmatter(text)
			if name == "INDEX.md" {
				if delimited {
					data, _ := parseFrontmatter(block)
					if rel != "INDEX.md" || data == nil || data["fdf_version"] == nil {
						warns = append(warns, fmt.Sprintf("%s: index file should not carry frontmatter", rel))
					} else if v, _ := data["fdf_version"].(string); !supportedVersions[v] {
						errs = append(errs, fmt.Sprintf("INDEX.md: fdf_version %q is not a supported version (%s) — run `fdf migrate` (F1)", v, supportedList()))
					}
				} else if rel == "INDEX.md" {
					warns = append(warns, "INDEX.md: root index should pin fdf_version")
				}
				if !regexp.MustCompile(`(?m)^\s*[-*]\s+\[.*\]\(.*\)`).MatchString(text) {
					warns = append(warns, fmt.Sprintf("%s: index file has no bulleted listing", rel))
				}
			} else { // LOG.md
				checkLogBody(rel, text, &errs, &warns)
			}
			return nil
		}

		// isContext: bundle-root Context docs. SURFACES.md is v0.4-only;
		// STACK/ARCHITECTURE/INFRA are recognized under any Context pin.
		isContext := false
		if len(parts) == 1 && contextNames[name] {
			if name == "SURFACES.md" {
				isContext = specStem
			} else if name == "DOMAIN.md" {
				isContext = specV6
			} else {
				isContext = specHasContext
			}
		}

		// Casing gate for everything non-reserved. Root SPEC.md is always
		// legal (type: Reference). Nested SPEC/PLAN/TEST.md uppercase trail
		// names remain legal only under pre-v0.4 (trap 13).
		_, isLegacyTrail := trailNames[name]
		isRootSpec := rel == "SPEC.md"
		if !isRootSpec && !isContext && !(!specStem && isLegacyTrail) {
			if !lowerFileRe.MatchString(name) && !taskFileRe.MatchString(name) {
				allowed := "INDEX/LOG/SPEC/PLAN/TEST.md"
				switch {
				case specV6:
					allowed = "INDEX/LOG/SPEC/STACK/ARCHITECTURE/SURFACES/INFRA/DOMAIN.md"
				case specStem:
					allowed = "INDEX/LOG/SPEC/STACK/ARCHITECTURE/SURFACES/INFRA.md"
				case specHasContext:
					allowed = "INDEX/LOG/SPEC/PLAN/TEST/STACK/ARCHITECTURE/INFRA.md"
				}
				errs = append(errs, fmt.Sprintf("%s: filenames are lowercase; uppercase is reserved for %s (F3)", rel, allowed))
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

		switch {
		case specV5 && parts[0] == "changes" && len(parts) > 1:
			// Positions under changes/: <slug>.md, <slug>.<role>.md,
			// <group>/<slug>.md, <group>/<slug>.<role>.md, and tasks one level
			// below a change document. A directory is a task directory when a
			// sibling <name>.md exists and a group otherwise.
			rest := parts[1:]
			dir := "changes"
			if len(rest) > 1 && !exists(filepath.Join(rootAbs, "changes", rest[0]+".md")) {
				// rest[0] is a group; shift into it.
				if !lowerDirRe.MatchString(rest[0]) {
					errs = append(errs, fmt.Sprintf("%s: changes/ group names are lowercase [a-z0-9-] (F3)", rel))
					return nil
				}
				dir, rest = "changes/"+rest[0], rest[1:]
			}
			switch {
			case len(rest) == 1:
				base := rest[0]
				if m := changeTrailRoleRe.FindStringSubmatch(base); m != nil {
					want := trailRoleType[m[2]]
					if docType != want {
						errs = append(errs, fmt.Sprintf("%s: expected `type: %s`, got %q (F3)", rel, want, docType))
					}
					cid := dir + "/" + m[1]
					p := pair(cid)
					switch m[2] {
					case "spec":
						p.spec = true
					case "plan":
						p.plan, p.planRel, p.planBody = true, rel, body
					case "log":
						checkLogBody(rel, text, &errs, &warns)
					}
					return nil
				}
				if m := anyTrailAttemptRe.FindStringSubmatch(base); m != nil {
					errs = append(errs, fmt.Sprintf("%s: unknown trail role %q under changes/ — allowed roles are spec, plan, log (test and surface belong to the affected feature) (F3)", rel, m[2]))
					return nil
				}
				if docType != "Change" && docType != "Fix" {
					errs = append(errs, fmt.Sprintf("%s: expected `type: Change` or `type: Fix`, got %q (F3)", rel, docType))
					return nil
				}
				if !in(changeStatuses, status) {
					errs = append(errs, fmt.Sprintf("%s: %s `status` must be one of %s, got %q (F2)", rel, docType, strings.Join(changeStatuses, "|"), status))
				}
				version, _ := data["version"].(string)
				cid := dir + "/" + strings.TrimSuffix(base, ".md")
				changes[cid] = &changeInfo{
					rel: rel, id: cid, docType: docType, status: status, version: version, body: body,
					affects: asList(data["affects"]), retires: asList(data["retires"]),
				}
				if len(fenceRe.FindAllStringSubmatch(body, -1)) > 0 {
					errs = append(errs, fmt.Sprintf("%s: %s documents carry no Gherkin — behavior statements belong in the features they amend (F5)", rel, docType))
				}
				for _, res := range asList(data["resource"]) {
					resources = append(resources, struct{ rel, field, path string }{rel, "resource", res})
				}
			case len(rest) == 2 && taskFileRe.MatchString(rest[1]):
				if docType != "Task" {
					errs = append(errs, fmt.Sprintf("%s: expected `type: Task`, got %q (F3)", rel, docType))
				}
				if !in(taskStatuses, status) {
					errs = append(errs, fmt.Sprintf("%s: Task `status` must be one of %s, got %q (F2)", rel, strings.Join(taskStatuses, "|"), status))
				}
				p := pair(dir + "/" + rest[0])
				p.tasks[rest[1]] = status
				p.deps[rest[1]] = asList(data["depends-on"])
				p.depRels[rest[1]] = rel
				for _, res := range asList(data["resource"]) {
					resources = append(resources, struct{ rel, field, path string }{rel, "resource", res})
				}
			case len(rest) == 2:
				errs = append(errs, fmt.Sprintf("%s: task directories may contain only NN-slug.md tasks (F3)", rel))
			default:
				errs = append(errs, fmt.Sprintf("%s: nested deeper than FDF structure allows (F3)", rel))
			}
		case specV6 && parts[0] == "debts" && len(parts) > 1:
			// Positions under debts/ mirror practices/: <slug>.md, its
			// optional <slug>.log.md, and one level of groups. No tasks.
			rest := parts[1:]
			dir := "debts"
			if len(rest) > 1 {
				if !lowerDirRe.MatchString(rest[0]) {
					errs = append(errs, fmt.Sprintf("%s: debts/ group names are lowercase [a-z0-9-] (F3)", rel))
					return nil
				}
				dir, rest = "debts/"+rest[0], rest[1:]
			}
			if len(rest) != 1 {
				errs = append(errs, fmt.Sprintf("%s: debts/ holds a debt, its optional <slug>.log.md, and one level of groups — nothing nests deeper (F3)", rel))
				return nil
			}
			base := rest[0]
			if m := logTrailRoleRe.FindStringSubmatch(base); m != nil {
				if docType != "Log" {
					errs = append(errs, fmt.Sprintf("%s: expected `type: Log`, got %q (F3)", rel, docType))
				}
				checkLogBody(rel, text, &errs, &warns)
				debtTrails[dir+"/"+m[1]] = rel
				return nil
			}
			if m := anyTrailAttemptRe.FindStringSubmatch(base); m != nil {
				errs = append(errs, fmt.Sprintf("%s: unknown trail role %q under debts/ — a debt is a register entry, not a unit of work; `log` is the only role (F3)", rel, m[2]))
				return nil
			}
			if docType != "Debt" {
				errs = append(errs, fmt.Sprintf("%s: expected `type: Debt`, got %q (F3)", rel, docType))
				return nil
			}
			if !in(debtStatuses, status) {
				errs = append(errs, fmt.Sprintf("%s: Debt `status` must be one of %s, got %q (F2)", rel, strings.Join(debtStatuses, "|"), status))
			}
			did := dir + "/" + strings.TrimSuffix(base, ".md")
			title, _ := data["title"].(string)
			timestamp, _ := data["timestamp"].(string)
			debts[did] = &debtInfo{
				rel: rel, id: did, status: status, body: body,
				title: title, timestamp: timestamp, resource: asList(data["resource"]),
			}
			for _, res := range asList(data["resource"]) {
				resources = append(resources, struct{ rel, field, path string }{rel, "resource", res})
			}
		case specV6 && parts[0] == "practices" && len(parts) > 1:
			// Positions under practices/: <slug>.md, <slug>.log.md, and the
			// same two inside one level of groups. No task directories.
			rest := parts[1:]
			dir := "practices"
			if len(rest) > 1 {
				if !lowerDirRe.MatchString(rest[0]) {
					errs = append(errs, fmt.Sprintf("%s: practices/ group names are lowercase [a-z0-9-] (F3)", rel))
					return nil
				}
				dir, rest = "practices/"+rest[0], rest[1:]
			}
			if len(rest) != 1 {
				errs = append(errs, fmt.Sprintf("%s: practices/ holds a practice, its optional <slug>.log.md, and one level of groups — nothing nests deeper (F3)", rel))
				return nil
			}
			base := rest[0]
			if m := logTrailRoleRe.FindStringSubmatch(base); m != nil {
				if docType != "Log" {
					errs = append(errs, fmt.Sprintf("%s: expected `type: Log`, got %q (F3)", rel, docType))
				}
				checkLogBody(rel, text, &errs, &warns)
				practiceTrails[dir+"/"+m[1]] = rel
				return nil
			}
			if m := anyTrailAttemptRe.FindStringSubmatch(base); m != nil {
				errs = append(errs, fmt.Sprintf("%s: unknown trail role %q under practices/ — a practice owns no spec, plan, test or surface; `log` is the only role (F3)", rel, m[2]))
				return nil
			}
			if docType != "Practice" {
				errs = append(errs, fmt.Sprintf("%s: expected `type: Practice`, got %q (F3)", rel, docType))
				return nil
			}
			if !in(practiceStatuses, status) {
				errs = append(errs, fmt.Sprintf("%s: Practice `status` must be one of %s, got %q (F2)", rel, strings.Join(practiceStatuses, "|"), status))
			}
			pid := dir + "/" + strings.TrimSuffix(base, ".md")
			supersededBy, _ := data["superseded-by"].(string)
			practices[pid] = &practiceInfo{
				rel: rel, id: pid, status: status, body: body,
				supersededBy: supersededBy, appliesTo: asList(data["applies-to"]),
			}
			for _, res := range asList(data["applies-to"]) {
				resources = append(resources, struct{ rel, field, path string }{rel, "applies-to", res})
			}
		case len(parts) == 1: // bundle root
			contextList := "STACK/ARCHITECTURE/INFRA.md"
			switch {
			case specV6:
				contextList = "STACK/ARCHITECTURE/SURFACES/INFRA/DOMAIN.md"
			case specStem:
				contextList = "STACK/ARCHITECTURE/SURFACES/INFRA.md"
			}
			switch {
			case isContext:
				if docType != "Context" {
					errs = append(errs, fmt.Sprintf("%s: expected `type: Context`, got %q (F3)", rel, docType))
				}
				contextDocs[name] = strings.Contains(text, stubSentinel)
			case docType == "Context":
				errs = append(errs, fmt.Sprintf("%s: `type: Context` is reserved for %s at the bundle root (F3)", rel, contextList))
			case structural[docType] || (specStem && structuralV4[docType]) || (specV5 && structuralV5[docType]) || (specV6 && structuralV6[docType]):
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
		case len(parts) == 2:
			// v0.4 group-level dotted basenames are stem-qualified trail files.
			// Trap 14: detect any dotted basename before the feature checks so
			// unknown roles don't fall through as features named "slug.notes".
			if specStem {
				if m := anyTrailAttemptRe.FindStringSubmatch(name); m != nil {
					stem, rolePart := m[1], m[2]
					if rm := trailRoleRe.FindStringSubmatch(name); rm != nil {
						role := rm[2]
						want := trailRoleType[role]
						if docType != want {
							errs = append(errs, fmt.Sprintf("%s: expected `type: %s`, got %q (F3)", rel, want, docType))
						}
						fid := parts[0] + "/" + stem
						p := pair(fid)
						switch role {
						case "spec":
							p.spec = true
						case "plan":
							p.plan, p.planRel, p.planBody = true, rel, body
						case "test":
							p.test, p.testBody = true, body
						case "log":
							// Trap 17: same ISO-date / newest-first rules as LOG.md.
							checkLogBody(rel, text, &errs, &warns)
						}
						// surface: optional; type checked above; no pair flag required for lifecycle.
					} else {
						errs = append(errs, fmt.Sprintf("%s: unknown trail role %q — allowed roles are spec, plan, test, surface, log (F3)", rel, rolePart))
					}
					return nil
				}
			}
			// Feature: group/slug.md (all versions; v0.4 slugs carry no dots).
			if docType != "Feature" {
				errs = append(errs, fmt.Sprintf("%s: expected `type: Feature`, got %q (F3)", rel, docType))
			}
			vocab := featureStatuses
			if specV5 {
				vocab = featureStatusesV5
			}
			if !in(vocab, status) {
				errs = append(errs, fmt.Sprintf("%s: Feature `status` must be one of %s, got %q (F2)", rel, strings.Join(vocab, "|"), status))
			}
			version, _ := data["version"].(string)
			fid := strings.TrimSuffix(filepath.ToSlash(rel), ".md")
			features[fid] = &featureInfo{rel, status, version, body}
			if specV5 {
				featureDeps[fid] = asList(data["depends-on"])
			}
			checkFeatureBody(rel, body, &errs)
		case len(parts) == 3:
			fid := parts[0] + "/" + parts[1]
			switch {
			case taskFileRe.MatchString(parts[2]): // tasks: all versions
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
					resources = append(resources, struct{ rel, field, path string }{rel, "resource", res})
				}
			case !specStem && trailNames[parts[2]] != "": // nested trail: pre-v0.4 only
				want := trailNames[parts[2]]
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
			case specStem: // v0.4 task dir: only NN-slug.md tasks (no nested trail docs)
				errs = append(errs, fmt.Sprintf("%s: task directories may contain only NN-slug.md tasks (F3)", rel))
			default:
				errs = append(errs, fmt.Sprintf("%s: paired directories may contain only SPEC.md, PLAN.md, TEST.md, and NN-slug.md tasks (F3)", rel))
			}
		default:
			errs = append(errs, fmt.Sprintf("%s: nested deeper than FDF structure allows (F3)", rel))
		}
		return nil
	})

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
			if specStem {
				// Trap 15: trail-file wording (pairs may come from slug.spec.md alone).
				errs = append(errs, fmt.Sprintf("%s: trail file or task directory has no sibling feature document %s.md (F3)", fid, fid))
			} else {
				errs = append(errs, fmt.Sprintf("%s/: paired directory has no sibling feature document %s.md (F3)", fid, fid))
			}
		}
	}

	// F4 + F8: status <-> artifact invariants.
	for fid, f := range features {
		p := pairs[fid]
		switch {
		case f.status == "draft":
			if p != nil {
				if specStem {
					errs = append(errs, fmt.Sprintf("%s: status 'draft' but trail siblings or task directory %s/ exist (F4)", f.rel, fid))
				} else {
					errs = append(errs, fmt.Sprintf("%s: status 'draft' but paired directory %s/ exists (F4)", f.rel, fid))
				}
			}
			continue
		case f.status == "retired":
			// The behavior is gone; the document stays as the record. Its
			// build trail must still be there, but F8 live-test coverage no
			// longer applies — that is the point of retiring.
			if p == nil || !p.spec {
				errs = append(errs, fmt.Sprintf("%s: status 'retired' requires %s.spec.md — the record of what was built (F4)", f.rel, fid))
			}
			continue
		case !in(featureStatusesV5, f.status):
			continue // already reported (F2)
		case p == nil:
			// No paired directory at all: fall through to the per-artifact
			// checks below (against an empty pairInfo) so the error names
			// the specific missing file(s) (SPEC.md/PLAN.md/TEST.md/tasks)
			// per the F4 rule catalog, instead of a generic directory-only
			// message that names nothing.
			p = &pairInfo{tasks: map[string]string{}, deps: map[string][]string{}, depRels: map[string]string{}}
		}
		// Artifact path messages: nested trail (v0.2/0.3) vs stem trail (v0.4).
		specPath, planPath, testPath := fid+"/SPEC.md", fid+"/PLAN.md", fid+"/TEST.md"
		if specStem {
			specPath, planPath, testPath = fid+".spec.md", fid+".plan.md", fid+".test.md"
		}
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
				for _, m := range scenarioRe.FindAllStringSubmatch(f.body, -1) {
					name := strings.TrimSpace(m[1])
					if !strings.Contains(p.testBody, name) {
						errs = append(errs, fmt.Sprintf("%s: scenario %q has no test case (F8)", testPath, name))
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

	if specV5 {
		checkChangeLifecycle(changes, pairs, &errs)
		checkChangeIntegrity(changes, features, pairs, specV6, &errs)
		// Feature depends-on: existing IDs, acyclic. A separate graph from the
		// task one — these are bundle-relative feature IDs, not siblings.
		for fid, deps := range featureDeps {
			for _, d := range deps {
				if features[d] == nil {
					errs = append(errs, fmt.Sprintf("%s: depends-on %q is not a known feature (F10)", features[fid].rel, d))
				}
			}
		}
		if cyc := findCycleIDs(featureDeps); cyc != "" {
			errs = append(errs, fmt.Sprintf("feature depends-on cycle: %s (F10)", cyc))
		}
	}

	if specV6 {
		checkPracticeIntegrity(practices, practiceTrails, &errs, &warns)
		checkDebtIntegrity(debts, debtTrails, &errs, &warns)
		checkDomain(rootAbs, features, changes, opts.StrictDomain, &errs, &warns)
	}

	// F9: from the first feature onward, Context documents must exist and be
	// past their stub state. Before any feature exists a missing-or-stub
	// context doc is only a nudge (warning). Three docs on v0.3; four on v0.4.
	if specHasContext {
		contextRequired := []string{"STACK.md", "ARCHITECTURE.md", "INFRA.md"}
		switch {
		case specV6:
			contextRequired = []string{"STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md", "DOMAIN.md"}
		case specStem:
			contextRequired = []string{"STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md"}
		}
		for _, name := range contextRequired {
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
			if specStem {
				errs = append(errs, fmt.Sprintf("tasks exist under %s/ but %s.plan.md is missing (F6)", fid, fid))
			} else {
				errs = append(errs, fmt.Sprintf("%s/: tasks exist but PLAN.md is missing (F6)", fid))
			}
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
	if specV5 {
		checkReleaseChanges(rootAbs, releases, changes, &errs)
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
	verLabel := pinnedVer
	if verLabel == "" {
		verLabel = "0.2"
	}
	if len(errs) > 0 {
		fmt.Fprintf(out, "Bundle is NOT conformant with FDF v%s.\n", verLabel)
		return 1
	}
	if len(repoErrs) > 0 {
		fmt.Fprintln(out, "Bundle is FDF-conformant but has drifted from the repo (R1).")
		return 1
	}
	if opts.RepoRoot == "" {
		fmt.Fprintf(out, "Bundle is conformant with FDF v%s (R1 skipped: no project root).\n", verLabel)
		return 0
	}
	fmt.Fprintf(out, "Bundle is conformant with FDF v%s; repo integrity (R1) verified.\n", verLabel)
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
