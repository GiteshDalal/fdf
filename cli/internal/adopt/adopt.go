// Package adopt implements the report half of `fdf adopt` (v0.7): the map of
// what the bundle documents and what it does not yet. A project adopting FDF
// has most of its capabilities in code already; it maps them first, breadth
// before depth, and backfills their scenarios as work reaches them. This
// report is how a team sees where that stands — which features are built,
// which adopted, how much of each is written down, and which code no document
// claims at all.
package adopt

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	typeRe     = regexp.MustCompile(`(?m)^type:\s*(\S+)`)
	statusRe   = regexp.MustCompile(`(?m)^status:\s*(\S+)`)
	scenarioRe = regexp.MustCompile(`(?m)^\s*Scenario(?: Outline)?:\s*(\S[^\n]*)`)
)

type feature struct {
	id, status        string
	scenarios, tested int
}

// Map prints the adoption map for the bundle at root. projectRoot is where
// `resource` paths live; empty skips the unclaimed-code section. depth is how
// many path segments the unclaimed code is grouped by.
func Map(root, projectRoot string, depth int, out io.Writer) int {
	features, claims := scan(root)
	if len(features) == 0 {
		fmt.Fprintln(out, "no features in the bundle yet — map what the code already does with `fdf adopt --resource <path> <group>/<slug>`.")
	} else {
		printFeatures(features, out)
	}
	if projectRoot == "" {
		fmt.Fprintln(out, "\n(standalone bundle: no project root, so no code to compare against)")
		return 0
	}
	return printUnclaimed(root, projectRoot, depth, claims, out)
}

// scan reads every feature, and every path a document claims as code it
// documents or built: an adopted feature's `resource`, and the `resource` of
// every task, change and fix.
func scan(root string) ([]feature, []string) {
	var features []feature
	var claims []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		text := string(raw)
		m := typeRe.FindStringSubmatch(text)
		if m == nil {
			return nil
		}
		switch strings.Trim(m[1], `"'`) {
		case "Feature":
			rel, _ := filepath.Rel(root, p)
			f := feature{id: strings.TrimSuffix(filepath.ToSlash(rel), ".md")}
			if sm := statusRe.FindStringSubmatch(text); sm != nil {
				f.status = strings.Trim(sm[1], `"'`)
			}
			test, _ := os.ReadFile(strings.TrimSuffix(p, ".md") + ".test.md")
			for _, s := range scenarioRe.FindAllStringSubmatch(text, -1) {
				f.scenarios++
				if name := strings.TrimSpace(s[1]); len(test) > 0 && strings.Contains(string(test), name) {
					f.tested++
				}
			}
			features = append(features, f)
			if f.status == "adopted" {
				claims = append(claims, listField(text, "resource")...)
			}
		case "Task", "Change", "Fix":
			claims = append(claims, listField(text, "resource")...)
		}
		return nil
	})
	sort.Slice(features, func(i, j int) bool { return features[i].id < features[j].id })
	return features, claims
}

func printFeatures(features []feature, out io.Writer) {
	w := len("FEATURE")
	for _, f := range features {
		if len(f.id) > w {
			w = len(f.id)
		}
	}
	fmt.Fprintf(out, "%-12s  %-*s  %9s  %6s\n", "STATUS", w, "FEATURE", "SCENARIOS", "TESTED")
	built, inFlight, adopted, withScen, retired := 0, 0, 0, 0, 0
	for _, f := range features {
		fmt.Fprintf(out, "%-12s  %-*s  %9d  %6d\n", f.status, w, f.id, f.scenarios, f.tested)
		switch f.status {
		case "adopted":
			adopted++
			if f.scenarios > 0 {
				withScen++
			}
		case "done":
			built++
		case "retired":
			retired++
		default:
			inFlight++
		}
	}
	fmt.Fprintf(out, "\n%d feature(s): %d built, %d in flight, %d retired, %d adopted (%d with scenarios, %d map entries).\n",
		len(features), built, inFlight, retired, adopted, withScen, adopted-withScen)
}

// printUnclaimed groups the project's tracked files by directory and lists
// the groups holding code no document claims, most unclaimed first.
func printUnclaimed(root, projectRoot string, depth int, claims []string, out io.Writer) int {
	cmd := exec.Command("git", "-C", projectRoot, "ls-files")
	raw, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(out, "\n(%s is not a git working tree, so there is no list of tracked code to compare against)\n", projectRoot)
		return 0
	}
	bundleRel := ""
	if abs, aerr := filepath.Abs(root); aerr == nil {
		if r, rerr := filepath.Rel(projectRoot, abs); rerr == nil && !strings.HasPrefix(r, "..") {
			bundleRel = filepath.ToSlash(r)
		}
	}
	for i, c := range claims {
		claims[i] = strings.TrimSuffix(filepath.ToSlash(c), "/")
	}

	type bucket struct {
		dir            string
		files, claimed int
	}
	buckets := map[string]*bucket{}
	for _, file := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if file == "" || strings.HasSuffix(file, ".md") || hiddenPath(file) ||
			(bundleRel != "" && (file == bundleRel || strings.HasPrefix(file, bundleRel+"/"))) {
			continue
		}
		dir := groupDir(file, depth)
		b := buckets[dir]
		if b == nil {
			b = &bucket{dir: dir}
			buckets[dir] = b
		}
		b.files++
		for _, c := range claims {
			if file == c || strings.HasPrefix(file, c+"/") {
				b.claimed++
				break
			}
		}
	}
	var rows []*bucket
	for _, b := range buckets {
		if b.claimed < b.files {
			rows = append(rows, b)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		ui, uj := rows[i].files-rows[i].claimed, rows[j].files-rows[j].claimed
		if ui != uj {
			return ui > uj
		}
		return rows[i].dir < rows[j].dir
	})
	if len(rows) == 0 {
		fmt.Fprintln(out, "\nevery tracked source file is claimed by a feature, task, change or fix.")
		return 0
	}
	fmt.Fprintf(out, "\ncode no document claims yet (git ls-files, grouped %d level(s) deep; markdown and dotfiles skipped):\n\n", depth)
	w := len("DIRECTORY")
	for _, b := range rows {
		if len(b.dir) > w {
			w = len(b.dir)
		}
	}
	const shown = 25
	fmt.Fprintf(out, "  %-*s  %9s  %s\n", w, "DIRECTORY", "UNCLAIMED", "OF")
	for i, b := range rows {
		if i == shown {
			fmt.Fprintf(out, "  …and %d more director(ies) — narrow with --depth\n", len(rows)-shown)
			break
		}
		fmt.Fprintf(out, "  %-*s  %9d  %d\n", w, b.dir, b.files-b.claimed, b.files)
	}
	fmt.Fprintln(out, "\nA claim is a `resource` path on an adopted feature, a task, a change or a fix. Map a capability")
	fmt.Fprintln(out, "with `fdf adopt --resource <path> <group>/<slug>`; this list is a heuristic, not a verdict.")
	return 0
}

// groupDir returns the first depth directories of a file's path, or the
// directory it sits in when that is shallower.
func groupDir(file string, depth int) string {
	parts := strings.Split(file, "/")
	if len(parts) <= 1 {
		return "."
	}
	parts = parts[:len(parts)-1]
	if len(parts) > depth {
		parts = parts[:depth]
	}
	return strings.Join(parts, "/") + "/"
}

func hiddenPath(file string) bool {
	for _, seg := range strings.Split(file, "/") {
		if strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

// listField reads a frontmatter key written as a scalar, an inline list, or a
// block list.
func listField(text, key string) []string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, key+":") {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(line, key+":"))
		var out []string
		if val == "" {
			for _, l := range lines[i+1:] {
				t := strings.TrimSpace(l)
				if !strings.HasPrefix(t, "- ") {
					break
				}
				out = append(out, strings.Trim(strings.TrimSpace(t[2:]), `"'`))
			}
			return out
		}
		for _, p := range strings.Split(strings.Trim(val, "[]"), ",") {
			if p = strings.Trim(strings.TrimSpace(p), `"'`); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return nil
}
