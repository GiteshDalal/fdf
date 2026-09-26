package bundle

// v0.7 bug documents. A Bug records a known defect — the software doing
// something wrong that someone could observe — that nobody is repairing yet.
// It is a register entry like a Debt, and shares its vocabulary; it differs in
// what it must say (a symptom and the expected behavior) and in one
// restriction: an accepted bug may not contradict a living scenario, because
// then the feature document would promise what the project has decided the
// software will not do. F14 is its integrity rule.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const violatesHeading = "Violates"

// clearedLineRe reads one `fdf bug --cleanup` entry in bugs/LOG.md:
// `* **bugs/<id>** — <title>. <resolution>`.
var clearedLineRe = regexp.MustCompile(`(?m)^\s*[-*]\s+\*\*(bugs/[a-z0-9][a-z0-9/-]*)\*\*`)

// clearedBugs returns the IDs of bugs already retired from the register into
// bugs/LOG.md. A done Fix or Change keeps its `resolves` edge after the bug it
// repaired is cleared; the log is where that bug's ID still exists.
func clearedBugs(rootAbs string) map[string]bool {
	out := map[string]bool{}
	raw, err := os.ReadFile(filepath.Join(rootAbs, "bugs", "LOG.md"))
	if err != nil {
		return out
	}
	for _, m := range clearedLineRe.FindAllStringSubmatch(string(raw), -1) {
		out[m[1]] = true
	}
	return out
}

// checkResolves is the `resolves` half of F10 (v0.7): a Fix or Change names
// the bugs it repairs, and once it has landed, none of them may still read as
// open — a register that calls a repaired defect open is lying. It returns,
// per bug ID, the done documents that resolve it, which F14 needs.
func checkResolves(changes map[string]*changeInfo, bugs map[string]*bugInfo, cleared map[string]bool, errs *[]string) map[string][]string {
	resolvedBy := map[string][]string{}
	ids := make([]string, 0, len(changes))
	for id := range changes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := changes[id]
		for _, bid := range c.resolves {
			b := bugs[bid]
			switch {
			case b != nil:
				if c.status == "done" {
					resolvedBy[bid] = append(resolvedBy[bid], c.rel)
					if b.status != "resolved" {
						*errs = append(*errs, fmt.Sprintf("%s: done, but the bug it resolves, %s, is still '%s' — flip it to 'resolved' with a `# Resolution` naming this document (F10)", c.rel, bid, b.status))
					}
				}
			case cleared[bid]:
				if c.status != "done" {
					*errs = append(*errs, fmt.Sprintf("%s: `resolves` names %s, which was already cleared from the register — work that has not landed cannot be what closed it (F10)", c.rel, bid))
				}
			default:
				*errs = append(*errs, fmt.Sprintf("%s: `resolves` names %q, which is not a bug under bugs/ nor one recorded in bugs/LOG.md (F10)", c.rel, bid))
			}
		}
	}
	return resolvedBy
}

type bugInfo struct {
	rel, id, status, body string
	affects, resource     []string
}

// violations parses `# Violates`: one `## <feature-id>` heading per feature and
// a `- <scenario name>` entry per contradicted scenario, in the regression-case
// grammar — so an entry may carry a note after an em dash, which is not part
// of the name.
func violations(body string) map[string][]string {
	out := map[string][]string{}
	for fid, d := range parseDecls(body, violatesHeading, false) {
		out[fid] = append(append([]string{}, d.regressions...), d.missingVerification...)
	}
	return out
}

// checkBugIntegrity enforces F14. resolvedBy maps a bug ID to the done Fix and
// Change documents whose `resolves` names it. It suggests the full ID of a
// feature named the 0.7 way (featureHint).
func checkBugIntegrity(bugs map[string]*bugInfo, trails map[string]string, features map[string]*featureInfo, resolvedBy map[string][]string, errs, warns *[]string) {
	ids := make([]string, 0, len(trails))
	for id := range trails {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if bugs[id] == nil {
			*errs = append(*errs, fmt.Sprintf("%s: log file has no sibling bug document %s.md (F3)", trails[id], id))
		}
	}

	bids := make([]string, 0, len(bugs))
	for id := range bugs {
		bids = append(bids, id)
	}
	sort.Strings(bids)
	for _, id := range bids {
		b := bugs[id]
		if s, ok := sectionText(b.body, "Symptom"); !ok || strings.TrimSpace(s) == "" {
			*errs = append(*errs, fmt.Sprintf("%s: a bug needs a non-empty `# Symptom` — what the software does wrong, with a reproduction or the code path that reaches it (F14)", b.rel))
		}
		if s, ok := sectionText(b.body, "Expected"); !ok || strings.TrimSpace(s) == "" {
			*errs = append(*errs, fmt.Sprintf("%s: a bug needs a non-empty `# Expected` — what should happen instead, or the open question when no document decides it (F14)", b.rel))
		}
		if len(fenceRe.FindAllStringSubmatch(b.body, -1)) > 0 {
			*errs = append(*errs, fmt.Sprintf("%s: bugs carry no Gherkin — behavior statements belong in the features that make them (F14)", b.rel))
		}

		affected := map[string]bool{}
		for _, fid := range b.affects {
			affected[fid] = true
			if features[fid] == nil {
				*errs = append(*errs, fmt.Sprintf("%s: `affects` names unknown feature %q%s (F14)", b.rel, fid, featureHint(fid, features)))
			}
		}

		for _, e := range strayDeclEntries(b.body, violatesHeading) {
			*errs = append(*errs, fmt.Sprintf("%s: `# Violates` entry %q sits under no `## <feature-id>` heading (F14)", b.rel, e))
		}
		for fid, d := range parseDecls(b.body, violatesHeading, false) {
			if d.headings > 1 {
				*errs = append(*errs, fmt.Sprintf("%s: `# Violates` has %d `## %s` headings — one per feature (F14)", b.rel, d.headings, fid))
			}
			for _, e := range d.malformed {
				*errs = append(*errs, fmt.Sprintf("%s: `## %s` entry %q is not `- <scenario name>` (F14)", b.rel, fid, e))
			}
		}

		viol := violations(b.body)
		vfids := make([]string, 0, len(viol))
		for fid := range viol {
			vfids = append(vfids, fid)
		}
		sort.Strings(vfids)
		for _, fid := range vfids {
			if !affected[fid] {
				*errs = append(*errs, fmt.Sprintf("%s: `# Violates` names `## %s`, which is not listed in `affects`%s (F14)", b.rel, fid, featureHint(fid, features)))
			}
		}

		switch b.status {
		case "accepted":
			if s, ok := sectionText(b.body, "Rationale"); !ok || strings.TrimSpace(s) == "" {
				*errs = append(*errs, fmt.Sprintf("%s: status 'accepted' requires a `# Rationale` — keeping a defect is a decision, and this is where it is recorded (F14)", b.rel))
			}
			if hasHeading(b.body, violatesHeading) {
				*errs = append(*errs, fmt.Sprintf("%s: an accepted bug carries no `# Violates` — a scenario the project knowingly and permanently contradicts makes its feature lie; repair the code (a Fix) or change the scenario (a Change), then resolve the bug (F14)", b.rel))
			}
		case "resolved":
			if s, ok := sectionText(b.body, "Resolution"); !ok || strings.TrimSpace(s) == "" {
				*errs = append(*errs, fmt.Sprintf("%s: status 'resolved' requires a `# Resolution` naming what closed it (F14)", b.rel))
			}
			// A defect known to contradict a scenario is closed only by the
			// document that repaired it: a bug is never repaired in place.
			if hasHeading(b.body, violatesHeading) && len(resolvedBy[id]) == 0 {
				*errs = append(*errs, fmt.Sprintf("%s: resolved while it still has a `# Violates` section, but no done Fix or Change names it in `resolves` — a repair goes through the document that authorizes it; if the finding was wrong, drop `# Violates` and say why in `# Resolution` (F14)", b.rel))
			}
		case "open":
			// While the defect is open, the scenarios it contradicts must be
			// the ones the features say today. A resolved bug's list is its
			// record, and is not held to later edits of the Gherkin.
			for _, fid := range vfids {
				f := features[fid]
				if f == nil {
					continue
				}
				// Only a delivered feature's scenario can be contradicted by a
				// defect: in a feature still being built, the repair is one of
				// its tasks, which has no `resolves` to close the bug with.
				if f.status != "done" && f.status != "adopted" {
					*errs = append(*errs, fmt.Sprintf("%s: `# Violates` names %s, whose status is '%s' — only a delivered feature ('done' or 'adopted') is repaired by a Fix; until then the defect is its tasks' to repair, so drop `# Violates` and keep the scenario in `# Expected` (F14)", b.rel, fid, f.status))
					continue
				}
				names := scenarioNames(f.body)
				for _, n := range viol[fid] {
					if !names[n] {
						*errs = append(*errs, fmt.Sprintf("%s: `# Violates` names scenario %q, which %s does not have — cite the scenario verbatim, or drop it: a defect no scenario covers is repaired by a Change (F14)", b.rel, n, fid))
					}
				}
			}
		}
		if b.status != "resolved" && len(b.resource) == 0 {
			*warns = append(*warns, fmt.Sprintf("%s: %s bug has no `resource` — nothing can route to it by path, so only a reader of the register will find it", b.rel, b.status))
		}
	}
}
