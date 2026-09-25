package migrate

// The rest of the project names the bundle too: a Markdown link into it, and
// a mention of its path in a README, a config file or a code comment.
// migrate rewrites both in the project's git-tracked text files, and the
// mentions of the bundle's path inside the bundle, logs aside. A mention it
// cannot be sure of is listed and left as it is. What fdf install manages is
// left to fdf install, which replaces it whole: an edited copy would read as
// a user's edit, and block the upgrade.

import (
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/install"
	"github.com/GiteshDalal/fdf/cli/internal/links"
	"github.com/GiteshDalal/fdf/cli/internal/refactor"
)

// mention is one mention of the bundle's path in a text: its bytes, the path
// as written, and the path it becomes, or why it is left as it is.
type mention struct {
	start, end int
	path, to   string
	why        string // "in a URL" or "after a longer path", for one left as it is
}

// elsewhere is why a link whose target spells the bundle's path is left as
// it is: it leads somewhere else, so the engine does not repair it.
const elsewhere = "in a link that leads elsewhere"

// left is a mention left as it is, where it is.
type left struct {
	file string // the file, from the project root, once migrated
	line int
	path string
	why  string
}

// dirPrefixRe matches the start of a path before a mention of the bundle's
// path that begins the path: nothing, or /, ./ and ../ steps.
var dirPrefixRe = regexp.MustCompile(`^(?:\.{0,2}/)*$`)

// isPathByte reports the characters a path is made of.
func isPathByte(c byte) bool {
	return c == '-' || c == '_' || c == '.' || c == '/' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// pathMentions finds each mention of the bundle's path in text that the move
// changes, outside the bytes skip holds: the path, followed by the rest of a
// path inside it, becomes where that is once migrated — the longest match
// first, as the move maps it. A mention counts where a path begins: at the
// start of a word, after a /, ./ or ../ that begins the path, or after a
// variable or a placeholder, as in $PWD/, ${ROOT}/ or <worktree>/. One inside
// a URL is left as it is, since a permalink keeps its path, and so is one
// after a longer path, such as repo/docs/features, which may name another
// bundle.
func (p *plan) pathMentions(text string, skip map[int]bool, mv links.Move) []mention {
	var out []mention
	for i := 0; ; {
		j := strings.Index(text[i:], p.old)
		if j < 0 {
			break
		}
		s := i + j
		i = s + 1
		e := s + len(p.old)
		k := e
		for k < len(text) && isPathByte(text[k]) {
			k++
		}
		for k > e && text[k-1] == '.' {
			k-- // a dot that ends a sentence
		}
		if skip[s] || k > e && text[e] != '/' {
			continue // a link the engine repairs, or a longer name: docs/features-old
		}
		written := text[s:k]
		to, moved := mv.New(written)
		if !moved || to == written {
			continue
		}
		m := mention{start: s, end: k, path: written, to: to}
		switch {
		case inURL(text, s):
			m.to, m.why = "", "in a URL"
		case !begins(text, s):
			m.to, m.why = "", "after a longer path"
		}
		out = append(out, m)
		i = k
	}
	return out
}

// inURL reports whether text[s] is inside a URL: the word it is in holds
// "://" before it.
func inURL(text string, s int) bool {
	t := s
	for t > 0 && !strings.ContainsRune(" \t\r\n\"'`<>()[]", rune(text[t-1])) {
		t--
	}
	return strings.Contains(text[t:s], "://")
}

// begins reports whether a mention at text[s] is where a path begins: at the
// start of a word, after /, ./ and ../ steps that begin the path, or after a
// variable ($PWD/) or a placeholder (${ROOT}/, <worktree>/).
func begins(text string, s int) bool {
	i := s
	for i > 0 && isPathByte(text[i-1]) {
		i--
	}
	prefix := text[i:s]
	if dirPrefixRe.MatchString(prefix) {
		return true
	}
	return i > 0 && text[i-1] == '$' && strings.Count(prefix, "/") == 1 && strings.HasSuffix(prefix, "/")
}

// pathTargets marks the bytes of every link target in text that is a path,
// which the link engine repairs: a mention never overlaps one. A URL is no
// path, and a mention in one is read, and left as it is; and a link in code
// is a sample, whose words are read as any code's.
func pathTargets(text string) map[int]bool {
	out := map[int]bool{}
	for _, l := range links.Find(text) {
		if _, ok := links.Resolve(l.Target, "x", ""); ok && !l.InCode {
			for i := l.Start; i < l.End; i++ {
				out[i] = true
			}
		}
	}
	return out
}

// leftLinks lists each link in text, from the file at site, that spells the
// bundle's path the move changes but leads somewhere else, so that the
// engine leaves it as it is: every such occurrence of the old path is
// listed, as a mention in a URL is. file is where the plan lists it. A URL
// is no such link: a mention in one is listed as in a URL. One in what fdf
// install manages (managed) is counted instead, as a mention there is.
func (p *plan) leftLinks(text, file string, site links.Site, mv links.Move, managed map[int]bool) {
	for _, l := range links.Find(text) {
		if _, ok := links.Resolve(l.Target, "x", ""); l.InCode || !ok {
			continue
		}
		if _, ok := links.Retarget(l.Target, site, mv); ok {
			continue
		}
		if len(p.pathMentions(l.Target, nil, mv)) == 0 {
			continue
		}
		if managed[l.Start] {
			p.managed++
			continue
		}
		p.left = append(p.left, left{file, lineOf(text, l.Start), l.Target, elsewhere})
	}
}

// lineOf is the line text[s] is on, counting from 1.
func lineOf(text string, s int) int { return strings.Count(text[:s], "\n") + 1 }

// installed reports whether the file at f, from the project root, is in a
// directory fdf install placed, which holds its marker: an installed skill.
func (p *plan) installed(f string) bool {
	for d := path.Dir(f); d != "." && d != "/"; d = path.Dir(d) {
		if _, err := os.Stat(filepath.Join(p.project, filepath.FromSlash(d), install.MarkerFile)); err == nil {
			return true
		}
	}
	return false
}

// outside plans the rewrite of every reference to the bundle from the rest
// of the project: in each file git tracks outside the bundle that holds
// text, each Markdown link into the bundle, and each mention of its path.
// Bare feature IDs are left alone: outside the bundle, ekpie/cancel-order
// cannot be told from a code path. What fdf install manages is skipped and
// counted: the primer section of an instruction file, and an installed
// skill. So is .gitmodules, which git mv keeps: it names a submodule, and a
// submodule's name is git's, not its path.
func (p *plan) outside() error {
	out, err := git(p.project, "ls-files", "-z")
	if err != nil {
		return err
	}
	mv := p.move()
	for _, f := range strings.Split(out, "\x00") {
		if f == "" || f == p.old || strings.HasPrefix(f, p.old+"/") || f == ".gitmodules" {
			continue
		}
		full := filepath.Join(p.project, filepath.FromSlash(f))
		if fi, err := os.Lstat(full); err != nil || !fi.Mode().IsRegular() {
			continue
		}
		text, err := readText(full)
		if err != nil {
			return err
		}
		if text == "" {
			continue // binary, or empty
		}
		managed := map[int]bool{}
		if p.installed(f) {
			for i := range text {
				managed[i] = true
			}
		} else if install.InstructionFile(path.Base(f)) {
			if s, e, ok := install.PrimerSection(text); ok {
				for i := s; i < e; i++ {
					managed[i] = true
				}
			}
		}
		var reps []refactor.Replacement
		nLinks, nMentions := 0, 0
		skip := map[int]bool{}
		if strings.HasSuffix(f, ".md") || strings.HasSuffix(f, ".markdown") {
			skip = pathTargets(text)
			for _, l := range links.Find(text) {
				if l.InCode {
					continue
				}
				if nt, ok := links.Retarget(l.Target, links.Site{OldPath: f, NewPath: f}, mv); ok {
					if managed[l.Start] {
						p.managed++
						continue
					}
					reps = append(reps, refactor.Replacement{Start: l.Start, End: l.End, Text: nt})
					nLinks++
				}
			}
			p.leftLinks(text, f, links.Site{OldPath: f, NewPath: f}, mv, managed)
		}
		for _, m := range p.pathMentions(text, skip, mv) {
			switch {
			case managed[m.start]:
				p.managed++
			case m.why != "":
				p.left = append(p.left, left{f, lineOf(text, m.start), m.path, m.why})
			default:
				reps = append(reps, refactor.Replacement{Start: m.start, End: m.end, Text: m.to})
				nMentions++
			}
		}
		if len(reps) == 0 {
			continue
		}
		p.outTexts[f] = replace(text, reps)
		if nLinks > 0 {
			p.outLinks += nLinks
			p.outLinkFiles++
			p.outWhy[f] = append(p.outWhy[f], count(nLinks, "link"))
		}
		if nMentions > 0 {
			p.outMentions += nMentions
			p.outMentionFiles++
			p.outWhy[f] = append(p.outWhy[f], count(nMentions, "mention"))
		}
	}
	return nil
}

// readText reads the file at name, or "" when it is binary: a NUL byte in its
// first 8,000 bytes, which it reads before the rest.
func readText(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	head := make([]byte, 8000)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", err
	}
	if bytes.IndexByte(head[:n], 0) >= 0 {
		return "", nil
	}
	rest, err := io.ReadAll(f)
	return string(head[:n]) + string(rest), err
}

// writeOutside writes the rewritten files outside the bundle.
func (p *plan) writeOutside() error {
	for _, f := range sortedKeys(p.outTexts) {
		if err := os.WriteFile(filepath.Join(p.project, filepath.FromSlash(f)), []byte(p.outTexts[f]), 0o644); err != nil {
			return err
		}
	}
	return nil
}
