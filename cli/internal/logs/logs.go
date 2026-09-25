// Package logs writes FDF log entries. Every FDF log — the bundle-root
// LOG.md, a group's LOG.md, and the <slug>.log.md beside a feature, change,
// fix, practice, debt or bug — keeps its entries newest first under
// `## YYYY-MM-DD` headings. An entry goes in the log of the one document it is
// about; the bundle-root LOG.md takes what concerns the bundle as a whole.
package logs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

var (
	dateHeadRe  = regexp.MustCompile(`(?m)^##[ \t]+\d{4}-\d{2}-\d{2}[ \t]*$`)
	typeRe      = regexp.MustCompile(`(?m)^type:\s*"?([A-Za-z]+)"?\s*$`)
	titleRe     = regexp.MustCompile(`(?m)^title:\s*(.+?)\s*$`)
	headingRe   = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	trailRe     = regexp.MustCompile(`^(.+)\.(spec|plan|test|surface|log)$`)
	taskRe      = regexp.MustCompile(`^(.+)/\d\d-[a-z0-9-]+$`)
	timestampRe = regexp.MustCompile(`(?m)^timestamp:[ \t]*(.*)$`)
	dateOnlyRe  = regexp.MustCompile(`^"?\d{4}-\d{2}-\d{2}"?$`)
)

// clock is the time source; tests pin it.
var clock = time.Now

// Today is the heading a new entry goes under: the UTC date, as every date
// and time an fdf command writes is UTC.
func Today() string { return clock().UTC().Format("2006-01-02") }

// Insert adds lines (each ending in a newline) under today's heading, the
// newest entry of a day first. Every FDF log is newest first, so a new heading
// goes above the first older date — below any later one, such as a date
// written by hand in a time zone ahead of UTC.
func Insert(body, lines string) string {
	today := Today()
	block := "## " + today + "\n" + lines + "\n"
	for _, loc := range dateHeadRe.FindAllStringIndex(body, -1) {
		date := strings.TrimSpace(strings.TrimPrefix(body[loc[0]:loc[1]], "##"))
		switch {
		case date == today:
			head, rest := body[:loc[1]]+"\n", strings.TrimPrefix(body[loc[1]:], "\n")
			return head + lines + rest
		case date < today:
			return body[:loc[0]] + block + body[loc[0]:]
		}
	}
	return strings.TrimRight(body, "\n") + "\n\n" + block
}

// Entry formats text as one log entry: a bullet, with any further lines
// indented so they stay part of it.
func Entry(text string) string {
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n")), "\n")
	if head := lines[0]; !strings.HasPrefix(head, "* ") && !strings.HasPrefix(head, "- ") {
		lines[0] = "* " + head
	}
	for i := 1; i < len(lines); i++ {
		if l := strings.TrimSpace(lines[i]); l == "" {
			lines[i] = ""
		} else if !strings.HasPrefix(lines[i], "  ") {
			lines[i] = "  " + l
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// A target is the log an entry for one ID goes in.
type target struct {
	rel    string // bundle-relative path of the log
	header string // what a new log starts with
	note   string // why the entry went somewhere other than the ID itself
}

// Resolve finds the log for id: "" is the bundle-root LOG.md; a register or
// a group is its LOG.md; a feature, change, fix, practice, debt or bug is its
// <id>.log.md; a task or trail document is logged with the document that owns
// it; a Context document, SPEC.md or a release is logged in the bundle-root
// LOG.md. Positions are layout's, so nothing with no place in a 1.0 bundle —
// a directory that is not a register or a group, or a document in one — is
// given a log.
func Resolve(root, id string) (target, error) {
	id = strings.Trim(filepath.ToSlash(strings.TrimSpace(id)), "/")
	id = strings.TrimSuffix(id, ".md")
	if id == "" || id == "LOG" {
		return rootLog(""), nil
	}
	if strings.HasSuffix(id, "/LOG") || strings.HasSuffix(id, "/INDEX") || id == "INDEX" {
		return Resolve(root, strings.TrimSuffix(strings.TrimSuffix(id, "LOG"), "INDEX"))
	}
	if m := trailRe.FindStringSubmatch(id); m != nil {
		t, err := Resolve(root, m[1])
		if err == nil && t.note == "" {
			t.note = fmt.Sprintf("%s is part of %s's trail, so the entry is in %s's log", id, m[1], m[1])
		}
		return t, err
	}
	b := layout.New(os.DirFS(root))
	if !b.Exists(id + ".md") {
		if st, serr := os.Stat(filepath.Join(root, id)); serr == nil && st.IsDir() && b.Exists(id) {
			switch pos := b.Dir(id); {
			case pos.Kind == layout.Stray:
				return target{}, fmt.Errorf("%s has no place in a 1.0 bundle — %s: %s (F3)", id, pos.Where, pos.Problem)
			case !b.HoldsMarkdown(id):
				return target{}, fmt.Errorf("%s/ holds no Markdown, so it is outside the bundle and has no log", id)
			}
			return groupLog(root, id), nil
		}
		return target{}, fmt.Errorf("no document or group %q in the bundle%s", id, scaffold.IDHint(root, id))
	}
	if pos := b.File(id + ".md"); pos.Kind == layout.Stray {
		return target{}, fmt.Errorf("%s has no place in a 1.0 bundle — %s: %s (F3)", id, pos.Where, pos.Problem)
	}
	raw, err := os.ReadFile(filepath.Join(root, id+".md"))
	if err != nil {
		return target{}, err
	}
	fm, _ := split(string(raw))
	docType := first(typeRe, fm)
	switch docType {
	case "Feature", "Change", "Fix", "Practice", "Debt", "Bug":
		// A draft feature may have a log: it is the one sibling a draft has.
		return siblingLog(id, docType, fm), nil
	case "Task":
		owner := id
		if m := taskRe.FindStringSubmatch(id); m != nil {
			owner = m[1]
		}
		t, err := Resolve(root, owner)
		if err == nil {
			t.note = fmt.Sprintf("a task has no log of its own, so the entry is in %s's", owner)
		}
		return t, err
	case "Context", "Reference", "Release":
		return rootLog(fmt.Sprintf("%s is a %s document, logged in the bundle-root LOG.md", id, docType)), nil
	case "Spec", "Plan", "Test", "Surface", "Log":
		return target{}, fmt.Errorf("%s is a %s with no owner beside it; name the feature, change or fix it belongs to", id, docType)
	case "":
		return target{}, fmt.Errorf("%s.md has no `type`, so there is no telling which log it belongs to", id)
	}
	return target{}, fmt.Errorf("%s is a %s document, which has no log", id, docType)
}

func rootLog(note string) target {
	return target{rel: "LOG.md", header: "# Bundle Update Log\n", note: note}
}

// groupLog is the LOG.md of a group directory, headed after its INDEX.md.
func groupLog(root, id string) target {
	name := filepath.Base(id)
	if raw, err := os.ReadFile(filepath.Join(root, id, "INDEX.md")); err == nil {
		_, body := split(string(raw))
		if h := first(headingRe, body); h != "" {
			name = h
		}
	}
	return target{rel: id + "/LOG.md", header: "# " + name + " — log\n"}
}

// siblingLog is the <id>.log.md beside a document. A new one carries the
// frontmatter every document does, titled after its owner.
func siblingLog(id, docType, ownerFM string) target {
	title := strings.Trim(first(titleRe, ownerFM), `"'`)
	if title == "" {
		title = filepath.Base(id)
	}
	header := fmt.Sprintf("---\ntype: Log\ntitle: %s — log\ndescription: What happened to this %s and why, newest first.\ntimestamp: %s\n---\n",
		yamlString(title), strings.ToLower(docType), now())
	return target{rel: id + ".log.md", header: header}
}

func now() string { return clock().UTC().Format("2006-01-02T15:04:05Z") }

// touch moves a log's `timestamp` to now, in the form it already has: the log
// changed when its newest entry was written.
func touch(body string) string {
	if !strings.HasPrefix(body, "---\n") {
		return body
	}
	end := strings.Index(body[len("---\n"):], "\n---")
	if end < 0 {
		return body
	}
	end += len("---\n")
	fm := body[:end]
	m := timestampRe.FindStringSubmatch(fm)
	if m == nil {
		return body
	}
	stamp := now()
	if dateOnlyRe.MatchString(m[1]) {
		stamp = Today()
	}
	return timestampRe.ReplaceAllLiteralString(fm, "timestamp: "+stamp) + body[end:]
}

// IsID reports whether s names a document or group in the bundle rather than
// reading as an entry, so `fdf log <id>` with the entry forgotten is caught
// instead of logging the ID itself.
func IsID(root, s string) bool {
	if strings.ContainsAny(strings.TrimSpace(s), " \t\n") {
		return false
	}
	t, err := Resolve(root, s)
	return err == nil && (t.rel != "LOG.md" || t.note != "")
}

// Append writes one entry to the log for id, creating the log on first use,
// and says where it went.
func Append(root, id, text string, out io.Writer) int {
	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(out, "error: the entry is empty")
		return 2
	}
	if err := fdfroot.CheckBundle(root); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if !scaffold.RequireSupported(root, out) {
		return 1
	}
	t, err := Resolve(root, id)
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	p := filepath.Join(root, filepath.FromSlash(t.rel))
	raw, rerr := os.ReadFile(p)
	body, created := string(raw), rerr != nil
	if created {
		body = t.header
	}
	if err := os.WriteFile(p, []byte(Insert(touch(body), Entry(text))), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if t.note != "" {
		fmt.Fprintf(out, "note: %s.\n", t.note)
	}
	if created {
		fmt.Fprintf(out, "logged in %s (new log)\n", t.rel)
	} else {
		fmt.Fprintf(out, "logged in %s\n", t.rel)
	}
	return 0
}

// split returns a document's frontmatter block (without its fences) and its
// body. A document with no frontmatter is all body.
func split(text string) (fm, body string) {
	text = strings.ReplaceAll(strings.TrimPrefix(text, "\xef\xbb\xbf"), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return "", text
	}
	rest := text[len("---\n"):]
	if strings.HasPrefix(rest, "---\n") {
		return "", rest[len("---\n"):]
	}
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if strings.HasSuffix(rest, "\n---") {
			return rest[:len(rest)-len("\n---")], ""
		}
		return "", text
	}
	return rest[:end], rest[end+len("\n---\n"):]
}

func first(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// yamlString quotes a title when plain YAML would misread it.
func yamlString(s string) string {
	if strings.ContainsAny(s, ":#{}[]&*!|>'\"%@`") {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return s
}
