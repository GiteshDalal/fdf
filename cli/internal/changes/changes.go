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
)

var (
	slugRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	groupedRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)/([a-z0-9][a-z0-9-]*)$`)
	featureRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9-]*$`)
	affectsRe = regexp.MustCompile(`(?m)^affects:\s*(.+)$`)
	typeRe    = regexp.MustCompile(`(?m)^type:\s*(\S+)`)
	statusRe  = regexp.MustCompile(`(?m)^status:\s*(\S+)`)
	titleRe   = regexp.MustCompile(`(?m)^title:\s*(.+)$`)
)

// New scaffolds a Change (docType "Change") or Fix (docType "Fix") at
// changes/<id>.md, where id is "<slug>" or "<group>/<slug>".
func New(root, id, docType string, affects []string, out io.Writer) int {
	if !slugRe.MatchString(id) && !groupedRe.MatchString(id) {
		fmt.Fprintf(out, "error: id must be <slug> or <group>/<slug>, lowercase [a-z0-9-]; got %q\n", id)
		return 1
	}
	if len(affects) == 0 {
		fmt.Fprintf(out, "error: --affects is required — name the delivered feature(s) this %s touches\n", strings.ToLower(docType))
		return 1
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
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntype: %s\ntitle: %s\ndescription: TODO — one sentence.\nstatus: draft\naffects: %s\ntimestamp: %s\n---\n\n",
		docType, title, affectsField, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	if docType == "Change" {
		b.WriteString("# Problem\n\nTODO — what is inadequate about the delivered behavior, and for whom.\n\n")
		b.WriteString("# Scenario changes\n\n")
		for _, f := range affects {
			fmt.Fprintf(&b, "## %s\n\n- add: TODO — a scenario name that must exist once this is done\n- modify: TODO — an existing scenario whose steps change (name unchanged)\n- remove: TODO — a scenario name that must not exist once this is done\n\n", f)
		}
		b.WriteString("Delete the lines that do not apply. Names are matched verbatim against\nthe feature's Gherkin (F10).\n\n# Impact\n\nTODO — migrations, compatibility, rollout. Optional.\n")
	} else {
		b.WriteString("# Symptom\n\nTODO — the observed wrong behavior, with a reproduction.\n\n")
		b.WriteString("# Root cause\n\nTODO — why the code diverged from the documented behavior.\n\n")
		b.WriteString("# Regression cases\n\n")
		for _, f := range affects {
			fmt.Fprintf(&b, "## %s\n\n- TODO scenario name (verbatim, must already exist) — TODO the command, test path, or manual procedure\n\n", f)
		}
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "created changes/%s.md (type: %s, status: draft)\n", id, docType)

	if code := ensureIndex(root, id, title, docType, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "\ndone: %s changes/%s affects %s\n", docType, id, strings.Join(affects, ", "))
	if docType == "Change" {
		fmt.Fprintln(out, "next: fill `# Scenario changes`, then write changes/"+id+".spec.md and get the design approved (status: specified).")
	} else {
		fmt.Fprintln(out, "next: fill `# Regression cases` with scenarios that already exist, and add the case to each affected feature's .test.md.")
	}
	return 0
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
	entry := fmt.Sprintf("* [%s](%s) - %s. (**draft**)\n", title, rel, strings.ToLower(docType))
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

type entry struct{ id, docType, status, title string }

// History prints every Change and Fix whose `affects` names a feature. The
// edge is computed from frontmatter, never from a hand-written back-link on
// the feature — a required back-link is a standing invitation to drift.
func History(root, featureID string, out io.Writer) int {
	if !featureRe.MatchString(featureID) {
		fmt.Fprintf(out, "error: feature id must be <group>/<slug>; got %q\n", featureID)
		return 1
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(featureID)+".md")); err != nil {
		fmt.Fprintf(out, "error: %s is not a feature in this bundle\n", featureID)
		return 1
	}
	var found []entry
	changesDir := filepath.Join(root, "changes")
	filepath.WalkDir(changesDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") || d.Name() == "INDEX.md" {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		text := string(raw)
		m := typeRe.FindStringSubmatch(text)
		if m == nil || (m[1] != "Change" && m[1] != "Fix") {
			return nil
		}
		am := affectsRe.FindStringSubmatch(text)
		if am == nil {
			return nil
		}
		list := strings.Trim(strings.TrimSpace(am[1]), "[]")
		hit := false
		for _, f := range strings.Split(list, ",") {
			if strings.Trim(strings.TrimSpace(f), `"'`) == featureID {
				hit = true
			}
		}
		if !hit {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		e := entry{id: strings.TrimSuffix(filepath.ToSlash(rel), ".md"), docType: m[1]}
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
	if len(found) == 0 {
		fmt.Fprintf(out, "%s: no changes or fixes since delivery\n", featureID)
		return 0
	}
	fmt.Fprintf(out, "%s — %d post-delivery document(s):\n\n", featureID, len(found))
	for _, e := range found {
		fmt.Fprintf(out, "  %-6s %-10s %s\n         %s\n", e.docType, e.status, e.title, e.id)
	}
	return 0
}
