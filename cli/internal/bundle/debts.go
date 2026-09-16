package bundle

// v0.6 debt documents. A Debt records a known gap between what the project
// says and what the code does — work deliberately left undone, or a rule the
// codebase does not follow everywhere yet. It is a register entry, not a unit
// of work: the work that closes it is a Change, a Fix, or plain code work.
// F13 is its integrity rule.

import (
	"fmt"
	"sort"
	"strings"
)

type debtInfo struct {
	rel, id, status, body, title, timestamp string
	resource                                []string
}

// checkDebtIntegrity enforces F13: body shape and the status invariants that
// keep a register from being closed silently or kept without a reason.
func checkDebtIntegrity(debts map[string]*debtInfo, trails map[string]string, errs *[]string, warns *[]string) {
	ids := make([]string, 0, len(trails))
	for id := range trails {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if debts[id] == nil {
			*errs = append(*errs, fmt.Sprintf("%s: log file has no sibling debt document %s.md (F3)", trails[id], id))
		}
	}

	dids := make([]string, 0, len(debts))
	for id := range debts {
		dids = append(dids, id)
	}
	sort.Strings(dids)
	for _, id := range dids {
		d := debts[id]
		if body, ok := sectionText(d.body, "Gap"); !ok || strings.TrimSpace(body) == "" {
			*errs = append(*errs, fmt.Sprintf("%s: a debt needs a non-empty `# Gap` section — a debt nobody can confirm is a TODO (F13)", d.rel))
		}
		if len(fenceRe.FindAllStringSubmatch(d.body, -1)) > 0 {
			*errs = append(*errs, fmt.Sprintf("%s: debts carry no Gherkin — behavior statements belong in the features that make them (F13)", d.rel))
		}
		switch d.status {
		case "accepted":
			if body, ok := sectionText(d.body, "Rationale"); !ok || strings.TrimSpace(body) == "" {
				*errs = append(*errs, fmt.Sprintf("%s: status 'accepted' requires a `# Rationale` — keeping a gap is a decision, and this is where it is recorded (F13)", d.rel))
			}
		case "resolved":
			if body, ok := sectionText(d.body, "Resolution"); !ok || strings.TrimSpace(body) == "" {
				*errs = append(*errs, fmt.Sprintf("%s: status 'resolved' requires a `# Resolution` saying what closed it (F13)", d.rel))
			}
		}
		if d.status != "resolved" && len(d.resource) == 0 {
			*warns = append(*warns, fmt.Sprintf("%s: %s debt has no `resource` — nothing can route to it by path, so only a reader of the register will find it", d.rel, d.status))
		}
	}
}
