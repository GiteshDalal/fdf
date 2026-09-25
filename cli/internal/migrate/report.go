package migrate

import (
	"fmt"
	"io"
	"path"
	"strings"
)

// print prints the plan: a summary of what migrate moves and repairs, then
// every file it moves, removes, edits or writes, with what changes in each.
// A dry run prints it and stops.
func (p *plan) print(out io.Writer, from string, dry bool) {
	head := fmt.Sprintf("plan: fdf_version %s → %s", from, target)
	if dry {
		head += " (dry run: nothing changed)"
	}
	fmt.Fprintln(out, head)
	row := func(label, text string) { fmt.Fprintf(out, "  %-10s %s\n", label, text) }
	if p.relocates() {
		row("bundle", fmt.Sprintf("%s/ → %s/", p.shown(p.old), p.shown(p.new)))
	}
	if s := p.layoutSummary(); s != "" {
		row("layout", s)
	}
	if len(p.groups) > 0 {
		row("features", fmt.Sprintf("%s → features/   (%s, %s)", p.groupList("", "/"), count(len(p.ids), "feature"), count(p.groupFiles(""), "file")))
	} else {
		row("features", "no feature groups")
	}
	row("ids", fmt.Sprintf("%s in %s; logs keep their words (%d)", count(p.mentions, "mention"), count(p.idFiles, "document"), p.logIDs))
	row("links", fmt.Sprintf("%d in %s", p.links, count(p.linkFiles, "file")))
	if p.project != "" && p.project != p.root {
		row("paths", fmt.Sprintf("%s of %s/ in %s; logs keep their words (%d)", count(p.paths, "mention"), p.old, count(p.pathFiles, "document"), p.logPaths))
	}
	indexes := fmt.Sprintf("features/INDEX.md: %s moved from INDEX.md", count(p.listings, "listing"))
	if p.generated > 0 {
		indexes += fmt.Sprintf(", %s written for groups INDEX.md did not list", count(p.generated, "listing"))
	}
	if p.tags > 0 {
		indexes += "; " + count(p.tags, "status tag") + " removed"
	}
	row("indexes", indexes)
	switch {
	case p.project == "":
		row("outside", "not searched, nor its path inside it: the bundle is not in a git repository")
	case p.project == p.root:
		row("outside", "nothing: the bundle is its own git repository")
	default:
		row("outside", fmt.Sprintf("%s in %s; %s in %s", count(p.outMentions, "mention"), count(p.outMentionFiles, "file"), count(p.outLinks, "link"), count(p.outLinkFiles, "file")))
		if len(p.left) > 0 {
			var byWhy []string
			for _, why := range []string{"after a longer path", "in a URL", elsewhere} {
				n := 0
				for _, l := range p.left {
					if l.why == why {
						n++
					}
				}
				if n > 0 {
					byWhy = append(byWhy, count(n, "mention")+" "+why)
				}
			}
			row("", "left as they are: "+strings.Join(byWhy, ", ")+" (listed below)")
		}
		if p.managed > 0 {
			row("", fmt.Sprintf("skipped: %s in what `fdf install` manages, which it rewrites", count(p.managed, "reference")))
		}
	}

	fmt.Fprintln(out, "\nfiles:")
	for _, g := range p.groups {
		fmt.Fprintf(out, "  move    %s/ → features/%s/  (%s)\n", g, g, count(p.groupFiles(g), "file"))
	}
	for _, o := range sortedKeys(p.moves) {
		fmt.Fprintf(out, "  move    %s → %s\n", o, p.to(o))
	}
	for _, g := range p.gone {
		fmt.Fprintf(out, "  remove  %s\n", g)
	}
	for _, rel := range sortedKeys(p.texts) {
		verb := "write"
		if p.isThere(rel) {
			verb = "edit"
		}
		line := fmt.Sprintf("  %-7s %s", verb, rel)
		why := p.why[rel]
		if n, ok := p.stubs[rel]; ok {
			why = append(why, "a test stub with "+count(n, "case"))
		}
		if len(why) > 0 {
			line += "  (" + strings.Join(why, ", ") + ")"
		}
		fmt.Fprintln(out, line)
	}
	for _, rel := range sortedKeys(p.relinks) {
		fmt.Fprintf(out, "  relink  %s → %s\n", rel, p.relinks[rel])
	}
	if len(p.outTexts) > 0 {
		fmt.Fprintln(out, "\noutside the bundle:")
		for _, f := range sortedKeys(p.outTexts) {
			fmt.Fprintf(out, "  edit    %s  (%s)\n", f, strings.Join(p.outWhy[f], ", "))
		}
	}
	if len(p.left) > 0 {
		fmt.Fprintln(out, "\nleft as they are:")
		for _, l := range p.left {
			fmt.Fprintf(out, "  %s:%d  %s  (%s)\n", l.file, l.line, l.path, l.why)
		}
	}
}

// layoutSummary says what the older layouts' steps do to the bundle, or ""
// when they do nothing.
func (p *plan) layoutSummary() string {
	renamed, lifted := 0, 0
	for o, n := range p.moves {
		if path.Dir(o) == path.Dir(n) {
			renamed++
		} else {
			lifted++
		}
	}
	var parts []string
	if renamed > 0 {
		parts = append(parts, count(renamed, "file")+" renamed (0.1)")
	}
	if lifted > 0 {
		parts = append(parts, count(lifted, "trail file")+" lifted (0.3)")
	}
	if len(p.stubs) > 0 {
		parts = append(parts, count(len(p.stubs), "test document")+" stubbed")
	}
	for _, g := range p.gone {
		parts = append(parts, g+" removed")
	}
	return strings.Join(parts, ", ")
}

// groupFiles counts the files in the feature group g, or in every group when
// g is "".
func (p *plan) groupFiles(g string) int {
	n := 0
	for _, f := range p.files {
		if dir, _, ok := strings.Cut(f, "/"); ok && p.isGroup[dir] && (g == "" || dir == g) {
			n++
		}
	}
	return n
}

// isThere reports whether a file of the bundle ends up at rel: a file
// migrate edits, rather than one it writes new.
func (p *plan) isThere(rel string) bool { return p.source(rel) != "" }
