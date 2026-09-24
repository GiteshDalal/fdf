package bundle

// What the validation walk records from each file, for the checks that run
// after it. The walk decides where a file sits; a collection records what
// the document there says, the same way whichever walk found it.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/links"
)

// crossLink is one link a document writes, for the broken-link soft check.
type crossLink struct{ src, target string }

// resourceRef is one path a document names in `resource` or `applies-to`,
// for R1.
type resourceRef struct{ rel, field, path string }

// doc is a document's frontmatter, read once.
type doc struct {
	data            map[string]any
	body            string
	docType, status string
}

// collection is what the walk gathers. Its slices point at Validate's own,
// and its maps are Validate's, so the checks after the walk read them as
// they always have.
type collection struct {
	// stem, v5, v6 and v7 gate the rules each document records under: the
	// pin is 0.4, 0.5, 0.6 or 0.7, or later. v1 is a 1.0 pin, whose links
	// the links engine reads.
	stem, v5, v6, v7, v1 bool

	errs, warns *[]string
	crossLinks  *[]crossLink
	resources   *[]resourceRef
	documents   *int

	features       map[string]*featureInfo
	featureDeps    map[string][]string
	pairs          map[string]*pairInfo
	releases       map[string]*releaseInfo
	changes        map[string]*changeInfo
	practices      map[string]*practiceInfo
	practiceTrails map[string]string // practice ID -> rel of its .log.md
	debts          map[string]*debtInfo
	debtTrails     map[string]string // debt ID -> rel of its .log.md
	bugs           map[string]*bugInfo
	bugTrails      map[string]string // bug ID -> rel of its .log.md
	texts          map[string]string // v0.7: every .md file's text, for F12's scan
	contextDocs    map[string]bool   // a root Context document -> whether it is still a stub (F9)
}

func (c *collection) fail(format string, a ...any) {
	*c.errs = append(*c.errs, fmt.Sprintf(format, a...))
}

func (c *collection) warn(format string, a ...any) {
	*c.warns = append(*c.warns, fmt.Sprintf(format, a...))
}

// pair is the trail of the feature, Change or Fix id, created on first use.
func (c *collection) pair(id string) *pairInfo {
	if c.pairs[id] == nil {
		c.pairs[id] = &pairInfo{tasks: map[string]string{}, deps: map[string][]string{}, depRels: map[string]string{}}
	}
	return c.pairs[id]
}

// resource records the paths a field names, for R1.
func (c *collection) resource(rel, field string, v any) {
	for _, p := range asList(v) {
		*c.resources = append(*c.resources, resourceRef{rel, field, p})
	}
}

// read records what every Markdown file gives the checks after the walk,
// whatever its position — its text for F12 and any placeholder a scaffold
// left in it (v0.7), and its links — and returns its text without a byte
// order mark.
func (c *collection) read(rel string, raw []byte) string {
	text := strings.TrimPrefix(string(raw), "\uFEFF")
	if c.v7 {
		c.texts[filepath.ToSlash(rel)] = string(raw) // F12 reads it again, from here
		if rel != "SPEC.md" && (placeholderRe.MatchString(linkScanText(text)) || gherkinPlaceholderRe.MatchString(text)) {
			c.warn("%s: still holds a scaffold's placeholder text (`TODO —`, `<role>`) — fill it in, or delete what does not apply", rel)
		}
	}
	if c.v1 {
		for _, l := range links.Find(text) {
			if !l.InCode {
				*c.crossLinks = append(*c.crossLinks, crossLink{rel, l.Target})
			}
		}
		return text
	}
	for _, m := range linkRe.FindAllStringSubmatch(linkScanText(text), -1) {
		*c.crossLinks = append(*c.crossLinks, crossLink{rel, m[1]})
	}
	return text
}

// listingRe is a bulleted link, the line an index lists a document with.
var listingRe = regexp.MustCompile(`(?m)^\s*[-*]\s+\[.*\]\(.*\)`)

// reserved checks an INDEX.md or a LOG.md, which is not a document.
func (c *collection) reserved(rel, name, text string) {
	if name != "INDEX.md" {
		checkLogBody(rel, text, c.v7, c.errs, c.warns)
		return
	}
	block, delimited, _ := splitFrontmatter(text)
	if delimited {
		data, _ := parseFrontmatter(block)
		if rel != "INDEX.md" || data == nil || data["fdf_version"] == nil {
			c.warn("%s: index file should not carry frontmatter", rel)
		} else if v, _ := data["fdf_version"].(string); !supportedVersions[v] {
			c.fail("%s", unsupportedPin(v))
		}
	} else if rel == "INDEX.md" {
		c.warn("INDEX.md: root index should pin fdf_version")
	}
	if !listingRe.MatchString(text) {
		c.warn("%s: index file has no bulleted listing", rel)
	}
}

// document reads a document's frontmatter, with the checks every document
// takes, and counts it. ok is false when there is no frontmatter to read,
// which F1 has reported.
func (c *collection) document(rel, text string) (doc, bool) {
	*c.documents++
	block, delimited, body := splitFrontmatter(text)
	if !delimited {
		c.fail("%s: missing or unterminated frontmatter block (F1)", rel)
		return doc{}, false
	}
	data, perr := parseFrontmatter(block)
	if perr != nil {
		c.fail("%s: frontmatter is not parseable (F1): %v", rel, perr)
		return doc{}, false
	}
	d := doc{data: data, body: body}
	d.docType, _ = data["type"].(string)
	if d.docType == "" {
		c.fail("%s: missing required non-empty `type` (F1)", rel)
	}
	for _, f := range recommended {
		if data[f] == nil {
			c.warn("%s: missing recommended `%s`", rel, f)
		}
	}
	if c.v7 {
		checkTimestamp(rel, data["timestamp"], c.errs, c.warns)
	}
	d.status, _ = data["status"].(string)
	return d, true
}

// root checks a document at the bundle root: a Context document, or another,
// which may not take a type that has a position of its own.
func (c *collection) root(rel, name string, isContext bool, d doc, text string) {
	contextList := "STACK/ARCHITECTURE/INFRA.md"
	switch {
	case c.v6:
		contextList = "STACK/ARCHITECTURE/SURFACES/INFRA/DOMAIN.md"
	case c.stem:
		contextList = "STACK/ARCHITECTURE/SURFACES/INFRA.md"
	}
	switch {
	case isContext:
		if d.docType != "Context" {
			c.fail("%s: expected `type: Context`, got %q (F3)", rel, d.docType)
		}
		c.contextDocs[name] = isStub(text)
	case d.docType == "Context":
		c.fail("%s: `type: Context` is reserved for %s at the bundle root (F3)", rel, contextList)
	case structural[d.docType] || (c.stem && structuralV4[d.docType]) || (c.v5 && structuralV5[d.docType]) || (c.v6 && structuralV6[d.docType]) || (c.v7 && structuralV7[d.docType]):
		c.fail("%s: `type: %s` documents cannot live at the bundle root — move it to its FDF position (F3)", rel, d.docType)
	}
}

// feature records a Feature.
func (c *collection) feature(rel, id string, d doc) {
	if d.docType != "Feature" {
		c.fail("%s: expected `type: Feature`, got %q (F3)", rel, d.docType)
	}
	vocab := featureStatuses
	switch {
	case c.v7:
		vocab = featureStatusesV7
	case c.v5:
		vocab = featureStatusesV5
	}
	if !in(vocab, d.status) {
		c.fail("%s: Feature `status` must be one of %s, got %q (F2)", rel, strings.Join(vocab, "|"), d.status)
	}
	version, _ := d.data["version"].(string)
	f := &featureInfo{rel: rel, status: d.status, version: version, body: d.body}
	f.replacedBy = asList(d.data["replaced-by"])
	f.surface, _ = d.data["surface"].(string)
	if c.v7 {
		// A feature's own `resource` is v0.7: required on an adopted
		// feature, which has no tasks to reach its code through.
		f.resource = asList(d.data["resource"])
		c.resource(rel, "resource", d.data["resource"])
	}
	c.features[id] = f
	if c.v5 {
		c.featureDeps[id] = asList(d.data["depends-on"])
	}
	// An adopted feature may be a map entry: a Feature: block and no
	// scenarios yet. A retired one may be too, when it was adopted and never
	// backfilled; whether it was built is known only after the walk (F4).
	checkFeatureBody(rel, d.body, c.v7 && (d.status == "adopted" || d.status == "retired"), c.errs)
}

// trail records <slug>.<role>.md beside the feature, Change or Fix id.
func (c *collection) trail(rel, id, role string, d doc, text string) {
	if want := trailRoleType[role]; d.docType != want {
		c.fail("%s: expected `type: %s`, got %q (F3)", rel, want, d.docType)
	}
	p := c.pair(id)
	switch role {
	case "spec":
		p.spec = true
	case "plan":
		p.plan, p.planRel, p.planBody = true, rel, d.body
	case "test":
		p.test, p.testBody = true, d.body
		p.testTimestamp = stamp(d.data["timestamp"])
	case "log":
		// The same ISO-date, newest-first rules as a LOG.md.
		p.log = true
		checkLogBody(rel, text, c.v7, c.errs, c.warns)
	case "surface":
		p.surface = true
	}
}

// task records NN-<name>.md in the task directory of the feature, Change or
// Fix id.
func (c *collection) task(rel, id, name string, d doc) {
	if d.docType != "Task" {
		c.fail("%s: expected `type: Task`, got %q (F3)", rel, d.docType)
	}
	if !in(taskStatuses, d.status) {
		c.fail("%s: Task `status` must be one of %s, got %q (F2)", rel, strings.Join(taskStatuses, "|"), d.status)
	}
	p := c.pair(id)
	p.tasks[name] = d.status
	p.deps[name] = asList(d.data["depends-on"])
	p.depRels[name] = rel
	c.resource(rel, "resource", d.data["resource"])
}

// change records a Change or Fix under changes/.
func (c *collection) change(rel, id string, d doc) {
	if d.docType != "Change" && d.docType != "Fix" {
		c.fail("%s: expected `type: Change` or `type: Fix`, got %q (F3)", rel, d.docType)
		return
	}
	if !in(changeStatuses, d.status) {
		c.fail("%s: %s `status` must be one of %s, got %q (F2)", rel, d.docType, strings.Join(changeStatuses, "|"), d.status)
	}
	version, _ := d.data["version"].(string)
	c.changes[id] = &changeInfo{
		rel: rel, id: id, docType: d.docType, status: d.status, version: version, body: d.body,
		affects: asList(d.data["affects"]), retires: asList(d.data["retires"]),
		resolves: asList(d.data["resolves"]), timestamp: stamp(d.data["timestamp"]),
	}
	if len(fenceRe.FindAllStringSubmatch(d.body, -1)) > 0 {
		c.fail("%s: %s documents carry no Gherkin — behavior statements belong in the features they amend (F5)", rel, d.docType)
	}
	c.resource(rel, "resource", d.data["resource"])
}

// practice records a Practice under practices/.
func (c *collection) practice(rel, id string, d doc) {
	if d.docType != "Practice" {
		c.fail("%s: expected `type: Practice`, got %q (F3)", rel, d.docType)
		return
	}
	if !in(practiceStatuses, d.status) {
		c.fail("%s: Practice `status` must be one of %s, got %q (F2)", rel, strings.Join(practiceStatuses, "|"), d.status)
	}
	supersededBy, _ := d.data["superseded-by"].(string)
	c.practices[id] = &practiceInfo{
		rel: rel, id: id, status: d.status, body: d.body,
		supersededBy: supersededBy, appliesTo: asList(d.data["applies-to"]),
	}
	c.resource(rel, "applies-to", d.data["applies-to"])
}

// entry records a Debt under debts/ or a Bug under bugs/: the two registers
// share one shape and one vocabulary.
func (c *collection) entry(rel, register, id string, d doc) {
	wantType := "Debt"
	if register == "bugs" {
		wantType = "Bug"
	}
	if d.docType != wantType {
		c.fail("%s: expected `type: %s`, got %q (F3)", rel, wantType, d.docType)
		return
	}
	if !in(debtStatuses, d.status) {
		c.fail("%s: %s `status` must be one of %s, got %q (F2)", rel, wantType, strings.Join(debtStatuses, "|"), d.status)
	}
	if register == "bugs" {
		c.bugs[id] = &bugInfo{
			rel: rel, id: id, status: d.status, body: d.body,
			affects: asList(d.data["affects"]), resource: asList(d.data["resource"]),
		}
	} else {
		title, _ := d.data["title"].(string)
		timestamp, _ := d.data["timestamp"].(string)
		c.debts[id] = &debtInfo{
			rel: rel, id: id, status: d.status, body: d.body,
			title: title, timestamp: timestamp, resource: asList(d.data["resource"]),
		}
	}
	c.resource(rel, "resource", d.data["resource"])
}

// registerLog records <slug>.log.md beside the practice, debt or bug id: the
// only sibling those documents take.
func (c *collection) registerLog(rel, register, id string, d doc, text string) {
	if d.docType != "Log" {
		c.fail("%s: expected `type: Log`, got %q (F3)", rel, d.docType)
	}
	checkLogBody(rel, text, c.v7, c.errs, c.warns)
	switch register {
	case "practices":
		c.practiceTrails[id] = rel
	case "bugs":
		c.bugTrails[id] = rel
	default:
		c.debtTrails[id] = rel
	}
}

// release records releases/<version>.md.
func (c *collection) release(rel, version string, d doc) {
	if d.docType != "Release" {
		c.fail("%s: expected `type: Release`, got %q (F3)", rel, d.docType)
	}
	if !in(releaseStatuses, d.status) {
		c.fail("%s: Release `status` must be one of %s, got %q (F2)", rel, strings.Join(releaseStatuses, "|"), d.status)
	}
	if d.data["date"] == nil {
		c.warn("%s: missing recommended `date`", rel)
	}
	body := d.body
	if c.v7 {
		body = linkScanText(body) // a link in code is a sample, not a listing
	}
	c.releases[version] = &releaseInfo{rel, d.status, body}
}
