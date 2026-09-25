// Package release implements `fdf release` — the one FDF document whose
// content is wholly derived. A release's `# Features` and `# Changes` lists
// are a projection of the `version:` fields already carried by features,
// changes and fixes; they are written down anyway because a bundle must be
// readable with `cat` alone. Deriving them is what keeps the two ends of that
// relationship (F7) from drifting.
//
// What this never derives is membership. Which work ships in a release is a
// human decision, recorded as `version:` on each document; this command is
// downstream of those decisions, never upstream.
package release

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

var (
	versionRe = regexp.MustCompile(`(?m)^version:\s*"?([^"\s]+)"?`)
	typeRe    = regexp.MustCompile(`(?m)^type:\s*(\S+)`)
	statusRe  = regexp.MustCompile(`(?m)^status:\s*(\S+)`)
	titleRe   = regexp.MustCompile(`(?m)^title:\s*(.+)$`)
	descRe    = regexp.MustCompile(`(?m)^description:\s*(.+)$`)
	dateRe    = regexp.MustCompile(`(?m)^date:\s*(\S+)`)
	notesRe   = regexp.MustCompile(`(?ms)^# Notes\s*$.*`)
)

type member struct {
	id, title, docType string
	status             string
}

// Sync creates or refreshes releases/<version>.md from the bundle's version
// fields. ship flips the release to `shipped`, refusing while any listed
// document is still open.
func Sync(root, version, date string, ship bool, out io.Writer) int {
	if strings.TrimSpace(version) == "" {
		fmt.Fprintln(out, "error: a release needs a version, such as 1.2.0")
		return 2
	}
	// The version names the release's file, which layout must place.
	switch pos := layout.New(os.DirFS(root)).File("releases/" + version + ".md"); pos.Kind {
	case layout.Document:
	case layout.Stray:
		fmt.Fprintf(out, "error: %s cannot name a release — %s: %s (F3)\n", version, pos.Where, pos.Problem)
		return 2
	default:
		// INDEX and LOG name the register's own index and log.
		fmt.Fprintf(out, "error: %s cannot name a release — releases/%s.md is the register's %s, not a release (F3)\n", version, version, pos.Kind)
		return 2
	}
	if !scaffold.RequireSupported(root, out) {
		return 1
	}
	feats, chgs := scan(root, version)
	if len(feats)+len(chgs) == 0 {
		fmt.Fprintf(out, "error: no feature, Change or Fix carries `version: %q` — set it on the documents that ship in this release first\n", version)
		fmt.Fprintln(out, "  membership is a human decision; this command only derives the lists from it")
		return 1
	}

	path := filepath.Join(root, "releases", version+".md")
	existing, _ := os.ReadFile(path)
	prior := string(existing)
	status, title, desc, when := "planned", version, "Release "+version+".", date
	if m := statusRe.FindStringSubmatch(prior); m != nil {
		status = m[1]
	}
	if m := titleRe.FindStringSubmatch(prior); m != nil {
		title = strings.TrimSpace(m[1])
	}
	if m := descRe.FindStringSubmatch(prior); m != nil {
		desc = strings.TrimSpace(m[1])
	}
	if when == "" {
		if m := dateRe.FindStringSubmatch(prior); m != nil {
			when = m[1]
		} else {
			when = time.Now().UTC().Format("2006-01-02")
		}
	}

	if ship {
		var open []string
		for _, m := range append(append([]member{}, feats...), chgs...) {
			if m.status != "done" {
				open = append(open, fmt.Sprintf("%s (%s)", m.id, m.status))
			}
		}
		if len(open) > 0 {
			sort.Strings(open)
			fmt.Fprintf(out, "error: cannot ship %s — %d listed document(s) are not done:\n", version, len(open))
			for _, o := range open {
				fmt.Fprintln(out, "  "+o)
			}
			return 1
		}
		status = "shipped"
		if date == "" {
			when = time.Now().UTC().Format("2006-01-02")
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "---\ntype: Release\ntitle: %s\ndescription: %s\nstatus: %s\ndate: %s\ntimestamp: %s\n---\n\n",
		title, desc, status, when, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	b.WriteString("# Features\n\n")
	if len(feats) == 0 {
		b.WriteString("* (none)\n")
	}
	for _, m := range feats {
		fmt.Fprintf(&b, "* [%s](/%s.md) - %s.\n", m.title, m.id, m.status)
	}
	if len(chgs) > 0 {
		b.WriteString("\n# Changes\n\n")
		for _, m := range chgs {
			fmt.Fprintf(&b, "* [%s](/%s.md) - %s (%s).\n", m.title, m.id, strings.ToLower(m.docType), m.status)
		}
	}
	// `# Notes` is human prose; a generator preserves it untouched.
	if n := notesRe.FindString(prior); n != "" {
		b.WriteString("\n" + strings.TrimRight(n, "\n") + "\n")
	} else {
		b.WriteString("\n# Notes\n\nTODO — optional.\n")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	verb := "refreshed"
	if prior == "" {
		verb = "created"
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "%s releases/%s.md (status: %s, date: %s)\n", verb, version, status, when)
	fmt.Fprintf(out, "  %d feature(s), %d Change(s) or Fix(es) derived from `version: %q`\n", len(feats), len(chgs), version)
	if code := ensureIndex(root, version, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "\ndone: release %s is %s\n", version, status)
	if status == "planned" {
		fmt.Fprintf(out, "next: set `version: %q` on anything else that ships, re-run `fdf release %s`, then `fdf release --ship %s` once everything is done.\n", version, version, version)
	}
	return 0
}

// scan collects every feature, Change and Fix whose `version` matches: the
// documents filed in features/ and changes/, as layout reads them.
func scan(root, version string) (feats, chgs []member) {
	b := layout.New(os.DirFS(root))
	for _, reg := range []string{"features", "changes"} {
		filepath.WalkDir(filepath.Join(root, reg), func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			slash := filepath.ToSlash(rel)
			if b.File(slash).Kind != layout.Document {
				return nil
			}
			raw, rerr := os.ReadFile(p)
			if rerr != nil {
				return nil
			}
			text := string(raw)
			tm := typeRe.FindStringSubmatch(text)
			vm := versionRe.FindStringSubmatch(text)
			if tm == nil || vm == nil || vm[1] != version {
				return nil
			}
			m := member{id: strings.TrimSuffix(slash, ".md"), docType: tm[1], title: strings.TrimSuffix(d.Name(), ".md")}
			if t := titleRe.FindStringSubmatch(text); t != nil {
				m.title = strings.TrimSpace(t[1])
			}
			if s := statusRe.FindStringSubmatch(text); s != nil {
				m.status = s[1]
			}
			switch tm[1] {
			case "Feature":
				feats = append(feats, m)
			case "Change", "Fix":
				chgs = append(chgs, m)
			}
			return nil
		})
	}
	sort.Slice(feats, func(i, j int) bool { return feats[i].id < feats[j].id })
	sort.Slice(chgs, func(i, j int) bool { return chgs[i].id < chgs[j].id })
	return feats, chgs
}

var listingRe = regexp.MustCompile(`^\s*[-*+]\s+.*\]\(`)

// ensureIndex lists the release in releases/INDEX.md, newest first: a new
// release goes above the first one listed. The first release creates the
// index, and lists the releases register in the root INDEX.md.
func ensureIndex(root, version string, out io.Writer) int {
	idx := filepath.Join(root, "releases", "INDEX.md")
	entry := fmt.Sprintf("* [%s](/releases/%s.md) - release.", version, version)
	raw, err := os.ReadFile(idx)
	if err == nil && strings.Contains(string(raw), "/releases/"+version+".md") {
		return 0
	}
	if err != nil {
		raw = []byte("# Releases\n\nNewest first.\n")
		if code := scaffold.ListRegister(root, "releases", out); code != 0 {
			return code
		}
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	at := -1
	for i, l := range lines {
		if listingRe.MatchString(l) {
			at = i
			break
		}
	}
	if at < 0 {
		lines = append(lines, "") // a list starts after a blank line
		at = len(lines)
	}
	lines = append(lines[:at], append([]string{entry}, lines[at:]...)...)
	if err := os.WriteFile(idx, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "updated releases/INDEX.md (now lists %q)\n", version)
	return 0
}
