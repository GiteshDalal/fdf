// Package changes implements `fdf change`, `fdf fix`, and `fdf history` —
// the v0.5 post-delivery surface. A Change alters documented behavior and
// carries a design gate; a Fix restores behavior the feature document already
// describes and needs none. Both declare, in their body, the effects they
// will have on the features they affect, which is what F10 later verifies.
package changes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	// Aliased: this package's tests name a helper bundle.
	validation "github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

var (
	slugRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	groupedRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)/([a-z0-9][a-z0-9-]*)$`)
	featureRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9-]*$`)
	typeRe    = regexp.MustCompile(`(?m)^type:\s*(\S+)`)
	statusRe  = regexp.MustCompile(`(?m)^status:\s*(\S+)`)
	titleRe   = regexp.MustCompile(`(?m)^title:\s*(.+)$`)
)

// New scaffolds a Change (docType "Change") or Fix (docType "Fix") at
// changes/<id>.md, where id is "<slug>" or "<group>/<slug>".
func New(root, id, docType string, affects []string, out io.Writer) int {
	return NewFrom(root, id, docType, affects, "", out)
}

// NewFrom is New for work that repairs a bug on the register (v0.7): the new
// document takes over the bug's analysis as its permanent record and names
// the bug in `resolves`, which F10 holds to: once the work is done, the bug
// must not read as open. affects defaults to the bug's own. id may also be
// the full ID, changes/<slug>, which files the document in the same place.
// Changes and Fixes are v0.5: an older pin reads changes/ as a feature group,
// so the command refuses there.
func NewFrom(root, id, docType string, affects []string, fromBug string, out io.Writer) int {
	if !scaffold.RequirePin(root, 5, "Changes and Fixes", "changes/ is a feature group, and a "+docType+" written there fails validation (F3)", out) {
		return 1
	}
	id = strings.TrimPrefix(id, "changes/")
	if !slugRe.MatchString(id) && !groupedRe.MatchString(id) {
		fmt.Fprintf(out, "error: id must be <slug> or <group>/<slug>, lowercase [a-z0-9-]; got %q\n", id)
		return 1
	}
	var b *bugDoc
	if fromBug != "" {
		var err error
		if b, err = readBug(root, fromBug); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		if len(affects) == 0 {
			affects = b.affects
		}
		if len(affects) == 0 {
			fmt.Fprintf(out, "error: %s names no feature in `affects`, so there is nothing for a %s to amend.\n", fromBug, docType)
			fmt.Fprintln(out, "  a defect in code no feature documents is repaired after its capability is adopted:")
			fmt.Fprintln(out, "  `fdf adopt --resource <path> <group>/<slug>`, add that feature to the bug's `affects`, then retry.")
			return 1
		}
	}
	if len(affects) == 0 {
		fmt.Fprintf(out, "error: --affects is required — name the delivered feature(s) this %s touches\n", docType)
		return 2
	}
	for _, f := range affects {
		if !featureRe.MatchString(f) {
			fmt.Fprintf(out, "error: --affects takes feature IDs of the form <group>/<slug>; got %q\n", f)
			return 1
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(f)+".md")); err != nil {
			fmt.Fprintf(out, "error: --affects names %s, which is not a feature in this bundle\n", f)
			return 1
		}
	}

	path := filepath.Join(root, "changes", filepath.FromSlash(id)+".md")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(out, "error: changes/%s.md already exists\n", id)
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
	affectsField := affects[0]
	if len(affects) > 1 {
		affectsField = "[" + strings.Join(affects, ", ") + "]"
	}
	var sb strings.Builder
	resolvesLine := ""
	if b != nil {
		resolvesLine = "resolves: " + b.id + "\n"
	}
	fmt.Fprintf(&sb, "---\ntype: %s\ntitle: %s\ndescription: TODO — one sentence.\nstatus: draft\naffects: %s\n%stimestamp: %s\n---\n\n",
		docType, title, affectsField, resolvesLine, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	if docType == "Change" {
		problem := "TODO — what is inadequate about the delivered behavior, and for whom."
		if b != nil {
			problem = orTODO(b.symptom, "the observed wrong behavior") + "\n\nWhat should happen instead: " + orTODO(b.expected, "the expected behavior")
		}
		sb.WriteString("# Problem\n\n" + problem + "\n\n")
		sb.WriteString("# Scenario changes\n\n")
		for _, f := range affects {
			fmt.Fprintf(&sb, "## %s\n\n- add: TODO — a scenario name that must exist once this is done\n- modify: TODO — an existing scenario whose steps change (name unchanged)\n- remove: TODO — a scenario name that must not exist once this is done\n\n", f)
		}
		sb.WriteString("TODO — delete the lines that do not apply, and this one. Names are matched\nverbatim against the feature's Gherkin (F10).\n\n# Impact\n\nTODO — migrations, compatibility, rollout. Optional.\n")
	} else {
		symptom, cause := "TODO — the observed wrong behavior, with a reproduction.", "TODO — why the code diverged from the documented behavior."
		if b != nil {
			symptom, cause = orTODO(b.symptom, "the observed wrong behavior"), orTODO(b.rootCause, "why the code diverged from the documented behavior")
		}
		sb.WriteString("# Symptom\n\n" + symptom + "\n\n")
		sb.WriteString("# Root cause\n\n" + cause + "\n\n")
		sb.WriteString("# Regression cases\n\n")
		for _, f := range affects {
			fmt.Fprintf(&sb, "## %s\n\n", f)
			if b != nil && len(b.violates[f]) > 0 {
				for _, n := range b.violates[f] {
					fmt.Fprintf(&sb, "- %s — TODO the command, test path, or manual procedure\n", n)
				}
				sb.WriteString("\n")
				continue
			}
			sb.WriteString("- TODO scenario name (verbatim, must already exist) — TODO the command, test path, or manual procedure\n\n")
		}
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "created changes/%s.md (type: %s, status: draft)\n", id, docType)

	if code := ensureIndex(root, id, title, docType, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "\ndone: %s changes/%s affects %s\n", docType, id, strings.Join(affects, ", "))
	if b != nil {
		fmt.Fprintf(out, "resolves %s — its analysis is copied here; once this is done, flip the bug to resolved with a\n", b.id)
		fmt.Fprintf(out, "  `# Resolution` naming changes/%s (F10 holds the bug to it).\n", id)
	}
	if docType == "Change" {
		fmt.Fprintln(out, "next: fill `# Scenario changes`, then write changes/"+id+".spec.md and get the design approved (status: specified).")
	} else {
		fmt.Fprintln(out, "next: fill `# Regression cases` with scenarios that already exist, and add the case to each affected feature's .test.md.")
	}
	return 0
}

// orTODO returns s, or a TODO naming what belongs there when s is empty or is
// still a scaffold's own TODO — copying a placeholder would only disguise it.
func orTODO(s, what string) string {
	if t := strings.TrimSpace(s); t == "" || strings.HasPrefix(t, "TODO") {
		return "TODO — " + what + "."
	}
	return s
}

var bugIDRe = regexp.MustCompile(`^bugs/[a-z0-9][a-z0-9-]*(/[a-z0-9][a-z0-9-]*)?$`)

// bugDoc is the part of a Bug a Fix or Change takes over.
type bugDoc struct {
	id                           string
	affects                      []string
	symptom, expected, rootCause string
	violates                     map[string][]string
}

// readBug loads bugs/<id>.md for NewFrom.
func readBug(root, id string) (*bugDoc, error) {
	id = strings.TrimSuffix(strings.TrimPrefix(id, "/"), ".md")
	if !bugIDRe.MatchString(id) {
		return nil, fmt.Errorf("--from takes a bug ID of the form bugs/<slug> or bugs/<group>/<slug>; got %q", id)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(id)+".md"))
	if err != nil {
		return nil, fmt.Errorf("--from names %s, which is not a bug on the register", id)
	}
	text := string(raw)
	if m := typeRe.FindStringSubmatch(text); m == nil || strings.Trim(m[1], `"'`) != "Bug" {
		return nil, fmt.Errorf("--from names %s, whose type is not Bug", id)
	}
	b := &bugDoc{
		id:        id,
		affects:   listField(text, "affects"),
		symptom:   section(text, "Symptom"),
		expected:  section(text, "Expected"),
		rootCause: section(text, "Root cause"),
		violates:  map[string][]string{},
	}
	cur := ""
	// A wrapped entry is one entry, as validation reads it.
	for _, line := range validation.LogicalLines(section(text, "Violates")) {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "## "):
			cur = strings.TrimSpace(t[3:])
		case cur != "" && (strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ")):
			name := strings.TrimSpace(t[2:])
			if loc := noteSepRe.FindStringIndex(name); loc != nil {
				name = strings.TrimSpace(name[:loc[0]])
			}
			b.violates[cur] = append(b.violates[cur], name)
		}
	}
	return b, nil
}

var noteSepRe = regexp.MustCompile(`\s(?:—|–|--)\s`)

// section returns the trimmed text under a top-level `# heading`.
func section(text, heading string) string {
	var b strings.Builder
	in := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "# ") {
			if in {
				break
			}
			in = strings.EqualFold(strings.TrimSpace(line[2:]), heading)
			continue
		}
		if in {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
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

// ensureIndex appends the new document to changes/INDEX.md (and a group index
// when one is in play), creating either if absent.
func ensureIndex(root, id, title, docType string, out io.Writer) int {
	rel := "/changes/" + id + ".md"
	idxDir := filepath.Join(root, "changes")
	listedIn := "changes/INDEX.md"
	if m := groupedRe.FindStringSubmatch(id); m != nil {
		idxDir = filepath.Join(idxDir, m[1])
		listedIn = "changes/" + m[1] + "/INDEX.md"
	}
	idx := filepath.Join(idxDir, "INDEX.md")
	entry := fmt.Sprintf("* [%s](%s) - %s.\n", title, rel, strings.ToLower(docType))
	raw, err := os.ReadFile(idx)
	if err != nil {
		raw = []byte("# Changes\n\nPost-delivery changes and fixes for delivered features.\n\n")
	}
	if err := os.WriteFile(idx, append(raw, []byte(entry)...), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "updated %s (now lists %q)\n", listedIn, title)
	return 0
}

type entry struct {
	id, docType, status, title string
	resolves                   []string
}

// History prints every Change and Fix whose `affects` names a feature, and
// (v0.7) every bug on the register that shows up in it. The edges are computed
// from frontmatter, never from a hand-written back-link on the feature — a
// required back-link is a standing invitation to drift.
func History(root, featureID string, out io.Writer) int {
	if !featureRe.MatchString(featureID) {
		fmt.Fprintf(out, "error: feature id must be <group>/<slug>; got %q\n", featureID)
		return 1
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(featureID)+".md")); err != nil {
		fmt.Fprintf(out, "error: %s is not a feature in this bundle\n", featureID)
		return 1
	}
	found := collect(root, "changes", featureID, func(t string) bool { return t == "Change" || t == "Fix" })
	bugs := collect(root, "bugs", featureID, func(t string) bool { return t == "Bug" })
	if len(found) == 0 {
		fmt.Fprintf(out, "%s: no Change or Fix names it in `affects`\n", featureID)
	} else {
		fmt.Fprintf(out, "%s — %d post-delivery document(s):\n\n", featureID, len(found))
		for _, e := range found {
			fmt.Fprintf(out, "  %-6s %-10s %s\n         %s\n", e.docType, e.status, e.title, e.id)
			for _, r := range e.resolves {
				fmt.Fprintf(out, "         resolves %s\n", r)
			}
		}
	}
	if len(bugs) > 0 {
		fmt.Fprintf(out, "\nknown bugs — %d on the register:\n\n", len(bugs))
		for _, e := range bugs {
			fmt.Fprintf(out, "  %-10s %s\n             %s\n", e.status, e.title, e.id)
		}
	}
	return 0
}

// collect reads the documents under one bundle directory whose type passes
// want and whose `affects` names featureID, in ID order.
func collect(root, dir, featureID string, want func(string) bool) []entry {
	var found []entry
	filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") || d.Name() == "INDEX.md" || d.Name() == "LOG.md" {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		text := string(raw)
		m := typeRe.FindStringSubmatch(text)
		if m == nil || !want(strings.Trim(m[1], `"'`)) {
			return nil
		}
		hit := false
		for _, f := range listField(text, "affects") {
			if f == featureID {
				hit = true
			}
		}
		if !hit {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		e := entry{id: strings.TrimSuffix(filepath.ToSlash(rel), ".md"), docType: m[1], resolves: listField(text, "resolves")}
		if sm := statusRe.FindStringSubmatch(text); sm != nil {
			e.status = sm[1]
		}
		if tm := titleRe.FindStringSubmatch(text); tm != nil {
			e.title = strings.TrimSpace(tm[1])
		}
		found = append(found, e)
		return nil
	})
	sort.Slice(found, func(i, j int) bool { return found[i].id < found[j].id })
	return found
}
