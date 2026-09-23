package bundle

// v0.6 domain language. DOMAIN.md is the fifth Context document: the canonical
// name for each thing the software is about, plus the words that must not be
// used for it instead. F12 checks the lexicon for internal consistency and
// reports banned words in the names the bundle chooses — every feature's
// Gherkin, and every scenario name a change or fix declares. Each finding is
// cleared by a lexicon fix, the one edit any document takes, frozen ones
// included; text quoting what the lexicon does not govern (a document ID, a
// command, a surface string) is not scanned.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// instead-of is the banned-word list; code/see are prose and unparsed.
var insteadOfRe = regexp.MustCompile(`^\s*[-*]\s+instead-of\s*:\s*(\S.*?)\s*$`)
var termBulletRe = regexp.MustCompile(`^\s*[-*]\s+[a-z-]+\s*:`)

type domainTerm struct {
	name    string
	banned  []string
	defined bool
}

// parseLexicon reads DOMAIN.md's `# Terms` section. It returns the terms in
// document order and false when the section is absent.
func parseLexicon(body string) ([]*domainTerm, bool) {
	section, ok := sectionText(body, "Terms")
	if !ok {
		return nil, false
	}
	var terms []*domainTerm
	var cur *domainTerm
	for _, line := range strings.Split(section, "\n") {
		if m := subHeadingRe.FindStringSubmatch(line); m != nil {
			cur = &domainTerm{name: strings.TrimSpace(m[1])}
			terms = append(terms, cur)
			continue
		}
		if cur == nil {
			continue
		}
		if m := insteadOfRe.FindStringSubmatch(line); m != nil {
			for _, w := range strings.Split(m[1], ",") {
				if w = strings.TrimSpace(w); w != "" {
					cur.banned = append(cur.banned, w)
				}
			}
			continue
		}
		if strings.TrimSpace(line) != "" && !termBulletRe.MatchString(line) {
			cur.defined = true
		}
	}
	return terms, true
}

// checkDomain enforces F12. strict promotes the banned-word findings from
// warnings to errors.
func checkDomain(rootAbs string, features map[string]*featureInfo, changes map[string]*changeInfo, strict bool, errs, warns *[]string) {
	raw, err := os.ReadFile(filepath.Join(rootAbs, "DOMAIN.md"))
	if err != nil {
		return // absent: F9 reports it; nothing to check here
	}
	_, delimited, body := splitFrontmatter(strings.TrimPrefix(string(raw), "\uFEFF"))
	if !delimited {
		return // F1 reports it
	}
	if strings.Contains(string(raw), stubSentinel) {
		return // unfilled stub: F9 reports it; its example terms are not the project's
	}

	terms, ok := parseLexicon(body)
	if !ok {
		*errs = append(*errs, "DOMAIN.md: missing required `# Terms` section (F12)")
		return
	}
	if len(terms) == 0 {
		*warns = append(*warns, "DOMAIN.md: `# Terms` names no terms — the lexicon is not doing any work yet")
		return
	}

	canonical := map[string]string{} // lowercased term -> as written
	bannedBy := map[string]string{}  // lowercased banned word -> owning term
	for _, t := range terms {
		key := strings.ToLower(t.name)
		if prev, dup := canonical[key]; dup {
			*errs = append(*errs, fmt.Sprintf("DOMAIN.md: term %q is defined twice (F12)", prev))
			continue
		}
		canonical[key] = t.name
		if !t.defined {
			*errs = append(*errs, fmt.Sprintf("DOMAIN.md: term %q has no definition (F12)", t.name))
		}
	}
	for _, t := range terms {
		for _, w := range t.banned {
			key := strings.ToLower(w)
			if c, isTerm := canonical[key]; isTerm {
				*errs = append(*errs, fmt.Sprintf("DOMAIN.md: %q is banned by %q but is itself a canonical term (F12)", c, t.name))
				continue
			}
			if owner, taken := bannedBy[key]; taken && owner != t.name {
				*errs = append(*errs, fmt.Sprintf("DOMAIN.md: %q is claimed by both %q and %q — one word, one term (F12)", w, owner, t.name))
				continue
			}
			bannedBy[key] = t.name
		}
	}
	if len(bannedBy) == 0 {
		return
	}

	// Canonical terms are blanked before the scan so a term that contains a
	// banned word ("Line Item" over a banned "item") does not report itself.
	termRes := make([]*regexp.Regexp, 0, len(canonical))
	for _, name := range canonical {
		termRes = append(termRes, wordRe(name))
	}
	bannedWords := make([]string, 0, len(bannedBy))
	for w := range bannedBy {
		bannedWords = append(bannedWords, w)
	}
	sort.Strings(bannedWords)

	report := func(rel, where, word, term string) {
		msg := fmt.Sprintf("%s: %s uses %q, which DOMAIN.md bans in favour of %q (F12)", rel, where, word, term)
		if strict {
			*errs = append(*errs, msg)
			return
		}
		*warns = append(*warns, msg)
	}
	scan := func(rel, where, text string) {
		for _, re := range termRes {
			text = re.ReplaceAllString(text, " ")
		}
		for _, w := range bannedWords {
			if wordRe(w).MatchString(text) {
				report(rel, where, w, bannedBy[w])
			}
		}
	}

	fids := make([]string, 0, len(features))
	for fid := range features {
		fids = append(fids, fid)
	}
	sort.Strings(fids)
	for _, fid := range fids {
		f := features[fid]
		var g strings.Builder
		for _, m := range fenceRe.FindAllStringSubmatch(f.body, -1) {
			g.WriteString(m[1])
			g.WriteString("\n")
		}
		scan(f.rel, "Gherkin", g.String())
	}

	cids := make([]string, 0, len(changes))
	for cid := range changes {
		cids = append(cids, cid)
	}
	sort.Strings(cids)
	for _, cid := range cids {
		c := changes[cid]
		isFix := c.docType == "Fix"
		heading := scenarioChangesHeading
		if isFix {
			heading = regressionCasesHeading
		}
		// Only the scenario names a declaration carries are scanned, at any
		// status: a lexicon fix corrects them in a finished document as in an
		// open one. The `## <feature-id>` headings and a regression case's
		// verification quote a document ID, a command, a path or a surface
		// string as it stands, which no lexicon fix may change.
		decls := parseDecls(c.body, heading, !isFix)
		declFids := make([]string, 0, len(decls))
		for fid := range decls {
			declFids = append(declFids, fid)
		}
		sort.Strings(declFids)
		seen := map[string]bool{}
		entry := func(where, name string) {
			if !seen[where] {
				seen[where] = true
				scan(c.rel, where, name)
			}
		}
		for _, fid := range declFids {
			d := decls[fid]
			for _, n := range d.adds {
				entry("`add: "+n+"`", n)
			}
			for _, n := range d.modifies {
				entry("`modify: "+n+"`", n)
			}
			for _, n := range d.removes {
				entry("`remove: "+n+"`", n)
			}
			for _, n := range append(append([]string{}, d.regressions...), d.missingVerification...) {
				entry(fmt.Sprintf("regression case %q", n), n)
			}
		}
	}
}

// wordRe matches a term or banned phrase whole-word and case-insensitively,
// tolerating a regular plural: scenario text says "3 items" far more often
// than "1 item", and a check that missed every plural would miss most drift.
func wordRe(s string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(s) + `(?:e?s)?\b`)
}
