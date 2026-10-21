// Package register implements `fdf debt` and `fdf bug` — the bundle's two
// registers. A debt is a known gap between what the project says and what the
// code does; a bug is a known defect, the software doing something wrong,
// that has not been repaired yet. Both are cheap to file on purpose: a register
// only works if writing an entry costs nothing. The chores here are what keep
// one worth reading — listing what is outstanding, and clearing out what is
// closed.
package register

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/logs"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

var (
	typeRe   = regexp.MustCompile(`(?m)^type:\s*(\S+)`)
	statusRe = regexp.MustCompile(`(?m)^status:\s*(\S+)`)
	titleRe  = regexp.MustCompile(`(?m)^title:\s*(.+)$`)
	stampRe  = regexp.MustCompile(`(?m)^timestamp:\s*(\S+)`)
)

// Statuses is the vocabulary both registers share, in lifecycle order.
var Statuses = []string{"open", "accepted", "resolved"}

// Kind is one register: where it lives, what it holds, and what a new entry
// looks like.
type Kind struct {
	Dir      string // "debts" or "bugs"
	Type     string // the document type: "Debt" or "Bug"
	noun     string
	logTitle string
	template func(title, now string, f Filing) string
	next     func(id string, f Filing) string
}

// Filing is what a new entry is filed with. Cost and Effort are -1 when not
// given, and Rated says whether the bundle's pin takes them at all: spec 1.2
// gave an entry its cost and effort.
type Filing struct {
	Affects, Resources []string
	Cost, Effort       int
	Rated              bool
}

// Unrated is a Cost or Effort not given.
const Unrated = -1

// scoreLine writes a `cost` or `effort` line, or, when none was given, a
// commented one that says what the number means — only for a bundle whose
// pin takes them.
func scoreLine(f Filing, field string, v int, what string) string {
	switch {
	case !f.Rated:
		return ""
	case v != Unrated:
		return fmt.Sprintf("%s: %d\n", field, v)
	}
	// The explanation has a line of its own: frontmatter has no comment after
	// a value, so uncommenting `# cost: 4   # …` would make the comment the
	// value's.
	return fmt.Sprintf("# %s, from SPEC.md's Cost and effort table:\n# %s: <0-9>\n", what, field)
}

// scoreLines are an entry's cost and effort lines.
func scoreLines(f Filing) string {
	return scoreLine(f, "cost", f.Cost, "What it costs while it stays, 0 negligible to 9 an emergency") +
		scoreLine(f, "effort", f.Effort, "The work to close it, 0 minutes to 9 longer than a quarter")
}

// costHint is the `# Cost` placeholder: from 1.2 on, the reasoning behind
// the cost number.
func costHint(f Filing, noun string) string {
	if f.Rated {
		return "TODO — what carrying this costs, and what it risks: why `cost` is the number it is. The highest level of the scale in SPEC.md, *Cost and effort*, that is true of it is its cost."
	}
	if noun == "bug" {
		return "TODO — what the defect costs while it stays. Optional, but it is what lets the register be prioritized."
	}
	return "TODO — what carrying this costs, and what it risks. Optional, but it is what lets the register be prioritized without a priority field."
}

// Debt is the debt register.
var Debt = Kind{
	Dir: "debts", Type: "Debt", noun: "debt",
	logTitle: "# Debt Log\n\nDebts retired from the register by `fdf debt --cleanup`.\nThe register itself is the files beside this one; this is what they became.\n",
	template: func(title, now string, f Filing) string {
		return fmt.Sprintf(`---
type: Debt
status: open
title: %s
description: TODO — one sentence on what is missing.
%s
%stimestamp: %s
---

# Gap

TODO — what is not as it should be, concretely enough that someone else could confirm it. Name files and counts, not impressions.

# Cost

%s
`, title, resourceLine(f.Resources, "the paths carrying the gap"), scoreLines(f), now, costHint(f, "debt"))
	},
	next: func(_ string, f Filing) string {
		next := "next: fill `# Gap` — concretely enough that someone else could confirm it."
		if len(f.Resources) == 0 {
			next = "next: fill `# Gap` and set `resource` to the paths that carry it — that is how work finds this debt."
		}
		return next + rateHint(f)
	},
}

// Bug is the bug register.
var Bug = Kind{
	Dir: "bugs", Type: "Bug", noun: "bug",
	logTitle: "# Bug Log\n\nBugs retired from the register by `fdf bug --cleanup`.\nThe register itself is the files beside this one; the Fix or Change that\nrepaired each one is its permanent record.\n",
	template: func(title, now string, f Filing) string {
		affectsLine := "# affects: [features/group/slug]     # the features it shows up in (each must exist)"
		if len(f.Affects) > 0 {
			affectsLine = "affects: [" + strings.Join(f.Affects, ", ") + "]"
		}
		return fmt.Sprintf(`---
type: Bug
status: open
title: %s
description: TODO — one sentence on what the software does wrong.
%s
%s
%stimestamp: %s
---

# Symptom

TODO — what the software does wrong, with the evidence: the commands you ran, verbatim, their failing output, and `+"`Reproduced.`"+` A defect nobody reported, spotted while reading the code, gives the code path, the input that reaches it, and `+"`Found by reading.`"+`

# Expected

TODO — what should happen instead. When a scenario already promises it, cite it under `+"`# Violates`"+`; when no document decides it, say so and name the open question — the repair is then a Change.

<!-- # Violates — when a scenario already promises the expected behavior. Its
     presence makes the repair a Fix. One heading per feature in `+"`affects`"+`:

## features/group/slug

- The scenario's name, verbatim
-->

# Root cause

TODO — the one-sentence diagnosis, with its file:line: a line of the code or of its build, CI or dependency configuration, never a bundle document. Not diagnosed yet? Write `+"`Not found yet:`"+` and what you ruled out.

# Cost

%s
`, title, affectsLine, resourceLine(f.Resources, "the paths carrying it"), scoreLines(f), now, costHint(f, "bug"))
	},
	next: func(id string, f Filing) string {
		var missing []string
		if len(f.Affects) == 0 {
			missing = append(missing, "`affects`")
		}
		if len(f.Resources) == 0 {
			missing = append(missing, "`resource`")
		}
		set := ""
		if len(missing) > 0 {
			set = ", and set " + strings.Join(missing, " and ")
		}
		return "next: fill `# Symptom` and `# Expected`" + set + ". When a scenario already promises the expected\n" +
			"      behavior, cite it under `# Violates` and the repair is `fdf fix --from " + id + " …`; when no\n" +
			"      scenario covers it, the repair is `fdf change --from " + id + " …`, and someone decides first." +
			rateHint(f)
	},
}

// rateHint is the line a new entry's next steps end with while it has no
// cost: an entry nobody has rated is untriaged.
func rateHint(f Filing) string {
	if !f.Rated || f.Cost != Unrated {
		return ""
	}
	return "\n      rate it: set `cost` from 0 to 9, the highest level in SPEC.md's *Cost and effort* that is true\n" +
		"      of it, and say why under `# Cost`; set `effort` once the work to close it is understood."
}

type entry struct {
	id, path, status, title, timestamp string
	resolution                         string
	cost, effort                       int // -1 when the entry has none
	tags                               []string
}

// pinned reports whether the commands work on the bundle's pin, and says why
// not when they do not: a 0.x bundle is upgraded with `fdf migrate` first.
func (k Kind) pinned(root string, out io.Writer) bool {
	return scaffold.RequireSupported(root, out)
}

// scan reads every entry of the register: each document layout files in it,
// at any depth, whose type is the register's. Trails, indexes and logs are
// not entries, nor is anything with no place in a 1.0 bundle, such as a file
// in a hidden directory or in a directory beside an entry.
func (k Kind) scan(root string) ([]entry, error) {
	dir := filepath.Join(root, k.Dir)
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}
	b := layout.New(os.DirFS(root))
	var out []entry
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case d.IsDir() && p != dir && strings.HasPrefix(d.Name(), "."):
			return filepath.SkipDir // a tool's state, not the bundle's
		case d.IsDir() || !strings.HasSuffix(p, ".md"):
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if pos := b.File(rel); pos.Kind != layout.Document || pos.Register != k.Dir {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		s := string(raw)
		if m := typeRe.FindStringSubmatch(s); m == nil || strings.Trim(m[1], `"'`) != k.Type {
			return nil
		}
		e := entry{id: strings.TrimSuffix(rel, ".md"), path: p}
		if m := statusRe.FindStringSubmatch(s); m != nil {
			e.status = strings.Trim(m[1], `"'`)
		}
		if m := titleRe.FindStringSubmatch(s); m != nil {
			e.title = strings.TrimSpace(strings.Trim(strings.TrimSpace(m[1]), `"'`))
		}
		if m := stampRe.FindStringSubmatch(s); m != nil {
			e.timestamp = strings.Trim(m[1], `"'`)
		}
		e.resolution = firstParagraphOf(s, "Resolution")
		e.cost, e.effort = Unrated, Unrated
		if data, _, ok := bundle.Frontmatter(s); ok {
			e.cost, e.effort = bundle.Score(data["cost"]), bundle.Score(data["effort"])
			e.tags = bundle.DocumentTags(data)
		}
		out = append(out, e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out, nil
}

// firstParagraphOf returns the first paragraph under a top-level `# heading`,
// its lines joined into one: prose is wrapped, and the first line alone stops
// mid-sentence.
func firstParagraphOf(text, heading string) string {
	in := false
	var para []string
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			if in {
				break
			}
			in = strings.EqualFold(strings.TrimSpace(t[2:]), heading)
			continue
		}
		if !in {
			continue
		}
		if t == "" {
			if len(para) > 0 {
				break
			}
			continue
		}
		para = append(para, t)
	}
	return strings.Join(para, " ")
}

// day is the UTC date an entry was filed, so the table stays narrow: a date
// as it is, and an RFC 3339 time on the UTC date of its instant — 23:30 at
// -05:00 is the next day in UTC, the date every fdf command writes. Anything
// else is cut to a date's width, as it was written.
func day(ts string) string {
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.UTC().Format(time.DateOnly)
	}
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// ListOptions narrows and orders a listing.
type ListOptions struct {
	Status    string // "" for every status, or one of Statuses
	Tag       string // "" for every entry, or only those that carry this tag
	MinCost   int    // 0 for every entry, or only those whose cost is at least this
	Untriaged bool   // only the open and accepted entries with no cost
	ByValue   bool   // order by cost − effort instead of by cost
}

// rated reports whether the bundle at root takes cost and effort: spec 1.2
// gave them to an entry.
func rated(root string) bool { return specver.Since(scaffold.Pin(root), layout.CostSince) }

// untriaged reports whether an entry still waits for its cost: open or
// accepted, and not rated.
func untriaged(e entry) bool {
	return e.cost == Unrated && (e.status == "open" || e.status == "accepted")
}

// List prints the register as a table. From spec 1.2 on it leads with each
// entry's cost and effort, costliest first and, among equal costs, cheapest
// to close first; opts.ByValue orders by cost − effort instead, which on two
// scales that each multiply step by step compares harm against work.
func (k Kind) List(root string, opts ListOptions, out io.Writer) int {
	if !k.pinned(root, out) {
		return 1
	}
	scored := rated(root)
	if !scored && (opts.Untriaged || opts.ByValue || opts.MinCost > 0) {
		fmt.Fprintf(out, "error: --untriaged, --min-cost and --sort value read `cost` and `effort`, which spec %s gave an entry, and this bundle pins %s — `fdf migrate` moves the pin, the user's decision\n", layout.CostSince, scaffold.Pin(root))
		return 1
	}
	entries, err := k.scan(root)
	if err != nil {
		fmt.Fprintf(out, "no %s/ directory at %s — nothing is on the register yet.\n", k.Dir, root)
		return 0
	}
	var rows []entry
	for _, e := range entries {
		switch {
		case opts.Status != "" && e.status != opts.Status:
		case opts.Tag != "" && !carries(e, opts.Tag):
		case opts.MinCost > 0 && e.cost < opts.MinCost:
		case opts.Untriaged && !untriaged(e):
		default:
			rows = append(rows, e)
		}
	}
	if len(rows) == 0 {
		var what []string
		if opts.Untriaged {
			what = append(what, "untriaged")
		}
		if opts.Status != "" {
			what = append(what, opts.Status)
		}
		what = append(what, k.noun)
		if opts.Tag != "" {
			what = append(what, "tagged "+opts.Tag)
		}
		if opts.MinCost > 0 {
			what = append(what, fmt.Sprintf("with a cost of %d or more", opts.MinCost))
		}
		fmt.Fprintf(out, "no %s on the register.\n", strings.Join(what, " "))
		return 0
	}
	if scored {
		order(rows, opts.ByValue)
	}

	head := strings.ToUpper(k.noun)
	wID, wStatus := len(head), len("STATUS")
	for _, e := range rows {
		if len(e.id) > wID {
			wID = len(e.id)
		}
		if len(e.status) > wStatus {
			wStatus = len(e.status)
		}
	}
	lead, leadHead := func(entry) string { return "" }, ""
	if scored {
		leadHead = "COST  EFFORT  "
		if opts.ByValue {
			leadHead = "COST  EFFORT  VALUE  "
		}
		lead = func(e entry) string {
			s := fmt.Sprintf("%4s  %6s  ", score(e.cost), score(e.effort))
			if opts.ByValue {
				v := "-"
				if e.cost != Unrated && e.effort != Unrated {
					v = fmt.Sprint(e.cost - e.effort)
				}
				s += fmt.Sprintf("%5s  ", v)
			}
			return s
		}
	}
	// UPDATED is the UTC date of the entry's timestamp: when it last changed
	// in substance — when it was filed, until it is amended or rated.
	fmt.Fprintf(out, "%s%-*s  %-*s  %-10s  %s\n", leadHead, wStatus, "STATUS", wID, head, "UPDATED", "TITLE")
	for _, e := range rows {
		fmt.Fprintf(out, "%s%-*s  %-*s  %-10s  %s\n", lead(e), wStatus, e.status, wID, e.id, day(e.timestamp), e.title)
	}

	counts, waiting := map[string]int{}, 0
	for _, e := range entries {
		counts[e.status]++
		if untriaged(e) {
			waiting++
		}
	}
	fmt.Fprintf(out, "\n%d shown; register holds %d open, %d accepted, %d resolved.\n",
		len(rows), counts["open"], counts["accepted"], counts["resolved"])
	if scored && waiting > 0 && !opts.Untriaged {
		fmt.Fprintf(out, "%d open or accepted %s(s) have no `cost` yet: `fdf %s --untriaged` lists them, and SPEC.md's *Cost and effort* says how to rate one.\n", waiting, k.noun, k.noun)
	}
	if counts["resolved"] > 0 && opts.Status != "resolved" {
		fmt.Fprintf(out, "run `fdf %s --cleanup` to fold the resolved ones into %s/LOG.md and clear them.\n", k.noun, k.Dir)
	}
	return 0
}

// carries reports whether e carries tag.
func carries(e entry, tag string) bool {
	for _, t := range e.tags {
		if t == tag {
			return true
		}
	}
	return false
}

// score is a cost or effort as the table shows it.
func score(v int) string {
	if v == Unrated {
		return "-"
	}
	return fmt.Sprint(v)
}

// order sorts a register's rows for reading. By cost: highest cost first,
// then lowest effort, an entry with none of either after those that have
// one. By value: the entries with both numbers first, highest cost − effort
// first and then highest cost; then those with a cost alone, by cost; then
// the untriaged. Ties keep the ID order scan gave them.
func order(rows []entry, byValue bool) {
	rank := func(v int) int { // higher first; an unrated one last
		if v == Unrated {
			return -100
		}
		return v
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if byValue {
			va, vb := a.cost != Unrated && a.effort != Unrated, b.cost != Unrated && b.effort != Unrated
			switch {
			case va != vb:
				return va
			case va && a.cost-a.effort != b.cost-b.effort:
				return a.cost-a.effort > b.cost-b.effort
			}
		}
		if rank(a.cost) != rank(b.cost) {
			return rank(a.cost) > rank(b.cost)
		}
		if ea, eb := a.effort == Unrated, b.effort == Unrated; ea != eb {
			return !ea
		}
		return a.effort < b.effort
	})
}

// Cleanup retires resolved entries from the register: each is recorded in
// <dir>/LOG.md and its file removed. Open and accepted entries are never
// touched. A group left with no entry goes too (emptied). dryRun prints the
// plan and changes nothing; noLog skips the log entry and only removes the
// files.
func (k Kind) Cleanup(root string, dryRun, noLog bool, out io.Writer) int {
	if !k.pinned(root, out) {
		return 1
	}
	entries, err := k.scan(root)
	if err != nil {
		fmt.Fprintf(out, "no %s/ directory at %s — nothing to clean up.\n", k.Dir, root)
		return 0
	}
	var done []entry
	for _, e := range entries {
		if e.status == "resolved" {
			done = append(done, e)
		}
	}
	if len(done) == 0 {
		fmt.Fprintf(out, "no resolved %ss to clear; the register is already what is outstanding.\n", k.noun)
		return 0
	}

	verb := "removing"
	if dryRun {
		verb = "would remove"
	}
	for _, e := range done {
		fmt.Fprintf(out, "%s %s.md — %s\n", verb, e.id, e.title)
		if e.resolution == "" {
			fmt.Fprintf(out, "  warning: no `# Resolution` to log (validate should have caught this)\n")
		}
		if !dryRun {
			continue
		}
		// Everything the real run does to an entry, so the plan hides nothing.
		if fileExists(logOf(e)) {
			fmt.Fprintf(out, "  would remove %s.log.md with it — its entries are not kept\n", e.id)
		}
		rel, _ := filepath.Rel(root, e.path)
		if idx := scaffold.ListedIn(root, filepath.ToSlash(rel)); idx != "" {
			fmt.Fprintf(out, "  would unlist it from %s\n", idx)
		}
	}
	gone, kept := k.emptied(root, done)
	goneSet := map[string]bool{}
	for _, g := range gone {
		goneSet[g] = true
	}
	if dryRun {
		for _, g := range gone {
			fmt.Fprintf(out, "would remove %s: the group would hold no entry\n", groupFiles(root, g))
			if idx := scaffold.GroupListedIn(root, g); idx != "" && !goneSet[path.Dir(g)] {
				fmt.Fprintf(out, "  would unlist %s/ from %s\n", g, idx)
			}
		}
		for _, g := range kept {
			fmt.Fprintf(out, "note: %s/INDEX.md would list nothing, but %s/ holds more than its entries — both would stay\n", g, g)
		}
		fmt.Fprintf(out, "\ndry run: %d resolved %s(s) left in place. Re-run without --dry-run to clear them.\n", len(done), k.noun)
		return 0
	}

	if !noLog {
		if err := k.appendLog(root, done); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
	}
	removed := 0
	for _, e := range done {
		if err := os.Remove(e.path); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		removed++
		// So does its listing: a link to a cleared entry would be broken.
		rel, _ := filepath.Rel(root, e.path)
		if idx, err := scaffold.Unlist(root, filepath.ToSlash(rel)); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		} else if idx != "" {
			fmt.Fprintf(out, "unlisted it from %s\n", idx)
		}
		// An entry's only legal sibling goes with it.
		if log := logOf(e); fileExists(log) {
			if err := os.Remove(log); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
			fmt.Fprintf(out, "removed %s.log.md\n", e.id)
		}
	}
	for _, g := range gone {
		dir := filepath.Join(root, filepath.FromSlash(g))
		what := groupFiles(root, g)
		for _, f := range []string{"INDEX.md", ".DS_Store"} {
			if err := os.Remove(filepath.Join(dir, f)); err != nil && !os.IsNotExist(err) {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
		}
		if err := os.Remove(dir); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		fmt.Fprintf(out, "removed %s: the group holds no entry\n", what)
		if goneSet[path.Dir(g)] {
			continue // its parent goes too, index and all
		}
		// A listing of the group would link to nothing.
		if idx, err := scaffold.UnlistGroup(root, g); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		} else if idx != "" {
			fmt.Fprintf(out, "unlisted %s/ from %s\n", g, idx)
		}
	}
	for _, g := range kept {
		fmt.Fprintf(out, "note: %s/INDEX.md lists nothing now, but %s/ holds more than its entries — both stay\n", g, g)
	}
	if noLog {
		fmt.Fprintf(out, "\ncleared %d resolved %s(s); nothing was logged (--no-log).\n", removed, k.noun)
		return 0
	}
	fmt.Fprintf(out, "\ncleared %d resolved %s(s); each is recorded in %s/LOG.md.\n", removed, k.noun, k.Dir)
	return 0
}

// emptied sorts out the groups a cleanup takes the last entries from, at any
// depth. A group whose directory then holds nothing but an index listing
// nothing is gone: the index, the directory and the group's own listing go
// too, or validation would warn of an index with no listing, linked from its
// parent's. The .DS_Store macOS Finder leaves in a directory it has shown is
// not the group's, and goes with it. A group that goes can empty its parent in turn, so groups are
// decided deepest first, and gone lists them in the order they are removed. A
// group whose index lists nothing but which holds anything else — its LOG.md,
// an entry the index never listed — is kept, and named so a person can
// decide.
func (k Kind) emptied(root string, done []entry) (gone, kept []string) {
	removed := map[string]bool{} // bundle-relative paths the cleanup removes
	pending := map[string]bool{} // groups to decide
	for _, e := range done {
		rel, _ := filepath.Rel(root, e.path)
		rel = filepath.ToSlash(rel)
		removed[rel] = true
		if fileExists(logOf(e)) {
			removed[strings.TrimSuffix(rel, ".md")+".log.md"] = true
		}
		if g := path.Dir(rel); g != k.Dir {
			pending[g] = true
		}
	}
	for len(pending) > 0 {
		g := deepest(pending)
		delete(pending, g)
		lists := false
		for _, t := range scaffold.Listed(root, g+"/INDEX.md") {
			lists = lists || !removed[t]
		}
		if lists {
			continue
		}
		files, _ := os.ReadDir(filepath.Join(root, filepath.FromSlash(g)))
		alone := true
		for _, f := range files {
			alone = alone && (f.Name() == "INDEX.md" || f.Name() == ".DS_Store" || removed[g+"/"+f.Name()])
		}
		switch {
		case alone:
			gone = append(gone, g)
			removed[g], removed[g+"/INDEX.md"] = true, true
			if parent := path.Dir(g); parent != k.Dir {
				pending[parent] = true
			}
		case fileExists(filepath.Join(root, filepath.FromSlash(g), "INDEX.md")):
			kept = append(kept, g)
		}
	}
	return gone, kept
}

// deepest is the group in groups with the most levels, and of those the
// first by name.
func deepest(groups map[string]bool) string {
	best := ""
	for g := range groups {
		if d, bd := strings.Count(g, "/"), strings.Count(best, "/"); best == "" || d > bd || d == bd && g < best {
			best = g
		}
	}
	return best
}

// groupFiles names what removing an emptied group removes: its directory,
// and its INDEX.md when it has one.
func groupFiles(root, g string) string {
	if fileExists(filepath.Join(root, filepath.FromSlash(g), "INDEX.md")) {
		return g + "/INDEX.md and " + g + "/"
	}
	return g + "/"
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// logOf is the path of an entry's log, its only legal sibling.
func logOf(e entry) string { return strings.TrimSuffix(e.path, ".md") + ".log.md" }

// appendLog writes one line per retired entry under today's date heading,
// newest first — the ordering every FDF log uses. The entry's ID leads the
// line in bold: a done Fix or Change that `resolves` a cleared bug is checked
// against it.
func (k Kind) appendLog(root string, done []entry) error {
	path := filepath.Join(root, k.Dir, "LOG.md")
	raw, err := os.ReadFile(path)
	body := string(raw)
	if err != nil {
		body = k.logTitle
	}
	var b strings.Builder
	for _, e := range done {
		res := e.resolution
		if res == "" {
			res = "resolved"
		}
		fmt.Fprintf(&b, "* **%s** — %s. %s\n", e.id, e.title, res)
	}
	body = logs.Insert(body, b.String())
	return os.WriteFile(path, []byte(body), 0o644)
}

// resourceLine writes the `resource` frontmatter line, or a commented example
// when none was given.
func resourceLine(resources []string, what string) string {
	if len(resources) == 0 {
		return "# resource: [internal/http/admin.go]   # " + what + " (R1: they must exist)"
	}
	return "resource: [" + strings.Join(resources, ", ") + "]"
}

// New scaffolds an entry at <dir>/[<group>/…]<slug>.md, groups nested to any
// depth — or at the full ID, which log, mv and --from take: typing it here
// files the entry where the ID says, not a level deeper. affects (bugs only)
// names the features the defect shows up in, by their full IDs; each must
// exist. resources are the project-relative paths carrying the entry.
func (k Kind) New(root, name string, affects, resources []string, out io.Writer) int {
	return k.File(root, name, Filing{Affects: affects, Resources: resources, Cost: Unrated, Effort: Unrated}, out)
}

// File is New with everything an entry may be filed with: from spec 1.2 on,
// its cost and effort too, each from 0 to 9. f.Rated is the bundle's to
// decide, by its pin.
func (k Kind) File(root, name string, f Filing, out io.Writer) int {
	if !k.pinned(root, out) {
		return 1
	}
	for _, s := range Statuses {
		if strings.TrimPrefix(name, k.Dir+"/") == s {
			fmt.Fprintf(out, "error: %q is a status, not a slug — did you mean `fdf %s --%s`?\n", s, k.noun, s)
			return 1
		}
	}
	f.Rated = rated(root)
	if !f.Rated && (f.Cost != Unrated || f.Effort != Unrated) {
		fmt.Fprintf(out, "error: --cost and --effort set fields spec %s gave an entry, and this bundle pins %s — `fdf migrate` moves the pin, the user's decision\n", layout.CostSince, scaffold.Pin(root))
		return 1
	}
	for _, v := range []int{f.Cost, f.Effort} {
		if v != Unrated && (v < 0 || v > 9) {
			fmt.Fprintf(out, "error: --cost and --effort take a whole number from 0 to 9, got %d — SPEC.md's *Cost and effort* says what each level means\n", v)
			return 2
		}
	}
	for _, a := range f.Affects {
		if !scaffold.IsFeature(root, a) {
			fmt.Fprintf(out, "error: --affects names %s, which is not a feature in this bundle%s\n", a, scaffold.FeatureHint(root, a))
			return 1
		}
	}
	id := scaffold.NewID(root, k.Dir, name, out)
	if id == "" {
		return 1
	}
	file := filepath.Join(root, filepath.FromSlash(id)+".md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	title := scaffold.Title(path.Base(id))
	body := k.template(title, time.Now().UTC().Format("2006-01-02T15:04:05Z"), f)
	if err := scaffold.WriteNew(root, id, body); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if code := scaffold.EnsureIndex(root, k.Dir, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "wrote %s.md (type: %s, status: open)\n", id, k.Type)
	if code := scaffold.ListEntry(root, k.Dir, strings.TrimPrefix(id, k.Dir+"/"), title, "TODO", out); code != 0 {
		return code
	}
	fmt.Fprintln(out, k.next(id, f))
	return 0
}
