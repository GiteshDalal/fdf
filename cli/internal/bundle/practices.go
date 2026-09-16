package bundle

// v0.6 practice documents. A Practice is the project's binding answer to "how
// do we do X" for one recurring mechanism. It is a living document with no
// episodic trail: its only legal sibling is <slug>.log.md, where approved
// amendments are recorded. F11 is its integrity rule.

import (
	"fmt"
	"sort"
	"strings"
)

type practiceInfo struct {
	rel, id, status, body, supersededBy string
	appliesTo                           []string
}

// sectionText returns the text under a top-level `# <heading>`, up to the next
// top-level heading, and whether the heading was present.
func sectionText(body, heading string) (string, bool) {
	var b strings.Builder
	in := false
	for _, line := range strings.Split(body, "\n") {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			if in {
				break
			}
			in = strings.EqualFold(strings.TrimSpace(m[1]), heading)
			continue
		}
		if in {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String(), in || b.Len() > 0
}

// checkPracticeIntegrity enforces F11: body shape, status invariants, and the
// superseded-by graph.
func checkPracticeIntegrity(practices map[string]*practiceInfo, trails map[string]string, errs *[]string, warns *[]string) {
	// A <slug>.log.md with no sibling practice is an orphan trail file.
	ids := make([]string, 0, len(trails))
	for id := range trails {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if practices[id] == nil {
			*errs = append(*errs, fmt.Sprintf("%s: log file has no sibling practice document %s.md (F3)", trails[id], id))
		}
	}

	graph := map[string][]string{}
	pids := make([]string, 0, len(practices))
	for id := range practices {
		pids = append(pids, id)
	}
	sort.Strings(pids)
	for _, id := range pids {
		p := practices[id]
		if body, ok := sectionText(p.body, "Rules"); !ok || strings.TrimSpace(body) == "" {
			*errs = append(*errs, fmt.Sprintf("%s: a practice needs a non-empty `# Rules` section — the binding statements are the document (F11)", p.rel))
		}
		if len(fenceRe.FindAllStringSubmatch(p.body, -1)) > 0 {
			*errs = append(*errs, fmt.Sprintf("%s: practices carry no Gherkin — behavior statements belong in the features that make them (F11)", p.rel))
		}
		switch p.status {
		case "superseded":
			if p.supersededBy == "" {
				*errs = append(*errs, fmt.Sprintf("%s: status 'superseded' requires `superseded-by` naming the practice that replaces it (F11)", p.rel))
			} else if practices[p.supersededBy] == nil {
				*errs = append(*errs, fmt.Sprintf("%s: superseded-by %q is not a known practice (F11)", p.rel, p.supersededBy))
			} else {
				graph[id] = []string{p.supersededBy}
			}
		case "active":
			if p.supersededBy != "" {
				*errs = append(*errs, fmt.Sprintf("%s: `superseded-by` on an 'active' practice — set status to 'superseded' or drop the field (F11)", p.rel))
			}
			if len(p.appliesTo) == 0 {
				*warns = append(*warns, fmt.Sprintf("%s: active practice has no `applies-to` — no work can be routed to it by path", p.rel))
			}
		}
	}
	if cyc := findCycleIDs(graph); cyc != "" {
		*errs = append(*errs, fmt.Sprintf("practice superseded-by cycle: %s (F11)", cyc))
	}
}
