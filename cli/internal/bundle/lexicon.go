package bundle

// The v0.7 lexicon scan. F12 reads every document the bundle writes in its
// own words — all but SPEC.md, DOMAIN.md, slug.test.md and slug.surface.md,
// which quote someone else's vocabulary — and every group, slug and task
// name. Inside a scanned document, text that quotes rather than chooses is
// masked first: code spans and non-Gherkin code blocks, link targets, URLs,
// HTML comments, double-quoted text outside Gherkin, and a declaration's
// `## <feature-id>` headings and regression-case verifications. Masking
// blanks bytes in place, so every finding keeps its exact position; that is
// what lets `fdf lexicon` report file:line:col and fix a word where it stands.

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
	fenceLineRe   = regexp.MustCompile("^(`{3,}|~{3,})(.*)$")
	codeSpanRe    = regexp.MustCompile("`+[^`\n]*?`+")
	linkTargetRe  = regexp.MustCompile(`\]\([^)\n]*\)`)
	refDefRe      = regexp.MustCompile(`^[ \t]{0,3}\[[^\]\n]+\]:[ \t]*\S`)
	autoLinkRe    = regexp.MustCompile(`<[a-zA-Z][a-zA-Z0-9+.-]*:[^>\s]*>`)
	bareURLRe     = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.-]*://[^\s)\]>]+`)
	dquoteRe      = regexp.MustCompile(`"[^"\n]*"|“[^”\n]*”`)
	chosenKeyRe   = regexp.MustCompile(`^(title|description|tags)\s*:`)
	anyKeyRe      = regexp.MustCompile(`^[A-Za-z0-9_-]+\s*:`)
	taskNameRe    = regexp.MustCompile(`^\d{2}-(.+)\.md$`)
)

// registerDirs are the reserved bundle-root directories: their own names are
// fixed by the format, and only the groups inside them are chosen.
var registerDirs = map[string]bool{"changes": true, "practices": true, "debts": true, "bugs": true, "releases": true}

// phrasePattern is the v0.7 matcher for a term, banned word or exception
// phrase: its words in order, with any run of spaces or hyphens between them —
// prose wraps a phrase across lines, and a name joins its words with hyphens —
// and a regular plural on the last one.
func phrasePattern(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = regexp.QuoteMeta(w)
	}
	return strings.Join(words, `[\s-]+`) + `(?:e?s)?`
}

// phraseRe matches one phrase whole-word and case-insensitively.
func phraseRe(s string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\b` + phrasePattern(s) + `\b`)
}

// phraseSet finds any of a set of phrases, leftmost first and, at one
// position, longest first — so "line item" claims both its words before "item"
// can claim one. A phrase can only start where a word starts, so instead of
// running one large case-insensitive alternation over every byte it looks each
// word up by its lowercased form and tries only the phrases that begin with it.
type phraseSet struct {
	byFirst map[string][]*regexp.Regexp // leading word, lowercased -> anchored matchers, longest phrase first
	all     *regexp.Regexp              // fallback when a phrase does not start with a word character
}

// isWordByte is regexp's \w: the characters a word boundary separates.
func isWordByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// leadingWord is the lowercased run of word characters a phrase starts with —
// the token a text's word must equal for the phrase to start there.
func leadingWord(p string) string {
	i := 0
	for i < len(p) && isWordByte(p[i]) {
		i++
	}
	return strings.ToLower(p[:i])
}

// newPhraseSet compiles phrases, which must already be ordered longest first.
func newPhraseSet(phrases []string) *phraseSet {
	if len(phrases) == 0 {
		return nil
	}
	ps := &phraseSet{byFirst: map[string][]*regexp.Regexp{}}
	parts := make([]string, len(phrases))
	odd := false
	for i, p := range phrases {
		parts[i] = phrasePattern(p)
		odd = odd || leadingWord(p) == ""
	}
	if odd {
		// A phrase that does not start with a word character cannot be found
		// by its leading word; match the whole set in one alternation instead
		// — correct, only slower.
		ps.all = regexp.MustCompile(`(?i)\b(?:` + strings.Join(parts, "|") + `)\b`)
		return ps
	}
	for i, p := range phrases {
		key := leadingWord(p)
		ps.byFirst[key] = append(ps.byFirst[key], regexp.MustCompile(`^(?i)`+parts[i]+`\b`))
	}
	return ps
}

// findAll returns the [start, end) of every match in b, in order, without
// overlaps.
func (ps *phraseSet) findAll(b []byte) [][2]int {
	var out [][2]int
	if ps.all != nil {
		for _, loc := range ps.all.FindAllIndex(b, -1) {
			out = append(out, [2]int{loc[0], loc[1]})
		}
		return out
	}
	var word []byte
	lookup := func(w []byte) []*regexp.Regexp { return ps.byFirst[string(w)] } // no allocation
	for i := 0; i < len(b); {
		if !isWordByte(b[i]) || (i > 0 && isWordByte(b[i-1])) {
			i++
			continue
		}
		j := i
		for j < len(b) && isWordByte(b[j]) {
			j++
		}
		word = append(word[:0], b[i:j]...)
		for k, c := range word {
			if c >= 'A' && c <= 'Z' {
				word[k] = c + 'a' - 'A'
			}
		}
		cands := lookup(word)
		if cands == nil && len(word) > 1 && word[len(word)-1] == 's' { // a plural first word
			if cands = lookup(word[:len(word)-1]); cands == nil && len(word) > 2 && word[len(word)-2] == 'e' {
				cands = lookup(word[:len(word)-2])
			}
		}
		matched := false
		for _, re := range cands {
			if loc := re.FindIndex(b[i:]); loc != nil {
				out = append(out, [2]int{i, i + loc[1]})
				i += loc[1]
				matched = true
				break
			}
		}
		if !matched {
			i = j
		}
	}
	return out
}

var spaceRunRe = regexp.MustCompile(`[\s-]+`)

// bannedFor maps a matched word back to the banned word it is: lowercased,
// with its separators normalized and a regular plural dropped.
func (l *Lexicon) bannedFor(matched string) string {
	n := spaceRunRe.ReplaceAllString(strings.ToLower(matched), " ")
	for _, cand := range []string{n, strings.TrimSuffix(n, "es"), strings.TrimSuffix(n, "s")} {
		if _, ok := l.bannedBy[cand]; ok {
			return cand
		}
	}
	for _, w := range l.words { // a banned word written with other separators
		if l.res[w].MatchString(matched) {
			return w
		}
	}
	return n
}

// byLengthThenAlpha orders phrases longest first, so a longer phrase claims
// its words before a shorter one inside it can.
func byLengthThenAlpha(s []string) {
	sort.Slice(s, func(i, j int) bool {
		li, lj := utf8.RuneCountInString(s[i]), utf8.RuneCountInString(s[j])
		if li != lj {
			return li > lj
		}
		return s[i] < s[j]
	})
}

// prepare compiles the v0.7 scanner: one pass for the masks, one for the
// banned words. A mask protects a canonical term or an `except:` phrase that
// contains a banned word — "Line Item" over a banned "item" — and nothing
// else: masking a term that contains no banned word would only hide it from a
// banned phrase built around it ("item order" when Order is a term).
func (l *Lexicon) prepare() {
	l.res = map[string]*regexp.Regexp{}
	for w := range l.bannedBy {
		l.words = append(l.words, w)
		l.res[w] = phraseRe(w)
	}
	byLengthThenAlpha(l.words)
	l.wordSet = newPhraseSet(l.words)

	var masks []string
	for _, name := range l.canonical {
		if len(l.wordSet.findAll([]byte(name))) > 0 {
			masks = append(masks, name)
		}
	}
	for _, t := range l.terms {
		masks = append(masks, t.except...)
	}
	byLengthThenAlpha(masks)
	l.maskSet = newPhraseSet(masks)
}

type hit struct {
	start, end   int
	banned, term string
}

// blank replaces b[s:e] with spaces, keeping line breaks, so masking never
// moves a byte.
func blank(b []byte, s, e int) {
	for i := s; i < e; i++ {
		if b[i] != '\n' && b[i] != '\r' {
			b[i] = ' '
		}
	}
}

// scan finds the banned words in already-masked text, in order.
func (l *Lexicon) scan(text string) []hit {
	b := []byte(text)
	if l.maskSet != nil {
		for _, loc := range l.maskSet.findAll(b) {
			blank(b, loc[0], loc[1])
		}
	}
	var hits []hit
	for _, loc := range l.wordSet.findAll(b) {
		w := l.bannedFor(string(b[loc[0]:loc[1]]))
		hits = append(hits, hit{loc[0], loc[1], w, l.bannedBy[w]})
	}
	return hits
}

// DomainScanned reports whether F12 reads a document's text (v0.7): every
// document except the four that quote another vocabulary — the vendored
// SPEC.md, DOMAIN.md itself, and the slug.test.md and slug.surface.md that
// quote what a surface shows. README.md at the root is not an FDF document.
func DomainScanned(rel string) bool {
	rel = filepath.ToSlash(rel)
	switch rel {
	case "SPEC.md", "DOMAIN.md", "README.md":
		return false
	}
	base := path.Base(rel)
	return strings.HasSuffix(base, ".md") &&
		!strings.HasSuffix(base, ".test.md") && !strings.HasSuffix(base, ".surface.md")
}

// eachLine calls fn with the bounds of every line in b[from:], excluding the
// newline.
func eachLine(b []byte, from int, fn func(s, e int)) {
	for pos := from; pos < len(b); {
		e := pos
		for e < len(b) && b[e] != '\n' {
			e++
		}
		fn(pos, e)
		pos = e + 1
	}
}

// maskFrontmatter keeps only the values of title, description and tags — the
// frontmatter a person writes in words — and blanks the rest: IDs, paths and
// statuses are not vocabulary.
func maskFrontmatter(b []byte, end int) {
	keep := false
	eachLine(b[:end], 0, func(s, e int) {
		line := string(b[s:e])
		switch {
		case chosenKeyRe.MatchString(line):
			loc := chosenKeyRe.FindStringIndex(line)
			blank(b, s, s+loc[1])
			keep = true
		case anyKeyRe.MatchString(line):
			blank(b, s, e)
			keep = false
		case keep && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "-")):
			// a continuation of title, description or tags (a list item)
		default:
			blank(b, s, e)
			keep = false
		}
	})
}

// maskBody blanks the text in a body that quotes rather than chooses.
func maskBody(b []byte, start int, docType string) {
	for _, loc := range htmlCommentRe.FindAllIndex(b[start:], -1) {
		blank(b, start+loc[0], start+loc[1])
	}

	fence, gherkin := "", false
	eachLine(b, start, func(s, e int) {
		line := string(b[s:e])
		t := strings.TrimRight(strings.TrimLeft(line, " \t"), " \t\r")
		if fence != "" {
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				fence = ""
				blank(b, s, e)
			} else if !gherkin {
				blank(b, s, e)
			}
			return
		}
		if m := fenceLineRe.FindStringSubmatch(t); m != nil {
			fence = m[1]
			gherkin = strings.HasPrefix(strings.ToLower(strings.TrimSpace(m[2])), "gherkin")
			blank(b, s, e)
			return
		}
		if refDefRe.MatchString(line) {
			blank(b, s, e)
			return
		}
		for _, re := range []*regexp.Regexp{codeSpanRe, autoLinkRe, bareURLRe, dquoteRe} {
			for _, loc := range re.FindAllIndex(b[s:e], -1) {
				blank(b, s+loc[0], s+loc[1])
			}
		}
		for _, loc := range linkTargetRe.FindAllIndex(b[s:e], -1) {
			blank(b, s+loc[0]+1, s+loc[1]) // keep "]": the link text is prose
		}
	})

	if docType != "Change" && docType != "Fix" && docType != "Bug" {
		return
	}
	// A declaration's `## <feature-id>` headings quote document IDs, and a
	// regression case's verification quotes a command, a path or a surface's
	// wording; the scenario names around them are the bundle's own words.
	section := ""
	eachLine(b, start, func(s, e int) {
		line := strings.TrimRight(string(b[s:e]), "\r")
		if m := headingRe.FindStringSubmatch(line); m != nil {
			section = strings.TrimSpace(m[1])
			return
		}
		inDecl := strings.EqualFold(section, scenarioChangesHeading) ||
			strings.EqualFold(section, regressionCasesHeading) ||
			strings.EqualFold(section, violatesHeading)
		if !inDecl {
			return
		}
		if subHeadingRe.MatchString(line) {
			blank(b, s, e)
			return
		}
		if strings.EqualFold(section, regressionCasesHeading) && listItemRe.MatchString(line) {
			if loc := verbatimSepRe.FindStringIndex(line); loc != nil {
				blank(b, s+loc[0], e)
			}
		}
	})
}

// maskDocument returns a copy of a document's text in which everything F12
// does not read is blanked, byte for byte.
func maskDocument(text string) []byte {
	b := []byte(text)
	start, docType := 0, ""
	if block, ok, body := splitFrontmatter(text); ok {
		start = len(text) - len(body)
		if data, _ := parseFrontmatter(block); data != nil {
			docType, _ = data["type"].(string)
		}
		maskFrontmatter(b, start)
	}
	maskBody(b, start, docType)
	return b
}

// Occurrence is one banned word the bundle chose: in a document's text, or —
// when Line is 0 — in the name of a document or a directory (Rel ends in "/").
type Occurrence struct {
	Rel        string // bundle-relative, slash-separated
	Line, Col  int    // 1-based; Col counts characters
	Start, End int    // byte offsets in the file (text occurrences only)
	Text       string // the word as written
	Banned     string // the instead-of word it matched
	Term       string // the canonical term it is banned in favour of
}

// InName reports whether the occurrence is in a name rather than in text.
func (o Occurrence) InName() bool { return o.Line == 0 }

// lineIndex holds the byte offset at which each line of a text starts, so
// every finding's position is a binary search rather than a rescan.
type lineIndex []int

func newLineIndex(text string) lineIndex {
	idx := lineIndex{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			idx = append(idx, i+1)
		}
	}
	return idx
}

// at converts a byte offset into a 1-based line and character column.
func (idx lineIndex) at(text string, off int) (int, int) {
	line := sort.Search(len(idx), func(i int) bool { return idx[i] > off }) // first line starting after off
	return line, utf8.RuneCountInString(text[idx[line-1]:off]) + 1
}

// ScanDocument returns the banned words a document's text chooses: its body
// and its title, description and tags, with quoting masked. It does not check
// DomainScanned; the caller decides what to read.
func (l *Lexicon) ScanDocument(rel, text string) []Occurrence {
	offset := 0
	if strings.HasPrefix(text, "\uFEFF") {
		offset = len("\uFEFF")
	}
	masked := maskDocument(text[offset:])
	hits := l.scan(string(masked))
	if len(hits) == 0 {
		return nil
	}
	idx := newLineIndex(text)
	out := make([]Occurrence, 0, len(hits))
	for _, h := range hits {
		s, e := h.start+offset, h.end+offset
		line, col := idx.at(text, s)
		out = append(out, Occurrence{Rel: rel, Line: line, Col: col, Start: s, End: e, Text: text[s:e], Banned: h.banned, Term: h.term})
	}
	return out
}

// scanName finds the banned words in a group, slug or task name.
func (l *Lexicon) scanName(rel, name string) []Occurrence {
	var out []Occurrence
	for _, h := range l.scan(strings.ReplaceAll(name, "-", " ")) {
		out = append(out, Occurrence{Rel: rel, Text: name[h.start:h.end], Banned: h.banned, Term: h.term})
	}
	return out
}

// dirName returns the name a directory's author chose: a feature group, or a
// group inside changes/, practices/, debts/ or bugs/. The register directories
// themselves and task directories (whose name is their document's) have none.
func dirName(rootAbs, rel string) (string, bool) {
	parts := strings.Split(rel, "/")
	switch {
	case len(parts) == 1 && !registerDirs[parts[0]]:
		return parts[0], true
	case len(parts) == 2 && (parts[0] == "practices" || parts[0] == "debts" || parts[0] == "bugs"):
		return parts[1], true
	case len(parts) == 2 && parts[0] == "changes" && !exists(filepath.Join(rootAbs, "changes", parts[1]+".md")):
		return parts[1], true
	}
	return "", false
}

// docName returns the name a document's author chose: its slug, or a task's
// name without its NN- prefix. Reserved and Context files, trail siblings
// (named by their document) and releases (named by a version) have none.
func docName(rel string) (string, bool) {
	parts := strings.Split(rel, "/")
	base := parts[len(parts)-1]
	if len(parts) == 1 || base == "INDEX.md" || base == "LOG.md" || parts[0] == "releases" {
		return "", false
	}
	if m := taskNameRe.FindStringSubmatch(base); m != nil {
		return m[1], true
	}
	stem := strings.TrimSuffix(base, ".md")
	if strings.Contains(stem, ".") {
		return "", false
	}
	return stem, true
}

// ScanBundle returns every banned word F12 sees in the bundle (v0.7), in path
// order: a path's name first, then its text.
func ScanBundle(root string, l *Lexicon) []Occurrence {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil
	}
	return scanBundle(rootAbs, l, nil)
}

// scanBundle is ScanBundle with the documents' texts already read, when the
// caller has them: validation reads every file once, and reading each again
// here would double the cost of the gate that runs after every edit.
func scanBundle(rootAbs string, l *Lexicon, texts map[string]string) []Occurrence {
	var out []Occurrence
	filepath.WalkDir(rootAbs, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel := relTo(rootAbs, p)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if name, ok := dirName(rootAbs, rel); ok {
				out = append(out, l.scanName(rel+"/", name)...)
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".md") {
			return nil
		}
		if name, ok := docName(rel); ok {
			out = append(out, l.scanName(rel, name)...)
		}
		if DomainScanned(rel) {
			text, ok := texts[rel]
			if !ok {
				raw, rerr := os.ReadFile(p)
				if rerr != nil {
					return nil
				}
				text = string(raw)
			}
			out = append(out, l.ScanDocument(rel, text)...)
		}
		return nil
	})
	return out
}

// LoadLexicon reads DOMAIN.md under v0.7 rules, for tools that scan a bundle
// outside validation (`fdf lexicon`). problems are F12's findings about the
// lexicon itself; lex is nil when there is nothing to scan with.
func LoadLexicon(root string) (*Lexicon, []string) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, nil
	}
	var errs, warns []string
	lex := readLexicon(rootAbs, true, &errs, &warns)
	return lex, append(errs, warns...)
}

// checkDomainV7 enforces F12 under a v0.7 pin: the lexicon's consistency, then
// one line per document or name that uses a banned word, capped so a sweep in
// progress does not bury every other finding. strict comes from the flag;
// DOMAIN.md's `strict: true` turns it on too. texts holds the documents the
// validation walk already read, by bundle-relative path.
func checkDomainV7(rootAbs string, strictFlag bool, texts map[string]string, errs, warns *[]string) {
	lex := readLexicon(rootAbs, true, errs, warns)
	if lex == nil {
		return
	}
	strict := strictFlag || lex.Strict

	type group struct {
		rel   string
		name  bool
		words []string
		count map[string]int
		term  map[string]string
	}
	var groups []*group
	byKey := map[string]*group{}
	for _, o := range scanBundle(rootAbs, lex, texts) {
		key := o.Rel
		if o.InName() {
			key = "\x00" + o.Rel
		}
		g := byKey[key]
		if g == nil {
			g = &group{rel: o.Rel, name: o.InName(), count: map[string]int{}, term: map[string]string{}}
			byKey[key] = g
			groups = append(groups, g)
		}
		if g.count[o.Banned] == 0 {
			g.words = append(g.words, o.Banned)
		}
		g.count[o.Banned]++
		g.term[o.Banned] = o.Term
	}

	const wordsPerLine = 8
	var lines []string
	for _, g := range groups {
		parts := make([]string, 0, len(g.words))
		for i, w := range g.words {
			if i == wordsPerLine {
				parts = append(parts, fmt.Sprintf("and %d more", len(g.words)-wordsPerLine))
				break
			}
			s := fmt.Sprintf("%q", w)
			if n := g.count[w]; n > 1 {
				s += fmt.Sprintf(" ×%d", n)
			}
			parts = append(parts, fmt.Sprintf("%s → %q", s, g.term[w]))
		}
		if g.name {
			lines = append(lines, fmt.Sprintf("%s: name uses banned %s — rename it with `fdf mv` (F12)", g.rel, strings.Join(parts, ", ")))
		} else {
			lines = append(lines, fmt.Sprintf("%s: uses banned %s (F12)", g.rel, strings.Join(parts, ", ")))
		}
	}
	if len(lines) > domainReportCap {
		more := len(lines) - domainReportCap
		lines = append(lines[:domainReportCap], fmt.Sprintf("and %d more document(s) or name(s) use banned words — `fdf lexicon` lists every occurrence (F12)", more))
	}
	for _, line := range lines {
		if strict {
			*errs = append(*errs, line)
		} else {
			*warns = append(*warns, line)
		}
	}
}

// TermNames lists the lexicon's canonical terms in document order.
func (l *Lexicon) TermNames() []string {
	out := make([]string, 0, len(l.terms))
	for _, t := range l.terms {
		out = append(out, t.name)
	}
	return out
}
