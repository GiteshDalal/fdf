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
	// inlineRe matches the destination of an inline link or image, ](target)
	// or ](<target>), which may hold spaces, then an optional title in "…",
	// '…' or (…); spaces may pad the inside of the parentheses. Find keeps a
	// match only when a [ opens its link text.
	inlineRe = regexp.MustCompile(`\]\([ \t]*(<[^<>\n]*>|[^)\s]+)(?:\s+(?:"[^"]*"|'[^']*'|\([^()]*\)))?[ \t]*\)`)
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
// returned with InCode set.
func Find(text string) []Link {
	code := Code(text)
	var out []Link
	for _, m := range inlineRe.FindAllStringSubmatchIndex(text, -1) {
		if opensLink(text, m[0]) {
			out = append(out, Link{Start: m[2], End: m[3], Target: text[m[2]:m[3]], InCode: inRanges(code, m[2])})
		}
	}
	for _, m := range defRe.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, Link{Start: m[2], End: m[3], Target: text[m[2]:m[3]], Def: true, InCode: inRanges(code, m[2])})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

// opensLink reports whether the ] at text[end] closes a link's text: a [
// before it, in the same paragraph, balances it. Brackets nest, so both
// links in [![badge](b.svg)](page.md) are found, while the ] in "a] (b)" is
// not a link at all.
func opensLink(text string, end int) bool {
	depth, blank := 0, false
	for i := end; i >= 0; i-- {
		c := text[i]
		switch c {
		case ']':
			depth++
		case '[':
			if depth--; depth == 0 {
				return true
			}
		case '\n':
			if blank {
				return false // a blank line: the paragraph starts after it
			}
			blank = true
			continue
		}
		if c != ' ' && c != '\t' && c != '\r' {
			blank = false
		}
	}
	return false
}

// Span is a run of a text's bytes, text[Start:End].
type Span struct{ Start, End int }

// Code returns the byte ranges of text that Find reads as code, in order:
// each fenced block, from its opening fence through its closing one, or to
// the end of the text when nothing closes it; each line of an indented code
// block; and each code span, its backticks included. A link whose target
// starts in one of them is returned with InCode set.
//
// A fence closes on a line of the same character at least as long as the
// one that opened it. An indented code block is a line indented four columns
// or more (a tab reaches the next multiple of four) after a blank line,
// outside a list, and the indented lines that follow it. The text's first
// line continues whatever came before it, so a line read on its own, such as
// one listing line, is never code for its indentation alone. Code spans are
// marked in each block of prose.
func Code(text string) []Span {
	var out []Span
	fence, fenceStart := "", 0
	prose := -1 // where the current block of prose starts
	endProse := func(at int) {
		if prose >= 0 {
			out = append(out, codeSpans(text, prose, at)...)
			prose = -1
		}
	}
	inList, blank, indented := false, false, false
	pos := 0
	for _, line := range strings.SplitAfter(text, "\n") {
		start := pos
		pos += len(line)
		t := strings.TrimSpace(line)
		switch {
		case fence != "":
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				out = append(out, Span{fenceStart, pos})
				fence = ""
			}
			continue
		case fenceRe.MatchString(t):
			endProse(start)
			fence, fenceStart = fenceRe.FindString(t), start
			indented = false
		case t == "":
			endProse(start)
			blank = true
			continue
		case columns(line) >= 4 && (indented || blank && !inList):
			endProse(start)
			out = append(out, Span{start, pos})
			indented = true
		default:
			indented = false
			item := listItemRe.MatchString(line)
			switch {
			case item:
				inList = true
			case columns(line) == 0 && (blank || headingRe.MatchString(line)):
				inList = false
			}
			if item || headingRe.MatchString(line) {
				endProse(start) // a list item or a heading starts a block of its own
			}
			if prose < 0 {
				prose = start
			}
		}
		blank = false
	}
	endProse(len(text))
	if fence != "" {
		out = append(out, Span{fenceStart, len(text)})
	}
	return out
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

func inRanges(rs []Span, at int) bool {
	for _, r := range rs {
		if at >= r.Start && at < r.End {
			return true
		}
	}
	return false
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
