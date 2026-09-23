package refactor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/logs"
)

// LexiconOptions selects what `fdf lexicon` does.
type LexiconOptions struct {
	Term   string // only this term's banned words; "" for all
	All    bool   // report every occurrence, not the first few per word
	Fix    bool   // replace them
	DryRun bool   // with Fix: show the diff, change nothing
}

var scenarioLineRe = regexp.MustCompile(`(?m)^\s*Scenario(?: Outline)?:\s*(\S[^\n]*?)\s*$`)

// Lexicon reports every banned word F12 sees, or — with Fix — replaces them
// with their terms. A replacement is a lexicon fix, the maintenance edit every
// document takes: it keeps plurals and sentence capitals, mends "a"/"an", and
// renames a scenario name everywhere it is a join. What it cannot decide it
// leaves alone and lists: a word in italics (usually a mention of the word,
// which belongs in a code span), a label quoted in a Gherkin step (the step is
// reworded around the concept by hand), a Gherkin table cell, and every name
// (a move, `fdf mv`, whose new name a person picks).
func Lexicon(root string, opts LexiconOptions, out io.Writer) int {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	lex, problems := bundle.LoadLexicon(rootAbs)
	for _, pr := range problems {
		fmt.Fprintln(out, "lexicon: "+pr)
	}
	if lex == nil {
		fmt.Fprintln(out, "nothing to scan with: DOMAIN.md is missing, still a stub, or bans no word.")
		if len(problems) > 0 {
			return 1
		}
		return 0
	}
	if opts.Fix && opts.Term == "" {
		fmt.Fprintf(out, "error: --fix sweeps one term at a time — add --term <Term> (terms: %s).\n", strings.Join(lex.TermNames(), ", "))
		fmt.Fprintln(out, "  A sweep is reviewed before it lands: `fdf lexicon --term <Term>` to triage, then --fix --dry-run.")
		return 1
	}
	occ := bundle.ScanBundle(rootAbs, lex)
	if opts.Term != "" {
		known := false
		var kept []bundle.Occurrence
		for _, o := range occ {
			if strings.EqualFold(o.Term, opts.Term) {
				kept = append(kept, o)
			}
		}
		for _, t := range lex.TermNames() {
			known = known || strings.EqualFold(t, opts.Term)
		}
		if !known {
			fmt.Fprintf(out, "error: %q is not a term in DOMAIN.md (terms: %s)\n", opts.Term, strings.Join(lex.TermNames(), ", "))
			return 1
		}
		occ = kept
	}
	texts := map[string]string{}
	textOf := func(rel string) string {
		if t, ok := texts[rel]; ok {
			return t
		}
		raw, _ := os.ReadFile(filepath.Join(rootAbs, filepath.FromSlash(rel)))
		texts[rel] = string(raw)
		return texts[rel]
	}
	if opts.Fix {
		return fixLexicon(rootAbs, lex, occ, textOf, opts.DryRun, out)
	}
	return reportLexicon(lex, occ, textOf, opts.All, out)
}

// manual reports why an occurrence is left for a person, or "".
func manual(o bundle.Occurrence, text string) string {
	if o.InName() {
		return "name"
	}
	if o.Start > 0 && o.End < len(text) && (text[o.Start-1] == '*' || text[o.Start-1] == '_') && text[o.End] == text[o.Start-1] {
		return "italic — a mention? put it in a code span"
	}
	lineStart := strings.LastIndex(text[:o.Start], "\n") + 1
	lineEnd := strings.Index(text[o.Start:], "\n")
	if lineEnd < 0 {
		lineEnd = len(text)
	} else {
		lineEnd += o.Start
	}
	line := text[lineStart:lineEnd]
	if inGherkin(text, lineStart) {
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			return "Gherkin table cell — test data, reword by hand"
		}
		if strings.Count(text[lineStart:o.Start], `"`)%2 == 1 {
			return "a label quoted in a Gherkin step — reword the step around the concept"
		}
	}
	return ""
}

// inGherkin reports whether a line starts inside a ```gherkin fence.
func inGherkin(text string, at int) bool {
	in, fence := false, ""
	for _, line := range strings.SplitAfter(text[:at], "\n") {
		t := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				fence, in = "", false
			}
			continue
		}
		if m := fenceRe.FindString(t); m != "" {
			fence = m
			in = strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(t, m))), "gherkin")
		}
	}
	return in
}

func reportLexicon(lex *bundle.Lexicon, occ []bundle.Occurrence, textOf func(string) string, all bool, out io.Writer) int {
	if len(occ) == 0 {
		fmt.Fprintln(out, "no banned word anywhere F12 reads — the bundle speaks its lexicon.")
		return 0
	}
	type group struct {
		banned, term string
		occ          []bundle.Occurrence
		docs         map[string]bool
	}
	byWord := map[string]*group{}
	var names []bundle.Occurrence
	docs := map[string]bool{}
	for _, o := range occ {
		docs[o.Rel] = true
		if o.InName() {
			names = append(names, o)
			continue
		}
		g := byWord[o.Banned]
		if g == nil {
			g = &group{banned: o.Banned, term: o.Term, docs: map[string]bool{}}
			byWord[o.Banned] = g
		}
		g.occ = append(g.occ, o)
		g.docs[o.Rel] = true
	}
	strict := "not strict"
	if lex.Strict {
		strict = "strict: every one is a validation error"
	}
	fmt.Fprintf(out, "F12 — %d banned word(s) in %d place(s); DOMAIN.md is %s.\n", len(occ), len(docs), strict)

	groups := make([]*group, 0, len(byWord))
	for _, g := range byWord {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].occ) != len(groups[j].occ) {
			return len(groups[i].occ) > len(groups[j].occ)
		}
		return groups[i].banned < groups[j].banned
	})
	const perWord = 8
	for _, g := range groups {
		fmt.Fprintf(out, "\n%q → %q — %d in %d document(s)\n", g.banned, g.term, len(g.occ), len(g.docs))
		for i, o := range g.occ {
			if !all && i == perWord {
				fmt.Fprintf(out, "  …%d more (--all, or --term %q)\n", len(g.occ)-perWord, g.term)
				break
			}
			text := textOf(o.Rel)
			note := ""
			if why := manual(o, text); why != "" {
				note = "   [" + why + "]"
			}
			fmt.Fprintf(out, "  %s:%d:%d  %s%s\n", o.Rel, o.Line, o.Col, context(text, o.Start, o.End), note)
		}
	}
	if len(names) > 0 {
		fmt.Fprintln(out, "\nnames — a move, whose new name you choose:")
		for _, o := range names {
			id := strings.TrimSuffix(strings.TrimSuffix(o.Rel, "/"), ".md")
			fmt.Fprintf(out, "  %s  %q → %q   e.g. fdf mv %s %s\n", o.Rel, o.Banned, o.Term, id, suggestName(id, o))
		}
	}
	fmt.Fprintln(out, "\nTriage before fixing: a word used in another sense is qualified and listed under the term's `except:`;")
	fmt.Fprintln(out, "a mention of the word goes in a code span. Then `fdf lexicon --fix --dry-run`, review, and `--fix`.")
	return 0
}

// context is the line around an occurrence, trimmed to a readable width.
func context(text string, s, e int) string {
	ls := strings.LastIndex(text[:s], "\n") + 1
	le := strings.Index(text[e:], "\n")
	if le < 0 {
		le = len(text)
	} else {
		le += e
	}
	from, to := ls, le
	const width = 36
	if s-from > width {
		from = s - width
		for from < s && !utf8.RuneStart(text[from]) {
			from++
		}
	}
	if to-e > width {
		to = e + width
		for to > e && to < len(text) && !utf8.RuneStart(text[to]) {
			to--
		}
	}
	pre, post := "", ""
	if from > ls {
		pre = "…"
	}
	if to < le {
		post = "…"
	}
	return pre + strings.TrimSpace(text[from:s]+"«"+text[s:e]+"»"+text[e:to]) + post
}

// suggestName replaces the banned word in a name with its term's slug form.
func suggestName(id string, o bundle.Occurrence) string {
	dir, base := "", id
	if i := strings.LastIndex(id, "/"); i >= 0 {
		dir, base = id[:i+1], id[i+1:]
	}
	num := ""
	if len(base) > 3 && base[2] == '-' && base[0] >= '0' && base[0] <= '9' {
		num, base = base[:3], base[3:]
	}
	repl := strings.ToLower(strings.Join(strings.Fields(o.Term), "-"))
	if strings.ToLower(o.Text) != strings.ReplaceAll(o.Banned, " ", "-") {
		repl = pluralize(repl)
	}
	return dir + num + strings.Replace(base, o.Text, repl, 1)
}

// pluralize is a regular English plural, on the last word.
func pluralize(s string) string {
	low := strings.ToLower(s)
	switch {
	case strings.HasSuffix(low, "s") || strings.HasSuffix(low, "x") || strings.HasSuffix(low, "z") ||
		strings.HasSuffix(low, "ch") || strings.HasSuffix(low, "sh"):
		return s + "es"
	case len(low) > 1 && strings.HasSuffix(low, "y") && !strings.ContainsRune("aeiou", rune(low[len(low)-2])):
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

// replacement is the term written in place of one occurrence: plural when the
// word was, capitalized at a sentence start or in a heading when the term is
// lowercase, all capitals when the word was.
func replacement(o bundle.Occurrence) string {
	term := o.Term
	matched := strings.ToLower(strings.Join(strings.Fields(strings.ReplaceAll(o.Text, "-", " ")), " "))
	if matched != o.Banned {
		term = pluralize(term)
	}
	first, _ := utf8.DecodeRuneInString(o.Text)
	switch {
	case len(o.Text) > 1 && strings.ToUpper(o.Text) == o.Text && strings.ToLower(o.Text) != o.Text:
		return strings.ToUpper(term)
	case unicode.IsUpper(first):
		r, n := utf8.DecodeRuneInString(term)
		return string(unicode.ToUpper(r)) + term[n:]
	}
	return term
}

var articleRe = regexp.MustCompile(`(?i)(?:^|[^\w])(an?)\s+$`)

// withArticle extends a replacement over the "a" or "an" before it, mended for
// the new word.
func withArticle(text string, s int, repl string) (int, string) {
	lead := text[max(0, s-4):s]
	m := articleRe.FindStringSubmatchIndex(lead)
	if m == nil {
		return s, repl
	}
	art := lead[m[2]:m[3]]
	vowel := strings.ContainsRune("aeiouAEIOU", rune(repl[0]))
	want := "a"
	if vowel {
		want = "an"
	}
	if strings.ToLower(art) == want {
		return s, repl
	}
	if unicode.IsUpper(rune(art[0])) {
		want = strings.ToUpper(want[:1]) + want[1:]
	}
	start := s - len(lead) + m[2]
	return start, want + text[s-len(lead)+m[3]:s] + repl
}

func fixLexicon(rootAbs string, lex *bundle.Lexicon, occ []bundle.Occurrence, textOf func(string) string, dryRun bool, out io.Writer) int {
	// Scenario names first: a name is a join, so it changes everywhere it
	// appears — the Gherkin, slug.test.md (which F12 never reads), task
	// acceptance, declarations — or F8 and F10 break.
	renames := map[string]string{}
	var skipped []string
	for _, o := range occ {
		if o.InName() {
			continue
		}
		text := textOf(o.Rel)
		if why := manual(o, text); why != "" {
			skipped = append(skipped, fmt.Sprintf("%s:%d:%d  %s  [%s]", o.Rel, o.Line, o.Col, context(text, o.Start, o.End), why))
			continue
		}
		ls := strings.LastIndex(text[:o.Start], "\n") + 1
		le := strings.Index(text[o.Start:], "\n")
		if le < 0 {
			le = len(text)
		} else {
			le += o.Start
		}
		if m := scenarioLineRe.FindStringSubmatchIndex(text[ls:le]); m != nil && inGherkin(text, ls) {
			name := text[ls+m[2] : ls+m[3]]
			if _, seen := renames[name]; !seen {
				renames[name] = fixString(lex, name)
			}
		}
	}

	byFile := map[string][]span{}
	for _, o := range occ {
		if o.InName() {
			continue
		}
		text := textOf(o.Rel)
		if manual(o, text) != "" {
			continue
		}
		start, repl := withArticle(text, o.Start, replacement(o))
		byFile[o.Rel] = append(byFile[o.Rel], span{start, o.End, repl})
	}
	// Join renames reach every document but the vendored spec and the lexicon.
	if len(renames) > 0 {
		filepath.WalkDir(rootAbs, func(q string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(q, ".md") {
				return nil
			}
			rel := relSlash(rootAbs, q)
			if rel == "SPEC.md" || rel == "DOMAIN.md" {
				return nil
			}
			text := textOf(rel)
			for old, nw := range renames {
				if old != nw {
					byFile[rel] = append(byFile[rel], joinSpans(text, old, nw)...)
				}
			}
			return nil
		})
	}

	files := make([]string, 0, len(byFile))
	for rel := range byFile {
		files = append(files, rel)
	}
	sort.Strings(files)
	total := 0
	for _, rel := range files {
		text := textOf(rel)
		spans := dropContained(byFile[rel])
		fixed := applySpans(text, spans)
		if fixed == text {
			continue
		}
		total += len(spans)
		if dryRun {
			printDiff(rel, text, fixed, out)
			continue
		}
		if err := os.WriteFile(filepath.Join(rootAbs, filepath.FromSlash(rel)), []byte(fixed), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		fmt.Fprintf(out, "fixed %s (%d)\n", rel, len(spans))
	}
	var renamed []string
	for old, nw := range renames {
		if old != nw {
			renamed = append(renamed, fmt.Sprintf("%q → %q", old, nw))
		}
	}
	sort.Strings(renamed)
	for _, r := range renamed {
		fmt.Fprintln(out, "scenario renamed across its joins: "+r)
	}
	if len(skipped) > 0 {
		fmt.Fprintf(out, "\nleft for a person (%d):\n", len(skipped))
		for _, s := range skipped {
			fmt.Fprintln(out, "  "+s)
		}
	}
	names := 0
	for _, o := range occ {
		if o.InName() {
			names++
		}
	}
	if names > 0 {
		fmt.Fprintf(out, "\n%d name(s) use a banned word — `fdf lexicon` suggests an `fdf mv` for each.\n", names)
	}
	if dryRun {
		fmt.Fprintf(out, "\ndry run: %d replacement(s) in %d document(s); nothing was changed.\n", total, len(files))
		return 0
	}
	if total == 0 {
		fmt.Fprintln(out, "nothing to fix.")
		return 0
	}
	if err := logLexiconFix(rootAbs, occ, total, len(renamed)); err != nil {
		fmt.Fprintln(out, "error: writing LOG.md:", err)
		return 1
	}
	fmt.Fprintf(out, "\ndone: %d replacement(s); logged in LOG.md.\n", total)
	return 0
}

// joinSpans finds a scenario name where it stands whole — after `Scenario:`, a
// heading, a declaration verb, a list dash or an opening quote, and ending at
// the end of the line, a closing quote, or the em dash before a verification.
// A name that is the start of a longer one ("Owner sets hours" in "Owner sets
// hours twice") is left alone: that is a different scenario.
func joinSpans(text, old, nw string) []span {
	var out []span
	for i := 0; ; {
		j := strings.Index(text[i:], old)
		if j < 0 {
			break
		}
		s, e := i+j, i+j+len(old)
		i = e
		before := strings.TrimRight(text[:s], " \t")
		okBefore := before == "" || strings.HasSuffix(before, "\n")
		for _, p := range []string{":", "#", "-", "*", `"`, "“", ">", "("} {
			okBefore = okBefore || strings.HasSuffix(before, p)
		}
		after := strings.TrimLeft(text[e:], " \t")
		okAfter := after == "" || after[0] == '\n' || after[0] == '\r'
		for _, p := range []string{`"`, "”", "—", "–", "--", "*", ")"} {
			okAfter = okAfter || strings.HasPrefix(after, p)
		}
		if len(after) > 0 && (after[0] == '.' || after[0] == ',') && (len(after) == 1 || after[1] == ' ' || after[1] == '\n') {
			okAfter = true
		}
		if okBefore && okAfter {
			out = append(out, span{s, e, nw})
		}
	}
	return out
}

// fixString applies the lexicon fix to a string on its own — a scenario name.
func fixString(lex *bundle.Lexicon, s string) string {
	var spans []span
	for _, o := range lex.ScanDocument("", s) {
		start, repl := withArticle(s, o.Start, replacement(o))
		spans = append(spans, span{start, o.End, repl})
	}
	return applySpans(s, spans)
}

// dropContained removes spans inside another span: a scenario-name rename
// already covers the word replacements within the name.
func dropContained(spans []span) []span {
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start < spans[j].start
		}
		return spans[i].end > spans[j].end
	})
	var out []span
	for _, s := range spans {
		if len(out) > 0 && s.start < out[len(out)-1].end {
			continue
		}
		out = append(out, s)
	}
	return out
}

func printDiff(rel, before, after string, out io.Writer) {
	a, b := strings.Split(before, "\n"), strings.Split(after, "\n")
	fmt.Fprintf(out, "--- %s\n", rel)
	if len(a) != len(b) {
		fmt.Fprintln(out, "  (line count changed)")
		return
	}
	for i := range a {
		if a[i] != b[i] {
			fmt.Fprintf(out, "%5d - %s\n      + %s\n", i+1, a[i], b[i])
		}
	}
}

func logLexiconFix(rootAbs string, occ []bundle.Occurrence, total, renamed int) error {
	pairs := map[string]string{}
	for _, o := range occ {
		if !o.InName() {
			pairs[o.Banned] = o.Term
		}
	}
	var words []string
	for w, t := range pairs {
		words = append(words, fmt.Sprintf("`%s` → %s", w, t))
	}
	sort.Strings(words)
	line := fmt.Sprintf("* **Lexicon fix**: %s — %d replacement(s), %d scenario name(s) renamed across their joins (fdf lexicon --fix).\n",
		strings.Join(words, ", "), total, renamed)
	p := filepath.Join(rootAbs, "LOG.md")
	raw, err := os.ReadFile(p)
	body := string(raw)
	if err != nil {
		body = "# Bundle Update Log\n"
	}
	return os.WriteFile(p, []byte(logs.Insert(body, line)), 0o644)
}
