// Package links finds the links in a Markdown text and repairs them after a
// move. It is the one link-repair engine: fdf mv uses it for a move inside the
// bundle, and fdf migrate for the 1.0 reorganisation, where the bundle itself
// moves too. The validator reads links through it, so the links it checks are
// the links the engine repairs.
//
// The rule: a relative link is recomputed whenever the file that holds it
// moves or the file it names moves, wherever that file is, inside the bundle
// or outside it. A target that does not move keeps its place; the new link is
// its path from the linking file's new directory.
package links

import (
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Link is one link target in a Markdown text.
type Link struct {
	Start, End int    // the target's byte offsets in the text
	Target     string // the target as written: path, then any fragment or query
	Def        bool   // a reference definition ([label]: target), not an inline link
	InCode     bool   // inside a code block or a code span: a sample, not a reference
}

var (
	// defRe matches a reference definition, [label]: target, at any
	// indentation: Code decides whether an indented one is code. A label
	// that starts with ^ is a footnote, whose text is not a link.
	defRe      = regexp.MustCompile(`(?m)^[ \t]*\[[^\]\n^][^\]\n]*\]:[ \t]*(<[^<>\n]*>|\S+)`)
	fenceRe    = regexp.MustCompile("^(`{3,}|~{3,})")
	listItemRe = regexp.MustCompile(`^ {0,3}(?:[-*+]|\d{1,9}[.)])(?:[ \t]|$)`)
	headingRe  = regexp.MustCompile(`^ {0,3}#{1,6}(?:[ \t]|$)`)
)

// Find returns every link target in text, in order: inline links and images,
// and reference definitions. A target inside a code block or a code span is
// returned with InCode set. A reference definition stands at the start of a
// block, as CommonMark reads one: a line that continues a paragraph is prose.
func Find(text string) []Link {
	code, defAt, _ := blocks(text)
	closes := closers(text, code)
	var out []Link
	for i := 0; ; {
		j := strings.Index(text[i:], "](")
		if j < 0 {
			break
		}
		at := i + j
		start, end, next, ok := inline(text, at+2)
		if !ok {
			i = at + 1
			continue
		}
		i = next
		if closes[at] {
			out = append(out, Link{Start: start, End: end, Target: text[start:end], InCode: inRanges(code, start)})
		}
	}
	for _, m := range defRe.FindAllStringSubmatchIndex(text, -1) {
		if inCode := inRanges(code, m[2]); inCode || defAt[m[0]] {
			out = append(out, Link{Start: m[2], End: m[3], Target: text[m[2]:m[3]], Def: true, InCode: inCode})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

// inline reads what follows an inline link's "](", from text[i]: spaces, the
// destination, an optional title, and the parenthesis that closes it. The
// destination is <…>, which may hold spaces, or a run without spaces in
// which parentheses balance and a backslash escapes the next character, as
// in five(1).md. The title is "…", '…' or (…), after white space. It returns
// the destination's offsets and where the link ends, or ok false when this
// is no inline link.
func inline(text string, i int) (start, end, next int, ok bool) {
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	start = i
	if i < len(text) && text[i] == '<' {
		j := i + 1
		for j < len(text) && text[j] != '>' && text[j] != '<' && text[j] != '\n' {
			j++
		}
		if j >= len(text) || text[j] != '>' {
			return 0, 0, 0, false
		}
		end = j + 1
	} else {
		depth, j := 0, i
	scan:
		for j < len(text) {
			switch c := text[j]; {
			case c == '\\' && j+1 < len(text) && text[j+1] > ' ':
				j++ // the escaped character is part of the destination
			case c == '(':
				depth++
			case c == ')':
				if depth == 0 {
					break scan
				}
				depth--
			case c <= ' ':
				break scan
			}
			j++
		}
		if j == i || depth != 0 {
			return 0, 0, 0, false
		}
		end = j
	}
	k := end
	if t := title(text, end); t > end {
		k = t
	}
	for k < len(text) && (text[k] == ' ' || text[k] == '\t') {
		k++
	}
	if k >= len(text) || text[k] != ')' {
		return 0, 0, 0, false
	}
	return start, end, k + 1, true
}

// title returns where a link title after text[i] ends: white space, then
// "…", '…' or (…). It returns i when there is none.
func title(text string, i int) int {
	k := i
	for k < len(text) && strings.IndexByte(" \t\n\r\f", text[k]) >= 0 {
		k++
	}
	if k == i || k >= len(text) {
		return i
	}
	closer := map[byte]byte{'"': '"', '\'': '\'', '(': ')'}[text[k]]
	if closer == 0 {
		return i
	}
	for j := k + 1; j < len(text); j++ {
		switch {
		case text[j] == closer:
			return j + 1
		case closer == ')' && text[j] == '(':
			return i
		}
	}
	return i
}

// escaped reports whether a backslash escapes text[i]: an odd number of them
// stand before it.
func escaped(text string, i int) bool {
	n := 0
	for j := i - 1; j >= 0 && text[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}

// closers finds the ] of each "](" in text that closes a link's text: a [
// before it, in the same paragraph, balances it. Brackets nest, so both
// links in [![badge](b.svg)](page.md) are found, while the ] in "a] (b)" is
// not a link at all. A bracket a backslash escapes is text, and so is one in
// a code span when the ] is not in it. It reads the text once, keeping two
// counts of the [ still open: every one, for a ] in code, and those outside
// code, for a ] outside it. A blank line starts a paragraph, with none open.
func closers(text string, code []Span) map[int]bool {
	out := map[int]bool{}
	all, prose := 0, 0
	blank, c := true, 0 // blank: the line so far is white space; c: the first range of code not behind i
	for i := 0; i < len(text); i++ {
		ch := text[i]
		switch ch {
		case '\n':
			if blank {
				all, prose = 0, 0
			}
			blank = true
			continue
		case ' ', '\t', '\r':
			continue
		}
		blank = false
		if ch != '[' && ch != ']' || escaped(text, i) {
			continue
		}
		for c < len(code) && code[c].End <= i {
			c++
		}
		inCode := c < len(code) && code[c].Start <= i
		if ch == '[' {
			all++
			if !inCode {
				prose++
			}
			continue
		}
		open := prose > 0
		if inCode {
			open = all > 0
		}
		if open && strings.HasPrefix(text[i+1:], "(") {
			out[i] = true
		}
		all = max(all-1, 0)
		if !inCode {
			prose = max(prose-1, 0)
		}
	}
	return out
}

// Span is a run of a text's bytes, text[Start:End].
type Span struct{ Start, End int }

// Code returns the byte ranges of text that Find reads as code, in order:
// each fenced block, from its opening fence through its closing one, or to
// the end of the text when nothing closes it; each line of an indented code
// block; and each code span, its backticks included. A link whose target
// starts in one of them is returned with InCode set.
//
// A fence opens on a line indented less than four columns, or on any line of
// a list, and closes on a line of the same character at least as long as
// the one that opened it. An indented code block is a line indented four
// columns or more (a tab reaches the next multiple of four) that starts a
// block outside a list, after a blank line, a heading or a closing fence,
// and the indented lines that follow it. The text's first line continues
// whatever came before it, so a line read on its own, such as one listing
// line, is never code for its indentation alone. Code spans are marked in
// each block of prose, and a heading is a block of its own.
func Code(text string) []Span {
	code, _, _ := blocks(text)
	return code
}

// Blocks returns the byte ranges of text's code blocks, in order: the ranges
// Code returns but its code spans, each fenced block and each line of an
// indented one.
func Blocks(text string) []Span {
	_, _, block := blocks(text)
	return block
}

// blocks reads text as CommonMark groups its lines into blocks, as far as
// Find needs to: the ranges Code returns, those of them that are code
// blocks (block), and the lines a reference definition may start (defAt, by
// offset): every line that does not continue a paragraph.
func blocks(text string) (code []Span, defAt map[int]bool, block []Span) {
	defAt = map[int]bool{}
	fence, fenceStart := "", 0
	prose := -1 // where the current block of prose starts
	endProse := func(at int) {
		if prose >= 0 {
			code = append(code, codeSpans(text, prose, at)...)
			prose = -1
		}
	}
	// boundary: the line before ended a block (a blank line, a heading, a
	// closing fence), so an indented line starts a code block. para: the
	// line before is prose, which a definition cannot interrupt.
	inList, boundary, para, indented := false, false, false, false
	pos := 0
	for _, line := range strings.SplitAfter(text, "\n") {
		start := pos
		pos += len(line)
		t := strings.TrimSpace(line)
		switch {
		case fence != "":
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				code = append(code, Span{fenceStart, pos})
				block = append(block, Span{fenceStart, pos})
				fence = ""
				boundary = true
			}
			continue
		case t == "":
			endProse(start)
			boundary, para = true, false
			continue
		case columns(line) >= 4 && (indented || boundary && !inList):
			endProse(start)
			code = append(code, Span{start, pos})
			block = append(block, Span{start, pos})
			indented, boundary, para = true, false, false
			continue
		case fenceRe.MatchString(t) && (columns(line) < 4 || inList):
			endProse(start)
			fence, fenceStart = fenceRe.FindString(t), start
			indented, boundary, para = false, false, false
			continue
		}
		indented = false
		item, heading := listItemRe.MatchString(line), headingRe.MatchString(line)
		switch {
		case item:
			inList = true
		case columns(line) == 0 && (boundary || heading):
			inList = false
		}
		if item || heading {
			endProse(start) // a list item or a heading starts a block of its own
		}
		defAt[start] = !para
		if prose < 0 {
			prose = start
		}
		switch {
		case heading:
			endProse(pos) // and a heading ends it
			boundary, para = true, false
		case defAt[start] && defRe.MatchString(line):
			boundary, para = false, false // another definition may follow
		default:
			boundary, para = false, true
		}
	}
	endProse(len(text))
	if fence != "" {
		code = append(code, Span{fenceStart, len(text)})
		block = append(block, Span{fenceStart, len(text)})
	}
	return code, defAt, block
}

// columns is how far a line is indented, a tab reaching the next multiple of
// four.
func columns(line string) int {
	n := 0
	for _, c := range line {
		switch c {
		case ' ':
			n++
		case '\t':
			n += 4 - n%4
		default:
			return n
		}
	}
	return n
}

// codeSpans marks the code spans in text[start:end], one block of prose. A run
// of backticks opens a span, and the next run of the same length closes it,
// on the same line or a later one; a run that nothing closes is literal.
func codeSpans(text string, start, end int) []Span {
	var out []Span
	run := func(i int) int { // the end of the backtick run at i
		for i < end && text[i] == '`' {
			i++
		}
		return i
	}
	for i := start; i < end; {
		if text[i] != '`' {
			i++
			continue
		}
		open := run(i)
		next := open
		for k := open; k < end; {
			if text[k] != '`' {
				k++
				continue
			}
			e := run(k)
			if e-k == open-i {
				out = append(out, Span{i, e})
				next = e
				break
			}
			k = e
		}
		i = next
	}
	return out
}

// inRanges reports whether at is in one of rs, which are in order.
func inRanges(rs []Span, at int) bool {
	i := sort.Search(len(rs), func(i int) bool { return rs[i].End > at })
	return i < len(rs) && rs[i].Start <= at
}

// Dest is a link target read as a path.
type Dest struct {
	Path     string // what it names, cleaned, in the paths of the file it was read from
	FromBase bool   // written from the base directory ("/…"), not relative to its file
	Dir      bool   // written with a trailing slash
	Suffix   string // the fragment or query after the path, as written
	Angle    bool   // written in angle brackets (<…>)
}

// Resolve reads target, as written in the file at from, as a path. base is
// the directory a target starting with "/" resolves against; from and base
// are slash-separated and relative to one directory, as a Move's paths are.
// ok is false when the target is not a path: it is empty, an in-page anchor,
// a URL, a mail or phone link, or an angle bracket that never closes.
func Resolve(target, from, base string) (Dest, bool) {
	var d Dest
	if strings.HasPrefix(target, "<") {
		if len(target) < 2 || !strings.HasSuffix(target, ">") {
			return Dest{}, false
		}
		d.Angle, target = true, target[1:len(target)-1]
	}
	if target == "" || strings.HasPrefix(target, "#") || strings.Contains(target, "://") ||
		strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "tel:") {
		return Dest{}, false
	}
	p := target
	if i := strings.IndexAny(target, "#?"); i >= 0 {
		p, d.Suffix = target[:i], target[i:]
	}
	if p == "" {
		return Dest{}, false
	}
	d.Dir = strings.HasSuffix(p, "/")
	d.FromBase = strings.HasPrefix(p, "/")
	if d.FromBase {
		d.Path = path.Join(base, strings.TrimPrefix(p, "/"))
	} else {
		d.Path = path.Join(path.Dir(from), p)
	}
	return d, true
}

// Move maps paths to where they are after a move. Paths are slash-separated
// and relative to one base directory, the project root or the bundle root,
// and may start with ../ for places above it.
type Move struct {
	Files map[string]string // a file's old path -> its new path
	Dirs  map[string]string // a directory's old path -> its new path; what is under it moves with it
}

// New returns where p is after the move, and whether it moved. A file's own
// entry wins; otherwise the longest directory entry that contains p.
func (m Move) New(p string) (string, bool) {
	if n, ok := m.Files[p]; ok {
		return n, true
	}
	best := ""
	for o := range m.Dirs {
		if (p == o || strings.HasPrefix(p, o+"/")) && len(o) > len(best) {
			best = o
		}
	}
	if best == "" {
		return p, false
	}
	return m.Dirs[best] + strings.TrimPrefix(p, best), true
}

// Site is where a linking file sits, before and after the move, in the
// Move's paths. Base is the directory a link starting with "/" resolves
// against: the bundle root for a file in the bundle, and the project root
// for one outside it.
type Site struct {
	OldPath, NewPath string
	OldBase, NewBase string
}

// Retarget returns target rewritten so that, written in the file at
// s.NewPath, it names what it named from s.OldPath before the move. It keeps
// the way the link was written, relative or from the base, in angle brackets
// or not, and any fragment, query and trailing slash. ok is false when the
// link needs no change: nothing it depends on moved, or it is not a path at
// all (a URL, an anchor, a mail address).
func Retarget(target string, s Site, m Move) (string, bool) {
	d, ok := Resolve(target, s.OldPath, s.OldBase)
	if !ok {
		return "", false
	}
	moved, targetMoved := m.New(d.Path)

	var nt string
	if d.FromBase {
		if !targetMoved && s.OldBase == s.NewBase {
			return "", false
		}
		r, ok := rel(s.NewBase, moved)
		switch {
		case !ok || r == ".." || strings.HasPrefix(r, "../"):
			// No longer under the base: only a relative link can reach it.
			if nt, ok = rel(path.Dir(s.NewPath), moved); !ok {
				return "", false
			}
		case r == ".":
			nt = "/"
		default:
			nt = "/" + r
		}
	} else {
		if !targetMoved && s.OldPath == s.NewPath {
			return "", false
		}
		var ok bool
		if nt, ok = rel(path.Dir(s.NewPath), moved); !ok {
			return "", false
		}
		// A link written ./x keeps its ./, so one that still names the
		// same place, as a sibling that moves with its file does, is left
		// as it is.
		if strings.HasPrefix(strings.TrimPrefix(target, "<"), "./") && nt != "." && !strings.HasPrefix(nt, "../") {
			nt = "./" + nt
		}
	}
	if d.Dir && !strings.HasSuffix(nt, "/") {
		nt += "/"
	}
	nt += d.Suffix
	if d.Angle {
		nt = "<" + nt + ">"
	}
	return nt, nt != target
}

// rel is filepath.Rel for the slash-separated paths of a Move.
func rel(from, to string) (string, bool) {
	r, err := filepath.Rel(filepath.FromSlash(from), filepath.FromSlash(to))
	if err != nil {
		return "", false
	}
	return filepath.ToSlash(r), true
}
