// Package links finds the links in a Markdown text and repairs them after a
// move. It is the one link-repair engine: fdf mv uses it for a move inside the
// bundle, and fdf migrate for the 1.0 reorganisation, where the bundle itself
// moves too.
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
	InCode     bool   // inside a fenced block or a code span: a sample, not a reference
}

var (
	// inlineRe matches the target of an inline link or image,
	// [text](target) or [text](target "title").
	inlineRe = regexp.MustCompile(`\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	// defRe matches a reference definition, [label]: target. A label that
	// starts with ^ is a footnote, whose text is not a link.
	defRe      = regexp.MustCompile(`(?m)^[ \t]{0,3}\[[^\]\n^][^\]\n]*\]:[ \t]*(\S+)`)
	fenceRe    = regexp.MustCompile("^(`{3,}|~{3,})")
	codeSpanRe = regexp.MustCompile("`+[^`\n]*?`+")
)

// Find returns every link target in text, in order: inline links and images,
// and reference definitions. A target inside a fenced block or a code span is
// returned with InCode set.
func Find(text string) []Link {
	code := codeRanges(text)
	var out []Link
	for _, m := range inlineRe.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, Link{Start: m[2], End: m[3], Target: text[m[2]:m[3]], InCode: inRanges(code, m[2])})
	}
	for _, m := range defRe.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, Link{Start: m[2], End: m[3], Target: text[m[2]:m[3]], Def: true, InCode: inRanges(code, m[2])})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

type span struct{ start, end int }

// codeRanges marks fenced blocks and code spans. A fence closes on a line of
// the same character at least as long as the one that opened it.
func codeRanges(text string) []span {
	var out []span
	fence, fenceStart, pos := "", 0, 0
	for _, line := range strings.SplitAfter(text, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case fence != "":
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				out = append(out, span{fenceStart, pos + len(line)})
				fence = ""
			}
		case fenceRe.MatchString(t):
			fence, fenceStart = fenceRe.FindString(t), pos
		default:
			for _, loc := range codeSpanRe.FindAllStringIndex(line, -1) {
				out = append(out, span{pos + loc[0], pos + loc[1]})
			}
		}
		pos += len(line)
	}
	if fence != "" {
		out = append(out, span{fenceStart, len(text)})
	}
	return out
}

func inRanges(rs []span, at int) bool {
	for _, r := range rs {
		if at >= r.start && at < r.end {
			return true
		}
	}
	return false
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
// the way the link was written, relative or from the base, and any fragment,
// query and trailing slash. ok is false when the link needs no change:
// nothing it depends on moved, or it is not a path at all (a URL, an anchor,
// a mail address).
func Retarget(target string, s Site, m Move) (string, bool) {
	if target == "" || strings.HasPrefix(target, "#") || strings.Contains(target, "://") ||
		strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "tel:") {
		return "", false
	}
	pathPart, suffix := target, ""
	if i := strings.IndexAny(target, "#?"); i >= 0 {
		pathPart, suffix = target[:i], target[i:]
	}
	if pathPart == "" {
		return "", false
	}
	fromBase := strings.HasPrefix(pathPart, "/")
	var resolved string
	if fromBase {
		resolved = path.Join(s.OldBase, strings.TrimPrefix(pathPart, "/"))
	} else {
		resolved = path.Join(path.Dir(s.OldPath), pathPart)
	}
	moved, targetMoved := m.New(resolved)

	var nt string
	if fromBase {
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
	if strings.HasSuffix(pathPart, "/") && !strings.HasSuffix(nt, "/") {
		nt += "/"
	}
	nt += suffix
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
