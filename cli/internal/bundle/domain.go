package bundle

// Domain language (v0.6, widened in v0.7). DOMAIN.md is the fifth Context
// document: the canonical name for each thing the software is about, plus the
// words that must not be used for it instead. F12 checks the lexicon for
// internal consistency and reports banned words where the bundle chooses its
// words. Under a v0.6 pin that is every feature's Gherkin and every scenario
// name a change or fix declares (checkDomain, below). From v0.7 it is every
// document except the four that quote another vocabulary, and every group,
// slug and task name (lexicon.go). Each finding is cleared by a maintenance
// edit — a lexicon fix for words, a move for names — which any document takes,
// frozen ones included; text quoting what the lexicon does not govern (code, a
// link target, a quotation, a document ID, a command) is not scanned.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// instead-of is the banned-word list and (v0.7) except the phrases where a
// banned word means something else; code/see are prose and unparsed.
var insteadOfRe = regexp.MustCompile(`^\s*[-*]\s+instead-of\s*:\s*(\S.*?)\s*$`)
var exceptRe = regexp.MustCompile(`^\s*[-*]\s+except\s*:\s*(\S.*?)\s*$`)
var termBulletRe = regexp.MustCompile(`^\s*[-*]\s+[a-z-]+\s*:`)

type domainTerm struct {
	name    string
	banned  []string
	except  []string
	defined bool
}

// commaList splits a lexicon bullet's value on commas, dropping empties.
func commaList(s string) []string {
	var out []string
	for _, w := range strings.Split(s, ",") {
		if w = strings.TrimSpace(w); w != "" {
			out = append(out, w)
		}
	}
	return out
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
			cur.banned = append(cur.banned, commaList(m[1])...)
			continue
		}
		if m := exceptRe.FindStringSubmatch(line); m != nil {
			cur.except = append(cur.except, commaList(m[1])...)
			continue
		}
		if strings.TrimSpace(line) != "" && !termBulletRe.MatchString(line) {
			cur.defined = true
		}
	}
	return terms, true
}

// Lexicon is DOMAIN.md, parsed and checked, ready to scan text with.
type Lexicon struct {
	// Strict is v0.7's `strict: true`: the bundle holds itself to its lexicon,
	// so every banned word is an error in every validation.
	Strict bool

	terms     []*domainTerm
	canonical map[string]string // lowercased term -> as written
	bannedBy  map[string]string // lowercased banned word -> owning term

	// v0.7 scanning: canonical terms and `except:` phrases are masked before
	// the scan; banned words are tried longest first, so "line item" is one
	// finding rather than a second one for "item".
	maskSet *phraseSet
	wordSet *phraseSet
	words   []string
	res     map[string]*regexp.Regexp
}

// readLexicon parses DOMAIN.md and reports F12's findings about the lexicon
// itself. v7 adds the `strict` and `except:` checks and prepares the v0.7
// scanner. It returns nil when there is nothing to scan with: the file is
// absent or still a stub (F9's business), `# Terms` is missing or empty, or no
// word is banned.
func readLexicon(rootAbs string, v7 bool, errs, warns *[]string) *Lexicon {
	raw, err := os.ReadFile(filepath.Join(rootAbs, "DOMAIN.md"))
	if err != nil {
		return nil // absent: F9 reports it; nothing to check here
	}
	block, delimited, body := splitFrontmatter(strings.TrimPrefix(string(raw), "\uFEFF"))
	if !delimited {
		return nil // F1 reports it
	}
	if strings.Contains(string(raw), stubSentinel) {
		return nil // unfilled stub: F9 reports it; its example terms are not the project's
	}

	lex := &Lexicon{canonical: map[string]string{}, bannedBy: map[string]string{}}
	if v7 {
		if data, _ := parseFrontmatter(block); data != nil {
			switch v := data["strict"].(type) {
			case nil:
			case string:
				switch v {
				case "true":
					lex.Strict = true
				case "false":
				default:
					*errs = append(*errs, fmt.Sprintf("DOMAIN.md: `strict` must be true or false, got %q (F12)", v))
				}
			default:
				*errs = append(*errs, "DOMAIN.md: `strict` must be true or false, not a list (F12)")
			}
		}
	}

	terms, ok := parseLexicon(body)
	if !ok {
		*errs = append(*errs, "DOMAIN.md: missing required `# Terms` section (F12)")
		return nil
	}
	if len(terms) == 0 {
		*warns = append(*warns, "DOMAIN.md: `# Terms` names no terms — the lexicon is not doing any work yet")
		return nil
	}
	lex.terms = terms

	for _, t := range terms {
		key := strings.ToLower(t.name)
		if prev, dup := lex.canonical[key]; dup {
			*errs = append(*errs, fmt.Sprintf("DOMAIN.md: term %q is defined twice (F12)", prev))
			continue
		}
		lex.canonical[key] = t.name
		if !t.defined {
			*errs = append(*errs, fmt.Sprintf("DOMAIN.md: term %q has no definition (F12)", t.name))
		}
	}
	for _, t := range terms {
		for _, w := range t.banned {
			key := strings.ToLower(w)
			if c, isTerm := lex.canonical[key]; isTerm {
				*errs = append(*errs, fmt.Sprintf("DOMAIN.md: %q is banned by %q but is itself a canonical term (F12)", c, t.name))
				continue
			}
			if owner, taken := lex.bannedBy[key]; taken && owner != t.name {
				*errs = append(*errs, fmt.Sprintf("DOMAIN.md: %q is claimed by both %q and %q — one word, one term (F12)", w, owner, t.name))
				continue
			}
			lex.bannedBy[key] = t.name
		}
	}
	if v7 {
		checkExceptions(terms, errs)
	}
	if len(lex.bannedBy) == 0 {
		return nil
	}
	if v7 {
		lex.prepare()
	}
	return lex
}

// wordCharRe finds a letter or digit: what an `except:` phrase needs besides
// its banned word, so that `store-` cannot pass for a phrase.
var wordCharRe = regexp.MustCompile(`[\p{L}\p{N}]`)

// checkExceptions is F12's check of `except:` (v0.7): an exception names a
// phrase in which one of its term's banned words means something else, so it
// must contain one of them, and something besides — the banned word alone
// would silently un-ban itself.
func checkExceptions(terms []*domainTerm, errs *[]string) {
	for _, t := range terms {
		for _, phrase := range t.except {
			var hit *regexp.Regexp
			for _, b := range t.banned {
				if re := phraseRe(b); re.MatchString(phrase) {
					hit = re
					break
				}
			}
			switch {
			case hit == nil:
				*errs = append(*errs, fmt.Sprintf("DOMAIN.md: `except: %s` under %q contains none of the words %q bans — list only phrases in which one of them means something else (F12)", phrase, t.name, t.name))
			case !wordCharRe.MatchString(hit.ReplaceAllString(phrase, " ")):
				*errs = append(*errs, fmt.Sprintf("DOMAIN.md: `except: %s` under %q is a banned word alone — an exception is a longer phrase that says which other thing it means, or it un-bans the word (F12)", phrase, t.name))
			}
		}
	}
}

// checkDomain enforces F12 under a v0.6 pin: the lexicon's consistency, and
// banned words in every feature's Gherkin and in every scenario name a change
// or fix declares. strict promotes the banned-word findings to errors.
func checkDomain(rootAbs string, features map[string]*featureInfo, changes map[string]*changeInfo, strict bool, errs, warns *[]string) {
	lex := readLexicon(rootAbs, false, errs, warns)
	if lex == nil {
		return
	}
	canonical, bannedBy := lex.canonical, lex.bannedBy

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
// It is the v0.6 matcher; v0.7 uses phraseRe.
func wordRe(s string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(s) + `(?:e?s)?\b`)
}
