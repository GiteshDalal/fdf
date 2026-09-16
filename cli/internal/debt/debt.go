// Package debt implements `fdf debt` — the register of known gaps between
// what the project says and what the code does. A debt is cheap to file on
// purpose: a register only works if writing an entry costs nothing. The
// chores here are what keep it worth reading — listing what is outstanding,
// and clearing out what is paid.
package debt

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

var (
	slugRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	groupedRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)/([a-z0-9][a-z0-9-]*)$`)
	typeRe    = regexp.MustCompile(`(?m)^type:\s*(\S+)`)
	statusRe  = regexp.MustCompile(`(?m)^status:\s*(\S+)`)
	titleRe   = regexp.MustCompile(`(?m)^title:\s*(.+)$`)
	stampRe   = regexp.MustCompile(`(?m)^timestamp:\s*(\S+)`)
	trailRe   = regexp.MustCompile(`\.[a-z]+\.md$`)
	dateHead  = regexp.MustCompile(`(?m)^##\s+\d{4}-\d{2}-\d{2}\s*$`)
)

// Statuses is the Debt vocabulary, in lifecycle order.
var Statuses = []string{"open", "accepted", "resolved"}

type entry struct {
	id, path, status, title, timestamp string
	resolution                         string
}

// scan reads every debt document under debts/. Trail siblings (<slug>.log.md)
// and INDEX/LOG files are skipped: only the register entries themselves.
func scan(root string) ([]entry, error) {
	dir := filepath.Join(root, "debts")
	var out []entry
	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		name := filepath.Base(path)
		if name == "INDEX.md" || name == "LOG.md" || trailRe.MatchString(name) {
			return nil
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		s := string(raw)
		if m := typeRe.FindStringSubmatch(s); m == nil || strings.Trim(m[1], `"'`) != "Debt" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		e := entry{id: strings.TrimSuffix(filepath.ToSlash(rel), ".md"), path: path}
		if m := statusRe.FindStringSubmatch(s); m != nil {
			e.status = strings.Trim(m[1], `"'`)
		}
		if m := titleRe.FindStringSubmatch(s); m != nil {
			e.title = strings.TrimSpace(strings.Trim(strings.TrimSpace(m[1]), `"'`))
		}
		if m := stampRe.FindStringSubmatch(s); m != nil {
			e.timestamp = strings.Trim(m[1], `"'`)
		}
		e.resolution = firstLineOf(s, "Resolution")
		out = append(out, e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out, nil
}

// firstLineOf returns the first non-empty line under a top-level `# heading`.
func firstLineOf(text, heading string) string {
	in := false
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			if in {
				return ""
			}
			in = strings.EqualFold(strings.TrimSpace(t[2:]), heading)
			continue
		}
		if in && t != "" {
			return t
		}
	}
	return ""
}

// day trims an ISO timestamp to its date, so the table stays narrow.
func day(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// List prints the register as a table. filter is "" for everything, or one of
// Statuses.
func List(root, filter string, out io.Writer) int {
	entries, err := scan(root)
	if err != nil {
		fmt.Fprintf(out, "no debts/ directory at %s — nothing is on the register yet.\n", root)
		return 0
	}
	var rows []entry
	for _, e := range entries {
		if filter == "" || e.status == filter {
			rows = append(rows, e)
		}
	}
	if len(rows) == 0 {
		what := "debt"
		if filter != "" {
			what = filter + " debt"
		}
		fmt.Fprintf(out, "no %s on the register.\n", what)
		return 0
	}

	wID, wStatus := len("DEBT"), len("STATUS")
	for _, e := range rows {
		if len(e.id) > wID {
			wID = len(e.id)
		}
		if len(e.status) > wStatus {
			wStatus = len(e.status)
		}
	}
	fmt.Fprintf(out, "%-*s  %-*s  %-10s  %s\n", wStatus, "STATUS", wID, "DEBT", "FILED", "TITLE")
	for _, e := range rows {
		fmt.Fprintf(out, "%-*s  %-*s  %-10s  %s\n", wStatus, e.status, wID, e.id, day(e.timestamp), e.title)
	}

	counts := map[string]int{}
	for _, e := range entries {
		counts[e.status]++
	}
	fmt.Fprintf(out, "\n%d shown; register holds %d open, %d accepted, %d resolved.\n",
		len(rows), counts["open"], counts["accepted"], counts["resolved"])
	if counts["resolved"] > 0 && filter != "resolved" {
		fmt.Fprintln(out, "run `fdf debt --cleanup` to fold the resolved ones into debts/LOG.md and clear them.")
	}
	return 0
}

// Cleanup retires resolved debts from the register: each is recorded in
// debts/LOG.md and its file removed. Open and accepted debts are never
// touched. dryRun prints the plan and changes nothing; noLog skips the log
// entry and only removes the files.
func Cleanup(root string, dryRun, noLog bool, out io.Writer) int {
	entries, err := scan(root)
	if err != nil {
		fmt.Fprintf(out, "no debts/ directory at %s — nothing to clean up.\n", root)
		return 0
	}
	var done []entry
	for _, e := range entries {
		if e.status == "resolved" {
			done = append(done, e)
		}
	}
	if len(done) == 0 {
		fmt.Fprintln(out, "no resolved debts to clear; the register is already what is outstanding.")
		return 0
	}

	verb := "removing"
	if dryRun {
		verb = "would remove"
	}
	for _, e := range done {
		fmt.Fprintf(out, "%s %s.md — %s\n", verb, e.id, e.title)
		if e.resolution == "" {
			fmt.Fprintf(out, "  warning: no `# Resolution` line to log (validate should have caught this)\n")
		}
	}
	if dryRun {
		fmt.Fprintf(out, "\ndry run: %d resolved debt(s) left in place. Re-run without --dry-run to clear them.\n", len(done))
		return 0
	}

	if !noLog {
		if err := appendLog(root, done); err != nil {
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
		// A debt's only legal sibling goes with it.
		if log := strings.TrimSuffix(e.path, ".md") + ".log.md"; fileExists(log) {
			if err := os.Remove(log); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
			fmt.Fprintf(out, "removed %s.log.md\n", e.id)
		}
	}
	if noLog {
		fmt.Fprintf(out, "\ncleared %d resolved debt(s); nothing was logged (--no-log).\n", removed)
		return 0
	}
	fmt.Fprintf(out, "\ncleared %d resolved debt(s); each is recorded in debts/LOG.md.\n", removed)
	return 0
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// appendLog writes one line per retired debt under today's date heading,
// newest first — the ordering every FDF log uses.
func appendLog(root string, done []entry) error {
	path := filepath.Join(root, "debts", "LOG.md")
	raw, err := os.ReadFile(path)
	body := string(raw)
	if err != nil {
		body = "# Debt Log\n\nDebts retired from the register by `fdf debt --cleanup`.\nThe register itself is the files beside this one; this is what they became.\n"
	}
	today := time.Now().UTC().Format("2006-01-02")
	var b strings.Builder
	for _, e := range done {
		res := e.resolution
		if res == "" {
			res = "resolved"
		}
		fmt.Fprintf(&b, "* **%s** — %s. %s\n", e.id, e.title, res)
	}

	if idx := strings.Index(body, "## "+today+"\n"); idx >= 0 {
		// Same-day run: extend the existing heading rather than repeat it.
		insert := idx + len("## "+today+"\n")
		body = body[:insert] + b.String() + body[insert:]
	} else {
		// New heading goes above every older one (newest first).
		entryBlock := "## " + today + "\n" + b.String() + "\n"
		if loc := dateHead.FindStringIndex(body); loc != nil {
			body = body[:loc[0]] + entryBlock + body[loc[0]:]
		} else {
			body = strings.TrimRight(body, "\n") + "\n\n" + entryBlock
		}
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// New scaffolds debts/<id>.md, where id is "<slug>" or "<group>/<slug>".
func New(root, id string, out io.Writer) int {
	for _, s := range Statuses {
		if id == s {
			fmt.Fprintf(out, "error: %q is a status, not a slug — did you mean `fdf debt --%s`?\n", id, s)
			return 1
		}
	}
	if !slugRe.MatchString(id) && !groupedRe.MatchString(id) {
		fmt.Fprintf(out, "error: id must be <slug> or <group>/<slug>, lowercase [a-z0-9-]; got %q\n", id)
		return 1
	}
	path := filepath.Join(root, "debts", filepath.FromSlash(id)+".md")
	if fileExists(path) {
		fmt.Fprintf(out, "error: debts/%s.md already exists\n", id)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	slug := id
	if m := groupedRe.FindStringSubmatch(id); m != nil {
		slug = m[2]
	}
	title := strings.ToUpper(slug[:1]) + strings.ReplaceAll(slug[1:], "-", " ")
	body := fmt.Sprintf(`---
type: Debt
status: open
title: %s
description: TODO — one sentence on what is missing.
# resource: [internal/http/admin.go]   # the paths carrying the gap (R1: they must exist)
timestamp: %s
---

# Gap

TODO — what is not as it should be, concretely enough that someone else could
confirm it. Name files and counts, not impressions.

# Cost

TODO — what carrying this costs, and what it risks. Optional, but it is what
lets the register be prioritized without a priority field.
`, title, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if code := scaffold.EnsureDebtsIndex(root, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "wrote debts/%s.md (type: Debt, status: open)\n", id)
	fmt.Fprintln(out, "next: fill `# Gap` and set `resource` to the paths that carry it — that is how work finds this debt.")
	return 0
}
